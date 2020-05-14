package codec

import (
	"github.com/pion/mediadevices"
	mio "github.com/pion/mediadevices/pkg/io"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v2"
)

const (
	defaultMTU = 1200
)

type rtpReadCloserImpl struct {
	rtpCodec         *webrtc.RTPCodec
	packetize        func(payload []byte) []*rtp.Packet
	encoder          ReadCloser
	buff             []byte
	unreadRTPPackets []*rtp.Packet
}

func NewVideoRTPReadCloser(rtpCodec *webrtc.RTPCodec, track *mediadevices.VideoTrack, reader ReadCloser) (RTPReadCloser, error) {
	return newRTPReadCloser(rtpCodec, track.SSRC(), reader, newVideoSampler(rtpCodec.ClockRate))
}

func NewAudioRTPReadCloser(rtpCodec *webrtc.RTPCodec, track *mediadevices.VideoTrack, reader ReadCloser) (RTPReadCloser, error) {
	return newRTPReadCloser(rtpCodec, track.SSRC(), reader, newAudioSampler(rtpCodec.ClockRate, track.GetSettings().Latency))
}

func newRTPReadCloser(rtpCodec *webrtc.RTPCodec, ssrc uint32, reader ReadCloser, sample samplerFunc) (RTPReadCloser, error) {
	packetizer := rtp.NewPacketizer(
		defaultMTU,
		rtpCodec.PayloadType,
		ssrc,
		rtpCodec.Payloader,
		rtp.NewRandomSequencer(),
		rtpCodec.ClockRate,
	)
	return &rtpReadCloserImpl{
		packetize: func(payload []byte) []*rtp.Packet {
			return packetizer.Packetize(payload, sample())
		},
		rtpCodec: rtpCodec,
		encoder:  reader,
	}, nil
}

func (rc *rtpReadCloserImpl) ReadRTP() (packet *rtp.Packet, err error) {
	var n int

	for {
		if len(rc.unreadRTPPackets) != 0 {
			packet, rc.unreadRTPPackets = rc.unreadRTPPackets[0], rc.unreadRTPPackets[1:]
			return
		}

		n, err = rc.encoder.Read(rc.buff)
		if err != nil {
			e, ok := err.(*mio.InsufficientBufferError)
			if !ok {
				return nil, err
			}

			rc.buff = make([]byte, 2*e.RequiredSize)
		} else {
			rc.unreadRTPPackets = rc.packetize(rc.buff[:n])
		}
	}
}

func (rc *rtpReadCloserImpl) Codec() *webrtc.RTPCodec {
	return rc.rtpCodec
}

func (rc *rtpReadCloserImpl) Close() {
	rc.encoder.Close()
}

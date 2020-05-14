package webrtc

import (
	"fmt"

	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec"
	"github.com/pion/webrtc/v2"
)

type Track interface {
	mediadevices.Track
}

type LocalTrack interface {
	codec.RTPReadCloser
}

type EncoderBuilder interface {
	Codec() *webrtc.RTPCodec
	BuildEncoder(Track) (LocalTrack, error)
}

type MediaEngine struct {
	webrtc.MediaEngine
	encoderBuilders []EncoderBuilder
}

func (engine *MediaEngine) AddEncoderBuilders(builders ...EncoderBuilder) {
	engine.encoderBuilders = append(engine.encoderBuilders, builders...)
	for _, builder := range builders {
		fmt.Printf("%+v", builder.Codec())
		engine.RegisterCodec(builder.Codec())
	}
}

type API struct {
	*webrtc.API
	mediaEngine MediaEngine
}

func NewAPI(options ...func(*API)) *API {
	api := API{API: webrtc.NewAPI()}
	for _, option := range options {
		option(&api)
	}
	return &api
}

func WithMediaEngine(m MediaEngine) func(*API) {
	return func(a *API) {
		a.mediaEngine = m
		webrtc.WithMediaEngine(m.MediaEngine)(a.API)
	}
}

func (api *API) NewPeerConnection(configuration webrtc.Configuration) (*PeerConnection, error) {
	pc, err := api.API.NewPeerConnection(configuration)
	return &PeerConnection{
		PeerConnection: pc,
		api:            api,
	}, err
}

type PeerConnection struct {
	*webrtc.PeerConnection
	api *API
}

func buildEncoder(encoderBuilders []EncoderBuilder, track Track) LocalTrack {
	for _, encoderBuilder := range encoderBuilders {
		encoder, err := encoderBuilder.BuildEncoder(track)
		if err == nil {
			return encoder
		}
	}
	return nil
}

func (pc *PeerConnection) ExtAddTransceiverFromTrack(track Track, init ...webrtc.RtpTransceiverInit) (*webrtc.RTPTransceiver, error) {
	encoder := buildEncoder(pc.api.mediaEngine.encoderBuilders, track)
	if encoder == nil {
		return nil, fmt.Errorf("failed to find a compatible encoder")
	}

	rtpCodec := encoder.Codec()
	trackImpl, err := pc.NewTrack(rtpCodec.PayloadType, track.SSRC(), track.ID(), rtpCodec.Type.String())
	if err != nil {
		return nil, err
	}

	trans, err := pc.AddTransceiverFromTrack(trackImpl, init...)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			rtpPacket, err := encoder.ReadRTP()
			if err != nil {
				panic(err)
			}

			err = trackImpl.WriteRTP(rtpPacket)
			if err != nil {
				panic(err)
			}
		}
	}()

	return trans, nil
}

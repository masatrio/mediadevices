package video

import (
	"image"
	"testing"
	"time"
)

func TestThrottle(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 640, 480))

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	var cntPush int
	trans := Throttle(50)
	r := trans(ReaderFunc(func() (image.Image, error) {
		<-ticker.C
		cntPush++
		return img, nil
	}))

	for i := 0; i < 20; i++ {
		_, err := r.Read()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	}
	cntExpected := 20
	if cntPush < cntExpected*8/10 || cntExpected*12/10 < cntPush {
		t.Fatalf("Number of pushed images is expected to be %d, but pushed %d", cntExpected, cntPush)
	}
	t.Log(cntPush)
}

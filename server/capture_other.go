//go:build !windows

package main

import (
	"image"
	"image/color"
	"time"
)

// Test-pattern capturer used when the server is built for Linux/macOS, so the
// tablet UI and streaming can be tested without DCS.
type testCapturer struct{ start time.Time }

func newCapturer() (Capturer, error) { return &testCapturer{start: time.Now()}, nil }

func (c *testCapturer) Capture(r Rect) (*image.RGBA, error) {
	img := image.NewRGBA(image.Rect(0, 0, r.W, r.H))
	t := time.Since(c.start).Seconds()
	sweep := int(t*120) % r.W
	g := uint8(40 + (r.X/10)%120)
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			v := color.RGBA{0, 12, 0, 255}
			if x%62 == 0 || y%62 == 0 {
				v = color.RGBA{0, g, 0, 255}
			}
			if x == sweep || x == sweep+1 {
				v = color.RGBA{60, 255, 60, 255}
			}
			img.SetRGBA(x, y, v)
		}
	}
	return img, nil
}

func (c *testCapturer) Close() {}

func virtualDesktop() Rect { return Rect{0, 0, 3840, 1080} }

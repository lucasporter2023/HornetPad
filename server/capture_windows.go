//go:build windows

package main

import (
	"errors"
	"image"
	"syscall"
	"unsafe"
)

var (
	user32 = syscall.NewLazyDLL("user32.dll")
	gdi32  = syscall.NewLazyDLL("gdi32.dll")

	procGetDC                         = user32.NewProc("GetDC")
	procReleaseDC                     = user32.NewProc("ReleaseDC")
	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procSetProcessDPIAware            = user32.NewProc("SetProcessDPIAware")
	procGetSystemMetrics              = user32.NewProc("GetSystemMetrics")

	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procBitBlt             = gdi32.NewProc("BitBlt")
	procGdiFlush           = gdi32.NewProc("GdiFlush")
)

const (
	srcCopy        = 0x00CC0020
	dibRGBColors   = 0
	smXVirtual     = 76
	smYVirtual     = 77
	smCXVirtual    = 78
	smCYVirtual    = 79
	dpiPerMonitor2 = ^uintptr(3) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 = -4
)

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

func init() {
	// Use physical pixels so capture coordinates match the MonitorSetup lua
	// even when Windows display scaling is not 100%.
	if procSetProcessDpiAwarenessContext.Find() == nil {
		if r, _, _ := procSetProcessDpiAwarenessContext.Call(dpiPerMonitor2); r != 0 {
			return
		}
	}
	procSetProcessDPIAware.Call()
}

// gdiCapturer keeps one memory DC + DIB section and reuses it every frame.
type gdiCapturer struct {
	memDC  uintptr
	bitmap uintptr
	old    uintptr
	bits   unsafe.Pointer
	w, h   int
}

func newCapturer() (Capturer, error) { return &gdiCapturer{}, nil }

func (c *gdiCapturer) ensure(w, h int) error {
	if c.memDC != 0 && c.w == w && c.h == h {
		return nil
	}
	c.release()
	screen, _, _ := procGetDC.Call(0)
	if screen == 0 {
		return errors.New("GetDC failed")
	}
	defer procReleaseDC.Call(0, screen)

	memDC, _, _ := procCreateCompatibleDC.Call(screen)
	if memDC == 0 {
		return errors.New("CreateCompatibleDC failed")
	}
	bi := bitmapInfoHeader{
		Size:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:    int32(w),
		Height:   -int32(h), // negative = top-down rows
		Planes:   1,
		BitCount: 32,
	}
	var bits unsafe.Pointer
	bmp, _, _ := procCreateDIBSection.Call(screen, uintptr(unsafe.Pointer(&bi)), dibRGBColors,
		uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 || bits == nil {
		procDeleteDC.Call(memDC)
		return errors.New("CreateDIBSection failed")
	}
	old, _, _ := procSelectObject.Call(memDC, bmp)
	c.memDC, c.bitmap, c.old, c.bits, c.w, c.h = memDC, bmp, old, bits, w, h
	return nil
}

func (c *gdiCapturer) release() {
	if c.memDC != 0 {
		procSelectObject.Call(c.memDC, c.old)
		procDeleteObject.Call(c.bitmap)
		procDeleteDC.Call(c.memDC)
	}
	c.memDC, c.bitmap, c.old, c.bits = 0, 0, 0, nil
}

// Capture copies one desktop rectangle into a new RGBA image.
func (c *gdiCapturer) Capture(r Rect) (*image.RGBA, error) {
	if err := c.ensure(r.W, r.H); err != nil {
		return nil, err
	}
	screen, _, _ := procGetDC.Call(0)
	if screen == 0 {
		return nil, errors.New("GetDC failed")
	}
	ok, _, _ := procBitBlt.Call(c.memDC, 0, 0, uintptr(r.W), uintptr(r.H), screen,
		uintptr(int32(r.X)), uintptr(int32(r.Y)), srcCopy)
	procReleaseDC.Call(0, screen)
	if ok == 0 {
		return nil, errors.New("BitBlt failed (is the desktop locked or a UAC prompt open?)")
	}
	procGdiFlush.Call()

	n := r.W * r.H * 4
	src := unsafe.Slice((*byte)(c.bits), n)
	img := image.NewRGBA(image.Rect(0, 0, r.W, r.H))
	dst := img.Pix
	for i := 0; i < n; i += 4 { // BGRA -> RGBA
		dst[i] = src[i+2]
		dst[i+1] = src[i+1]
		dst[i+2] = src[i]
		dst[i+3] = 255
	}
	return img, nil
}

func (c *gdiCapturer) Close() { c.release() }

// virtualDesktop returns the bounds of all monitors combined, for diagnostics.
func virtualDesktop() Rect {
	x, _, _ := procGetSystemMetrics.Call(smXVirtual)
	y, _, _ := procGetSystemMetrics.Call(smYVirtual)
	w, _, _ := procGetSystemMetrics.Call(smCXVirtual)
	h, _, _ := procGetSystemMetrics.Call(smCYVirtual)
	return Rect{X: int(int32(x)), Y: int(int32(y)), W: int(int32(w)), H: int(int32(h))}
}

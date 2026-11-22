package main

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type Capturer interface {
	Capture(r Rect) (*image.RGBA, error)
	Close()
}

// Feed holds the latest JPEG for one display and wakes waiting viewers.
type Feed struct {
	mu    sync.Mutex
	frame []byte
	seq   uint64
	ch    chan struct{}
}

func newFeed() *Feed { return &Feed{ch: make(chan struct{})} }

func (f *Feed) publish(b []byte) {
	f.mu.Lock()
	f.frame = b
	f.seq++
	close(f.ch)
	f.ch = make(chan struct{})
	f.mu.Unlock()
}

// next blocks until a frame newer than seq exists, or timeout (then returns the current one).
func (f *Feed) next(seq uint64, timeout time.Duration, done <-chan struct{}) ([]byte, uint64, bool) {
	f.mu.Lock()
	if f.seq != seq && f.frame != nil {
		b, s := f.frame, f.seq
		f.mu.Unlock()
		return b, s, true
	}
	ch := f.ch
	f.mu.Unlock()
	select {
	case <-ch:
	case <-time.After(timeout):
	case <-done:
		return nil, seq, false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.frame, f.seq, f.frame != nil
}

type Streamer struct {
	cfg     Config
	feeds   map[string]*Feed
	viewers atomic.Int32
	fps     atomic.Value // string, for the status line
}

func newStreamer(cfg Config) *Streamer {
	s := &Streamer{cfg: cfg, feeds: map[string]*Feed{}}
	for name := range cfg.Displays {
		s.feeds[name] = newFeed()
	}
	s.fps.Store("idle")
	return s
}

// run captures all displays at cfg.FPS while at least one viewer is connected.
func (s *Streamer) run() {
	runtime.LockOSThread() // GDI objects belong to the thread that created them
	caps := map[string]Capturer{}
	for name := range s.cfg.Displays {
		c, err := newCapturer()
		if err != nil {
			log.Fatalf("capture init: %v", err)
		}
		caps[name] = c
	}
	lastCRC := map[string]uint32{}
	interval := time.Second / time.Duration(s.cfg.FPS)
	opts := &jpeg.Options{Quality: s.cfg.JPEGQuality}
	var buf bytes.Buffer
	frames, sent := 0, 0
	statT := time.Now()
	lastErr := ""

	for {
		if s.viewers.Load() == 0 {
			s.fps.Store("idle (no tablet viewing)")
			lastCRC = map[string]uint32{}
			time.Sleep(200 * time.Millisecond)
			continue
		}
		t0 := time.Now()
		for name, r := range s.cfg.Displays {
			img, err := caps[name].Capture(r)
			if err != nil {
				if err.Error() != lastErr {
					log.Printf("capture %s: %v", name, err)
					lastErr = err.Error()
				}
				continue
			}
			crc := crc32.ChecksumIEEE(img.Pix)
			if old, ok := lastCRC[name]; ok && old == crc {
				continue // unchanged; viewers get a keep-alive resend instead
			}
			lastCRC[name] = crc
			buf.Reset()
			if err := jpeg.Encode(&buf, img, opts); err != nil {
				continue
			}
			s.feeds[name].publish(append([]byte(nil), buf.Bytes()...))
			sent++
		}
		frames++
		if el := time.Since(statT); el >= 2*time.Second {
			s.fps.Store(fmt.Sprintf("%.0f fps capture, %.0f img/s sent", float64(frames)/el.Seconds(), float64(sent)/el.Seconds()))
			frames, sent, statT = 0, 0, time.Now()
		}
		if d := interval - time.Since(t0); d > 0 {
			time.Sleep(d)
		}
	}
}

// ServeMJPEG streams one display as multipart/x-mixed-replace (plays in any <img>).
func (s *Streamer) ServeMJPEG(w http.ResponseWriter, r *http.Request, name string) {
	feed, ok := s.feeds[name]
	if !ok {
		http.NotFound(w, r)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	s.viewers.Add(1)
	defer s.viewers.Add(-1)

	h := w.Header()
	h.Set("Content-Type", "multipart/x-mixed-replace; boundary=hpframe")
	h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	h.Set("Pragma", "no-cache")
	h.Set("Connection", "close")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("--hpframe\r\n")); err != nil {
		return
	}
	flusher.Flush()

	var seq uint64
	done := r.Context().Done()
	for {
		frame, newSeq, ok := feed.next(seq, 2*time.Second, done)
		select {
		case <-done:
			return
		default:
		}
		if !ok {
			continue
		}
		seq = newSeq
		// Boundary is written right after each frame so the browser shows it
		// immediately instead of waiting for the next one.
		hdr := fmt.Sprintf("Content-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", len(frame))
		if _, err := w.Write([]byte(hdr)); err != nil {
			return
		}
		if _, err := w.Write(frame); err != nil {
			return
		}
		if _, err := w.Write([]byte("\r\n--hpframe\r\n")); err != nil {
			return
		}
		flusher.Flush()
	}
}

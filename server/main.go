// HornetPad - DCS F/A-18C DDI/AMPCD touch export for an Android tablet over USB.
//
// The PC side captures the three exported displays from the screen, streams
// them to the tablet as MJPEG, and turns tablet button presses into DCS-BIOS
// commands sent to DCS over UDP.
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"image/png"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const version = "1.0.0"

//go:embed web
var webFiles embed.FS

func main() {
	snapshot := flag.Bool("snapshot", false, "save a PNG of each capture region next to the exe and exit")
	flag.Parse()
	log.SetFlags(log.Ltime)

	fmt.Printf("HornetPad %s - F/A-18C DDI/AMPCD tablet export\n\n", version)
	cfg, cfgPath, err := loadConfig()
	if err != nil {
		fatal(err)
	}
	fmt.Printf("Config:  %s\n", cfgPath)
	vd := virtualDesktop()
	fmt.Printf("Desktop: %dx%d at (%d,%d)\n", vd.W, vd.H, vd.X, vd.Y)
	for _, name := range cfg.Order {
		r := cfg.Displays[name]
		warn := ""
		if r.X < vd.X || r.Y < vd.Y || r.X+r.W > vd.X+vd.W || r.Y+r.H > vd.Y+vd.H {
			warn = "   <-- OUTSIDE the desktop, check config.json / virtual display"
		}
		fmt.Printf("  %-5s  %dx%d at (%d,%d)%s\n", name, r.W, r.H, r.X, r.Y, warn)
	}

	if *snapshot {
		doSnapshot(cfg)
		return
	}

	bios, err := newBIOS(cfg.DCSBIOS)
	if err != nil {
		fatal(fmt.Errorf("DCS-BIOS address %q: %w", cfg.DCSBIOS, err))
	}
	streamer := newStreamer(cfg)
	go streamer.run()
	adb := newADB(cfg)
	go adb.loop()

	mux := http.NewServeMux()
	web, _ := fs.Sub(webFiles, "web")
	mux.Handle("/", noCache(http.FileServer(http.FS(web))))
	mux.HandleFunc("/stream/", func(w http.ResponseWriter, r *http.Request) {
		streamer.ServeMJPEG(w, r, strings.TrimPrefix(r.URL.Path, "/stream/"))
	})
	mux.Handle("/cmd", bios)
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		disp := map[string][2]int{}
		for k, v := range cfg.Displays {
			disp[k] = [2]int{v.W, v.H}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(map[string]any{
			"app": "HornetPad", "version": version, "order": cfg.Order, "displays": disp,
		})
	})

	host := "127.0.0.1"
	if cfg.AllowLAN {
		host = "0.0.0.0"
	}
	addr := fmt.Sprintf("%s:%d", host, cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fatal(fmt.Errorf("cannot listen on %s (is HornetPad already running?): %w", addr, err))
	}
	fmt.Printf("\nServer:  http://127.0.0.1:%d  (tablet uses the same address over USB)\n", cfg.Port)
	fmt.Printf("DCS-BIOS commands -> udp://%s\n\n", cfg.DCSBIOS)

	go statusLoop(adb, streamer, bios)
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	fatal(srv.Serve(ln))
}

func statusLoop(adb *ADB, s *Streamer, b *BIOS) {
	last := ""
	for {
		line := fmt.Sprintf("USB: %s | Video: %s | viewers %d | commands sent %d (last: %s)",
			adb.Status(), s.fps.Load(), s.viewers.Load(), b.count.Load(), b.last.Load())
		if line != last {
			log.Println(line)
			last = line
		}
		time.Sleep(2 * time.Second)
	}
}

func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		h.ServeHTTP(w, r)
	})
}

func doSnapshot(cfg Config) {
	c, err := newCapturer()
	if err != nil {
		fatal(err)
	}
	defer c.Close()
	for name, r := range cfg.Displays {
		img, err := c.Capture(r)
		if err != nil {
			fmt.Printf("%s: %v\n", name, err)
			continue
		}
		path := filepath.Join(exeDir(), "snapshot_"+name+".png")
		f, err := os.Create(path)
		if err != nil {
			fmt.Printf("%s: %v\n", name, err)
			continue
		}
		png.Encode(f, img)
		f.Close()
		fmt.Printf("saved %s\n", path)
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "\nERROR: %v\n\nPress Enter to close.", err)
	fmt.Scanln()
	os.Exit(1)
}

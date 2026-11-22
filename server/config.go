package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Rect is a capture region in Windows virtual-desktop pixels.
type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type Config struct {
	// HTTP port the tablet connects to (forwarded over USB with "adb reverse").
	Port int `json:"port"`
	// false = only reachable over USB (127.0.0.1). true = also reachable over Wi-Fi/LAN.
	AllowLAN bool `json:"allowLan"`
	// Capture rate and JPEG quality (1-100).
	FPS         int `json:"fps"`
	JPEGQuality int `json:"jpegQuality"`
	// Where DCS-BIOS listens for commands (default 127.0.0.1:7778).
	DCSBIOS string `json:"dcsBios"`
	// adb.exe location. Empty = auto-detect (next to this exe, PATH, Android SDK).
	ADBPath string `json:"adbPath"`
	// Run "adb reverse" automatically for every tablet plugged in.
	AutoADB bool `json:"autoAdb"`
	// Launch the HornetPad app on the tablet when it is plugged in.
	AutoLaunchApp bool `json:"autoLaunchApp"`
	// Screen regions of the DCS exports (must match your MonitorSetup lua).
	Displays map[string]Rect `json:"displays"`
	// Left-to-right order on the tablet.
	Order []string `json:"order"`
}

func defaultConfig() Config {
	return Config{
		Port:          8765,
		AllowLAN:      false,
		FPS:           30,
		JPEGQuality:   80,
		DCSBIOS:       "127.0.0.1:7778",
		AutoADB:       true,
		AutoLaunchApp: true,
		// Matches dcs/HornetPad.lua: virtual 1920x1080 display right of the
		// 1920x1080 primary, three 620x620 exports with 20 px gaps.
		Displays: map[string]Rect{
			"LDDI":  {X: 1920, Y: 0, W: 620, H: 620},
			"AMPCD": {X: 2560, Y: 0, W: 620, H: 620},
			"RDDI":  {X: 3200, Y: 0, W: 620, H: 620},
		},
		Order: []string{"LDDI", "AMPCD", "RDDI"},
	}
}

func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// loadConfig reads config.json next to the exe, creating it with defaults if missing.
func loadConfig() (Config, string, error) {
	path := filepath.Join(exeDir(), "config.json")
	cfg := defaultConfig()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		out, _ := json.MarshalIndent(cfg, "", "  ")
		if werr := os.WriteFile(path, out, 0o644); werr != nil {
			return cfg, path, fmt.Errorf("could not write default config: %w", werr)
		}
		return cfg, path, nil
	}
	if err != nil {
		return cfg, path, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, path, fmt.Errorf("config.json is not valid JSON: %w", err)
	}
	if cfg.FPS < 1 || cfg.FPS > 60 {
		cfg.FPS = 30
	}
	if cfg.JPEGQuality < 10 || cfg.JPEGQuality > 100 {
		cfg.JPEGQuality = 80
	}
	if cfg.Port == 0 {
		cfg.Port = 8765
	}
	if cfg.DCSBIOS == "" {
		cfg.DCSBIOS = "127.0.0.1:7778"
	}
	for _, name := range []string{"LDDI", "AMPCD", "RDDI"} {
		r, ok := cfg.Displays[name]
		if !ok || r.W <= 0 || r.H <= 0 {
			return cfg, path, fmt.Errorf("config.json: display %q missing or has zero size", name)
		}
	}
	if len(cfg.Order) != 3 {
		cfg.Order = []string{"LDDI", "AMPCD", "RDDI"}
	}
	return cfg, path, nil
}

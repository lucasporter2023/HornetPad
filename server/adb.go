package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const appComponent = "com.hornetpad/.MainActivity"

// ADB keeps "adb reverse tcp:PORT tcp:PORT" in place for every connected tablet,
// so the tablet reaches this PC at http://127.0.0.1:PORT over the USB cable.
type ADB struct {
	cfg    Config
	path   string
	mu     sync.Mutex
	status string
	done   map[string]bool
}

func findADB(cfg Config) string {
	name := "adb"
	if runtime.GOOS == "windows" {
		name = "adb.exe"
	}
	cands := []string{}
	if cfg.ADBPath != "" {
		cands = append(cands, cfg.ADBPath)
	}
	cands = append(cands,
		filepath.Join(exeDir(), name),
		filepath.Join(exeDir(), "platform-tools", name))
	if p, err := exec.LookPath(name); err == nil {
		cands = append(cands, p)
	}
	if la := os.Getenv("LOCALAPPDATA"); la != "" {
		cands = append(cands, filepath.Join(la, "Android", "Sdk", "platform-tools", name))
	}
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		cands = append(cands, filepath.Join(pf, "platform-tools", name))
	}
	cands = append(cands, `C:\platform-tools\adb.exe`, `C:\adb\adb.exe`)
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

func newADB(cfg Config) *ADB {
	a := &ADB{cfg: cfg, done: map[string]bool{}}
	a.path = findADB(cfg)
	if a.path == "" {
		a.status = "adb.exe not found - put platform-tools next to HornetPad.exe (see README)"
	} else {
		a.status = "waiting for tablet (USB debugging on, cable plugged in)"
	}
	return a
}

func (a *ADB) Status() string { a.mu.Lock(); defer a.mu.Unlock(); return a.status }

func (a *ADB) set(s string) { a.mu.Lock(); a.status = s; a.mu.Unlock() }

func (a *ADB) run(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, a.path, args...)
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func (a *ADB) loop() {
	if a.path == "" {
		return
	}
	port := fmt.Sprintf("tcp:%d", a.cfg.Port)
	for {
		out, err := a.run("devices")
		if err != nil {
			a.set("adb error: " + err.Error())
			time.Sleep(3 * time.Second)
			continue
		}
		online := map[string]bool{}
		unauthorized := false
		for _, line := range strings.Split(out, "\n") {
			f := strings.Fields(line)
			if len(f) != 2 {
				continue
			}
			switch f[1] {
			case "device":
				online[f[0]] = true
			case "unauthorized":
				unauthorized = true
			}
		}
		for serial := range a.done {
			if !online[serial] {
				delete(a.done, serial) // unplugged: redo when it comes back
			}
		}
		for serial := range online {
			list, _ := a.run("-s", serial, "reverse", "--list")
			if a.done[serial] && strings.Contains(list, port) {
				continue
			}
			if res, err := a.run("-s", serial, "reverse", port, port); err != nil {
				a.set(fmt.Sprintf("adb reverse failed on %s: %s", serial, res))
				continue
			}
			if !a.done[serial] && a.cfg.AutoLaunchApp {
				a.run("-s", serial, "shell", "am", "start", "-n", appComponent)
			}
			a.done[serial] = true
		}
		switch {
		case len(online) > 0:
			names := []string{}
			for s := range online {
				names = append(names, s)
			}
			a.set("tablet connected over USB (" + strings.Join(names, ", ") + ")")
		case unauthorized:
			a.set("tablet found but not authorized - tap 'Allow' on the USB debugging prompt")
		default:
			a.set("waiting for tablet (USB debugging on, cable plugged in)")
		}
		time.Sleep(3 * time.Second)
	}
}

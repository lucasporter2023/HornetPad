package main

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
)

// DCS-BIOS input protocol: plain text "CONTROL_NAME ARGUMENT\n" sent over UDP.
var cmdRe = regexp.MustCompile(`^[A-Z0-9_]{2,64} [A-Za-z0-9+\-]{1,12}$`)

type BIOS struct {
	mu    sync.Mutex
	conn  net.Conn
	addr  string
	count atomic.Int64
	last  atomic.Value
}

func newBIOS(addr string) (*BIOS, error) {
	c, err := net.Dial("udp", addr)
	if err != nil {
		return nil, err
	}
	b := &BIOS{conn: c, addr: addr}
	b.last.Store("-")
	return b, nil
}

func (b *BIOS) Send(cmd string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, err := b.conn.Write([]byte(cmd + "\n"))
	if err == nil {
		b.count.Add(1)
		b.last.Store(cmd)
	}
	return err
}

// ServeHTTP accepts POST /cmd with one command per line, e.g. "LEFT_DDI_PB_18 1".
func (b *BIOS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	sc := bufio.NewScanner(io.LimitReader(r.Body, 4096))
	n := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if !cmdRe.MatchString(line) {
			http.Error(w, "bad command: "+line, http.StatusBadRequest)
			return
		}
		if err := b.Send(line); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		n++
	}
	w.WriteHeader(http.StatusNoContent)
}

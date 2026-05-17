// SPDX-FileCopyrightText: 2026 Markus Katharina Brechtel <markus.katharina.brechtel@thengo.net>
// SPDX-License-Identifier: EUPL-1.2

// Lightweight asciicast v2 writer for the PTY-driven e2e tests.
// Spec: https://docs.asciinema.org/manual/asciicast/v2/
//
// When ASCIINEMA_OUT_DIR is set, each PTY test wraps its captured
// output through castWriter and dumps `<TestName>.cast` to that
// directory. The cast can then be played with `asciinema play
// <file>` or rendered to GIF with `agg <file>`. Goal: give the
// reviewer a one-command way to SEE what each test does, instead of
// having to read scripted keystrokes and infer.

package e2e_test

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// castEnvVar names the directory env var the e2e tests look for.
const castEnvVar = "ASCIINEMA_OUT_DIR"

// castWriter wraps an io.Writer and serialises every chunk written
// to it as an asciicast `["o"]` event timestamped relative to start.
// Safe for concurrent writes (the pty drain goroutine and the test
// driver share the same writer).
type castWriter struct {
	mu    sync.Mutex
	out   io.Writer
	start time.Time
}

func (c *castWriter) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(p) == 0 {
		return 0, nil
	}
	ev := [3]any{
		time.Since(c.start).Seconds(),
		"o",
		string(p),
	}
	enc, err := json.Marshal(ev)
	if err != nil {
		return 0, err
	}
	if _, err := c.out.Write(enc); err != nil {
		return 0, err
	}
	if _, err := c.out.Write([]byte("\n")); err != nil {
		return 0, err
	}
	return len(p), nil
}

// openCast prepares an asciicast v2 file for `t` if ASCIINEMA_OUT_DIR
// is set. Returns the writer (as an io.Writer interface — a true nil
// when recording is off, so callers can pass it straight to
// teeWriters without a typed-nil trap) and a cleanup that closes
// the file.
//
// width/height/title go into the cast header; pick the same Winsize
// you give the PTY so playback matches the test geometry.
func openCast(t *testing.T, width, height int) (io.Writer, func()) {
	t.Helper()
	dir := os.Getenv(castEnvVar)
	if dir == "" {
		return nil, func() {}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("openCast: mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, t.Name()+".cast")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("openCast: %v", err)
	}
	header := map[string]any{
		"version":   2,
		"width":     width,
		"height":    height,
		"timestamp": time.Now().Unix(),
		"title":     t.Name(),
		"env": map[string]string{
			"TERM":  "xterm-256color",
			"SHELL": "/bin/sh",
		},
	}
	hb, err := json.Marshal(header)
	if err != nil {
		_ = f.Close()
		t.Fatalf("openCast: header: %v", err)
	}
	if _, err := fmt.Fprintln(f, string(hb)); err != nil {
		_ = f.Close()
		t.Fatalf("openCast: write header: %v", err)
	}
	t.Logf("asciicast → %s", path)
	cw := &castWriter{out: f, start: time.Now()}
	cleanup := func() {
		_ = f.Sync()
		_ = f.Close()
	}
	return cw, cleanup
}

// tee returns a writer that fans `src` writes into all `dsts` plus
// the captured-bytes buffer used for assertions. nil entries are
// skipped so an absent cast writer is harmless.
func teeWriters(dsts ...io.Writer) io.Writer {
	var live []io.Writer
	for _, w := range dsts {
		if w != nil {
			live = append(live, w)
		}
	}
	switch len(live) {
	case 0:
		return io.Discard
	case 1:
		return live[0]
	default:
		return io.MultiWriter(live...)
	}
}

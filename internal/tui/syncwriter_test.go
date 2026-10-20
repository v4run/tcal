package tui

import (
	"bytes"
	"testing"
)

func TestSyncWriter_BracketsEachFlush(t *testing.T) {
	var buf bytes.Buffer
	w := NewSyncWriter(&buf)
	n, err := w.Write([]byte("frame"))
	if err != nil || n != 5 {
		t.Fatalf("Write = %d, %v; want 5, nil", n, err)
	}
	if got, want := buf.String(), "\x1b[?2026hframe\x1b[?2026l"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	buf.Reset()
	w.Write([]byte("a"))
	w.Write([]byte("b"))
	if got, want := buf.String(), "\x1b[?2026ha\x1b[?2026l\x1b[?2026hb\x1b[?2026l"; got != want {
		t.Errorf("two writes: got %q, want %q", got, want)
	}
	buf.Reset()
	if n, err := w.Write(nil); n != 0 || err != nil || buf.Len() != 0 {
		t.Errorf("empty write: %d, %v, %q; want passthrough", n, err, buf.String())
	}
}

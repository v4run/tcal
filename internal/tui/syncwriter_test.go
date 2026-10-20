package tui

import (
	"io"
	"os"
	"testing"

	"github.com/charmbracelet/x/term"
)

var _ term.File = (*syncFile)(nil)

func syncPipe(t *testing.T) (*syncFile, *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close(); w.Close() })
	return NewSyncWriter(w), r
}

func readN(t *testing.T, r *os.File, n int) string {
	t.Helper()
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSyncWriter_BracketsEachFlush(t *testing.T) {
	w, r := syncPipe(t)
	n, err := w.Write([]byte("frame"))
	if err != nil || n != 5 {
		t.Fatalf("Write = %d, %v; want 5, nil", n, err)
	}
	want := "\x1b[?2026hframe\x1b[?2026l"
	if got := readN(t, r, len(want)); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	w.Write([]byte("a"))
	w.Write([]byte("b"))
	want = "\x1b[?2026ha\x1b[?2026l\x1b[?2026hb\x1b[?2026l"
	if got := readN(t, r, len(want)); got != want {
		t.Errorf("two writes: got %q, want %q", got, want)
	}
	if n, err := w.Write(nil); n != 0 || err != nil {
		t.Errorf("empty write: %d, %v", n, err)
	}
}

func TestSyncWriter_WriteStringBrackets(t *testing.T) {
	w, r := syncPipe(t)
	n, err := io.WriteString(w, "hi")
	if err != nil || n != 2 {
		t.Fatalf("WriteString = %d, %v; want 2, nil", n, err)
	}
	want := "\x1b[?2026hhi\x1b[?2026l"
	if got := readN(t, r, len(want)); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSyncWriter_KeepsFd(t *testing.T) {
	w, _ := syncPipe(t)
	if w.Fd() != w.File.Fd() {
		t.Errorf("Fd = %d, want %d", w.Fd(), w.File.Fd())
	}
}

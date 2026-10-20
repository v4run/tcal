package tui

import "os"

const (
	syncBegin = "\x1b[?2026h"
	syncEnd   = "\x1b[?2026l"
)

// syncFile embeds the *os.File so Fd, Read and Close pass through: Bubble
// Tea only treats its output as a terminal (size, restore) when it
// implements term.File.
type syncFile struct{ *os.File }

// NewSyncWriter wraps f so every Write and WriteString is bracketed by DEC
// mode 2026 (synchronized output). Bubble Tea's renderer flushes once per
// frame, so each frame is applied atomically by terminals that support the
// mode; others ignore it. Empty writes pass through untouched.
func NewSyncWriter(f *os.File) *syncFile { return &syncFile{f} }

func (s *syncFile) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return s.File.Write(p)
	}
	buf := make([]byte, 0, len(syncBegin)+len(p)+len(syncEnd))
	buf = append(buf, syncBegin...)
	buf = append(buf, p...)
	buf = append(buf, syncEnd...)
	n, err := s.File.Write(buf)
	if err != nil {
		return max(n-len(syncBegin), 0), err
	}
	return len(p), nil
}

func (s *syncFile) WriteString(str string) (int, error) { return s.Write([]byte(str)) }

package tui

import "io"

const (
	syncBegin = "\x1b[?2026h"
	syncEnd   = "\x1b[?2026l"
)

type syncWriter struct{ w io.Writer }

// NewSyncWriter wraps w so every Write is bracketed by DEC mode 2026
// (synchronized output). Bubble Tea's renderer flushes once per frame, so
// each frame is applied atomically by terminals that support the mode; others
// ignore it. Empty writes pass through untouched.
func NewSyncWriter(w io.Writer) io.Writer { return syncWriter{w} }

func (s syncWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return s.w.Write(p)
	}
	buf := make([]byte, 0, len(syncBegin)+len(p)+len(syncEnd))
	buf = append(buf, syncBegin...)
	buf = append(buf, p...)
	buf = append(buf, syncEnd...)
	n, err := s.w.Write(buf)
	if err != nil {
		return max(n-len(syncBegin), 0), err
	}
	return len(p), nil
}

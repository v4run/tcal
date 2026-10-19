package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/life"
)

const (
	lifeGlyph = "·"
	faintOn   = "\x1b[2m"
	faintOff  = "\x1b[22m"
)

// overlayLife draws the live cells of g into the blank margins of frame.
// The calendar's bounding box, plus one cell of padding, is left untouched.
// frame must be exactly g.Width() × g.Height() (as produced by render.Fit);
// otherwise it is returned unchanged.
func overlayLife(frame string, g *life.Grid, color bool) string {
	w, h := g.Width(), g.Height()
	lines := strings.Split(frame, "\n")
	if len(lines) != h {
		return frame
	}
	for _, ln := range lines {
		if lipgloss.Width(ln) != w {
			return frame
		}
	}

	// Content rows are those with any visible non-space character.
	y0, y1 := -1, -1
	for y, ln := range lines {
		if strings.TrimSpace(ansi.Strip(ln)) != "" {
			if y0 < 0 {
				y0 = y
			}
			y1 = y
		}
	}

	if y0 < 0 {
		for y := range lines {
			lines[y] = cells(g, y, 0, w, color)
		}
		return strings.Join(lines, "\n")
	}

	// render.Fit pads with plain spaces, so leading/trailing ' ' bytes on
	// content rows are pure margin and safe to replace byte-for-byte.
	x0, x1 := w, 0
	for _, ln := range lines[y0 : y1+1] {
		lead := len(ln) - len(strings.TrimLeft(ln, " "))
		trail := len(ln) - len(strings.TrimRight(ln, " "))
		x0 = min(x0, lead)
		x1 = max(x1, w-trail)
	}

	// Pad the box by one cell on every side.
	y0, y1 = y0-1, y1+1
	x0, x1 = max(x0-1, 0), min(x1+1, w)

	for y, ln := range lines {
		if y < y0 || y > y1 {
			lines[y] = cells(g, y, 0, w, color)
			continue
		}
		lines[y] = cells(g, y, 0, x0, color) + ln[x0:len(ln)-(w-x1)] + cells(g, y, x1, w, color)
	}
	return strings.Join(lines, "\n")
}

// cells renders row y of g over columns [from, to) as glyphs and spaces.
func cells(g *life.Grid, y, from, to int, color bool) string {
	if from >= to {
		return ""
	}
	var b strings.Builder
	if color {
		b.WriteString(faintOn)
	}
	for x := from; x < to; x++ {
		if g.Alive(x, y) {
			b.WriteString(lifeGlyph)
		} else {
			b.WriteByte(' ')
		}
	}
	if color {
		b.WriteString(faintOff)
	}
	return b.String()
}

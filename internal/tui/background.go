package tui

import (
	"math/rand/v2"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/life"
	"github.com/v4run/tcal/internal/runner"
)

// Background selects the ambient animation drawn behind the calendar.
type Background int

const (
	BackgroundNone Background = iota
	BackgroundLife
	BackgroundContainer
)

const (
	faintOn  = "\x1b[2m"
	faintOff = "\x1b[22m"
)

// box is the calendar's bounding box plus one cell of padding. Columns are
// half-open [x0, x1); rows are inclusive [y0, y1]. Nothing is drawn inside.
type box struct{ x0, x1, y0, y1 int }

// scene is an ambient animation drawn into the blank margins of the frame.
type scene interface {
	Resize(w, h int)
	Tick()
	Interval() time.Duration
	// Glyph returns the rune at screen (x, y), or 0 for empty. b is the
	// region the calendar occupies, for scenes that need to avoid it wholesale.
	Glyph(x, y int, b box) rune
}

func newScene(bg Background, w, h int) scene {
	seed := uint64(time.Now().UnixNano())
	rng := rand.New(rand.NewPCG(seed, seed>>32))
	switch bg {
	case BackgroundLife:
		s := &lifeScene{rng: rng}
		s.Resize(w, h)
		return s
	case BackgroundContainer:
		return &runnerScene{s: runner.New(w, h, rng), w: w, h: h}
	}
	return nil
}

// lifeScene runs Game of Life over the whole screen; a resize reseeds it.
type lifeScene struct {
	g   *life.Grid
	rng *rand.Rand
}

func (l *lifeScene) Resize(w, h int) {
	l.g = life.New(w, h, l.rng)
	l.g.Seed()
}

func (l *lifeScene) Tick()                   { l.g.Tick() }
func (l *lifeScene) Interval() time.Duration { return 250 * time.Millisecond }

func (l *lifeScene) Glyph(x, y int, _ box) rune {
	if l.g.Alive(x, y) {
		return '·'
	}
	return 0
}

// runnerScene runs the container along the bottom of the screen, hiding
// it entirely when the strip would overlap the calendar.
type runnerScene struct {
	s    *runner.Scene
	w, h int
}

func (d *runnerScene) Resize(w, h int) {
	d.w, d.h = w, h
	d.s.Resize(w, h)
}

func (d *runnerScene) Tick()                   { d.s.Tick() }
func (d *runnerScene) Interval() time.Duration { return 40 * time.Millisecond }

func (d *runnerScene) Glyph(x, y int, b box) rune {
	if d.w < runner.MinWidth || b.y1 >= d.h-runner.Height {
		return 0
	}
	return d.s.Glyph(x, y)
}

// overlay draws sc into the blank margins of frame, leaving the calendar's
// bounding box (plus one cell of padding) untouched. frame must be exactly
// w × h (as produced by render.Fit); otherwise it is returned unchanged.
func overlay(frame string, sc scene, w, h int, color bool) string {
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
		empty := box{y0: -1, y1: -1}
		for y := range lines {
			lines[y] = cells(sc, y, 0, w, empty, color)
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
	b := box{x0: max(x0-1, 0), x1: min(x1+1, w), y0: y0 - 1, y1: y1 + 1}

	for y, ln := range lines {
		if y < b.y0 || y > b.y1 {
			lines[y] = cells(sc, y, 0, w, b, color)
			continue
		}
		lines[y] = cells(sc, y, 0, b.x0, b, color) + ln[b.x0:len(ln)-(w-b.x1)] + cells(sc, y, b.x1, w, b, color)
	}
	return strings.Join(lines, "\n")
}

// cells renders row y of sc over columns [from, to) as glyphs and spaces.
func cells(sc scene, y, from, to int, b box, color bool) string {
	if from >= to {
		return ""
	}
	var sb strings.Builder
	if color {
		sb.WriteString(faintOn)
	}
	for x := from; x < to; x++ {
		if r := sc.Glyph(x, y, b); r != 0 {
			sb.WriteRune(r)
		} else {
			sb.WriteByte(' ')
		}
	}
	if color {
		sb.WriteString(faintOff)
	}
	return sb.String()
}

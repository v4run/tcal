package tui

import (
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/dino"
	"github.com/v4run/tcal/internal/life"
	"github.com/v4run/tcal/internal/render"
)

func fullLife(w, h int) scene {
	g := life.New(w, h, rand.New(rand.NewPCG(1, 2)))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			g.Set(x, y, true)
		}
	}
	return &lifeScene{g: g}
}

func testFrame(width, height int) string {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	m.state.Width, m.state.Height = width, height
	m.opts.WithClock = true
	m.opts.Color = true
	return render.Frame(m.state, m.opts)
}

func runeGrid(s string) [][]rune {
	var out [][]rune
	for _, ln := range strings.Split(ansi.Strip(s), "\n") {
		out = append(out, []rune(ln))
	}
	return out
}

func TestOverlay_NeverTouchesCalendarRegion(t *testing.T) {
	const w, h = 100, 30
	frame := testFrame(w, h)
	got := overlay(frame, fullLife(w, h), w, h, true)

	before, after := runeGrid(frame), runeGrid(got)
	if len(after) != h {
		t.Fatalf("got %d lines, want %d", len(after), h)
	}

	// Bounding box of the original content.
	y0, y1, x0, x1 := h, -1, w, -1
	for y, row := range before {
		for x, r := range row {
			if r != ' ' {
				y0, y1 = min(y0, y), max(y1, y)
				x0, x1 = min(x0, x), max(x1, x)
			}
		}
	}

	dots := 0
	for y, row := range after {
		if len(row) != w {
			t.Fatalf("line %d has width %d, want %d", y, len(row), w)
		}
		for x, r := range row {
			inBox := y >= y0-1 && y <= y1+1 && x >= x0-1 && x <= x1+1
			switch {
			case inBox && r != before[y][x]:
				t.Fatalf("cell (%d,%d) inside calendar box changed: %q -> %q", x, y, before[y][x], r)
			case !inBox && r == '·':
				dots++
			}
		}
	}
	if dots == 0 {
		t.Error("expected live cells in the margins, found none")
	}
}

func TestOverlay_NoColorEmitsNoEscapes(t *testing.T) {
	const w, h = 100, 30
	got := overlay(testFrame(w, h), fullLife(w, h), w, h, false)
	if strings.Contains(got, "\x1b[2m") {
		t.Error("faint escape emitted with color disabled")
	}
	if !strings.Contains(got, "·") {
		t.Error("expected plain live-cell glyphs")
	}
}

func TestOverlay_LeavesMismatchedFrameAlone(t *testing.T) {
	frame := testFrame(30, 30) // too narrow: hint line, no centering
	if got := overlay(frame, fullLife(30, 30), 30, 30, true); got != frame {
		t.Error("expected unpadded frame to be returned unchanged")
	}
}

func dinoModel(w, h int) Model {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	m.state.Width, m.state.Height = w, h
	m.opts.WithClock = true
	m.opts.Color = true
	return m
}

func dinoScreen(w, h int) [][]rune {
	m := dinoModel(w, h).WithBackground(BackgroundDino)
	for i := 0; i < 60; i++ {
		m.bg.Tick()
	}
	return runeGrid(m.View())
}

func TestDino_DrawsGroundBelowCalendar(t *testing.T) {
	const w, h = 100, 40
	got := dinoScreen(w, h)
	ground := string(got[h-1])
	if strings.Count(ground, "▔")+strings.Count(ground, "▀") != w {
		t.Errorf("bottom row is not a full ground line: %q", ground)
	}
	for y := 0; y < h-dino.Height; y++ {
		if strings.ContainsAny(string(got[y]), "▔▄") {
			t.Fatalf("scene glyph above the strip on row %d: %q", y, string(got[y]))
		}
	}
}

func TestDino_HiddenWhenNoRoomBelowCalendar(t *testing.T) {
	const w, h = 100, 24 // calendar + block clock fit, but not the dino strip
	m := dinoModel(w, h)
	plain := runeGrid(render.Frame(m.state, m.opts))
	if len(plain) != h {
		t.Fatalf("frame has %d lines, want %d: test must exercise the overlay path", len(plain), h)
	}
	got := dinoScreen(w, h)
	for y := range got {
		if string(got[y]) != string(plain[y]) {
			t.Fatalf("row %d changed: %q", y, string(got[y]))
		}
	}
}

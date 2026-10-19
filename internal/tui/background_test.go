package tui

import (
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/life"
	"github.com/v4run/tcal/internal/render"
	"github.com/v4run/tcal/internal/runner"
	"github.com/v4run/tcal/internal/scene"
)

func fullLife(w, h int) scene.Scene {
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
	got := overlay(frame, fullLife(w, h), w, h, styleColor)

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
	got := overlay(testFrame(w, h), fullLife(w, h), w, h, stylePlain)
	if strings.Contains(got, "\x1b[2m") {
		t.Error("faint escape emitted with color disabled")
	}
	if !strings.Contains(got, "·") {
		t.Error("expected plain live-cell glyphs")
	}
}

func TestOverlay_LeavesMismatchedFrameAlone(t *testing.T) {
	frame := testFrame(30, 30) // too narrow: hint line, no centering
	if got := overlay(frame, fullLife(30, 30), 30, 30, styleColor); got != frame {
		t.Error("expected unpadded frame to be returned unchanged")
	}
}

func runnerModel(w, h int) Model {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	m.state.Width, m.state.Height = w, h
	m.opts.WithClock = true
	m.opts.Color = true
	return m
}

func runnerScreen(w, h int) [][]rune {
	m := runnerModel(w, h).WithBackground("container")
	for i := 0; i < 60; i++ {
		m.bg.Tick()
	}
	return runeGrid(m.View())
}

func TestRunner_DrawsGroundBelowCalendar(t *testing.T) {
	const w, h = 100, 40
	got := runnerScreen(w, h)
	ground := string(got[h-1])
	if strings.Count(ground, "▔")+strings.Count(ground, "▀") != w {
		t.Errorf("bottom row is not a full ground line: %q", ground)
	}
	for y := 0; y < h-runner.Height; y++ {
		if strings.ContainsAny(string(got[y]), "▔┌╭") {
			t.Fatalf("scene glyph above the strip on row %d: %q", y, string(got[y]))
		}
	}
}

func TestRunner_HiddenWhenNoRoomBelowCalendar(t *testing.T) {
	const w, h = 100, 24 // calendar + block clock fit, but not the runner strip
	m := runnerModel(w, h)
	plain := runeGrid(render.Frame(m.state, m.opts))
	if len(plain) != h {
		t.Fatalf("frame has %d lines, want %d: test must exercise the overlay path", len(plain), h)
	}
	got := runnerScreen(w, h)
	for y := range got {
		if string(got[y]) != string(plain[y]) {
			t.Fatalf("row %d changed: %q", y, string(got[y]))
		}
	}
}

// Every registered background must leave the calendar's padded box intact,
// at several terminal sizes, across many ticks.
func TestAllBackgrounds_NeverTouchCalendar(t *testing.T) {
	sizes := [][2]int{{100, 40}, {80, 24}, {160, 50}}
	for _, name := range Backgrounds() {
		for _, sz := range sizes {
			w, h := sz[0], sz[1]
			m := runnerModel(w, h)
			plain := runeGrid(render.Frame(m.state, m.opts))
			if len(plain) != h {
				continue // frame does not fit; overlay is a no-op
			}
			y0, y1, x0, x1 := h, -1, w, -1
			for y, row := range plain {
				for x, r := range row {
					if r != ' ' {
						y0, y1 = min(y0, y), max(y1, y)
						x0, x1 = min(x0, x), max(x1, x)
					}
				}
			}
			m = m.WithBackground(name)
			for i := 0; i < 120; i++ {
				next, _ := m.Update(bgTickMsg{})
				m = next.(Model)
				if i%20 != 0 {
					continue
				}
				got := runeGrid(m.View())
				for y := max(y0-1, 0); y <= min(y1+1, h-1); y++ {
					for x := max(x0-1, 0); x <= min(x1+1, w-1); x++ {
						if got[y][x] != plain[y][x] {
							t.Fatalf("%s %dx%d tick %d: cell (%d,%d) changed %q -> %q", name, w, h, i, x, y, plain[y][x], got[y][x])
						}
					}
				}
				for y, row := range got {
					if len(row) != w {
						t.Fatalf("%s %dx%d: row %d width %d", name, w, h, y, len(row))
					}
				}
			}
		}
	}
}

func TestOverlay_ColorStyleTintsGlyphs(t *testing.T) {
	const w, h = 100, 30
	got := overlay(testFrame(w, h), fullLife(w, h), w, h, styleColor)
	if !strings.Contains(got, faintOn+sgrColor(scene.Green)+"·") {
		t.Error("expected dim green life cells")
	}
}

func TestOverlay_FaintStyleIgnoresSceneColors(t *testing.T) {
	const w, h = 100, 30
	got := overlay(testFrame(w, h), fullLife(w, h), w, h, styleFaint)
	if strings.Contains(got, sgrColor(scene.Green)) {
		t.Error("mono background emitted a scene color")
	}
	if !strings.Contains(got, faintOn) {
		t.Error("mono background is not dim")
	}
}

func TestOverlay_ColorResetsAtSegmentEnd(t *testing.T) {
	const w, h = 100, 30
	got := overlay(testFrame(w, h), fullLife(w, h), w, h, styleColor)
	for i, ln := range strings.Split(got, "\n") {
		if strings.Contains(ln, sgrColor(scene.Green)) && !strings.Contains(ln, fgReset+faintOff) {
			t.Fatalf("line %d leaves the color set", i)
		}
	}
}

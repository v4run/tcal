package tui

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/eventart"
	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/life"
	"github.com/v4run/tcal/internal/palette"
	"github.com/v4run/tcal/internal/render"
	"github.com/v4run/tcal/internal/runner"
	"github.com/v4run/tcal/internal/scene"
	"github.com/v4run/tcal/internal/scenes/lava"
	"github.com/v4run/tcal/internal/scenes/rain"
	"github.com/v4run/tcal/internal/scenes/seasons"
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
	got := overlay(frame, fullLife(w, h), w, h, bgPaint{style: styleColor})

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
	got := overlay(testFrame(w, h), fullLife(w, h), w, h, bgPaint{style: stylePlain})
	if strings.Contains(got, "\x1b[2m") {
		t.Error("faint escape emitted with color disabled")
	}
	if !strings.Contains(got, "·") {
		t.Error("expected plain live-cell glyphs")
	}
}

func TestOverlay_LeavesMismatchedFrameAlone(t *testing.T) {
	frame := testFrame(30, 30) // too narrow: hint line, no centering
	if got := overlay(frame, fullLife(30, 30), 30, 30, bgPaint{style: styleColor}); got != frame {
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
	m := runnerModel(w, h).WithBackground("container", BackgroundOptions{})
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

// assertCalendarIntact fails t when any cell of plain's padded content box
// differs in got, or when a row of got is not plain's width.
func assertCalendarIntact(t *testing.T, label string, plain, got [][]rune) {
	t.Helper()
	h, w := len(plain), len(plain[0])
	y0, y1, x0, x1 := h, -1, w, -1
	for y, row := range plain {
		for x, r := range row {
			if r != ' ' {
				y0, y1 = min(y0, y), max(y1, y)
				x0, x1 = min(x0, x), max(x1, x)
			}
		}
	}
	for y := max(y0-1, 0); y <= min(y1+1, h-1); y++ {
		for x := max(x0-1, 0); x <= min(x1+1, w-1); x++ {
			if got[y][x] != plain[y][x] {
				t.Fatalf("%s: cell (%d,%d) changed %q -> %q", label, x, y, plain[y][x], got[y][x])
			}
		}
	}
	for y, row := range got {
		if len(row) != w {
			t.Fatalf("%s: row %d width %d, want %d", label, y, len(row), w)
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
			m = m.WithBackground(name, BackgroundOptions{})
			for i := 0; i < 120; i++ {
				next, _ := m.Update(bgTickMsg(time.Time{}))
				m = next.(Model)
				if i%20 != 0 {
					continue
				}
				assertCalendarIntact(t, fmt.Sprintf("%s %dx%d tick %d", name, w, h, i), plain, runeGrid(m.View()))
			}
		}
	}
}

// Every event with art, shown in seasons, leaves the calendar's padded box
// intact.
func TestAllEvents_NeverTouchCalendar(t *testing.T) {
	const w, h = 100, 40
	bday := &events.MonthDay{Month: time.May, Day: 17}
	hasArt := map[events.Kind]bool{}
	for _, k := range eventart.Kinds() {
		hasArt[k] = true
	}
	tested := 0
	for _, occ := range events.Occurrences(2026, bday) {
		if !hasArt[occ.Kind] {
			continue
		}
		tested++
		m := localModel(w, h)
		plain := runeGrid(render.Frame(m.state, m.opts))
		if len(plain) != h {
			t.Fatalf("frame has %d lines, want %d", len(plain), h)
		}
		m = m.WithBackgroundTime(occ.Start.Add(time.Hour)).
			WithBackground("seasons", BackgroundOptions{Events: true, Birthday: bday})
		if k, ok := m.bg.(*seasons.Scene).Event(); !ok || k != occ.Kind {
			t.Fatalf("%v: active event %v, %v", occ.Kind, k, ok)
		}
		for i := 0; i < 100; i++ {
			next, _ := m.Update(bgTickMsg(time.Time{}))
			m = next.(Model)
		}
		assertCalendarIntact(t, occ.Kind.String(), plain, runeGrid(m.View()))
	}
	if tested == 0 {
		t.Fatal("no event with art was tested")
	}
}

// localModel is runnerModel with its clock in the local zone, as in the
// real app, so events are judged by local dates.
func localModel(w, h int) Model {
	m := runnerModel(w, h)
	m.state.Now = m.state.Now.Local()
	return m
}

func pick(t *testing.T, o BackgroundOptions, bgTime time.Time) scene.Scene {
	t.Helper()
	m := localModel(100, 40).WithBackgroundTime(bgTime).WithBackground(Random, o)
	if m.bg == nil {
		t.Fatal("random picked nothing")
	}
	return m.bg
}

func TestRandom_PoolOnly(t *testing.T) {
	o := BackgroundOptions{RandomPool: []string{"rain", "lava"}}
	day := time.Date(2026, time.April, 1, 12, 0, 0, 0, time.Local)
	for range 200 {
		switch bg := pick(t, o, day).(type) {
		case *rain.Scene, *lava.Scene:
		default:
			t.Fatalf("picked %T outside the pool", bg)
		}
	}
}

func TestRandom_EventDayPicksSeasons(t *testing.T) {
	o := BackgroundOptions{Events: true, RandomPool: []string{"rain", "seasons"}}
	xmas := time.Date(2026, time.December, 25, 12, 0, 0, 0, time.Local)
	for range 50 {
		if bg, ok := pick(t, o, xmas).(*seasons.Scene); !ok {
			t.Fatalf("picked %T on an event day, want seasons", bg)
		}
	}
}

func TestRandom_EventDayWithoutSeasonsInPool(t *testing.T) {
	o := BackgroundOptions{Events: true, RandomPool: []string{"rain"}}
	xmas := time.Date(2026, time.December, 25, 12, 0, 0, 0, time.Local)
	if bg, ok := pick(t, o, xmas).(*rain.Scene); !ok {
		t.Fatalf("picked %T, want rain", bg)
	}
}

func TestOverlay_ColorStyleTintsGlyphs(t *testing.T) {
	const w, h = 100, 30
	got := overlay(testFrame(w, h), fullLife(w, h), w, h, bgPaint{style: styleColor})
	if !strings.Contains(got, faintOn+sgrColor(scene.Green)+"·") {
		t.Error("expected dim green life cells")
	}
}

func TestOverlay_FaintStyleIgnoresSceneColors(t *testing.T) {
	const w, h = 100, 30
	got := overlay(testFrame(w, h), fullLife(w, h), w, h, bgPaint{style: styleFaint})
	if strings.Contains(got, sgrColor(scene.Green)) {
		t.Error("mono background emitted a scene color")
	}
	if !strings.Contains(got, faintOn) {
		t.Error("mono background is not dim")
	}
}

func TestOverlay_ColorResetsAtSegmentEnd(t *testing.T) {
	const w, h = 100, 30
	got := overlay(testFrame(w, h), fullLife(w, h), w, h, bgPaint{style: styleColor})
	for i, ln := range strings.Split(got, "\n") {
		if strings.Contains(ln, sgrColor(scene.Green)) && !strings.Contains(ln, fgReset+faintOff) {
			t.Fatalf("line %d leaves the color set", i)
		}
	}
}

// Every registered background is tinted, and colored rows never leak a
// color past the end of a background segment into the calendar.
func TestAllBackgrounds_ColorWellFormed(t *testing.T) {
	for _, name := range Backgrounds() {
		m := runnerModel(100, 40)
		m = m.WithBackground(name, BackgroundOptions{})
		if _, ok := m.bg.(scene.Colored); !ok {
			t.Errorf("%s does not implement scene.Colored", name)
			continue
		}
		tinted := false
		for i := 0; i < 200; i++ {
			next, _ := m.Update(bgTickMsg(time.Time{}))
			m = next.(Model)
			if i%25 != 0 {
				continue
			}
			assertColorWellFormed(t, fmt.Sprintf("%s tick %d", name, i), m.View())
			// Check tinting on the scene itself: what is visible depends on
			// where things happen to be relative to the calendar.
			colored := m.bg.(scene.Colored)
			for y := 0; y < 40 && !tinted; y++ {
				for x := 0; x < 100; x++ {
					if m.bg.Glyph(x, y, scene.Box{Y0: -1, Y1: -1}) != 0 && colored.Color(x, y, scene.Box{Y0: -1, Y1: -1}) != scene.Default {
						tinted = true
						break
					}
				}
			}
		}
		if !tinted {
			t.Errorf("%s never emitted a color", name)
		}
	}
}

func TestSeasons_EventSwitchesAtMidnight(t *testing.T) {
	m := localModel(100, 40)
	m.state.Now = time.Date(2026, time.November, 30, 23, 59, 0, 0, time.Local)
	m = m.WithBackground("seasons", BackgroundOptions{Events: true})
	sc := m.bg.(*seasons.Scene)
	if k, ok := sc.Event(); ok {
		t.Fatalf("Nov 30 23:59: Event() = %v, true; want none", k)
	}
	next, _ := m.Update(tickMsg(time.Date(2026, time.December, 1, 0, 0, 1, 0, time.Local)))
	m = next.(Model)
	if k, ok := m.bg.(*seasons.Scene).Event(); !ok || k != events.Christmas {
		t.Errorf("Dec 1 00:00:01: Event() = %v, %v; want christmas, true", k, ok)
	}
}

// hueScene is a true-colour scene of one fixed colour.
type hueScene struct{ c scene.RGB }

func (hueScene) Resize(int, int)                     {}
func (hueScene) Tick()                               {}
func (hueScene) Interval() time.Duration             { return time.Second }
func (hueScene) Glyph(int, int, scene.Box) rune      { return 'x' }
func (h hueScene) RGB(int, int, scene.Box) scene.RGB { return h.c }

func TestCells_ANSIFallbackKeepsHue(t *testing.T) {
	sc := hueScene{scene.RGB{R: 200, G: 120, B: 120}}
	got := cells(sc, 0, 0, 1, scene.Box{Y0: -1, Y1: -1}, bgPaint{style: styleColor, mode: palette.ANSI, bg: palette.Light})
	if !strings.Contains(got, "31m") {
		t.Errorf("got %q; want red (31m)", got)
	}
}

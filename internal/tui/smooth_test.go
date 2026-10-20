package tui

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/eventart"
	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/palette"
	"github.com/v4run/tcal/internal/render"
	"github.com/v4run/tcal/internal/scene"
	"github.com/v4run/tcal/internal/scenes/rain"
	"github.com/v4run/tcal/internal/scenes/starfield"
)

var smoothOpts = BackgroundOptions{Style: StyleSmooth}

// frameDT is one frame at about 30 fps.
const frameDT = 33 * time.Millisecond

// advance moves sc forward by dt: Step for a Stepper, one Tick otherwise.
func advance(sc scene.Scene, dt time.Duration) {
	if s, ok := sc.(scene.Stepper); ok {
		s.Step(dt)
		return
	}
	sc.Tick()
}

// stepModel sends n background ticks frameDT apart, starting after at, and
// returns the model and the time of the last tick.
func stepModel(m Model, at time.Time, n int) (Model, time.Time) {
	for range n {
		at = at.Add(frameDT)
		next, _ := m.Update(bgTickMsg(at))
		m = next.(Model)
	}
	return m, at
}

// fakeScene is a configurable test scene. It fills every cell with a braille
// glyph, records the dt of each Step, and draws in rgb when trueColor is set.
type fakeScene struct {
	rgb       scene.RGB
	steps     []time.Duration
	ticks     int
	interval  time.Duration
	trueColor bool
}

func (f *fakeScene) Resize(int, int)                {}
func (f *fakeScene) Tick()                          { f.ticks++ }
func (f *fakeScene) Interval() time.Duration        { return f.interval }
func (f *fakeScene) Glyph(int, int, scene.Box) rune { return '⣿' }

// stepperScene is a fakeScene that advances by elapsed time.
type stepperScene struct{ fakeScene }

func (s *stepperScene) Step(dt time.Duration) { s.steps = append(s.steps, dt) }

// rgbScene is a fakeScene that draws in true colour.
type rgbScene struct{ fakeScene }

func (r *rgbScene) RGB(int, int, scene.Box) scene.RGB { return r.rgb }

// dimViolation sweeps sc over w × h and returns the first colour that, blended
// toward bg by keep, contrasts with bg by more than 4.5. checked counts the
// coloured cells seen.
func dimViolation(sc scene.Scene, w, h int, bg scene.RGB, keep float64) (bad scene.RGB, ratio float64, checked int) {
	tc, ok := sc.(scene.TrueColored)
	if !ok {
		return scene.RGB{}, 0, 0
	}
	empty := scene.Box{Y0: -1, Y1: -1}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if sc.Glyph(x, y, empty) == 0 {
				continue
			}
			checked++
			c := palette.Blend(tc.RGB(x, y, empty), bg, keep)
			if r := palette.Contrast(c, bg); r > 4.5 {
				return c, r, checked
			}
		}
	}
	return scene.RGB{}, 0, checked
}

// The dimness check must be able to fail: full white on black is far too
// bright unblended, and passes once blended by keepFactor. The same holds for
// black on white.
func TestSmooth_DimCheckCanFail(t *testing.T) {
	cases := []struct {
		name string
		rgb  scene.RGB
		bg   scene.RGB
	}{
		{"white on dark", scene.RGB{R: 255, G: 255, B: 255}, palette.Dark},
		{"black on light", scene.RGB{}, palette.Light},
	}
	for _, tc := range cases {
		sc := &rgbScene{fakeScene{rgb: tc.rgb}}
		if _, r, n := dimViolation(sc, 4, 4, tc.bg, 1); r == 0 || n == 0 {
			t.Errorf("%s: unblended colour passed the dimness check", tc.name)
		}
		if c, r, n := dimViolation(sc, 4, 4, tc.bg, keepFactor); r != 0 || n == 0 {
			t.Errorf("%s: blended %v has contrast %.2f > 4.5", tc.name, c, r)
		}
	}
}

// keepFactor keeps every possible colour within the contrast limit on both
// reference backgrounds.
func TestKeepFactor_WorstCase(t *testing.T) {
	for _, bg := range []scene.RGB{palette.Dark, palette.Light} {
		for _, c := range []scene.RGB{{R: 255, G: 255, B: 255}, {}, {R: 255}, {G: 255}, {B: 255}, {R: 255, G: 255}, {G: 255, B: 255}, {R: 255, B: 255}} {
			if r := palette.Contrast(palette.Blend(c, bg, keepFactor), bg); r > 4.5 {
				t.Errorf("%v on %v: contrast %.2f > 4.5", c, bg, r)
			}
		}
	}
}

// TestSmooth_DimOnLightAndDark checks every true-colour smooth scene stays
// dim against both reference backgrounds, including every seasons event.
func TestSmooth_DimOnLightAndDark(t *testing.T) {
	const w, h = 120, 40
	type run struct {
		label string
		build func(m Model) Model
	}
	var runs []run
	for _, name := range Backgrounds() {
		runs = append(runs, run{name, func(m Model) Model { return m.WithBackground(name, smoothOpts) }})
	}
	bday := &events.MonthDay{Month: time.May, Day: 17}
	hasArt := map[events.Kind]bool{}
	for _, k := range eventart.Kinds() {
		hasArt[k] = true
	}
	for _, occ := range events.Occurrences(2026, bday) {
		if !hasArt[occ.Kind] {
			continue
		}
		runs = append(runs, run{"seasons/" + occ.Kind.String(), func(m Model) Model {
			o := smoothOpts
			o.Events, o.Birthday = true, bday
			return m.WithBackgroundTime(occ.Start.Add(time.Hour)).WithBackground("seasons", o)
		}})
	}

	trueColored := 0
	for _, r := range runs {
		for _, bg := range []scene.RGB{palette.Dark, palette.Light} {
			m := r.build(localModel(w, h).WithBackgroundPalette(palette.TrueColor, bg))
			if _, ok := m.bg.(scene.TrueColored); !ok {
				continue
			}
			trueColored++
			at := m.state.Now
			for i := range 100 {
				m, at = stepModel(m, at, 1)
				if c, ratio, _ := dimViolation(m.bg, w, h, bg, keepFactor); ratio != 0 {
					t.Fatalf("%s on %v step %d: %v has contrast %.2f > 4.5", r.label, bg, i, c, ratio)
				}
			}
		}
	}
	if trueColored == 0 {
		t.Fatal("no true-colour scenes were checked")
	}
	t.Logf("checked %d true-colour scene runs (classic fallbacks are skipped)", trueColored)
}

// sgrAny matches every foreground escape the overlay may emit.
var sgrAny = regexp.MustCompile(`\x1b\[(3[1-7]|39|38;2;\d+;\d+;\d+)m`)

// assertColorWellFormed fails when a faint background segment of view ends
// with a colour still set.
func assertColorWellFormed(t *testing.T, label, view string) {
	t.Helper()
	for y, ln := range strings.Split(view, "\n") {
		for _, seg := range strings.Split(ln, faintOff) {
			ss := sgrAny.FindAllStringSubmatch(seg, -1)
			if len(ss) > 0 && ss[len(ss)-1][1] != "39" && strings.Contains(seg, faintOn) {
				t.Fatalf("%s row %d: segment ends with colour %s set", label, y, ss[len(ss)-1][1])
			}
		}
	}
}

// assertSingleWidth fails unless view is h lines, each exactly w cells wide
// and made of single-width runes.
func assertSingleWidth(t *testing.T, label, view string, w, h int) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if len(lines) != h {
		t.Fatalf("%s: %d lines, want %d", label, len(lines), h)
	}
	for y, ln := range lines {
		plain := ansi.Strip(ln)
		if got := ansi.StringWidth(plain); got != w || len([]rune(plain)) != w {
			t.Fatalf("%s row %d: width %d (%d runes), want %d", label, y, got, len([]rune(plain)), w)
		}
	}
}

func TestSmooth_SharedRules(t *testing.T) {
	const w, h = 120, 40
	for _, name := range Backgrounds() {
		for _, mode := range []palette.Mode{palette.ANSI, palette.TrueColor} {
			label := fmt.Sprintf("%s mode %d", name, mode)
			m := runnerModel(w, h).WithBackgroundPalette(mode, palette.Dark)
			plain := runeGrid(render.Frame(m.state, m.opts))
			m = m.WithBackground(name, smoothOpts)
			if m.bg == nil {
				t.Fatalf("%s: no smooth scene", name)
			}
			at := m.state.Now
			for i := range 60 {
				m, at = stepModel(m, at, 1)
				if i%20 != 0 {
					continue
				}
				view := m.View()
				l := fmt.Sprintf("%s tick %d", label, i)
				assertCalendarIntact(t, l, plain, runeGrid(view))
				assertColorWellFormed(t, l, view)
				assertSingleWidth(t, l, view, w, h)
			}

			// Shrink to nothing and back mid-run.
			for _, sz := range [][2]int{{1, 1}, {0, 0}, {w, h}} {
				next, _ := m.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
				m = next.(Model)
				m, at = stepModel(m, at, 3)
				_ = m.View()
				empty := scene.Box{Y0: -1, Y1: -1}
				for y := 0; y < sz[1]; y++ {
					for x := 0; x < sz[0]; x++ {
						m.bg.Glyph(x, y, empty)
					}
				}
			}
			m, _ = stepModel(m, at, 10)
			view := m.View()
			assertCalendarIntact(t, label+" after resize", plain, runeGrid(view))
			assertColorWellFormed(t, label+" after resize", view)
			assertSingleWidth(t, label+" after resize", view, w, h)
		}
	}
}

func TestSmooth_FrameCost(t *testing.T) {
	if testing.Short() {
		t.Skip("timing test")
	}
	if raceEnabled {
		t.Skip("timing budget does not hold under the race detector")
	}
	const w, h, frames = 200, 60, 30
	empty := scene.Box{Y0: -1, Y1: -1}
	for _, name := range Backgrounds() {
		sc := newScene(name, w, h, smoothOpts, time.Date(2026, time.April, 15, 10, 0, 0, 0, time.Local))
		tc, _ := sc.(scene.TrueColored)
		col, _ := sc.(scene.Colored)
		start := time.Now()
		for range frames {
			advance(sc, frameDT)
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					if sc.Glyph(x, y, empty) == 0 {
						continue
					}
					switch {
					case tc != nil:
						tc.RGB(x, y, empty)
					case col != nil:
						col.Color(x, y, empty)
					}
				}
			}
		}
		if avg := time.Since(start) / frames; avg > 2*time.Millisecond {
			t.Errorf("%s: %v per frame, want ≤ 2ms", name, avg)
		}
	}
}

func TestStyle_DefaultIsClassic(t *testing.T) {
	m := runnerModel(100, 40).WithBackground("rain", BackgroundOptions{})
	if _, ok := m.bg.(*rain.Scene); !ok {
		t.Fatalf("got %T, want *rain.Scene", m.bg)
	}
}

func TestStyles(t *testing.T) {
	if got := Styles(); len(got) != 2 || got[0] != StyleClassic || got[1] != StyleSmooth {
		t.Errorf("Styles() = %v", got)
	}
}

// Names missing from the smooth registry fall back to their classic scene.
func TestSmooth_FallsBackToClassic(t *testing.T) {
	m := runnerModel(100, 40).WithBackground("starfield", smoothOpts)
	if _, ok := m.bg.(*starfield.Scene); !ok {
		t.Fatalf("got %T, want classic *starfield.Scene", m.bg)
	}
	for name := range smoothBackgrounds {
		if _, ok := backgrounds[name]; !ok {
			t.Errorf("smooth background %q has no classic counterpart", name)
		}
	}
}

func stepperModel(sc scene.Scene) Model {
	m := runnerModel(100, 40)
	m.bg = sc
	return m
}

func TestStep_UsesElapsedTime(t *testing.T) {
	sc := &stepperScene{fakeScene{interval: frameDT}}
	m := stepperModel(sc)
	t0 := time.Date(2026, 4, 15, 10, 0, 0, 0, time.Local)
	for _, at := range []time.Time{t0, t0.Add(50 * time.Millisecond)} {
		next, cmd := m.Update(bgTickMsg(at))
		m = next.(Model)
		if cmd == nil {
			t.Fatal("tick did not re-arm")
		}
	}
	if len(sc.steps) != 2 || sc.steps[0] != frameDT || sc.steps[1] != 50*time.Millisecond {
		t.Errorf("steps = %v, want [Interval 50ms]", sc.steps)
	}
	if sc.ticks != 0 {
		t.Errorf("Stepper was ticked %d times", sc.ticks)
	}
}

func TestStep_ClampsLongGap(t *testing.T) {
	sc := &stepperScene{fakeScene{interval: frameDT}}
	m := stepperModel(sc)
	t0 := time.Date(2026, 4, 15, 10, 0, 0, 0, time.Local)
	for _, at := range []time.Time{t0, t0.Add(10 * time.Minute), t0} {
		next, _ := m.Update(bgTickMsg(at))
		m = next.(Model)
	}
	if want := []time.Duration{frameDT, 100 * time.Millisecond, 0}; fmt.Sprint(sc.steps) != fmt.Sprint(want) {
		t.Errorf("steps = %v, want %v", sc.steps, want)
	}
}

func TestStep_NonStepperTicksOncePerMessage(t *testing.T) {
	sc := &fakeScene{interval: time.Second}
	m := stepperModel(sc)
	t0 := time.Date(2026, 4, 15, 10, 0, 0, 0, time.Local)
	for i := range 5 {
		next, _ := m.Update(bgTickMsg(t0.Add(time.Duration(i) * 10 * time.Minute)))
		m = next.(Model)
	}
	if sc.ticks != 5 {
		t.Errorf("ticks = %d, want 5", sc.ticks)
	}
}

func overlayModel(mode palette.Mode, color bool) Model {
	m := runnerModel(100, 30).WithBackgroundPalette(mode, palette.Dark)
	m.opts.Color = color
	m.bg = &rgbScene{fakeScene{rgb: scene.RGB{R: 200, G: 40, B: 40}}}
	return m
}

func TestOverlay_TrueColorEmits24Bit(t *testing.T) {
	got := overlayModel(palette.TrueColor, true).View()
	want := palette.Blend(scene.RGB{R: 200, G: 40, B: 40}, palette.Dark, keepFactor)
	if esc := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", want.R, want.G, want.B); !strings.Contains(got, faintOn+esc+"⣿") {
		t.Errorf("true colour mode did not emit the blended %q", esc)
	}
	assertColorWellFormed(t, "truecolor", got)

	got = overlayModel(palette.ANSI, true).View()
	if strings.Contains(got, "38;2;") {
		t.Error("ANSI mode emitted a 24-bit escape")
	}
	if !strings.Contains(got, faintOn+sgrColor(palette.Nearest(want))+"⣿") {
		t.Error("ANSI mode did not emit the nearest palette colour")
	}
	bg := got[strings.Index(got, faintOn):]
	bg = bg[:strings.Index(bg, faintOff)]
	for _, esc := range regexp.MustCompile(`\x1b\[[0-9;]*m`).FindAllString(bg, -1) {
		if !regexp.MustCompile(`^\x1b\[(2|3[0-7]|39)m$`).MatchString(esc) {
			t.Errorf("ANSI mode emitted %q", esc)
		}
	}
}

func TestOverlay_NoColorStillPlain(t *testing.T) {
	got := overlayModel(palette.TrueColor, false).View()
	if strings.Contains(got, "\x1b[") {
		t.Error("plain style emitted an escape")
	}
	if !strings.Contains(got, "⣿") {
		t.Error("expected plain glyphs")
	}
}

func TestOverlay_MonoIgnoresTrueColor(t *testing.T) {
	got := overlayModel(palette.TrueColor, true).WithBackgroundMono().View()
	if strings.Contains(got, "38;2;") {
		t.Error("mono background emitted a scene colour")
	}
}

func TestFrameCache_InvalidatesOnLayoutKey(t *testing.T) {
	m := New(runnerModel(100, 40).state, runnerModel(100, 40).opts)
	before := m.View()
	next, _ := m.Update(key('2'))
	m = next.(Model)
	got := m.View()
	if got == before {
		t.Fatal("view unchanged after switching layout")
	}
	if want := render.Frame(m.state, m.opts); got != want {
		t.Error("view after switching layout is not the grid frame")
	}
}

func TestFrameCache_SameStateReusesFrame(t *testing.T) {
	calls := 0
	orig := renderFrame
	renderFrame = func(s render.State, o render.Options) string { calls++; return orig(s, o) }
	t.Cleanup(func() { renderFrame = orig })

	base := runnerModel(100, 40)
	m := New(base.state, base.opts).WithBackground("rain", BackgroundOptions{})
	m.View()
	m, _ = stepModel(m, m.state.Now, 5)
	m.View()
	if calls != 1 {
		t.Fatalf("render.Frame called %d times for an unchanged state, want 1", calls)
	}
	// Sub-second clock changes reuse the frame; a new second does not.
	next, _ := m.Update(tickMsg(m.state.Now.Add(100 * time.Millisecond)))
	m = next.(Model)
	m.View()
	if calls != 1 {
		t.Fatalf("sub-second tick re-rendered: %d calls", calls)
	}
	for _, msg := range []tea.Msg{tickMsg(m.state.Now.Add(time.Second)), tea.WindowSizeMsg{Width: 110, Height: 40}, key('l')} {
		want := calls + 1
		next, _ := m.Update(msg)
		m = next.(Model)
		m.View()
		if calls != want {
			t.Errorf("%T: render.Frame calls %d, want %d", msg, calls, want)
		}
	}
}

package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/v4run/tcal/internal/render"
	"github.com/v4run/tcal/internal/scene"
)

func newTestModel(anchor time.Time) Model {
	return Model{
		state: render.State{
			Anchor: anchor,
			Today:  anchor,
			Now:    anchor,
			Layout: render.LayoutHorizontal,
			Width:  100,
			Height: 30,
		},
		opts: render.Options{
			WeekStart: time.Sunday,
			Highlight: render.HighlightBracket,
		},
	}
}

func key(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestUpdate_LMovesAnchorForwardOneMonth(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	next, _ := m.Update(key('l'))
	got := next.(Model).state.Anchor
	want := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUpdate_HMovesAnchorBackOneMonth(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	next, _ := m.Update(key('h'))
	got := next.(Model).state.Anchor
	want := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUpdate_KAddsOneYear(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	next, _ := m.Update(key('k'))
	got := next.(Model).state.Anchor.Year()
	if got != 2027 {
		t.Errorf("got %d, want 2027", got)
	}
}

func TestUpdate_JSubtractsOneYear(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	next, _ := m.Update(key('j'))
	got := next.(Model).state.Anchor.Year()
	if got != 2025 {
		t.Errorf("got %d, want 2025", got)
	}
}

func TestUpdate_TJumpsAnchorToToday(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	m.state.Today = time.Date(2027, 11, 15, 0, 0, 0, 0, time.UTC)
	next, _ := m.Update(key('t'))
	got := next.(Model).state.Anchor
	want := time.Date(2027, 11, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUpdate_LayoutKeysSwitchLayout(t *testing.T) {
	cases := []struct {
		key  rune
		want render.Layout
	}{
		{'1', render.LayoutHorizontal},
		{'2', render.LayoutGrid},
	}
	for _, tc := range cases {
		t.Run(string(tc.key), func(t *testing.T) {
			m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
			next, _ := m.Update(key(tc.key))
			if got := next.(Model).state.Layout; got != tc.want {
				t.Errorf("key %c: got %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestUpdate_QReturnsQuitCommand(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	_, cmd := m.Update(key('q'))
	if cmd == nil {
		t.Fatal("expected a tea.Cmd from q, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestUpdate_WindowSizeMsgUpdatesDimensions(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	next, _ := m.Update(tea.WindowSizeMsg{Width: 150, Height: 50})
	st := next.(Model).state
	if st.Width != 150 || st.Height != 50 {
		t.Errorf("got %dx%d, want 150x50", st.Width, st.Height)
	}
}

func TestUpdate_TickAdvancesNow(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 22, 14, 32, 7, 0, time.UTC))
	later := time.Date(2026, 4, 22, 14, 32, 8, 0, time.UTC)
	next, _ := m.Update(tickMsg(later))
	got := next.(Model).state.Now
	if !got.Equal(later) {
		t.Errorf("got %v, want %v", got, later)
	}
}

func TestWithLife_ResizeRebuildsGrid(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)).WithBackground("life", BackgroundOptions{})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	g := next.(Model).bg.(*lifeScene).g
	if g.Width() != 120 || g.Height() != 40 {
		t.Errorf("grid is %dx%d, want 120x40", g.Width(), g.Height())
	}
}

func TestUpdate_BackgroundTickRearms(t *testing.T) {
	for _, bg := range []string{"life", "container"} {
		m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)).WithBackground(bg, BackgroundOptions{})
		if _, cmd := m.Update(bgTickMsg{}); cmd == nil {
			t.Errorf("background %s: expected bgTickMsg to schedule the next tick", bg)
		}
	}
}

func TestView_WithoutBackgroundMatchesFrame(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	if got, want := m.View(), render.Frame(m.state, m.opts); got != want {
		t.Error("View differs from render.Frame with no background")
	}
}

func TestWithBackground_RandomPicksARegisteredScene(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)).WithBackground(Random, BackgroundOptions{})
	if m.bg == nil {
		t.Fatal("random background left the background off")
	}
}

func TestWithBackground_UnknownLeavesBackgroundOff(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)).WithBackground("nope", BackgroundOptions{})
	if m.bg != nil {
		t.Fatal("unknown background name enabled a scene")
	}
}

type clockSpy struct {
	now, shown time.Time
}

func (c *clockSpy) Resize(int, int)                {}
func (c *clockSpy) Tick()                          {}
func (c *clockSpy) Interval() time.Duration        { return time.Second }
func (c *clockSpy) Glyph(int, int, scene.Box) rune { return 0 }
func (c *clockSpy) SetTime(now, shown time.Time)   { c.now, c.shown = now, shown }

func TestWithBackgroundTime_ShiftsSceneClockOnly(t *testing.T) {
	real := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	m := newTestModel(real)
	spy := &clockSpy{}
	m.bg = spy
	forced := time.Date(2024, 4, 23, 22, 30, 0, 0, time.UTC)
	m = m.WithBackgroundTime(forced)
	if !spy.now.Equal(forced) {
		t.Fatalf("scene sees %v, want %v", spy.now, forced)
	}
	later := real.Add(10 * time.Second)
	next, _ := m.Update(tickMsg(later))
	if want := forced.Add(10 * time.Second); !spy.now.Equal(want) {
		t.Errorf("after tick scene sees %v, want %v", spy.now, want)
	}
	if !next.(Model).state.Now.Equal(later) {
		t.Error("calendar clock was shifted too")
	}
}

func TestBgStyle(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	m.opts.Color = true
	if got := m.bgStyle(); got != styleColor {
		t.Errorf("default style = %v, want color", got)
	}
	if got := m.WithBackgroundMono().bgStyle(); got != styleFaint {
		t.Errorf("mono style = %v, want faint", got)
	}
	m.opts.Color = false
	if got := m.bgStyle(); got != stylePlain {
		t.Errorf("no-color style = %v, want plain", got)
	}
}

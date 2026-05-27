package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/varun/tcal/internal/render"
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
		{'2', render.LayoutVertical},
		{'3', render.LayoutGrid},
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

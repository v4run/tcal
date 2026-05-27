package render

import (
	"strings"
	"testing"
	"time"

	"github.com/varun/tcal/internal/golden"
)

func TestFrame_Focus_April2026_Today22(t *testing.T) {
	state := State{
		Anchor: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Today:  time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
		Now:    time.Date(2026, 4, 22, 14, 32, 7, 0, time.UTC),
		Layout: LayoutFocus,
		Width:  100,
		Height: 30,
	}
	o := Options{
		WeekStart:  time.Sunday,
		Highlight:  HighlightBracket,
		ClockStyle: ClockBlock,
		Color:      false,
		WithClock:  true,
	}
	got := Frame(state, o)
	golden.Assert(t, "testdata/frame_focus_april2026_today22.golden", got)

	if !strings.Contains(got, "[22]") {
		t.Error("frame should include today highlight")
	}
}

func TestFrame_TooNarrow_PrependsHint(t *testing.T) {
	state := State{
		Anchor: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Today:  time.Time{},
		Now:    time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
		Layout: LayoutHorizontal,
		Width:  10, // way too narrow for 3 horizontal months
		Height: 0,
	}
	o := Options{WeekStart: time.Sunday, Highlight: HighlightNone}
	got := Frame(state, o)
	if !strings.HasPrefix(got, "terminal too narrow") {
		t.Errorf("expected hint prefix, got first line: %q", strings.SplitN(got, "\n", 2)[0])
	}
}

func TestFrame_PrintMode_NoClockByDefault(t *testing.T) {
	state := State{
		Anchor: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Now:    time.Date(2026, 4, 22, 14, 32, 7, 0, time.UTC),
		Layout: LayoutGrid,
		Width:  80,
		Height: 0, // print mode signal
	}
	o := Options{WeekStart: time.Sunday, WithClock: false}
	got := Frame(state, o)
	// The block clock contains '█'; its absence confirms WithClock=false dropped it.
	if strings.Contains(got, "█") {
		t.Error("WithClock=false should suppress the block clock")
	}
	// The textual print header still appears.
	if !strings.Contains(got, "2026") {
		t.Error("frame should still contain a date header")
	}
}

package render

import (
	"strings"
	"testing"
	"time"

	"github.com/v4run/tcal/internal/golden"
)

func TestFrame_Horizontal_April2026_Today22(t *testing.T) {
	state := State{
		Anchor: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Today:  time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
		Now:    time.Date(2026, 4, 22, 14, 32, 7, 0, time.UTC),
		Layout: LayoutHorizontal,
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
	golden.Assert(t, "testdata/frame_horizontal_april2026_today22.golden", got)

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

func gridState(w, h int) State {
	return State{
		Anchor: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Today:  time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
		Now:    time.Date(2026, 4, 22, 14, 32, 7, 0, time.UTC),
		Layout: LayoutGrid,
		Width:  w,
		Height: h,
	}
}

// gridShape returns how many month titles share the first title row and how
// many title rows there are.
func gridShape(frame string) (cols, rows int) {
	for _, l := range strings.Split(frame, "\n") {
		n := 0
		for _, m := range []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"} {
			if strings.Contains(l, m+" 2026") {
				n++
			}
		}
		if n > 0 {
			if rows == 0 {
				cols = n
			}
			rows++
		}
	}
	return cols, rows
}

func TestFrame_GridPicksWidestArrangementThatFits(t *testing.T) {
	o := Options{WeekStart: time.Sunday, WithClock: true, ClockStyle: ClockBlock}
	for _, tc := range []struct {
		w, h       int
		cols, rows int
		blockClock bool
	}{
		{160, 45, 6, 2, true},
		{120, 36, 4, 3, true},
		{80, 45, 3, 4, true},
		{50, 70, 2, 6, true},
		{120, 30, 4, 3, false}, // only fits under the one-line clock
	} {
		got := Frame(gridState(tc.w, tc.h), o)
		cols, rows := gridShape(got)
		if cols != tc.cols || rows != tc.rows {
			t.Errorf("%dx%d: got %d×%d, want %d×%d", tc.w, tc.h, cols, rows, tc.cols, tc.rows)
		}
		if lines := strings.Split(got, "\n"); len(lines) != tc.h {
			t.Errorf("%dx%d: %d lines, want the frame to fill the screen", tc.w, tc.h, len(lines))
		}
		if block := strings.Contains(got, "█"); block != tc.blockClock {
			t.Errorf("%dx%d: block clock shown = %v, want %v", tc.w, tc.h, block, tc.blockClock)
		}
	}
}

func TestFrame_GridTooSmallKeepsDefaultArrangement(t *testing.T) {
	o := Options{WeekStart: time.Sunday, WithClock: true, ClockStyle: ClockBlock}
	cols, rows := gridShape(Frame(gridState(70, 20), o))
	if cols != defaultGridCols || rows != 4 {
		t.Errorf("got %d×%d, want %d×4", cols, rows, defaultGridCols)
	}
}

func TestFrame_GridIsAlwaysClassic(t *testing.T) {
	o := Options{WeekStart: time.Sunday, WithClock: true, Look: LookThin}
	if got := Frame(gridState(400, 200), o); strings.Contains(got, "┏") {
		t.Error("the grid should never use the thin look")
	}
}

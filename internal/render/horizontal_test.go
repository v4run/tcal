package render

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func horizState(w, h int) State {
	return State{
		Anchor: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		Today:  time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC),
		Now:    time.Date(2026, 10, 21, 20, 10, 30, 0, time.UTC),
		Layout: LayoutHorizontal,
		Width:  w,
		Height: h,
	}
}

// shape describes a frame: which month titles show, whether day numbers are
// thin, whether the clock is the thin one and whether it is one line.
type shape struct {
	months   string // titles, in order, e.g. "Sep Oct Nov"
	thinDays bool
	oneLine  bool // the clock is a single line
	small    bool // some months are in the classic look
}

func shapeOf(frame string) shape {
	plain := ansi.Strip(frame)
	// Month titles carry the year; list them in calendar order.
	var titles []string
	for m := time.January; m <= time.December; m++ {
		if strings.Contains(plain, m.String()+" 2026") {
			titles = append(titles, m.String()[:3])
		}
	}
	return shape{
		months:   strings.Join(titles, " "),
		thinDays: strings.Contains(plain, "┏┛"),
		oneLine:  strings.Contains(plain, "·  20:10:30"),
		small:    strings.Contains(plain, "Su Mo Tu We Th Fr Sa"),
	}
}

func TestFrame_HorizontalThinStepsDown(t *testing.T) {
	o := Options{WeekStart: time.Sunday, Highlight: HighlightAll, WithClock: true, Look: LookThin}
	for _, tc := range []struct {
		w, h int
		want shape
		desc string
	}{
		{160, 45, shape{"Sep Oct Nov", true, false, false}, "three thin months"},
		{120, 36, shape{"Sep Oct Nov", true, false, true}, "thin October between small neighbours"},
		{70, 36, shape{"Oct", true, false, false}, "thin October alone"},
		{80, 24, shape{"Oct", true, true, false}, "thin October under a one-line clock"},
		{60, 20, shape{"Oct", false, false, true}, "classic October"},
	} {
		if got := shapeOf(Frame(horizState(tc.w, tc.h), o)); got != tc.want {
			t.Errorf("%dx%d (%s): got %+v, want %+v", tc.w, tc.h, tc.desc, got, tc.want)
		}
	}
}

func TestFrame_FocusDimsTheNeighbours(t *testing.T) {
	o := Options{WeekStart: time.Sunday, Highlight: HighlightAll, WithClock: true, Look: LookThin, Color: true}
	got := Frame(horizState(120, 36), o)
	if !strings.Contains(got, "\x1b[2m") || !strings.Contains(ansi.Strip(got), "September 2026") {
		t.Fatal("expected September drawn faint beside thin October")
	}
	// The big month is not faint: its digits never sit inside a faint run.
	for _, l := range strings.Split(got, "\n") {
		if i := strings.Index(l, "┏┛"); i >= 0 && strings.Contains(l[:i], "\x1b[2m") && !strings.Contains(l[:i], "\x1b[0m") {
			t.Errorf("thin digits drawn faint: %q", l)
		}
	}
}

func TestFrame_HorizontalClassicDropsMonthsBeforeHint(t *testing.T) {
	o := Options{WeekStart: time.Sunday, WithClock: true}
	if got := shapeOf(Frame(horizState(100, 30), o)); got.months != "Sep Oct Nov" {
		t.Errorf("100 cols: got %q, want three months", got.months)
	}
	got := Frame(horizState(40, 30), o)
	if strings.Contains(got, "too narrow") || shapeOf(got).months != "Oct" {
		t.Errorf("40 cols: want October alone and no hint, got %q", shapeOf(got).months)
	}
	if got := shapeOf(Frame(horizState(40, 30), Options{WeekStart: time.Sunday, WithClock: true, ClockStyle: ClockBlock})); got.oneLine {
		t.Error("40 cols: the block clock fits; it should not shrink")
	}
	got = Frame(horizState(15, 30), o)
	if !strings.HasPrefix(got, "terminal too narrow — needs ≥ 20 cols") {
		t.Errorf("15 cols: want the hint for one month, got %q", strings.SplitN(got, "\n", 2)[0])
	}
}

func TestFrame_FiveMonthsStepToThree(t *testing.T) {
	st := horizState(70, 30)
	st.Months = 5
	if got := shapeOf(Frame(st, Options{WeekStart: time.Sunday})); got.months != "Sep Oct Nov" {
		t.Errorf("got %q, want the three months around October", got.months)
	}
}

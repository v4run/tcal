package render

import (
	"strings"
	"testing"
	"time"

	"github.com/varun/tcal/internal/calendar"
	"github.com/varun/tcal/internal/golden"
)

func anchorApril2026() time.Time {
	return time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
}

func opts() Options {
	return Options{
		WeekStart: time.Sunday,
		Highlight: HighlightBracket,
		Color:     false,
	}
}

func TestLayout_Horizontal_3Months(t *testing.T) {
	o := opts()
	months := calendar.BuildMonths(anchorApril2026().AddDate(0, -1, 0), 3, o.WeekStart)
	got := RenderLayout(LayoutHorizontal, months, time.Time{}, o)
	golden.Assert(t, "testdata/layout_horizontal_3.golden", got)
	// Smoke check: at least one line should be wide enough for 3 grids + gutters.
	for _, ln := range strings.Split(got, "\n") {
		if len(ln) >= 3*20+2*2 {
			return
		}
	}
	t.Error("no line is wide enough for 3 horizontal months")
}

func TestLayout_Vertical_3Months(t *testing.T) {
	o := opts()
	months := calendar.BuildMonths(anchorApril2026().AddDate(0, -1, 0), 3, o.WeekStart)
	got := RenderLayout(LayoutVertical, months, time.Time{}, o)
	golden.Assert(t, "testdata/layout_vertical_3.golden", got)
	// Smoke check: every line should be <= one grid width (20).
	for _, ln := range strings.Split(got, "\n") {
		if visibleWidth(ln) > 22 { // grid 20 + small slop
			t.Errorf("vertical layout line too wide: %q", ln)
		}
	}
}

package render

import (
	"strings"
	"testing"
	"time"

	"github.com/v4run/tcal/internal/calendar"
	"github.com/v4run/tcal/internal/golden"
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

func TestLayout_Grid_12Months(t *testing.T) {
	o := opts()
	jan := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	months := calendar.BuildMonths(jan, 12, o.WeekStart)
	got := RenderLayout(LayoutGrid, months, time.Time{}, o)
	golden.Assert(t, "testdata/layout_grid_12.golden", got)

	// Sanity: should produce 4 rows of months (default cols=3 ⇒ ceil(12/3) = 4).
	// We can't directly assert row count without parsing, but we can assert the
	// height is at least 4 * (1 header + 1 weekday + 6 week rows) = 32 lines.
	lines := strings.Count(got, "\n") + 1
	if lines < 32 {
		t.Errorf("grid layout too short: %d lines", lines)
	}
}

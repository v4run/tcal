package render

import (
	"fmt"

	"github.com/v4run/tcal/internal/calendar"
)

func defaultMonths(layout Layout) int {
	switch layout {
	case LayoutGrid:
		return 12
	default:
		return 3
	}
}

func layoutStart(layout Layout, anchorYear int, anchorMonth int) (int, int) {
	if layout == LayoutGrid {
		return anchorYear, 1
	}
	// horizontal / vertical: anchor - 1 month
	y, m := anchorYear, anchorMonth-1
	if m < 1 {
		m = 12
		y--
	}
	return y, m
}

func minWidthFor(layout Layout, months int) int {
	const grid = 20
	const gutter = 2
	switch layout {
	case LayoutHorizontal:
		return months*grid + (months-1)*gutter
	case LayoutGrid:
		cols := 3
		return cols*grid + (cols-1)*gutter
	default: // vertical
		return grid
	}
}

// Frame is the top-level renderer used by both tui and printout.
func Frame(state State, opts Options) string {
	months := state.Months
	if months <= 0 {
		months = defaultMonths(state.Layout)
	}

	y, m := layoutStart(state.Layout, state.Anchor.Year(), int(state.Anchor.Month()))
	startTime := state.Anchor.AddDate(y-state.Anchor.Year(), m-int(state.Anchor.Month()), 0)
	monthData := calendar.BuildMonths(startTime, months, opts.WeekStart)

	body := RenderLayout(state.Layout, monthData, state.Today, opts)

	var header string
	if opts.WithClock {
		clock := RenderClock(state.Now, opts.ClockStyle)
		header = clock
	} else {
		// Single-line text header used in print mode by default.
		header = state.Now.Format("Mon, 02 Jan 2006 15:04")
	}

	block := header + "\n\n" + body

	// Width hint
	min := minWidthFor(state.Layout, months)
	if state.Width > 0 && state.Width < min {
		hint := fmt.Sprintf("terminal too narrow — needs ≥ %d cols", min)
		return hint + "\n" + block
	}

	width := state.Width
	if width <= 0 {
		width = min
	}
	return Fit(block, width, state.Height)
}

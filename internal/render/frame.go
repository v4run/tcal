package render

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
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

func layoutStart(layout Layout, months, anchorYear, anchorMonth int) (int, int) {
	if layout == LayoutGrid {
		return anchorYear, 1
	}
	if months == 1 {
		return anchorYear, anchorMonth
	}
	// horizontal: anchor - 1 month
	y, m := anchorYear, anchorMonth-1
	if m < 1 {
		m = 12
		y--
	}
	return y, m
}

func minWidthFor(layout Layout, months int) int {
	if layout == LayoutGrid {
		// The narrowest arrangement Frame falls back to.
		cols := gridCols[len(gridCols)-1]
		return cols*20 + (cols-1)*2
	}
	return blockWidth(months, 20)
}

// gridCols are the grid's arrangements, widest first: 6×2, 4×3, 3×4 and
// 2×6 for a year.
var gridCols = []int{6, 4, 3, 2}

// defaultGridCols is the arrangement used when nothing fits, or when the
// screen size is unknown.
const defaultGridCols = 3

// blockWidth is the width of months laid out side by side, each month
// grid cols wide.
func blockWidth(months, grid int) int {
	const gutter = 2
	return months*grid + (months-1)*gutter
}

// Frame is the top-level renderer used by both tui and printout.
func Frame(state State, opts Options) string {
	months := state.Months
	if months <= 0 {
		months = defaultMonths(state.Layout)
	}

	y, m := layoutStart(state.Layout, months, state.Anchor.Year(), int(state.Anchor.Month()))
	startTime := state.Anchor.AddDate(y-state.Anchor.Year(), m-int(state.Anchor.Month()), 0)
	monthData := calendar.BuildMonths(startTime, months, opts.WeekStart)

	var block string
	min := minWidthFor(state.Layout, months)
	if state.Layout == LayoutGrid {
		// The year view is an overview: it always uses the classic look.
		opts.Look = LookClassic
		block = gridBlock(state, opts, monthData)
	} else {
		var fits bool
		block, fits = horizontalBlock(state, opts, months, monthData)
		if fits {
			min = 0
		} else {
			min = minWidthFor(LayoutHorizontal, 1)
		}
	}

	// Width hint
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

// header is the clock in the chosen style, or a one-line date and time
// without the clock.
func header(state State, opts Options) string {
	if opts.WithClock {
		return RenderClock(state.Now, opts.ClockStyle)
	}
	// Single-line text header used in print mode by default.
	return state.Now.Format("Mon, 02 Jan 2006 15:04")
}

// gridBlock lays the months out in the widest arrangement that fits the
// screen. When none fits under a tall clock, it tries again with the
// one-line clock; when still none fits, or the screen size is unknown, it
// uses defaultGridCols under the chosen clock.
func gridBlock(state State, opts Options, monthData []calendar.Month) string {
	build := func(o Options, cols int) string {
		return header(state, o) + "\n\n" + RenderGrid(monthData, state.Today, cols, o)
	}
	if state.Width <= 0 {
		return build(opts, defaultGridCols)
	}
	tries := []Options{opts}
	if opts.WithClock && opts.ClockStyle != ClockInline {
		compact := opts
		compact.ClockStyle = ClockInline
		tries = append(tries, compact)
	}
	for _, o := range tries {
		for _, cols := range gridCols {
			if cols > len(monthData) {
				continue
			}
			b := build(o, cols)
			if lipgloss.Width(b) <= state.Width && (state.Height <= 0 || lipgloss.Height(b) <= state.Height) {
				return b
			}
		}
	}
	return build(opts, defaultGridCols)
}

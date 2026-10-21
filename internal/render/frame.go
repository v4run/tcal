package render

import (
	"fmt"
	"strings"

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
	return blockWidth(layout, months, 20)
}

// blockWidth is the width of the months laid out with each month grid
// cols wide.
func blockWidth(layout Layout, months, grid int) int {
	const gutter = 2
	switch layout {
	case LayoutHorizontal:
		return months*grid + (months-1)*gutter
	case LayoutGrid:
		cols := 3
		return cols*grid + (cols-1)*gutter
	default:
		return months*grid + (months-1)*gutter
	}
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

	if opts.Look == LookThin {
		if out, ok := thinFrame(state, opts, months, monthData); ok {
			return out
		}
		opts.Look = LookClassic
	}

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

// thinFrame renders the thin look. It reports false when the block would
// not fit the screen, so the caller can fall back to classic.
func thinFrame(state State, opts Options, months int, monthData []calendar.Month) (string, bool) {
	header := state.Now.Format("Mon, 02 Jan 2006 15:04")
	if opts.WithClock {
		header = RenderThinClock(state.Now, opts.Color)
	}
	body := RenderLayout(state.Layout, monthData, state.Today, opts)
	// Centre the header over the months; Fit centres the block as a whole.
	bw := lipgloss.Width(body)
	hw := lipgloss.Width(header)
	lines := strings.Split(header, "\n")
	for i, l := range lines {
		lines[i] = strings.Repeat(" ", max(0, (bw-hw)/2)) + l
	}
	block := strings.Join(lines, "\n") + "\n\n" + body

	width := state.Width
	if width <= 0 {
		width = blockWidth(state.Layout, months, thinMonthWidth)
	}
	if lipgloss.Width(block) > width {
		return "", false
	}
	if state.Height > 0 && lipgloss.Height(block) > state.Height {
		return "", false
	}
	return Fit(block, width, state.Height), true
}

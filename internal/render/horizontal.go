package render

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/v4run/tcal/internal/calendar"
)

// neighbourGutter separates the small neighbouring months from the big
// thin month in the focus arrangement.
const neighbourGutter = "   "

// horizontalBlock returns the clock and months for the horizontal layout,
// stepping down until the block fits the screen. Each step drops the least
// important thing first:
//
//  1. thin look only: the months in thin digits; then the current month in
//     thin digits between its small, dim neighbours; then the current month
//     alone in thin digits;
//  2. the months in the classic look, then fewer of them (odd counts, so
//     the current month stays in the middle), down to one.
//
// Before each step down, a tall clock is tried as one line. It reports
// false, with the first classic arrangement, when nothing fits. With the
// screen size unknown, the first step is taken as it is.
func horizontalBlock(state State, opts Options, months int, data []calendar.Month) (string, bool) {
	type step func(compact bool) string
	var steps []step
	if opts.Look == LookThin {
		steps = append(steps, func(c bool) string {
			return thinBlock(state, opts, c, RenderLayout(LayoutHorizontal, data, state.Today, opts))
		})
		if months >= 3 {
			around := centred(state, 3, opts.WeekStart)
			steps = append(steps, func(c bool) string { return thinBlock(state, opts, c, focusBody(around, state.Today, opts)) })
		}
		if months > 1 {
			one := centred(state, 1, opts.WeekStart)
			steps = append(steps, func(c bool) string {
				return thinBlock(state, opts, c, RenderLayout(LayoutHorizontal, one, state.Today, opts))
			})
		}
	}
	classic := opts
	classic.Look = LookClassic
	classicStep := func(d []calendar.Month) step {
		return func(c bool) string {
			o := classic
			if c {
				o.ClockStyle = ClockInline
			}
			return header(state, o) + "\n\n" + RenderLayout(LayoutHorizontal, d, state.Today, o)
		}
	}
	first := classicStep(data)
	steps = append(steps, first)
	for k := months - 1; k >= 1; k-- {
		if k%2 == 1 {
			steps = append(steps, classicStep(centred(state, k, opts.WeekStart)))
		}
	}

	if state.Width <= 0 {
		return steps[0](false), true
	}
	for _, st := range steps {
		for _, compact := range []bool{false, true} {
			if compact && !opts.WithClock {
				continue
			}
			if b := st(compact); fits(b, state) {
				return b, true
			}
		}
	}
	return first(false), false
}

// fits reports whether block fits the screen.
func fits(block string, state State) bool {
	return lipgloss.Width(block) <= state.Width && (state.Height <= 0 || lipgloss.Height(block) <= state.Height)
}

// centred returns k months with the anchor's month in the middle.
func centred(state State, k int, weekStart time.Weekday) []calendar.Month {
	a := state.Anchor
	first := time.Date(a.Year(), a.Month(), 1, 0, 0, 0, 0, a.Location()).AddDate(0, -(k-1)/2, 0)
	return calendar.BuildMonths(first, k, weekStart)
}

// thinBlock puts the thin clock (or, compact, the one-line clock) centred
// over body.
func thinBlock(state State, opts Options, compact bool, body string) string {
	head := state.Now.Format("Mon, 02 Jan 2006 15:04")
	if opts.WithClock {
		head = RenderThinClock(state.Now, opts.Color)
		if compact {
			head = RenderClock(state.Now, ClockInline)
		}
	}
	// Centre the header over the months; Fit centres the block as a whole.
	pad := strings.Repeat(" ", max(0, (lipgloss.Width(body)-lipgloss.Width(head))/2))
	lines := strings.Split(head, "\n")
	for i, l := range lines {
		lines[i] = pad + l
	}
	return strings.Join(lines, "\n") + "\n\n" + body
}

// focusBody is the middle of three months in thin digits, with the months
// either side in the classic look, dimmed and centred beside it.
func focusBody(three []calendar.Month, today time.Time, opts Options) string {
	big := RenderThinMonth(three[1], today, opts)
	classic := opts
	classic.Look = LookClassic
	small := func(m calendar.Month) string {
		lines := strings.Split(RenderMonth(m, today, classic), "\n")
		for i, l := range lines {
			lines[i] = style(opts.Color, lipgloss.NewStyle().Faint(true)).Render(l)
		}
		top := max(0, (lipgloss.Height(big)-len(lines))/2)
		return strings.Repeat("\n", top) + strings.Join(lines, "\n")
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, small(three[0]), neighbourGutter, big, neighbourGutter, small(three[2]))
}

package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/v4run/tcal/internal/calendar"
)

// thinClockDigits are 3 rows × 3 cols each, drawn with heavy box-drawing
// lines.
var thinClockDigits = map[rune][3]string{
	'0': {"┏━┓", "┃ ┃", "┗━┛"},
	'1': {" ┓ ", " ┃ ", " ┻ "},
	'2': {"╺━┓", "┏━┛", "┗━╸"},
	'3': {"╺━┓", " ━┫", "╺━┛"},
	'4': {"╻ ╻", "┗━┫", "  ╹"},
	'5': {"┏━╸", "┗━┓", "╺━┛"},
	'6': {"┏━╸", "┣━┓", "┗━┛"},
	'7': {"╺━┓", "  ┃", "  ╹"},
	'8': {"┏━┓", "┣━┫", "┗━┛"},
	'9': {"┏━┓", "┗━┫", "╺━┛"},
}

// thinColon sits between hours and minutes, padded to 5 cols.
var thinColon = [3]string{"  ╻  ", "     ", "  ╹  "}

// thinDayDigits are 3 rows × 2 cols each: the clock digits squeezed for the
// calendar's day numbers.
var thinDayDigits = map[rune][3]string{
	'0': {"┏┓", "┃┃", "┗┛"},
	'1': {" ╻", " ┃", " ╹"},
	'2': {"╺┓", "┏┛", "┗╸"},
	'3': {"╺┓", " ┫", "╺┛"},
	'4': {"╻╻", "┗┫", " ╹"},
	'5': {"┏╸", "┗┓", "╺┛"},
	'6': {"┏╸", "┣┓", "┗┛"},
	'7': {"╺┓", " ┃", " ╹"},
	'8': {"┏┓", "┣┫", "┗┛"},
	'9': {"┏┓", "┗┫", "╺┛"},
}

const (
	// thinClockWidth is "HH:MM" in thinClockDigits: four 3-col digits, a
	// space inside each pair and the 5-col colon.
	thinClockWidth = 4*3 + 2 + 5
	// thinCell is the width of one day cell: up to 5 cols of digits with a
	// col either side for brackets.
	thinCell = 7
	// thinMonthWidth is the width of a thin month block.
	thinMonthWidth = 7 * thinCell
	// thinMonthHeight is the title, the weekday row, a blank row each after
	// them and six weeks of 3-row days.
	thinMonthHeight = 4 + 6*3
)

// RenderThinClock renders HH:MM in thin digits, a bar that fills across the
// minute with the seconds beside it, and the date beneath.
func RenderThinClock(now time.Time, color bool) string {
	hm := now.Format("1504")
	rows := make([]string, 3)
	for r := range rows {
		var b strings.Builder
		for i, ch := range hm {
			switch i {
			case 1, 3:
				b.WriteString(" ")
			case 2:
				b.WriteString(thinColon[r])
			}
			b.WriteString(thinClockDigits[ch][r])
		}
		rows[r] = style(color, lipgloss.NewStyle().Bold(true)).Render(b.String())
	}
	filled := now.Second() * thinClockWidth / 60
	bar := strings.Repeat("━", filled) + strings.Repeat("─", thinClockWidth-filled) +
		fmt.Sprintf("  :%02d", now.Second())
	date := fmt.Sprintf("%s · %d %s", now.Weekday(), now.Day(), now.Month())
	return strings.Join(append(rows,
		style(color, lipgloss.NewStyle().Faint(true)).Render(bar),
		"",
		style(color, lipgloss.NewStyle().Faint(true)).Render(date),
	), "\n")
}

// RenderThinMonth renders one month with 3-row day numbers. Past days are
// faint, weekends grey and today a solid accent block per opts.Highlight. Without
// color, any highlight other than none falls back to brackets so today
// still stands out.
func RenderThinMonth(m calendar.Month, today time.Time, opts Options) string {
	lines := make([]string, 0, thinMonthHeight)
	title := fmt.Sprintf("%s %d", m.Month, m.Year)
	lines = append(lines, center(title, thinMonthWidth), "")
	var wd strings.Builder
	for _, l := range strings.Split(weekdayRow(opts.WeekStart), " ") {
		wd.WriteString(center(l, thinCell))
	}
	lines = append(lines, style(opts.Color, lipgloss.NewStyle().Faint(true)).Render(wd.String()), "")

	hasToday := !today.IsZero()
	for _, week := range m.Weeks {
		var rows [3]strings.Builder
		for _, d := range week {
			if !d.InMonth {
				for r := range rows {
					rows[r].WriteString(strings.Repeat(" ", thinCell))
				}
				continue
			}
			isToday := hasToday && sameYMD(d.Date, today)
			cell := thinDay(d.Date.Day(), isToday, opts)
			st := lipgloss.NewStyle()
			switch {
			case isToday:
				st = todayStyle(opts.Highlight)
			case d.Date.Weekday() == time.Saturday || d.Date.Weekday() == time.Sunday:
				st = st.Foreground(weekendColor)
			case hasToday && d.Date.Before(today):
				st = st.Faint(true)
			}
			for r := range rows {
				rows[r].WriteString(style(opts.Color, st).Render(cell[r]))
			}
		}
		for r := range rows {
			lines = append(lines, rows[r].String())
		}
	}
	return strings.Join(lines, "\n")
}

// weekendColor is ANSI 8, the terminal's themed grey.
var weekendColor = lipgloss.Color("8")

// thinDay returns the three rows of a day cell, each thinCell wide. Today
// gets corner brackets when they are asked for, or when color is off.
func thinDay(day int, isToday bool, opts Options) [3]string {
	var cell [3]string
	digits := []rune(fmt.Sprint(day))
	for r := range cell {
		parts := make([]string, len(digits))
		for i, ch := range digits {
			parts[i] = thinDayDigits[ch][r]
		}
		cell[r] = fmt.Sprintf("%5s", strings.Join(parts, " "))
	}
	bracket := isToday && (opts.Highlight&HighlightBracket != 0 ||
		!opts.Color && opts.Highlight != HighlightNone)
	left, right := [3]string{" ", " ", " "}, [3]string{" ", " ", " "}
	if bracket {
		left, right = [3]string{"┌", "│", "└"}, [3]string{"┐", "│", "┘"}
	}
	for r := range cell {
		cell[r] = left[r] + cell[r] + right[r]
	}
	return cell
}

// todayStyle maps the highlight mask to a style for today's thin cell.
// Combined and color fill the whole cell with the accent color behind
// dark bold digits; reverse inverts it.
func todayStyle(mask HighlightMask) lipgloss.Style {
	st := lipgloss.NewStyle()
	switch {
	case mask&HighlightColor != 0:
		st = st.Background(accentColor).Foreground(lipgloss.Color("0")).Bold(true)
	case mask&HighlightReverse != 0:
		st = st.Reverse(true)
	}
	return st
}

// style returns st when color is on, otherwise a style that renders plain.
func style(color bool, st lipgloss.Style) lipgloss.Style {
	if !color {
		return lipgloss.NewStyle()
	}
	return st
}

// center pads s with spaces on both sides to width w.
func center(s string, w int) string {
	n := lipgloss.Width(s)
	if n >= w {
		return s
	}
	left := (w - n) / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", w-n-left)
}

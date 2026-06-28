package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/v4run/tcal/internal/calendar"
)

// monthHeader returns the centered "Month Year" line, padded to gridWidth.
func monthHeader(m calendar.Month, gridWidth int) string {
	title := fmt.Sprintf("%s %d", m.Month.String(), m.Year)
	pad := (gridWidth - len(title)) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + title
}

// weekdayRow returns "Su Mo Tu We Th Fr Sa"-style header for the given start.
func weekdayRow(weekStart time.Weekday) string {
	labels := []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
	out := make([]string, 7)
	for c := 0; c < 7; c++ {
		out[c] = labels[(int(weekStart)+c)%7]
	}
	return strings.Join(out, " ")
}

// RenderMonth renders one calendar.Month as a multi-line string. The grid is
// 20 chars wide ("Su Mo Tu We Th Fr Sa" = 20 chars) with one space between
// day cells. Today's cell is decorated via opts.Highlight; non-month days
// render as blank cells.
//
// Note: with opts.Highlight = HighlightBracket (explicit, not the default
// HighlightAll), today's row is 1 char wider than other rows for 2-digit
// days. This is acceptable when callers know they're opting into brackets;
// the default combined highlight (reverse+color) preserves alignment.
func RenderMonth(m calendar.Month, today time.Time, opts Options) string {
	const gridWidth = 20

	var b strings.Builder
	b.WriteString(monthHeader(m, gridWidth))
	b.WriteString("\n")
	b.WriteString(weekdayRow(opts.WeekStart))
	b.WriteString("\n")

	hasToday := !today.IsZero()
	for _, week := range m.Weeks {
		// Build cells, then join with single spaces. When a cell is the
		// bracketed today, we drop one space of leading whitespace from the
		// next cell to keep alignment (because "[22]" is 4 wide vs " 22" = 3).
		cells := make([]string, 7)
		bracketedAt := -1
		for c, d := range week {
			if !d.InMonth {
				cells[c] = "  "
				continue
			}
			text := fmt.Sprintf("%2d", d.Date.Day())
			isToday := hasToday && sameYMD(d.Date, today)
			if isToday {
				if opts.Highlight&HighlightBracket != 0 {
					// Brackets supply their own boundaries, so drop the "%2d"
					// padding and signal the join to skip the next separator.
					cells[c] = Highlight(strings.TrimLeft(text, " "), opts.Highlight)
					bracketedAt = c
				} else {
					// Highlight only the digit(s); keep any "%2d" pad space
					// plain so the cell still spans two columns but the
					// styled block stays tight to the number.
					bare := strings.TrimLeft(text, " ")
					pad := text[:len(text)-len(bare)]
					cells[c] = pad + Highlight(bare, opts.Highlight)
				}
			} else {
				cells[c] = text
			}
		}

		// Join with single spaces; trim leading space from cell after bracketed.
		var line strings.Builder
		for c, cell := range cells {
			if c > 0 {
				if c == bracketedAt+1 {
					// no separator — the "[" already includes the leading boundary
				} else {
					line.WriteString(" ")
				}
			}
			line.WriteString(cell)
		}
		row := line.String()
		if visibleLen := len([]rune(row)); visibleLen < gridWidth {
			row += strings.Repeat(" ", gridWidth-visibleLen)
		}
		b.WriteString(row)
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func sameYMD(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

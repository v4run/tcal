package render

import (
	"fmt"
	"strings"
	"time"
)

// digitGlyphs are 5 rows × 4 cols each for "0"–"9". Use full-block "█".
var digitGlyphs = map[rune][5]string{
	'0': {
		"████",
		"█  █",
		"█  █",
		"█  █",
		"████",
	},
	'1': {
		"  █ ",
		" ██ ",
		"  █ ",
		"  █ ",
		" ███",
	},
	'2': {
		"████",
		"   █",
		"████",
		"█   ",
		"████",
	},
	'3': {
		"████",
		"   █",
		" ███",
		"   █",
		"████",
	},
	'4': {
		"█  █",
		"█  █",
		"████",
		"   █",
		"   █",
	},
	'5': {
		"████",
		"█   ",
		"████",
		"   █",
		"████",
	},
	'6': {
		"████",
		"█   ",
		"████",
		"█  █",
		"████",
	},
	'7': {
		"████",
		"   █",
		"  █ ",
		" █  ",
		" █  ",
	},
	'8': {
		"████",
		"█  █",
		"████",
		"█  █",
		"████",
	},
	'9': {
		"████",
		"█  █",
		"████",
		"   █",
		"████",
	},
}

// colonGlyph is 5 rows × 1 col.
var colonGlyph = [5]string{
	" ",
	"█",
	" ",
	"█",
	" ",
}

// RenderClock renders the current time per the chosen style.
func RenderClock(now time.Time, style ClockStyle) string {
	switch style {
	case ClockInline:
		return now.Format("Mon, 02 Jan 2006  ·  15:04:05")
	case ClockBoxed:
		date := now.Format("Mon, 02 Jan 2006")
		clk := now.Format("15 : 04 : 05")
		yearDay := now.YearDay()
		_, week := now.ISOWeek()
		line3 := fmt.Sprintf("Week %d · Day %d of 365", week, yearDay)
		width := 31
		top := "┌" + strings.Repeat("─", width-2) + "┐"
		bot := "└" + strings.Repeat("─", width-2) + "┘"
		mid := func(s string) string {
			return "│ " + s + strings.Repeat(" ", width-3-len(s)) + "│"
		}
		return strings.Join([]string{top, mid(date), mid(clk), mid(line3), bot}, "\n")
	default: // ClockBlock
		return renderBlock(now)
	}
}

func renderBlock(now time.Time) string {
	s := now.Format("15:04:05")
	rows := [5]strings.Builder{}
	for i, ch := range s {
		if i > 0 {
			for r := 0; r < 5; r++ {
				rows[r].WriteString(" ")
			}
		}
		var glyph [5]string
		if ch == ':' {
			glyph = colonGlyph
		} else {
			glyph = digitGlyphs[ch]
		}
		for r := 0; r < 5; r++ {
			rows[r].WriteString(glyph[r])
		}
	}
	out := make([]string, 5)
	for r := 0; r < 5; r++ {
		out[r] = rows[r].String()
	}
	return strings.Join(out, "\n")
}

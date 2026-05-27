package render

import "github.com/charmbracelet/lipgloss"

// HighlightMask is a bitmask of decorations applied to today's date.
type HighlightMask uint8

const (
	HighlightNone    HighlightMask = 0
	HighlightReverse HighlightMask = 1 << 0
	HighlightBracket HighlightMask = 1 << 1
	HighlightColor   HighlightMask = 1 << 2
	HighlightAll                   = HighlightReverse | HighlightColor
)

// accentColor is used when HighlightColor is set. Lipgloss degrades to the
// nearest supported color on limited terminals and emits no escape sequences
// when NO_COLOR is set or the writer is not a terminal.
var accentColor = lipgloss.Color("214") // soft orange

// Highlight wraps text (a day number string with no surrounding spaces) with
// the selected decorations and returns the resulting renderable string.
func Highlight(text string, mask HighlightMask) string {
	out := text
	if mask&HighlightBracket != 0 {
		out = "[" + out + "]"
	}
	style := lipgloss.NewStyle()
	if mask&HighlightReverse != 0 {
		style = style.Reverse(true).Bold(true)
	}
	if mask&HighlightColor != 0 {
		style = style.Foreground(accentColor)
	}
	if mask&(HighlightReverse|HighlightColor) != 0 {
		out = style.Render(out)
	}
	return out
}

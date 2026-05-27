package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Fit centers block (a newline-separated string) within (width, height).
// height=0 disables vertical centering (used by print mode).
// If the block is wider than width, content is returned unchanged (no truncation;
// callers may render a "too narrow" hint elsewhere). If taller than height,
// content is returned unchanged.
func Fit(block string, width, height int) string {
	lines := strings.Split(block, "\n")

	maxW := 0
	for _, ln := range lines {
		if n := visibleWidth(ln); n > maxW {
			maxW = n
		}
	}

	// Horizontal: only pad if the block fits.
	if maxW <= width {
		leftPad := (width - maxW) / 2
		for i, ln := range lines {
			rightPad := width - leftPad - visibleWidth(ln)
			if rightPad < 0 {
				rightPad = 0
			}
			lines[i] = strings.Repeat(" ", leftPad) + ln + strings.Repeat(" ", rightPad)
		}
	}

	// Vertical: only pad if the block fits AND height > 0.
	if height > 0 && len(lines) <= height {
		topPad := (height - len(lines)) / 2
		bottomPad := height - len(lines) - topPad
		// Use a blank line of full width so vertical padding lines are uniform.
		blank := strings.Repeat(" ", width)
		out := make([]string, 0, height)
		for i := 0; i < topPad; i++ {
			out = append(out, blank)
		}
		out = append(out, lines...)
		for i := 0; i < bottomPad; i++ {
			out = append(out, blank)
		}
		lines = out
	}

	return strings.Join(lines, "\n")
}

// visibleWidth returns the visible width of s, stripping ANSI escape sequences.
func visibleWidth(s string) int {
	return lipgloss.Width(s)
}

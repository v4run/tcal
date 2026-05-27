package render

import (
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/v4run/tcal/internal/calendar"
)

const monthGutter = "  " // two-space gutter between months in horizontal/grid

// RenderLayout joins month blocks per the selected layout. today may be the
// zero value to suppress highlighting.
func RenderLayout(layout Layout, months []calendar.Month, today time.Time, opts Options) string {
	blocks := make([]string, len(months))
	for i, m := range months {
		blocks[i] = RenderMonth(m, today, opts)
	}
	switch layout {
	case LayoutHorizontal:
		return joinHorizontal(blocks)
	case LayoutVertical:
		return joinVertical(blocks)
	case LayoutGrid:
		return joinGrid(blocks, 3)
	default:
		return joinVertical(blocks)
	}
}

// joinGrid arranges blocks in rows of cols, joined horizontally within each
// row and vertically across rows.
func joinGrid(blocks []string, cols int) string {
	if cols < 1 {
		cols = 1
	}
	rows := make([]string, 0)
	for i := 0; i < len(blocks); i += cols {
		end := i + cols
		if end > len(blocks) {
			end = len(blocks)
		}
		row := blocks[i:end]
		rows = append(rows, joinHorizontal(row))
	}
	withGutters := make([]string, 0, 2*len(rows)-1)
	for i, r := range rows {
		if i > 0 {
			withGutters = append(withGutters, "")
		}
		withGutters = append(withGutters, r)
	}
	return lipgloss.JoinVertical(lipgloss.Left, withGutters...)
}

func joinHorizontal(blocks []string) string {
	if len(blocks) == 0 {
		return ""
	}
	// lipgloss.JoinHorizontal aligns by top and pads short blocks.
	withGutters := make([]string, 0, 2*len(blocks)-1)
	for i, b := range blocks {
		if i > 0 {
			withGutters = append(withGutters, monthGutter)
		}
		withGutters = append(withGutters, b)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, withGutters...)
}

func joinVertical(blocks []string) string {
	if len(blocks) == 0 {
		return ""
	}
	withGutters := make([]string, 0, 2*len(blocks)-1)
	for i, b := range blocks {
		if i > 0 {
			withGutters = append(withGutters, "")
		}
		withGutters = append(withGutters, b)
	}
	return lipgloss.JoinVertical(lipgloss.Left, withGutters...)
}

package dots

import "github.com/v4run/tcal/internal/scene"

// Sprite is a small bitmap in dot space.
type Sprite struct {
	w, h int
	on   [][]bool
}

// ParseSprite builds a sprite from rows where '#' is a dot and any other
// character is empty. Rows may differ in length.
func ParseSprite(rows ...string) Sprite {
	s := Sprite{h: len(rows), on: make([][]bool, len(rows))}
	for y, row := range rows {
		line := make([]bool, 0, len(row))
		for _, r := range row {
			line = append(line, r == '#')
		}
		s.on[y] = line
		if len(line) > s.w {
			s.w = len(line)
		}
	}
	return s
}

// Size returns the sprite size in dots.
func (s Sprite) Size() (dw, dh int) { return s.w, s.h }

// DrawSprite draws s with its top-left corner at cell-unit (x, y).
func (c *Canvas) DrawSprite(s Sprite, x, y float64, col scene.RGB) {
	for dy, line := range s.on {
		for dx, on := range line {
			if on {
				c.Dot(x+float64(dx)/2, y+float64(dy)/4, col)
			}
		}
	}
}

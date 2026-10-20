// Package dots is a braille dot canvas: each terminal cell holds a 2x4 grid
// of dots, giving smooth sub-cell motion for animated backgrounds.
package dots

import (
	"math"

	"github.com/v4run/tcal/internal/scene"
)

// bit gives the braille bit for a dot at (column, row) within a cell.
var bit = [4][2]uint8{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

// Canvas is a w x h grid of cells. The zero value is a usable empty canvas.
type Canvas struct {
	w, h int
	mask []uint8     // braille dot bits per cell
	col  []scene.RGB // colour per cell
	txt  []rune      // non-zero when the cell holds ordinary text
}

// Reset resizes the canvas to w x h cells (negatives clamp to 0) and clears it.
func (c *Canvas) Reset(w, h int) {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	c.w, c.h = w, h
	n := w * h
	if cap(c.mask) < n {
		c.mask = make([]uint8, n)
		c.col = make([]scene.RGB, n)
		c.txt = make([]rune, n)
		return
	}
	c.mask, c.col, c.txt = c.mask[:n], c.col[:n], c.txt[:n]
	c.Clear()
}

// Clear empties every cell, reusing the buffers.
func (c *Canvas) Clear() {
	clear(c.mask)
	clear(c.col)
	clear(c.txt)
}

// Size reports the canvas size in cells.
func (c *Canvas) Size() (w, h int) { return c.w, c.h }

// Dot sets the dot at cell-unit position (x, y), which lands on dot column
// floor(x*2) and dot row floor(y*4). Off-canvas dots are ignored. The cell
// takes col.
func (c *Canvas) Dot(x, y float64, col scene.RGB) {
	fx, fy := math.Floor(x*2), math.Floor(y*4)
	// Compared as floats first so NaN and huge values never convert to int.
	if !(fx >= 0 && fx < float64(c.w*2) && fy >= 0 && fy < float64(c.h*4)) {
		return
	}
	dx, dy := int(fx), int(fy)
	i := (dy/4)*c.w + dx/2
	if c.txt[i] != 0 {
		c.txt[i] = 0
		c.mask[i] = 0
	}
	c.mask[i] |= bit[dy&3][dx&1]
	c.col[i] = col
}

// Glyph returns the rune for a cell, or 0 if it is empty or off-canvas.
func (c *Canvas) Glyph(cx, cy int) rune {
	if cx < 0 || cy < 0 || cx >= c.w || cy >= c.h {
		return 0
	}
	i := cy*c.w + cx
	if r := c.txt[i]; r != 0 {
		return r
	}
	if m := c.mask[i]; m != 0 {
		return 0x2800 | rune(m)
	}
	return 0
}

// RGB returns a cell's colour (zero for off-canvas cells).
func (c *Canvas) RGB(cx, cy int) scene.RGB {
	if cx < 0 || cy < 0 || cx >= c.w || cy >= c.h {
		return scene.RGB{}
	}
	return c.col[cy*c.w+cx]
}

// Text places single-width runes starting at cell (cx, cy), overwriting any
// dots in those cells. Cells outside the canvas are skipped.
func (c *Canvas) Text(cx, cy int, s string, col scene.RGB) {
	if cy < 0 || cy >= c.h {
		return
	}
	for _, r := range s {
		if cx >= 0 && cx < c.w && r != 0 {
			i := cy*c.w + cx
			c.txt[i] = r
			c.mask[i] = 0
			c.col[i] = col
		}
		cx++
	}
}

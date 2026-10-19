// Package eventart draws small dim scenes for holiday events. Each art sits
// in one region of the screen margin (a corner or a band along an edge)
// and hides entirely when that region would touch the calendar box.
package eventart

import (
	"math/rand/v2"
	"sort"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

// Region is a rectangle of screen cells.
type Region struct{ X0, Y0, W, H int }

// Touches reports whether the region overlaps the box by at least one cell.
// An empty box (Y0 == -1) never touches.
func (r Region) Touches(b scene.Box) bool {
	if b.Y0 == -1 || r.W <= 0 || r.H <= 0 {
		return false
	}
	return b.X0 < r.X0+r.W && b.X1 > r.X0 && b.Y0 < r.Y0+r.H && b.Y1 >= r.Y0
}

// Art draws one event's scene within Bounds, hiding entirely when
// Bounds touches the calendar box.
type Art interface {
	Resize(w, h int)
	Tick()
	SetTime(now time.Time, occ events.Occurrence)
	Bounds() Region                                 // the full area the art may ever draw in, for the current size
	Cell(x, y int, b scene.Box) (rune, scene.Color) // 0 rune = empty
}

var arts = map[events.Kind]func(*rand.Rand) Art{}

// New returns the art for k, or nil if k has none yet.
func New(k events.Kind, rng *rand.Rand) Art {
	f, ok := arts[k]
	if !ok {
		return nil
	}
	return f(rng)
}

// Kinds returns the kinds that have art, sorted.
func Kinds() []events.Kind {
	ks := make([]events.Kind, 0, len(arts))
	for k := range arts {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })
	return ks
}

// canvas is a region-sized cell buffer. Arts redraw it on Resize and Tick,
// so Cell is a bounds check and a lookup.
type canvas struct {
	r  Region
	rs []rune
	cs []scene.Color
}

// reset sizes the canvas to r and clears it; a degenerate r empties it.
func (c *canvas) reset(r Region) {
	if r.W <= 0 || r.H <= 0 {
		r = Region{}
	}
	c.r = r
	n := r.W * r.H
	if cap(c.rs) < n {
		c.rs = make([]rune, n)
		c.cs = make([]scene.Color, n)
	}
	c.rs, c.cs = c.rs[:n], c.cs[:n]
	c.clear()
}

func (c *canvas) clear() {
	clear(c.rs)
	clear(c.cs)
}

// set draws r at region-local (x, y); cells outside the region are dropped.
func (c *canvas) set(x, y int, r rune, col scene.Color) {
	if x < 0 || y < 0 || x >= c.r.W || y >= c.r.H {
		return
	}
	c.rs[y*c.r.W+x], c.cs[y*c.r.W+x] = r, col
}

// text draws s left to right from region-local (x, y), skipping spaces.
func (c *canvas) text(x, y int, s string, col scene.Color) {
	for _, r := range s {
		if r != ' ' {
			c.set(x, y, r, col)
		}
		x++
	}
}

// cell returns the screen cell (x, y), or nothing at all when the region
// touches the calendar box b.
func (c *canvas) cell(x, y int, b scene.Box) (rune, scene.Color) {
	if c.r.W == 0 || c.r.Touches(b) {
		return 0, scene.Default
	}
	lx, ly := x-c.r.X0, y-c.r.Y0
	if lx < 0 || ly < 0 || lx >= c.r.W || ly >= c.r.H {
		return 0, scene.Default
	}
	i := ly*c.r.W + lx
	return c.rs[i], c.cs[i]
}

// swayOffset is a slow one-column swing, -1, 0 or +1, for a hanging thing
// with the given phase (in ticks).
func swayOffset(tick, phase int) int {
	const quarter = 10
	switch (tick + phase) / quarter % 4 {
	case 1:
		return 1
	case 3:
		return -1
	}
	return 0
}

// hangString draws a string of length n from the top of column x down to
// a weight swung by off columns: straight rows, then a leaning last row.
func hangString(c *canvas, x, n, off int, col scene.Color) {
	for y := 0; y < n; y++ {
		g := '|'
		if y == n-1 {
			switch off {
			case 1:
				g = '\\'
			case -1:
				g = '/'
			}
		}
		c.set(x, y, g, col)
	}
}

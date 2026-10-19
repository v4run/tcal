// Package life implements Conway's Game of Life on a wrapping (toroidal)
// grid, used as an ambient background in live mode.
package life

import (
	"math/rand/v2"
	"slices"
)

// Density is the fraction of cells alive after a fresh Seed.
const Density = 0.2

// settleLimit is how many consecutive settled Ticks are tolerated before
// the grid reseeds itself.
const settleLimit = 12

// Grid is a fixed-size Game of Life board whose edges wrap around.
type Grid struct {
	w, h    int
	cells   []bool
	prev    []bool // generation before cells
	prev2   []bool // generation before prev
	next    []bool // scratch buffer for Step
	settled int    // consecutive Ticks spent settled
	rng     *rand.Rand
}

// New returns an empty w×h grid that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Grid {
	w, h = max(w, 0), max(h, 0)
	n := w * h
	return &Grid{
		w: w, h: h,
		cells: make([]bool, n),
		prev:  make([]bool, n),
		prev2: make([]bool, n),
		next:  make([]bool, n),
		rng:   rng,
	}
}

// Width returns the number of columns.
func (g *Grid) Width() int { return g.w }

// Height returns the number of rows.
func (g *Grid) Height() int { return g.h }

// Alive reports whether the cell at (x, y) is alive. Out-of-range cells are dead.
func (g *Grid) Alive(x, y int) bool {
	if x < 0 || y < 0 || x >= g.w || y >= g.h {
		return false
	}
	return g.cells[y*g.w+x]
}

// Set sets the cell at (x, y). Out-of-range coordinates are ignored.
func (g *Grid) Set(x, y int, alive bool) {
	if x < 0 || y < 0 || x >= g.w || y >= g.h {
		return
	}
	g.cells[y*g.w+x] = alive
}

// Seed fills the grid randomly at Density and clears its history.
func (g *Grid) Seed() {
	for i := range g.cells {
		g.cells[i] = g.rng.Float64() < Density
	}
	clear(g.prev)
	clear(g.prev2)
	g.settled = 0
}

// Step advances the grid by one generation.
func (g *Grid) Step() {
	for y := 0; y < g.h; y++ {
		for x := 0; x < g.w; x++ {
			n := g.neighbors(x, y)
			alive := g.cells[y*g.w+x]
			g.next[y*g.w+x] = n == 3 || (alive && n == 2)
		}
	}
	// Rotate buffers: prev2 <- prev <- cells <- next.
	g.prev2, g.prev, g.cells, g.next = g.prev, g.cells, g.next, g.prev2
}

func (g *Grid) neighbors(x, y int) int {
	n := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx := (x + dx + g.w) % g.w
			ny := (y + dy + g.h) % g.h
			if g.cells[ny*g.w+nx] {
				n++
			}
		}
	}
	return n
}

// Settled reports whether the grid is empty, still, or in a period-2 loop.
func (g *Grid) Settled() bool {
	if !slices.Contains(g.cells, true) {
		return true
	}
	return slices.Equal(g.cells, g.prev) || slices.Equal(g.cells, g.prev2)
}

// Tick advances one generation, reseeding once the grid has stayed settled
// for settleLimit consecutive Ticks.
func (g *Grid) Tick() {
	g.Step()
	if !g.Settled() {
		g.settled = 0
		return
	}
	g.settled++
	if g.settled > settleLimit {
		g.Seed()
	}
}

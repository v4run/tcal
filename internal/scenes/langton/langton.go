// Package langton implements a Langton's ant background: one to three ants
// wander a wrapping grid the size of the screen, flipping cells as they go.
// The grid is cleared and new ants placed after a step budget or once too
// much of it is black, so it never becomes a solid mess.
package langton

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

const (
	// maxAnts bounds the number of ants placed on each reset.
	maxAnts = 3
	// stepsPerTick is how many moves each ant makes per Tick.
	stepsPerTick = 4
	// stepBudget is how many moves each ant makes before a reset; a
	// single ant usually starts building its highway near 10,000.
	stepBudget = 12000
	// maxCoverage is the black-cell fraction that forces a reset.
	maxCoverage = 0.12
)

const (
	cellGlyph = '·'
	antGlyph  = '+'
)

// Directions, clockwise: turning right is +1, turning left is +3.
const (
	up = iota
	right
	down
	left
)

var (
	dxs = [4]int{0, 1, 0, -1}
	dys = [4]int{-1, 0, 1, 0}
)

type ant struct{ x, y, dir int }

// Scene is a Langton's ant colony on a wrapping grid.
type Scene struct {
	w, h   int
	black  []bool
	nblack int
	steps  int // moves since the last reset, over all ants
	ants   []ant
	rng    *rand.Rand
}

var _ scene.Scene = (*Scene)(nil)

// New returns a w×h colony that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.Resize(w, h)
	return s
}

// Resize resets the colony onto a fresh w×h grid.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.black = make([]bool, s.w*s.h)
	s.reset()
}

// reset clears the grid and places 1–maxAnts ants at random.
func (s *Scene) reset() {
	clear(s.black)
	s.nblack, s.steps = 0, 0
	s.ants = s.ants[:0]
	if len(s.black) == 0 {
		return
	}
	for range 1 + s.rng.IntN(maxAnts) {
		s.ants = append(s.ants, ant{s.rng.IntN(s.w), s.rng.IntN(s.h), s.rng.IntN(4)})
	}
}

// step moves every ant once: on a white cell turn right, on a black cell
// turn left; then flip the cell and move forward.
func (s *Scene) step() {
	for i := range s.ants {
		a := &s.ants[i]
		c := a.y*s.w + a.x
		if s.black[c] {
			a.dir = (a.dir + 3) % 4
			s.nblack--
		} else {
			a.dir = (a.dir + 1) % 4
			s.nblack++
		}
		s.black[c] = !s.black[c]
		a.x = (a.x + dxs[a.dir] + s.w) % s.w
		a.y = (a.y + dys[a.dir] + s.h) % s.h
		s.steps++
	}
}

// Tick advances the ants, resetting once the step budget is spent or the
// grid is too crowded.
func (s *Scene) Tick() {
	for range stepsPerTick {
		s.step()
	}
	if s.steps >= stepBudget*len(s.ants) || float64(s.nblack) > maxCoverage*float64(len(s.black)) {
		s.reset()
	}
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return 80 * time.Millisecond }

// Glyph returns the ant or black-cell glyph at (x, y), or 0 for white.
func (s *Scene) Glyph(x, y int, _ scene.Box) rune {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return 0
	}
	for _, a := range s.ants {
		if a.x == x && a.y == y {
			return antGlyph
		}
	}
	if s.black[y*s.w+x] {
		return cellGlyph
	}
	return 0
}

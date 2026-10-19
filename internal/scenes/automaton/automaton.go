// Package automaton implements an elementary cellular automaton background
// that prints downward like a scroll. Each screen cell is a braille glyph
// holding a 2×4 block of automaton cells, so dots are roughly square and the
// pattern stays light. Generations are appended at the bottom one per tick;
// once the screen is full the picture scrolls up.
package automaton

import (
	"math/rand/v2"
	"slices"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

// rules are the Wolfram rule numbers chosen from at random. Grown from a
// single cell, each draws a Sierpinski-style fractal whose large empty
// triangles keep the screen light; chaotic rules like 30 and 110 fill it.
// All are even, so an empty background stays empty instead of flashing.
var rules = []uint8{18, 22, 90}

const (
	// dotsX and dotsY are automaton cells per screen cell (braille is 2×4).
	dotsX, dotsY = 2, 4
	// maxScreens is how many screenfuls of generations a rule runs for
	// before a new rule and seed take over.
	maxScreens = 3
)

// dotBits maps a dot at (column, row) within a braille cell to its bit.
var dotBits = [dotsX][dotsY]rune{
	{0x01, 0x02, 0x04, 0x40},
	{0x08, 0x10, 0x20, 0x80},
}

const brailleBase = 0x2800

// Scene is a scrolling 1D cellular automaton filling the whole screen.
type Scene struct {
	w, h  int
	rule  uint8
	gens  [][]bool // visible generations, oldest first; at most dotsY*h
	run   int      // generations since the last restart
	spare []bool   // scratch row for the next generation
	buf   []rune   // rendered frame, row-major
	rng   *rand.Rand
}

var _ scene.Scene = (*Scene)(nil)

// New returns a w×h automaton that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.Resize(w, h)
	return s
}

// Resize clears the screen and starts a fresh rule for a w×h screen.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.buf = make([]rune, s.w*s.h)
	s.gens = nil
	s.spare = make([]bool, s.cols())
	if s.cols() > 0 && s.h > 0 {
		s.restart(rules[s.rng.IntN(len(rules))])
	}
	s.render()
}

func (s *Scene) cols() int    { return s.w * dotsX }
func (s *Scene) maxGens() int { return s.h * dotsY }

// restart switches to rule and appends a seed row holding a single centre
// cell. Older generations keep scrolling up undisturbed.
func (s *Scene) restart(rule uint8) {
	s.rule, s.run = rule, 0
	row := s.spare
	clear(row)
	row[len(row)/2] = true
	s.push()
}

// push appends s.spare as the newest generation, recycling the oldest row
// once the screen is full.
func (s *Scene) push() {
	row := s.spare
	if len(s.gens) < s.maxGens() {
		s.spare = make([]bool, s.cols())
		s.gens = append(s.gens, row)
	} else {
		s.spare = s.gens[0]
		copy(s.gens, s.gens[1:])
		s.gens[len(s.gens)-1] = row
	}
}

// next computes the generation after cur into out, wrapping at the edges.
func (s *Scene) next(cur, out []bool) {
	n := len(cur)
	for i := range n {
		idx := 0
		if cur[(i+n-1)%n] {
			idx |= 4
		}
		if cur[i] {
			idx |= 2
		}
		if cur[(i+1)%n] {
			idx |= 1
		}
		out[i] = s.rule>>idx&1 == 1
	}
}

// uniform reports whether every cell of row has the same state.
func uniform(row []bool) bool {
	return !slices.Contains(row, !row[0])
}

// Tick appends one generation, restarting with a new rule and seed when the
// pattern dies out, freezes, or has run for maxScreens screens.
func (s *Scene) Tick() {
	if len(s.gens) == 0 {
		return
	}
	last := s.gens[len(s.gens)-1]
	s.next(last, s.spare)
	s.run++
	if uniform(s.spare) || slices.Equal(s.spare, last) || s.run >= maxScreens*s.maxGens() {
		s.restart(rules[s.rng.IntN(len(rules))])
	} else {
		s.push()
	}
	s.render()
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return 120 * time.Millisecond }

// Glyph returns the braille glyph at (x, y), or 0 if it has no dots.
func (s *Scene) Glyph(x, y int, _ scene.Box) rune {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return 0
	}
	return s.buf[y*s.w+x]
}

func (s *Scene) render() {
	clear(s.buf)
	for g, row := range s.gens {
		y, dy := g/dotsY, g%dotsY
		for c, alive := range row {
			if alive {
				s.buf[y*s.w+c/dotsX] |= dotBits[c%dotsX][dy]
			}
		}
	}
	for i, r := range s.buf {
		if r != 0 {
			s.buf[i] = brailleBase + r
		}
	}
}

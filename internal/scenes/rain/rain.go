// Package rain implements a light drizzle, used as an ambient background in
// live mode. Sparse drops fall at slightly varied speeds and land on a ground
// line along the bottom row with a brief splash.
package rain

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

var _ scene.Scene = (*Scene)(nil)

const (
	cellsPerDrop = 70   // screen cells per raindrop; higher is lighter rain
	minSpeed     = 0.6  // slowest fall, in rows per Tick
	speedSpread  = 0.45 // extra random fall speed, in rows per Tick
	splashFrames = 2    // Ticks a splash stays visible after landing
)

const (
	dropGlyph   = '╎'
	groundGlyph = '─'
	impactGlyph = '┴' // ground cell a drop just hit
)

// splashGlyphs[i] is drawn either side of a landing point on frame i, just
// above the ground; the spray widens and fades before vanishing.
var splashGlyphs = [splashFrames]rune{'.', '·'}

type drop struct {
	x     int
	y     float64 // row of the drop; negative while still above the screen
	speed float64 // rows per Tick
}

type splash struct {
	x   int
	age int // Ticks since landing, 0..splashFrames-1
}

// Scene is the rain. All state is rendered into buf on every Tick and
// Resize so Glyph is a plain lookup.
type Scene struct {
	w, h     int
	drops    []drop
	splashes []splash
	buf      []rune
	rng      *rand.Rand
}

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.Resize(w, h)
	return s
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return 70 * time.Millisecond }

// Resize changes the screen size, dropping or adding rain to keep the
// density constant. Drops that fall outside the new bounds are re-spawned.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.buf = make([]rune, s.w*s.h)

	want := s.target()
	if len(s.drops) > want {
		s.drops = s.drops[:want]
	}
	for i := range s.drops {
		if s.drops[i].x >= s.w || s.drops[i].y >= float64(s.ground()) {
			s.drops[i] = s.spawn(true)
		}
	}
	for len(s.drops) < want {
		s.drops = append(s.drops, s.spawn(true))
	}
	s.splashes = s.splashes[:0]
	s.render()
}

// Tick moves every drop down; drops reaching the ground splash and respawn
// above the screen.
func (s *Scene) Tick() {
	kept := s.splashes[:0]
	for _, sp := range s.splashes {
		if sp.age++; sp.age < splashFrames {
			kept = append(kept, sp)
		}
	}
	s.splashes = kept

	ground := float64(s.ground())
	for i := range s.drops {
		d := &s.drops[i]
		d.y += d.speed
		if d.y >= ground {
			s.splashes = append(s.splashes, splash{x: d.x})
			*d = s.spawn(false)
		}
	}
	s.render()
}

// Glyph returns the rune drawn at screen (x, y), or 0 for empty.
func (s *Scene) Glyph(x, y int, _ scene.Box) rune {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return 0
	}
	return s.buf[y*s.w+x]
}

// ground is the row of the ground line.
func (s *Scene) ground() int { return s.h - 1 }

// target is the number of drops for the current screen size.
func (s *Scene) target() int {
	if s.w == 0 || s.h < 2 {
		return 0
	}
	return max(1, s.w*(s.h-1)/cellsPerDrop)
}

// spawn returns a new drop at a random column. Scattered drops start
// anywhere on screen (used to fill an empty sky at once); others start a
// little above the top edge so arrivals are staggered.
func (s *Scene) spawn(scattered bool) drop {
	d := drop{
		x:     s.rng.IntN(s.w),
		speed: minSpeed + s.rng.Float64()*speedSpread,
	}
	if scattered {
		d.y = s.rng.Float64() * float64(s.ground())
	} else {
		d.y = -s.rng.Float64() * float64(s.h)
	}
	return d
}

func (s *Scene) render() {
	if s.h == 0 {
		return
	}
	clear(s.buf)
	g := s.ground()
	for x := 0; x < s.w; x++ {
		s.buf[g*s.w+x] = groundGlyph
	}
	for _, d := range s.drops {
		s.set(d.x, int(d.y), dropGlyph)
	}
	for _, sp := range s.splashes {
		if sp.age == 0 {
			s.set(sp.x, g, impactGlyph)
		}
		r := splashGlyphs[sp.age]
		s.set(sp.x-1-sp.age, g-1, r)
		s.set(sp.x+1+sp.age, g-1, r)
	}
}

// set draws r at (x, y), ignoring cells off screen.
func (s *Scene) set(x, y int, r rune) {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return
	}
	s.buf[y*s.w+x] = r
}

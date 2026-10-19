// Package starfield implements a slowly drifting night sky, used as an
// ambient background in live mode. Stars in three depth layers drift
// sideways with parallax, and a shooting star streaks past now and then.
package starfield

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

var _ scene.Scene = (*Scene)(nil)

// interval is the delay between Ticks.
const interval = 100 * time.Millisecond

const cellsPerStar = 70 // screen cells per star; higher is a sparser sky

// layer is one depth plane of stars.
type layer struct {
	glyph rune
	speed float64 // columns per Tick, leftwards
	share float64 // fraction of all stars in this layer
}

// layers run from far (dim, slow, common) to near (bright, fast, rare).
var layers = []layer{
	{glyph: '.', speed: 0.02, share: 0.55},
	{glyph: '·', speed: 0.05, share: 0.30},
	{glyph: '+', speed: 0.10, share: 0.15},
}

const (
	meteorMinWait = 100 // min Ticks between shooting stars (10 s)
	meteorWaitVar = 200 // extra random Ticks between shooting stars
	meteorLife    = 14  // Ticks before a shooting star starts burning out
	meteorDX      = 2   // columns moved per Tick (sign gives direction)
)

// meteorTrail is drawn from the head backwards; index 1 is overwritten by
// a slash matching the direction of travel.
var meteorTrail = []rune{'*', 0, '·', '.'}

type star struct {
	x     float64
	y     int
	layer int
}

type meteor struct {
	x, y int
	dir  int // +1 travels right, -1 left; always moves down
	age  int
}

// Scene is the sky. All state is rendered into buf on every Tick and Resize
// so Glyph is a plain lookup.
type Scene struct {
	w, h   int
	stars  []star
	meteor *meteor // nil when none is in flight
	wait   int     // Ticks until the next shooting star
	buf    []rune
	rng    *rand.Rand
}

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.wait = s.nextWait()
	s.Resize(w, h)
	return s
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return interval }

// Resize changes the screen size, keeping stars that still fit and adding
// or removing stars to keep the density constant.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.buf = make([]rune, s.w*s.h)

	kept := s.stars[:0]
	for _, st := range s.stars {
		if int(st.x) < s.w && st.y < s.h {
			kept = append(kept, st)
		}
	}
	s.stars = kept
	want := s.target()
	if len(s.stars) > want {
		s.stars = s.stars[:want]
	}
	for len(s.stars) < want {
		s.stars = append(s.stars, s.spawn(s.rng.Float64()*float64(s.w)))
	}
	if s.meteor != nil && (s.meteor.x >= s.w || s.meteor.y >= s.h) {
		s.meteor = nil
	}
	s.render()
}

// Tick drifts every star left by its layer's speed, wrapping stars that
// leave the left edge to a fresh row on the right, and advances any
// shooting star.
func (s *Scene) Tick() {
	for i := range s.stars {
		st := &s.stars[i]
		st.x -= layers[st.layer].speed
		if st.x < 0 {
			*st = s.spawn(float64(s.w) - 0.001)
		}
	}

	if m := s.meteor; m != nil {
		m.x += m.dir * meteorDX
		m.y++
		m.age++
		// Done once the whole trail has left the screen or burned out.
		n := len(meteorTrail) - 1
		tx, ty := m.x-m.dir*meteorDX*n, m.y-n
		if m.burnt() >= len(meteorTrail) || ty >= s.h || (m.dir > 0 && tx >= s.w) || (m.dir < 0 && tx < 0) {
			s.meteor = nil
		}
	} else if s.wait--; s.wait <= 0 {
		s.launch()
		s.wait = s.nextWait()
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

// target is the number of stars for the current screen size.
func (s *Scene) target() int {
	if s.w == 0 || s.h == 0 {
		return 0
	}
	return max(1, s.w*s.h/cellsPerStar)
}

// spawn returns a star at column x on a random row and layer.
func (s *Scene) spawn(x float64) star {
	l, p := 0, s.rng.Float64()
	for l < len(layers)-1 && p >= layers[l].share {
		p -= layers[l].share
		l++
	}
	return star{x: x, y: s.rng.IntN(s.h), layer: l}
}

// burnt is how many trail cells, from the head back, have burned out.
func (m *meteor) burnt() int { return max(0, m.age-meteorLife) }

func (s *Scene) nextWait() int { return meteorMinWait + s.rng.IntN(meteorWaitVar) }

// launch starts a shooting star in the upper part of the sky, heading
// diagonally down and away from the side it starts on.
func (s *Scene) launch() {
	if s.w < 8 || s.h < 4 {
		return
	}
	m := &meteor{x: s.rng.IntN(s.w), y: s.rng.IntN(max(1, s.h/3)), dir: 1}
	if m.x >= s.w/2 {
		m.dir = -1
	}
	s.meteor = m
}

func (s *Scene) render() {
	clear(s.buf)
	for _, st := range s.stars {
		s.set(int(st.x), st.y, layers[st.layer].glyph)
	}
	if m := s.meteor; m != nil {
		slash := '\\'
		if m.dir < 0 {
			slash = '/'
		}
		// Draw tail first so the head wins any overlap.
		for i := len(meteorTrail) - 1; i >= m.burnt(); i-- {
			r := meteorTrail[i]
			if i == 1 {
				r = slash
			}
			s.set(m.x-m.dir*meteorDX*i, m.y-i, r)
		}
	}
}

// set draws r at (x, y), ignoring cells off screen.
func (s *Scene) set(x, y int, r rune) {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return
	}
	s.buf[y*s.w+x] = r
}

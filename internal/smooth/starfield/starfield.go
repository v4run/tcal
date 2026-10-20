// Package starfield is the smooth starfield scene: stars in three depth
// layers glide sideways with parallax at sub-cell resolution, and a shooting
// star streaks across the sky now and then.
package starfield

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/dots"
	"github.com/v4run/tcal/internal/scene"
)

var (
	_ scene.Scene       = (*Scene)(nil)
	_ scene.Stepper     = (*Scene)(nil)
	_ scene.TrueColored = (*Scene)(nil)
)

const (
	step         = 33 * time.Millisecond
	cellsPerStar = 70

	minWait  = 10 * time.Second
	waitVar  = 20 * time.Second
	lifeSpan = 1400 * time.Millisecond // flight time before burning out
	burnTime = 400 * time.Millisecond  // fade-out after lifeSpan
	// Shooting-star motion in columns and rows per second (classic: 2 cols
	// and 1 row per 100 ms).
	meteorVX = 20.0
	meteorVY = 10.0
	// The trail samples the path every trailGap seconds behind the head.
	trailDots = 7
	trailGap  = 0.05
)

type layer struct {
	speed float64 // columns per second, leftwards
	share float64 // fraction of all stars in this layer
}

// layers run from far (dim, slow, common) to near (bright, fast, rare).
var layers = []layer{
	{speed: 0.02 / 0.1, share: 0.55},
	{speed: 0.05 / 0.1, share: 0.30},
	{speed: 0.10 / 0.1, share: 0.15},
}

var layerCol = []scene.RGB{
	{R: 90, G: 90, B: 120},
	{R: 140, G: 140, B: 175},
	{R: 200, G: 200, B: 230},
}

var headCol = scene.RGB{R: 235, G: 235, B: 245}

type star struct {
	x, y  float64
	layer int
}

type meteor struct {
	x, y float64 // head position
	dir  int     // +1 travels right, -1 left; always moves down
	age  time.Duration
}

// Scene is the starfield.
type Scene struct {
	rng    *rand.Rand
	w, h   int
	stars  []star
	meteor *meteor       // nil when none is in flight
	wait   time.Duration // until the next shooting star
	canvas dots.Canvas
}

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.wait = s.nextWait()
	s.Resize(w, h)
	return s
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return step }

// Tick advances one interval.
func (s *Scene) Tick() { s.Step(step) }

// Resize changes the screen size, keeping the star density constant.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	want := 0
	if s.w > 0 && s.h > 0 {
		want = max(1, s.w*s.h/cellsPerStar)
	}
	if len(s.stars) > want {
		s.stars = s.stars[:want]
	}
	for i := range s.stars {
		if s.stars[i].x >= float64(s.w) || s.stars[i].y >= float64(s.h) {
			s.stars[i] = s.spawn()
		}
	}
	for len(s.stars) < want {
		s.stars = append(s.stars, s.spawn())
	}
	s.meteor = nil
	s.canvas.Reset(s.w, s.h)
	s.render()
}

// Step advances the animation by dt.
func (s *Scene) Step(dt time.Duration) {
	dt = max(dt, 0)
	sec := dt.Seconds()
	for i := range s.stars {
		st := &s.stars[i]
		if st.x -= layers[st.layer].speed * sec; st.x < 0 {
			st.x += float64(s.w)
			st.y = s.rng.Float64() * float64(s.h)
			if st.x >= float64(s.w) { // dt longer than a full screen crossing
				st.x = s.rng.Float64() * float64(s.w)
			}
		}
	}
	if m := s.meteor; m != nil {
		m.age += dt
		m.x += float64(m.dir) * meteorVX * sec
		m.y += meteorVY * sec
		if m.age >= lifeSpan+burnTime {
			s.meteor = nil
			s.wait = s.nextWait()
		}
	} else if s.wait -= dt; s.wait <= 0 {
		if s.launch() {
			s.meteor.age = -s.wait // carry over the overshoot
			s.meteor.x += float64(s.meteor.dir) * meteorVX * -s.wait.Seconds()
			s.meteor.y += meteorVY * -s.wait.Seconds()
		} else {
			s.wait = s.nextWait()
		}
	}
	s.render()
}

// Glyph returns the rune at screen (x, y), or 0 for empty.
func (s *Scene) Glyph(x, y int, _ scene.Box) rune { return s.canvas.Glyph(x, y) }

// RGB returns the colour of the glyph at (x, y).
func (s *Scene) RGB(x, y int, _ scene.Box) scene.RGB { return s.canvas.RGB(x, y) }

func (s *Scene) nextWait() time.Duration {
	return minWait + time.Duration(s.rng.Float64()*float64(waitVar))
}

// spawn returns a star anywhere on the screen.
func (s *Scene) spawn() star {
	r, l := s.rng.Float64(), 0
	for l < len(layers)-1 && r >= layers[l].share {
		r -= layers[l].share
		l++
	}
	return star{
		x:     s.rng.Float64() * float64(s.w),
		y:     s.rng.Float64() * float64(s.h),
		layer: l,
	}
}

// launch starts a shooting star in the upper part of the sky, heading down
// and away from the side it starts on. It reports whether one was started.
func (s *Scene) launch() bool {
	if s.w < 8 || s.h < 4 {
		return false
	}
	m := &meteor{
		x:   s.rng.Float64() * float64(s.w),
		y:   s.rng.Float64() * float64(max(1, s.h/3)),
		dir: 1,
	}
	if m.x >= float64(s.w)/2 {
		m.dir = -1
	}
	s.meteor = m
	return true
}

func scale(c scene.RGB, k float64) scene.RGB {
	f := func(x uint8) uint8 { return uint8(float64(x)*k + 0.5) }
	return scene.RGB{R: f(c.R), G: f(c.G), B: f(c.B)}
}

func mix(a, b scene.RGB, t float64) scene.RGB {
	f := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return scene.RGB{R: f(a.R, b.R), G: f(a.G, b.G), B: f(a.B, b.B)}
}

func (s *Scene) render() {
	s.canvas.Clear()
	if s.w == 0 || s.h == 0 {
		return
	}
	for _, st := range s.stars {
		s.canvas.Dot(st.x, st.y, layerCol[st.layer])
	}
	m := s.meteor
	if m == nil {
		return
	}
	k := 1.0
	if m.age > lifeSpan {
		k = max(0, 1-float64(m.age-lifeSpan)/float64(burnTime))
	}
	// Trail first so the head wins shared cells.
	for i := trailDots; i >= 1; i-- {
		back := float64(i) * trailGap
		c := mix(headCol, layerCol[0], float64(i)/float64(trailDots))
		s.canvas.Dot(m.x-float64(m.dir)*meteorVX*back, m.y-meteorVY*back, scale(c, k))
	}
	s.canvas.Dot(m.x, m.y, scale(headCol, k))
}

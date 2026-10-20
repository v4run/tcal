// Package starfield is the smooth starfield scene: stars stream outward from
// the screen centre at sub-cell speed and brighten as they approach.
package starfield

import (
	"math"
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
	// aspect is the height of a terminal cell relative to its width.
	aspect = 2.0
	// Radial speed in columns per second.
	minSpeed    = 2.0
	speedSpread = 10.0
	// spawnR is the largest radius a respawned star starts at.
	spawnR = 3.0
)

var (
	farCol  = scene.RGB{R: 90, G: 90, B: 120}
	nearCol = scene.RGB{R: 200, G: 200, B: 230}
)

type star struct {
	angle float64 // radians
	r     float64 // distance from the centre, in columns
	speed float64 // columns per second
}

// Scene is the starfield.
type Scene struct {
	rng    *rand.Rand
	w, h   int
	stars  []star
	canvas dots.Canvas
}

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
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
	for len(s.stars) < want {
		s.stars = append(s.stars, s.spawn(true))
	}
	for i := range s.stars {
		if s.stars[i].r > s.edge(s.stars[i].angle) {
			s.stars[i] = s.spawn(false)
		}
	}
	s.canvas.Reset(s.w, s.h)
	s.render()
}

// Step advances the animation by dt.
func (s *Scene) Step(dt time.Duration) {
	sec := max(dt, 0).Seconds()
	for i := range s.stars {
		st := &s.stars[i]
		if st.r += st.speed * sec; st.r > s.edge(st.angle) {
			*st = s.spawn(false)
		}
	}
	s.render()
}

// Glyph returns the rune at screen (x, y), or 0 for empty.
func (s *Scene) Glyph(x, y int, _ scene.Box) rune { return s.canvas.Glyph(x, y) }

// RGB returns the colour of the glyph at (x, y).
func (s *Scene) RGB(x, y int, _ scene.Box) scene.RGB { return s.canvas.RGB(x, y) }

// maxR is the radius at which a star has certainly left the screen.
func (s *Scene) maxR() float64 {
	return math.Hypot(float64(s.w)/2, float64(s.h)/2*aspect)
}

// edge is the radius at which a star leaving at angle a reaches the screen
// border.
func (s *Scene) edge(a float64) float64 {
	c, sn := math.Abs(math.Cos(a)), math.Abs(math.Sin(a))
	e := s.maxR()
	if c > 1e-9 {
		e = min(e, float64(s.w)/2/c)
	}
	if sn > 1e-9 {
		e = min(e, float64(s.h)/2*aspect/sn)
	}
	return e
}

// spawn returns a star near the centre, or anywhere on the screen if
// scattered.
func (s *Scene) spawn(scattered bool) star {
	st := star{
		angle: s.rng.Float64() * 2 * math.Pi,
		speed: minSpeed + s.rng.Float64()*speedSpread,
	}
	if scattered {
		st.r = s.rng.Float64() * s.maxR()
	} else {
		st.r = s.rng.Float64() * spawnR
	}
	return st
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
	cx, cy := float64(s.w)/2, float64(s.h)/2
	for _, st := range s.stars {
		depth := min(st.r/max(s.edge(st.angle), 1e-9), 1)
		s.canvas.Dot(cx+st.r*math.Cos(st.angle), cy+st.r*math.Sin(st.angle)/aspect, mix(farCol, nearCol, depth))
	}
}

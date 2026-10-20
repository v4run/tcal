// Package flow is the smooth flow-field scene: particles glide along a
// smoothly varying, slowly changing vector field, each leaving a short
// trail that fades toward dim.
package flow

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
	step = 33 * time.Millisecond
	// cellsPerParticle sets particle density; the count is clamped to
	// [minParticles, maxParticles].
	cellsPerParticle           = 300
	minParticles, maxParticles = 6, 48
	// speed is how far a particle moves per second, in columns
	// (0.6 columns per 80 ms).
	speed = 0.6 / 0.08
	// aspect is the height of a terminal cell relative to its width.
	aspect = 2.0
	// fieldRate is how fast the field's time advances per second
	// (0.008 per 80 ms).
	fieldRate = 0.008 / 0.08
	// Particle lifetimes in seconds (60-180 ticks of 80 ms).
	minLife, maxLife = 60 * 0.08, 180 * 0.08
	// trailLen is the number of fading trail dots behind a head, sampled
	// every trailGap.
	trailLen = 3
	trailGap = 90 * time.Millisecond
)

var (
	particleCols = []scene.RGB{{R: 80, G: 190, B: 220}, {R: 80, G: 110, B: 230}}
	dimCol       = scene.RGB{R: 40, G: 45, B: 70}
	// trailFade is how far toward dimCol each trail dot has faded.
	trailFade = [trailLen]float64{0.35, 0.6, 0.8}
)

type particle struct {
	x, y  float64 // in cells
	life  float64 // seconds left
	hue   int     // index into particleCols, fixed for the particle slot
	trail [trailLen][2]float64
	n     int           // trail points recorded
	since time.Duration // time since the last trail point
}

// Scene is a flow field filling the whole screen.
type Scene struct {
	rng    *rand.Rand
	w, h   int
	t      float64 // field time
	ps     []particle
	gap    time.Duration
	field  func(x, y float64) float64 // field direction override (tests)
	canvas dots.Canvas
}

// New returns a w×h flow field that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng, t: rng.Float64() * 100, gap: trailGap}
	s.Resize(w, h)
	return s
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return step }

// Tick advances one interval.
func (s *Scene) Tick() { s.Step(step) }

// Resize respawns the particles for a w×h screen.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.canvas.Reset(s.w, s.h)
	s.ps = s.ps[:0]
	if s.w == 0 || s.h == 0 {
		return
	}
	n := min(max(s.w*s.h/cellsPerParticle, minParticles), maxParticles)
	for i := range n {
		// Alternate hues by slot so colour never consumes randomness.
		p := particle{hue: i % len(particleCols)}
		s.spawn(&p)
		p.life = 1 + s.rng.Float64()*(maxLife-1) // stagger respawns
		s.ps = append(s.ps, p)
	}
	s.render()
}

func (s *Scene) spawn(p *particle) {
	p.x = s.rng.Float64() * float64(s.w)
	p.y = s.rng.Float64() * float64(s.h)
	p.life = minLife + s.rng.Float64()*(maxLife-minLife)
	p.n, p.since = 0, 0
}

// angle returns the field direction at (x, y), in radians. x is scaled by
// aspect so the field's features are round on screen.
func (s *Scene) angle(x, y float64) float64 {
	if s.field != nil {
		return s.field(x, y)
	}
	x /= aspect
	t := s.t
	v := math.Sin(x*0.09+t) + math.Cos(y*0.11-0.8*t) + 0.5*math.Sin((x-y)*0.05+0.5*t)
	return v * math.Pi / 1.25
}

// Step advances the animation by dt.
func (s *Scene) Step(dt time.Duration) {
	dt = max(dt, 0)
	sec := dt.Seconds()
	s.t += fieldRate * sec
	for i := range s.ps {
		p := &s.ps[i]
		if p.since += dt; p.since >= s.gap {
			p.since -= s.gap
			copy(p.trail[1:], p.trail[:trailLen-1])
			p.trail[0] = [2]float64{p.x, p.y}
			p.n = min(p.n+1, trailLen)
		}
		a := s.angle(p.x, p.y)
		p.x += math.Cos(a) * speed * sec
		p.y += math.Sin(a) * speed * sec / aspect
		p.life -= sec
		if p.life <= 0 || p.x < 0 || p.y < 0 || p.x >= float64(s.w) || p.y >= float64(s.h) {
			s.spawn(p)
		}
	}
	s.render()
}

// Glyph returns the rune at screen (x, y), or 0 for empty.
func (s *Scene) Glyph(x, y int, _ scene.Box) rune { return s.canvas.Glyph(x, y) }

// RGB returns the colour of the glyph at (x, y).
func (s *Scene) RGB(x, y int, _ scene.Box) scene.RGB { return s.canvas.RGB(x, y) }

func mix(a, b scene.RGB, t float64) scene.RGB {
	f := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return scene.RGB{R: f(a.R, b.R), G: f(a.G, b.G), B: f(a.B, b.B)}
}

func (s *Scene) render() {
	s.canvas.Clear()
	for _, p := range s.ps {
		col := particleCols[p.hue%len(particleCols)]
		// Oldest first so the head wins a shared cell.
		for i := p.n - 1; i >= 0; i-- {
			s.canvas.Dot(p.trail[i][0], p.trail[i][1], mix(col, dimCol, trailFade[i]))
		}
		s.canvas.Dot(p.x, p.y, col)
	}
}

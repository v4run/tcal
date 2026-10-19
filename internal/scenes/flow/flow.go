// Package flow implements a flow-field background: a few dozen particles
// drift along a smoothly varying, slowly changing vector field, leaving
// short trails that fade with age.
package flow

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

const (
	// cellsPerParticle sets particle density; the count is clamped to
	// [minParticles, maxParticles].
	cellsPerParticle           = 300
	minParticles, maxParticles = 6, 48
	// speed is how far a particle moves per tick, in columns.
	speed = 0.6
	// aspect is the height of a terminal cell relative to its width.
	aspect = 2.0
	// dt is how far the field's time advances per tick.
	dt = 0.008
	// minLife and maxLife bound a particle's lifetime, in ticks.
	minLife, maxLife = 60, 180
	// headAge and trailAge are the ages (in ticks) below which a visited
	// cell is drawn as a fresh or an old trail.
	headAge, trailAge = 4, 12
)

const (
	headGlyph  = '·'
	trailGlyph = '.'
)

// never marks a cell no particle has visited.
const never = math.MinInt32

type particle struct {
	x, y float64 // in cells
	life int     // ticks left
}

// Scene is a flow field filling the whole screen.
type Scene struct {
	w, h  int
	tick  int
	t     float64 // field time
	ps    []particle
	stamp []int32 // tick each cell was last visited
	rng   *rand.Rand
}

var _ scene.Scene = (*Scene)(nil)

// New returns a w×h flow field that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng, t: rng.Float64() * 100}
	s.Resize(w, h)
	return s
}

// Resize clears all trails and respawns the particles for a w×h screen.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.stamp = make([]int32, s.w*s.h)
	for i := range s.stamp {
		s.stamp[i] = never
	}
	s.ps = s.ps[:0]
	if s.w == 0 || s.h == 0 {
		return
	}
	n := min(max(s.w*s.h/cellsPerParticle, minParticles), maxParticles)
	for range n {
		var p particle
		s.spawn(&p)
		p.life = 1 + s.rng.IntN(maxLife) // stagger respawns
		s.ps = append(s.ps, p)
	}
}

func (s *Scene) spawn(p *particle) {
	p.x = s.rng.Float64() * float64(s.w)
	p.y = s.rng.Float64() * float64(s.h)
	p.life = minLife + s.rng.IntN(maxLife-minLife+1)
}

// angle returns the field direction at (x, y), in radians. x is scaled by
// aspect so the field's features are round on screen.
func (s *Scene) angle(x, y float64) float64 {
	x /= aspect
	t := s.t
	v := math.Sin(x*0.09+t) + math.Cos(y*0.11-0.8*t) + 0.5*math.Sin((x-y)*0.05+0.5*t)
	return v * math.Pi / 1.25
}

// Tick advances every particle and stamps the cells they occupy.
func (s *Scene) Tick() {
	s.tick++
	s.t += dt
	for i := range s.ps {
		p := &s.ps[i]
		a := s.angle(p.x, p.y)
		p.x += math.Cos(a) * speed
		p.y += math.Sin(a) * speed / aspect
		p.life--
		if p.life <= 0 || p.x < 0 || p.y < 0 || p.x >= float64(s.w) || p.y >= float64(s.h) {
			s.spawn(p)
			continue
		}
		s.stamp[int(p.y)*s.w+int(p.x)] = int32(s.tick)
	}
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return 80 * time.Millisecond }

// Glyph returns the trail glyph at (x, y) by its age, or 0 if none.
func (s *Scene) Glyph(x, y int, _ scene.Box) rune {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return 0
	}
	switch age := s.tick - int(s.stamp[y*s.w+x]); {
	case age < headAge:
		return headGlyph
	case age < trailAge:
		return trailGlyph
	}
	return 0
}

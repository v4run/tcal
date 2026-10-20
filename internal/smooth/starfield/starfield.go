// Package starfield is the smooth starfield scene: stars in three depth
// layers glide sideways with parallax at sub-cell resolution, and a shooting
// slow comet with a spreading tail crosses the sky now and then.
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

	firstMin  = 30 * time.Second
	firstVar  = 20 * time.Second
	repeatMin = 30 * time.Second
	repeatVar = 20 * time.Second
	// A comet takes crossMin..crossMin+crossVar seconds to cross the width.
	crossMin = 6.0
	crossVar = 4.0
	// cometSlope is the rows fallen per column travelled.
	cometSlope = 0.12
	// tailCols is the tail length in columns; the tail is sampled every
	// half column (one dot).
	tailCols = 20.0
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

var (
	coreCol     = scene.RGB{R: 235, G: 240, B: 250}
	ringCol     = scene.RGB{R: 205, G: 218, B: 242}
	tailNearCol = scene.RGB{R: 200, G: 215, B: 240}
)

type star struct {
	x, y  float64
	layer int
}

// comet is a glowing head dragging a tail behind it.
type comet struct {
	x, y   float64 // head position
	vx, vy float64 // columns and rows per second; vy > 0 is down
	seed   uint64  // fixes which tail dots are drawn
	age    time.Duration
}

// Scene is the starfield.
type Scene struct {
	rng    *rand.Rand
	w, h   int
	stars  []star
	crng   *rand.Rand    // comet randomness, kept apart from the stars'
	comet  *comet        // nil when none is in flight
	wait   time.Duration // until the next comet
	canvas dots.Canvas
}

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng, crng: rand.New(rand.NewPCG(rng.Uint64(), rng.Uint64()))}
	s.wait = firstMin + time.Duration(s.crng.Float64()*float64(firstVar))
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
	s.comet = nil
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
	if c := s.comet; c != nil {
		c.age += dt
		c.x += c.vx * sec
		c.y += c.vy * sec
		// Done once the tail has left the far side too.
		var past float64 // columns beyond the finishing line
		if c.vx > 0 {
			past = c.x - (float64(s.w) + tailCols + 2)
		} else {
			past = -tailCols - 2 - c.x
		}
		if past > 0 {
			s.comet = nil
			// Time already elapsed since the finish counts toward the wait.
			s.wait = repeatMin + time.Duration(s.crng.Float64()*float64(repeatVar)) -
				time.Duration(past/math.Abs(c.vx)*float64(time.Second))
		}
	} else if s.wait -= dt; s.wait <= 0 {
		if s.launch() {
			over := -s.wait.Seconds()
			s.comet.age = -s.wait // carry over the overshoot
			s.comet.x += s.comet.vx * over
			s.comet.y += s.comet.vy * over
		} else {
			s.wait = repeatMin
		}
	}
	s.render()
}

// Glyph returns the rune at screen (x, y), or 0 for empty.
func (s *Scene) Glyph(x, y int, _ scene.Box) rune { return s.canvas.Glyph(x, y) }

// RGB returns the colour of the glyph at (x, y).
func (s *Scene) RGB(x, y int, _ scene.Box) scene.RGB { return s.canvas.RGB(x, y) }

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

// launch sends a comet in from just off one side, in the upper half of the
// sky, on a gentle downward slant. It reports whether one was started.
func (s *Scene) launch() bool {
	if s.w < 20 || s.h < 6 {
		return false
	}
	speed := float64(s.w) / (crossMin + s.crng.Float64()*crossVar)
	c := &comet{
		y:    s.crng.Float64() * float64(s.h) / 2,
		vx:   speed,
		vy:   speed * cometSlope,
		seed: s.crng.Uint64(),
	}
	if s.crng.IntN(2) == 0 {
		c.vx = -speed
		c.x = float64(s.w) + 1
	} else {
		c.x = -1
	}
	s.comet = c
	return true
}

// hash maps (seed, i, j) to a stable value in [0, 1).
func hash(seed uint64, i, j int) float64 {
	z := seed + uint64(i)*0x9e3779b97f4a7c15 + uint64(j+16)*0xbf58476d1ce4e5b9
	z = (z ^ z>>30) * 0xbf58476d1ce4e5b9
	z = (z ^ z>>27) * 0x94d049bb133111eb
	z ^= z >> 31
	return float64(z>>11) / (1 << 53)
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
	c := s.comet
	if c == nil {
		return
	}
	dir := 1.0
	if c.vx < 0 {
		dir = -1
	}
	slope := c.vy / max(dir*c.vx, 1e-9)
	flick := float64(c.age) / float64(time.Second)
	// Tail first, head last. Dots are sampled one dot (half a column) apart
	// along the path; the tail widens and thins with distance.
	n := int(tailCols * 2)
	for i := n; i >= 1; i-- {
		u := float64(i) / float64(n)
		k := int(1 + 4*u + 0.5)
		dens := 1 - 0.7*u + 0.06*math.Sin(flick*4+float64(i)*0.7)
		col := scale(mix(tailNearCol, layerCol[0], min(1, u*1.5)), 1-0.35*u)
		bx := c.x - dir*float64(i)*0.5
		by := c.y - slope*float64(i)*0.5
		for jj := range k {
			if hash(c.seed, i, jj) >= dens {
				continue
			}
			j := float64(jj) - float64(k-1)/2
			s.canvas.Dot(bx, by+j*0.25, col)
		}
	}
	for _, d := range [4][2]float64{{-0.5, 0}, {0.5, 0}, {0, -0.25}, {0, 0.25}} {
		s.canvas.Dot(c.x+d[0], c.y+d[1], ringCol)
	}
	s.canvas.Dot(c.x, c.y, coreCol)
}

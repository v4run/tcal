// Package coast is the smooth coast scene, seen from the side along the
// bottom of the screen: a sandy beach on the left, the sea on the right
// with rocks standing in it. Swells roll in from the right, burst into
// spray against the rocks and wash up the sand.
package coast

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/dots"
	"github.com/v4run/tcal/internal/scene"
)

// Height is the number of rows the strip occupies at the bottom of the
// screen.
const Height = 8

// MinWidth is the narrowest screen the scene will draw on.
const MinWidth = 40

const step = 33 * time.Millisecond

// Dot geometry of the strip (Height*4 = 32 dot rows).
const (
	seaRow   = 22.0 // dot row of the calm sea surface
	sandTop  = 14.0 // dot row of the sand at the screen's left edge
	beachEnd = 0.32 // fraction of the width where the sand meets the sea
)

// Swell timing and shape.
const (
	waveMin    = 4.0  // seconds between swells, at least
	waveSpread = 3.5  // extra random seconds
	waveSpeed  = 22.0 // dots per second, leftwards
	waveAmp    = 3.2  // peak height in dot rows
	waveSigma  = 9.0  // half-width of the swell in dots
	rockDamp   = 0.55 // fraction of a swell left after a rock
)

var (
	surfCol  = scene.RGB{R: 95, G: 145, B: 195}
	seaCol   = scene.RGB{R: 55, G: 95, B: 145}
	deepCol  = scene.RGB{R: 40, G: 70, B: 115}
	sandCol  = scene.RGB{R: 185, G: 165, B: 120}
	wetCol   = scene.RGB{R: 125, G: 110, B: 85}
	rockCol  = scene.RGB{R: 105, G: 105, B: 112}
	edgeCol  = scene.RGB{R: 150, G: 150, B: 158}
	sprayCol = scene.RGB{R: 225, G: 235, B: 245}
	foamCol  = scene.RGB{R: 200, G: 215, B: 228}
)

var (
	_ scene.Scene       = (*Scene)(nil)
	_ scene.Stepper     = (*Scene)(nil)
	_ scene.TrueColored = (*Scene)(nil)
)

// rock is a rounded outcrop standing on the sea floor.
type rock struct {
	cx, r  float64   // centre and half-width, in dots
	height float64   // dots above the sea surface at its peak
	tops   []float64 // dot row of the top at each column from cx-r
}

// swell is one wave travelling leftwards.
type swell struct {
	x    float64 // dot column of its crest
	amp  float64
	hits map[int]bool // rocks it has already hit
}

// drop is a spray or foam particle.
type drop struct {
	x, y, vx, vy float64 // dots and dots per second
	age, life    float64 // seconds
	col          scene.RGB
}

// Scene is the coast.
type Scene struct {
	rng    *rand.Rand
	w, h   int
	t      float64 // seconds since start
	rocks  []rock
	swells []*swell
	drops  []drop
	wait   float64 // seconds until the next swell
	// The wash up the beach: the front's dot column, its speed, and how
	// wet each sand column is (0 dry, 1 just washed).
	runup    float64
	runupV   float64
	running  bool
	wet      []float64
	canvas   dots.Canvas
	shoreCol float64 // dot column where the sand meets the calm sea
}

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng, wait: 1 + rng.Float64()*2}
	s.Resize(w, h)
	return s
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return step }

// Tick advances one interval.
func (s *Scene) Tick() { s.Step(step) }

// Resize lays the coast out afresh for the new width.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	W := float64(s.w * 2)
	s.shoreCol = W * beachEnd
	s.wet = make([]float64, s.w*2)
	s.swells, s.drops, s.running = nil, nil, false
	s.rocks = s.rocks[:0]
	if s.w >= MinWidth {
		// A low rock in mid-sea and a tall one near the right edge.
		s.rocks = append(s.rocks,
			s.newRock(W*0.58, W*0.035+4, 7),
			s.newRock(W*0.86, W*0.05+6, 15),
		)
		if s.w >= 100 {
			s.rocks = append(s.rocks, s.newRock(W*0.72, W*0.02+3, 4))
		}
	}
	s.canvas.Reset(s.w, Height)
	s.render()
}

func (s *Scene) newRock(cx, r, height float64) rock {
	rk := rock{cx: cx, r: r, height: height}
	n := int(2*r) + 1
	phase := s.rng.Float64() * 10
	for i := range n {
		u := (float64(i) - r) / r
		shape := math.Pow(max(0, 1-u*u), 0.6)
		bump := 0.12*math.Sin(float64(i)*0.9+phase) + 0.08*math.Sin(float64(i)*2.3+phase*2)
		rk.tops = append(rk.tops, seaRow-height*shape*(1+bump)+1.5)
	}
	return rk
}

// Step advances the animation by dt.
func (s *Scene) Step(dt time.Duration) {
	sec := max(dt, 0).Seconds()
	s.t += sec
	if s.w < MinWidth {
		return
	}
	W := float64(s.w * 2)

	if s.wait -= sec; s.wait <= 0 {
		s.swells = append(s.swells, &swell{x: W + 3*waveSigma, amp: waveAmp * (0.75 + 0.5*s.rng.Float64()), hits: map[int]bool{}})
		s.wait = waveMin + s.rng.Float64()*waveSpread
	}

	live := s.swells[:0]
	for _, sw := range s.swells {
		sw.x -= waveSpeed * sec
		for i, rk := range s.rocks {
			face := rk.cx + rk.r*0.6
			if !sw.hits[i] && sw.x <= face {
				sw.hits[i] = true
				s.splash(rk, sw.amp)
				sw.amp *= rockDamp
			}
		}
		if sw.x <= s.shoreCol+waveSigma {
			s.startRunup(sw.amp)
			continue
		}
		live = append(live, sw)
	}
	s.swells = live

	if s.running {
		// The front climbs the beach, slows, then slides back.
		s.runupV += 18 * sec // decelerates the leftward climb
		s.runup += s.runupV * sec
		lo := int(max(s.runup, 0))
		for x := lo; x < int(s.shoreCol) && x < len(s.wet); x++ {
			s.wet[x] = 1
		}
		if s.runup >= s.shoreCol {
			s.running = false
		}
	}
	for x := range s.wet {
		s.wet[x] = max(0, s.wet[x]-sec/4)
	}

	alive := s.drops[:0]
	for _, d := range s.drops {
		d.age += sec
		if d.age >= d.life {
			continue
		}
		d.vy += 40 * sec // gravity, dots per second squared
		d.x += d.vx * sec
		d.y += d.vy * sec
		if d.y > seaRow+1 {
			continue
		}
		alive = append(alive, d)
	}
	s.drops = alive
	s.render()
}

// splash throws spray up the seaward face of rk, more for a bigger swell.
func (s *Scene) splash(rk rock, amp float64) {
	n := int(18 * amp)
	face := rk.cx + rk.r*0.5
	for range n {
		up := 14 + s.rng.Float64()*10*amp/waveAmp + rk.height*0.6
		s.drops = append(s.drops, drop{
			x:    face + s.rng.Float64()*rk.r*0.6,
			y:    seaRow - 1 - s.rng.Float64()*3,
			vx:   (s.rng.Float64() - 0.3) * 14,
			vy:   -up,
			life: 0.9 + s.rng.Float64()*0.8,
			col:  sprayCol,
		})
	}
}

// startRunup sends a sheet of water up the sand.
func (s *Scene) startRunup(amp float64) {
	s.running = true
	s.runup = s.shoreCol
	s.runupV = -(10 + 8*amp/waveAmp) // leftwards, dots per second
}

// Glyph returns the rune at screen (x, y), or 0 for empty. The scene hides
// whole when its strip would touch the calendar box b.
func (s *Scene) Glyph(x, y int, b scene.Box) rune {
	if s.w < MinWidth || b.Y1 >= s.h-Height {
		return 0
	}
	return s.canvas.Glyph(x, y-(s.h-Height))
}

// RGB returns the colour of the glyph at (x, y).
func (s *Scene) RGB(x, y int, _ scene.Box) scene.RGB {
	return s.canvas.RGB(x, y-(s.h-Height))
}

func (s *Scene) dot(x, y float64, col scene.RGB) {
	if y < 0 || y >= Height*4 {
		return
	}
	s.canvas.Dot(x/2, y/4, col)
}

// sandRow is the dot row of the beach surface at column x.
func (s *Scene) sandRow(x float64) float64 {
	return sandTop + (seaRow+3-sandTop)*x/s.shoreCol
}

// surface is the dot row of the sea surface at column x.
func (s *Scene) surface(x float64) float64 {
	y := seaRow + 0.6*math.Sin(x*0.21-s.t*1.7) + 0.4*math.Sin(x*0.08+s.t*0.9)
	for _, sw := range s.swells {
		d := (x - sw.x) / waveSigma
		y -= sw.amp * math.Exp(-d*d)
	}
	return y
}

func scale(c scene.RGB, k float64) scene.RGB {
	f := func(x uint8) uint8 { return uint8(float64(x)*k + 0.5) }
	return scene.RGB{R: f(c.R), G: f(c.G), B: f(c.B)}
}

func mix(a, b scene.RGB, t float64) scene.RGB {
	f := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return scene.RGB{R: f(a.R, b.R), G: f(a.G, b.G), B: f(a.B, b.B)}
}

// hash maps (x, y) to a stable value in [0, 1).
func hash(x, y int) float64 {
	z := uint64(x)*0x9e3779b97f4a7c15 ^ uint64(y)*0xbf58476d1ce4e5b9
	z = (z ^ z>>30) * 0xbf58476d1ce4e5b9
	z = (z ^ z>>27) * 0x94d049bb133111eb
	z ^= z >> 31
	return float64(z>>11) / (1 << 53)
}

func (s *Scene) render() {
	s.canvas.Clear()
	if s.w < MinWidth {
		return
	}
	W := s.w * 2
	H := Height * 4

	for x := range W {
		fx := float64(x)
		top := float64(H)
		if fx < s.shoreCol+6 {
			top = s.sandRow(fx)
		}
		// Sea: a bright surface line over sparse, darkening dots.
		if fx >= s.shoreCol-4 {
			sy := s.surface(fx)
			if sy < top {
				s.dot(fx, sy, surfCol)
				for y := int(sy) + 2; y < min(H, int(top)); y++ {
					depth := float64(y) - sy
					if hash(x, y) < 0.16-depth*0.005 {
						s.dot(fx, float64(y), mix(seaCol, deepCol, min(1, depth/10)))
					}
				}
			}
		}
		// Sand: a solid surface over speckles, darker where wet.
		if fx < s.shoreCol+6 {
			wet := 0.0
			if x < len(s.wet) {
				wet = s.wet[x]
			}
			col := mix(sandCol, wetCol, wet)
			s.dot(fx, top, col)
			for y := int(top) + 1; y < H; y++ {
				if hash(x, y) < 0.1 {
					s.dot(fx, float64(y), col)
				}
			}
		}
	}

	// The wash: a thin sheet with foam at its leading edge.
	if s.running {
		for x := int(max(s.runup, 0)); x < int(s.shoreCol); x++ {
			fx := float64(x)
			col := surfCol
			if fx < s.runup+3 {
				col = foamCol
			}
			s.dot(fx, s.sandRow(fx)-1, col)
		}
	}

	for _, rk := range s.rocks {
		for i, top := range rk.tops {
			x := rk.cx - rk.r + float64(i)
			// Facets: a shade per cell column, so neighbouring dots in
			// one cell always share a colour.
			facet := 0.85 + 0.3*hash(int(x)/2, 7)
			for y := int(top); y < H; y++ {
				depth := float64(y) - top
				col := mix(edgeCol, scale(rockCol, facet), min(1, depth/3))
				if float64(y) > seaRow {
					col = mix(col, deepCol, 0.7)
				}
				s.dot(x, float64(y), col)
			}
		}
	}

	for _, d := range s.drops {
		k := 1 - d.age/d.life
		s.dot(d.x, d.y, scale(d.col, 0.4+0.6*k))
	}
}

// Package lava is the smooth lava-lamp scene: metaballs drift slowly around
// the screen, bouncing off its edges, and merge and split as they pass each
// other. The field is sampled per braille dot and graded by heat.
package lava

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
	// minBlobs and maxBlobs bound the number of metaballs.
	minBlobs, maxBlobs = 3, 6
	// aspect is the height of a terminal cell relative to its width; x
	// distances are divided by it so blobs look round.
	aspect = 2.0
	// rimLevel is the field threshold for the blob edge. A lone blob of
	// radius r reaches it at distance r. heatLevel is the field at which the
	// colour is fully hot.
	rimLevel, heatLevel = 1.0, 3.2
	// reach is how far, in radii, a blob contributes to the field. Beyond it
	// a blob adds under 1/reach² and is ignored, which bounds the work to
	// each blob's bounding box.
	reach = 2.0
	// minSpeed and maxSpeed bound blob speed, in rows per second
	// (0.04-0.09 rows per 100 ms).
	minSpeed, maxSpeed = 0.4, 0.9
)

var (
	coolCol = scene.RGB{R: 150, G: 40, B: 20}
	hotCol  = scene.RGB{R: 230, G: 140, B: 40}
)

// blob is one metaball. x is in columns, y and r in rows; velocities are per
// second.
type blob struct {
	x, y, vx, vy, r float64
}

// Scene is a lava lamp filling the whole screen.
type Scene struct {
	rng    *rand.Rand
	w, h   int
	blobs  []blob
	field  []float32 // summed field per dot, row-major, 2w x 4h
	canvas dots.Canvas
}

// New returns a w×h lava lamp that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.Resize(w, h)
	return s
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return step }

// Tick advances one interval.
func (s *Scene) Tick() { s.Step(step) }

// Resize scatters a fresh set of blobs over a w×h screen.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.canvas.Reset(s.w, s.h)
	s.field = make([]float32, s.w*2*s.h*4)
	s.blobs = s.blobs[:0]
	if s.w > 0 && s.h > 0 {
		// Keep blobs modest: roughly a ninth of the screen height.
		base := min(max(float64(s.h)/9, 1.5), 5)
		for range minBlobs + s.rng.IntN(maxBlobs-minBlobs+1) {
			a := s.rng.Float64() * 2 * math.Pi
			v := minSpeed + s.rng.Float64()*(maxSpeed-minSpeed)
			s.blobs = append(s.blobs, blob{
				x:  s.rng.Float64() * float64(s.w),
				y:  s.rng.Float64() * float64(s.h),
				vx: math.Cos(a) * v * aspect,
				vy: math.Sin(a) * v,
				r:  base * (0.7 + 0.6*s.rng.Float64()),
			})
		}
	}
	s.render()
}

// Step moves every blob by dt, bouncing off the screen edges.
func (s *Scene) Step(dt time.Duration) {
	sec := max(dt, 0).Seconds()
	for i := range s.blobs {
		b := &s.blobs[i]
		b.x, b.vx = bounce(b.x+b.vx*sec, b.vx, float64(s.w))
		b.y, b.vy = bounce(b.y+b.vy*sec, b.vy, float64(s.h))
	}
	s.render()
}

// bounce reflects position p (moving at v) back into [0, limit].
func bounce(p, v, limit float64) (float64, float64) {
	switch {
	case p < 0:
		return min(-p, limit), math.Abs(v)
	case p > limit:
		return max(2*limit-p, 0), -math.Abs(v)
	}
	return p, v
}

// Glyph returns the rune at screen (x, y), or 0 for empty.
func (s *Scene) Glyph(x, y int, _ scene.Box) rune { return s.canvas.Glyph(x, y) }

// RGB returns the colour of the glyph at (x, y).
func (s *Scene) RGB(x, y int, _ scene.Box) scene.RGB { return s.canvas.RGB(x, y) }

func mix(a, b scene.RGB, t float64) scene.RGB {
	f := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return scene.RGB{R: f(a.R, b.R), G: f(a.G, b.G), B: f(a.B, b.B)}
}

// heat is the colour for field strength f: cool at the rim, hot in the core.
func heat(f float32) scene.RGB {
	t := (float64(f) - rimLevel) / (heatLevel - rimLevel)
	return mix(coolCol, hotCol, min(max(t, 0), 1))
}

// accumulate adds every blob's Σ r²/d² to the dot field, sampling only the
// dots inside each blob's bounding box.
func (s *Scene) accumulate() {
	clear(s.field)
	dw, dh := s.w*2, s.h*4
	for _, b := range s.blobs {
		rx, ry := b.r*reach*aspect, b.r*reach // bounding box half-extent, cells
		x0 := max(int(math.Floor((b.x-rx)*2)), 0)
		x1 := min(int(math.Ceil((b.x+rx)*2)), dw-1)
		y0 := max(int(math.Floor((b.y-ry)*4)), 0)
		y1 := min(int(math.Ceil((b.y+ry)*4)), dh-1)
		r2, cut := b.r*b.r, (b.r*reach)*(b.r*reach)
		for y := y0; y <= y1; y++ {
			dy := (float64(y)+0.5)/4 - b.y
			row := s.field[y*dw : (y+1)*dw]
			for x := x0; x <= x1; x++ {
				dx := ((float64(x)+0.5)/2 - b.x) / aspect
				if d2 := dx*dx + dy*dy; d2 < cut {
					row[x] += float32(r2 / (d2 + 1e-6))
				}
			}
		}
	}
}

func (s *Scene) render() {
	s.canvas.Clear()
	if s.w == 0 || s.h == 0 {
		return
	}
	s.accumulate()
	dw := s.w * 2
	for cy := range s.h {
		for cx := range s.w {
			var on [8]bool
			peak, any := float32(0), false
			for r := range 4 {
				row := s.field[(cy*4+r)*dw+cx*2:]
				for c := range 2 {
					if f := row[c]; f >= rimLevel {
						on[r*2+c], any = true, true
						peak = max(peak, f)
					}
				}
			}
			if !any {
				continue
			}
			col := heat(peak)
			for i, d := range on {
				if d {
					s.canvas.Dot(float64(cx)+float64(i%2)/2, float64(cy)+float64(i/2)/4, col)
				}
			}
		}
	}
}

// Package lava implements a lava-lamp background: a handful of metaballs
// drift slowly around the screen, bouncing off its edges, and merge and
// split as they pass each other.
package lava

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

const (
	// minBlobs and maxBlobs bound the number of metaballs.
	minBlobs, maxBlobs = 3, 6
	// aspect is the height of a terminal cell relative to its width; x
	// distances are divided by it so blobs look round.
	aspect = 2.0
	// rimLevel and coreLevel are the field thresholds for the blob rim
	// and interior. A lone blob of radius r reaches rimLevel at distance r.
	rimLevel, coreLevel = 1.0, 2.2
	// minSpeed and maxSpeed bound blob speed, in rows per tick.
	minSpeed, maxSpeed = 0.04, 0.09
)

const (
	rimGlyph  = '·'
	coreGlyph = '░'
)

// blobColors are the hues blobs are tinted with.
var blobColors = []scene.Color{scene.Red, scene.Magenta, scene.Yellow}

// blob is one metaball. x is in columns, y and r in rows; hue indexes
// blobColors and is fixed for the blob's life.
type blob struct {
	x, y, vx, vy, r float64
	hue             int
}

// Scene is a lava lamp filling the whole screen.
type Scene struct {
	w, h  int
	blobs []blob
	buf   []rune        // rendered frame, row-major
	col   []scene.Color // color of each buf cell
	made  uint64        // blobs created so far; seeds each blob's hue
	rng   *rand.Rand
}

var (
	_ scene.Scene   = (*Scene)(nil)
	_ scene.Colored = (*Scene)(nil)
)

// New returns a w×h lava lamp that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.Resize(w, h)
	return s
}

// Resize scatters a fresh set of blobs over a w×h screen.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.buf = make([]rune, s.w*s.h)
	s.col = make([]scene.Color, s.w*s.h)
	s.blobs = s.blobs[:0]
	if s.w > 0 && s.h > 0 {
		// Keep blobs modest: roughly a ninth of the screen height.
		base := min(max(float64(s.h)/9, 1.5), 5)
		for range minBlobs + s.rng.IntN(maxBlobs-minBlobs+1) {
			a := s.rng.Float64() * 2 * math.Pi
			v := minSpeed + s.rng.Float64()*(maxSpeed-minSpeed)
			s.blobs = append(s.blobs, blob{
				x:   s.rng.Float64() * float64(s.w),
				y:   s.rng.Float64() * float64(s.h),
				vx:  math.Cos(a) * v * aspect,
				vy:  math.Sin(a) * v,
				r:   base * (0.7 + 0.6*s.rng.Float64()),
				hue: s.nextHue(),
			})
		}
	}
	s.render()
}

// Tick moves every blob one step, bouncing off the screen edges.
func (s *Scene) Tick() {
	for i := range s.blobs {
		b := &s.blobs[i]
		b.x, b.vx = bounce(b.x+b.vx, b.vx, float64(s.w))
		b.y, b.vy = bounce(b.y+b.vy, b.vy, float64(s.h))
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

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return 100 * time.Millisecond }

// Glyph returns the shade at (x, y), or 0 outside every blob.
func (s *Scene) Glyph(x, y int, _ scene.Box) rune {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return 0
	}
	return s.buf[y*s.w+x]
}

// Color returns the hue of the blob contributing most to the field at
// (x, y), so merging blobs meet at a boundary rather than mixing.
func (s *Scene) Color(x, y int, _ scene.Box) scene.Color {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return scene.Default
	}
	return s.col[y*s.w+x]
}

// nextHue picks a blob hue without drawing from rng, so the animation's
// random sequence is unaffected by color.
func (s *Scene) nextHue() int {
	s.made++
	z := s.made + 0x9e3779b97f4a7c15 // splitmix64 finalizer
	z = (z ^ z>>30) * 0xbf58476d1ce4e5b9
	z = (z ^ z>>27) * 0x94d049bb133111eb
	return int((z ^ z>>31) % uint64(len(blobColors)))
}

// field returns the summed metaball field Σ r²/d² at cell (x, y), and the
// index of the blob contributing most to it (-1 if there are none).
func (s *Scene) field(x, y int) (float64, int) {
	px, py := (float64(x)+0.5)/aspect, float64(y)+0.5
	f, top, best := 0.0, 0.0, -1
	for i, b := range s.blobs {
		dx, dy := px-b.x/aspect, py-b.y
		c := b.r * b.r / (dx*dx + dy*dy + 1e-6)
		f += c
		if c > top {
			top, best = c, i
		}
	}
	return f, best
}

func (s *Scene) render() {
	for y := range s.h {
		for x := range s.w {
			var r rune
			c := scene.Default
			f, top := s.field(x, y)
			switch {
			case f >= coreLevel:
				r = coreGlyph
			case f >= rimLevel:
				r = rimGlyph
			}
			if r != 0 {
				c = blobColors[s.blobs[top].hue%len(blobColors)]
			}
			s.buf[y*s.w+x], s.col[y*s.w+x] = r, c
		}
	}
}

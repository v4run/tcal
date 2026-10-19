// Package aquarium implements a quiet fish tank, used as an ambient
// background in live mode. A few fish wander left and right, bubbles rise
// from the bottom and pop at the surface, and seaweed sways along the floor.
package aquarium

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

var _ scene.Scene = (*Scene)(nil)

// interval is the delay between Ticks.
const interval = 100 * time.Millisecond

const (
	cellsPerFish = 700 // screen cells per fish; higher is an emptier tank
	maxFish      = 8   // cap on fish however large the screen
	minSpeed     = 0.1 // slowest swim, in columns per Tick
	speedSpread  = 0.3 // extra random swim speed, in columns per Tick
	driftChance  = 0.01
	turnChance   = 0.003
)

const (
	bubbleChance   = 0.04 // chance per Tick per 100 columns of a new bubble
	bubbleMin      = 0.15 // slowest rise, in rows per Tick
	bubbleSpread   = 0.2  // extra random rise speed, in rows per Tick
	bubblePopGlyph = '*'  // drawn for one Tick where a bubble bursts
)

const (
	colsPerWeed  = 22 // screen columns per seaweed strand
	weedMaxH     = 5  // tallest strand, in rows
	swayPeriod   = 8  // Ticks between seaweed sways
	minTankWidth = 10 // narrower screens get no seaweed
)

// shape is a fish drawn facing right and facing left.
type shape struct{ right, left string }

var shapes = []shape{
	{"><>", "<><"},
	{"><>", "<><"},
	{"><))>", "<((><"},
	{"><(((('>", "<')))><"},
}

type fish struct {
	x     float64 // column of the leftmost cell
	y     int
	dir   int     // +1 swims right, -1 left
	speed float64 // columns per Tick
	shape int
}

func (f fish) art() string {
	if f.dir > 0 {
		return shapes[f.shape].right
	}
	return shapes[f.shape].left
}

type bubble struct {
	x      int
	y      float64
	speed  float64
	popped bool // burst on the previous Tick; removed on the next
}

type weed struct{ x, h, phase int }

// Scene is the tank. All state is rendered into buf on every Tick and
// Resize so Glyph is a plain lookup.
type Scene struct {
	w, h    int
	fish    []fish
	bubbles []bubble
	weeds   []weed
	ticks   int
	buf     []rune
	rng     *rand.Rand
}

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.Resize(w, h)
	return s
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return interval }

// Resize changes the screen size. Fish keep swimming where they still fit
// (clamped to the new depth), the population is adjusted to the new area,
// bubbles that no longer fit are dropped, and seaweed is replanted.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.buf = make([]rune, s.w*s.h)

	want := s.target()
	if len(s.fish) > want {
		s.fish = s.fish[:want]
	}
	for i := range s.fish {
		f := &s.fish[i]
		f.y = clamp(f.y, 0, s.maxFishRow())
		if int(f.x) >= s.w {
			*f = s.spawn(false)
		}
	}
	for len(s.fish) < want {
		s.fish = append(s.fish, s.spawn(true))
	}

	kept := s.bubbles[:0]
	for _, b := range s.bubbles {
		if b.x < s.w && int(b.y) < s.h {
			kept = append(kept, b)
		}
	}
	s.bubbles = kept

	s.plant()
	s.render()
}

// Tick moves fish and bubbles and sways the seaweed.
func (s *Scene) Tick() {
	s.ticks++
	if s.w == 0 || s.h == 0 {
		return
	}

	for i := range s.fish {
		f := &s.fish[i]
		f.x += float64(f.dir) * f.speed
		if s.rng.Float64() < driftChance {
			f.y = clamp(f.y+s.rng.IntN(3)-1, 0, s.maxFishRow())
		}
		if s.rng.Float64() < turnChance {
			f.dir = -f.dir
		}
		if n := len(f.art()); f.x+float64(n) < 0 || f.x >= float64(s.w) {
			*f = s.spawn(false)
		}
	}

	kept := s.bubbles[:0]
	for _, b := range s.bubbles {
		if b.popped {
			continue
		}
		b.y -= b.speed
		if b.y < 1 {
			b.y, b.popped = 0, true
		}
		kept = append(kept, b)
	}
	s.bubbles = kept
	if s.rng.Float64() < bubbleChance*float64(s.w)/100 {
		s.bubbles = append(s.bubbles, s.newBubble())
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

// target is the number of fish for the current screen size.
func (s *Scene) target() int {
	if s.w == 0 || s.h == 0 {
		return 0
	}
	return min(maxFish, max(1, s.w*s.h/cellsPerFish))
}

// maxFishRow is the lowest row a fish swims in, keeping clear of the floor
// where it can.
func (s *Scene) maxFishRow() int { return max(0, s.h-2) }

// spawn returns a new fish. Scattered fish start anywhere on screen (used
// to stock an empty tank); others enter from just off the edge they face
// away from.
func (s *Scene) spawn(scattered bool) fish {
	f := fish{
		y:     s.rng.IntN(s.maxFishRow() + 1),
		dir:   1,
		speed: minSpeed + s.rng.Float64()*speedSpread,
		shape: s.rng.IntN(len(shapes)),
	}
	if s.rng.IntN(2) == 0 {
		f.dir = -1
	}
	n := float64(len(f.art()))
	switch {
	case scattered:
		f.x = s.rng.Float64() * float64(s.w)
	case f.dir > 0:
		f.x = -n + 0.5
	default:
		f.x = float64(s.w) - 0.5
	}
	return f
}

// newBubble starts a bubble on the floor, preferring to rise from seaweed.
func (s *Scene) newBubble() bubble {
	b := bubble{x: s.rng.IntN(s.w), y: float64(s.h - 1), speed: bubbleMin + s.rng.Float64()*bubbleSpread}
	if len(s.weeds) > 0 && s.rng.IntN(2) == 0 {
		wd := s.weeds[s.rng.IntN(len(s.weeds))]
		b.x, b.y = wd.x, float64(s.h-1-wd.h)
	}
	return b
}

// plant spreads seaweed strands evenly-ish across the floor.
func (s *Scene) plant() {
	s.weeds = s.weeds[:0]
	if s.w < minTankWidth || s.h < 6 {
		return
	}
	n := max(1, s.w/colsPerWeed)
	span := s.w / n
	for i := range n {
		s.weeds = append(s.weeds, weed{
			x:     i*span + s.rng.IntN(span),
			h:     2 + s.rng.IntN(min(weedMaxH, s.h/3)-1),
			phase: s.rng.IntN(2),
		})
	}
}

func (s *Scene) render() {
	clear(s.buf)
	sway := s.ticks / swayPeriod
	for _, wd := range s.weeds {
		for i := range wd.h {
			r := '('
			if (i+wd.phase+sway)%2 == 1 {
				r = ')'
			}
			s.set(wd.x, s.h-1-i, r)
		}
	}
	for _, b := range s.bubbles {
		s.set(b.x, int(b.y), s.bubbleGlyph(b))
	}
	for _, f := range s.fish {
		x0 := int(math.Floor(f.x)) // not int(), which rounds toward zero off the left edge
		for i, r := range f.art() {
			s.set(x0+i, f.y, r)
		}
	}
}

// bubbleGlyph grows a bubble as it rises: small near the floor, larger
// near the surface.
func (s *Scene) bubbleGlyph(b bubble) rune {
	switch {
	case b.popped:
		return bubblePopGlyph
	case b.y > float64(s.h)*2/3:
		return '.'
	case b.y > float64(s.h)/3:
		return 'o'
	default:
		return 'O'
	}
}

// set draws r at (x, y), ignoring cells off screen.
func (s *Scene) set(x, y int, r rune) {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return
	}
	s.buf[y*s.w+x] = r
}

func clamp(v, lo, hi int) int { return max(lo, min(v, hi)) }

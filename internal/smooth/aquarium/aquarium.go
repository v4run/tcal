// Package aquarium is the smooth fish tank: braille-dot fish glide left and
// right with a gentle bob, turning now and then; bubbles rise at sub-cell
// speed and pop at the surface; seaweed sways along the floor.
package aquarium

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/dots"
	"github.com/v4run/tcal/internal/scene"
)

const step = 33 * time.Millisecond

// Fish, from classic's 100 ms ticks: 0.1-0.4 columns per tick, and a
// 0.3% chance per tick of turning.
const (
	cellsPerFish = 700 // screen cells per fish; higher is an emptier tank
	maxFish      = 8
	minSpeed     = 1.0  // columns per second
	speedSpread  = 3.0  // extra random columns per second
	turnRate     = 0.03 // turns per fish per second
	bobAmp       = 0.3  // rows either side of a fish's depth
	minBob       = 3.0  // fastest bob, seconds per cycle
	bobSpread    = 3.0  // extra random seconds per cycle
)

// Bubbles, from classic: 0.04 per tick per 100 columns, rising 0.15-0.35
// rows per tick.
const (
	bubbleRate   = 0.4 // new bubbles per second per 100 columns
	bubbleMin    = 1.5 // rows per second
	bubbleSpread = 2.0
	wobbleAmp    = 0.25 // columns either side
	popLife      = 0.15 // seconds a burst shows
)

// Seaweed, from classic: a strand per 22 columns, up to 5 rows tall.
const (
	colsPerWeed  = 22
	weedMaxH     = 5
	minTankWidth = 10
	swayPeriod   = 3.2 // seconds per sway
	swayAmp      = 0.8 // columns at the tip
)

var (
	// fishCols are muted goldfish hues.
	fishCols = []scene.RGB{
		{R: 210, G: 165, B: 80},
		{R: 205, G: 120, B: 95},
		{R: 180, G: 120, B: 160},
	}
	bubbleCol = scene.RGB{R: 150, G: 195, B: 215}
	weedCol   = scene.RGB{R: 80, G: 145, B: 95}
)

var (
	_ scene.Scene       = (*Scene)(nil)
	_ scene.Stepper     = (*Scene)(nil)
	_ scene.TrueColored = (*Scene)(nil)
)

// shape is a right-facing fish sprite.
type shape struct {
	w, h int
	dots [][2]int
}

func parse(rows ...string) shape {
	s := shape{h: len(rows)}
	for y, row := range rows {
		s.w = max(s.w, len(row))
		for x, ch := range row {
			if ch == '#' {
				s.dots = append(s.dots, [2]int{x, y})
			}
		}
	}
	return s
}

// shapes echo classic's ><>, ><))> and ><(((('>: a forked tail on the left,
// an eye gap near the nose on the right. Small fish are twice as common.
var shapes = []shape{
	parse(
		"#...####.",
		".#.####.#",
		".#.######",
		"#...####.",
	),
	parse(
		"#...####.",
		".#.####.#",
		".#.######",
		"#...####.",
	),
	parse(
		"#.....####...",
		".#..######.#.",
		"..###########",
		".#..########.",
		"#.....####...",
	),
	parse(
		"#.......#####....",
		"##....#########..",
		".##..########.##.",
		"..###############",
		".##..###########.",
		"##....#########..",
		"#.......#####....",
	),
}

type fish struct {
	x      float64 // column of the left edge
	base   float64 // row of the top edge, before bobbing
	dir    int     // +1 swims right, -1 left
	speed  float64 // columns per second
	shape  int
	hue    int
	period float64 // seconds per bob
	phase  float64
}

// turn reverses the fish in place; it is drawn mirrored from then on.
func (f *fish) turn() { f.dir = -f.dir }

func (f *fish) width() float64 { return float64(shapes[f.shape].w) / 2 }

type bubble struct {
	x, x0  float64 // drawn column and the column it wobbles about
	y      float64
	speed  float64 // rows per second
	age    float64
	popped float64 // seconds since bursting, 0 while rising
}

type weed struct {
	x, h  int
	phase float64
}

// Scene is the tank.
type Scene struct {
	rng        *rand.Rand
	w, h       int
	t          float64
	fish       []fish
	bubbles    []bubble
	weeds      []weed
	turns      int     // turns so far
	turnRate   float64 // per fish per second
	bubbleRate float64 // per second per 100 columns
	spawned    int
	canvas     dots.Canvas
}

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng, turnRate: turnRate, bubbleRate: bubbleRate}
	s.Resize(w, h)
	return s
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return step }

// Tick advances one interval.
func (s *Scene) Tick() { s.Step(step) }

// Resize changes the screen size: fish that still fit keep swimming, the
// population follows the new area, stray bubbles go and seaweed is
// replanted.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	want := s.target()
	if len(s.fish) > want {
		s.fish = s.fish[:want]
	}
	for i := range s.fish {
		f := &s.fish[i]
		f.base = min(f.base, s.maxBase(f.shape))
		if f.x >= float64(s.w) {
			*f = s.spawn(false)
		}
	}
	for len(s.fish) < want {
		s.fish = append(s.fish, s.spawn(true))
	}
	kept := s.bubbles[:0]
	for _, b := range s.bubbles {
		if b.x < float64(s.w) && b.y < float64(s.h) {
			kept = append(kept, b)
		}
	}
	s.bubbles = kept
	s.plant()
	s.canvas.Reset(s.w, s.h)
	s.render()
}

// Step moves fish and bubbles and sways the seaweed by dt.
func (s *Scene) Step(dt time.Duration) {
	sec := max(dt, 0).Seconds()
	s.t += sec
	if s.w == 0 || s.h == 0 {
		return
	}
	pTurn := 1 - math.Exp(-s.turnRate*sec)
	for i := range s.fish {
		f := &s.fish[i]
		f.x += float64(f.dir) * f.speed * sec
		if s.rng.Float64() < pTurn {
			f.turn()
			s.turns++
		}
		if f.x+f.width() < 0 || f.x >= float64(s.w) {
			*f = s.spawn(false)
		}
	}

	kept := s.bubbles[:0]
	for _, b := range s.bubbles {
		if b.popped > 0 {
			if b.popped += sec; b.popped < popLife {
				kept = append(kept, b)
			}
			continue
		}
		b.age += sec
		b.y -= b.speed * sec
		b.x = b.x0 + wobbleAmp*math.Sin(b.age*5)
		if b.y < 0.25 {
			b.y, b.popped = 0, 1e-9
		}
		kept = append(kept, b)
	}
	s.bubbles = kept
	if s.rng.Float64() < s.bubbleRate*float64(s.w)/100*sec {
		s.bubbles = append(s.bubbles, s.newBubble())
	}
	s.render()
}

// Glyph returns the rune at screen (x, y), or 0 for empty.
func (s *Scene) Glyph(x, y int, _ scene.Box) rune { return s.canvas.Glyph(x, y) }

// RGB returns the colour of the glyph at (x, y).
func (s *Scene) RGB(x, y int, _ scene.Box) scene.RGB { return s.canvas.RGB(x, y) }

func (s *Scene) target() int {
	if s.w == 0 || s.h == 0 {
		return 0
	}
	return min(maxFish, max(1, s.w*s.h/cellsPerFish))
}

// maxBase is the deepest a fish of the given shape swims, clear of the
// floor where it can be.
func (s *Scene) maxBase(shape int) float64 {
	return max(0, float64(s.h)-1-float64(shapes[shape].h)/4-bobAmp)
}

// spawn returns a new fish: anywhere on screen when scattered, otherwise
// just off the edge it swims in from.
func (s *Scene) spawn(scattered bool) fish {
	s.spawned++
	f := fish{
		dir:    1,
		speed:  minSpeed + s.rng.Float64()*speedSpread,
		shape:  s.rng.IntN(len(shapes)),
		hue:    s.spawned % len(fishCols),
		period: minBob + s.rng.Float64()*bobSpread,
		phase:  s.rng.Float64() * 2 * math.Pi,
	}
	lo := min(bobAmp, s.maxBase(f.shape))
	f.base = lo + s.rng.Float64()*(s.maxBase(f.shape)-lo)
	if s.rng.IntN(2) == 0 {
		f.dir = -1
	}
	switch {
	case scattered:
		f.x = s.rng.Float64() * float64(s.w)
	case f.dir > 0:
		f.x = -f.width() + 0.5
	default:
		f.x = float64(s.w) - 0.5
	}
	return f
}

// newBubble starts a bubble on the floor, often from a seaweed tip.
func (s *Scene) newBubble() bubble {
	b := bubble{x: s.rng.Float64() * float64(s.w), y: float64(s.h) - 0.25, speed: bubbleMin + s.rng.Float64()*bubbleSpread}
	if len(s.weeds) > 0 && s.rng.IntN(2) == 0 {
		wd := s.weeds[s.rng.IntN(len(s.weeds))]
		b.x, b.y = float64(wd.x)+0.5, float64(s.h-wd.h)
	}
	b.x0 = b.x
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
			phase: s.rng.Float64() * 2 * math.Pi,
		})
	}
}

func (s *Scene) render() {
	s.canvas.Clear()
	if s.w == 0 || s.h == 0 {
		return
	}
	floor := s.h*4 - 1
	for _, wd := range s.weeds {
		n := wd.h * 4
		for i := range n {
			frac := float64(i) / float64(n)
			x := float64(wd.x) + 0.5 + swayAmp*frac*math.Sin(2*math.Pi*s.t/swayPeriod+wd.phase-float64(i)*0.15)
			y := float64(floor-i) / 4
			s.canvas.Dot(x, y, weedCol)
			if i%4 == 2 && i < n-1 { // leaves on alternate sides
				side := 0.5
				if (i/4)%2 == 1 {
					side = -0.5
				}
				s.canvas.Dot(x+side, y, weedCol)
			}
		}
	}
	for _, b := range s.bubbles {
		s.drawBubble(b)
	}
	for _, f := range s.fish {
		sh := shapes[f.shape]
		y := f.base + bobAmp*math.Sin(2*math.Pi*s.t/f.period+f.phase)
		c := fishCols[f.hue%len(fishCols)]
		for _, d := range sh.dots {
			dx := d[0]
			if f.dir < 0 {
				dx = sh.w - 1 - dx
			}
			s.canvas.Dot(f.x+float64(dx)/2, y+float64(d[1])/4, c)
		}
	}
}

// drawBubble grows a bubble as it rises, a dot near the floor to a ring
// near the surface, and draws a burst once it pops.
func (s *Scene) drawBubble(b bubble) {
	c := s.canvas.Dot
	h := float64(s.h)
	switch {
	case b.popped > 0:
		r := 0.5 + b.popped/popLife
		c(b.x-r/2, b.y, bubbleCol)
		c(b.x+r/2, b.y, bubbleCol)
		c(b.x-r/4, b.y+0.25, bubbleCol)
		c(b.x+r/4, b.y+0.25, bubbleCol)
	case b.y > h*2/3:
		c(b.x, b.y, bubbleCol)
	case b.y > h/3:
		c(b.x, b.y, bubbleCol)
		c(b.x+0.5, b.y, bubbleCol)
		c(b.x, b.y+0.25, bubbleCol)
		c(b.x+0.5, b.y+0.25, bubbleCol)
	default:
		c(b.x, b.y-0.25, bubbleCol)
		c(b.x-0.5, b.y, bubbleCol)
		c(b.x+0.5, b.y, bubbleCol)
		c(b.x, b.y+0.25, bubbleCol)
	}
}

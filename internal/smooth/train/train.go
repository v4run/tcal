// Package train is the smooth railway scene: a rail line along the bottom
// of the screen that sits empty most of the time, with a freight train of
// braille-dot cars gliding through every few minutes, trailing smoke.
package train

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/dots"
	"github.com/v4run/tcal/internal/scene"
)

// Height is the number of rows the strip occupies at the bottom of the
// screen, the same as classic: the rail, three rows of car and the smoke.
const Height = 5

// MinWidth is the narrowest screen the scene will draw on.
const MinWidth = 40

// Classic timing, converted from its 100 ms ticks.
const (
	step       = 33 * time.Millisecond
	classicTk  = 100 * time.Millisecond
	firstWait  = 30 * classicTk   // max wait before the first train
	minWait    = 600 * classicTk  // minimum wait between trains
	waitSpread = 1800 * classicTk // extra random wait between trains
	minCars    = 3                // fewest cars behind the locomotive
	carSpread  = 6                // cars are minCars..minCars+carSpread-1
	bigEvery   = 8                // about one car in this many is the big container
)

// speed is the train's speed in columns per second: classic moves one
// column per tick.
const speed = 1 / 0.1

// Strip dot rows (Height*4 = 20): smoke 0-3, car bodies 4-11, frame and
// wheels 12-15, rail 16-19.
const (
	bodyTop  = 4
	frameRow = 12
	railRow  = 16
	tieEvery = 8 // dots between rail ties (classic: 4 columns)
	coupler  = 2 // dots between cars
)

// Smoke: puffs leave the chimney every puffEvery seconds, rise riseSpeed
// rows per second, drift back driftSpeed columns per second and fade over
// puffLife seconds.
const (
	puffEvery  = 0.09
	riseSpeed  = 1.3
	driftSpeed = 1.5
	puffLife   = 0.7
)

var (
	locoCol  = scene.RGB{R: 185, G: 155, B: 85}
	ironCol  = scene.RGB{R: 105, G: 105, B: 110}
	railCol  = scene.RGB{R: 130, G: 125, B: 115}
	bigCol   = scene.RGB{R: 175, G: 80, B: 65} // the container scene's container
	smokeNew = scene.RGB{R: 200, G: 200, B: 195}
	smokeOld = scene.RGB{R: 105, G: 105, B: 105}
	// boxCols are a flatcar's container paints, the harbour's.
	boxCols = []scene.RGB{
		{R: 170, G: 85, B: 70},
		{R: 190, G: 150, B: 80},
		{R: 90, G: 125, B: 160},
		{R: 105, G: 145, B: 100},
		{R: 110, G: 150, B: 150},
		{R: 145, G: 105, B: 135},
	}
)

var (
	_ scene.Scene       = (*Scene)(nil)
	_ scene.Stepper     = (*Scene)(nil)
	_ scene.TrueColored = (*Scene)(nil)
)

// part is a right-facing body sprite: rows bodyTop..frameRow-1, '#' a dot.
type part struct {
	body   []string
	col    scene.RGB // body colour, unless paint is set
	wheels []int     // dot columns of each wheel's left edge
	paint  bool      // body takes a container paint instead
}

// loco is the locomotive: cab at the rear, boiler and chimney ahead.
var loco = part{
	body: []string{
		".................####.....",
		"#########.........##......",
		"#.......#.........##......",
		"#.......##############....",
		"#.......#.............##..",
		"#.......#..............##.",
		"#.......#...............#.",
		"##########################",
	},
	col:    locoCol,
	wheels: []int{2, 9, 16, 21},
}

// chimney is the dot column of the loco's chimney.
const chimney = 18

// cars are the rolling stock. The last is the big container, rarer than
// the rest (see bigEvery).
var cars = []part{
	{ // boxcar with sliding doors
		body: []string{
			"################",
			"#..............#",
			"#...#......#...#",
			"#...#......#...#",
			"#...#......#...#",
			"#...#......#...#",
			"#..............#",
			"################",
		},
		col:    scene.RGB{R: 155, G: 85, B: 70},
		wheels: []int{2, 10},
	},
	{ // tanker with a dome
		body: []string{
			".......####.......",
			"..##############..",
			".#..............#.",
			"#................#",
			"#................#",
			".#..............#.",
			"..##############..",
			"..................",
		},
		col:    scene.RGB{R: 175, G: 175, B: 170},
		wheels: []int{2, 12},
	},
	{ // hopper
		body: []string{
			"################",
			"#..............#",
			"#..............#",
			".#............#.",
			"..#..........#..",
			"...#........#...",
			"....########....",
			"................",
		},
		col:    scene.RGB{R: 125, G: 110, B: 95},
		wheels: []int{1, 11},
	},
	{ // flatcar with a small container
		body: []string{
			"................",
			"................",
			"................",
			"................",
			".#############..",
			".#.#.#.#.#.#.#..",
			".#.#.#.#.#.#.#..",
			".#############..",
		},
		paint:  true,
		wheels: []int{2, 10},
	},
	{ // flatcar with the big container
		body: []string{
			"#############################",
			"#.#.#.#.#.#.#.#.#.#.#.#.#.#.#",
			"#.#.#.#.#.#.#.#.#.#.#.#.#.#.#",
			"#.#.#.#.#.#.#.#.#.#.#.#.#.#.#",
			"#.#.#.#.#.#.#.#.#.#.#.#.#.#.#",
			"#.#.#.#.#.#.#.#.#.#.#.#.#.#.#",
			"#.#.#.#.#.#.#.#.#.#.#.#.#.#.#",
			"#############################",
		},
		col:    bigCol,
		wheels: []int{3, 22},
	},
}

// wheel is a small ring under the frame.
var wheel = []string{".##.", "#..#", ".##."}

// pix is one dot of the train, offset from its left edge and the strip top.
type pix struct {
	dx, dy int
	col    scene.RGB
}

// puff is a smoke puff in strip-local cells, drifting vx columns per second.
type puff struct{ x, y, vx, age float64 }

// Scene is the railway.
type Scene struct {
	rng       *rand.Rand
	w, h      int
	wait      float64 // seconds until the next train while the track is empty
	train     []pix   // facing its direction of travel
	tw        int     // train width in dots
	x         float64 // screen column of the train's left edge
	dir       int     // +1 rolls right, -1 left, 0 when no train
	chimX     int     // dot column of the chimney within the train
	puffs     []*puff
	puffClock float64
	puffEvery float64
	canvas    dots.Canvas // the strip, Height rows
}

// New returns a w×h scene that draws randomness from rng. The first train
// arrives within a few seconds.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng, puffEvery: puffEvery}
	s.wait = rng.Float64() * firstWait.Seconds()
	s.Resize(w, h)
	return s
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return step }

// Tick advances one interval.
func (s *Scene) Tick() { s.Step(step) }

// Resize changes the screen size. A train in progress keeps rolling.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.cull()
	s.canvas.Reset(s.w, Height)
	s.render()
}

// Step advances the train and its smoke by dt, or counts down to the next.
func (s *Scene) Step(dt time.Duration) {
	sec := max(dt, 0).Seconds()
	s.smoke(sec)
	switch {
	case s.dir != 0:
		s.x += float64(s.dir) * speed * sec
		s.cull()
	default:
		if s.wait -= sec; s.wait <= 0 {
			dir := 1
			if s.rng.IntN(2) == 0 {
				dir = -1
			}
			s.depart(dir)
			s.wait = minWait.Seconds() + s.rng.Float64()*waitSpread.Seconds()
		}
	}
	s.render()
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

// Passing reports whether a train is on the screen or entering it.
func (s *Scene) Passing() bool { return s.dir != 0 }

// depart assembles a random train just off the edge it enters from.
func (s *Scene) depart(dir int) {
	s.train = s.train[:0]
	x := 0
	add := func(p part, body scene.RGB) {
		for r, row := range p.body {
			for c, ch := range row {
				if ch == '#' {
					s.train = append(s.train, pix{x + c, bodyTop + r, body})
				}
			}
		}
		w := len(p.body[0])
		for c := 1; c < w-1; c++ {
			s.train = append(s.train, pix{x + c, frameRow, ironCol})
		}
		for _, wx := range p.wheels {
			for r, row := range wheel {
				for c, ch := range row {
					if ch == '#' {
						s.train = append(s.train, pix{x + wx + c, frameRow + 1 + r, ironCol})
					}
				}
			}
		}
		x += w
	}
	paint := s.painter()
	for range minCars + s.rng.IntN(carSpread) {
		i := s.rng.IntN(len(cars) - 1)
		if s.rng.IntN(bigEvery) == 0 {
			i = len(cars) - 1
		}
		c := cars[i].col
		if cars[i].paint {
			c = paint()
		}
		add(cars[i], c)
		for d := range coupler + 2 {
			s.train = append(s.train, pix{x - 1 + d, frameRow + 1, ironCol})
		}
		x += coupler
	}
	s.chimX = x + chimney
	add(loco, loco.col)
	s.tw = x

	s.dir = dir
	if dir < 0 {
		for i := range s.train {
			s.train[i].dx = s.tw - 1 - s.train[i].dx
		}
		s.chimX = s.tw - 1 - s.chimX
		s.x = float64(s.w)
	} else {
		s.x = -float64(s.tw) / 2
	}
	s.puffClock = 0
}

// cull clears the train once it has fully left the screen.
func (s *Scene) cull() {
	if s.dir == 0 {
		return
	}
	w := float64(s.tw) / 2
	if (s.dir > 0 && s.x >= float64(s.w)) || (s.dir < 0 && s.x+w <= 0) {
		s.dir, s.train = 0, s.train[:0]
	}
}

// painter returns a source of container paints in which neighbours differ.
func (s *Scene) painter() func() scene.RGB {
	prev := -1
	return func() scene.RGB {
		i := s.rng.IntN(len(boxCols))
		if i == prev {
			i = (i + 1) % len(boxCols)
		}
		prev = i
		return boxCols[i]
	}
}

// smoke ages, lifts and drops puffs, and puffs new ones from the chimney
// while a train runs.
func (s *Scene) smoke(sec float64) {
	kept := s.puffs[:0]
	for _, p := range s.puffs {
		p.age += sec
		p.y -= riseSpeed * sec
		p.x += p.vx * sec
		if p.age < puffLife && p.y >= 0 {
			kept = append(kept, p)
		}
	}
	s.puffs = kept
	if s.dir == 0 {
		return
	}
	for s.puffClock += sec; s.puffClock >= s.puffEvery; s.puffClock -= s.puffEvery {
		s.puffs = append(s.puffs, &puff{
			x:  s.x + float64(s.chimX)/2 + (s.rng.Float64()-0.5)*0.5,
			y:  float64(bodyTop-1) / 4,
			vx: -float64(s.dir) * driftSpeed,
		})
	}
}

// smokeColour fades a puff from near white to grey over its life (f 0..1).
func smokeColour(f float64) scene.RGB {
	f = min(max(f, 0), 1)
	l := func(a, b uint8) uint8 { return uint8(math.Round(float64(a) + (float64(b)-float64(a))*f)) }
	return scene.RGB{R: l(smokeNew.R, smokeOld.R), G: l(smokeNew.G, smokeOld.G), B: l(smokeNew.B, smokeOld.B)}
}

// render redraws the strip: rail, train, then smoke.
func (s *Scene) render() {
	s.canvas.Clear()
	if s.w == 0 {
		return
	}
	for dx := range s.w * 2 {
		x := float64(dx) / 2
		s.canvas.Dot(x, float64(railRow)/4, railCol)
		if dx%tieEvery >= tieEvery-2 {
			s.canvas.Dot(x, float64(railRow+1)/4, railCol)
			s.canvas.Dot(x, float64(railRow+2)/4, railCol)
		}
	}
	if s.dir != 0 {
		for _, p := range s.train {
			s.canvas.Dot(s.x+float64(p.dx)/2, float64(p.dy)/4, p.col)
		}
	}
	for _, p := range s.puffs {
		f := p.age / puffLife
		c := smokeColour(f)
		s.canvas.Dot(p.x, p.y, c)
		if f < 0.45 { // a young puff is a little cluster
			s.canvas.Dot(p.x+0.5, p.y, c)
			s.canvas.Dot(p.x, p.y+0.25, c)
		}
	}
}

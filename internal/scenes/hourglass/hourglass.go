// Package hourglass draws sand piling up along the bottom of the screen over
// the current hour, used as an ambient background in live mode. The pile's
// volume is the elapsed fraction of the hour, so it is right from the first
// frame; a thin stream of grains falls onto it, and it empties on the hour.
package hourglass

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

var (
	_ scene.Scene   = (*Scene)(nil)
	_ scene.Clocked = (*Scene)(nil)
	_ scene.Colored = (*Scene)(nil)
)

const (
	interval    = 100 * time.Millisecond
	maxPileRows = 2    // tallest the mound gets, at the end of the hour
	pileRowsDiv = 6    // the mound is at most 1/pileRowsDiv of the screen
	grainCols   = 30   // screen columns per falling grain
	maxGrains   = 4    // grains in flight on even the widest screen
	grainSpeed  = 0.6  // rows per tick
	moundFloor  = 0.25 // pile height at the edges relative to the peak
	streamWidth = 0.15 // grains fall within this fraction of the width of centre
)

// eighths are the pile surface glyphs by height in eighths of a cell;
// index 0 is empty.
var eighths = []rune{0, '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

type grain struct {
	x int
	y float64
}

// Scene is the sand over a w×h screen.
type Scene struct {
	w, h    int
	frac    float64 // elapsed fraction of the hour, [0, 1)
	hasTime bool
	heights []int // pile height per column, in eighths of a cell
	grains  []grain
	buf     []rune
	rng     *rand.Rand
}

// New returns a w×h scene that draws randomness from rng. Nothing is drawn
// until the first SetTime.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.Resize(w, h)
	return s
}

// Resize changes the screen size and restarts the falling grains.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.buf = make([]rune, s.w*s.h)
	s.heights = make([]int, s.w)
	s.grains = s.grains[:0]
	if s.h > 0 {
		for range min(max(s.w/grainCols, 1), maxGrains) {
			s.grains = append(s.grains, s.spawn(s.rng.Float64()*float64(s.h)))
		}
	}
	s.pile()
	s.render()
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return interval }

// SetTime sizes the pile for the minute and second of now. shown is ignored.
func (s *Scene) SetTime(now, _ time.Time) {
	s.frac = HourFraction(now)
	s.hasTime = true
	s.pile()
	s.render()
}

// Tick drops the grains one step.
func (s *Scene) Tick() {
	for i := range s.grains {
		g := &s.grains[i]
		g.y += grainSpeed
		if int(g.y) >= s.surface(g.x) {
			*g = s.spawn(0)
		}
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

// Color returns the tint of the glyph at (x, y). Everything drawn is sand.
func (s *Scene) Color(int, int, scene.Box) scene.Color { return scene.Yellow }

// HourFraction is the elapsed fraction of t's hour, in [0, 1).
func HourFraction(t time.Time) float64 {
	d := time.Duration(t.Minute())*time.Minute +
		time.Duration(t.Second())*time.Second +
		time.Duration(t.Nanosecond())
	return float64(d) / float64(time.Hour)
}

// maxRows is the mound's peak height, in rows, when the hour is up.
func (s *Scene) maxRows() int {
	return min(max(s.h/pileRowsDiv, 1), maxPileRows)
}

// profile is the mound's shape at column x, from moundFloor at the edges
// to 1 in the middle.
func (s *Scene) profile(x int) float64 {
	if s.w <= 1 {
		return 1
	}
	u := 2*(float64(x)+0.5)/float64(s.w) - 1
	return moundFloor + (1-moundFloor)*(1-u*u)
}

// Capacity is the pile's volume, in eighths of a cell, when the hour is up.
func (s *Scene) Capacity() float64 {
	sum := 0.0
	for x := range s.w {
		sum += s.profile(x)
	}
	return sum * float64(s.maxRows()*8)
}

// Volume is the pile's current volume, in eighths of a cell.
func (s *Scene) Volume() int {
	n := 0
	for _, h := range s.heights {
		n += h
	}
	return n
}

// pile recomputes the column heights so the mound holds frac of Capacity.
func (s *Scene) pile() {
	scale := 0.0
	if s.hasTime {
		scale = s.frac * float64(s.maxRows()*8)
	}
	for x := range s.heights {
		s.heights[x] = int(math.Round(scale * s.profile(x)))
	}
}

// surface is the screen row a grain landing in column x stops at.
func (s *Scene) surface(x int) int {
	if x < 0 || x >= s.w {
		return s.h
	}
	return s.h - (s.heights[x]+7)/8
}

// spawn returns a grain at row y, near the middle of the screen.
func (s *Scene) spawn(y float64) grain {
	spread := max(int(float64(s.w)*streamWidth), 1)
	x := s.w/2 - spread/2 + s.rng.IntN(spread)
	return grain{x: min(max(x, 0), max(s.w-1, 0)), y: y}
}

func (s *Scene) render() {
	clear(s.buf)
	if !s.hasTime || s.w == 0 || s.h == 0 {
		return
	}
	for x, ht := range s.heights {
		for r := 0; ht > 0 && r < s.h; r++ {
			s.buf[(s.h-1-r)*s.w+x] = eighths[min(ht, 8)]
			ht -= 8
		}
	}
	for _, g := range s.grains {
		if y := int(g.y); y >= 0 && y < s.surface(g.x) && g.x < s.w {
			s.buf[y*s.w+g.x] = '·'
		}
	}
}

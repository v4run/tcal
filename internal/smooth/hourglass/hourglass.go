// Package hourglass is the smooth hourglass scene: sand piles up in braille
// dot rows along the bottom of the screen over the current hour, a thin
// stream of grains falls onto it, and it empties on the hour. The pile's
// volume is the elapsed fraction of the hour, so it is right from the first
// frame.
package hourglass

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
	_ scene.Clocked     = (*Scene)(nil)
)

const (
	step        = 33 * time.Millisecond
	maxPileRows = 2    // tallest the mound gets, at the end of the hour
	pileRowsDiv = 6    // the mound is at most 1/pileRowsDiv of the screen
	grainCols   = 30   // screen columns per falling grain
	maxGrains   = 4    // grains in flight on even the widest screen
	grainSpeed  = 6.0  // rows per second (0.6 rows per 100 ms)
	moundFloor  = 0.25 // pile height at the edges relative to the peak
	streamWidth = 0.15 // grains fall within this fraction of the width of centre
)

var (
	sandCol  = scene.RGB{R: 205, G: 170, B: 80}
	grainCol = scene.RGB{R: 230, G: 200, B: 110}
)

// grain is one falling sand grain: x is a dot column, y a position in rows.
type grain struct {
	x int
	y float64
}

// Scene is the sand over a w×h screen.
type Scene struct {
	rng     *rand.Rand
	w, h    int
	frac    float64 // elapsed fraction of the hour, [0, 1)
	hasTime bool
	heights []int // pile height per dot column, in dot rows
	grains  []grain
	canvas  dots.Canvas
}

// New returns a w×h scene that draws randomness from rng. Nothing is drawn
// until the first SetTime.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.Resize(w, h)
	return s
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return step }

// Tick advances one interval.
func (s *Scene) Tick() { s.Step(step) }

// Resize changes the screen size and restarts the falling grains.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.canvas.Reset(s.w, s.h)
	s.heights = make([]int, s.w*2)
	s.grains = s.grains[:0]
	if s.h > 0 {
		for range min(max(s.w/grainCols, 1), maxGrains) {
			s.grains = append(s.grains, s.spawn(s.rng.Float64()*float64(s.h)))
		}
	}
	s.pile()
	s.render()
}

// SetTime sizes the pile for the minute and second of now. shown is ignored.
func (s *Scene) SetTime(now, _ time.Time) {
	s.frac = HourFraction(now)
	s.hasTime = true
	s.pile()
	s.render()
}

// Step drops the grains by dt.
func (s *Scene) Step(dt time.Duration) {
	fall := grainSpeed * max(dt, 0).Seconds()
	for i := range s.grains {
		g := &s.grains[i]
		g.y += fall
		if int(g.y*4) >= s.surface(g.x) {
			*g = s.spawn(0)
		}
	}
	s.render()
}

// Glyph returns the rune at screen (x, y), or 0 for empty.
func (s *Scene) Glyph(x, y int, _ scene.Box) rune { return s.canvas.Glyph(x, y) }

// RGB returns the colour of the glyph at (x, y).
func (s *Scene) RGB(x, y int, _ scene.Box) scene.RGB { return s.canvas.RGB(x, y) }

// HourFraction is the elapsed fraction of t's hour, in [0, 1).
func HourFraction(t time.Time) float64 {
	d := time.Duration(t.Minute())*time.Minute +
		time.Duration(t.Second())*time.Second +
		time.Duration(t.Nanosecond())
	return float64(d) / float64(time.Hour)
}

// maxRows is the mound's peak height, in cell rows, when the hour is up.
func (s *Scene) maxRows() int {
	return min(max(s.h/pileRowsDiv, 1), maxPileRows)
}

// profile is the mound's shape at dot column x, from moundFloor at the edges
// to 1 in the middle.
func (s *Scene) profile(x int) float64 {
	n := s.w * 2
	if n <= 1 {
		return 1
	}
	u := 2*(float64(x)+0.5)/float64(n) - 1
	return moundFloor + (1-moundFloor)*(1-u*u)
}

// Capacity is the pile's volume, in dots, when the hour is up.
func (s *Scene) Capacity() float64 {
	sum := 0.0
	for x := range s.heights {
		sum += s.profile(x)
	}
	return sum * float64(s.maxRows()*4)
}

// Volume is the pile's current volume, in dots.
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
		scale = s.frac * float64(s.maxRows()*4)
	}
	for x := range s.heights {
		s.heights[x] = int(math.Round(scale * s.profile(x)))
	}
}

// surface is the dot row of the top of the pile in dot column x; a grain
// whose dot row reaches it has landed.
func (s *Scene) surface(x int) int {
	if x < 0 || x >= len(s.heights) {
		return s.h * 4
	}
	return s.h*4 - s.heights[x]
}

// spawn returns a grain at row y, near the middle of the screen.
func (s *Scene) spawn(y float64) grain {
	n := s.w * 2
	spread := max(int(float64(n)*streamWidth), 1)
	x := n/2 - spread/2 + s.rng.IntN(spread)
	return grain{x: min(max(x, 0), max(n-1, 0)), y: y}
}

func (s *Scene) render() {
	s.canvas.Clear()
	if !s.hasTime || s.w == 0 || s.h == 0 {
		return
	}
	for dx, ht := range s.heights {
		for r := range ht {
			s.canvas.Dot(float64(dx)/2, float64(s.h*4-1-r)/4, sandCol)
		}
	}
	for _, g := range s.grains {
		if g.y >= 0 && int(g.y*4) < s.surface(g.x) && g.x < s.w*2 {
			s.canvas.Dot(float64(g.x)/2, g.y, grainCol)
		}
	}
}

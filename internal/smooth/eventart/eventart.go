// Package eventart draws small dim holiday scenes for the smooth seasons
// background, in braille dots and true colour. Each art sits in one region
// of the screen margin (a corner or a band along an edge) and hides
// entirely when that region would touch the calendar box. Motion is in
// units per second scaled by the step, and each art renders once per
// Step, Resize or SetTime into a region-sized dots.Canvas, so Cell is a
// lookup.
package eventart

import (
	"math"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/v4run/tcal/internal/dots"
	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

// Region is a rectangle of screen cells.
type Region struct{ X0, Y0, W, H int }

// Touches reports whether the region overlaps the box by at least one cell.
// An empty box (Y0 == Y1 == -1) never touches; a real box starting at row 0
// also has Y0 == -1 (one row above it) but a non-negative Y1.
func (r Region) Touches(b scene.Box) bool {
	if (b.Y0 == -1 && b.Y1 == -1) || r.W <= 0 || r.H <= 0 {
		return false
	}
	return b.X0 < r.X0+r.W && b.X1 > r.X0 && b.Y0 < r.Y0+r.H && b.Y1 >= r.Y0
}

// Art draws one event's scene within Bounds, hiding entirely when Bounds
// touches the calendar box.
type Art interface {
	Resize(w, h int)
	Step(dt time.Duration)
	SetTime(now time.Time, occ events.Occurrence)
	Bounds() Region                               // the full area the art may ever draw in, for the current size
	Cell(x, y int, b scene.Box) (rune, scene.RGB) // 0 rune = empty
}

var arts = map[events.Kind]func(*rand.Rand) Art{}

// New returns the art for k, or nil if k has none.
func New(k events.Kind, rng *rand.Rand) Art {
	f, ok := arts[k]
	if !ok {
		return nil
	}
	return f(rng)
}

// Kinds returns the kinds that have art, sorted.
func Kinds() []events.Kind {
	ks := make([]events.Kind, 0, len(arts))
	for k := range arts {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })
	return ks
}

// frame is an art's region and a dot canvas the size of it. Arts draw in
// region-local cell units; anything outside the region is dropped.
type frame struct {
	r Region
	c dots.Canvas
}

// reset sizes the frame to r and clears it; a degenerate r empties it.
func (f *frame) reset(r Region) {
	if r.W <= 0 || r.H <= 0 {
		r = Region{}
	}
	f.r = r
	f.c.Reset(r.W, r.H)
}

// cell returns screen cell (x, y), or nothing at all when the region
// touches the calendar box b.
func (f *frame) cell(x, y int, b scene.Box) (rune, scene.RGB) {
	if f.r.W == 0 || f.r.Touches(b) || x < f.r.X0 || y < f.r.Y0 {
		return 0, scene.RGB{}
	}
	lx, ly := x-f.r.X0, y-f.r.Y0
	g := f.c.Glyph(lx, ly)
	if g == 0 {
		return 0, scene.RGB{}
	}
	return g, f.c.RGB(lx, ly)
}

// Sway: a hanging thing swings swayAmp columns either side of its anchor,
// once every swayPeriod seconds (classic swung one column over 4 s).
const (
	swayAmp    = 0.8
	swayPeriod = 4.0
)

// sway is the sideways offset in cells, at t seconds, of a hanging thing
// with the given phase in radians.
func sway(t, phase float64) float64 {
	return swayAmp * math.Sin(2*math.Pi*t/swayPeriod+phase)
}

// line draws a dotted line of dots from (x0, y0) to (x1, y1) in cell units.
func line(c *dots.Canvas, x0, y0, x1, y1 float64, col scene.RGB) {
	n := int(math.Ceil(max(math.Abs(x1-x0)*2, math.Abs(y1-y0)*4)))
	for i := 0; i <= n; i++ {
		f := 0.0
		if n > 0 {
			f = float64(i) / float64(n)
		}
		c.Dot(x0+(x1-x0)*f, y0+(y1-y0)*f, col)
	}
}

// scale multiplies a colour's channels by k (0..1).
func scale(c scene.RGB, k float64) scene.RGB {
	k = min(max(k, 0), 1)
	f := func(v uint8) uint8 { return uint8(math.Round(float64(v) * k)) }
	return scene.RGB{R: f(c.R), G: f(c.G), B: f(c.B)}
}

// lerp blends a toward b by t (0..1).
func lerp(a, b scene.RGB, t float64) scene.RGB {
	t = min(max(t, 0), 1)
	f := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*t)) }
	return scene.RGB{R: f(a.R, b.R), G: f(a.G, b.G), B: f(a.B, b.B)}
}

// hash mixes a seed and two integers into a pseudo-random value, so random
// scatters stay put from frame to frame.
func hash(seed uint32, a, b int) uint32 {
	k := seed ^ uint32(a+64)*0x9E3779B1 ^ uint32(b+64)*0x85EBCA77
	k ^= k >> 15
	k *= 0x2C1B3C6D
	k ^= k >> 12
	k *= 0x297A2D39
	k ^= k >> 15
	return k
}

// unit maps a hash to [0, 1).
func unit(h uint32) float64 { return float64(h) / (1 << 32) }

// chance is the probability of at least one event in dt at rate per second.
func chance(rate float64, dt time.Duration) float64 {
	return 1 - math.Exp(-rate*dt.Seconds())
}

// edgeMargin is the columns kept clear between a side region and the
// screen's left or right edge.
const edgeMargin = 2

// edgeRegion is an rw×rh region against the left or right side of the
// screen, edgeMargin columns in and one row clear of the top or bottom
// edge, or empty when the screen cannot hold it.
func edgeRegion(w, h, rw, rh int, right, bottom bool) Region {
	if rw <= 0 || rh <= 0 || w < rw+2*edgeMargin || h < rh+2 {
		return Region{}
	}
	r := Region{X0: edgeMargin, Y0: 1, W: rw, H: rh}
	if right {
		r.X0 = w - edgeMargin - rw
	}
	if bottom {
		r.Y0 = h - 1 - rh
	}
	return r
}

// columnHeight is the height of a side column on an h-row screen: as tall
// as the screen allows up to maxH, or 0 when that is under minH.
func columnHeight(h, minH, maxH int) int {
	ch := min(h-2, maxH)
	if ch < minH {
		return 0
	}
	return ch
}

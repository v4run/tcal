// Package sky is the smooth day/night sky: by day a dot-disc sun with
// shimmering rays glides along an arc from sunrise on the left to sunset on
// the right while clouds drift past; by night the moon, in its real lunar
// phase, follows the same arc among twinkling stars. It keeps to the upper
// third of the screen, like the classic sky.
package sky

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
	step       = 33 * time.Millisecond
	dawnHour   = 6  // start of day; the sun rises at the left edge
	duskHour   = 18 // start of night; the moon rises at the left edge
	arcTop     = 1  // screen row of the arc's peak (noon / midnight)
	maxArcRows = 7  // lowest row the arc reaches, at most
	starCells  = 30 // sky cells per star
	cloudCols  = 45 // screen columns per cloud
	maxClouds  = 3  // clouds on even the widest screen

	// The sun disc's centre sits sunDrop dots below the arc point so the
	// top ray is still on screen at noon, when the arc peaks at arcTop.
	sunDrop   = 4
	sunRadius = 4.0 // disc radius in dots: 4 cells wide
	rayIn     = 5.5 // rays start this far from the centre, in dots
	rayOut    = 8.5 // and end this far out
	rayShort  = 6.6 // side rays shrink to this while shimmering
	warmHours = 1.0 // the sun warms within this many hours of the horizon

	moonDots = 8 // moon diameter in dots: 4 cells wide, 2 rows tall

	minStarPeriod = 8.0  // seconds between a star's twinkles, at least
	maxStarPeriod = 24.0 // and at most
	twinkleSecs   = 0.4  // how long a twinkle lasts

	minCloudSpeed = 0.2 // cells per second
	maxCloudSpeed = 0.4
)

// shimmerPeriod is how long the sun's rays hold each of their two frames.
const shimmerPeriod = 1600 * time.Millisecond

var (
	sunNoon    = scene.RGB{R: 240, G: 200, B: 90}
	sunLow     = scene.RGB{R: 240, G: 140, B: 70}
	moonCol    = scene.RGB{R: 210, G: 210, B: 225}
	cloudCol   = scene.RGB{R: 170, G: 170, B: 180}
	starCol    = scene.RGB{R: 190, G: 190, B: 210}
	starBright = scene.RGB{R: 240, G: 225, B: 150}
)

// synodicMonth is the mean length of a lunar cycle, in days.
const synodicMonth = 29.530588853

// refNewMoon is a known new moon used as the phase origin.
var refNewMoon = time.Date(2000, time.January, 6, 18, 14, 0, 0, time.UTC)

// sunFrames are the sun's dots as offsets from its centre, which lies on a
// dot corner: frame 0 has full rays, frame 1 short side rays and diagonals
// faded to a single dot, as classic's rays shimmer. The vertical rays stay.
var sunFrames = [2][][2]int{sunDots(false), sunDots(true)}

func sunDots(faded bool) [][2]int {
	var out [][2]int
	const n = 10 // half the grid, in dots
	for j := -n; j < n; j++ {
		for i := -n; i < n; i++ {
			u, v := float64(i)+0.5, float64(j)+0.5
			r := math.Hypot(u, v)
			if r <= sunRadius {
				out = append(out, [2]int{i, j})
				continue
			}
			if r < rayIn || r > rayOut {
				continue
			}
			vert := math.Abs(u) < 1
			side := math.Abs(v) < 1
			diag := math.Abs(u) == math.Abs(v)
			switch {
			case vert, side && (!faded || r <= rayShort):
				out = append(out, [2]int{i, j})
			case diag && (!faded || r > rayOut-1.5):
				out = append(out, [2]int{i, j})
			}
		}
	}
	return out
}

// cloudSprites echo classic's three cloud shapes as dot outlines.
var cloudSprites = []sprite{
	parseSprite(
		"....######....",
		"..##......##..",
		".#..........#.",
		"#............#",
		"#............#",
		".############.",
	),
	parseSprite(
		"......####.......",
		"....##....##.....",
		".###........#....",
		"#............###.",
		"#...............#",
		"#...............#",
		".###############.",
	),
	parseSprite(
		"..####..",
		".#....#.",
		"#......#",
		".######.",
	),
}

// rayReach is how far, in dots, the sun's rays extend below its centre.
const rayReach = 8

// cloudMaxH is the height in dots of the tallest cloud sprite.
const cloudMaxH = 7

// sprite is dot art as offsets from its top left, with its width in dots.
type sprite struct {
	w    int
	dots [][2]int
}

// parseSprite reads rows where '#' is a dot.
func parseSprite(rows ...string) sprite {
	var sp sprite
	for y, row := range rows {
		sp.w = max(sp.w, len(row))
		for x, r := range row {
			if r == '#' {
				sp.dots = append(sp.dots, [2]int{x, y})
			}
		}
	}
	return sp
}

type star struct {
	x, y   int     // dot position
	period float64 // seconds between twinkles
	offset float64 // phase within period, in seconds
}

type cloud struct {
	x     float64 // left edge, in cells
	y     int     // top edge, in dot rows
	speed float64 // cells per second
	shape int     // index into cloudSprites
}

// Scene is a sky over a w×h screen.
type Scene struct {
	rng     *rand.Rand
	w, h    int
	now     time.Time
	hasTime bool
	t       float64 // seconds stepped, for shimmer and twinkle
	stars   []star
	clouds  []cloud
	canvas  dots.Canvas
}

// New returns a w×h sky that draws randomness from rng. Nothing is drawn
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

// Resize changes the screen size and rescatters stars and clouds.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.canvas.Reset(s.w, s.h)
	s.scatter()
	s.render()
}

// SetTime positions the sun or moon for now. The shown month is ignored.
func (s *Scene) SetTime(now, _ time.Time) {
	s.now, s.hasTime = now, true
	s.render()
}

// Step drifts the clouds and advances the shimmer and twinkle clock by dt.
func (s *Scene) Step(dt time.Duration) {
	sec := max(dt, 0).Seconds()
	s.t += sec
	for i := range s.clouds {
		c := &s.clouds[i]
		c.x += c.speed * sec
		if c.x > float64(s.w) {
			c.x = -float64(cloudSprites[c.shape].w) / 2
			c.y = s.cloudRow()
		}
	}
	s.render()
}

// Glyph returns the rune at screen (x, y), or 0 for empty.
func (s *Scene) Glyph(x, y int, _ scene.Box) rune { return s.canvas.Glyph(x, y) }

// RGB returns the colour of the glyph at (x, y).
func (s *Scene) RGB(x, y int, _ scene.Box) scene.RGB { return s.canvas.RGB(x, y) }

// IsDay reports whether t falls in the sun's half of the day.
func IsDay(t time.Time) bool {
	h := t.Hour()
	return h >= dawnHour && h < duskHour
}

// hours is t's time of day in hours.
func hours(t time.Time) float64 {
	return float64(t.Hour()) + float64(t.Minute())/60 +
		(float64(t.Second())+float64(t.Nanosecond())/1e9)/3600
}

// arcFraction is how far across the sky the sun (by day) or moon (by
// night) has travelled at t, in [0, 1).
func arcFraction(t time.Time) float64 {
	start := float64(dawnHour)
	if !IsDay(t) {
		start = duskHour
	}
	return math.Mod(hours(t)-start+24, 24) / (duskHour - dawnHour)
}

// arcBottom is the row where the sun or moon rises and sets.
func (s *Scene) arcBottom() int {
	return max(arcTop, min(s.h/3, maxArcRows))
}

// arcPos is the point, in cells, on the arc at fraction f. Unlike classic it
// is not rounded to a cell, so the sun and moon glide.
func (s *Scene) arcPos(f float64) (x, y float64) {
	bottom := float64(s.arcBottom())
	x = f * float64(max(s.w-1, 0))
	y = bottom - (bottom-arcTop)*math.Sin(math.Pi*f)
	return x, y
}

// sunCentre is the dot corner at the middle of the sun's disc.
func (s *Scene) sunCentre() (dx, dy int) {
	x, y := s.arcPos(arcFraction(s.now))
	// Keep the disc and rays whole above the limit on short screens.
	return int(math.Round(x * 2)), min(int(math.Round(y*4))+sunDrop, s.limit()-rayReach-1)
}

// moonOrigin is the dot at the top left of the moon's bounding square.
func (s *Scene) moonOrigin() (dx, dy int) {
	x, y := s.arcPos(arcFraction(s.now))
	return int(math.Round(x*2)) - moonDots/2, min(int(math.Round(y*4))-moonDots/2, s.limit()-moonDots)
}

// cloudDot is cloud i's left edge in dot columns, as plotted.
func (s *Scene) cloudDot(i int) int { return int(math.Floor(s.clouds[i].x * 2)) }

// sunColour is the sun's tint at t: warm near the horizon, yellow by noon.
func sunColour(t time.Time) scene.RGB {
	h := hours(t)
	d := min(h-dawnHour, duskHour-h) / warmHours
	if d >= 1 {
		return sunNoon
	}
	d = max(d, 0)
	mix := func(a, b uint8) uint8 { return uint8(math.Round(float64(a) + (float64(b)-float64(a))*d)) }
	return scene.RGB{R: mix(sunLow.R, sunNoon.R), G: mix(sunLow.G, sunNoon.G), B: mix(sunLow.B, sunNoon.B)}
}

// MoonPhase returns the lunar phase at t in [0, 1): 0 is new, 0.25 first
// quarter, 0.5 full and 0.75 last quarter.
func MoonPhase(t time.Time) float64 {
	days := t.Sub(refNewMoon).Hours() / 24
	p := math.Mod(days/synodicMonth, 1)
	if p < 0 {
		p++
	}
	return p
}

// moonLit reports whether the point (u, v) of the unit disc is sunlit at
// phase p, as seen from the northern hemisphere (waxing lit on the right).
func moonLit(p, u, v float64) bool {
	if u*u+v*v > 1 {
		return false
	}
	t := math.Cos(2*math.Pi*p) * math.Sqrt(1-v*v)
	if p < 0.5 {
		return u > t
	}
	return u < -t
}

func (s *Scene) cloudRow() int {
	row := arcTop*4 + s.rng.IntN(max((s.arcBottom()-arcTop)*4, 1))
	// Cap the row so even the tallest cloud fits above the limit.
	return max(min(row, s.limit()-cloudMaxH), 0)
}

func (s *Scene) scatter() {
	rows := s.arcBottom() + 2
	s.stars = s.stars[:0]
	for range s.w * rows / starCells {
		period := minStarPeriod + (maxStarPeriod-minStarPeriod)*s.rng.Float64()
		s.stars = append(s.stars, star{
			x: s.rng.IntN(s.w * 2), y: s.rng.IntN(rows * 4),
			period: period, offset: period * s.rng.Float64(),
		})
	}
	s.clouds = s.clouds[:0]
	for range min(s.w/cloudCols, maxClouds) {
		s.clouds = append(s.clouds, cloud{
			x:     float64(s.rng.IntN(max(s.w, 1))),
			y:     s.cloudRow(),
			speed: minCloudSpeed + (maxCloudSpeed-minCloudSpeed)*s.rng.Float64(),
			shape: s.rng.IntN(len(cloudSprites)),
		})
	}
}

// limit is the first dot row below the upper third, which nothing reaches.
func (s *Scene) limit() int { return s.h / 3 * 4 }

// dot plots the dot at dot column dx, dot row dy, clipped to the upper third.
func (s *Scene) dot(dx, dy int, col scene.RGB) {
	if dy < s.limit() {
		s.canvas.Dot(float64(dx)/2, float64(dy)/4, col)
	}
}

func (s *Scene) render() {
	s.canvas.Clear()
	if !s.hasTime || s.w == 0 || s.h == 0 {
		return
	}
	if IsDay(s.now) {
		cx, cy := s.sunCentre()
		col := sunColour(s.now)
		for _, d := range sunFrames[int(s.t/shimmerPeriod.Seconds())%2] {
			s.dot(cx+d[0], cy+d[1], col)
		}
		for _, c := range s.clouds {
			x0 := int(math.Floor(c.x * 2))
			for _, d := range cloudSprites[c.shape].dots {
				s.dot(x0+d[0], c.y+d[1], cloudCol)
			}
		}
		return
	}
	// The moon blots out the stars behind it, lit or not.
	mx, my := s.moonOrigin()
	for _, st := range s.stars {
		if st.x >= mx-2 && st.x < mx+moonDots+2 && st.y >= my-4 && st.y < my+moonDots+4 {
			continue
		}
		if math.Mod(s.t+st.offset, st.period) < twinkleSecs {
			for _, d := range [][2]int{{0, 0}, {-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				s.dot(st.x+d[0], st.y+d[1], starBright)
			}
			continue
		}
		s.dot(st.x, st.y, starCol)
	}
	p := MoonPhase(s.now)
	for dy := range moonDots {
		for dx := range moonDots {
			u := (float64(dx)+0.5)/(moonDots/2) - 1
			v := (float64(dy)+0.5)/(moonDots/2) - 1
			if moonLit(p, u, v) {
				s.dot(mx+dx, my+dy, moonCol)
			}
		}
	}
}

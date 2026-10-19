// Package sky draws a day/night sky across the top of the screen, used as an
// ambient background in live mode. By day a small sun follows an arc from
// sunrise on the left to sunset on the right while a few clouds drift past;
// by night the moon, in its real lunar phase, follows the same arc among
// sparse twinkling stars.
package sky

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
	interval    = 200 * time.Millisecond
	dawnHour    = 6  // start of day; the sun rises at the left edge
	duskHour    = 18 // start of night; the moon rises at the left edge
	arcTop      = 1  // screen row of the arc's peak (noon / midnight)
	maxArcRows  = 7  // lowest row the arc reaches, at most
	starCells   = 30 // sky cells per star
	cloudCols   = 45 // screen columns per cloud
	maxClouds   = 3  // clouds on even the widest screen
	twinkleTick = 2  // ticks a star spends twinkling per period
)

// synodicMonth is the mean length of a lunar cycle, in days.
const synodicMonth = 29.530588853

// refNewMoon is a known new moon used as the phase origin.
var refNewMoon = time.Date(2000, time.January, 6, 18, 14, 0, 0, time.UTC)

// sun is centred on its arc position; spaces are transparent.
var sun = []string{
	`\|/`,
	`-O-`,
	`/|\`,
}

// cloudShapes are drawn with their top-left at the cloud position.
var cloudShapes = [][]string{
	{" .--. ", "(____)"},
	{"  .-. ", ".(   ).", "(___.__)"},
	{".--.", "(__)"},
}

const (
	moonCols = 4 // moon width in cells: 8 braille dots
	moonRows = 2 // moon height in cells: 8 braille dots
)

type star struct {
	x, y   int
	glyph  rune
	period int // ticks between twinkles
	offset int // phase within period
}

type cloud struct {
	x     float64
	y     int
	speed float64 // columns per tick
	shape int     // index into cloudShapes
}

// Scene is a sky over a w×h screen.
type Scene struct {
	w, h    int
	buf     []rune
	colors  []scene.Color // parallel to buf
	now     time.Time
	hasTime bool
	tick    int
	stars   []star
	clouds  []cloud
	rng     *rand.Rand
}

// New returns a w×h sky that draws randomness from rng. Nothing is drawn
// until the first SetTime.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.Resize(w, h)
	return s
}

// Resize changes the screen size and rescatters stars and clouds.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.buf = make([]rune, s.w*s.h)
	s.colors = make([]scene.Color, s.w*s.h)
	s.scatter()
	s.render()
}

// Tick drifts the clouds and twinkles the stars.
func (s *Scene) Tick() {
	s.tick++
	for i := range s.clouds {
		c := &s.clouds[i]
		c.x += c.speed
		if int(c.x) > s.w {
			c.x = -float64(shapeWidth(cloudShapes[c.shape]))
			c.y = s.cloudRow()
		}
	}
	s.render()
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return interval }

// SetTime positions the sun or moon for now. The shown month is ignored.
func (s *Scene) SetTime(now, _ time.Time) {
	s.now, s.hasTime = now, true
	s.render()
}

// Glyph returns the rune drawn at screen (x, y), or 0 for empty.
func (s *Scene) Glyph(x, y int, _ scene.Box) rune {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return 0
	}
	return s.buf[y*s.w+x]
}

// Color returns the tint of the glyph at (x, y): a yellow sun, white clouds
// and moon, and plain stars that flash yellow as they twinkle.
func (s *Scene) Color(x, y int, _ scene.Box) scene.Color {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return scene.Default
	}
	return s.colors[y*s.w+x]
}

// arcBottom is the row where the sun or moon rises and sets.
func (s *Scene) arcBottom() int {
	return max(arcTop, min(s.h/3, maxArcRows))
}

func (s *Scene) cloudRow() int {
	return arcTop + s.rng.IntN(max(s.arcBottom()-arcTop, 1))
}

func (s *Scene) scatter() {
	rows := s.arcBottom() + 2
	n := s.w * rows / starCells
	s.stars = s.stars[:0]
	for range n {
		g := '·'
		if s.rng.IntN(3) == 0 {
			g = '.'
		}
		period := 40 + s.rng.IntN(80)
		s.stars = append(s.stars, star{
			x: s.rng.IntN(s.w), y: s.rng.IntN(rows),
			glyph: g, period: period, offset: s.rng.IntN(period),
		})
	}
	s.clouds = s.clouds[:0]
	for range min(s.w/cloudCols, maxClouds) {
		s.clouds = append(s.clouds, cloud{
			x:     float64(s.rng.IntN(max(s.w, 1))),
			y:     s.cloudRow(),
			speed: 0.04 + 0.04*s.rng.Float64(),
			shape: s.rng.IntN(len(cloudShapes)),
		})
	}
}

// IsDay reports whether t falls in the sun's half of the day.
func IsDay(t time.Time) bool {
	h := t.Hour()
	return h >= dawnHour && h < duskHour
}

// arcFraction is how far across the sky the sun (by day) or moon (by
// night) has travelled at t, in [0, 1).
func arcFraction(t time.Time) float64 {
	hours := float64(t.Hour()) + float64(t.Minute())/60 + float64(t.Second())/3600
	start := float64(dawnHour)
	if !IsDay(t) {
		start = duskHour
	}
	hours = math.Mod(hours-start+24, 24)
	return hours / (duskHour - dawnHour)
}

// arcPos is the screen cell on the arc at fraction f.
func (s *Scene) arcPos(f float64) (x, y int) {
	bottom := s.arcBottom()
	x = int(math.Round(f * float64(s.w-1)))
	y = bottom - int(math.Round(float64(bottom-arcTop)*math.Sin(math.Pi*f)))
	return x, y
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

// braille dot bits indexed by [row][col] within a cell.
var brailleBits = [4][2]rune{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

// moonGlyphs renders the moon at phase p as moonRows×moonCols braille runes;
// 0 marks a cell with no lit dots.
func moonGlyphs(p float64) [moonRows][moonCols]rune {
	const dw, dh = moonCols * 2, moonRows * 4
	var g [moonRows][moonCols]rune
	for dy := range dh {
		for dx := range dw {
			u := (float64(dx)+0.5)/(dw/2) - 1
			v := (float64(dy)+0.5)/(dh/2) - 1
			if moonLit(p, u, v) {
				g[dy/4][dx/2] |= brailleBits[dy%4][dx%2]
			}
		}
	}
	for r := range g {
		for c := range g[r] {
			if g[r][c] != 0 {
				g[r][c] += 0x2800
			}
		}
	}
	return g
}

func (s *Scene) set(x, y int, r rune, c scene.Color) {
	if r == 0 || r == ' ' || x < 0 || y < 0 || x >= s.w || y >= s.h {
		return
	}
	s.buf[y*s.w+x] = r
	s.colors[y*s.w+x] = c
}

func (s *Scene) render() {
	clear(s.buf)
	clear(s.colors)
	if !s.hasTime || s.w == 0 || s.h == 0 {
		return
	}
	x, y := s.arcPos(arcFraction(s.now))
	if IsDay(s.now) {
		for r, row := range sun {
			for c, ch := range []rune(row) {
				s.set(x-1+c, y-1+r, ch, scene.Yellow)
			}
		}
		for _, cl := range s.clouds {
			for r, row := range cloudShapes[cl.shape] {
				for c, ch := range []rune(row) {
					s.set(int(math.Floor(cl.x))+c, cl.y+r, ch, scene.White)
				}
			}
		}
		return
	}
	for _, st := range s.stars {
		g, c := st.glyph, scene.Default
		if (s.tick+st.offset)%st.period < twinkleTick {
			g, c = '+', scene.Yellow
		}
		s.set(st.x, st.y, g, c)
	}
	// The moon blots out the stars behind it, lit or not.
	mx, my := x-moonCols/2, y-moonRows/2
	moon := moonGlyphs(MoonPhase(s.now))
	for r := range moonRows {
		for c := range moonCols {
			if px, py := mx+c, my+r; px >= 0 && py >= 0 && px < s.w && py < s.h {
				s.buf[py*s.w+px] = moon[r][c]
				s.colors[py*s.w+px] = scene.White
			}
		}
	}
}

func shapeWidth(art []string) int {
	w := 0
	for _, row := range art {
		w = max(w, len([]rune(row)))
	}
	return w
}

// Package rain is the smooth rain scene: two-dot streaks falling at
// sub-cell resolution, slanted by a gusting wind, splashing on a ground line.
package rain

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/dots"
	"github.com/v4run/tcal/internal/scene"
)

const (
	step         = 33 * time.Millisecond
	cellsPerDrop = 70
	// Fall speed in rows per second (0.6-1.05 rows per 70 ms).
	minSpeed    = 0.6 / 0.07
	speedSpread = 0.45 / 0.07
	maxWind     = 1.2 // columns per row fallen
	easeRate    = 0.02
	splashLife  = 240 * time.Millisecond
	streakGap   = 0.25 // rows between the two dots
)

var (
	dropCol   = scene.RGB{R: 110, G: 150, B: 230}
	splashCol = scene.RGB{R: 120, G: 200, B: 210}
	groundCol = scene.RGB{R: 55, G: 55, B: 60}
	noBox     = scene.Box{Y0: -1, Y1: -1}
)

type drop struct {
	fx, y float64 // column (fractional) and row of the leading dot
	speed float64 // rows per second
}

type splash struct {
	x   float64
	age time.Duration
}

// Scene is the rain animation.
type Scene struct {
	rng      *rand.Rand
	w, h     int
	drops    []drop
	splashes []splash
	wind     float64
	windGoal float64
	windLeft time.Duration
	canvas   dots.Canvas
}

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.Resize(w, h)
	return s
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return step }

// Tick advances one interval.
func (s *Scene) Tick() { s.Step(step) }

// Resize changes the screen size, keeping the drop density constant.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	want := s.target()
	if len(s.drops) > want {
		s.drops = s.drops[:want]
	}
	for i := range s.drops {
		if s.drops[i].fx >= float64(s.w) || s.drops[i].y >= float64(s.ground()) {
			s.drops[i] = s.spawn(true)
		}
	}
	for len(s.drops) < want {
		s.drops = append(s.drops, s.spawn(true))
	}
	s.splashes = s.splashes[:0]
	s.canvas.Reset(s.w, s.h)
	s.render()
}

// Step advances the animation by dt.
func (s *Scene) Step(dt time.Duration) {
	if dt < 0 {
		dt = 0
	}
	sec := dt.Seconds()

	kept := s.splashes[:0]
	for _, sp := range s.splashes {
		if sp.age += dt; sp.age < splashLife {
			kept = append(kept, sp)
		}
	}
	s.splashes = kept

	s.gust(dt)
	ground := float64(s.ground())
	w := float64(s.w)
	for i := range s.drops {
		d := &s.drops[i]
		dy := d.speed * sec
		d.y += dy
		d.fx = wrap(d.fx+s.wind*dy, w)
		if d.y >= ground {
			s.splashes = append(s.splashes, splash{x: d.fx})
			*d = s.spawn(false)
		}
	}
	s.render()
}

// Glyph returns the rune at screen (x, y), or 0 for empty.
func (s *Scene) Glyph(x, y int, _ scene.Box) rune { return s.canvas.Glyph(x, y) }

// RGB returns the colour of the glyph at (x, y).
func (s *Scene) RGB(x, y int, _ scene.Box) scene.RGB { return s.canvas.RGB(x, y) }

func (s *Scene) ground() int { return s.h - 1 }

func (s *Scene) target() int {
	if s.w == 0 || s.h < 2 {
		return 0
	}
	return max(1, s.w*(s.h-1)/cellsPerDrop)
}

// spawn returns a new drop. Scattered drops start anywhere on screen; others
// start above the top edge so arrivals are staggered.
func (s *Scene) spawn(scattered bool) drop {
	d := drop{
		fx:    s.rng.Float64() * float64(s.w),
		speed: minSpeed + s.rng.Float64()*speedSpread,
	}
	if scattered {
		d.y = s.rng.Float64() * float64(s.ground())
	} else {
		d.y = -s.rng.Float64() * float64(s.h)
	}
	return d
}

// gust eases the wind toward a goal that changes every 2-7 s.
func (s *Scene) gust(dt time.Duration) {
	if s.windLeft -= dt; s.windLeft <= 0 {
		s.windLeft = 2*time.Second + time.Duration(s.rng.Float64()*float64(5*time.Second))
		if s.rng.Float64() < 0.3 {
			s.windGoal = 0
		} else {
			s.windGoal = (s.rng.Float64()*2 - 1) * maxWind
		}
	}
	k := 1 - math.Pow(1-easeRate, float64(dt)/float64(step))
	s.wind += (s.windGoal - s.wind) * k
}

func wrap(x, w float64) float64 {
	if w <= 0 {
		return 0
	}
	x = math.Mod(x, w)
	if x < 0 {
		x += w
	}
	return x
}

func (s *Scene) render() {
	s.canvas.Clear()
	if s.h == 0 || s.w == 0 {
		return
	}
	g := s.ground()
	gy := float64(g)
	for _, d := range s.drops {
		if d.y < 0 {
			continue
		}
		for i := range 2 {
			y := d.y - float64(i)*streakGap
			if y >= gy {
				continue
			}
			s.canvas.Dot(wrap(d.fx-s.wind*float64(i)*streakGap, float64(s.w)), y, dropCol)
		}
	}
	for _, sp := range s.splashes {
		p := float64(sp.age) / float64(splashLife)
		spread := 0.4 + 1.6*p
		y := gy - streakGap - 0.5*math.Sin(math.Pi*p)
		s.canvas.Dot(sp.x-spread, y, splashCol)
		s.canvas.Dot(sp.x+spread, y, splashCol)
	}
	for x := range s.w {
		s.canvas.Text(x, g, "─", groundCol)
	}
}

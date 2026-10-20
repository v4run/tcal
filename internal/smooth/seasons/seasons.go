// Package seasons is the smooth seasons background: sparse seasonal
// particles drifting at sub-cell speed in braille dots. The season follows
// the month the calendar is showing (northern hemisphere): snow in winter,
// tumbling petals in spring, softly blinking fireflies in summer and
// swaying leaves in autumn. With events enabled, the active holiday is
// reported by Event and its art, if it has any, drawn over the particles;
// the event follows the real (shifted) time, not the shown month.
package seasons

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/dots"
	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
	"github.com/v4run/tcal/internal/smooth/eventart"
)

var (
	_ scene.Scene       = (*Scene)(nil)
	_ scene.Stepper     = (*Scene)(nil)
	_ scene.TrueColored = (*Scene)(nil)
	_ scene.Clocked     = (*Scene)(nil)
)

const step = 33 * time.Millisecond

// Season is a quarter of the year.
type Season int

// Seasons, in calendar order from the start of the year.
const (
	None Season = iota // no time set yet; nothing is drawn
	Winter
	Spring
	Summer
	Autumn
)

// SeasonOf returns the northern-hemisphere season of month m.
func SeasonOf(m time.Month) Season {
	switch m {
	case time.December, time.January, time.February:
		return Winter
	case time.March, time.April, time.May:
		return Spring
	case time.June, time.July, time.August:
		return Summer
	default:
		return Autumn
	}
}

// cellsPer is the number of screen cells per particle in each season.
var cellsPer = map[Season]int{
	Winter: 90,
	Spring: 150,
	Summer: 150,
	Autumn: 160,
}

// Motion constants, per second (classic's per-100 ms values times ten).
const (
	snowMinSpeed, snowSpread = 1.2, 1.2 // rows
	snowSway, snowRate       = 0.8, 0.5 // columns, radians
	petalMinSpeed            = 0.6      // rows
	petalSpread              = 0.6
	petalDrift               = 0.6 // columns, rightward breeze
	petalSway, petalRate     = 1.5, 0.8
	leafMinSpeed, leafSpread = 0.8, 0.6 // rows
	leafSway, leafRate       = 3.0, 1.0

	flySpeed   = 0.8 // cells per second
	flyTurn    = 2.0 // radians per second, at most
	flyGlow    = 1.2 // seconds a blink lasts
	flyMinWait = 1.5 // seconds dark between blinks
	flyWait    = 4.0
)

var (
	snowCol   = scene.RGB{R: 220, G: 225, B: 235}
	petalCol  = scene.RGB{R: 230, G: 160, B: 190}
	flyCol    = scene.RGB{R: 230, G: 220, B: 120}
	leafCols  = []scene.RGB{{R: 210, G: 120, B: 50}, {R: 190, G: 150, B: 60}}
	leafSlant = [3][2][2]int{
		{{0, 0}, {1, 1}}, // '\'
		{{1, 0}, {0, 1}}, // '/'
		{{0, 1}, {1, 1}}, // '_'
	}
)

type particle struct {
	x, y  float64 // position in cells
	x0    float64 // sway centre column
	speed float64 // rows per second (fireflies: cells per second)
	phase float64 // sway / tumble phase, radians
	kind  int     // snow: dot count variant
	head  float64 // fireflies: heading, radians
	glow  float64 // fireflies: seconds into the blink cycle; <0 is dark wait
}

// eventArt is the holiday art drawn over the particles. eventart.Art
// satisfies it (it also has Bounds, which seasons never needs).
type eventArt interface {
	Resize(w, h int)
	Step(dt time.Duration)
	SetTime(now time.Time, occ events.Occurrence)
	Cell(x, y int, b scene.Box) (rune, scene.RGB)
}

// newArt builds the art for a holiday kind, or returns nil when the kind has
// none. The nil is untyped: a nil eventart.Art wrapped in eventArt would not
// compare equal to nil.
func newArt(kind events.Kind, rng *rand.Rand) eventArt {
	if a := eventart.New(kind, rng); a != nil {
		return a
	}
	return nil
}

// Scene is a field of seasonal particles over a w×h screen.
type Scene struct {
	rng    *rand.Rand
	w, h   int
	season Season
	ps     []particle
	canvas dots.Canvas

	events   bool
	birthday *events.MonthDay
	event    events.Kind
	hasEvent bool
	art      eventArt
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

// Resize changes the screen size and rescatters the particles.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.canvas.Reset(s.w, s.h)
	s.reset()
	if s.art != nil {
		s.art.Resize(s.w, s.h)
	}
}

// EnableEvents turns on holiday events, including the birthday if non-nil.
func (s *Scene) EnableEvents(birthday *events.MonthDay) {
	s.events = true
	s.birthday = birthday
}

// Event returns the active holiday event, if any.
func (s *Scene) Event() (events.Kind, bool) { return s.event, s.hasEvent }

// Season returns the season currently drawn.
func (s *Scene) Season() Season { return s.season }

// SetTime switches the season to that of the shown month and, with events
// enabled, the holiday event to the one active at now.
func (s *Scene) SetTime(now, shown time.Time) {
	if se := SeasonOf(shown.Month()); se != s.season {
		s.season = se
		s.reset()
	}
	if s.events {
		occ, ok := events.Active(now, s.birthday)
		if ok != s.hasEvent || occ.Kind != s.event {
			s.event, s.hasEvent = occ.Kind, ok
			s.art = nil
			if ok {
				s.art = newArt(occ.Kind, rand.New(rand.NewPCG(s.rng.Uint64(), s.rng.Uint64())))
			}
			if s.art != nil {
				s.art.Resize(s.w, s.h)
			}
		}
		if s.art != nil {
			s.art.SetTime(now, occ)
		}
	}
	s.render()
}

// Glyph returns the rune at screen (x, y), or 0 for empty. Event art covers
// the particles.
func (s *Scene) Glyph(x, y int, b scene.Box) rune {
	if s.art != nil {
		if r, _ := s.art.Cell(x, y, b); r != 0 {
			return r
		}
	}
	return s.canvas.Glyph(x, y)
}

// RGB returns the colour of the glyph at (x, y).
func (s *Scene) RGB(x, y int, b scene.Box) scene.RGB {
	if s.art != nil {
		if r, c := s.art.Cell(x, y, b); r != 0 {
			return c
		}
	}
	return s.canvas.RGB(x, y)
}

// reset scatters a fresh set of particles over the whole screen.
func (s *Scene) reset() {
	s.ps = s.ps[:0]
	if s.season != None && s.w > 0 && s.h > 0 {
		for range max(1, s.w*s.h/cellsPer[s.season]) {
			s.ps = append(s.ps, s.spawn(s.rng.Float64()*float64(s.h)))
		}
	}
	s.render()
}

// spawn returns a new particle at row y and a random column.
func (s *Scene) spawn(y float64) particle {
	x := s.rng.Float64() * float64(s.w)
	p := particle{x: x, y: y, x0: x, phase: s.rng.Float64() * 2 * math.Pi}
	switch s.season {
	case Winter:
		p.speed = snowMinSpeed + snowSpread*s.rng.Float64()
		p.kind = s.rng.IntN(3)
	case Spring:
		p.speed = petalMinSpeed + petalSpread*s.rng.Float64()
	case Summer:
		p.speed = flySpeed
		p.head = p.phase
		p.glow = -s.rng.Float64() * (flyMinWait + flyWait)
	case Autumn:
		p.speed = leafMinSpeed + leafSpread*s.rng.Float64()
	}
	return p
}

// Step advances every particle and the event art by dt.
func (s *Scene) Step(dt time.Duration) {
	dt = max(dt, 0)
	sec := dt.Seconds()
	if s.art != nil {
		s.art.Step(dt)
	}
	w, h := float64(s.w), float64(s.h)
	for i := range s.ps {
		p := &s.ps[i]
		switch s.season {
		case Winter:
			p.y += p.speed * sec
			p.phase += snowRate * sec
			p.x = p.x0 + snowSway*math.Sin(p.phase)
		case Spring:
			p.y += p.speed * sec
			p.x0 += petalDrift * sec
			p.phase += petalRate * sec
			p.x = p.x0 + petalSway*math.Sin(p.phase)
		case Summer:
			p.head += (s.rng.Float64()*2 - 1) * flyTurn * sec
			p.x = wrap(p.x+math.Cos(p.head)*p.speed*sec, w)
			p.y = wrap(p.y+math.Sin(p.head)*p.speed*sec*0.5, h)
			if p.glow += sec; p.glow > flyGlow {
				p.glow = -(flyMinWait + flyWait*s.rng.Float64())
			}
		case Autumn:
			p.y += p.speed * sec
			p.phase += leafRate * sec
			p.x = p.x0 + leafSway*math.Sin(p.phase)
		}
		if s.season != Summer {
			if p.y >= h {
				*p = s.spawn(0)
			}
			p.x0 = wrap(p.x0, w)
		}
	}
	s.render()
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

// plot draws a dot at dot column dx, dot row dy, wrapping sideways.
func (s *Scene) plot(dx, dy int, col scene.RGB) {
	n := s.w * 2
	dx = ((dx % n) + n) % n
	s.canvas.Dot((float64(dx)+0.5)/2, (float64(dy)+0.5)/4, col)
}

func (s *Scene) render() {
	s.canvas.Clear()
	if s.w == 0 || s.h == 0 {
		return
	}
	for i := range s.ps {
		p := &s.ps[i]
		if p.y < 0 || p.y >= float64(s.h) {
			continue
		}
		dx, dy := int(math.Floor(p.x*2)), int(math.Floor(p.y*4))
		switch s.season {
		case Winter:
			s.plot(dx, dy, snowCol)
			if p.kind == 1 {
				s.plot(dx+1, dy, snowCol)
			} else if p.kind == 2 {
				s.plot(dx, dy+1, snowCol)
			}
		case Spring:
			// A petal tumbles between upright, flat and slanted.
			switch int(math.Floor(p.phase*2/math.Pi)) & 3 {
			case 0:
				s.plot(dx, dy, petalCol)
				s.plot(dx, dy+1, petalCol)
			case 1, 3:
				s.plot(dx, dy, petalCol)
				s.plot(dx+1, dy+1, petalCol)
			default:
				s.plot(dx, dy, petalCol)
				s.plot(dx+1, dy, petalCol)
			}
		case Summer:
			if p.glow < 0 {
				continue
			}
			env := math.Sin(math.Pi * p.glow / flyGlow)
			if env < 0.15 {
				continue
			}
			s.plot(dx, dy, scale(flyCol, env))
		case Autumn:
			// Lean with the sway: cos is the sideways velocity.
			shape := 2
			if v := math.Cos(p.phase); v > 0.4 {
				shape = 0
			} else if v < -0.4 {
				shape = 1
			}
			col := leafCols[i%len(leafCols)]
			for _, d := range leafSlant[shape] {
				s.plot(dx+d[0], dy+d[1], col)
			}
		}
	}
}

func scale(c scene.RGB, k float64) scene.RGB {
	f := func(v uint8) uint8 { return uint8(float64(v) * k) }
	return scene.RGB{R: f(c.R), G: f(c.G), B: f(c.B)}
}

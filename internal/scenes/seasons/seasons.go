// Package seasons draws sparse seasonal particles over the whole screen,
// used as an ambient background in live mode. The season follows the month
// the calendar is showing (northern hemisphere): snow in winter, drifting
// petals in spring, blinking fireflies in summer and swaying leaves in
// autumn. With events enabled, a holiday's art is drawn over the particles
// while that holiday is active; the event follows the real (shifted) time,
// not the shown month.
package seasons

import (
	"math"
	mrand "math/rand"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/eventart"
	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

var (
	_ scene.Scene   = (*Scene)(nil)
	_ scene.Clocked = (*Scene)(nil)
	_ scene.Colored = (*Scene)(nil)
)

const interval = 100 * time.Millisecond

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

var (
	snowGlyphs  = []rune{'·', '*', '.'}
	petalGlyphs = []rune{'`', '\'', ',', '.'} // a petal tumbling as it falls
	// fireflyGlyphs is one blink, dark → bright → dark.
	fireflyGlyphs = []rune{'.', '·', '+', '*', '+', '·', '.'}
)

type particle struct {
	x, y  float64
	x0    float64 // sway centre column
	speed float64 // rows per tick (or wander step for fireflies)
	phase float64 // sway / tumble phase
	kind  int     // glyph variant
	blink int     // fireflies: ticks until the next blink starts (<0: blinking)
}

// Scene is a field of seasonal particles over a w×h screen.
type Scene struct {
	w, h   int
	season Season
	ps     []particle
	buf    []rune
	colors []scene.Color // parallel to buf
	tick   int
	rng    *rand.Rand

	events   bool             // holiday events enabled
	birthday *events.MonthDay // nil: no birthday event
	event    events.Kind      // valid when hasEvent
	hasEvent bool
	art      eventart.Art // nil when no event is active or it has no art
}

// New returns a w×h scene that draws randomness from rng. Nothing is drawn
// until the first SetTime.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.Resize(w, h)
	return s
}

// Resize changes the screen size and rescatters the particles.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.buf = make([]rune, s.w*s.h)
	s.colors = make([]scene.Color, s.w*s.h)
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

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return interval }

// Season returns the season currently drawn.
func (s *Scene) Season() Season { return s.season }

// SetTime switches the season to that of the shown month and, with events
// enabled, the holiday event to the one active at now.
func (s *Scene) SetTime(now, shown time.Time) {
	if se := SeasonOf(shown.Month()); se != s.season {
		s.season = se
		s.reset()
	}
	if !s.events {
		return
	}
	occ, ok := events.Active(now, s.birthday)
	if ok != s.hasEvent || occ.Kind != s.event {
		s.event, s.hasEvent = occ.Kind, ok
		s.art = nil
		if ok {
			s.art = eventart.New(occ.Kind, mrand.New(mrand.NewSource(int64(s.rng.Uint64()))))
		}
		if s.art != nil {
			s.art.Resize(s.w, s.h)
		}
	}
	if s.art != nil {
		s.art.SetTime(now, occ)
	}
}

// Glyph returns the rune drawn at screen (x, y), or 0 for empty. Event art
// covers the particles.
func (s *Scene) Glyph(x, y int, b scene.Box) rune {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return 0
	}
	if s.art != nil {
		if r, _ := s.art.Cell(x, y, b); r != 0 {
			return r
		}
	}
	return s.buf[y*s.w+x]
}

// Color returns the tint of the glyph at (x, y): the event art's color
// where it draws, otherwise white snow, magenta petals, yellow fireflies,
// and red or yellow leaves.
func (s *Scene) Color(x, y int, b scene.Box) scene.Color {
	if x < 0 || y < 0 || x >= s.w || y >= s.h {
		return scene.Default
	}
	if s.art != nil {
		if r, c := s.art.Cell(x, y, b); r != 0 {
			return c
		}
	}
	return s.colors[y*s.w+x]
}

// leafColors alternate across particle slots, so each leaf keeps its color
// as it falls (a respawned leaf inherits its slot's color).
var leafColors = []scene.Color{scene.Red, scene.Yellow}

// color returns the tint of the i'th particle.
func (s *Scene) color(i int) scene.Color {
	switch s.season {
	case Winter:
		return scene.White
	case Spring:
		return scene.Magenta
	case Summer:
		return scene.Yellow
	case Autumn:
		return leafColors[i%len(leafColors)]
	}
	return scene.Default
}

// reset scatters a fresh set of particles over the whole screen.
func (s *Scene) reset() {
	s.ps = s.ps[:0]
	if s.season != None && s.w > 0 && s.h > 0 {
		n := max(1, s.w*s.h/cellsPer[s.season])
		for range n {
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
		p.speed = 0.12 + 0.12*s.rng.Float64()
		p.kind = s.rng.IntN(len(snowGlyphs))
	case Spring:
		p.speed = 0.06 + 0.06*s.rng.Float64()
		p.kind = s.rng.IntN(len(petalGlyphs))
	case Summer:
		p.speed = 0.08
		p.blink = s.rng.IntN(40)
	case Autumn:
		p.speed = 0.08 + 0.06*s.rng.Float64()
	}
	return p
}

// Tick moves every particle one step and advances the event art.
func (s *Scene) Tick() {
	s.tick++
	if s.art != nil {
		s.art.Tick()
	}
	for i := range s.ps {
		p := &s.ps[i]
		switch s.season {
		case Winter:
			p.y += p.speed
			p.phase += 0.05
			p.x = p.x0 + 0.8*math.Sin(p.phase)
		case Spring:
			// Petals ride a light breeze to the right.
			p.y += p.speed
			p.x0 += 0.06
			p.phase += 0.08
			p.x = p.x0 + 1.5*math.Sin(p.phase)
		case Summer:
			p.x += (s.rng.Float64()*2 - 1) * p.speed * 2
			p.y += (s.rng.Float64()*2 - 1) * p.speed
			p.x = math.Mod(p.x+float64(s.w), float64(s.w))
			p.y = math.Mod(p.y+float64(s.h), float64(s.h))
			p.blink--
			if p.blink < -len(fireflyGlyphs) {
				p.blink = 20 + s.rng.IntN(40)
			}
		case Autumn:
			p.y += p.speed
			p.phase += 0.1
			p.x = p.x0 + 3*math.Sin(p.phase)
		}
		if s.season != Summer {
			if p.y >= float64(s.h) {
				*p = s.spawn(0)
			}
			p.x0 = math.Mod(p.x0+float64(s.w), float64(s.w))
		}
	}
	s.render()
}

// glyph returns the rune for p this tick, or 0 when it is not visible.
func (s *Scene) glyph(p *particle) rune {
	switch s.season {
	case Winter:
		return snowGlyphs[p.kind]
	case Spring:
		return petalGlyphs[(p.kind+int(p.y*2))%len(petalGlyphs)]
	case Summer:
		if p.blink >= 0 {
			return 0
		}
		return fireflyGlyphs[-p.blink-1]
	case Autumn:
		// Lean with the sway: cos is the sideways velocity.
		switch v := math.Cos(p.phase); {
		case v > 0.4:
			return '\\'
		case v < -0.4:
			return '/'
		default:
			return '_'
		}
	}
	return 0
}

func (s *Scene) render() {
	clear(s.buf)
	clear(s.colors)
	for i := range s.ps {
		p := &s.ps[i]
		x := int(math.Floor(p.x))
		x = ((x % max(s.w, 1)) + s.w) % max(s.w, 1)
		y := int(math.Floor(p.y))
		if y < 0 || y >= s.h || s.w == 0 {
			continue
		}
		if r := s.glyph(p); r != 0 {
			s.buf[y*s.w+x] = r
			s.colors[y*s.w+x] = s.color(i)
		}
	}
}

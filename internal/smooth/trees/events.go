package trees

import (
	"math"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

// Celebrates reports whether the scene has a way to mark the event.
func Celebrates(k events.Kind) bool {
	switch k {
	case events.Diwali, events.Onam, events.Christmas, events.Birthday:
		return true
	}
	return false
}

// EnableEvents turns on holiday events, including the birthday if non-nil.
func (s *Scene) EnableEvents(birthday *events.MonthDay) {
	s.events, s.birthday = true, birthday
}

// Event returns the active holiday event the scene is celebrating, if any.
func (s *Scene) Event() (events.Kind, bool) { return s.event, s.hasEvent }

func (s *Scene) celebrating(k events.Kind) bool { return s.hasEvent && s.event == k }

var (
	// Diwali: clay lamps with flames.
	clayCol  = scene.RGB{R: 175, G: 100, B: 60}
	flameCol = scene.RGB{R: 255, G: 195, B: 90}
	tipCol   = scene.RGB{R: 255, G: 235, B: 160}
	warmGlow = scene.RGB{R: 255, G: 180, B: 90}
	// Onam: the pookalam's rings, outermost first, and the petals.
	pookalam = []scene.RGB{
		{R: 90, G: 150, B: 70},   // leaves
		{R: 235, G: 200, B: 70},  // yellow
		{R: 235, G: 140, B: 50},  // orange
		{R: 210, G: 70, B: 60},   // red
		{R: 235, G: 230, B: 220}, // white
		{R: 160, G: 90, B: 170},  // purple
	}
	petals = []scene.RGB{{R: 235, G: 200, B: 70}, {R: 235, G: 140, B: 50}, {R: 235, G: 230, B: 220}}
	// Christmas: fairy lights and the star.
	fairyCols = []scene.RGB{
		{R: 230, G: 80, B: 70},
		{R: 110, G: 200, B: 110},
		{R: 240, G: 200, B: 90},
		{R: 110, G: 160, B: 235},
	}
	starCol = scene.RGB{R: 250, G: 220, B: 120}
	// Birthday: balloons, string and cake.
	balloonCols = []scene.RGB{{R: 225, G: 85, B: 85}, {R: 95, G: 150, B: 225}, {R: 235, G: 200, B: 80}}
	stringCol   = scene.RGB{R: 150, G: 150, B: 155}
	spongeCol   = scene.RGB{R: 225, G: 170, B: 190}
	icingCol    = scene.RGB{R: 240, G: 235, B: 230}
	candleCol   = scene.RGB{R: 120, G: 170, B: 230}
)

// look is the season's look with the event's changes: at Onam, petals fall
// and lie on the ground instead of leaves.
func (s *Scene) look() look {
	lk := looks[s.season]
	if s.celebrating(events.Onam) {
		lk.fall, lk.carpetCol = petals, petals
		lk.carpet = max(lk.carpet, 0.6)
		lk.rate = max(lk.rate, 0.4)
	}
	return lk
}

// glow is the lamp's colour: warmer at Diwali.
func (s *Scene) glow() scene.RGB {
	if s.celebrating(events.Diwali) {
		return warmGlow
	}
	return glowCol
}

// placer maps a point of tree t (dots from its foot, y up) to the screen,
// swaying with the wind: more the higher it is, more in a gust.
func (s *Scene) placer(t tree, ground float64) func(x, y float64) (float64, float64) {
	wind := s.wind()
	sway := (0.8 + 1.5*math.Abs(wind)/gustPush) * math.Sin(s.t*1.1+t.phase)
	sway += 2 * wind / gustPush
	return func(x, y float64) (float64, float64) {
		u := y / t.height
		return t.baseX + x + sway*u*u, ground - y
	}
}

// twinkle is how bright light i is now, 0 to 1: each light fades in and
// out on its own slow beat, and a few are off at any moment.
func (s *Scene) twinkle(i int) float64 {
	k := 0.55 + 0.45*math.Sin(s.t*(1.3+hash(i, 1, 5))+hash(i, 2, 5)*6.3)
	if hash(i, int(s.t*1.5), 6) < 0.12 {
		return 0
	}
	return k
}

// drawLights strings fairy lights along t's thinner branches at Christmas,
// with a star on top.
func (s *Scene) drawLights(t tree, at func(x, y float64) (float64, float64)) {
	if !s.celebrating(events.Christmas) {
		return
	}
	n := 0
	for i, l := range t.limbs {
		if l.width >= 3 {
			continue
		}
		for _, u := range []float64{0.35, 0.8} {
			if hash(t.seed, i, int(u*10)) < 0.3 {
				continue // not every spot gets a light
			}
			x, y := at(l.x0+(l.x1-l.x0)*u, l.y0+(l.y1-l.y0)*u)
			if k := s.twinkle(t.seed + n); k > 0 {
				s.dot(x, y, scale(fairyCols[n%len(fairyCols)], k))
			}
			n++
		}
	}
	// The star crowns the highest twig.
	top := t.tips[0]
	for _, tp := range t.tips {
		if tp.y > top.y {
			top = tp
		}
	}
	x, y := at(top.x, top.y+2)
	for _, d := range [][2]float64{{0, 0}, {-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		s.dot(x+d[0], y+d[1], starCol)
	}
}

// drawEvent draws the event's things on and around the bench and ground.
func (s *Scene) drawEvent(ground float64) {
	switch {
	case s.celebrating(events.Diwali):
		s.drawDiyas(ground)
	case s.celebrating(events.Onam):
		s.drawPookalam(ground)
	case s.celebrating(events.Birthday):
		s.drawBirthday(ground)
	}
}

// diya draws a clay lamp standing on dot row y, centred on x, its flame
// flickering on its own beat.
func (s *Scene) diya(x, y float64, i int) {
	s.dot(x-1, y, clayCol)
	s.dot(x, y, clayCol)
	s.dot(x+1, y, clayCol)
	s.dot(x+2, y, clayCol)
	s.dot(x, y-1, clayCol)
	s.dot(x+1, y-1, clayCol)
	s.dot(x+0.5, y-2, flameCol)
	if math.Sin(s.t*9+float64(i)*1.7)+0.3*math.Sin(s.t*23+float64(i)) > -0.2 {
		s.dot(x+0.5, y-3, tipCol) // the flame's tip comes and goes
	}
}

// drawDiyas lines diyas along the ground and three on the bench seat.
func (s *Scene) drawDiyas(ground float64) {
	solid := s.solid
	s.solid = true // the lamps sit with the bird, not under it
	defer func() { s.solid = solid }()
	const gap = 16 // dots between lamps
	for i, x := 0, 6.0; x < float64(s.w*2)-4; i, x = i+1, x+gap {
		s.diya(x, ground, i)
	}
	if bx, by, bench, _, _, _ := s.park(); bench {
		for i := range 3 {
			s.diya(bx+5+float64(i)*8, by+5, 100+i)
		}
	}
}

// drawPookalam lays a flower carpet on the ground beside the bench (or
// below the calendar without one), seen from the side: bands of colour
// from the outer ring to the centre and back.
func (s *Scene) drawPookalam(ground float64) {
	const half = 18 // dots from the edge to the centre
	cx := float64(s.w)
	if s.boxSet && s.box.Y0 != -1 {
		cx = float64(s.box.X0 + s.box.X1)
	}
	if bx, _, bench, _, _, _ := s.park(); bench {
		cx = bx - half - 6
	}
	for dx := -half; dx <= half; dx++ {
		ring := min(len(pookalam)-1, (half-abs(dx))*len(pookalam)/(half+1))
		col := pookalam[ring]
		s.dot(cx+float64(dx), ground, col)
		// The middle rings are heaped a little higher.
		if ring >= 2 {
			s.dot(cx+float64(dx), ground-1, col)
		}
	}
}

// drawBirthday ties three balloons to the bench, or to the first tree's
// trunk without one, and puts a cake with a candle on the seat.
func (s *Scene) drawBirthday(ground float64) {
	bx, by, bench, _, _, _ := s.park()
	ax, ay := bx+1, by // where the strings are tied
	room := s.freeBelow() - benchH
	if !bench || room < 14 {
		if len(s.trees) == 0 {
			return
		}
		ax, ay, room = s.trees[0].baseX+2, ground-8, 30
	}
	lift := min(room-4, 20.0) // how far above the knot the balloons float
	for i, col := range balloonCols {
		sw := 1.5 * math.Sin(s.t*0.8+float64(i)*2.1)
		x, y := ax+float64(i-1)*5+sw, ay-lift+float64(i%2)*3
		s.line(ax, ay, x, y+4, 1, stringCol)
		for dy := -3.0; dy <= 3; dy++ {
			w := 2.0
			if math.Abs(dy) == 3 {
				w = 1
			}
			for dx := -w; dx <= w; dx++ {
				s.dot(x+dx, y+dy, col)
			}
		}
	}
	if !bench {
		return
	}
	// The cake sits on the seat, just right of the middle.
	cx, seat := bx+benchW/2+2, by+5
	for dx := -3.0; dx <= 3; dx++ {
		s.dot(cx+dx, seat, spongeCol)
		s.dot(cx+dx, seat-1, icingCol)
	}
	s.dot(cx, seat-2, candleCol)
	if math.Sin(s.t*8) > -0.4 {
		s.dot(cx, seat-3, flameCol)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

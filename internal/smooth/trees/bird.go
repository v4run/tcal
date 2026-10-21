package trees

import "math"

// Where the bird can perch.
const (
	perchCrown = iota
	perchBench
	perchLamp
	perchGround
)

// perchWeight is how likely the bird is to pick each kind of perch.
var perchWeight = [4]float64{perchCrown: 3, perchBench: 2, perchLamp: 1, perchGround: 2}

const (
	birdSpeed = 35.0 // dots per second in flight
	peckEvery = 1.5  // seconds between pecks on the ground, on average
	peckLen   = 0.3  // seconds a peck lasts
)

type pt struct{ x, y float64 }

// bird is a small bird that rests on a perch, then flies to another.
type bird struct {
	x, y  float64 // its feet, in dots
	dir   float64 // 1 facing right, -1 left
	kind  int     // the perch it is on, or flying to
	rest  float64 // seconds left on this perch
	path  []pt    // the flight, one dot apart; empty when perched
	along float64 // dots flown along path
	flap  float64 // seconds in flight, for the wing beat
	peck  float64 // seconds left in the current peck
}

// spot is a place to perch.
type spot struct {
	p    pt
	kind int
}

// boxDots is the calendar box in dots, grown by one dot, and whether there
// is one.
func (s *Scene) boxDots() (x0, y0, x1, y1 float64, ok bool) {
	if !s.boxSet || s.box.Y0 == -1 {
		return 0, 0, 0, 0, false
	}
	return float64(s.box.X0*2) - 1, float64(s.box.Y0*4) - 1, float64(s.box.X1*2) + 1, float64((s.box.Y1+1)*4) + 1, true
}

// clear reports whether the path stays on screen and out of the calendar.
func (s *Scene) clear(path []pt) bool {
	x0, y0, x1, y1, ok := s.boxDots()
	ground := float64(s.h*4 - 2)
	for _, p := range path {
		if p.y < 0 || p.y > ground || p.x < 0 || p.x >= float64(s.w*2) {
			return false
		}
		if ok && p.x >= x0 && p.x < x1 && p.y >= y0 && p.y < y1 {
			return false
		}
	}
	return true
}

// spots lists where the bird could perch now.
func (s *Scene) spots() []spot {
	ground := float64(s.h*4 - 2)
	snow := 0.0
	if looks[s.season].bare {
		snow = 1
	}
	var out []spot
	for _, t := range s.trees {
		tp := t.tips[s.rng.IntN(len(t.tips))]
		y := ground - tp.y
		if !looks[s.season].bare {
			y -= tp.r * 0.3 // on the leaves, not inside them
		}
		out = append(out, spot{pt{t.baseX + tp.x, y - 1}, perchCrown})
	}
	bx, by, bench, lx, ltop, lamp := s.park()
	if bench {
		out = append(out, spot{pt{bx + 3 + s.rng.Float64()*(benchW-6), by - 1 - snow}, perchBench})
	}
	if lamp {
		out = append(out, spot{pt{lx, ltop - 1 - snow}, perchLamp})
	}
	out = append(out, spot{pt{3 + s.rng.Float64()*float64(s.w*2-6), ground}, perchGround})
	return out
}

// stepBird moves the bird along its flight, or keeps it resting and
// pecking until it is time to fly.
func (s *Scene) stepBird(sec float64) {
	if s.bird == nil {
		if len(s.trees) == 0 {
			return
		}
		for _, sp := range s.spots() {
			if sp.kind == perchCrown {
				s.bird = &bird{x: sp.p.x, y: sp.p.y, dir: 1, kind: perchCrown, rest: 3 + 5*s.rng.Float64()}
				break
			}
		}
		return
	}
	b := s.bird
	if len(b.path) > 0 {
		b.flap += sec
		b.along += birdSpeed * sec
		i := int(b.along)
		if i >= len(b.path)-1 {
			last := b.path[len(b.path)-1]
			b.x, b.y, b.path = last.x, last.y, nil
			b.rest = s.restFor(b.kind)
			return
		}
		f := b.along - float64(i)
		b.x = b.path[i].x + (b.path[i+1].x-b.path[i].x)*f
		b.y = b.path[i].y + (b.path[i+1].y-b.path[i].y)*f
		return
	}
	if b.kind == perchGround {
		if b.peck -= sec; b.peck < 0 && s.rng.Float64() < sec/peckEvery {
			b.peck = peckLen
		}
	}
	if b.rest -= sec; b.rest > 0 {
		return
	}
	if b.kind == perchGround && s.rng.Float64() < 0.5 {
		// A short hop along the ground.
		to := pt{b.x + (3+5*s.rng.Float64())*sign(s.rng.Float64()-0.5), b.y}
		if path := arc(pt{b.x, b.y}, to, 2); s.clear(path) {
			s.fly(path, perchGround)
			return
		}
	}
	// Fly to another perch by the first clear route.
	sps := s.spots()
	for range 6 {
		sp := s.pick(sps, b.kind)
		if path := s.route(pt{b.x, b.y}, sp.p); path != nil {
			s.fly(path, sp.kind)
			return
		}
	}
	b.rest = 2 // nowhere to go; look again soon
}

func (s *Scene) fly(path []pt, kind int) {
	b := s.bird
	b.path, b.along, b.flap, b.kind, b.peck = path, 0, 0, kind, 0
	if d := path[len(path)-1].x - path[0].x; d != 0 {
		b.dir = sign(d)
	}
}

func (s *Scene) restFor(kind int) float64 {
	if kind == perchCrown {
		return 6 + 10*s.rng.Float64()
	}
	return 4 + 6*s.rng.Float64()
}

// pick chooses a perch by weight, avoiding the same bench, lamp or crown
// twice in a row.
func (s *Scene) pick(sps []spot, from int) spot {
	total := 0.0
	for _, sp := range sps {
		if sp.kind != from || from == perchCrown || from == perchGround {
			total += perchWeight[sp.kind]
		}
	}
	r := s.rng.Float64() * total
	for _, sp := range sps {
		if sp.kind == from && from != perchCrown && from != perchGround {
			continue
		}
		if r -= perchWeight[sp.kind]; r < 0 {
			return sp
		}
	}
	return sps[len(sps)-1]
}

// route returns a clear flight from a to b, trying a plain arc, then a
// level crossing over the calendar, then a glide in under it.
func (s *Scene) route(a, b pt) []pt {
	dist := math.Hypot(b.x-a.x, b.y-a.y)
	if path := arc(a, b, 4+0.12*dist); s.clear(path) {
		return path
	}
	x0, y0, x1, y1, ok := s.boxDots()
	if !ok {
		return nil
	}
	if peak := y0 - 3; peak > 2 && peak < min(a.y, b.y) {
		if path := over(a, b, peak); s.clear(path) {
			return path
		}
	}
	// Under: drop to below the calendar beside it, glide across, rise.
	under := y1 + 4
	if under < float64(s.h*4-4) {
		side := func(p pt) pt {
			x := p.x
			if p.y < y1 && x >= x0 && x < x1 {
				return p // already beside or below; no corner needed
			}
			return pt{x, under}
		}
		poly := []pt{a, side(a), side(b), b}
		if path := resample(chaikin(chaikin(poly))); s.clear(path) {
			return path
		}
	}
	return nil
}

// arc is a flight from a to b that rises lift dots above the straight
// line at its middle.
func arc(a, b pt, lift float64) []pt {
	n := int(math.Hypot(b.x-a.x, b.y-a.y)*4) + 8
	path := make([]pt, 0, n+1)
	for i := 0; i <= n; i++ {
		u := float64(i) / float64(n)
		path = append(path, pt{a.x + (b.x-a.x)*u, a.y + (b.y-a.y)*u - lift*4*u*(1-u)})
	}
	return resample(path)
}

// over is a flat-topped flight from a to b that crosses level at row peak.
func over(a, b pt, peak float64) []pt {
	n := int(math.Abs(b.x-a.x)*4) + 8
	path := make([]pt, 0, n+1)
	for i := 0; i <= n; i++ {
		u := float64(i) / float64(n)
		v := 2*u - 1
		base := a.y + (b.y-a.y)*u
		path = append(path, pt{a.x + (b.x-a.x)*u, base - (base-peak)*(1-v*v*v*v)})
	}
	return resample(path)
}

// chaikin rounds the corners of a polyline.
func chaikin(p []pt) []pt {
	out := []pt{p[0]}
	for i := 0; i+1 < len(p); i++ {
		a, b := p[i], p[i+1]
		out = append(out, pt{0.75*a.x + 0.25*b.x, 0.75*a.y + 0.25*b.y}, pt{0.25*a.x + 0.75*b.x, 0.25*a.y + 0.75*b.y})
	}
	return append(out, p[len(p)-1])
}

// resample returns points one dot apart along p.
func resample(p []pt) []pt {
	out := []pt{p[0]}
	carry := 0.0
	for i := 0; i+1 < len(p); i++ {
		a, b := p[i], p[i+1]
		seg := math.Hypot(b.x-a.x, b.y-a.y)
		for d := 1 - carry; d <= seg; d++ {
			u := d / seg
			out = append(out, pt{a.x + (b.x-a.x)*u, a.y + (b.y-a.y)*u})
		}
		carry = math.Mod(carry+seg, 1)
	}
	if last := p[len(p)-1]; out[len(out)-1] != last {
		out = append(out, last)
	}
	return out
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}

// drawBird draws the bird perched (pecking on the ground now and then) or
// flying with its wings beating.
func (s *Scene) drawBird(_ look) {
	b := s.bird
	if b == nil {
		return
	}
	d := b.dir
	if len(b.path) > 0 {
		wing := -1.0
		if math.Sin(b.flap*14) < 0 {
			wing = 1
		}
		s.dot(b.x, b.y-1, birdCol)
		s.dot(b.x-1, b.y-1+wing, birdCol)
		s.dot(b.x+1, b.y-1+wing, birdCol)
		s.dot(b.x-2, b.y-1+wing*0.5, birdCol)
		s.dot(b.x+2, b.y-1+wing*0.5, birdCol)
		return
	}
	// Feet at (x, y): legs, a body row, the head up front and the tail
	// behind; pecking puts the head down.
	s.dot(b.x, b.y, birdCol)
	for dx := -1.0; dx <= 1; dx++ {
		s.dot(b.x+dx*d, b.y-1, birdCol)
	}
	s.dot(b.x-2*d, b.y-2, birdCol)
	if b.peck > 0 {
		s.dot(b.x+2*d, b.y, birdCol)
	} else {
		s.dot(b.x+d, b.y-2, birdCol)
	}
}

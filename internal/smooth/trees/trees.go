// Package trees is the smooth trees scene: a large tree in each margin
// beside the calendar that follows the season of the month on screen.
// Blossoms drift off them in spring, a leaf now and then in summer, many in
// autumn, and snow slips off bare branches in winter. Fallen leaves rest on
// the ground for a while, then fade.
package trees

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/dots"
	"github.com/v4run/tcal/internal/scene"
	"github.com/v4run/tcal/internal/smooth/seasons"
)

const step = 33 * time.Millisecond

// minMargin is the narrowest margin, in columns, that gets a tree.
const minMargin = 12

const (
	fallSpeed = 6.0  // dots per second, downwards
	swayAmp   = 3.0  // dots
	restTime  = 8.0  // seconds a fallen leaf rests before fading
	fadeTime  = 3.0  // seconds it takes to fade
	gustMin   = 8.0  // seconds between gusts, at least
	gustVar   = 10.0 // extra random seconds
	gustLen   = 2.5  // seconds a gust lasts
	gustPush  = 10.0 // dots per second of sideways push at a gust's peak
	maxDepth  = 5    // branchings from trunk to twig
)

var (
	_ scene.Scene       = (*Scene)(nil)
	_ scene.Stepper     = (*Scene)(nil)
	_ scene.TrueColored = (*Scene)(nil)
	_ scene.Clocked     = (*Scene)(nil)
)

// look is how a season dresses the trees.
type look struct {
	crown   []scene.RGB // leaf colours, picked per dot pair
	fill    float64     // fraction of a cluster's dot pairs drawn
	accent  scene.RGB   // blossoms in spring
	accentP float64     // fraction of drawn pairs in the accent colour
	fall    []scene.RGB // colours of what falls
	rate    float64     // falling things per tree per second
	ground  scene.RGB
	bare    bool // winter: no leaves, snow along the branches
}

var looks = map[seasons.Season]look{
	seasons.Spring: {
		crown:  []scene.RGB{{R: 120, G: 170, B: 90}, {R: 100, G: 155, B: 80}},
		fill:   0.42,
		accent: scene.RGB{R: 230, G: 170, B: 190}, accentP: 0.4,
		fall:   []scene.RGB{{R: 230, G: 170, B: 190}, {R: 240, G: 200, B: 210}},
		rate:   0.5,
		ground: scene.RGB{R: 85, G: 125, B: 65},
	},
	seasons.Summer: {
		crown:  []scene.RGB{{R: 60, G: 120, B: 60}, {R: 80, G: 140, B: 70}, {R: 50, G: 105, B: 55}},
		fill:   0.6,
		fall:   []scene.RGB{{R: 90, G: 145, B: 70}},
		rate:   0.08,
		ground: scene.RGB{R: 75, G: 120, B: 60},
	},
	seasons.Autumn: {
		crown:  []scene.RGB{{R: 200, G: 110, B: 45}, {R: 210, G: 160, B: 60}, {R: 170, G: 70, B: 40}},
		fill:   0.38,
		fall:   []scene.RGB{{R: 200, G: 110, B: 45}, {R: 210, G: 160, B: 60}, {R: 170, G: 70, B: 40}},
		rate:   1.2,
		ground: scene.RGB{R: 125, G: 100, B: 65},
	},
	seasons.Winter: {
		fall:   []scene.RGB{{R: 220, G: 225, B: 235}},
		rate:   0.3,
		ground: scene.RGB{R: 195, G: 200, B: 210},
		bare:   true,
	},
}

var (
	trunkCol = scene.RGB{R: 115, G: 85, B: 60}
	snowCol  = scene.RGB{R: 220, G: 225, B: 235}
)

// limb is one branch segment, in dots relative to the tree's foot, y up.
type limb struct {
	x0, y0, x1, y1 float64
	width          float64 // dots
}

// tip is a leaf cluster at the end of a twig.
type tip struct{ x, y, r float64 }

type tree struct {
	baseX  float64 // dot column of the trunk's foot
	height float64 // dots from foot to the top of the crown
	limbs  []limb
	tips   []tip
	phase  float64
	seed   int
}

// leaf is something falling from a tree, or resting on the ground.
type leaf struct {
	x, y, x0 float64
	phase    float64
	col      scene.RGB
	rest     float64 // seconds on the ground; -1 while falling
}

// Scene is the pair of trees.
type Scene struct {
	rng    *rand.Rand
	w, h   int
	t      float64
	season seasons.Season
	box    scene.Box // the calendar box the trees are fitted around
	boxSet bool
	seeds  [2]uint64 // tree shapes, fixed per side across re-layouts
	trees  []tree
	leaves []leaf
	gustIn float64 // seconds until the next gust
	gustT  float64 // seconds into the current gust, or -1
	canvas dots.Canvas
}

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng, season: seasons.SeasonOf(time.Now().Month()), gustT: -1}
	s.seeds = [2]uint64{rng.Uint64(), rng.Uint64()}
	s.gustIn = gustMin + rng.Float64()*gustVar
	s.Resize(w, h)
	return s
}

// SetTime switches to the season of the shown month.
func (s *Scene) SetTime(_, shown time.Time) {
	if se := seasons.SeasonOf(shown.Month()); se != s.season {
		s.season = se
		s.leaves = nil
		s.render()
	}
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return step }

// Tick advances one interval.
func (s *Scene) Tick() { s.Step(step) }

// Resize changes the screen size and refits the trees.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.canvas.Reset(s.w, s.h)
	s.layout()
	s.render()
}

// layout fits a tree into each margin beside the calendar box, or into the
// outer thirds of an empty screen.
func (s *Scene) layout() {
	s.trees, s.leaves = nil, nil
	if s.w == 0 || s.h < 6 {
		return
	}
	left, right := [2]int{0, s.w * 35 / 100}, [2]int{s.w * 65 / 100, s.w}
	if s.boxSet && s.box.Y0 != -1 {
		left, right = [2]int{0, s.box.X0}, [2]int{s.box.X1, s.w}
	}
	for i, m := range [][2]int{left, right} {
		if m[1]-m[0] < minMargin {
			continue
		}
		rng := rand.New(rand.NewPCG(s.seeds[i], 7))
		s.trees = append(s.trees, grow(rng, float64(m[0]*2), float64(m[1]*2), float64(s.h*4-2)))
	}
}

// grow builds a branching tree that fits between dot columns x0 and x1 and
// is at most maxH dots tall, its foot on the ground.
func grow(rng *rand.Rand, x0, x1, maxH float64) tree {
	t := tree{phase: rng.Float64() * 2 * math.Pi, seed: rng.IntN(1 << 20)}
	var branch func(x, y, angle, length, width float64, depth int)
	branch = func(x, y, angle, length, width float64, depth int) {
		ex, ey := x+length*math.Sin(angle), y+length*math.Cos(angle)
		t.limbs = append(t.limbs, limb{x, y, ex, ey, width})
		if depth == maxDepth {
			t.tips = append(t.tips, tip{ex, ey, length * 0.9})
			return
		}
		if depth >= maxDepth-1 {
			t.tips = append(t.tips, tip{ex, ey, length * 0.7})
		}
		n := 2
		if depth > 0 && rng.Float64() < 0.35 {
			n = 3
		}
		spread := 0.3 + 0.15*rng.Float64()
		for k := range n {
			a := angle + spread*(2*float64(k)/float64(n-1)-1)
			a += (rng.Float64() - 0.5) * 0.25
			a *= 0.92 // lean back toward upright
			branch(ex, ey, a, length*(0.72+0.1*rng.Float64()), width*0.65, depth+1)
		}
	}
	// A long trunk makes the tree tall rather than wide.
	branch(0, 0, (rng.Float64()-0.5)*0.1, 1.7, 0.16, 0)

	// Scale the unit tree into the margin.
	minX, maxX, maxY := 0.0, 0.0, 0.0
	for _, tp := range t.tips {
		minX, maxX = min(minX, tp.x-tp.r), max(maxX, tp.x+tp.r)
		maxY = max(maxY, tp.y+tp.r)
	}
	// Fit the width; stretch upward (to at most maxStretch) to fill the
	// height a narrow margin leaves unused. Leaf clusters stay round.
	const maxStretch = 1.8
	kx := min((x1-x0-4)/(maxX-minX), maxH*0.92/maxY)
	ky := min(maxH*0.92/maxY, kx*maxStretch)
	t.baseX = (x0+x1)/2 - (minX+maxX)/2*kx
	t.height = maxY * ky
	for i := range t.limbs {
		l := &t.limbs[i]
		l.x0, l.y0, l.x1, l.y1 = l.x0*kx, l.y0*ky, l.x1*kx, l.y1*ky
		l.width = max(1, l.width*kx)
	}
	for i := range t.tips {
		tp := &t.tips[i]
		tp.x, tp.y, tp.r = tp.x*kx, tp.y*ky, max(2, tp.r*kx)
	}
	return t
}

// wind is the sideways push of the current gust, in dots per second.
func (s *Scene) wind() float64 {
	if s.gustT < 0 {
		return 0
	}
	return gustPush * math.Sin(math.Pi*s.gustT/gustLen)
}

// Step advances the animation by dt.
func (s *Scene) Step(dt time.Duration) {
	sec := max(dt, 0).Seconds()
	s.t += sec
	if s.gustT >= 0 {
		if s.gustT += sec; s.gustT >= gustLen {
			s.gustT = -1
			s.gustIn = gustMin + s.rng.Float64()*gustVar
		}
	} else if s.gustIn -= sec; s.gustIn <= 0 {
		s.gustT = 0
	}

	lk := looks[s.season]
	wind := s.wind()
	for i := range s.trees {
		// Gusts shake more loose.
		if s.rng.Float64() < lk.rate*sec*(1+3*math.Abs(wind)/gustPush) {
			s.drop(&s.trees[i], lk)
		}
	}

	ground := float64(s.h*4 - 2)
	kept := s.leaves[:0]
	for _, l := range s.leaves {
		if l.rest >= 0 {
			if l.rest += sec; l.rest >= restTime+fadeTime {
				continue
			}
			kept = append(kept, l)
			continue
		}
		l.phase += 1.6 * sec
		l.x0 += wind * sec
		l.y += fallSpeed * sec
		l.x = l.x0 + swayAmp*math.Sin(l.phase)
		if l.y >= ground {
			l.y, l.rest = ground, 0
		}
		if l.x < -2 || l.x > float64(s.w*2)+2 {
			continue
		}
		kept = append(kept, l)
	}
	s.leaves = kept
	s.render()
}

// drop lets one leaf (or petal, or clump of snow) go from a random twig.
func (s *Scene) drop(t *tree, lk look) {
	tp := t.tips[s.rng.IntN(len(t.tips))]
	a := s.rng.Float64() * 2 * math.Pi
	rr := tp.r * math.Sqrt(s.rng.Float64())
	x := t.baseX + tp.x + rr*math.Cos(a)
	s.leaves = append(s.leaves, leaf{
		x: x, x0: x,
		y:     float64(s.h*4-2) - (tp.y + rr*math.Sin(a)),
		phase: s.rng.Float64() * 2 * math.Pi,
		col:   lk.fall[s.rng.IntN(len(lk.fall))],
		rest:  -1,
	})
}

// Glyph returns the rune at screen (x, y), or 0 for empty. When it sees a
// new calendar box it refits the trees around it.
func (s *Scene) Glyph(x, y int, b scene.Box) rune {
	// An empty box also comes from cells drawn behind the calendar, so it
	// only counts before any real box has been seen.
	if b != s.box && (b.Y0 != -1 || !s.boxSet) {
		s.box, s.boxSet = b, true
		s.layout()
		s.render()
	}
	return s.canvas.Glyph(x, y)
}

// RGB returns the colour of the glyph at (x, y).
func (s *Scene) RGB(x, y int, _ scene.Box) scene.RGB { return s.canvas.RGB(x, y) }

func (s *Scene) dot(x, y float64, col scene.RGB) {
	if y < 0 || x < 0 {
		return
	}
	s.canvas.Dot(x/2, y/4, col)
}

// line plots a line of dots from (x0, y0) to (x1, y1), width dots wide.
func (s *Scene) line(x0, y0, x1, y1, width float64, col scene.RGB) {
	n := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))) + 1
	for i := 0; i <= n; i++ {
		u := float64(i) / float64(n)
		x, y := x0+(x1-x0)*u, y0+(y1-y0)*u
		for d := 0.0; d < width; d++ {
			s.dot(x+d-(width-1)/2, y, col)
		}
	}
}

func scale(c scene.RGB, k float64) scene.RGB {
	f := func(x uint8) uint8 { return uint8(float64(x)*k + 0.5) }
	return scene.RGB{R: f(c.R), G: f(c.G), B: f(c.B)}
}

// hash maps (a, b, c) to a stable value in [0, 1).
func hash(a, b, c int) float64 {
	z := uint64(a)*0x9e3779b97f4a7c15 ^ uint64(b)*0xbf58476d1ce4e5b9 ^ uint64(c)*0x94d049bb133111eb
	z = (z ^ z>>30) * 0xbf58476d1ce4e5b9
	z = (z ^ z>>27) * 0x94d049bb133111eb
	z ^= z >> 31
	return float64(z>>11) / (1 << 53)
}

func (s *Scene) render() {
	s.canvas.Clear()
	if s.w == 0 || s.h == 0 {
		return
	}
	lk := looks[s.season]
	wind := s.wind()
	ground := float64(s.h*4 - 2)

	for x := range s.w * 2 {
		s.dot(float64(x), ground+1, scale(lk.ground, 0.8))
	}

	for _, t := range s.trees {
		// Sway grows with height, more in a gust.
		sway := (0.8 + 1.5*math.Abs(wind)/gustPush) * math.Sin(s.t*1.1+t.phase)
		sway += 2 * wind / gustPush
		bend := func(y float64) float64 {
			u := y / t.height
			return sway * u * u
		}
		at := func(x, y float64) (float64, float64) {
			return t.baseX + x + bend(y), ground - y
		}

		for _, l := range t.limbs {
			ax, ay := at(l.x0, l.y0)
			bx, by := at(l.x1, l.y1)
			s.line(ax, ay, bx, by, l.width, trunkCol)
			if lk.bare && l.width < 3 {
				// Snow lies along the upper side of the thinner branches.
				s.line(ax, ay-1, bx, by-1, 1, snowCol)
			}
		}
		if lk.bare {
			continue
		}
		for _, tp := range t.tips {
			cx, cy := at(tp.x, tp.y)
			for y := int(cy - tp.r); y <= int(cy+tp.r); y++ {
				for x := int(cx - tp.r); x <= int(cx+tp.r); x++ {
					dx, dy := float64(x)-cx, float64(y)-cy
					if dx*dx+dy*dy > tp.r*tp.r {
						continue
					}
					// Dots come in vertical pairs within a cell, fixed to
					// the tree, so no cell holds a lone dot.
					hx := x - int(math.Floor(cx-tp.x-t.baseX))
					h := hash(t.seed, hx, y/2)
					if h >= lk.fill {
						continue
					}
					col := lk.crown[int(h*1000)%len(lk.crown)]
					if lk.accentP > 0 && h < lk.fill*lk.accentP {
						col = lk.accent
					}
					s.dot(float64(x), float64(y), col)
				}
			}
		}
	}

	for _, l := range s.leaves {
		col := l.col
		if l.rest > restTime {
			col = scale(col, 1-(l.rest-restTime)/fadeTime)
		}
		s.dot(l.x, l.y, col)
		if l.rest < 0 && math.Sin(l.phase*2) <= 0 {
			// A falling leaf turns: two dots, side by side or stacked.
			s.dot(l.x, l.y+1, col)
		} else {
			s.dot(l.x+1, l.y, col)
		}
	}
}

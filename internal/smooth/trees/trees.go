// Package trees is the smooth trees scene: a row of trees along the bottom
// of the screen that follow the season of the month on screen. Blossoms
// drift off them in spring, a leaf now and then in summer, many in autumn,
// and snow slips off bare branches in winter. Fallen leaves rest on the
// ground for a while, then fade.
package trees

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/dots"
	"github.com/v4run/tcal/internal/scene"
	"github.com/v4run/tcal/internal/smooth/seasons"
)

// Height is the number of rows the strip occupies at the bottom of the
// screen.
const Height = 10

// MinWidth is the narrowest screen the scene will draw on.
const MinWidth = 40

const step = 33 * time.Millisecond

// Dot geometry of the strip (Height*4 = 40 dot rows).
const groundRow = 38

const (
	fallSpeed = 5.0  // dots per second, downwards
	swayAmp   = 2.5  // dots
	restTime  = 8.0  // seconds a fallen leaf rests before fading
	fadeTime  = 3.0  // seconds it takes to fade
	gustMin   = 8.0  // seconds between gusts, at least
	gustVar   = 10.0 // extra random seconds
	gustLen   = 2.5  // seconds a gust lasts
	gustPush  = 9.0  // dots per second of sideways push at a gust's peak
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
	fill    float64     // fraction of the crown's dot pairs drawn
	accent  scene.RGB   // blossoms in spring
	accentP float64     // fraction of drawn pairs in the accent colour
	fall    []scene.RGB // colours of what falls
	rate    float64     // falling things per tree per second
	ground  scene.RGB
	bare    bool // winter: no crown, snow along the branches
}

var looks = map[seasons.Season]look{
	seasons.Spring: {
		crown:  []scene.RGB{{R: 120, G: 170, B: 90}, {R: 100, G: 155, B: 80}},
		fill:   0.38,
		accent: scene.RGB{R: 230, G: 170, B: 190}, accentP: 0.35,
		fall:   []scene.RGB{{R: 230, G: 170, B: 190}, {R: 240, G: 200, B: 210}},
		rate:   0.3,
		ground: scene.RGB{R: 85, G: 125, B: 65},
	},
	seasons.Summer: {
		crown:  []scene.RGB{{R: 60, G: 120, B: 60}, {R: 80, G: 140, B: 70}, {R: 50, G: 105, B: 55}},
		fill:   0.55,
		fall:   []scene.RGB{{R: 90, G: 145, B: 70}},
		rate:   0.04,
		ground: scene.RGB{R: 75, G: 120, B: 60},
	},
	seasons.Autumn: {
		crown:  []scene.RGB{{R: 200, G: 110, B: 45}, {R: 210, G: 160, B: 60}, {R: 170, G: 70, B: 40}},
		fill:   0.32,
		fall:   []scene.RGB{{R: 200, G: 110, B: 45}, {R: 210, G: 160, B: 60}, {R: 170, G: 70, B: 40}},
		rate:   0.7,
		ground: scene.RGB{R: 125, G: 100, B: 65},
	},
	seasons.Winter: {
		fall:   []scene.RGB{{R: 220, G: 225, B: 235}},
		rate:   0.15,
		ground: scene.RGB{R: 195, G: 200, B: 210},
		bare:   true,
	},
}

var (
	trunkCol = scene.RGB{R: 115, G: 85, B: 60}
	snowCol  = scene.RGB{R: 220, G: 225, B: 235}
)

// blob is one round part of a crown, relative to the trunk's top.
type blob struct{ dx, dy, r float64 }

// branch is a line from the trunk's top, in dots.
type branch struct{ dx0, dy0, dx1, dy1 float64 }

type tree struct {
	x     float64 // dot column of the trunk's centre
	top   float64 // dot row of the trunk's top
	blobs []blob
	limbs []branch
	phase float64
	seed  int
}

// leaf is something falling from a tree, or resting on the ground.
type leaf struct {
	x, y, x0 float64
	phase    float64
	col      scene.RGB
	rest     float64 // seconds on the ground; -1 while falling
}

// Scene is the row of trees.
type Scene struct {
	rng    *rand.Rand
	w, h   int
	t      float64
	season seasons.Season
	trees  []tree
	leaves []leaf
	gustIn float64 // seconds until the next gust
	gustT  float64 // seconds into the current gust, or -1
	canvas dots.Canvas
}

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng, season: seasons.SeasonOf(time.Now().Month()), gustT: -1}
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

// Resize plants trees across the new width.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.trees, s.leaves = nil, nil
	if s.w >= MinWidth {
		W := float64(s.w * 2)
		n := max(2, s.w/28)
		for i := range n {
			x := W * (float64(i) + 0.5 + 0.3*(s.rng.Float64()-0.5)) / float64(n)
			s.trees = append(s.trees, s.newTree(x))
		}
	}
	s.canvas.Reset(s.w, Height)
	s.render()
}

func (s *Scene) newTree(x float64) tree {
	// The tallest crown top, at size 1.2, is 2 dot rows below the strip's
	// top: trunk 14*size plus crown 2.3*r with r = 7*size.
	size := 0.9 + 0.3*s.rng.Float64()
	t := tree{
		x:     x,
		top:   groundRow - 14*size,
		phase: s.rng.Float64() * 2 * math.Pi,
		seed:  s.rng.IntN(1 << 20),
	}
	r := 7 * size
	t.blobs = []blob{
		{0, -r * 0.9, r},
		{-r * 0.8, -r * 0.4, r * 0.75},
		{r * 0.8, -r * 0.5, r * 0.8},
		{(s.rng.Float64() - 0.5) * r, -r * 1.6, r * 0.7},
	}
	t.limbs = []branch{
		{0, 0, -r * 0.9, -r * 0.8},
		{0, 0, r * 0.9, -r * 0.9},
		{0, 0, 0, -r * 1.6},
		{-r * 0.45, -r * 0.4, -r * 1.3, -r * 1.1},
		{r * 0.45, -r * 0.45, r * 1.3, -r * 1.3},
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
	if s.w < MinWidth {
		return
	}
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
		t := &s.trees[i]
		// Gusts shake more loose.
		if s.rng.Float64() < lk.rate*sec*(1+3*math.Abs(wind)/gustPush) {
			s.drop(t, lk)
		}
	}

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
		if l.y >= groundRow-1 {
			l.y, l.rest = groundRow-1, 0
		}
		if l.x < -2 || l.x > float64(s.w*2)+2 {
			continue
		}
		kept = append(kept, l)
	}
	s.leaves = kept
	s.render()
}

// drop lets one leaf (or petal, or clump of snow) go from somewhere in the
// tree's crown.
func (s *Scene) drop(t *tree, lk look) {
	b := t.blobs[s.rng.IntN(len(t.blobs))]
	a := s.rng.Float64() * 2 * math.Pi
	rr := b.r * math.Sqrt(s.rng.Float64())
	x := t.x + b.dx + rr*math.Cos(a)
	s.leaves = append(s.leaves, leaf{
		x: x, x0: x,
		y:     t.top + b.dy + rr*math.Sin(a),
		phase: s.rng.Float64() * 2 * math.Pi,
		col:   lk.fall[s.rng.IntN(len(lk.fall))],
		rest:  -1,
	})
}

// Glyph returns the rune at screen (x, y), or 0 for empty. The scene hides
// whole when its strip would touch the calendar box b.
func (s *Scene) Glyph(x, y int, b scene.Box) rune {
	if s.w < MinWidth || b.Y1 >= s.h-Height {
		return 0
	}
	return s.canvas.Glyph(x, y-(s.h-Height))
}

// RGB returns the colour of the glyph at (x, y).
func (s *Scene) RGB(x, y int, _ scene.Box) scene.RGB {
	return s.canvas.RGB(x, y-(s.h-Height))
}

func (s *Scene) dot(x, y float64, col scene.RGB) {
	if y < 0 || y >= Height*4 || x < 0 {
		return
	}
	s.canvas.Dot(x/2, y/4, col)
}

// line plots a line of dots from (x0, y0) to (x1, y1).
func (s *Scene) line(x0, y0, x1, y1 float64, col scene.RGB) {
	n := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))) + 1
	for i := 0; i <= n; i++ {
		u := float64(i) / float64(n)
		s.dot(x0+(x1-x0)*u, y0+(y1-y0)*u, col)
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
	if s.w < MinWidth {
		return
	}
	lk := looks[s.season]
	wind := s.wind()

	for x := range s.w * 2 {
		s.dot(float64(x), groundRow+1, scale(lk.ground, 0.8))
	}

	for _, t := range s.trees {
		// The crown sways a little, more in a gust.
		sway := (0.6 + 1.2*math.Abs(wind)/gustPush) * math.Sin(s.t*1.1+t.phase)
		sway += 1.5 * wind / gustPush
		bend := func(dy float64) float64 { return sway * math.Min(1, -dy/14) }

		s.line(t.x, groundRow, t.x, t.top, trunkCol)
		s.line(t.x+1, groundRow, t.x+1, t.top+2, trunkCol)
		for _, l := range t.limbs {
			if !lk.bare {
				// Keep the tips inside the crown.
				l.dx1, l.dy1 = l.dx0+(l.dx1-l.dx0)*0.6, l.dy0+(l.dy1-l.dy0)*0.6
			}
			s.line(t.x+l.dx0+bend(l.dy0), t.top+l.dy0, t.x+l.dx1+bend(l.dy1), t.top+l.dy1, trunkCol)
			if lk.bare {
				// Snow lies along the upper side of each branch.
				s.line(t.x+l.dx0+bend(l.dy0)+0.5, t.top+l.dy0-1, t.x+l.dx1+bend(l.dy1)+0.5, t.top+l.dy1-1, snowCol)
			}
		}
		if lk.bare {
			continue
		}
		for _, b := range t.blobs {
			cx, cy := t.x+b.dx+bend(b.dy), t.top+b.dy
			for y := int(cy - b.r); y <= int(cy+b.r); y++ {
				for x := int(cx - b.r); x <= int(cx+b.r); x++ {
					dx, dy := float64(x)-cx, float64(y)-cy
					if dx*dx+dy*dy > b.r*b.r {
						continue
					}
					// Dots come in vertical pairs within a cell, fixed to
					// the tree, so no cell holds a lone dot.
					hx := x - int(math.Floor(cx-t.x))
					h := hash(t.seed, hx, (y-int(t.top))/2+100)
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
		if l.rest < 0 {
			// A falling leaf turns: two dots, side by side or stacked.
			if math.Sin(l.phase*2) > 0 {
				s.dot(l.x+1, l.y, col)
			} else {
				s.dot(l.x, l.y+1, col)
			}
		} else {
			s.dot(l.x+1, l.y, col)
		}
	}
}

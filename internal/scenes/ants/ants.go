// Package ants implements an ant trail for live mode: small groups of ants
// marching clockwise around the calendar, one cell outside its padded box,
// now and then stopping for a moment before carrying on.
package ants

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

const (
	interval    = 150 * time.Millisecond
	minGroup    = 2   // fewest ants in a group
	groupSpread = 4   // groups are minGroup..minGroup+groupSpread-1 ants
	minStride   = 2   // minimum cells from one ant to the next in a group
	strideSpan  = 2   // strides are minStride..minStride+strideSpan-1
	minGap      = 10  // minimum empty cells between groups
	gapSpread   = 22  // extra random cells between groups
	keepClear   = 3   // a group stops this many cells behind the next
	pauseOdds   = 250 // a marching group pauses with chance 1/pauseOdds per tick
	minPause    = 6   // shortest pause, in ticks
	pauseSpread = 20  // extra random pause ticks
)

// legs are the two strides of an ant, alternating as it walks.
var legs = [2]rune{'⠖', '⠲'}

// group is a file of ants walking together. Positions are offsets along
// the ring, which is walked clockwise from its top-left corner.
type group struct {
	lead   int   // ring offset of the front ant
	behind []int // distance of each ant behind lead, ascending, starting at 0
	pause  int   // ticks left to stand still
}

func (g *group) tail() int { return g.lead - g.behind[len(g.behind)-1] }

// Scene is the trail. Ant positions live on the ring as offsets, advanced
// by Tick; Glyph maps them to cells for the box it is given.
type Scene struct {
	w, h   int
	groups []*group // in ring order: groups[i+1] walks ahead of groups[i]
	perim  int      // length of the ring the offsets refer to; 0 before the first box

	box   scene.Box
	known bool   // box and ring are valid for the current size
	ring  []int  // ring offset → index into grid, or -1 when off screen
	grid  []rune // w×h cells holding the ants drawn this tick
	drawn []int  // grid indexes set in grid, to clear on redraw
	dirty bool   // grid is stale
	rng   *rand.Rand
}

var _ scene.Scene = (*Scene)(nil)

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.Resize(w, h)
	return s
}

// Resize changes the screen size; the ring is rebuilt on the next Glyph.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.grid = make([]rune, s.w*s.h)
	s.drawn = s.drawn[:0]
	s.known = false
	s.dirty = true
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return interval }

// Tick walks every group one cell along the ring, unless it is pausing or
// would close in on the group ahead.
func (s *Scene) Tick() {
	if s.perim == 0 || len(s.groups) == 0 {
		return
	}
	moved := false
	for i, g := range s.groups {
		if g.pause > 0 {
			g.pause--
			continue
		}
		if s.rng.IntN(pauseOdds) == 0 {
			g.pause = minPause + s.rng.IntN(pauseSpread)
			continue
		}
		if len(s.groups) > 1 {
			ahead := s.groups[(i+1)%len(s.groups)]
			if mod(ahead.tail()-g.lead, s.perim) <= keepClear {
				continue
			}
		}
		g.lead = mod(g.lead+1, s.perim)
		moved = true
	}
	if !moved {
		// Everyone is waiting on everyone (e.g. after the ring shrank):
		// nudge the lot forward so the trail can't jam.
		for _, g := range s.groups {
			if g.pause == 0 {
				g.lead = mod(g.lead+1, s.perim)
			}
		}
	}
	s.dirty = true
}

// Glyph returns the ant at screen (x, y), or 0. Ants walk the ring of
// cells just outside b, or the screen border when b is empty.
func (s *Scene) Glyph(x, y int, b scene.Box) rune {
	if x < 0 || x >= s.w || y < 0 || y >= s.h {
		return 0
	}
	if !s.known || b != s.box {
		s.setBox(b)
	}
	if s.dirty {
		s.draw()
	}
	return s.grid[y*s.w+x]
}

// setBox rebuilds the ring for b and rescales ant offsets onto it,
// populating the trail the first time a ring is seen.
func (s *Scene) setBox(b scene.Box) {
	s.box, s.known, s.dirty = b, true, true
	l, t, r, bot := b.X0-1, b.Y0-1, b.X1, b.Y1+1
	if b.Y0 < 0 {
		l, t, r, bot = 0, 0, s.w-1, s.h-1
	}
	s.ring = s.ring[:0]
	add := func(x, y int) {
		if x >= 0 && x < s.w && y >= 0 && y < s.h {
			s.ring = append(s.ring, y*s.w+x)
		} else {
			s.ring = append(s.ring, -1)
		}
	}
	if r >= l && bot >= t {
		for x := l; x <= r; x++ {
			add(x, t)
		}
		for y := t + 1; y <= bot; y++ {
			add(r, y)
		}
		if bot > t {
			for x := r - 1; x >= l; x-- {
				add(x, bot)
			}
		}
		if r > l {
			for y := bot - 1; y > t; y-- {
				add(l, y)
			}
		}
	}

	n := len(s.ring)
	switch {
	case n == 0:
		// Nothing to walk on; keep the offsets for when a ring returns.
	case s.perim == 0:
		s.perim = n
		s.populate()
	case n != s.perim:
		for _, g := range s.groups {
			g.lead = g.lead * n / s.perim
		}
		s.perim = n
	}
}

// populate lays groups out around the ring with random gaps.
func (s *Scene) populate() {
	s.groups = s.groups[:0]
	pos := s.rng.IntN(s.perim)
	start := pos
	for {
		size := minGroup + s.rng.IntN(groupSpread)
		g := &group{behind: make([]int, size)}
		for i := 1; i < size; i++ {
			g.behind[i] = g.behind[i-1] + minStride + s.rng.IntN(strideSpan)
		}
		span := g.behind[size-1]
		gap := minGap + s.rng.IntN(gapSpread)
		if pos-start+span+gap > s.perim && len(s.groups) > 0 {
			break
		}
		g.lead = mod(pos+span, s.perim)
		s.groups = append(s.groups, g)
		pos += span + gap
		if pos-start >= s.perim {
			break
		}
	}
}

// draw clears the previous ants from grid and places the current ones.
func (s *Scene) draw() {
	s.dirty = false
	for _, i := range s.drawn {
		s.grid[i] = 0
	}
	s.drawn = s.drawn[:0]
	if len(s.ring) == 0 || len(s.ring) != s.perim {
		return
	}
	for _, g := range s.groups {
		for _, d := range g.behind {
			off := mod(g.lead-d, s.perim)
			if i := s.ring[off]; i >= 0 {
				s.grid[i] = legs[off%2]
				s.drawn = append(s.drawn, i)
			}
		}
	}
}

func mod(a, n int) int {
	a %= n
	if a < 0 {
		a += n
	}
	return a
}

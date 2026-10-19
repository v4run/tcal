package eventart

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const (
	kiteMin    = 2
	kiteMax    = 4
	kiteColW   = 18 // column width
	kiteMinH   = 12
	kiteMaxH   = 20
	kiteTailEv = 4 // ticks between tail wiggles
)

var kiteColors = [...]scene.Color{scene.Red, scene.Yellow, scene.Blue, scene.Green, scene.Magenta}

// sankranti is a few diamond kites bobbing in the upper half of a column
// on the left, each with a wiggling tail and a dotted string running down
// to one flyer at the column's bottom edge.
type sankranti struct {
	w, h   int
	tick   int
	n      int
	home   [kiteMax]int // home row of each kite
	phase  [kiteMax][2]int
	colors [kiteMax]scene.Color
	c      canvas
}

func init() {
	arts[events.Sankranti] = func(rng *rand.Rand) Art {
		s := &sankranti{n: kiteMin + rng.IntN(kiteMax-kiteMin+1)}
		for i, p := range rng.Perm(len(kiteColors))[:kiteMax] {
			s.colors[i] = kiteColors[p]
		}
		for i := range s.home {
			s.home[i] = rng.IntN(1 << 16) // scaled to the column in render
			s.phase[i] = [2]int{rng.IntN(40), rng.IntN(40)}
		}
		return s
	}
}

func (s *sankranti) Resize(w, h int) {
	s.w, s.h = w, h
	s.c.reset(s.Bounds())
	s.render()
}

func (s *sankranti) Tick() {
	s.tick++
	s.render()
}

func (s *sankranti) render() {
	if s.c.r.W == 0 {
		return
	}
	s.c.clear()
	w, h := s.c.r.W, s.c.r.H
	ax, ay := w/2, h-1 // where the flyer holds every string
	// Kites sit in the top half; bobbing moves them down up to two rows.
	span := max(1, h/2-4)
	type kite struct{ x, y int }
	var ks [kiteMax]kite
	for i := range s.n {
		x := 1 + i*(w-4)/max(1, s.n-1) + swayOffset(s.tick, s.phase[i][0])
		y := s.home[i]%span + 1 + swayOffset(s.tick, s.phase[i][1])
		ks[i] = kite{x, y}
		// The string, from the kite's lower tip to the flyer.
		top := y + 2
		for r := top; r <= ay; r++ {
			cx := x
			if ay > top {
				cx = x + (ax-x)*(r-top)/(ay-top)
			}
			s.c.set(cx, r, '.', scene.Default)
		}
	}
	for i, k := range ks[:s.n] {
		col := s.colors[i]
		s.c.text(k.x, k.y, `/\`, col)
		s.c.text(k.x, k.y+1, `\/`, col)
		tail := [2]rune{'(', ')'}
		if (s.tick/kiteTailEv+i)%2 == 1 {
			tail = [2]rune{')', '('}
		}
		s.c.set(k.x+1, k.y+2, tail[0], col)
		s.c.set(k.x+1, k.y+3, tail[1], col)
	}
}

func (s *sankranti) SetTime(time.Time, events.Occurrence) {}

// Bounds is a column down the left side, from near the top.
func (s *sankranti) Bounds() Region {
	return edgeRegion(s.w, s.h, kiteColW, columnHeight(s.h, kiteMinH, kiteMaxH), false, false)
}

func (s *sankranti) Cell(x, y int, b scene.Box) (rune, scene.Color) { return s.c.cell(x, y, b) }

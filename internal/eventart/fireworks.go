package eventart

import (
	"math"
	"math/rand/v2"

	"github.com/v4run/tcal/internal/scene"
)

const (
	burstGrow = 2 // ticks per radius step while a burst expands
	burstFade = 6 // ticks the full ring lingers while it fades
)

// fadeGlyphs are the full ring's glyphs as it fades, brightest first.
var fadeGlyphs = [...]rune{'+', '·', '.'}

type burst struct {
	x, y, r, age int // center, full radius, ticks alive
	color        scene.Color
}

func (b burst) life() int { return (b.r+1)*burstGrow + burstFade }

// fireworks is a set of bursts that expand and fade within a w×h area.
// Bursts are round on screen: two columns per radius step for each row.
type fireworks struct {
	w, h   int
	maxR   int
	colors []scene.Color
	rng    *rand.Rand
	bursts []burst
}

func newFireworks(rng *rand.Rand, maxR int, colors ...scene.Color) *fireworks {
	return &fireworks{rng: rng, maxR: maxR, colors: colors}
}

// resize sets the area and drops every burst.
func (f *fireworks) resize(w, h int) {
	f.w, f.h = max(w, 0), max(h, 0)
	f.bursts = f.bursts[:0]
}

// step ages every burst and, with probability p, launches a new one.
func (f *fireworks) step(p float64) {
	live := f.bursts[:0]
	for _, b := range f.bursts {
		b.age++
		if b.age < b.life() {
			live = append(live, b)
		}
	}
	f.bursts = live
	if f.w == 0 || f.h == 0 || len(f.bursts) >= max(1, f.w/10) || f.rng.Float64() >= p {
		return
	}
	r := min(f.maxR, (f.h-1)/2, (f.w-1)/4)
	if r < 0 {
		return
	}
	if r > 0 {
		r = 1 + f.rng.IntN(r) // a lone dot is too small to read as a burst
	}
	f.bursts = append(f.bursts, burst{
		x:     2*r + f.rng.IntN(f.w-4*r),
		y:     r + f.rng.IntN(f.h-2*r),
		r:     r,
		color: f.colors[f.rng.IntN(len(f.colors))],
	})
}

// draw paints the bursts into c with the area's top-left at (ox, oy).
func (f *fireworks) draw(c *canvas, ox, oy int) {
	for _, b := range f.bursts {
		grown := (b.r + 1) * burstGrow
		if b.age < grown {
			k := b.age / burstGrow
			if k == 0 {
				c.set(ox+b.x, oy+b.y, '*', b.color)
				continue
			}
			ring(c, ox+b.x, oy+b.y, k, '*', b.color)
			if k > 1 {
				ring(c, ox+b.x, oy+b.y, k-1, '.', b.color)
			}
			continue
		}
		g := fadeGlyphs[(b.age-grown)*len(fadeGlyphs)/burstFade]
		ring(c, ox+b.x, oy+b.y, b.r, g, b.color)
	}
}

// ring draws points around (x, y) at radius k rows (2k columns).
func ring(c *canvas, x, y, k int, g rune, col scene.Color) {
	n := 8
	if k > 1 {
		n = 12
	}
	for i := range n {
		a := 2 * math.Pi * float64(i) / float64(n)
		dx := int(math.Round(2 * float64(k) * math.Cos(a)))
		dy := int(math.Round(float64(k) * math.Sin(a)))
		c.set(x+dx, y+dy, g, col)
	}
}

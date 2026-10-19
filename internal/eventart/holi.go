package eventart

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const (
	holiMaxR    = 3    // largest burst radius, in rows
	holiGrow    = 2    // ticks per radius step while a burst blooms
	holiHold    = 8    // ticks at full size
	holiFade    = 12   // ticks to fade out
	holiDensity = 55   // percent of a burst's cells holding a dot at full strength
	holiRate    = 0.03 // launches per tick, so one every few seconds
	holiLive    = 2    // most bursts at once
	holiColW    = 20
	holiMinH    = 10
	holiMaxH    = 26
)

var (
	holiGlyphs = [...]rune{'·', '.', ':'}
	holiColors = [...]scene.Color{scene.Magenta, scene.Yellow, scene.Green, scene.Cyan, scene.Red}
)

type powder struct {
	x, y, r, age int
	seed         uint32
	color        scene.Color
}

func (p powder) life() int { return p.r*holiGrow + holiHold + holiFade }

// holi is soft bursts of colored powder that bloom and fade in a column
// on the right, one every few seconds.
type holi struct {
	w, h   int
	rng    *rand.Rand
	shot   bool // the first burst has gone up
	bursts []powder
	c      canvas
}

func init() {
	arts[events.Holi] = func(rng *rand.Rand) Art { return &holi{rng: rng} }
}

func (o *holi) Resize(w, h int) {
	o.w, o.h = w, h
	o.c.reset(o.Bounds())
	o.bursts, o.shot = o.bursts[:0], false
	o.render()
}

func (o *holi) Tick() {
	live := o.bursts[:0]
	for _, p := range o.bursts {
		if p.age++; p.age < p.life() {
			live = append(live, p)
		}
	}
	o.bursts = live
	p := holiRate
	if !o.shot {
		p = 1 // open with a burst so the art shows at once
	}
	if len(o.bursts) < holiLive && o.rng.Float64() < p {
		o.launch()
	}
	o.render()
}

func (o *holi) launch() {
	w, h := o.c.r.W, o.c.r.H
	r := min(holiMaxR, (h-1)/2, (w-1)/4)
	if r < 1 {
		return
	}
	o.shot = true
	o.bursts = append(o.bursts, powder{
		x:     2*r + o.rng.IntN(w-4*r),
		y:     r + o.rng.IntN(h-2*r),
		r:     r,
		seed:  o.rng.Uint32(),
		color: holiColors[o.rng.IntN(len(holiColors))],
	})
}

func (o *holi) render() {
	if o.c.r.W == 0 {
		return
	}
	o.c.clear()
	for _, p := range o.bursts {
		r, density := p.r, holiDensity
		if grown := p.r * holiGrow; p.age < grown {
			r = 1 + p.age/holiGrow
		} else if f := p.age - grown - holiHold; f > 0 {
			density = holiDensity * (holiFade - f) / holiFade
		}
		for dy := -r; dy <= r; dy++ {
			for dx := -2 * r; dx <= 2*r; dx++ {
				if dx*dx+4*dy*dy > 4*r*r {
					continue
				}
				// A fixed scatter per burst, so the dots stay put as it fades.
				k := scatter(p.seed, dx, dy)
				if int(k%100) < density || (dx == 0 && dy == 0 && density > 0) {
					o.c.set(p.x+dx, p.y+dy, holiGlyphs[k/100%uint32(len(holiGlyphs))], p.color)
				}
			}
		}
	}
}

// scatter hashes a burst's seed and a cell offset to a pseudo-random value.
func scatter(seed uint32, dx, dy int) uint32 {
	k := seed ^ uint32(dx+64)*0x9E3779B1 ^ uint32(dy+64)*0x85EBCA77
	k ^= k >> 15
	k *= 0x2C1B3C6D
	k ^= k >> 12
	return k
}

func (o *holi) SetTime(time.Time, events.Occurrence) {}

// Bounds is a column down the right side, from near the top.
func (o *holi) Bounds() Region {
	return edgeRegion(o.w, o.h, holiColW, columnHeight(o.h, holiMinH, holiMaxH), true, false)
}

func (o *holi) Cell(x, y int, b scene.Box) (rune, scene.Color) { return o.c.cell(x, y, b) }

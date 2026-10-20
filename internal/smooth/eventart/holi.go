package eventart

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const (
	holiMaxR   = 3.0                     // largest burst radius, in rows
	holiBloom  = 600 * time.Millisecond  // a burst puffs out for this long
	holiHold   = 900 * time.Millisecond  // then hangs in the air
	holiFade   = 1400 * time.Millisecond // then settles away
	holiSpread = 0.25                    // extra radius gained while it hangs and fades
	holiDots   = 70                      // powder grains per burst
	holiMinGap = 500 * time.Millisecond  // launches come every 0.5–1.5 s
	holiMaxGap = 1500 * time.Millisecond
	holiLive   = 4 // most bursts at once; the oldest makes way
	holiColW   = 20
	holiMinH   = 10
	holiMaxH   = 26
)

var holiCols = []scene.RGB{
	{R: 220, G: 95, B: 170},  // pink
	{R: 230, G: 200, B: 85},  // yellow
	{R: 105, G: 190, B: 115}, // green
	{R: 90, G: 160, B: 225},  // blue
	{R: 235, G: 140, B: 65},  // orange
	{R: 160, G: 110, B: 215}, // violet
}

type powder struct {
	x, y, r float64 // centre in cells, full radius in rows
	age     time.Duration
	seed    uint32
	col     scene.RGB
}

// holi is soft bursts of coloured powder that puff out, hang and fade in a
// column on the right, a new one every second or so.
type holi struct {
	w, h     int
	rng      *rand.Rand
	next     time.Duration // until the next launch
	launched int           // bursts launched so far
	bursts   []powder
	f        frame
}

func init() {
	arts[events.Holi] = func(rng *rand.Rand) Art { return &holi{rng: rng} }
}

func (o *holi) Resize(w, h int) {
	o.w, o.h = max(w, 0), max(h, 0)
	o.f.reset(o.Bounds())
	o.bursts, o.next = o.bursts[:0], 0 // open with a burst so the art shows at once
	o.render()
}

func (o *holi) Step(dt time.Duration) {
	dt = max(dt, 0)
	live := o.bursts[:0]
	for _, p := range o.bursts {
		if p.age += dt; p.age < holiBloom+holiHold+holiFade {
			live = append(live, p)
		}
	}
	o.bursts = live
	if o.next -= dt; o.next <= 0 && o.launch() {
		o.next = holiMinGap + time.Duration(o.rng.Float64()*float64(holiMaxGap-holiMinGap))
	}
	o.render()
}

// launch puffs a new burst somewhere it fits whole, or nothing on a tiny
// region.
func (o *holi) launch() bool {
	w, h := float64(o.f.r.W), float64(o.f.r.H)
	r := min(holiMaxR, (h-1)/2, (w-1)/4) / (1 + holiSpread)
	if r < 1 {
		return false
	}
	r *= 0.7 + 0.3*o.rng.Float64()
	R := r * (1 + holiSpread)
	if len(o.bursts) >= holiLive {
		o.bursts = append(o.bursts[:0], o.bursts[1:]...)
	}
	o.bursts = append(o.bursts, powder{
		x:    2*R + o.rng.Float64()*(w-4*R),
		y:    R + o.rng.Float64()*(h-2*R),
		r:    r,
		seed: o.rng.Uint32(),
		col:  holiCols[o.rng.IntN(len(holiCols))],
	})
	o.launched++
	return true
}

func (o *holi) render() {
	c := &o.f.c
	c.Clear()
	for _, p := range o.bursts {
		k := 1.0
		if p.age < holiBloom {
			e := 1 - float64(p.age)/float64(holiBloom)
			k = 1 - e*e
		}
		after := float64(max(p.age-holiBloom, 0)) / float64(holiHold+holiFade)
		R := p.r * (k + holiSpread*after)
		fade := float64(max(p.age-holiBloom-holiHold, 0)) / float64(holiFade)
		for i := range holiDots {
			h := hash(p.seed, i, 0)
			if fade >= unit(hash(p.seed, i, 1)) {
				continue // this grain has settled
			}
			a := 2 * math.Pi * unit(h)
			// Denser in the middle: a soft cloud, not a ring.
			rf := unit(hash(p.seed, i, 2))
			c.Dot(p.x+2*R*rf*math.Cos(a), p.y+R*rf*math.Sin(a), p.col)
		}
	}
}

func (o *holi) SetTime(time.Time, events.Occurrence) {}

// Bounds is a column down the right side, from near the top.
func (o *holi) Bounds() Region {
	return edgeRegion(o.w, o.h, holiColW, columnHeight(o.h, holiMinH, holiMaxH), true, false)
}

func (o *holi) Cell(x, y int, b scene.Box) (rune, scene.RGB) { return o.f.cell(x, y, b) }

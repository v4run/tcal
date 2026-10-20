package eventart

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/dots"
	"github.com/v4run/tcal/internal/scene"
)

const (
	fwGrow  = 500 * time.Millisecond // a burst expands for this long
	fwFade  = 800 * time.Millisecond // then droops and thins out
	fwDroop = 1.2                    // rows per second², pulling sparks down as they fade
	fwFlash = 100 * time.Millisecond // the centre dot shows this long at launch
	fwInner = 0.6                    // the inner trail ring, as a fraction of the radius
)

// golden spreads spark fade-out order evenly: spark i vanishes once the
// fade passes 1 - frac(i·golden), so spark 0 is always the last to go.
const golden = 0.6180339887

type burst struct {
	x, y, r float64 // centre in cells and full radius in rows (2r columns)
	age     time.Duration
	col     scene.RGB
	spin    float64 // angle of spark 0, radians
}

// fireworks is a set of bursts that expand, droop and fade within a w×h
// area. Bursts are round on screen: two columns per row of radius.
type fireworks struct {
	w, h       int
	maxR       float64
	grow, fade time.Duration
	cols       []scene.RGB
	rng        *rand.Rand
	bursts     []burst
}

func newFireworks(rng *rand.Rand, maxR float64, cols ...scene.RGB) *fireworks {
	return &fireworks{rng: rng, maxR: maxR, grow: fwGrow, fade: fwFade, cols: cols}
}

// resize sets the area and drops every burst.
func (f *fireworks) resize(w, h int) {
	f.w, f.h = max(w, 0), max(h, 0)
	f.bursts = f.bursts[:0]
}

// ease is the fraction of the full radius a burst of this age has reached:
// fast at first, settling at 1 (ease-out).
func (f *fireworks) ease(age time.Duration) float64 {
	if f.grow <= 0 || age >= f.grow {
		return 1
	}
	k := 1 - float64(age)/float64(f.grow)
	return 1 - k*k
}

// droop is how far, in rows, a burst's sparks have fallen at this age.
func (f *fireworks) droop(age time.Duration) float64 {
	s := max(age-f.grow, 0).Seconds()
	return fwDroop * s * s
}

// step ages every burst by dt and launches a new one with probability p.
func (f *fireworks) step(dt time.Duration, p float64) {
	live := f.bursts[:0]
	for _, b := range f.bursts {
		b.age += dt
		if b.age < f.grow+f.fade {
			live = append(live, b)
		}
	}
	f.bursts = live
	if len(f.bursts) >= max(1, f.w/10) || f.rng.Float64() >= p {
		return
	}
	f.launch()
}

// launch adds a burst wherever it fits whole, droop and all, or nothing on
// a tiny area.
func (f *fireworks) launch() bool {
	// The sparks fall this far by the end of the fade; a hair less keeps the
	// lowest one off the area's bottom edge.
	drop := f.droop(f.grow+f.fade) + 1e-6
	r := min(f.maxR, (float64(f.h)-drop)/2, float64(f.w)/4)
	if r < 0.5 {
		return false
	}
	r *= 0.6 + 0.4*f.rng.Float64()
	f.bursts = append(f.bursts, burst{
		x:    2*r + f.rng.Float64()*(float64(f.w)-4*r),
		y:    r + f.rng.Float64()*(float64(f.h)-2*r-drop),
		r:    r,
		col:  f.cols[f.rng.IntN(len(f.cols))],
		spin: f.rng.Float64() * 2 * math.Pi,
	})
	return true
}

// draw paints the bursts into c with the area's top-left at (ox, oy).
func (f *fireworks) draw(c *dots.Canvas, ox, oy float64) {
	for _, b := range f.bursts {
		x, y := ox+b.x, oy+b.y
		if b.age < fwFlash {
			c.Dot(x, y, b.col)
		}
		pr := b.r * f.ease(b.age)
		fall := f.droop(b.age)
		fade := 0.0
		if f.fade > 0 && b.age > f.grow {
			fade = float64(b.age-f.grow) / float64(f.fade)
		}
		n := 12
		if b.r > 1.5 {
			n = 16
		}
		for i := range n {
			if _, keep := math.Modf(float64(i) * golden); fade >= 1-keep {
				continue
			}
			a := b.spin + 2*math.Pi*float64(i)/float64(n)
			dx, dy := 2*math.Cos(a), math.Sin(a)
			c.Dot(x+dx*pr, y+dy*pr+fall, b.col)
			// A short trail behind each spark while it flies out.
			if b.age < f.grow && pr > 0.5 {
				c.Dot(x+dx*pr*fwInner, y+dy*pr*fwInner+fall, b.col)
			}
		}
	}
}

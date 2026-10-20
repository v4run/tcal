package eventart

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const (
	nyMaxH     = 6   // band height on tall screens
	nyMinH     = 3   // below this the band is too thin for bursts
	nyMinW     = 20  // narrower screens get no fireworks
	nyMargin   = 1   // columns kept clear at each side
	nyMaxR     = 2.0 // largest burst radius, in rows
	nyPeakRate = 4.0 // launches per second within nyPeak of midnight
	nyBaseRate = 0.4 // launches per second from nyCalm away and beyond
	nyPeak     = 30 * time.Minute
	nyCalm     = 2 * time.Hour
)

var nyCols = []scene.RGB{
	{R: 225, G: 90, B: 80},   // red
	{R: 235, G: 195, B: 95},  // gold
	{R: 205, G: 110, B: 200}, // magenta
	{R: 100, G: 195, B: 215}, // cyan
	{R: 120, G: 200, B: 125}, // green
}

// newYear is fireworks in a band across the top, busiest around midnight.
type newYear struct {
	w, h int
	rate float64 // launches per second
	shot bool    // the first burst has gone up
	fw   *fireworks
	f    frame
}

func init() {
	arts[events.NewYear] = func(rng *rand.Rand) Art {
		return &newYear{rate: nyPeakRate, fw: newFireworks(rng, nyMaxR, nyCols...)}
	}
}

func (n *newYear) Resize(w, h int) {
	n.w, n.h = max(w, 0), max(h, 0)
	n.shot = false // the bursts are gone, so open with a fresh one
	n.f.reset(n.Bounds())
	n.fw.resize(n.f.r.W, n.f.r.H)
	n.render()
}

func (n *newYear) Step(dt time.Duration) {
	dt = max(dt, 0)
	p := chance(n.rate, dt)
	if !n.shot && n.f.r.W > 0 {
		p, n.shot = 1, true // open with a burst so the art shows at once
	}
	n.fw.step(dt, p)
	n.render()
}

func (n *newYear) render() {
	n.f.c.Clear()
	n.fw.draw(&n.f.c, 0, 0)
}

// SetTime sets the launch rate from now's distance to the nearest midnight:
// peak within half an hour, easing down to a trickle two hours out.
func (n *newYear) SetTime(now time.Time, _ events.Occurrence) {
	h, m, s := now.Clock()
	d := time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s)*time.Second
	d = min(d, 24*time.Hour-d)
	switch {
	case d <= nyPeak:
		n.rate = nyPeakRate
	case d >= nyCalm:
		n.rate = nyBaseRate
	default:
		f := float64(d-nyPeak) / float64(nyCalm-nyPeak)
		n.rate = nyPeakRate + f*(nyBaseRate-nyPeakRate)
	}
	n.render()
}

// Bounds is a band across the top edge.
func (n *newYear) Bounds() Region {
	bh := min(nyMaxH, n.h/6)
	if n.w < nyMinW || bh < nyMinH {
		return Region{}
	}
	return Region{X0: nyMargin, Y0: 0, W: n.w - 2*nyMargin, H: bh}
}

func (n *newYear) Cell(x, y int, b scene.Box) (rune, scene.RGB) { return n.f.cell(x, y, b) }

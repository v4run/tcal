package eventart

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const (
	nyMaxH     = 6    // band height on tall screens
	nyMinH     = 3    // below this the band is too thin for bursts
	nyMinW     = 20   // narrower screens get no fireworks
	nyMargin   = 1    // columns kept clear at each side
	nyPeakRate = 0.4  // launches per tick within nyPeak of midnight
	nyBaseRate = 0.04 // launches per tick from nyCalm away and beyond
	nyPeak     = 30 * time.Minute
	nyCalm     = 2 * time.Hour
)

// newYear is fireworks in a band across the top, busiest around midnight.
type newYear struct {
	w, h int
	rate float64
	shot bool // the first burst has gone up
	fw   *fireworks
	c    canvas
}

func init() {
	arts[events.NewYear] = func(rng *rand.Rand) Art {
		return &newYear{
			rate: nyPeakRate,
			fw:   newFireworks(rng, 2, scene.Red, scene.Yellow, scene.Magenta, scene.Cyan, scene.Green),
		}
	}
}

func (n *newYear) Resize(w, h int) {
	n.w, n.h = w, h
	n.shot = false // the bursts are gone, so open with a fresh one
	bd := n.Bounds()
	n.c.reset(bd)
	n.fw.resize(bd.W, bd.H)
	n.fw.maxR = min(2, (bd.H-1)/2)
}

func (n *newYear) Tick() {
	p := n.rate
	if !n.shot {
		p, n.shot = 1, true // open with a burst so the art shows at once
	}
	n.fw.step(p)
	n.c.clear()
	n.fw.draw(&n.c, 0, 0)
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
}

func (n *newYear) Bounds() Region {
	bh := min(nyMaxH, n.h/6)
	if n.w < nyMinW || bh < nyMinH {
		return Region{}
	}
	return Region{X0: nyMargin, Y0: 0, W: n.w - 2*nyMargin, H: bh}
}

func (n *newYear) Cell(x, y int, b scene.Box) (rune, scene.Color) { return n.c.cell(x, y, b) }

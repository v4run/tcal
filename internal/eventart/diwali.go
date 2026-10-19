package eventart

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const (
	diyaLamp      = `\_/` // the clay lamp; its flame sits above the middle
	diyaW         = len(diyaLamp)
	diyaSpacing   = 7 // columns from one diya to the next
	diyaMin       = 5
	diyaMax       = 9
	diyaMaxH      = 6    // lamps plus room for bursts above them
	diyaFlicker   = 0.35 // chance per tick that a flame changes shape
	diyaBurstRate = 0.05
	diyaMargin    = 1 // columns kept clear at each side
)

var flameGlyphs = [...]rune{'^', '(', ')'}

// diwali is a row of diyas along the bottom edge with flickering flames,
// and the odd small burst above them.
type diwali struct {
	w, h   int
	rng    *rand.Rand
	n      int // diyas
	flames [diyaMax]rune
	fw     *fireworks
	c      canvas
}

func init() {
	arts[events.Diwali] = func(rng *rand.Rand) Art {
		d := &diwali{rng: rng, fw: newFireworks(rng, 1, scene.Yellow, scene.Red)}
		for i := range d.flames {
			d.flames[i] = flameGlyphs[rng.IntN(len(flameGlyphs))]
		}
		return d
	}
}

func (d *diwali) Resize(w, h int) {
	d.w, d.h = w, h
	d.n = min(diyaMax, max(diyaMin, w/14))
	bd := d.Bounds()
	d.c.reset(bd)
	d.fw.resize(bd.W, bd.H-2)
	d.render()
}

func (d *diwali) Tick() {
	for i := range d.n {
		if d.rng.Float64() < diyaFlicker {
			d.flames[i] = flameGlyphs[d.rng.IntN(len(flameGlyphs))]
		}
	}
	d.fw.step(diyaBurstRate)
	d.render()
}

func (d *diwali) render() {
	if d.c.r.W == 0 {
		return
	}
	d.c.clear()
	d.fw.draw(&d.c, 0, 0)
	base := d.c.r.H - 1
	for i := range d.n {
		x := i * diyaSpacing
		d.c.set(x+diyaW/2, base-1, d.flames[i], scene.Yellow)
		d.c.text(x, base, diyaLamp, scene.Red)
	}
}

func (d *diwali) SetTime(time.Time, events.Occurrence) {}

// Bounds is a strip centered on the bottom edge: the lamps on the last two
// rows and burst room above, shrinking on short screens.
func (d *diwali) Bounds() Region {
	bw := (d.n-1)*diyaSpacing + diyaW
	bh := min(diyaMaxH, d.h/6)
	if bh < 2 || d.w < bw+2*diyaMargin {
		return Region{}
	}
	return Region{X0: (d.w - bw) / 2, Y0: d.h - bh, W: bw, H: bh}
}

func (d *diwali) Cell(x, y int, b scene.Box) (rune, scene.Color) { return d.c.cell(x, y, b) }

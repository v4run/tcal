package eventart

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/dots"
	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const (
	diyaW         = 3 // cells; the flame sits over the middle one
	diyaSpacing   = 7 // columns from one diya to the next
	diyaMin       = 5
	diyaMax       = 9
	diyaMaxH      = 6   // lamps plus room for bursts above them
	diyaFlicker   = 3.5 // flame shape changes per second, each
	diyaGlow      = 9.0 // radians per second of the flame's brightness wobble
	diyaBurstRate = 0.5 // bursts per second
	diyaMargin    = 1   // columns kept clear at each side
)

var (
	diyaCol    = scene.RGB{R: 190, G: 105, B: 65} // terracotta
	flameCol   = scene.RGB{R: 240, G: 185, B: 80} // warm gold
	flameTip   = scene.RGB{R: 235, G: 120, B: 60} // orange, as a flame burns low
	diwaliCols = []scene.RGB{{R: 235, G: 195, B: 95}, {R: 220, G: 95, B: 80}, {R: 230, G: 150, B: 80}}

	// diyaLamp is a shallow clay bowl, 3 cells wide, at the top of its cell
	// row so it sits right under the flame.
	diyaLamp = dots.ParseSprite(
		"#....#",
		"######",
		".####.",
		"..##..",
	)
	// flames are the shapes a flame flickers between, one cell each, resting
	// on the bottom of the cell above the lamp.
	flames = [...]dots.Sprite{
		dots.ParseSprite(".#", "##", "##", "##"),
		dots.ParseSprite("#.", "##", "##", "##"),
		dots.ParseSprite("..", "#.", "##", "##"),
		dots.ParseSprite("..", ".#", "##", "##"),
	}
)

// diwali is a row of diyas along the bottom edge with flickering flames,
// and the odd small burst above them.
type diwali struct {
	w, h  int
	rng   *rand.Rand
	t     float64 // seconds
	n     int     // diyas
	shape [diyaMax]int
	glow  [diyaMax]float64 // brightness wobble phase
	fw    *fireworks
	f     frame
}

func init() {
	arts[events.Diwali] = func(rng *rand.Rand) Art {
		d := &diwali{rng: rng, fw: newFireworks(rng, 1, diwaliCols...)}
		for i := range d.shape {
			d.shape[i] = rng.IntN(len(flames))
			d.glow[i] = rng.Float64() * 2 * math.Pi
		}
		return d
	}
}

func (d *diwali) Resize(w, h int) {
	d.w, d.h = max(w, 0), max(h, 0)
	d.n = min(diyaMax, max(diyaMin, d.w/14))
	d.f.reset(d.Bounds())
	d.fw.resize(d.f.r.W, max(d.f.r.H-2, 0))
	d.render()
}

func (d *diwali) Step(dt time.Duration) {
	dt = max(dt, 0)
	d.t += dt.Seconds()
	p := chance(diyaFlicker, dt)
	for i := range d.n {
		if d.rng.Float64() < p {
			d.shape[i] = (d.shape[i] + 1 + d.rng.IntN(len(flames)-1)) % len(flames)
		}
	}
	d.fw.step(dt, chance(diyaBurstRate, dt))
	d.render()
}

func (d *diwali) render() {
	c := &d.f.c
	c.Clear()
	if d.f.r.W == 0 {
		return
	}
	d.fw.draw(c, 0, 0)
	base := float64(d.f.r.H - 1)
	for i := range d.n {
		x := float64(i * diyaSpacing)
		k := 0.5 + 0.5*math.Sin(diyaGlow*d.t+d.glow[i])
		c.DrawSprite(flames[d.shape[i]], x+1, base-1, lerp(flameTip, flameCol, 0.4+0.6*k))
		c.DrawSprite(diyaLamp, x, base, diyaCol)
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

func (d *diwali) Cell(x, y int, b scene.Box) (rune, scene.RGB) { return d.f.cell(x, y, b) }

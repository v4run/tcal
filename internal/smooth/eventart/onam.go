package eventart

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const (
	// onamRings is the number of rings in the full pookalam. It is at least
	// the longest Onam (11 days), so every day before the last shows fewer
	// rings.
	onamRings = 11
	onamW     = 21 // cells; 42×44 dots, round on screen
	onamH     = 11
	onamDW    = 2 * onamW
	onamDH    = 4 * onamH
)

// onamStyle is how each ring is laid: solid flowers, or petals that scallop
// outward. ringCols are the flowers: marigold, yellow chrysanthemum,
// crimson, jasmine white, leaf green and violet, as a pookalam is laid.
var (
	onamPetals = [onamRings]int{0, 0, 6, 0, 8, 0, 12, 0, 16, 0, 20} // 0 = solid ring
	ringCols   = [onamRings]scene.RGB{
		{R: 230, G: 195, B: 80},  // yellow heart
		{R: 225, G: 140, B: 60},  // marigold
		{R: 220, G: 215, B: 200}, // jasmine
		{R: 190, G: 70, B: 75},   // crimson
		{R: 230, G: 195, B: 80},  // yellow
		{R: 150, G: 105, B: 170}, // violet
		{R: 225, G: 140, B: 60},  // marigold
		{R: 220, G: 215, B: 200}, // jasmine
		{R: 190, G: 70, B: 75},   // crimson
		{R: 230, G: 195, B: 80},  // yellow
		{R: 95, G: 150, B: 85},   // leaf green, the outer border
	}
)

// pookalamRing is the ring of each dot of the full pookalam, counted from
// the centre out, or -1 where no flower lies.
var pookalamRing = pookalam()

func pookalam() (out [onamDH][onamDW]int8) {
	const cx, cy = onamDW / 2.0, onamDH / 2.0
	// Dots are square on screen, so the radius is the same both ways; the
	// rings split it evenly.
	const r = onamDW / 2.0
	const band = r / onamRings
	for y := range out {
		for x := range out[y] {
			out[y][x] = -1
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			d := math.Hypot(dx, dy)
			k := int(d / band)
			if k >= onamRings {
				continue
			}
			if p := onamPetals[k]; p > 0 {
				// A petal ring: each dot is kept if it lies within its petal,
				// whose tip reaches the ring's outer edge.
				u := d/band - float64(k)
				a := math.Atan2(dy, dx)
				if u > 0.35+0.65*math.Abs(math.Cos(float64(p)/2*a)) {
					continue
				}
			}
			out[y][x] = int8(k)
		}
	}
	return out
}

// onam is a round pookalam in the bottom-left corner that gains rings day
// by day, from the centre out, and is complete on the last day.
type onam struct {
	w, h  int
	rings int
	f     frame
}

func init() {
	arts[events.Onam] = func(*rand.Rand) Art { return &onam{rings: onamRings} }
}

func (o *onam) Resize(w, h int) {
	o.w, o.h = max(w, 0), max(h, 0)
	o.f.reset(o.Bounds())
	o.render()
}

// Step does nothing: the pookalam is laid once a day and lies still.
func (o *onam) Step(time.Duration) {}

// SetTime fills ceil((day+1)/days × onamRings) rings.
func (o *onam) SetTime(now time.Time, occ events.Occurrence) {
	o.rings = onamRings
	if day, days := occ.Day(now); days > 0 {
		day = min(max(day, 0), days-1)
		o.rings = ((day+1)*onamRings + days - 1) / days
	}
	o.render()
}

// render lays the rings from the centre out, so a cell shared by two rings
// takes the outer one's colour.
func (o *onam) render() {
	c := &o.f.c
	c.Clear()
	if o.f.r.W == 0 {
		return
	}
	for k := range o.rings {
		for y, row := range pookalamRing {
			for x, rk := range row {
				if int(rk) == k {
					c.Dot(float64(x)/2, float64(y)/4, ringCols[k])
				}
			}
		}
	}
}

// Bounds is the bottom-left corner.
func (o *onam) Bounds() Region { return edgeRegion(o.w, o.h, onamW, onamH, false, true) }

func (o *onam) Cell(x, y int, b scene.Box) (rune, scene.RGB) { return o.f.cell(x, y, b) }

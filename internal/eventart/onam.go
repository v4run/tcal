package eventart

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const (
	// onamRings is the number of rings in the full pookalam. Rings are half
	// a row apart, so the flower is 2*onamR+1 rows by 4*onamR+1 columns. It
	// is at least the longest Onam (11 days), so every day before the last
	// shows fewer rings.
	onamRings = 11
	onamR     = (onamRings - 1) / 2
	onamW     = 4*onamR + 1
	onamH     = 2*onamR + 1
)

// pookalamRing is the ring index of each cell of the full pookalam, counted
// from the center out, or -1 outside it.
var pookalamRing = pookalam()

func pookalam() [onamH][onamW]int8 {
	var out [onamH][onamW]int8
	for y := range out {
		for x := range out[y] {
			// Two columns per row keeps the flower round on screen; the
			// distance is in half rows.
			dx, dy := float64(x-2*onamR), float64(y-onamR)
			k := int(math.Round(math.Sqrt(dx*dx + 4*dy*dy)))
			if k >= onamRings {
				k = -1
			}
			out[y][x] = int8(k)
		}
	}
	return out
}

var (
	ringColors = [...]scene.Color{scene.Yellow, scene.Red, scene.Magenta, scene.Green}
	ringGlyphs = [...]rune{'*', 'o'}
)

// onam is a round pookalam in the bottom-left corner that gains rings day
// by day, from the center out, and is complete on the last day.
type onam struct {
	w, h  int
	rings int
	c     canvas
}

func init() {
	arts[events.Onam] = func(*rand.Rand) Art { return &onam{rings: onamRings} }
}

func (o *onam) Resize(w, h int) {
	o.w, o.h = w, h
	o.c.reset(o.Bounds())
	o.render()
}

func (o *onam) Tick() {}

// SetTime fills ceil((day+1)/days × onamRings) rings.
func (o *onam) SetTime(now time.Time, occ events.Occurrence) {
	o.rings = onamRings
	if day, days := occ.Day(now); days > 0 {
		day = min(max(day, 0), days-1)
		o.rings = ((day+1)*onamRings + days - 1) / days
	}
	o.render()
}

func (o *onam) render() {
	if o.c.r.W == 0 {
		return
	}
	o.c.clear()
	for y, row := range pookalamRing {
		for x, k := range row {
			if k >= 0 && int(k) < o.rings {
				o.c.set(x, y, ringGlyphs[int(k)%len(ringGlyphs)], ringColors[int(k)%len(ringColors)])
			}
		}
	}
}

// Bounds is the bottom-left corner.
func (o *onam) Bounds() Region { return edgeRegion(o.w, o.h, onamW, onamH, false, true) }

func (o *onam) Cell(x, y int, b scene.Box) (rune, scene.Color) { return o.c.cell(x, y, b) }

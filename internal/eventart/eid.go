package eventart

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const (
	moonW = 8 // braille cells; 2×4 dots each, so 16×16 dots
	moonH = 4

	fanousW      = 3
	fanousMaxStr = 3
	fanousSlot   = fanousW + 3
	eidH         = fanousMaxStr + len(fanousRows)
	eidMargin    = 2 // columns kept clear of the right edge
	eidGap       = 2 // columns between the last lantern and the moon
)

// fanousRows is a small hanging lantern.
var fanousRows = [...]string{
	`/^\`,
	`|o|`,
	`\_/`,
}

// moonRows is a crescent opening to the right, in braille.
var moonRows = crescent()

// crescent rasterizes a disc minus an offset disc onto braille dots.
func crescent() [moonH][moonW]rune {
	const d = 2 * moonW // dots per side
	bits := [4][2]rune{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}
	var out [moonH][moonW]rune
	c := float64(d-1) / 2
	ro, ri := float64(d)/2, float64(d)/2*0.8
	ix, iy := c+ro*0.5, c
	for y := range d {
		for x := range d {
			fx, fy := float64(x), float64(y)
			if math.Hypot(fx-c, fy-c) <= ro && math.Hypot(fx-ix, fy-iy) > ri {
				out[y/4][x/2] |= bits[y%4][x%2]
			}
		}
	}
	for y := range out {
		for x := range out[y] {
			if out[y][x] != 0 {
				out[y][x] += 0x2800
			}
		}
	}
	return out
}

// eid is a white crescent in the top-right corner with a few yellow
// lanterns hanging beside it.
type eid struct {
	w, h  int
	tick  int
	n     int
	str   [3]int
	phase [3]int
	c     canvas
}

func init() {
	arts[events.Eid] = func(rng *rand.Rand) Art {
		e := &eid{}
		for i := range e.str {
			e.str[i] = 1 + rng.IntN(fanousMaxStr)
			e.phase[i] = rng.IntN(40)
		}
		return e
	}
}

func (e *eid) Resize(w, h int) {
	e.w, e.h = w, h
	e.n = 2
	if w >= 100 {
		e.n = 3
	}
	e.c.reset(e.Bounds())
	e.render()
}

func (e *eid) Tick() {
	e.tick++
	e.render()
}

func (e *eid) render() {
	if e.c.r.W == 0 {
		return
	}
	e.c.clear()
	for i := range e.n {
		ax := 1 + i*fanousSlot + fanousSlot/2
		off := swayOffset(e.tick, e.phase[i])
		hangString(&e.c, ax, e.str[i], off, scene.Yellow)
		for y, row := range fanousRows {
			e.c.text(ax+off-fanousW/2, e.str[i]+y, row, scene.Yellow)
		}
	}
	mx := e.c.r.W - moonW
	for y, row := range moonRows {
		for x, r := range row {
			if r != 0 {
				e.c.set(mx+x, 1+y, r, scene.White)
			}
		}
	}
}

func (e *eid) SetTime(time.Time, events.Occurrence) {}

// Bounds is the top-right corner: the lanterns, then the moon.
func (e *eid) Bounds() Region {
	bw := 1 + e.n*fanousSlot + eidGap + moonW
	if e.w < bw+eidMargin+1 || e.h < 3*eidH {
		return Region{}
	}
	return Region{X0: e.w - eidMargin - bw, Y0: 0, W: bw, H: eidH}
}

func (e *eid) Cell(x, y int, b scene.Box) (rune, scene.Color) { return e.c.cell(x, y, b) }

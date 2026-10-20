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
	moonW = 8 // cells; 16×16 dots
	moonH = 4

	fanousW      = 3 // cells
	fanousRows   = 3
	fanousMaxStr = 3
	fanousSlot   = fanousW + 3
	eidMaxLamps  = 3
	eidH         = fanousMaxStr + fanousRows
	eidMargin    = 2 // columns kept clear of the right edge
	eidGap       = 2 // columns between the last lantern and the moon
)

var (
	moonCol   = scene.RGB{R: 235, G: 232, B: 210} // pale moonlight
	fanousCol = scene.RGB{R: 225, G: 180, B: 90}  // brass

	// fanous is a small hanging lantern: a ring, a domed roof, a lit window
	// and a pointed base, 3 cells wide.
	fanous = dots.ParseSprite(
		"..##..",
		".#..#.",
		"######",
		"#....#",
		"#.##.#",
		"#.##.#",
		"#....#",
		"######",
		".#..#.",
		"..##..",
	)
	moon = crescent()
)

// crescent is a disc minus an offset disc, opening to the right.
func crescent() dots.Sprite {
	const d = 2 * moonW
	c := float64(d-1) / 2
	ro, ri := float64(d)/2, float64(d)/2*0.8
	ix, iy := c+ro*0.5, c
	rows := make([]string, d)
	for y := range d {
		row := make([]byte, d)
		for x := range d {
			row[x] = '.'
			fx, fy := float64(x), float64(y)
			if math.Hypot(fx-c, fy-c) <= ro && math.Hypot(fx-ix, fy-iy) > ri {
				row[x] = '#'
			}
		}
		rows[y] = string(row)
	}
	return dots.ParseSprite(rows...)
}

// eid is a pale crescent in the top-right corner with a few brass lanterns
// swinging beside it.
type eid struct {
	w, h  int
	t     float64 // seconds
	n     int
	str   [eidMaxLamps]int
	phase [eidMaxLamps]float64
	f     frame
}

func init() {
	arts[events.Eid] = func(rng *rand.Rand) Art {
		e := &eid{}
		for i := range e.str {
			e.str[i] = 1 + rng.IntN(fanousMaxStr)
			e.phase[i] = rng.Float64() * 2 * math.Pi
		}
		return e
	}
}

// fanousAnchor is the region-local column lantern i hangs from.
func fanousAnchor(i int) float64 {
	return 1 + float64(i*fanousSlot) + float64(fanousSlot)/2
}

func (e *eid) lamps() int {
	if e.w >= 100 {
		return 3
	}
	return 2
}

func (e *eid) Resize(w, h int) {
	e.w, e.h = max(w, 0), max(h, 0)
	e.n = e.lamps()
	e.f.reset(e.Bounds())
	e.render()
}

func (e *eid) Step(dt time.Duration) {
	e.t += max(dt, 0).Seconds()
	e.render()
}

func (e *eid) render() {
	c := &e.f.c
	c.Clear()
	if e.f.r.W == 0 {
		return
	}
	for i := range e.n {
		ax := fanousAnchor(i)
		lx := ax + sway(e.t, e.phase[i])
		top := float64(e.str[i])
		line(c, ax, 0, lx, top, fanousCol)
		c.DrawSprite(fanous, lx-float64(fanousW)/2, top, fanousCol)
	}
	c.DrawSprite(moon, float64(e.f.r.W-moonW), 1, moonCol)
}

func (e *eid) SetTime(time.Time, events.Occurrence) {}

// Bounds is the top-right corner: the lanterns, then the moon.
func (e *eid) Bounds() Region {
	bw := 1 + e.lamps()*fanousSlot + eidGap + moonW
	if e.w < bw+eidMargin+1 || e.h < 3*eidH {
		return Region{}
	}
	return Region{X0: e.w - eidMargin - bw, Y0: 0, W: bw, H: eidH}
}

func (e *eid) Cell(x, y int, b scene.Box) (rune, scene.RGB) { return e.f.cell(x, y, b) }

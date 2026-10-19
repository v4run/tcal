package eventart

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

// lanternRows is a red lantern; the last row is its yellow tassel.
var lanternRows = [...]string{
	` .-. `,
	`(|||)`,
	` '-' `,
	`  |  `,
}

const (
	lanternW      = 5
	lanternMaxStr = 2 // longest string above a lantern
	lanternSlot   = lanternW + 3
	lanternMin    = 3
	lanternMax    = 5
	lnyH          = lanternMaxStr + len(lanternRows)
	lnyMargin     = 2 // columns kept clear at each side
)

// lunarNewYear is a row of red lanterns hanging from the top edge,
// each swaying a column back and forth on its string.
type lunarNewYear struct {
	w, h  int
	tick  int
	n     int
	str   [lanternMax]int // string length per lantern, 1..lanternMaxStr
	phase [lanternMax]int
	c     canvas
}

func init() {
	arts[events.LunarNewYear] = func(rng *rand.Rand) Art {
		l := &lunarNewYear{}
		for i := range lanternMax {
			l.str[i] = 1 + rng.IntN(lanternMaxStr)
			l.phase[i] = rng.IntN(40)
		}
		return l
	}
}

func (l *lunarNewYear) Resize(w, h int) {
	l.w, l.h = w, h
	l.n = min(lanternMax, max(lanternMin, w/30))
	l.c.reset(l.Bounds())
	l.render()
}

func (l *lunarNewYear) Tick() {
	l.tick++
	l.render()
}

func (l *lunarNewYear) render() {
	if l.c.r.W == 0 {
		return
	}
	l.c.clear()
	for i := range l.n {
		// Centered in its share of the band, so lanterns spread evenly.
		ax := (2*i + 1) * l.c.r.W / (2 * l.n)
		off := swayOffset(l.tick, l.phase[i])
		hangString(&l.c, ax, l.str[i], off, scene.Yellow)
		x0 := ax + off - lanternW/2
		for y, row := range lanternRows {
			col := scene.Red
			if y == len(lanternRows)-1 {
				col = scene.Yellow
			}
			l.c.text(x0, l.str[i]+y, row, col)
		}
	}
}

func (l *lunarNewYear) SetTime(time.Time, events.Occurrence) {}

// Bounds is a band across the top edge.
func (l *lunarNewYear) Bounds() Region {
	bw := l.w - 2*lnyMargin
	if bw < l.n*lanternSlot || l.h < 3*lnyH {
		return Region{}
	}
	return Region{X0: lnyMargin, Y0: 0, W: bw, H: lnyH}
}

func (l *lunarNewYear) Cell(x, y int, b scene.Box) (rune, scene.Color) { return l.c.cell(x, y, b) }

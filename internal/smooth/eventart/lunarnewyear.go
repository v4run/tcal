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
	lanternW      = 5 // cells
	lanternRows   = 4 // cap, two rows of body, tassel
	lanternMaxStr = 2 // longest string above a lantern, in rows
	lanternSlot   = lanternW + 3
	lanternMin    = 3
	lanternMax    = 5
	lnyH          = lanternMaxStr + lanternRows
	lnyMargin     = 2 // columns kept clear at each side
)

var (
	lanternRed  = scene.RGB{R: 205, G: 65, B: 55}
	lanternGold = scene.RGB{R: 220, G: 180, B: 85}

	// The lantern is drawn a cell row at a time, so the red body never shares
	// a cell with the gold cap or tassel. Each is 10 dots wide, centred.
	lanternCap = dots.ParseSprite(
		"",
		"",
		"...####...",
		"..######..",
	)
	lanternBody = dots.ParseSprite(
		"..######..",
		".#..##..#.",
		"#..#..#..#",
		"#..#..#..#",
		"#..#..#..#",
		"#..#..#..#",
		".#..##..#.",
		"..######..",
	)
	lanternTassel = dots.ParseSprite(
		"...####...",
		"....##....",
		"....##....",
		"....##....",
	)
)

// lunarNewYear is a row of red lanterns hanging from the top edge, each
// swinging gently on its string.
type lunarNewYear struct {
	w, h  int
	t     float64 // seconds
	n     int
	str   [lanternMax]int // string length per lantern, 1..lanternMaxStr rows
	phase [lanternMax]float64
	f     frame
}

func init() {
	arts[events.LunarNewYear] = func(rng *rand.Rand) Art {
		l := &lunarNewYear{}
		for i := range lanternMax {
			l.str[i] = 1 + rng.IntN(lanternMaxStr)
			l.phase[i] = rng.Float64() * 2 * math.Pi
		}
		return l
	}
}

// lanternAnchor is the column lantern i of n hangs from in a band w wide:
// centred in its share of the band, so lanterns spread evenly.
func lanternAnchor(i, n, w int) float64 {
	return float64((2*i+1)*w) / float64(2*n)
}

func (l *lunarNewYear) Resize(w, h int) {
	l.w, l.h = max(w, 0), max(h, 0)
	l.n = min(lanternMax, max(lanternMin, l.w/30))
	l.f.reset(l.Bounds())
	l.render()
}

func (l *lunarNewYear) Step(dt time.Duration) {
	l.t += max(dt, 0).Seconds()
	l.render()
}

func (l *lunarNewYear) render() {
	c := &l.f.c
	c.Clear()
	if l.f.r.W == 0 {
		return
	}
	for i := range l.n {
		ax := lanternAnchor(i, l.n, l.f.r.W)
		lx := ax + sway(l.t, l.phase[i])
		top := float64(l.str[i])
		// The string swings from a fixed point on the top edge.
		line(c, ax, 0, lx, top+0.5, lanternGold)
		x0 := lx - float64(lanternW)/2
		c.DrawSprite(lanternCap, x0, top, lanternGold)
		c.DrawSprite(lanternBody, x0, top+1, lanternRed)
		c.DrawSprite(lanternTassel, x0, top+3, lanternGold)
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

func (l *lunarNewYear) Cell(x, y int, b scene.Box) (rune, scene.RGB) { return l.f.cell(x, y, b) }

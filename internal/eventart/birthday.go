package eventart

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

// cakeRows is a two-tier cake below its candles; '~' is white frosting,
// the rest is drawn Magenta.
var cakeRows = [...]string{
	` .---------. `,
	` |~~~~~~~~~| `,
	`.'---------'.`,
	`|~~~~~~~~~~~|`,
	`|___________|`,
}

const (
	cakeW        = len(`|___________|`)
	cakeH        = len(cakeRows) + 2 // plus a candle row and a flame row
	candleMin    = 3
	candleMax    = 5
	candleX0     = 2 // first and last candle columns, on the top tier
	candleX1     = 10
	cakeFlicker  = 0.35
	confettiN    = 10
	confettiFall = 4 // ticks per row, at the fastest
	bdayColW     = 15
	bdayMinH     = 12
	bdayMaxH     = 22
)

var (
	confettiGlyphs = [...]rune{'.', ',', '\'', '`', '·'}
	confettiColors = [...]scene.Color{scene.Red, scene.Yellow, scene.Green, scene.Blue, scene.Magenta, scene.Cyan}
)

type confetto struct {
	x, y, speed, age int
	glyph            rune
	color            scene.Color
}

// birthday is a cake in the bottom-right corner with flickering candles,
// and confetti drifting down the column above it.
type birthday struct {
	w, h     int
	rng      *rand.Rand
	n        int
	flames   [candleMax]rune
	confetti [confettiN]confetto
	c        canvas
}

func init() {
	arts[events.Birthday] = func(rng *rand.Rand) Art {
		b := &birthday{rng: rng, n: candleMin + rng.IntN(candleMax-candleMin+1)}
		for i := range b.flames {
			b.flames[i] = flameGlyphs[rng.IntN(len(flameGlyphs))]
		}
		return b
	}
}

func (b *birthday) Resize(w, h int) {
	b.w, b.h = w, h
	b.c.reset(b.Bounds())
	// Spread the confetti over the whole fall so it shows at once.
	for i := range b.confetti {
		b.drop(&b.confetti[i])
		b.confetti[i].y = b.rng.IntN(max(1, b.fallH()))
	}
	b.render()
}

// fallH is the rows above the cake that confetti falls through.
func (b *birthday) fallH() int { return b.c.r.H - cakeH }

// drop restarts a confetto at the top of the column.
func (b *birthday) drop(p *confetto) {
	*p = confetto{
		x:     b.rng.IntN(max(1, b.c.r.W)),
		speed: confettiFall + b.rng.IntN(4),
		glyph: confettiGlyphs[b.rng.IntN(len(confettiGlyphs))],
		color: confettiColors[b.rng.IntN(len(confettiColors))],
	}
}

func (b *birthday) Tick() {
	for i := range b.n {
		if b.rng.Float64() < cakeFlicker {
			b.flames[i] = flameGlyphs[b.rng.IntN(len(flameGlyphs))]
		}
	}
	for i := range b.confetti {
		p := &b.confetti[i]
		p.age++
		if p.age%p.speed != 0 {
			continue
		}
		p.y++
		if b.rng.IntN(3) == 0 {
			p.x = min(max(p.x+b.rng.IntN(3)-1, 0), max(b.c.r.W-1, 0))
		}
		if p.y >= b.fallH() {
			b.drop(p)
		}
	}
	b.render()
}

func (b *birthday) render() {
	if b.c.r.W == 0 {
		return
	}
	b.c.clear()
	for _, p := range b.confetti {
		b.c.set(p.x, p.y, p.glyph, p.color)
	}
	x0, y0 := b.c.r.W-cakeW, b.c.r.H-cakeH
	for i := range b.n {
		x := x0 + candleX0 + i*(candleX1-candleX0)/(b.n-1)
		b.c.set(x, y0, b.flames[i], scene.Yellow)
		b.c.set(x, y0+1, '|', scene.White)
	}
	for y, row := range cakeRows {
		for x, r := range row {
			switch r {
			case ' ':
			case '~':
				b.c.set(x0+x, y0+2+y, r, scene.White)
			default:
				b.c.set(x0+x, y0+2+y, r, scene.Magenta)
			}
		}
	}
}

func (b *birthday) SetTime(time.Time, events.Occurrence) {}

// Bounds is a column up the right side, down to the bottom corner.
func (b *birthday) Bounds() Region {
	return edgeRegion(b.w, b.h, bdayColW, columnHeight(b.h, bdayMinH, bdayMaxH), true, true)
}

func (b *birthday) Cell(x, y int, bx scene.Box) (rune, scene.Color) { return b.c.cell(x, y, bx) }

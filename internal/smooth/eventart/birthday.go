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
	cakeW     = 13 // cells
	cakeH     = 7  // a flame row, a candle row and five rows of cake
	candleMin = 3
	candleMax = 5
	candleX0  = 3 // first and last candle cells, on the top tier
	candleX1  = 9

	confettiN     = 14
	confettiFall  = 1.0 // slowest rows per second; each piece adds up to confettiFallX more
	confettiFallX = 0.8
	confettiSway  = 0.5 // columns a piece flutters either side
	confettiSpin  = 4.0 // turns per second, flipping a fleck between flat and upright
	bdayColW      = 15
	bdayMinH      = 12
	bdayMaxH      = 22
)

var (
	frostingCol = scene.RGB{R: 230, G: 222, B: 210} // cream
	spongeCol   = scene.RGB{R: 200, G: 120, B: 140} // rose
	candleCols  = [...]scene.RGB{
		{R: 120, G: 165, B: 210}, // blue
		{R: 215, G: 135, B: 165}, // pink
		{R: 125, G: 190, B: 150}, // mint
		{R: 225, G: 200, B: 110}, // lemon
		{R: 170, G: 140, B: 205}, // lilac
	}
	confettiCols = [...]scene.RGB{
		{R: 210, G: 95, B: 90},
		{R: 225, G: 190, B: 85},
		{R: 105, G: 175, B: 115},
		{R: 95, G: 150, B: 210},
		{R: 190, G: 115, B: 200},
		{R: 95, G: 185, B: 190},
	}

	// The cake is drawn a cell row at a time so the cream never shares a
	// cell with the rose sponge: a top tier 9 cells wide on a full-width
	// bottom tier, each iced with drips.
	candle      = dots.ParseSprite("##", "##", "##", "##")
	topFrosting = dots.ParseSprite(
		"....##################....",
		"....##################....",
		"....##.###.##.###.###.....",
		"....#...#...#...#...#.....",
	)
	topSponge = dots.ParseSprite(
		"....##################....",
		"....##################....",
		"....##################....",
		"....##################....",
	)
	bottomFrosting = dots.ParseSprite(
		"##########################",
		"##########################",
		"##.####.###.####.###.####.",
		"#...##...#...##...#...##..",
	)
	bottomSponge = dots.ParseSprite(
		"##########################",
		"##########################",
		"##########################",
		"##########################",
		"##########################",
		"##.#.#.#.#.#.#.#.#.#.#.###",
		"##########################",
		".########################.",
	)
	// confettiShapes are a fleck lying flat and turned upright.
	confettiShapes = [...]shape{parseShape("##"), parseShape("#", "#")}
)

type confetto struct {
	x, y  float64 // home column and row, in region cells
	speed float64 // rows per second
	phase float64
	col   scene.RGB
}

// birthday is a cake in the bottom-right corner with flickering candles,
// and confetti fluttering down the column above it.
type birthday struct {
	w, h     int
	t        float64 // seconds
	rng      *rand.Rand
	n        int
	shape    [candleMax]int
	glow     [candleMax]float64
	cols     [candleMax]scene.RGB
	confetti [confettiN]confetto
	f        frame
}

func init() {
	arts[events.Birthday] = func(rng *rand.Rand) Art {
		b := &birthday{rng: rng, n: candleMin + rng.IntN(candleMax-candleMin+1)}
		for i := range b.shape {
			b.shape[i] = rng.IntN(len(flames))
			b.glow[i] = rng.Float64() * 2 * math.Pi
			b.cols[i] = candleCols[(i+rng.IntN(2))%len(candleCols)]
		}
		return b
	}
}

func (b *birthday) Resize(w, h int) {
	b.w, b.h = max(w, 0), max(h, 0)
	b.f.reset(b.Bounds())
	// Spread the confetti over the whole fall so it shows at once.
	for i := range b.confetti {
		b.drop(&b.confetti[i])
		b.confetti[i].y = b.rng.Float64() * b.fallH()
	}
	b.render()
}

// fallH is the rows above the cake that confetti falls through; a piece is
// taken away half a row early, so even upright it stays clear of the cake.
func (b *birthday) fallH() float64 { return max(float64(b.f.r.H-cakeH)-0.5, 0) }

// drop restarts a confetto at the top of the column.
func (b *birthday) drop(p *confetto) {
	*p = confetto{
		x:     confettiSway + b.rng.Float64()*max(float64(b.f.r.W)-1-2*confettiSway, 0),
		speed: confettiFall + confettiFallX*b.rng.Float64(),
		phase: b.rng.Float64() * 2 * math.Pi,
		col:   confettiCols[b.rng.IntN(len(confettiCols))],
	}
}

func (b *birthday) Step(dt time.Duration) {
	dt = max(dt, 0)
	b.t += dt.Seconds()
	p := chance(diyaFlicker, dt)
	for i := range b.n {
		if b.rng.Float64() < p {
			b.shape[i] = (b.shape[i] + 1 + b.rng.IntN(len(flames)-1)) % len(flames)
		}
	}
	for i := range b.confetti {
		c := &b.confetti[i]
		if c.y += c.speed * dt.Seconds(); c.y >= b.fallH() {
			b.drop(c)
		}
	}
	b.render()
}

// candleX is the cake-local cell of candle i.
func (b *birthday) candleX(i int) int {
	return candleX0 + i*(candleX1-candleX0)/max(b.n-1, 1)
}

func (b *birthday) render() {
	c := &b.f.c
	c.Clear()
	if b.f.r.W == 0 {
		return
	}
	for _, p := range b.confetti {
		x := p.x + confettiSway*math.Sin(2*math.Pi*b.t/2.5+p.phase)
		turn := int(math.Floor(2*confettiSpin*b.t+p.phase)) % 2
		confettiShapes[turn].draw(c, x, p.y, p.col, nil)
	}
	x0, y0 := float64(b.f.r.W-cakeW), float64(b.f.r.H-cakeH)
	for i := range b.n {
		x := x0 + float64(b.candleX(i))
		k := 0.5 + 0.5*math.Sin(diyaGlow*b.t+b.glow[i])
		c.DrawSprite(flames[b.shape[i]], x, y0, lerp(flameTip, flameCol, 0.4+0.6*k))
		c.DrawSprite(candle, x, y0+1, b.cols[i])
	}
	c.DrawSprite(topFrosting, x0, y0+2, frostingCol)
	c.DrawSprite(topSponge, x0, y0+3, spongeCol)
	c.DrawSprite(bottomFrosting, x0, y0+4, frostingCol)
	c.DrawSprite(bottomSponge, x0, y0+5, spongeCol)
}

func (b *birthday) SetTime(time.Time, events.Occurrence) {}

// Bounds is a column up the right side, down to the bottom corner.
func (b *birthday) Bounds() Region {
	return edgeRegion(b.w, b.h, bdayColW, columnHeight(b.h, bdayMinH, bdayMaxH), true, true)
}

func (b *birthday) Cell(x, y int, bx scene.Box) (rune, scene.RGB) { return b.f.cell(x, y, bx) }

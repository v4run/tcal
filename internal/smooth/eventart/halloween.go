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
	pumpkinW = 9 // cells; 18×16 dots
	pumpkinH = 4

	batW       = 3 // cells
	batMax     = 2
	batBeat    = 0.08 // seconds per wing position; four make a beat
	batSpeed   = 3.0  // columns per second (rows count double)
	batTurn    = 1.2  // new headings per second, besides on arrival
	pumpkinFl  = 7.0  // radians per second of the candle's flicker
	hallowColW = 14
	hallowMinH = 10
	hallowMaxH = 22
)

var (
	pumpkinCol = scene.RGB{R: 210, G: 120, B: 50}
	stemCol    = scene.RGB{R: 110, G: 140, B: 70}
	faceDim    = scene.RGB{R: 225, G: 135, B: 55}  // candlelight through the carving
	faceBright = scene.RGB{R: 240, G: 190, B: 90}  // and as it flares
	batCol     = scene.RGB{R: 125, G: 110, B: 150} // dusk violet

	pumpkin, pumpkinFace, pumpkinStem = pumpkinSprites()

	// batFrames are a wing beat: up, level, down, level.
	batFrames = [...]dots.Sprite{
		dots.ParseSprite("#....#", "##..##", ".####.", "..##.."),
		dots.ParseSprite("", ".#..#.", "######", "..##.."),
		dots.ParseSprite("", "..##..", ".####.", "##..##"),
		dots.ParseSprite("", ".#..#.", "######", "..##.."),
	}
)

// pumpkinSprites draws a jack-o'-lantern: a ribbed round body with a
// carved face, split into the body, the cells around the face (lit from
// within) and the stem, so each takes its own colour.
func pumpkinSprites() (body, face, stem dots.Sprite) {
	const dw, dh = 2 * pumpkinW, 4 * pumpkinH
	carved := map[[2]int]bool{}
	carve := func(y int, xs ...int) {
		for _, x := range xs {
			carved[[2]int{x, y}] = true
			carved[[2]int{dw - 1 - x, y}] = true // the face is symmetric
		}
	}
	carve(6, 5)       // eyes, triangles pointing up
	carve(7, 4, 5, 6) //
	carve(10, 4, 5, 7, 8)
	carve(11, 5, 6, 7, 8)
	stemAt := map[[2]int]bool{{9, 0}: true, {9, 1}: true, {8, 1}: true, {8, 2}: true}
	faceCell := map[[2]int]bool{}
	for p := range carved {
		faceCell[[2]int{p[0] / 2, p[1] / 4}] = true
	}
	rows := func(keep func(x, y int) bool) dots.Sprite {
		out := make([]string, dh)
		for y := range dh {
			row := make([]byte, dw)
			for x := range dw {
				row[x] = '.'
				if keep(x, y) {
					row[x] = '#'
				}
			}
			out[y] = string(row)
		}
		return dots.ParseSprite(out...)
	}
	in := func(x, y int) bool {
		// An ellipse from dot row 3 down, with a dip at the top and ribs.
		fx, fy := (float64(x)+0.5-dw/2)/(dw/2), (float64(y)+0.5-9.5)/6.5
		if fx*fx+fy*fy > 1 || carved[[2]int{x, y}] {
			return false
		}
		rib := x == 4 || x == dw-5
		return !(rib && (y <= 4 || y >= 14))
	}
	stemCell := func(x, y int) bool { return x/2 == 4 && y/4 == 0 }
	body = rows(func(x, y int) bool { return in(x, y) && !faceCell[[2]int{x / 2, y / 4}] && !stemCell(x, y) })
	face = rows(func(x, y int) bool { return in(x, y) && faceCell[[2]int{x / 2, y / 4}] })
	stem = rows(func(x, y int) bool { return stemAt[[2]int{x, y}] || (stemCell(x, y) && in(x, y)) })
	return body, face, stem
}

type bat struct {
	x, y   float64 // top-left, in region cells
	tx, ty float64 // where it is heading
	phase  float64 // wing beats
}

// halloween is a jack-o'-lantern in the bottom-left corner with a bat or
// two flitting about the upper half of the column above it.
type halloween struct {
	w, h  int
	t     float64 // seconds
	speed float64 // the bats', in columns per second
	rng   *rand.Rand
	bats  []bat
	f     frame
}

func init() {
	arts[events.Halloween] = func(rng *rand.Rand) Art {
		hw := &halloween{rng: rng, speed: batSpeed, bats: make([]bat, 1+rng.IntN(batMax))}
		for i := range hw.bats {
			hw.bats[i].phase = rng.Float64() * 4
		}
		return hw
	}
}

// sky is the area a bat's top-left corner may be in: the upper half.
func (hw *halloween) sky() (w, h float64) {
	return max(float64(hw.f.r.W-batW), 0), max(float64(hw.f.r.H/2-1), 0)
}

func (hw *halloween) Resize(w, h int) {
	hw.w, hw.h = max(w, 0), max(h, 0)
	hw.f.reset(hw.Bounds())
	// Scatter the bats over the new upper half.
	sw, sh := hw.sky()
	for i := range hw.bats {
		b := &hw.bats[i]
		b.x, b.y = hw.rng.Float64()*sw, hw.rng.Float64()*sh
		b.tx, b.ty = b.x, b.y
	}
	hw.render()
}

func (hw *halloween) Step(dt time.Duration) {
	dt = max(dt, 0)
	s := dt.Seconds()
	hw.t += s
	sw, sh := hw.sky()
	p := chance(batTurn, dt)
	for i := range hw.bats {
		b := &hw.bats[i]
		// Distance in columns, a row counting as two.
		dx, dy := b.tx-b.x, 2*(b.ty-b.y)
		d := math.Hypot(dx, dy)
		if d < 0.1 || hw.rng.Float64() < p {
			b.tx, b.ty = hw.rng.Float64()*sw, hw.rng.Float64()*sh
			continue
		}
		k := min(hw.speed*s/d, 1)
		b.x = min(max(b.x+dx*k, 0), sw)
		b.y = min(max(b.y+dy/2*k, 0), sh)
	}
	hw.render()
}

func (hw *halloween) render() {
	c := &hw.f.c
	c.Clear()
	if hw.f.r.W == 0 {
		return
	}
	for _, b := range hw.bats {
		frame := int(math.Floor(hw.t/batBeat+b.phase)) % len(batFrames)
		c.DrawSprite(batFrames[frame], b.x, b.y, batCol)
	}
	y0 := float64(hw.f.r.H - pumpkinH)
	k := 0.5 + 0.25*math.Sin(pumpkinFl*hw.t) + 0.25*math.Sin(2.3*pumpkinFl*hw.t+1)
	c.DrawSprite(pumpkin, 0, y0, pumpkinCol)
	c.DrawSprite(pumpkinFace, 0, y0, lerp(faceDim, faceBright, k))
	c.DrawSprite(pumpkinStem, 0, y0, stemCol)
}

func (hw *halloween) SetTime(time.Time, events.Occurrence) {}

// Bounds is a column up the left side, down to the bottom corner.
func (hw *halloween) Bounds() Region {
	return edgeRegion(hw.w, hw.h, hallowColW, columnHeight(hw.h, hallowMinH, hallowMaxH), false, true)
}

func (hw *halloween) Cell(x, y int, b scene.Box) (rune, scene.RGB) { return hw.f.cell(x, y, b) }

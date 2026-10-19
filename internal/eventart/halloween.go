package eventart

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

// pumpkinRows is a jack-o'-lantern with a carved face.
var pumpkinRows = [...]string{
	` .--)--. `,
	`( ^   ^ )`,
	`(  vvv  )`,
	` '-----' `,
}

// batFrames are a bat's wings up and level.
var batFrames = [...]string{`^v^`, `-v-`}

const (
	pumpkinW   = len(` .--)--. `)
	batW       = 3
	batMax     = 2
	batFlap    = 3   // ticks per wing beat
	batFlit    = 0.3 // chance per tick that a bat moves
	hallowColW = 14
	hallowMinH = 10
	hallowMaxH = 22
)

type bat struct{ x, y, phase int }

// halloween is a jack-o'-lantern in the bottom-left corner with a bat or
// two flitting about the upper half of the column above it.
type halloween struct {
	w, h int
	tick int
	rng  *rand.Rand
	bats []bat
	c    canvas
}

func init() {
	arts[events.Halloween] = func(rng *rand.Rand) Art {
		hw := &halloween{rng: rng, bats: make([]bat, 1+rng.IntN(batMax))}
		for i := range hw.bats {
			hw.bats[i] = bat{x: rng.IntN(1 << 16), y: rng.IntN(1 << 16), phase: i}
		}
		return hw
	}
}

func (hw *halloween) Resize(w, h int) {
	hw.w, hw.h = w, h
	hw.c.reset(hw.Bounds())
	// Scatter the bats over the new upper half.
	for i := range hw.bats {
		b := &hw.bats[i]
		b.x, b.y = b.x%max(1, hw.c.r.W-batW+1), b.y%max(1, hw.c.r.H/2)
	}
	hw.render()
}

func (hw *halloween) Tick() {
	hw.tick++
	maxX, maxY := hw.c.r.W-batW, hw.c.r.H/2-1
	for i := range hw.bats {
		if hw.rng.Float64() < batFlit {
			b := &hw.bats[i]
			b.x = min(max(b.x+hw.rng.IntN(3)-1, 0), max(maxX, 0))
			b.y = min(max(b.y+hw.rng.IntN(3)-1, 0), max(maxY, 0))
		}
	}
	hw.render()
}

func (hw *halloween) render() {
	if hw.c.r.W == 0 {
		return
	}
	hw.c.clear()
	for _, b := range hw.bats {
		hw.c.text(b.x, b.y, batFrames[(hw.tick/batFlap+b.phase)%len(batFrames)], scene.Default)
	}
	y0 := hw.c.r.H - len(pumpkinRows)
	for y, row := range pumpkinRows {
		hw.c.text(0, y0+y, row, scene.Yellow)
	}
}

func (hw *halloween) SetTime(time.Time, events.Occurrence) {}

// Bounds is a column up the left side, down to the bottom corner.
func (hw *halloween) Bounds() Region {
	return edgeRegion(hw.w, hw.h, hallowColW, columnHeight(hw.h, hallowMinH, hallowMaxH), false, true)
}

func (hw *halloween) Cell(x, y int, b scene.Box) (rune, scene.Color) { return hw.c.cell(x, y, b) }

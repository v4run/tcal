package eventart

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const (
	flagW      = 9 // cloth columns
	flagPole   = 4 // pole rows below the cloth
	flagH      = len(flagBands) + flagPole
	flagRipple = 5 // ticks per ripple frame
)

// flagBands are the cloth's colors top to bottom: saffron (drawn Red),
// white and green.
var flagBands = [...]scene.Color{scene.Red, scene.White, scene.Green}

// independence is a flag on a pole in the top-left corner: three bands
// with a wheel in the center, its free edge rippling between two frames.
type independence struct {
	w, h int
	tick int
	c    canvas
}

func init() {
	arts[events.IndependenceDay] = func(*rand.Rand) Art { return &independence{} }
}

func (f *independence) Resize(w, h int) {
	f.w, f.h = w, h
	f.c.reset(f.Bounds())
	f.render()
}

func (f *independence) Tick() {
	f.tick++
	if f.tick%flagRipple == 0 {
		f.render()
	}
}

func (f *independence) render() {
	if f.c.r.W == 0 {
		return
	}
	f.c.clear()
	for y := range flagH {
		f.c.set(0, y, '|', scene.White)
	}
	frame := f.tick / flagRipple % 2
	for y, col := range flagBands {
		// Alternate bands reach one column further, swapping each frame.
		n := flagW - 1
		if (y+frame)%2 == 0 {
			n = flagW
		}
		for x := 1; x < n; x++ {
			f.c.set(x, y, '=', col)
		}
		f.c.set(n, y, '~', col)
	}
	f.c.set(1+flagW/2, 1, 'o', scene.Blue)
}

func (f *independence) SetTime(time.Time, events.Occurrence) {}

// Bounds is the top-left corner: the pole and the cloth beside it.
func (f *independence) Bounds() Region { return edgeRegion(f.w, f.h, 1+flagW, flagH, false, false) }

func (f *independence) Cell(x, y int, b scene.Box) (rune, scene.Color) { return f.c.cell(x, y, b) }

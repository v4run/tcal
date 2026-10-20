package eventart

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const (
	flagW     = 9 // cloth cells; 18×12 dots, the flag's 3:2
	flagBandN = 3
	flagPole  = 4 // pole rows below the cloth
	flagH     = flagBandN + flagPole
	flagBandD = 3 // dot rows of cloth in each band's cell row; one is left for the ripple

	flagWave   = 12.0 // dots from one ripple crest to the next
	flagPeriod = 1.4  // seconds for a crest to run the cloth's length of one wave
	flagShade  = 0.2  // how much the folds darken the cloth
)

var (
	flagSaffron = scene.RGB{R: 225, G: 145, B: 70}
	flagWhite   = scene.RGB{R: 225, G: 222, B: 212}
	flagGreen   = scene.RGB{R: 70, G: 145, B: 85}
	chakraCol   = scene.RGB{R: 60, G: 75, B: 160} // navy
	poleCol     = scene.RGB{R: 150, G: 120, B: 90}
	flagBands   = [flagBandN]scene.RGB{flagSaffron, flagWhite, flagGreen}
)

// independence is a flag on a pole in the top-left corner: three bands
// with the wheel in the centre, a ripple running out to its free edge.
type independence struct {
	w, h int
	t    float64 // seconds
	f    frame
}

func init() {
	arts[events.IndependenceDay] = func(*rand.Rand) Art { return &independence{} }
}

func (f *independence) Resize(w, h int) {
	f.w, f.h = max(w, 0), max(h, 0)
	f.f.reset(f.Bounds())
	f.render()
}

func (f *independence) Step(dt time.Duration) {
	f.t += max(dt, 0).Seconds()
	f.render()
}

// wave is the ripple at cloth dot column x: 0..1, growing from nothing at
// the pole to full at the free edge.
func (f *independence) wave(x float64) float64 {
	env := x / (2*flagW - 1)
	return env * (0.5 + 0.5*math.Sin(2*math.Pi*(x/flagWave-f.t/flagPeriod)))
}

func (f *independence) render() {
	c := &f.f.c
	c.Clear()
	if f.f.r.W == 0 {
		return
	}
	// The pole: a finial, then a staff down the first cell's right side.
	c.Dot(0, 0, poleCol)
	for y := range 4 * flagH {
		c.Dot(0.5, float64(y)/4, poleCol)
	}
	const wheel = 1 + flagW/2 // the cell holding the wheel
	for x := range 2 * flagW {
		cx := 1 + x/2
		w := f.wave(float64(x))
		// The ripple lifts the cloth a dot at its crests and shades the folds.
		off := math.Floor(w + 0.5)
		k := 1 - flagShade*f.wave(float64(x/2*2)+0.5)
		for b, col := range flagBands {
			if b == 1 && cx == wheel {
				continue
			}
			for j := range flagBandD {
				c.Dot(1+float64(x)/2, float64(b)+(float64(j)+off)/4, scale(col, k))
			}
		}
	}
	c.Text(wheel, 1, "o", chakraCol)
}

func (f *independence) SetTime(time.Time, events.Occurrence) {}

// Bounds is the top-left corner: the pole and the cloth beside it.
func (f *independence) Bounds() Region { return edgeRegion(f.w, f.h, 1+flagW, flagH, false, false) }

func (f *independence) Cell(x, y int, b scene.Box) (rune, scene.RGB) { return f.f.cell(x, y, b) }

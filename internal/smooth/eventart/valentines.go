package eventart

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const (
	heartMin   = 3
	heartMax   = 6
	heartRise  = 0.7 // slowest rows per second; each heart adds up to heartRiseX more
	heartRiseX = 0.5
	heartSway  = 0.5 // columns a heart drifts either side as it rises
	heartDrift = 3.0 // seconds per drift
	valColW    = 14
	valMinH    = 10
	valMaxH    = 24
)

var (
	heartRed  = scene.RGB{R: 200, G: 70, B: 85}
	heartPink = scene.RGB{R: 215, G: 125, B: 170}

	// hearts are the sizes a heart can be.
	heartShapes = [...]shape{
		parseShape(
			".##.##.",
			"#######",
			".#####.",
			"..###..",
			"...#...",
		),
		parseShape(
			".##...##.",
			"####.####",
			"#########",
			".#######.",
			"..#####..",
			"...###...",
			"....#....",
		),
	}
)

type heart struct {
	x, y  float64 // home column and top row, in region cells
	speed float64 // rows per second
	phase float64
	size  int
	seed  uint32
}

// valentines is a few hearts rising slowly up a column on the right, red
// low down, turning pink and thinning away near the top.
type valentines struct {
	w, h   int
	t      float64 // seconds
	rng    *rand.Rand
	hearts []heart
	f      frame
}

func init() {
	arts[events.Valentines] = func(rng *rand.Rand) Art {
		return &valentines{rng: rng, hearts: make([]heart, heartMin+rng.IntN(heartMax-heartMin+1))}
	}
}

func (v *valentines) Resize(w, h int) {
	v.w, v.h = max(w, 0), max(h, 0)
	v.f.reset(v.Bounds())
	// Spread the hearts over the column so they show at once.
	n := len(v.hearts)
	for i := range v.hearts {
		v.launch(&v.hearts[i])
		v.hearts[i].y = float64((i+1)*max(v.f.r.H-2, 0)) / float64(n)
	}
	v.render()
}

// launch starts a heart at the bottom of the column.
func (v *valentines) launch(p *heart) {
	size := v.rng.IntN(len(heartShapes))
	dw, dh := heartShapes[size].size()
	room := float64(v.f.r.W) - float64(dw)/2 - 2*heartSway
	*p = heart{
		x:     heartSway + v.rng.Float64()*max(room, 0),
		y:     float64(v.f.r.H) - float64(dh)/4,
		speed: heartRise + heartRiseX*v.rng.Float64(),
		phase: v.rng.Float64() * 2 * math.Pi,
		size:  size,
		seed:  v.rng.Uint32(),
	}
}

func (v *valentines) Step(dt time.Duration) {
	dt = max(dt, 0)
	v.t += dt.Seconds()
	for i := range v.hearts {
		p := &v.hearts[i]
		if p.y -= p.speed * dt.Seconds(); p.y <= 0 {
			v.launch(p)
		}
	}
	v.render()
}

func (v *valentines) render() {
	c := &v.f.c
	c.Clear()
	if v.f.r.W == 0 {
		return
	}
	h := float64(v.f.r.H)
	for _, p := range v.hearts {
		col := heartRed
		if p.y < h/2 {
			col = lerp(heartRed, heartPink, (h/2-p.y)/(h/2))
		}
		// In the top fifth the heart thins away dot by dot.
		keep := min(p.y/(h/5), 1)
		x := p.x + heartSway*math.Sin(2*math.Pi*v.t/heartDrift+p.phase)
		heartShapes[p.size].draw(c, x, p.y, col, func(i int) bool {
			return unit(hash(p.seed, i, 0)) < keep
		})
	}
}

func (v *valentines) SetTime(time.Time, events.Occurrence) {}

// Bounds is a column up the right side, from near the top.
func (v *valentines) Bounds() Region {
	return edgeRegion(v.w, v.h, valColW, columnHeight(v.h, valMinH, valMaxH), true, false)
}

func (v *valentines) Cell(x, y int, b scene.Box) (rune, scene.RGB) { return v.f.cell(x, y, b) }

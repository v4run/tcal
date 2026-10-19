package eventart

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

// heartRows is a small ASCII heart; while fading only its point is left.
var heartRows = [...]string{`(\/)`, ` \/ `}

const (
	heartW    = 4
	heartMin  = 3
	heartMax  = 6
	heartSlow = 6 // fastest ticks per row; each heart adds up to 4 more
	valColW   = 14
	valMinH   = 10
	valMaxH   = 24
)

type heart struct{ x, y, speed, age int }

// valentines is a few hearts rising slowly up a column on the right, red
// low down, turning magenta and fading to a point near the top.
type valentines struct {
	w, h   int
	rng    *rand.Rand
	hearts []heart
	c      canvas
}

func init() {
	arts[events.Valentines] = func(rng *rand.Rand) Art {
		return &valentines{rng: rng, hearts: make([]heart, heartMin+rng.IntN(heartMax-heartMin+1))}
	}
}

func (v *valentines) Resize(w, h int) {
	v.w, v.h = w, h
	v.c.reset(v.Bounds())
	// Spread the hearts over the column so they show at once.
	n := len(v.hearts)
	for i := range v.hearts {
		v.launch(&v.hearts[i])
		v.hearts[i].y = (i + 1) * (v.c.r.H - len(heartRows)) / n
	}
	v.render()
}

// launch starts a heart at the bottom of the column.
func (v *valentines) launch(p *heart) {
	*p = heart{
		x:     v.rng.IntN(max(1, v.c.r.W-heartW+1)),
		y:     v.c.r.H - len(heartRows),
		speed: heartSlow + v.rng.IntN(5),
	}
}

func (v *valentines) Tick() {
	for i := range v.hearts {
		p := &v.hearts[i]
		p.age++
		if p.age%p.speed != 0 {
			continue
		}
		if p.y--; p.y < 0 {
			v.launch(p)
		}
	}
	v.render()
}

func (v *valentines) render() {
	if v.c.r.W == 0 {
		return
	}
	v.c.clear()
	h := v.c.r.H
	for _, p := range v.hearts {
		switch {
		case p.y >= h/2:
			v.c.text(p.x, p.y, heartRows[0], scene.Red)
			v.c.text(p.x, p.y+1, heartRows[1], scene.Red)
		case p.y >= h/5:
			v.c.text(p.x, p.y, heartRows[0], scene.Magenta)
			v.c.text(p.x, p.y+1, heartRows[1], scene.Magenta)
		default:
			v.c.text(p.x, p.y+1, heartRows[1], scene.Magenta)
		}
	}
}

func (v *valentines) SetTime(time.Time, events.Occurrence) {}

// Bounds is a column up the right side, from near the top.
func (v *valentines) Bounds() Region {
	return edgeRegion(v.w, v.h, valColW, columnHeight(v.h, valMinH, valMaxH), true, false)
}

func (v *valentines) Cell(x, y int, b scene.Box) (rune, scene.Color) { return v.c.cell(x, y, b) }

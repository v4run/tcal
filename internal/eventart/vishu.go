package eventart

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

// lampRows is a brass nilavilakku below its row of three flames.
var lampRows = [...]string{
	`\_____/`,
	`   |   `,
	`   |   `,
	`  _|_  `,
	` /___\ `,
}

// konnaClusters are the shapes a falling cluster of flowers can take, as
// offsets from its anchor.
var konnaClusters = [...][3]struct {
	dx, dy int
	g      rune
}{
	{{0, 0, '*'}, {1, 1, '·'}, {-1, 1, ','}},
	{{0, 0, '·'}, {1, 0, '*'}, {0, 1, ','}},
	{{0, 0, '*'}, {-1, 1, '·'}, {1, 2, ','}},
}

const (
	lampW       = len(`\_____/`)
	lampH       = len(lampRows) + 1 // plus the flame row
	lampFlicker = 0.35
	konnaN      = 4
	konnaFall   = 6 // fastest ticks per row
	vishuColW   = 14
	vishuMinH   = 10
	vishuMaxH   = 22
)

type cluster struct{ x, y, shape, speed, age, phase int }

// vishu is a lit lamp in the bottom-right corner, with golden konna
// flowers drifting down the column above it in light clusters.
type vishu struct {
	w, h     int
	tick     int
	rng      *rand.Rand
	flames   [3]rune
	clusters [konnaN]cluster
	c        canvas
}

func init() {
	arts[events.Vishu] = func(rng *rand.Rand) Art {
		v := &vishu{rng: rng}
		for i := range v.flames {
			v.flames[i] = flameGlyphs[rng.IntN(len(flameGlyphs))]
		}
		return v
	}
}

func (v *vishu) Resize(w, h int) {
	v.w, v.h = w, h
	v.c.reset(v.Bounds())
	// Spread the clusters over the fall so they show at once.
	for i := range v.clusters {
		v.drop(&v.clusters[i])
		v.clusters[i].y = i * v.c.r.H / konnaN
	}
	v.render()
}

// drop restarts a cluster just above the top of the column.
func (v *vishu) drop(c *cluster) {
	*c = cluster{
		x:     1 + v.rng.IntN(max(1, v.c.r.W-2)),
		y:     -1,
		shape: v.rng.IntN(len(konnaClusters)),
		speed: konnaFall + v.rng.IntN(4),
		phase: v.rng.IntN(40),
	}
}

func (v *vishu) Tick() {
	v.tick++
	for i := range v.flames {
		if v.rng.Float64() < lampFlicker {
			v.flames[i] = flameGlyphs[v.rng.IntN(len(flameGlyphs))]
		}
	}
	for i := range v.clusters {
		c := &v.clusters[i]
		c.age++
		if c.age%c.speed == 0 {
			if c.y++; c.y >= v.c.r.H {
				v.drop(c)
			}
		}
	}
	v.render()
}

func (v *vishu) render() {
	if v.c.r.W == 0 {
		return
	}
	v.c.clear()
	for _, c := range v.clusters {
		x := c.x + swayOffset(v.tick, c.phase)
		for _, f := range konnaClusters[c.shape] {
			v.c.set(x+f.dx, c.y+f.dy, f.g, scene.Yellow)
		}
	}
	x0, y0 := v.c.r.W-lampW, v.c.r.H-lampH
	for i, f := range v.flames {
		v.c.set(x0+1+2*i, y0, f, scene.Red)
	}
	for y, row := range lampRows {
		v.c.text(x0, y0+1+y, row, scene.Yellow)
	}
}

func (v *vishu) SetTime(time.Time, events.Occurrence) {}

// Bounds is a column up the right side, down to the bottom corner.
func (v *vishu) Bounds() Region {
	return edgeRegion(v.w, v.h, vishuColW, columnHeight(v.h, vishuMinH, vishuMaxH), true, true)
}

func (v *vishu) Cell(x, y int, b scene.Box) (rune, scene.Color) { return v.c.cell(x, y, b) }

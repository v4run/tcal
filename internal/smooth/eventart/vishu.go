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
	lampW      = 7 // cells
	lampH      = 6 // a flame row above the 5-row lamp
	lampFlames = 3

	konnaN      = 5
	konnaFall   = 0.9 // slowest rows per second; each cluster adds up to konnaFallX more
	konnaFallX  = 0.5
	konnaSway   = 0.6 // columns a cluster drifts either side as it falls
	konnaDrift  = 3.5 // seconds per drift
	konnaBlooms = 3   // blossoms in a cluster

	vishuColW = 14
	vishuMinH = 10
	vishuMaxH = 22
)

var (
	konnaCol = scene.RGB{R: 235, G: 190, B: 60} // golden shower
	lampCol  = scene.RGB{R: 195, G: 155, B: 80} // brass

	// nilavilakku is a brass lamp, 7 cells by 5 rows: a wide dish holding
	// the wicks, a stem with a knop, and a stepped foot.
	nilavilakku = dots.ParseSprite(
		"##..........##",
		".############.",
		"..##########..",
		"....######....",
		"......##......",
		"......##......",
		"......##......",
		"......##......",
		"......##......",
		"......##......",
		".....####.....",
		"......##......",
		"......##......",
		".....####.....",
		"......##......",
		"......##......",
		".....####.....",
		"...########...",
		"..##########..",
		".############.",
	)
	// konnaBloom is one small blossom: a cross of petals.
	konnaBloom = parseShape(
		".#.",
		"#.#",
		".#.",
	)
)

type cluster struct {
	x, y  float64 // home column and top row, in region cells
	speed float64 // rows per second
	phase float64
	seed  uint32
}

// vishu is a lit nilavilakku in the bottom-right corner, with golden konna
// flowers drifting down the column above it in light clusters.
type vishu struct {
	w, h    int
	t       float64 // seconds
	rng     *rand.Rand
	shape   [lampFlames]int
	glow    [lampFlames]float64
	flowers [konnaN]cluster
	f       frame
}

func init() {
	arts[events.Vishu] = func(rng *rand.Rand) Art {
		v := &vishu{rng: rng}
		for i := range v.shape {
			v.shape[i] = rng.IntN(len(flames))
			v.glow[i] = rng.Float64() * 2 * math.Pi
		}
		return v
	}
}

func (v *vishu) Resize(w, h int) {
	v.w, v.h = max(w, 0), max(h, 0)
	v.f.reset(v.Bounds())
	// Spread the clusters over the fall so they show at once.
	for i := range v.flowers {
		v.drop(&v.flowers[i])
		v.flowers[i].y = float64(i * v.f.r.H / konnaN)
	}
	v.render()
}

// drop restarts a cluster just above the top of the column.
func (v *vishu) drop(c *cluster) {
	*c = cluster{
		x:     konnaSway + v.rng.Float64()*max(float64(v.f.r.W)-2-2*konnaSway, 0),
		y:     -1,
		speed: konnaFall + konnaFallX*v.rng.Float64(),
		phase: v.rng.Float64() * 2 * math.Pi,
		seed:  v.rng.Uint32(),
	}
}

func (v *vishu) Step(dt time.Duration) {
	dt = max(dt, 0)
	v.t += dt.Seconds()
	p := chance(diyaFlicker, dt)
	for i := range v.shape {
		if v.rng.Float64() < p {
			v.shape[i] = (v.shape[i] + 1 + v.rng.IntN(len(flames)-1)) % len(flames)
		}
	}
	for i := range v.flowers {
		c := &v.flowers[i]
		if c.y += c.speed * dt.Seconds(); c.y >= float64(v.f.r.H) {
			v.drop(c)
		}
	}
	v.render()
}

func (v *vishu) render() {
	c := &v.f.c
	c.Clear()
	if v.f.r.W == 0 {
		return
	}
	x0, y0 := float64(v.f.r.W-lampW), float64(v.f.r.H-lampH)
	for _, fl := range v.flowers {
		x := fl.x + konnaSway*math.Sin(2*math.Pi*v.t/konnaDrift+fl.phase)
		// Each blossom sits at its own offset within the cluster, and passes
		// behind the lamp.
		for b := range konnaBlooms {
			bx := x + 2*unit(hash(fl.seed, b, 0))
			by := fl.y + 0.4*float64(b) + 0.3*unit(hash(fl.seed, b, 1))
			for _, p := range konnaBloom {
				dx, dy := bx+float64(p[0])/2, by+float64(p[1])/4
				if math.Floor(dx) >= x0 && math.Floor(dy) >= y0 {
					continue
				}
				c.Dot(dx, dy, konnaCol)
			}
		}
	}
	for i := range lampFlames {
		k := 0.5 + 0.5*math.Sin(diyaGlow*v.t+v.glow[i])
		c.DrawSprite(flames[v.shape[i]], x0+1+2*float64(i), y0, lerp(flameTip, flameCol, 0.4+0.6*k))
	}
	c.DrawSprite(nilavilakku, x0, y0+1, lampCol)
}

func (v *vishu) SetTime(time.Time, events.Occurrence) {}

// Bounds is a column up the right side, down to the bottom corner.
func (v *vishu) Bounds() Region {
	return edgeRegion(v.w, v.h, vishuColW, columnHeight(v.h, vishuMinH, vishuMaxH), true, true)
}

func (v *vishu) Cell(x, y int, b scene.Box) (rune, scene.RGB) { return v.f.cell(x, y, b) }

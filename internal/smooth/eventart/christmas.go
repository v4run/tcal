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
	treeW      = 13 // cells; 26×32 dots
	treeH      = 8
	treeMargin = 1
	treeTiers  = 3

	twinkleRate   = 4.0 // radians per second of each light's brightness pulse
	twinkleChange = 1.5 // seconds each light holds a colour before crossfading
	twinkleBlend  = 0.4 // seconds of that crossfade
)

var (
	treeCol   = scene.RGB{R: 70, G: 145, B: 85}  // pine
	trunkCol  = scene.RGB{R: 135, G: 95, B: 60}  // bark
	starCol   = scene.RGB{R: 240, G: 205, B: 95} // gold
	lightCols = [...]scene.RGB{
		{R: 225, G: 75, B: 65},   // red
		{R: 240, G: 195, B: 80},  // gold
		{R: 95, G: 145, B: 230},  // blue
		{R: 230, G: 230, B: 220}, // white
	}

	// treeLights is the cells inside the body that hold a light, scattered
	// down the tiers like a garland.
	treeLights = [][2]int{
		{6, 2},
		{7, 3},
		{5, 4}, {8, 4},
		{4, 5}, {7, 5},
		{3, 6}, {6, 6}, {9, 6},
	}
	tree, trunk = treeSprites()
	// lightDot is a light: two dots, upright in the middle of its cell.
	lightDot = dots.ParseSprite("", "#", "#")
)

// inTree reports whether dot (x, y) of the 26×32 tree grid is in the body:
// three stacked, widening tiers below the star's cell, each a triangle.
func inTree(x, y int) bool {
	for k := range treeTiers {
		j := y - (4 + 7*k)
		if j < 0 || j > 9 {
			continue
		}
		hw := 1 + j*(5+3*k)/9
		if x >= treeW-hw && x <= treeW-1+hw {
			return true
		}
	}
	return false
}

// treeSprites draws the tree body, solid but for the cells where the
// lights hang, and its trunk.
func treeSprites() (body, base dots.Sprite) {
	lit := map[[2]int]bool{}
	for _, l := range treeLights {
		lit[l] = true
	}
	rows := make([]string, 4*treeH)
	for y := range rows {
		row := make([]byte, 2*treeW)
		for x := range row {
			row[x] = '.'
			if lit[[2]int{x / 2, y / 4}] || !inTree(x, y) {
				continue
			}
			row[x] = '#'
		}
		rows[y] = string(row)
	}
	return dots.ParseSprite(rows...), dots.ParseSprite(
		"...........####",
		"...........####",
		"..........######",
	)
}

// christmas is a tree in the bottom-right corner with twinkling lights.
type christmas struct {
	w, h  int
	t     float64 // seconds
	phase []float64
	f     frame
}

func init() {
	arts[events.Christmas] = func(rng *rand.Rand) Art {
		c := &christmas{phase: make([]float64, len(treeLights))}
		for i := range c.phase {
			c.phase[i] = rng.Float64() * 2 * math.Pi
		}
		return c
	}
}

func (c *christmas) Resize(w, h int) {
	c.w, c.h = max(w, 0), max(h, 0)
	c.f.reset(c.Bounds())
	c.render()
}

func (c *christmas) Step(dt time.Duration) {
	c.t += max(dt, 0).Seconds()
	c.render()
}

func (c *christmas) render() {
	cv := &c.f.c
	cv.Clear()
	if c.f.r.W == 0 {
		return
	}
	cv.DrawSprite(tree, 0, 0, treeCol)
	cv.DrawSprite(trunk, 0, treeH-1, trunkCol)
	cv.Text(treeW/2, 0, "*", starCol)
	for i, l := range treeLights {
		cv.DrawSprite(lightDot, float64(l[0])+0.5*float64(i%2), float64(l[1]), c.light(i))
	}
}

// light is light i's colour now: it holds each colour for a while, then
// crossfades to the next, pulsing gently all the while.
func (c *christmas) light(i int) scene.RGB {
	p := c.phase[i]
	s := c.t/twinkleChange + p
	n := int(math.Floor(s))
	from := lightCols[(n%len(lightCols)+len(lightCols))%len(lightCols)]
	to := lightCols[((n+1)%len(lightCols)+len(lightCols))%len(lightCols)]
	frac := s - float64(n)
	blend := max(0, (frac*twinkleChange-(twinkleChange-twinkleBlend))/twinkleBlend)
	return scale(lerp(from, to, blend), 0.7+0.3*math.Sin(twinkleRate*c.t+p))
}

func (c *christmas) SetTime(time.Time, events.Occurrence) {}

// Bounds is the bottom-right corner, one cell in from each edge.
func (c *christmas) Bounds() Region {
	if c.w < treeW+2*treeMargin || c.h < treeH+2*treeMargin {
		return Region{}
	}
	return Region{X0: c.w - treeW - treeMargin, Y0: c.h - treeH - treeMargin, W: treeW, H: treeH}
}

func (c *christmas) Cell(x, y int, b scene.Box) (rune, scene.RGB) { return c.f.cell(x, y, b) }

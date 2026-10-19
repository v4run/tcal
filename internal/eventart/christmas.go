package eventart

import (
	"math/rand"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

// treeRows is the tree; 'o' marks a light, '*' the star.
var treeRows = [treeH]string{
	`      *      `,
	`     /o\     `,
	`    /o o\    `,
	`   /o o o\   `,
	`  /o o o o\  `,
	` /o o o o o\ `,
	`/___________\`,
	`     |||     `,
}

const (
	treeW         = 13
	treeH         = 8
	treeMargin    = 1
	twinkleTicks  = 3
	treeLightKind = 'o'
)

var lightColors = [...]scene.Color{scene.Red, scene.Yellow, scene.Blue}

type christmas struct {
	w, h int
	tick int
	rng  *rand.Rand
	// phase offsets each light so neighbours differ; indexed like the template.
	phase [treeH][treeW]int
}

func init() {
	arts[events.Christmas] = func(rng *rand.Rand) Art {
		c := &christmas{rng: rng}
		for y := range c.phase {
			for x := range c.phase[y] {
				c.phase[y][x] = rng.Intn(len(lightColors))
			}
		}
		return c
	}
}

func (c *christmas) Resize(w, h int) { c.w, c.h = w, h }
func (c *christmas) Tick()           { c.tick++ }

func (c *christmas) SetTime(time.Time, events.Occurrence) {}

func (c *christmas) Bounds() Region {
	if c.w < treeW+2*treeMargin || c.h < treeH+2*treeMargin {
		return Region{}
	}
	return Region{X0: c.w - treeW - treeMargin, Y0: c.h - treeH - treeMargin, W: treeW, H: treeH}
}

func (c *christmas) Cell(x, y int, b scene.Box) (rune, scene.Color) {
	bd := c.Bounds()
	if bd.W == 0 || bd.Touches(b) {
		return 0, scene.Default
	}
	tx, ty := x-bd.X0, y-bd.Y0
	if tx < 0 || tx >= treeW || ty < 0 || ty >= treeH {
		return 0, scene.Default
	}
	r := rune(treeRows[ty][tx])
	switch r {
	case ' ':
		return 0, scene.Default
	case '*':
		return r, scene.Yellow
	case treeLightKind:
		return r, lightColors[(c.phase[ty][tx]+c.tick/twinkleTicks)%len(lightColors)]
	}
	return r, scene.Green
}

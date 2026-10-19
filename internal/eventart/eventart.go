// Package eventart draws small dim scenes for holiday events. Each art sits
// in a corner of the screen and hides entirely when it would touch the
// calendar box.
package eventart

import (
	"math/rand"
	"sort"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

// Region is a rectangle of screen cells.
type Region struct{ X0, Y0, W, H int }

// Touches reports whether the region overlaps the box by at least one cell.
// An empty box (Y0 == -1) never touches.
func (r Region) Touches(b scene.Box) bool {
	if b.Y0 == -1 || r.W <= 0 || r.H <= 0 {
		return false
	}
	return b.X0 < r.X0+r.W && b.X1 > r.X0 && b.Y0 < r.Y0+r.H && b.Y1 >= r.Y0
}

// Art draws one event's scene within Bounds, hiding entirely when
// Bounds touches the calendar box.
type Art interface {
	Resize(w, h int)
	Tick()
	SetTime(now time.Time, occ events.Occurrence)
	Bounds() Region                                 // the full area the art may ever draw in, for the current size
	Cell(x, y int, b scene.Box) (rune, scene.Color) // 0 rune = empty
}

var arts = map[events.Kind]func(*rand.Rand) Art{}

// New returns the art for k, or nil if k has none yet.
func New(k events.Kind, rng *rand.Rand) Art {
	f, ok := arts[k]
	if !ok {
		return nil
	}
	return f(rng)
}

// Kinds returns the kinds that have art, sorted.
func Kinds() []events.Kind {
	ks := make([]events.Kind, 0, len(arts))
	for k := range arts {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })
	return ks
}

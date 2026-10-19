package tui

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/life"
	"github.com/v4run/tcal/internal/runner"
	"github.com/v4run/tcal/internal/scene"
	"github.com/v4run/tcal/internal/scenes/ants"
	"github.com/v4run/tcal/internal/scenes/aquarium"
	"github.com/v4run/tcal/internal/scenes/automaton"
	"github.com/v4run/tcal/internal/scenes/flow"
	"github.com/v4run/tcal/internal/scenes/harbor"
	"github.com/v4run/tcal/internal/scenes/hourglass"
	"github.com/v4run/tcal/internal/scenes/langton"
	"github.com/v4run/tcal/internal/scenes/lava"
	"github.com/v4run/tcal/internal/scenes/rain"
	"github.com/v4run/tcal/internal/scenes/seasons"
	"github.com/v4run/tcal/internal/scenes/sky"
	"github.com/v4run/tcal/internal/scenes/starfield"
	"github.com/v4run/tcal/internal/scenes/train"
)

const (
	faintOn  = "\x1b[2m"
	faintOff = "\x1b[22m"
	fgReset  = "\x1b[39m"
)

// bgStyle is how background glyphs are styled.
type bgStyle int

const (
	stylePlain bgStyle = iota // no escapes at all (NO_COLOR, --no-color)
	styleFaint                // dim, in the terminal's foreground color
	styleColor                // dim, tinted by scenes that implement scene.Colored
)

// sgrColor returns the escape that selects c as the foreground color.
func sgrColor(c scene.Color) string {
	if c == scene.Default {
		return fgReset
	}
	return fmt.Sprintf("\x1b[%dm", 30+int(c))
}

// backgrounds maps --bg names to scene constructors. Constructors receive
// the initial terminal size; Resize is called again on every resize.
var backgrounds = map[string]func(w, h int, rng *rand.Rand) scene.Scene{
	"life": func(w, h int, rng *rand.Rand) scene.Scene {
		s := &lifeScene{rng: rng}
		s.Resize(w, h)
		return s
	},
	"container": func(w, h int, rng *rand.Rand) scene.Scene {
		return &runnerScene{s: runner.New(w, h, rng), w: w, h: h}
	},
	"rain":      func(w, h int, rng *rand.Rand) scene.Scene { return rain.New(w, h, rng) },
	"starfield": func(w, h int, rng *rand.Rand) scene.Scene { return starfield.New(w, h, rng) },
	"aquarium":  func(w, h int, rng *rand.Rand) scene.Scene { return aquarium.New(w, h, rng) },
	"lava":      func(w, h int, rng *rand.Rand) scene.Scene { return lava.New(w, h, rng) },
	"flow":      func(w, h int, rng *rand.Rand) scene.Scene { return flow.New(w, h, rng) },
	"langton":   func(w, h int, rng *rand.Rand) scene.Scene { return langton.New(w, h, rng) },
	"automaton": func(w, h int, rng *rand.Rand) scene.Scene { return automaton.New(w, h, rng) },
	"sky":       func(w, h int, rng *rand.Rand) scene.Scene { return sky.New(w, h, rng) },
	"seasons":   func(w, h int, rng *rand.Rand) scene.Scene { return seasons.New(w, h, rng) },
	"hourglass": func(w, h int, rng *rand.Rand) scene.Scene { return hourglass.New(w, h, rng) },
	"harbor":    func(w, h int, rng *rand.Rand) scene.Scene { return harbor.New(w, h, rng) },
	"train":     func(w, h int, rng *rand.Rand) scene.Scene { return train.New(w, h, rng) },
	"ants":      func(w, h int, rng *rand.Rand) scene.Scene { return ants.New(w, h, rng) },
}

// Random is the background name that picks one of Backgrounds at startup.
const Random = "random"

// Backgrounds returns the registered background names, sorted. WithBackground
// also accepts Random.
func Backgrounds() []string {
	return slices.Sorted(maps.Keys(backgrounds))
}

func newScene(name string, w, h int) scene.Scene {
	seed := uint64(time.Now().UnixNano())
	rng := rand.New(rand.NewPCG(seed, seed>>32))
	if name == Random {
		names := Backgrounds()
		name = names[rng.IntN(len(names))]
	}
	ctor, ok := backgrounds[name]
	if !ok {
		return nil
	}
	return ctor(w, h, rng)
}

// lifeScene runs Game of Life over the whole screen; a resize reseeds it.
type lifeScene struct {
	g   *life.Grid
	rng *rand.Rand
}

func (l *lifeScene) Resize(w, h int) {
	l.g = life.New(w, h, l.rng)
	l.g.Seed()
}

func (l *lifeScene) Tick()                   { l.g.Tick() }
func (l *lifeScene) Interval() time.Duration { return 250 * time.Millisecond }

func (l *lifeScene) Glyph(x, y int, _ scene.Box) rune {
	if l.g.Alive(x, y) {
		return '·'
	}
	return 0
}

func (l *lifeScene) Color(int, int, scene.Box) scene.Color { return scene.Green }

// runnerScene runs the container along the bottom of the screen, hiding
// it entirely when the strip would overlap the calendar.
type runnerScene struct {
	s    *runner.Scene
	w, h int
}

func (d *runnerScene) Resize(w, h int) {
	d.w, d.h = w, h
	d.s.Resize(w, h)
}

func (d *runnerScene) Tick()                   { d.s.Tick() }
func (d *runnerScene) Interval() time.Duration { return 40 * time.Millisecond }

func (d *runnerScene) Glyph(x, y int, b scene.Box) rune {
	if d.w < runner.MinWidth || b.Y1 >= d.h-runner.Height {
		return 0
	}
	return d.s.Glyph(x, y)
}

func (d *runnerScene) Color(x, y int, _ scene.Box) scene.Color { return d.s.Color(x, y) }

// overlay draws sc into the blank margins of frame, leaving the calendar's
// bounding box (plus one cell of padding) untouched. frame must be exactly
// w × h (as produced by render.Fit); otherwise it is returned unchanged.
func overlay(frame string, sc scene.Scene, w, h int, style bgStyle) string {
	lines := strings.Split(frame, "\n")
	if len(lines) != h {
		return frame
	}
	for _, ln := range lines {
		if lipgloss.Width(ln) != w {
			return frame
		}
	}

	// Content rows are those with any visible non-space character.
	y0, y1 := -1, -1
	for y, ln := range lines {
		if strings.TrimSpace(ansi.Strip(ln)) != "" {
			if y0 < 0 {
				y0 = y
			}
			y1 = y
		}
	}

	if y0 < 0 {
		empty := scene.Box{Y0: -1, Y1: -1}
		for y := range lines {
			lines[y] = cells(sc, y, 0, w, empty, style)
		}
		return strings.Join(lines, "\n")
	}

	// render.Fit pads with plain spaces, so leading/trailing ' ' bytes on
	// content rows are pure margin and safe to replace byte-for-byte.
	x0, x1 := w, 0
	for _, ln := range lines[y0 : y1+1] {
		lead := len(ln) - len(strings.TrimLeft(ln, " "))
		trail := len(ln) - len(strings.TrimRight(ln, " "))
		x0 = min(x0, lead)
		x1 = max(x1, w-trail)
	}

	// Pad the box by one cell on every side.
	b := scene.Box{X0: max(x0-1, 0), X1: min(x1+1, w), Y0: y0 - 1, Y1: y1 + 1}

	for y, ln := range lines {
		if y < b.Y0 || y > b.Y1 {
			lines[y] = cells(sc, y, 0, w, b, style)
			continue
		}
		lines[y] = cells(sc, y, 0, b.X0, b, style) + ln[b.X0:len(ln)-(w-b.X1)] + cells(sc, y, b.X1, w, b, style)
	}
	return strings.Join(lines, "\n")
}

// cells renders row y of sc over columns [from, to) as glyphs and spaces.
// Color escapes are emitted only when the color changes between glyphs.
func cells(sc scene.Scene, y, from, to int, b scene.Box, style bgStyle) string {
	if from >= to {
		return ""
	}
	colored, _ := sc.(scene.Colored)
	if style != styleColor {
		colored = nil
	}
	var sb strings.Builder
	if style != stylePlain {
		sb.WriteString(faintOn)
	}
	cur := scene.Default
	for x := from; x < to; x++ {
		r := sc.Glyph(x, y, b)
		if r == 0 {
			sb.WriteByte(' ')
			continue
		}
		if colored != nil {
			if c := colored.Color(x, y, b); c != cur {
				sb.WriteString(sgrColor(c))
				cur = c
			}
		}
		sb.WriteRune(r)
	}
	if cur != scene.Default {
		sb.WriteString(fgReset)
	}
	if style != stylePlain {
		sb.WriteString(faintOff)
	}
	return sb.String()
}

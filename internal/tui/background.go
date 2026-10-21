package tui

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/life"
	"github.com/v4run/tcal/internal/palette"
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
	"github.com/v4run/tcal/internal/smooth/coast"
	"github.com/v4run/tcal/internal/smooth/trees"
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
	styleColor                // dim, tinted by scenes that implement scene.Colored or scene.TrueColored
)

// bgPaint is how the overlay colours background glyphs: the style, plus the
// colour mode and terminal background used for true-colour scenes.
type bgPaint struct {
	style bgStyle
	mode  palette.Mode
	bg    scene.RGB
	// behind is the background's opacity behind the calendar, 0 to 100;
	// 0 leaves the calendar's box alone.
	behind int
}

// sgrColor returns the escape that selects c as the foreground color.
func sgrColor(c scene.Color) string {
	if c == scene.Default {
		return fgReset
	}
	return fmt.Sprintf("\x1b[%dm", 30+int(c))
}

// BackgroundOptions configures the ambient background.
type BackgroundOptions struct {
	// Events shows holiday scenes in the seasons background.
	Events bool
	// Birthday adds a birthday event when non-nil.
	Birthday *events.MonthDay
	// RandomPool lists the backgrounds Random picks from; nil means all.
	RandomPool []string
	// Style is StyleClassic or StyleSmooth; empty means classic.
	Style string
}

// backgrounds maps --bg names to scene constructors. Constructors receive
// the initial terminal size; Resize is called again on every resize.
var backgrounds = map[string]func(w, h int, rng *rand.Rand, o BackgroundOptions) scene.Scene{
	"life": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene {
		s := &lifeScene{rng: rng}
		s.Resize(w, h)
		return s
	},
	"container": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene {
		return &runnerScene{s: runner.New(w, h, rng), w: w, h: h}
	},
	"rain":      func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene { return rain.New(w, h, rng) },
	"starfield": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene { return starfield.New(w, h, rng) },
	"aquarium":  func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene { return aquarium.New(w, h, rng) },
	"lava":      func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene { return lava.New(w, h, rng) },
	"flow":      func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene { return flow.New(w, h, rng) },
	"langton":   func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene { return langton.New(w, h, rng) },
	"automaton": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene { return automaton.New(w, h, rng) },
	"sky":       func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene { return sky.New(w, h, rng) },
	"seasons": func(w, h int, rng *rand.Rand, o BackgroundOptions) scene.Scene {
		s := seasons.New(w, h, rng)
		if o.Events {
			s.EnableEvents(o.Birthday)
		}
		return s
	},
	"hourglass": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene { return hourglass.New(w, h, rng) },
	"harbor":    func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene { return harbor.New(w, h, rng) },
	"train":     func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene { return train.New(w, h, rng) },
	"ants":      func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene { return ants.New(w, h, rng) },
	// trees and coast have no 16-colour versions: classic draws the
	// smooth scenes with the nearest palette colours.
	"trees": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene { return trees.New(w, h, rng) },
	"coast": func(w, h int, rng *rand.Rand, _ BackgroundOptions) scene.Scene { return coast.New(w, h, rng) },
}

// Random is the background name that picks one of Backgrounds at startup.
const Random = "random"

// Backgrounds returns the registered background names, sorted. WithBackground
// also accepts Random.
func Backgrounds() []string {
	return slices.Sorted(maps.Keys(backgrounds))
}

// newScene builds the named background in style o.Style; a name with no
// smooth constructor uses its classic one. For Random it draws uniformly from
// o.RandomPool (all backgrounds when nil), except that seasons is chosen
// whenever events are on, seasons is in the pool and an event is active at
// bgNow.
func newScene(name string, w, h int, o BackgroundOptions, bgNow time.Time) scene.Scene {
	seed := uint64(time.Now().UnixNano())
	rng := rand.New(rand.NewPCG(seed, seed>>32))
	if name == Random {
		pool := o.RandomPool
		if pool == nil {
			pool = Backgrounds()
		}
		if len(pool) == 0 {
			return nil
		}
		name = pool[rng.IntN(len(pool))]
		if o.Events && slices.Contains(pool, "seasons") {
			if _, ok := events.Active(bgNow, o.Birthday); ok {
				name = "seasons"
			}
		}
	}
	ctor, ok := smoothBackgrounds[name]
	if o.Style != StyleSmooth || !ok {
		ctor, ok = backgrounds[name]
	}
	if !ok {
		return nil
	}
	return ctor(w, h, rng, o)
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
func overlay(frame string, sc scene.Scene, w, h int, p bgPaint) string {
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
			lines[y] = cells(sc, y, 0, w, empty, p)
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
			lines[y] = cells(sc, y, 0, w, b, p)
			continue
		}
		mid := ln[b.X0 : len(ln)-(w-b.X1)]
		if p.behind > 0 {
			mid = behind(sc, y, b.X0, mid, p)
		}
		lines[y] = cells(sc, y, 0, b.X0, b, p) + mid + cells(sc, y, b.X1, w, b, p)
	}
	return strings.Join(lines, "\n")
}

// cells renders row y of sc over columns [from, to) as glyphs and spaces.
// Color escapes are emitted only when the color changes between glyphs.
// True-colour scenes are blended toward p.bg by keepFactor, then emitted as
// 24-bit escapes (without faint) in TrueColor mode or, in ANSI mode, as the
// palette colour nearest the unblended scene colour.
func cells(sc scene.Scene, y, from, to int, b scene.Box, p bgPaint) string {
	if from >= to {
		return ""
	}
	colored, _ := sc.(scene.Colored)
	trueColored, _ := sc.(scene.TrueColored)
	if p.style != styleColor {
		colored, trueColored = nil, nil
	}
	// In true colour the blend toward the background already dims the
	// scene; faint on top would dim it twice.
	faint := p.style != stylePlain && (trueColored == nil || p.mode != palette.TrueColor)
	var sb strings.Builder
	if faint {
		sb.WriteString(faintOn)
	}
	cur := scene.Default
	var curRGB scene.RGB
	rgbSet := false
	for x := from; x < to; x++ {
		r := sc.Glyph(x, y, b)
		if r == 0 {
			sb.WriteByte(' ')
			continue
		}
		switch {
		case trueColored != nil:
			raw := trueColored.RGB(x, y, b)
			rgb := palette.Blend(raw, p.bg, keepFactor)
			if p.mode == palette.TrueColor {
				if !rgbSet || rgb != curRGB {
					fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm", rgb.R, rgb.G, rgb.B)
					curRGB, rgbSet = rgb, true
				}
				break
			}
			// Nearest of the unblended colour: blending toward a light
			// background desaturates and would lose the hue.
			if c := palette.Nearest(raw); c != cur {
				sb.WriteString(sgrColor(c))
				cur = c
			}
		case colored != nil:
			if c := colored.Color(x, y, b); c != cur {
				sb.WriteString(sgrColor(c))
				cur = c
			}
		}
		sb.WriteRune(r)
	}
	if cur != scene.Default || rgbSet {
		sb.WriteString(fgReset)
	}
	if faint {
		sb.WriteString(faintOff)
	}
	return sb.String()
}

// behind replaces the blank cells of seg, which starts at column x0 of row
// y, with the scene's glyphs. Spaces inside a run with a background or
// reverse (today's block) are kept; inside other styled runs (faint past
// days) the glyph is drawn plainly and the run's style restored after it.
// The scene sees an empty box, so it draws as if the calendar were not
// there.
func behind(sc scene.Scene, y, x0 int, seg string, p bgPaint) string {
	empty := scene.Box{Y0: -1, Y1: -1}
	colored, _ := sc.(scene.Colored)
	trueColored, _ := sc.(scene.TrueColored)
	if p.style != styleColor {
		colored, trueColored = nil, nil
	}
	var sb strings.Builder
	active, solid := "", false // SGR sequences since the last reset
	x := x0
	for i := 0; i < len(seg); {
		if seg[i] == 0x1b {
			j := i + 1
			for j < len(seg) && (seg[j] < 0x40 || seg[j] > 0x7e || seg[j] == '[') {
				j++
			}
			esc := seg[i : j+1]
			params := strings.TrimSuffix(strings.TrimPrefix(esc, "\x1b["), "m")
			if params == "" || params == "0" {
				active, solid = "", false
			} else {
				active += esc
				solid = solid || fillsCell(params)
			}
			sb.WriteString(esc)
			i = j + 1
			continue
		}
		r, size := utf8.DecodeRuneInString(seg[i:])
		i += size
		g := rune(0)
		if r == ' ' && !solid {
			g = sc.Glyph(x, y, empty)
		}
		x++
		if g == 0 {
			sb.WriteRune(r)
			continue
		}
		if active != "" {
			sb.WriteString(sgrReset)
		}
		switch {
		case p.style == stylePlain:
			sb.WriteRune(g)
		case trueColored != nil && p.mode == palette.TrueColor:
			// At 100 the background is as bright as in the margins.
			keep := keepFactor * float64(min(p.behind, 100)) / 100
			c := palette.Blend(trueColored.RGB(x-1, y, empty), p.bg, keep)
			fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm%c%s", c.R, c.G, c.B, g, fgReset)
		case trueColored != nil:
			sb.WriteString(faintOn + sgrColor(palette.Nearest(trueColored.RGB(x-1, y, empty))) + string(g) + fgReset + faintOff)
		case colored != nil:
			sb.WriteString(faintOn + sgrColor(colored.Color(x-1, y, empty)) + string(g) + fgReset + faintOff)
		default:
			sb.WriteString(faintOn + string(g) + faintOff)
		}
		if active != "" {
			sb.WriteString(sgrReset + active)
		}
	}
	return sb.String()
}

const sgrReset = "\x1b[0m"

// fillsCell reports whether SGR params set a background or reverse video.
func fillsCell(params string) bool {
	ps := strings.Split(params, ";")
	for k := 0; k < len(ps); k++ {
		switch n, _ := strconv.Atoi(ps[k]); {
		case n == 38 && k+1 < len(ps): // extended foreground: skip its arguments
			if ps[k+1] == "5" {
				k += 2
			} else {
				k += 4
			}
		case n == 7, n >= 40 && n <= 48, n >= 100 && n <= 107:
			return true
		}
	}
	return false
}

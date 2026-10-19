// Package harbor implements a quiet harbour scene for live mode: a gently
// shifting water line along the bottom of the screen, with container ships
// gliding slowly across it now and then.
package harbor

import (
	"math/rand/v2"
	"strings"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

// Height is the number of rows the scene occupies at the bottom of the
// screen: the water line plus the tallest ship above it.
const Height = shipH

// MinWidth is the narrowest screen the scene will draw on.
const MinWidth = 40

const (
	interval   = 50 * time.Millisecond
	waterEvery = 15   // ticks per column of water drift
	firstWait  = 45   // max ticks before the first ship sets sail
	minWait    = 450  // minimum ticks of empty sea between ships
	waitSpread = 1350 // extra random ticks of empty sea
	minPeriod  = 2    // fastest ship: ticks per column
	periodSpan = 3    // ship periods are minPeriod..minPeriod+periodSpan-1
	minGap     = 12   // minimum columns between a ship and its companion
	gapSpread  = 30   // extra random columns between companions
)

const (
	shipH    = 6 // rows in a ship sprite; the last row sits in the water
	deckRow  = 3 // sprite row of the deck line
	cargoBot = 2 // sprite row cargo stands on (just above the deck)
	stern    = 2 // columns of deck behind the bridge
	bow      = 4 // columns of deck ahead of the cargo
)

// bridge is the wheelhouse at the stern, bottom-aligned on cargoBot.
var bridge = []string{
	"┌┐",
	"├┤",
	"││",
}

// boxes are small ribbed containers: a single box and a stack of two,
// in two lengths. All are bottom-aligned on cargoBot.
var boxes = [][]string{
	{"┌┬┬┐", "└┴┴┘"},
	{"┌┬┬┬┐", "└┴┴┴┘"},
	{"┌┬┬┐", "├┼┼┤", "└┴┴┘"},
	{"┌┬┬┬┐", "├┼┼┼┤", "└┴┴┴┘"},
}

// bigBox is the runner scene's shipping container, carried whole.
var bigBox = []string{
	"┌────────────┐",
	"│┃┃┃┃┃┃┃┃┃┃┃┃│",
	"└────────────┘",
}

// mirror maps each glyph to its left-right reflection.
var mirror = map[rune]rune{
	'╱': '╲', '╲': '╱',
	'┌': '┐', '┐': '┌', '└': '┘', '┘': '└',
	'╭': '╮', '╮': '╭', '╰': '╯', '╯': '╰',
	'├': '┤', '┤': '├',
}

type ship struct {
	art    [][]rune        // shipH rows, all the same width; ' ' is transparent
	tint   [][]scene.Color // color of each art cell, fixed at launch
	x      int             // screen column of the left edge
	dir    int             // +1 sails right, -1 sails left
	period int             // ticks per column
	sub    int             // ticks since last move
}

func (s *ship) width() int { return len(s.art[0]) }

// Scene is the harbour. Ships move in screen space; the water drifts.
type Scene struct {
	w, h  int
	t     int // ticks elapsed
	ships []*ship
	wait  int // ticks until the next ship sets sail on an empty sea
	// pending is a companion ship waiting for room behind ships[0].
	pending    *ship
	pendingGap int
	frame      [][]rune        // Height rows × w, rebuilt every Tick
	tint       [][]scene.Color // color of each frame cell, rebuilt with it
	built      uint64          // ships built so far; seeds container colors
	rng        *rand.Rand
}

var (
	_ scene.Scene   = (*Scene)(nil)
	_ scene.Colored = (*Scene)(nil)
)

// Colors: the sea is blue, hulls and bridges keep the terminal's
// foreground, and each container is painted when its ship is built.
const (
	waterColor = scene.Blue
	hullColor  = scene.Default
	bigColor   = scene.Red // matches the runner scene's container
)

// boxColors are the paints small containers are picked from.
var boxColors = []scene.Color{scene.Red, scene.Yellow, scene.Blue, scene.Green, scene.Cyan, scene.Magenta}

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng}
	s.wait = rng.IntN(firstWait + 1)
	s.Resize(w, h)
	return s
}

// Resize changes the screen size, dropping ships that end up off screen.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.cull()
	s.frame = make([][]rune, Height)
	s.tint = make([][]scene.Color, Height)
	for i := range s.frame {
		s.frame[i] = make([]rune, s.w)
		s.tint[i] = make([]scene.Color, s.w)
	}
	s.render()
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return interval }

// Tick moves the ships and the water by one step.
func (s *Scene) Tick() {
	s.t++
	for _, sh := range s.ships {
		sh.sub++
		if sh.sub >= sh.period {
			sh.sub = 0
			sh.x += sh.dir
		}
	}
	s.cull()

	if s.pending != nil && (len(s.ships) == 0 || s.roomBehind(s.ships[0]) >= s.pendingGap) {
		s.ships = append(s.ships, s.pending)
		s.pending = nil
	}
	if len(s.ships) == 0 && s.pending == nil {
		if s.wait > 0 {
			s.wait--
		} else {
			s.launch()
			s.wait = minWait + s.rng.IntN(waitSpread)
		}
	}
	s.render()
}

// Glyph returns the rune drawn at screen (x, y), or 0 for empty. The scene
// hides entirely when its strip would touch the calendar box b.
func (s *Scene) Glyph(x, y int, b scene.Box) rune {
	if s.w < MinWidth || b.Y1 >= s.h-Height {
		return 0
	}
	row := y - (s.h - Height)
	if x < 0 || x >= s.w || row < 0 || row >= Height {
		return 0
	}
	return s.frame[row][x]
}

// Color returns the tint of the glyph at screen (x, y): Default anywhere
// nothing is drawn or off the strip.
func (s *Scene) Color(x, y int, _ scene.Box) scene.Color {
	row := y - (s.h - Height)
	if x < 0 || x >= s.w || row < 0 || row >= Height {
		return scene.Default
	}
	return s.tint[row][x]
}

// launch sends a ship onto the empty sea and, sometimes, queues a slower
// companion to follow in the same direction.
func (s *Scene) launch() {
	dir := 1
	if s.rng.IntN(2) == 0 {
		dir = -1
	}
	lead := s.newShip(dir, minPeriod+s.rng.IntN(periodSpan))
	s.ships = append(s.ships, lead)
	if s.rng.IntN(4) == 0 {
		// A companion no faster than the leader never catches it up.
		s.pending = s.newShip(dir, lead.period+s.rng.IntN(2))
		s.pendingGap = minGap + s.rng.IntN(gapSpread)
	}
}

// newShip builds a ship just off the edge it enters from.
func (s *Scene) newShip(dir, period int) *ship {
	art, tint := s.buildArt(dir)
	sh := &ship{art: art, tint: tint, dir: dir, period: period}
	if dir > 0 {
		sh.x = -sh.width()
	} else {
		sh.x = s.w
	}
	return sh
}

// roomBehind is the number of clear columns between the edge sh entered
// from and its stern.
func (s *Scene) roomBehind(sh *ship) int {
	if sh.dir > 0 {
		return sh.x
	}
	return s.w - (sh.x + sh.width())
}

// cull drops ships that have sailed fully off screen.
func (s *Scene) cull() {
	kept := s.ships[:0]
	for _, sh := range s.ships {
		gone := (sh.dir > 0 && sh.x >= s.w) || (sh.dir < 0 && sh.x+sh.width() <= 0)
		if !gone {
			kept = append(kept, sh)
		}
	}
	s.ships = kept
}

// buildArt draws a random right-facing ship and mirrors it for dir < 0,
// along with the color of each cell. Rows top to bottom: cargo and bridge,
// deck, hull side, keel in the water.
func (s *Scene) buildArt(dir int) ([][]rune, [][]scene.Color) {
	var cargo [][]string
	big := s.rng.IntN(4) == 0
	if big {
		cargo = [][]string{bigBox}
	} else {
		cargo = s.stacks()
	}

	width := stern + len([]rune(bridge[0])) + 1 + bow
	for _, c := range cargo {
		width += len([]rune(c[0])) + 1
	}

	art := make([][]rune, shipH)
	tint := make([][]scene.Color, shipH)
	for i := range art {
		art[i] = []rune(strings.Repeat(" ", width))
		tint[i] = make([]scene.Color, width) // hullColor
	}
	put := func(sprite []string, col int, c scene.Color) {
		top := cargoBot - len(sprite) + 1
		for r, line := range sprite {
			for i, ch := range []rune(line) {
				if ch != ' ' {
					art[top+r][col+i] = ch
					tint[top+r][col+i] = c
				}
			}
		}
	}

	// Hull: square stern on the left, raked bow on the right.
	last := width - 1
	art[deckRow][0], art[deckRow][last] = '┌', '┐'
	for c := 1; c < last; c++ {
		art[deckRow][c] = '─'
	}
	art[deckRow+1][0], art[deckRow+1][last-1] = '│', '╱'
	art[deckRow+2][0], art[deckRow+2][last-2] = '╰', '╱'
	for c := 1; c < last-2; c++ {
		art[deckRow+2][c] = '─'
	}

	col := stern
	put(bridge, col, hullColor)
	bw := len([]rune(bridge[0]))
	for c := col; c < col+bw; c++ {
		art[deckRow][c] = '┴'
	}
	col += bw + 1
	paint := s.painter()
	for _, c := range cargo {
		col++
		color := bigColor
		if !big {
			color = paint()
		}
		put(c, col, color)
		col += len([]rune(c[0]))
	}

	if dir < 0 {
		for r, row := range art {
			t := tint[r]
			for i, j := 0, len(row)-1; i <= j; i, j = i+1, j-1 {
				row[i], row[j] = flip(row[j]), flip(row[i])
				t[i], t[j] = t[j], t[i]
			}
		}
	}
	return art, tint
}

// painter returns a source of container colors for the next ship built.
// Colors are hashed from a build counter rather than drawn from rng, so
// the scene's random sequence (and so its animation) is unaffected.
// Neighbouring containers never share a color.
func (s *Scene) painter() func() scene.Color {
	s.built++
	h := s.built * 0x9e3779b97f4a7c15
	prev := -1
	return func() scene.Color {
		h ^= h >> 29
		h *= 0xbf58476d1ce4e5b9
		h ^= h >> 32
		i := int(h % uint64(len(boxColors)))
		if i == prev {
			i = (i + 1) % len(boxColors)
		}
		prev = i
		return boxColors[i]
	}
}

// stacks picks 1–3 small containers and arranges them as single boxes or
// stacks of two, side by side.
func (s *Scene) stacks() [][]string {
	n := 1 + s.rng.IntN(3)
	var out [][]string
	for n > 0 {
		tall := 0
		if n >= 2 && s.rng.IntN(2) == 0 {
			tall = 2
		}
		out = append(out, boxes[tall+s.rng.IntN(2)])
		if tall > 0 {
			n -= 2
		} else {
			n--
		}
	}
	return out
}

// render rebuilds the strip: water first, ships painted over it.
func (s *Scene) render() {
	shift := s.t / waterEvery
	for row := range s.frame {
		line, tint := s.frame[row], s.tint[row]
		for x := range line {
			line[x] = 0
			tint[x] = scene.Default
		}
		if row == Height-1 {
			for x := range line {
				line[x] = waterGlyph(x + shift)
				tint[x] = waterColor
			}
		}
	}
	for _, sh := range s.ships {
		for r, artRow := range sh.art {
			for c, ch := range artRow {
				x := sh.x + c
				if ch != ' ' && x >= 0 && x < s.w {
					s.frame[r][x] = ch
					s.tint[r][x] = sh.tint[r][c]
				}
			}
		}
	}
}

// waterGlyph is a mostly-calm ripple line with the odd flat patch, fixed
// per water column so the pattern drifts as a whole.
func waterGlyph(wx int) rune {
	h := uint32(wx) * 2654435761
	switch {
	case h>>24 < 40:
		return '-'
	case h>>24 < 52:
		return '_'
	}
	return '~'
}

func flip(r rune) rune {
	if m, ok := mirror[r]; ok {
		return m
	}
	return r
}

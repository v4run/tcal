// Package train implements a quiet railway scene for live mode: a rail
// line along the bottom of the screen that sits empty most of the time,
// with a freight train rolling through every few minutes.
package train

import (
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

// Height is the number of rows the scene occupies at the bottom of the
// screen: the rail line plus the tallest car (and the smoke) above it.
const Height = 1 + trainH

// MinWidth is the narrowest screen the scene will draw on.
const MinWidth = 40

const (
	interval   = 100 * time.Millisecond
	firstWait  = 30   // max ticks before the first train (3s)
	minWait    = 600  // minimum ticks between trains (1 min)
	waitSpread = 1800 // extra random ticks between trains (up to 3 more min)
	minCars    = 3    // fewest cars behind the locomotive
	carSpread  = 6    // cars are minCars..minCars+carSpread-1
	tieEvery   = 4    // columns between rail ties
	smokeEvery = 4    // ticks per smoke puff frame
	bigEvery   = 8    // about one car in this many carries the big container
)

const trainH = 4 // rows in a train sprite: smoke, then three rows of car

// loco is the right-facing locomotive; chimney is the column of its stack.
// Row 0 is reserved for smoke.
var loco = []string{
	"            ",
	"┌──┐   ┌┐   ",
	"│  ├───┴┴──╮",
	"└o─o────o─o╯",
}

const chimney = 7

// smoke is the puff over the chimney, cycling; columns are relative to the
// chimney and drift backwards (left, for a right-facing train).
var smoke = []map[int]rune{
	{0: '.', -2: '\''},
	{0: '\'', -3: '.'},
	{-1: '.', -4: '\''},
	{0: '.', -2: '.'},
}

// cars are the rolling stock; all are trainH rows and bottom-aligned. The
// last one is rarer than the rest (see bigEvery).
var cars = [][]string{
	{ // boxcar
		"        ",
		"┌──────┐",
		"│ ┃  ┃ │",
		"└o────o┘",
	},
	{ // tanker
		"   ┌┐    ",
		"╭──┴┴───╮",
		"│       │",
		"╰o─────o╯",
	},
	{ // hopper
		"        ",
		"┌──────┐",
		"╲      ╱",
		"─╲o──o╱─",
	},
	{ // flatcar with a small container
		"        ",
		"┌┬┬┬┬┬┐ ",
		"└┴┴┴┴┴┘ ",
		"═o════o═",
	},
	{ // flatcar with the runner scene's shipping container
		"┌────────────┐",
		"│┃┃┃┃┃┃┃┃┃┃┃┃│",
		"└────────────┘",
		"═o══════════o═",
	},
}

// mirror maps each glyph to its left-right reflection.
var mirror = map[rune]rune{
	'╱': '╲', '╲': '╱',
	'┌': '┐', '┐': '┌', '└': '┘', '┘': '└',
	'╭': '╮', '╮': '╭', '╰': '╯', '╯': '╰',
	'├': '┤', '┤': '├',
}

// Scene is the railway. The train moves one column per Tick.
type Scene struct {
	w, h  int
	t     int // ticks elapsed
	wait  int // ticks until the next train while the track is empty
	train [][]rune
	x     int // screen column of the train's left edge
	dir   int // +1 rolls right, -1 rolls left, 0 when no train
	chimX int // column of the chimney within train
	frame [][]rune
	rng   *rand.Rand
}

var _ scene.Scene = (*Scene)(nil)

// New returns a w×h scene that draws randomness from rng. The first train
// arrives within a few seconds.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng, wait: rng.IntN(firstWait + 1)}
	s.Resize(w, h)
	return s
}

// Resize changes the screen size. A train in progress keeps rolling.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.frame = make([][]rune, Height)
	for i := range s.frame {
		s.frame[i] = make([]rune, s.w)
	}
	s.cull()
	s.render()
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return interval }

// Tick advances the train by one column, or counts down to the next one.
func (s *Scene) Tick() {
	s.t++
	switch {
	case s.dir != 0:
		s.x += s.dir
		s.cull()
	case s.wait > 0:
		s.wait--
	default:
		s.depart()
		s.wait = minWait + s.rng.IntN(waitSpread)
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

// Passing reports whether a train is on the screen or entering it.
func (s *Scene) Passing() bool { return s.dir != 0 }

// depart assembles a random train just off one edge of the screen.
func (s *Scene) depart() {
	rows := make([]string, trainH)
	n := minCars + s.rng.IntN(carSpread)
	for range n {
		car := cars[s.rng.IntN(len(cars)-1)]
		if s.rng.IntN(bigEvery) == 0 {
			car = cars[len(cars)-1]
		}
		for r := range rows {
			gap := " "
			if r == trainH-1 {
				gap = "─" // coupler
			}
			rows[r] += car[r] + gap
		}
	}
	s.chimX = len([]rune(rows[0])) + chimney
	for r := range rows {
		rows[r] += loco[r]
	}

	s.train = make([][]rune, trainH)
	for r, line := range rows {
		s.train[r] = []rune(line)
	}
	s.dir = 1
	if s.rng.IntN(2) == 0 {
		s.dir = -1
		width := len(s.train[0])
		for _, row := range s.train {
			for i, j := 0, len(row)-1; i <= j; i, j = i+1, j-1 {
				row[i], row[j] = flip(row[j]), flip(row[i])
			}
		}
		s.chimX = width - 1 - s.chimX
	}
	if s.dir > 0 {
		s.x = -len(s.train[0])
	} else {
		s.x = s.w
	}
}

// cull clears the train once it has fully left the screen.
func (s *Scene) cull() {
	if s.dir == 0 {
		return
	}
	width := len(s.train[0])
	if (s.dir > 0 && s.x >= s.w) || (s.dir < 0 && s.x+width <= 0) {
		s.dir, s.train = 0, nil
	}
}

// render rebuilds the strip: rails, then the train and its smoke.
func (s *Scene) render() {
	for row, line := range s.frame {
		for x := range line {
			line[x] = 0
			if row == Height-1 {
				line[x] = railGlyph(x)
			}
		}
	}
	if s.dir == 0 {
		return
	}
	for r, row := range s.train {
		for c, ch := range row {
			if ch != ' ' {
				s.set(r, s.x+c, ch)
			}
		}
	}
	for dx, ch := range smoke[(s.t/smokeEvery)%len(smoke)] {
		s.set(0, s.x+s.chimX+dx*s.dir, ch)
	}
}

func (s *Scene) set(row, x int, ch rune) {
	if x >= 0 && x < s.w {
		s.frame[row][x] = ch
	}
}

// railGlyph is a double rail with a tie every few columns.
func railGlyph(x int) rune {
	if x%tieEvery == tieEvery-1 {
		return '╤'
	}
	return '═'
}

func flip(r rune) rune {
	if m, ok := mirror[r]; ok {
		return m
	}
	return r
}

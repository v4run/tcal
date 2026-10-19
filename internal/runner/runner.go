// Package runner implements a self-playing side-scroller, in the spirit of
// Chrome's offline T-rex game, used as an ambient background in live mode.
// A shipping container slides along the bottom of the screen and hops over
// bushes.
package runner

import "math/rand/v2"

// Height is the number of rows the scene occupies at the bottom of the
// screen: the ground line plus the sprite at peak jump.
const Height = 1 + spriteH + maxJump

// MinWidth is the narrowest screen the scene will draw on.
const MinWidth = 40

const (
	spriteX   = 4  // screen column of the sprite's left edge
	minGap    = 40 // minimum columns between bushes
	gapSpread = 50 // extra random columns between bushes
)

// sprite is a shipping container in oblique 3D; spaces are transparent.
var sprite = []string{
	" ┌────────────┐",
	"┌┴───────────┐│",
	"│┃┃┃┃┃┃┃┃┃┃┃┃├┘",
	"└────────────┘ ",
}

const (
	spriteH = 4  // rows in sprite
	spriteW = 15 // columns in sprite
)

// jumpArc is the sprite's height above the ground, in rows, on each frame
// of a jump.
var jumpArc = []int{1, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 1}

const maxJump = 2 // max(jumpArc)

// jumpLead is the distance from the sprite's right edge to a bush's left
// edge at takeoff. Tuned so every bush shape is cleared (see tests).
const jumpLead = 0

// bushShapes are drawn bottom-aligned on the ground; spaces are transparent.
var bushShapes = [][]string{
	{"╭─╮"},
	{" ╭╮ ", "╭╯╰╮"},
	{" ╭─╮ ", "╭╯ ╰╮"},
	{" ╭╮╭╮ ", "╭╯╰╯╰╮"},
}

type bush struct {
	x     int // world column of the left edge
	shape int // index into bushShapes
}

// Scene is the scrolling world. The camera advances one column per Tick.
type Scene struct {
	w, h   int
	offset int // world column shown at screen x=0
	bushes []bush
	nextX  int // world column of the next bush to spawn
	jump   int // index into jumpArc, or -1 when on the ground
	rng    *rand.Rand
}

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{jump: -1, rng: rng}
	s.Resize(w, h)
	s.nextX = s.w // first bush scrolls in from the right edge
	return s
}

// Resize changes the screen size without resetting the world.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
}

// Tick advances the world by one column.
func (s *Scene) Tick() {
	s.offset++

	for len(s.bushes) > 0 && s.bushes[0].x+bushWidth(s.bushes[0]) <= s.offset {
		s.bushes = s.bushes[1:]
	}
	for s.nextX < s.offset+s.w {
		s.bushes = append(s.bushes, bush{x: s.nextX, shape: s.rng.IntN(len(bushShapes))})
		s.nextX += minGap + s.rng.IntN(gapSpread)
	}

	if s.jump >= 0 {
		s.jump++
		if s.jump == len(jumpArc) {
			s.jump = -1
		}
	}
	if s.jump < 0 {
		front := s.offset + spriteX + spriteW
		for _, b := range s.bushes {
			if b.x-front == jumpLead {
				s.jump = 0
				break
			}
		}
	}
}

// Glyph returns the rune drawn at screen (x, y), or 0 for empty.
func (s *Scene) Glyph(x, y int) rune {
	if x < 0 || x >= s.w || y < s.h-Height || y >= s.h {
		return 0
	}
	if y == s.h-1 {
		return groundGlyph(s.offset + x)
	}
	if r := s.spriteGlyph(x, y); r != 0 {
		return r
	}
	return s.bushGlyph(x, y)
}

func (s *Scene) lift() int {
	if s.jump < 0 {
		return 0
	}
	return jumpArc[s.jump]
}

func (s *Scene) spriteGlyph(x, y int) rune {
	top := s.h - 1 - spriteH - s.lift()
	return cell(sprite, y-top, x-spriteX)
}

func (s *Scene) bushGlyph(x, y int) rune {
	wx := s.offset + x
	for _, b := range s.bushes {
		shape := bushShapes[b.shape]
		if r := cell(shape, y-(s.h-1-len(shape)), wx-b.x); r != 0 {
			return r
		}
	}
	return 0
}

// collides reports whether any sprite cell overlaps a bush cell.
func (s *Scene) collides() bool {
	for y := s.h - Height; y < s.h-1; y++ {
		for x := spriteX; x < spriteX+spriteW; x++ {
			if s.spriteGlyph(x, y) != 0 && s.bushGlyph(x, y) != 0 {
				return true
			}
		}
	}
	return false
}

// cell returns the rune at (row, col) of art, treating spaces as empty.
func cell(art []string, row, col int) rune {
	if row < 0 || row >= len(art) || col < 0 {
		return 0
	}
	rs := []rune(art[row])
	if col >= len(rs) || rs[col] == ' ' {
		return 0
	}
	return rs[col]
}

func bushWidth(b bush) int {
	shape := bushShapes[b.shape]
	return len([]rune(shape[len(shape)-1]))
}

// groundGlyph is a flat line with the occasional pebble, fixed per world column.
func groundGlyph(wx int) rune {
	h := uint32(wx) * 2654435761
	if h>>24 < 16 {
		return '▀'
	}
	return '▔'
}

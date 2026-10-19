// Package dino implements a self-playing side-scroller, in the spirit of
// Chrome's offline T-rex game, used as an ambient background in live mode.
// A dino runs along the bottom of the screen and jumps over bushes.
package dino

import "math/rand/v2"

// Height is the number of rows the scene occupies at the bottom of the screen.
const Height = 1 + spriteRows + maxJump

// MinWidth is the narrowest screen the scene will draw on.
const MinWidth = 30

const (
	dinoX      = 4  // screen column of the sprite's left edge
	spriteRows = 3  // sprite height
	maxJump    = 3  // peak jump height, in rows
	minGap     = 22 // minimum columns between bushes
	gapSpread  = 40 // extra random columns between bushes
)

// Two running frames; legs alternate. Spaces are transparent.
var sprites = [2][spriteRows]string{
	{
		"    ▄█▀█▄",
		"▚▄▄███▀▀ ",
		"  ▜▘ ▀▌  ",
	},
	{
		"    ▄█▀█▄",
		"▚▄▄███▀▀ ",
		"  ▀▌ ▜▘  ",
	},
}

// jumpArc is the dino's height above the ground on each frame of a jump.
var jumpArc = []int{1, 2, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 2, 1}

// jumpLead is how many columns ahead of the sprite a bush must be for the
// dino to take off. Tuned so every bush shape is cleared (see tests).
const jumpLead = 1

// bushShapes are drawn bottom-aligned on the ground; spaces are transparent.
var bushShapes = [][]string{
	{"▟▙"},
	{"▟█▙"},
	{" ▄ ", "▟█▙"},
	{" ▄▖ ", "▟██▙"},
}

type bush struct {
	x     int // world column of the left edge
	shape int // index into bushShapes
}

// Scene is the scrolling world. The camera advances one column per Tick.
type Scene struct {
	w, h    int
	offset  int // world column shown at screen x=0
	bushes  []bush
	nextX   int // world column of the next bush to spawn
	jump    int // index into jumpArc, or -1 when on the ground
	running int // frame counter for the leg animation
	rng     *rand.Rand
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
	s.running++

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
		front := s.offset + dinoX + spriteWidth()
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
	ground := s.h - 1
	if y == ground {
		return groundGlyph(s.offset + x)
	}
	if r := s.dinoGlyph(x, y); r != 0 {
		return r
	}
	return s.bushGlyph(x, y)
}

func (s *Scene) height() int {
	if s.jump < 0 {
		return 0
	}
	return jumpArc[s.jump]
}

func (s *Scene) dinoGlyph(x, y int) rune {
	top := s.h - 1 - spriteRows - s.height()
	row, col := y-top, x-dinoX
	if row < 0 || row >= spriteRows || col < 0 {
		return 0
	}
	frame := (s.running / 2) % 2
	if s.jump >= 0 {
		frame = 0 // legs still mid-air
	}
	return cell(sprites[frame][row], col)
}

func (s *Scene) bushGlyph(x, y int) rune {
	wx := s.offset + x
	for _, b := range s.bushes {
		col := wx - b.x
		if col < 0 || col >= bushWidth(b) {
			continue
		}
		shape := bushShapes[b.shape]
		row := y - (s.h - 1 - len(shape))
		if row < 0 || row >= len(shape) {
			continue
		}
		return cell(shape[row], col)
	}
	return 0
}

// collides reports whether any dino cell overlaps a bush cell.
func (s *Scene) collides() bool {
	for y := s.h - Height; y < s.h-1; y++ {
		for x := dinoX; x < dinoX+spriteWidth(); x++ {
			if s.dinoGlyph(x, y) != 0 && s.bushGlyph(x, y) != 0 {
				return true
			}
		}
	}
	return false
}

// cell returns the rune at column col of row, treating spaces as empty.
func cell(row string, col int) rune {
	rs := []rune(row)
	if col < 0 || col >= len(rs) || rs[col] == ' ' {
		return 0
	}
	return rs[col]
}

func spriteWidth() int { return len([]rune(sprites[0][0])) }

func bushWidth(b bush) int { return len([]rune(bushShapes[b.shape][len(bushShapes[b.shape])-1])) }

// groundGlyph is a flat line with the occasional pebble, fixed per world column.
func groundGlyph(wx int) rune {
	h := uint32(wx) * 2654435761
	if h>>24 < 12 {
		return '▂'
	}
	return '▁'
}

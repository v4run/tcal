// Package dino implements a self-playing side-scroller, in the spirit of
// Chrome's offline T-rex game, used as an ambient background in live mode.
// A dino runs along the bottom of the screen and jumps over bushes.
//
// The scene is drawn on a pixel canvas where each terminal cell holds two
// vertically stacked pixels (rendered with ▀ ▄ █), which keeps pixels
// roughly square and lets jumps move half a row at a time.
package dino

import "math/rand/v2"

// Height is the number of rows the scene occupies at the bottom of the
// screen: the ground line plus enough canvas for the sprite at peak jump.
const Height = 1 + (spriteH+maxJump+1)/2

// MinWidth is the narrowest screen the scene will draw on.
const MinWidth = 40

const (
	dinoX     = 4  // screen column of the sprite's left edge
	minGap    = 40 // minimum columns between bushes
	gapSpread = 50 // extra random columns between bushes
)

// Two running frames, as pixel art: '#' is solid, anything else is empty.
var sprites = [2][]string{
	{
		"........######",
		".......##.####",
		".......#######",
		"#......####...",
		"#.....######..",
		"##...####..#..",
		"#########.....",
		".#######......",
		"..##..#.......",
		"..#...##......",
	},
	{
		"........######",
		".......##.####",
		".......#######",
		"#......####...",
		"#.....######..",
		"##...####..#..",
		"#########.....",
		".#######......",
		"..##..##......",
		"..##...#......",
	},
}

const (
	spriteH = 10 // rows in each sprite frame
	spriteW = 14 // columns in each sprite frame
)

// jumpArc is the dino's height above the ground, in pixels, on each frame
// of a jump. Heights are even so the sprite moves in whole rows; odd
// heights re-split the art across cells and look garbled.
var jumpArc = []int{2, 4, 4, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 4, 4, 2}

const maxJump = 6 // max(jumpArc)

// jumpLead is the distance from the sprite's right edge to a bush's left
// edge at takeoff. It is negative because the head sits high: the bush may
// already be under it. Tuned so every bush shape is cleared (see tests).
const jumpLead = -4

// bushShapes are pixel art drawn bottom-aligned on the ground.
var bushShapes = [][]string{
	{
		".##.",
		"####",
		"####",
	},
	{
		"..##..",
		".####.",
		"######",
		"######",
	},
	{
		".##.###.",
		"########",
		"########",
	},
	{
		"...##...",
		".######.",
		"########",
		"########",
	},
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
		front := s.offset + dinoX + spriteW
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
	// Canvas rows count up from the ground: row s.h-2 holds pixel
	// heights 1 (top half) and 0 (bottom half).
	ph := 2 * (s.h - 2 - y)
	return halfBlock(s.pixel(x, ph+1), s.pixel(x, ph))
}

// pixel reports whether the pixel in screen column x, ph pixels above the
// ground, is solid.
func (s *Scene) pixel(x, ph int) bool {
	return s.dinoPixel(x, ph) || s.bushPixel(x, ph)
}

func (s *Scene) lift() int {
	if s.jump < 0 {
		return 0
	}
	return jumpArc[s.jump]
}

func (s *Scene) dinoPixel(x, ph int) bool {
	frame := (s.running / 3) % 2
	if s.jump >= 0 {
		frame = 0 // legs still mid-air
	}
	row := spriteH - 1 - (ph - s.lift())
	return solid(sprites[frame], row, x-dinoX)
}

func (s *Scene) bushPixel(x, ph int) bool {
	wx := s.offset + x
	for _, b := range s.bushes {
		shape := bushShapes[b.shape]
		if solid(shape, len(shape)-1-ph, wx-b.x) {
			return true
		}
	}
	return false
}

// collides reports whether any dino pixel overlaps a bush pixel.
func (s *Scene) collides() bool {
	for ph := 0; ph < spriteH+maxJump; ph++ {
		for x := dinoX; x < dinoX+spriteW; x++ {
			if s.dinoPixel(x, ph) && s.bushPixel(x, ph) {
				return true
			}
		}
	}
	return false
}

func solid(art []string, row, col int) bool {
	return row >= 0 && row < len(art) && col >= 0 && col < len(art[row]) && art[row][col] == '#'
}

func halfBlock(top, bottom bool) rune {
	switch {
	case top && bottom:
		return '█'
	case top:
		return '▀'
	case bottom:
		return '▄'
	}
	return 0
}

func bushWidth(b bush) int { return len(bushShapes[b.shape][0]) }

// groundGlyph is a flat line with the occasional pebble, fixed per world column.
func groundGlyph(wx int) rune {
	h := uint32(wx) * 2654435761
	if h>>24 < 16 {
		return '▀'
	}
	return '▔'
}

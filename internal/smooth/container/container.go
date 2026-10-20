// Package container is the smooth side-scroller: a ribbed shipping
// container, drawn in braille dots, slides along the bottom of the screen
// and hops over bushes on a parabola.
package container

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/v4run/tcal/internal/dots"
	"github.com/v4run/tcal/internal/scene"
)

// Height is the number of rows the strip occupies at the bottom of the
// screen, the same as classic: the ground plus the container at peak jump.
const Height = 6

// MinWidth is the narrowest screen the scene will draw on.
const MinWidth = 40

const step = 33 * time.Millisecond

// dotSpeed is the scroll speed in dots per second: classic moves one
// column (two dots) every 40 ms.
const dotSpeed = 2 / 0.04

// Gaps between bushes, in columns, as classic.
const (
	minGap    = 40
	gapSpread = 50
)

// Strip geometry in dots (Height*4 = 24 rows). The container rests with its
// top on restTop and its bottom just above the ground on groundRow.
const (
	spriteX   = 8  // dot column of the container's left edge (classic: 4 columns)
	spriteW   = 25 // odd, so both end walls carry a rib
	spriteH   = 10
	groundRow = 20
	restTop   = groundRow - spriteH
)

// The jump: jumpH dots high at its peak, jumpDur seconds long (classic:
// 0.88 s). Bushes are at most 5 dots tall and 10 wide, so the container is
// above every bush for 0.71*jumpDur = 0.71 s, longer than the 0.68 s the
// widest bush takes to pass under it.
const (
	jumpH   = restTop // the peak touches the strip's top
	jumpDur = 1.0
)

// bushShapes are drawn bottom-aligned on the ground; '#' is a dot.
var bushShapes = [][]string{
	{
		".####.",
		"######",
		"######",
	},
	{
		"..##....",
		".######.",
		"########",
		"########",
	},
	{
		"..##..##..",
		".########.",
		"##########",
		"##########",
		"##########",
	},
	{
		"...##...",
		"..####..",
		".######.",
		"########",
		"########",
	},
}

var (
	containerCol = scene.RGB{R: 175, G: 80, B: 65}
	bushCol      = scene.RGB{R: 95, G: 140, B: 85}
	groundCol    = scene.RGB{R: 140, G: 130, B: 110}
)

var (
	_ scene.Scene       = (*Scene)(nil)
	_ scene.Stepper     = (*Scene)(nil)
	_ scene.TrueColored = (*Scene)(nil)
)

type bush struct {
	x     int // world dot column of the left edge
	shape int // index into bushShapes
}

// Scene is the scrolling world.
type Scene struct {
	rng    *rand.Rand
	w, h   int
	offset float64 // world dot column shown at screen dot 0
	bushes []bush  // sorted by x
	nextX  int     // world dot column of the next bush to spawn
	jumpT  float64 // seconds into the current jump, or -1 on the ground
	jumped int     // x of the last bush jumped
	canvas dots.Canvas
}

// New returns a w×h scene that draws randomness from rng.
func New(w, h int, rng *rand.Rand) *Scene {
	s := &Scene{rng: rng, jumpT: -1, jumped: math.MinInt}
	s.Resize(w, h)
	s.nextX = s.w * 2 // the first bush scrolls in from the right edge
	return s
}

// Interval is the delay between Ticks.
func (s *Scene) Interval() time.Duration { return step }

// Tick advances one interval.
func (s *Scene) Tick() { s.Step(step) }

// Resize changes the screen size without resetting the world.
func (s *Scene) Resize(w, h int) {
	s.w, s.h = max(w, 0), max(h, 0)
	s.canvas.Reset(s.w, Height)
	s.render()
}

// Step scrolls the world by dt and flies or lands the container.
func (s *Scene) Step(dt time.Duration) {
	sec := max(dt, 0).Seconds()
	s.offset += dotSpeed * sec

	for len(s.bushes) > 0 && float64(s.bushes[0].x+bushWidth(s.bushes[0].shape)) <= s.offset {
		s.bushes = s.bushes[1:]
	}
	for float64(s.nextX) < s.offset+float64(s.w*2) {
		s.bushes = append(s.bushes, bush{x: s.nextX, shape: s.rng.IntN(len(bushShapes))})
		s.nextX += 2 * (minGap + s.rng.IntN(gapSpread))
	}

	if s.jumpT >= 0 {
		if s.jumpT += sec; s.jumpT >= jumpDur {
			s.jumpT = -1
		}
	}
	if s.jumpT < 0 {
		s.takeoff()
	}
	s.render()
}

// takeoff starts a jump when the next bush is lead(shape) dots ahead of the
// container's front. A frame that lands past that point starts the jump as
// far along as the overshoot, so the arc stays centred on the bush.
func (s *Scene) takeoff() {
	front := float64(spriteX + spriteW)
	for _, b := range s.bushes {
		if b.x <= s.jumped {
			continue
		}
		dist := float64(b.x) - s.offset - front
		l := lead(b.shape)
		if dist > l {
			return
		}
		s.jumped = b.x
		if late := (l - dist) / dotSpeed; late < jumpDur {
			s.jumpT = late
			return
		}
	}
}

// lead is how far ahead of the container's front a bush's left edge is at
// takeoff, in dots: the apex then falls with the container centred over
// the bush.
func lead(shape int) float64 {
	return dotSpeed*jumpDur/2 - float64(spriteW+bushWidth(shape))/2
}

// Glyph returns the rune at screen (x, y), or 0 for empty. The scene hides
// whole when its strip would touch the calendar box b.
func (s *Scene) Glyph(x, y int, b scene.Box) rune {
	if s.w < MinWidth || b.Y1 >= s.h-Height {
		return 0
	}
	return s.canvas.Glyph(x, y-(s.h-Height))
}

// RGB returns the colour of the glyph at (x, y).
func (s *Scene) RGB(x, y int, _ scene.Box) scene.RGB {
	return s.canvas.RGB(x, y-(s.h-Height))
}

// lift is the container's height above its resting place, in dots.
func (s *Scene) lift() float64 {
	if s.jumpT < 0 {
		return 0
	}
	tau := s.jumpT / jumpDur
	return 4 * jumpH * tau * (1 - tau)
}

// top is the strip dot row of the container's top.
func (s *Scene) top() int { return int(math.Floor(restTop - s.lift())) }

// dot plots strip dot (dx, dy).
func (s *Scene) dot(dx, dy int, col scene.RGB) {
	s.canvas.Dot(float64(dx)/2, float64(dy)/4, col)
}

// bushLeft is the screen dot column of a bush's left edge.
func (s *Scene) bushLeft(b bush) int { return int(math.Floor(float64(b.x) - s.offset)) }

func bushWidth(shape int) int {
	sh := bushShapes[shape]
	return len(sh[len(sh)-1])
}

func (s *Scene) render() {
	s.canvas.Clear()
	if s.w == 0 {
		return
	}
	// Ground: a line with pebbles fixed to the world.
	gy := float64(groundRow) / 4
	for dx := range s.w * 2 {
		s.canvas.Dot(float64(dx)/2, gy, groundCol)
	}
	first := int(math.Floor(s.offset))
	for wx := first; wx <= first+s.w*2; wx++ {
		if h := hash(wx); h%11 == 0 {
			s.canvas.Dot((float64(wx)-s.offset)/2, float64(groundRow+1+int(h>>8)%2)/4, groundCol)
		}
	}

	// Bushes and the container are placed on whole dots, the same ones
	// the collision tests check.
	for _, b := range s.bushes {
		sh := bushShapes[b.shape]
		left := s.bushLeft(b)
		for dy, row := range sh {
			for dx, ch := range row {
				if ch == '#' {
					s.dot(left+dx, groundRow-len(sh)+dy, bushCol)
				}
			}
		}
	}

	// The container: solid outline with a rib on every other dot column.
	top := s.top()
	for dy := range spriteH {
		for dx := range spriteW {
			if dy == 0 || dy == spriteH-1 || dx%2 == 0 {
				s.dot(spriteX+dx, top+dy, containerCol)
			}
		}
	}
}

// hash scrambles a world column into a pseudo-random value.
func hash(a int) uint32 {
	k := uint32(a) * 0x9E3779B1
	k ^= k >> 15
	k *= 0x2C1B3C6D
	k ^= k >> 12
	return k
}

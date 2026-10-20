package container

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/scene"
)

const dt = 33 * time.Millisecond

var noBox = scene.Box{Y0: -1, Y1: -1}

func newTest(w, h int, seed uint64) *Scene {
	return New(w, h, rand.New(rand.NewPCG(seed, seed^0x9e37)))
}

func lowBox(h int) scene.Box { return scene.Box{X0: 20, X1: 60, Y0: 1, Y1: h - Height - 1} }

// lit reports whether strip dot (dx, dy) is plotted.
func lit(s *Scene, dx, dy int) bool {
	g := s.canvas.Glyph(dx/2, dy/4)
	if g < 0x2800 || g > 0x28ff {
		return false
	}
	return (g-0x2800)&rune([4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}[dy&3][dx&1]) != 0
}

// collides reports whether any bush dot lies inside the container's box,
// at dot resolution, as drawn.
func (s *Scene) collides() bool {
	top := s.top()
	for _, b := range s.bushes {
		left := s.bushLeft(b)
		sh := bushShapes[b.shape]
		for dy, row := range sh {
			for dx, ch := range row {
				if ch != '#' {
					continue
				}
				x, y := left+dx, groundRow-len(sh)+dy
				if x >= spriteX && x < spriteX+spriteW && y >= top && y < top+spriteH {
					return true
				}
			}
		}
	}
	return false
}

// Every bush shape is cleared, whatever the frame rate.
func TestContainer_JumpClearsBushes(t *testing.T) {
	for _, step := range []time.Duration{16 * time.Millisecond, dt, 50 * time.Millisecond, 100 * time.Millisecond} {
		for shape := range bushShapes {
			s := newTest(80, 20, 1)
			s.nextX = math.MaxInt / 2 // no random spawns
			s.bushes = []bush{{x: 120, shape: shape}}
			jumped, cleared := false, false
			for i := 0; i < 400; i++ {
				s.Step(step)
				jumped = jumped || s.jumpT >= 0
				if s.collides() {
					t.Fatalf("step %v shape %d: container hit bush at step %d", step, shape, i)
				}
				if len(s.bushes) == 0 {
					cleared = true
					break
				}
			}
			if !jumped || !cleared {
				t.Fatalf("step %v shape %d: jumped %v, cleared %v", step, shape, jumped, cleared)
			}
		}
	}
}

func TestContainer_NeverCollidesOverLongRun(t *testing.T) {
	for seed := uint64(0); seed < 20; seed++ {
		s := newTest(120, 30, seed)
		jumps := 0
		for i := 0; i < 5000; i++ {
			was := s.jumpT >= 0
			s.Step(dt)
			if !was && s.jumpT >= 0 {
				jumps++
			}
			if s.collides() {
				t.Fatalf("seed %d: collision at step %d", seed, i)
			}
		}
		if jumps < 10 {
			t.Fatalf("seed %d: only %d jumps", seed, jumps)
		}
	}
}

// The narrowest gap leaves room to land before the next takeoff.
func TestContainer_MinGapLeavesRoomToLand(t *testing.T) {
	maxLead := 0.0
	for i := range bushShapes {
		maxLead = max(maxLead, lead(i))
	}
	if need := dotSpeed*jumpDur + maxLead; float64(minGap*2) < need {
		t.Fatalf("minGap %d dots < jump span %.1f", minGap*2, need)
	}
}

// Bushes glide left at classic's 25 columns per second; the container's
// jump follows a parabola at dot resolution.
func TestContainer_Glides(t *testing.T) {
	s := newTest(120, 30, 2)
	s.nextX = math.MaxInt / 2
	const bx = 200
	s.bushes = []bush{{x: bx, shape: 0}}
	s.Step(0)
	p0 := float64(bx) - s.offset
	for i := 1; i <= 100; i++ {
		s.Step(dt)
		tt := float64(i) * dt.Seconds()
		want := int(math.Floor(p0 - dotSpeed*tt))
		if got := s.bushLeft(s.bushes[0]); got < want-1 || got > want+1 {
			t.Fatalf("step %d: bush at %d, ideal %d", i, got, want)
		}
		// The drawn bush matches its position.
		sh := bushShapes[0]
		row := groundRow - len(sh) + len(sh) - 1
		if got := s.bushLeft(s.bushes[0]); got >= 0 && got < 240 && !lit(s, got, row) {
			t.Fatalf("step %d: bush dot (%d,%d) not drawn", i, got, row)
		}
	}

	s = newTest(120, 30, 3)
	s.nextX = math.MaxInt / 2
	s.bushes = nil
	s.jumpT = 0
	s.Step(0)
	for i := 1; s.jumpT >= 0; i++ {
		s.Step(dt)
		tau := float64(i) * dt.Seconds() / jumpDur
		if tau >= 1 {
			break
		}
		want := int(math.Floor(restTop - 4*jumpH*tau*(1-tau)))
		got := -1
		for dy := 0; dy < Height*4; dy++ {
			if lit(s, spriteX, dy) {
				got = dy
				break
			}
		}
		if d := got - want; d < -1 || d > 1 {
			t.Fatalf("step %d: container top %d, ideal %d", i, got, want)
		}
	}
	if s.top() != restTop {
		t.Fatalf("container did not land: top %d", s.top())
	}
}

func TestContainer_HidesWhenNoRoom(t *testing.T) {
	const w, h = 100, 30
	s := newTest(w, h, 4)
	s.Step(dt)
	touching := scene.Box{X0: 20, X1: 60, Y0: 2, Y1: h - Height}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if r := s.Glyph(x, y, touching); r != 0 {
				t.Fatalf("drew %q at (%d,%d) with the box in the strip", r, x, y)
			}
		}
	}
	n := newTest(MinWidth-1, 30, 3)
	n.Step(dt)
	for y := 0; y < 30; y++ {
		for x := 0; x < MinWidth-1; x++ {
			if r := n.Glyph(x, y, noBox); r != 0 {
				t.Fatalf("drew %q on a narrow screen", r)
			}
		}
	}
}

func TestContainer_StripGroundAndPalette(t *testing.T) {
	const w, h = 100, 30
	s := newTest(w, h, 5)
	s.nextX = math.MaxInt / 2
	s.bushes = []bush{{x: int(s.offset) + 100, shape: 3}}
	s.Step(dt)
	b := lowBox(h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := s.Glyph(x, y, b)
			if r == 0 {
				if y == h-1 {
					t.Fatalf("no ground at column %d", x)
				}
				continue
			}
			if y < h-Height {
				t.Fatalf("glyph %q above the strip at (%d,%d)", r, x, y)
			}
			if ansi.StringWidth(string(r)) != 1 {
				t.Fatalf("glyph %q not single width", r)
			}
		}
	}
	if c := s.RGB(0, h-1, b); c != groundCol {
		t.Errorf("ground is %v, want %v", c, groundCol)
	}
	if c := s.RGB(spriteX/2+1, h-2, b); c != containerCol {
		t.Errorf("container is %v, want %v", c, containerCol)
	}
	if c := s.RGB(s.bushLeft(s.bushes[0])/2+1, h-2, b); c != bushCol {
		t.Errorf("bush is %v, want %v", c, bushCol)
	}
}

func TestContainer_ResizeAnySize(t *testing.T) {
	s := newTest(0, 0, 5)
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {3, 2}, {200, 60}, {10, 3}, {0, 50}, {-1, -1}, {80, 24}} {
		s.Resize(sz[0], sz[1])
		for range 50 {
			s.Step(100 * time.Millisecond)
		}
		for y := -1; y <= sz[1]; y++ {
			for x := -1; x <= sz[0]; x++ {
				s.Glyph(x, y, noBox)
				s.RGB(x, y, noBox)
			}
		}
	}
}

func TestContainer_Interface(t *testing.T) {
	s := newTest(80, 24, 6)
	if s.Interval() != dt {
		t.Fatal("interval")
	}
	s.Tick()
	s.Step(-time.Second)
}

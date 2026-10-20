package train

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

func leftmost(s *Scene, dy int) int {
	for dx := 0; dx < s.w*2; dx++ {
		if lit(s, dx, dy) {
			return dx
		}
	}
	return -1
}

// artLeft is the smallest dot offset of the train art in strip dot row dy.
func artLeft(s *Scene, dy int) int {
	m := math.MaxInt
	for _, p := range s.train {
		if p.dy == dy {
			m = min(m, p.dx)
		}
	}
	return m
}

func TestTrain_Glides(t *testing.T) {
	const x0 = 40.0
	const row = 12 // the frame under the car bodies
	for _, dir := range []int{1, -1} {
		s := newTest(160, 30, 1)
		s.depart(dir)
		s.wait = math.Inf(1)
		s.puffs, s.puffEvery = nil, math.Inf(1)
		s.x = x0
		s.Step(0)
		off := artLeft(s, row)
		for i := 1; i <= 100; i++ {
			s.Step(dt)
			tt := float64(i) * dt.Seconds()
			want := int(math.Floor(2*(x0+float64(dir)*speed*tt))) + off
			got := leftmost(s, row)
			if d := got - want; d < -1 || d > 1 {
				t.Fatalf("dir %d step %d: train starts at dot %d, ideal %d", dir, i, got, want)
			}
		}
	}
}

// Classic rolls one column per 100 ms tick.
func TestTrain_SpeedMatchesClassic(t *testing.T) {
	if speed != 10 {
		t.Fatalf("speed %v cols/s, want 10", speed)
	}
}

// Smoke puffs rise from the chimney, fade as they age and stay on the
// smoke row.
func TestTrain_SmokeRisesAndFades(t *testing.T) {
	s := newTest(160, 30, 2)
	s.depart(1)
	s.wait = math.Inf(1)
	s.x = 40
	for range 30 {
		s.Step(dt)
	}
	if len(s.puffs) == 0 {
		t.Fatal("no smoke")
	}
	type seen struct{ y, age float64 }
	before := map[*puff]seen{}
	for _, p := range s.puffs {
		before[p] = seen{p.y, p.age}
		if p.y < 0 || p.y >= 1 {
			t.Fatalf("puff at row %v, want the smoke row", p.y)
		}
	}
	s.Step(dt)
	for _, p := range s.puffs {
		if b, ok := before[p]; ok && p.y >= b.y {
			t.Fatalf("puff did not rise: %v -> %v", b.y, p.y)
		}
	}
	young, old := smokeColour(0), smokeColour(1)
	lum := func(c scene.RGB) int { return int(c.R) + int(c.G) + int(c.B) }
	if lum(old) >= lum(young) {
		t.Fatalf("smoke does not fade: %v -> %v", young, old)
	}
}

// A train arrives within 3 s, rolls off, and the track then waits 60-240 s.
func TestTrain_PassesAndTrackEmpties(t *testing.T) {
	const w, h = 80, 24
	for seed := uint64(0); seed < 10; seed++ {
		s := newTest(w, h, seed)
		n := 0
		for !s.Passing() {
			s.Step(dt)
			n++
		}
		if el := time.Duration(n) * dt; el > firstWait+dt {
			t.Fatalf("seed %d: first train after %v", seed, el)
		}
		if s.wait < minWait.Seconds() || s.wait >= (minWait+waitSpread).Seconds() {
			t.Fatalf("seed %d: next wait %v s", seed, s.wait)
		}
		sawBody := false
		for s.Passing() {
			s.Step(dt)
			for x := 0; x < w; x++ {
				if s.Glyph(x, h-3, lowBox(h)) != 0 {
					sawBody = true
				}
			}
		}
		if !sawBody {
			t.Fatalf("seed %d: train never drawn", seed)
		}
		for range 10 {
			s.Step(dt)
		}
		for y := h - Height; y < h-1; y++ {
			for x := 0; x < w; x++ {
				if r := s.Glyph(x, y, lowBox(h)); r != 0 && len(s.puffs) == 0 {
					t.Fatalf("seed %d: %q left at (%d,%d) after the train", seed, r, x, y)
				}
			}
		}
		for x := 0; x < w; x++ {
			if s.Glyph(x, h-1, lowBox(h)) == 0 {
				t.Fatalf("seed %d: no rail at column %d", seed, x)
			}
		}
	}
}

func TestTrain_HidesWhenNoRoom(t *testing.T) {
	const w, h = 100, 30
	s := newTest(w, h, 3)
	s.depart(1)
	s.x = 30
	s.Step(dt)
	touching := scene.Box{X0: 20, X1: 60, Y0: 2, Y1: h - Height}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if r := s.Glyph(x, y, touching); r != 0 {
				t.Fatalf("drew %q at (%d,%d) with the box in the strip", r, x, y)
			}
		}
	}
	if s.Glyph(31, h-3, lowBox(h)) == 0 {
		t.Fatal("train not drawn with room")
	}
	n := newTest(MinWidth-1, 30, 3)
	n.Step(dt)
	for x := 0; x < MinWidth-1; x++ {
		if r := n.Glyph(x, 29, noBox); r != 0 {
			t.Fatalf("drew %q on a narrow screen", r)
		}
	}
}

func TestTrain_StripAndGlyphs(t *testing.T) {
	const w, h = 160, 30
	s := newTest(w, h, 4)
	for seed := range 20 {
		s.depart(1 - 2*(seed%2))
		s.x = 10
		s.Step(dt)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r := s.Glyph(x, y, lowBox(h))
				if r == 0 {
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
	}
	if c := s.RGB(0, h-1, lowBox(h)); c != railCol {
		t.Errorf("rail is %v, want %v", c, railCol)
	}
}

func TestTrain_ResizeAnySize(t *testing.T) {
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

func TestTrain_Interface(t *testing.T) {
	s := newTest(80, 24, 6)
	if s.Interval() != dt {
		t.Fatal("interval")
	}
	s.Tick()
	s.Step(-time.Second)
}

package aquarium

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

// lit reports whether dot (dx, dy) is plotted.
func lit(s *Scene, dx, dy int) bool {
	g := s.canvas.Glyph(dx/2, dy/4)
	if g < 0x2800 || g > 0x28ff {
		return false
	}
	return (g-0x2800)&rune([4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}[dy&3][dx&1]) != 0
}

// litDots returns every plotted dot.
func litDots(s *Scene) [][2]int {
	var out [][2]int
	for dy := 0; dy < s.h*4; dy++ {
		for dx := 0; dx < s.w*2; dx++ {
			if lit(s, dx, dy) {
				out = append(out, [2]int{dx, dy})
			}
		}
	}
	return out
}

// lone empties the tank but for one fish, with no bubbles, weeds or turns.
func (s *Scene) lone(f fish) {
	s.fish = []fish{f}
	s.bubbles, s.weeds = nil, nil
	s.bubbleRate, s.turnRate = 0, 0
}

func TestAquarium_FishGlide(t *testing.T) {
	for _, dir := range []int{1, -1} {
		s := newTest(120, 40, 1)
		const x0, base, v = 50.25, 12.0, 2.5
		f := fish{x: x0, base: base, dir: dir, speed: v, shape: 3, period: 4, phase: 0.3}
		s.lone(f)
		s.t = 0
		s.Step(0)
		for i := 1; i <= 100; i++ {
			s.Step(dt)
			tt := float64(i) * dt.Seconds()
			wantX := int(math.Floor(2 * (x0 + float64(dir)*v*tt)))
			wantY := int(math.Floor(4 * (base + bobAmp*math.Sin(2*math.Pi*tt/f.period+f.phase))))
			ds := litDots(s)
			gx, gy := math.MaxInt, math.MaxInt
			for _, d := range ds {
				gx, gy = min(gx, d[0]), min(gy, d[1])
			}
			if d := gx - wantX; d < -1 || d > 1 {
				t.Fatalf("dir %d step %d: fish left at dot %d, ideal %d", dir, i, gx, wantX)
			}
			if d := gy - wantY; d < -1 || d > 1 {
				t.Fatalf("dir %d step %d: fish top at dot %d, ideal %d", dir, i, gy, wantY)
			}
		}
	}
}

// A turning fish is drawn as the mirror image of itself, in place.
func TestAquarium_FishTurnMirrored(t *testing.T) {
	for shape := range shapes {
		s := newTest(120, 40, 2)
		s.lone(fish{x: 30.25, base: 10, dir: 1, speed: 2, shape: shape, period: 5})
		s.Step(0)
		right := litDots(s)
		x0 := int(math.Floor(2 * s.fish[0].x))
		s.fish[0].turn()
		if s.fish[0].x != 30.25 {
			t.Fatalf("shape %d: turning moved the fish to %v", shape, s.fish[0].x)
		}
		s.Step(0)
		left := map[[2]int]bool{}
		for _, d := range litDots(s) {
			left[d] = true
		}
		w := shapes[shape].w
		if len(left) != len(right) || len(right) == 0 {
			t.Fatalf("shape %d: %d dots facing right, %d facing left", shape, len(right), len(left))
		}
		symmetric := true
		for _, d := range right {
			m := [2]int{x0 + w - 1 - (d[0] - x0), d[1]}
			if !left[m] {
				t.Fatalf("shape %d: mirror of %v (%v) not drawn", shape, d, m)
			}
			symmetric = symmetric && left[d]
		}
		if symmetric {
			t.Fatalf("shape %d: fish looks the same both ways", shape)
		}
	}
	// Fish turn at random, in place: a high rate turns a lone fish within
	// one frame, and the classic rate turns some fish within a few minutes.
	s := newTest(120, 40, 3)
	s.lone(fish{x: 30.25, base: 10, dir: 1, speed: 2, period: 5})
	s.turnRate = 1e6
	s.Step(dt)
	if f := s.fish[0]; f.dir != -1 || math.Abs(f.x-(30.25+2*dt.Seconds())) > 1e-9 {
		t.Fatalf("fish did not turn in place: dir %d x %v", f.dir, f.x)
	}
	s = newTest(120, 40, 4)
	turns := 0
	for range int(3 * time.Minute / dt) {
		before := s.turns
		s.Step(dt)
		turns += s.turns - before
	}
	if turns == 0 {
		t.Fatal("no fish turned in three minutes")
	}
}

// Bubbles rise a fraction of a row per frame and pop at the surface.
func TestAquarium_BubblesRise(t *testing.T) {
	s := newTest(80, 30, 4)
	s.lone(fish{x: -100, base: 5, dir: -1, speed: 0, period: 5})
	s.fish = nil
	const v = 2.0
	s.bubbles = []bubble{{x: 40.25, x0: 40.25, y: 28, speed: v}}
	prev := 28.0
	for i := 1; ; i++ {
		s.Step(dt)
		b := s.bubbles[0]
		if b.popped > 0 {
			break
		}
		if d := prev - b.y; math.Abs(d-v*dt.Seconds()) > 1e-9 || d >= 1 {
			t.Fatalf("step %d: rose %v rows, want %v", i, d, v*dt.Seconds())
		}
		prev = b.y
		if len(litDots(s)) == 0 {
			t.Fatalf("step %d: bubble not drawn", i)
		}
		if i > 1000 {
			t.Fatal("bubble never popped")
		}
	}
	for range 20 {
		s.Step(dt)
	}
	if len(s.bubbles) != 0 {
		t.Fatalf("%d bubbles left after popping", len(s.bubbles))
	}
}

func TestAquarium_SeaweedSways(t *testing.T) {
	s := newTest(80, 30, 5)
	s.fish, s.bubbles, s.bubbleRate = nil, nil, 0
	if len(s.weeds) == 0 {
		t.Fatal("no seaweed")
	}
	wd := s.weeds[0]
	tipRow := s.h*4 - wd.h*4
	tips := map[int]bool{}
	for range 100 {
		s.Step(dt)
		for dx := 0; dx < s.w*2; dx++ {
			if lit(s, dx, tipRow) && abs(dx-wd.x*2) < 6 {
				tips[dx] = true
			}
		}
		if !lit(s, wd.x*2, s.h*4-1) && !lit(s, wd.x*2+1, s.h*4-1) {
			t.Fatal("seaweed base moved")
		}
	}
	if len(tips) < 2 {
		t.Fatalf("seaweed tip never moved: %v", tips)
	}
}

func TestAquarium_FishCountAndPalette(t *testing.T) {
	for _, sz := range [][2]int{{80, 24}, {200, 60}, {20, 10}} {
		s := newTest(sz[0], sz[1], 6)
		want := min(maxFish, max(1, sz[0]*sz[1]/cellsPerFish))
		if len(s.fish) != want {
			t.Errorf("%v: %d fish, want %d", sz, len(s.fish), want)
		}
	}
	s := newTest(120, 40, 7)
	allowed := map[scene.RGB]bool{bubbleCol: true, weedCol: true}
	for _, c := range fishCols {
		allowed[c] = true
	}
	for range 200 {
		s.Step(dt)
	}
	for y := 0; y < 40; y++ {
		for x := 0; x < 120; x++ {
			r := s.Glyph(x, y, noBox)
			if r == 0 {
				continue
			}
			if ansi.StringWidth(string(r)) != 1 {
				t.Fatalf("glyph %q not single width", r)
			}
			if c := s.RGB(x, y, noBox); !allowed[c] {
				t.Fatalf("(%d,%d) colour %v not in the palette", x, y, c)
			}
		}
	}
}

func TestAquarium_ResizeAnySize(t *testing.T) {
	s := newTest(0, 0, 8)
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

func TestAquarium_Interface(t *testing.T) {
	s := newTest(80, 24, 9)
	if s.Interval() != dt {
		t.Fatal("interval")
	}
	s.Tick()
	s.Step(-time.Second)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

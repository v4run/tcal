package rain

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"
)

const dt = 33 * time.Millisecond

// calm pins the wind at zero for the whole test.
func (s *Scene) calm() {
	s.wind, s.windGoal, s.windLeft = 0, 0, time.Hour
}

// lowestDotRow returns the lowest plotted dot row in cell column cx, or -1.
func lowestDotRow(s *Scene, cx int) int {
	_, h := s.canvas.Size()
	for cy := h - 1; cy >= 0; cy-- {
		g := s.canvas.Glyph(cx, cy)
		if g < 0x2800 || g > 0x28ff {
			continue
		}
		m := int(g - 0x2800)
		for r := 3; r >= 0; r-- {
			if m&([4]int{0x09, 0x12, 0x24, 0xC0}[r]) != 0 {
				return cy*4 + r
			}
		}
	}
	return -1
}

func TestRain_DropGlides(t *testing.T) {
	const v, y0 = 12.0, 1.0
	s := New(40, 60, rand.New(rand.NewPCG(1, 2)))
	s.calm()
	s.drops = []drop{{fx: 10.5, y: y0, speed: v}}
	s.Step(0)
	prev := lowestDotRow(s, 10)
	maxJump := int(math.Ceil(v*0.033*4)) + 1
	for i := 1; i <= 100; i++ {
		s.Step(dt)
		tt := float64(i) * 0.033
		want := int(math.Floor((y0 + v*tt) * 4))
		got := lowestDotRow(s, 10)
		if d := got - want; d < -1 || d > 1 {
			t.Fatalf("step %d: row %d, ideal %d", i, got, want)
		}
		if got < prev {
			t.Fatalf("step %d: row went back %d -> %d", i, prev, got)
		}
		if got-prev > maxJump {
			t.Fatalf("step %d: jump %d > %d", i, got-prev, maxJump)
		}
		prev = got
	}
}

func TestRain_WindSlantsAndChanges(t *testing.T) {
	s := New(80, 24, rand.New(rand.NewPCG(7, 8)))
	goals := map[float64]bool{}
	for range int(20 * time.Second / dt) {
		s.Step(dt)
		goals[s.windGoal] = true
		if math.Abs(s.wind) > 1.2 {
			t.Fatalf("wind %v exceeds 1.2", s.wind)
		}
	}
	if len(goals) < 2 {
		t.Fatalf("only %d wind goals in 20s", len(goals))
	}
}

func TestRain_WrapsHorizontally(t *testing.T) {
	s := New(40, 30, rand.New(rand.NewPCG(1, 2)))
	s.wind, s.windGoal, s.windLeft = 1.2, 1.2, time.Hour
	s.drops = []drop{{fx: 39.99, y: 1, speed: 10}}
	s.Step(dt)
	if fx := s.drops[0].fx; fx < 0 || fx > 2 {
		t.Fatalf("fx = %v, want wrapped near the left edge", fx)
	}
}

func TestRain_DensityAndResize(t *testing.T) {
	s := New(80, 24, rand.New(rand.NewPCG(1, 2)))
	for _, sz := range [][2]int{{80, 24}, {200, 60}, {30, 10}, {5, 3}} {
		s.Resize(sz[0], sz[1])
		want := max(1, sz[0]*(sz[1]-1)/70)
		if len(s.drops) != want {
			t.Errorf("%v: %d drops, want %d", sz, len(s.drops), want)
		}
	}
	s.Resize(0, 0)
	s.Step(dt)
	s.Resize(1, 1)
	s.Step(dt)
	s.Glyph(0, 0, noBox)
}

func TestRain_GroundAndSplash(t *testing.T) {
	const w, h = 40, 20
	s := New(w, h, rand.New(rand.NewPCG(1, 2)))
	s.calm()
	s.drops = []drop{{fx: 20.5, y: float64(h-1) - 0.05, speed: 10}}
	s.Step(dt)
	if s.drops[0].y >= 0 {
		t.Fatalf("drop y = %v, want respawned above the screen", s.drops[0].y)
	}
	n := 0
	for x := 0; x < w; x++ {
		if s.Glyph(x, h-2, noBox) != 0 {
			n++
		}
	}
	if n == 0 {
		t.Fatal("no splash dots on row h-2")
	}
	for x := 0; x < w; x++ {
		if s.Glyph(x, h-1, noBox) != '─' {
			t.Fatalf("ground missing at %d", x)
		}
	}
}

func TestRain_Interface(t *testing.T) {
	s := New(20, 10, rand.New(rand.NewPCG(1, 2)))
	if s.Interval() != dt {
		t.Fatal("interval")
	}
	s.Tick()
	_ = s.RGB(0, 0, noBox)
}

package aquarium

import (
	"math/rand/v2"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/scene"
)

func newRNG(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, seed)) }

func TestGlyphsAreSingleWidth(t *testing.T) {
	for _, sh := range shapes {
		for _, art := range []string{sh.right, sh.left} {
			if ansi.StringWidth(art) != len([]rune(art)) {
				t.Fatalf("fish %q has wide runes", art)
			}
		}
	}
	s := New(80, 24, newRNG(1))
	for i := 0; i < 1000; i++ {
		s.Tick()
		for y := 0; y < 24; y++ {
			for x := 0; x < 80; x++ {
				if r := s.Glyph(x, y, scene.Box{}); r != 0 && ansi.StringWidth(string(r)) != 1 {
					t.Fatalf("tick %d: glyph %q at (%d,%d) is not single width", i, r, x, y)
				}
			}
		}
	}
}

func TestFishSwimAndRespawn(t *testing.T) {
	const w, h = 40, 12
	s := New(w, h, newRNG(2))
	s.fish = []fish{{x: 10, y: 4, dir: 1, speed: 0.5, shape: 0}}
	s.Tick()
	if s.fish[0].x <= 10 && s.fish[0].dir == 1 {
		t.Fatalf("right-facing fish did not move right: x=%v", s.fish[0].x)
	}
	for i := 0; i < 5000; i++ {
		s.Tick()
		for _, f := range s.fish {
			n := float64(len(f.art()))
			if f.x+n < 0 || f.x >= w || f.y < 0 || f.y > h-2 {
				t.Fatalf("tick %d: fish out of bounds: %+v", i, f)
			}
		}
	}
}

func TestFishDrawnFacingTheirDirection(t *testing.T) {
	s := New(40, 10, newRNG(3))
	s.bubbles, s.weeds = nil, nil
	s.fish = []fish{{x: 5, y: 2, dir: 1, shape: 0}, {x: 20, y: 6, dir: -1, shape: 0}}
	s.render()
	got := string([]rune{s.Glyph(5, 2, scene.Box{}), s.Glyph(6, 2, scene.Box{}), s.Glyph(7, 2, scene.Box{})})
	if got != "><>" {
		t.Fatalf("right fish = %q", got)
	}
	got = string([]rune{s.Glyph(20, 6, scene.Box{}), s.Glyph(21, 6, scene.Box{}), s.Glyph(22, 6, scene.Box{})})
	if got != "<><" {
		t.Fatalf("left fish = %q", got)
	}
}

func TestFishCountScalesButStaysSparse(t *testing.T) {
	small, big := New(40, 10, newRNG(4)), New(300, 80, newRNG(4))
	if len(small.fish) < 1 || len(big.fish) <= len(small.fish) || len(big.fish) > maxFish {
		t.Fatalf("fish counts small=%d big=%d", len(small.fish), len(big.fish))
	}
}

func TestBubblesRiseAndPop(t *testing.T) {
	const h = 12
	s := New(40, h, newRNG(5))
	s.fish, s.weeds = nil, nil
	s.bubbles = []bubble{{x: 7, y: h - 1, speed: 0.5}}
	prev := s.bubbles[0].y
	popped := false
	for i := 0; i < 100 && !popped; i++ {
		s.Tick()
		b := s.bubbles[0] // spawned bubbles are appended after it
		if b.y > prev {
			t.Fatalf("bubble sank from %v to %v", prev, b.y)
		}
		prev = b.y
		if b.popped {
			popped = true
			if got := s.Glyph(7, 0, scene.Box{}); got != bubblePopGlyph {
				t.Fatalf("pop glyph = %q", got)
			}
		}
	}
	if !popped {
		t.Fatal("bubble never reached the surface")
	}
	s.Tick()
	if len(s.bubbles) > 0 && s.bubbles[0].x == 7 && s.bubbles[0].popped {
		t.Fatal("popped bubble lingered")
	}
}

func TestSeaweedSways(t *testing.T) {
	const h = 15
	s := New(60, h, newRNG(6))
	s.fish, s.bubbles = nil, nil
	if len(s.weeds) == 0 {
		t.Fatal("no seaweed planted")
	}
	wd := s.weeds[0]
	s.render()
	before := s.Glyph(wd.x, h-1, scene.Box{})
	if before != '(' && before != ')' {
		t.Fatalf("seaweed base = %q", before)
	}
	for range swayPeriod {
		s.Tick()
	}
	s.fish, s.bubbles = nil, nil
	s.render()
	if after := s.Glyph(wd.x, h-1, scene.Box{}); after == before {
		t.Fatalf("seaweed did not sway: still %q", after)
	}
}

func TestResizeAnySize(t *testing.T) {
	s := New(0, 0, newRNG(7))
	sizes := [][2]int{{0, 0}, {1, 1}, {200, 60}, {2, 1}, {1, 2}, {3, 3}, {0, 10}, {10, 0}, {-1, -1}, {80, 24}, {12, 6}, {5, 40}}
	for _, sz := range sizes {
		s.Resize(sz[0], sz[1])
		for i := 0; i < 300; i++ {
			s.Tick()
		}
		for y := -1; y <= sz[1]; y++ {
			for x := -1; x <= sz[0]; x++ {
				s.Glyph(x, y, scene.Box{})
			}
		}
	}
}

func TestIntervalNotTooFast(t *testing.T) {
	if iv := New(10, 10, newRNG(8)).Interval(); iv.Milliseconds() < 40 {
		t.Fatalf("interval %v below 40ms", iv)
	}
}

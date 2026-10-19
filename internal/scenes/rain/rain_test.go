package rain

import (
	"math/rand/v2"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/scene"
)

func newRNG(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, seed)) }

func TestGlyphsAreSingleWidth(t *testing.T) {
	s := New(80, 24, newRNG(1))
	for i := 0; i < 500; i++ {
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

func TestDropFallsAndSplashes(t *testing.T) {
	const w, h = 20, 10
	s := New(w, h, newRNG(2))
	s.drops = []drop{{x: 10, y: 0, speed: 1}}
	s.render()
	if got := s.Glyph(10, 0, scene.Box{}); got != dropGlyph {
		t.Fatalf("drop at top = %q, want %q", got, dropGlyph)
	}
	for y := 1; y < h-1; y++ {
		s.Tick()
		if got := s.Glyph(10, y, scene.Box{}); got != dropGlyph {
			t.Fatalf("after %d ticks (10,%d) = %q, want drop", y, y, got)
		}
	}

	s.Tick() // lands
	if got := s.Glyph(10, h-1, scene.Box{}); got != impactGlyph {
		t.Fatalf("impact = %q, want %q", got, impactGlyph)
	}
	if s.Glyph(9, h-2, scene.Box{}) != splashGlyphs[0] || s.Glyph(11, h-2, scene.Box{}) != splashGlyphs[0] {
		t.Fatal("no splash beside the landing point")
	}

	s.Tick()
	if got := s.Glyph(10, h-1, scene.Box{}); got != groundGlyph {
		t.Fatalf("ground after impact = %q, want %q", got, groundGlyph)
	}
	if s.Glyph(8, h-2, scene.Box{}) != splashGlyphs[1] {
		t.Fatal("splash did not widen on its second frame")
	}

	s.Tick()
	if len(s.splashes) != 0 {
		t.Fatalf("splash lingered: %v", s.splashes)
	}
	for x := 0; x < w; x++ {
		if r := s.Glyph(x, h-2, scene.Box{}); r == splashGlyphs[0] || r == splashGlyphs[1] {
			t.Fatalf("splash glyph still drawn at (%d,%d)", x, h-2)
		}
	}
}

func TestGroundLine(t *testing.T) {
	s := New(30, 8, newRNG(3))
	s.drops = nil
	s.render()
	for x := 0; x < 30; x++ {
		if got := s.Glyph(x, 7, scene.Box{}); got != groundGlyph {
			t.Fatalf("ground at %d = %q", x, got)
		}
	}
}

func TestDensityIsLight(t *testing.T) {
	s := New(120, 40, newRNG(4))
	for i := 0; i < 100; i++ {
		s.Tick()
	}
	n := 0
	for y := 0; y < 39; y++ {
		for x := 0; x < 120; x++ {
			if s.Glyph(x, y, scene.Box{}) != 0 {
				n++
			}
		}
	}
	if frac := float64(n) / (120 * 39); frac > 0.03 {
		t.Fatalf("sky is %.1f%% rain, want a light drizzle", frac*100)
	}
	if n == 0 {
		t.Fatal("no rain at all")
	}
}

func TestResizeAnySize(t *testing.T) {
	s := New(0, 0, newRNG(5))
	sizes := [][2]int{{0, 0}, {1, 1}, {1, 2}, {2, 1}, {100, 40}, {3, 3}, {0, 10}, {10, 0}, {-1, -1}, {80, 24}, {5, 50}}
	for _, sz := range sizes {
		s.Resize(sz[0], sz[1])
		for i := 0; i < 30; i++ {
			s.Tick()
			for y := -1; y <= sz[1]; y++ {
				for x := -1; x <= sz[0]; x++ {
					s.Glyph(x, y, scene.Box{})
				}
			}
		}
	}
}

func TestIntervalNotTooFast(t *testing.T) {
	if iv := New(10, 10, newRNG(6)).Interval(); iv.Milliseconds() < 40 {
		t.Fatalf("interval %v below 40ms", iv)
	}
}

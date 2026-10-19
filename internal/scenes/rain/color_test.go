package rain

import (
	"testing"

	"github.com/v4run/tcal/internal/scene"
)

func TestColorsInPalette(t *testing.T) {
	const w, h = 80, 24
	s := New(w, h, newRNG(1))
	want := map[rune]scene.Color{
		dropGlyph:       scene.Blue,
		groundGlyph:     scene.Default,
		impactGlyph:     scene.Cyan,
		splashGlyphs[0]: scene.Cyan,
		splashGlyphs[1]: scene.Cyan,
	}
	seen := map[rune]bool{}
	for i := 0; i < 500; i++ {
		s.Tick()
		for y := range h {
			for x := range w {
				g := s.Glyph(x, y, scene.Box{})
				if g == 0 {
					continue
				}
				wc, ok := want[g]
				if !ok {
					t.Fatalf("unexpected glyph %q", g)
				}
				if c := s.Color(x, y, scene.Box{}); c != wc {
					t.Fatalf("tick %d: %q at (%d,%d) color %d, want %d", i, g, x, y, c, wc)
				}
				seen[g] = true
			}
		}
	}
	if len(seen) != len(want) {
		t.Errorf("saw glyphs %v, want all of %v", seen, want)
	}
}

func TestGroundDefault(t *testing.T) {
	const w, h = 30, 10
	s := New(w, h, newRNG(3))
	for x := range w {
		if s.Glyph(x, h-1, scene.Box{}) == groundGlyph {
			if c := s.Color(x, h-1, scene.Box{}); c != scene.Default {
				t.Fatalf("ground at %d color %d", x, c)
			}
		}
	}
}

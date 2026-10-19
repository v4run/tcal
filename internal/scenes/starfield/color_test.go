package starfield

import (
	"testing"

	"github.com/v4run/tcal/internal/scene"
)

func TestColorsInPalette(t *testing.T) {
	allowed := map[scene.Color]bool{scene.Default: true, scene.White: true, scene.Yellow: true}
	const w, h = 80, 24
	s := New(w, h, newRNG(1))
	for i := 0; i < 1000; i++ {
		s.Tick()
		for y := range h {
			for x := range w {
				if s.Glyph(x, y, scene.Box{}) == 0 {
					continue
				}
				if c := s.Color(x, y, scene.Box{}); !allowed[c] {
					t.Fatalf("tick %d: color %d at (%d,%d)", i, c, x, y)
				}
			}
		}
	}
}

func TestShootingStarColors(t *testing.T) {
	s := New(60, 20, newRNG(5))
	s.stars = nil
	s.meteor = &meteor{x: 20, y: 5, dir: 1}
	s.render()
	if c := s.Color(20, 5, scene.Box{}); c != scene.Yellow {
		t.Fatalf("head color %d, want Yellow", c)
	}
	for i := 1; i < len(meteorTrail); i++ {
		x, y := 20-meteorDX*i, 5-i
		if s.Glyph(x, y, scene.Box{}) == 0 {
			t.Fatalf("trail %d missing", i)
		}
		if c := s.Color(x, y, scene.Box{}); c != scene.Default {
			t.Fatalf("trail %d color %d, want Default", i, c)
		}
	}
}

func TestStarLayerColors(t *testing.T) {
	s := New(80, 24, newRNG(2))
	s.meteor = nil
	s.render()
	for _, st := range s.stars {
		x := int(st.x)
		want := scene.Default
		if st.layer == len(layers)-1 {
			want = scene.White
		}
		// Overlapping stars: the later one wins the cell.
		if g := s.Glyph(x, st.y, scene.Box{}); g != layers[st.layer].glyph {
			continue
		}
		if c := s.Color(x, st.y, scene.Box{}); c != want {
			t.Fatalf("layer %d star color %d, want %d", st.layer, c, want)
		}
	}
}

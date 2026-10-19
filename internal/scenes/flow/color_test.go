package flow

import (
	"slices"
	"testing"

	"github.com/v4run/tcal/internal/scene"
)

func TestColorsInPalette(t *testing.T) {
	const w, h = 80, 24
	s := New(w, h, newRNG())
	seen := map[scene.Color]bool{}
	for i := range 1000 {
		s.Tick()
		for y := range h {
			for x := range w {
				if s.Glyph(x, y, scene.Box{}) == 0 {
					continue
				}
				c := s.Color(x, y, scene.Box{})
				if !slices.Contains(particleColors, c) {
					t.Fatalf("tick %d: color %d at (%d,%d) not in palette", i, c, x, y)
				}
				seen[c] = true
			}
		}
	}
	if len(seen) != len(particleColors) {
		t.Errorf("saw colors %v, want all of %v", seen, particleColors)
	}
}

func TestTrailKeepsParticleColor(t *testing.T) {
	s := New(80, 24, newRNG())
	s.ps = []particle{{x: 40, y: 12, life: 100, hue: 1}}
	var cells [][2]int
	for range 3 {
		s.Tick()
		p := s.ps[0]
		cells = append(cells, [2]int{int(p.x), int(p.y)})
	}
	for _, c := range cells {
		if s.Glyph(c[0], c[1], scene.Box{}) == 0 {
			t.Fatalf("no trail at %v", c)
		}
		if got := s.Color(c[0], c[1], scene.Box{}); got != particleColors[1] {
			t.Errorf("trail at %v color = %d, want %d", c, got, particleColors[1])
		}
	}
}

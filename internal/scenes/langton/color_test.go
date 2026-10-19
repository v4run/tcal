package langton

import (
	"slices"
	"testing"

	"github.com/v4run/tcal/internal/scene"
)

func TestColorsInPalette(t *testing.T) {
	const w, h = 60, 20
	s := New(w, h, newRNG())
	for i := range 3000 {
		s.Tick()
		if i%7 != 0 {
			continue
		}
		for y := range h {
			for x := range w {
				if s.Glyph(x, y, scene.Box{}) == 0 {
					continue
				}
				if c := s.Color(x, y, scene.Box{}); !slices.Contains(antColors[:], c) {
					t.Fatalf("tick %d: color %d at (%d,%d) not in palette", i, c, x, y)
				}
			}
		}
	}
}

func TestCellsTakeTheirAntsColor(t *testing.T) {
	s := New(20, 10, newRNG())
	clear(s.black)
	clear(s.col)
	s.nblack = 0
	s.ants = []ant{{3, 3, up}, {15, 6, up}}
	for range 3 {
		s.step()
	}
	a0, a1 := s.antColor(0), s.antColor(1)
	if a0 == a1 {
		t.Fatalf("ants share color %d", a0)
	}
	// Each ant blackened its start cell first, then two more.
	if s.Color(3, 3, scene.Box{}) != a0 || s.Color(4, 3, scene.Box{}) != a0 {
		t.Errorf("ant 0's cells not colored %d", a0)
	}
	if s.Color(15, 6, scene.Box{}) != a1 || s.Color(16, 6, scene.Box{}) != a1 {
		t.Errorf("ant 1's cells not colored %d", a1)
	}
	a := s.ants[1]
	if s.Glyph(a.x, a.y, scene.Box{}) != antGlyph || s.Color(a.x, a.y, scene.Box{}) != a1 {
		t.Errorf("ant 1 not drawn in its color")
	}
}

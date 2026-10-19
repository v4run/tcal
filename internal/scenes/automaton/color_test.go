package automaton

import (
	"slices"
	"testing"

	"github.com/v4run/tcal/internal/scene"
)

func TestColorsInPalette(t *testing.T) {
	const w, h = 60, 10
	s := New(w, h, newRNG())
	for i := range 2000 {
		s.Tick()
		for y := range h {
			for x := range w {
				if s.Glyph(x, y, scene.Box{}) == 0 {
					continue
				}
				if c := s.Color(x, y, scene.Box{}); !slices.Contains(runColors, c) {
					t.Fatalf("tick %d: color %d at (%d,%d) not in palette", i, c, x, y)
				}
			}
		}
	}
}

func TestRestartChangesColorOfNewRowsOnly(t *testing.T) {
	s := New(20, 8, newRNG())
	startRule90(s)
	for range 7 { // 8 generations: screen rows 0 and 1
		s.Tick()
	}
	mid := s.cols() / 2 / dotsX
	old := s.Color(mid, 0, scene.Box{})
	if s.Glyph(mid, 0, scene.Box{}) == 0 {
		t.Fatal("no apex glyph")
	}
	s.restart(90)
	for range 7 {
		s.Tick()
	}
	if got := s.Color(mid, 0, scene.Box{}); got != old {
		t.Errorf("old row color changed from %d to %d", old, got)
	}
	// Row 2 is the new run's apex.
	if s.Glyph(mid, 2, scene.Box{}) == 0 {
		t.Fatal("no new apex glyph")
	}
	if got := s.Color(mid, 2, scene.Box{}); got == old {
		t.Errorf("new run has the old run's color %d", got)
	}
}

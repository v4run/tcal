package lava

import (
	"slices"
	"testing"

	"github.com/v4run/tcal/internal/scene"
)

func TestColorsInPalette(t *testing.T) {
	const w, h = 80, 24
	s := New(w, h, newRNG())
	for i := range 500 {
		s.Tick()
		for y := range h {
			for x := range w {
				if s.Glyph(x, y, scene.Box{}) == 0 {
					continue
				}
				if c := s.Color(x, y, scene.Box{}); !slices.Contains(blobColors, c) {
					t.Fatalf("tick %d: color %d at (%d,%d) not in palette", i, c, x, y)
				}
			}
		}
	}
}

func TestCellTakesNearestBlobColor(t *testing.T) {
	s := New(80, 24, newRNG())
	s.blobs = []blob{{x: 20, y: 12, r: 3, hue: 0}, {x: 30, y: 12, r: 3, hue: 1}}
	s.render()
	if c := s.Color(20, 12, scene.Box{}); c != blobColors[0] {
		t.Errorf("left centre color = %d, want %d", c, blobColors[0])
	}
	if c := s.Color(30, 12, scene.Box{}); c != blobColors[1] {
		t.Errorf("right centre color = %d, want %d", c, blobColors[1])
	}
	// The merged bridge splits at the midpoint.
	if c := s.Color(23, 12, scene.Box{}); s.Glyph(23, 12, scene.Box{}) != 0 && c != blobColors[0] {
		t.Errorf("left of bridge color = %d, want %d", c, blobColors[0])
	}
	if c := s.Color(26, 12, scene.Box{}); s.Glyph(26, 12, scene.Box{}) != 0 && c != blobColors[1] {
		t.Errorf("right of bridge color = %d, want %d", c, blobColors[1])
	}
}

func TestBlobsKeepTheirColor(t *testing.T) {
	s := New(80, 24, newRNG())
	hues := make([]int, len(s.blobs))
	for i, b := range s.blobs {
		hues[i] = b.hue
	}
	for range 300 {
		s.Tick()
	}
	for i, b := range s.blobs {
		if b.hue != hues[i] {
			t.Fatalf("blob %d hue %d -> %d", i, hues[i], b.hue)
		}
	}
}

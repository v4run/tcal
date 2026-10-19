package aquarium

import (
	"slices"
	"testing"

	"github.com/v4run/tcal/internal/scene"
)

func TestColorsInPalette(t *testing.T) {
	const w, h = 80, 24
	s := New(w, h, newRNG(11))
	allowed := append(slices.Clone(fishColors), bubbleColor, weedColor)
	for i := range 2000 {
		s.Tick()
		for y := range h {
			for x := range w {
				if s.Glyph(x, y, scene.Box{}) == 0 {
					continue
				}
				if c := s.Color(x, y, scene.Box{}); !slices.Contains(allowed, c) {
					t.Fatalf("tick %d: color %d at (%d,%d) not in palette", i, c, x, y)
				}
			}
		}
	}
}

func TestPartColors(t *testing.T) {
	const h = 15
	s := New(60, h, newRNG(12))
	s.fish = []fish{{x: 5, y: 2, dir: 1, shape: 0, hue: 1}}
	s.bubbles = []bubble{{x: 30, y: 5}}
	s.render()
	for i := range 3 {
		if c := s.Color(5+i, 2, scene.Box{}); c != fishColors[1] {
			t.Errorf("fish cell %d color = %d, want %d", i, c, fishColors[1])
		}
	}
	if c := s.Color(30, 5, scene.Box{}); c != scene.Cyan {
		t.Errorf("bubble color = %d, want cyan", c)
	}
	if len(s.weeds) == 0 {
		t.Fatal("no seaweed planted")
	}
	if c := s.Color(s.weeds[0].x, h-1, scene.Box{}); c != scene.Green {
		t.Errorf("seaweed color = %d, want green", c)
	}
}

func TestFishKeepTheirColor(t *testing.T) {
	s := New(200, 60, newRNG(13))
	seen := map[int]bool{}
	for range 2000 {
		before := slices.Clone(s.fish)
		spawned := s.spawned
		s.Tick()
		for i, f := range s.fish {
			seen[f.hue] = true
			// Only a respawn (which bumps s.spawned) may change a hue.
			if s.spawned == spawned && f.hue != before[i].hue {
				t.Fatalf("fish %d changed hue %d -> %d without respawning", i, before[i].hue, f.hue)
			}
		}
	}
	if len(seen) != len(fishColors) {
		t.Errorf("fish hues %v; want all %d", seen, len(fishColors))
	}
}

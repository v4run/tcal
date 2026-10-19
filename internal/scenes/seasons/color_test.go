package seasons

import (
	"testing"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

func TestColorsInPalette(t *testing.T) {
	want := map[Season]map[scene.Color]bool{
		Winter: {scene.White: true},
		Spring: {scene.Magenta: true},
		Summer: {scene.Yellow: true},
		Autumn: {scene.Red: true, scene.Yellow: true},
	}
	const w, h = 100, 30
	for m := time.January; m <= time.December; m++ {
		s := newTest(w, h)
		s.SetTime(time.Time{}, month(m))
		allowed := want[s.Season()]
		seen := map[scene.Color]bool{}
		for i := 0; i < 200; i++ {
			s.Tick()
			for y := range h {
				for x := range w {
					if s.Glyph(x, y, scene.Box{}) == 0 {
						continue
					}
					c := s.Color(x, y, scene.Box{})
					if !allowed[c] {
						t.Fatalf("%v tick %d: color %d at (%d,%d)", m, i, c, x, y)
					}
					seen[c] = true
				}
			}
		}
		if len(seen) != len(allowed) {
			t.Errorf("%v: saw colors %v, want %v", m, seen, allowed)
		}
	}
}

func TestFirefliesYellow(t *testing.T) {
	const w, h = 80, 24
	s := newTest(w, h)
	s.SetTime(time.Time{}, month(time.July))
	n := 0
	for i := 0; i < 100; i++ {
		s.Tick()
		for y := range h {
			for x := range w {
				if s.Glyph(x, y, scene.Box{}) != 0 {
					n++
					if c := s.Color(x, y, scene.Box{}); c != scene.Yellow {
						t.Fatalf("firefly color %d", c)
					}
				}
			}
		}
	}
	if n == 0 {
		t.Fatal("no fireflies drawn")
	}
}

func TestLeafColorStable(t *testing.T) {
	s := newTest(80, 24)
	s.SetTime(time.Time{}, month(time.October))
	for i := range s.ps {
		c := s.color(i)
		for range 50 {
			s.Tick()
			if s.color(i) != c {
				t.Fatalf("leaf %d changed color", i)
			}
		}
	}
}

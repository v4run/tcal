package sky

import (
	"testing"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

func TestColorsInPalette(t *testing.T) {
	allowed := map[scene.Color]bool{scene.Default: true, scene.Yellow: true, scene.White: true}
	const w, h = 100, 30
	s := newTest(w, h)
	for hr := 0; hr < 24; hr += 2 {
		s.SetTime(at(hr, 17), time.Time{})
		for i := 0; i < 150; i++ {
			s.Tick()
			for y := range h {
				for x := range w {
					if s.Glyph(x, y, scene.Box{}) == 0 {
						continue
					}
					if c := s.Color(x, y, scene.Box{}); !allowed[c] {
						t.Fatalf("%02d:17 tick %d: color %d at (%d,%d)", hr, i, c, x, y)
					}
				}
			}
		}
	}
}

func TestSunYellowAtNoon(t *testing.T) {
	s := newTest(81, 30)
	s.clouds = nil
	s.SetTime(at(12, 0), time.Time{})
	x, y := s.arcPos(arcFraction(at(12, 0)))
	if g := s.Glyph(x, y, scene.Box{}); g != 'O' {
		t.Fatalf("sun centre = %q", g)
	}
	if c := s.Color(x, y, scene.Box{}); c != scene.Yellow {
		t.Fatalf("sun color = %d, want Yellow", c)
	}
}

func TestNightColors(t *testing.T) {
	const w, h = 81, 30
	s := newTest(w, h)
	full := time.Date(2024, time.April, 23, 0, 0, 0, 0, time.UTC) // near full moon
	s.SetTime(full, time.Time{})
	moon, stars, twinkles := 0, 0, 0
	for i := 0; i < 200; i++ {
		s.Tick()
		for y := range h {
			for x := range w {
				g, c := s.Glyph(x, y, scene.Box{}), s.Color(x, y, scene.Box{})
				switch {
				case g == 0:
				case g >= 0x2800 && g <= 0x28ff:
					moon++
					if c != scene.White {
						t.Fatalf("moon cell color %d", c)
					}
				case g == '+':
					twinkles++
					if c != scene.Yellow {
						t.Fatalf("twinkle color %d", c)
					}
				default:
					stars++
					if c != scene.Default {
						t.Fatalf("star %q color %d", g, c)
					}
				}
			}
		}
	}
	if moon == 0 || stars == 0 || twinkles == 0 {
		t.Fatalf("moon=%d stars=%d twinkles=%d; want all > 0", moon, stars, twinkles)
	}
}

package seasons

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/scene"
)

func newTest(w, h int) *Scene {
	return New(w, h, rand.New(rand.NewPCG(1, 2)))
}

func month(m time.Month) time.Time {
	return time.Date(2024, m, 1, 0, 0, 0, 0, time.UTC)
}

func TestSeasonOf(t *testing.T) {
	want := map[time.Month]Season{
		time.January: Winter, time.February: Winter, time.March: Spring,
		time.April: Spring, time.May: Spring, time.June: Summer,
		time.July: Summer, time.August: Summer, time.September: Autumn,
		time.October: Autumn, time.November: Autumn, time.December: Winter,
	}
	for m, se := range want {
		if got := SeasonOf(m); got != se {
			t.Errorf("SeasonOf(%v) = %v, want %v", m, got, se)
		}
	}
}

// count returns the number of drawn cells and whether every one is in glyphs.
func count(s *Scene, w, h int, glyphs []rune) (n int, ok bool) {
	ok = true
	for y := range h {
		for x := range w {
			r := s.Glyph(x, y, scene.Box{})
			if r == 0 {
				continue
			}
			n++
			found := false
			for _, g := range glyphs {
				found = found || g == r
			}
			ok = ok && found
		}
	}
	return n, ok
}

func TestShownMonthSwitchesSeasonImmediately(t *testing.T) {
	const w, h = 80, 24
	s := newTest(w, h)
	now := time.Date(2024, 7, 10, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		shown  time.Month
		season Season
		glyphs []rune
	}{
		{time.January, Winter, snowGlyphs},
		{time.April, Spring, petalGlyphs},
		{time.October, Autumn, []rune{'/', '\\', '_'}},
		{time.December, Winter, snowGlyphs},
	}
	for _, tc := range tests {
		s.SetTime(now, month(tc.shown))
		if s.Season() != tc.season {
			t.Fatalf("%v: season %v, want %v", tc.shown, s.Season(), tc.season)
		}
		n, ok := count(s, w, h, tc.glyphs)
		if n == 0 || !ok {
			t.Errorf("%v: %d glyphs drawn, all in season set = %v", tc.shown, n, ok)
		}
	}
}

func TestDensityIsLow(t *testing.T) {
	const w, h = 120, 40
	s := newTest(w, h)
	for m := time.January; m <= time.December; m++ {
		s.SetTime(month(m), month(m))
		for range 200 {
			s.Tick()
			if n, _ := count(s, w, h, nil); n > w*h/50 {
				t.Fatalf("%v: %d of %d cells drawn", m, n, w*h)
			}
		}
	}
}

func TestParticlesKeepFalling(t *testing.T) {
	const w, h = 80, 24
	for _, m := range []time.Month{time.January, time.April, time.October} {
		s := newTest(w, h)
		s.SetTime(month(m), month(m))
		n0 := len(s.ps)
		for range 2000 {
			s.Tick()
		}
		if len(s.ps) != n0 {
			t.Fatalf("%v: particle count changed %d -> %d", m, n0, len(s.ps))
		}
		if n, _ := count(s, w, h, nil); n == 0 {
			t.Errorf("%v: nothing drawn after 2000 ticks", m)
		}
	}
}

func TestFirefliesBlink(t *testing.T) {
	const w, h = 80, 24
	s := newTest(w, h)
	s.SetTime(month(time.July), month(time.July))
	lit, dark := 0, 0
	for range 300 {
		s.Tick()
		n, ok := count(s, w, h, fireflyGlyphs)
		if !ok {
			t.Fatal("non-firefly glyph in summer")
		}
		lit += n
		dark += len(s.ps) - n
	}
	if lit == 0 || dark < lit {
		t.Errorf("fireflies lit %d, dark %d: want mostly dark with blinks", lit, dark)
	}
}

func TestGlyphsAreSingleWidth(t *testing.T) {
	for seed := uint64(0); seed < 3; seed++ {
		s := New(120, 40, rand.New(rand.NewPCG(seed, seed)))
		for m := time.January; m <= time.December; m++ {
			s.SetTime(month(m), month(m))
			for range 150 {
				s.Tick()
				for y := range 40 {
					for x := range 120 {
						if r := s.Glyph(x, y, scene.Box{}); r != 0 && ansi.StringWidth(string(r)) != 1 {
							t.Fatalf("rune %q (%U) has width %d", r, r, ansi.StringWidth(string(r)))
						}
					}
				}
			}
		}
	}
}

func TestTinyAndUnsetDoNotPanic(t *testing.T) {
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {0, 10}, {10, 0}, {3, 2}, {1, 30}, {200, 1}, {-1, -1}} {
		s := newTest(sz[0], sz[1])
		s.Tick()
		s.Glyph(0, 0, scene.Box{})
		for m := time.January; m <= time.December; m++ {
			s.SetTime(month(m), month(m))
			for range 50 {
				s.Tick()
			}
			for y := -1; y <= sz[1]; y++ {
				for x := -1; x <= sz[0]; x++ {
					s.Glyph(x, y, scene.Box{})
				}
			}
		}
		s.Resize(0, 0)
		s.Tick()
		s.Resize(40, 12)
		s.Tick()
	}
}

func TestIntervalAtLeast40ms(t *testing.T) {
	if iv := newTest(10, 10).Interval(); iv < 40*time.Millisecond {
		t.Fatalf("interval %v", iv)
	}
}

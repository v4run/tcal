package sky

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

func at(hour, min int) time.Time {
	return time.Date(2024, time.April, 23, hour, min, 0, 0, time.UTC)
}

func TestSunFollowsArc(t *testing.T) {
	s := newTest(81, 30)
	bottom := s.arcBottom()
	tests := []struct {
		name  string
		t     time.Time
		x, y  int
		check func(x, y int) bool
	}{
		{"sunrise", at(6, 0), 0, bottom, nil},
		{"noon", at(12, 0), 40, arcTop, nil},
		{"before sunset", at(17, 59), 0, 0, func(x, y int) bool { return x >= 78 && y >= bottom-1 }},
		{"midnight moon", at(0, 0), 40, arcTop, nil},
	}
	for _, tc := range tests {
		x, y := s.arcPos(arcFraction(tc.t))
		if tc.check != nil {
			if !tc.check(x, y) {
				t.Errorf("%s: got (%d, %d)", tc.name, x, y)
			}
			continue
		}
		if x != tc.x || y != tc.y {
			t.Errorf("%s: got (%d, %d), want (%d, %d)", tc.name, x, y, tc.x, tc.y)
		}
	}
}

func TestIsDay(t *testing.T) {
	for h, want := range map[int]bool{0: false, 5: false, 6: true, 12: true, 17: true, 18: false, 23: false} {
		if got := IsDay(at(h, 0)); got != want {
			t.Errorf("IsDay(%02d:00) = %v, want %v", h, got, want)
		}
	}
}

func litDots(p float64) (lit, disc int) {
	g := moonGlyphs(p)
	for r := range g {
		for _, ch := range g[r] {
			if ch != 0 {
				for b := ch - 0x2800; b != 0; b &= b - 1 {
					lit++
				}
			}
		}
	}
	full := moonGlyphs(0.5)
	for r := range full {
		for _, ch := range full[r] {
			if ch != 0 {
				for b := ch - 0x2800; b != 0; b &= b - 1 {
					disc++
				}
			}
		}
	}
	return lit, disc
}

func TestMoonPhase(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want float64
	}{
		{"new 2024-04-08", time.Date(2024, 4, 8, 18, 21, 0, 0, time.UTC), 0},
		{"first quarter 2024-04-15", time.Date(2024, 4, 15, 19, 13, 0, 0, time.UTC), 0.25},
		{"full 2024-04-23", time.Date(2024, 4, 23, 23, 49, 0, 0, time.UTC), 0.5},
		{"last quarter 2024-05-01", time.Date(2024, 5, 1, 11, 27, 0, 0, time.UTC), 0.75},
	}
	for _, tc := range tests {
		p := MoonPhase(tc.t)
		d := p - tc.want
		if d > 0.5 {
			d -= 1
		}
		if d < -0.05 || d > 0.05 {
			t.Errorf("%s: phase %.3f, want ~%.2f", tc.name, p, tc.want)
		}
	}
}

func TestMoonShape(t *testing.T) {
	_, disc := litDots(0.5)
	if disc < 40 {
		t.Fatalf("full disc only %d dots", disc)
	}
	full, _ := litDots(MoonPhase(time.Date(2024, 4, 23, 22, 0, 0, 0, time.UTC)))
	if full != disc {
		t.Errorf("full moon: %d of %d dots lit", full, disc)
	}
	if n, _ := litDots(MoonPhase(time.Date(2024, 4, 8, 18, 0, 0, 0, time.UTC))); n != 0 {
		t.Errorf("new moon: %d dots lit", n)
	}
	// First quarter: right half lit (waxing), left half dark.
	g := moonGlyphs(0.25)
	for r := range moonRows {
		if g[r][0] != 0 || g[r][1] != 0 || g[r][2] == 0 || g[r][3] == 0 {
			t.Errorf("first quarter row %d: %q", r, string(g[r][:]))
		}
	}
	g = moonGlyphs(0.75)
	for r := range moonRows {
		if g[r][0] == 0 || g[r][1] == 0 || g[r][2] != 0 || g[r][3] != 0 {
			t.Errorf("last quarter row %d: %q", r, string(g[r][:]))
		}
	}
}

func TestStaysInUpperScreen(t *testing.T) {
	const w, h = 100, 40
	s := newTest(w, h)
	for hour := range 24 {
		s.SetTime(at(hour, 30), at(0, 0))
		for range 50 {
			s.Tick()
		}
		for y := h / 3; y < h; y++ {
			for x := range w {
				if r := s.Glyph(x, y, scene.Box{}); r != 0 {
					t.Fatalf("%02d:30: glyph %q at row %d", hour, r, y)
				}
			}
		}
	}
}

func TestDayAndNightDraw(t *testing.T) {
	s := newTest(80, 24)
	count := func() (n int, sun bool) {
		for y := range 24 {
			for x := range 80 {
				r := s.Glyph(x, y, scene.Box{})
				if r != 0 {
					n++
				}
				if r == 'O' {
					sun = true
				}
			}
		}
		return n, sun
	}
	s.SetTime(at(10, 0), at(0, 0))
	if n, sun := count(); n == 0 || !sun {
		t.Errorf("day: %d glyphs, sun=%v", n, sun)
	}
	s.SetTime(at(22, 0), at(0, 0))
	if n, sun := count(); n == 0 || sun {
		t.Errorf("night: %d glyphs, sun=%v", n, sun)
	}
}

func TestGlyphsAreSingleWidth(t *testing.T) {
	for seed := uint64(0); seed < 5; seed++ {
		s := New(120, 40, rand.New(rand.NewPCG(seed, seed)))
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
		for i := range 400 {
			now := start.Add(time.Duration(i) * 97 * time.Minute)
			s.SetTime(now, now)
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

func TestTinyAndUnsetDoNotPanic(t *testing.T) {
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {0, 10}, {10, 0}, {3, 2}, {5, 1}, {200, 3}, {-1, -1}} {
		s := newTest(sz[0], sz[1])
		s.Tick()
		s.Glyph(0, 0, scene.Box{})
		for hour := range 24 {
			s.SetTime(at(hour, 0), at(0, 0))
			s.Tick()
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

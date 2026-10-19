package hourglass

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/scene"
)

func newTest(w, h int) *Scene {
	return New(w, h, rand.New(rand.NewPCG(1, 2)))
}

func at(min, sec int) time.Time {
	return time.Date(2024, 4, 23, 14, min, sec, 0, time.UTC)
}

func TestHourFraction(t *testing.T) {
	tests := []struct {
		t    time.Time
		want float64
	}{
		{at(0, 0), 0},
		{at(15, 0), 0.25},
		{at(30, 0), 0.5},
		{at(59, 59), 3599.0 / 3600},
	}
	for _, tc := range tests {
		if got := HourFraction(tc.t); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("HourFraction(%s) = %v, want %v", tc.t.Format("15:04:05"), got, tc.want)
		}
	}
}

func TestPileVolumeTracksHour(t *testing.T) {
	const w, h = 80, 24
	s := newTest(w, h)
	tests := []struct {
		t    time.Time
		frac float64
	}{
		{at(0, 0), 0},
		{at(30, 0), 0.5},
		{at(59, 0), 59.0 / 60},
	}
	for _, tc := range tests {
		s.SetTime(tc.t, tc.t)
		want := tc.frac * s.Capacity()
		// Per-column rounding is off by at most half an eighth per column.
		if got := float64(s.Volume()); math.Abs(got-want) > float64(w)/2 {
			t.Errorf("%s: volume %v, want ~%v", tc.t.Format("04:05"), got, want)
		}
	}
	s.SetTime(at(0, 0), at(0, 0))
	if v := s.Volume(); v != 0 {
		t.Errorf("on the hour: volume %d, want 0", v)
	}
}

func TestPileIsMound(t *testing.T) {
	const w, h = 80, 24
	s := newTest(w, h)
	s.SetTime(at(45, 0), at(0, 0))
	mid, edge := s.heights[w/2], s.heights[0]
	if mid <= edge || edge == 0 {
		t.Errorf("mid %d, edge %d: want a mound over a thin base", mid, edge)
	}
	if peak := slices.Max(s.heights); peak > s.maxRows()*8 {
		t.Errorf("peak %d eighths exceeds %d rows", peak, s.maxRows())
	}
	// Bottom row is drawn under every column, rows above maxRows are not.
	for x := range w {
		if s.Glyph(x, h-1, scene.Box{}) == 0 {
			t.Fatalf("column %d has no sand", x)
		}
	}
}

func TestPileSurfaceUsesEighths(t *testing.T) {
	const w, h = 40, 24
	s := newTest(w, h)
	s.SetTime(at(20, 0), at(0, 0))
	s.grains = nil // just the pile
	s.render()
	for x, ht := range s.heights {
		full, part := ht/8, ht%8
		for r := range full {
			if g := s.Glyph(x, h-1-r, scene.Box{}); g != '█' {
				t.Fatalf("col %d row %d: %q, want full block", x, r, g)
			}
		}
		if g := s.Glyph(x, h-1-full, scene.Box{}); g != eighths[part] {
			t.Fatalf("col %d top: %q, want %q", x, g, eighths[part])
		}
	}
}

func TestGrainsFallAndLand(t *testing.T) {
	const w, h = 80, 24
	s := newTest(w, h)
	s.SetTime(at(30, 0), at(0, 0))
	seen := 0
	for range 500 {
		s.Tick()
		for _, g := range s.grains {
			if int(g.y) >= s.surface(g.x) {
				t.Fatalf("grain at row %d inside the pile (surface %d)", int(g.y), s.surface(g.x))
			}
		}
		for y := range h {
			for x := range w {
				if s.Glyph(x, y, scene.Box{}) == '·' {
					seen++
				}
			}
		}
	}
	if seen == 0 {
		t.Error("no grains ever drawn")
	}
}

func TestGlyphsAreSingleWidth(t *testing.T) {
	for seed := uint64(0); seed < 3; seed++ {
		s := New(120, 40, rand.New(rand.NewPCG(seed, seed)))
		start := at(0, 0)
		for i := range 600 {
			now := start.Add(time.Duration(i) * 7 * time.Second)
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
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {0, 10}, {10, 0}, {3, 2}, {1, 30}, {200, 1}, {-1, -1}} {
		s := newTest(sz[0], sz[1])
		s.Tick()
		s.Glyph(0, 0, scene.Box{})
		for m := range 60 {
			s.SetTime(at(m, 30), at(0, 0))
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

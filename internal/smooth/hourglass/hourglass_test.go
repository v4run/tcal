package hourglass

import (
	"math"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

const dt = 33 * time.Millisecond

var noBox = scene.Box{Y0: -1, Y1: -1}

var dotBits = [4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

func newTest(w, h int) *Scene { return New(w, h, rand.New(rand.NewPCG(1, 2))) }

func at(min, sec int) time.Time { return time.Date(2024, 4, 23, 14, min, sec, 0, time.UTC) }

// lit reports whether the dot at dot column dx, dot row dy is set.
func lit(s *Scene, dx, dy int) bool {
	g := s.canvas.Glyph(dx/2, dy/4)
	return g >= 0x2800 && g <= 0x28ff && int(g-0x2800)&dotBits[dy%4][dx%2] != 0
}

func TestHourFraction(t *testing.T) {
	for _, tc := range []struct {
		t    time.Time
		want float64
	}{{at(0, 0), 0}, {at(15, 0), 0.25}, {at(30, 0), 0.5}, {at(59, 59), 3599.0 / 3600}} {
		if got := HourFraction(tc.t); math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("HourFraction(%s) = %v, want %v", tc.t.Format("15:04:05"), got, tc.want)
		}
	}
}

func TestHourglass_FillFollowsTime(t *testing.T) {
	const w, h = 80, 24
	s := newTest(w, h)
	if s.Glyph(w/2, h-1, noBox) != 0 {
		t.Fatal("drew before the first SetTime")
	}
	prev := -1
	for _, tc := range []struct {
		t    time.Time
		frac float64
	}{{at(0, 0), 0}, {at(15, 0), 0.25}, {at(30, 0), 0.5}, {at(59, 0), 59.0 / 60}} {
		s.SetTime(tc.t, tc.t)
		want := tc.frac * s.Capacity()
		// Per-column rounding is off by at most half a dot per dot column.
		if got := float64(s.Volume()); math.Abs(got-want) > float64(w) {
			t.Errorf("%s: volume %v, want ~%v", tc.t.Format("04:05"), got, want)
		}
		if v := s.Volume(); v <= prev {
			t.Errorf("%s: volume %d did not grow past %d", tc.t.Format("04:05"), v, prev)
		} else {
			prev = v
		}
	}
	s.SetTime(at(0, 0), at(0, 0))
	if v := s.Volume(); v != 0 {
		t.Errorf("on the hour: volume %d, want 0", v)
	}
	// SetTime ignores the shown month.
	s.SetTime(at(30, 0), at(0, 0).AddDate(0, 3, 0))
	a := s.Volume()
	s.SetTime(at(30, 0), at(30, 0))
	if s.Volume() != a {
		t.Error("shown time changed the fill")
	}
}

func TestHourglass_PileIsMoundInDotRows(t *testing.T) {
	const w, h = 80, 24
	s := newTest(w, h)
	s.SetTime(at(45, 0), at(0, 0))
	s.grains = nil
	s.render()
	mid, edge := s.heights[w], s.heights[0]
	if mid <= edge || edge == 0 {
		t.Errorf("mid %d, edge %d: want a mound over a thin base", mid, edge)
	}
	if peak := slices.Max(s.heights); peak > s.maxRows()*4 {
		t.Errorf("peak %d dot rows exceeds %d rows", peak, s.maxRows())
	}
	for dx, ht := range s.heights {
		for r := range ht {
			if !lit(s, dx, h*4-1-r) {
				t.Fatalf("column %d: dot row %d under the surface is empty", dx, r)
			}
		}
		if lit(s, dx, h*4-1-ht) {
			t.Fatalf("column %d: dot above the surface is set", dx)
		}
	}
}

func TestHourglass_GrainsGlide(t *testing.T) {
	const w, h = 80, 24
	const y0, v = 1.0, grainSpeed
	s := newTest(w, h)
	s.SetTime(at(1, 0), at(1, 0)) // a low pile, so the grain falls far
	gx := w                       // a dot column in the middle
	s.grains = []grain{{x: gx, y: y0}}
	s.render()
	row := func() int {
		for dy := 0; dy < h*4; dy++ {
			if s.heights[gx] > 0 && dy >= h*4-s.heights[gx] {
				break
			}
			if lit(s, gx, dy) {
				return dy
			}
		}
		return -1
	}
	prev := row()
	for i := 1; i <= 100; i++ {
		s.Step(dt)
		want := int(math.Floor((y0 + v*float64(i)*0.033) * 4))
		if s.surface(gx) <= want {
			t.Fatalf("grain should still be falling at step %d", i)
		}
		got := row()
		if d := got - want; d < -1 || d > 1 {
			t.Fatalf("step %d: dot row %d, ideal %d", i, got, want)
		}
		if got < prev {
			t.Fatalf("step %d: went back", i)
		}
		if got-prev > int(math.Ceil(v*0.033*4))+1 {
			t.Fatalf("step %d: jump %d", i, got-prev)
		}
		prev = got
	}
}

func TestHourglass_GrainsLandAndRespawn(t *testing.T) {
	const w, h = 80, 24
	s := newTest(w, h)
	s.SetTime(at(30, 0), at(0, 0))
	seen := 0
	for range 500 {
		s.Step(dt)
		for _, g := range s.grains {
			if int(g.y*4) >= s.surface(g.x) {
				t.Fatalf("grain at dot row %d inside the pile (surface %d)", int(g.y*4), s.surface(g.x))
			}
			seen++
		}
	}
	if seen == 0 {
		t.Error("no grains ever drawn")
	}
}

func TestHourglass_TinyAndUnset(t *testing.T) {
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {0, 10}, {10, 0}, {3, 2}, {1, 30}, {200, 1}, {-1, -1}} {
		s := newTest(sz[0], sz[1])
		s.Tick()
		for m := range 60 {
			s.SetTime(at(m, 30), at(0, 0))
			s.Step(dt)
			for y := -1; y <= sz[1]; y++ {
				for x := -1; x <= sz[0]; x++ {
					s.Glyph(x, y, noBox)
					s.RGB(x, y, noBox)
				}
			}
		}
		s.Resize(0, 0)
		s.Tick()
		s.Resize(40, 12)
		s.Tick()
	}
	if newTest(10, 10).Interval() != dt {
		t.Fatal("interval")
	}
}

package seasons

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const dt = 33 * time.Millisecond

var noBox = scene.Box{Y0: -1, Y1: -1}

func newTest(w, h int) *Scene { return New(w, h, rand.New(rand.NewPCG(1, 2))) }

func at(y int, m time.Month, d, h int) time.Time {
	return time.Date(y, m, d, h, 0, 0, 0, time.Local)
}

var dotBits = [4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

// litDots returns every lit dot as (column, row).
func litDots(s *Scene) [][2]int {
	var out [][2]int
	for cy := range s.h {
		for cx := range s.w {
			g := s.canvas.Glyph(cx, cy)
			if g < 0x2800 || g > 0x28ff {
				continue
			}
			for r := range 4 {
				for c := range 2 {
					if int(g-0x2800)&dotBits[r][c] != 0 {
						out = append(out, [2]int{cx*2 + c, cy*4 + r})
					}
				}
			}
		}
	}
	return out
}

// A swaying snowflake stays within a dot of p0 + v*t, in both axes.
func TestSeasons_ParticlesGlide(t *testing.T) {
	const x0, y0, v, ph0 = 20.0, 2.0, 1.5, 0.7
	s := newTest(60, 60)
	jan := at(2026, time.January, 10, 12)
	s.SetTime(jan, jan)
	s.ps = []particle{{x: x0, y: y0, x0: x0, speed: v, phase: ph0}}
	s.Step(0)
	moved := false
	var first [2]int
	for i := 1; i <= 100; i++ {
		s.Step(dt)
		tt := float64(i) * 0.033
		wx := int(math.Floor((x0 + snowSway*math.Sin(ph0+snowRate*tt)) * 2))
		wy := int(math.Floor((y0 + v*tt) * 4))
		ds := litDots(s)
		if len(ds) != 1 {
			t.Fatalf("step %d: %d dots lit, want 1", i, len(ds))
		}
		if math.Abs(float64(ds[0][0]-wx)) > 1 || math.Abs(float64(ds[0][1]-wy)) > 1 {
			t.Fatalf("step %d: dot %v, ideal (%d, %d)", i, ds[0], wx, wy)
		}
		if i == 1 {
			first = ds[0]
		}
		moved = moved || ds[0] != first
	}
	if !moved {
		t.Error("flake never moved")
	}
}

func TestSeasons_FollowsShownMonth(t *testing.T) {
	s := newTest(80, 24)
	now := at(2026, time.July, 10, 12)
	if len(litDots(s)) != 0 {
		t.Fatal("drawn before SetTime")
	}
	cols := map[Season]scene.RGB{Winter: snowCol, Spring: petalCol, Autumn: leafCols[0]}
	for m, se := range map[time.Month]Season{
		time.January: Winter, time.April: Spring, time.July: Summer,
		time.October: Autumn, time.December: Winter,
	} {
		s.SetTime(now, at(2026, m, 1, 0))
		if s.Season() != se {
			t.Fatalf("%v: season %v, want %v", m, s.Season(), se)
		}
		for i := 0; i < 60; i++ {
			s.Step(dt)
		}
		if len(litDots(s)) == 0 {
			t.Errorf("%v: nothing drawn", m)
		}
		if want, ok := cols[se]; ok && se != Autumn {
			found := false
			for y := range s.h {
				for x := range s.w {
					if s.Glyph(x, y, noBox) != 0 && s.RGB(x, y, noBox) == want {
						found = true
					}
				}
			}
			if !found {
				t.Errorf("%v: no %v-coloured particle", m, want)
			}
		}
	}
}

func TestSeasons_EventReported(t *testing.T) {
	s := newTest(100, 40)
	dec := at(2026, time.December, 25, 12)
	jul := at(2026, time.July, 1, 0)
	s.SetTime(dec, jul)
	if _, ok := s.Event(); ok {
		t.Fatal("event reported without EnableEvents")
	}
	s.EnableEvents(nil)
	s.SetTime(dec, jul)
	if k, ok := s.Event(); !ok || k != events.Christmas {
		t.Errorf("Event() = %v, %v; want christmas, true", k, ok)
	}
	if s.Season() != Summer {
		t.Errorf("Season() = %v, want Summer (shown month)", s.Season())
	}
	s.SetTime(at(2026, time.November, 30, 23), jul)
	if k, ok := s.Event(); ok {
		t.Errorf("Nov 30: event %v active, want none", k)
	}
}

func TestSeasons_ResizeAndTinyScreens(t *testing.T) {
	s := newTest(80, 24)
	jan := at(2026, time.January, 10, 12)
	s.SetTime(jan, jan)
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {5, 0}, {120, 50}} {
		s.Resize(sz[0], sz[1])
		s.Step(dt)
		s.Glyph(0, 0, noBox)
	}
}

// On Christmas Day the tree from the event art shows through Glyph and RGB,
// covering the particles in its corner.
func TestSeasons_DrawsEventArt(t *testing.T) {
	const w, h = 120, 40
	s := newTest(w, h)
	s.EnableEvents(nil)
	xmas := at(2026, time.December, 25, 12)
	s.SetTime(xmas, xmas)
	for range 10 {
		s.Step(dt)
	}
	if s.art == nil {
		t.Fatal("no event art on Christmas Day")
	}
	// The tree is in the bottom-right corner.
	tree := 0
	for y := h - 10; y < h; y++ {
		for x := w - 15; x < w; x++ {
			r, c := s.art.Cell(x, y, noBox)
			if r == 0 {
				continue
			}
			tree++
			if g := s.Glyph(x, y, noBox); g != r {
				t.Fatalf("Glyph(%d, %d) = %q, want the tree's %q", x, y, g, r)
			}
			if got := s.RGB(x, y, noBox); got != c {
				t.Fatalf("RGB(%d, %d) = %v, want the tree's %v", x, y, got, c)
			}
		}
	}
	if tree < 20 {
		t.Fatalf("tree has %d cells, want a whole tree", tree)
	}
	if s.Glyph(w-8, h-9, noBox) != '*' {
		t.Errorf("no star atop the tree")
	}
	// Beside the calendar the art hides and the particles show instead.
	box := scene.Box{X0: 0, X1: w, Y0: 0, Y1: h - 1}
	for y := h - 10; y < h; y++ {
		for x := w - 15; x < w; x++ {
			if g := s.Glyph(x, y, box); g != s.canvas.Glyph(x, y) {
				t.Fatalf("Glyph(%d, %d) = %q under the calendar, want the particle layer", x, y, g)
			}
		}
	}
}

// A holiday with no art yet builds none: an untyped nil, not a typed nil
// that would read as present.
func TestSeasons_NoArtForKindWithout(t *testing.T) {
	if a := newArt(events.Halloween, rand.New(rand.NewPCG(1, 2))); a != nil {
		t.Fatalf("newArt(Halloween) = %T, want nil", a)
	}
	s := newTest(120, 40)
	s.EnableEvents(nil)
	hw := at(2026, time.October, 31, 20)
	s.SetTime(hw, hw)
	if k, ok := s.Event(); !ok || k != events.Halloween {
		t.Fatalf("Event() = %v, %v", k, ok)
	}
	if s.art != nil {
		t.Fatalf("art %T for Halloween", s.art)
	}
	s.Step(dt)
	s.Glyph(0, 0, noBox)
}

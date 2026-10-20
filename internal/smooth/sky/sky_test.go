package sky

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/scene"
)

const dt = 33 * time.Millisecond

var noBox = scene.Box{Y0: -1, Y1: -1}

var dotBits = [4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

func newTest(w, h int) *Scene { return New(w, h, rand.New(rand.NewPCG(1, 2))) }

func at(hour, min int) time.Time {
	return time.Date(2024, time.April, 23, hour, min, 0, 0, time.UTC)
}

// lit reports whether the dot at dot column dx, dot row dy is set.
func lit(s *Scene, dx, dy int) bool {
	g := s.canvas.Glyph(dx/2, dy/4)
	return g >= 0x2800 && g <= 0x28ff && int(g-0x2800)&dotBits[dy%4][dx%2] != 0
}

func TestIsDay(t *testing.T) {
	for h, want := range map[int]bool{0: false, 5: false, 6: true, 12: true, 17: true, 18: false, 23: false} {
		if got := IsDay(at(h, 0)); got != want {
			t.Errorf("IsDay(%02d:00) = %v, want %v", h, got, want)
		}
	}
}

// The arc matches classic: rises bottom-left, peaks mid-screen at noon and
// midnight, sets bottom-right.
func TestSky_ArcMatchesClassic(t *testing.T) {
	s := newTest(81, 30)
	bottom := float64(s.arcBottom())
	for _, tc := range []struct {
		name string
		t    time.Time
		x, y float64
	}{
		{"sunrise", at(6, 0), 0, bottom},
		{"noon", at(12, 0), 40, arcTop},
		{"midnight moon", at(0, 0), 40, arcTop},
		{"moonrise", at(18, 0), 0, bottom},
	} {
		x, y := s.arcPos(arcFraction(tc.t))
		if math.Abs(x-tc.x) > 1e-9 || math.Abs(y-tc.y) > 1e-9 {
			t.Errorf("%s: got (%.2f, %.2f), want (%.0f, %.0f)", tc.name, x, y, tc.x, tc.y)
		}
	}
	if b := newTest(80, 40).arcBottom(); b != maxArcRows {
		t.Errorf("tall screen arc bottom %d, want %d", b, maxArcRows)
	}
	if b := newTest(80, 12).arcBottom(); b != 4 {
		t.Errorf("short screen arc bottom %d, want 4", b)
	}
}

// Across 10 minutes of shifted time stepped by 1 s, the sun's dot centre
// moves monotonically (right and up in the morning) by at most 1 dot a step,
// and the disc is drawn there.
func TestSky_SunGlides(t *testing.T) {
	const w, h = 120, 40
	s := newTest(w, h)
	s.clouds = nil
	start := at(9, 0)
	s.SetTime(start, start)
	px, py := s.sunCentre()
	moved := 0
	for i := 1; i <= 600; i++ {
		now := start.Add(time.Duration(i) * time.Second)
		s.SetTime(now, start)
		x, y := s.sunCentre()
		if x < px || y > py {
			t.Fatalf("step %d: centre (%d, %d) went back from (%d, %d)", i, x, y, px, py)
		}
		if x-px > 1 || py-y > 1 {
			t.Fatalf("step %d: centre jumped from (%d, %d) to (%d, %d)", i, px, py, x, y)
		}
		if x != px || y != py {
			moved++
		}
		// The disc's middle dots are lit around the centre.
		for _, d := range [][2]int{{-1, -1}, {0, 0}, {-3, 0}, {2, -1}} {
			if !lit(s, x+d[0], y+d[1]) {
				t.Fatalf("step %d: disc dot (%d, %d) not lit", i, x+d[0], y+d[1])
			}
		}
		px, py = x, y
	}
	if moved == 0 {
		t.Error("sun never moved in 10 minutes")
	}
}

// The sun warms from {240,200,90} at noon to {240,140,70} near the horizon.
func TestSky_DuskWarmer(t *testing.T) {
	s := newTest(120, 40)
	s.clouds = nil
	colourAt := func(tm time.Time) scene.RGB {
		s.SetTime(tm, tm)
		x, y := s.sunCentre()
		return s.canvas.RGB(x/2, y/4)
	}
	if c := colourAt(at(12, 0)); c != sunNoon {
		t.Errorf("noon sun %v, want %v", c, sunNoon)
	}
	if c := colourAt(at(9, 0)); c != sunNoon {
		t.Errorf("mid-morning sun %v, want %v", c, sunNoon)
	}
	dusk := colourAt(at(17, 30))
	if !(dusk.G < sunNoon.G && dusk.G > sunLow.G) {
		t.Errorf("17:30 sun %v, want between %v and %v", dusk, sunLow, sunNoon)
	}
	if c := colourAt(at(17, 59)); c.G > dusk.G || c.G-sunLow.G > 2 {
		t.Errorf("17:59 sun %v, want ~%v", c, sunLow)
	}
	if c := colourAt(at(6, 1)); c.G-sunLow.G > 2 {
		t.Errorf("06:01 sun %v, want ~%v", c, sunLow)
	}
	if c := colourAt(at(6, 30)); !(c.G < sunNoon.G && c.G > sunLow.G) {
		t.Errorf("06:30 sun %v, want between", c)
	}
}

func TestSky_StaysInUpperThird(t *testing.T) {
	for _, sz := range [][2]int{{100, 40}, {120, 40}, {80, 24}, {60, 12}, {200, 60}} {
		w, h := sz[0], sz[1]
		s := newTest(w, h)
		for hour := range 24 {
			for _, min := range []int{0, 30, 59} {
				s.SetTime(at(hour, min), at(0, 0))
				for range 50 {
					s.Step(dt)
				}
				for y := h / 3; y < h; y++ {
					for x := range w {
						if r := s.Glyph(x, y, noBox); r != 0 {
							t.Fatalf("%dx%d %02d:%02d: glyph %q at row %d", w, h, hour, min, r, y)
						}
					}
				}
			}
		}
	}
}

func TestSky_MoonPhaseMatchesClassicMaths(t *testing.T) {
	for _, tc := range []struct {
		name string
		t    time.Time
		want float64
	}{
		{"new 2024-04-08", time.Date(2024, 4, 8, 18, 21, 0, 0, time.UTC), 0},
		{"first quarter 2024-04-15", time.Date(2024, 4, 15, 19, 13, 0, 0, time.UTC), 0.25},
		{"full 2024-04-23", time.Date(2024, 4, 23, 23, 49, 0, 0, time.UTC), 0.5},
		{"last quarter 2024-05-01", time.Date(2024, 5, 1, 11, 27, 0, 0, time.UTC), 0.75},
	} {
		p := MoonPhase(tc.t)
		d := p - tc.want
		if d > 0.5 {
			d--
		}
		if d < -0.05 || d > 0.05 {
			t.Errorf("%s: phase %.3f, want ~%.2f", tc.name, p, tc.want)
		}
	}
	// The drawn moon: full on 2024-04-23, dark on 2024-04-08, waxing lit on
	// the right.
	count := func(tm time.Time) (left, right int) {
		s := newTest(120, 40)
		s.SetTime(tm, tm)
		mx, my := s.moonOrigin()
		for dy := range moonDots {
			for dx := range moonDots {
				if lit(s, mx+dx, my+dy) {
					if dx < moonDots/2 {
						left++
					} else {
						right++
					}
				}
			}
		}
		return left, right
	}
	l, r := count(time.Date(2024, 4, 23, 22, 0, 0, 0, time.UTC))
	if l+r < 40 || l != r {
		t.Errorf("full moon: %d left, %d right dots", l, r)
	}
	if l, r := count(time.Date(2024, 4, 8, 22, 0, 0, 0, time.UTC)); l+r != 0 {
		t.Errorf("new moon: %d dots lit", l+r)
	}
	if l, r := count(time.Date(2024, 4, 15, 22, 0, 0, 0, time.UTC)); l > 2 || r < 12 {
		t.Errorf("first quarter: %d left, %d right", l, r)
	}
}

func TestSky_MoonTint(t *testing.T) {
	s := newTest(120, 40)
	tm := time.Date(2024, 4, 23, 0, 0, 0, 0, time.UTC)
	s.SetTime(tm, tm)
	mx, my := s.moonOrigin()
	if c := s.canvas.RGB((mx+moonDots/2)/2, (my+moonDots/2)/4); c != moonCol {
		t.Errorf("moon colour %v, want %v", c, moonCol)
	}
}

// A cloud's plotted dot column stays within 1 dot of x0 + v*t over 100
// steps, and is drawn in cloud grey.
func TestSky_CloudsGlide(t *testing.T) {
	s := newTest(200, 40)
	if len(s.clouds) == 0 {
		t.Fatal("no clouds")
	}
	s.SetTime(at(10, 0), at(0, 0))
	c := &s.clouds[0]
	c.x = 20
	x0, v := c.x, c.speed
	if v <= 0 || v*dt.Seconds() >= 0.5 {
		t.Fatalf("cloud speed %v cells/s is not sub-cell", v)
	}
	for i := 1; i <= 100; i++ {
		s.Step(dt)
		ideal := (x0 + v*float64(i)*dt.Seconds()) * 2
		got := s.cloudDot(0)
		if math.Abs(float64(got)-ideal) > 1 {
			t.Fatalf("step %d: dot column %d, ideal %.2f", i, got, ideal)
		}
	}
	found := false
	for y := range 40 {
		for x := range 200 {
			if s.Glyph(x, y, noBox) != 0 && s.RGB(x, y, noBox) == cloudCol {
				found = true
			}
		}
	}
	if !found {
		t.Error("no cloud-coloured cells by day")
	}
}

func TestSky_ShimmerAndDayNight(t *testing.T) {
	s := newTest(120, 40)
	s.clouds = nil
	s.SetTime(at(12, 0), at(0, 0))
	snap := func() string {
		var b []rune
		for y := range 14 {
			for x := range 120 {
				b = append(b, s.Glyph(x, y, noBox))
			}
		}
		return string(b)
	}
	a := snap()
	changed := false
	for range int(2 * shimmerPeriod / dt) {
		s.Step(dt)
		if snap() != a {
			changed = true
		}
	}
	if !changed {
		t.Error("rays did not shimmer")
	}
	// Top ray stays on screen at noon.
	x, _ := s.sunCentre()
	if !lit(s, x, 0) && !lit(s, x-1, 0) {
		t.Error("top ray clipped at noon")
	}

	// Night: stars and moon, no sun colour anywhere; day: no stars.
	s = newTest(120, 40)
	s.SetTime(at(22, 0), at(0, 0))
	stars := 0
	for y := range 40 {
		for x := range 120 {
			if s.Glyph(x, y, noBox) == 0 {
				continue
			}
			c := s.RGB(x, y, noBox)
			if c == sunNoon || c == sunLow || c == cloudCol {
				t.Fatalf("night cell (%d, %d) has day colour %v", x, y, c)
			}
			if c != moonCol {
				stars++
			}
		}
	}
	if stars < 20 {
		t.Errorf("night: %d star cells", stars)
	}
}

func TestSky_StarsTwinkle(t *testing.T) {
	s := newTest(120, 40)
	s.SetTime(at(22, 0), at(0, 0))
	twinkled := false
	for range int(30 * time.Second / dt) {
		s.Step(dt)
		for y := range 14 {
			for x := range 120 {
				if s.Glyph(x, y, noBox) != 0 && s.RGB(x, y, noBox) == starBright {
					twinkled = true
				}
			}
		}
		if twinkled {
			break
		}
	}
	if !twinkled {
		t.Error("no star twinkled in 30 s")
	}
}

func TestSky_NothingBeforeSetTime(t *testing.T) {
	s := newTest(80, 24)
	s.Step(dt)
	for y := range 24 {
		for x := range 80 {
			if s.Glyph(x, y, noBox) != 0 {
				t.Fatal("drew before the first SetTime")
			}
		}
	}
}

func TestSky_SingleWidthAndTinySizes(t *testing.T) {
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {0, 10}, {10, 0}, {3, 2}, {5, 1}, {200, 3}, {-1, -1}, {120, 40}} {
		s := newTest(sz[0], sz[1])
		s.Step(dt)
		for i := range 60 {
			now := at(0, 0).Add(time.Duration(i) * 97 * time.Minute)
			s.SetTime(now, now)
			s.Step(100 * time.Millisecond)
			for y := -1; y <= sz[1]; y++ {
				for x := -1; x <= sz[0]; x++ {
					if r := s.Glyph(x, y, noBox); r != 0 && ansi.StringWidth(string(r)) != 1 {
						t.Fatalf("rune %q has width %d", r, ansi.StringWidth(string(r)))
					}
				}
			}
		}
		s.Resize(0, 0)
		s.Step(dt)
		s.Resize(40, 12)
		s.Tick()
	}
	if iv := newTest(10, 10).Interval(); iv != dt {
		t.Errorf("interval %v", iv)
	}
}

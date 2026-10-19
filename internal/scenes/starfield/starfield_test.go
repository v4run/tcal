package starfield

import (
	"math/rand/v2"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/scene"
)

func newRNG(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, seed)) }

func TestGlyphsAreSingleWidth(t *testing.T) {
	s := New(80, 24, newRNG(1))
	for i := 0; i < 1000; i++ {
		s.Tick()
		for y := 0; y < 24; y++ {
			for x := 0; x < 80; x++ {
				if r := s.Glyph(x, y, scene.Box{}); r != 0 && ansi.StringWidth(string(r)) != 1 {
					t.Fatalf("tick %d: glyph %q at (%d,%d) is not single width", i, r, x, y)
				}
			}
		}
	}
}

func TestStarsDriftLeftWithParallax(t *testing.T) {
	s := New(100, 20, newRNG(2))
	s.stars = []star{{x: 50, y: 5, layer: 0}, {x: 50, y: 10, layer: len(layers) - 1}}
	for i := 0; i < 20; i++ {
		s.Tick()
	}
	far, near := s.stars[0].x, s.stars[1].x
	if far >= 50 || near >= 50 {
		t.Fatalf("stars did not drift left: far=%v near=%v", far, near)
	}
	if near >= far {
		t.Fatalf("near star (%v) should outpace far star (%v)", near, far)
	}
	if s.stars[0].y != 5 || s.stars[1].y != 10 {
		t.Fatal("stars changed rows while drifting")
	}
}

func TestStarsStayInBoundsAndWrap(t *testing.T) {
	const w, h = 40, 12
	s := New(w, h, newRNG(3))
	n := len(s.stars)
	s.stars[0] = star{x: 0.01, y: 3, layer: len(layers) - 1}
	s.Tick()
	if x := int(s.stars[0].x); x != w-1 {
		t.Fatalf("star leaving left edge respawned at column %d, want %d", x, w-1)
	}
	for i := 0; i < 3000; i++ {
		s.Tick()
		for _, st := range s.stars {
			if st.x < 0 || int(st.x) >= w || st.y < 0 || st.y >= h {
				t.Fatalf("tick %d: star out of bounds: %+v", i, st)
			}
		}
	}
	if len(s.stars) != n {
		t.Fatalf("star count changed from %d to %d", n, len(s.stars))
	}
}

func TestShootingStarAppearsAndPasses(t *testing.T) {
	s := New(100, 30, newRNG(4))
	seen, ended := false, false
	for i := 0; i < meteorMinWait+meteorWaitVar+meteorLife+10; i++ {
		s.Tick()
		if s.meteor != nil {
			seen = true
		} else if seen {
			ended = true
			break
		}
	}
	if !seen {
		t.Fatal("no shooting star within the maximum wait")
	}
	if !ended {
		t.Fatal("shooting star never finished")
	}
}

func TestShootingStarDrawsHeadAndTrail(t *testing.T) {
	s := New(60, 20, newRNG(5))
	s.stars = nil
	s.meteor = &meteor{x: 20, y: 5, dir: 1}
	s.render()
	if got := s.Glyph(20, 5, scene.Box{}); got != '*' {
		t.Fatalf("head = %q", got)
	}
	if got := s.Glyph(20-meteorDX, 4, scene.Box{}); got != '\\' {
		t.Fatalf("trail = %q, want '\\\\'", got)
	}
	s.Tick()
	if got := s.Glyph(20+meteorDX, 6, scene.Box{}); got != '*' {
		t.Fatalf("head after tick = %q", got)
	}
}

func TestSkyIsSparse(t *testing.T) {
	s := New(120, 40, newRNG(6))
	n := 0
	for y := 0; y < 40; y++ {
		for x := 0; x < 120; x++ {
			if s.Glyph(x, y, scene.Box{}) != 0 {
				n++
			}
		}
	}
	if frac := float64(n) / (120 * 40); n == 0 || frac > 0.03 {
		t.Fatalf("sky is %.1f%% stars", frac*100)
	}
}

func TestResizeAnySize(t *testing.T) {
	s := New(0, 0, newRNG(7))
	sizes := [][2]int{{0, 0}, {1, 1}, {100, 40}, {2, 1}, {1, 2}, {3, 3}, {0, 10}, {10, 0}, {-1, -1}, {80, 24}, {9, 5}}
	for _, sz := range sizes {
		s.Resize(sz[0], sz[1])
		for i := 0; i < 400; i++ {
			s.Tick()
		}
		for y := -1; y <= sz[1]; y++ {
			for x := -1; x <= sz[0]; x++ {
				s.Glyph(x, y, scene.Box{})
			}
		}
	}
}

func TestIntervalNotTooFast(t *testing.T) {
	if iv := New(10, 10, newRNG(8)).Interval(); iv.Milliseconds() < 40 {
		t.Fatalf("interval %v below 40ms", iv)
	}
}

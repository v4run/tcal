package flow

import (
	"math/rand/v2"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/scene"
)

func newRNG() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }

func TestParticlesStayInBounds(t *testing.T) {
	s := New(80, 24, newRNG())
	if len(s.ps) < minParticles {
		t.Fatalf("got %d particles, want at least %d", len(s.ps), minParticles)
	}
	for range 2000 {
		s.Tick()
		for _, p := range s.ps {
			if p.x < 0 || p.y < 0 || p.x >= 80 || p.y >= 24 {
				t.Fatalf("particle out of bounds: %+v", p)
			}
		}
	}
}

func TestTrailsAgeOut(t *testing.T) {
	s := New(20, 10, newRNG())
	s.ps = nil
	s.stamp[5*20+5] = int32(s.tick)
	want := func(r rune) {
		t.Helper()
		if got := s.Glyph(5, 5, scene.Box{}); got != r {
			t.Fatalf("tick %d: glyph = %q, want %q", s.tick, got, r)
		}
	}
	want(headGlyph)
	for range headAge {
		s.Tick()
	}
	want(trailGlyph)
	for range trailAge - headAge {
		s.Tick()
	}
	want(0)
}

func TestTrailsAreSparse(t *testing.T) {
	s := New(200, 60, newRNG())
	for range 500 {
		s.Tick()
	}
	n := 0
	for y := range 60 {
		for x := range 200 {
			if s.Glyph(x, y, scene.Box{}) != 0 {
				n++
			}
		}
	}
	if n == 0 || n > 200*60/10 {
		t.Errorf("%d drawn cells, want some but under 10%%", n)
	}
}

func TestGlyphWidthsAndTinySizes(t *testing.T) {
	s := New(0, 0, newRNG())
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {3, 0}, {0, 5}, {2, 3}, {120, 40}, {7, 2}} {
		s.Resize(sz[0], sz[1])
		for range 300 {
			s.Tick()
			for y := -1; y <= sz[1]; y++ {
				for x := -1; x <= sz[0]; x++ {
					if r := s.Glyph(x, y, scene.Box{}); r != 0 && ansi.StringWidth(string(r)) != 1 {
						t.Fatalf("glyph %q at (%d,%d) has width %d", r, x, y, ansi.StringWidth(string(r)))
					}
				}
			}
		}
	}
}

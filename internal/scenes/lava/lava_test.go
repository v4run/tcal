package lava

import (
	"math/rand/v2"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/scene"
)

func newRNG() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }

func TestBlobFieldNearAndFar(t *testing.T) {
	s := New(80, 24, newRNG())
	s.blobs = []blob{{x: 20, y: 12, r: 3}}
	s.render()
	if got := s.Glyph(20, 12, scene.Box{}); got != coreGlyph {
		t.Errorf("centre glyph = %q, want %q", got, coreGlyph)
	}
	// At the rim (just inside r rows above the centre) the shade is light.
	if got := s.Glyph(20, 9, scene.Box{}); got != rimGlyph {
		t.Errorf("rim glyph = %q, want %q", got, rimGlyph)
	}
	if got := s.Glyph(70, 2, scene.Box{}); got != 0 {
		t.Errorf("far glyph = %q, want empty", got)
	}
}

func TestBlobsLookRound(t *testing.T) {
	s := New(80, 40, newRNG())
	s.blobs = []blob{{x: 40, y: 20, r: 5}}
	s.render()
	// Horizontal extent in columns should be about twice the vertical
	// extent in rows.
	wide, tall := 0, 0
	for x := range s.w {
		if s.Glyph(x, 20, scene.Box{}) != 0 {
			wide++
		}
	}
	for y := range s.h {
		if s.Glyph(40, y, scene.Box{}) != 0 {
			tall++
		}
	}
	if wide < 2*tall-2 || wide > 2*tall+2 {
		t.Errorf("blob is %d cols × %d rows, want ~2:1", wide, tall)
	}
}

func TestBlobsStayOnScreen(t *testing.T) {
	s := New(50, 20, newRNG())
	for range 5000 {
		s.Tick()
		for _, b := range s.blobs {
			if b.x < 0 || b.x > 50 || b.y < 0 || b.y > 20 {
				t.Fatalf("blob escaped: %+v", b)
			}
		}
	}
}

func TestCoverageIsModest(t *testing.T) {
	s := New(200, 60, newRNG())
	for range 300 {
		s.Tick()
		n := 0
		for _, r := range s.buf {
			if r != 0 {
				n++
			}
		}
		if frac := float64(n) / float64(len(s.buf)); frac > 0.35 {
			t.Fatalf("coverage %.2f too high", frac)
		}
	}
}

func TestGlyphWidthsAndTinySizes(t *testing.T) {
	s := New(0, 0, newRNG())
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {3, 0}, {0, 5}, {2, 3}, {120, 40}, {7, 2}} {
		s.Resize(sz[0], sz[1])
		for range 200 {
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

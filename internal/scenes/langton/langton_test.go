package langton

import (
	"math/rand/v2"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/scene"
)

func newRNG() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }

func TestFirstMoves(t *testing.T) {
	s := New(10, 10, newRNG())
	clear(s.black)
	s.nblack = 0
	s.ants = []ant{{5, 5, up}}

	// On white: turn right, flip, advance. The ant traces a small square
	// clockwise, then on returning to the black start cell turns left.
	want := []ant{
		{6, 5, right},
		{6, 6, down},
		{5, 6, left},
		{5, 5, up},
		{4, 5, left},
	}
	for i, w := range want {
		s.step()
		if s.ants[0] != w {
			t.Fatalf("step %d: ant = %+v, want %+v", i+1, s.ants[0], w)
		}
	}
	// Cells (6,5), (6,6), (5,6) are black; (5,5) flipped back to white.
	for _, c := range [][2]int{{6, 5}, {6, 6}, {5, 6}} {
		if !s.black[c[1]*10+c[0]] {
			t.Errorf("cell %v should be black", c)
		}
	}
	if s.black[5*10+5] || s.nblack != 3 {
		t.Errorf("start cell black=%v, nblack=%d; want white, 3", s.black[55], s.nblack)
	}
	if s.Glyph(4, 5, scene.Box{}) != antGlyph || s.Glyph(6, 5, scene.Box{}) != cellGlyph {
		t.Error("glyphs do not show ant and black cell")
	}
}

func TestWrapsAround(t *testing.T) {
	s := New(4, 3, newRNG())
	clear(s.black)
	s.nblack = 0
	s.ants = []ant{{3, 0, up}} // turns right, then steps off the right edge
	s.step()
	if s.ants[0].x != 0 || s.ants[0].y != 0 {
		t.Errorf("ant = %+v, want wrapped to (0,0)", s.ants[0])
	}
}

func TestCoverageStaysBounded(t *testing.T) {
	s := New(30, 10, newRNG())
	for range 20000 {
		s.Tick()
		if float64(s.nblack) > maxCoverage*300 {
			t.Fatalf("coverage %d/300 above limit", s.nblack)
		}
		if s.steps > stepBudget*maxAnts {
			t.Fatalf("steps %d over budget", s.steps)
		}
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

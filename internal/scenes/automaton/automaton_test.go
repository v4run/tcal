package automaton

import (
	"math/rand/v2"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/scene"
)

func newRNG() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }

// sierpinski reports whether rule 90 from a single cell at the origin has
// cell j alive at generation k: C(k, (k+j)/2) is odd.
func sierpinski(k, j int) bool {
	if (k+j)%2 != 0 || j < -k || j > k {
		return false
	}
	n := (k + j) / 2
	return n&k == n // Lucas' theorem
}

func startRule90(s *Scene) {
	s.gens = nil
	s.restart(90)
	s.render()
}

func TestRule90IsSierpinski(t *testing.T) {
	s := New(20, 8, newRNG()) // 40 columns, 32 generations visible
	startRule90(s)
	mid := s.cols() / 2
	for range 19 {
		s.Tick()
	}
	if len(s.gens) != 20 {
		t.Fatalf("got %d generations, want 20", len(s.gens))
	}
	for k, row := range s.gens {
		for c, alive := range row {
			if want := sierpinski(k, c-mid); alive != want {
				t.Fatalf("gen %d col %d = %v, want %v", k, c, alive, want)
			}
		}
	}
	// The centre cell covers columns mid, mid+1 over generations 0–3:
	// gen 0 → (0,0), gen 1 → (1,1), gen 2 → none, gen 3 → (1,3).
	if got, want := s.Glyph(mid/dotsX, 0, scene.Box{}), rune(brailleBase+0x01+0x10+0x80); got != want {
		t.Errorf("apex glyph = %q, want %q", got, want)
	}
}

func TestScrollsWhenFull(t *testing.T) {
	s := New(40, 2, newRNG()) // 80 columns, 8 generations visible
	startRule90(s)
	mid := s.cols() / 2
	const ticks = 20 // < maxScreens*8, and the pattern stays inside 80 cols
	for range ticks {
		s.Tick()
	}
	if len(s.gens) != s.maxGens() {
		t.Fatalf("got %d generations, want %d", len(s.gens), s.maxGens())
	}
	first := ticks + 1 - s.maxGens() // generation shown on the top dot row
	for i, row := range s.gens {
		for c, alive := range row {
			if want := sierpinski(first+i, c-mid); alive != want {
				t.Fatalf("row %d (gen %d) col %d = %v, want %v", i, first+i, c, alive, want)
			}
		}
	}
}

func TestRestartsWhenDead(t *testing.T) {
	s := New(10, 4, newRNG())
	s.gens = nil
	s.restart(90)
	// Rule 0 kills everything, which must trigger a restart.
	s.rule = 0
	s.Tick()
	if s.rule == 0 || s.run != 0 {
		t.Errorf("rule %d run %d; want a restart with a new rule", s.rule, s.run)
	}
}

func TestRunsAreBounded(t *testing.T) {
	s := New(30, 3, newRNG())
	for range 2000 {
		s.Tick()
		if s.run >= maxScreens*s.maxGens() {
			t.Fatalf("run %d not restarted", s.run)
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

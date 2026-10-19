package life

import (
	"math/rand/v2"
	"testing"
)

func newGrid(w, h int, alive ...[2]int) *Grid {
	g := New(w, h, rand.New(rand.NewPCG(1, 2)))
	for _, c := range alive {
		g.Set(c[0], c[1], true)
	}
	return g
}

func aliveCells(g *Grid) map[[2]int]bool {
	out := map[[2]int]bool{}
	for y := 0; y < g.Height(); y++ {
		for x := 0; x < g.Width(); x++ {
			if g.Alive(x, y) {
				out[[2]int{x, y}] = true
			}
		}
	}
	return out
}

func assertAlive(t *testing.T, g *Grid, want ...[2]int) {
	t.Helper()
	got := aliveCells(g)
	if len(got) != len(want) {
		t.Fatalf("got %d live cells %v, want %v", len(got), got, want)
	}
	for _, c := range want {
		if !got[c] {
			t.Fatalf("cell %v not alive; live cells: %v", c, got)
		}
	}
}

func TestStep_BlockIsStill(t *testing.T) {
	g := newGrid(6, 6, [2]int{2, 2}, [2]int{3, 2}, [2]int{2, 3}, [2]int{3, 3})
	g.Step()
	assertAlive(t, g, [2]int{2, 2}, [2]int{3, 2}, [2]int{2, 3}, [2]int{3, 3})
}

func TestStep_BlinkerOscillates(t *testing.T) {
	g := newGrid(5, 5, [2]int{1, 2}, [2]int{2, 2}, [2]int{3, 2})
	g.Step()
	assertAlive(t, g, [2]int{2, 1}, [2]int{2, 2}, [2]int{2, 3})
	g.Step()
	assertAlive(t, g, [2]int{1, 2}, [2]int{2, 2}, [2]int{3, 2})
}

func TestStep_GliderTranslatesAfterFourGenerations(t *testing.T) {
	g := newGrid(8, 8, [2]int{1, 0}, [2]int{2, 1}, [2]int{0, 2}, [2]int{1, 2}, [2]int{2, 2})
	for i := 0; i < 4; i++ {
		g.Step()
	}
	assertAlive(t, g, [2]int{2, 1}, [2]int{3, 2}, [2]int{1, 3}, [2]int{2, 3}, [2]int{3, 3})
}

func TestStep_WrapsAtEdges(t *testing.T) {
	// Horizontal blinker straddling the left/right edge.
	g := newGrid(5, 5, [2]int{4, 2}, [2]int{0, 2}, [2]int{1, 2})
	g.Step()
	assertAlive(t, g, [2]int{0, 1}, [2]int{0, 2}, [2]int{0, 3})
}

func TestSettled(t *testing.T) {
	cases := []struct {
		name  string
		cells [][2]int
		want  bool
	}{
		{"empty", nil, true},
		{"block", [][2]int{{2, 2}, {3, 2}, {2, 3}, {3, 3}}, true},
		{"blinker", [][2]int{{1, 2}, {2, 2}, {3, 2}}, true},
		{"glider", [][2]int{{1, 0}, {2, 1}, {0, 2}, {1, 2}, {2, 2}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := newGrid(10, 10, tc.cells...)
			g.Step()
			g.Step()
			if got := g.Settled(); got != tc.want {
				t.Errorf("Settled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTick_ReseedsAfterSettlingForAWhile(t *testing.T) {
	g := newGrid(20, 20)
	for i := 0; i < settleLimit+1; i++ {
		g.Tick()
	}
	if len(aliveCells(g)) == 0 {
		t.Fatal("expected empty grid to be reseeded")
	}
}

func TestSeed_RoughlyMatchesDensity(t *testing.T) {
	g := newGrid(100, 100)
	g.Seed()
	n := len(aliveCells(g))
	if n < 1500 || n > 2500 {
		t.Errorf("got %d live cells of 10000, want ~%d", n, int(Density*10000))
	}
}

func TestNew_ZeroSizeIsSafe(t *testing.T) {
	g := newGrid(0, 0)
	g.Seed()
	g.Tick()
	if g.Alive(0, 0) {
		t.Error("zero-size grid reported a live cell")
	}
}

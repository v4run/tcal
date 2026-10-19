package train

import (
	"math/rand/v2"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/scene"
)

func rng(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, seed^0x9e37)) }

// lowBox is a calendar box whose bottom sits well clear of the strip.
func lowBox(h int) scene.Box { return scene.Box{X0: 20, X1: 60, Y0: 1, Y1: h - Height - 1} }

func TestSpritesAreRectangular(t *testing.T) {
	for i, art := range append([][]string{loco}, cars...) {
		if len(art) != trainH {
			t.Fatalf("sprite %d has %d rows", i, len(art))
		}
		for _, row := range art {
			if len([]rune(row)) != len([]rune(art[0])) {
				t.Fatalf("sprite %d is ragged", i)
			}
		}
	}
}

func TestGlyphsAreSingleWidth(t *testing.T) {
	const w, h = 100, 30
	for seed := uint64(0); seed < 6; seed++ {
		s := New(w, h, rng(seed))
		b := lowBox(h)
		for i := 0; i < 300; i++ {
			s.Tick()
			for y := h - Height; y < h; y++ {
				for x := 0; x < w; x++ {
					if r := s.Glyph(x, y, b); r != 0 && ansi.StringWidth(string(r)) != 1 {
						t.Fatalf("glyph %q at (%d,%d) has width %d", r, x, y, ansi.StringWidth(string(r)))
					}
				}
			}
		}
	}
}

func TestHidesWhenBoxTouchesStrip(t *testing.T) {
	const w, h = 100, 30
	s := New(w, h, rng(2))
	b := scene.Box{X0: 20, X1: 60, Y0: 2, Y1: h - Height}
	for i := 0; i < 300; i++ {
		s.Tick()
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if r := s.Glyph(x, y, b); r != 0 {
					t.Fatalf("drew %q at (%d,%d) with box in the strip", r, x, y)
				}
			}
		}
	}
}

func TestHidesWhenTooNarrow(t *testing.T) {
	s := New(MinWidth-1, 30, rng(3))
	s.Tick()
	if r := s.Glyph(0, 29, scene.Box{Y0: -1, Y1: -1}); r != 0 {
		t.Fatalf("drew %q on a narrow screen", r)
	}
}

// TestTrainPassesAndTrackEmpties fast-forwards through several trains,
// checking each arrives soon (first) or within the 1–4 minute window,
// crosses the whole screen, and leaves only rails behind.
func TestTrainPassesAndTrackEmpties(t *testing.T) {
	const w, h = 80, 24
	for seed := uint64(0); seed < 10; seed++ {
		s := New(w, h, rng(seed))
		b := lowBox(h)
		for n := 0; n < 3; n++ {
			idle := 0
			for !s.Passing() {
				s.Tick()
				idle++
			}
			lo, hi := minWait, minWait+waitSpread+1
			if n == 0 {
				lo, hi = 1, firstWait+1
			}
			if idle < lo || idle > hi {
				t.Fatalf("seed %d train %d: arrived after %d ticks, want %d..%d", seed, n, idle, lo, hi)
			}

			seenLeft, seenRight := false, false
			for ticks := 0; s.Passing(); ticks++ {
				if ticks > w+200 {
					t.Fatalf("seed %d train %d: still passing after %d ticks", seed, n, ticks)
				}
				for y := 0; y < h; y++ {
					for x := 0; x < w; x++ {
						r := s.Glyph(x, y, b)
						if y < h-Height && r != 0 {
							t.Fatalf("seed %d: glyph %q above strip at (%d,%d)", seed, r, x, y)
						}
						if y < h-1 && r != 0 {
							seenLeft = seenLeft || x == 0
							seenRight = seenRight || x == w-1
						}
					}
				}
				s.Tick()
			}
			if !seenLeft || !seenRight {
				t.Fatalf("seed %d train %d: did not cross the screen", seed, n)
			}
			for y := h - Height; y < h; y++ {
				for x := 0; x < w; x++ {
					r := s.Glyph(x, y, b)
					if y < h-1 && r != 0 {
						t.Fatalf("seed %d: leftover %q at (%d,%d)", seed, r, x, y)
					}
					if y == h-1 && r == 0 {
						t.Fatalf("seed %d: missing rail at column %d", seed, x)
					}
				}
			}
		}
	}
}

func TestResizeAnySize(t *testing.T) {
	s := New(0, 0, rng(5))
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {3, 2}, {200, 60}, {10, 3}, {0, 50}, {-1, -1}} {
		s.Resize(sz[0], sz[1])
		for i := 0; i < 100; i++ {
			s.Tick()
		}
		for y := -1; y <= sz[1]; y++ {
			for x := -1; x <= sz[0]; x++ {
				s.Glyph(x, y, scene.Box{Y0: -1, Y1: -1})
			}
		}
	}
}

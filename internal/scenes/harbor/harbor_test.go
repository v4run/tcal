package harbor

import (
	"math/rand/v2"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/scene"
)

func rng(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, seed^0x9e37)) }

// lowBox is a calendar box whose bottom sits well clear of the strip.
func lowBox(h int) scene.Box { return scene.Box{X0: 20, X1: 60, Y0: 1, Y1: h - Height - 1} }

func TestGlyphsAreSingleWidth(t *testing.T) {
	const w, h = 100, 30
	s := New(w, h, rng(1))
	b := lowBox(h)
	for i := 0; i < 3000; i++ {
		s.Tick()
		if i%7 != 0 {
			continue
		}
		for y := h - Height; y < h; y++ {
			for x := 0; x < w; x++ {
				if r := s.Glyph(x, y, b); r != 0 && ansi.StringWidth(string(r)) != 1 {
					t.Fatalf("glyph %q at (%d,%d) has width %d", r, x, y, ansi.StringWidth(string(r)))
				}
			}
		}
	}
}

func TestHidesWhenBoxTouchesStrip(t *testing.T) {
	const w, h = 100, 30
	s := New(w, h, rng(2))
	b := scene.Box{X0: 20, X1: 60, Y0: 2, Y1: h - Height}
	for i := 0; i < 500; i++ {
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

func TestDrawsOnlyInStripAndShipsAppear(t *testing.T) {
	const w, h = 80, 24
	for seed := uint64(0); seed < 5; seed++ {
		s := New(w, h, rng(seed))
		b := lowBox(h)
		sawShip := false
		for i := 0; i < 2000; i++ {
			s.Tick()
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					r := s.Glyph(x, y, b)
					if y < h-Height && r != 0 {
						t.Fatalf("seed %d: glyph %q above strip at (%d,%d)", seed, r, x, y)
					}
					if y == h-1 && r == 0 {
						t.Fatalf("seed %d: no water at column %d", seed, x)
					}
					if y < h-1 && r != 0 {
						sawShip = true
					}
				}
			}
		}
		if !sawShip {
			t.Fatalf("seed %d: no ship in 2000 ticks", seed)
		}
	}
}

func TestShipsNeverOverlap(t *testing.T) {
	for seed := uint64(0); seed < 30; seed++ {
		s := New(120, 30, rng(seed))
		for i := 0; i < 6000; i++ {
			s.Tick()
			if len(s.ships) > 2 {
				t.Fatalf("seed %d: %d ships at once", seed, len(s.ships))
			}
			if len(s.ships) == 2 {
				a, b := s.ships[0], s.ships[1]
				if a.dir != b.dir {
					t.Fatalf("seed %d: ships sailing opposite ways together", seed)
				}
				if a.x < b.x+b.width() && b.x < a.x+a.width() {
					t.Fatalf("seed %d: ships overlap at tick %d", seed, i)
				}
			}
		}
	}
}

func TestShipArtIsRectangular(t *testing.T) {
	s := New(100, 30, rng(4))
	for i := 0; i < 200; i++ {
		for _, dir := range []int{1, -1} {
			art := s.buildArt(dir)
			if len(art) != shipH {
				t.Fatalf("ship has %d rows", len(art))
			}
			for _, row := range art {
				if len(row) != len(art[0]) {
					t.Fatalf("ragged ship art")
				}
			}
		}
	}
}

func TestResizeAnySize(t *testing.T) {
	s := New(0, 0, rng(5))
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {3, 2}, {200, 60}, {10, 3}, {0, 50}, {-1, -1}} {
		s.Resize(sz[0], sz[1])
		for i := 0; i < 50; i++ {
			s.Tick()
		}
		for y := -1; y <= sz[1]; y++ {
			for x := -1; x <= sz[0]; x++ {
				s.Glyph(x, y, scene.Box{Y0: -1, Y1: -1})
			}
		}
	}
}

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
			art, tint := s.buildArt(dir)
			if len(art) != shipH || len(tint) != shipH {
				t.Fatalf("ship has %d rows, %d tint rows", len(art), len(tint))
			}
			for r, row := range art {
				if len(row) != len(art[0]) || len(tint[r]) != len(row) {
					t.Fatalf("ragged ship art or tint")
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
				s.Color(x, y, scene.Box{Y0: -1, Y1: -1})
			}
		}
	}
}

func TestColors(t *testing.T) {
	allowed := map[scene.Color]bool{scene.Default: true}
	for _, c := range boxColors {
		allowed[c] = true
	}
	const w, h = 120, 30
	for seed := uint64(0); seed < 5; seed++ {
		s := New(w, h, rng(seed))
		b := lowBox(h)
		sawBox := false
		for i := 0; i < 2000; i++ {
			s.Tick()
			for y := h - Height; y < h; y++ {
				for x := 0; x < w; x++ {
					r := s.Glyph(x, y, b)
					if r == 0 {
						continue
					}
					c := s.Color(x, y, b)
					switch {
					case y == h-1 && (r == '~' || r == '-' || r == '_'):
						if c != scene.Blue {
							t.Fatalf("seed %d: water %q at %d is color %d", seed, r, x, c)
						}
					case y-(h-Height) > cargoBot: // deck, hull and keel
						if c != scene.Default {
							t.Fatalf("seed %d: hull %q at (%d,%d) is color %d", seed, r, x, y, c)
						}
					case !allowed[c]:
						t.Fatalf("seed %d: %q at (%d,%d) has color %d", seed, r, x, y, c)
					case c != scene.Default:
						sawBox = true
					}
				}
			}
		}
		if !sawBox {
			t.Fatalf("seed %d: no colored container in 2000 ticks", seed)
		}
	}
}

func TestContainerColorsAreStable(t *testing.T) {
	s := New(120, 30, rng(7))
	for i := 0; i < 50; i++ {
		art, tint := s.buildArt(1 + i%2*-2)
		box := 0
		for r := 0; r <= cargoBot; r++ {
			for c, ch := range art[r] {
				col := tint[r][c]
				if ch == ' ' {
					if col != scene.Default {
						t.Fatalf("blank cell tinted %d", col)
					}
					continue
				}
				if col != scene.Default {
					box++
				}
			}
		}
		if box == 0 {
			t.Fatalf("ship %d has no colored cargo", i)
		}
	}
	// A ship's colors travel with it: the cell under a container keeps
	// its color as the ship sails.
	s = New(120, 30, rng(8))
	for len(s.ships) == 0 {
		s.Tick()
	}
	sh := s.ships[0]
	want := append([][]scene.Color(nil), sh.tint...)
	for i := 0; i < 300 && len(s.ships) > 0 && s.ships[0] == sh; i++ {
		s.Tick()
		for r := range sh.art {
			for c, ch := range sh.art[r] {
				x := sh.x + c
				if ch == ' ' || x < 0 || x >= s.w {
					continue
				}
				if got := s.Color(x, s.h-Height+r, lowBox(s.h)); got != want[r][c] {
					t.Fatalf("tick %d: cell (%d,%d) of ship is %d, want %d", i, r, c, got, want[r][c])
				}
			}
		}
	}
}

func TestBigContainerIsRed(t *testing.T) {
	s := New(120, 30, rng(9))
	found := false
	for i := 0; i < 400 && !found; i++ {
		art, tint := s.buildArt(1)
		for c, ch := range art[cargoBot-1] {
			if ch == '┃' && c+1 < len(art[cargoBot-1]) && art[cargoBot-1][c+1] == '┃' &&
				art[cargoBot][c] == '─' {
				found = true
				if tint[cargoBot-1][c] != scene.Red {
					t.Fatalf("big container is %d, want Red", tint[cargoBot-1][c])
				}
			}
		}
	}
	if !found {
		t.Fatal("no big container built")
	}
}

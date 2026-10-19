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
				s.Color(x, y, scene.Box{Y0: -1, Y1: -1})
			}
		}
	}
}

func TestColors(t *testing.T) {
	allowed := map[scene.Color]bool{scene.Default: true, scene.White: true, locoColor: true, bigColor: true}
	for _, c := range boxColors {
		allowed[c] = true
	}
	const w, h = 100, 30
	for seed := uint64(0); seed < 6; seed++ {
		s := New(w, h, rng(seed))
		b := lowBox(h)
		sawLoco := false
		for i := 0; i < 2000; i++ {
			s.Tick()
			for y := h - Height; y < h; y++ {
				row := y - (h - Height)
				for x := 0; x < w; x++ {
					r := s.Glyph(x, y, b)
					if r == 0 {
						continue
					}
					c := s.Color(x, y, b)
					if !allowed[c] {
						t.Fatalf("seed %d: %q at (%d,%d) has color %d", seed, r, x, y, c)
					}
					if !s.Passing() && c != scene.Default {
						t.Fatalf("seed %d: bare rail %q at %d is color %d", seed, r, x, c)
					}
					if r == '┃' && c != scene.Red {
						t.Fatalf("seed %d: boxcar or big container %q at (%d,%d) is %d, want Red", seed, r, x, y, c)
					}
					if row == 1 && (r == '╭' || r == '╮') && c != scene.White {
						t.Fatalf("seed %d: tanker %q at (%d,%d) is %d, want White", seed, r, x, y, c)
					}
					if row == 2 && (r == '╮' || r == '╭') {
						if c != locoColor {
							t.Fatalf("seed %d: locomotive nose at (%d,%d) is %d", seed, x, y, c)
						}
						sawLoco = true
					}
				}
			}
		}
		if !sawLoco {
			t.Fatalf("seed %d: never saw the locomotive", seed)
		}
	}
}

// TestCarColorsAreStable follows each cell of a train across the screen
// and checks its color never changes.
func TestCarColorsAreStable(t *testing.T) {
	const w, h = 100, 30
	b := lowBox(h)
	for seed := uint64(0); seed < 10; seed++ {
		s := New(w, h, rng(seed))
		for !s.Passing() {
			s.Tick()
		}
		want := make([][]scene.Color, len(s.tint))
		for r := range s.tint {
			want[r] = append([]scene.Color(nil), s.tint[r]...)
		}
		train := s.train
		colored := false
		for s.Passing() {
			for r, row := range train {
				for c, ch := range row {
					x := s.x + c
					if ch == ' ' || x < 0 || x >= w || (r == 0 && s.frame[0][x] != ch) {
						continue // off screen, blank, or under smoke
					}
					got := s.Color(x, h-Height+r, b)
					if got != want[r][c] {
						t.Fatalf("seed %d: train cell (%d,%d) is %d, want %d", seed, r, c, got, want[r][c])
					}
					colored = colored || got != scene.Default
				}
			}
			s.Tick()
		}
		if !colored {
			t.Fatalf("seed %d: train had no color", seed)
		}
	}
}

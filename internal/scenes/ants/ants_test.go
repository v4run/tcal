package ants

import (
	"math/rand/v2"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/scene"
)

func rng(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, seed^0x9e37)) }

// onRing reports whether (x, y) is on the ring of cells just outside b.
func onRing(x, y int, b scene.Box) bool {
	if x < b.X0-1 || x > b.X1 || y < b.Y0-1 || y > b.Y1+1 {
		return false
	}
	return x == b.X0-1 || x == b.X1 || y == b.Y0-1 || y == b.Y1+1
}

// frame collects every drawn cell.
func frame(s *Scene, w, h int, b scene.Box) map[[2]int]rune {
	out := map[[2]int]rune{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if r := s.Glyph(x, y, b); r != 0 {
				out[[2]int{x, y}] = r
			}
		}
	}
	return out
}

func TestAntsStayOnRingAndMarchAround(t *testing.T) {
	const w, h = 100, 30
	boxes := []scene.Box{
		{X0: 30, X1: 70, Y0: 8, Y1: 20},
		{X0: 0, X1: 40, Y0: 0, Y1: 12},    // ring clipped top and left
		{X0: 60, X1: 100, Y0: 15, Y1: 29}, // ring clipped right and bottom
		{X0: 10, X1: 12, Y0: 5, Y1: 5},    // tiny box
	}
	for seed := uint64(0); seed < 5; seed++ {
		for _, b := range boxes {
			s := New(w, h, rng(seed))
			sides := map[string]bool{}
			var prev map[[2]int]rune
			changed := 0
			for i := 0; i < 600; i++ {
				f := frame(s, w, h, b)
				for p, r := range f {
					x, y := p[0], p[1]
					if b.Contains(x, y) || !onRing(x, y, b) {
						t.Fatalf("seed %d box %+v: ant %q at (%d,%d) off the ring", seed, b, r, x, y)
					}
					switch {
					case y == b.Y0-1:
						sides["top"] = true
					case y == b.Y1+1:
						sides["bottom"] = true
					case x == b.X0-1:
						sides["left"] = true
					case x == b.X1:
						sides["right"] = true
					}
				}
				if prev != nil && !sameFrame(prev, f) {
					changed++
				}
				prev = f
				s.Tick()
			}
			if changed < 300 {
				t.Fatalf("seed %d box %+v: ants moved on only %d of 600 ticks", seed, b, changed)
			}
			if b == boxes[0] && len(sides) != 4 {
				t.Fatalf("seed %d: ants only visited %v", seed, sides)
			}
		}
	}
}

func TestEmptyBoxMarchesScreenBorder(t *testing.T) {
	const w, h = 50, 16
	s := New(w, h, rng(7))
	empty := scene.Box{Y0: -1, Y1: -1}
	seen := 0
	for i := 0; i < 300; i++ {
		for p := range frame(s, w, h, empty) {
			x, y := p[0], p[1]
			if x != 0 && y != 0 && x != w-1 && y != h-1 {
				t.Fatalf("ant at (%d,%d) off the screen border", x, y)
			}
			seen++
		}
		s.Tick()
	}
	if seen == 0 {
		t.Fatal("no ants on an empty screen")
	}
}

func TestBoxChangesKeepAntsOnRing(t *testing.T) {
	const w, h = 90, 30
	s := New(w, h, rng(8))
	boxes := []scene.Box{
		{X0: 20, X1: 70, Y0: 5, Y1: 25},
		{Y0: -1, Y1: -1},
		{X0: 40, X1: 50, Y0: 10, Y1: 14},
		{X0: 1, X1: 89, Y0: 1, Y1: 28},
	}
	for i := 0; i < 400; i++ {
		b := boxes[(i/25)%len(boxes)]
		for p := range frame(s, w, h, b) {
			x, y := p[0], p[1]
			ok := onRing(x, y, b)
			if b.Y0 < 0 {
				ok = x == 0 || y == 0 || x == w-1 || y == h-1
			}
			if !ok || (b.Y0 >= 0 && b.Contains(x, y)) {
				t.Fatalf("tick %d box %+v: ant at (%d,%d) off the ring", i, b, x, y)
			}
		}
		s.Tick()
	}
}

func TestGlyphsAreSingleWidth(t *testing.T) {
	const w, h = 80, 24
	s := New(w, h, rng(9))
	b := scene.Box{X0: 20, X1: 60, Y0: 4, Y1: 18}
	for i := 0; i < 500; i++ {
		for _, r := range frame(s, w, h, b) {
			if ansi.StringWidth(string(r)) != 1 {
				t.Fatalf("glyph %q has width %d", r, ansi.StringWidth(string(r)))
			}
		}
		s.Tick()
	}
}

func TestResizeAnySize(t *testing.T) {
	s := New(0, 0, rng(5))
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {3, 2}, {2, 1}, {200, 60}, {10, 3}, {0, 50}, {-1, -1}} {
		s.Resize(sz[0], sz[1])
		for _, b := range []scene.Box{{Y0: -1, Y1: -1}, {X0: 0, X1: max(sz[0], 0), Y0: 0, Y1: max(sz[1]-1, 0)}} {
			for i := 0; i < 50; i++ {
				for y := -1; y <= sz[1]; y++ {
					for x := -1; x <= sz[0]; x++ {
						s.Glyph(x, y, b)
					}
				}
				s.Tick()
			}
		}
	}
}

func sameFrame(a, b map[[2]int]rune) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

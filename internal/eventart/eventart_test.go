package eventart

import (
	"math/rand"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

var emptyBox = scene.Box{X0: 0, X1: 0, Y0: -1, Y1: -1}

func newArt(t *testing.T, k events.Kind) Art {
	t.Helper()
	a := New(k, rand.New(rand.NewSource(1)))
	if a == nil {
		t.Fatalf("New(%v) = nil for registered kind", k)
	}
	now := time.Date(2026, time.December, 25, 12, 0, 0, 0, time.UTC)
	a.SetTime(now, events.Occurrence{Kind: k, Start: now.Add(-12 * time.Hour), End: now.Add(12 * time.Hour)})
	return a
}

func TestAllArt_SingleWidth(t *testing.T) {
	for _, k := range Kinds() {
		for _, sz := range [][2]int{{120, 40}, {80, 24}} {
			a := newArt(t, k)
			a.Resize(sz[0], sz[1])
			for i := 0; i < 300; i++ {
				a.Tick()
				for y := 0; y < sz[1]; y++ {
					for x := 0; x < sz[0]; x++ {
						if r, _ := a.Cell(x, y, emptyBox); r != 0 && ansi.StringWidth(string(r)) != 1 {
							t.Fatalf("%v: rune %q at %d,%d has width %d", k, r, x, y, ansi.StringWidth(string(r)))
						}
					}
				}
			}
		}
	}
}

func TestAllArt_StaysInBounds(t *testing.T) {
	for _, k := range Kinds() {
		a := newArt(t, k)
		a.Resize(120, 40)
		for i := 0; i < 100; i++ {
			a.Tick()
			bd := a.Bounds()
			for y := -2; y < 42; y++ {
				for x := -2; x < 122; x++ {
					if r, _ := a.Cell(x, y, emptyBox); r != 0 {
						if x < bd.X0 || x >= bd.X0+bd.W || y < bd.Y0 || y >= bd.Y0+bd.H {
							t.Fatalf("%v: cell %d,%d outside bounds %+v", k, x, y, bd)
						}
					}
				}
			}
		}
	}
}

func TestAllArt_HidesWhenTouchingBox(t *testing.T) {
	for _, k := range Kinds() {
		for _, sz := range [][2]int{{120, 40}, {60, 20}, {30, 10}} {
			a := newArt(t, k)
			a.Resize(sz[0], sz[1])
			bd := a.Bounds()
			if bd.W == 0 || bd.H == 0 {
				continue
			}
			box := scene.Box{X0: bd.X0, X1: bd.X0 + 1, Y0: bd.Y0, Y1: bd.Y0}
			for i := 0; i < 20; i++ {
				a.Tick()
				for y := 0; y < sz[1]; y++ {
					for x := 0; x < sz[0]; x++ {
						if r, _ := a.Cell(x, y, box); r != 0 {
							t.Fatalf("%v %v: drew %q at %d,%d while touching box", k, sz, r, x, y)
						}
					}
				}
			}
		}
	}
}

func TestAllArt_TinySizesNoPanic(t *testing.T) {
	for _, k := range Kinds() {
		a := newArt(t, k)
		for _, sz := range [][2]int{{0, 0}, {1, 1}, {3, 2}, {-1, -1}} {
			a.Resize(sz[0], sz[1])
			a.Tick()
			_ = a.Bounds()
			for y := -3; y < 6; y++ {
				for x := -3; x < 6; x++ {
					a.Cell(x, y, emptyBox)
					a.Cell(x, y, scene.Box{X0: 0, X1: 2, Y0: 0, Y1: 1})
				}
			}
			a.Cell(1<<30, -(1 << 30), emptyBox)
		}
	}
}

func TestAllArt_DrawsSomething(t *testing.T) {
	for _, k := range Kinds() {
		a := newArt(t, k)
		a.Resize(120, 40)
		a.Tick()
		n := 0
		for y := 0; y < 40; y++ {
			for x := 0; x < 120; x++ {
				if r, _ := a.Cell(x, y, emptyBox); r != 0 {
					n++
				}
			}
		}
		if n == 0 {
			t.Errorf("%v draws nothing at 120x40", k)
		}
	}
}

func TestChristmas_StarYellowTreeGreen(t *testing.T) {
	a := newArt(t, events.Christmas)
	a.Resize(120, 40)
	a.Tick()
	bd := a.Bounds()
	stars, green := 0, 0
	for y := bd.Y0; y < bd.Y0+bd.H; y++ {
		for x := bd.X0; x < bd.X0+bd.W; x++ {
			r, c := a.Cell(x, y, emptyBox)
			switch {
			case r == '*':
				stars++
				if c != scene.Yellow {
					t.Errorf("star color = %v, want Yellow", c)
				}
			case r == '/' || r == '\\':
				if c != scene.Green {
					t.Errorf("tree %q color = %v, want Green", r, c)
				}
				green++
			}
		}
	}
	if stars != 1 || green == 0 {
		t.Errorf("stars=%d green=%d", stars, green)
	}
}

func TestRegion_Touches(t *testing.T) {
	r := Region{X0: 10, Y0: 10, W: 5, H: 5}
	cases := []struct {
		b    scene.Box
		want bool
	}{
		{emptyBox, false},
		{scene.Box{X0: 0, X1: 10, Y0: 0, Y1: 30}, false},
		{scene.Box{X0: 0, X1: 11, Y0: 0, Y1: 30}, true},
		{scene.Box{X0: 14, X1: 20, Y0: 14, Y1: 14}, true},
		{scene.Box{X0: 15, X1: 20, Y0: 0, Y1: 30}, false},
		{scene.Box{X0: 0, X1: 30, Y0: 15, Y1: 20}, false},
	}
	for i, c := range cases {
		if got := r.Touches(c.b); got != c.want {
			t.Errorf("case %d: Touches = %v, want %v", i, got, c.want)
		}
	}
	if (Region{}).Touches(scene.Box{X0: 0, X1: 5, Y0: 0, Y1: 5}) {
		t.Error("empty region touches")
	}
}

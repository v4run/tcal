package eventart

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

var emptyBox = scene.Box{X0: 0, X1: 0, Y0: -1, Y1: -1}

func newArt(t *testing.T, k events.Kind) Art {
	t.Helper()
	a := New(k, rand.New(rand.NewPCG(1, 2)))
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

// typicalBox is a centered ~22x64 calendar (plus padding) at 120x40.
var typicalBox = scene.Box{X0: 27, X1: 93, Y0: 8, Y1: 31}

func TestAllArt_StaysInBounds(t *testing.T) {
	for _, k := range Kinds() {
		for _, sz := range [][2]int{{80, 24}, {120, 40}, {200, 60}, {97, 31}} {
			a := newArt(t, k)
			a.Resize(sz[0], sz[1])
			if sz == [2]int{200, 60} {
				if bd := a.Bounds(); bd.W <= 0 || bd.H <= 0 {
					t.Fatalf("%v: empty bounds %+v at 200x60", k, bd)
				}
			}
			for i := 0; i < 100; i++ {
				a.Tick()
				bd := a.Bounds()
				if bd.W > 0 && (bd.X0 < 0 || bd.Y0 < 0 || bd.X0+bd.W > sz[0] || bd.Y0+bd.H > sz[1]) {
					t.Fatalf("%v %v: bounds %+v off screen", k, sz, bd)
				}
				for y := -2; y < sz[1]+2; y++ {
					for x := -2; x < sz[0]+2; x++ {
						if r, _ := a.Cell(x, y, emptyBox); r != 0 {
							if x < bd.X0 || x >= bd.X0+bd.W || y < bd.Y0 || y >= bd.Y0+bd.H {
								t.Fatalf("%v %v: cell %d,%d outside bounds %+v", k, sz, x, y, bd)
							}
						}
					}
				}
			}
		}
	}
}

// touchingBoxes returns boxes that each overlap bd by exactly one cell at
// a corner or edge midpoint, extending away from bd.
func touchingBoxes(bd Region) []scene.Box {
	x1, y1 := bd.X0+bd.W-1, bd.Y0+bd.H-1
	mx, my := bd.X0+bd.W/2, bd.Y0+bd.H/2
	cell := func(x, y, dx, dy int) scene.Box {
		b := scene.Box{X0: x, X1: x + 1, Y0: y, Y1: y}
		if dx < 0 {
			b.X0 = x - 5
		} else if dx > 0 {
			b.X1 = x + 6
		}
		if dy < 0 {
			b.Y0 = max(y-5, 0)
		} else if dy > 0 {
			b.Y1 = y + 5
		}
		return b
	}
	return []scene.Box{
		cell(bd.X0, bd.Y0, -1, -1), cell(mx, bd.Y0, 0, -1), cell(x1, bd.Y0, 1, -1),
		cell(x1, my, 1, 0), cell(x1, y1, 1, 1), cell(mx, y1, 0, 1),
		cell(bd.X0, y1, -1, 1), cell(bd.X0, my, -1, 0),
	}
}

func TestAllArt_HidesWhenTouchingBox(t *testing.T) {
	for _, k := range Kinds() {
		for _, sz := range [][2]int{{120, 40}, {200, 60}, {60, 20}, {30, 10}} {
			a := newArt(t, k)
			a.Resize(sz[0], sz[1])
			bd := a.Bounds()
			if bd.W == 0 || bd.H == 0 {
				if sz[0] >= 120 {
					t.Fatalf("%v: empty bounds at %v", k, sz)
				}
				continue
			}
			for _, box := range touchingBoxes(bd) {
				if !bd.Touches(box) {
					t.Fatalf("%v %v: probe %+v does not touch %+v", k, sz, box, bd)
				}
				for i := 0; i < 20; i++ {
					a.Tick()
					for y := 0; y < sz[1]; y++ {
						for x := 0; x < sz[0]; x++ {
							if r, _ := a.Cell(x, y, box); r != 0 {
								t.Fatalf("%v %v: drew %q at %d,%d while touching box %+v", k, sz, r, x, y, box)
							}
						}
					}
				}
			}
		}
	}
}

// Every art must fit the margins of a typical terminal around a centered
// calendar, or users would rarely see it.
func TestAllArt_FitsTypicalMargins(t *testing.T) {
	for _, k := range Kinds() {
		a := newArt(t, k)
		a.Resize(120, 40)
		bd := a.Bounds()
		if bd.W <= 0 || bd.H <= 0 {
			t.Fatalf("%v: empty bounds at 120x40", k)
		}
		if bd.Touches(typicalBox) {
			t.Errorf("%v: bounds %+v touch a typical calendar %+v", k, bd, typicalBox)
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

// busy counts drawn cells summed over n ticks.
func busy(a Art, w, h, n int) int {
	c := 0
	for i := 0; i < n; i++ {
		a.Tick()
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if r, _ := a.Cell(x, y, emptyBox); r != 0 {
					c++
				}
			}
		}
	}
	return c
}

func TestNewYear_MoreBurstsAtMidnight(t *testing.T) {
	occ := events.Occurrence{Kind: events.NewYear,
		Start: time.Date(2026, time.December, 31, 18, 0, 0, 0, time.UTC),
		End:   time.Date(2027, time.January, 2, 0, 0, 0, 0, time.UTC)}
	at := func(hour, min int) int {
		a := newArt(t, events.NewYear)
		a.Resize(120, 40)
		a.SetTime(time.Date(2026, time.December, 31, hour, min, 0, 0, time.UTC), occ)
		return busy(a, 120, 40, 200)
	}
	mid, eve := at(23, 59), at(20, 0)
	if mid <= 2*eve {
		t.Errorf("midnight %d cells, 20:00 %d cells; want midnight > 2x", mid, eve)
	}
	if eve == 0 {
		t.Error("no fireworks at all at 20:00")
	}
}

func TestDiwali_FlamesFlicker(t *testing.T) {
	a := newArt(t, events.Diwali)
	a.Resize(120, 40)
	a.Tick()
	bd := a.Bounds()
	snap := func() (string, int) {
		var s []rune
		flames := 0
		for y := bd.Y0 + bd.H - 2; y < bd.Y0+bd.H; y++ {
			for x := bd.X0; x < bd.X0+bd.W; x++ {
				r, c := a.Cell(x, y, emptyBox)
				s = append(s, r)
				if r != 0 && c == scene.Yellow {
					flames++
				}
			}
		}
		return string(s), flames
	}
	first, flames := snap()
	if flames < 5 || flames > 9 {
		t.Fatalf("%d flames, want 5-9", flames)
	}
	for i := 0; i < 10; i++ {
		a.Tick()
		if s, _ := snap(); s != first {
			return
		}
	}
	t.Error("no flame changed within 10 ticks")
}

func TestLanterns_Sway(t *testing.T) {
	a := newArt(t, events.LunarNewYear)
	a.Resize(120, 40)
	bd := a.Bounds()
	leftmost := func() int {
		for x := bd.X0; x < bd.X0+bd.W; x++ {
			for y := bd.Y0; y < bd.Y0+bd.H; y++ {
				if r, c := a.Cell(x, y, emptyBox); r != 0 && c == scene.Red {
					return x
				}
			}
		}
		return -1
	}
	a.Tick()
	x0 := leftmost()
	if x0 < 0 {
		t.Fatal("no red lantern drawn")
	}
	for i := 0; i < 40; i++ {
		a.Tick()
		if leftmost() != x0 {
			return
		}
	}
	t.Error("lanterns did not sway within 40 ticks")
}

func TestEid_MoonWhite(t *testing.T) {
	a := newArt(t, events.Eid)
	a.Resize(120, 40)
	a.Tick()
	moon, lantern := 0, 0
	for y := 0; y < 40; y++ {
		for x := 0; x < 120; x++ {
			r, c := a.Cell(x, y, emptyBox)
			switch {
			case r == 0:
			case r >= 0x2800 && r <= 0x28FF:
				moon++
				if c != scene.White {
					t.Errorf("moon %q at %d,%d color %v, want White", r, x, y, c)
				}
			case c == scene.Yellow:
				lantern++
			default:
				t.Errorf("%q at %d,%d color %v, want White moon or Yellow lantern", r, x, y, c)
			}
		}
	}
	if moon == 0 || lantern == 0 {
		t.Errorf("moon cells %d, lantern cells %d", moon, lantern)
	}
}

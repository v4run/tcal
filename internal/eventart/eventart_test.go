package eventart

import (
	"math/rand/v2"
	"sort"
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

func TestKinds_AllThirteen(t *testing.T) {
	if got := len(Kinds()); got != 13 {
		t.Errorf("Kinds() has %d kinds, want 13: %v", got, Kinds())
	}
}

// onamAt returns the Onam art at 120x40 on the given day of a days-long
// occurrence, and the number of cells it draws.
func onamAt(t *testing.T, day, days int) (Art, int) {
	t.Helper()
	start := time.Date(2026, time.August, 26, 0, 0, 0, 0, time.UTC)
	occ := events.Occurrence{Kind: events.Onam, Start: start, End: start.AddDate(0, 0, days)}
	a := newArt(t, events.Onam)
	a.Resize(120, 40)
	a.SetTime(start.AddDate(0, 0, day).Add(12*time.Hour), occ)
	a.Tick()
	n := 0
	for y := 0; y < 40; y++ {
		for x := 0; x < 120; x++ {
			if r, _ := a.Cell(x, y, emptyBox); r != 0 {
				n++
			}
		}
	}
	return a, n
}

// pookalamCells is the number of cells in the complete pookalam.
func pookalamCells() int {
	n := 0
	for _, row := range pookalamRing {
		for _, k := range row {
			if k >= 0 {
				n++
			}
		}
	}
	return n
}

func TestOnam_RingsGrowWithDay(t *testing.T) {
	_, d0 := onamAt(t, 0, 10)
	_, d4 := onamAt(t, 4, 10)
	_, d9 := onamAt(t, 9, 10)
	if !(d0 < d4 && d4 < d9) {
		t.Errorf("cells by day: day0 %d, day4 %d, day9 %d; want strictly growing", d0, d4, d9)
	}
	if full := pookalamCells(); d9 != full {
		t.Errorf("last day draws %d cells, want the full pookalam's %d", d9, full)
	}
}

func TestOnam_NineAndElevenDayYears(t *testing.T) {
	full := pookalamCells()
	for _, days := range []int{9, 10, 11} {
		for day := 0; day < days; day++ {
			_, n := onamAt(t, day, days)
			if last := day == days-1; last != (n == full) {
				t.Errorf("%d-day Onam, day %d: %d cells, full is %d", days, day, n, full)
			}
		}
	}
}

// snapshot returns the runes of the art's drawn cells that keep accepts, in
// reading order.
func snapshot(a Art, keep func(rune, scene.Color) bool) string {
	bd := a.Bounds()
	var s []rune
	for y := bd.Y0; y < bd.Y0+bd.H; y++ {
		for x := bd.X0; x < bd.X0+bd.W; x++ {
			if r, c := a.Cell(x, y, emptyBox); r != 0 && keep(r, c) {
				s = append(s, r)
			}
		}
	}
	return string(s)
}

func TestHalloween_BatsFlap(t *testing.T) {
	a := newArt(t, events.Halloween)
	a.Resize(120, 40)
	a.Tick()
	// Bat glyphs, ignoring where the bats are.
	bats := func() string {
		s := []rune(snapshot(a, func(_ rune, c scene.Color) bool { return c == scene.Default }))
		sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
		return string(s)
	}
	first := bats()
	if first == "" {
		t.Fatal("no bats drawn")
	}
	if p := snapshot(a, func(_ rune, c scene.Color) bool { return c == scene.Yellow }); p == "" {
		t.Fatal("no pumpkin drawn")
	}
	for i := 0; i < 10; i++ {
		a.Tick()
		if bats() != first {
			return
		}
	}
	t.Error("no bat flapped within 10 ticks")
}

func TestBirthday_CandlesFlicker(t *testing.T) {
	a := newArt(t, events.Birthday)
	a.Resize(120, 40)
	a.Tick()
	bd := a.Bounds()
	// The flames are the yellow cells in the cake's rows at the bottom.
	flames := func() string {
		var s []rune
		for y := bd.Y0 + bd.H - cakeH; y < bd.Y0+bd.H; y++ {
			for x := bd.X0; x < bd.X0+bd.W; x++ {
				if r, c := a.Cell(x, y, emptyBox); r != 0 && c == scene.Yellow {
					s = append(s, r)
				}
			}
		}
		return string(s)
	}
	first := flames()
	if n := len([]rune(first)); n < 3 || n > 5 {
		t.Fatalf("%d flames, want 3-5", n)
	}
	for i := 0; i < 10; i++ {
		a.Tick()
		if flames() != first {
			return
		}
	}
	t.Error("no candle flickered within 10 ticks")
}

func TestKites_HaveStrings(t *testing.T) {
	a := newArt(t, events.Sankranti)
	a.Resize(120, 40)
	bd := a.Bounds()
	for i := 0; i < 60; i++ {
		a.Tick()
		kites := 0
		for y := bd.Y0; y < bd.Y0+bd.H-1; y++ {
			for x := bd.X0; x < bd.X0+bd.W-1; x++ {
				r, c := a.Cell(x, y, emptyBox)
				r2, _ := a.Cell(x+1, y, emptyBox)
				r3, _ := a.Cell(x, y+1, emptyBox)
				if r != '/' || r2 != '\\' || r3 != '\\' || c == scene.Default {
					continue
				}
				kites++
				strung := false
				for yy := y + 2; yy < bd.Y0+bd.H; yy++ {
					if r, c := a.Cell(x, yy, emptyBox); r != 0 && c == scene.Default {
						strung = true
						break
					}
				}
				if !strung {
					t.Fatalf("tick %d: kite at %d,%d has no string below it", i, x, y)
				}
			}
		}
		if kites < 2 || kites > 4 {
			t.Fatalf("tick %d: %d kites, want 2-4", i, kites)
		}
	}
}

package eventart

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/v4run/tcal/internal/dots"
	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

const dt = 33 * time.Millisecond

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

func TestNew_UnknownKindIsNil(t *testing.T) {
	if a := New(events.Kind(-1), rand.New(rand.NewPCG(1, 2))); a != nil {
		t.Fatalf("New(unknown) = %T, want nil", a)
	}
}

func TestKinds_FirstSix(t *testing.T) {
	want := []events.Kind{events.NewYear, events.LunarNewYear, events.Holi, events.Eid, events.Diwali, events.Christmas}
	got := map[events.Kind]bool{}
	for _, k := range Kinds() {
		got[k] = true
	}
	if len(got) != len(want) {
		t.Fatalf("Kinds() = %v, want %v", Kinds(), want)
	}
	for _, k := range want {
		if !got[k] {
			t.Errorf("Kinds() lacks %v", k)
		}
	}
}

func TestAllArt_SingleWidth(t *testing.T) {
	for _, k := range Kinds() {
		for _, sz := range [][2]int{{120, 40}, {80, 24}} {
			a := newArt(t, k)
			a.Resize(sz[0], sz[1])
			for i := 0; i < 300; i++ {
				a.Step(dt)
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
				a.Step(dt)
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
			probes := touchingBoxes(bd)
			if len(probes) != 8 {
				t.Fatalf("%d probes, want 8", len(probes))
			}
			for _, box := range probes {
				if !bd.Touches(box) {
					t.Fatalf("%v %v: probe %+v does not touch %+v", k, sz, box, bd)
				}
				for i := 0; i < 20; i++ {
					a.Step(dt)
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

// A calendar starting at screen row 0 has Y0 == -1 (one row above it) but a
// real Y1; it must hide the art, not be mistaken for the empty box.
func TestAllArt_HidesWhenCalendarAtRowZero(t *testing.T) {
	for _, k := range Kinds() {
		a := newArt(t, k)
		a.Resize(120, 40)
		bd := a.Bounds()
		if bd.W == 0 || bd.H == 0 {
			t.Fatalf("%v: empty bounds", k)
		}
		box := scene.Box{X0: bd.X0, X1: bd.X0 + bd.W, Y0: -1, Y1: bd.Y0 + bd.H}
		if !bd.Touches(box) {
			t.Fatalf("%v: row-0 box %+v does not touch %+v", k, box, bd)
		}
		for i := 0; i < 5; i++ {
			a.Step(dt)
			for y := 0; y < 40; y++ {
				for x := 0; x < 120; x++ {
					if r, _ := a.Cell(x, y, box); r != 0 {
						t.Fatalf("%v: drew %q at %d,%d beside a row-0 calendar", k, r, x, y)
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
			a.Step(dt)
			a.Step(-dt)
			a.Step(time.Hour)
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
		a.Step(dt)
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
		{scene.Box{X0: 0, X1: 30, Y0: -1, Y1: 12}, true}, // calendar at row 0
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

// litDots returns every lit dot of the art's frame as region-local (column,
// row) dot coordinates, keeping only cells whose colour keep accepts.
func litDots(f *frame, keep func(scene.RGB) bool) [][2]int {
	var out [][2]int
	w, h := f.c.Size()
	for cy := range h {
		for cx := range w {
			g := f.c.Glyph(cx, cy)
			if g < 0x2800 || g > 0x28ff || (keep != nil && !keep(f.c.RGB(cx, cy))) {
				continue
			}
			for r := range 4 {
				for c := range 2 {
					if int(g-0x2800)&dotBits[r][c] != 0 {
						out = append(out, [2]int{cx*2 + c, cy*4 + r})
					}
				}
			}
		}
	}
	return out
}

var dotBits = [4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

func is(c scene.RGB) func(scene.RGB) bool { return func(x scene.RGB) bool { return x == c } }

// lowest returns the lowest of ds (the largest row), or ok=false.
func lowest(ds [][2]int) (d [2]int, ok bool) {
	for _, p := range ds {
		if !ok || p[1] > d[1] {
			d, ok = p, true
		}
	}
	return d, ok
}

// near reports whether some dot of ds is within one dot of (x, y).
func near(ds [][2]int, x, y int) bool {
	for _, d := range ds {
		if abs(d[0]-x) <= 1 && abs(d[1]-y) <= 1 {
			return true
		}
	}
	return false
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// TestAllArt_Glides checks each art with steady motion: its tracked element
// stays within one dot of the analytic position at every one of 100 steps.
// Christmas only twinkles, Diwali's flames flicker at random and Holi's
// powder is a random scatter, so their motion is covered by the fireworks
// (shared with Diwali) and the lantern sway.
func TestAllArt_Glides(t *testing.T) {
	t.Run("lunar new year lantern", func(t *testing.T) {
		a := newArt(t, events.LunarNewYear).(*lunarNewYear)
		a.Resize(120, 40)
		a.n = 1
		a.Step(0)
		ax := lanternAnchor(0, 1, a.f.r.W)
		moved := false
		var first int
		for i := 1; i <= 100; i++ {
			a.Step(dt)
			tt := float64(i) * dt.Seconds()
			off := swayAmp * math.Sin(2*math.Pi*tt/swayPeriod+a.phase[0])
			wx := int(math.Floor(2 * (ax + off)))
			tip, ok := lowest(litDots(&a.f, is(lanternGold)))
			if !ok {
				t.Fatalf("step %d: no tassel", i)
			}
			if abs(tip[0]-wx) > 1 {
				t.Fatalf("step %d: tassel dot %v, ideal column %d", i, tip, wx)
			}
			if i == 1 {
				first = tip[0]
			}
			moved = moved || tip[0] != first
		}
		if !moved {
			t.Error("lantern never moved")
		}
	})
	t.Run("eid lantern", func(t *testing.T) {
		a := newArt(t, events.Eid).(*eid)
		a.Resize(120, 40)
		a.n = 1
		a.Step(0)
		ax := fanousAnchor(0)
		moved := false
		var first int
		for i := 1; i <= 100; i++ {
			a.Step(dt)
			tt := float64(i) * dt.Seconds()
			off := swayAmp * math.Sin(2*math.Pi*tt/swayPeriod+a.phase[0])
			wx := int(math.Floor(2 * (ax + off)))
			tip, ok := lowest(litDots(&a.f, is(fanousCol)))
			if !ok {
				t.Fatalf("step %d: no lantern", i)
			}
			if abs(tip[0]-wx) > 1 {
				t.Fatalf("step %d: lantern tip %v, ideal column %d", i, tip, wx)
			}
			if i == 1 {
				first = tip[0]
			}
			moved = moved || tip[0] != first
		}
		if !moved {
			t.Error("lantern never moved")
		}
	})
	t.Run("new year burst", func(t *testing.T) {
		a := newArt(t, events.NewYear).(*newYear)
		a.Resize(120, 40)
		a.rate, a.shot = 0, true
		fw := a.fw
		// Long enough for 100 steps, with the droop staying inside the band.
		fw.grow, fw.fade = 1300*time.Millisecond, 2100*time.Millisecond
		const cx, cy, r = 50.0, 1.0, 2.0
		fw.bursts = []burst{{x: cx, y: cy, r: r, col: fw.cols[0]}}
		a.Step(0)
		for i := 1; i <= 100; i++ {
			a.Step(dt)
			age := time.Duration(i) * dt
			pr := r * fw.ease(age)
			droop := fw.droop(age)
			wx := int(math.Floor(2 * (cx + 2*pr)))
			wy := int(math.Floor(4 * (cy + droop)))
			ds := litDots(&a.f, nil)
			if len(ds) == 0 {
				t.Fatalf("step %d: burst gone", i)
			}
			right := ds[0][0]
			for _, d := range ds {
				right = max(right, d[0])
			}
			if abs(right-wx) > 1 || !near(ds, wx, wy) {
				t.Fatalf("step %d: rightmost column %d, ideal spark (%d, %d)", i, right, wx, wy)
			}
		}
	})
}

// busy counts drawn cells summed over n steps.
func busy(a Art, w, h, n int) int {
	c := 0
	for i := 0; i < n; i++ {
		a.Step(dt)
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
	if after := at(0, 20); after <= 2*eve {
		t.Errorf("00:20 %d cells, 20:00 %d cells; want the peak to last past midnight", after, eve)
	}
}

func TestDiwali_FlamesFlicker(t *testing.T) {
	a := newArt(t, events.Diwali)
	a.Resize(120, 40)
	a.Step(dt)
	bd := a.Bounds()
	snap := func() (string, int) {
		var s []rune
		flames := 0
		for y := bd.Y0 + bd.H - 2; y < bd.Y0+bd.H; y++ {
			for x := bd.X0; x < bd.X0+bd.W; x++ {
				r, c := a.Cell(x, y, emptyBox)
				s = append(s, r)
				if r != 0 && c != diyaCol {
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
		a.Step(dt)
		if s, _ := snap(); s != first {
			return
		}
	}
	t.Error("no flame changed within 10 steps")
}

func TestLanterns_Sway(t *testing.T) {
	a := newArt(t, events.LunarNewYear)
	a.Resize(120, 40)
	f := &a.(*lunarNewYear).f
	leftmost := func() int {
		x := -1
		for _, d := range litDots(f, is(lanternRed)) {
			if x < 0 || d[0] < x {
				x = d[0]
			}
		}
		return x
	}
	a.Step(dt)
	x0 := leftmost()
	if x0 < 0 {
		t.Fatal("no red lantern drawn")
	}
	for i := 0; i < 40; i++ {
		a.Step(dt)
		if leftmost() != x0 {
			return
		}
	}
	t.Error("lanterns did not sway within 40 steps")
}

func TestEid_MoonWhite(t *testing.T) {
	a := newArt(t, events.Eid)
	a.Resize(120, 40)
	a.Step(dt)
	moon, lantern := 0, 0
	for y := 0; y < 40; y++ {
		for x := 0; x < 120; x++ {
			r, c := a.Cell(x, y, emptyBox)
			switch {
			case r == 0:
			case r < 0x2800 || r > 0x28FF:
				t.Errorf("%q at %d,%d is not braille", r, x, y)
			case c == moonCol:
				moon++
			case c == fanousCol:
				lantern++
			default:
				t.Errorf("%q at %d,%d colour %v, want moon or lantern", r, x, y, c)
			}
		}
	}
	if moon < 10 || lantern == 0 {
		t.Errorf("moon cells %d, lantern cells %d", moon, lantern)
	}
	if lum := int(moonCol.R) + int(moonCol.G) + int(moonCol.B); lum < 600 {
		t.Errorf("moon colour %v is not near white", moonCol)
	}
}

func TestChristmas_StarTreeLights(t *testing.T) {
	a := newArt(t, events.Christmas)
	a.Resize(120, 40)
	a.Step(dt)
	bd := a.Bounds()
	stars, tree, trunk, lights := 0, 0, 0, map[[2]int]bool{}
	for y := bd.Y0; y < bd.Y0+bd.H; y++ {
		for x := bd.X0; x < bd.X0+bd.W; x++ {
			r, c := a.Cell(x, y, emptyBox)
			switch {
			case r == 0:
			case r == '*':
				stars++
				if c != starCol {
					t.Errorf("star colour %v, want %v", c, starCol)
				}
			case c == treeCol:
				tree++
			case c == trunkCol:
				trunk++
			default:
				lights[[2]int{x, y}] = true
			}
		}
	}
	if stars != 1 || tree < 20 || trunk == 0 || len(lights) < 6 {
		t.Fatalf("stars %d, tree %d, trunk %d, lights %d", stars, tree, trunk, len(lights))
	}
	if bd.X0+bd.W != 120-treeMargin || bd.Y0+bd.H != 40-treeMargin {
		t.Errorf("bounds %+v not in the bottom-right corner", bd)
	}
	// The lights twinkle: some light's colour changes within 2 s.
	before := map[[2]int]scene.RGB{}
	for p := range lights {
		_, before[p] = a.Cell(p[0], p[1], emptyBox)
	}
	for range 60 {
		a.Step(dt)
		for p, c := range before {
			if _, now := a.Cell(p[0], p[1], emptyBox); now != c {
				return
			}
		}
	}
	t.Error("no light changed within 2 s")
}

// Holi launches a burst at least every 1.5 s, several times as often as the
// classic art's one every few seconds.
func TestHoli_Busier(t *testing.T) {
	a := newArt(t, events.Holi).(*holi)
	a.Resize(120, 40)
	last, n, prev := time.Duration(0), 0, a.launched
	var el time.Duration
	for el < 15*time.Second {
		a.Step(dt)
		el += dt
		if a.launched != prev {
			if gap := el - last; n > 0 && gap > 1500*time.Millisecond+dt {
				t.Fatalf("gap %v between bursts", gap)
			}
			n += a.launched - prev
			prev, last = a.launched, el
		}
	}
	if n < 10 {
		t.Errorf("%d bursts in 15 s, want at least 10", n)
	}
	if el-last > 1500*time.Millisecond+dt {
		t.Errorf("no burst in the last %v", el-last)
	}
	busyCells := 0
	for range 30 {
		busyCells += busy(a, 120, 40, 1)
		if busyCells > 0 {
			break
		}
	}
	if busyCells == 0 {
		t.Error("holi draws nothing")
	}
}

// Fades and colours are true colour: a burst's sparks thin out as it ages.
func TestFireworks_FadeThinsSparks(t *testing.T) {
	fw := newFireworks(rand.New(rand.NewPCG(1, 2)), 2, scene.RGB{R: 200})
	fw.resize(40, 6)
	fw.bursts = []burst{{x: 20, y: 3, r: 2, col: fw.cols[0]}}
	var c dots.Canvas
	count := func() int {
		c.Reset(40, 6)
		fw.draw(&c, 0, 0)
		n := 0
		for y := range 6 {
			for x := range 40 {
				if c.Glyph(x, y) != 0 {
					n++
				}
			}
		}
		return n
	}
	fw.bursts[0].age = fw.grow
	full := count()
	fw.bursts[0].age = fw.grow + fw.fade*3/4
	late := count()
	if full == 0 || late >= full {
		t.Errorf("cells at full %d, late in fade %d; want fewer late", full, late)
	}
}

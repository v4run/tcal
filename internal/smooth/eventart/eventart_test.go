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

func TestKinds_AllThirteen(t *testing.T) {
	want := []events.Kind{
		events.NewYear, events.Sankranti, events.LunarNewYear, events.Valentines, events.Holi,
		events.Vishu, events.Eid, events.IndependenceDay, events.Onam, events.Halloween,
		events.Diwali, events.Christmas, events.Birthday,
	}
	got := Kinds()
	if len(got) != 13 || len(got) != len(want) {
		t.Fatalf("Kinds() = %v, want all 13: %v", got, want)
	}
	for i, k := range want {
		if got[i] != k {
			t.Errorf("Kinds()[%d] = %v, want %v", i, got[i], k)
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

// A burst launched anywhere in the area keeps every spark inside it, even
// after drooping to the end of its fade.
func TestFireworks_SparksStayInArea(t *testing.T) {
	for _, h := range []int{2, 3, 4, 6} {
		fw := newFireworks(rand.New(rand.NewPCG(3, 4)), 2, scene.RGB{R: 200})
		fw.resize(40, h)
		var c dots.Canvas
		launched := 0
		for i := 0; i < 3000; i++ {
			before := len(fw.bursts)
			fw.step(dt, 0.3)
			if len(fw.bursts) > before {
				launched++
			}
			for _, b := range fw.bursts {
				if low := b.y + b.r + fw.droop(fw.grow+fw.fade); low >= float64(h) {
					t.Fatalf("h=%d: burst at y %.2f r %.2f droops to %.2f, past the area", h, b.y, b.r, low)
				}
				if b.y-b.r < 0 {
					t.Fatalf("h=%d: burst at y %.2f r %.2f pokes above the area", h, b.y, b.r)
				}
			}
			c.Reset(40, h+3)
			fw.draw(&c, 0, 0)
			for y := h; y < h+3; y++ {
				for x := range 40 {
					if c.Glyph(x, y) != 0 {
						t.Fatalf("h=%d step %d: spark at row %d, below the area", h, i, y)
					}
				}
			}
		}
		if launched == 0 {
			t.Errorf("h=%d: no bursts launched", h)
		}
	}
}

// onamAt returns the Onam art at 120x40 on the given day of a days-long
// occurrence, and the number of dots it draws.
func onamAt(t *testing.T, day, days int) (*onam, int) {
	t.Helper()
	start := time.Date(2026, time.August, 26, 0, 0, 0, 0, time.UTC)
	occ := events.Occurrence{Kind: events.Onam, Start: start, End: start.AddDate(0, 0, days)}
	a := newArt(t, events.Onam).(*onam)
	a.Resize(120, 40)
	a.SetTime(start.AddDate(0, 0, day).Add(12*time.Hour), occ)
	a.Step(dt)
	return a, len(litDots(&a.f, nil))
}

// pookalamDots is the number of dots in the complete pookalam.
func pookalamDots() int {
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
	if !(0 < d0 && d0 < d4 && d4 < d9) {
		t.Errorf("dots by day: day0 %d, day4 %d, day9 %d; want strictly growing", d0, d4, d9)
	}
	if full := pookalamDots(); d9 != full {
		t.Errorf("last day draws %d dots, want the full pookalam's %d", d9, full)
	}
	if onamRings < 11 {
		t.Errorf("%d rings, want at least 11 so no day before the last is full", onamRings)
	}
}

func TestOnam_NineAndElevenDayYears(t *testing.T) {
	full := pookalamDots()
	for _, days := range []int{9, 10, 11} {
		prev := 0
		for day := 0; day < days; day++ {
			a, n := onamAt(t, day, days)
			if want := ((day+1)*onamRings + days - 1) / days; a.rings != want {
				t.Errorf("%d-day Onam, day %d: %d rings, want %d", days, day, a.rings, want)
			}
			if last := day == days-1; last != (n == full) {
				t.Errorf("%d-day Onam, day %d: %d dots, full is %d", days, day, n, full)
			}
			if n < prev {
				t.Errorf("%d-day Onam, day %d: %d dots, fewer than the day before (%d)", days, day, n, prev)
			}
			prev = n
		}
	}
}

func TestHalloween_BatsFlap(t *testing.T) {
	a := newArt(t, events.Halloween).(*halloween)
	a.Resize(120, 40)
	a.Step(dt)
	// Per-cell bat glyphs (bat-coloured cells only). Comparing cells, not
	// sorted runes, so two bats in opposite wing phase cannot cancel out.
	bd := a.Bounds()
	bats := func() map[[2]int]rune {
		m := map[[2]int]rune{}
		for y := bd.Y0; y < bd.Y0+bd.H; y++ {
			for x := bd.X0; x < bd.X0+bd.W; x++ {
				if r, c := a.Cell(x, y, emptyBox); r != 0 && c == batCol {
					m[[2]int{x, y}] = r
				}
			}
		}
		return m
	}
	same := func(p, q map[[2]int]rune) bool {
		if len(p) != len(q) {
			return false
		}
		for pos, r := range p {
			if q[pos] != r {
				return false
			}
		}
		return true
	}
	first := bats()
	if len(first) == 0 {
		t.Fatal("no bats drawn")
	}
	if len(litDots(&a.f, is(pumpkinCol))) < 40 {
		t.Fatal("no pumpkin drawn")
	}
	changed := false
	for i := 0; i < 36 && !changed; i++ {
		a.Step(dt)
		changed = !same(first, bats())
	}
	if !changed {
		t.Error("no bat cell changed within 36 steps")
	}
	// Held in place, the bats still flap: their cells cycle through more
	// than one shape and come back round to the first.
	a.speed = 0
	a.Step(dt)
	still := bats()
	shapes, back := 1, false
	prev := still
	for i := 0; i < 60; i++ {
		a.Step(dt)
		cur := bats()
		if !same(cur, prev) {
			shapes++
			back = back || same(cur, still)
		}
		prev = cur
	}
	if shapes < 3 || !back {
		t.Errorf("held still, bat cells changed shape %d times and returned=%v; want wings that beat", shapes-1, back)
	}
}

func TestBirthday_CandlesFlicker(t *testing.T) {
	a := newArt(t, events.Birthday).(*birthday)
	a.Resize(120, 40)
	a.Step(dt)
	bd := a.Bounds()
	// The flames are the cells of the row above the candles.
	y := bd.Y0 + bd.H - cakeH
	flames := func() (string, int) {
		var s []rune
		n := 0
		for x := bd.X0; x < bd.X0+bd.W; x++ {
			r, _ := a.Cell(x, y, emptyBox)
			s = append(s, r)
			if r != 0 {
				n++
			}
		}
		return string(s), n
	}
	first, n := flames()
	if n < 3 || n > 5 || n != a.n {
		t.Fatalf("%d flames for %d candles, want 3-5", n, a.n)
	}
	for i := 0; i < 30; i++ {
		a.Step(dt)
		if s, _ := flames(); s != first {
			return
		}
	}
	t.Error("no candle flickered within 30 steps")
}

func TestKites_HaveStrings(t *testing.T) {
	a := newArt(t, events.Sankranti).(*sankranti)
	a.Resize(120, 40)
	if a.n < 2 || a.n > 4 {
		t.Fatalf("%d kites, want 2-4", a.n)
	}
	for i := 0; i < 60; i++ {
		a.Step(dt)
		str := litDots(&a.f, is(kiteString))
		fx, fy := a.flyer()
		for k := range a.n {
			if len(litDots(&a.f, is(a.cols[k]))) < 10 {
				t.Fatalf("step %d: kite %d barely drawn", i, k)
			}
			tx, ty := a.tip(k)
			// The string runs from the kite's tip down to the flyer. Its upper
			// half may pass behind other kites; the lower half is clear.
			for f := 0.5; f <= 0.95; f += 0.05 {
				x, y := tx+(fx-tx)*f, ty+(fy-ty)*f
				if !near(str, int(math.Floor(2*x)), int(math.Floor(4*y))) {
					t.Fatalf("step %d: kite %d string missing at %.2f of the way (%.2f, %.2f)", i, k, f, x, y)
				}
			}
		}
	}
}

func TestKites_Bob(t *testing.T) {
	a := newArt(t, events.Sankranti).(*sankranti)
	a.Resize(120, 40)
	a.Step(dt)
	x0, y0 := a.tip(0)
	for i := 0; i < 60; i++ {
		a.Step(dt)
		x, y := a.tip(0)
		if math.Abs(x-x0) > 0.1 || math.Abs(y-y0) > 0.1 {
			return
		}
	}
	t.Error("kite did not move within 2 s")
}

func TestIndependence_FlagRipples(t *testing.T) {
	a := newArt(t, events.IndependenceDay).(*independence)
	a.Resize(120, 40)
	a.Step(dt)
	bd := a.Bounds()
	if bd.X0 != edgeMargin || bd.Y0 != 1 {
		t.Fatalf("bounds %+v not in the top-left corner", bd)
	}
	// Each band is its own cell row, saffron over white over green, with
	// the navy wheel in the white band.
	rowOf := map[scene.RGB]int{}
	for y := 0; y < bd.H; y++ {
		for x := 0; x < bd.W; x++ {
			if g := a.f.c.Glyph(x, y); g != 0 {
				c := a.f.c.RGB(x, y)
				for _, band := range []scene.RGB{flagSaffron, flagWhite, flagGreen, chakraCol} {
					if lerpBand(c, band) {
						if r, ok := rowOf[band]; ok && r != y {
							t.Errorf("%v in rows %d and %d", band, r, y)
						}
						rowOf[band] = y
					}
				}
			}
		}
	}
	s, ok1 := rowOf[flagSaffron]
	w, ok2 := rowOf[flagWhite]
	g, ok3 := rowOf[flagGreen]
	c, ok4 := rowOf[chakraCol]
	if !ok1 || !ok2 || !ok3 || !ok4 || !(s < w && w < g) || c != w {
		t.Fatalf("band rows saffron %d white %d green %d wheel %d (%v %v %v %v)", s, w, g, c, ok1, ok2, ok3, ok4)
	}
	if len(litDots(&a.f, is(poleCol))) < 4*(bd.H-1) {
		t.Error("pole too short")
	}
	snap := func() string {
		var s []rune
		for y := 0; y < 3; y++ {
			for x := 1; x < bd.W; x++ {
				s = append(s, a.f.c.Glyph(x, y))
			}
		}
		return string(s)
	}
	first := snap()
	for i := 0; i < 30; i++ {
		a.Step(dt)
		if snap() != first {
			return
		}
	}
	t.Error("cloth did not ripple within 1 s")
}

// lerpBand reports whether c is band shaded by the ripple: the same hue,
// scaled by some brightness.
func lerpBand(c, band scene.RGB) bool {
	for k := 0.5; k <= 1.0001; k += 0.01 {
		s := scale(band, k)
		if abs(int(s.R)-int(c.R)) <= 1 && abs(int(s.G)-int(c.G)) <= 1 && abs(int(s.B)-int(c.B)) <= 1 {
			return true
		}
	}
	return false
}

func TestValentines_HeartsRise(t *testing.T) {
	a := newArt(t, events.Valentines).(*valentines)
	a.Resize(120, 40)
	a.Step(dt)
	if n := len(a.hearts); n < heartMin || n > heartMax {
		t.Fatalf("%d hearts, want %d-%d", n, heartMin, heartMax)
	}
	before := make([]float64, len(a.hearts))
	for i, p := range a.hearts {
		before[i] = p.y
	}
	for range 15 {
		a.Step(dt)
	}
	rose := 0
	for i, p := range a.hearts {
		if p.y < before[i] {
			rose++
		}
	}
	if rose < len(a.hearts)-1 {
		t.Errorf("%d of %d hearts rose in 0.5 s", rose, len(a.hearts))
	}
	// Low hearts are red, high ones pinker.
	red, pink := 0, 0
	for range 300 {
		a.Step(dt)
		w, h := a.f.c.Size()
		for y := range h {
			for x := range w {
				if a.f.c.Glyph(x, y) == 0 {
					continue
				}
				switch c := a.f.c.RGB(x, y); {
				case y >= h/2 && c == heartRed:
					red++
				case y < h/2 && c != heartRed:
					pink++
				}
			}
		}
	}
	if red == 0 || pink == 0 {
		t.Errorf("red cells low %d, pinker cells high %d", red, pink)
	}
}

func TestVishu_FlowersFallLampLit(t *testing.T) {
	a := newArt(t, events.Vishu).(*vishu)
	a.Resize(120, 40)
	a.Step(dt)
	bd := a.Bounds()
	if bd.X0+bd.W != 120-edgeMargin || bd.Y0+bd.H != 39 {
		t.Fatalf("bounds %+v not against the bottom-right corner", bd)
	}
	if len(litDots(&a.f, is(lampCol))) < 30 {
		t.Fatal("no lamp drawn")
	}
	// Three flames sit in the row above the lamp's dish.
	fy := bd.H - lampH
	flames := func() string {
		var s []rune
		for x := 0; x < bd.W; x++ {
			if c := a.f.c.RGB(x, fy); a.f.c.Glyph(x, fy) != 0 && c != konnaCol {
				s = append(s, a.f.c.Glyph(x, fy))
			}
		}
		return string(s)
	}
	first := flames()
	if n := len([]rune(first)); n < 3 {
		t.Fatalf("%d flame cells, want 3", n)
	}
	before := make([]float64, len(a.flowers))
	for i, p := range a.flowers {
		before[i] = p.y
	}
	flickered := false
	for range 30 {
		a.Step(dt)
		flickered = flickered || flames() != first
	}
	if !flickered {
		t.Error("no flame flickered within 1 s")
	}
	fell := 0
	for i, p := range a.flowers {
		if p.y > before[i] {
			fell++
		}
	}
	if fell < len(a.flowers)-1 {
		t.Errorf("%d of %d flowers fell in 1 s", fell, len(a.flowers))
	}
	if len(litDots(&a.f, is(konnaCol))) == 0 {
		t.Error("no konna flowers drawn")
	}
}

// Motion is scaled by the step: thirty 33 ms steps and fifteen 66 ms steps
// leave each glider in the same place.
func TestSetB_MotionScalesWithDt(t *testing.T) {
	type pos func(a Art) []float64
	cases := []struct {
		k   events.Kind
		pos pos
	}{
		{events.Sankranti, func(a Art) []float64 {
			s := a.(*sankranti)
			var out []float64
			for i := range s.n {
				x, y := s.tip(i)
				out = append(out, x, y)
			}
			return out
		}},
		{events.Valentines, func(a Art) []float64 {
			var out []float64
			for _, p := range a.(*valentines).hearts {
				out = append(out, p.y)
			}
			return out
		}},
		{events.Vishu, func(a Art) []float64 {
			var out []float64
			for _, p := range a.(*vishu).flowers {
				out = append(out, p.y)
			}
			return out
		}},
		{events.Birthday, func(a Art) []float64 {
			var out []float64
			for _, p := range a.(*birthday).confetti {
				out = append(out, p.y)
			}
			return out
		}},
	}
	for _, c := range cases {
		run := func(step time.Duration, n int) (before, after []float64) {
			a := newArt(t, c.k)
			a.Resize(120, 40)
			before = c.pos(a)
			for range n {
				a.Step(step)
			}
			return before, c.pos(a)
		}
		start, fine := run(dt, 30)
		_, coarse := run(2*dt, 15)
		// A piece that falls (or, for hearts, rises) off the end restarts
		// at random, so only pieces still on their first pass are compared.
		// Kites never restart.
		onward := func(i int, y float64) bool {
			switch c.k {
			case events.Sankranti:
				return true
			case events.Valentines:
				return y < start[i]
			default:
				return y > start[i]
			}
		}
		same, kept := 0, 0
		for i := range fine {
			if !onward(i, fine[i]) || !onward(i, coarse[i]) {
				continue
			}
			kept++
			if math.Abs(fine[i]-coarse[i]) < 1e-9 {
				same++
			}
		}
		if kept == 0 || same != kept {
			t.Errorf("%v: %d of %d positions agree across step sizes: %v vs %v", c.k, same, len(fine), fine, coarse)
		}
	}
}

package harbor

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/v4run/tcal/internal/scene"
)

const dt = 33 * time.Millisecond

var noBox = scene.Box{Y0: -1, Y1: -1}

func newTest(w, h int, seed uint64) *Scene {
	return New(w, h, rand.New(rand.NewPCG(seed, seed^0x9e37)))
}

// lowBox is a calendar box whose bottom sits well clear of the strip.
func lowBox(h int) scene.Box { return scene.Box{X0: 20, X1: 60, Y0: 1, Y1: h - Height - 1} }

// lit reports whether strip dot (dx, dy) is plotted.
func lit(s *Scene, dx, dy int) bool {
	g := s.canvas.Glyph(dx/2, dy/4)
	if g < 0x2800 || g > 0x28ff {
		return false
	}
	return (g-0x2800)&rune([4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}[dy&3][dx&1]) != 0
}

// leftmost returns the leftmost plotted dot column in strip dot row dy, or -1.
func leftmost(s *Scene, dy int) int {
	for dx := 0; dx < s.w*2; dx++ {
		if lit(s, dx, dy) {
			return dx
		}
	}
	return -1
}

func TestHarbor_ShipGlides(t *testing.T) {
	const v, x0 = 6.5, 30.0
	for _, dir := range []int{1, -1} {
		s := newTest(120, 30, 1)
		s.wait = math.Inf(1)
		sh := s.newShip(dir, v)
		sh.x = x0
		s.ships, s.pending = []*ship{sh}, nil
		s.Step(0)
		prev := leftmost(s, deckRow)
		for i := 1; i <= 100; i++ {
			s.Step(dt)
			tt := float64(i) * dt.Seconds()
			want := int(math.Floor(2 * (x0 + float64(dir)*v*tt)))
			got := leftmost(s, deckRow)
			if d := got - want; d < -1 || d > 1 {
				t.Fatalf("dir %d step %d: deck starts at dot %d, ideal %d", dir, i, got, want)
			}
			if (got-prev)*dir < 0 || abs(got-prev) > int(math.Ceil(v*0.033*2))+1 {
				t.Fatalf("dir %d step %d: jumped %d -> %d", dir, i, prev, got)
			}
			prev = got
		}
	}
}

// Speeds come from classic: a ship moves one column every 2-4 ticks of
// 50 ms, and a companion is never faster than its leader.
func TestHarbor_SpeedsMatchClassic(t *testing.T) {
	allowed := []float64{10, 20.0 / 3, 5}
	s := newTest(120, 30, 3)
	for range 300 {
		s.ships, s.pending = nil, nil
		s.launch()
		lead := s.ships[0].speed
		ok := false
		for _, a := range allowed {
			ok = ok || math.Abs(lead-a) < 1e-9
		}
		if !ok {
			t.Fatalf("lead speed %v, want one of %v", lead, allowed)
		}
		if s.pending != nil && s.pending.speed > lead+1e-9 {
			t.Fatalf("companion %v faster than lead %v", s.pending.speed, lead)
		}
	}
}

// The first ship sails within 2.25 s, and an empty sea then waits
// 22.5-90 s before the next.
func TestHarbor_Waits(t *testing.T) {
	for seed := uint64(0); seed < 20; seed++ {
		s := newTest(120, 30, seed)
		n := 0
		for len(s.ships) == 0 {
			s.Step(dt)
			n++
		}
		if el := time.Duration(n) * dt; el > firstWait+dt {
			t.Fatalf("seed %d: first ship after %v", seed, el)
		}
		if s.wait < minWait.Seconds() || s.wait >= (minWait+waitSpread).Seconds() {
			t.Fatalf("seed %d: next wait %v s", seed, s.wait)
		}
	}
}

func TestHarbor_ShipsNeverOverlap(t *testing.T) {
	for seed := uint64(0); seed < 20; seed++ {
		s := newTest(120, 30, seed)
		for i := 0; i < 6000; i++ {
			s.Step(dt)
			if len(s.ships) > 2 {
				t.Fatalf("seed %d: %d ships at once", seed, len(s.ships))
			}
			if len(s.ships) == 2 {
				a, b := s.ships[0], s.ships[1]
				if a.dir != b.dir {
					t.Fatalf("seed %d: ships sailing opposite ways together", seed)
				}
				if a.x < b.x+b.width() && b.x < a.x+a.width() {
					t.Fatalf("seed %d: ships overlap at step %d", seed, i)
				}
			}
		}
	}
}

func TestHarbor_HidesWhenNoRoom(t *testing.T) {
	const w, h = 100, 30
	s := newTest(w, h, 2)
	for len(s.ships) == 0 {
		s.Step(dt)
	}
	s.Step(time.Second)
	touching := scene.Box{X0: 20, X1: 60, Y0: 2, Y1: h - Height}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if r := s.Glyph(x, y, touching); r != 0 {
				t.Fatalf("drew %q at (%d,%d) with the box in the strip", r, x, y)
			}
		}
	}
	// With room, the strip shows: water across the bottom row.
	for x := 0; x < w; x++ {
		if s.Glyph(x, h-1, lowBox(h)) == 0 {
			t.Fatalf("no water at column %d", x)
		}
	}
	n := newTest(MinWidth-1, 30, 3)
	n.Step(dt)
	for x := 0; x < MinWidth-1; x++ {
		if r := n.Glyph(x, 29, noBox); r != 0 {
			t.Fatalf("drew %q on a narrow screen", r)
		}
	}
}

// Only the strip is drawn; the deck and hull are hull-coloured, the bridge
// sail-coloured and the bottom row water.
func TestHarbor_StripAndPalette(t *testing.T) {
	const w, h = 120, 30
	s := newTest(w, h, 4)
	s.wait = math.Inf(1)
	sh := s.newShip(1, 5)
	sh.x = 40
	s.ships = []*ship{sh}
	s.Step(0)
	b := lowBox(h)
	saw := map[scene.RGB]bool{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := s.Glyph(x, y, b)
			if r == 0 {
				continue
			}
			if y < h-Height {
				t.Fatalf("glyph %q above the strip at (%d,%d)", r, x, y)
			}
			if ansi.StringWidth(string(r)) != 1 {
				t.Fatalf("glyph %q is not single width", r)
			}
			saw[s.RGB(x, y, b)] = true
		}
	}
	for _, c := range []scene.RGB{hullCol, sailCol, waterCol} {
		if !saw[c] {
			t.Errorf("colour %v not drawn", c)
		}
	}
	if c := s.RGB(41, h-Height+3, b); c != hullCol {
		t.Errorf("deck cell is %v, want hull %v", c, hullCol)
	}
	if c := s.RGB(41, h-1, b); c != waterCol {
		t.Errorf("water cell is %v, want %v", c, waterCol)
	}
}

func TestHarbor_ResizeAnySize(t *testing.T) {
	s := newTest(0, 0, 5)
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {3, 2}, {200, 60}, {10, 3}, {0, 50}, {-1, -1}, {80, 24}} {
		s.Resize(sz[0], sz[1])
		for range 50 {
			s.Step(100 * time.Millisecond)
		}
		for y := -1; y <= sz[1]; y++ {
			for x := -1; x <= sz[0]; x++ {
				s.Glyph(x, y, noBox)
				s.RGB(x, y, noBox)
			}
		}
	}
}

func TestHarbor_Interface(t *testing.T) {
	s := newTest(80, 24, 6)
	if s.Interval() != dt {
		t.Fatal("interval")
	}
	s.Tick()
	s.Step(-time.Second)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

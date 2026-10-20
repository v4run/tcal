package starfield

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

const dt = 33 * time.Millisecond

var noBox = scene.Box{Y0: -1, Y1: -1}

func newScene(w, h int) *Scene { return New(w, h, rand.New(rand.NewPCG(1, 2))) }

// quiet removes all stars and postpones the shooting star.
func quiet(s *Scene) {
	s.stars = nil
	s.wait = time.Hour
	s.meteor = nil
}

// dotPos returns the (column, row) in dots of the first plotted dot.
func dotPos(s *Scene) (int, int) {
	w, h := s.canvas.Size()
	rows := [4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}
	for cy := range h {
		for cx := range w {
			g := s.canvas.Glyph(cx, cy)
			if g < 0x2800 || g > 0x28ff {
				continue
			}
			m := int(g - 0x2800)
			for r := range 4 {
				for c := range 2 {
					if m&rows[r][c] != 0 {
						return cx*2 + c, cy*4 + r
					}
				}
			}
		}
	}
	return -1, -1
}

func lum(c scene.RGB) float64 {
	return 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
}

func TestStarfield_Glides(t *testing.T) {
	for l := range layers {
		v := layers[l].speed
		s := newScene(100, 40)
		quiet(s)
		const x0 = 90.0
		s.stars = []star{{x: x0, y: 10, layer: l}}
		s.Step(0)
		prev, _ := dotPos(s)
		for i := 1; i <= 100; i++ {
			s.Step(dt)
			want := (x0 - v*float64(i)*dt.Seconds()) * 2
			got, _ := dotPos(s)
			if math.Abs(float64(got)-want) > 1.5 {
				t.Fatalf("layer %d step %d: dot %d, ideal %.2f", l, i, got, want)
			}
			if got > prev {
				t.Fatalf("layer %d step %d: moved right %d -> %d", l, i, prev, got)
			}
			prev = got
		}
	}
}

func TestStarfield_Parallax(t *testing.T) {
	s := newScene(200, 40)
	quiet(s)
	for l := range layers {
		s.stars = append(s.stars, star{x: 150, y: float64(5 + l*10), layer: l})
	}
	for range 90 {
		s.Step(dt)
	}
	var moved []float64
	for _, st := range s.stars {
		moved = append(moved, 150-st.x)
	}
	if !(moved[0] < moved[1] && moved[1] < moved[2]) {
		t.Fatalf("displacements not increasing with layer: %v", moved)
	}
	if !(lum(layerCol[0]) < lum(layerCol[1]) && lum(layerCol[1]) < lum(layerCol[2])) {
		t.Fatal("near layers not brighter")
	}
	for l, st := range s.stars {
		got := s.canvas.RGB(int(st.x), int(st.y))
		if got != layerCol[l] {
			t.Errorf("layer %d drawn %v, want %v", l, got, layerCol[l])
		}
	}
}

func TestStarfield_Wraps(t *testing.T) {
	s := newScene(40, 20)
	quiet(s)
	s.stars = []star{{x: 0.1, y: 3, layer: 2}}
	s.Step(time.Second)
	st := s.stars[0]
	if st.x < float64(s.w)-1.5 || st.x >= float64(s.w) {
		t.Fatalf("x = %v, want near right edge", st.x)
	}
	if st.y < 0 || st.y >= float64(s.h) {
		t.Fatalf("y = %v off screen", st.y)
	}
	if x, _ := dotPos(s); x < (s.w-2)*2 {
		t.Fatalf("not drawn at right edge: dot %d", x)
	}
}

func TestStarfield_DensityAndResize(t *testing.T) {
	s := newScene(80, 24)
	for _, sz := range [][2]int{{200, 60}, {30, 10}, {5, 3}, {0, 0}, {1, 1}, {80, 24}} {
		s.Resize(sz[0], sz[1])
		s.Step(dt)
		want := 0
		if sz[0] > 0 && sz[1] > 0 {
			want = max(1, sz[0]*sz[1]/cellsPerStar)
		}
		if len(s.stars) != want {
			t.Errorf("%v: %d stars, want %d", sz, len(s.stars), want)
		}
		for _, st := range s.stars {
			if st.x < 0 || st.x >= float64(sz[0]) || st.y < 0 || st.y >= float64(sz[1]) {
				t.Fatalf("%v: star off screen %+v", sz, st)
			}
		}
	}
}

func TestStarfield_LayerShares(t *testing.T) {
	s := newScene(400, 200)
	var n [3]float64
	for _, st := range s.stars {
		n[st.layer]++
	}
	for l := range layers {
		if f := n[l] / float64(len(s.stars)); math.Abs(f-layers[l].share) > 0.03 {
			t.Errorf("layer %d share %.2f, want %.2f", l, f, layers[l].share)
		}
	}
}

func TestStarfield_Interface(t *testing.T) {
	s := newScene(20, 10)
	if s.Interval() != dt {
		t.Fatal("interval")
	}
	s.Tick()
	_ = s.RGB(0, 0, noBox)
	_ = s.Glyph(0, 0, noBox)
}

func TestStarfield_ShootingStar(t *testing.T) {
	s := newScene(120, 40)
	s.stars = nil
	var (
		starts          []time.Duration
		ends            []time.Duration
		now             time.Duration
		prevX, prevY    float64
		wasOn           bool
		dir             int
		burnedAfterLife bool
	)
	for now < 60*time.Second {
		s.Step(dt)
		now += dt
		m := s.meteor
		if m == nil {
			if wasOn {
				ends = append(ends, now)
				if n := lit(s); n != 0 {
					t.Fatalf("%d cells still lit after burn-out", n)
				}
			}
			wasOn = false
			continue
		}
		if !wasOn {
			starts = append(starts, now)
			dir = m.dir
		} else {
			if float64(dir)*(m.x-prevX) <= 0 {
				t.Fatalf("head not moving in direction %d: %v -> %v", dir, prevX, m.x)
			}
			if m.y <= prevY {
				t.Fatalf("head not moving down: %v -> %v", prevY, m.y)
			}
		}
		prevX, prevY, wasOn = m.x, m.y, true
		checkTrail(t, s)
		if m.age > lifeSpan {
			burnedAfterLife = true
		}
	}
	if len(starts) == 0 {
		t.Fatal("no shooting star in 60 s")
	}
	if !burnedAfterLife {
		t.Fatal("never saw burn-out phase")
	}
	if starts[0] < 10*time.Second-dt || starts[0] > 30*time.Second+2*dt {
		t.Fatalf("first launch at %v, want 10-30 s", starts[0])
	}
	for i := 1; i < len(starts); i++ {
		if gap := starts[i] - ends[i-1]; gap < 10*time.Second-dt {
			t.Fatalf("gap %v < 10 s", gap)
		}
	}
}

func lit(s *Scene) int {
	w, h := s.canvas.Size()
	n := 0
	for y := range h {
		for x := range w {
			if s.canvas.Glyph(x, y) != 0 {
				n++
			}
		}
	}
	return n
}

// checkTrail checks that, while the streak is still full strength, every lit
// cell other than the head's is dimmer than the head.
func checkTrail(t *testing.T, s *Scene) {
	t.Helper()
	m := s.meteor
	if m.age > lifeSpan {
		return
	}
	hx, hy := int(m.x), int(m.y)
	if hx < 0 || hy < 0 || hx >= s.w || hy >= s.h {
		return
	}
	head := lum(s.canvas.RGB(hx, hy))
	if s.canvas.RGB(hx, hy) != headCol {
		t.Fatalf("head cell colour %v, want %v", s.canvas.RGB(hx, hy), headCol)
	}
	for y := range s.h {
		for x := range s.w {
			if s.canvas.Glyph(x, y) == 0 || (x == hx && y == hy) {
				continue
			}
			if l := lum(s.canvas.RGB(x, y)); l >= head {
				t.Fatalf("trail cell (%d,%d) lum %v not below head %v", x, y, l, head)
			}
		}
	}
}

func TestStarfield_ShootingStarTrailLength(t *testing.T) {
	s := newScene(120, 40)
	quiet(s)
	s.launch()
	s.meteor.x, s.meteor.y = 60, 5
	s.Step(0)
	if n := lit(s); n < 5 || n > trailDots+1 {
		t.Fatalf("%d cells lit, want about %d", n, trailDots+1)
	}
}

func TestStarfield_StepSizeIndependent(t *testing.T) {
	a, b := newScene(100, 40), newScene(100, 40)
	for range 50 {
		a.Step(2 * dt)
		b.Step(dt)
		b.Step(dt)
	}
	for i := range a.stars {
		if math.Abs(a.stars[i].x-b.stars[i].x) > 1e-6 || a.stars[i].y != b.stars[i].y {
			t.Fatalf("star %d differs: %+v vs %+v", i, a.stars[i], b.stars[i])
		}
	}
	if (a.meteor == nil) != (b.meteor == nil) {
		t.Fatal("meteor state differs")
	}
	if a.meteor != nil && math.Abs(a.meteor.x-b.meteor.x) > 1e-6 {
		t.Fatal("meteor position differs")
	}
}

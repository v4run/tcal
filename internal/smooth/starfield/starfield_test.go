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
	s.comet = nil
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

func hasDot(s *Scene, dx, dy int) bool {
	rows := [4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}
	g := s.canvas.Glyph(dx>>1, dy>>2)
	if dx < 0 || dy < 0 || g < 0x2800 || g > 0x28ff {
		return false
	}
	return int(g-0x2800)&rows[dy&3][dx&1] != 0
}

func TestStarfield_CometGlides(t *testing.T) {
	s := newScene(120, 40)
	quiet(s)
	const x0, y0, vx, vy = 5.0, 6.0, 15.0, 1.8
	s.comet = &comet{x: x0, y: y0, vx: vx, vy: vy, seed: 7}
	s.Step(0)
	for i := 1; i <= 100; i++ {
		s.Step(dt)
		ts := float64(i) * dt.Seconds()
		ix, iy := int(math.Floor((x0+vx*ts)*2)), int(math.Floor((y0+vy*ts)*4))
		if s.canvas.RGB(ix>>1, iy>>2) != coreCol {
			t.Fatalf("step %d: head cell not core coloured", i)
		}
		ok := false
		for d := -1; d <= 1; d++ {
			ok = ok || hasDot(s, ix+d, iy)
		}
		if !ok {
			t.Fatalf("step %d: no head dot within 1 of (%d,%d)", i, ix, iy)
		}
	}
}

func TestStarfield_CometTailWidensAndFades(t *testing.T) {
	s := newScene(200, 60)
	quiet(s)
	const hx, hy, vx, vy = 100.0, 20.0, 15.0, 1.8
	s.comet = &comet{x: hx, y: hy, vx: vx, vy: vy, seed: 3}
	s.Step(0)
	hx2, hy4 := int(hx*2), int(hy*4)
	slope := vy / vx
	type bin struct{ lo, hi int }
	spread := func(b bin) int {
		lo, hi := math.MaxInt, math.MinInt
		for dd := b.lo; dd <= b.hi; dd++ {
			for dy := range s.h * 4 {
				if hasDot(s, hx2-dd, dy) {
					r := dy - (hy4 - int(2*slope*float64(dd)))
					lo, hi = min(lo, r), max(hi, r)
				}
			}
		}
		if hi < lo {
			t.Fatalf("no tail dots in %v", b)
		}
		return hi - lo + 1
	}
	meanLum := func(b bin) float64 {
		sum, n := 0.0, 0
		for dd := b.lo; dd <= b.hi; dd++ {
			for dy := range s.h * 4 {
				if hasDot(s, hx2-dd, dy) {
					sum += lum(s.canvas.RGB((hx2-dd)>>1, dy>>2))
					n++
				}
			}
		}
		if n == 0 {
			t.Fatalf("no tail dots in %v", b)
		}
		return sum / float64(n)
	}
	near, mid, far := bin{6, 10}, bin{18, 26}, bin{32, 38}
	if sn, sf := spread(near), spread(far); sf < sn+1 {
		t.Fatalf("tail spread near %d, far %d: not widening", sn, sf)
	}
	ln, lm, lf := meanLum(near), meanLum(mid), meanLum(far)
	if !(ln > lm && lm > lf) {
		t.Fatalf("tail luminance not falling: %v %v %v", ln, lm, lf)
	}
	if ln >= lum(coreCol) {
		t.Fatal("tail not dimmer than head")
	}
}

func TestStarfield_CometTiming(t *testing.T) {
	const w = 120
	s := newScene(w, 40)
	s.stars = nil
	var (
		now, lastEnd, crossIn, crossOut time.Duration
		starts                          int
		wasOn, entered                  bool
	)
	for now < 400*time.Second {
		s.Step(dt)
		now += dt
		c := s.comet
		if c == nil {
			if wasOn {
				lastEnd = now
			}
			wasOn = false
			continue
		}
		if !wasOn {
			starts++
			entered = false
			if starts == 1 {
				if now < 30*time.Second-dt || now > 50*time.Second+dt {
					t.Fatalf("first comet at %v, want 30-50 s", now)
				}
			} else if gap := now - lastEnd; gap < 30*time.Second-dt {
				t.Fatalf("gap %v < 30 s", gap)
			}
		}
		wasOn = true
		if c.x >= 0 && c.x <= w {
			if !entered {
				crossIn, entered = now, true
			}
			crossOut = now
			if d := (crossOut - crossIn).Seconds(); d > 10.1 {
				t.Fatalf("crossing %v s > 10", d)
			}
		} else if entered && (c.x < 0) == (c.vx < 0) {
			if d := (crossOut - crossIn).Seconds(); d < 5.9 {
				t.Fatalf("crossing %v s < 6", d)
			}
			entered = false
		}
	}
	if starts < 5 {
		t.Fatalf("only %d comets in 400 s", starts)
	}
}

func TestStarfield_StepSizeIndependent(t *testing.T) {
	a, b := newScene(100, 40), newScene(100, 40)
	sawComet := false
	for range 3000 {
		a.Step(2 * dt)
		b.Step(dt)
		b.Step(dt)
		if (a.comet == nil) != (b.comet == nil) {
			t.Fatal("comet state differs")
		}
		if a.comet != nil {
			sawComet = true
			if math.Abs(a.comet.x-b.comet.x) > 1e-6 || math.Abs(a.comet.y-b.comet.y) > 1e-6 {
				t.Fatalf("comet position differs %+v %+v", *a.comet, *b.comet)
			}
		}
	}
	if !sawComet {
		t.Fatal("no comet seen")
	}
	for i := range a.stars {
		if math.Abs(a.stars[i].x-b.stars[i].x) > 1e-6 || a.stars[i].y != b.stars[i].y {
			t.Fatalf("star %d differs: %+v vs %+v", i, a.stars[i], b.stars[i])
		}
	}
}

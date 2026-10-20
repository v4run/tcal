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

// dotPos returns the (column, row) of the only plotted dot, or -1, -1.
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
	const v, r0 = 9.0, 3.0
	s := New(100, 40, rand.New(rand.NewPCG(1, 2)))
	s.stars = []star{{angle: 0, r: r0, speed: v}}
	s.Step(0)
	cx := float64(s.w) / 2
	prev, _ := dotPos(s)
	for i := 1; i <= 100; i++ {
		s.Step(dt)
		r := r0 + v*float64(i)*0.033
		if cx+r >= float64(s.w) {
			break
		}
		want := int(math.Floor((cx + r) * 2))
		got, _ := dotPos(s)
		if d := got - want; d < -1 || d > 1 {
			t.Fatalf("step %d: dot col %d, ideal %d", i, got, want)
		}
		if got < prev {
			t.Fatalf("step %d: went back %d -> %d", i, prev, got)
		}
		if got-prev > int(math.Ceil(v*0.033*2))+1 {
			t.Fatalf("step %d: jump %d", i, got-prev)
		}
		prev = got
	}
}

func TestStarfield_DepthBrightens(t *testing.T) {
	s := New(100, 40, rand.New(rand.NewPCG(1, 2)))
	s.stars = []star{{angle: 0, r: 1, speed: 8}}
	last := -1.0
	for i := range 100 {
		s.Step(dt)
		x, y := dotPos(s)
		if x < 0 {
			break
		}
		l := lum(s.canvas.RGB(x/2, y/4))
		if l < last {
			t.Fatalf("step %d: luminance fell %v -> %v", i, last, l)
		}
		last = l
	}
	if far, near := lum(farCol), lum(nearCol); !(far < near) {
		t.Fatal("palette not far < near")
	}
	if last <= lum(farCol)+1 {
		t.Fatalf("star never brightened: %v", last)
	}
}

func TestStarfield_RespawnsAndDensity(t *testing.T) {
	s := New(80, 24, rand.New(rand.NewPCG(1, 2)))
	for range 600 {
		s.Step(dt)
	}
	for _, st := range s.stars {
		if st.r < 0 || st.r > s.maxR() {
			t.Fatalf("star out of range: %+v", st)
		}
	}
	for _, sz := range [][2]int{{200, 60}, {30, 10}, {5, 3}, {0, 0}, {1, 1}} {
		s.Resize(sz[0], sz[1])
		s.Step(dt)
		want := 0
		if sz[0] > 0 && sz[1] > 0 {
			want = max(1, sz[0]*sz[1]/cellsPerStar)
		}
		if len(s.stars) != want {
			t.Errorf("%v: %d stars, want %d", sz, len(s.stars), want)
		}
	}
}

func TestStarfield_Interface(t *testing.T) {
	s := New(20, 10, rand.New(rand.NewPCG(1, 2)))
	if s.Interval() != dt {
		t.Fatal("interval")
	}
	s.Tick()
	_ = s.RGB(0, 0, noBox)
	_ = s.Glyph(0, 0, noBox)
}

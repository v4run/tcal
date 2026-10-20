package flow

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

const dt = 33 * time.Millisecond

var noBox = scene.Box{Y0: -1, Y1: -1}

// straight makes every particle travel right.
func (s *Scene) straight() {
	s.field = func(_, _ float64) float64 { return 0 }
}

// dotsOnRow returns the dot columns of every set dot on dot row r.
func dotsOnRow(s *Scene, r int) []int {
	rows := [4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}
	var out []int
	for cx := 0; cx < s.w; cx++ {
		g := s.canvas.Glyph(cx, r/4)
		if g < 0x2800 || g > 0x28ff {
			continue
		}
		for c := range 2 {
			if int(g-0x2800)&rows[r%4][c] != 0 {
				out = append(out, cx*2+c)
			}
		}
	}
	return out
}

func lum(c scene.RGB) float64 {
	return 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
}

func TestFlow_Glides(t *testing.T) {
	const x0, y0 = 5.0, 20.0
	s := New(200, 40, rand.New(rand.NewPCG(1, 2)))
	s.straight()
	s.ps = []particle{{x: x0, y: y0, life: 1000}}
	s.Step(0)
	prev := dotsOnRow(s, int(y0*4))
	if len(prev) == 0 {
		t.Fatal("no dot")
	}
	prevHead := prev[len(prev)-1]
	for i := 1; i <= 100; i++ {
		s.Step(dt)
		want := int(math.Floor((x0 + speed*float64(i)*0.033) * 2))
		d := dotsOnRow(s, int(y0*4))
		got := d[len(d)-1] // rightmost dot is the head
		if e := got - want; e < -1 || e > 1 {
			t.Fatalf("step %d: dot %d, ideal %d", i, got, want)
		}
		if got < prevHead {
			t.Fatalf("step %d: went back", i)
		}
		if got-prevHead > int(math.Ceil(speed*0.033*2))+1 {
			t.Fatalf("step %d: jump %d", i, got-prevHead)
		}
		prevHead = got
	}
}

func TestFlow_TrailFades(t *testing.T) {
	s := New(200, 40, rand.New(rand.NewPCG(1, 2)))
	s.straight()
	s.gap = 300 * time.Millisecond
	s.ps = []particle{{x: 5, y: 20, life: 1000}}
	for range 60 { // let the history fill
		s.Step(dt)
	}
	d := dotsOnRow(s, 80)
	if len(d) != trailLen+1 {
		t.Fatalf("%d dots, want head + %d trail: %v", len(d), trailLen, d)
	}
	prev := math.Inf(1)
	for i := len(d) - 1; i >= 0; i-- { // head first, then back along the trail
		l := lum(s.canvas.RGB(d[i]/2, 20))
		if l >= prev {
			t.Fatalf("dot %d (col %d) luminance %v not below %v", i, d[i], l, prev)
		}
		prev = l
	}
}

func TestFlow_DensityRespawnAndResize(t *testing.T) {
	s := New(80, 24, rand.New(rand.NewPCG(1, 2)))
	for range 1000 {
		s.Step(dt)
	}
	for _, p := range s.ps {
		if p.x < 0 || p.y < 0 || p.x >= 80 || p.y >= 24 {
			t.Fatalf("particle out of bounds: %+v", p)
		}
	}
	for _, sz := range [][2]int{{200, 60}, {30, 10}, {5, 3}, {0, 0}, {1, 1}} {
		s.Resize(sz[0], sz[1])
		s.Step(dt)
		want := 0
		if sz[0] > 0 && sz[1] > 0 {
			want = min(max(sz[0]*sz[1]/cellsPerParticle, minParticles), maxParticles)
		}
		if len(s.ps) != want {
			t.Errorf("%v: %d particles, want %d", sz, len(s.ps), want)
		}
	}
}

func TestFlow_Interface(t *testing.T) {
	s := New(20, 10, rand.New(rand.NewPCG(1, 2)))
	if s.Interval() != dt {
		t.Fatal("interval")
	}
	s.Tick()
	_ = s.RGB(0, 0, noBox)
	_ = s.Glyph(0, 0, noBox)
}

package lava

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

const dt = 33 * time.Millisecond

var noBox = scene.Box{Y0: -1, Y1: -1}

var dotBits = [4][2]int{{0x01, 0x08}, {0x02, 0x10}, {0x04, 0x20}, {0x40, 0x80}}

// centroid returns the centre of mass of every lit dot, in cell units.
func centroid(s *Scene) (x, y float64, n int) {
	cw, ch := s.canvas.Size()
	for cy := range ch {
		for cx := range cw {
			g := s.canvas.Glyph(cx, cy)
			if g < 0x2800 || g > 0x28ff {
				continue
			}
			for r := range 4 {
				for c := range 2 {
					if int(g-0x2800)&dotBits[r][c] != 0 {
						x += float64(cx*2+c) + 0.5
						y += float64(cy*4+r) + 0.5
						n++
					}
				}
			}
		}
	}
	if n == 0 {
		return 0, 0, 0
	}
	return x / float64(n) / 2, y / float64(n) / 4, n
}

func lone(vx, vy float64) *Scene {
	s := New(200, 60, rand.New(rand.NewPCG(1, 2)))
	s.blobs = []blob{{x: 80, y: 20, vx: vx, vy: vy, r: 5}}
	s.Step(0)
	return s
}

func TestLava_Glides(t *testing.T) {
	const x0, y0, vx, vy = 80.0, 20.0, 1.6, 0.8
	s := lone(vx, vy)
	px, py, n := centroid(s)
	if n == 0 {
		t.Fatal("blob not drawn")
	}
	prevX, prevY := px, py
	for i := 1; i <= 100; i++ {
		s.Step(dt)
		tt := float64(i) * 0.033
		gx, gy, _ := centroid(s)
		// Within one dot of the ideal centre.
		if math.Abs(gx-(x0+vx*tt)) > 0.5 || math.Abs(gy-(y0+vy*tt)) > 0.25 {
			t.Fatalf("step %d: centre (%.2f, %.2f), ideal (%.2f, %.2f)", i, gx, gy, x0+vx*tt, y0+vy*tt)
		}
		if gx < prevX-0.01 || gy < prevY-0.01 {
			t.Fatalf("step %d: centre moved backwards", i)
		}
		prevX, prevY = gx, gy
	}
}

func lum(c scene.RGB) int { return int(c.R) + int(c.G) + int(c.B) }

func TestLava_ColourByHeat(t *testing.T) {
	s := lone(0, 0)
	cx, cy := 80, 20 // the blob's centre cell
	core := s.RGB(cx, cy, noBox)
	// Find the outermost lit cell on the same row: the rim.
	rx := cx
	for x := cx; x < 200 && s.Glyph(x, cy, noBox) != 0; x++ {
		rx = x
	}
	rim := s.RGB(rx, cy, noBox)
	if rx == cx {
		t.Fatal("blob has no extent")
	}
	if lum(core) <= lum(rim) {
		t.Errorf("core %v not hotter than rim %v", core, rim)
	}
	if core.R <= rim.R || core.G <= rim.G {
		t.Errorf("core %v should be brighter in red and green than rim %v", core, rim)
	}
	for _, c := range []scene.RGB{core, rim} {
		if c.R < 150 || c.R > 230 || c.G < 40 || c.G > 140 || c.B < 20 || c.B > 40 {
			t.Errorf("colour %v outside the lava gradient", c)
		}
	}
}

func TestLava_BlobsStayOnScreen(t *testing.T) {
	s := New(80, 24, rand.New(rand.NewPCG(3, 4)))
	for range 3000 {
		s.Step(dt)
		for _, b := range s.blobs {
			if b.x < 0 || b.x > 80 || b.y < 0 || b.y > 24 {
				t.Fatalf("blob escaped: %+v", b)
			}
		}
	}
}

func TestLava_ResizeAndTiny(t *testing.T) {
	s := New(80, 24, rand.New(rand.NewPCG(1, 2)))
	for _, sz := range [][2]int{{200, 60}, {1, 1}, {0, 0}, {0, 10}, {10, 0}, {-1, -1}, {40, 12}} {
		s.Resize(sz[0], sz[1])
		s.Step(dt)
		s.Tick()
		for y := -1; y <= sz[1]; y++ {
			for x := -1; x <= sz[0]; x++ {
				s.Glyph(x, y, noBox)
				s.RGB(x, y, noBox)
			}
		}
	}
	if s.Interval() != dt {
		t.Fatal("interval")
	}
}

func TestLava_BlobCount(t *testing.T) {
	for seed := uint64(0); seed < 20; seed++ {
		s := New(100, 30, rand.New(rand.NewPCG(seed, seed)))
		if n := len(s.blobs); n < minBlobs || n > maxBlobs {
			t.Fatalf("%d blobs", n)
		}
	}
}

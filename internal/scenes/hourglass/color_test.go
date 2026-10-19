package hourglass

import (
	"testing"
	"time"

	"github.com/v4run/tcal/internal/scene"
)

func TestSandIsYellow(t *testing.T) {
	const w, h = 90, 30
	s := newTest(w, h)
	pile, grains := 0, 0
	for m := 0; m < 60; m += 7 {
		s.SetTime(at(m, 30), time.Time{})
		for i := 0; i < 60; i++ {
			s.Tick()
			for y := range h {
				for x := range w {
					g := s.Glyph(x, y, scene.Box{})
					if g == 0 {
						continue
					}
					if g == '·' {
						grains++
					} else {
						pile++
					}
					if c := s.Color(x, y, scene.Box{}); c != scene.Yellow {
						t.Fatalf("%02d min tick %d: %q color %d, want Yellow", m, i, g, c)
					}
				}
			}
		}
	}
	if pile == 0 || grains == 0 {
		t.Fatalf("pile=%d grains=%d; want both > 0", pile, grains)
	}
}

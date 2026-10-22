package coast

import (
	"math/rand/v2"
	"testing"
	"time"
)

// lit counts sky cells (the top skyRows of the strip) drawn in a colour
// other than spray, rock or sea, over a few seconds at now.
func lit(s *Scene, now time.Time) int {
	s.SetTime(now, now)
	n := 0
	for range 150 {
		s.Step(33 * time.Millisecond)
		for y := range skyRows {
			for x := range s.w {
				if s.canvas.Glyph(x, y) == 0 {
					continue
				}
				switch c := s.canvas.RGB(x, y); {
				case c.R > 200 && c.G < 150, c.G > 180 && c.R < 140, c.B > 180 && c.R < 120:
					n++ // red, green or cyan bursts
				}
			}
		}
	}
	return n
}

func TestEvents_NewYearFireworksOverTheSea(t *testing.T) {
	midnight := time.Date(2026, 12, 31, 23, 55, 0, 0, time.Local)
	s := New(120, 36, rand.New(rand.NewPCG(1, 2)))
	if lit(s, midnight) != 0 {
		t.Error("fireworks with events off")
	}
	s = New(120, 36, rand.New(rand.NewPCG(1, 2)))
	s.EnableEvents(nil)
	if lit(s, midnight) == 0 {
		t.Error("no fireworks at New Year with events on")
	}
	s.SetTime(time.Date(2027, 2, 10, 23, 55, 0, 0, time.Local), time.Time{})
	if s.fw != nil {
		t.Error("fireworks still on after New Year")
	}
}

func TestEvents_Celebrates(t *testing.T) {
	s := New(120, 36, rand.New(rand.NewPCG(1, 2)))
	s.EnableEvents(nil)
	s.SetTime(time.Date(2026, 12, 25, 12, 0, 0, 0, time.Local), time.Time{})
	if s.fw != nil {
		t.Error("coast has nothing for Christmas")
	}
}

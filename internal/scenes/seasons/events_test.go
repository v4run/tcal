package seasons

import (
	"testing"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

func at(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, time.Local)
}

func TestSeasons_EventFollowsNowNotShown(t *testing.T) {
	s := newTest(100, 40)
	s.EnableEvents(nil)
	s.SetTime(at(2026, time.December, 25, 12, 0), at(2026, time.July, 1, 0, 0))
	if k, ok := s.Event(); !ok || k != events.Christmas {
		t.Errorf("Event() = %v, %v; want christmas, true", k, ok)
	}
	if got := s.Season(); got != Summer {
		t.Errorf("Season() = %v, want Summer", got)
	}
}

func TestSeasons_EventStartsAtMidnight(t *testing.T) {
	s := newTest(100, 40)
	s.EnableEvents(nil)
	shown := at(2026, time.November, 1, 0, 0)
	s.SetTime(at(2026, time.November, 30, 23, 59), shown)
	if k, ok := s.Event(); ok {
		t.Fatalf("Nov 30 23:59: event %v active, want none", k)
	}
	s.SetTime(at(2026, time.December, 1, 0, 0), shown)
	if k, ok := s.Event(); !ok || k != events.Christmas {
		t.Errorf("Dec 1 00:00: Event() = %v, %v; want christmas, true", k, ok)
	}
}

func TestSeasons_EventsDisabledByDefault(t *testing.T) {
	s := newTest(100, 40)
	s.SetTime(at(2026, time.December, 25, 12, 0), at(2026, time.December, 1, 0, 0))
	if k, ok := s.Event(); ok {
		t.Errorf("event %v active without EnableEvents", k)
	}
}

func TestSeasons_ArtDrawsOverParticles(t *testing.T) {
	const w, h = 100, 40
	s := newTest(w, h)
	s.EnableEvents(nil)
	s.SetTime(at(2026, time.December, 25, 12, 0), at(2026, time.December, 1, 0, 0))
	if s.art == nil {
		t.Fatal("no art for christmas")
	}
	empty := scene.Box{Y0: -1, Y1: -1}
	bd := s.art.Bounds()
	found := false
	for y := bd.Y0; y < bd.Y0+bd.H; y++ {
		for x := bd.X0; x < bd.X0+bd.W; x++ {
			r, c := s.art.Cell(x, y, empty)
			if r == 0 {
				continue
			}
			found = true
			if got := s.Glyph(x, y, empty); got != r {
				t.Fatalf("Glyph(%d,%d) = %q, want art rune %q", x, y, got, r)
			}
			if got := s.Color(x, y, empty); got != c {
				t.Fatalf("Color(%d,%d) = %v, want art color %v", x, y, got, c)
			}
		}
	}
	if !found {
		t.Fatal("art drew nothing inside its bounds")
	}
}

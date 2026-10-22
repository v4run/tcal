package trees

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/v4run/tcal/internal/events"
	"github.com/v4run/tcal/internal/scene"
)

// during returns a time inside kind's 2026 occurrence.
func during(t *testing.T, k events.Kind, bday *events.MonthDay) time.Time {
	t.Helper()
	for _, o := range events.Occurrences(2026, bday) {
		if o.Kind == k {
			return o.Start.Add(time.Hour)
		}
	}
	t.Fatalf("no %v in 2026", k)
	return time.Time{}
}

// shows reports whether any cell of the scene is in one of cols, after a
// few seconds at now with the calendar box b.
func shows(s *Scene, b scene.Box, now time.Time, cols ...scene.RGB) bool {
	s.SetTime(now, now)
	s.Glyph(0, 0, b)
	for range 60 {
		s.Step(33 * time.Millisecond)
	}
	w, h := s.canvas.Size()
	for y := range h {
		for x := range w {
			if s.canvas.Glyph(x, y) == 0 {
				continue
			}
			for _, c := range cols {
				if s.canvas.RGB(x, y) == c {
					return true
				}
			}
		}
	}
	return false
}

func TestEvents_EachCelebratedEventShows(t *testing.T) {
	bday := &events.MonthDay{Month: time.May, Day: 17}
	b := scene.Box{X0: 34, X1: 86, Y0: 4, Y1: 22}
	for _, tc := range []struct {
		k    events.Kind
		cols []scene.RGB
	}{
		{events.Diwali, []scene.RGB{clayCol, flameCol}},
		{events.Onam, pookalam},
		{events.Christmas, append([]scene.RGB{starCol}, fairyCols...)},
		{events.Birthday, balloonCols},
	} {
		now := during(t, tc.k, bday)
		s := New(120, 36, rand.New(rand.NewPCG(1, 2)))
		if shows(s, b, now, tc.cols...) {
			t.Errorf("%v: shown with events off", tc.k)
		}
		s = New(120, 36, rand.New(rand.NewPCG(1, 2)))
		s.EnableEvents(bday)
		if !shows(s, b, now, tc.cols...) {
			t.Errorf("%v: not shown with events on", tc.k)
		}
		if k, ok := s.Event(); !ok || k != tc.k {
			t.Errorf("%v: Event() = %v, %v", tc.k, k, ok)
		}
	}
}

func TestEvents_OthersAreIgnored(t *testing.T) {
	s := New(120, 36, rand.New(rand.NewPCG(1, 2)))
	s.EnableEvents(nil)
	now := during(t, events.Halloween, nil)
	s.SetTime(now, now)
	if _, ok := s.Event(); ok {
		t.Error("trees has nothing for Halloween; it should not report it")
	}
}

func TestEvents_OnamPetalsFall(t *testing.T) {
	s := New(120, 36, rand.New(rand.NewPCG(1, 2)))
	s.EnableEvents(nil)
	now := during(t, events.Onam, nil)
	s.SetTime(now, now)
	for _, c := range s.look().fall {
		if c != petals[0] && c != petals[1] && c != petals[2] {
			t.Fatalf("falling %v at Onam, want petals", c)
		}
	}
}

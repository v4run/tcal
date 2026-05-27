package calendar

import (
	"testing"
	"time"
)

func TestBuildMonths_AprilCount3_SundayStart(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	got := BuildMonths(start, 3, time.Sunday)

	if len(got) != 3 {
		t.Fatalf("got %d months, want 3", len(got))
	}

	apr := got[0]
	if apr.Year != 2026 || apr.Month != time.April {
		t.Errorf("first month: got %d-%s, want 2026-April", apr.Year, apr.Month)
	}
	if len(apr.Weeks) != 6 {
		t.Errorf("got %d weeks, want 6", len(apr.Weeks))
	}
	if len(apr.Weeks[0]) != 7 {
		t.Errorf("got %d days in first week, want 7", len(apr.Weeks[0]))
	}

	// April 1, 2026 is a Wednesday — with Sunday start, that's column 3.
	// Columns 0..2 should be March 29, 30, 31 (InMonth=false).
	w0 := apr.Weeks[0]
	if w0[3].Date.Day() != 1 || !w0[3].InMonth {
		t.Errorf("week 0 col 3: got day=%d inMonth=%v, want day=1 inMonth=true",
			w0[3].Date.Day(), w0[3].InMonth)
	}
	if w0[2].Date.Day() != 31 || w0[2].InMonth {
		t.Errorf("week 0 col 2 (leading padding): got day=%d inMonth=%v, want day=31 inMonth=false",
			w0[2].Date.Day(), w0[2].InMonth)
	}

	// Verify month sequencing: got[1] = May 2026, got[2] = June 2026.
	if got[1].Month != time.May || got[1].Year != 2026 {
		t.Errorf("second month: got %d-%s, want 2026-May", got[1].Year, got[1].Month)
	}
	if got[2].Month != time.June || got[2].Year != 2026 {
		t.Errorf("third month: got %d-%s, want 2026-June", got[2].Year, got[2].Month)
	}
}

func TestBuildMonths_LeapYearFebruary(t *testing.T) {
	start := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	got := BuildMonths(start, 1, time.Monday)

	feb := got[0]
	// Feb 2024 has 29 days. Find day 29 in the grid.
	var found bool
	for _, w := range feb.Weeks {
		for _, d := range w {
			if d.InMonth && d.Date.Day() == 29 && d.Date.Month() == time.February {
				found = true
			}
		}
	}
	if !found {
		t.Error("Feb 29 2024 not found in grid")
	}

	// Grid must always have exactly 6 weeks.
	if len(feb.Weeks) != 6 {
		t.Errorf("got %d weeks, want 6", len(feb.Weeks))
	}

	// Feb 28 2024 must appear in the grid with InMonth=true.
	var foundFeb28 bool
	for _, w := range feb.Weeks {
		for _, d := range w {
			if d.InMonth && d.Date.Day() == 28 && d.Date.Month() == time.February {
				foundFeb28 = true
			}
		}
	}
	if !foundFeb28 {
		t.Error("Feb 28 2024 not found in grid with InMonth=true")
	}
}

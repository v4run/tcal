package events

import (
	"testing"
	"time"
)

func d(m time.Month, dd int) span { return span{day{m, dd}, day{m, dd}} }

// useFixture swaps the moving-date table for the duration of the test.
func useFixture(t *testing.T, f map[Kind]map[int]span) {
	t.Helper()
	old := moving
	moving = f
	t.Cleanup(func() { moving = old })
}

func fixture() map[Kind]map[int]span {
	return map[Kind]map[int]span{
		LunarNewYear: {2026: d(time.February, 17)},
		Holi:         {2026: d(time.March, 4)},
		Vishu:        {2026: d(time.April, 14)},
		Eid:          {2026: d(time.March, 20)},
		Onam:         {2026: {day{time.August, 17}, day{time.August, 26}}},
		Diwali:       {2026: d(time.November, 8)},
	}
}

func TestActive_Boundaries(t *testing.T) {
	useFixture(t, fixture())
	loc := time.UTC
	bd := &MonthDay{time.June, 10}
	at := func(m time.Month, dd, h int) time.Time { return time.Date(2026, m, dd, h, 0, 0, 0, loc) }
	tests := []struct {
		kind       Kind
		start, end time.Time
	}{
		{NewYear, time.Date(2025, 12, 31, 18, 0, 0, 0, loc), at(time.January, 2, 0)},
		{Sankranti, at(time.January, 14, 0), at(time.January, 16, 0)},
		{LunarNewYear, at(time.February, 17, 0), at(time.February, 20, 0)},
		{Valentines, at(time.February, 14, 0), at(time.February, 15, 0)},
		{Holi, at(time.March, 3, 18), at(time.March, 5, 0)},
		{Vishu, at(time.April, 14, 0), at(time.April, 15, 0)},
		{Eid, at(time.March, 19, 0), at(time.March, 22, 0)},
		{IndependenceDay, at(time.August, 15, 0), at(time.August, 16, 0)},
		{Onam, at(time.August, 17, 0), at(time.August, 27, 0)},
		{Halloween, at(time.October, 31, 0), at(time.November, 1, 0)},
		{Diwali, at(time.November, 6, 0), at(time.November, 11, 0)},
		{Christmas, at(time.December, 1, 0), at(time.December, 27, 0)},
		{Birthday, at(time.June, 10, 0), at(time.June, 11, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.kind.String(), func(t *testing.T) {
			check := func(name string, now time.Time, want bool) {
				o, ok := Active(now, bd)
				got := ok && o.Kind == tt.kind
				if got != want {
					t.Errorf("%s (%v): active=%v want %v (got %v, %v)", name, now, got, want, o.Kind, ok)
				}
			}
			check("start", tt.start, true)
			check("end-1ns", tt.end.Add(-1), true)
			check("start-1ns", tt.start.Add(-1), false)
			check("end", tt.end, false)
		})
	}
}

func TestActive_NewYearCrossesYear(t *testing.T) {
	useFixture(t, fixture())
	for _, now := range []time.Time{
		time.Date(2026, 12, 31, 23, 59, 0, 0, time.UTC),
		time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC),
	} {
		if o, ok := Active(now, nil); !ok || o.Kind != NewYear {
			t.Errorf("%v: got %v, %v; want NewYear", now, o.Kind, ok)
		}
	}
	now := time.Date(2026, 12, 31, 17, 59, 0, 0, time.UTC)
	// The brief says Christmas here, but its own table ends Christmas on Dec 27.
	if o, ok := Active(now, nil); ok {
		t.Errorf("%v: got %v; want no event before New Year starts", now, o.Kind)
	}
}

func TestActive_Priority(t *testing.T) {
	f := fixture()
	f[Diwali] = map[int]span{2026: d(time.November, 1)}
	useFixture(t, f)

	now := time.Date(2026, 12, 25, 10, 0, 0, 0, time.UTC)
	if o, ok := Active(now, &MonthDay{time.December, 25}); !ok || o.Kind != Birthday {
		t.Errorf("birthday: got %v, %v", o.Kind, ok)
	}
	now = time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC)
	if o, ok := Active(now, nil); !ok || o.Kind != Halloween {
		t.Errorf("halloween vs diwali: got %v, %v", o.Kind, ok)
	}
}

func TestActive_Feb29BirthdayNonLeap(t *testing.T) {
	useFixture(t, fixture())
	bd := &MonthDay{time.February, 29}
	for _, now := range []time.Time{
		time.Date(2027, 2, 28, 12, 0, 0, 0, time.UTC),
		time.Date(2028, 2, 29, 12, 0, 0, 0, time.UTC),
	} {
		if o, ok := Active(now, bd); !ok || o.Kind != Birthday {
			t.Errorf("%v: got %v, %v", now, o.Kind, ok)
		}
	}
}

func TestActive_MissingYearIsInactive(t *testing.T) {
	useFixture(t, fixture())
	now := time.Date(2031, 11, 1, 12, 0, 0, 0, time.UTC)
	if o, ok := Active(now, nil); ok {
		t.Errorf("got %v, want none", o.Kind)
	}
}

func TestOccurrence_Day(t *testing.T) {
	useFixture(t, fixture())
	var onam Occurrence
	for _, o := range Occurrences(2026, nil) {
		if o.Kind == Onam {
			onam = o
		}
	}
	tests := []struct {
		now             time.Time
		wantDay, wantOf int
	}{
		{time.Date(2026, 8, 17, 0, 0, 0, 0, time.Local), 0, 10},
		{time.Date(2026, 8, 26, 23, 0, 0, 0, time.Local), 9, 10},
	}
	for _, tt := range tests {
		if day, days := onam.Day(tt.now); day != tt.wantDay || days != tt.wantOf {
			t.Errorf("%v: got (%d,%d) want (%d,%d)", tt.now, day, days, tt.wantDay, tt.wantOf)
		}
	}
}

func TestParseBirthday(t *testing.T) {
	ok := map[string]MonthDay{
		"05-17":      {time.May, 17},
		"1990-05-17": {time.May, 17},
		"02-29":      {time.February, 29},
	}
	for in, want := range ok {
		got, err := ParseBirthday(in)
		if err != nil || got != want {
			t.Errorf("%q: got %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"13-01", "02-30", "may 17", "", "1990-02-30"} {
		if _, err := ParseBirthday(in); err == nil {
			t.Errorf("%q: expected error", in)
		}
	}
}

package events

import (
	"testing"
	"time"
)

var movingKinds = []Kind{Sankranti, LunarNewYear, Holi, Vishu, Eid, Onam, Diwali}

func TestTable_NearYearsComplete(t *testing.T) {
	for _, k := range movingKinds {
		for y := 2026; y <= 2030; y++ {
			if _, ok := moving[k][y]; !ok {
				t.Errorf("%v: no entry for %d", k, y)
			}
		}
	}
}

func TestTable_WellFormed(t *testing.T) {
	date := func(y int, d day) time.Time { return time.Date(y, d.m, d.d, 0, 0, 0, 0, time.UTC) }
	in := func(d day, from, to day) bool {
		v := int(d.m)*100 + d.d
		return v >= int(from.m)*100+from.d && v <= int(to.m)*100+to.d
	}
	for k, years := range moving {
		for y, s := range years {
			start, end := date(y, s.start), date(y, s.end)
			// time.Date normalises out-of-range days, so a round trip catches them.
			if start.Month() != s.start.m || start.Day() != s.start.d || end.Month() != s.end.m || end.Day() != s.end.d {
				t.Errorf("%v %d: invalid date in %+v", k, y, s)
				continue
			}
			if end.Before(start) {
				t.Errorf("%v %d: end %v before start %v", k, y, end, start)
			}
			if k != Onam && s.start != s.end {
				t.Errorf("%v %d: single-day event has span %+v", k, y, s)
			}
			var ok bool
			switch k {
			case Onam:
				n := int(end.Sub(start).Hours() / 24)
				ok = n >= 8 && n <= 10 && (s.end.m == time.August || s.end.m == time.September)
			case Diwali:
				ok = s.start.m == time.October || s.start.m == time.November
			case Holi:
				ok = s.start.m == time.February || s.start.m == time.March
			case Vishu:
				ok = in(s.start, day{time.April, 13}, day{time.April, 15})
			case Sankranti:
				ok = in(s.start, day{time.January, 13}, day{time.January, 16})
			case LunarNewYear:
				ok = in(s.start, day{time.January, 20}, day{time.February, 21})
			case Eid:
				// The date must be a real day of the key's year (an Eid near
				// a year boundary must not be filed under the wrong year).
				ok = start.Year() == y && start.Month() == s.start.m && start.Day() == s.start.d
			default:
				t.Errorf("unexpected kind %v in moving table", k)
				continue
			}
			if !ok {
				t.Errorf("%v %d: implausible span %+v", k, y, s)
			}
		}
	}
}

// Spot checks are copied from the cited sources, not from the table.
func TestTable_SpotChecks(t *testing.T) {
	type check struct {
		kind       Kind
		year       int
		start, end day
	}
	one := func(k Kind, y int, m time.Month, d int) check { return check{k, y, day{m, d}, day{m, d}} }
	checks := []check{
		// Makar Sankranti, AstroSage (New Delhi); 2027 also drikpanchang.
		one(Sankranti, 2026, time.January, 14),
		one(Sankranti, 2027, time.January, 15),
		one(Sankranti, 2035, time.January, 15),
		// Lunar New Year, Hong Kong Observatory conversion tables.
		one(LunarNewYear, 2026, time.February, 17),
		one(LunarNewYear, 2033, time.January, 31),
		one(LunarNewYear, 2040, time.February, 12),
		// Holi (Rangwali), drikpanchang New Delhi.
		one(Holi, 2026, time.March, 4),
		one(Holi, 2029, time.March, 1),
		one(Holi, 2031, time.March, 9),
		// Eid al-Fitr, Umm al-Qura via al-habib.info.
		one(Eid, 2026, time.March, 20),
		one(Eid, 2028, time.February, 26),
		one(Eid, 2040, time.October, 7),
		// Vishu: Kerala 2026 holiday list; drikpanchang (Cochin).
		one(Vishu, 2026, time.April, 15),
		one(Vishu, 2028, time.April, 14),
		one(Vishu, 2030, time.April, 15),
		// Onam, Atham to Thiruvonam: drikpanchang Onam calendar (Cochin).
		{Onam, 2026, day{time.August, 16}, day{time.August, 26}},
		{Onam, 2029, day{time.August, 13}, day{time.August, 22}},
		{Onam, 2030, day{time.August, 31}, day{time.September, 9}},
		// Diwali (Lakshmi Puja), drikpanchang New Delhi.
		one(Diwali, 2026, time.November, 8),
		one(Diwali, 2028, time.October, 17),
		one(Diwali, 2033, time.October, 22),
	}
	for _, c := range checks {
		got, ok := moving[c.kind][c.year]
		if !ok {
			t.Errorf("%v %d: missing", c.kind, c.year)
			continue
		}
		if want := (span{c.start, c.end}); got != want {
			t.Errorf("%v %d = %+v, want %+v", c.kind, c.year, got, want)
		}
	}
}

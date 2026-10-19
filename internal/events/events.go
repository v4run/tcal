// Package events decides which holiday event, if any, is active at an instant.
package events

import (
	"fmt"
	"sort"
	"strconv"
	"time"
)

// Kind identifies a holiday event.
type Kind int

// The event kinds. The order is the final tie-break in priority.
const (
	NewYear Kind = iota
	Sankranti
	LunarNewYear
	Valentines
	Holi
	Vishu
	Eid
	IndependenceDay
	Onam
	Halloween
	Diwali
	Christmas
	Birthday
)

var kindNames = [...]string{
	"new-year", "sankranti", "lunar-new-year", "valentines", "holi", "vishu",
	"eid", "independence-day", "onam", "halloween", "diwali", "christmas", "birthday",
}

// String returns the kebab-case name of the kind.
func (k Kind) String() string {
	if k < 0 || int(k) >= len(kindNames) {
		return fmt.Sprintf("kind(%d)", int(k))
	}
	return kindNames[k]
}

// MonthDay is a calendar date without a year.
type MonthDay struct {
	Month time.Month
	Day   int
}

// ParseBirthday parses "MM-DD" or "YYYY-MM-DD", ignoring the year. Feb 29 is valid.
func ParseBirthday(s string) (MonthDay, error) {
	md := s
	if len(s) == len("2006-01-02") {
		if _, err := strconv.Atoi(s[:4]); err != nil || s[4] != '-' {
			return MonthDay{}, fmt.Errorf("invalid birthday %q: want MM-DD or YYYY-MM-DD", s)
		}
		md = s[5:]
	}
	// Parse in leap year 2000 so that 02-29 is accepted.
	t, err := time.Parse("2006-01-02", "2000-"+md)
	if err != nil || len(md) != len("01-02") {
		return MonthDay{}, fmt.Errorf("invalid birthday %q: want MM-DD or YYYY-MM-DD", s)
	}
	return MonthDay{t.Month(), t.Day()}, nil
}

func isLeap(y int) bool { return y%4 == 0 && (y%100 != 0 || y%400 == 0) }

// Occurrence is one instance of an event. End is exclusive.
type Occurrence struct {
	Kind       Kind
	Start, End time.Time
}

// civil returns t's calendar date as a UTC midnight, so differences count
// calendar days regardless of DST.
func civil(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// Day returns the 0-based calendar-day index of now within the occurrence and
// the total number of calendar days the occurrence touches.
func (o Occurrence) Day(now time.Time) (day, days int) {
	start := civil(o.Start)
	days = int(civil(o.End.Add(-1)).Sub(start).Hours()/24+0.5) + 1
	day = int(civil(now).Sub(start).Hours()/24 + 0.5)
	return day, days
}

// Active returns the event active at now, in now's location. A birthday wins;
// otherwise the shortest window wins, then the lowest Kind.
func Active(now time.Time, birthday *MonthDay) (Occurrence, bool) {
	var cands []Occurrence
	// Only New Year starts in the previous year, but checking both is cheap.
	for _, y := range []int{now.Year() - 1, now.Year()} {
		for _, o := range occurrences(y, now.Location(), birthday) {
			if !now.Before(o.Start) && now.Before(o.End) {
				cands = append(cands, o)
			}
		}
	}
	if len(cands) == 0 {
		return Occurrence{}, false
	}
	sort.Slice(cands, func(i, j int) bool {
		a, b := cands[i], cands[j]
		if (a.Kind == Birthday) != (b.Kind == Birthday) {
			return a.Kind == Birthday
		}
		if da, db := a.End.Sub(a.Start), b.End.Sub(b.Start); da != db {
			return da < db
		}
		return a.Kind < b.Kind
	})
	return cands[0], true
}

// Occurrences returns every occurrence starting in year, in the local zone.
func Occurrences(year int, birthday *MonthDay) []Occurrence {
	return occurrences(year, time.Local, birthday)
}

func occurrences(year int, loc *time.Location, birthday *MonthDay) []Occurrence {
	at := func(m time.Month, d, h int) time.Time { return time.Date(year, m, d, h, 0, 0, 0, loc) }
	var out []Occurrence
	add := func(k Kind, start, end time.Time) { out = append(out, Occurrence{k, start, end}) }

	add(NewYear, at(time.December, 31, 18), time.Date(year+1, time.January, 2, 0, 0, 0, 0, loc))
	add(Valentines, at(time.February, 14, 0), at(time.February, 15, 0))
	add(IndependenceDay, at(time.August, 15, 0), at(time.August, 16, 0))
	add(Halloween, at(time.October, 31, 0), at(time.November, 1, 0))
	add(Christmas, at(time.December, 1, 0), at(time.December, 27, 0))

	// Moving events: offsets are in days from the span's start (and end for Onam).
	mv := func(k Kind, before, after int, startHour int) {
		s, ok := moving[k][year]
		if !ok {
			return
		}
		start := at(s.start.m, s.start.d+before, startHour)
		end := at(s.end.m, s.end.d+after, 0)
		add(k, start, end)
	}
	mv(Sankranti, 0, 2, 0)
	mv(LunarNewYear, 0, 3, 0)
	mv(Holi, -1, 1, 18)
	mv(Vishu, 0, 1, 0)
	mv(Eid, -1, 2, 0)
	mv(Onam, 0, 1, 0)
	mv(Diwali, -2, 3, 0)

	if birthday != nil {
		m, d := birthday.Month, birthday.Day
		if m == time.February && d == 29 && !isLeap(year) {
			d = 28
		}
		add(Birthday, at(m, d, 0), at(m, d+1, 0))
	}
	return out
}

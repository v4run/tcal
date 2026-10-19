package events

import "time"

// day is a month and day of month, without a year.
type day struct {
	m time.Month
	d int
}

// span is an inclusive range of dates. For single-date events end == start;
// Onam runs from Atham to Thiruvonam.
type span struct{ start, end day }

// moving holds the per-year dates of events that follow lunar or other
// non-Gregorian calendars. A year with no entry has no occurrence.
var moving = map[Kind]map[int]span{}

package calendar

import "time"

// Day is one cell in the calendar grid.
type Day struct {
	Date    time.Time // local midnight
	InMonth bool      // true if Date.Month() == the Month this Day belongs to
}

// Month is a calendar page: 6 rows × 7 days, padded with adjacent-month days.
type Month struct {
	Year  int
	Month time.Month
	Weeks [][]Day // always 6 rows of 7
}

// BuildMonths returns count consecutive Month values starting at start's month.
// start may be any day in the first month; only the year/month are read.
func BuildMonths(start time.Time, count int, weekStart time.Weekday) []Month {
	out := make([]Month, 0, count)
	cursor := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())
	for i := 0; i < count; i++ {
		out = append(out, buildOne(cursor, weekStart))
		cursor = cursor.AddDate(0, 1, 0)
	}
	return out
}

func buildOne(first time.Time, weekStart time.Weekday) Month {
	year, mon, _ := first.Date()
	leadingPad := WeekdayIndex(first.Weekday(), weekStart)

	weeks := make([][]Day, 6)
	cursor := first.AddDate(0, 0, -leadingPad)
	for w := 0; w < 6; w++ {
		row := make([]Day, 7)
		for c := 0; c < 7; c++ {
			row[c] = Day{
				Date:    cursor,
				InMonth: cursor.Month() == mon && cursor.Year() == year,
			}
			cursor = cursor.AddDate(0, 0, 1)
		}
		weeks[w] = row
	}

	return Month{Year: year, Month: mon, Weeks: weeks}
}

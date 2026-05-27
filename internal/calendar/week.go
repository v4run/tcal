package calendar

import "time"

// WeekdayIndex returns the 0..6 column index of day given a week-start day.
func WeekdayIndex(day, weekStart time.Weekday) int {
	return (int(day) - int(weekStart) + 7) % 7
}

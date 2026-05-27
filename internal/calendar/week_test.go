package calendar

import (
	"testing"
	"time"
)

func TestWeekdayIndex(t *testing.T) {
	tests := []struct {
		name      string
		day       time.Weekday
		weekStart time.Weekday
		want      int
	}{
		{"sunday-start: Sunday", time.Sunday, time.Sunday, 0},
		{"sunday-start: Saturday", time.Saturday, time.Sunday, 6},
		{"sunday-start: Wednesday", time.Wednesday, time.Sunday, 3},
		{"monday-start: Monday", time.Monday, time.Monday, 0},
		{"monday-start: Sunday", time.Sunday, time.Monday, 6},
		{"monday-start: Wednesday", time.Wednesday, time.Monday, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := WeekdayIndex(tc.day, tc.weekStart)
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

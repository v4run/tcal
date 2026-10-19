package main

import (
	"testing"
	"time"
)

func TestParseBgTime(t *testing.T) {
	now := time.Date(2026, 10, 19, 9, 15, 42, 0, time.UTC)
	cases := []struct {
		in   string
		want time.Time
	}{
		{"", now},
		{"22:30", time.Date(2026, 10, 19, 22, 30, 0, 0, time.UTC)},
		{"2024-04-23", time.Date(2024, 4, 23, 9, 15, 42, 0, time.UTC)},
		{"2024-04-23T22:30", time.Date(2024, 4, 23, 22, 30, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		got, err := parseBgTime(tc.in, now)
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if !got.Equal(tc.want) {
			t.Errorf("%q: got %v, want %v", tc.in, got, tc.want)
		}
	}
	for _, bad := range []string{"25:00", "noon", "2024-13-01", "2024-04-23 22:30"} {
		if _, err := parseBgTime(bad, now); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}

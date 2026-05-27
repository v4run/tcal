package render

import (
	"testing"
	"time"

	"github.com/v4run/tcal/internal/calendar"
	"github.com/v4run/tcal/internal/golden"
)

func TestRenderMonth_April2026_SundayStart_NoToday(t *testing.T) {
	months := calendar.BuildMonths(
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		1,
		time.Sunday,
	)
	opts := Options{
		WeekStart: time.Sunday,
		Highlight: HighlightAll,
		Color:     false, // disable ANSI for stable goldens
	}
	got := RenderMonth(months[0], time.Time{}, opts)
	golden.Assert(t, "testdata/month_april_2026_sunday.golden", got)
}

func TestRenderMonth_April2026_SundayStart_Today22(t *testing.T) {
	months := calendar.BuildMonths(
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		1,
		time.Sunday,
	)
	today := time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC)
	opts := Options{
		WeekStart: time.Sunday,
		Highlight: HighlightBracket, // bracket-only keeps the golden printable
		Color:     false,
	}
	got := RenderMonth(months[0], today, opts)
	golden.Assert(t, "testdata/month_april_2026_sunday_today_22.golden", got)
}

func TestRenderMonth_February2024_MondayStart_Today29(t *testing.T) {
	months := calendar.BuildMonths(
		time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
		1,
		time.Monday,
	)
	today := time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)
	opts := Options{
		WeekStart: time.Monday,
		Highlight: HighlightBracket,
		Color:     false,
	}
	got := RenderMonth(months[0], today, opts)
	golden.Assert(t, "testdata/month_february_2024_monday_today_29.golden", got)
}

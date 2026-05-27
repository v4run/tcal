package render

import (
	"strings"
	"testing"
	"time"

	"github.com/varun/tcal/internal/golden"
)

func TestClock_Block_FullTime(t *testing.T) {
	now := time.Date(2026, 4, 22, 14, 32, 7, 0, time.UTC)
	got := RenderClock(now, ClockBlock)
	golden.Assert(t, "testdata/clock_block_143207.golden", got)
}

func TestClock_Block_HasFiveLines(t *testing.T) {
	now := time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC)
	got := RenderClock(now, ClockBlock)
	if lines := strings.Count(got, "\n") + 1; lines != 5 {
		t.Errorf("expected 5 rows, got %d", lines)
	}
}

func TestClock_Inline_OneLine(t *testing.T) {
	now := time.Date(2026, 4, 22, 14, 32, 7, 0, time.UTC)
	got := RenderClock(now, ClockInline)
	if strings.Contains(got, "\n") {
		t.Errorf("inline clock should be one line, got %q", got)
	}
	golden.Assert(t, "testdata/clock_inline.golden", got)
}

func TestClock_Boxed_ContainsBorders(t *testing.T) {
	now := time.Date(2026, 4, 22, 14, 32, 7, 0, time.UTC)
	got := RenderClock(now, ClockBoxed)
	if !strings.Contains(got, "│") || !strings.Contains(got, "─") {
		t.Errorf("boxed clock should contain box-drawing characters; got %q", got)
	}
}

func TestClock_Boxed_FullBox(t *testing.T) {
	now := time.Date(2026, 4, 22, 14, 32, 7, 0, time.UTC)
	got := RenderClock(now, ClockBoxed)
	golden.Assert(t, "testdata/clock_boxed.golden", got)
}

func TestClock_Boxed_LeapYearDec31(t *testing.T) {
	now := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	got := RenderClock(now, ClockBoxed)
	if !strings.Contains(got, "Day 366 of 366") {
		t.Errorf("expected 'Day 366 of 366' on leap-year Dec 31, got:\n%s", got)
	}
}

package render

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/v4run/tcal/internal/calendar"
	"github.com/v4run/tcal/internal/golden"
)

func thinState(width, height int) State {
	return State{
		Anchor: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		Today:  time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC),
		Now:    time.Date(2026, 10, 21, 20, 10, 30, 0, time.UTC),
		Layout: LayoutHorizontal,
		Months: 1,
		Width:  width,
		Height: height,
	}
}

func thinOpts() Options {
	return Options{WeekStart: time.Sunday, Highlight: HighlightAll, WithClock: true, Look: LookThin}
}

func TestThinFrame_OneMonth(t *testing.T) {
	got := Frame(thinState(60, 32), thinOpts())
	golden.Assert(t, "testdata/thin_frame_october2026_today21.golden", got)
}

func TestThinMonth_RowsAreFullWidth(t *testing.T) {
	m := calendar.BuildMonths(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), 1, time.Monday)[0]
	o := thinOpts()
	o.Color = true
	got := RenderThinMonth(m, time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC), o)
	lines := strings.Split(got, "\n")
	if len(lines) != thinMonthHeight {
		t.Fatalf("got %d lines, want %d", len(lines), thinMonthHeight)
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w != thinMonthWidth && strings.TrimSpace(ansi.Strip(l)) != "" {
			t.Errorf("line %d is %d wide, want %d: %q", i, w, thinMonthWidth, l)
		}
	}
}

func TestThinMonth_TodayWithoutColorGetsBrackets(t *testing.T) {
	m := calendar.BuildMonths(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), 1, time.Sunday)[0]
	o := thinOpts() // Color false, highlight combined
	got := RenderThinMonth(m, time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC), o)
	if !strings.Contains(got, "┌") || !strings.Contains(got, "┘") {
		t.Error("today should be bracketed when color is off")
	}
	if strings.Contains(got, "\x1b[") {
		t.Error("no escapes expected without color")
	}

	o.Highlight = HighlightNone
	if got := RenderThinMonth(m, time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC), o); strings.Contains(got, "┌") {
		t.Error("highlight none should not bracket today")
	}
}

func TestThinMonth_TodaySolidAccentWithColor(t *testing.T) {
	m := calendar.BuildMonths(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), 1, time.Sunday)[0]
	o := thinOpts()
	o.Color = true
	got := RenderThinMonth(m, time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC), o)
	if strings.Contains(got, "┌") {
		t.Error("combined highlight should not bracket today when color is on")
	}
	block := lipgloss.NewStyle().Background(accentColor).Foreground(lipgloss.Color("0")).Bold(true)
	if !strings.Contains(got, block.Render(" ╺┓  ╻ ")) {
		t.Error("today's top row should be a solid accent block")
	}
}

func TestThinClock_SecondsBar(t *testing.T) {
	for _, tc := range []struct {
		sec    int
		filled int
	}{{0, 0}, {30, thinClockWidth / 2}, {59, thinClockWidth * 59 / 60}} {
		now := time.Date(2026, 10, 21, 9, 5, tc.sec, 0, time.UTC)
		bar := strings.Split(RenderThinClock(now, false), "\n")[3]
		if got := strings.Count(bar, "━"); got != tc.filled {
			t.Errorf("sec %d: %d filled, want %d", tc.sec, got, tc.filled)
		}
		if got := strings.Count(bar, "━") + strings.Count(bar, "─"); got != thinClockWidth {
			t.Errorf("sec %d: bar is %d wide, want %d", tc.sec, got, thinClockWidth)
		}
	}
}

func TestThinFrame_FallsBackWhenTooSmall(t *testing.T) {
	classic := thinOpts()
	classic.Look = LookClassic
	for _, st := range []State{thinState(40, 32), thinState(60, 20)} {
		if got, want := Frame(st, thinOpts()), Frame(st, classic); got != want {
			t.Errorf("%dx%d: thin should fall back to classic", st.Width, st.Height)
		}
	}
}

func TestThinFrame_ThreeMonthsSideBySide(t *testing.T) {
	st := thinState(160, 0)
	st.Months = 3
	got := ansi.Strip(Frame(st, thinOpts()))
	line := ""
	for _, l := range strings.Split(got, "\n") {
		if strings.Contains(l, "October 2026") {
			line = l
		}
	}
	if !strings.Contains(line, "September 2026") || !strings.Contains(line, "November 2026") {
		t.Errorf("months should share one title row, got %q", line)
	}
}

func TestFrame_OneMonthShowsAnchorMonth(t *testing.T) {
	st := thinState(40, 0)
	o := thinOpts()
	o.Look = LookClassic
	if got := Frame(st, o); !strings.Contains(got, "October 2026") || strings.Contains(got, "September") {
		t.Errorf("one month should be the anchor month:\n%s", got)
	}
}

# tcal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

## Post-implementation amendments

- 2026-05-27: `HighlightAll` semantics changed to `Reverse | Color` (no brackets) per user feedback after Task 14. See commit history.
- 2026-05-27: Dropped `LayoutFocus` per user feedback after Task 14. Focus and Horizontal produced identical output in v0.1.0; collapsing them frees the `4` keybinding. Default layout is now `horizontal`. See commit history.

**Goal:** Build `tcal`, a Go CLI that renders a live, center-aligned calendar widget with a block-digit clock and four multi-month layouts, plus a one-shot `--print` mode that emits the same layouts to stdout.

**Architecture:** Pure renderer + thin wrappers. A single `internal/render` package transforms `(state, opts) → string`; a Bubble Tea model wraps it for live mode, a one-shot handler wraps it for print. The renderer never imports the TUI; both modes share 100% of layout/centering code.

**Tech Stack:** Go (≥1.22), `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss`, `golang.org/x/term`. Tests use the standard `testing` package + golden-file fixtures.

---

## Conventions used throughout this plan

- **TDD:** every code task writes a failing test first, runs it to confirm failure, then implements minimal code, then runs to confirm pass, then commits.
- **Commits:** each task ends with one commit. Commit messages use conventional-commit style (`feat:`, `test:`, `chore:`, `docs:`).
- **Goldens:** golden-file tests live under `testdata/` next to the test file. They are updated via `go test ./... -update` (the `-update` flag is wired in Task 2).
- **No `time.Now()` inside the renderer.** Time always enters via the `State` struct so tests are deterministic. The TUI/print orchestrators inject `time.Now()`.
- **Hex `0x` for terminal bytes; ANSI through lipgloss only.** Never hand-roll escape sequences in `render/*`.

---

## Task 0: Project scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/tcal/main.go` (placeholder)
- Create: `README.md` (placeholder)

- [ ] **Step 1: Initialize Go module**

Run from project root:
```bash
go mod init github.com/v4run/tcal
```

- [ ] **Step 2: Add direct dependencies**

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get golang.org/x/term@latest
```

- [ ] **Step 3: Create placeholder entry point**

Write `cmd/tcal/main.go`:
```go
package main

import "fmt"

const version = "0.0.0-dev"

func main() {
	fmt.Println("tcal", version)
}
```

- [ ] **Step 4: Verify it builds and runs**

```bash
go build -o tcal ./cmd/tcal
./tcal
```
Expected output: `tcal 0.0.0-dev`

- [ ] **Step 5: Create placeholder README**

Write `README.md`:
```markdown
# tcal

A live terminal calendar widget with a block-digit clock and four multi-month layouts.

Status: in development. See `docs/superpowers/specs/2026-05-27-tcal-design.md`.
```

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum cmd/tcal/main.go README.md
git commit -m "chore: scaffold Go module and placeholder entry point"
```

---

## Task 1: `internal/calendar` — Day, Month, BuildMonths

**Files:**
- Create: `internal/calendar/month.go`
- Create: `internal/calendar/month_test.go`
- Create: `internal/calendar/week.go`
- Create: `internal/calendar/week_test.go`

**Design notes:**
- `BuildMonths(start, count, weekStart) → []Month`. `start` is the first month to build (callers compute it; e.g. the horizontal layout passes `anchor minus one month`).
- Each `Month` holds a 6×7 grid of `Day` values; leading/trailing cells are filled with adjacent-month days and marked `InMonth=false`.
- `Today` is a separate parameter (passed via opts later) so the calendar package stays pure.

- [ ] **Step 1: Write failing test for week.WeekdayIndex**

Write `internal/calendar/week_test.go`:
```go
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
```

- [ ] **Step 2: Run test, confirm fail**

```bash
go test ./internal/calendar/...
```
Expected: build fails — `WeekdayIndex` undefined.

- [ ] **Step 3: Implement WeekdayIndex**

Write `internal/calendar/week.go`:
```go
package calendar

import "time"

// WeekdayIndex returns the 0..6 column index of day given a week-start day.
func WeekdayIndex(day, weekStart time.Weekday) int {
	return (int(day) - int(weekStart) + 7) % 7
}
```

- [ ] **Step 4: Run test, confirm pass**

```bash
go test ./internal/calendar/...
```
Expected: PASS.

- [ ] **Step 5: Write failing test for BuildMonths**

Write `internal/calendar/month_test.go`:
```go
package calendar

import (
	"testing"
	"time"
)

func TestBuildMonths_AprilCount3_SundayStart(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	got := BuildMonths(start, 3, time.Sunday)

	if len(got) != 3 {
		t.Fatalf("got %d months, want 3", len(got))
	}

	apr := got[0]
	if apr.Year != 2026 || apr.Month != time.April {
		t.Errorf("first month: got %d-%s, want 2026-April", apr.Year, apr.Month)
	}
	if len(apr.Weeks) != 6 {
		t.Errorf("got %d weeks, want 6", len(apr.Weeks))
	}
	if len(apr.Weeks[0]) != 7 {
		t.Errorf("got %d days in first week, want 7", len(apr.Weeks[0]))
	}

	// April 1, 2026 is a Wednesday — with Sunday start, that's column 3.
	// Columns 0..2 should be March 29, 30, 31 (InMonth=false).
	w0 := apr.Weeks[0]
	if w0[3].Date.Day() != 1 || !w0[3].InMonth {
		t.Errorf("week 0 col 3: got day=%d inMonth=%v, want day=1 inMonth=true",
			w0[3].Date.Day(), w0[3].InMonth)
	}
	if w0[2].Date.Day() != 31 || w0[2].InMonth {
		t.Errorf("week 0 col 2 (leading padding): got day=%d inMonth=%v, want day=31 inMonth=false",
			w0[2].Date.Day(), w0[2].InMonth)
	}
}

func TestBuildMonths_LeapYearFebruary(t *testing.T) {
	start := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	got := BuildMonths(start, 1, time.Monday)

	feb := got[0]
	// Feb 2024 has 29 days. Find day 29 in the grid.
	var found bool
	for _, w := range feb.Weeks {
		for _, d := range w {
			if d.InMonth && d.Date.Day() == 29 && d.Date.Month() == time.February {
				found = true
			}
		}
	}
	if !found {
		t.Error("Feb 29 2024 not found in grid")
	}
}
```

- [ ] **Step 6: Run test, confirm fail**

```bash
go test ./internal/calendar/...
```
Expected: build fails — `BuildMonths`, `Day`, `Month` undefined.

- [ ] **Step 7: Implement Day, Month, BuildMonths**

Write `internal/calendar/month.go`:
```go
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
```

- [ ] **Step 8: Run tests, confirm pass**

```bash
go test ./internal/calendar/...
```
Expected: PASS for both tests.

- [ ] **Step 9: Commit**

```bash
git add internal/calendar/
git commit -m "feat(calendar): Day/Month types + BuildMonths with week-start support"
```

---

## Task 2: Golden-file test helper + `-update` flag

**Files:**
- Create: `internal/golden/golden.go`
- Create: `internal/golden/golden_test.go`

**Why a shared helper:** every renderer test compares strings to fixtures. One helper keeps the pattern consistent and centralizes the `-update` flag.

- [ ] **Step 1: Write failing test for the helper**

Write `internal/golden/golden_test.go`:
```go
package golden

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAssert_PassesWhenContentMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.golden")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	Assert(t, path, "hello\n")
}

func TestAssert_FailsWhenContentDiffers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.golden")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fake := &recordingT{}
	Assert(fake, path, "goodbye\n")
	if !fake.failed {
		t.Error("expected Assert to fail on mismatch")
	}
}

type recordingT struct {
	testing.TB
	failed bool
}

func (r *recordingT) Errorf(format string, args ...any) { r.failed = true }
func (r *recordingT) Helper()                           {}
```

- [ ] **Step 2: Run test, confirm fail**

```bash
go test ./internal/golden/...
```
Expected: build fails — `Assert` undefined.

- [ ] **Step 3: Implement the helper + `-update` flag**

Write `internal/golden/golden.go`:
```go
// Package golden provides a tiny golden-file test helper with an -update flag.
//
// Usage in a test:
//
//	golden.Assert(t, "testdata/april_sunday.golden", got)
//
// To refresh fixtures after an intentional render change:
//
//	go test ./... -update
package golden

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update golden files in place")

// tb is the subset of testing.TB we use; the indirection lets tests fake it.
type tb interface {
	Helper()
	Errorf(format string, args ...any)
}

// Assert compares got to the contents of the file at path. When -update is
// passed to the test binary, Assert writes got to path instead of comparing.
func Assert(t tb, path, got string) {
	t.Helper()
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Errorf("golden: mkdir: %v", err)
			return
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Errorf("golden: write %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("golden: read %s: %v (run `go test ./... -update` to create)", path, err)
		return
	}
	if string(want) != got {
		t.Errorf("golden mismatch in %s\n--- want ---\n%s\n--- got ---\n%s", path, string(want), got)
	}
}
```

- [ ] **Step 4: Run tests, confirm pass**

```bash
go test ./internal/golden/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/golden/
git commit -m "test: add golden-file helper with -update flag"
```

---

## Task 3: `internal/render/center` — H+V centering

**Files:**
- Create: `internal/render/center.go`
- Create: `internal/render/center_test.go`

**Spec ref:** "centering edge cases" section. Block wider/taller than viewport falls back to top-left align; print mode disables vertical centering by passing height=0.

- [ ] **Step 1: Write failing tests**

Write `internal/render/center_test.go`:
```go
package render

import (
	"strings"
	"testing"
)

func TestFit_CentersHorizontallyAndVertically(t *testing.T) {
	block := "abc\nde"
	got := Fit(block, 7, 5)

	// width=7, content=3 → 2 spaces left, 2 right.
	// height=5, content=2 → 1 blank line above, 2 below (extra line goes below on odd diff).
	want := strings.Join([]string{
		"       ",
		"  abc  ",
		"  de   ",
		"       ",
		"       ",
	}, "\n")
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestFit_NoVerticalCenteringWhenHeightZero(t *testing.T) {
	block := "abc"
	got := Fit(block, 7, 0)
	want := "  abc  "
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFit_TooNarrow_FallsBackToLeftAlign(t *testing.T) {
	block := "abcdefgh" // 8 wide
	got := Fit(block, 5, 0)
	if got != "abcdefgh" {
		t.Errorf("expected unchanged content on overflow, got %q", got)
	}
}

func TestFit_TooTall_TopAlignsWithinViewport(t *testing.T) {
	block := "a\nb\nc\nd\ne"
	got := Fit(block, 1, 3)
	// 5 lines into height=3 → top-align, no padding above.
	want := "a\nb\nc\nd\ne"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFit_PadsRightToFullWidth(t *testing.T) {
	block := "ab\ncde"
	got := Fit(block, 7, 0)
	// Each line padded to width=7 with content centered (block max width = 3).
	// (7-3)/2 = 2 spaces left, 2 right. Shorter lines still padded to 7.
	lines := strings.Split(got, "\n")
	for i, ln := range lines {
		if len(ln) != 7 {
			t.Errorf("line %d width=%d, want 7 (%q)", i, len(ln), ln)
		}
	}
}
```

- [ ] **Step 2: Run tests, confirm fail**

```bash
go test ./internal/render/...
```
Expected: build fails — `Fit` undefined.

- [ ] **Step 3: Implement Fit**

Write `internal/render/center.go`:
```go
package render

import "strings"

// Fit centers block (a newline-separated string) within (width, height).
// height=0 disables vertical centering (used by print mode).
// If the block is wider than width, content is returned unchanged (no truncation;
// callers may render a "too narrow" hint elsewhere). If taller than height,
// content is returned unchanged.
func Fit(block string, width, height int) string {
	lines := strings.Split(block, "\n")

	maxW := 0
	for _, ln := range lines {
		if n := visibleWidth(ln); n > maxW {
			maxW = n
		}
	}

	// Horizontal: only pad if the block fits.
	if maxW <= width {
		leftPad := (width - maxW) / 2
		for i, ln := range lines {
			rightPad := width - leftPad - visibleWidth(ln)
			if rightPad < 0 {
				rightPad = 0
			}
			lines[i] = strings.Repeat(" ", leftPad) + ln + strings.Repeat(" ", rightPad)
		}
	}

	// Vertical: only pad if the block fits AND height > 0.
	if height > 0 && len(lines) <= height {
		topPad := (height - len(lines)) / 2
		bottomPad := height - len(lines) - topPad
		// Use a blank line of full width so vertical padding lines are uniform.
		blank := strings.Repeat(" ", min(width, maxW))
		if maxW <= width {
			blank = strings.Repeat(" ", width)
		}
		out := make([]string, 0, height)
		for i := 0; i < topPad; i++ {
			out = append(out, blank)
		}
		out = append(out, lines...)
		for i := 0; i < bottomPad; i++ {
			out = append(out, blank)
		}
		lines = out
	}

	return strings.Join(lines, "\n")
}

// visibleWidth returns rune-count, ignoring ANSI escape sequences.
// (We won't have ANSI in center's input until layouts are built; this is a
// forward-compatible placeholder swapped for lipgloss.Width in a later task.)
func visibleWidth(s string) int {
	return len([]rune(s))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

- [ ] **Step 4: Run tests, confirm pass**

```bash
go test ./internal/render/...
```
Expected: PASS for all four tests.

- [ ] **Step 5: Commit**

```bash
git add internal/render/center.go internal/render/center_test.go
git commit -m "feat(render): Fit centers blocks horizontally and vertically with graceful overflow"
```

---

## Task 4: `internal/render/highlight` — today decoration

**Files:**
- Create: `internal/render/highlight.go`
- Create: `internal/render/highlight_test.go`

**Spec ref:** combined highlight = reverse + bracket + accent color. Each can also be enabled individually via the `--highlight` flag. The bracket variant turns ` 22` into `[22]`; the renderer that calls `Highlight` is responsible for trimming one space from the adjacent gutter to preserve alignment.

- [ ] **Step 1: Write failing tests**

Write `internal/render/highlight_test.go`:
```go
package render

import (
	"strings"
	"testing"
)

func TestHighlight_None(t *testing.T) {
	got := Highlight("22", HighlightNone)
	if got != "22" {
		t.Errorf("got %q, want %q", got, "22")
	}
}

func TestHighlight_Bracket(t *testing.T) {
	got := Highlight("22", HighlightBracket)
	if got != "[22]" {
		t.Errorf("got %q, want %q", got, "[22]")
	}
}

func TestHighlight_BracketWithSingleDigit(t *testing.T) {
	got := Highlight("3", HighlightBracket)
	if got != "[3]" {
		t.Errorf("got %q, want %q", got, "[3]")
	}
}

func TestHighlight_ReverseProducesAnsi(t *testing.T) {
	got := Highlight("22", HighlightReverse)
	// We don't assert exact escape codes (lipgloss owns those), only that
	// the visible text is preserved and ANSI escapes wrap it.
	if !strings.Contains(got, "22") {
		t.Errorf("reverse should preserve text, got %q", got)
	}
	if got == "22" {
		t.Errorf("reverse should add styling, got identical plain string")
	}
}

func TestHighlight_Combined_IncludesBrackets(t *testing.T) {
	got := Highlight("22", HighlightReverse|HighlightBracket|HighlightColor)
	if !strings.Contains(got, "[22]") {
		t.Errorf("combined must contain [22], got %q", got)
	}
}
```

- [ ] **Step 2: Run tests, confirm fail**

```bash
go test ./internal/render/...
```
Expected: fails — `Highlight`, `HighlightMask`, constants undefined.

- [ ] **Step 3: Implement Highlight**

Write `internal/render/highlight.go`:
```go
package render

import "github.com/charmbracelet/lipgloss"

// HighlightMask is a bitmask of decorations applied to today's date.
type HighlightMask uint8

const (
	HighlightNone    HighlightMask = 0
	HighlightReverse HighlightMask = 1 << 0
	HighlightBracket HighlightMask = 1 << 1
	HighlightColor   HighlightMask = 1 << 2
	HighlightAll                   = HighlightReverse | HighlightBracket | HighlightColor
)

// accentColor is used when HighlightColor is set. Lipgloss degrades to the
// nearest supported color on limited terminals and emits no escape sequences
// when NO_COLOR is set or the writer is not a terminal.
var accentColor = lipgloss.Color("214") // soft orange

// Highlight wraps text (a day number string with no surrounding spaces) with
// the selected decorations and returns the resulting renderable string.
func Highlight(text string, mask HighlightMask) string {
	out := text
	if mask&HighlightBracket != 0 {
		out = "[" + out + "]"
	}
	style := lipgloss.NewStyle()
	if mask&HighlightReverse != 0 {
		style = style.Reverse(true).Bold(true)
	}
	if mask&HighlightColor != 0 {
		style = style.Foreground(accentColor)
	}
	if mask&(HighlightReverse|HighlightColor) != 0 {
		out = style.Render(out)
	}
	return out
}
```

- [ ] **Step 4: Run tests, confirm pass**

```bash
go test ./internal/render/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/render/highlight.go internal/render/highlight_test.go
git commit -m "feat(render): Highlight applies reverse/bracket/color decorations"
```

---

## Task 5: `internal/render/month` — single month block

**Files:**
- Create: `internal/render/month.go`
- Create: `internal/render/month_test.go`
- Create: `internal/render/testdata/month_april_2026_sunday.golden`
- Create: `internal/render/testdata/month_april_2026_sunday_today_22.golden`
- Create: `internal/render/testdata/month_february_2024_monday_today_29.golden`

**Output shape:**
```
   April 2026
Su Mo Tu We Th Fr Sa
          1  2  3  4
 5  6  7  8  9 10 11
12 13 14 15 16 17 18
19 20 [21] 22 23 24 25      ← if today=21, bracketed with reverse + color
26 27 28 29 30
```

Each day cell is right-justified to 2 chars with a single-space gutter (`" DD"` repeated). When a day is highlighted as `[DD]`, that cell consumes 4 chars; the leading space of the *following* cell is removed to keep the next column aligned.

- [ ] **Step 1: Write failing test (no today)**

Write `internal/render/month_test.go`:
```go
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
```

- [ ] **Step 2: Run tests, confirm fail**

```bash
go test ./internal/render/...
```
Expected: build fails — `Options`, `RenderMonth` undefined.

- [ ] **Step 3: Define Options and stub State**

Append to `internal/render/highlight.go` (or create `internal/render/types.go`) the shared types. Create `internal/render/types.go`:
```go
package render

import "time"

// Layout selects the multi-month arrangement.
type Layout int

const (
	LayoutFocus Layout = iota
	LayoutHorizontal
	LayoutVertical
	LayoutGrid
)

// ClockStyle selects the clock rendering style.
type ClockStyle int

const (
	ClockBlock ClockStyle = iota
	ClockInline
	ClockBoxed
)

// State is the input to Frame.
type State struct {
	Anchor time.Time
	Today  time.Time
	Now    time.Time
	Layout Layout
	Months int // 0 = layout default
	Width  int
	Height int // 0 disables vertical centering (print mode)
}

// Options configures rendering. WithClock applies only to print mode.
type Options struct {
	WeekStart  time.Weekday
	Color      bool
	Highlight  HighlightMask
	ClockStyle ClockStyle
	WithClock  bool
}
```

- [ ] **Step 4: Implement RenderMonth**

Write `internal/render/month.go`:
```go
package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/v4run/tcal/internal/calendar"
)

// monthHeader returns the centered "Month Year" line, padded to gridWidth.
func monthHeader(m calendar.Month, gridWidth int) string {
	title := fmt.Sprintf("%s %d", m.Month.String(), m.Year)
	pad := (gridWidth - len(title)) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + title
}

// weekdayRow returns "Su Mo Tu We Th Fr Sa"-style header for the given start.
func weekdayRow(weekStart time.Weekday) string {
	labels := []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
	out := make([]string, 7)
	for c := 0; c < 7; c++ {
		out[c] = labels[(int(weekStart)+c)%7]
	}
	return strings.Join(out, " ")
}

// RenderMonth renders one calendar.Month as a multi-line string. The grid is
// 20 chars wide ("Su Mo Tu We Th Fr Sa" = 20 chars) with one space between
// day cells. Today's cell is decorated via opts.Highlight; non-month days
// render as blank cells.
func RenderMonth(m calendar.Month, today time.Time, opts Options) string {
	const gridWidth = 20

	var b strings.Builder
	b.WriteString(monthHeader(m, gridWidth))
	b.WriteString("\n")
	b.WriteString(weekdayRow(opts.WeekStart))
	b.WriteString("\n")

	hasToday := !today.IsZero()
	for _, week := range m.Weeks {
		// Build cells, then join with single spaces. When a cell is the
		// bracketed today, we drop one space of leading whitespace from the
		// next cell to keep alignment (because "[22]" is 4 wide vs " 22" = 3).
		cells := make([]string, 7)
		bracketedAt := -1
		for c, d := range week {
			if !d.InMonth {
				cells[c] = "  "
				continue
			}
			text := fmt.Sprintf("%2d", d.Date.Day())
			isToday := hasToday && sameYMD(d.Date, today)
			if isToday {
				bare := strings.TrimLeft(text, " ")
				cells[c] = Highlight(bare, opts.Highlight)
				if opts.Highlight&HighlightBracket != 0 {
					bracketedAt = c
				}
			} else {
				cells[c] = text
			}
		}

		// Join with single spaces; trim leading space from cell after bracketed.
		var line strings.Builder
		for c, cell := range cells {
			if c > 0 {
				if c == bracketedAt+1 {
					// no separator — the "[" already includes the leading boundary
				} else {
					line.WriteString(" ")
				}
			}
			// If the cell is a 1-digit non-today day, it's "  " or " D"; pad to 2.
			line.WriteString(cell)
		}
		b.WriteString(strings.TrimRight(line.String(), " "))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func sameYMD(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
```

- [ ] **Step 5: Run tests with `-update` to create the goldens**

```bash
go test ./internal/render/... -run RenderMonth -update
```
Expected: PASS; three `.golden` files now exist under `internal/render/testdata/`.

- [ ] **Step 6: Inspect the goldens visually**

Open the three golden files. Verify:
- Month header centered above the grid.
- Weekday labels match the requested week start.
- Today's day appears bracketed in the today-tests; blank days appear as empty cells.
- No trailing whitespace on lines.

- [ ] **Step 7: Run tests without `-update`, confirm pass**

```bash
go test ./internal/render/...
```
Expected: PASS for all RenderMonth tests against the goldens just written.

- [ ] **Step 8: Commit**

```bash
git add internal/render/month.go internal/render/month_test.go internal/render/types.go internal/render/testdata/
git commit -m "feat(render): RenderMonth with golden fixtures for week-start + today highlight"
```

---

## Task 6: `internal/render/clock` — block-digit ASCII clock

**Files:**
- Create: `internal/render/clock.go`
- Create: `internal/render/clock_test.go`
- Create: `internal/render/testdata/clock_block_143207.golden`
- Create: `internal/render/testdata/clock_inline.golden`

**Glyph design:** each digit is 5 rows tall, 4 columns wide using `█`; colon is 1 column wide with `█` at rows 1 and 3. A 1-column gap sits between characters. `14:32:07` is 8 chars; final width = (4×6 digits) + (1×2 colons) + (7 gaps) = 24 + 2 + 7 = 33 cols.

- [ ] **Step 1: Write failing tests**

Write `internal/render/clock_test.go`:
```go
package render

import (
	"strings"
	"testing"
	"time"

	"github.com/v4run/tcal/internal/golden"
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
```

- [ ] **Step 2: Run tests, confirm fail**

```bash
go test ./internal/render/... -run Clock
```
Expected: fails — `RenderClock` undefined.

- [ ] **Step 3: Implement RenderClock**

Write `internal/render/clock.go`:
```go
package render

import (
	"fmt"
	"strings"
	"time"
)

// digitGlyphs are 5 rows × 4 cols each for "0"–"9". Use full-block "█".
var digitGlyphs = map[rune][5]string{
	'0': {
		"████",
		"█  █",
		"█  █",
		"█  █",
		"████",
	},
	'1': {
		"  █ ",
		" ██ ",
		"  █ ",
		"  █ ",
		" ███",
	},
	'2': {
		"████",
		"   █",
		"████",
		"█   ",
		"████",
	},
	'3': {
		"████",
		"   █",
		" ███",
		"   █",
		"████",
	},
	'4': {
		"█  █",
		"█  █",
		"████",
		"   █",
		"   █",
	},
	'5': {
		"████",
		"█   ",
		"████",
		"   █",
		"████",
	},
	'6': {
		"████",
		"█   ",
		"████",
		"█  █",
		"████",
	},
	'7': {
		"████",
		"   █",
		"  █ ",
		" █  ",
		" █  ",
	},
	'8': {
		"████",
		"█  █",
		"████",
		"█  █",
		"████",
	},
	'9': {
		"████",
		"█  █",
		"████",
		"   █",
		"████",
	},
}

// colonGlyph is 5 rows × 1 col.
var colonGlyph = [5]string{
	" ",
	"█",
	" ",
	"█",
	" ",
}

// RenderClock renders the current time per the chosen style.
func RenderClock(now time.Time, style ClockStyle) string {
	switch style {
	case ClockInline:
		return now.Format("Mon, 02 Jan 2006  ·  15:04:05")
	case ClockBoxed:
		date := now.Format("Mon, 02 Jan 2006")
		clk := now.Format("15 : 04 : 05")
		yearDay := now.YearDay()
		_, week := now.ISOWeek()
		line3 := fmt.Sprintf("Week %d · Day %d of 365", week, yearDay)
		width := 31
		top := "┌" + strings.Repeat("─", width-2) + "┐"
		bot := "└" + strings.Repeat("─", width-2) + "┘"
		mid := func(s string) string {
			return "│ " + s + strings.Repeat(" ", width-3-len(s)) + "│"
		}
		return strings.Join([]string{top, mid(date), mid(clk), mid(line3), bot}, "\n")
	default: // ClockBlock
		return renderBlock(now)
	}
}

func renderBlock(now time.Time) string {
	s := now.Format("15:04:05")
	rows := [5]strings.Builder{}
	for i, ch := range s {
		if i > 0 {
			for r := 0; r < 5; r++ {
				rows[r].WriteString(" ")
			}
		}
		var glyph [5]string
		if ch == ':' {
			glyph = colonGlyph
		} else {
			glyph = digitGlyphs[ch]
		}
		for r := 0; r < 5; r++ {
			rows[r].WriteString(glyph[r])
		}
	}
	out := make([]string, 5)
	for r := 0; r < 5; r++ {
		out[r] = rows[r].String()
	}
	return strings.Join(out, "\n")
}
```

- [ ] **Step 4: Run tests with `-update` to create the goldens**

```bash
go test ./internal/render/... -run Clock -update
```
Expected: PASS; two goldens created.

- [ ] **Step 5: Inspect the block clock golden**

Open `internal/render/testdata/clock_block_143207.golden`. Verify:
- 5 lines tall, characters render `14:32:07`.
- Roughly 33 columns wide; consistent gaps between characters.

- [ ] **Step 6: Run tests without `-update`, confirm pass**

```bash
go test ./internal/render/...
```
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/render/clock.go internal/render/clock_test.go internal/render/testdata/clock_*.golden
git commit -m "feat(render): RenderClock with block, inline, boxed styles"
```

---

## Task 7: `internal/render/layouts` — horizontal & vertical

**Files:**
- Create: `internal/render/layouts.go`
- Create: `internal/render/layouts_test.go`
- Create: `internal/render/testdata/layout_horizontal_3.golden`
- Create: `internal/render/testdata/layout_vertical_3.golden`

These two are the simplest layouts; we add grid and focus in the next task.

- [ ] **Step 1: Write failing tests**

Write `internal/render/layouts_test.go`:
```go
package render

import (
	"strings"
	"testing"
	"time"

	"github.com/v4run/tcal/internal/calendar"
	"github.com/v4run/tcal/internal/golden"
)

func anchorApril2026() time.Time {
	return time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
}

func opts() Options {
	return Options{
		WeekStart: time.Sunday,
		Highlight: HighlightBracket,
		Color:     false,
	}
}

func TestLayout_Horizontal_3Months(t *testing.T) {
	o := opts()
	months := calendar.BuildMonths(anchorApril2026().AddDate(0, -1, 0), 3, o.WeekStart)
	got := RenderLayout(LayoutHorizontal, months, time.Time{}, o)
	golden.Assert(t, "testdata/layout_horizontal_3.golden", got)
	// Smoke check: at least one line should be wide enough for 3 grids + gutters.
	for _, ln := range strings.Split(got, "\n") {
		if len(ln) >= 3*20+2*2 {
			return
		}
	}
	t.Error("no line is wide enough for 3 horizontal months")
}

func TestLayout_Vertical_3Months(t *testing.T) {
	o := opts()
	months := calendar.BuildMonths(anchorApril2026().AddDate(0, -1, 0), 3, o.WeekStart)
	got := RenderLayout(LayoutVertical, months, time.Time{}, o)
	golden.Assert(t, "testdata/layout_vertical_3.golden", got)
	// Smoke check: every line should be <= one grid width (20).
	for _, ln := range strings.Split(got, "\n") {
		if visibleWidth(ln) > 22 { // grid 20 + small slop
			t.Errorf("vertical layout line too wide: %q", ln)
		}
	}
}
```

- [ ] **Step 2: Run tests, confirm fail**

```bash
go test ./internal/render/...
```
Expected: fails — `RenderLayout` undefined.

- [ ] **Step 3: Implement horizontal + vertical layouts**

Write `internal/render/layouts.go`:
```go
package render

import (
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/v4run/tcal/internal/calendar"
)

const monthGutter = "  " // two-space gutter between months in horizontal/grid

// RenderLayout joins month blocks per the selected layout. today may be the
// zero value to suppress highlighting.
func RenderLayout(layout Layout, months []calendar.Month, today time.Time, opts Options) string {
	blocks := make([]string, len(months))
	for i, m := range months {
		blocks[i] = RenderMonth(m, today, opts)
	}
	switch layout {
	case LayoutHorizontal:
		return joinHorizontal(blocks)
	case LayoutVertical:
		return joinVertical(blocks)
	default:
		// Filled in by Task 8 (grid, focus).
		return joinVertical(blocks)
	}
}

func joinHorizontal(blocks []string) string {
	if len(blocks) == 0 {
		return ""
	}
	// lipgloss.JoinHorizontal aligns by top and pads short blocks.
	withGutters := make([]string, 0, 2*len(blocks)-1)
	for i, b := range blocks {
		if i > 0 {
			withGutters = append(withGutters, monthGutter)
		}
		withGutters = append(withGutters, b)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, withGutters...)
}

func joinVertical(blocks []string) string {
	if len(blocks) == 0 {
		return ""
	}
	withGutters := make([]string, 0, 2*len(blocks)-1)
	for i, b := range blocks {
		if i > 0 {
			withGutters = append(withGutters, "")
		}
		withGutters = append(withGutters, b)
	}
	return lipgloss.JoinVertical(lipgloss.Left, withGutters...)
}
```

- [ ] **Step 4: Replace `visibleWidth` placeholder with lipgloss.Width**

Edit `internal/render/center.go` — replace the body of `visibleWidth`:
```go
func visibleWidth(s string) int {
	return lipgloss.Width(s)
}
```
Add import `"github.com/charmbracelet/lipgloss"`. Remove the now-unused `len([]rune(s))` line.

- [ ] **Step 5: Run tests with `-update` to create the goldens**

```bash
go test ./internal/render/... -run Layout -update
```
Expected: PASS; two goldens created.

- [ ] **Step 6: Visually inspect goldens**

Open `internal/render/testdata/layout_horizontal_3.golden`. Confirm:
- Three month grids side-by-side, separated by two spaces.
- Headers ("March 2026", "April 2026", "May 2026") align vertically.

Open `internal/render/testdata/layout_vertical_3.golden`. Confirm:
- Three months stacked top-to-bottom with one blank line between.

- [ ] **Step 7: Run tests without `-update`, confirm pass**

```bash
go test ./internal/render/...
```
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/render/layouts.go internal/render/layouts_test.go internal/render/center.go internal/render/testdata/layout_*.golden
git commit -m "feat(render): horizontal and vertical layouts via lipgloss join"
```

---

## Task 8: `internal/render/layouts` — grid & focus

**Files:**
- Modify: `internal/render/layouts.go`
- Modify: `internal/render/layouts_test.go`
- Create: `internal/render/testdata/layout_grid_12.golden`
- Create: `internal/render/testdata/layout_focus_3.golden`

**Grid:** lays months in rows of 3 by default; if `width >= ~4 * 22`, lays out in rows of 4 (computed at compose-time in Task 10 — for this task, accept rows-per-row as a parameter into a helper, but the public `RenderLayout` defaults to 3 cols).

**Focus:** prev (small) | current (big) | next (small) horizontally; the "big" version is the same RenderMonth output (we don't enlarge it for v1 — the visual focus comes from being flanked by smaller-feeling neighbors via two-space gutters and from being the only month containing today). v2 could scale the focus month to wider cell widths.

- [ ] **Step 1: Write failing tests**

Append to `internal/render/layouts_test.go`:
```go
func TestLayout_Grid_12Months(t *testing.T) {
	o := opts()
	jan := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	months := calendar.BuildMonths(jan, 12, o.WeekStart)
	got := RenderLayout(LayoutGrid, months, time.Time{}, o)
	golden.Assert(t, "testdata/layout_grid_12.golden", got)

	// Sanity: should produce 4 rows of months (default cols=3 ⇒ ceil(12/3) = 4).
	// We can't directly assert row count without parsing, but we can assert the
	// height is at least 4 * (1 header + 1 weekday + 6 week rows) = 32 lines.
	lines := strings.Count(got, "\n") + 1
	if lines < 32 {
		t.Errorf("grid layout too short: %d lines", lines)
	}
}

func TestLayout_Focus_3Months(t *testing.T) {
	o := opts()
	months := calendar.BuildMonths(anchorApril2026().AddDate(0, -1, 0), 3, o.WeekStart)
	today := time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC)
	got := RenderLayout(LayoutFocus, months, today, o)
	golden.Assert(t, "testdata/layout_focus_3.golden", got)
	// Should contain a bracketed today, since the centre month is April 2026.
	if !strings.Contains(got, "[22]") {
		t.Error("focus layout should highlight today (22) in the centre month")
	}
}
```

- [ ] **Step 2: Run tests, confirm fail**

```bash
go test ./internal/render/...
```
Expected: tests for grid/focus fail — current implementation falls through to joinVertical.

- [ ] **Step 3: Implement grid + focus**

Replace the `switch` in `RenderLayout` and add helpers:
```go
func RenderLayout(layout Layout, months []calendar.Month, today time.Time, opts Options) string {
	blocks := make([]string, len(months))
	for i, m := range months {
		blocks[i] = RenderMonth(m, today, opts)
	}
	switch layout {
	case LayoutHorizontal:
		return joinHorizontal(blocks)
	case LayoutVertical:
		return joinVertical(blocks)
	case LayoutGrid:
		return joinGrid(blocks, 3)
	case LayoutFocus:
		return joinHorizontal(blocks)
	default:
		return joinVertical(blocks)
	}
}

// joinGrid arranges blocks in rows of cols, joined horizontally within each
// row and vertically across rows. Empty trailing cells in the last row are
// padded with whitespace blocks of the same width to keep alignment.
func joinGrid(blocks []string, cols int) string {
	if cols < 1 {
		cols = 1
	}
	rows := make([]string, 0)
	for i := 0; i < len(blocks); i += cols {
		end := i + cols
		if end > len(blocks) {
			end = len(blocks)
		}
		row := blocks[i:end]
		rows = append(rows, joinHorizontal(row))
	}
	withGutters := make([]string, 0, 2*len(rows)-1)
	for i, r := range rows {
		if i > 0 {
			withGutters = append(withGutters, "")
		}
		withGutters = append(withGutters, r)
	}
	return lipgloss.JoinVertical(lipgloss.Left, withGutters...)
}
```

- [ ] **Step 4: Run tests with `-update` to create the goldens**

```bash
go test ./internal/render/... -run Layout -update
```
Expected: PASS; goldens for grid_12 and focus_3 created.

- [ ] **Step 5: Visually inspect goldens**

Open `internal/render/testdata/layout_grid_12.golden`. Confirm:
- 4 rows of 3 months each: Jan/Feb/Mar, Apr/May/Jun, Jul/Aug/Sep, Oct/Nov/Dec.
- One blank line between rows; two-space gutters within rows.

Open `internal/render/testdata/layout_focus_3.golden`. Confirm:
- March, April (with `[22]`), May side-by-side.

- [ ] **Step 6: Run tests without `-update`, confirm pass**

```bash
go test ./internal/render/...
```
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/render/layouts.go internal/render/layouts_test.go internal/render/testdata/layout_grid_12.golden internal/render/testdata/layout_focus_3.golden
git commit -m "feat(render): grid and focus layouts"
```

---

## Task 9: `render.Frame` — top-level composition

**Files:**
- Create: `internal/render/frame.go`
- Create: `internal/render/frame_test.go`
- Create: `internal/render/testdata/frame_focus_april2026_today22.golden`

**Behavior:**
1. Pick month count (`State.Months` if > 0, else layout default: 3/3/12/3).
2. Compute the starting month: horizontal/vertical/focus = anchor − 1 month; grid = Jan of anchor's year.
3. Build months via `calendar.BuildMonths`.
4. Render layout block via `RenderLayout`.
5. Render clock via `RenderClock` (skipped in print mode when `!opts.WithClock`).
6. Stack clock above layout with one blank gutter line.
7. Center via `Fit(state.Width, state.Height)`.
8. If `Width` is below the minimum needed for the layout, prepend a one-line hint.

- [ ] **Step 1: Write failing test**

Write `internal/render/frame_test.go`:
```go
package render

import (
	"strings"
	"testing"
	"time"

	"github.com/v4run/tcal/internal/golden"
)

func TestFrame_Focus_April2026_Today22(t *testing.T) {
	state := State{
		Anchor: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Today:  time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
		Now:    time.Date(2026, 4, 22, 14, 32, 7, 0, time.UTC),
		Layout: LayoutFocus,
		Width:  100,
		Height: 30,
	}
	o := Options{
		WeekStart:  time.Sunday,
		Highlight:  HighlightBracket,
		ClockStyle: ClockBlock,
		Color:      false,
		WithClock:  true,
	}
	got := Frame(state, o)
	golden.Assert(t, "testdata/frame_focus_april2026_today22.golden", got)

	if !strings.Contains(got, "[22]") {
		t.Error("frame should include today highlight")
	}
}

func TestFrame_TooNarrow_PrependsHint(t *testing.T) {
	state := State{
		Anchor: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Today:  time.Time{},
		Now:    time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC),
		Layout: LayoutHorizontal,
		Width:  10, // way too narrow for 3 horizontal months
		Height: 0,
	}
	o := Options{WeekStart: time.Sunday, Highlight: HighlightNone}
	got := Frame(state, o)
	if !strings.HasPrefix(got, "terminal too narrow") {
		t.Errorf("expected hint prefix, got first line: %q", strings.SplitN(got, "\n", 2)[0])
	}
}

func TestFrame_PrintMode_NoClockByDefault(t *testing.T) {
	state := State{
		Anchor: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Now:    time.Date(2026, 4, 22, 14, 32, 7, 0, time.UTC),
		Layout: LayoutGrid,
		Width:  80,
		Height: 0, // print mode signal
	}
	o := Options{WeekStart: time.Sunday, WithClock: false}
	got := Frame(state, o)
	// The block clock contains '█'; its absence confirms WithClock=false dropped it.
	if strings.Contains(got, "█") {
		t.Error("WithClock=false should suppress the block clock")
	}
	// The textual print header still appears.
	if !strings.Contains(got, "2026") {
		t.Error("frame should still contain a date header")
	}
}
```

- [ ] **Step 2: Run tests, confirm fail**

```bash
go test ./internal/render/...
```
Expected: fails — `Frame` undefined.

- [ ] **Step 3: Implement Frame**

Write `internal/render/frame.go`:
```go
package render

import (
	"fmt"

	"github.com/v4run/tcal/internal/calendar"
)

func defaultMonths(layout Layout) int {
	switch layout {
	case LayoutGrid:
		return 12
	default:
		return 3
	}
}

func layoutStart(layout Layout, anchorYear int, anchorMonth int) (int, int) {
	if layout == LayoutGrid {
		return anchorYear, 1
	}
	// horizontal / vertical / focus: anchor - 1 month
	y, m := anchorYear, anchorMonth-1
	if m < 1 {
		m = 12
		y--
	}
	return y, m
}

func minWidthFor(layout Layout, months int) int {
	const grid = 20
	const gutter = 2
	switch layout {
	case LayoutHorizontal, LayoutFocus:
		return months*grid + (months-1)*gutter
	case LayoutGrid:
		cols := 3
		return cols*grid + (cols-1)*gutter
	default: // vertical
		return grid
	}
}

// Frame is the top-level renderer used by both tui and printout.
func Frame(state State, opts Options) string {
	months := state.Months
	if months <= 0 {
		months = defaultMonths(state.Layout)
	}

	y, m := layoutStart(state.Layout, state.Anchor.Year(), int(state.Anchor.Month()))
	startTime := state.Anchor.AddDate(y-state.Anchor.Year(), m-int(state.Anchor.Month()), 0)
	monthData := calendar.BuildMonths(startTime, months, opts.WeekStart)

	body := RenderLayout(state.Layout, monthData, state.Today, opts)

	var header string
	if opts.WithClock {
		clock := RenderClock(state.Now, opts.ClockStyle)
		header = clock
	} else {
		// Single-line text header used in print mode by default.
		header = state.Now.Format("Mon, 02 Jan 2006 15:04")
	}

	block := header + "\n\n" + body

	// Width hint
	min := minWidthFor(state.Layout, months)
	if state.Width > 0 && state.Width < min {
		hint := fmt.Sprintf("terminal too narrow — needs ≥ %d cols", min)
		return hint + "\n" + block
	}

	width := state.Width
	if width <= 0 {
		width = min
	}
	return Fit(block, width, state.Height)
}
```

- [ ] **Step 4: Run tests with `-update` for the focus golden**

```bash
go test ./internal/render/... -run Frame -update
```
Expected: PASS; one golden file created.

- [ ] **Step 5: Inspect the frame golden**

Open `internal/render/testdata/frame_focus_april2026_today22.golden`. Confirm:
- Block clock at the top showing `14:32:07`.
- Blank line gutter.
- Three months side-by-side; April contains `[22]`.
- Whole block centered horizontally within 100 cols and vertically within 30 rows.

- [ ] **Step 6: Run tests without `-update`, confirm pass**

```bash
go test ./internal/render/...
```
Expected: PASS for all three Frame tests.

- [ ] **Step 7: Commit**

```bash
git add internal/render/frame.go internal/render/frame_test.go internal/render/testdata/frame_focus_april2026_today22.golden
git commit -m "feat(render): Frame composes clock + layout + centering"
```

---

## Task 10: `internal/printout` — one-shot print mode

**Files:**
- Create: `internal/printout/print.go`
- Create: `internal/printout/print_test.go`

**Behavior:**
- `Run(state, opts, w io.Writer) error` invokes `render.Frame` once with `state.Height = 0` and writes the result + final newline. Swallows `EPIPE`.
- Width resolution happens in `cmd/tcal/main.go` (Task 12); `printout` just consumes whatever `state.Width` is set to.

- [ ] **Step 1: Write failing tests**

Write `internal/printout/print_test.go`:
```go
package printout

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/v4run/tcal/internal/render"
)

func TestRun_WritesGrid(t *testing.T) {
	state := render.State{
		Anchor: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Now:    time.Date(2026, 4, 22, 14, 32, 7, 0, time.UTC),
		Layout: render.LayoutGrid,
		Width:  80,
		Height: 0, // print mode
	}
	opts := render.Options{
		WeekStart: time.Sunday,
		Highlight: render.HighlightNone,
	}

	var buf bytes.Buffer
	if err := Run(state, opts, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.Contains(out, "January 2026") || !strings.Contains(out, "December 2026") {
		t.Error("grid output should contain January and December 2026")
	}
	if !strings.HasSuffix(out, "\n") {
		t.Error("output should end with newline")
	}
}
```

- [ ] **Step 2: Run tests, confirm fail**

```bash
go test ./internal/printout/...
```
Expected: build fails — `Run` undefined.

- [ ] **Step 3: Implement Run**

Write `internal/printout/print.go`:
```go
// Package printout renders one frame to an io.Writer and exits.
package printout

import (
	"errors"
	"io"
	"syscall"

	"github.com/v4run/tcal/internal/render"
)

// Run renders state once via render.Frame and writes the result + trailing
// newline to w. EPIPE (broken pipe) is treated as success so `tcal --print
// | head` is well-behaved.
func Run(state render.State, opts render.Options, w io.Writer) error {
	state.Height = 0 // print mode never vertically centers
	out := render.Frame(state, opts) + "\n"
	_, err := io.WriteString(w, out)
	if err != nil {
		if errors.Is(err, syscall.EPIPE) {
			return nil
		}
		return err
	}
	return nil
}
```

- [ ] **Step 4: Run tests, confirm pass**

```bash
go test ./internal/printout/...
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/printout/
git commit -m "feat(printout): one-shot Run with EPIPE-safe write"
```

---

## Task 11: `internal/tui` — Bubble Tea model + keybindings

**Files:**
- Create: `internal/tui/model.go`
- Create: `internal/tui/model_test.go`

**Behavior:**
- `tea.Tick(time.Second, ...)` self-rescheduling tick updates `state.Now`.
- `tea.WindowSizeMsg` updates `state.Width/Height`.
- Keys: h/l/j/k arrows, 1–4 layouts, t today, ? help (just toggles a bool field for v1; rendering of the overlay is deferred to a v2), q / Ctrl-C quit.

- [ ] **Step 1: Write failing tests**

Write `internal/tui/model_test.go`:
```go
package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/v4run/tcal/internal/render"
)

func newTestModel(anchor time.Time) Model {
	return Model{
		state: render.State{
			Anchor: anchor,
			Today:  anchor,
			Now:    anchor,
			Layout: render.LayoutFocus,
			Width:  100,
			Height: 30,
		},
		opts: render.Options{
			WeekStart: time.Sunday,
			Highlight: render.HighlightBracket,
		},
	}
}

func key(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestUpdate_LMovesAnchorForwardOneMonth(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	next, _ := m.Update(key('l'))
	got := next.(Model).state.Anchor
	want := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUpdate_HMovesAnchorBackOneMonth(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	next, _ := m.Update(key('h'))
	got := next.(Model).state.Anchor
	want := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUpdate_KAddsOneYear(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	next, _ := m.Update(key('k'))
	got := next.(Model).state.Anchor.Year()
	if got != 2027 {
		t.Errorf("got %d, want 2027", got)
	}
}

func TestUpdate_JSubtractsOneYear(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	next, _ := m.Update(key('j'))
	got := next.(Model).state.Anchor.Year()
	if got != 2025 {
		t.Errorf("got %d, want 2025", got)
	}
}

func TestUpdate_TJumpsAnchorToToday(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	m.state.Today = time.Date(2027, 11, 15, 0, 0, 0, 0, time.UTC)
	next, _ := m.Update(key('t'))
	got := next.(Model).state.Anchor
	want := time.Date(2027, 11, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUpdate_LayoutKeysSwitchLayout(t *testing.T) {
	cases := []struct {
		key  rune
		want render.Layout
	}{
		{'1', render.LayoutHorizontal},
		{'2', render.LayoutVertical},
		{'3', render.LayoutGrid},
		{'4', render.LayoutFocus},
	}
	for _, tc := range cases {
		t.Run(string(tc.key), func(t *testing.T) {
			m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
			next, _ := m.Update(key(tc.key))
			if got := next.(Model).state.Layout; got != tc.want {
				t.Errorf("key %c: got %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestUpdate_QReturnsQuitCommand(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	_, cmd := m.Update(key('q'))
	if cmd == nil {
		t.Fatal("expected a tea.Cmd from q, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestUpdate_WindowSizeMsgUpdatesDimensions(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	next, _ := m.Update(tea.WindowSizeMsg{Width: 150, Height: 50})
	st := next.(Model).state
	if st.Width != 150 || st.Height != 50 {
		t.Errorf("got %dx%d, want 150x50", st.Width, st.Height)
	}
}

func TestUpdate_TickAdvancesNow(t *testing.T) {
	m := newTestModel(time.Date(2026, 4, 22, 14, 32, 7, 0, time.UTC))
	later := time.Date(2026, 4, 22, 14, 32, 8, 0, time.UTC)
	next, _ := m.Update(tickMsg(later))
	got := next.(Model).state.Now
	if !got.Equal(later) {
		t.Errorf("got %v, want %v", got, later)
	}
}
```

- [ ] **Step 2: Run tests, confirm fail**

```bash
go test ./internal/tui/...
```
Expected: build fails — `Model`, `tickMsg`, etc. undefined.

- [ ] **Step 3: Implement Model**

Write `internal/tui/model.go`:
```go
// Package tui hosts the Bubble Tea live-mode wrapper around render.Frame.
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/v4run/tcal/internal/render"
)

// Model is the Bubble Tea model for the live calendar widget.
type Model struct {
	state render.State
	opts  render.Options
}

// New returns a Model initialized with the given state and options. Caller
// owns providing Today/Now/Anchor; the model only mutates them in response
// to user input or tick messages.
func New(state render.State, opts render.Options) Model {
	return Model{state: state, opts: opts}
}

// tickMsg is sent every second to advance the clock.
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Init starts the periodic tick.
func (m Model) Init() tea.Cmd {
	return tick()
}

// Update is the message handler.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.state.Width = msg.Width
		m.state.Height = msg.Height
		return m, nil
	case tickMsg:
		m.state.Now = time.Time(msg)
		return m, tick()
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "h", "left":
		m.state.Anchor = m.state.Anchor.AddDate(0, -1, 0)
	case "l", "right":
		m.state.Anchor = m.state.Anchor.AddDate(0, 1, 0)
	case "j", "down":
		m.state.Anchor = m.state.Anchor.AddDate(-1, 0, 0)
	case "k", "up":
		m.state.Anchor = m.state.Anchor.AddDate(1, 0, 0)
	case "t":
		t := m.state.Today
		m.state.Anchor = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case "1":
		m.state.Layout = render.LayoutHorizontal
	case "2":
		m.state.Layout = render.LayoutVertical
	case "3":
		m.state.Layout = render.LayoutGrid
	case "4":
		m.state.Layout = render.LayoutFocus
	}
	return m, nil
}

// View renders the current frame.
func (m Model) View() string {
	return render.Frame(m.state, m.opts)
}
```

- [ ] **Step 4: Run tests, confirm pass**

```bash
go test ./internal/tui/...
```
Expected: PASS for all nine tests.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/
git commit -m "feat(tui): Bubble Tea model with keybindings and 1s tick"
```

---

## Task 12: `cmd/tcal/main.go` — flag parsing & dispatch

**Files:**
- Modify: `cmd/tcal/main.go` (replace placeholder)

**Flag surface (matching spec table):**

```
--layout         focus|horizontal|vertical|grid  (default focus)
--date           YYYY-MM or YYYY-MM-DD
--year           YYYY (shorthand for --date=YYYY-01)
--months         N
--week-start     sun|mon                          (default sun)
--highlight      combined|reverse|bracket|color|none  (default combined)
--no-color
--print
--width          N
--with-clock                                       (print mode only)
--clock-style    block|inline|boxed                (default block)
--help
--version
```

- [ ] **Step 1: Replace placeholder main**

Write `cmd/tcal/main.go`:
```go
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/v4run/tcal/internal/printout"
	"github.com/v4run/tcal/internal/render"
	"github.com/v4run/tcal/internal/tui"
)

const version = "0.1.0"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, errVersion) || errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "tcal:", err)
		os.Exit(2)
	}
}

var errVersion = errors.New("--version")

func run(args []string, stdout, stderr *os.File) error {
	fs := flag.NewFlagSet("tcal", flag.ContinueOnError)
	fs.SetOutput(stderr)

	layout := fs.String("layout", "focus", "horizontal|vertical|grid|focus")
	date := fs.String("date", "", "YYYY-MM or YYYY-MM-DD (default today)")
	year := fs.Int("year", 0, "shorthand for --date=YYYY-01")
	months := fs.Int("months", 0, "number of months (0 = layout default)")
	weekStart := fs.String("week-start", "sun", "sun|mon")
	highlight := fs.String("highlight", "combined", "combined|reverse|bracket|color|none")
	noColor := fs.Bool("no-color", false, "disable ANSI color (also honors NO_COLOR env)")
	print := fs.Bool("print", false, "one-shot render to stdout, no TUI")
	width := fs.Int("width", 0, "terminal width (print mode only)")
	withClock := fs.Bool("with-clock", false, "include block clock in print mode")
	clockStyle := fs.String("clock-style", "block", "block|inline|boxed")
	showVersion := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Fprintln(stdout, "tcal", version)
		return errVersion
	}

	state, opts, err := buildState(*layout, *date, *year, *months, *weekStart, *highlight, *clockStyle, *noColor, *withClock)
	if err != nil {
		return err
	}

	if *print {
		state.Height = 0
		state.Width = resolveWidth(*width, stdout)
		return printout.Run(state, opts, stdout)
	}

	state.Width, state.Height = currentTermSize(stdout)
	model := tui.New(state, opts)
	prog := tea.NewProgram(model, tea.WithAltScreen())
	_, err = prog.Run()
	return err
}

func buildState(layoutS, dateS string, year, months int, weekStartS, highlightS, clockStyleS string, noColor, withClock bool) (render.State, render.Options, error) {
	now := time.Now()
	anchor := now

	switch {
	case dateS != "":
		t, err := parseDate(dateS)
		if err != nil {
			return render.State{}, render.Options{}, err
		}
		anchor = t
	case year != 0:
		anchor = time.Date(year, 1, 1, 0, 0, 0, 0, now.Location())
	}

	layout, err := parseLayout(layoutS)
	if err != nil {
		return render.State{}, render.Options{}, err
	}
	weekStart, err := parseWeekStart(weekStartS)
	if err != nil {
		return render.State{}, render.Options{}, err
	}
	highlight, err := parseHighlight(highlightS)
	if err != nil {
		return render.State{}, render.Options{}, err
	}
	clockStyle, err := parseClockStyle(clockStyleS)
	if err != nil {
		return render.State{}, render.Options{}, err
	}

	color := !noColor && os.Getenv("NO_COLOR") == ""

	return render.State{
			Anchor: anchor,
			Today:  now,
			Now:    now,
			Layout: layout,
			Months: months,
		},
		render.Options{
			WeekStart:  weekStart,
			Color:      color,
			Highlight:  highlight,
			ClockStyle: clockStyle,
			WithClock:  withClock,
		}, nil
}

func parseDate(s string) (time.Time, error) {
	formats := []string{"2006-01-02", "2006-01"}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf(`invalid --date %q: expected YYYY-MM or YYYY-MM-DD`, s)
}

func parseLayout(s string) (render.Layout, error) {
	switch strings.ToLower(s) {
	case "horizontal", "h":
		return render.LayoutHorizontal, nil
	case "vertical", "v":
		return render.LayoutVertical, nil
	case "grid", "g":
		return render.LayoutGrid, nil
	case "focus", "f":
		return render.LayoutFocus, nil
	}
	return 0, fmt.Errorf("unknown --layout %q (expected horizontal|vertical|grid|focus)", s)
}

func parseWeekStart(s string) (time.Weekday, error) {
	switch strings.ToLower(s) {
	case "sun", "sunday":
		return time.Sunday, nil
	case "mon", "monday":
		return time.Monday, nil
	}
	return 0, fmt.Errorf("unknown --week-start %q (expected sun|mon)", s)
}

func parseHighlight(s string) (render.HighlightMask, error) {
	switch strings.ToLower(s) {
	case "combined":
		return render.HighlightAll, nil
	case "reverse":
		return render.HighlightReverse, nil
	case "bracket":
		return render.HighlightBracket, nil
	case "color":
		return render.HighlightColor, nil
	case "none":
		return render.HighlightNone, nil
	}
	return 0, fmt.Errorf("unknown --highlight %q (expected combined|reverse|bracket|color|none)", s)
}

func parseClockStyle(s string) (render.ClockStyle, error) {
	switch strings.ToLower(s) {
	case "block":
		return render.ClockBlock, nil
	case "inline":
		return render.ClockInline, nil
	case "boxed":
		return render.ClockBoxed, nil
	}
	return 0, fmt.Errorf("unknown --clock-style %q (expected block|inline|boxed)", s)
}

func resolveWidth(flagW int, f *os.File) int {
	if flagW > 0 {
		return flagW
	}
	if w, _, err := term.GetSize(int(f.Fd())); err == nil && w > 0 {
		return w
	}
	return 80
}

func currentTermSize(f *os.File) (int, int) {
	if w, h, err := term.GetSize(int(f.Fd())); err == nil && w > 0 && h > 0 {
		return w, h
	}
	return 80, 24
}
```

- [ ] **Step 2: Build and run the binary**

```bash
go build -o tcal ./cmd/tcal
./tcal --version
```
Expected output: `tcal 0.1.0`.

- [ ] **Step 3: Smoke-test the print mode**

```bash
./tcal --print --layout grid --date 2026-04 --width 80 | head -40
```
Expected: a header line with date/time, blank line, then a 4×3 grid of 2026 months. No ANSI codes.

- [ ] **Step 4: Smoke-test the live mode (manual)**

```bash
./tcal --layout focus
```
Expected: block-digit clock at the top, three months below, everything centered. `q` exits cleanly. Try `1`, `2`, `3`, `4` to switch layouts. Try `h`/`l` to navigate months.

- [ ] **Step 5: Smoke-test error handling**

```bash
./tcal --date 2026-13
```
Expected: stderr line `tcal: invalid --date "2026-13": expected YYYY-MM or YYYY-MM-DD`, exit code 2.

```bash
./tcal --layout xyz
```
Expected: stderr line `tcal: unknown --layout "xyz" ...`, exit code 2.

```bash
NO_COLOR=1 ./tcal --print --layout focus | head -20
```
Expected: no ANSI escapes in output (verify with `cat -v`).

- [ ] **Step 6: Commit**

```bash
git add cmd/tcal/main.go
git commit -m "feat(cmd): wire flag parsing, mode dispatch, terminal size detection"
```

---

## Task 13: README + manual smoke checklist

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Replace placeholder README**

Write `README.md`:
```markdown
# tcal

A live terminal calendar widget with a block-digit clock and four multi-month layouts. Also runs as a one-shot static renderer for piping or printing.

## Install

```
go install github.com/v4run/tcal/cmd/tcal@latest
```

Or from source:

```
git clone <this-repo>
cd tcal
go build -o tcal ./cmd/tcal
```

## Quick start

```
tcal                          # live focus layout, block clock, centered in terminal
tcal --layout grid            # full-year wallchart
tcal --layout horizontal      # 3 months in a row
tcal --print --layout grid --date 2026-04 > year.txt
tcal --print --layout focus | lpr
```

## Keys (live mode)

| Key             | Action                          |
|-----------------|---------------------------------|
| h / ←           | Previous month                  |
| l / →           | Next month                      |
| j / ↓           | Previous year                   |
| k / ↑           | Next year                       |
| 1 / 2 / 3 / 4   | Layout: horizontal / vertical / grid / focus |
| t               | Jump to today                   |
| q / Ctrl-C      | Quit                            |

## Flags

| Flag             | Default        | Notes                                      |
|------------------|----------------|--------------------------------------------|
| `--layout`       | `focus`        | `horizontal \| vertical \| grid \| focus`  |
| `--date`         | today          | `YYYY-MM` or `YYYY-MM-DD`                  |
| `--year`         | (anchor year)  | shorthand for `--date=YYYY-01`             |
| `--months`       | layout default | per-layout: 3 / 3 / 12 / 3                 |
| `--week-start`   | `sun`          | `sun \| mon`                               |
| `--highlight`    | `combined`     | `combined \| reverse \| bracket \| color \| none` |
| `--no-color`     | off            | also honors `NO_COLOR`                     |
| `--print`        | off            | one-shot render to stdout                  |
| `--width`        | TTY width / 80 | print mode only                            |
| `--with-clock`   | off            | include block clock in print mode          |
| `--clock-style`  | `block`        | `block \| inline \| boxed`                 |

## Manual smoke checklist

After any change to layout/centering code, run through:

- [ ] `tcal` — focus layout, clock ticks, terminal resize re-centers.
- [ ] `tcal --layout grid` — 12-month wallchart fits in a standard terminal.
- [ ] `tcal --print --layout grid --date 2026-01 > /tmp/y.txt && cat /tmp/y.txt` — clean static output.
- [ ] `NO_COLOR=1 tcal` — combined highlight degrades to bracket-only; no ANSI escapes.
- [ ] Run in a 40×10 terminal — "too narrow" hint appears at the top.
- [ ] `tcal --date 2026-13` — exits non-zero with a one-line error.

## License

MIT
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: README with usage, keys, flags, and smoke checklist"
```

---

## Task 14: Final verification — full test run + smoke pass

This task does not write code; it gates the implementation as "done".

- [ ] **Step 1: Run the entire test suite**

```bash
go test ./...
```
Expected: PASS across `calendar`, `golden`, `render`, `printout`, `tui` packages.

- [ ] **Step 2: Run with race detector**

```bash
go test ./... -race
```
Expected: PASS, no data races.

- [ ] **Step 3: go vet**

```bash
go vet ./...
```
Expected: no warnings.

- [ ] **Step 4: Walk the README smoke checklist**

Execute each item under "Manual smoke checklist" in the README. Note any visual issues. If anything looks off, file a follow-up task — do not silently fix in this loop.

- [ ] **Step 5: Tag v0.1.0**

```bash
git tag v0.1.0
```

(Do not push tags or branches without explicit user confirmation.)

---

## Self-review notes

Checked against the spec at `docs/superpowers/specs/2026-05-27-tcal-design.md`:

- ✅ Four layouts (horizontal, vertical, grid, focus): Tasks 7–8.
- ✅ Block-digit clock, plus inline + boxed variants: Task 6.
- ✅ Today highlight combined (reverse + bracket + color) and individual modes: Task 4.
- ✅ Week start sun/mon: Tasks 1, 11–12.
- ✅ H+V centering with too-narrow / too-tall graceful degradation: Tasks 3, 9.
- ✅ Live mode tick at 1s, keybindings, navigation: Task 11.
- ✅ Print mode one-shot, EPIPE-safe, width fallback: Tasks 10, 12.
- ✅ NO_COLOR env + `--no-color` flag: Task 12 (renderer code already respects opts.Color via lipgloss).
- ✅ Golden-file testing with `-update`: Task 2.
- ✅ Error handling: parse → exit 2 with stderr line (Task 12); EPIPE swallow (Task 10); too-narrow hint (Task 9).
- ✅ README + smoke checklist: Task 13.

The two spec-acknowledged open questions (vim j/k semantics; clock-style defaults verification) are deferred to manual smoke (Task 14, Step 4) — they're not gating for v0.1.0.

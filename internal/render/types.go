package render

import "time"

// Layout selects the multi-month arrangement.
type Layout int

const (
	LayoutHorizontal Layout = iota
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

package render

import "time"

// Layout selects the multi-month arrangement.
type Layout int

const (
	LayoutHorizontal Layout = iota
	LayoutGrid
)

// ClockStyle selects the clock rendering style.
type ClockStyle int

const (
	ClockBlock ClockStyle = iota
	ClockInline
	ClockBoxed
)

// Look selects how the clock and calendar are drawn.
type Look int

const (
	// LookClassic is the original look: block clock, one-line day numbers.
	LookClassic Look = iota
	// LookThin draws the clock and day numbers 3 rows tall in thin lines.
	// Frame falls back to classic when the thin block does not fit.
	LookThin
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
	Look       Look
	Neighbors  Neighbors
}

// Neighbors selects how the thin look shows the months either side of the
// current one in the horizontal layout.
type Neighbors int

const (
	// NeighborsAuto shows them big when they fit, else small, else not at
	// all.
	NeighborsAuto Neighbors = iota
	// NeighborsSmall always shows them small beside the big current month,
	// or not at all when that doesn't fit.
	NeighborsSmall
	// NeighborsBig shows them big when they fit, else not at all.
	NeighborsBig
)

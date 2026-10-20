// Package scene defines the interface shared by live-mode ambient
// backgrounds. The TUI composites a scene into the blank margins around the
// calendar; anything a scene draws inside the calendar's Box is discarded.
package scene

import "time"

// Box is the calendar's bounding box plus one cell of padding, in screen
// cells. Columns are half-open [X0, X1); rows are inclusive [Y0, Y1].
// An empty frame has Y0 = Y1 = -1.
type Box struct{ X0, X1, Y0, Y1 int }

// Contains reports whether screen cell (x, y) lies inside the box.
func (b Box) Contains(x, y int) bool {
	return x >= b.X0 && x < b.X1 && y >= b.Y0 && y <= b.Y1
}

// Scene is an ambient animation. All methods are called from the Bubble Tea
// event loop, never concurrently.
type Scene interface {
	// Resize is called with the terminal size, before the first Tick and on
	// every terminal resize.
	Resize(w, h int)
	// Tick advances the animation by one step.
	Tick()
	// Interval is the delay between Ticks.
	Interval() time.Duration
	// Glyph returns the rune to draw at screen (x, y), or 0 for empty.
	// Glyphs are drawn dim; cells inside b are never drawn, so scenes only
	// need b when they want to react to the calendar (e.g. hide or orbit it).
	Glyph(x, y int, b Box) rune
}

// Clocked is implemented by scenes that depend on the time of day or the
// month being displayed. SetTime is called before the first frame and after
// every update.
type Clocked interface {
	SetTime(now, shown time.Time)
}

// Color is a terminal palette color. Scenes use the 16-color ANSI palette
// so backgrounds follow the user's terminal theme.
type Color uint8

const (
	Default Color = iota // the terminal's foreground color
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

// Colored is implemented by scenes that tint their glyphs. Color is only
// called for cells where Glyph returned a rune, with the same arguments.
type Colored interface {
	Color(x, y int, b Box) Color
}

// RGB is a 24-bit true colour.
type RGB struct{ R, G, B uint8 }

// TrueColored is implemented by scenes that draw in 24-bit colour. RGB is
// only called for cells where Glyph returned a rune.
type TrueColored interface {
	RGB(x, y int, b Box) RGB
}

// Stepper is implemented by scenes that advance by elapsed time. When a scene
// implements it, the TUI calls Step instead of Tick.
type Stepper interface {
	Step(dt time.Duration)
}

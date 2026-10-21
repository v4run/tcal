// Package tui hosts the Bubble Tea live-mode wrapper around render.Frame.
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/v4run/tcal/internal/palette"
	"github.com/v4run/tcal/internal/render"
	"github.com/v4run/tcal/internal/scene"
)

// Model is the Bubble Tea model for the live calendar widget.
type Model struct {
	state render.State
	opts  render.Options
	bg    scene.Scene // nil unless an ambient background is enabled
	// bgMono draws the background in the plain dim foreground color,
	// ignoring scene colors.
	bgMono bool
	// bgOffset shifts the clock that time-aware backgrounds see.
	bgOffset time.Duration
	// bgMode and bgRGB select how true-colour scenes are drawn: the colour
	// depth and the terminal background they are blended toward.
	bgMode palette.Mode
	bgRGB  scene.RGB
	// bgBehind is the background's opacity behind the calendar, 0 to 100.
	bgBehind int
	// bgLast is the time of the previous background tick; zero before the
	// first.
	bgLast time.Time
	// frame caches render.Frame between state changes; nil disables it.
	frame *frameCache
}

// maxStep caps the time a Stepper scene advances in one tick, so a
// suspended terminal does not make scenes jump.
const maxStep = 100 * time.Millisecond

// renderFrame is render.Frame; tests replace it to count calls.
var renderFrame = render.Frame

// frameCache holds the last rendered calendar frame and the inputs it was
// rendered from. Models copied from one another share it.
type frameCache struct {
	key frameKey
	out string
	ok  bool
}

// frameKey is everything render.Frame reads. Times are reduced to what the
// frame can show: Now to the second, all three with their location.
type frameKey struct {
	state               render.State
	opts                render.Options
	anchor, today, now  int64
	anchorLoc, todayLoc *time.Location
	nowLoc              *time.Location
}

func keyOf(s render.State, o render.Options) frameKey {
	k := frameKey{
		opts:      o,
		anchor:    s.Anchor.UnixNano(),
		today:     s.Today.UnixNano(),
		now:       s.Now.Unix(),
		anchorLoc: s.Anchor.Location(),
		todayLoc:  s.Today.Location(),
		nowLoc:    s.Now.Location(),
	}
	s.Anchor, s.Today, s.Now = time.Time{}, time.Time{}, time.Time{}
	k.state = s
	return k
}

// New returns a Model initialized with the given state and options. Caller
// owns providing Today/Now/Anchor; the model only mutates them in response
// to user input or tick messages.
func New(state render.State, opts render.Options) Model {
	return Model{state: state, opts: opts, frame: &frameCache{}}
}

// WithBackground enables the named ambient background (see Backgrounds),
// sized to the current state and configured by o. Unknown names leave the
// background off. Call WithBackgroundTime first: Random consults the
// shifted time to prefer seasons on event days.
func (m Model) WithBackground(name string, o BackgroundOptions) Model {
	m.bg = newScene(name, m.state.Width, m.state.Height, o, m.state.Now.Add(m.bgOffset))
	m.syncClock()
	return m
}

// WithBackgroundPalette sets how true-colour backgrounds are drawn: mode is
// the colour depth and bg the terminal background colours are blended toward.
// The default is ANSI on palette.Dark.
func (m Model) WithBackgroundPalette(mode palette.Mode, bg scene.RGB) Model {
	m.bgMode, m.bgRGB = mode, bg
	return m
}

// WithBackgroundBehind sets the background's opacity behind the calendar,
// clamped to 0 (off) to 100 (as bright as in the margins). The [ and ]
// keys change it in steps of behindStep.
func (m Model) WithBackgroundBehind(pct int) Model {
	m.bgBehind = min(max(pct, 0), 100)
	return m
}

// behindStep is how much [ and ] change the opacity behind the calendar.
const behindStep = 10

// WithBackgroundMono draws the background without scene colors: dim, in
// the terminal's foreground color.
func (m Model) WithBackgroundMono() Model {
	m.bgMono = true
	return m
}

// WithBackgroundTime makes time-aware backgrounds behave as if the time
// were t. The offset from the real clock is kept, so their time still
// advances. The calendar itself keeps the real time. Call it before
// WithBackground, so Random's choice sees the shifted time.
func (m Model) WithBackgroundTime(t time.Time) Model {
	m.bgOffset = t.Sub(m.state.Now)
	m.syncClock()
	return m
}

// syncClock tells a time-aware background the current time and the month
// being displayed.
func (m Model) syncClock() {
	if c, ok := m.bg.(scene.Clocked); ok {
		c.SetTime(m.state.Now.Add(m.bgOffset), m.state.Anchor)
	}
}

// tickMsg is sent every second to advance the clock.
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// bgTickMsg advances the ambient background by one step. It carries the time
// it fired, from which Stepper scenes get their elapsed time.
type bgTickMsg time.Time

func bgTick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return bgTickMsg(t) })
}

// Init starts the periodic ticks.
func (m Model) Init() tea.Cmd {
	if m.bg != nil {
		return tea.Batch(tick(), bgTick(m.bg.Interval()))
	}
	return tick()
}

// Update is the message handler.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	next.syncClock()
	return next, cmd
}

func (m Model) update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.state.Width = msg.Width
		m.state.Height = msg.Height
		if m.bg != nil {
			m.bg.Resize(msg.Width, msg.Height)
		}
		return m, nil
	case tickMsg:
		m.state.Now = time.Time(msg)
		return m, tick()
	case bgTickMsg:
		if m.bg == nil {
			return m, nil
		}
		now := time.Time(msg)
		if s, ok := m.bg.(scene.Stepper); ok {
			dt := m.bg.Interval()
			if !m.bgLast.IsZero() {
				dt = min(max(now.Sub(m.bgLast), 0), maxStep)
			}
			s.Step(dt)
		} else {
			m.bg.Tick()
		}
		m.bgLast = now
		return m, bgTick(m.bg.Interval())
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
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
		m.state.Layout = render.LayoutGrid
	case "[":
		m = m.WithBackgroundBehind(m.bgBehind - behindStep)
	case "]":
		m = m.WithBackgroundBehind(m.bgBehind + behindStep)
	}
	return m, nil
}

// View renders the current frame. The calendar frame is reused until the
// state it depends on changes; the background is redrawn every time.
func (m Model) View() string {
	out := m.calendarFrame()
	if m.bg != nil {
		out = overlay(out, m.bg, m.state.Width, m.state.Height, bgPaint{style: m.bgStyle(), mode: m.bgMode, bg: m.bgRGB, behind: m.bgBehind})
	}
	return out
}

func (m Model) calendarFrame() string {
	if m.frame == nil {
		return renderFrame(m.state, m.opts)
	}
	k := keyOf(m.state, m.opts)
	if !m.frame.ok || m.frame.key != k {
		m.frame.key, m.frame.out, m.frame.ok = k, renderFrame(m.state, m.opts), true
	}
	return m.frame.out
}

func (m Model) bgStyle() bgStyle {
	switch {
	case !m.opts.Color:
		return stylePlain
	case m.bgMono:
		return styleFaint
	}
	return styleColor
}

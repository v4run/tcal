// Package tui hosts the Bubble Tea live-mode wrapper around render.Frame.
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/v4run/tcal/internal/render"
	"github.com/v4run/tcal/internal/scene"
)

// Model is the Bubble Tea model for the live calendar widget.
type Model struct {
	state render.State
	opts  render.Options
	bg    scene.Scene // nil unless an ambient background is enabled
}

// New returns a Model initialized with the given state and options. Caller
// owns providing Today/Now/Anchor; the model only mutates them in response
// to user input or tick messages.
func New(state render.State, opts render.Options) Model {
	return Model{state: state, opts: opts}
}

// WithBackground enables the named ambient background (see Backgrounds),
// sized to the current state. Unknown names leave the background off.
func (m Model) WithBackground(name string) Model {
	m.bg = newScene(name, m.state.Width, m.state.Height)
	m.syncClock()
	return m
}

// syncClock tells a time-aware background the current time and the month
// being displayed.
func (m Model) syncClock() {
	if c, ok := m.bg.(scene.Clocked); ok {
		c.SetTime(m.state.Now, m.state.Anchor)
	}
}

// tickMsg is sent every second to advance the clock.
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// bgTickMsg advances the ambient background by one step.
type bgTickMsg struct{}

func bgTick(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return bgTickMsg{} })
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
		m.bg.Tick()
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
	}
	return m, nil
}

// View renders the current frame.
func (m Model) View() string {
	out := render.Frame(m.state, m.opts)
	if m.bg != nil {
		out = overlay(out, m.bg, m.state.Width, m.state.Height, m.opts.Color)
	}
	return out
}

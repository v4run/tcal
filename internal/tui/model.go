// Package tui hosts the Bubble Tea live-mode wrapper around render.Frame.
package tui

import (
	"math/rand/v2"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/v4run/tcal/internal/life"
	"github.com/v4run/tcal/internal/render"
)

// Model is the Bubble Tea model for the live calendar widget.
type Model struct {
	state render.State
	opts  render.Options
	life  *life.Grid // nil unless the Game of Life background is enabled
}

// New returns a Model initialized with the given state and options. Caller
// owns providing Today/Now/Anchor; the model only mutates them in response
// to user input or tick messages.
func New(state render.State, opts render.Options) Model {
	return Model{state: state, opts: opts}
}

// WithLife enables the Game of Life background, sized to the current state.
func (m Model) WithLife() Model {
	m.life = newLife(m.state.Width, m.state.Height)
	return m
}

func newLife(w, h int) *life.Grid {
	seed := uint64(time.Now().UnixNano())
	g := life.New(w, h, rand.New(rand.NewPCG(seed, seed>>32)))
	g.Seed()
	return g
}

// tickMsg is sent every second to advance the clock.
type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// lifeTickMsg advances the Game of Life background by one generation.
type lifeTickMsg struct{}

const lifeInterval = 250 * time.Millisecond

func lifeTick() tea.Cmd {
	return tea.Tick(lifeInterval, func(time.Time) tea.Msg { return lifeTickMsg{} })
}

// Init starts the periodic ticks.
func (m Model) Init() tea.Cmd {
	if m.life != nil {
		return tea.Batch(tick(), lifeTick())
	}
	return tick()
}

// Update is the message handler.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.state.Width = msg.Width
		m.state.Height = msg.Height
		if m.life != nil {
			m.life = newLife(msg.Width, msg.Height)
		}
		return m, nil
	case tickMsg:
		m.state.Now = time.Time(msg)
		return m, tick()
	case lifeTickMsg:
		if m.life == nil {
			return m, nil
		}
		m.life.Tick()
		return m, lifeTick()
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
		m.state.Layout = render.LayoutGrid
	}
	return m, nil
}

// View renders the current frame.
func (m Model) View() string {
	out := render.Frame(m.state, m.opts)
	if m.life != nil {
		out = overlayLife(out, m.life, m.opts.Color)
	}
	return out
}

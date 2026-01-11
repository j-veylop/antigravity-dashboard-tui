// Package info provides the info tab for the Antigravity Dashboard TUI.
package info

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/app"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/config"
)

// keyMap defines the key bindings specific to the info tab.
type keyMap struct {
	Refresh key.Binding
	Copy    key.Binding
	Up      key.Binding
	Down    key.Binding
}

// defaultKeyMap returns the default key bindings for the info tab.
func defaultKeyMap() keyMap {
	return keyMap{
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Copy: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy path"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
	}
}

// Model represents the info tab state.
type Model struct {
	state    *app.State
	config   *config.Config
	width    int
	height   int
	keys     keyMap
	viewport viewport.Model
}

// New creates a new info model.
func New(state *app.State, cfg *config.Config) *Model {
	return &Model{
		state:    state,
		config:   cfg,
		keys:     defaultKeyMap(),
		viewport: viewport.New(0, 0),
	}
}

// Init initializes the info tab.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the info tab.
func (m *Model) Update(msg tea.Msg) (app.Tab, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(keyMsg, m.keys.Copy):
			// Copy config path
			if m.config != nil {
				return m, func() tea.Msg {
					return app.CopyToClipboardMsg{Text: m.config.AccountsPath}
				}
			}
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(keyMsg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// SetSize sets the available size for the info tab.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
}

// ShortHelp returns the key bindings for the short help view.
func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keys.Copy,
	}
}

// FullHelp returns the key bindings for the full help view.
func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.keys.Copy},
		{m.keys.Refresh},
	}
}

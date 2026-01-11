// Package history provides the history tab for viewing historical statistics.
package history

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/app"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/services"
)

// keyMap defines the key bindings specific to the history tab.
type keyMap struct {
	ToggleRange key.Binding
	Refresh     key.Binding
	Up          key.Binding
	Down        key.Binding
}

// defaultKeyMap returns the default key bindings for the history tab.
func defaultKeyMap() keyMap {
	return keyMap{
		ToggleRange: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "toggle time range"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "scroll up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "scroll down"),
		),
	}
}

// historyLoadedMsg is sent when history data is loaded.
type historyLoadedMsg struct {
	stats *models.AccountHistoryStats
}

// historyErrorMsg is sent when there's an error loading history.
type historyErrorMsg struct {
	err string
}

// Model represents the history tab state.
type Model struct {
	state    *app.State
	services *services.Manager
	width    int
	height   int
	keys     keyMap
	viewport viewport.Model

	// Current view state
	timeRange   models.TimeRange
	historyData *models.AccountHistoryStats
	loading     bool
	lastRefresh time.Time
	errorMsg    string
}

// New creates a new history model.
func New(state *app.State, svc *services.Manager) *Model {
	return &Model{
		state:     state,
		services:  svc,
		keys:      defaultKeyMap(),
		viewport:  viewport.New(0, 0),
		timeRange: models.TimeRange30Days,
	}
}

// Init initializes the history tab.
func (m *Model) Init() tea.Cmd {
	return m.loadHistoryCmd()
}

// loadHistoryCmd creates a command to load history data.
func (m *Model) loadHistoryCmd() tea.Cmd {
	return func() tea.Msg {
		if m.services == nil {
			return historyErrorMsg{err: "Services not initialized"}
		}

		// Get selected account from shared state (synced with Dashboard)
		accounts := m.state.GetAccounts()
		if len(accounts) == 0 {
			return historyErrorMsg{err: "No accounts configured"}
		}

		// Use selected account index from state, or active account, or first
		selectedIdx := m.state.GetSelectedAccountIndex()
		var email string

		if selectedIdx >= 0 && selectedIdx < len(accounts) {
			email = accounts[selectedIdx].Email
		} else {
			// Fallback to active account
			for _, acc := range accounts {
				if acc.IsActive {
					email = acc.Email
					break
				}
			}
			if email == "" && len(accounts) > 0 {
				email = accounts[0].Email
			}
		}

		if email == "" {
			return historyErrorMsg{err: "No account selected"}
		}

		stats, err := m.services.GetAccountHistory(email, m.timeRange)
		if err != nil {
			return historyErrorMsg{err: err.Error()}
		}
		return historyLoadedMsg{stats: stats}
	}
}

// Update handles messages for the history tab.
func (m *Model) Update(msg tea.Msg) (app.Tab, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case historyLoadedMsg:
		m.historyData = msg.stats
		m.loading = false
		m.lastRefresh = time.Now()
		m.errorMsg = ""

	case historyErrorMsg:
		m.loading = false
		m.errorMsg = msg.err
		cmds = append(cmds, func() tea.Msg {
			return app.AddNotificationMsg{
				Type:     app.NotificationError,
				Message:  fmt.Sprintf("History error: %s", msg.err),
				Duration: app.LongNotificationDuration,
			}
		})

	case app.AccountsLoadedMsg:
		return m.handleAccountsLoaded()

	case app.TabSwitchMsg:
		if msg.Tab == app.TabHistory {
			return m.handleAccountsLoaded()
		}

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case app.SelectedAccountChangedMsg:
		// Selected account changed from Dashboard - reload history
		if !m.loading {
			m.loading = true
			cmds = append(cmds, m.loadHistoryCmd())
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleAccountsLoaded() (app.Tab, tea.Cmd) {
	var cmds []tea.Cmd
	// Account data changed - might need to reload
	// If we have no history data yet (e.g. initial load failed), try again
	if m.historyData == nil {
		m.loading = true
		cmds = append(cmds, m.loadHistoryCmd())
		return m, tea.Batch(cmds...)
	}

	if !m.loading {
		// Check if selected account changed
		accounts := m.state.GetAccounts()
		selectedIdx := m.state.GetSelectedAccountIndex()
		if selectedIdx >= 0 && selectedIdx < len(accounts) {
			if accounts[selectedIdx].Email != m.historyData.Email {
				m.loading = true
				cmds = append(cmds, m.loadHistoryCmd())
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (app.Tab, tea.Cmd) {
	var cmds []tea.Cmd
	switch {
	case key.Matches(msg, m.keys.ToggleRange):
		m.timeRange = m.timeRange.Next()
		m.loading = true
		cmds = append(cmds, m.loadHistoryCmd())

	case key.Matches(msg, m.keys.Refresh):
		m.loading = true
		cmds = append(cmds, m.loadHistoryCmd())

	default:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

// SetSize sets the available size for the history tab.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
}

// ShortHelp returns the key bindings for the short help view.
func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keys.ToggleRange,
		m.keys.Refresh,
	}
}

// FullHelp returns the key bindings for the full help view.
func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.keys.ToggleRange, m.keys.Refresh},
		{m.keys.Up, m.keys.Down},
	}
}

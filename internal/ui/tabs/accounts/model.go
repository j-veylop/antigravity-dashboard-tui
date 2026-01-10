// Package accounts provides the accounts management tab for the Antigravity Dashboard TUI.
package accounts

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/app"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/components"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/styles"
)

// formField represents which field is currently focused in the add form.
type formField int

const (
	fieldEmail formField = iota
	fieldRefreshToken
	fieldSubmit
	fieldCancel
)

// keyMap defines the key bindings specific to the accounts tab.
type keyMap struct {
	Enter   key.Binding
	Delete  key.Binding
	Add     key.Binding
	Refresh key.Binding
	Escape  key.Binding
}

// defaultKeyMap returns the default key bindings for the accounts tab.
func defaultKeyMap() keyMap {
	return keyMap{
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "switch account"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d", "delete"),
			key.WithHelp("d", "delete"),
		),
		Add: key.NewBinding(
			key.WithKeys("n", "a"),
			key.WithHelp("n", "add account"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// Model represents the accounts tab state.
type Model struct {
	state         *app.AppState
	table         table.Model
	width         int
	height        int
	adding        bool
	focusedField  formField
	emailInput    textinput.Model
	tokenInput    textinput.Model
	spinner       components.LoadingSpinner
	keys          keyMap
	confirmDelete bool
	deleteEmail   string
}

// New creates a new accounts model.
func New(state *app.AppState) *Model {
	// Create email input
	emailInput := textinput.New()
	emailInput.Placeholder = "user@example.com"
	emailInput.CharLimit = 100
	emailInput.Width = 40

	// Create token input
	tokenInput := textinput.New()
	tokenInput.Placeholder = "Paste refresh token..."
	tokenInput.CharLimit = 500
	tokenInput.Width = 40
	tokenInput.EchoMode = textinput.EchoPassword

	// Create table
	columns := []table.Column{
		{Title: "Email", Width: 30},
		{Title: "Tier", Width: 8},
		{Title: "Claude", Width: 10},
		{Title: "Gemini", Width: 10},
		{Title: "Status", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.Subtle).
		BorderBottom(true).
		Bold(true).
		Foreground(styles.Primary)
	s.Selected = s.Selected.
		Foreground(styles.TextPrimary).
		Background(styles.BgAccent).
		Bold(true)
	t.SetStyles(s)

	return &Model{
		state:        state,
		table:        t,
		emailInput:   emailInput,
		tokenInput:   tokenInput,
		spinner:      components.NewSpinner("Loading accounts..."),
		keys:         defaultKeyMap(),
		adding:       false,
		focusedField: fieldEmail,
	}
}

// Init initializes the accounts tab.
func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the accounts tab.
func (m *Model) Update(msg tea.Msg) (app.Tab, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle form mode
	if m.adding {
		return m.updateAddForm(msg)
	}

	// Handle delete confirmation
	if m.confirmDelete {
		return m.updateDeleteConfirm(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Enter):
			// Switch to selected account
			if row := m.table.SelectedRow(); len(row) > 0 {
				email := row[0]
				return m, func() tea.Msg {
					return app.SwitchAccountMsg{Email: email}
				}
			}

		case key.Matches(msg, m.keys.Delete):
			// Confirm delete
			if row := m.table.SelectedRow(); len(row) > 0 {
				m.confirmDelete = true
				m.deleteEmail = row[0]
			}

		case key.Matches(msg, m.keys.Add):
			m.adding = true
			m.focusedField = fieldEmail
			m.emailInput.Focus()
			m.emailInput.SetValue("")
			m.tokenInput.SetValue("")
			return m, textinput.Blink

		default:
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			cmds = append(cmds, cmd)
		}

	case app.AccountsLoadedMsg:
		m.updateTableData()
	}

	return m, tea.Batch(cmds...)
}

// updateAddForm handles the add account form.
func (m *Model) updateAddForm(msg tea.Msg) (app.Tab, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.adding = false
			m.emailInput.Blur()
			m.tokenInput.Blur()
			return m, nil

		case "tab", "down":
			m.focusedField = (m.focusedField + 1) % 4
			m.updateFormFocus()
			return m, textinput.Blink

		case "shift+tab", "up":
			m.focusedField = (m.focusedField - 1 + 4) % 4
			m.updateFormFocus()
			return m, textinput.Blink

		case "enter":
			switch m.focusedField {
			case fieldSubmit:
				// Submit the form
				email := m.emailInput.Value()
				token := m.tokenInput.Value()
				if email != "" && token != "" {
					m.adding = false
					m.emailInput.Blur()
					m.tokenInput.Blur()
					// Note: We'd need to integrate with the accounts service here
					// For now, just close the form
					return m, func() tea.Msg {
						return app.AddNotificationMsg{
							Type:    app.NotificationInfo,
							Message: "Account adding not implemented in TUI",
						}
					}
				}
			case fieldCancel:
				m.adding = false
				m.emailInput.Blur()
				m.tokenInput.Blur()
				return m, nil
			default:
				// Move to next field
				m.focusedField = (m.focusedField + 1) % 4
				m.updateFormFocus()
				return m, textinput.Blink
			}
		}
	}

	// Update the focused input
	var cmd tea.Cmd
	switch m.focusedField {
	case fieldEmail:
		m.emailInput, cmd = m.emailInput.Update(msg)
		cmds = append(cmds, cmd)
	case fieldRefreshToken:
		m.tokenInput, cmd = m.tokenInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// updateDeleteConfirm handles the delete confirmation.
func (m *Model) updateDeleteConfirm(msg tea.Msg) (app.Tab, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.confirmDelete = false
			email := m.deleteEmail
			m.deleteEmail = ""
			return m, func() tea.Msg {
				return app.DeleteAccountMsg{Email: email}
			}
		case "n", "N", "esc":
			m.confirmDelete = false
			m.deleteEmail = ""
			return m, nil
		}
	}
	return m, nil
}

// updateFormFocus updates which form field is focused.
func (m *Model) updateFormFocus() {
	m.emailInput.Blur()
	m.tokenInput.Blur()

	switch m.focusedField {
	case fieldEmail:
		m.emailInput.Focus()
	case fieldRefreshToken:
		m.tokenInput.Focus()
	}
}

// updateTableData updates the table with current account data.
func (m *Model) updateTableData() {
	accounts := m.state.GetAccounts()
	rows := make([]table.Row, 0, len(accounts))

	for _, acc := range accounts {
		tier := "UNKNOWN"
		claudeQuota := "-"
		geminiQuota := "-"
		status := "OK"

		if acc.QuotaInfo != nil {
			tier = acc.QuotaInfo.SubscriptionTier
			if tier == "" {
				tier = "UNKNOWN"
			}

			// Extract quotas
			for _, mq := range acc.QuotaInfo.ModelQuotas {
				if mq.Limit > 0 {
					percent := float64(mq.Remaining) / float64(mq.Limit) * 100
					if mq.ModelFamily == "claude" {
						claudeQuota = formatPercent(percent)
					} else if mq.ModelFamily == "gemini" {
						geminiQuota = formatPercent(percent)
					}
				}
				if mq.IsRateLimited {
					status = "RATE LIMITED"
				}
			}

			if acc.QuotaInfo.Error != "" {
				status = "ERROR"
			}
		}

		if acc.IsActive {
			status = "* " + status
		}

		rows = append(rows, table.Row{
			acc.Account.Email,
			tier,
			claudeQuota,
			geminiQuota,
			status,
		})
	}

	m.table.SetRows(rows)
}

// formatPercent formats a percentage for display.
func formatPercent(p float64) string {
	if p >= 100 {
		return "100%"
	}
	if p < 1 {
		return "<1%"
	}
	return fmt.Sprintf("%.0f%%", p)
}

// SetSize sets the available size for the accounts tab.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.table.SetHeight(height - 10)

	// Adjust column widths based on available width
	emailWidth := width - 55
	if emailWidth < 20 {
		emailWidth = 20
	}
	if emailWidth > 40 {
		emailWidth = 40
	}

	columns := []table.Column{
		{Title: "Email", Width: emailWidth},
		{Title: "Tier", Width: 8},
		{Title: "Claude", Width: 10},
		{Title: "Gemini", Width: 10},
		{Title: "Status", Width: 15},
	}
	m.table.SetColumns(columns)
}

// ShortHelp returns the key bindings for the short help view.
func (m *Model) ShortHelp() []key.Binding {
	if m.adding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next field")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
			m.keys.Escape,
		}
	}
	return []key.Binding{
		m.keys.Enter,
		m.keys.Delete,
		m.keys.Add,
	}
}

// FullHelp returns the key bindings for the full help view.
func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.keys.Enter, m.keys.Delete},
		{m.keys.Add, m.keys.Refresh},
	}
}

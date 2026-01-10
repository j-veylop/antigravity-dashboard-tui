package accounts

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/components"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/styles"
)

// View renders the accounts tab.
func (m *Model) View() string {
	if m.state.IsInitialLoading() {
		return m.renderLoading()
	}

	var sections []string

	// Title
	sections = append(sections, m.renderTitle())

	// Main content area
	if m.adding {
		sections = append(sections, m.renderAddForm())
	} else if m.confirmDelete {
		sections = append(sections, m.renderDeleteConfirm())
		sections = append(sections, m.renderTable())
	} else {
		sections = append(sections, m.renderTable())
	}

	// Footer with shortcuts
	sections = append(sections, m.renderFooter())

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return styles.DocStyle.
		Width(m.width).
		Height(m.height).
		Render(content)
}

// renderLoading renders the loading state.
func (m *Model) renderLoading() string {
	return components.RenderSpinnerCentered(m.spinner, m.width, m.height)
}

// renderTitle renders the accounts tab title.
func (m *Model) renderTitle() string {
	title := styles.TitleStyle.Render("Account Management")

	accountCount := m.state.GetAccountCount()
	subtitle := styles.HelpStyle.Render(fmt.Sprintf("%d accounts configured", accountCount))

	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "")
}

// renderTable renders the accounts table.
func (m *Model) renderTable() string {
	accounts := m.state.GetAccounts()

	if len(accounts) == 0 {
		return m.renderEmptyState()
	}

	// Update table data
	m.updateTableData()

	cardWidth := m.width - 6
	if cardWidth < 60 {
		cardWidth = 60
	}

	return styles.CardStyle.Width(cardWidth).Render(m.table.View())
}

// renderEmptyState renders the empty state when no accounts exist.
func (m *Model) renderEmptyState() string {
	cardWidth := m.width - 6
	if cardWidth < 40 {
		cardWidth = 40
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		styles.SubTitleStyle.Render("No Accounts Configured"),
		"",
		styles.HelpStyle.Render("Add accounts to start monitoring quota usage."),
		"",
		styles.InfoTextStyle.Render("Press 'n' to add a new account"),
		"",
	)

	return styles.CardStyle.Width(cardWidth).Render(content)
}

// renderAddForm renders the add account form.
func (m *Model) renderAddForm() string {
	cardWidth := m.width - 10
	if cardWidth < 50 {
		cardWidth = 50
	}
	if cardWidth > 80 {
		cardWidth = 80
	}

	var rows []string

	// Form title
	rows = append(rows, styles.CardTitleStyle.Render("Add New Account"))
	rows = append(rows, "")

	// Email field
	emailLabel := "Email:"
	if m.focusedField == fieldEmail {
		emailLabel = styles.FocusedStyle.Render("> Email:")
	} else {
		emailLabel = styles.BlurredStyle.Render("  Email:")
	}
	rows = append(rows, emailLabel)

	emailInputStyle := styles.BlurredBorderStyle
	if m.focusedField == fieldEmail {
		emailInputStyle = styles.FocusedBorderStyle
	}
	rows = append(rows, emailInputStyle.Width(cardWidth-10).Render(m.emailInput.View()))
	rows = append(rows, "")

	// Refresh token field
	tokenLabel := "Refresh Token:"
	if m.focusedField == fieldRefreshToken {
		tokenLabel = styles.FocusedStyle.Render("> Refresh Token:")
	} else {
		tokenLabel = styles.BlurredStyle.Render("  Refresh Token:")
	}
	rows = append(rows, tokenLabel)

	tokenInputStyle := styles.BlurredBorderStyle
	if m.focusedField == fieldRefreshToken {
		tokenInputStyle = styles.FocusedBorderStyle
	}
	rows = append(rows, tokenInputStyle.Width(cardWidth-10).Render(m.tokenInput.View()))
	rows = append(rows, "")

	// Buttons
	submitStyle := styles.ButtonInactiveStyle
	cancelStyle := styles.ButtonInactiveStyle

	if m.focusedField == fieldSubmit {
		submitStyle = styles.ButtonActiveStyle
	}
	if m.focusedField == fieldCancel {
		cancelStyle = styles.ButtonActiveStyle
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center,
		submitStyle.Render(" Add Account "),
		"  ",
		cancelStyle.Render(" Cancel "),
	)
	rows = append(rows, buttons)
	rows = append(rows, "")

	// Help text
	rows = append(rows, styles.HelpStyle.Render("Tab: next field | Enter: submit | Esc: cancel"))

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	return styles.ModalContentStyle.Width(cardWidth).Render(content)
}

// renderDeleteConfirm renders the delete confirmation dialog.
func (m *Model) renderDeleteConfirm() string {
	cardWidth := 50

	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		styles.WarningTextStyle.Bold(true).Render("Delete Account?"),
		"",
		fmt.Sprintf("Are you sure you want to delete:"),
		styles.ErrorTextStyle.Render(m.deleteEmail),
		"",
		"This action cannot be undone.",
		"",
		lipgloss.JoinHorizontal(lipgloss.Center,
			styles.ButtonActiveStyle.Render(" (Y)es "),
			"  ",
			styles.ButtonInactiveStyle.Render(" (N)o "),
		),
		"",
	)

	return styles.CenterHorizontal(
		styles.ModalContentStyle.Width(cardWidth).Render(content),
		m.width,
	)
}

// renderFooter renders the footer with keyboard shortcuts.
func (m *Model) renderFooter() string {
	var shortcuts []string

	if m.adding {
		shortcuts = []string{
			styles.HelpKeyStyle.Render("Tab") + " next",
			styles.HelpKeyStyle.Render("Enter") + " submit",
			styles.HelpKeyStyle.Render("Esc") + " cancel",
		}
	} else if m.confirmDelete {
		shortcuts = []string{
			styles.HelpKeyStyle.Render("Y") + " confirm",
			styles.HelpKeyStyle.Render("N") + " cancel",
		}
	} else {
		shortcuts = []string{
			styles.HelpKeyStyle.Render("Enter") + " switch",
			styles.HelpKeyStyle.Render("d") + " delete",
			styles.HelpKeyStyle.Render("n") + " add",
			styles.HelpKeyStyle.Render("r") + " refresh",
		}
	}

	footer := ""
	for i, s := range shortcuts {
		if i > 0 {
			footer += styles.HelpSeparatorStyle.Render(" | ")
		}
		footer += s
	}

	return lipgloss.NewStyle().
		MarginTop(1).
		Foreground(styles.TextMuted).
		Render(footer)
}

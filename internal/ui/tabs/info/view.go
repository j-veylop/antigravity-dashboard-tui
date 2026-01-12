package info

import (
	"fmt"
	"runtime"

	"github.com/charmbracelet/lipgloss"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/styles"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/version"
)

// View renders the info tab.
func (m *Model) View() string {
	var sections []string

	// Title
	sections = append(sections, m.renderTitle())

	// Configuration card
	sections = append(sections, m.renderConfigCard())

	// About card
	sections = append(sections, m.renderAboutCard())

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	m.viewport.SetContent(content)

	return styles.DocStyle.
		Width(m.width).
		Height(m.height).
		Render(m.viewport.View())
}

// renderTitle renders the info tab title.
func (m *Model) renderTitle() string {
	title := styles.TitleStyle.Render("Info")
	subtitle := styles.HelpStyle.Render("Configuration and application information")

	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "")
}

// renderConfigCard renders the configuration paths card.
func (m *Model) renderConfigCard() string {
	cardWidth := min(max(m.width-6, 50), 80)

	var rows []string
	rows = append(rows, styles.CardTitleStyle.Render("Configuration"))
	rows = append(rows, "")

	if m.config != nil {
		rows = append(rows, m.renderConfigRow("Accounts File", m.config.AccountsPath))
		rows = append(rows, m.renderConfigRow("Database", m.config.DatabasePath))
		rows = append(rows, m.renderConfigRow("Quota Refresh", m.config.QuotaRefreshInterval.String()))
	} else {
		rows = append(rows, styles.HelpStyle.Render("Configuration not loaded"))
	}

	rows = append(rows, "")
	rows = append(rows, styles.HelpStyle.Render("Press 'c' to copy paths"))

	return styles.CardStyle.Width(cardWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

// renderConfigRow renders a configuration key-value row.
func (m *Model) renderConfigRow(label, value string) string {
	labelStyle := lipgloss.NewStyle().
		Width(18).
		Foreground(styles.TextMuted)

	valueStyle := lipgloss.NewStyle().
		Foreground(styles.TextPrimary)

	return labelStyle.Render(label+":") + " " + valueStyle.Render(value)
}

// renderAboutCard renders the about/version information card.
func (m *Model) renderAboutCard() string {
	cardWidth := min(max(m.width-6, 50), 80)

	var rows []string
	rows = append(rows, styles.CardTitleStyle.Render("About Antigravity Dashboard TUI"))
	rows = append(rows, "")

	rows = append(rows, m.renderConfigRow("Version", version.GetVersion()))
	rows = append(rows, m.renderConfigRow("Build Date", version.GetDate()))
	rows = append(rows, m.renderConfigRow("Git Commit", version.GetCommit()))
	rows = append(rows, m.renderConfigRow("Go Version", runtime.Version()))
	rows = append(rows, m.renderConfigRow("Platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)))
	rows = append(rows, "")

	accountCount := m.state.GetAccountCount()
	rows = append(rows, fmt.Sprintf("Accounts: %s", styles.InfoTextStyle.Render(fmt.Sprintf("%d", accountCount))))

	return styles.CardStyle.Width(cardWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

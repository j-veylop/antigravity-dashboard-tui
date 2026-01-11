package dashboard

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/components"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/styles"
)

// View renders the dashboard component.
func (m *Model) View() string {
	if m.state.IsInitialLoading() {
		return m.renderLoading()
	}

	var sections []string

	sections = append(sections, m.renderTitle())

	sections = append(sections, m.renderQuotaList())

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	m.viewport.SetContent(content)

	return styles.DocStyle.
		Width(m.width).
		Height(m.height).
		Render(m.viewport.View())
}

// renderLoading renders the loading state.
func (m *Model) renderLoading() string {
	return components.RenderSpinnerCentered(m.spinner, m.width, m.height)
}

// renderTitle renders the dashboard title.
func (m *Model) renderTitle() string {
	title := styles.TitleStyle.Render("Antigravity Dashboard")
	subtitle := styles.HelpStyle.Render("Multi-account Google Cloud quota monitor")

	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "")
}

// renderQuotaList renders the list of accounts with their quotas.
func (m *Model) renderQuotaList() string {
	accounts := m.state.GetAccounts()

	cardWidth := max(m.width-6, 40)

	var rows []string

	titleIcon := lipgloss.NewStyle().Foreground(styles.Primary).Render("◈")
	rows = append(rows, fmt.Sprintf("%s %s", titleIcon, styles.CardTitleStyle.Render("Account Quotas")))

	if len(accounts) == 0 {
		rows = append(rows, "")
		emptyIcon := lipgloss.NewStyle().Foreground(styles.Subtle).Render("○")
		rows = append(rows, fmt.Sprintf("  %s %s", emptyIcon, styles.HelpStyle.Render("No accounts configured")))
		rows = append(rows, "")
		rows = append(rows, styles.InfoTextStyle.Render("  ╰─▶ Add accounts by editing accounts.json"))

		return styles.CardStyle.Width(cardWidth).Render(
			lipgloss.JoinVertical(lipgloss.Left, rows...),
		)
	}

	dividerWidth := max(cardWidth-8, 20)
	divider := lipgloss.NewStyle().Foreground(styles.Subtle).Render(
		"  ├" + strings.Repeat("─", dividerWidth) + "┤",
	)

	rows = append(rows, "")

	for i, acc := range accounts {
		accountRow := m.renderAccountRow(acc, i == m.selectedIndex, cardWidth-4)
		rows = append(rows, accountRow)
		if i < len(accounts)-1 {
			rows = append(rows, "")
			rows = append(rows, divider)
			rows = append(rows, "")
		}
	}

	rows = append(rows, "")

	return styles.CardStyle.Width(cardWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

func (m *Model) renderAccountRow(acc models.AccountWithQuota, selected bool, width int) string {
	var lines []string

	lines = append(lines, m.renderAccountHeader(acc, selected))
	lines = append(lines, "")

	// Quota bars
	contentWidth := max(width-4, 20)

	switch {
	case acc.QuotaInfo == nil:
		lines = append(lines, m.renderAccountLoading(contentWidth)...)
	case acc.QuotaInfo.Error != "":
		lines = append(lines, m.renderAccountError(acc, contentWidth)...)
	default:
		lines = append(lines, m.renderAccountQuotas(acc, contentWidth)...)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *Model) renderAccountHeader(acc models.AccountWithQuota, selected bool) string {
	activeIndicator := lipgloss.NewStyle().Foreground(styles.Subtle).Render("○ ")
	if acc.IsActive {
		activeIndicator = styles.SuccessTextStyle.Render("● ")
	}

	selectionPrefix := "  "
	if selected {
		selectionPrefix = styles.FocusedStyle.Render("▸ ")
	}

	email := acc.Email
	if len(email) > 35 {
		email = email[:32] + "..."
	}

	tier := "UNKNOWN"
	tierStyle := styles.TierUnknownStyle
	tierIcon := "◇"
	if acc.QuotaInfo != nil {
		tier = acc.QuotaInfo.SubscriptionTier
		tierStyle = styles.GetTierStyle(tier)
		if tier == "PRO" {
			tierIcon = "◆"
		}
	}

	return fmt.Sprintf("%s%s%s %s",
		selectionPrefix,
		activeIndicator,
		lipgloss.NewStyle().Bold(true).Render(email),
		tierStyle.Render(tierIcon+" "+tier),
	)
}

func (m *Model) renderAccountQuotas(acc models.AccountWithQuota, width int) []string {
	var lines []string
	proj := m.state.GetProjection(acc.Email)
	tier := acc.QuotaInfo.SubscriptionTier

	claudePercent, geminiPercent, claudeResetSec, geminiResetSec := m.calculateDisplayQuotas(acc.QuotaInfo)

	var claudeProj, geminiProj *models.ModelProjection
	if proj != nil {
		claudeProj = proj.Claude
		geminiProj = proj.Gemini
	}

	if claudePercent >= 0 {
		lines = append(lines, m.renderModelQuota(
			"Claude", "⬡", "#cc785c", acc.Email+":claude", claudePercent, width, claudeResetSec, tier, claudeProj)...)
	}

	if claudePercent >= 0 && geminiPercent >= 0 {
		lines = append(lines, "")
	}

	if geminiPercent >= 0 {
		lines = append(lines, m.renderModelQuota(
			"Gemini", "◎", "#4285f4", acc.Email+":gemini", geminiPercent, width, geminiResetSec, tier, geminiProj)...)
	}

	if acc.QuotaInfo.TotalLimit > 0 {
		lines = append(lines, "")
		totalPercent := float64(acc.QuotaInfo.TotalRemaining) / float64(acc.QuotaInfo.TotalLimit) * 100
		totalIcon := lipgloss.NewStyle().Foreground(styles.Primary).Render("◈")
		totalLabel := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true).Render("Total Quota Left")
		lines = append(lines, fmt.Sprintf("  %s %s", totalIcon, totalLabel))
		block := m.renderTotalBar(totalPercent, width)
		lines = append(lines, block)
	}

	return lines
}

func (m *Model) calculateDisplayQuotas(
	quotaInfo *models.QuotaInfo,
) (claudePercent, geminiPercent float64, claudeResetSec, geminiResetSec int64) {
	claudePercent = -1.0
	geminiPercent = -1.0
	claudeResetSec = 0
	geminiResetSec = 0

	for _, mq := range quotaInfo.ModelQuotas {
		switch mq.ModelFamily {
		case "claude":
			currentPercent := 0.0
			if mq.Limit > 0 && !mq.IsRateLimited {
				currentPercent = float64(mq.Remaining) / float64(mq.Limit) * 100
			}

			if claudePercent < 0 || currentPercent < claudePercent {
				claudePercent = currentPercent
				if !mq.ResetTime.IsZero() {
					claudeResetSec = max(int64(time.Until(mq.ResetTime).Seconds()), 0)
				}
			}
		case "gemini":
			currentPercent := 0.0
			if mq.Limit > 0 && !mq.IsRateLimited {
				currentPercent = float64(mq.Remaining) / float64(mq.Limit) * 100
			}

			if geminiPercent < 0 || currentPercent < geminiPercent {
				geminiPercent = currentPercent
				if !mq.ResetTime.IsZero() {
					geminiResetSec = max(int64(time.Until(mq.ResetTime).Seconds()), 0)
				}
			}
		}
	}
	return claudePercent, geminiPercent, claudeResetSec, geminiResetSec
}

func (m *Model) renderModelQuota(
	label, icon, colorHex, animKey string,
	percent float64,
	width int,
	resetSec int64,
	tier string,
	proj *models.ModelProjection,
) []string {
	var lines []string
	iconStr := lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex)).Render(icon)
	labelStr := lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex)).Bold(true).Render(label)
	lines = append(lines, fmt.Sprintf("  %s %s", iconStr, labelStr))

	displayPercent := percent
	if anim, ok := m.animations[animKey]; ok {
		displayPercent = anim.CurrentPercent
	}
	block := m.renderQuotaBarWithTime(displayPercent, width, resetSec, tier, proj)
	lines = append(lines, block)
	return lines
}

func (m *Model) renderAccountError(acc models.AccountWithQuota, width int) []string {
	var lines []string
	tier := acc.QuotaInfo.SubscriptionTier

	claudeIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc785c")).Render("⬡")
	claudeLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc785c")).Bold(true).Render("Claude")
	lines = append(lines, fmt.Sprintf("  %s %s", claudeIcon, claudeLabel))
	block1 := m.renderQuotaBarWithTime(0, width, 0, tier, nil)
	lines = append(lines, block1)

	lines = append(lines, "")

	geminiIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#4285f4")).Render("◎")
	geminiLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#4285f4")).Bold(true).Render("Gemini")
	lines = append(lines, fmt.Sprintf("  %s %s", geminiIcon, geminiLabel))
	block2 := m.renderQuotaBarWithTime(0, width, 0, tier, nil)
	lines = append(lines, block2)

	return lines
}

func (m *Model) renderAccountLoading(width int) []string {
	var lines []string

	claudeIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc785c")).Render("⬡")
	claudeLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc785c")).Bold(true).Render("Claude")
	lines = append(lines, fmt.Sprintf("  %s %s", claudeIcon, claudeLabel))
	block1 := m.renderLoadingBar("claude", width)
	lines = append(lines, block1)

	lines = append(lines, "")

	geminiIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#4285f4")).Render("◎")
	geminiLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#4285f4")).Bold(true).Render("Gemini")
	lines = append(lines, fmt.Sprintf("  %s %s", geminiIcon, geminiLabel))
	block2 := m.renderLoadingBar("gemini", width)
	lines = append(lines, block2)

	lines = append(lines, "")

	totalIcon := lipgloss.NewStyle().Foreground(styles.Primary).Render("◈")
	totalLabel := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true).Render("Total Quota Left")
	lines = append(lines, fmt.Sprintf("  %s %s", totalIcon, totalLabel))
	block3 := m.renderLoadingTotalBar(width)
	lines = append(lines, block3)

	return lines
}

func (m *Model) renderQuotaBarWithTime(
	percent float64,
	width int,
	resetSec int64,
	tier string,
	proj *models.ModelProjection,
) string {
	const (
		indentWidth  = 4
		percentWidth = 6
		rateWidth    = 10
		badgeWidth   = 10
	)

	rightSideWidth := percentWidth + rateWidth + badgeWidth
	barWidth := max(width-indentWidth-rightSideWidth-4, 10)

	line1 := m.renderQuotaBarFirstLine(percent, barWidth, percentWidth, rateWidth, badgeWidth, proj)

	if resetSec > 0 {
		line2 := m.renderQuotaBarSecondLine(barWidth, percentWidth, rateWidth, badgeWidth, resetSec, tier, proj)
		return lipgloss.JoinVertical(lipgloss.Left, line1, line2)
	}

	return line1
}

const indentSpace = "    "

func (m *Model) renderQuotaBarFirstLine(
	percent float64,
	barWidth, percentWidth, rateWidth, badgeWidth int,
	proj *models.ModelProjection,
) string {
	percentStr := styles.GetQuotaStyle(percent, false).
		Width(percentWidth).
		Align(lipgloss.Right).
		Render(fmt.Sprintf("%.0f%%", percent))

	rateStr := ""
	if proj != nil && proj.SessionRate > 0 {
		rateStyle := styles.HelpStyle
		switch proj.Status {
		case models.ProjectionWarning:
			rateStyle = styles.WarningTextStyle
		case models.ProjectionCritical:
			rateStyle = styles.ErrorTextStyle
		}
		rateStr = rateStyle.Width(rateWidth).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f%%/hr", proj.SessionRate))
	} else {
		rateStr = lipgloss.NewStyle().Width(rateWidth).Render("")
	}

	badgeStr := ""
	if proj != nil && proj.Status != models.ProjectionUnknown {
		var badgeStyle lipgloss.Style
		var badgeText string

		switch proj.Status {
		case models.ProjectionCritical:
			badgeStyle = styles.ProjectionCriticalStyle
			badgeText = "▲ CRITICAL"
		case models.ProjectionWarning:
			badgeStyle = styles.ProjectionWarningStyle
			badgeText = "▲ WARNING"
		case models.ProjectionSafe:
			badgeStyle = styles.ProjectionSafeStyle
			badgeText = "● SAFE"
		}
		badgeStr = badgeStyle.Width(badgeWidth).Align(lipgloss.Right).Render(badgeText)
	} else {
		badgeStr = lipgloss.NewStyle().Width(badgeWidth).Render("")
	}

	bar1 := components.RenderGradientBar(percent, barWidth)

	return lipgloss.JoinHorizontal(lipgloss.Left,
		indentSpace,
		bar1,
		" ",
		percentStr,
		" ",
		rateStr,
		" ",
		badgeStr,
	)
}

func (m *Model) renderQuotaBarSecondLine(
	barWidth, timeWidth, rateWidth, badgeWidth int,
	resetSec int64,
	tier string,
	proj *models.ModelProjection,
) string {
	period := m.calculatePeriod(tier, resetSec)

	timePercent := 1.0
	if period > 0 {
		timePercent = 1.0 - (float64(resetSec) / float64(period))
		if timePercent < 0 {
			timePercent = 0
		}
		if timePercent > 1 {
			timePercent = 1
		}
	}

	resetTimeText := formatDuration(float64(resetSec) / 3600.0)
	resetTimeStr := lipgloss.NewStyle().
		Foreground(styles.TextSecondary).
		Width(timeWidth).
		Align(lipgloss.Right).
		Render(resetTimeText)

	depleteWidth := rateWidth + badgeWidth
	depleteStr := ""
	if proj != nil && proj.SessionRate > 0 && !math.IsInf(proj.SessionHoursLeft, 0) {
		depleteText := fmt.Sprintf("(Depletes: %s)", formatDuration(proj.SessionHoursLeft))
		if len(depleteText) > depleteWidth {
			depleteText = depleteText[:depleteWidth-1] + ")"
		}
		depleteStyle := styles.HelpStyle
		switch proj.Status {
		case models.ProjectionCritical:
			depleteStyle = styles.ErrorTextStyle
		case models.ProjectionWarning:
			depleteStyle = styles.WarningTextStyle
		}
		depleteStr = depleteStyle.Width(depleteWidth).Align(lipgloss.Right).Render(depleteText)
	} else {
		depleteStr = lipgloss.NewStyle().Width(depleteWidth).Render("")
	}

	bar2 := components.RenderTimeBarChars(timePercent, barWidth)

	return lipgloss.JoinHorizontal(lipgloss.Left,
		indentSpace,
		bar2,
		" ",
		resetTimeStr,
		" ",
		depleteStr,
	)
}

func (m *Model) calculatePeriod(tier string, resetSec int64) int64 {
	const hourInSeconds int64 = 3600
	const dayInSeconds int64 = 86400
	const proPeriodSeconds int64 = 5 * 3600

	switch {
	case tier == "PRO":
		return proPeriodSeconds
	case resetSec <= hourInSeconds:
		return hourInSeconds
	default:
		return dayInSeconds
	}
}

func formatDuration(hours float64) string {
	if hours <= 0 || math.IsInf(hours, 0) || math.IsNaN(hours) {
		return "---"
	}

	h := int(hours)
	m := int((hours - float64(h)) * 60)

	if h >= 24 {
		days := h / 24
		remainingHours := h % 24
		return fmt.Sprintf("%dd %02dh", days, remainingHours)
	}

	return fmt.Sprintf("%dh %02dm", h, m)
}

func (m *Model) renderTotalBar(percent float64, width int) string {
	const (
		indentWidth  = 4
		percentWidth = 6
		rateWidth    = 10
		badgeWidth   = 10
	)

	rightSideWidth := percentWidth + rateWidth + badgeWidth
	barWidth := max(width-indentWidth-rightSideWidth-4, 10)

	indent := indentSpace

	percentStr := styles.GetQuotaStyle(percent, false).
		Width(percentWidth).
		Align(lipgloss.Right).
		Render(fmt.Sprintf("%.0f%%", percent))

	rateStr := lipgloss.NewStyle().Width(rateWidth).Render("")
	badgeStr := lipgloss.NewStyle().Width(badgeWidth).Render("")

	bar := components.RenderGradientBar(percent, barWidth)

	return lipgloss.JoinHorizontal(lipgloss.Left,
		indent,
		bar,
		" ",
		percentStr,
		" ",
		rateStr,
		" ",
		badgeStr,
	)
}

func (m *Model) renderLoadingBar(label string, width int) string {
	quotaBar := components.SimpleQuotaBarLoading(label, width, m.animationFrame)
	timeBar := components.SimpleTimeBarLoading(label, width, m.animationFrame)
	return lipgloss.JoinVertical(lipgloss.Left, quotaBar, timeBar)
}

func (m *Model) renderLoadingTotalBar(width int) string {
	return components.SimpleQuotaBarLoading("total", width, m.animationFrame)
}

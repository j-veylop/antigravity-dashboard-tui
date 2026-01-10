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

	cardWidth := m.width - 6
	if cardWidth < 40 {
		cardWidth = 40
	}

	var rows []string

	titleIcon := lipgloss.NewStyle().Foreground(styles.Primary).Render("◈")
	rows = append(rows, fmt.Sprintf("%s %s", titleIcon, styles.CardTitleStyle.Render("Account Quotas")))

	if len(accounts) == 0 {
		rows = append(rows, "")
		emptyIcon := lipgloss.NewStyle().Foreground(styles.Subtle).Render("○")
		rows = append(rows, fmt.Sprintf("  %s %s", emptyIcon, styles.HelpStyle.Render("No accounts configured")))
		rows = append(rows, "")
		rows = append(rows, styles.InfoTextStyle.Render("  ╰─▶ Add accounts via the Accounts tab"))

		return styles.CardStyle.Width(cardWidth).Render(
			lipgloss.JoinVertical(lipgloss.Left, rows...),
		)
	}

	dividerWidth := cardWidth - 8
	if dividerWidth < 20 {
		dividerWidth = 20
	}
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

	activeIndicator := lipgloss.NewStyle().Foreground(styles.Subtle).Render("○ ")
	if acc.IsActive {
		activeIndicator = styles.SuccessTextStyle.Render("● ")
	}

	selectionPrefix := "  "
	if selected {
		selectionPrefix = styles.FocusedStyle.Render("▸ ")
	}

	email := acc.Account.Email
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

	header := fmt.Sprintf("%s%s%s %s",
		selectionPrefix,
		activeIndicator,
		lipgloss.NewStyle().Bold(true).Render(email),
		tierStyle.Render(tierIcon+" "+tier),
	)
	lines = append(lines, header)
	lines = append(lines, "")

	// Quota bars
	contentWidth := width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	proj := m.state.GetProjection(acc.Account.Email)
	var claudeProj, geminiProj *models.ModelProjection
	if proj != nil {
		claudeProj = proj.Claude
		geminiProj = proj.Gemini
	}

	if acc.QuotaInfo != nil && acc.QuotaInfo.Error == "" {
		claudePercent := -1.0
		geminiPercent := -1.0
		claudeResetSec := int64(0)
		geminiResetSec := int64(0)

		for _, mq := range acc.QuotaInfo.ModelQuotas {
			if mq.ModelFamily == "claude" {
				currentPercent := 0.0
				if mq.Limit > 0 && !mq.IsRateLimited {
					currentPercent = float64(mq.Remaining) / float64(mq.Limit) * 100
				}

				if claudePercent < 0 || currentPercent < claudePercent {
					claudePercent = currentPercent
					if !mq.ResetTime.IsZero() {
						claudeResetSec = int64(time.Until(mq.ResetTime).Seconds())
						if claudeResetSec < 0 {
							claudeResetSec = 0
						}
					}
				}
			} else if mq.ModelFamily == "gemini" {
				currentPercent := 0.0
				if mq.Limit > 0 && !mq.IsRateLimited {
					currentPercent = float64(mq.Remaining) / float64(mq.Limit) * 100
				}

				if geminiPercent < 0 || currentPercent < geminiPercent {
					geminiPercent = currentPercent
					if !mq.ResetTime.IsZero() {
						geminiResetSec = int64(time.Until(mq.ResetTime).Seconds())
						if geminiResetSec < 0 {
							geminiResetSec = 0
						}
					}
				}
			}
		}

		if claudePercent >= 0 {
			claudeIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc785c")).Render("⬡")
			claudeLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc785c")).Bold(true).Render("Claude")
			lines = append(lines, fmt.Sprintf("  %s %s", claudeIcon, claudeLabel))

			animKey := acc.Account.Email + ":claude"
			displayPercent := claudePercent
			if anim, ok := m.animations[animKey]; ok {
				displayPercent = anim.CurrentPercent
			}
			block := m.renderQuotaBarWithTime(displayPercent, contentWidth, claudeResetSec, tier, claudeProj)
			lines = append(lines, block)
		}

		if claudePercent >= 0 && geminiPercent >= 0 {
			lines = append(lines, "")
		}

		if geminiPercent >= 0 {
			geminiIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#4285f4")).Render("◎")
			geminiLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#4285f4")).Bold(true).Render("Gemini")
			lines = append(lines, fmt.Sprintf("  %s %s", geminiIcon, geminiLabel))

			animKey := acc.Account.Email + ":gemini"
			displayPercent := geminiPercent
			if anim, ok := m.animations[animKey]; ok {
				displayPercent = anim.CurrentPercent
			}
			block := m.renderQuotaBarWithTime(displayPercent, contentWidth, geminiResetSec, tier, geminiProj)
			lines = append(lines, block)
		}

		if acc.QuotaInfo != nil && acc.QuotaInfo.TotalLimit > 0 {
			lines = append(lines, "")
			totalPercent := float64(acc.QuotaInfo.TotalRemaining) / float64(acc.QuotaInfo.TotalLimit) * 100
			totalIcon := lipgloss.NewStyle().Foreground(styles.Primary).Render("◈")
			totalLabel := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true).Render("Total Quota Left")
			lines = append(lines, fmt.Sprintf("  %s %s", totalIcon, totalLabel))
			block := m.renderTotalBar(totalPercent, contentWidth)
			lines = append(lines, block)
		}
	} else if acc.QuotaInfo != nil && acc.QuotaInfo.Error != "" {
		claudeIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc785c")).Render("⬡")
		claudeLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc785c")).Bold(true).Render("Claude")
		lines = append(lines, fmt.Sprintf("  %s %s", claudeIcon, claudeLabel))
		block1 := m.renderQuotaBarWithTime(0, contentWidth, 0, tier, nil)
		lines = append(lines, block1)

		lines = append(lines, "")

		geminiIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#4285f4")).Render("◎")
		geminiLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#4285f4")).Bold(true).Render("Gemini")
		lines = append(lines, fmt.Sprintf("  %s %s", geminiIcon, geminiLabel))
		block2 := m.renderQuotaBarWithTime(0, contentWidth, 0, tier, nil)
		lines = append(lines, block2)
	} else {
		claudeIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc785c")).Render("⬡")
		claudeLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#cc785c")).Bold(true).Render("Claude")
		lines = append(lines, fmt.Sprintf("  %s %s", claudeIcon, claudeLabel))
		block1 := m.renderLoadingBar("claude", contentWidth)
		lines = append(lines, block1)

		lines = append(lines, "")

		geminiIcon := lipgloss.NewStyle().Foreground(lipgloss.Color("#4285f4")).Render("◎")
		geminiLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#4285f4")).Bold(true).Render("Gemini")
		lines = append(lines, fmt.Sprintf("  %s %s", geminiIcon, geminiLabel))
		block2 := m.renderLoadingBar("gemini", contentWidth)
		lines = append(lines, block2)

		lines = append(lines, "")

		totalIcon := lipgloss.NewStyle().Foreground(styles.Primary).Render("◈")
		totalLabel := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true).Render("Total Quota Left")
		lines = append(lines, fmt.Sprintf("  %s %s", totalIcon, totalLabel))
		block3 := m.renderLoadingTotalBar(contentWidth)
		lines = append(lines, block3)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *Model) renderQuotaBarWithTime(percent float64, width int, resetSec int64, tier string, proj *models.ModelProjection) string {
	const (
		indentWidth  = 4
		percentWidth = 6
		rateWidth    = 10
		badgeWidth   = 10
	)

	rightSideWidth := percentWidth + rateWidth + badgeWidth
	barWidth := width - indentWidth - rightSideWidth - 4
	if barWidth < 10 {
		barWidth = 10
	}

	timeWidth := percentWidth
	depleteWidth := rateWidth + badgeWidth

	indent := "    "

	percentStr := styles.GetQuotaStyle(percent, false).
		Width(percentWidth).
		Align(lipgloss.Right).
		Render(fmt.Sprintf("%.0f%%", percent))

	rateStr := ""
	if proj != nil && proj.SessionRate > 0 {
		rateStyle := styles.HelpStyle
		if proj.Status == models.ProjectionWarning {
			rateStyle = styles.WarningTextStyle
		} else if proj.Status == models.ProjectionCritical {
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

	line1 := lipgloss.JoinHorizontal(lipgloss.Left,
		indent,
		bar1,
		" ",
		percentStr,
		" ",
		rateStr,
		" ",
		badgeStr,
	)

	if resetSec > 0 {
		const hourInSeconds int64 = 3600
		const dayInSeconds int64 = 86400
		const proPeriodSeconds int64 = 5 * 3600

		var period int64
		if tier == "PRO" {
			period = proPeriodSeconds
		} else if resetSec <= hourInSeconds {
			period = hourInSeconds
		} else {
			period = dayInSeconds
		}

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

		depleteStr := ""
		if proj != nil && proj.SessionRate > 0 && !math.IsInf(proj.SessionHoursLeft, 0) {
			depleteText := fmt.Sprintf("(Depletes: %s)", formatDuration(proj.SessionHoursLeft))
			if len(depleteText) > depleteWidth {
				depleteText = depleteText[:depleteWidth-1] + ")"
			}
			depleteStyle := styles.HelpStyle
			if proj.Status == models.ProjectionCritical {
				depleteStyle = styles.ErrorTextStyle
			} else if proj.Status == models.ProjectionWarning {
				depleteStyle = styles.WarningTextStyle
			}
			depleteStr = depleteStyle.Width(depleteWidth).Align(lipgloss.Right).Render(depleteText)
		} else {
			depleteStr = lipgloss.NewStyle().Width(depleteWidth).Render("")
		}

		bar2 := components.RenderTimeBarChars(timePercent, barWidth)

		line2 := lipgloss.JoinHorizontal(lipgloss.Left,
			indent,
			bar2,
			" ",
			resetTimeStr,
			" ",
			depleteStr,
		)

		return lipgloss.JoinVertical(lipgloss.Left, line1, line2)
	}

	return line1
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
	barWidth := width - indentWidth - rightSideWidth - 4
	if barWidth < 10 {
		barWidth = 10
	}

	indent := "    "

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

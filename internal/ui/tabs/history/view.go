package history

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/components"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/styles"
)

const notAvailable = "N/A"

// View renders the history tab.
func (m *Model) View() string {
	if m.loading {
		return m.renderLoading()
	}
	if m.errorMsg != "" {
		return m.renderError()
	}
	if m.historyData == nil || !m.historyData.HasData() {
		return m.renderEmpty()
	}

	var sections []string

	// Header with account name and time range selector
	sections = append(sections, m.renderHeader())

	// Summary stats card
	sections = append(sections, m.renderSummaryCard())

	// Rate limit stats card
	sections = append(sections, m.renderRateLimitCard())

	// Exhaustion stats card
	sections = append(sections, m.renderExhaustionCard())

	// Daily consumption chart
	sections = append(sections, m.renderConsumptionChart())

	// Hourly heatmap
	sections = append(sections, m.renderHourlyHeatmap())

	// Weekly pattern
	sections = append(sections, m.renderWeeklyPattern())

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	m.viewport.SetContent(content)

	return styles.DocStyle.
		Width(m.width).
		Height(m.height).
		Render(m.viewport.View())
}

func (m *Model) renderLoading() string {
	return styles.DocStyle.
		Width(m.width).
		Height(m.height).
		Render(styles.HelpStyle.Render("Loading history data..."))
}

func (m *Model) renderError() string {
	content := fmt.Sprintf("%s %s",
		styles.ErrorTextStyle.Render("Error:"),
		m.errorMsg,
	)
	return styles.DocStyle.
		Width(m.width).
		Height(m.height).
		Render(content)
}

func (m *Model) renderEmpty() string {
	content := lipgloss.JoinVertical(lipgloss.Left,
		styles.TitleStyle.Render("History"),
		"",
		styles.HelpStyle.Render("No historical data available yet."),
		styles.HelpStyle.Render("Data will appear as quota snapshots are recorded."),
	)
	return styles.DocStyle.
		Width(m.width).
		Height(m.height).
		Render(content)
}

func (m *Model) renderHeader() string {
	// Account name
	email := m.historyData.Email
	if len(email) > 40 {
		email = email[:37] + "..."
	}

	title := styles.TitleStyle.Render("History: " + email)

	// Time range indicator with toggle hint
	rangeStyle := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary)

	rangeIndicator := rangeStyle.Render(fmt.Sprintf("[t] %s", m.timeRange.String()))

	header := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", rangeIndicator)

	// Subtitle with data range
	var subtitle string
	if !m.historyData.FirstDataPoint.IsZero() {
		dataRange := fmt.Sprintf("Data: %s → %s (%d days)",
			m.historyData.FirstDataPoint.Format("Jan 2, 2006"),
			m.historyData.LastDataPoint.Format("Jan 2, 2006"),
			m.historyData.TotalDataDays,
		)
		subtitle = styles.HelpStyle.Render(dataRange)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, subtitle, "")
}

func (m *Model) renderSummaryCard() string {
	cardWidth := m.width - 6
	if cardWidth < 40 {
		cardWidth = 40
	}

	var rows []string

	titleIcon := lipgloss.NewStyle().Foreground(styles.Primary).Render("◈")
	rows = append(rows, fmt.Sprintf("%s %s", titleIcon, styles.CardTitleStyle.Render("Summary")))
	rows = append(rows, "")

	// Stats grid
	h := m.historyData

	col1 := fmt.Sprintf("  Rate Limit Hits: %s",
		lipgloss.NewStyle().Bold(true).Foreground(styles.Warning).Render(fmt.Sprintf("%d", h.RateLimits.HitsInRange)))
	col2 := fmt.Sprintf("Sessions: %s",
		lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", h.Exhaustion.TotalSessions)))

	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(cardWidth/2).Render(col1),
		lipgloss.NewStyle().Width(cardWidth/2).Render(col2),
	))

	// Second row
	avgExhaust := notAvailable
	if h.Exhaustion.AvgTimeToExhaust > 0 {
		avgExhaust = formatDuration(h.Exhaustion.AvgTimeToExhaust)
	}
	col3 := fmt.Sprintf("  Avg Exhaust: %s",
		lipgloss.NewStyle().Bold(true).Foreground(styles.Success).Render(avgExhaust))
	col4 := fmt.Sprintf("Data Points: %s",
		lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", h.TotalDataPoints)))

	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(cardWidth/2).Render(col3),
		lipgloss.NewStyle().Width(cardWidth/2).Render(col4),
	))

	rows = append(rows, "")

	return styles.CardStyle.Width(cardWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

func (m *Model) renderRateLimitCard() string {
	cardWidth := m.width - 6
	if cardWidth < 40 {
		cardWidth = 40
	}

	var rows []string

	titleIcon := lipgloss.NewStyle().Foreground(styles.Warning).Render("🚫")
	rows = append(rows, fmt.Sprintf("%s %s", titleIcon, styles.CardTitleStyle.Render("Rate Limits (Transitions)")))
	rows = append(rows, "")

	rl := m.historyData.RateLimits

	// Total hits
	totalStyle := lipgloss.NewStyle().Bold(true)
	if rl.TotalHits > 10 {
		totalStyle = totalStyle.Foreground(styles.Error)
	} else if rl.TotalHits > 5 {
		totalStyle = totalStyle.Foreground(styles.Warning)
	} else {
		totalStyle = totalStyle.Foreground(styles.Success)
	}

	rows = append(rows, fmt.Sprintf("  Total Hits (all time): %s", totalStyle.Render(fmt.Sprintf("%d", rl.TotalHits))))

	// Hits in range
	rows = append(rows, fmt.Sprintf("  Hits in range: %s    Last 7d: %s    Last 30d: %s",
		lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%d", rl.HitsInRange)),
		lipgloss.NewStyle().Render(fmt.Sprintf("%d", rl.HitsLast7Days)),
		lipgloss.NewStyle().Render(fmt.Sprintf("%d", rl.HitsLast30Days)),
	))

	// Last hit time
	if !rl.LastHitTime.IsZero() {
		ago := time.Since(rl.LastHitTime)
		agoStr := formatTimeAgo(ago)
		rows = append(rows, fmt.Sprintf("  Last hit: %s", styles.HelpStyle.Render(agoStr)))
	}

	rows = append(rows, "")

	return styles.CardStyle.Width(cardWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

func (m *Model) renderExhaustionCard() string {
	cardWidth := m.width - 6
	if cardWidth < 40 {
		cardWidth = 40
	}

	var rows []string

	titleIcon := lipgloss.NewStyle().Foreground(styles.Success).Render("⏱️")
	rows = append(rows, fmt.Sprintf("%s %s", titleIcon, styles.CardTitleStyle.Render("Time to Exhaustion")))
	rows = append(rows, "")

	ex := m.historyData.Exhaustion

	if ex.TotalSessions == 0 {
		rows = append(rows, styles.HelpStyle.Render("  No session data available"))
	} else {
		// Time stats
		avgStr := notAvailable
		medianStr := notAvailable
		minStr := notAvailable
		maxStr := notAvailable

		if ex.AvgTimeToExhaust > 0 {
			avgStr = formatDuration(ex.AvgTimeToExhaust)
		}
		if ex.MedianTimeToExhaust > 0 {
			medianStr = formatDuration(ex.MedianTimeToExhaust)
		}
		if ex.MinTimeToExhaust > 0 {
			minStr = formatDuration(ex.MinTimeToExhaust)
		}
		if ex.MaxTimeToExhaust > 0 {
			maxStr = formatDuration(ex.MaxTimeToExhaust)
		}

		rows = append(rows, fmt.Sprintf("  Avg: %s    Median: %s    Min: %s    Max: %s",
			lipgloss.NewStyle().Bold(true).Foreground(styles.Primary).Render(avgStr),
			lipgloss.NewStyle().Bold(true).Render(medianStr),
			styles.HelpStyle.Render(minStr),
			styles.HelpStyle.Render(maxStr),
		))

		// Exhaustion rate
		exhaustedPct := ex.ExhaustionRate
		exhaustStyle := lipgloss.NewStyle().Bold(true)
		if exhaustedPct > 50 {
			exhaustStyle = exhaustStyle.Foreground(styles.Error)
		} else if exhaustedPct > 25 {
			exhaustStyle = exhaustStyle.Foreground(styles.Warning)
		} else {
			exhaustStyle = exhaustStyle.Foreground(styles.Success)
		}

		rows = append(rows, fmt.Sprintf("  Exhausted: %s/%d sessions (%s)",
			exhaustStyle.Render(fmt.Sprintf("%d", ex.ExhaustedSessions)),
			ex.TotalSessions,
			exhaustStyle.Render(fmt.Sprintf("%.0f%%", exhaustedPct)),
		))

		// Average start percent
		if ex.AvgStartPercent > 0 {
			rows = append(rows, fmt.Sprintf("  Avg starting quota: %s",
				styles.HelpStyle.Render(fmt.Sprintf("%.0f%%", ex.AvgStartPercent)),
			))
		}
	}

	rows = append(rows, "")

	return styles.CardStyle.Width(cardWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

func (m *Model) renderConsumptionChart() string {
	cardWidth := m.width - 6
	if cardWidth < 40 {
		cardWidth = 40
	}

	var rows []string

	titleIcon := lipgloss.NewStyle().Foreground(styles.Primary).Render("📈")
	rows = append(rows, fmt.Sprintf("%s %s", titleIcon, styles.CardTitleStyle.Render("Daily Consumption")))
	rows = append(rows, "")

	daily := m.historyData.DailyUsage
	if len(daily) == 0 {
		rows = append(rows, styles.HelpStyle.Render("  No daily data available"))
	} else {
		// Extract data for chart
		claudeData := make([]float64, len(daily))
		geminiData := make([]float64, len(daily))
		for i, d := range daily {
			claudeData[i] = d.ClaudeConsumed
			geminiData[i] = d.GeminiConsumed
		}

		// Chart dimensions
		chartWidth := cardWidth - 8
		if chartWidth < 30 {
			chartWidth = 30
		}
		chartHeight := 8

		// Render dual chart
		chart := components.RenderDualLineChart(claudeData, geminiData, chartWidth, chartHeight,
			fmt.Sprintf("Last %d days - Claude (red) vs Gemini (blue)", len(daily)))

		// Indent the chart
		chartLines := strings.Split(chart, "\n")
		for _, line := range chartLines {
			rows = append(rows, "  "+line)
		}

		// Legend
		rows = append(rows, "")
		legend := components.RenderLegend([]components.LegendItem{
			{Label: "Claude", Color: components.ChartClaudeColor},
			{Label: "Gemini", Color: components.ChartGeminiColor},
		})
		rows = append(rows, "  "+legend)
	}

	rows = append(rows, "")

	return styles.CardStyle.Width(cardWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

func (m *Model) renderHourlyHeatmap() string {
	cardWidth := m.width - 6
	if cardWidth < 40 {
		cardWidth = 40
	}

	var rows []string

	titleIcon := lipgloss.NewStyle().Foreground(styles.Primary).Render("🕐")
	rows = append(rows, fmt.Sprintf("%s %s", titleIcon, styles.CardTitleStyle.Render("Hourly Pattern")))
	rows = append(rows, "")

	hourly := m.historyData.HourlyPatterns
	if len(hourly) == 0 {
		rows = append(rows, styles.HelpStyle.Render("  No hourly data available"))
	} else {
		// Convert to float slice for heatmap
		hourlyData := make([]float64, 24)
		for _, h := range hourly {
			if h.Hour >= 0 && h.Hour < 24 {
				hourlyData[h.Hour] = h.AvgConsumed
			}
		}

		// Render heatmap
		heatmap := components.RenderHourlyHeatmap(hourlyData)
		rows = append(rows, "  "+heatmap)

		// Peak hour info
		peakHour, peakVal := m.historyData.GetPeakHour()
		rows = append(rows, fmt.Sprintf("  Peak: %s (avg %.1f%% consumed)",
			lipgloss.NewStyle().Bold(true).Foreground(styles.Primary).Render(fmt.Sprintf("%02d:00-%02d:00", peakHour, (peakHour+1)%24)),
			peakVal,
		))
	}

	rows = append(rows, "")

	return styles.CardStyle.Width(cardWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

func (m *Model) renderWeeklyPattern() string {
	cardWidth := m.width - 6
	if cardWidth < 40 {
		cardWidth = 40
	}

	var rows []string

	titleIcon := lipgloss.NewStyle().Foreground(styles.Primary).Render("📅")
	rows = append(rows, fmt.Sprintf("%s %s", titleIcon, styles.CardTitleStyle.Render("Weekly Pattern")))
	rows = append(rows, "")

	weekly := m.historyData.WeekdayPatterns
	if len(weekly) == 0 {
		rows = append(rows, styles.HelpStyle.Render("  No weekly data available"))
	} else {
		// Convert to float slice for weekly chart
		weeklyData := make([]float64, 7)
		dayNames := make([]string, 7)
		for _, w := range weekly {
			if w.DayOfWeek >= 0 && w.DayOfWeek < 7 {
				weeklyData[w.DayOfWeek] = w.AvgConsumed
				dayNames[w.DayOfWeek] = w.DayName[:3] // "Sun", "Mon", etc.
			}
		}

		// Render weekly pattern
		pattern := components.RenderWeeklyPattern(weeklyData, dayNames)
		rows = append(rows, "  "+pattern)

		// Peak day info
		peakDay, peakVal := m.historyData.GetPeakDay()
		rows = append(rows, fmt.Sprintf("  Peak day: %s (avg %.1f%% consumed)",
			lipgloss.NewStyle().Bold(true).Foreground(styles.Primary).Render(peakDay),
			peakVal,
		))
	}

	rows = append(rows, "")

	return styles.CardStyle.Width(cardWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

// Helper functions

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return notAvailable
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours >= 24 {
		days := hours / 24
		remainingHours := hours % 24
		return fmt.Sprintf("%dd %dh", days, remainingHours)
	}

	return fmt.Sprintf("%dh %dm", hours, minutes)
}

func formatTimeAgo(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "yesterday"
	}
	return fmt.Sprintf("%d days ago", days)
}

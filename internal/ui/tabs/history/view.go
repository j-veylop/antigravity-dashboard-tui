package history

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/components"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/styles"
)

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
	sections = append(sections,
		m.renderHeader(),
		m.renderConsumptionChart(),
		m.renderHourlyHeatmap(),
		m.renderWeeklyPattern(),
	)

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

func (m *Model) renderConsumptionChart() string {
	cardWidth := max(m.width-6, 40)

	var rows []string

	titleIcon := lipgloss.NewStyle().Foreground(styles.Primary).Render("📈")
	rows = append(rows, fmt.Sprintf("%s %s", titleIcon, styles.CardTitleStyle.Render("Daily Consumption")), "")

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
		chartWidth := max(cardWidth-12, 30) // More padding for axis labels
		chartHeight := 8

		// Render dual chart
		chart := components.RenderDualLineChart(claudeData, geminiData, chartWidth, chartHeight,
			fmt.Sprintf("Last %d days - Claude (red) vs Gemini (blue)", len(daily)))

		// Indent the chart
		chartLines := strings.SplitSeq(chart, "\n")
		for line := range chartLines {
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
	cardWidth := max(m.width-6, 40)

	var rows []string

	titleIcon := lipgloss.NewStyle().Foreground(styles.Primary).Render("🕐")
	rows = append(rows,
		fmt.Sprintf("%s %s", titleIcon, styles.CardTitleStyle.Render("Hourly Pattern")),
		"",
	)

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

		chartWidth := max(cardWidth-12, 30)
		chartHeight := 8

		chart := components.RenderLineChart(hourlyData, chartWidth, chartHeight, "Average Hourly Consumption (%)")

		chartLines := strings.SplitSeq(chart, "\n")
		for line := range chartLines {
			rows = append(rows, "  "+line)
		}

		// Peak hour info
		peakHour, peakVal := m.historyData.GetPeakHour()
		rows = append(rows, fmt.Sprintf("  Peak: %s (avg %.1f%% consumed)",
			lipgloss.NewStyle().Bold(true).Foreground(styles.Primary).
				Render(fmt.Sprintf("%02d:00-%02d:00", peakHour, (peakHour+1)%24)),
			peakVal,
		))
	}

	rows = append(rows, "")

	return styles.CardStyle.Width(cardWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

func (m *Model) renderWeeklyPattern() string {
	cardWidth := max(m.width-6, 40)

	var rows []string

	titleIcon := lipgloss.NewStyle().Foreground(styles.Primary).Render("📅")
	rows = append(rows,
		fmt.Sprintf("%s %s", titleIcon, styles.CardTitleStyle.Render("Weekly Pattern")),
		"",
	)

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

		chartWidth := max(cardWidth-12, 30)

		barChart := components.RenderBarChart(weeklyData, dayNames, chartWidth)

		chartLines := strings.SplitSeq(barChart, "\n")
		for line := range chartLines {
			rows = append(rows, "  "+line)
		}

		// Peak day info
		peakDay, peakVal := m.historyData.GetPeakDay()
		rows = append(rows,
			"",
			fmt.Sprintf("  Peak day: %s (avg %.1f%% consumed)",
				lipgloss.NewStyle().Bold(true).Foreground(styles.Primary).Render(peakDay),
				peakVal,
			),
		)
	}

	rows = append(rows, "")

	return styles.CardStyle.Width(cardWidth).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

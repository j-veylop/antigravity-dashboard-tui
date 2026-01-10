// Package components provides reusable UI components for the TUI.
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/guptarohit/asciigraph"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/styles"
)

// ChartColors defines colors for chart elements.
var (
	ChartClaudeColor  = lipgloss.Color("#cc785c")
	ChartGeminiColor  = lipgloss.Color("#4285f4")
	ChartPrimaryColor = lipgloss.Color("#7D56F4")
)

// RenderLineChart creates a single-series ASCII line chart.
func RenderLineChart(data []float64, width, height int, caption string) string {
	if len(data) == 0 {
		return styles.HelpStyle.Render("No data available")
	}

	// Ensure minimum dimensions
	if width < 20 {
		width = 20
	}
	if height < 3 {
		height = 3
	}

	graph := asciigraph.Plot(data,
		asciigraph.Height(height),
		asciigraph.Width(width),
		asciigraph.Caption(caption),
	)

	return graph
}

// RenderDualLineChart creates a two-series chart for Claude vs Gemini.
func RenderDualLineChart(claude, gemini []float64, width, height int, caption string) string {
	if len(claude) == 0 && len(gemini) == 0 {
		return styles.HelpStyle.Render("No data available")
	}

	// Ensure minimum dimensions
	if width < 20 {
		width = 20
	}
	if height < 3 {
		height = 3
	}

	// Normalize lengths - pad shorter array with zeros
	maxLen := len(claude)
	if len(gemini) > maxLen {
		maxLen = len(gemini)
	}

	claudeData := make([]float64, maxLen)
	geminiData := make([]float64, maxLen)
	copy(claudeData, claude)
	copy(geminiData, gemini)

	graph := asciigraph.PlotMany([][]float64{claudeData, geminiData},
		asciigraph.Height(height),
		asciigraph.Width(width),
		asciigraph.Caption(caption),
		asciigraph.SeriesColors(
			asciigraph.Red,
			asciigraph.Blue,
		),
	)

	return graph
}

// RenderBarChart creates a simple horizontal bar chart.
func RenderBarChart(values []float64, labels []string, width int) string {
	if len(values) == 0 {
		return ""
	}

	// Find max value for scaling
	maxVal := 0.0
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	// Find max label length
	maxLabelLen := 0
	for _, l := range labels {
		if len(l) > maxLabelLen {
			maxLabelLen = len(l)
		}
	}

	barWidth := width - maxLabelLen - 10 // Leave room for label and value
	if barWidth < 10 {
		barWidth = 10
	}

	var lines []string
	for i, v := range values {
		label := ""
		if i < len(labels) {
			label = labels[i]
		}

		// Pad label
		paddedLabel := fmt.Sprintf("%*s", maxLabelLen, label)

		// Calculate bar length
		barLen := int((v / maxVal) * float64(barWidth))
		if barLen < 0 {
			barLen = 0
		}

		bar := strings.Repeat("█", barLen)
		valueStr := fmt.Sprintf(" %.1f", v)

		line := paddedLabel + " │" + bar + valueStr
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// HeatmapBlocks are Unicode block characters for heatmaps (low to high intensity).
var HeatmapBlocks = []rune{'░', '▒', '▓', '█'}

// RenderHourlyHeatmap creates a 24-hour usage heatmap.
func RenderHourlyHeatmap(patterns []float64) string {
	if len(patterns) != 24 {
		// Pad or truncate to 24 hours
		padded := make([]float64, 24)
		copy(padded, patterns)
		patterns = padded
	}

	// Find max value for normalization
	maxVal := 0.0
	for _, v := range patterns {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	var result strings.Builder
	result.WriteString("00 ")

	for i, v := range patterns {
		intensity := int((v / maxVal) * float64(len(HeatmapBlocks)-1))
		if intensity >= len(HeatmapBlocks) {
			intensity = len(HeatmapBlocks) - 1
		}
		if intensity < 0 {
			intensity = 0
		}

		// Color based on intensity
		var style lipgloss.Style
		switch intensity {
		case 0:
			style = lipgloss.NewStyle().Foreground(styles.Subtle)
		case 1:
			style = lipgloss.NewStyle().Foreground(styles.Success)
		case 2:
			style = lipgloss.NewStyle().Foreground(styles.Warning)
		case 3:
			style = lipgloss.NewStyle().Foreground(styles.Error)
		}

		result.WriteString(style.Render(string(HeatmapBlocks[intensity])))

		// Add gap at noon for readability
		if i == 11 {
			result.WriteString(" ")
		}
	}

	result.WriteString(" 23")
	return result.String()
}

// RenderWeeklyPattern creates a weekly usage visualization.
func RenderWeeklyPattern(patterns []float64, dayNames []string) string {
	if len(patterns) != 7 {
		padded := make([]float64, 7)
		copy(padded, patterns)
		patterns = padded
	}
	if len(dayNames) != 7 {
		dayNames = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	}

	// Find max for normalization
	maxVal := 0.0
	for _, v := range patterns {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	// Sparkline characters
	sparkChars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	var parts []string
	for i, v := range patterns {
		intensity := int((v / maxVal) * float64(len(sparkChars)-1))
		if intensity >= len(sparkChars) {
			intensity = len(sparkChars) - 1
		}
		if intensity < 0 {
			intensity = 0
		}

		dayLabel := dayNames[i]
		spark := string(sparkChars[intensity])
		parts = append(parts, fmt.Sprintf("%s %s", dayLabel, spark))
	}

	return strings.Join(parts, " ")
}

// RenderSparkline creates a compact inline sparkline chart.
func RenderSparkline(values []float64, width int) string {
	if len(values) == 0 {
		return ""
	}

	sparkChars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// Find max value
	maxVal := 0.0
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	// Sample values to fit width
	var result strings.Builder
	step := float64(len(values)) / float64(width)
	if step < 1 {
		step = 1
	}

	for i := 0; i < width && int(float64(i)*step) < len(values); i++ {
		idx := int(float64(i) * step)
		val := values[idx]
		normalized := int((val / maxVal) * float64(len(sparkChars)-1))
		if normalized >= len(sparkChars) {
			normalized = len(sparkChars) - 1
		}
		if normalized < 0 {
			normalized = 0
		}
		result.WriteRune(sparkChars[normalized])
	}

	return result.String()
}

// RenderColoredSparkline creates a sparkline with gradient coloring.
func RenderColoredSparkline(values []float64, width int) string {
	if len(values) == 0 {
		return ""
	}

	sparkChars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	maxVal := 0.0
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == 0 {
		maxVal = 1
	}

	var result strings.Builder
	step := float64(len(values)) / float64(width)
	if step < 1 {
		step = 1
	}

	for i := 0; i < width && int(float64(i)*step) < len(values); i++ {
		idx := int(float64(i) * step)
		val := values[idx]
		normalized := int((val / maxVal) * float64(len(sparkChars)-1))
		if normalized >= len(sparkChars) {
			normalized = len(sparkChars) - 1
		}
		if normalized < 0 {
			normalized = 0
		}

		// Color based on intensity (high consumption = warning/error colors)
		percent := (val / maxVal) * 100
		style := styles.GetQuotaStyle(100-percent, false) // Invert for consumption coloring
		result.WriteString(style.Render(string(sparkChars[normalized])))
	}

	return result.String()
}

// RenderLegend creates a chart legend.
func RenderLegend(items []LegendItem) string {
	var parts []string
	for _, item := range items {
		colorBox := lipgloss.NewStyle().Foreground(item.Color).Render("■")
		parts = append(parts, fmt.Sprintf("%s %s", colorBox, item.Label))
	}
	return strings.Join(parts, "  ")
}

// LegendItem represents a single legend entry.
type LegendItem struct {
	Label string
	Color lipgloss.Color
}

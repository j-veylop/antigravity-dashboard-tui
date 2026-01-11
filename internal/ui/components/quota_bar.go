// Package components provides reusable UI components.
package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/logger"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/styles"
)

// AnimationTickMsg is a message used for animation ticks.
type AnimationTickMsg time.Time

func animationTick() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return AnimationTickMsg(t)
	})
}

// QuotaBar renders a quota progress bar with label and percentage.
type QuotaBar struct {
	progress       progress.Model
	label          string
	percent        float64
	animationFrame int
	isAnimating    bool
	targetPercent  float64
	currentPercent float64
}

// NewQuotaBar creates a new quota bar with gradient colors.
func NewQuotaBar() QuotaBar {
	p := progress.New(
		progress.WithScaledGradient("#ff6b6b", "#51cf66"),
		progress.WithWidth(30),
		progress.WithoutPercentage(),
	)

	return QuotaBar{
		progress:       p,
		label:          "",
		percent:        0,
		animationFrame: 0,
		isAnimating:    false,
		targetPercent:  0,
		currentPercent: 0,
	}
}

// NewQuotaBarWithWidth creates a quota bar with a specific width.
func NewQuotaBarWithWidth(width int) QuotaBar {
	p := progress.New(
		progress.WithScaledGradient("#ff6b6b", "#51cf66"),
		progress.WithWidth(width),
		progress.WithoutPercentage(),
	)

	return QuotaBar{
		progress:       p,
		label:          "",
		percent:        0,
		animationFrame: 0,
		isAnimating:    false,
		targetPercent:  0,
		currentPercent: 0,
	}
}

// Init initializes the model.
func (q QuotaBar) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (q QuotaBar) Update(msg tea.Msg) (QuotaBar, tea.Cmd) {
	var cmds []tea.Cmd

	if _, ok := msg.(AnimationTickMsg); ok {
		if q.isAnimating {
			q.animationFrame++

			switch {
			case q.currentPercent < q.targetPercent:
				step := (q.targetPercent - q.currentPercent) / 10
				if step < 0.5 {
					step = 0.5
				}
				q.currentPercent += step
				if q.currentPercent > q.targetPercent {
					q.currentPercent = q.targetPercent
				}
				cmds = append(cmds, animationTick())
			case q.currentPercent > q.targetPercent:
				step := (q.currentPercent - q.targetPercent) / 10
				if step < 0.5 {
					step = 0.5
				}
				q.currentPercent -= step
				if q.currentPercent < q.targetPercent {
					q.currentPercent = q.targetPercent
				}
				cmds = append(cmds, animationTick())
			default:
				q.isAnimating = false
			}
		}
	}

	var cmd tea.Cmd
	model, cmd := q.progress.Update(msg)
	q.progress = model.(progress.Model)
	cmds = append(cmds, cmd)

	return q, tea.Batch(cmds...)
}

// SetPercent sets the percentage.
func (q *QuotaBar) SetPercent(percent float64) tea.Cmd {
	q.percent = percent
	q.targetPercent = percent

	if !q.isAnimating {
		q.isAnimating = true
		q.animationFrame = 0
		return tea.Batch(
			q.progress.SetPercent(percent/100),
			animationTick(),
		)
	}

	return q.progress.SetPercent(percent / 100)
}

// SetLabel sets the label.
func (q *QuotaBar) SetLabel(label string) {
	q.label = label
}

// SetWidth sets the width.
func (q *QuotaBar) SetWidth(width int) {
	q.progress.Width = width
}

// View returns the string representation.
func (q QuotaBar) View(percent float64, label string, width int) string {
	barWidth := width - 30
	if barWidth < 10 {
		barWidth = 10
	}
	q.progress.Width = barWidth

	bar := q.progress.ViewAs(percent / 100)

	percentStyle := styles.GetQuotaStyle(percent, false)
	percentStr := percentStyle.Width(6).Align(lipgloss.Right).Render(fmt.Sprintf("%.0f%%", percent))

	labelStyle := styles.ProgressLabelStyle
	labelStr := labelStyle.Width(15).Render(label)

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		labelStr,
		bar,
		" ",
		percentStr,
	)
}

// ViewCompact renders a compact version.
func (q QuotaBar) ViewCompact(percent float64, width int) string {
	barWidth := width - 8
	if barWidth < 5 {
		barWidth = 5
	}
	q.progress.Width = barWidth

	bar := q.progress.ViewAs(percent / 100)
	percentStyle := styles.GetQuotaStyle(percent, false)
	percentStr := percentStyle.Render(fmt.Sprintf("%.0f%%", percent))

	return lipgloss.JoinHorizontal(lipgloss.Center, bar, " ", percentStr)
}

// ViewRateLimited renders a rate-limited state.
func (q QuotaBar) ViewRateLimited(label string, width int) string {
	labelStyle := styles.ProgressLabelStyle
	labelStr := labelStyle.Width(15).Render(label)

	barWidth := width - 30
	if barWidth < 10 {
		barWidth = 10
	}

	emptyBar := lipgloss.NewStyle().
		Foreground(styles.Error).
		Render(strings.Repeat("░", barWidth))

	statusStr := styles.QuotaRateLimitedStyle.
		Width(14).
		Align(lipgloss.Right).
		Render("RATE LIMITED")

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		labelStr,
		emptyBar,
		" ",
		statusStr,
	)
}

// TimeBar renders a time-based progress bar for reset timers.
type TimeBar struct {
	progress progress.Model
}

// NewTimeBar creates a new time bar for visualizing time remaining.
func NewTimeBar() TimeBar {
	p := progress.New(
		progress.WithScaledGradient("#ffd93d", "#6c5ce7"),
		progress.WithWidth(30),
		progress.WithoutPercentage(),
	)

	return TimeBar{
		progress: p,
	}
}

// RenderTimeBarChars renders just the bar characters for a time bar.
func RenderTimeBarChars(percent float64, width int) string {
	if width < 1 {
		return ""
	}

	filled := int(float64(width) * percent)
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	var barChars []string
	for i := 0; i < width; i++ {
		if i < filled {
			t := float64(i) / float64(max(1, width-1))
			color := interpolateColor("#ffd93d", "#6c5ce7", t)
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
			barChars = append(barChars, style.Render("█"))
		} else {
			style := lipgloss.NewStyle().Foreground(styles.Subtle)
			barChars = append(barChars, style.Render("░"))
		}
	}

	return strings.Join(barChars, "")
}

// ViewWithLabel renders the time bar with label padding to align with quota bars.
func (t TimeBar) ViewWithLabel(secondsRemaining int64, label string, width int, tier string) string {
	const hourInSeconds int64 = 3600
	const dayInSeconds int64 = 86400
	const proPeriodSeconds int64 = 5 * 3600

	var period int64
	switch {
	case tier == "PRO":
		period = proPeriodSeconds
	case secondsRemaining <= hourInSeconds:
		period = hourInSeconds
	default:
		period = dayInSeconds
	}

	percent := 1.0
	if period > 0 {
		percent = 1.0 - (float64(secondsRemaining) / float64(period))
		if percent < 0 {
			percent = 0
		}
		if percent > 1 {
			percent = 1
		}
	}

	hours := secondsRemaining / hourInSeconds
	minutes := (secondsRemaining % hourInSeconds) / 60
	timeStr := fmt.Sprintf("%dh %02dm", hours, minutes)

	labelWidth := len(label)
	percentWidth := 8
	barWidth := width - (labelWidth + 1) - percentWidth - 2

	if barWidth < 10 {
		barWidth = 10
	}

	bar := RenderTimeBarChars(percent, barWidth)
	labelPadding := strings.Repeat(" ", labelWidth)

	timeStyle := lipgloss.NewStyle().
		Foreground(styles.TextSecondary).
		Width(percentWidth).
		Align(lipgloss.Right)

	return fmt.Sprintf("%s [%s] %s", labelPadding, bar, timeStyle.Render(timeStr))
}

// ViewFromSecondsWithLabel renders a time bar from seconds remaining with label alignment.
func (t TimeBar) ViewFromSecondsWithLabel(secondsRemaining int64, label string, width int, tier string) string {
	return t.ViewWithLabel(secondsRemaining, label, width, tier)
}

// RenderGradientBar renders just the bar part with gradient colors.
func RenderGradientBar(percent float64, width int) string {
	if width < 1 {
		return ""
	}

	filled := int(float64(width) * percent / 100)
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	var barChars []string
	for i := 0; i < width; i++ {
		if i < filled {
			t := float64(i) / float64(max(1, width-1))
			color := interpolateColor("#ff6b6b", "#51cf66", t)
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
			barChars = append(barChars, style.Render("█"))
		} else {
			style := lipgloss.NewStyle().Foreground(styles.Subtle)
			barChars = append(barChars, style.Render("░"))
		}
	}

	return strings.Join(barChars, "")
}

// SimpleQuotaBar renders a simple ASCII progress bar with gradient colors.
func SimpleQuotaBar(percent float64, label string, width int) string {
	labelWidth := len(label) + 1
	percentWidth := 6
	barWidth := width - labelWidth - percentWidth - 4

	if barWidth < 5 {
		barWidth = 5
	}

	bar := RenderGradientBar(percent, barWidth)

	labelStr := lipgloss.NewStyle().
		Foreground(styles.TextSecondary).
		Render(label)

	percentStr := styles.GetQuotaStyle(percent, false).
		Width(percentWidth).
		Align(lipgloss.Right).
		Render(fmt.Sprintf("%.0f%%", percent))

	return fmt.Sprintf("%s [%s] %s", labelStr, bar, percentStr)
}

func interpolateColor(fromHex, toHex string, t float64) string {
	from := hexToRGB(fromHex)
	to := hexToRGB(toHex)

	r := int(float64(from[0]) + t*(float64(to[0])-float64(from[0])))
	g := int(float64(from[1]) + t*(float64(to[1])-float64(from[1])))
	b := int(float64(from[2]) + t*(float64(to[2])-float64(from[2])))

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func hexToRGB(hex string) [3]int {
	hex = strings.TrimPrefix(hex, "#")
	var r, g, b int
	if _, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b); err != nil {
		logger.Error("failed to parse hex color", "hex", hex, "error", err)
		return [3]int{0, 0, 0}
	}
	return [3]int{r, g, b}
}

// SimpleQuotaBarLoading renders a loading state for the quota bar.
func SimpleQuotaBarLoading(label string, width int, frame int) string {
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

	accentColor := styles.Gemini
	if strings.Contains(strings.ToLower(label), "claude") {
		accentColor = styles.Claude
	} else if strings.Contains(strings.ToLower(label), "total") {
		accentColor = styles.Primary
	}

	bar := renderShimmerBar(barWidth, frame, 120, accentColor, false)

	indent := "    "

	dots := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	dot := dots[(frame/2)%len(dots)]

	loadingStr := lipgloss.NewStyle().
		Width(percentWidth).
		Align(lipgloss.Right).
		Foreground(accentColor).
		Render(dot)

	rateStr := lipgloss.NewStyle().Width(rateWidth).Render("")
	badgeStr := lipgloss.NewStyle().Width(badgeWidth).Render("")

	return lipgloss.JoinHorizontal(lipgloss.Left,
		indent,
		bar,
		" ",
		loadingStr,
		" ",
		rateStr,
		" ",
		badgeStr,
	)
}

// SimpleTimeBarLoading renders a loading state for the time bar.
func SimpleTimeBarLoading(label string, width int, frame int) string {
	const (
		indentWidth  = 4
		percentWidth = 6
		rateWidth    = 10
		badgeWidth   = 10
	)

	timeWidth := percentWidth
	depleteWidth := rateWidth + badgeWidth

	rightSideWidth := percentWidth + rateWidth + badgeWidth
	barWidth := width - indentWidth - rightSideWidth - 4
	if barWidth < 10 {
		barWidth = 10
	}

	accentColor := styles.Gemini
	cycle := 100
	if strings.Contains(strings.ToLower(label), "claude") {
		accentColor = styles.Claude
		cycle = 80
	} else if strings.Contains(strings.ToLower(label), "total") {
		accentColor = styles.Primary
	}

	bar := renderShimmerBar(barWidth, frame, cycle, accentColor, true)

	indent := "    "

	dots := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	dot := dots[(frame/2)%len(dots)]

	loadingStr := lipgloss.NewStyle().
		Width(timeWidth).
		Align(lipgloss.Right).
		Foreground(accentColor).
		Render(dot)

	depleteStr := lipgloss.NewStyle().Width(depleteWidth).Render("")

	return lipgloss.JoinHorizontal(lipgloss.Left,
		indent,
		bar,
		" ",
		loadingStr,
		" ",
		depleteStr,
	)
}

func renderShimmerBar(width int, frame, cycle int, accentColor lipgloss.Color, reverse bool) string {
	t := float64(frame%cycle) / float64(cycle)
	var p float64
	if t < 0.5 {
		p = t * 2
	} else {
		p = (1 - t) * 2
	}
	eased := p * p * (3 - 2*p)

	var shimmerPos int
	if reverse {
		shimmerPos = int((1.0 - eased) * float64(width))
	} else {
		shimmerPos = int(eased * float64(width))
	}

	var barChars []string
	for i := 0; i < width; i++ {
		dist := shimmerPos - i
		if dist < 0 {
			dist = -dist
		}

		var char string
		var style lipgloss.Style

		switch {
		case dist < 3:
			char = "▓"
			style = lipgloss.NewStyle().Foreground(accentColor)
		case dist < 5:
			char = "▒"
			style = lipgloss.NewStyle().Foreground(styles.TextSecondary)
		default:
			char = "░"
			style = lipgloss.NewStyle().Foreground(styles.BgLight)
		}

		barChars = append(barChars, style.Render(char))
	}

	return strings.Join(barChars, "")
}

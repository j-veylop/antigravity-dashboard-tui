// Package styles defines the visual styling for the application.
package styles

import "github.com/charmbracelet/lipgloss"

// Color definitions for the Antigravity theme.
var (
	// Primary colors
	Primary   = lipgloss.Color("205") // Pink
	Secondary = lipgloss.Color("63")  // Purple
	Subtle    = lipgloss.Color("240") // Gray

	// Brand colors
	Claude = lipgloss.Color("208") // Orange
	Gemini = lipgloss.Color("39")  // Blue

	// Status colors
	Success = lipgloss.Color("42")  // Green
	Error   = lipgloss.Color("196") // Red
	Warning = lipgloss.Color("220") // Yellow
	Info    = lipgloss.Color("39")  // Blue

	// Background colors
	BgDark   = lipgloss.Color("235")
	BgLight  = lipgloss.Color("237")
	BgAccent = lipgloss.Color("236")

	// Text colors
	TextPrimary   = lipgloss.Color("252")
	TextSecondary = lipgloss.Color("245")
	TextMuted     = lipgloss.Color("240")

	// ToastStyle for floating notifications.
	ToastStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(0, 1).
			MarginBottom(1)
)

// TitleStyle is used for main headings.
var TitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(Primary).
	MarginBottom(1)

// SubTitleStyle is used for section headings.
var SubTitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(Secondary).
	MarginBottom(1)

// DocStyle provides consistent document margins.
var DocStyle = lipgloss.NewStyle().
	Margin(1, 2).
	Padding(0, 1)

// ActiveTabStyle styles the currently selected tab.
var ActiveTabStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("229")).
	Background(Primary).
	Padding(0, 2).
	MarginRight(1)

// InactiveTabStyle styles non-selected tabs.
var InactiveTabStyle = lipgloss.NewStyle().
	Foreground(TextSecondary).
	Background(BgLight).
	Padding(0, 2).
	MarginRight(1)

// TabNumberStyle styles the tab number indicator.
var TabNumberStyle = lipgloss.NewStyle().
	Foreground(Subtle).
	MarginRight(0)

// CardStyle creates a bordered card container.
var CardStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Subtle).
	Padding(1, 2).
	MarginBottom(1)

// CardTitleStyle styles card headers.
var CardTitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(Primary).
	MarginBottom(1)

// FocusedStyle is used for focused input elements.
var FocusedStyle = lipgloss.NewStyle().
	Foreground(Primary).
	Bold(true)

// BlurredStyle is used for unfocused input elements.
var BlurredStyle = lipgloss.NewStyle().
	Foreground(TextMuted)

// FocusedBorderStyle creates a focused border.
var FocusedBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Primary).
	Padding(0, 1)

// BlurredBorderStyle creates an unfocused border.
var BlurredBorderStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Subtle).
	Padding(0, 1)

// NotificationBaseStyle is the base for all notification types.
var NotificationBaseStyle = lipgloss.NewStyle().
	Padding(0, 2).
	MarginBottom(1).
	Border(lipgloss.RoundedBorder())

// NotificationSuccessStyle for success notifications.
var NotificationSuccessStyle = NotificationBaseStyle.
	BorderForeground(Success).
	Foreground(Success)

// NotificationErrorStyle for error notifications.
var NotificationErrorStyle = NotificationBaseStyle.
	BorderForeground(Error).
	Foreground(Error)

// NotificationWarningStyle for warning notifications.
var NotificationWarningStyle = NotificationBaseStyle.
	BorderForeground(Warning).
	Foreground(Warning)

// NotificationInfoStyle for info notifications.
var NotificationInfoStyle = NotificationBaseStyle.
	BorderForeground(Info).
	Foreground(Info)

// ProgressBarStyle styles the progress bar container.
var ProgressBarStyle = lipgloss.NewStyle().
	PaddingLeft(1).
	PaddingRight(1)

// ProgressLabelStyle styles progress bar labels.
var ProgressLabelStyle = lipgloss.NewStyle().
	Foreground(TextSecondary).
	Width(20)

// ProgressPercentStyle styles the percentage display.
var ProgressPercentStyle = lipgloss.NewStyle().
	Foreground(TextPrimary).
	Width(6).
	Align(lipgloss.Right)

// HelpStyle is the base style for help text.
var HelpStyle = lipgloss.NewStyle().
	Foreground(TextMuted)

// HelpKeyStyle styles keyboard shortcut keys.
var HelpKeyStyle = lipgloss.NewStyle().
	Foreground(Primary).
	Bold(true)

// HelpDescStyle styles help descriptions.
var HelpDescStyle = lipgloss.NewStyle().
	Foreground(TextSecondary)

// HelpSeparatorStyle styles separators in help text.
var HelpSeparatorStyle = lipgloss.NewStyle().
	Foreground(Subtle)

// HelpPanelStyle creates the help overlay panel.
var HelpPanelStyle = lipgloss.NewStyle().
	Border(lipgloss.DoubleBorder()).
	BorderForeground(Primary).
	Padding(1, 3).
	Background(BgDark)

// ListItemStyle styles list items.
var ListItemStyle = lipgloss.NewStyle().
	PaddingLeft(2)

// SelectedListItemStyle styles selected list items.
var SelectedListItemStyle = lipgloss.NewStyle().
	PaddingLeft(1).
	Foreground(Primary).
	Bold(true).
	SetString("> ")

// TableHeaderStyle styles table headers.
var TableHeaderStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(Primary).
	BorderStyle(lipgloss.NormalBorder()).
	BorderBottom(true).
	BorderForeground(Subtle)

// TableCellStyle styles table cells.
var TableCellStyle = lipgloss.NewStyle().
	Padding(0, 1)

// TableSelectedStyle styles selected table rows.
var TableSelectedStyle = lipgloss.NewStyle().
	Background(BgAccent).
	Foreground(TextPrimary).
	Bold(true)

// TierProStyle styles PRO tier indicators.
var TierProStyle = lipgloss.NewStyle().
	Foreground(Success).
	Bold(true)

// TierFreeStyle styles FREE tier indicators.
var TierFreeStyle = lipgloss.NewStyle().
	Foreground(Warning)

// TierUnknownStyle styles UNKNOWN tier indicators.
var TierUnknownStyle = lipgloss.NewStyle().
	Foreground(Subtle)

// QuotaHighStyle for high quota percentages (>50%).
var QuotaHighStyle = lipgloss.NewStyle().
	Foreground(Success)

// QuotaMediumStyle for medium quota percentages (20-50%).
var QuotaMediumStyle = lipgloss.NewStyle().
	Foreground(Warning)

// QuotaLowStyle for low quota percentages (<20%).
var QuotaLowStyle = lipgloss.NewStyle().
	Foreground(Error)

// QuotaRateLimitedStyle for rate-limited accounts.
var QuotaRateLimitedStyle = lipgloss.NewStyle().
	Foreground(Error).
	Bold(true).
	Italic(true)

// ErrorTextStyle for error messages.
var ErrorTextStyle = lipgloss.NewStyle().
	Foreground(Error)

// SuccessTextStyle for success messages.
var SuccessTextStyle = lipgloss.NewStyle().
	Foreground(Success)

// WarningTextStyle for warning messages.
var WarningTextStyle = lipgloss.NewStyle().
	Foreground(Warning)

// InfoTextStyle for info messages.
var InfoTextStyle = lipgloss.NewStyle().
	Foreground(Info)

// ModalOverlayStyle creates a modal overlay background.
var ModalOverlayStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("0"))

// ModalContentStyle styles modal content.
var ModalContentStyle = lipgloss.NewStyle().
	Border(lipgloss.DoubleBorder()).
	BorderForeground(Primary).
	Padding(1, 2).
	Background(BgDark)

// ButtonStyle is the base button style.
var ButtonStyle = lipgloss.NewStyle().
	Padding(0, 2).
	MarginRight(1)

// ButtonActiveStyle styles active/focused buttons.
var ButtonActiveStyle = ButtonStyle.
	Background(Primary).
	Foreground(lipgloss.Color("229")).
	Bold(true)

var ButtonInactiveStyle = ButtonStyle.
	Background(BgLight).
	Foreground(TextSecondary)

var ProjectionSafeStyle = lipgloss.NewStyle().
	Foreground(Success)

var ProjectionWarningStyle = lipgloss.NewStyle().
	Foreground(Warning).
	Bold(true)

var ProjectionCriticalStyle = lipgloss.NewStyle().
	Foreground(Error).
	Bold(true)

var ProjectionUnknownStyle = lipgloss.NewStyle().
	Foreground(Subtle)

var ProjectionCardStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Secondary).
	Padding(1, 2).
	MarginBottom(1)

// GetQuotaStyle returns the appropriate style based on quota percentage.
func GetQuotaStyle(percent float64, isRateLimited bool) lipgloss.Style {
	if isRateLimited {
		return QuotaRateLimitedStyle
	}
	switch {
	case percent > 50:
		return QuotaHighStyle
	case percent > 20:
		return QuotaMediumStyle
	default:
		return QuotaLowStyle
	}
}

// GetTierStyle returns the appropriate style for an account tier.
func GetTierStyle(tier string) lipgloss.Style {
	switch tier {
	case "PRO":
		return TierProStyle
	case "FREE":
		return TierFreeStyle
	default:
		return TierUnknownStyle
	}
}

// CenterHorizontal centers content horizontally within a given width.
func CenterHorizontal(content string, width int) string {
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(content)
}

// CenterBoth centers content both horizontally and vertically.
func CenterBoth(content string, width, height int) string {
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(content)
}

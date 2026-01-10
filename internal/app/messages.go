package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/services"
)

// TickMsg is sent periodically to trigger state refresh.
type TickMsg struct {
	Time time.Time
}

// WindowSizeMsg is sent when the terminal window is resized.
type WindowSizeMsg struct {
	Width  int
	Height int
}

// StartLoadingMsg signals that a resource is starting to load.
type StartLoadingMsg struct {
	Resource string
}

// StopLoadingMsg signals that a resource has finished loading.
type StopLoadingMsg struct {
	Resource string
}

// InitialLoadCompleteMsg signals that initial data loading is complete.
type InitialLoadCompleteMsg struct{}

// AccountsLoadedMsg contains loaded account data.
type AccountsLoadedMsg struct {
	Accounts []models.AccountWithQuota
	Stats    services.StatsEvent
}

// QuotaRefreshedMsg contains refreshed quota data for an account.
type QuotaRefreshedMsg struct {
	Email     string
	QuotaInfo *models.QuotaInfo
	Error     error
}

// StatsLoadedMsg contains loaded statistics.
type StatsLoadedMsg struct {
	Stats services.StatsEvent
}

// SwitchAccountMsg requests switching to a different active account.
type SwitchAccountMsg struct {
	Email string
}

// SwitchAccountResultMsg contains the result of an account switch.
type SwitchAccountResultMsg struct {
	Email   string
	Success bool
	Error   error
}

// DeleteAccountMsg requests deletion of an account.
type DeleteAccountMsg struct {
	Email string
}

// DeleteAccountResultMsg contains the result of an account deletion.
type DeleteAccountResultMsg struct {
	Email   string
	Success bool
	Error   error
}

// RefreshMsg requests a refresh of data.
type RefreshMsg struct {
	Resource string // "all", "accounts", "quota", "stats"
}

// RefreshQuotaForAccountMsg requests quota refresh for a specific account.
type RefreshQuotaForAccountMsg struct {
	Email string
}

// AddNotificationMsg requests adding a new notification.
type AddNotificationMsg struct {
	Type     NotificationType
	Message  string
	Duration time.Duration
}

// RemoveNotificationMsg requests removal of a notification.
type RemoveNotificationMsg struct {
	ID string
}

// ClearNotificationsMsg requests clearing all notifications.
type ClearNotificationsMsg struct{}

// NotificationAddedMsg confirms a notification was added.
type NotificationAddedMsg struct {
	ID string
}

// ServiceEventMsg wraps a service event from the service manager.
type ServiceEventMsg struct {
	Event services.ServiceEvent
}

// AccountsChangedEventMsg wraps an accounts changed event.
type AccountsChangedEventMsg struct {
	Event services.AccountsChangedEvent
}

type QuotaUpdatedEventMsg struct {
	Event services.QuotaUpdatedEvent
}

type ProjectionUpdatedMsg struct {
	Email      string
	Projection *models.AccountProjection
}

// ErrorEventMsg wraps an error event from services.
type ErrorEventMsg struct {
	Event services.ErrorEvent
}

// ErrorMsg represents a general error.
type ErrorMsg struct {
	Error   error
	Context string
}

// QuitMsg requests the application to quit.
type QuitMsg struct{}

// TabSwitchMsg requests switching to a specific tab.
type TabSwitchMsg struct {
	Tab TabID
}

// ToggleHelpMsg toggles the help display.
type ToggleHelpMsg struct{}

// FocusNextMsg focuses the next focusable element.
type FocusNextMsg struct{}

// FocusPrevMsg focuses the previous focusable element.
type FocusPrevMsg struct{}

// SelectItemMsg selects the current item in a list.
type SelectItemMsg struct{}

// CopyToClipboardMsg requests copying text to clipboard.
type CopyToClipboardMsg struct {
	Text string
}

// ClipboardResultMsg contains the result of a clipboard operation.
type ClipboardResultMsg struct {
	Success bool
	Error   error
}

// OpenURLMsg requests opening a URL in the browser.
type OpenURLMsg struct {
	URL string
}

// OpenURLResultMsg contains the result of opening a URL.
type OpenURLResultMsg struct {
	URL     string
	Success bool
	Error   error
}

// ScrollUpMsg requests scrolling up.
type ScrollUpMsg struct {
	Lines int
}

// ScrollDownMsg requests scrolling down.
type ScrollDownMsg struct {
	Lines int
}

// PageUpMsg requests scrolling up by one page.
type PageUpMsg struct{}

// PageDownMsg requests scrolling down by one page.
type PageDownMsg struct{}

// GoToTopMsg requests scrolling to the top.
type GoToTopMsg struct{}

// GoToBottomMsg requests scrolling to the bottom.
type GoToBottomMsg struct{}

// FilterMsg sets a filter on the current view.
type FilterMsg struct {
	Query string
}

// SortMsg changes the sort order of the current view.
type SortMsg struct {
	Field     string
	Ascending bool
}

// ExportMsg requests exporting data.
type ExportMsg struct {
	Format string // "json", "csv", etc.
	Path   string
}

// ExportResultMsg contains the result of an export operation.
type ExportResultMsg struct {
	Path    string
	Success bool
	Error   error
}

// SettingsChangedMsg signals that settings have changed.
type SettingsChangedMsg struct {
	Key   string
	Value any
}

// ThemeChangedMsg signals that the theme has changed.
type ThemeChangedMsg struct {
	Theme string
}

// SubscriptionEventMsg is the callback wrapper for service subscription.
type SubscriptionEventMsg struct {
	Channel chan services.ServiceEvent
}

// ClearExpiredNotificationsMsg triggers clearing of expired notifications.
type ClearExpiredNotificationsMsg struct{}

// DelayedMsg wraps a message to be sent after a delay.
type DelayedMsg struct {
	Delay time.Duration
	Msg   tea.Msg
}

// BatchMsg contains multiple messages to be processed.
type BatchMsg struct {
	Messages []tea.Msg
}

// SelectedAccountChangedMsg signals that the selected account in the UI has changed.
type SelectedAccountChangedMsg struct {
	Index int
	Email string
}

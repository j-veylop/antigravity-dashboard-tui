package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/services"
)

const (
	// DefaultTickInterval is the default interval between ticks.
	DefaultTickInterval = 2 * time.Second

	// DefaultNotificationDuration is the default duration for notifications.
	DefaultNotificationDuration = 5 * time.Second

	// QuickNotificationDuration is for brief notifications.
	QuickNotificationDuration = 3 * time.Second

	// LongNotificationDuration is for important notifications.
	LongNotificationDuration = 10 * time.Second
)

// tickCmd returns a command that sends a TickMsg after the specified interval.
func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return TickMsg{Time: t}
	})
}

// defaultTickCmd returns a command that sends a TickMsg after the default interval.
func defaultTickCmd() tea.Cmd {
	return tickCmd(DefaultTickInterval)
}

// loadInitialData returns a command that loads all initial data.
func loadInitialData(mgr *services.Manager) tea.Cmd {
	return tea.Batch(
		loadAccountsCmd(mgr),
		loadStatsCmd(mgr),
	)
}

// loadAccountsCmd returns a command that loads accounts with quota.
func loadAccountsCmd(mgr *services.Manager) tea.Cmd {
	return func() tea.Msg {
		accounts := mgr.GetAccountsWithQuota()
		stats := mgr.GetStats()

		return AccountsLoadedMsg{
			Accounts: accounts,
			Stats:    stats,
		}
	}
}

// loadStatsCmd returns a command that loads statistics.
func loadStatsCmd(mgr *services.Manager) tea.Cmd {
	return func() tea.Msg {
		stats := mgr.GetStats()
		return StatsLoadedMsg{Stats: stats}
	}
}

// refreshQuotaCmd returns a command that refreshes quota for a specific account.
func refreshQuotaCmd(mgr *services.Manager, email string) tea.Cmd {
	return func() tea.Msg {
		quotaInfo, err := mgr.RefreshQuotaForAccount(email)
		return QuotaRefreshedMsg{
			Email:     email,
			QuotaInfo: quotaInfo,
			Error:     err,
		}
	}
}

// refreshAllQuotaCmd returns a command that refreshes quota for all accounts.
func refreshAllQuotaCmd(mgr *services.Manager) tea.Cmd {
	return func() tea.Msg {
		mgr.RefreshQuota()
		accounts := mgr.GetAccountsWithQuota()
		stats := mgr.GetStats()
		return AccountsLoadedMsg{
			Accounts: accounts,
			Stats:    stats,
		}
	}
}

// subscribeToServicesCmd returns a command that subscribes to service events.
func subscribeToServicesCmd(mgr *services.Manager) tea.Cmd {
	ch, _ := mgr.Subscribe()
	return func() tea.Msg {
		return SubscriptionEventMsg{Channel: ch}
	}
}

// waitForServiceEventCmd returns a command that waits for the next service event.
func waitForServiceEventCmd(ch <-chan services.ServiceEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-ch
		if !ok {
			return nil
		}
		return ServiceEventMsg{Event: event}
	}
}

// WaitForServiceEvent is the public version for use in models.
func WaitForServiceEvent(ch <-chan services.ServiceEvent) tea.Cmd {
	return services.WaitForEvent(ch)
}

// clearNotificationCmd returns a command that removes a notification after a delay.
func clearNotificationCmd(id string, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(_ time.Time) tea.Msg {
		return RemoveNotificationMsg{ID: id}
	})
}

// switchAccountCmd returns a command that switches the active account.
func switchAccountCmd(mgr *services.Manager, email string) tea.Cmd {
	return func() tea.Msg {
		err := mgr.Accounts().SetActiveAccount(email)
		return SwitchAccountResultMsg{
			Email:   email,
			Success: err == nil,
			Error:   err,
		}
	}
}

// deleteAccountCmd returns a command that deletes an account.
func deleteAccountCmd(mgr *services.Manager, email string) tea.Cmd {
	return func() tea.Msg {
		err := mgr.Accounts().DeleteAccount(email)
		return DeleteAccountResultMsg{
			Email:   email,
			Success: err == nil,
			Error:   err,
		}
	}
}

// notifySuccessCmd returns a command that adds a success notification.
func notifySuccessCmd(message string) tea.Cmd {
	return func() tea.Msg {
		return AddNotificationMsg{
			Type:     NotificationSuccess,
			Message:  message,
			Duration: DefaultNotificationDuration,
		}
	}
}

// notifyErrorCmd returns a command that adds an error notification.
func notifyErrorCmd(message string) tea.Cmd {
	return func() tea.Msg {
		return AddNotificationMsg{
			Type:     NotificationError,
			Message:  message,
			Duration: LongNotificationDuration,
		}
	}
}

// notifyWarningCmd returns a command that adds a warning notification.
func notifyWarningCmd(message string) tea.Cmd {
	return func() tea.Msg {
		return AddNotificationMsg{
			Type:     NotificationWarning,
			Message:  message,
			Duration: DefaultNotificationDuration,
		}
	}
}

// notifyInfoCmd returns a command that adds an info notification.
func notifyInfoCmd(message string) tea.Cmd {
	return func() tea.Msg {
		return AddNotificationMsg{
			Type:     NotificationInfo,
			Message:  message,
			Duration: QuickNotificationDuration,
		}
	}
}

// delayedCmd returns a command that sends a message after a delay.
func delayedCmd(delay time.Duration, msg tea.Msg) tea.Cmd {
	return tea.Tick(delay, func(_ time.Time) tea.Msg {
		return msg
	})
}

// batchCmds combines multiple commands into one.
func batchCmds(cmds ...tea.Cmd) tea.Cmd {
	return tea.Batch(cmds...)
}

// quitCmd returns a command that quits the application.
func quitCmd() tea.Cmd {
	return tea.Quit
}

// Commands provides a public interface to the command functions.
type Commands struct {
	manager *services.Manager
}

// NewCommands creates a new Commands instance.
func NewCommands(mgr *services.Manager) *Commands {
	return &Commands{manager: mgr}
}

// Tick returns a tick command with the specified interval.
func (c *Commands) Tick(interval time.Duration) tea.Cmd {
	return tickCmd(interval)
}

// DefaultTick returns a tick command with the default interval.
func (c *Commands) DefaultTick() tea.Cmd {
	return defaultTickCmd()
}

// LoadInitialData returns a command that loads all initial data.
func (c *Commands) LoadInitialData() tea.Cmd {
	return loadInitialData(c.manager)
}

// LoadAccounts returns a command that loads accounts.
func (c *Commands) LoadAccounts() tea.Cmd {
	return loadAccountsCmd(c.manager)
}

// LoadStats returns a command that loads statistics.
func (c *Commands) LoadStats() tea.Cmd {
	return loadStatsCmd(c.manager)
}

// RefreshQuota returns a command that refreshes quota for an account.
func (c *Commands) RefreshQuota(email string) tea.Cmd {
	return refreshQuotaCmd(c.manager, email)
}

// RefreshAllQuota returns a command that refreshes quota for all accounts.
func (c *Commands) RefreshAllQuota() tea.Cmd {
	return refreshAllQuotaCmd(c.manager)
}

// SubscribeToServices returns a command that subscribes to service events.
func (c *Commands) SubscribeToServices() tea.Cmd {
	return subscribeToServicesCmd(c.manager)
}

// SwitchAccount returns a command that switches the active account.
func (c *Commands) SwitchAccount(email string) tea.Cmd {
	return switchAccountCmd(c.manager, email)
}

// DeleteAccount returns a command that deletes an account.
func (c *Commands) DeleteAccount(email string) tea.Cmd {
	return deleteAccountCmd(c.manager, email)
}

// NotifySuccess returns a command that adds a success notification.
func (c *Commands) NotifySuccess(message string) tea.Cmd {
	return notifySuccessCmd(message)
}

// NotifyError returns a command that adds an error notification.
func (c *Commands) NotifyError(message string) tea.Cmd {
	return notifyErrorCmd(message)
}

// NotifyWarning returns a command that adds a warning notification.
func (c *Commands) NotifyWarning(message string) tea.Cmd {
	return notifyWarningCmd(message)
}

// NotifyInfo returns a command that adds an info notification.
func (c *Commands) NotifyInfo(message string) tea.Cmd {
	return notifyInfoCmd(message)
}

// ClearNotification returns a command that removes a notification after a delay.
func (c *Commands) ClearNotification(id string, delay time.Duration) tea.Cmd {
	return clearNotificationCmd(id, delay)
}

// Quit returns a command that quits the application.
func (c *Commands) Quit() tea.Cmd {
	return quitCmd()
}

// Delayed returns a command that sends a message after a delay.
func (c *Commands) Delayed(delay time.Duration, msg tea.Msg) tea.Cmd {
	return delayedCmd(delay, msg)
}

// Batch combines multiple commands into one.
func (c *Commands) Batch(cmds ...tea.Cmd) tea.Cmd {
	return batchCmds(cmds...)
}

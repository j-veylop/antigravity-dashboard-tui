// Package services provides service orchestration for the TUI.
package services

import (
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gen2brain/beeep"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/config"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/db"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/services/accounts"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/services/projection"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/services/quota"
)

type (
	// AccountsChangedEvent is emitted when the accounts list changes.
	AccountsChangedEvent struct {
		Accounts      []models.Account
		ActiveAccount *models.Account
	}

	// QuotaUpdatedEvent is emitted when quota information is updated for an account.
	QuotaUpdatedEvent struct {
		AccountEmail string
		QuotaInfo    *models.QuotaInfo
	}

	// ProjectionUpdatedEvent is emitted when projections are updated for an account.
	ProjectionUpdatedEvent struct {
		Email      string
		Projection *models.AccountProjection
	}

	// ErrorEvent is emitted when an error occurs in any service.
	ErrorEvent struct {
		Service string
		Error   error
	}

	// StatsEvent is emitted when global statistics change.
	StatsEvent struct {
		AccountCount   int
		QuotaCached    int
		TotalRemaining int64
		TotalLimit     int64
	}
)

// ServiceEvent is the interface implemented by all service events.
type ServiceEvent interface {
	isServiceEvent()
}

func (AccountsChangedEvent) isServiceEvent()   {}
func (QuotaUpdatedEvent) isServiceEvent()      {}
func (ProjectionUpdatedEvent) isServiceEvent() {}
func (ErrorEvent) isServiceEvent()             {}
func (StatsEvent) isServiceEvent()             {}

// Manager orchestrates services and event routing.
type Manager struct {
	mu             sync.RWMutex
	accounts       *accounts.Service
	quota          *quota.Service
	projection     *projection.Service
	database       *db.DB
	eventChan      chan ServiceEvent
	stopChan       chan struct{}
	subscribers    []chan<- ServiceEvent
	previousQuotas map[string]*models.QuotaInfo
}

// NewManager creates a new service manager.
func NewManager(cfg *config.Config) (*Manager, error) {
	m := &Manager{
		eventChan:      make(chan ServiceEvent, 100),
		stopChan:       make(chan struct{}),
		previousQuotas: make(map[string]*models.QuotaInfo),
	}

	var err error
	m.accounts, err = accounts.New(cfg.AccountsPath)
	if err != nil {
		return nil, err
	}

	m.database, err = db.New(cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	m.projection = projection.New(m.database)

	quotaConfig := quota.DefaultConfig()
	quotaConfig.ClientID = cfg.GoogleClientID
	quotaConfig.ClientSecret = cfg.GoogleClientSecret
	quotaConfig.PollInterval = cfg.QuotaRefreshInterval

	m.quota = quota.New(m.accounts, quotaConfig)

	go m.routeEvents()

	return m, nil
}

// routeEvents routes events from individual services to subscribers.
func (m *Manager) routeEvents() {
	for {
		select {
		case event := <-m.accounts.Events():
			m.handleAccountEvent(event)

		case event := <-m.quota.Events():
			m.handleQuotaEvent(event)

		case <-m.stopChan:
			return
		}
	}
}

// handleAccountEvent converts and broadcasts account events.
func (m *Manager) handleAccountEvent(event accounts.Event) {
	switch event.Type {
	case accounts.EventAccountsLoaded, accounts.EventAccountsChanged,
		accounts.EventAccountAdded, accounts.EventAccountUpdated,
		accounts.EventAccountDeleted, accounts.EventActiveAccountChanged:

		m.broadcast(AccountsChangedEvent{
			Accounts:      m.accounts.GetAccounts(),
			ActiveAccount: m.accounts.GetActiveAccount(),
		})

	case accounts.EventError:
		m.broadcast(ErrorEvent{
			Service: "accounts",
			Error:   event.Error,
		})
	}
}

func (m *Manager) handleQuotaEvent(event quota.Event) {
	switch event.Type {
	case quota.EventQuotaUpdated:
		m.broadcast(QuotaUpdatedEvent{
			AccountEmail: event.AccountEmail,
			QuotaInfo:    event.QuotaInfo,
		})

		if event.QuotaInfo != nil {
			m.checkNotifications(event.AccountEmail, event.QuotaInfo)
		}

		if m.projection != nil && event.QuotaInfo != nil {
			go m.updateProjection(event.AccountEmail, event.QuotaInfo)
		}

	case quota.EventQuotaError, quota.EventTokenError:
		m.broadcast(ErrorEvent{
			Service: "quota",
			Error:   event.Error,
		})
	}
}

func (m *Manager) checkNotifications(email string, newQuota *models.QuotaInfo) {
	oldQuota, exists := m.previousQuotas[email]
	m.previousQuotas[email] = newQuota

	if !exists {
		return
	}

	// Check for critical quota (< 5%)
	// Only notify if we crossed the threshold downwards
	if newQuota.TotalLimit > 0 {
		newPercent := float64(newQuota.TotalRemaining) / float64(newQuota.TotalLimit) * 100
		oldPercent := float64(oldQuota.TotalRemaining) / float64(oldQuota.TotalLimit) * 100

		if newPercent < 5.0 && oldPercent >= 5.0 {
			title := fmt.Sprintf("Critical Quota: %s", email)
			body := fmt.Sprintf("Remaining quota is below 5%% (%.1f%%)", newPercent)
			_ = beeep.Notify(title, body, "")
		}
	}

	// Check for reset (significant increase)
	// If remaining increases by more than 20% of limit
	if newQuota.TotalRemaining > oldQuota.TotalRemaining {
		diff := newQuota.TotalRemaining - oldQuota.TotalRemaining
		if newQuota.TotalLimit > 0 {
			percentDiff := float64(diff) / float64(newQuota.TotalLimit) * 100
			if percentDiff > 20.0 {
				title := fmt.Sprintf("Quota Reset: %s", email)
				body := "Your quota has been refreshed."
				_ = beeep.Notify(title, body, "")
			}
		}
	}
}

func (m *Manager) updateProjection(email string, quotaInfo *models.QuotaInfo) {
	claudePercent := 0.0
	geminiPercent := 0.0
	var claudeReset, geminiReset time.Time
	tier := quotaInfo.SubscriptionTier

	for _, mq := range quotaInfo.ModelQuotas {
		percent := 0.0
		if mq.Limit > 0 {
			percent = float64(mq.Remaining) / float64(mq.Limit) * 100
		}
		switch mq.ModelFamily {
		case "claude":
			if claudePercent == 0 || percent < claudePercent {
				claudePercent = percent
				claudeReset = mq.ResetTime
			}
		case "gemini":
			if geminiPercent == 0 || percent < geminiPercent {
				geminiPercent = percent
				geminiReset = mq.ResetTime
			}
		}
	}

	sessionID := m.projection.GetOrCreateSessionID(email, claudeReset)

	_ = m.projection.AggregateSnapshot(email, claudePercent, geminiPercent, tier, sessionID)

	proj, _ := m.projection.CalculateProjections(email, claudePercent, geminiPercent, claudeReset, geminiReset)
	if proj != nil {
		m.broadcast(ProjectionUpdatedEvent{
			Email:      email,
			Projection: proj,
		})
	}
}

// broadcast sends an event to all subscribers.
func (m *Manager) broadcast(event ServiceEvent) {
	// Send to main event channel
	select {
	case m.eventChan <- event:
	default:
	}

	// Send to subscribers
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, sub := range m.subscribers {
		select {
		case sub <- event:
		default:
			// Subscriber channel full, skip
		}
	}
}

// Subscribe creates a channel for receiving service events.
// Returns a tea.Cmd that can be used in Bubble Tea's Init or Update.
func (m *Manager) Subscribe() (chan ServiceEvent, tea.Cmd) {
	ch := make(chan ServiceEvent, 50)

	m.mu.Lock()
	m.subscribers = append(m.subscribers, ch)
	m.mu.Unlock()

	return ch, waitForEvent(ch)
}

// waitForEvent returns a tea.Cmd that waits for the next event.
func waitForEvent(ch <-chan ServiceEvent) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// WaitForEvent returns a tea.Cmd for the next event on a channel.
func WaitForEvent(ch <-chan ServiceEvent) tea.Cmd {
	return waitForEvent(ch)
}

// Unsubscribe removes a subscriber channel.
func (m *Manager) Unsubscribe(ch chan ServiceEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, sub := range m.subscribers {
		if sub == ch {
			m.subscribers = append(m.subscribers[:i], m.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// GetAccountsWithQuota returns all accounts with their quota information.
func (m *Manager) GetAccountsWithQuota() []models.AccountWithQuota {
	accs := m.accounts.GetAccounts()
	quotas := m.quota.GetAllQuotas()
	activeID := m.accounts.GetActiveAccountID()

	result := make([]models.AccountWithQuota, len(accs))
	for i, acc := range accs {
		result[i] = models.AccountWithQuota{
			Account:   acc,
			QuotaInfo: quotas[acc.Email],
			IsActive:  acc.ID == activeID || acc.Email == activeID,
		}
	}
	return result
}

// RefreshQuota forces a refresh of quota for all accounts.
func (m *Manager) RefreshQuota() {
	m.quota.RefreshAllQuotas()
}

// RefreshQuotaForAccount forces a refresh of quota for a specific account.
func (m *Manager) RefreshQuotaForAccount(email string) (*models.QuotaInfo, error) {
	return m.quota.RefreshQuota(email)
}

// GetStats returns aggregated statistics.
func (m *Manager) GetStats() StatsEvent {
	quotaStats := m.quota.GetStats()

	return StatsEvent{
		AccountCount:   m.accounts.Count(),
		QuotaCached:    quotaStats.CachedQuotas,
		TotalRemaining: quotaStats.TotalRemaining,
		TotalLimit:     quotaStats.TotalLimit,
	}
}

// Accounts returns the accounts service.
func (m *Manager) Accounts() *accounts.Service {
	return m.accounts
}

// Quota returns the quota service.
func (m *Manager) Quota() *quota.Service {
	return m.quota
}

// Projection returns the projection service.
func (m *Manager) Projection() *projection.Service {
	return m.projection
}

// GetAllProjections returns all projections.
func (m *Manager) GetAllProjections() map[string]*models.AccountProjection {
	if m.projection == nil {
		return nil
	}
	return m.projection.GetAllProjections()
}

// GetAccountHistory retrieves historical statistics for a specific account.
func (m *Manager) GetAccountHistory(email string, timeRange models.TimeRange) (*models.AccountHistoryStats, error) {
	if m.database == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return m.database.GetAccountHistoryStats(email, timeRange)
}

// Database returns the database instance for direct access.
func (m *Manager) Database() *db.DB {
	return m.database
}

// Close closes the manager and all its services.
func (m *Manager) Close() error {
	close(m.stopChan)

	m.mu.Lock()
	for _, sub := range m.subscribers {
		close(sub)
	}
	m.subscribers = nil
	m.mu.Unlock()

	var errs []error

	if err := m.accounts.Close(); err != nil {
		errs = append(errs, err)
	}

	if err := m.quota.Close(); err != nil {
		errs = append(errs, err)
	}

	if m.database != nil {
		if err := m.database.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// InitialState returns the initial state of all services for TUI initialization.
func (m *Manager) InitialState() ([]models.AccountWithQuota, StatsEvent) {
	accounts := m.GetAccountsWithQuota()
	stats := m.GetStats()

	return accounts, stats
}

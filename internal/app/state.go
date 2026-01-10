// Package app provides the main Bubble Tea application model and state management.
package app

import (
	"sync"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/services"
)

// NotificationType defines the type of notification.
type NotificationType int

const (
	// NotificationSuccess represents a success notification.
	NotificationSuccess NotificationType = iota
	// NotificationError represents an error notification.
	NotificationError
	// NotificationWarning represents a warning notification.
	NotificationWarning
	// NotificationInfo represents an informational notification.
	NotificationInfo
	// NotificationLoading represents a loading notification with spinner.
	NotificationLoading
)

const (
	// LoadingNotificationID is the fixed ID for loading notifications.
	LoadingNotificationID = "__loading__"
)

// String returns the string representation of a NotificationType.
func (n NotificationType) String() string {
	switch n {
	case NotificationSuccess:
		return "success"
	case NotificationError:
		return "error"
	case NotificationWarning:
		return "warning"
	case NotificationInfo:
		return "info"
	default:
		return "unknown"
	}
}

// Notification represents a user-facing notification message.
type Notification struct {
	ID        string
	Type      NotificationType
	Message   string
	CreatedAt time.Time
	Duration  time.Duration
}

// IsExpired returns true if the notification has expired.
func (n *Notification) IsExpired() bool {
	if n.Duration <= 0 {
		return false
	}
	return time.Since(n.CreatedAt) > n.Duration
}

// LoadingState tracks loading states for different resources.
type LoadingState struct {
	Initial  bool
	Accounts bool
	Quota    bool
	Stats    bool
}

type AppState struct {
	mu sync.RWMutex

	Accounts             []models.AccountWithQuota
	ActiveAccount        *models.AccountWithQuota
	Stats                *services.StatsEvent
	Projections          map[string]*models.AccountProjection
	SelectedAccountIndex int

	Loading LoadingState

	LastUpdated time.Time

	notifications   []Notification
	notificationSeq int
}

func NewAppState() *AppState {
	return &AppState{
		Accounts:             make([]models.AccountWithQuota, 0),
		Projections:          make(map[string]*models.AccountProjection),
		SelectedAccountIndex: 0,
		notifications:        make([]Notification, 0),
		Loading: LoadingState{
			Initial: true,
		},
	}
}

// SetLoading sets the loading state for a specific resource.
func (s *AppState) SetLoading(resource string, loading bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch resource {
	case "initial":
		s.Loading.Initial = loading
	case "accounts":
		s.Loading.Accounts = loading
	case "quota":
		s.Loading.Quota = loading
	case "stats":
		s.Loading.Stats = loading
	}
}

// AnyLoading returns true if any resource is currently loading.
func (s *AppState) AnyLoading() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.Loading.Initial ||
		s.Loading.Accounts ||
		s.Loading.Quota ||
		s.Loading.Stats
}

// IsInitialLoading returns true if initial data is still loading.
func (s *AppState) IsInitialLoading() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Loading.Initial
}

// GetLoadingResources returns a list of currently loading resources.
func (s *AppState) GetLoadingResources() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var resources []string
	if s.Loading.Initial {
		resources = append(resources, "initial")
	}
	if s.Loading.Accounts {
		resources = append(resources, "accounts")
	}
	if s.Loading.Quota {
		resources = append(resources, "quota")
	}
	if s.Loading.Stats {
		resources = append(resources, "stats")
	}
	return resources
}

// SetAccounts updates the accounts list and finds the active account.
func (s *AppState) SetAccounts(accounts []models.AccountWithQuota) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Accounts = accounts
	s.LastUpdated = time.Now()

	// Find and update active account
	s.ActiveAccount = nil
	for i := range accounts {
		if accounts[i].IsActive {
			s.ActiveAccount = &accounts[i]
			break
		}
	}
}

// GetAccounts returns a copy of the accounts list.
func (s *AppState) GetAccounts() []models.AccountWithQuota {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accounts := make([]models.AccountWithQuota, len(s.Accounts))
	copy(accounts, s.Accounts)
	return accounts
}

// GetAccountCount returns the number of accounts.
func (s *AppState) GetAccountCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Accounts)
}

// GetActiveAccount returns the active account.
func (s *AppState) GetActiveAccount() *models.AccountWithQuota {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ActiveAccount
}

// SetStats updates the statistics.
func (s *AppState) SetStats(stats services.StatsEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Stats = &stats
}

// GetStats returns the current statistics.
func (s *AppState) GetStats() *services.StatsEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Stats
}

// AddNotification adds a new notification and returns its ID.
func (s *AppState) AddNotification(notifType NotificationType, message string, duration time.Duration) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.notificationSeq++
	id := time.Now().Format("20060102150405") + "-" + string(rune('A'+s.notificationSeq%26))

	notification := Notification{
		ID:        id,
		Type:      notifType,
		Message:   message,
		CreatedAt: time.Now(),
		Duration:  duration,
	}

	s.notifications = append(s.notifications, notification)

	// Keep only the last 10 notifications
	if len(s.notifications) > 10 {
		s.notifications = s.notifications[len(s.notifications)-10:]
	}

	return id
}

// RemoveNotification removes a notification by ID.
func (s *AppState) RemoveNotification(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, n := range s.notifications {
		if n.ID == id {
			s.notifications = append(s.notifications[:i], s.notifications[i+1:]...)
			return
		}
	}
}

// ClearExpiredNotifications removes all expired notifications.
func (s *AppState) ClearExpiredNotifications() {
	s.mu.Lock()
	defer s.mu.Unlock()

	active := make([]Notification, 0, len(s.notifications))
	for _, n := range s.notifications {
		if !n.IsExpired() {
			active = append(active, n)
		}
	}
	s.notifications = active
}

// GetNotifications returns a copy of all active notifications.
func (s *AppState) GetNotifications() []Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Clear expired inline when reading
	active := make([]Notification, 0, len(s.notifications))
	for _, n := range s.notifications {
		if !n.IsExpired() {
			active = append(active, n)
		}
	}

	return active
}

// ClearAllNotifications removes all notifications.
func (s *AppState) ClearAllNotifications() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifications = make([]Notification, 0)
}

// SetLoadingNotification sets a loading notification message.
func (s *AppState) SetLoadingNotification(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, n := range s.notifications {
		if n.ID == LoadingNotificationID {
			s.notifications[i].Message = message
			return
		}
	}

	s.notifications = append(s.notifications, Notification{
		ID:        LoadingNotificationID,
		Type:      NotificationLoading,
		Message:   message,
		CreatedAt: time.Now(),
		Duration:  0,
	})
}

// ClearLoadingNotification removes the loading notification.
func (s *AppState) ClearLoadingNotification() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, n := range s.notifications {
		if n.ID == LoadingNotificationID {
			s.notifications = append(s.notifications[:i], s.notifications[i+1:]...)
			return
		}
	}
}

// GetLastUpdated returns the last time the state was updated.
func (s *AppState) GetLastUpdated() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.LastUpdated
}

// TimeSinceUpdate returns the duration since the last update.
func (s *AppState) TimeSinceUpdate() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.LastUpdated.IsZero() {
		return 0
	}
	return time.Since(s.LastUpdated)
}

func (s *AppState) UpdateQuotaForAccount(email string, quotaInfo any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.Accounts {
		if s.Accounts[i].Account.Email == email {
			if qi, ok := quotaInfo.(interface{ GetAccountEmail() string }); ok {
				_ = qi
			}
			s.LastUpdated = time.Now()
			break
		}
	}
}

func (s *AppState) SetProjection(email string, proj *models.AccountProjection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Projections == nil {
		s.Projections = make(map[string]*models.AccountProjection)
	}
	s.Projections[email] = proj
}

func (s *AppState) GetProjection(email string) *models.AccountProjection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Projections == nil {
		return nil
	}
	return s.Projections[email]
}

func (s *AppState) GetProjections() map[string]*models.AccountProjection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Projections == nil {
		return nil
	}
	result := make(map[string]*models.AccountProjection, len(s.Projections))
	for k, v := range s.Projections {
		result[k] = v
	}
	return result
}

// GetSelectedAccountIndex returns the currently selected account index.
func (s *AppState) GetSelectedAccountIndex() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SelectedAccountIndex
}

// SetSelectedAccountIndex updates the selected account index.
func (s *AppState) SetSelectedAccountIndex(idx int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SelectedAccountIndex = idx
}

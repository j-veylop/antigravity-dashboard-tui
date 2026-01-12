package app

import (
	"testing"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/services"
)

func TestNewState(t *testing.T) {
	s := NewState()
	if s == nil {
		t.Fatal("NewState returned nil")
	}
	if len(s.Accounts) != 0 {
		t.Error("Accounts should be empty")
	}
	if s.Loading.Initial != true {
		t.Error("Initial loading should be true")
	}
}

func TestState_SetLoading(t *testing.T) {
	s := NewState()

	s.SetLoading("accounts", true)
	if !s.Loading.Accounts {
		t.Error("Accounts loading should be true")
	}
	if !s.AnyLoading() {
		t.Error("AnyLoading should be true")
	}

	s.SetLoading("accounts", false)
	// Initial is still true
	if !s.AnyLoading() {
		t.Error("AnyLoading should be true (Initial is true)")
	}

	s.SetLoading("initial", false)
	if s.AnyLoading() {
		t.Error("AnyLoading should be false")
	}

	resources := s.GetLoadingResources()
	if len(resources) != 0 {
		t.Errorf("GetLoadingResources should be empty, got %v", resources)
	}

	s.SetLoading("quota", true)
	resources = s.GetLoadingResources()
	if len(resources) != 1 || resources[0] != "quota" {
		t.Errorf("GetLoadingResources should contain quota, got %v", resources)
	}
}

func TestState_Accounts(t *testing.T) {
	s := NewState()

	accs := []models.AccountWithQuota{
		{Account: models.Account{Email: "a@test.com"}},
		{Account: models.Account{Email: "b@test.com"}, IsActive: true},
	}

	s.SetAccounts(accs)

	if s.GetAccountCount() != 2 {
		t.Errorf("GetAccountCount = %d, want 2", s.GetAccountCount())
	}

	active := s.GetActiveAccount()
	if active == nil {
		t.Fatal("GetActiveAccount returned nil")
	}
	if active.Email != "b@test.com" {
		t.Errorf("active email = %s, want b@test.com", active.Email)
	}

	gotAccs := s.GetAccounts()
	if len(gotAccs) != 2 {
		t.Errorf("GetAccounts returned %d items", len(gotAccs))
	}
}

func TestState_Notifications(t *testing.T) {
	s := NewState()

	id := s.AddNotification(NotificationInfo, "test", time.Minute)
	if id == "" {
		t.Error("AddNotification returned empty ID")
	}

	notifs := s.GetNotifications()
	if len(notifs) != 1 {
		t.Errorf("GetNotifications len = %d, want 1", len(notifs))
	}
	if notifs[0].Message != "test" {
		t.Errorf("Notification message = %s, want test", notifs[0].Message)
	}

	s.RemoveNotification(id)
	if len(s.GetNotifications()) != 0 {
		t.Error("Notification should be removed")
	}
}

func TestState_ClearExpiredNotifications(t *testing.T) {
	s := NewState()

	// Expired
	s.notifications = append(s.notifications, Notification{
		ID:        "expired",
		CreatedAt: time.Now().Add(-2 * time.Minute),
		Duration:  time.Minute,
	})

	// Active
	s.notifications = append(s.notifications, Notification{
		ID:        "active",
		CreatedAt: time.Now(),
		Duration:  time.Minute,
	})

	s.ClearExpiredNotifications()

	notifs := s.GetNotifications()
	if len(notifs) != 1 {
		t.Fatalf("Expected 1 notification, got %d", len(notifs))
	}
	if notifs[0].ID != "active" {
		t.Errorf("Expected active notification, got %s", notifs[0].ID)
	}
}

func TestState_LoadingNotification(t *testing.T) {
	s := NewState()

	s.SetLoadingNotification("loading...")
	notifs := s.GetNotifications()
	if len(notifs) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(notifs))
	}
	if notifs[0].ID != LoadingNotificationID {
		t.Errorf("Expected ID %s, got %s", LoadingNotificationID, notifs[0].ID)
	}
	if notifs[0].Message != "loading..." {
		t.Errorf("Expected message loading..., got %s", notifs[0].Message)
	}

	// Update message
	s.SetLoadingNotification("still loading...")
	notifs = s.GetNotifications()
	if len(notifs) != 1 {
		t.Errorf("Expected 1 notification after update")
	}
	if notifs[0].Message != "still loading..." {
		t.Errorf("Expected message still loading..., got %s", notifs[0].Message)
	}

	s.ClearLoadingNotification()
	if len(s.GetNotifications()) != 0 {
		t.Error("Loading notification should be cleared")
	}
}

func TestState_Stats(t *testing.T) {
	s := NewState()
	stats := services.StatsEvent{AccountCount: 10}

	s.SetStats(stats)
	got := s.GetStats()
	if got == nil {
		t.Fatal("GetStats returned nil")
	}
	if got.AccountCount != 10 {
		t.Errorf("AccountCount = %d, want 10", got.AccountCount)
	}
}

func TestState_Projections(t *testing.T) {
	s := NewState()
	email := "test@example.com"
	proj := &models.AccountProjection{Email: email}

	s.SetProjection(email, proj)

	got := s.GetProjection(email)
	if got != proj {
		t.Errorf("GetProjection = %v, want %v", got, proj)
	}

	all := s.GetProjections()
	if len(all) != 1 {
		t.Errorf("GetProjections len = %d, want 1", len(all))
	}
}

func TestState_SelectedAccountIndex(t *testing.T) {
	s := NewState()

	s.SetSelectedAccountIndex(5)
	if s.GetSelectedAccountIndex() != 5 {
		t.Errorf("GetSelectedAccountIndex = %d, want 5", s.GetSelectedAccountIndex())
	}
}

func TestState_UpdateQuotaForAccount(t *testing.T) {
	s := NewState()
	s.Accounts = []models.AccountWithQuota{
		{Account: models.Account{Email: "a@test.com"}},
	}

	// Just verify it doesn't panic and updates timestamp
	before := s.LastUpdated
	time.Sleep(time.Millisecond) // Ensure time advances

	// Mock quota info
	s.UpdateQuotaForAccount("a@test.com", nil)

	if !s.LastUpdated.After(before) {
		t.Error("LastUpdated should be updated")
	}

	if s.TimeSinceUpdate() == 0 {
		t.Error("TimeSinceUpdate should be > 0")
	}
}

func TestNotificationType_String(t *testing.T) {
	tests := []struct {
		t    NotificationType
		want string
	}{
		{NotificationSuccess, "success"},
		{NotificationError, "error"},
		{NotificationWarning, "warning"},
		{NotificationInfo, "info"},
		{NotificationType(999), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.t.String(); got != tt.want {
			t.Errorf("String() = %q, want %q", got, tt.want)
		}
	}
}

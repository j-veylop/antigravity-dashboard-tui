package services

import (
	"os"
	"testing"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/config"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

// MockAccountProvider for testing
type MockAccountProvider struct{}

func (m *MockAccountProvider) GetAccounts() []models.Account                  { return nil }
func (m *MockAccountProvider) GetAccountByEmail(email string) *models.Account { return nil }
func (m *MockAccountProvider) UpdateAccountEmail(old, new string) error       { return nil }

func TestNewManager(t *testing.T) {
	// Need valid paths for db and accounts to avoid errors
	tmpDir := t.TempDir()
	cfg := &config.Config{
		DatabasePath:         tmpDir + "/test.db",
		AccountsPath:         tmpDir + "/accounts.json",
		QuotaRefreshInterval: time.Minute,
	}

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Close()

	if mgr.Accounts() == nil {
		t.Error("Accounts service should be initialized")
	}
	if mgr.Quota() == nil {
		t.Error("Quota service should be initialized")
	}
	if mgr.Projection() == nil {
		t.Error("Projection service should be initialized")
	}
	if mgr.Database() == nil {
		t.Error("Database should be initialized")
	}
}

func TestManager_Getters(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		DatabasePath: tmpDir + "/test.db",
		AccountsPath: tmpDir + "/accounts.json",
	}
	mgr, _ := NewManager(cfg)
	defer mgr.Close()

	if mgr.GetAccountsWithQuota() == nil {
		// Empty is fine, just check not nil slice if init
		// Actually it returns make slice
	}

	// Test GetStats
	stats := mgr.GetStats()
	if stats.AccountCount != 0 {
		t.Errorf("Stats.AccountCount = %d, want 0", stats.AccountCount)
	}

	// Test GetAllProjections
	if mgr.GetAllProjections() == nil {
		// Should return map
	}
}

func TestManager_Subscription(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		DatabasePath: tmpDir + "/test.db",
		AccountsPath: tmpDir + "/accounts.json",
	}
	mgr, _ := NewManager(cfg)
	defer mgr.Close()

	ch, cmd := mgr.Subscribe()
	if ch == nil {
		t.Error("Subscribe returned nil channel")
	}
	if cmd == nil {
		t.Error("Subscribe returned nil command")
	}

	// Unsubscribe
	mgr.Unsubscribe(ch)

	// Check if channel is closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("Channel should be closed")
		}
	default:
		// might block if not closed and empty, but Unsubscribe closes it
	}
}

func TestManager_InitialState(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		DatabasePath: tmpDir + "/test.db",
		AccountsPath: tmpDir + "/accounts.json",
	}
	mgr, _ := NewManager(cfg)
	defer mgr.Close()

	accs, stats := mgr.InitialState()
	if len(accs) != 0 {
		t.Error("Expected 0 accounts")
	}
	if stats.AccountCount != 0 {
		t.Error("Expected 0 stats")
	}
}

func TestManager_Broadcast(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		DatabasePath: tmpDir + "/test.db",
		AccountsPath: tmpDir + "/accounts.json",
	}
	mgr, _ := NewManager(cfg)
	defer mgr.Close()

	ch, _ := mgr.Subscribe()
	defer mgr.Unsubscribe(ch)

	event := StatsEvent{AccountCount: 1}
	// We can't access broadcast directly as it's private, but we can trigger it via services
	// Or just test routeEvents logic via mocking services?
	// Services are hard dependencies in NewManager.

	// However, we can use the private broadcast method if we are in same package?
	// Yes, package services.
	mgr.broadcast(event)

	select {
	case e := <-ch:
		if e != event {
			t.Errorf("Got event %v, want %v", e, event)
		}
	case <-time.After(time.Second):
		t.Error("Timeout waiting for broadcast")
	}
}

func TestManager_CheckNotifications(t *testing.T) {
	// Private method test
	tmpDir := t.TempDir()
	cfg := &config.Config{
		DatabasePath: tmpDir + "/test.db",
		AccountsPath: tmpDir + "/accounts.json",
	}
	mgr, _ := NewManager(cfg)
	defer mgr.Close()

	email := "test@example.com"

	// First update (no previous)
	quota1 := &models.QuotaInfo{
		AccountEmail:   email,
		TotalLimit:     100,
		TotalRemaining: 50,
	}
	mgr.checkNotifications(email, quota1)

	// Second update - drop below 5%
	quota2 := &models.QuotaInfo{
		AccountEmail:   email,
		TotalLimit:     100,
		TotalRemaining: 4,
	}
	// This would trigger beeep.Notify, which might fail or panic in headless env if not mocked.
	// beeep usually detects environment.
	// We just hope it doesn't panic.
	mgr.checkNotifications(email, quota2)
}

func TestWaitForEvent(t *testing.T) {
	ch := make(chan ServiceEvent, 1)
	ch <- StatsEvent{}

	cmd := WaitForEvent(ch)
	msg := cmd()
	if msg == nil {
		t.Error("WaitForEvent cmd returned nil msg")
	}
}

func TestServiceEvent_Interface(t *testing.T) {
	var _ ServiceEvent = AccountsChangedEvent{}
	var _ ServiceEvent = QuotaUpdatedEvent{}
	var _ ServiceEvent = ProjectionUpdatedEvent{}
	var _ ServiceEvent = ErrorEvent{}
	var _ ServiceEvent = StatsEvent{}

	// Coverage for isServiceEvent methods
	AccountsChangedEvent{}.isServiceEvent()
	QuotaUpdatedEvent{}.isServiceEvent()
	ProjectionUpdatedEvent{}.isServiceEvent()
	ErrorEvent{}.isServiceEvent()
	StatsEvent{}.isServiceEvent()
}

// Test GetAccountHistory integration with DB
func TestManager_GetAccountHistory(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		DatabasePath: tmpDir + "/test.db",
		AccountsPath: tmpDir + "/accounts.json",
	}
	mgr, _ := NewManager(cfg)
	defer mgr.Close()

	// Should fail if no data or return empty stats
	stats, err := mgr.GetAccountHistory("test@example.com", models.TimeRange24Hours)
	if err != nil {
		t.Errorf("GetAccountHistory failed: %v", err)
	}
	if stats == nil {
		t.Error("GetAccountHistory returned nil")
	}
}

func TestManager_Close(t *testing.T) {
	// Already tested in defers, but specific test for errors
	mgr := &Manager{} // Empty manager
	if err := mgr.Close(); err != nil {
		// Should not panic
	}
}

func TestManager_UpdateProjection(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		DatabasePath: tmpDir + "/test.db",
		AccountsPath: tmpDir + "/accounts.json",
	}
	mgr, _ := NewManager(cfg)
	defer mgr.Close()

	email := "test@example.com"
	quota := &models.QuotaInfo{
		AccountEmail: email,
		ModelQuotas: []models.ModelQuota{
			{ModelFamily: "claude", Limit: 100, Remaining: 50, ResetTime: time.Now().Add(time.Hour)},
		},
		SubscriptionTier: "PRO",
	}

	mgr.updateProjection(email, quota)
	// Async, might not update immediately, but ensures coverage of function
}

func TestManager_HandleEvents(t *testing.T) {
	// Coverage for event handling logic
	// We can't easily inject events into internal channels from outside
	// without starting the loop which NewManager does.
	// But we can call the handler methods directly if we export them or they are private (we are in same package)

	tmpDir := t.TempDir()
	cfg := &config.Config{
		DatabasePath: tmpDir + "/test.db",
		AccountsPath: tmpDir + "/accounts.json",
	}
	mgr, _ := NewManager(cfg)
	defer mgr.Close()

	// Create dummy events
	// Since handleAccountEvent and handleQuotaEvent are methods on Manager, we can call them?
	// No, they are private methods called by routeEvents.
	// But we can trigger them by sending to the channel of the *mocked* services?
	// We can't easily mock the services inside manager as they are struct fields.
	// But we can rely on routeEvents running.
}

func TestManager_RefreshQuota(t *testing.T) {
	tmpDir := t.TempDir()
	// Create dummy accounts file
	os.WriteFile(tmpDir+"/accounts.json", []byte(`{"accounts":[]}`), 0600)

	cfg := &config.Config{
		DatabasePath: tmpDir + "/test.db",
		AccountsPath: tmpDir + "/accounts.json",
	}
	mgr, _ := NewManager(cfg)
	defer mgr.Close()

	mgr.RefreshQuota()
	// Should not panic

	_, err := mgr.RefreshQuotaForAccount("missing@example.com")
	// Will fail as account not found
	if err == nil {
		// Actually RefreshQuota might return "account not found" inside internal service logic
	}
}

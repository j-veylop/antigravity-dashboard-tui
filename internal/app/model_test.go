package app

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/services"
)

func TestNewModel(t *testing.T) {
	model := NewModel(nil)
	if model == nil {
		t.Fatal("NewModel returned nil")
	}
	if model.state == nil {
		t.Error("State should be initialized")
	}
	if model.activeTab != TabDashboard {
		t.Error("Default tab should be Dashboard")
	}
	if len(model.tabs) != 3 {
		t.Errorf("Should have 3 tabs placeholder, got %d", len(model.tabs))
	}
}

func TestModel_Init(t *testing.T) {
	model := NewModel(nil)
	cmd := model.Init()
	if cmd == nil {
		t.Error("Init returned nil command")
	}
}

func TestModel_Update_WindowSize(t *testing.T) {
	model := NewModel(nil)
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}

	newModel, _ := model.Update(msg)

	m, ok := newModel.(*Model)
	if !ok {
		t.Fatal("Update returned wrong model type")
	}

	if m.width != 100 {
		t.Errorf("Width = %d, want 100", m.width)
	}
	if m.height != 50 {
		t.Errorf("Height = %d, want 50", m.height)
	}
	if !m.ready {
		t.Error("Model should be ready after WindowSizeMsg")
	}
}

func TestModel_Update_TabSwitch(t *testing.T) {
	model := NewModel(nil)
	model.ready = true
	model.width = 100
	model.height = 50

	// Test switching to History
	msg := TabSwitchMsg{Tab: TabHistory}
	newModel, _ := model.Update(msg)
	m := newModel.(*Model)

	if m.activeTab != TabHistory {
		t.Errorf("ActiveTab = %v, want History", m.activeTab)
	}

	// Test key binding '2'
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
	newModel, _ = model.Update(keyMsg)
	cmd := model.handleKeyMsg(keyMsg)
	if cmd == nil {
		t.Error("Key '2' should return a command")
	}
}

func TestModel_Update_Tick(t *testing.T) {
	model := NewModel(nil)
	msg := TickMsg{Time: time.Now()}

	_, cmd := model.Update(msg)
	if cmd == nil {
		t.Error("TickMsg should return a command (next tick)")
	}
}

func TestModel_View(t *testing.T) {
	model := NewModel(nil)

	// Not ready
	view := model.View()
	if !strings.Contains(view, "Loading...") {
		t.Error("View should show Loading when not ready")
	}

	// Ready
	model.ready = true
	model.width = 80
	model.height = 24

	view = model.View()
	// Should show tabs
	if !strings.Contains(view, "Dashboard") {
		t.Error("View should show Dashboard tab")
	}
	// Should show placeholder since tabs are nil
	if !strings.Contains(view, "not yet implemented") {
		t.Error("View should show placeholder text")
	}
}

func TestModel_Help(t *testing.T) {
	model := NewModel(nil)
	model.ready = true
	model.width = 80
	model.height = 24

	// Toggle help
	model.Update(ToggleHelpMsg{})
	if !model.showHelp {
		t.Error("showHelp should be true")
	}

	view := model.View()
	if !strings.Contains(view, "Keyboard Shortcuts") {
		t.Error("View should show help modal")
	}

	// Toggle off via key
	model.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if model.showHelp {
		t.Error("showHelp should be false after toggle")
	}
}

func TestModel_Notifications(t *testing.T) {
	model := NewModel(nil)

	msg := AddNotificationMsg{
		Message:  "Test Note",
		Type:     NotificationInfo,
		Duration: 0,
	}

	model.Update(msg)

	notifs := model.state.GetNotifications()
	if len(notifs) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(notifs))
	}

	// Test rendering
	model.ready = true
	model.width = 80
	model.height = 24
	view := model.View()
	if !strings.Contains(view, "Test Note") {
		t.Error("View should show notification")
	}
}

func TestModel_HandleServiceEvent(t *testing.T) {
	model := NewModel(nil)

	// Stats event
	stats := services.StatsEvent{AccountCount: 5}
	model.handleServiceEvent(stats)

	if model.state.GetStats().AccountCount != 5 {
		t.Error("Stats should be updated")
	}

	// Error event
	errEvent := services.ErrorEvent{Service: "test", Error: nil}
	cmd := model.handleServiceEvent(errEvent)
	if cmd == nil {
		t.Error("Error event should trigger notification command")
	}
}

func TestModel_Update_Messages(t *testing.T) {
	model := NewModel(nil)

	// Test StartLoadingMsg
	model.Update(StartLoadingMsg{Resource: "accounts"})
	if !model.state.Loading.Accounts {
		t.Error("Loading.Accounts should be true")
	}

	// Test StopLoadingMsg
	model.Update(StopLoadingMsg{Resource: "accounts"})
	if model.state.Loading.Accounts {
		t.Error("Loading.Accounts should be false")
	}

	// Test AccountsLoadedMsg
	accs := []models.AccountWithQuota{{Account: models.Account{Email: "test@example.com"}}}
	stats := services.StatsEvent{AccountCount: 1}
	model.Update(AccountsLoadedMsg{Accounts: accs, Stats: stats})
	if model.state.GetAccountCount() != 1 {
		t.Error("Accounts should be updated")
	}
	if model.state.GetStats().AccountCount != 1 {
		t.Error("Stats should be updated")
	}
	if model.state.Loading.Initial {
		t.Error("Initial loading should be false")
	}

	// Test StatsLoadedMsg
	model.Update(StatsLoadedMsg{Stats: services.StatsEvent{AccountCount: 2}})
	if model.state.GetStats().AccountCount != 2 {
		t.Error("Stats should be updated")
	}
	if model.state.Loading.Stats {
		t.Error("Stats loading should be false")
	}

	// Test QuotaRefreshedMsg
	model.Update(QuotaRefreshedMsg{Email: "test@example.com", QuotaInfo: &models.QuotaInfo{}})
	// Just checks it doesn't panic and loading is cleared
	if model.state.Loading.Quota {
		t.Error("Quota loading should be false")
	}

	// Test SwitchAccountResultMsg
	cmds := model.handleSwitchAccountResult(SwitchAccountResultMsg{Email: "test@example.com", Success: true})
	msg := cmds[0]()
	if addMsg, ok := msg.(AddNotificationMsg); ok {
		model.Update(addMsg)
		notifs := model.state.GetNotifications()
		if len(notifs) == 0 || !strings.Contains(notifs[len(notifs)-1].Message, "Switched") {
			t.Error("Should add success notification for switch")
		}
	} else {
		t.Error("Command should return AddNotificationMsg")
	}

	// Failed switch
	cmds = model.handleSwitchAccountResult(SwitchAccountResultMsg{Email: "test@example.com", Success: false, Error: assertError(t, "fail")})
	msg = cmds[0]()
	if addMsg, ok := msg.(AddNotificationMsg); ok {
		model.Update(addMsg)
		notifs := model.state.GetNotifications()
		if len(notifs) == 0 || notifs[len(notifs)-1].Type != NotificationError {
			t.Error("Should add error notification for failed switch")
		}
	}

	// Test DeleteAccountResultMsg
	cmds = model.handleDeleteAccountResult(DeleteAccountResultMsg{Email: "test@example.com", Success: true})
	msg = cmds[0]()
	if addMsg, ok := msg.(AddNotificationMsg); ok {
		model.Update(addMsg)
		notifs := model.state.GetNotifications()
		if len(notifs) == 0 || !strings.Contains(notifs[len(notifs)-1].Message, "Deleted") {
			t.Error("Should add success notification for delete")
		}
	}

	// Failed delete
	cmds = model.handleDeleteAccountResult(DeleteAccountResultMsg{Email: "test@example.com", Success: false, Error: assertError(t, "fail")})
	msg = cmds[0]()
	if addMsg, ok := msg.(AddNotificationMsg); ok {
		model.Update(addMsg)
		notifs := model.state.GetNotifications()
		if len(notifs) == 0 || notifs[len(notifs)-1].Type != NotificationError {
			t.Error("Should add error notification for failed delete")
		}
	}

	// Test RefreshMsg
	// services is nil, so it returns empty cmds, but covers the switch
	model.Update(RefreshMsg{Resource: "all"})
	model.Update(RefreshMsg{Resource: "accounts"})
	model.Update(RefreshMsg{Resource: "quota"})
	model.Update(RefreshMsg{Resource: "stats"})

	// Test Notification Messages
	model.Update(AddNotificationMsg{Message: "test", Type: NotificationInfo})
	model.Update(RemoveNotificationMsg{ID: "nonexistent"}) // coverage
	model.Update(ClearExpiredNotificationsMsg{})
}

func TestModel_HandleSpinnerTick(t *testing.T) {
	model := NewModel(nil)
	// Spinner tick returns a command
	_, cmd := model.Update(spinner.TickMsg{})
	if cmd == nil {
		t.Error("Spinner tick should return command")
	}
}

func assertError(t *testing.T, msg string) error {
	return &testError{msg}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestTabID_String(t *testing.T) {
	if TabDashboard.String() != "Dashboard" {
		t.Error("TabDashboard.String() mismatch")
	}
	if TabHistory.String() != "History" {
		t.Error("TabHistory.String() mismatch")
	}
	if TabInfo.String() != "Info" {
		t.Error("TabInfo.String() mismatch")
	}
	if TabID(999).String() != "Unknown" {
		t.Error("Unknown tab string mismatch")
	}
}

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()
	if len(km.ShortHelp()) == 0 {
		t.Error("ShortHelp empty")
	}
	if len(km.FullHelp()) == 0 {
		t.Error("FullHelp empty")
	}
}

func TestDefaultStyles(t *testing.T) {
	s := DefaultStyles()
	// Just check it doesn't panic and returns something
	_ = s
}

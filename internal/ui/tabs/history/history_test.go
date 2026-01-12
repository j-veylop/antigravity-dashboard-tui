package history

import (
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/app"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/config"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/services"
)

func TestNew(t *testing.T) {
	state := app.NewState()
	m := New(state, nil)
	if m == nil {
		t.Fatal("New returned nil")
	}
}

func TestModel_Init(t *testing.T) {
	state := app.NewState()
	m := New(state, nil)
	if m.Init() == nil {
		t.Error("Init returned nil")
	}
}

func TestModel_Update(t *testing.T) {
	state := app.NewState()
	m := New(state, nil)

	updated, _ := m.Update(nil)
	if updated == nil {
		t.Error("Update returned nil model")
	}
}

func TestModel_View(t *testing.T) {
	state := app.NewState()
	state.SetLoading("initial", false)
	m := New(state, nil)

	view := m.View()
	if view == "" {
		t.Error("View returned empty string")
	}
}

func TestModel_WithData(t *testing.T) {
	// Setup real manager with DB
	tmpDir := t.TempDir()
	cfg := &config.Config{
		DatabasePath: filepath.Join(tmpDir, "test.db"),
		AccountsPath: filepath.Join(tmpDir, "accounts.json"),
	}
	mgr, err := services.NewManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer mgr.Close()

	// Seed DB
	database := mgr.Database()
	now := time.Now()
	err = database.UpsertAggregatedSnapshot(&models.AggregatedSnapshot{
		Email:          "test@example.com",
		BucketTime:     now,
		ClaudeConsumed: 10,
		Tier:           "PRO",
	})
	if err != nil {
		t.Fatalf("Failed to seed DB: %v", err)
	}

	state := app.NewState()
	state.SetLoading("initial", false)
	state.ActiveAccount = &models.AccountWithQuota{
		Account: models.Account{Email: "test@example.com"},
	}

	m := New(state, mgr)
	m.SetSize(100, 50)

	// Inject loaded message directly
	stats := &models.AccountHistoryStats{
		Email:           "test@example.com",
		TotalDataPoints: 10,
		// Populate other fields if necessary for view
	}

	m.Update(historyLoadedMsg{stats: stats})

	// Check view
	view := m.View()
	// Should show something related to history or at least not empty/loading
	if view == "" {
		t.Error("View after load is empty")
	}

	// Test navigation
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
}

func TestModel_SetSize(t *testing.T) {
	state := app.NewState()
	m := New(state, nil)
	m.SetSize(100, 50)
}

func TestModel_Help(t *testing.T) {
	state := app.NewState()
	m := New(state, nil)
	if m.ShortHelp() == nil {
		// might be empty
	}
	if len(m.FullHelp()) == 0 {
		t.Error("FullHelp empty")
	}
}

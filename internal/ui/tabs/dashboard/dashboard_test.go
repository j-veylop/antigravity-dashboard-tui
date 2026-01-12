package dashboard

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/app"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

func TestNew(t *testing.T) {
	state := app.NewState()
	m := New(state)
	if m == nil {
		t.Fatal("New returned nil")
	}
}

func TestModel_Init(t *testing.T) {
	state := app.NewState()
	m := New(state)
	if m.Init() == nil {
		t.Error("Init returned nil")
	}
}

func TestModel_Update(t *testing.T) {
	state := app.NewState()
	m := New(state)

	// Test nil msg
	updated, cmd := m.Update(nil)
	if updated == nil {
		t.Error("Update returned nil model")
	}
	_ = cmd
}

func TestModel_View(t *testing.T) {
	state := app.NewState()
	state.SetLoading("initial", false)
	m := New(state)

	// View with no data
	view := m.View()
	if view == "" {
		t.Error("View returned empty string")
	}

	// View with accounts
	accs := []models.AccountWithQuota{
		{
			Account: models.Account{Email: "test@example.com", ID: "1"},
			QuotaInfo: &models.QuotaInfo{
				TotalRemaining: 50,
				TotalLimit:     100,
				ModelQuotas: []models.ModelQuota{
					{ModelFamily: "claude", Remaining: 10, Limit: 20, ResetTime: time.Now().Add(time.Hour)},
				},
			},
			IsActive: true,
		},
	}
	state.SetAccounts(accs)

	// Need to set size to ensure rendering
	m.SetSize(80, 24)

	view = m.View()
	if !strings.Contains(view, "test@example.com") {
		t.Logf("View content: %q", view)
		t.Error("View should contain email")
	}
	// Check case-insensitive as UI might capitalize or style it
	if !strings.Contains(strings.ToLower(view), "claude") {
		t.Logf("View content: %q", view)
		t.Error("View should contain model family")
	}
}

func TestModel_Animation(t *testing.T) {
	state := app.NewState()
	m := New(state)
	// Trigger animation tick
	// We need to inspect internal state, but it's private.
	// We can checking coverage by running update loop

	// Use tea.TickMsg to simulate animation tick if we knew the type,
	// but it's internal to component.
	// However, we can call Update with standard messages.
	m.Update(nil)

	// Just use m to satisfy unused variable check
	_ = m
}

func TestModel_SetSize(t *testing.T) {
	state := app.NewState()
	m := New(state)
	m.SetSize(100, 50)
}

func TestModel_Help(t *testing.T) {
	state := app.NewState()
	m := New(state)
	if len(m.ShortHelp()) == 0 {
		t.Error("ShortHelp empty")
	}
	if len(m.FullHelp()) == 0 {
		t.Error("FullHelp empty")
	}
}

func TestModel_KeyBindings(t *testing.T) {
	state := app.NewState()
	m := New(state)

	// Test navigation keys
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Since there are no accounts, selection logic might be limited, but coverage should increase
}

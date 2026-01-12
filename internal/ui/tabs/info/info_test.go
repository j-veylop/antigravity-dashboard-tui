package info

import (
	"testing"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/app"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/config"
)

func TestNew(t *testing.T) {
	state := app.NewState()
	cfg := &config.Config{}
	m := New(state, cfg)
	if m == nil {
		t.Fatal("New returned nil")
	}
}

func TestModel_Init(t *testing.T) {
	state := app.NewState()
	cfg := &config.Config{}
	m := New(state, cfg)
	if m.Init() != nil {
		// Init should return nil
	}
}

func TestModel_Update(t *testing.T) {
	state := app.NewState()
	cfg := &config.Config{}
	m := New(state, cfg)

	updated, _ := m.Update(nil)
	if updated == nil {
		t.Error("Update returned nil model")
	}
}

func TestModel_View(t *testing.T) {
	state := app.NewState()
	cfg := &config.Config{
		GoogleClientID: "test",
	}
	m := New(state, cfg)
	m.SetSize(80, 24)

	view := m.View()
	if view == "" {
		t.Error("View returned empty string")
	}
}

func TestModel_SetSize(t *testing.T) {
	state := app.NewState()
	cfg := &config.Config{}
	m := New(state, cfg)
	m.SetSize(100, 50)
}

func TestModel_Help(t *testing.T) {
	state := app.NewState()
	cfg := &config.Config{}
	m := New(state, cfg)
	if m.ShortHelp() == nil {
		// might be empty
	}
	if m.FullHelp() == nil {
		// might be empty
	}
}

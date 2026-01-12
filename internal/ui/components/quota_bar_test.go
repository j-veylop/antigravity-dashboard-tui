package components

import (
	"strings"
	"testing"
)

func TestNewQuotaBar(t *testing.T) {
	bar := NewQuotaBar()
	if bar.percent != 0 {
		t.Errorf("percent = %f, want 0.0", bar.percent)
	}
}

func TestQuotaBar_Setters(t *testing.T) {
	bar := NewQuotaBar()
	bar.SetPercent(75.5)
	if bar.percent != 75.5 {
		t.Errorf("percent = %f, want 75.5", bar.percent)
	}

	bar.SetLabel("Test")
	if bar.label != "Test" {
		t.Errorf("label = %s, want Test", bar.label)
	}

	bar.SetWidth(20)
}

func TestQuotaBar_View(t *testing.T) {
	bar := NewQuotaBar()
	view := bar.View(50.0, "Test", 40)
	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestQuotaBar_ViewCompact(t *testing.T) {
	bar := NewQuotaBar()
	view := bar.ViewCompact(50.0, 20)
	if !strings.Contains(view, "50%") {
		t.Error("ViewCompact() should contain percentage")
	}
}

func TestQuotaBar_ViewRateLimited(t *testing.T) {
	bar := NewQuotaBar()
	view := bar.ViewRateLimited("Test", 40)
	if !strings.Contains(view, "RATE LIMITED") {
		t.Error("ViewRateLimited() should contain warning")
	}
}

func TestNewTimeBar(t *testing.T) {
	_ = NewTimeBar()
}

func TestRenderTimeBarChars(t *testing.T) {
	s := RenderTimeBarChars(0.5, 10)
	if len(s) == 0 {
		t.Error("RenderTimeBarChars returned empty")
	}
}

func TestViewWithLabel(t *testing.T) {
	bar := NewTimeBar()
	view := bar.ViewWithLabel(3600, "Label", 40, "FREE")
	if !strings.Contains(view, "Label") {
		t.Error("ViewWithLabel missing label")
	}
}

func TestRenderGradientBar(t *testing.T) {
	s := RenderGradientBar(50.0, 10)
	if len(s) == 0 {
		t.Error("RenderGradientBar returned empty")
	}
}

func TestSimpleQuotaBar(t *testing.T) {
	s := SimpleQuotaBar(50.0, "Test", 40)
	if len(s) == 0 {
		t.Error("SimpleQuotaBar returned empty")
	}
}

func TestLoadingBars(t *testing.T) {
	s := SimpleQuotaBarLoading("Test", 40, 0)
	if len(s) == 0 {
		t.Error("SimpleQuotaBarLoading returned empty")
	}

	s2 := SimpleTimeBarLoading("Test", 40, 0)
	if len(s2) == 0 {
		t.Error("SimpleTimeBarLoading returned empty")
	}
}

func TestNewQuotaBarWithWidth(t *testing.T) {
	bar := NewQuotaBarWithWidth(30)
	_ = bar
}

func TestQuotaBar_InitUpdate(t *testing.T) {
	bar := NewQuotaBar()
	if bar.Init() != nil {
		t.Error("Init should return nil")
	}

	model, cmd := bar.Update(nil)
	if cmd != nil {
		// Just to use cmd if it's not nil, though nil msg should return nil cmd
	}
	_ = model
}

func TestViewFromSecondsWithLabel(t *testing.T) {
	bar := NewTimeBar()
	view := bar.ViewFromSecondsWithLabel(3600, "Time", 40, "FREE")
	if !strings.Contains(view, "Time") {
		t.Error("ViewFromSecondsWithLabel missing label")
	}
}

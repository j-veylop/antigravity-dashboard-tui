package components

import (
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

func TestNewSpinner(t *testing.T) {
	s := NewSpinner("Loading")
	if s.label != "Loading" {
		t.Error("Spinner label mismatch")
	}
}

func TestSpinner_Methods(t *testing.T) {
	s := NewSpinner("Init")

	s.SetLabel("Loading")
	if s.Label() != "Loading" {
		t.Errorf("Label = %s, want Loading", s.Label())
	}

	// Test View
	view := s.View()
	if view == "" {
		t.Error("View returned empty")
	}

	// Test ViewWithLabel
	view = s.ViewWithLabel()
	if view == "" {
		t.Error("ViewWithLabel returned empty")
	}

	// Test Init
	if s.Init() == nil {
		t.Error("Init should return command")
	}

	// Test Update
	m, cmd := s.Update(spinner.TickMsg{})
	_ = m
	if cmd == nil {
		t.Error("Update should return command for tick")
	}

	// Test Tick
	if s.Tick() == nil {
		t.Error("Tick should return command")
	}

	// Test Spinner accessor
	if s.Spinner().Spinner.Frames == nil {
		t.Error("Spinner accessor failed")
	}
}

func TestRenderSpinnerCentered(t *testing.T) {
	s := NewSpinner("Loading...")
	view := RenderSpinnerCentered(&s, 20, 5)
	if view == "" {
		t.Error("RenderSpinnerCentered returned empty")
	}
}

func TestRenderLineChart(t *testing.T) {
	data := []float64{1, 2, 3, 4}
	s := RenderLineChart(data, 20, 5, "Test")
	if s == "" {
		t.Error("RenderLineChart returned empty")
	}
}

func TestRenderDualLineChart(t *testing.T) {
	data1 := []float64{1, 2, 3}
	data2 := []float64{3, 2, 1}
	s := RenderDualLineChart(data1, data2, 20, 5, "Title")
	if s == "" {
		t.Error("RenderDualLineChart returned empty")
	}
}

func TestRenderBarChart(t *testing.T) {
	values := []float64{10, 20}
	labels := []string{"A", "B"}
	s := RenderBarChart(values, labels, 20)
	if s == "" {
		t.Error("RenderBarChart returned empty")
	}
}

func TestRenderHourlyHeatmap(t *testing.T) {
	data := make([]float64, 24)
	s := RenderHourlyHeatmap(data)
	if s == "" {
		t.Error("RenderHourlyHeatmap returned empty")
	}
}

func TestRenderWeeklyPattern(t *testing.T) {
	data := make([]float64, 7)
	names := []string{"S", "M", "T", "W", "T", "F", "S"}
	s := RenderWeeklyPattern(data, names)
	if s == "" {
		t.Error("RenderWeeklyPattern returned empty")
	}
}

func TestRenderSparkline(t *testing.T) {
	data := []float64{1, 2, 3}
	s := RenderSparkline(data, 10)
	if s == "" {
		t.Error("RenderSparkline returned empty")
	}
}

func TestRenderColoredSparkline(t *testing.T) {
	data := []float64{1, 2, 3}
	s := RenderColoredSparkline(data, 10)
	if s == "" {
		t.Error("RenderColoredSparkline returned empty")
	}
}

func TestRenderLegend(t *testing.T) {
	items := []LegendItem{
		{Label: "A", Color: lipgloss.Color("#ffffff")},
	}
	s := RenderLegend(items)
	if s == "" {
		t.Error("RenderLegend returned empty")
	}
}

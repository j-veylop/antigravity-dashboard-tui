package components

import (
	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/styles"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LoadingSpinner wraps a bubble spinner with label support.
type LoadingSpinner struct {
	spinner spinner.Model
	label   string
	style   lipgloss.Style
}

// NewSpinner creates a new loading spinner with the given label.
func NewSpinner(label string) LoadingSpinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(styles.Primary)

	return LoadingSpinner{
		spinner: s,
		label:   label,
		style:   lipgloss.NewStyle().Foreground(styles.TextSecondary),
	}
}

// Init initializes the spinner model.
func (l LoadingSpinner) Init() tea.Cmd {
	return l.spinner.Tick
}

// Update handles spinner tick messages.
func (l LoadingSpinner) Update(msg tea.Msg) (LoadingSpinner, tea.Cmd) {
	var cmd tea.Cmd
	l.spinner, cmd = l.spinner.Update(msg)
	return l, cmd
}

// View renders the spinner without label.
func (l LoadingSpinner) View() string {
	return l.spinner.View()
}

// ViewWithLabel renders the spinner with its label.
func (l LoadingSpinner) ViewWithLabel() string {
	return l.spinner.View() + " " + l.style.Render(l.label)
}

// SetLabel updates the spinner's label.
func (l *LoadingSpinner) SetLabel(label string) {
	l.label = label
}

// Label returns the current label.
func (l LoadingSpinner) Label() string {
	return l.label
}

// Spinner returns the underlying spinner model.
func (l LoadingSpinner) Spinner() spinner.Model {
	return l.spinner
}

// Tick returns the tick command for the spinner.
func (l LoadingSpinner) Tick() tea.Cmd {
	return l.spinner.Tick
}

// RenderSpinnerCentered renders a spinner centered in a given width and height.
func RenderSpinnerCentered(s LoadingSpinner, width, height int) string {
	content := s.ViewWithLabel()
	return styles.CenterBoth(content, width, height)
}

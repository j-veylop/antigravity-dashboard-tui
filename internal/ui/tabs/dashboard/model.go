// Package dashboard provides the main dashboard tab for the Antigravity Dashboard TUI.
package dashboard

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/app"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/components"
)

type animationTickMsg time.Time

func animationTickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*40, func(t time.Time) tea.Msg {
		return animationTickMsg(t)
	})
}

// keyMap defines the key bindings specific to the dashboard tab.
type keyMap struct {
	NextAccount key.Binding
	PrevAccount key.Binding
	Refresh     key.Binding
	Up          key.Binding
	Down        key.Binding
}

// defaultKeyMap returns the default key bindings for the dashboard tab.
func defaultKeyMap() keyMap {
	return keyMap{
		NextAccount: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next account"),
		),
		PrevAccount: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "prev account"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
	}
}

// AnimationState tracks the state of an animation.
type AnimationState struct {
	CurrentPercent float64
	TargetPercent  float64
	StartPercent   float64
	StartTime      time.Time
}

// Model represents the dashboard tab state.
type Model struct {
	state          *app.AppState
	width          int
	height         int
	spinner        components.LoadingSpinner
	claudeQuotaBar components.QuotaBar
	geminiQuotaBar components.QuotaBar
	timeBar        components.TimeBar
	keys           keyMap
	selectedIndex  int
	viewport       viewport.Model
	animations     map[string]*AnimationState
	animationFrame int
}

// New creates a new dashboard model.
func New(state *app.AppState) *Model {
	// Initialize spinner but don't configure it here since NewSpinner does it
	return &Model{
		state:          state,
		spinner:        components.NewSpinner("Loading accounts..."),
		claudeQuotaBar: components.NewQuotaBar(),
		geminiQuotaBar: components.NewQuotaBar(),
		timeBar:        components.NewTimeBar(),
		keys:           defaultKeyMap(),
		selectedIndex:  0,
		viewport:       viewport.New(0, 0),
		animations:     make(map[string]*AnimationState),
	}
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Init(), animationTickCmd())
}

// Update handles messages and updates the model.
// Update is the main update loop.
// Update processes messages.
func (m *Model) Update(msg tea.Msg) (app.Tab, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case animationTickMsg:
		m.animationFrame++
		now := time.Time(msg)

		animating, hasPendingData := m.syncAnimationTargets(now)
		m.stepAnimations(now)

		shouldTick := animating || m.state.AnyLoading() || m.state.IsInitialLoading() || hasPendingData
		if shouldTick {
			cmds = append(cmds, animationTickCmd())
		}

	case app.StartLoadingMsg:
		cmds = append(cmds, animationTickCmd())

	case app.QuotaUpdatedEventMsg, app.AccountsLoadedMsg, app.RefreshMsg:
		m.syncAnimationTargets(time.Now())
		cmds = append(cmds, animationTickCmd())

	case app.ProjectionUpdatedMsg:
		m.state.SetProjection(msg.Email, msg.Projection)

	case tea.KeyMsg:
		accounts := m.state.GetAccounts()
		accountCount := len(accounts)

		switch {
		case key.Matches(msg, m.keys.NextAccount):
			if accountCount > 0 {
				m.selectedIndex = (m.selectedIndex + 1) % accountCount
			}
		case key.Matches(msg, m.keys.PrevAccount):
			if accountCount > 0 {
				m.selectedIndex = (m.selectedIndex - 1 + accountCount) % accountCount
			}
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// SetSize sets the available size for the dashboard.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
}

func (m *Model) syncAnimationTargets(now time.Time) (animating, hasPendingData bool) {
	accounts := m.state.GetAccounts()

	for _, acc := range accounts {
		if acc.QuotaInfo == nil {
			hasPendingData = true
			continue
		}

		claudeTarget := -1.0
		geminiTarget := -1.0

		for _, mq := range acc.QuotaInfo.ModelQuotas {
			target := 0.0
			if mq.Limit > 0 && !mq.IsRateLimited {
				target = float64(mq.Remaining) / float64(mq.Limit) * 100
			}

			if mq.ModelFamily == "claude" {
				if claudeTarget < 0 || target < claudeTarget {
					claudeTarget = target
				}
			} else if mq.ModelFamily == "gemini" {
				if geminiTarget < 0 || target < geminiTarget {
					geminiTarget = target
				}
			}
		}

		if claudeTarget >= 0 {
			key := acc.Account.Email + ":claude"
			state, exists := m.animations[key]
			if !exists {
				state = &AnimationState{
					CurrentPercent: 0,
					StartPercent:   0,
					TargetPercent:  0,
					StartTime:      now,
				}
				m.animations[key] = state
			}

			if claudeTarget != state.TargetPercent {
				state.StartPercent = state.CurrentPercent
				state.TargetPercent = claudeTarget
				state.StartTime = now
			}

			if state.CurrentPercent != state.TargetPercent {
				animating = true
			}
		}

		if geminiTarget >= 0 {
			key := acc.Account.Email + ":gemini"
			state, exists := m.animations[key]
			if !exists {
				state = &AnimationState{
					CurrentPercent: 0,
					StartPercent:   0,
					TargetPercent:  0,
					StartTime:      now,
				}
				m.animations[key] = state
			}

			if geminiTarget != state.TargetPercent {
				state.StartPercent = state.CurrentPercent
				state.TargetPercent = geminiTarget
				state.StartTime = now
			}

			if state.CurrentPercent != state.TargetPercent {
				animating = true
			}
		}
	}

	return animating, hasPendingData
}

func (m *Model) stepAnimations(now time.Time) {
	for _, state := range m.animations {
		if state.CurrentPercent != state.TargetPercent {
			elapsed := now.Sub(state.StartTime).Seconds()
			duration := 1.5

			if elapsed >= duration {
				state.CurrentPercent = state.TargetPercent
			} else {
				progress := elapsed / duration
				ease := 1.0 - (1.0-progress)*(1.0-progress)
				state.CurrentPercent = state.StartPercent + (state.TargetPercent-state.StartPercent)*ease
			}
		}
	}
}

// ShortHelp returns the key bindings for the short help view.
func (m *Model) ShortHelp() []key.Binding {
	return []key.Binding{
		m.keys.NextAccount,
		m.keys.PrevAccount,
		m.keys.Refresh,
	}
}

// FullHelp returns the key bindings for the full help view.
func (m *Model) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.keys.NextAccount, m.keys.PrevAccount},
		{m.keys.Refresh},
	}
}

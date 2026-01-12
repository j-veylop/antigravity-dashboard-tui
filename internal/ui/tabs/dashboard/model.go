// Package dashboard provides the main dashboard tab for the Antigravity Dashboard TUI.
package dashboard

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/app"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
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
	NextAccount  key.Binding
	PrevAccount  key.Binding
	FirstAccount key.Binding
	LastAccount  key.Binding
	Refresh      key.Binding
}

// defaultKeyMap returns the default key bindings for the dashboard tab.
func defaultKeyMap() keyMap {
	return keyMap{
		NextAccount: key.NewBinding(
			key.WithKeys("n", "j", "down"),
			key.WithHelp("j/n", "next account"),
		),
		PrevAccount: key.NewBinding(
			key.WithKeys("p", "k", "up"),
			key.WithHelp("k/p", "prev account"),
		),
		FirstAccount: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "first account"),
		),
		LastAccount: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "last account"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
	}
}

// AnimationState tracks the state of an animation.
type AnimationState struct {
	StartTime      time.Time
	CurrentPercent float64
	TargetPercent  float64
	StartPercent   float64
}

// Model represents the dashboard tab state.
type Model struct {
	state          *app.State
	animations     map[string]*AnimationState
	spinner        components.LoadingSpinner
	keys           keyMap
	viewport       viewport.Model
	timeBar        components.TimeBar
	claudeQuotaBar components.QuotaBar
	geminiQuotaBar components.QuotaBar
	width          int
	height         int
	selectedIndex  int
	animationFrame int
}

// New creates a new dashboard model.
func New(state *app.State) *Model {
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
func (m *Model) Update(msg tea.Msg) (app.Tab, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case animationTickMsg:
		cmds = append(cmds, m.handleAnimationTick(msg))

	case app.StartLoadingMsg:
		cmds = append(cmds, animationTickCmd())

	case app.QuotaUpdatedEventMsg, app.AccountsLoadedMsg, app.RefreshMsg:
		m.syncAnimationTargets(time.Now())
		cmds = append(cmds, animationTickCmd())

	case app.ProjectionUpdatedMsg:
		m.state.SetProjection(msg.Email, msg.Projection)

	case tea.KeyMsg:
		cmds = append(cmds, m.handleKeyMsg(msg))

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleAnimationTick(msg animationTickMsg) tea.Cmd {
	m.animationFrame++
	now := time.Time(msg)

	animating, hasPendingData := m.syncAnimationTargets(now)
	m.stepAnimations(now)

	shouldTick := animating || m.state.AnyLoading() || m.state.IsInitialLoading() || hasPendingData
	if shouldTick {
		return animationTickCmd()
	}
	return nil
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
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
	case key.Matches(msg, m.keys.FirstAccount):
		if accountCount > 0 {
			m.selectedIndex = 0
		}
	case key.Matches(msg, m.keys.LastAccount):
		if accountCount > 0 {
			m.selectedIndex = accountCount - 1
		}
	default:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return cmd
	}
	return nil
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

	for i := range accounts {
		acc := &accounts[i]
		if acc.QuotaInfo == nil {
			hasPendingData = true
			continue
		}

		claudeTarget, geminiTarget := m.calculateTargets(acc.QuotaInfo)

		if m.updateAnimationState(acc.Email+":claude", claudeTarget, now) {
			animating = true
		}
		if m.updateAnimationState(acc.Email+":gemini", geminiTarget, now) {
			animating = true
		}
	}

	return animating, hasPendingData
}

func (m *Model) calculateTargets(quotaInfo *models.QuotaInfo) (claudeTarget, geminiTarget float64) {
	claudeTarget = -1.0
	geminiTarget = -1.0

	for i := range quotaInfo.ModelQuotas {
		mq := &quotaInfo.ModelQuotas[i]
		target := 0.0
		if mq.Limit > 0 && !mq.IsRateLimited {
			target = float64(mq.Remaining) / float64(mq.Limit) * 100
		}

		switch mq.ModelFamily {
		case "claude":
			if claudeTarget < 0 || target < claudeTarget {
				claudeTarget = target
			}
		case "gemini":
			if geminiTarget < 0 || target < geminiTarget {
				geminiTarget = target
			}
		}
	}
	return claudeTarget, geminiTarget
}

func (m *Model) updateAnimationState(animKey string, target float64, now time.Time) bool {
	if target < 0 {
		return false
	}

	state, exists := m.animations[animKey]
	if !exists {
		state = &AnimationState{
			CurrentPercent: 0,
			StartPercent:   0,
			TargetPercent:  0,
			StartTime:      now,
		}
		m.animations[animKey] = state
	}

	if target != state.TargetPercent {
		state.StartPercent = state.CurrentPercent
		state.TargetPercent = target
		state.StartTime = now
	}

	return state.CurrentPercent != state.TargetPercent
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
		{m.keys.FirstAccount, m.keys.LastAccount},
		{m.keys.Refresh},
	}
}

// Package app implements the main Bubble Tea application with tab-based navigation.
package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/services"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/styles"
)

// TabID represents the identifier for a tab in the application.
type TabID int

const (
	// TabDashboard is the ID for the dashboard tab.
	TabDashboard TabID = iota
	// TabHistory is the ID for the history tab.
	TabHistory
	// TabInfo is the ID for the info tab.
	TabInfo
)

// String returns the string representation of the TabID.
func (t TabID) String() string {
	switch t {
	case TabDashboard:
		return "Dashboard"
	case TabHistory:
		return "History"
	case TabInfo:
		return "Info"
	default:
		return "Unknown"
	}
}

// Tab defines the interface that all tabs must implement.
type Tab interface {
	// Init initializes the tab and returns any initial commands.
	Init() tea.Cmd

	// Update handles messages and returns the updated tab and any commands.
	Update(msg tea.Msg) (Tab, tea.Cmd)

	// View renders the tab content.
	View() string

	// SetSize sets the available size for the tab.
	SetSize(width, height int)

	// ShortHelp returns key bindings for the short help view.
	ShortHelp() []key.Binding

	// FullHelp returns key bindings for the full help view.
	FullHelp() [][]key.Binding
}

// KeyMap defines the keybindings for the application.
type KeyMap struct {
	Tab1        key.Binding
	Tab2        key.Binding
	Tab3        key.Binding
	NextTab     key.Binding
	PrevTab     key.Binding
	Refresh     key.Binding
	Help        key.Binding
	Quit        key.Binding
	Up          key.Binding
	Down        key.Binding
	Left        key.Binding
	Right       key.Binding
	Enter       key.Binding
	Escape      key.Binding
	PageUp      key.Binding
	PageDown    key.Binding
	Home        key.Binding
	End         key.Binding
	Filter      key.Binding
	Copy        key.Binding
	Delete      key.Binding
	SwitchFocus key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	km := KeyMap{}
	km = setTabKeys(km)
	km = setActionKeys(km)
	km = setNavigationKeys(km)
	km = setListKeys(km)
	return km
}

func setTabKeys(k KeyMap) KeyMap {
	k.Tab1 = key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "dashboard"))
	k.Tab2 = key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "history"))
	k.Tab3 = key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "info"))
	k.NextTab = key.NewBinding(key.WithKeys("tab", "l", "right"), key.WithHelp("tab/→", "next tab"))
	k.PrevTab = key.NewBinding(key.WithKeys("shift+tab", "h", "left"), key.WithHelp("shift+tab/←", "prev tab"))
	return k
}

func setActionKeys(k KeyMap) KeyMap {
	k.Refresh = key.NewBinding(key.WithKeys("r", "ctrl+r"), key.WithHelp("r", "refresh"))
	k.Help = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help"))
	k.Quit = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))
	k.Copy = key.NewBinding(key.WithKeys("c", "ctrl+c"), key.WithHelp("c", "copy"))
	k.Delete = key.NewBinding(key.WithKeys("d", "delete"), key.WithHelp("d", "delete"))
	return k
}

func setNavigationKeys(k KeyMap) KeyMap {
	k.Up = key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up"))
	k.Down = key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down"))
	k.Left = key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left"))
	k.Right = key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right"))
	k.Enter = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select"))
	k.Escape = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))
	k.SwitchFocus = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch focus"))
	return k
}

func setListKeys(k KeyMap) KeyMap {
	k.PageUp = key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("pgup", "page up"))
	k.PageDown = key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("pgdn", "page down"))
	k.Home = key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("home", "go to top"))
	k.End = key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("end", "go to bottom"))
	k.Filter = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter"))
	return k
}

// ShortHelp returns key bindings for the short help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Refresh, k.Quit}
}

// FullHelp returns key bindings for the full help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3},
		{k.NextTab, k.PrevTab},
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.Refresh, k.Help, k.Quit},
	}
}

// Styles defines the application styles.
type Styles struct {
	// Tab bar styles
	TabBar       lipgloss.Style
	ActiveTab    lipgloss.Style
	InactiveTab  lipgloss.Style
	TabSeparator lipgloss.Style

	// Notification styles
	NotificationSuccess lipgloss.Style
	NotificationError   lipgloss.Style
	NotificationWarning lipgloss.Style
	NotificationInfo    lipgloss.Style

	// Content styles
	Content lipgloss.Style
	Help    lipgloss.Style
	Spinner lipgloss.Style
	Toast   lipgloss.Style

	// Common styles
	Title     lipgloss.Style
	Subtle    lipgloss.Style
	Highlight lipgloss.Style
	Error     lipgloss.Style
	Success   lipgloss.Style
	Warning   lipgloss.Style
}

// DefaultStyles returns the default application styles.
func DefaultStyles() Styles {
	subtle := lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}
	highlight := lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	success := lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}
	warning := lipgloss.AdaptiveColor{Light: "#FF8C00", Dark: "#FF8C00"}
	errorColor := lipgloss.AdaptiveColor{Light: "#FF5F87", Dark: "#FF5F87"}
	info := lipgloss.AdaptiveColor{Light: "#0087D7", Dark: "#5FAFFF"}

	s := Styles{}
	s.TabBar = lipgloss.NewStyle().Padding(0, 1).BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).BorderForeground(subtle)
	s.ActiveTab = lipgloss.NewStyle().Bold(true).Foreground(highlight).Padding(0, 2)
	s.InactiveTab = lipgloss.NewStyle().Foreground(subtle).Padding(0, 2)
	s.TabSeparator = lipgloss.NewStyle().Foreground(subtle).SetString(" | ")

	s.NotificationSuccess = lipgloss.NewStyle().Foreground(success).Padding(0, 1)
	s.NotificationError = lipgloss.NewStyle().Foreground(errorColor).Bold(true).Padding(0, 1)
	s.NotificationWarning = lipgloss.NewStyle().Foreground(warning).Padding(0, 1)
	s.NotificationInfo = lipgloss.NewStyle().Foreground(info).Padding(0, 1)

	s.Content = lipgloss.NewStyle().Padding(1, 2)
	s.Help = lipgloss.NewStyle().Foreground(subtle).Padding(0, 1)
	s.Spinner = lipgloss.NewStyle().Foreground(highlight)
	s.Toast = styles.ToastStyle

	s.Title = lipgloss.NewStyle().Bold(true).Foreground(highlight)
	s.Subtle = lipgloss.NewStyle().Foreground(subtle)
	s.Highlight = lipgloss.NewStyle().Foreground(highlight)
	s.Error = lipgloss.NewStyle().Foreground(errorColor)
	s.Success = lipgloss.NewStyle().Foreground(success)
	s.Warning = lipgloss.NewStyle().Foreground(warning)

	return s
}

// Model is the main application model.
type Model struct {
	// Tab management
	activeTab TabID
	tabs      []Tab
	tabNames  []string

	// Shared state
	state    *State
	services *services.Manager
	commands *Commands
	keymap   KeyMap
	styles   Styles

	// UI components
	spinner spinner.Model

	// Window dimensions
	width  int
	height int

	// UI state
	showHelp bool
	ready    bool

	// Service subscription
	eventChannel chan services.ServiceEvent
}

// NewModel initializes a new application model.
func NewModel(mgr *services.Manager) *Model {
	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Create shared state
	state := NewState()

	// Create model
	m := &Model{
		activeTab: TabDashboard,
		tabNames:  []string{"Dashboard", "History", "Info"},
		tabs:      make([]Tab, 3), // Placeholder - tabs will be set externally
		state:     state,
		services:  mgr,
		commands:  NewCommands(mgr),
		keymap:    DefaultKeyMap(),
		styles:    DefaultStyles(),
		spinner:   s,
		showHelp:  false,
		ready:     false,
	}

	return m
}

// SetTabs sets the tabs for the model.
func (m *Model) SetTabs(tabs []Tab) {
	m.tabs = tabs
	if m.width > 0 && m.height > 0 {
		m.updateTabSizes()
	}
}

// GetState returns the application state.
func (m *Model) GetState() *State {
	return m.state
}

// GetServices returns the service manager.
func (m *Model) GetServices() *services.Manager {
	return m.services
}

// GetCommands returns the commands helper.
func (m *Model) GetCommands() *Commands {
	return m.commands
}

// GetKeyMap returns the key bindings.
func (m *Model) GetKeyMap() KeyMap {
	return m.keymap
}

// GetStyles returns the application styles.
func (m *Model) GetStyles() Styles {
	return m.styles
}

// GetActiveTab returns the currently active tab ID.
func (m *Model) GetActiveTab() TabID {
	return m.activeTab
}

// GetWidth returns the window width.
func (m *Model) GetWidth() int {
	return m.width
}

// GetHeight returns the window height.
func (m *Model) GetHeight() int {
	return m.height
}

// IsReady returns true if the model is ready (window size received).
func (m *Model) IsReady() bool {
	return m.ready
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	m.state.SetLoadingNotification("Loading...")

	cmds := []tea.Cmd{
		m.spinner.Tick,
		defaultTickCmd(),
	}

	if m.services != nil {
		cmds = append(cmds, subscribeToServicesCmd(m.services))
		cmds = append(cmds, loadInitialData(m.services))
	}

	for _, tab := range m.tabs {
		if tab != nil {
			cmds = append(cmds, tab.Init())
		}
	}

	return tea.Batch(cmds...)
}

// Update handles messages and updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg, tea.KeyMsg, spinner.TickMsg:
		if cmd := m.handleTeaMsg(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	default:
		if appCmds := m.handleAppMsg(msg); len(appCmds) > 0 {
			cmds = append(cmds, appCmds...)
		}
	}

	if cmd := m.updateActiveTab(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleTeaMsg(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)
	}
	return nil
}

func (m *Model) handleAppMsg(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case TickMsg:
		cmds = append(cmds, m.handleTick())
	case SubscriptionEventMsg:
		cmds = append(cmds, m.handleSubscriptionEvent(msg)...)
	case ServiceEventMsg:
		cmds = append(cmds, m.handleServiceEventMsg(msg)...)
	case AccountsLoadedMsg:
		m.handleAccountsLoaded(msg)
	case StatsLoadedMsg:
		m.handleStatsLoaded(msg)
	case QuotaRefreshedMsg:
		cmds = append(cmds, m.handleQuotaRefreshed(msg)...)
	case SwitchAccountResultMsg:
		cmds = append(cmds, m.handleSwitchAccountResult(msg)...)
	case DeleteAccountResultMsg:
		cmds = append(cmds, m.handleDeleteAccountResult(msg)...)
	case AddNotificationMsg:
		cmds = append(cmds, m.handleAddNotification(msg)...)
	case RemoveNotificationMsg:
		m.state.RemoveNotification(msg.ID)
	case ClearExpiredNotificationsMsg:
		m.state.ClearExpiredNotifications()
	case StartLoadingMsg:
		m.handleStartLoading(msg)
	case StopLoadingMsg:
		m.handleStopLoading(msg)
	case ErrorMsg:
		cmds = append(cmds, notifyErrorCmd(msg.Error.Error()))
	case RefreshMsg:
		cmds = append(cmds, m.handleRefresh(msg)...)
	case TabSwitchMsg:
		m.activeTab = msg.Tab
		m.updateTabSizes()
	case ToggleHelpMsg:
		m.showHelp = !m.showHelp
	}
	return cmds
}

func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	m.ready = true
	m.updateTabSizes()
}

func (m *Model) handleSpinnerTick(msg spinner.TickMsg) tea.Cmd {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return cmd
}

func (m *Model) handleTick() tea.Cmd {
	m.state.ClearExpiredNotifications()
	return defaultTickCmd()
}

func (m *Model) handleSubscriptionEvent(msg SubscriptionEventMsg) []tea.Cmd {
	var cmds []tea.Cmd
	m.eventChannel = msg.Channel
	cmds = append(cmds, waitForServiceEventCmd(m.eventChannel))
	if m.services != nil {
		cmds = append(cmds, loadAccountsCmd(m.services))
	}
	return cmds
}

func (m *Model) handleServiceEventMsg(msg ServiceEventMsg) []tea.Cmd {
	var cmds []tea.Cmd
	if cmd := m.handleServiceEvent(msg.Event); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if m.eventChannel != nil {
		cmds = append(cmds, waitForServiceEventCmd(m.eventChannel))
	}
	return cmds
}

func (m *Model) handleAccountsLoaded(msg AccountsLoadedMsg) {
	m.state.SetLoading("initial", false)
	m.state.SetLoading("accounts", false)
	m.state.SetAccounts(msg.Accounts)
	m.state.SetStats(msg.Stats)
	if !m.state.AnyLoading() {
		m.state.ClearLoadingNotification()
	}
}

func (m *Model) handleStatsLoaded(msg StatsLoadedMsg) {
	m.state.SetLoading("stats", false)
	m.state.SetStats(msg.Stats)
}

func (m *Model) handleQuotaRefreshed(msg QuotaRefreshedMsg) []tea.Cmd {
	var cmds []tea.Cmd
	m.state.SetLoading("quota", false)
	if msg.Error != nil {
		cmds = append(cmds, notifyErrorCmd(fmt.Sprintf("Failed to refresh quota: %v", msg.Error)))
	} else {
		cmds = append(cmds, notifySuccessCmd(fmt.Sprintf("Quota refreshed for %s", msg.Email)))
	}
	if m.services != nil {
		cmds = append(cmds, loadAccountsCmd(m.services))
	}
	return cmds
}

func (m *Model) handleSwitchAccountResult(msg SwitchAccountResultMsg) []tea.Cmd {
	var cmds []tea.Cmd
	if msg.Success {
		cmds = append(cmds, notifySuccessCmd(fmt.Sprintf("Switched to %s", msg.Email)))
		if m.services != nil {
			cmds = append(cmds, loadAccountsCmd(m.services))
		}
	} else {
		cmds = append(cmds, notifyErrorCmd(fmt.Sprintf("Failed to switch account: %v", msg.Error)))
	}
	return cmds
}

func (m *Model) handleDeleteAccountResult(msg DeleteAccountResultMsg) []tea.Cmd {
	var cmds []tea.Cmd
	if msg.Success {
		cmds = append(cmds, notifySuccessCmd(fmt.Sprintf("Deleted account %s", msg.Email)))
		if m.services != nil {
			cmds = append(cmds, loadAccountsCmd(m.services))
		}
	} else {
		cmds = append(cmds, notifyErrorCmd(fmt.Sprintf("Failed to delete account: %v", msg.Error)))
	}
	return cmds
}

func (m *Model) handleAddNotification(msg AddNotificationMsg) []tea.Cmd {
	var cmds []tea.Cmd
	id := m.state.AddNotification(msg.Type, msg.Message, msg.Duration)
	if msg.Duration > 0 {
		cmds = append(cmds, clearNotificationCmd(id, msg.Duration))
	}
	return cmds
}

func (m *Model) handleStartLoading(msg StartLoadingMsg) {
	m.state.SetLoading(msg.Resource, true)
	m.state.SetLoadingNotification("Refreshing...")
}

func (m *Model) handleStopLoading(msg StopLoadingMsg) {
	m.state.SetLoading(msg.Resource, false)
	if !m.state.AnyLoading() {
		m.state.ClearLoadingNotification()
	}
}

func (m *Model) handleRefresh(msg RefreshMsg) []tea.Cmd {
	var cmds []tea.Cmd
	if m.services == nil {
		return cmds
	}

	cmds = append(cmds, func() tea.Msg { return StartLoadingMsg(msg) })

	switch msg.Resource {
	case "all", "accounts":
		cmds = append(cmds, loadAccountsCmd(m.services))
	case "quota":
		cmds = append(cmds, refreshAllQuotaCmd(m.services))
	case "stats":
		cmds = append(cmds, loadStatsCmd(m.services))
	}
	return cmds
}

func (m *Model) updateActiveTab(msg tea.Msg) tea.Cmd {
	if int(m.activeTab) < len(m.tabs) && m.tabs[m.activeTab] != nil {
		var cmd tea.Cmd
		m.tabs[m.activeTab], cmd = m.tabs[m.activeTab].Update(msg)
		return cmd
	}
	return nil
}

func (m *Model) updateTabSizes() {
	contentHeight := m.height - 5
	contentHeight = max(0, contentHeight)

	for _, tab := range m.tabs {
		if tab != nil {
			tab.SetSize(m.width, contentHeight)
		}
	}
}

// handleKeyMsg handles keyboard input.
func (m *Model) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	// Global keybindings (work regardless of tab)
	switch {
	case key.Matches(msg, m.keymap.Quit):
		return tea.Quit

	case key.Matches(msg, m.keymap.Help):
		m.showHelp = !m.showHelp
		return nil

	case key.Matches(msg, m.keymap.Tab1):
		m.activeTab = TabDashboard
		m.updateTabSizes()
		return nil

	case key.Matches(msg, m.keymap.Tab2):
		m.activeTab = TabHistory
		m.updateTabSizes()
		return nil

	case key.Matches(msg, m.keymap.Tab3):
		m.activeTab = TabInfo
		m.updateTabSizes()
		return nil

	case key.Matches(msg, m.keymap.NextTab):
		if !m.showHelp {
			m.activeTab = TabID((int(m.activeTab) + 1) % len(m.tabs))
			m.updateTabSizes()
		}
		return nil

	case key.Matches(msg, m.keymap.PrevTab):
		if !m.showHelp {
			m.activeTab = TabID((int(m.activeTab) - 1 + len(m.tabs)) % len(m.tabs))
			m.updateTabSizes()
		}
		return nil

	case key.Matches(msg, m.keymap.Refresh):
		if m.services != nil {
			return tea.Batch(
				func() tea.Msg { return StartLoadingMsg{Resource: "accounts"} },
				loadAccountsCmd(m.services),
			)
		}
		return nil

	case key.Matches(msg, m.keymap.Escape):
		if m.showHelp {
			m.showHelp = false
			return nil
		}
	}

	// Let the tab handle other keys
	return nil
}

func (m *Model) handleServiceEvent(event services.ServiceEvent) tea.Cmd {
	switch e := event.(type) {
	case services.AccountsChangedEvent:
		if m.services != nil {
			return loadAccountsCmd(m.services)
		}

	case services.QuotaUpdatedEvent:
		if m.services != nil {
			return loadAccountsCmd(m.services)
		}

	case services.ProjectionUpdatedEvent:
		m.state.SetProjection(e.Email, e.Projection)
		return func() tea.Msg {
			return ProjectionUpdatedMsg{
				Email:      e.Email,
				Projection: e.Projection,
			}
		}

	case services.ErrorEvent:
		return notifyErrorCmd(fmt.Sprintf("[%s] %v", e.Service, e.Error))

	case services.StatsEvent:
		m.state.SetStats(e)
	}

	return nil
}

// View renders the application UI.
func (m *Model) View() string {
	var b strings.Builder

	if m.width > 0 {
		b.WriteString(m.renderNavbar())
		b.WriteString("\n")
	}

	if !m.ready {
		b.WriteString(m.styles.Content.Render(fmt.Sprintf("%s Loading...", m.spinner.View())))
		return b.String()
	}

	if int(m.activeTab) < len(m.tabs) && m.tabs[m.activeTab] != nil {
		b.WriteString(m.tabs[m.activeTab].View())
	} else {
		b.WriteString(m.renderPlaceholder())
	}

	mainView := b.String()

	if m.showHelp {
		// Render help modal
		helpView := m.renderHelp()
		mainView = m.overlayCentered(mainView, helpView)
	}

	notifications := m.renderNotifications()

	if len(notifications) > 0 {
		return m.overlayToasts(mainView, notifications)
	}

	return mainView
}

func (m *Model) overlayCentered(mainView string, overlay string) string {
	mainLines := strings.Split(mainView, "\n")
	overlayLines := strings.Split(overlay, "\n")

	overlayHeight := len(overlayLines)
	overlayWidth := lipgloss.Width(overlay)

	// Calculate center position
	y := (m.height - overlayHeight) / 2
	x := (m.width - overlayWidth) / 2

	if y < 0 {
		y = 0
	}
	if x < 0 {
		x = 0
	}

	for i, overlayLine := range overlayLines {
		mainY := y + i
		if mainY >= len(mainLines) {
			break
		}

		mainLine := mainLines[mainY]

		// Truncate main line to the start of the overlay
		left := ansi.Truncate(mainLine, x, "")

		// Calculate how much to cut from the left for the right part
		// We want to skip 'x + overlayWidth' visual cells
		right := ansi.TruncateLeft(mainLine, x+overlayWidth, "")

		// If the line was shorter than the overlay start, pad it
		if lipgloss.Width(left) < x {
			left += strings.Repeat(" ", x-lipgloss.Width(left))
		}

		mainLines[mainY] = left + overlayLine + right
	}

	return strings.Join(mainLines, "\n")
}

func (m *Model) renderNavbar() string {
	var tabs []string

	for i, name := range m.tabNames {
		if TabID(i) == m.activeTab {
			tabs = append(tabs, m.styles.ActiveTab.Render(fmt.Sprintf("[%d] %s", i+1, name)))
		} else {
			tabs = append(tabs, m.styles.InactiveTab.Render(fmt.Sprintf(" %d  %s", i+1, name)))
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	return m.styles.TabBar.Width(m.width).Render(tabBar)
}

func (m *Model) renderNotifications() []string {
	notifications := m.state.GetNotifications()
	if len(notifications) == 0 {
		return nil
	}

	var toasts []string
	for _, n := range notifications {
		var style lipgloss.Style
		var prefix string

		switch n.Type {
		case NotificationSuccess:
			style = m.styles.NotificationSuccess
			prefix = "[OK]"
		case NotificationError:
			style = m.styles.NotificationError
			prefix = "[ERR]"
		case NotificationWarning:
			style = m.styles.NotificationWarning
			prefix = "[WARN]"
		case NotificationInfo:
			style = m.styles.NotificationInfo
			prefix = "[INFO]"
		case NotificationLoading:
			style = m.styles.NotificationInfo
			prefix = m.spinner.View()
		}

		content := style.Render(fmt.Sprintf("%s %s", prefix, n.Message))
		toast := m.styles.Toast.Render(content)
		toasts = append(toasts, toast)
	}

	return toasts
}

func (m *Model) overlayToasts(mainView string, toasts []string) string {
	if len(toasts) == 0 {
		return mainView
	}

	toastStack := lipgloss.JoinVertical(lipgloss.Right, toasts...)
	toastLines := strings.Split(toastStack, "\n")
	mainLines := strings.Split(mainView, "\n")

	toastWidth := lipgloss.Width(toastStack)
	startX := max(m.width-toastWidth-2, 0)

	startY := 2

	for i, toastLine := range toastLines {
		lineIdx := startY + i
		if lineIdx >= len(mainLines) {
			break
		}

		mainLine := mainLines[lineIdx]
		mainLineWidth := lipgloss.Width(mainLine)

		if mainLineWidth < startX {
			padding := strings.Repeat(" ", startX-mainLineWidth)
			mainLines[lineIdx] = mainLine + padding + toastLine
		} else {
			truncated := ansi.Truncate(mainLine, startX, "")
			mainLines[lineIdx] = truncated + toastLine
		}
	}

	return strings.Join(mainLines, "\n")
}

func (m *Model) renderHelp() string {
	var lines []string

	lines = append(lines, m.styles.Title.Render("Keyboard Shortcuts"))
	lines = append(lines, "")

	lines = append(lines, m.styles.Highlight.Render("Navigation"))
	lines = append(lines, "  1-2        Switch tabs")
	lines = append(lines, "  Tab        Next tab")
	lines = append(lines, "  Shift+Tab  Previous tab")
	lines = append(lines, "")

	lines = append(lines, m.styles.Highlight.Render("Actions"))
	lines = append(lines, "  r          Refresh data")
	lines = append(lines, "  ?          Toggle help")
	lines = append(lines, "  q/Ctrl+C   Quit")
	lines = append(lines, "")

	lines = append(lines, m.styles.Highlight.Render("Lists"))
	lines = append(lines, "  j/k, ↑/↓   Move up/down")
	lines = append(lines, "  Enter      Select item")
	lines = append(lines, "  /          Filter")
	lines = append(lines, "")

	if int(m.activeTab) < len(m.tabs) && m.tabs[m.activeTab] != nil {
		tabHelp := m.tabs[m.activeTab].ShortHelp()
		if len(tabHelp) > 0 {
			lines = append(lines, m.styles.Highlight.Render(fmt.Sprintf("%s Tab", m.tabNames[m.activeTab])))
			for _, binding := range tabHelp {
				lines = append(lines, fmt.Sprintf("  %-10s %s", binding.Help().Key, binding.Help().Desc))
			}
		}
	}

	lines = append(lines, "")
	lines = append(lines, m.styles.Subtle.Render("Press ? or Esc to close"))

	return styles.HelpPanelStyle.Render(strings.Join(lines, "\n"))
}

func (m *Model) renderPlaceholder() string {
	content := fmt.Sprintf(
		"Tab %d: %s\n\n%s",
		m.activeTab+1,
		m.tabNames[m.activeTab],
		m.styles.Subtle.Render("This tab is not yet implemented."),
	)
	return m.styles.Content.Render(content)
}

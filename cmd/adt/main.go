// Package main is the entry point for the Antigravity Dashboard TUI application.
// It initializes configuration, services, and runs the Bubble Tea program.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/app"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/config"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/services"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/tabs/dashboard"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/tabs/history"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/ui/tabs/info"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/version"
)

func main() {
	// Handle version flag
	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Println(version.Info())
		os.Exit(0)
	}

	// Handle help flag
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		printUsage()
		os.Exit(0)
	}

	// Run the application
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run contains the main application logic, separated for cleaner error handling.
func run() error {
	// 1. Load configuration from .env files and environment variables
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// 2. Initialize the service manager
	// This starts all background services: accounts and quota fetching
	svcManager, err := services.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Ensure cleanup on exit
	defer func() {
		if closeErr := svcManager.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: error closing services: %v\n", closeErr)
		}
	}()

	// 3. Create the root Bubble Tea model
	model := app.NewModel(svcManager)

	// 4. Initialize tabs with shared state and services
	// Each tab receives the shared application state for consistent data access
	state := model.GetState()
	tabs := []app.Tab{
		dashboard.New(state),           // Tab 0: Dashboard - quota overview
		history.New(state, svcManager), // Tab 1: History - usage history
		info.New(state, cfg),           // Tab 2: Info - configuration and app info
	}
	model.SetTabs(tabs)

	// 5. Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 6. Create and configure the Bubble Tea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer (full terminal)
		tea.WithMouseCellMotion(), // Enable mouse support for selection
	)

	// 7. Handle signals in a separate goroutine
	go func() {
		<-sigChan
		// Send quit message to the program
		p.Send(tea.Quit())
	}()

	// 8. Run the TUI program
	// This blocks until the user quits or an error occurs
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}

// printUsage prints the command-line usage information.
func printUsage() {
	fmt.Println(`Antigravity Dashboard TUI - Multi-account Google Cloud quota monitor

Usage:
  antigravity [flags]

Flags:
  -h, --help      Show this help message
  -v, --version   Show version information

Keyboard Shortcuts:
  1-3             Switch between tabs (Dashboard, History, Info)
  Tab/Shift+Tab   Navigate between tabs
  j/k, Up/Down    Navigate lists
  Enter           Select/confirm
  r               Refresh data
  ?               Toggle help
  q, Ctrl+C       Quit

Environment Variables:
  DATABASE_PATH           SQLite database path
  ACCOUNTS_PATH           Accounts JSON file path
  QUOTA_REFRESH_INTERVAL  Quota polling interval (default: 30s)

Configuration:
  The application looks for .env files in the following locations:
  - Current directory
  - ~/.config/opencode/antigravity-tui/.env
  - ~/.config/opencode/.env
  - ~/.antigravity/.env

For more information, visit: https://github.com/j-veylop/antigravity-dashboard-tui`)
}

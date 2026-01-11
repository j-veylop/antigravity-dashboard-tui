# Application Core Module

## Purpose
The `internal/app` package implements the core logic of the Bubble Tea application. It manages the global application state, coordinates between different UI tabs, and handles communication between services and the TUI.

## Components
- **`Model` (`model.go`)**: The root `tea.Model`. It implements `Init`, `Update`, and `View`. It manages the active tab, window dimensions, and global keybindings.
- **`State` (`state.go`)**: A thread-safe container for shared application data, including:
    - Account lists and active account.
    - Global statistics.
    - Quota projections.
    - Notifications and loading states.
- **`Messages` (`messages.go`)**: Defines all custom Bubble Tea messages (`tea.Msg`) used for internal communication (e.g., `AccountsLoadedMsg`, `QuotaRefreshedMsg`, `ServiceEventMsg`).
- **`Commands` (`commands.go`)**: Helper functions for creating `tea.Cmd` that wrap service calls (e.g., `loadAccountsCmd`, `refreshAllQuotaCmd`).
- **`Tab` Interface**: Defines the contract for UI pages (`Init`, `Update`, `View`, `SetSize`).

## Interactions
- **`internal/services`**: Subscribes to service events and triggers background tasks via the `Manager`.
- **`internal/ui/tabs/*`**: Delegates rendering and tab-specific updates to the implementations in `internal/ui/tabs`.
- **`internal/ui/styles`**: Uses centralized Lip Gloss styles for consistent rendering.
- **`internal/models`**: Uses core data structures for state management.

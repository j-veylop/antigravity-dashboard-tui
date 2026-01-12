# Antigravity Dashboard TUI - Codebase Summary

This document provides a hierarchical overview of the `antigravity-dashboard-tui` project structure, its modules, and component responsibilities.

## Project Overview

- **Module Name**: `github.com/j-veylop/antigravity-dashboard-tui`
- **Description**: A multi-account Google Cloud quota monitor with a terminal user interface (TUI) built using the Bubble Tea framework.
- **Language**: Go 1.25.5

---

## Hierarchical Structure

### 1. Command Entry Points (`cmd/`)

The `cmd/` directory contains the executable entry points for the application.

- **`cmd/adt/`** ([Command Module Documentation](./cmd/adt/AGENTS.md)):
  - `main.go`: The primary entry point for the `adt` application. It handles CLI flags, initializes configuration and database, starts background services, and launches the Bubble Tea TUI.

### 2. Internal Packages (`internal/`)

Core application logic is contained within the `internal/` directory.

#### `internal/app/` ([Application Core Documentation](./internal/app/AGENTS.md))

Manages the root application model and global state.

- **`model.go`**: Implements the main `tea.Model`, coordinating tab switching, window resizing, and global keybindings.
- **`state.go`**: Manages shared application state, including account lists, active notifications, and loading indicators.
- **`commands.go`**: Helper for creating Bubble Tea commands (`tea.Cmd`).

#### `internal/services/` ([Services Documentation](./internal/services/AGENTS.md))

Orchestrates background tasks and business logic.

- **`manager.go`**: A central service orchestrator that manages communication between services and broadcasts events to the UI.
- **`accounts/`**: Manages account definitions and persistent storage in JSON files.
- **`quota/`**: Handles OAuth2-based polling of Google Cloud quotas for multiple accounts.
- **`projection/`**: Analyzes historical usage data to calculate consumption rates and predict exhaustion times.

#### `internal/db/` ([Database Documentation](./internal/db/AGENTS.md))

Handles data persistence using SQLite.

- **`db.go`**: Connection management and schema initialization (WAL mode enabled).
- **`queries.go`**, **`historical_queries.go`**, **`projection_queries.go`**: SQL implementations for recording snapshots, usage events, and performing analytical queries.

#### `internal/ui/` ([User Interface Documentation](./internal/ui/AGENTS.md))

- **`tabs/`**: Contains the individual page implementations:
  - **`dashboard/`**: Provides an overview of current quotas with real-time progress bars and reset timers.
  - **`history/`**: Displays analytical views including consumption charts, rate limit hits, and hourly patterns.
  - **`info/`**: Shows application configuration and version details.
  - **`accounts/`**: Management interface for Google Cloud accounts (currently unused/WIP).
- **`components/`**: Reusable TUI widgets like gradient progress bars, charts, and spinners.
- **`styles/`**: Centralized Lip Gloss styles and color schemes.

#### `internal/config/` ([Configuration Documentation](./internal/config/AGENTS.md))

- **`config.go`**: Loads application settings from environment variables and `.env` files across multiple locations (local, `~/.config`).

#### `internal/models/` ([Models Documentation](./internal/models/AGENTS.md))

- Defines the core data structures used across all packages (e.g., `Account`, `QuotaInfo`, `AccountProjection`, `HistoryStats`).

#### `internal/version/`

- Manages application versioning information.

#### `internal/logger/`

- Provides structured logging utilities for application debugging using `slog`.

### 3. Documentation (`docs/`)

- **`ARCHITECTURE.md`**: Provides high-level architectural overview and design patterns used in the project.

---

## Component Relationships

1. **`cmd/adt`** initializes the **`services.Manager`**.
2. **`services.Manager`** starts background polling in **`services/quota`** and **`services/accounts`**.
3. Services record data into **`internal/db`** and send **`ServiceEvents`** to the **`internal/app`** model.
4. **`internal/app`** updates the **`internal/ui/tabs`** which render the visual representation using **`internal/ui/components`**.

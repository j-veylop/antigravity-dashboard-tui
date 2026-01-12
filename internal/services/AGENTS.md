# Services Module

## Purpose

The `internal/services` package handles the background business logic, including account management, Google Cloud quota polling, and historical data analysis for projections.

## Components

- **`Manager` (`manager.go`)**: The central orchestrator. It:
  - Initializes and holds references to all sub-services.
  - Routes events from sub-services to subscribers (primarily the TUI).
  - Provides a unified API for the application to interact with background logic.
- **`accounts` Service**: Manages the persistent accounts JSON file. Uses `fsnotify` to watch for external changes.
- **`quota` Service**: Handles OAuth2 token management and fetches real-time quota data from Google APIs.
- **`projection` Service**: Analyzes historical data stored in the database to calculate consumption rates and predict quota exhaustion.

## Interactions

- **`internal/db`**: Sub-services (like `projection`) use the database to store and retrieve historical usage snapshots.
- **`internal/config`**: Receives configuration settings (paths, intervals, API credentials) during initialization.
- **`internal/models`**: Uses shared domain models for data transfer and persistence.
- **`internal/app`**: Emits `ServiceEvent`s that are wrapped into Bubble Tea messages for UI updates.

# Configuration Module

## Purpose

The `internal/config` package handles application settings and environment variable management. It provides a centralized way to load and access configuration parameters.

## Components

- **`Config` (`config.go`)**:
  - Defines the main `Config` struct (database paths, account paths, API credentials).
  - Implements `Load()` to read from environment variables and `.env` files.
  - Searches multiple locations for `.env` (cwd, `~/.config/opencode`, etc.).
- **`Constants` (`constants.go`)**:
  - Handles automated discovery of Google OAuth credentials from the `opencode-antigravity-auth` tool.
  - Parses `constants.d.ts` if available to simplify user setup.

## Interactions

- **`cmd/adt`**: Initialized at the very beginning of the application lifecycle.
- **`internal/services`**: Configuration values are passed to the `Manager` to initialize sub-services.
- **`internal/ui/tabs/info`**: Uses configuration data to display environment information to the user.

# User Interface Module

## Purpose
The `internal/ui` package contains all the visual components and layout logic for the TUI. It uses Lip Gloss for styling and follows the Bubble Tea MVU pattern.

## Components
- **`tabs/`**: individual page implementations:
    - **`dashboard`**: Real-time quota monitoring with progress bars.
    - **`history`**: Analytical views, charts, and detailed usage logs.
    - **`info`**: Displays application configuration, version, and help.
    - **`accounts`**: Management interface for Google Cloud accounts.
- **`components/`**: Reusable TUI widgets:
    - **`quota_bar`**: Gradient-colored progress bars for quota visualization.
    - **`chart`**: ASCII-based line charts for historical trends.
    - **`spinner`**: Loading indicators.
- **`styles` (`styles.go`)**: Centralized Lip Gloss style definitions (colors, borders, padding) to ensure a consistent look and feel.

## Interactions
- **`internal/app`**: Tab models implement the `app.Tab` interface and are managed by the root `app.Model`.
- **`internal/services`**: Tabs often receive the `Manager` or `AppState` to access data for rendering.
- **`internal/models`**: Uses domain models (e.g., `AccountWithQuota`, `AccountProjection`) to populate UI elements.

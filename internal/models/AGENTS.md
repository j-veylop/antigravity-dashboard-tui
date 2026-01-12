# Models Module

## Purpose

The `internal/models` package defines the core domain types and data structures used across the entire application. It ensures consistent data representation between services, the database, and the UI.

## Components

- **`Account` (`account.go`)**: Represents a Google Cloud account with OAuth credentials and project identifiers.
- **`Quota` (`quota.go`)**: Defines `QuotaInfo` and `ModelQuota` for representing real-time API usage.
- **`Projection` (`projection.go`)**: Structs for analytical projections (e.g., `AccountProjection`, `ModelProjection`, `HistoricalContext`).
- **`History` (`history.go`)**: Data structures for time-series analysis and historical statistics.
- **`Stats` (`stats.go`)**: Aggregated global application statistics.
- **`API Call` (`api_call.go`)**: Detailed record of individual API interactions (recorded for history).

## Interactions

- **Used by ALL packages**: This is the most foundational package in the codebase.
- **`internal/db`**: Used for Object-Relational Mapping (ORM) between SQLite and Go.
- **`internal/services`**: Used for passing data between background workers.
- **`internal/ui`**: Used for populating views with structured data.

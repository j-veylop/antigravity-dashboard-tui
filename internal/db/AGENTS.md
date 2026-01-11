# Database Module

## Purpose
The `internal/db` package manages data persistence using SQLite. it is responsible for storing historical quota snapshots, API call logs, and session-based usage data for analytical purposes.

## Components
- **`DB` (`db.go`)**: manages the SQLite connection, initializes the schema, and configures performance pragmas (WAL mode, cache size).
- **`Queries` (`queries.go`)**: Implements basic CRUD operations for account status and API calls.
- **`Historical Queries` (`historical_queries.go`)**: Provides analytical queries for retrieving usage stats over specific time ranges.
- **`Projection Queries` (`projection_queries.go`)**: Handles aggregation of quota snapshots into 5-minute buckets (`quota_snapshots_agg`) for long-term trend analysis.
- **`Migrations` (`migrations.go`)**: Handles schema updates and legacy data fixes.

## Interactions
- **`internal/services/projection`**: Heavily utilized by the projection service to calculate consumption velocities.
- **`internal/models`**: maps database rows to application-level structs (e.g., `QuotaSnapshot`, `AggregatedSnapshot`).
- **`internal/services/manager`**: Initialized by the manager and passed to sub-services that require persistence.

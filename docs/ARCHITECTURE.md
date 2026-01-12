# Architecture Documentation

## Overview

Antigravity Dashboard TUI is a terminal-based quota monitoring application for tracking Claude and Gemini API usage across multiple Google Cloud accounts. It's built with Go using the Bubble Tea framework for the TUI.

## System Architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│                         Main Application                          │
│                    (cmd/adt/main.go)                             │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                    App Model (Bubble Tea)                        │
│                  (internal/app/model.go)                         │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │  Dashboard   │  │   History    │  │   Accounts   │          │
│  │     Tab      │  │     Tab      │  │     Tab      │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└───────────┬──────────────────────────────────────────┬──────────┘
            │                                           │
┌───────────▼──────────┐                   ┌───────────▼──────────┐
│  Service Manager     │                   │   Shared State       │
│  (Background Tasks)  │                   │  (Thread-Safe)       │
│                      │                   │                      │
│  ┌────────────────┐  │                   │  ┌────────────────┐  │
│  │ Accounts Svc   │  │                   │  │ Account Data   │  │
│  │ Quota Svc      │  │                   │  │ Quota Stats    │  │
│  │ Projection Svc │  │                   │  │ Projections    │  │
│  └────────────────┘  │                   │  └────────────────┘  │
└──────────┬───────────┘                   └─────────┬────────────┘
           │                                         │
           │         ┌──────────────────────────────┘
           │         │
┌──────────▼─────────▼─────────┐
│       SQLite Database         │
│    (internal/db)              │
│                               │
│  Tables:                      │
│  - api_calls                  │
│  - account_status             │
│  - quota_snapshots            │
│  - quota_snapshots_agg        │
│  - session_events             │
└───────────────────────────────┘
```

## Core Components

### 1. Main Application (cmd/adt)

**Responsibilities:**

- Initialize configuration
- Set up database connection
- Bootstrap the Bubble Tea application
- Handle graceful shutdown

**Key Files:**

- `cmd/adt/main.go` - Application entry point

### 2. App Layer (internal/app)

**Responsibilities:**

- Manage Bubble Tea program lifecycle
- Route messages between tabs
- Coordinate UI updates
- Handle global key bindings

**Key Files:**

- `internal/app/model.go` - Main Bubble Tea model
- `internal/app/state.go` - Shared application state

**Design Pattern:** Model-View-Update (MVU)

### 3. UI Tabs (internal/ui/tabs)

Each tab is an independent Bubble Tea component with its own model, update, and view logic.

#### Dashboard Tab

- Display quota bars for Claude and Gemini
- Show reset countdown timers
- Display tier information (FREE/PRO)
- Real-time quota animations

#### History Tab

- Scrollable list of historical quota snapshots
- Filter and search functionality
- Display rate limit transitions
- Session exhaustion statistics

#### Accounts Tab  

- Manage multiple Google accounts
- Add/remove accounts
- Switch active account
- Display account status

**Key Pattern:** Each tab implements:

```go
type Tab interface {
    Init() tea.Cmd
    Update(tea.Msg) (Tab, tea.Cmd)
    View() string
}
```

### 4. Services Layer (internal/services)

Background services run independently and communicate via channels.

#### Accounts Service (`services/accounts`)

- **Responsibilities:**
  - Load accounts from JSON file
  - Watch for file changes (fsnotify)
  - Manage active account selection
  - Persist account changes
- **Events:** AccountsLoaded, AccountAdded, AccountDeleted, ActiveAccountChanged

#### Quota Service (`services/quota`)

- **Responsibilities:**
  - Fetch quota data from Google OAuth API
  - Refresh quota at configurable intervals
  - Detect rate limiting
  - Calculate tier (FREE/PRO)
- **Dependencies:** Google OAuth2, HTTP client

#### Projection Service (`services/projection`)

- **Responsibilities:**
  - Aggregate quota snapshots
  - Calculate consumption rates
  - Project quota exhaustion times
  - Clean up old data
- **Aggregation:** 5-minute buckets for efficient querying

#### Service Manager

- **Responsibilities:**
  - Start/stop all services
  - Coordinate service communication
  - Handle service errors
  - Manage graceful shutdown

### 5. Database Layer (internal/db)

**Technology:** SQLite with WAL mode for concurrency

#### Schema

**api_calls** - Individual API call logs (future use)

```sql
- id, timestamp, email, model, provider
- input_tokens, output_tokens, cache_tokens
- duration_ms, status_code, error
```

**account_status** - Current account quota state

```sql
- email, claude_quota, gemini_quota, total_quota
- tier, is_rate_limited, last_updated
- claude_reset_sec, gemini_reset_sec
```

**quota_snapshots** - Raw point-in-time quota readings

```sql
- id, email, timestamp
- claude_quota, gemini_quota, total_quota
- tier, is_rate_limited
```

**quota_snapshots_agg** - Aggregated 5-minute bucket data

```sql
- email, bucket_time, session_id
- claude_quota, gemini_quota, total_quota
- claude_consumed, gemini_consumed, total_consumed
- tier, is_rate_limited
```

**session_events** - Session lifecycle tracking

```sql
- id, email, session_id, event_type
- timestamp, metadata
```

#### Query Patterns

**Historical Queries** (`internal/db/historical_queries.go`)

- Monthly statistics
- Usage patterns (hourly, daily)
- Rate limit transitions
- Session exhaustion analysis

**Projection Queries** (`internal/db/projection_queries.go`)

- Consumption rate calculation
- Time-to-exhaustion projection
- Session-based aggregation

### 6. Models (internal/models)

Domain models shared across the application:

- `Account` - User account with OAuth credentials
- `QuotaInfo` - Current quota status
- `QuotaSnapshot` - Point-in-time quota reading
- `ModelProjection` - Consumption rate projections
- `RateLimitStats` - Rate limit hit statistics
- `ExhaustionStats` - Session exhaustion analytics

## Data Flow

### Quota Refresh Flow

```text
1. Timer triggers quota refresh
   ↓
2. Quota Service fetches from Google API
   ↓
3. Service updates Database (account_status, quota_snapshots)
   ↓
4. Event sent to App Model
   ↓
5. App updates Shared State
   ↓
6. Dashboard Tab re-renders with new data
```

### Account File Watch Flow

```text
1. User edits antigravity-accounts.json externally
   ↓
2. fsnotify detects file change
   ↓
3. Accounts Service reloads accounts
   ↓
4. Event sent to App Model
   ↓
5. App updates Shared State
   ↓
6. All tabs receive updated account data
```

### Projection Calculation Flow

```text
1. Projection Service wakes up (timer-based)
   ↓
2. Fetch recent quota snapshots from DB
   ↓
3. Aggregate into 5-minute buckets
   ↓
4. Calculate consumption rates
   ↓
5. Project exhaustion time
   ↓
6. Store aggregated snapshot
   ↓
7. Cleanup old raw snapshots
   ↓
8. Event sent to Dashboard
```

## Concurrency & Thread Safety

### Service Communication

- **Pattern:** Channels for inter-service communication
- **Event Channel:** Buffered channel (100 capacity) for service events
- **Stop Channel:** Used for graceful shutdown coordination

### Shared State

- **Pattern:** Read-write mutex (sync.RWMutex)
- **Read Operations:** Multiple concurrent readers
- **Write Operations:** Exclusive access
- **Thread-Safe Methods:** All state access methods use proper locking

### File Watching

- **Library:** fsnotify
- **Debouncing:** 100ms debounce to handle rapid file changes
- **Pattern:** Single watcher per file with event aggregation

## Configuration

### Configuration Sources (Priority Order)

1. Command-line flags (future)
2. Environment variables
3. `.env` file
4. `antigravity-accounts.json` (for OAuth credentials)
5. Default values

### Key Configuration

- `DATABASE_PATH` - SQLite database location
- `ACCOUNTS_PATH` - Accounts JSON file location  
- `QUOTA_REFRESH_INTERVAL` - How often to poll Google API (default: 30s)
- `GOOGLE_CLIENT_ID` - OAuth client ID
- `GOOGLE_CLIENT_SECRET` - OAuth client secret

## Error Handling

### Strategy

- **Database Errors:** Logged and returned to caller
- **Service Errors:** Sent via error event channel
- **UI Errors:** Displayed as notifications in the TUI
- **Graceful Degradation:** Continue operation when non-critical components fail

### Logging

- **Library:** Custom logger (internal/logger)
- **Levels:** ERROR, WARN, INFO, DEBUG
- **Output:** Stderr to not interfere with TUI rendering

## Performance Considerations

### Database Optimization

- **Indexes:** On email, timestamp, bucket_time columns
- **WAL Mode:** Enables concurrent reads during writes
- **Aggregation:** 5-minute buckets reduce query load
- **Cleanup:** Periodic removal of old raw snapshots

### UI Rendering

- **Debouncing:** Limit re-renders during rapid updates
- **Lazy Loading:** Only render visible content
- **String Building:** Efficient string concatenation for views
- **Animation:** Smooth quota bar animations with lerp

### Memory Management

- **Bounded Channels:** Prevent unbounded memory growth
- **Snapshot Cleanup:** Automatic deletion of old data
- **Connection Pooling:** SQLite connection reuse

## Testing Strategy

### Unit Tests

- Models: Domain logic and transformations
- Services: Business logic with mocked dependencies
- Database: Query correctness with in-memory SQLite

### Integration Tests

- Service coordination
- End-to-end data flows
- File watching behavior

### Current Coverage

- config: 80.5%
- accounts: 67.8%
- models: 48.4%
- db: 43.9%
- projection: 86.7%

**Target:** 70%+ coverage for all packages

## Build & Deployment

### Build Process

```bash
make build       # Build binary
make test        # Run tests with race detector
make lint        # Run golangci-lint
make coverage    # Generate coverage report
make check       # Full pre-commit check
```

### CI/CD

- **Platform:** GitHub Actions
- **Checks:** Lint, test (with race detector), build
- **Artifacts:** Binary uploaded on successful build
- **Coverage:** Automatic upload to Codecov

### Release

```bash
make release     # Build optimized binary (future)
```

## Future Enhancements

### Planned Features

1. **Real-time Alerts** - Desktop notifications for rate limits
2. **Historical Charts** - Visual quota trends
3. **Export Data** - CSV/JSON export of usage history
4. **Multiple Dashboards** - Compare accounts side-by-side
5. **API Call Logging** - Track individual API calls (schema exists)

### Technical Improvements

1. **Metrics** - Prometheus-compatible metrics endpoint
2. **Configuration UI** - In-app settings management
3. **Backup/Restore** - Database backup functionality
4. **Plugin System** - Extensible quota providers
5. **Remote Sync** - Optional cloud sync for multi-device use

## Troubleshooting

### Common Issues

#### Database locked

- Ensure only one instance is running
- Check WAL mode is enabled
- Verify file permissions

#### Quota not refreshing

- Check OAuth credentials
- Verify network connectivity
- Check refresh interval configuration

#### File watch not working

- Ensure fsnotify supports your filesystem
- Check file permissions
- Verify file path configuration

### Debug Mode

```bash
export LOG_LEVEL=DEBUG
./adt
```

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for development guidelines.

## References

- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [SQLite WAL Mode](https://www.sqlite.org/wal.html)
- [fsnotify](https://github.com/fsnotify/fsnotify)
- [Google OAuth2](https://developers.google.com/identity/protocols/oauth2)

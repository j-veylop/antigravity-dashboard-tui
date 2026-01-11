// Package db manages the database connection
package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	// Import modernc.org/sqlite as a blank import to register the driver
	_ "modernc.org/sqlite"
	// sqlite driver
)

// DB wraps the SQL database connection with application-specific methods.
type DB struct {
	*sql.DB
	path string
}

// New creates a new database connection and initializes the schema.
func New(path string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Open database connection
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := sqlDB.PingContext(context.Background()); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	db := &DB{
		DB:   sqlDB,
		path: path,
	}

	// Configure database
	if err := db.configure(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to configure database: %w", err)
	}

	// Create schema
	if err := db.createSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// Fix legacy time formats
	if err := db.FixLegacyTimeFormats(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to fix legacy time formats: %w", err)
	}

	return db, nil
}

// Path returns the database file path.
func (db *DB) Path() string {
	return db.path
}

// configure sets up database pragmas for optimal performance.
func (db *DB) configure() error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000", // 64MB cache
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
		"PRAGMA temp_store=MEMORY",
	}

	for _, pragma := range pragmas {
		if _, err := db.ExecContext(context.Background(), pragma); err != nil {
			return fmt.Errorf("failed to execute %s: %w", pragma, err)
		}
	}

	return nil
}

func (db *DB) createSchema() error {
	if err := db.createAPICallsTable(); err != nil {
		return err
	}
	if err := db.createAccountStatusTable(); err != nil {
		return err
	}
	if err := db.createSessionEventsTable(); err != nil {
		return err
	}
	if err := db.createQuotaSnapshotsTable(); err != nil {
		return err
	}
	return db.createAggregatedSnapshotsTable()
}

func (db *DB) createAPICallsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS api_calls (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		email TEXT NOT NULL,
		model TEXT NOT NULL,
		provider TEXT NOT NULL DEFAULT 'anthropic',
		input_tokens INTEGER DEFAULT 0,
		output_tokens INTEGER DEFAULT 0,
		cache_read_tokens INTEGER DEFAULT 0,
		cache_write_tokens INTEGER DEFAULT 0,
		duration_ms INTEGER DEFAULT 0,
		status_code INTEGER DEFAULT 200,
		error TEXT,
		request_id TEXT,
		session_id TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_api_calls_timestamp ON api_calls(timestamp);
	CREATE INDEX IF NOT EXISTS idx_api_calls_email ON api_calls(email);
	CREATE INDEX IF NOT EXISTS idx_api_calls_session ON api_calls(session_id);
	`
	_, err := db.ExecContext(context.Background(), query)
	return err
}

func (db *DB) createAccountStatusTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS account_status (
		email TEXT PRIMARY KEY,
		claude_quota REAL,
		gemini_quota REAL,
		total_quota REAL DEFAULT 0,
		tier TEXT DEFAULT 'UNKNOWN',
		is_rate_limited INTEGER DEFAULT 0,
		last_error TEXT,
		last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
		claude_reset_sec INTEGER,
		gemini_reset_sec INTEGER
	);
	`
	_, err := db.ExecContext(context.Background(), query)
	return err
}

func (db *DB) createSessionEventsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS session_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		event_type TEXT NOT NULL,
		email TEXT,
		metadata TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_session_events_session ON session_events(session_id);
	CREATE INDEX IF NOT EXISTS idx_session_events_timestamp ON session_events(timestamp);
	`
	_, err := db.ExecContext(context.Background(), query)
	return err
}

func (db *DB) createQuotaSnapshotsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS quota_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL,
		claude_quota REAL,
		gemini_quota REAL,
		total_quota REAL DEFAULT 0,
		tier TEXT DEFAULT 'UNKNOWN',
		is_rate_limited INTEGER DEFAULT 0,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_quota_snapshots_email ON quota_snapshots(email);
	CREATE INDEX IF NOT EXISTS idx_quota_snapshots_timestamp ON quota_snapshots(timestamp);
	`
	_, err := db.ExecContext(context.Background(), query)
	return err
}

func (db *DB) createAggregatedSnapshotsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS quota_snapshots_agg (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL,
		bucket_time DATETIME NOT NULL,
		claude_quota_avg REAL DEFAULT 0,
		gemini_quota_avg REAL DEFAULT 0,
		claude_consumed REAL DEFAULT 0,
		gemini_consumed REAL DEFAULT 0,
		sample_count INTEGER DEFAULT 1,
		session_id TEXT,
		tier TEXT DEFAULT 'UNKNOWN',
		year INTEGER GENERATED ALWAYS AS (CAST(strftime('%Y', bucket_time) AS INTEGER)) STORED,
		month INTEGER GENERATED ALWAYS AS (CAST(strftime('%m', bucket_time) AS INTEGER)) STORED,
		week INTEGER GENERATED ALWAYS AS (CAST(strftime('%W', bucket_time) AS INTEGER)) STORED,
		day_of_week INTEGER GENERATED ALWAYS AS (CAST(strftime('%w', bucket_time) AS INTEGER)) STORED,
		hour INTEGER GENERATED ALWAYS AS (CAST(strftime('%H', bucket_time) AS INTEGER)) STORED,
		UNIQUE(email, bucket_time)
	);
	CREATE INDEX IF NOT EXISTS idx_agg_email_time ON quota_snapshots_agg(email, bucket_time);
	CREATE INDEX IF NOT EXISTS idx_agg_year_month ON quota_snapshots_agg(email, year, month);
	CREATE INDEX IF NOT EXISTS idx_agg_session ON quota_snapshots_agg(session_id);
	CREATE INDEX IF NOT EXISTS idx_agg_dow_hour ON quota_snapshots_agg(email, day_of_week, hour);
	`
	_, err := db.ExecContext(context.Background(), query)
	return err
}

// Close closes the database connection gracefully.
func (db *DB) Close() error {
	// Checkpoint WAL before closing
	_, _ = db.ExecContext(context.Background(), "PRAGMA wal_checkpoint(TRUNCATE)")
	return db.DB.Close()
}

// Vacuum performs database maintenance to reclaim space.
func (db *DB) Vacuum() error {
	_, err := db.ExecContext(context.Background(), "VACUUM")
	return err
}

package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/logger"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

// InsertAPICall logs an API call to the database.
func (db *DB) InsertAPICall(call *models.APICall) error {
	query := `
		INSERT INTO api_calls (
			timestamp, email, model, provider, input_tokens, output_tokens,
			cache_read_tokens, cache_write_tokens, duration_ms, status_code,
			error, request_id, session_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	timestamp := call.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	result, err := db.ExecContext(context.Background(), query,
		timestamp.Format("2006-01-02 15:04:05"),
		call.Email,
		call.Model,
		call.Provider,
		call.InputTokens,
		call.OutputTokens,
		call.CacheReadTokens,
		call.CacheWriteTokens,
		call.DurationMs,
		call.StatusCode,
		nullString(call.Error),
		nullString(call.RequestID),
		nullString(call.SessionID),
	)
	if err != nil {
		return fmt.Errorf("failed to insert API call: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		call.ID = id
	}

	return nil
}

// GetRecentAPICalls returns the most recent API calls.
func (db *DB) GetRecentAPICalls(limit int) ([]models.APICall, error) {
	query := `
		SELECT id, timestamp, email, model, provider, input_tokens, output_tokens,
			   cache_read_tokens, cache_write_tokens, duration_ms, status_code,
			   error, request_id, session_id
		FROM api_calls
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := db.QueryContext(context.Background(), query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent API calls: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var calls []models.APICall
	for rows.Next() {
		var call models.APICall
		var errStr, reqID, sessID sql.NullString

		err := rows.Scan(
			&call.ID,
			&call.Timestamp,
			&call.Email,
			&call.Model,
			&call.Provider,
			&call.InputTokens,
			&call.OutputTokens,
			&call.CacheReadTokens,
			&call.CacheWriteTokens,
			&call.DurationMs,
			&call.StatusCode,
			&errStr,
			&reqID,
			&sessID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API call: %w", err)
		}

		call.Error = errStr.String
		call.RequestID = reqID.String
		call.SessionID = sessID.String
		calls = append(calls, call)
	}

	return calls, rows.Err()
}

// GetHourlyStats returns aggregated statistics grouped by hour.
func (db *DB) GetHourlyStats(hours int) ([]models.HourlyStats, error) {
	query := `
		SELECT 
			strftime('%Y-%m-%d %H:00:00', timestamp) as hour,
			COUNT(*) as total_calls,
			COALESCE(SUM(input_tokens), 0) as total_input,
			COALESCE(SUM(output_tokens), 0) as total_output,
			COALESCE(SUM(cache_read_tokens), 0) as total_cache_read,
			COALESCE(SUM(cache_write_tokens), 0) as total_cache_write,
			COALESCE(AVG(duration_ms), 0) as avg_duration,
			SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) as error_count
		FROM api_calls
		WHERE timestamp >= datetime('now', ?)
		GROUP BY hour
		ORDER BY hour DESC
	`

	rows, err := db.QueryContext(context.Background(), query, fmt.Sprintf("-%d hours", hours))
	if err != nil {
		return nil, fmt.Errorf("failed to query hourly stats: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Error("failed to close rows", "error", err)
		}
	}()

	var stats []models.HourlyStats
	for rows.Next() {
		var s models.HourlyStats
		var hourStr string

		err := rows.Scan(
			&hourStr,
			&s.TotalCalls,
			&s.TotalInputTokens,
			&s.TotalOutputTokens,
			&s.TotalCacheRead,
			&s.TotalCacheWrite,
			&s.AvgDurationMs,
			&s.ErrorCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan hourly stats: %w", err)
		}

		s.Hour, _ = time.Parse("2006-01-02 15:04:05", hourStr)
		stats = append(stats, s)
	}

	return stats, rows.Err()
}

// GetTotalStats returns overall aggregated statistics.
func (db *DB) GetTotalStats() (*models.TotalStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_calls,
			COALESCE(SUM(input_tokens), 0) as total_input,
			COALESCE(SUM(output_tokens), 0) as total_output,
			COALESCE(SUM(cache_read_tokens), 0) as total_cache_read,
			COALESCE(SUM(cache_write_tokens), 0) as total_cache_write,
			COALESCE(AVG(duration_ms), 0) as avg_duration,
			SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) as error_count,
			COUNT(DISTINCT email) as unique_accounts,
			COUNT(DISTINCT model) as unique_models
		FROM api_calls
	`

	var stats models.TotalStats
	err := db.QueryRowContext(context.Background(), query).Scan(
		&stats.TotalCalls,
		&stats.TotalInputTokens,
		&stats.TotalOutputTokens,
		&stats.TotalCacheRead,
		&stats.TotalCacheWrite,
		&stats.AvgDurationMs,
		&stats.ErrorCount,
		&stats.UniqueAccounts,
		&stats.UniqueModels,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query total stats: %w", err)
	}

	return &stats, nil
}

// GetAccountStatus retrieves the status for a specific account.
func (db *DB) GetAccountStatus(email string) (*models.AccountStatus, error) {
	query := `
		SELECT email, claude_quota, gemini_quota, total_quota, tier,
			   is_rate_limited, last_error, last_updated, claude_reset_sec, gemini_reset_sec
		FROM account_status
		WHERE email = ?
	`

	var status models.AccountStatus
	err := db.QueryRowContext(context.Background(), query, email).Scan(
		&status.Email,
		&status.ClaudeQuota,
		&status.GeminiQuota,
		&status.TotalQuota,
		&status.Tier,
		&status.IsRateLimited,
		&status.LastError,
		&status.LastUpdated,
		&status.ClaudeResetSec,
		&status.GeminiResetSec,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account status: %w", err)
	}

	return &status, nil
}

// GetAllAccountStatuses retrieves all account statuses.
func (db *DB) GetAllAccountStatuses() ([]models.AccountStatus, error) {
	query := `
		SELECT email, claude_quota, gemini_quota, total_quota, tier,
			   is_rate_limited, last_error, last_updated, claude_reset_sec, gemini_reset_sec
		FROM account_status
		ORDER BY total_quota DESC
	`

	rows, err := db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query account statuses: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Error("failed to close rows", "error", err)
		}
	}()

	var statuses []models.AccountStatus
	for rows.Next() {
		var status models.AccountStatus
		err := rows.Scan(
			&status.Email,
			&status.ClaudeQuota,
			&status.GeminiQuota,
			&status.TotalQuota,
			&status.Tier,
			&status.IsRateLimited,
			&status.LastError,
			&status.LastUpdated,
			&status.ClaudeResetSec,
			&status.GeminiResetSec,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account status: %w", err)
		}
		statuses = append(statuses, status)
	}

	return statuses, rows.Err()
}

// InsertQuotaSnapshot records a point-in-time quota reading.
func (db *DB) InsertQuotaSnapshot(snapshot *models.QuotaSnapshot) error {
	query := `
		INSERT INTO quota_snapshots (
			email, claude_quota, gemini_quota, total_quota, tier, is_rate_limited, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	timestamp := snapshot.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	result, err := db.ExecContext(context.Background(), query,
		snapshot.Email,
		snapshot.ClaudeQuota,
		snapshot.GeminiQuota,
		snapshot.TotalQuota,
		snapshot.Tier,
		snapshot.IsRateLimited,
		timestamp.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return fmt.Errorf("failed to insert quota snapshot: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		snapshot.ID = id
	}

	return nil
}

// DeleteAccountStatus removes an account status entry.
func (db *DB) DeleteAccountStatus(email string) error {
	_, err := db.ExecContext(context.Background(), "DELETE FROM account_status WHERE email = ?", email)
	if err != nil {
		return fmt.Errorf("failed to delete account status: %w", err)
	}
	return nil
}

// nullString returns a sql.NullString from a string.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

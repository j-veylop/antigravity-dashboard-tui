package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

func (db *DB) UpsertAggregatedSnapshot(snapshot *models.AggregatedSnapshot) error {
	query := `
		INSERT INTO quota_snapshots_agg (
			email, bucket_time, claude_quota_avg, gemini_quota_avg,
			claude_consumed, gemini_consumed, sample_count, session_id, tier
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(email, bucket_time) DO UPDATE SET
			claude_quota_avg = (quota_snapshots_agg.claude_quota_avg * quota_snapshots_agg.sample_count + excluded.claude_quota_avg) / (quota_snapshots_agg.sample_count + 1),
			gemini_quota_avg = (quota_snapshots_agg.gemini_quota_avg * quota_snapshots_agg.sample_count + excluded.gemini_quota_avg) / (quota_snapshots_agg.sample_count + 1),
			claude_consumed = quota_snapshots_agg.claude_consumed + excluded.claude_consumed,
			gemini_consumed = quota_snapshots_agg.gemini_consumed + excluded.gemini_consumed,
			sample_count = quota_snapshots_agg.sample_count + 1,
			session_id = COALESCE(excluded.session_id, quota_snapshots_agg.session_id),
			tier = COALESCE(excluded.tier, quota_snapshots_agg.tier)
	`

	result, err := db.ExecContext(context.Background(), query,
		snapshot.Email,
		snapshot.BucketTime,
		snapshot.ClaudeQuotaAvg,
		snapshot.GeminiQuotaAvg,
		snapshot.ClaudeConsumed,
		snapshot.GeminiConsumed,
		snapshot.SampleCount,
		snapshot.SessionID,
		snapshot.Tier,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert aggregated snapshot: %w", err)
	}

	if snapshot.ID == 0 {
		id, err := result.LastInsertId()
		if err == nil {
			snapshot.ID = id
		}
	}

	return nil
}

func (db *DB) GetSessionSnapshots(email string, sessionWindow time.Duration) ([]models.AggregatedSnapshot, error) {
	query := `
		SELECT id, email, bucket_time, claude_quota_avg, gemini_quota_avg,
			   claude_consumed, gemini_consumed, sample_count, session_id, tier
		FROM quota_snapshots_agg
		WHERE email = ? AND bucket_time >= datetime('now', ?)
		ORDER BY bucket_time DESC
	`

	windowStr := fmt.Sprintf("-%d hours", int(sessionWindow.Hours()))
	rows, err := db.QueryContext(context.Background(), query, email, windowStr)
	if err != nil {
		return nil, fmt.Errorf("failed to query session snapshots: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var snapshots []models.AggregatedSnapshot
	for rows.Next() {
		var s models.AggregatedSnapshot
		var sessionID sql.NullString
		err := rows.Scan(
			&s.ID, &s.Email, &s.BucketTime, &s.ClaudeQuotaAvg, &s.GeminiQuotaAvg,
			&s.ClaudeConsumed, &s.GeminiConsumed, &s.SampleCount, &sessionID, &s.Tier,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}
		s.SessionID = sessionID.String
		snapshots = append(snapshots, s)
	}

	return snapshots, rows.Err()
}

func (db *DB) GetConsumptionRates(email string, sessionID string) (*models.ConsumptionRates, error) {
	rates := &models.ConsumptionRates{Email: email}

	sessionQuery := `
		SELECT
			COALESCE(AVG(claude_consumed) * 12, 0) as claude_rate,
			COALESCE(AVG(gemini_consumed) * 12, 0) as gemini_rate,
			COUNT(*) as data_points,
			MIN(bucket_time) as session_start
		FROM quota_snapshots_agg
		WHERE email = ? AND session_id = ?
	`

	var sessionStartStr sql.NullString
	err := db.QueryRowContext(context.Background(), sessionQuery, email, sessionID).Scan(
		&rates.SessionClaudeRate,
		&rates.SessionGeminiRate,
		&rates.SessionDataPoints,
		&sessionStartStr,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query session rates: %w", err)
	}
	if sessionStartStr.Valid && sessionStartStr.String != "" {
		if t, ok := parseTimeString(sessionStartStr.String); ok {
			rates.SessionStart = t
		}
	}

	historicalQuery := `
		SELECT
			COALESCE(AVG(claude_consumed) * 12, 0) as claude_rate,
			COALESCE(AVG(gemini_consumed) * 12, 0) as gemini_rate,
			COUNT(DISTINCT session_id) as sessions
		FROM quota_snapshots_agg
		WHERE email = ? AND session_id != ?
	`

	err = db.QueryRowContext(context.Background(), historicalQuery, email, sessionID).Scan(
		&rates.HistoricalClaudeRate,
		&rates.HistoricalGeminiRate,
		&rates.HistoricalSessions,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query historical rates: %w", err)
	}

	return rates, nil
}

func (db *DB) GetLastAggregatedSnapshot(email string) (*models.AggregatedSnapshot, error) {
	query := `
		SELECT id, email, bucket_time, claude_quota_avg, gemini_quota_avg,
			   claude_consumed, gemini_consumed, sample_count, session_id, tier
		FROM quota_snapshots_agg
		WHERE email = ?
		ORDER BY bucket_time DESC
		LIMIT 1
	`

	var s models.AggregatedSnapshot
	var sessionID sql.NullString
	err := db.QueryRowContext(context.Background(), query, email).Scan(
		&s.ID, &s.Email, &s.BucketTime, &s.ClaudeQuotaAvg, &s.GeminiQuotaAvg,
		&s.ClaudeConsumed, &s.GeminiConsumed, &s.SampleCount, &sessionID, &s.Tier,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last snapshot: %w", err)
	}
	s.SessionID = sessionID.String
	return &s, nil
}

func (db *DB) CleanupOldRawSnapshots(olderThanDays int) (int64, error) {
	query := `DELETE FROM quota_snapshots WHERE timestamp < datetime('now', ?)`
	windowStr := fmt.Sprintf("-%d days", olderThanDays)

	result, err := db.ExecContext(context.Background(), query, windowStr)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old snapshots: %w", err)
	}

	return result.RowsAffected()
}

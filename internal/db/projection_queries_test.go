package db

import (
	"context"
	"testing"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

func TestUpsertAggregatedSnapshot(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	snapshot := &models.AggregatedSnapshot{
		Email:          "test@example.com",
		BucketTime:     time.Now().UTC().Truncate(5 * time.Minute),
		ClaudeQuotaAvg: 75.5,
		GeminiQuotaAvg: 80.0,
		ClaudeConsumed: 2.5,
		GeminiConsumed: 1.5,
		SampleCount:    1,
		SessionID:      "ses_test123",
		Tier:           "PRO",
	}

	err := db.UpsertAggregatedSnapshot(snapshot)
	if err != nil {
		t.Fatalf("Failed to upsert snapshot: %v", err)
	}

	if snapshot.ID == 0 {
		t.Error("Expected snapshot ID to be set after insert")
	}

	retrieved, err := db.GetLastAggregatedSnapshot("test@example.com")
	if err != nil {
		t.Fatalf("Failed to get snapshot: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Expected to retrieve snapshot")
	}
	if retrieved.ClaudeQuotaAvg != 75.5 {
		t.Errorf("Expected ClaudeQuotaAvg 75.5, got %f", retrieved.ClaudeQuotaAvg)
	}
}

func TestUpsertAggregatedSnapshot_Updates(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	bucketTime := time.Now().UTC().Truncate(5 * time.Minute)

	snapshot1 := &models.AggregatedSnapshot{
		Email:          "test@example.com",
		BucketTime:     bucketTime,
		ClaudeQuotaAvg: 70.0,
		GeminiQuotaAvg: 80.0,
		SampleCount:    1,
		Tier:           "PRO",
	}
	if err := db.UpsertAggregatedSnapshot(snapshot1); err != nil {
		t.Fatalf("First upsert failed: %v", err)
	}

	snapshot2 := &models.AggregatedSnapshot{
		Email:          "test@example.com",
		BucketTime:     bucketTime,
		ClaudeQuotaAvg: 60.0,
		GeminiQuotaAvg: 70.0,
		SampleCount:    1,
		Tier:           "PRO",
	}
	if err := db.UpsertAggregatedSnapshot(snapshot2); err != nil {
		t.Fatalf("Second upsert failed: %v", err)
	}

	retrieved, _ := db.GetLastAggregatedSnapshot("test@example.com")
	if retrieved.SampleCount != 2 {
		t.Errorf("Expected SampleCount 2, got %d", retrieved.SampleCount)
	}
	expectedAvg := 65.0
	if retrieved.ClaudeQuotaAvg != expectedAvg {
		t.Errorf("Expected averaged ClaudeQuotaAvg %f, got %f", expectedAvg, retrieved.ClaudeQuotaAvg)
	}
}

func TestGetSessionSnapshots(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		snapshot := &models.AggregatedSnapshot{
			Email:          "test@example.com",
			BucketTime:     now.Add(time.Duration(-i) * time.Hour).Truncate(5 * time.Minute),
			ClaudeQuotaAvg: float64(100 - i*5),
			GeminiQuotaAvg: float64(100 - i*3),
			ClaudeConsumed: 2.0,
			GeminiConsumed: 1.0,
			SampleCount:    1,
			Tier:           "PRO",
		}
		if err := db.UpsertAggregatedSnapshot(snapshot); err != nil {
			t.Fatalf("Failed to insert snapshot %d: %v", i, err)
		}
	}

	snapshots, err := db.GetSessionSnapshots("test@example.com", 3*time.Hour)
	if err != nil {
		t.Fatalf("Failed to get session snapshots: %v", err)
	}

	if len(snapshots) < 3 {
		t.Errorf("Expected at least 3 snapshots, got %d", len(snapshots))
	}
}

func TestGetConsumptionRates_Empty(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	rates, err := db.GetConsumptionRates("nonexistent@example.com", "ses_test")
	if err != nil {
		t.Fatalf("Failed to get rates for empty DB: %v", err)
	}
	if rates == nil {
		t.Fatal("Expected non-nil rates")
	}
	if rates.SessionDataPoints != 0 {
		t.Errorf("Expected 0 data points, got %d", rates.SessionDataPoints)
	}
}

func TestGetConsumptionRates_WithData(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	now := time.Now().UTC()
	for i := 0; i < 12; i++ {
		snapshot := &models.AggregatedSnapshot{
			Email:          "test@example.com",
			BucketTime:     now.Add(time.Duration(-i*5) * time.Minute).Truncate(5 * time.Minute),
			ClaudeQuotaAvg: float64(100 - i),
			GeminiQuotaAvg: 100.0 - float64(i)*0.5,
			ClaudeConsumed: 1.0,
			GeminiConsumed: 0.5,
			SampleCount:    1,
			SessionID:      "ses_test",
			Tier:           "PRO",
		}
		if err := db.UpsertAggregatedSnapshot(snapshot); err != nil {
			t.Fatalf("Failed to insert snapshot: %v", err)
		}
	}

	rates, err := db.GetConsumptionRates("test@example.com", "ses_test")
	if err != nil {
		t.Fatalf("Failed to get rates: %v", err)
	}

	expectedClaudeRate := 12.0
	if rates.SessionClaudeRate < 10 || rates.SessionClaudeRate > 14 {
		t.Errorf("Expected Claude rate ~%f, got %f", expectedClaudeRate, rates.SessionClaudeRate)
	}

	if rates.SessionDataPoints != 12 {
		t.Errorf("Expected 12 data points, got %d", rates.SessionDataPoints)
	}
}

func TestGetLastAggregatedSnapshot_NotFound(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	snapshot, err := db.GetLastAggregatedSnapshot("nonexistent@example.com")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if snapshot != nil {
		t.Error("Expected nil snapshot for nonexistent email")
	}
}

func TestCleanupOldRawSnapshots(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO quota_snapshots (email, claude_quota, gemini_quota, timestamp)
		VALUES ('test@example.com', 50.0, 60.0, datetime('now', '-10 days'))
	`)
	if err != nil {
		t.Fatalf("Failed to insert old snapshot: %v", err)
	}

	_, err = db.ExecContext(context.Background(), `
		INSERT INTO quota_snapshots (email, claude_quota, gemini_quota, timestamp)
		VALUES ('test@example.com', 50.0, 60.0, datetime('now'))
	`)
	if err != nil {
		t.Fatalf("Failed to insert recent snapshot: %v", err)
	}

	deleted, err := db.CleanupOldRawSnapshots(7)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("Expected 1 deleted, got %d", deleted)
	}

	var count int
	db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM quota_snapshots").Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 remaining, got %d", count)
	}
}

func TestGetConsumptionRates_NullSessionStart(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	rates, err := db.GetConsumptionRates("empty@example.com", "ses_test")
	if err != nil {
		t.Fatalf("Failed to get rates with NULL session_start: %v", err)
	}

	if rates.SessionStart.IsZero() == false && !rates.SessionStart.Before(time.Now().Add(-100*365*24*time.Hour)) {
		t.Logf("SessionStart is zero (expected for empty data): %v", rates.SessionStart)
	}
}

func TestGetSessionSnapshots_Empty(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	snapshots, err := db.GetSessionSnapshots("nonexistent@example.com", 5*time.Hour)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(snapshots) != 0 {
		t.Errorf("Expected 0 snapshots, got %d", len(snapshots))
	}
}

func TestUpsertAggregatedSnapshot_NilSessionID(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	snapshot := &models.AggregatedSnapshot{
		Email:          "test@example.com",
		BucketTime:     time.Now().UTC().Truncate(5 * time.Minute),
		ClaudeQuotaAvg: 75.5,
		GeminiQuotaAvg: 80.0,
		SampleCount:    1,
		SessionID:      "",
		Tier:           "FREE",
	}

	err := db.UpsertAggregatedSnapshot(snapshot)
	if err != nil {
		t.Fatalf("Failed to upsert snapshot with empty session ID: %v", err)
	}

	retrieved, _ := db.GetLastAggregatedSnapshot("test@example.com")
	if retrieved.SessionID != "" {
		t.Errorf("Expected empty session ID, got %s", retrieved.SessionID)
	}
}

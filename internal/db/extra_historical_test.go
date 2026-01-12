package db

import (
	"testing"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

func TestGetSessionExhaustionStats(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	now := time.Now().UTC()

	snapshots := []models.AggregatedSnapshot{
		// Session 1: Exhausted (Consumed spike)
		{
			Email:          "test@example.com",
			SessionID:      "s1",
			BucketTime:     now.Add(-3 * time.Hour),
			ClaudeConsumed: 10,
			Tier:           "PRO",
		},
		{
			Email:          "test@example.com",
			SessionID:      "s1",
			BucketTime:     now.Add(-2 * time.Hour),
			ClaudeConsumed: 100, // Spike > 99
			Tier:           "PRO",
		},
		{
			Email:          "test@example.com",
			SessionID:      "s1",
			BucketTime:     now.Add(-1 * time.Hour),
			ClaudeConsumed: 0,
			Tier:           "PRO",
		},
		// Session 2: Not exhausted
		{
			Email:          "test@example.com",
			SessionID:      "s2",
			BucketTime:     now.Add(-30 * time.Minute),
			ClaudeConsumed: 10,
			Tier:           "PRO",
		},
		{
			Email:          "test@example.com",
			SessionID:      "s2",
			BucketTime:     now.Add(-25 * time.Minute),
			ClaudeConsumed: 10,
			Tier:           "PRO",
		},
	}

	for _, s := range snapshots {
		if err := db.UpsertAggregatedSnapshot(&s); err != nil {
			t.Fatalf("UpsertAggregatedSnapshot failed: %v", err)
		}
	}

	stats, err := db.GetSessionExhaustionStats("test@example.com", int(models.TimeRange24Hours))
	if err != nil {
		t.Fatalf("GetSessionExhaustionStats failed: %v", err)
	}

	if stats.TotalSessions != 2 {
		t.Errorf("TotalSessions = %d, want 2", stats.TotalSessions)
	}
	if stats.ExhaustedSessions != 1 {
		t.Errorf("ExhaustedSessions = %d, want 1", stats.ExhaustedSessions)
	}
}

func TestGetDailyUsageTrend(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)
	yesterday := today.Add(-24 * time.Hour)

	snapshots := []models.AggregatedSnapshot{
		{
			Email:          "test@example.com",
			BucketTime:     today,
			ClaudeConsumed: 10,
			GeminiConsumed: 5,
			Tier:           "PRO",
		},
		{
			Email:          "test@example.com",
			BucketTime:     yesterday,
			ClaudeConsumed: 20,
			GeminiConsumed: 10,
			Tier:           "PRO",
		},
	}

	for _, s := range snapshots {
		if err := db.UpsertAggregatedSnapshot(&s); err != nil {
			t.Fatalf("UpsertAggregatedSnapshot failed: %v", err)
		}
	}

	trend, err := db.GetDailyUsageTrend("test@example.com", int(models.TimeRange7Days))
	if err != nil {
		t.Fatalf("GetDailyUsageTrend failed: %v", err)
	}

	if len(trend) < 1 {
		t.Errorf("trend length = %d, want >= 1", len(trend))
	}

	total := 0.0
	for _, p := range trend {
		total += p.TotalConsumed
	}
	// 10+5 + 20+10 = 45
	if total < 44.0 || total > 46.0 {
		t.Errorf("total consumed = %f, want ~45", total)
	}
}

func TestGetHourlyPatterns(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	now := time.Now().UTC()
	t10 := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, time.UTC)

	s := models.AggregatedSnapshot{
		Email:          "test@example.com",
		BucketTime:     t10,
		ClaudeConsumed: 100,
		Tier:           "PRO",
	}
	if err := db.UpsertAggregatedSnapshot(&s); err != nil {
		t.Fatalf("UpsertAggregatedSnapshot failed: %v", err)
	}

	patterns, err := db.GetHourlyPatterns("test@example.com", int(models.TimeRange7Days))
	if err != nil {
		t.Fatalf("GetHourlyPatterns failed: %v", err)
	}

	found := false
	for _, p := range patterns {
		if p.Hour == 10 {
			if p.AvgConsumed < 1.0 {
				t.Errorf("Hour 10 AvgConsumed too low: %f", p.AvgConsumed)
			}
			found = true
		}
	}
	if !found {
		t.Error("Did not find pattern for hour 10")
	}
}

func TestGetWeekdayPatterns(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	s := models.AggregatedSnapshot{
		Email:          "test@example.com",
		BucketTime:     time.Now().UTC(),
		ClaudeConsumed: 100,
		Tier:           "PRO",
	}
	if err := db.UpsertAggregatedSnapshot(&s); err != nil {
		t.Fatalf("UpsertAggregatedSnapshot failed: %v", err)
	}

	patterns, err := db.GetWeekdayPatterns("test@example.com", int(models.TimeRange7Days))
	if err != nil {
		t.Fatalf("GetWeekdayPatterns failed: %v", err)
	}

	if len(patterns) == 0 {
		t.Error("GetWeekdayPatterns returned empty list")
	}
}

func TestGetAccountHistoryStats(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	s := models.AggregatedSnapshot{
		Email:          "test@example.com",
		BucketTime:     time.Now().UTC(),
		ClaudeConsumed: 100,
		Tier:           "PRO",
	}
	if err := db.UpsertAggregatedSnapshot(&s); err != nil {
		t.Fatalf("UpsertAggregatedSnapshot failed: %v", err)
	}

	stats, err := db.GetAccountHistoryStats("test@example.com", models.TimeRange7Days)
	if err != nil {
		t.Fatalf("GetAccountHistoryStats failed: %v", err)
	}

	if stats.Email != "test@example.com" {
		t.Errorf("Email = %q, want test@example.com", stats.Email)
	}
	if !stats.HasData() {
		t.Error("Stats should have data")
	}
}

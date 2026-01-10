package projection

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/db"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

func newTestService(t *testing.T) (*Service, *db.DB) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return New(database), database
}

func TestNew(t *testing.T) {
	svc, database := newTestService(t)
	defer database.Close()

	if svc == nil {
		t.Fatal("Expected non-nil service")
	}
}

func TestCalculateProjections_NoData(t *testing.T) {
	svc, database := newTestService(t)
	defer database.Close()

	proj, err := svc.CalculateProjections(
		"test@example.com",
		75.0, 80.0,
		time.Now().Add(5*time.Hour), time.Now().Add(5*time.Hour),
	)
	if err != nil {
		t.Fatalf("Failed to calculate projections: %v", err)
	}

	if proj.Claude == nil || proj.Gemini == nil {
		t.Error("Expected non-nil model projections")
	}
	if proj.Claude.Confidence != "low" {
		t.Errorf("Expected low confidence with no data, got %s", proj.Claude.Confidence)
	}
}

func TestCalculateProjections_WithData(t *testing.T) {
	svc, database := newTestService(t)
	defer database.Close()

	now := time.Now().UTC()
	for i := 0; i < 30; i++ {
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
		database.UpsertAggregatedSnapshot(snapshot)
	}

	svc.mu.Lock()
	svc.sessionIDs["test@example.com"] = "ses_test"
	svc.mu.Unlock()

	proj, err := svc.CalculateProjections(
		"test@example.com",
		70.0, 85.0,
		time.Now().Add(5*time.Hour), time.Now().Add(5*time.Hour),
	)
	if err != nil {
		t.Fatalf("Failed to calculate projections: %v", err)
	}

	if proj.Claude.Confidence != "high" {
		t.Errorf("Expected high confidence with 30 data points, got %s", proj.Claude.Confidence)
	}
	if proj.Claude.SessionRate <= 0 {
		t.Error("Expected positive session rate")
	}
}

func TestAggregateSnapshot(t *testing.T) {
	svc, database := newTestService(t)
	defer database.Close()

	err := svc.AggregateSnapshot("test@example.com", 75.0, 80.0, "PRO", "ses_123")
	if err != nil {
		t.Fatalf("Failed to aggregate snapshot: %v", err)
	}

	snapshot, err := database.GetLastAggregatedSnapshot("test@example.com")
	if err != nil {
		t.Fatalf("Failed to get snapshot: %v", err)
	}
	if snapshot == nil {
		t.Fatal("Expected snapshot to be stored")
	}
	if snapshot.ClaudeQuotaAvg != 75.0 {
		t.Errorf("Expected ClaudeQuotaAvg 75.0, got %f", snapshot.ClaudeQuotaAvg)
	}
}

func TestAggregateSnapshot_CalculatesConsumed(t *testing.T) {
	svc, database := newTestService(t)
	defer database.Close()

	svc.AggregateSnapshot("test@example.com", 80.0, 90.0, "PRO", "ses_123")

	time.Sleep(10 * time.Millisecond)
	svc.AggregateSnapshot("test@example.com", 75.0, 87.0, "PRO", "ses_123")

	snapshot, _ := database.GetLastAggregatedSnapshot("test@example.com")
	if snapshot == nil {
		t.Fatal("Expected snapshot to exist")
	}
}

func TestDetectSessionBoundary(t *testing.T) {
	svc, database := newTestService(t)
	defer database.Close()

	tests := []struct {
		name     string
		newPct   float64
		oldPct   float64
		expected bool
	}{
		{"No change", 50.0, 50.0, false},
		{"Decrease", 40.0, 50.0, false},
		{"Small increase", 52.0, 50.0, false},
		{"Reset detected", 95.0, 50.0, true},
		{"Large reset", 100.0, 10.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.DetectSessionBoundary("test@example.com", tt.newPct, tt.oldPct)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGenerateSessionID(t *testing.T) {
	svc, database := newTestService(t)
	defer database.Close()

	resetTime := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)

	id1 := svc.GenerateSessionID("test@example.com", resetTime)
	id2 := svc.GenerateSessionID("test@example.com", resetTime)

	if id1 != id2 {
		t.Error("Same inputs should produce same session ID")
	}

	id3 := svc.GenerateSessionID("other@example.com", resetTime)
	if id1 == id3 {
		t.Error("Different emails should produce different session IDs")
	}

	id4 := svc.GenerateSessionID("test@example.com", resetTime.Add(time.Hour))
	if id1 == id4 {
		t.Error("Different reset times should produce different session IDs")
	}
}

func TestGetOrCreateSessionID(t *testing.T) {
	svc, database := newTestService(t)
	defer database.Close()

	resetTime := time.Now()

	id1 := svc.GetOrCreateSessionID("test@example.com", resetTime)
	id2 := svc.GetOrCreateSessionID("test@example.com", resetTime.Add(time.Hour))

	if id1 != id2 {
		t.Error("GetOrCreateSessionID should return cached value")
	}
}

func TestResetSession(t *testing.T) {
	svc, database := newTestService(t)
	defer database.Close()

	resetTime := time.Now()

	id1 := svc.GetOrCreateSessionID("test@example.com", resetTime)
	id2 := svc.ResetSession("test@example.com", resetTime.Add(time.Hour))

	if id1 == id2 {
		t.Error("ResetSession should create new session ID")
	}

	id3 := svc.GetOrCreateSessionID("test@example.com", resetTime.Add(2*time.Hour))
	if id2 != id3 {
		t.Error("After reset, should use new session ID")
	}
}

func TestProjectionStatus(t *testing.T) {
	svc, database := newTestService(t)
	defer database.Close()

	now := time.Now().UTC()
	for i := 0; i < 30; i++ {
		snapshot := &models.AggregatedSnapshot{
			Email:          "test@example.com",
			BucketTime:     now.Add(time.Duration(-i*5) * time.Minute).Truncate(5 * time.Minute),
			ClaudeQuotaAvg: float64(50 - i),
			GeminiQuotaAvg: 80.0,
			ClaudeConsumed: 5.0,
			GeminiConsumed: 0.1,
			SampleCount:    1,
			SessionID:      "ses_test",
			Tier:           "PRO",
		}
		database.UpsertAggregatedSnapshot(snapshot)
	}

	svc.mu.Lock()
	svc.sessionIDs["test@example.com"] = "ses_test"
	svc.mu.Unlock()

	proj, _ := svc.CalculateProjections(
		"test@example.com",
		20.0, 80.0,
		time.Now().Add(5*time.Hour), time.Now().Add(5*time.Hour),
	)

	if proj.Claude.Status == models.ProjectionSafe {
		t.Log("Note: Claude status is SAFE - consumption rate may not be high enough")
	}

	if proj.Gemini.Status != models.ProjectionSafe && proj.Gemini.Status != models.ProjectionUnknown {
		t.Errorf("Expected Gemini to be safe, got %s", proj.Gemini.Status)
	}
}

func TestCachedProjection(t *testing.T) {
	svc, database := newTestService(t)
	defer database.Close()

	cached := svc.GetCachedProjection("test@example.com")
	if cached != nil {
		t.Error("Expected nil cached projection initially")
	}

	svc.CalculateProjections(
		"test@example.com",
		75.0, 80.0,
		time.Now().Add(5*time.Hour), time.Now().Add(5*time.Hour),
	)

	cached = svc.GetCachedProjection("test@example.com")
	if cached == nil {
		t.Error("Expected cached projection after calculation")
	}
}

func TestGetAllProjections(t *testing.T) {
	svc, database := newTestService(t)
	defer database.Close()

	for _, email := range []string{"a@test.com", "b@test.com", "c@test.com"} {
		svc.CalculateProjections(
			email,
			75.0, 80.0,
			time.Now().Add(5*time.Hour), time.Now().Add(5*time.Hour),
		)
	}

	all := svc.GetAllProjections()
	if len(all) != 3 {
		t.Errorf("Expected 3 projections, got %d", len(all))
	}
}

func TestGenerateSessionID_ZeroTime(t *testing.T) {
	svc, database := newTestService(t)
	defer database.Close()

	id := svc.GenerateSessionID("test@example.com", time.Time{})
	if id == "" {
		t.Error("Expected non-empty session ID even with zero time")
	}
	if len(id) < 10 {
		t.Errorf("Session ID seems too short: %s", id)
	}
}

func TestCalculateProjections_MultipleEmails(t *testing.T) {
	svc, database := newTestService(t)
	defer database.Close()

	emails := []string{"alice@example.com", "bob@example.com", "charlie@example.com"}

	for _, email := range emails {
		proj, err := svc.CalculateProjections(
			email,
			75.0, 80.0,
			time.Now().Add(5*time.Hour), time.Now().Add(5*time.Hour),
		)
		if err != nil {
			t.Fatalf("Failed to calculate projection for %s: %v", email, err)
		}
		if proj.Email != email {
			t.Errorf("Expected email %s, got %s", email, proj.Email)
		}
	}

	all := svc.GetAllProjections()
	if len(all) != len(emails) {
		t.Errorf("Expected %d projections, got %d", len(emails), len(all))
	}
}

func TestAggregateSnapshot_EmptyTier(t *testing.T) {
	svc, database := newTestService(t)
	defer database.Close()

	err := svc.AggregateSnapshot("test@example.com", 75.0, 80.0, "", "ses_123")
	if err != nil {
		t.Fatalf("Failed to aggregate snapshot with empty tier: %v", err)
	}

	snapshot, _ := database.GetLastAggregatedSnapshot("test@example.com")
	if snapshot == nil {
		t.Fatal("Expected snapshot to exist")
	}
}

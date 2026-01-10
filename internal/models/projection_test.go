package models

import (
	"testing"
	"time"
)

func TestProjectionStatus_Constants(t *testing.T) {
	statuses := []ProjectionStatus{
		ProjectionSafe,
		ProjectionWarning,
		ProjectionCritical,
		ProjectionUnknown,
	}

	seen := make(map[ProjectionStatus]bool)
	for _, s := range statuses {
		if seen[s] {
			t.Errorf("Duplicate status constant: %s", s)
		}
		seen[s] = true
	}
}

func TestModelProjection_Defaults(t *testing.T) {
	proj := &ModelProjection{
		Model: "claude",
	}

	if proj.CurrentPercent != 0 {
		t.Error("Expected zero default for CurrentPercent")
	}
	if proj.WillDepleteBefore != false {
		t.Error("Expected false default for WillDepleteBefore")
	}
}

func TestAccountProjection(t *testing.T) {
	proj := &AccountProjection{
		Email: "test@example.com",
		Claude: &ModelProjection{
			Model:          "claude",
			CurrentPercent: 75.0,
			Status:         ProjectionSafe,
		},
		Gemini: &ModelProjection{
			Model:          "gemini",
			CurrentPercent: 80.0,
			Status:         ProjectionWarning,
		},
		LastUpdated: time.Now(),
	}

	if proj.Claude.Model != "claude" {
		t.Error("Expected claude model")
	}
	if proj.Gemini.Status != ProjectionWarning {
		t.Error("Expected warning status for gemini")
	}
}

func TestAggregatedSnapshot(t *testing.T) {
	snapshot := &AggregatedSnapshot{
		Email:          "test@example.com",
		BucketTime:     time.Now().Truncate(5 * time.Minute),
		ClaudeQuotaAvg: 75.5,
		GeminiQuotaAvg: 80.0,
		ClaudeConsumed: 2.5,
		GeminiConsumed: 1.5,
		SampleCount:    3,
		SessionID:      "ses_abc123",
		Tier:           "PRO",
	}

	if snapshot.ClaudeQuotaAvg != 75.5 {
		t.Errorf("Expected 75.5, got %f", snapshot.ClaudeQuotaAvg)
	}
}

func TestConsumptionRates(t *testing.T) {
	rates := &ConsumptionRates{
		Email:                "test@example.com",
		SessionClaudeRate:    12.5,
		SessionGeminiRate:    8.0,
		SessionDataPoints:    30,
		SessionStart:         time.Now().Add(-2 * time.Hour),
		HistoricalClaudeRate: 10.0,
		HistoricalGeminiRate: 7.5,
		HistoricalSessions:   15,
	}

	if rates.SessionClaudeRate <= 0 {
		t.Error("Expected positive session rate")
	}
	if rates.HistoricalSessions != 15 {
		t.Errorf("Expected 15 historical sessions, got %d", rates.HistoricalSessions)
	}
}

func TestHistoricalContext(t *testing.T) {
	ctx := &HistoricalContext{
		CurrentMonthRate:   12.0,
		LastMonthRate:      10.0,
		MonthOverMonthDiff: 20.0,
		AllTimeAvgRate:     11.0,
		AllTimePeakRate:    25.0,
		TotalSessionsEver:  50,
		PeakUsageDay:       "Wednesday",
		PeakUsageHour:      14,
		FirstDataPoint:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		TotalDataDays:      180,
	}

	if ctx.MonthOverMonthDiff != 20.0 {
		t.Errorf("Expected 20%% MoM diff, got %f", ctx.MonthOverMonthDiff)
	}
	if ctx.PeakUsageDay != "Wednesday" {
		t.Errorf("Expected Wednesday, got %s", ctx.PeakUsageDay)
	}
}

func TestPeriodStats(t *testing.T) {
	stats := &PeriodStats{
		Period:          "2024-06",
		TotalConsumed:   450.5,
		AvgRatePerHour:  8.5,
		PeakRatePerHour: 22.0,
		SessionCount:    30,
		DataPoints:      8640,
		StartTime:       time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		EndTime:         time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC),
	}

	if stats.Period != "2024-06" {
		t.Errorf("Expected 2024-06, got %s", stats.Period)
	}
}

func TestUsagePattern(t *testing.T) {
	pattern := &UsagePattern{
		DayOfWeek:   3,
		Hour:        14,
		AvgConsumed: 3.5,
		Occurrences: 52,
	}

	if pattern.DayOfWeek != 3 {
		t.Errorf("Expected day 3, got %d", pattern.DayOfWeek)
	}
}

func TestProjectionStatus_StringValues(t *testing.T) {
	if ProjectionSafe != "SAFE" {
		t.Errorf("Expected SAFE, got %s", ProjectionSafe)
	}
	if ProjectionWarning != "WARNING" {
		t.Errorf("Expected WARNING, got %s", ProjectionWarning)
	}
	if ProjectionCritical != "CRITICAL" {
		t.Errorf("Expected CRITICAL, got %s", ProjectionCritical)
	}
	if ProjectionUnknown != "UNKNOWN" {
		t.Errorf("Expected UNKNOWN, got %s", ProjectionUnknown)
	}
}

func TestAggregatedSnapshot_ZeroValues(t *testing.T) {
	snapshot := &AggregatedSnapshot{}

	if snapshot.ID != 0 {
		t.Error("Expected zero ID")
	}
	if snapshot.Email != "" {
		t.Error("Expected empty email")
	}
	if !snapshot.BucketTime.IsZero() {
		t.Error("Expected zero bucket time")
	}
}

func TestModelProjection_AllFields(t *testing.T) {
	historical := &HistoricalContext{
		AllTimeAvgRate: 10.0,
	}

	proj := &ModelProjection{
		Model:             "claude",
		CurrentPercent:    75.0,
		SessionRate:       12.0,
		SessionHoursLeft:  6.25,
		SessionDepleteAt:  time.Now().Add(6 * time.Hour),
		HistoricalRate:    10.0,
		TypicalDuration:   8.0,
		Historical:        historical,
		VsLastMonth:       "20% higher",
		VsHistorical:      "Above average",
		ResetTime:         time.Now().Add(5 * time.Hour),
		TimeUntilReset:    5 * time.Hour,
		WillDepleteBefore: true,
		Status:            ProjectionWarning,
		Confidence:        "high",
		DataPoints:        50,
	}

	if proj.WillDepleteBefore != true {
		t.Error("Expected WillDepleteBefore to be true")
	}
	if proj.Status != ProjectionWarning {
		t.Errorf("Expected WARNING status, got %s", proj.Status)
	}
	if proj.Historical.AllTimeAvgRate != 10.0 {
		t.Error("Expected historical context to be set")
	}
}

func TestConsumptionRates_ZeroSessions(t *testing.T) {
	rates := &ConsumptionRates{
		Email:             "test@example.com",
		SessionDataPoints: 0,
	}

	if rates.SessionClaudeRate != 0 {
		t.Error("Expected zero Claude rate")
	}
	if rates.SessionGeminiRate != 0 {
		t.Error("Expected zero Gemini rate")
	}
}

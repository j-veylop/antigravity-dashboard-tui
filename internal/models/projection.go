// Package models defines data structures and domain types.
package models

import "time"

// AggregatedSnapshot represents a 5-minute quota bucket for long-term analysis.
type AggregatedSnapshot struct {
	BucketTime     time.Time
	SessionID      string
	Email          string
	Tier           string
	ClaudeQuotaAvg float64
	ClaudeConsumed float64
	GeminiConsumed float64
	SampleCount    int
	GeminiQuotaAvg float64
	ID             int64
	Year           int
	Month          int
	Week           int
	DayOfWeek      int
	Hour           int
}

// ProjectionStatus indicates urgency level for quota depletion.
type ProjectionStatus string

const (
	// ProjectionSafe indicates quota is sufficient for the session.
	ProjectionSafe ProjectionStatus = "SAFE"
	// ProjectionWarning indicates quota might run out before reset.
	ProjectionWarning ProjectionStatus = "WARNING"
	// ProjectionCritical indicates quota will likely run out very soon.
	ProjectionCritical ProjectionStatus = "CRITICAL"
	// ProjectionUnknown indicates insufficient data for projection.
	ProjectionUnknown ProjectionStatus = "UNKNOWN"
)

// HistoricalContext provides long-term usage context for projections.
type HistoricalContext struct {
	FirstDataPoint     time.Time
	PeakUsageDay       string
	CurrentMonthRate   float64
	LastMonthRate      float64
	MonthOverMonthDiff float64
	AllTimeAvgRate     float64
	AllTimePeakRate    float64
	TotalSessionsEver  int
	PeakUsageHour      int
	TotalDataDays      int
}

// ModelProjection contains projection for a single model (Claude or Gemini).
type ModelProjection struct {
	SessionDepleteAt  time.Time
	ResetTime         time.Time
	Historical        *HistoricalContext
	VsLastMonth       string
	Model             string
	VsHistorical      string
	Status            ProjectionStatus
	Confidence        string
	SessionHoursLeft  float64
	HistoricalRate    float64
	TypicalDuration   float64
	SessionRate       float64
	CurrentPercent    float64
	TimeUntilReset    time.Duration
	DataPoints        int
	WillDepleteBefore bool
}

// AccountProjection contains projections for all models of an account.
type AccountProjection struct {
	LastUpdated time.Time
	Claude      *ModelProjection
	Gemini      *ModelProjection
	Email       string
}

// ConsumptionRates holds calculated consumption velocities for an account.
type ConsumptionRates struct {
	SessionStart         time.Time
	Email                string
	SessionClaudeRate    float64
	SessionGeminiRate    float64
	SessionDataPoints    int
	HistoricalClaudeRate float64
	HistoricalGeminiRate float64
	HistoricalSessions   int
}

// PeriodStats represents usage statistics for a time period.
type PeriodStats struct {
	StartTime       time.Time
	EndTime         time.Time
	Period          string
	TotalConsumed   float64
	AvgRatePerHour  float64
	PeakRatePerHour float64
	SessionCount    int
	DataPoints      int
}

// UsagePattern represents usage patterns by day/hour for pattern analysis.
type UsagePattern struct {
	DayOfWeek   int     // 0-6 (Sunday-Saturday)
	Hour        int     // 0-23
	AvgConsumed float64 // Average consumption in this slot
	Occurrences int     // How many times this slot was observed
}

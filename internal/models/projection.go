// Package models defines data structures and domain types.
package models

import "time"

// AggregatedSnapshot represents a 5-minute quota bucket for long-term analysis.
type AggregatedSnapshot struct {
	ID             int64
	Email          string
	BucketTime     time.Time
	ClaudeQuotaAvg float64
	GeminiQuotaAvg float64
	ClaudeConsumed float64
	GeminiConsumed float64
	SampleCount    int
	SessionID      string
	Tier           string
	// Generated time dimensions
	Year      int
	Month     int
	Week      int
	DayOfWeek int
	Hour      int
}

// ProjectionStatus indicates urgency level for quota depletion.
type ProjectionStatus string

const (
	ProjectionSafe     ProjectionStatus = "SAFE"
	ProjectionWarning  ProjectionStatus = "WARNING"
	ProjectionCritical ProjectionStatus = "CRITICAL"
	ProjectionUnknown  ProjectionStatus = "UNKNOWN"
)

// HistoricalContext provides long-term usage context for projections.
type HistoricalContext struct {
	CurrentMonthRate   float64   // Avg consumption rate this month (%/hr)
	LastMonthRate      float64   // Avg consumption rate last month (%/hr)
	MonthOverMonthDiff float64   // Percentage change from last month
	AllTimeAvgRate     float64   // Average rate across all history
	AllTimePeakRate    float64   // Peak hourly rate ever observed
	TotalSessionsEver  int       // Total number of unique sessions tracked
	PeakUsageDay       string    // Day of week with highest usage
	PeakUsageHour      int       // Hour of day with highest usage
	FirstDataPoint     time.Time // When we started tracking
	TotalDataDays      int       // Total days of data collected
}

// ModelProjection contains projection for a single model (Claude or Gemini).
type ModelProjection struct {
	Model             string             // "claude" or "gemini"
	CurrentPercent    float64            // Current quota remaining %
	SessionRate       float64            // Consumption rate in current session (%/hr)
	SessionHoursLeft  float64            // Hours until depletion at current rate
	SessionDepleteAt  time.Time          // Predicted depletion time
	HistoricalRate    float64            // Average historical consumption rate (%/hr)
	TypicalDuration   float64            // Typical session duration based on history
	Historical        *HistoricalContext // Full historical context
	VsLastMonth       string             // Comparison text vs last month
	VsHistorical      string             // Comparison text vs all-time average
	ResetTime         time.Time          // When quota resets
	TimeUntilReset    time.Duration      // Duration until reset
	WillDepleteBefore bool               // True if will deplete before reset
	Status            ProjectionStatus   // SAFE, WARNING, CRITICAL, UNKNOWN
	Confidence        string             // "low", "medium", "high"
	DataPoints        int                // Number of data points used
}

// AccountProjection contains projections for all models of an account.
type AccountProjection struct {
	Email       string
	Claude      *ModelProjection
	Gemini      *ModelProjection
	LastUpdated time.Time
}

// ConsumptionRates holds calculated consumption velocities for an account.
type ConsumptionRates struct {
	Email                string
	SessionClaudeRate    float64   // Current session Claude consumption (%/hr)
	SessionGeminiRate    float64   // Current session Gemini consumption (%/hr)
	SessionDataPoints    int       // Data points in current session
	SessionStart         time.Time // When current session started
	HistoricalClaudeRate float64   // Historical avg Claude rate (%/hr)
	HistoricalGeminiRate float64   // Historical avg Gemini rate (%/hr)
	HistoricalSessions   int       // Number of historical sessions
}

// PeriodStats represents usage statistics for a time period.
type PeriodStats struct {
	Period          string    // e.g., "2024-01", "Week 4"
	TotalConsumed   float64   // Total % consumed in period
	AvgRatePerHour  float64   // Average rate per hour
	PeakRatePerHour float64   // Peak hourly rate
	SessionCount    int       // Number of sessions in period
	DataPoints      int       // Total aggregated buckets
	StartTime       time.Time // Period start
	EndTime         time.Time // Period end
}

// UsagePattern represents usage patterns by day/hour for pattern analysis.
type UsagePattern struct {
	DayOfWeek   int     // 0-6 (Sunday-Saturday)
	Hour        int     // 0-23
	AvgConsumed float64 // Average consumption in this slot
	Occurrences int     // How many times this slot was observed
}

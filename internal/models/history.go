// Package models defines data structures and domain types.
package models

import "time"

// TimeRange represents the selected history time range.
type TimeRange int

const (
	// TimeRange24Hours shows data from the last 24 hours.
	TimeRange24Hours TimeRange = iota
	// TimeRange7Days shows data from the last 7 days.
	TimeRange7Days
	// TimeRange30Days shows data from the last 30 days.
	TimeRange30Days
	// TimeRangeAllTime shows all available historical data.
	TimeRangeAllTime
)

// String returns the display name for a time range.
func (t TimeRange) String() string {
	switch t {
	case TimeRange24Hours:
		return "24 Hours"
	case TimeRange7Days:
		return "7 Days"
	case TimeRange30Days:
		return "30 Days"
	case TimeRangeAllTime:
		return "All Time"
	default:
		return "Unknown"
	}
}

// Days returns the number of days for the time range (0 = unlimited).
func (t TimeRange) Days() int {
	switch t {
	case TimeRange24Hours:
		return 1
	case TimeRange7Days:
		return 7
	case TimeRange30Days:
		return 30
	case TimeRangeAllTime:
		return 0
	default:
		return 30
	}
}

// Next cycles to the next time range.
func (t TimeRange) Next() TimeRange {
	return (t + 1) % 4
}

// RateLimitHit represents a single rate limit transition event.
type RateLimitHit struct {
	Timestamp time.Time
	Email     string
	Model     string // "claude" or "gemini"
	SessionID string
}

// RateLimitStats aggregates rate limit transition counts.
type RateLimitStats struct {
	LastHitTime    time.Time
	HitsByDay      []DailyHitCount
	TotalHits      int
	HitsInRange    int
	HitsLast7Days  int
	HitsLast30Days int
	AvgTimeBetween time.Duration
	ClaudeHits     int
	GeminiHits     int
}

// DailyHitCount tracks rate limit hits per day.
type DailyHitCount struct {
	Date  time.Time
	Count int
}

// SessionExhaustionEvent represents one session's exhaustion data.
type SessionExhaustionEvent struct {
	StartTime    time.Time
	EndTime      time.Time
	SessionID    string
	Email        string
	Model        string
	StartPercent float64
	EndPercent   float64
	Duration     time.Duration
	WasExhausted bool
}

// ExhaustionStats aggregates time-to-exhaustion data.
type ExhaustionStats struct {
	AvgTimeToExhaust    time.Duration // Average session duration to quota depletion
	MedianTimeToExhaust time.Duration
	MinTimeToExhaust    time.Duration
	MaxTimeToExhaust    time.Duration
	TotalSessions       int
	ExhaustedSessions   int     // Sessions that reached near 0%
	ExhaustionRate      float64 // % of sessions that exhaust quota
	AvgStartPercent     float64 // Typical starting quota %
	AvgConsumptionRate  float64 // %/hour average consumption
}

// DailyUsagePoint contains data for a single day in trend charts.
type DailyUsagePoint struct {
	Date           time.Time
	ClaudeConsumed float64 // Total Claude consumption that day
	GeminiConsumed float64 // Total Gemini consumption that day
	TotalConsumed  float64 // Combined consumption
	RateLimitHits  int     // Number of rate limit events that day
	SessionCount   int     // Number of unique sessions that day
	DataPoints     int     // Number of snapshots recorded
}

// HourlyPattern represents usage patterns by hour of day.
type HourlyPattern struct {
	Hour        int     // 0-23
	AvgConsumed float64 // Average consumption in this hour slot
	TotalHits   int     // Total rate limit hits in this hour
	Occurrences int     // How many times this hour was observed
}

// WeekdayPattern represents usage patterns by day of week.
type WeekdayPattern struct {
	DayName     string
	DayOfWeek   int
	AvgConsumed float64
	TotalHits   int
	Occurrences int
}

// AccountHistoryStats contains all history data for a single account.
type AccountHistoryStats struct {
	FirstDataPoint  time.Time
	LastUpdated     time.Time
	LastDataPoint   time.Time
	Exhaustion      *ExhaustionStats
	Historical      *HistoricalContext
	RateLimits      *RateLimitStats
	Email           string
	DailyUsage      []DailyUsagePoint
	HourlyPatterns  []HourlyPattern
	WeekdayPatterns []WeekdayPattern
	TotalDataDays   int
	TotalDataPoints int
	TimeRange       TimeRange
}

// HasData returns true if the account has any historical data.
func (a *AccountHistoryStats) HasData() bool {
	return a.TotalDataPoints > 0
}

// GetPeakHour returns the hour with highest average consumption.
func (a *AccountHistoryStats) GetPeakHour() (peakHour int, peakVal float64) {
	if len(a.HourlyPatterns) == 0 {
		return 0, 0
	}
	peakHour = 0
	peakVal = 0.0
	for _, p := range a.HourlyPatterns {
		if p.AvgConsumed > peakVal {
			peakVal = p.AvgConsumed
			peakHour = p.Hour
		}
	}
	return peakHour, peakVal
}

// GetPeakDay returns the weekday with highest average consumption.
func (a *AccountHistoryStats) GetPeakDay() (peakDay string, peakVal float64) {
	if len(a.WeekdayPatterns) == 0 {
		return "Unknown", 0
	}
	peakDay = ""
	peakVal = 0.0
	for _, p := range a.WeekdayPatterns {
		if p.AvgConsumed > peakVal {
			peakVal = p.AvgConsumed
			peakDay = p.DayName
		}
	}
	return peakDay, peakVal
}

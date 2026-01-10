// Package models defines data structures and domain types.
package models

import "time"

// TimeRange represents the selected history time range.
type TimeRange int

const (
	// TimeRange7Days shows data from the last 7 days.
	TimeRange7Days TimeRange = iota
	// TimeRange30Days shows data from the last 30 days.
	TimeRange30Days
	// TimeRangeAllTime shows all available historical data.
	TimeRangeAllTime
)

// String returns the display name for a time range.
func (t TimeRange) String() string {
	switch t {
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
	return (t + 1) % 3
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
	TotalHits      int           // All-time transition count
	HitsInRange    int           // Hits within selected time range
	HitsLast7Days  int           // Last 7 days
	HitsLast30Days int           // Last 30 days
	LastHitTime    time.Time     // Most recent rate limit
	AvgTimeBetween time.Duration // Average time between rate limit events
	HitsByDay      []DailyHitCount
	ClaudeHits     int // Claude-specific hits
	GeminiHits     int // Gemini-specific hits
}

// DailyHitCount tracks rate limit hits per day.
type DailyHitCount struct {
	Date  time.Time
	Count int
}

// SessionExhaustionEvent represents one session's exhaustion data.
type SessionExhaustionEvent struct {
	SessionID    string
	Email        string
	StartTime    time.Time
	EndTime      time.Time
	StartPercent float64       // Quota % at session start (100 - consumed)
	EndPercent   float64       // Quota % at session end
	Duration     time.Duration // Session duration
	WasExhausted bool          // True if reached near 0%
	Model        string        // "claude", "gemini", or "combined"
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
	DayOfWeek   int     // 0=Sunday, 6=Saturday
	DayName     string  // "Sunday", "Monday", etc.
	AvgConsumed float64 // Average daily consumption
	TotalHits   int     // Total rate limit hits on this day
	Occurrences int     // How many times this day was observed
}

// AccountHistoryStats contains all history data for a single account.
type AccountHistoryStats struct {
	Email           string
	TimeRange       TimeRange
	RateLimits      *RateLimitStats
	Exhaustion      *ExhaustionStats
	DailyUsage      []DailyUsagePoint
	HourlyPatterns  []HourlyPattern  // 24 entries, one per hour
	WeekdayPatterns []WeekdayPattern // 7 entries, one per day
	Historical      *HistoricalContext
	FirstDataPoint  time.Time
	LastDataPoint   time.Time
	TotalDataDays   int
	TotalDataPoints int
	LastUpdated     time.Time
}

// HasData returns true if the account has any historical data.
func (a *AccountHistoryStats) HasData() bool {
	return a.TotalDataPoints > 0
}

// GetPeakHour returns the hour with highest average consumption.
func (a *AccountHistoryStats) GetPeakHour() (int, float64) {
	if len(a.HourlyPatterns) == 0 {
		return 0, 0
	}
	peakHour := 0
	peakVal := 0.0
	for _, p := range a.HourlyPatterns {
		if p.AvgConsumed > peakVal {
			peakVal = p.AvgConsumed
			peakHour = p.Hour
		}
	}
	return peakHour, peakVal
}

// GetPeakDay returns the weekday with highest average consumption.
func (a *AccountHistoryStats) GetPeakDay() (string, float64) {
	if len(a.WeekdayPatterns) == 0 {
		return "Unknown", 0
	}
	peakDay := ""
	peakVal := 0.0
	for _, p := range a.WeekdayPatterns {
		if p.AvgConsumed > peakVal {
			peakVal = p.AvgConsumed
			peakDay = p.DayName
		}
	}
	return peakDay, peakVal
}

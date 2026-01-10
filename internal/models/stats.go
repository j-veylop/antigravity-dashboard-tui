// Package models defines data structures and domain types.
package models

import "time"

// HourlyStats represents usage statistics grouped by hour.
type HourlyStats struct {
	Hour              time.Time
	TotalCalls        int
	TotalInputTokens  int64
	TotalOutputTokens int64
	TotalCacheRead    int64
	TotalCacheWrite   int64
	AvgDurationMs     float64
	ErrorCount        int
}

// TotalStats represents overall aggregated statistics.
type TotalStats struct {
	TotalCalls        int
	TotalInputTokens  int64
	TotalOutputTokens int64
	TotalCacheRead    int64
	TotalCacheWrite   int64
	AvgDurationMs     float64
	ErrorCount        int
	UniqueAccounts    int
	UniqueModels      int
}

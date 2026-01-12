// Package models defines data structures and domain types.
package models

import "time"

// APICall represents a logged API call to the database.
type APICall struct {
	Timestamp        time.Time
	Error            string
	Email            string
	Model            string
	Provider         string
	SessionID        string
	RequestID        string
	OutputTokens     int
	CacheWriteTokens int
	DurationMs       int
	StatusCode       int
	CacheReadTokens  int
	ID               int64
	InputTokens      int
}

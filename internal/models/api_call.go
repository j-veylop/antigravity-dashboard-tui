// Package models defines data structures and domain types.
package models

import "time"

// APICall represents a logged API call to the database.
type APICall struct {
	ID               int64
	Timestamp        time.Time
	Email            string
	Model            string
	Provider         string
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
	DurationMs       int
	StatusCode       int
	Error            string
	RequestID        string
	SessionID        string
}

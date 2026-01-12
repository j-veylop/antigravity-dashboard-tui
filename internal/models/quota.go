// Package models defines data structures and domain types.
package models

import "time"

// ModelQuota represents quota information for a specific model.
type ModelQuota struct {
	ResetTime        time.Time `json:"resetTime"`
	LastUpdated      time.Time `json:"lastUpdated"`
	ModelFamily      string    `json:"modelFamily"`
	Tier             string    `json:"tier,omitempty"`
	SubscriptionTier string    `json:"subscriptionTier,omitempty"`
	Used             int64     `json:"used"`
	Limit            int64     `json:"limit"`
	Remaining        int64     `json:"remaining,omitempty"`
	UsagePercentage  float64   `json:"usagePercentage,omitempty"`
	IsRateLimited    bool      `json:"isRateLimited,omitempty"`
}

// QuotaInfo represents aggregated quota information for an account.
type QuotaInfo struct {
	LastUpdated      time.Time    `json:"lastUpdated"`
	AccountEmail     string       `json:"accountEmail"`
	Error            string       `json:"error,omitempty"`
	SubscriptionTier string       `json:"subscriptionTier,omitempty"`
	ModelQuotas      []ModelQuota `json:"modelQuotas"`
	TotalRemaining   int64        `json:"totalRemaining,omitempty"`
	TotalLimit       int64        `json:"totalLimit,omitempty"`
	OverallPercent   float64      `json:"overallPercent,omitempty"`
}

// AccountStatus represents the status for a specific account (DB model).
type AccountStatus struct {
	LastUpdated    time.Time
	Email          string
	Tier           string
	LastError      string
	ClaudeQuota    float64
	GeminiQuota    float64
	TotalQuota     float64
	ClaudeResetSec int64
	GeminiResetSec int64
	IsRateLimited  bool
}

// QuotaSnapshot represents a point-in-time quota reading (DB model).
type QuotaSnapshot struct {
	Timestamp     time.Time
	Email         string
	Tier          string
	ID            int64
	ClaudeQuota   float64
	GeminiQuota   float64
	TotalQuota    float64
	IsRateLimited bool
}

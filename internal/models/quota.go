// Package models defines data structures and domain types.
package models

import "time"

// ModelQuota represents quota information for a specific model.
type ModelQuota struct {
	ModelFamily      string    `json:"modelFamily"`
	Tier             string    `json:"tier,omitempty"`
	Used             int64     `json:"used"`
	Limit            int64     `json:"limit"`
	ResetTime        time.Time `json:"resetTime"`
	Remaining        int64     `json:"remaining,omitempty"`
	UsagePercentage  float64   `json:"usagePercentage,omitempty"`
	IsRateLimited    bool      `json:"isRateLimited,omitempty"`
	LastUpdated      time.Time `json:"lastUpdated,omitempty"`
	SubscriptionTier string    `json:"subscriptionTier,omitempty"`
}

// QuotaInfo represents aggregated quota information for an account.
type QuotaInfo struct {
	AccountEmail     string       `json:"accountEmail"`
	ModelQuotas      []ModelQuota `json:"modelQuotas"`
	LastUpdated      time.Time    `json:"lastUpdated"`
	Error            string       `json:"error,omitempty"`
	SubscriptionTier string       `json:"subscriptionTier,omitempty"`
	TotalRemaining   int64        `json:"totalRemaining,omitempty"`
	TotalLimit       int64        `json:"totalLimit,omitempty"`
	OverallPercent   float64      `json:"overallPercent,omitempty"`
}

// AccountStatus represents the status for a specific account (DB model).
type AccountStatus struct {
	Email          string
	ClaudeQuota    float64
	GeminiQuota    float64
	TotalQuota     float64
	Tier           string
	IsRateLimited  bool
	LastError      string
	LastUpdated    time.Time
	ClaudeResetSec int64
	GeminiResetSec int64
}

// QuotaSnapshot represents a point-in-time quota reading (DB model).
type QuotaSnapshot struct {
	ID            int64
	Email         string
	ClaudeQuota   float64
	GeminiQuota   float64
	TotalQuota    float64
	Tier          string
	IsRateLimited bool
	Timestamp     time.Time
}

// Package models defines data structures and domain types.
package models

import (
	"encoding/json"
	"maps"
	"time"
)

// Account represents a Google Cloud account with OAuth credentials.
// This is the unified account type used throughout the application.
type Account struct {
	ExpiresAt           time.Time        `json:"expiresAt"`
	LastUsed            time.Time        `json:"lastUsed"`
	AddedAt             time.Time        `json:"addedAt"`
	RateLimitResetTimes map[string]int64 `json:"rateLimitResetTimes,omitempty"`
	Picture             string           `json:"picture,omitempty"`
	AccessToken         string           `json:"accessToken,omitempty"`
	RefreshToken        string           `json:"refreshToken"`
	ProjectID           string           `json:"projectId,omitempty"`
	ManagedProjectID    string           `json:"managedProjectId,omitempty"`
	ID                  string           `json:"id"`
	DisplayName         string           `json:"displayName,omitempty"`
	Email               string           `json:"email"`
	IsActive            bool             `json:"isActive,omitempty"`
}

// GetEmail returns the account email (implements interface for quota service).
func (a *Account) GetEmail() string {
	return a.Email
}

// GetRefreshToken returns the refresh token (implements interface for quota service).
func (a *Account) GetRefreshToken() string {
	return a.RefreshToken
}

// Clone returns a deep copy of the account.
func (a *Account) Clone() Account {
	clone := Account{
		ID:               a.ID,
		Email:            a.Email,
		DisplayName:      a.DisplayName,
		Picture:          a.Picture,
		RefreshToken:     a.RefreshToken,
		AccessToken:      a.AccessToken,
		ExpiresAt:        a.ExpiresAt,
		ProjectID:        a.ProjectID,
		ManagedProjectID: a.ManagedProjectID,
		IsActive:         a.IsActive,
		AddedAt:          a.AddedAt,
		LastUsed:         a.LastUsed,
	}

	if a.RateLimitResetTimes != nil {
		clone.RateLimitResetTimes = make(map[string]int64, len(a.RateLimitResetTimes))
		maps.Copy(clone.RateLimitResetTimes, a.RateLimitResetTimes)
	}

	return clone
}

// AccountWithQuota combines account information with its current quota status.
type AccountWithQuota struct {
	QuotaInfo *QuotaInfo `json:"quotaInfo,omitempty"`
	Account
	IsActive bool `json:"isActive"`
}

// RawAccountData represents the JSON structure of an account in the accounts file.
// Used for backward-compatible JSON parsing from antigravity-accounts.json.
type RawAccountData struct {
	RateLimitResetTimes map[string]float64 `json:"rateLimitResetTimes,omitempty"`
	Email               string             `json:"email"`
	RefreshToken        string             `json:"refreshToken"`
	ProjectID           string             `json:"projectId"`
	ManagedProjectID    string             `json:"managedProjectId,omitempty"`
	AddedAt             json.RawMessage    `json:"addedAt,omitempty"`
	LastUsed            json.RawMessage    `json:"lastUsed,omitempty"`
}

// RawAccountsFile represents the top-level structure of the accounts JSON file.
type RawAccountsFile struct {
	Accounts []RawAccountData `json:"accounts"`
	Version  int              `json:"version"`
}

// ToAccount converts RawAccountData to Account, parsing date fields.
func (r *RawAccountData) ToAccount() Account {
	acc := Account{
		Email:            r.Email,
		RefreshToken:     r.RefreshToken,
		ProjectID:        r.ProjectID,
		ManagedProjectID: r.ManagedProjectID,
	}

	if r.RateLimitResetTimes != nil {
		acc.RateLimitResetTimes = make(map[string]int64, len(r.RateLimitResetTimes))
		for k, v := range r.RateLimitResetTimes {
			acc.RateLimitResetTimes[k] = int64(v)
		}
	}

	// Parse AddedAt - can be ISO string or Unix timestamp
	if len(r.AddedAt) > 0 {
		acc.AddedAt = parseTimeField(r.AddedAt)
	}

	// Parse LastUsed
	if len(r.LastUsed) > 0 {
		acc.LastUsed = parseTimeField(r.LastUsed)
	}

	return acc
}

// parseTimeField attempts to parse a JSON time value as either ISO string or Unix timestamp.
func parseTimeField(data json.RawMessage) time.Time {
	// Try as string first (ISO 8601)
	var strVal string
	if err := json.Unmarshal(data, &strVal); err == nil {
		if t, err := time.Parse(time.RFC3339, strVal); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339Nano, strVal); err == nil {
			return t
		}
		if t, err := time.Parse("2006-01-02T15:04:05.000Z", strVal); err == nil {
			return t
		}
	}

	// Try as number (Unix timestamp in milliseconds or seconds)
	var numVal float64
	if err := json.Unmarshal(data, &numVal); err == nil {
		if numVal > 1e12 {
			// Milliseconds
			return time.UnixMilli(int64(numVal))
		}
		// Seconds
		return time.Unix(int64(numVal), 0)
	}

	return time.Time{}
}

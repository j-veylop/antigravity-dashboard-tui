package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAccount_GetEmail(t *testing.T) {
	acc := Account{Email: "test@example.com"}

	if acc.GetEmail() != "test@example.com" {
		t.Errorf("GetEmail() = %q, want %q", acc.GetEmail(), "test@example.com")
	}
}

func TestAccount_GetRefreshToken(t *testing.T) {
	acc := Account{RefreshToken: "test-token"}

	if acc.GetRefreshToken() != "test-token" {
		t.Errorf("GetRefreshToken() = %q, want %q", acc.GetRefreshToken(), "test-token")
	}
}

func TestAccount_Clone(t *testing.T) {
	original := Account{
		ID:               "id-123",
		Email:            "test@example.com",
		DisplayName:      "Test User",
		RefreshToken:     "token-abc",
		ProjectID:        "project-1",
		ManagedProjectID: "managed-1",
		IsActive:         true,
		AddedAt:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		RateLimitResetTimes: map[string]int64{
			"claude": 1234567890,
			"gemini": 9876543210,
		},
	}

	clone := original.Clone()

	if clone.ID != original.ID {
		t.Errorf("clone.ID = %q, want %q", clone.ID, original.ID)
	}
	if clone.Email != original.Email {
		t.Errorf("clone.Email = %q, want %q", clone.Email, original.Email)
	}
	if !clone.AddedAt.Equal(original.AddedAt) {
		t.Errorf("clone.AddedAt = %v, want %v", clone.AddedAt, original.AddedAt)
	}

	if len(clone.RateLimitResetTimes) != len(original.RateLimitResetTimes) {
		t.Fatalf("RateLimitResetTimes length mismatch")
	}

	for k, v := range original.RateLimitResetTimes {
		if clone.RateLimitResetTimes[k] != v {
			t.Errorf("RateLimitResetTimes[%q] = %d, want %d", k, clone.RateLimitResetTimes[k], v)
		}
	}

	clone.RateLimitResetTimes["claude"] = 99999
	if original.RateLimitResetTimes["claude"] == 99999 {
		t.Error("modifying clone should not affect original (deep copy check)")
	}
}

func TestAccount_Clone_NilRateLimits(t *testing.T) {
	original := Account{
		ID:    "id-123",
		Email: "test@example.com",
	}

	clone := original.Clone()

	if clone.RateLimitResetTimes != nil {
		t.Errorf("clone.RateLimitResetTimes should be nil when original is nil")
	}
}

func TestRawAccountData_ToAccount(t *testing.T) {
	raw := RawAccountData{
		Email:            "test@example.com",
		RefreshToken:     "token-123",
		ProjectID:        "project-abc",
		ManagedProjectID: "managed-xyz",
		RateLimitResetTimes: map[string]float64{
			"claude": 1234567890,
			"gemini": 9876543210,
		},
	}

	acc := raw.ToAccount()

	if acc.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", acc.Email, "test@example.com")
	}

	if acc.RefreshToken != "token-123" {
		t.Errorf("RefreshToken = %q, want %q", acc.RefreshToken, "token-123")
	}

	if acc.ProjectID != "project-abc" {
		t.Errorf("ProjectID = %q, want %q", acc.ProjectID, "project-abc")
	}

	if acc.RateLimitResetTimes["claude"] != 1234567890 {
		t.Errorf("RateLimitResetTimes[claude] = %d, want 1234567890", acc.RateLimitResetTimes["claude"])
	}
}

func TestParseTimeField_RFC3339(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Time
	}{
		{
			name:  "RFC3339 format",
			input: `"2024-01-15T10:30:00Z"`,
			want:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:  "RFC3339Nano format",
			input: `"2024-01-15T10:30:00.123456789Z"`,
			want:  time.Date(2024, 1, 15, 10, 30, 0, 123456789, time.UTC),
		},
		{
			name:  "custom format with milliseconds",
			input: `"2024-01-15T10:30:00.123Z"`,
			want:  time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTimeField(json.RawMessage(tt.input))

			if !got.Equal(tt.want) {
				t.Errorf("parseTimeField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseTimeField_UnixTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Time
	}{
		{
			name:  "Unix seconds",
			input: `1705318200`,
			want:  time.Unix(1705318200, 0),
		},
		{
			name:  "Unix milliseconds",
			input: `1705318200000`,
			want:  time.UnixMilli(1705318200000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTimeField(json.RawMessage(tt.input))

			if !got.Equal(tt.want) {
				t.Errorf("parseTimeField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseTimeField_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "invalid JSON", input: `{invalid`},
		{name: "invalid format", input: `"not-a-date"`},
		{name: "empty string", input: `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTimeField(json.RawMessage(tt.input))

			if !got.IsZero() {
				t.Errorf("parseTimeField() with invalid input should return zero time, got %v", got)
			}
		})
	}
}

func TestParseTimeField_Boolean(t *testing.T) {
	got := parseTimeField(json.RawMessage(`true`))
	if !got.IsZero() {
		t.Errorf("parseTimeField(true) should return zero time, got %v", got)
	}
}

func TestParseTimeField_Boundary(t *testing.T) {
	// Case: Exactly 1e12. Should be seconds.
	input := `1000000000000` // 1e12
	got := parseTimeField(json.RawMessage(input))
	// 1e12 seconds from epoch
	expected := time.Unix(1000000000000, 0)
	if !got.Equal(expected) {
		t.Errorf("parseTimeField(1e12) = %v, want %v", got, expected)
	}

	// Case: 1e12 + 1. Should be milliseconds.
	input2 := `1000000000001`
	got2 := parseTimeField(json.RawMessage(input2))
	expected2 := time.UnixMilli(1000000000001)
	if !got2.Equal(expected2) {
		t.Errorf("parseTimeField(1e12+1) = %v, want %v", got2, expected2)
	}
}

func TestParseTimeField_Null(t *testing.T) {
	got := parseTimeField(json.RawMessage(`null`))

	if got.Unix() != 0 {
		t.Errorf("parseTimeField(null) = %v, expected Unix epoch or zero time", got)
	}
}

func TestRawAccountData_ToAccount_WithDates(t *testing.T) {
	addedAtJSON := json.RawMessage(`"2024-01-15T10:00:00Z"`)
	lastUsedJSON := json.RawMessage(`1705318200`)

	raw := RawAccountData{
		Email:        "test@example.com",
		RefreshToken: "token",
		AddedAt:      addedAtJSON,
		LastUsed:     lastUsedJSON,
	}

	acc := raw.ToAccount()

	if acc.AddedAt.IsZero() {
		t.Error("AddedAt should be parsed")
	}

	if acc.LastUsed.IsZero() {
		t.Error("LastUsed should be parsed")
	}

	expectedAddedAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	if !acc.AddedAt.Equal(expectedAddedAt) {
		t.Errorf("AddedAt = %v, want %v", acc.AddedAt, expectedAddedAt)
	}
}

func TestRawAccountData_ToAccount_EmptyDates(t *testing.T) {
	raw := RawAccountData{
		Email:        "test@example.com",
		RefreshToken: "token",
	}

	acc := raw.ToAccount()

	if !acc.AddedAt.IsZero() {
		t.Error("AddedAt should be zero when not provided")
	}

	if !acc.LastUsed.IsZero() {
		t.Error("LastUsed should be zero when not provided")
	}
}

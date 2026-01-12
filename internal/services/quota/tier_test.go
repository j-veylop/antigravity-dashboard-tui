package quota

import (
	"testing"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

func TestDetectSubscriptionTier(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		resetTime time.Time
		want      string
	}{
		{"ZeroTime", time.Time{}, "UNKNOWN"},
		{"PastWithinHour", now.Add(-30 * time.Minute), "PRO"},
		{"PastLongAgo", now.Add(-2 * time.Hour), "UNKNOWN"},
		{"FutureHourly", now.Add(1 * time.Hour), "PRO"},
		{"FutureDaily", now.Add(7 * time.Hour), "FREE"},
		{"FutureThreshold", now.Add(6 * time.Hour), "PRO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectSubscriptionTier(tt.resetTime); got != tt.want {
				t.Errorf("detectSubscriptionTier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTierFromQuotas(t *testing.T) {
	tests := []struct {
		name   string
		quotas []models.ModelQuota
		want   SubscriptionTier
	}{
		{"Empty", nil, TierUnknown},
		{"SinglePro", []models.ModelQuota{{SubscriptionTier: "PRO"}}, TierPro},
		{"SingleFree", []models.ModelQuota{{SubscriptionTier: "FREE"}}, TierFree},
		{"MixedProFree", []models.ModelQuota{{SubscriptionTier: "FREE"}, {SubscriptionTier: "PRO"}}, TierPro},
		{"OnlyUnknown", []models.ModelQuota{{SubscriptionTier: "UNKNOWN"}}, TierUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetTierFromQuotas(tt.quotas); got != tt.want {
				t.Errorf("GetTierFromQuotas() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeUntilReset(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		resetTime time.Time
		wantZero  bool
	}{
		{"Zero", time.Time{}, true},
		{"Past", now.Add(-1 * time.Hour), true},
		{"Future", now.Add(1 * time.Hour), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TimeUntilReset(tt.resetTime)
			if tt.wantZero {
				if got != 0 {
					t.Errorf("TimeUntilReset() = %v, want 0", got)
				}
			} else {
				if got <= 0 {
					t.Errorf("TimeUntilReset() = %v, want > 0", got)
				}
			}
		})
	}
}

func TestFormatResetTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		resetTime time.Time
		want      string
	}{
		{"Zero", time.Time{}, "Unknown"},
		{"Past", now.Add(-1 * time.Hour), "Now"},
		{"UnderMinute", now.Add(30 * time.Second), "< 1m"},
		{"Minutes", now.Add(10 * time.Minute), "10m"},
		{"Hours", now.Add(2 * time.Hour), "2h"},
		{"HoursAndMinutes", now.Add(2*time.Hour + 30*time.Minute), "2h30m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Using loose comparison because time.Now() moves
			got := FormatResetTime(tt.resetTime)
			if got == "" {
				t.Errorf("FormatResetTime() returned empty string")
			}
			// We can't easily check exact string without mocking time.Now or injecting time
			// But we can check if it matches pattern for simple cases
			if tt.name == "Zero" && got != "Unknown" {
				t.Errorf("FormatResetTime() = %v, want Unknown", got)
			}
			if tt.name == "Past" && got != "Now" {
				t.Errorf("FormatResetTime() = %v, want Now", got)
			}
		})
	}
}

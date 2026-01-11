package quota

import (
	"fmt"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

// SubscriptionTier represents the user's subscription level.
type SubscriptionTier string

const (
	// TierFree represents the free subscription tier.
	TierFree SubscriptionTier = "FREE"
	// TierPro represents the paid pro subscription tier.
	TierPro SubscriptionTier = "PRO"
	// TierUnknown represents an unknown subscription tier.
	TierUnknown SubscriptionTier = "UNKNOWN"
)

// TierThreshold is the reset time threshold for tier detection.
// PRO tier resets hourly (<=6 hours), FREE tier resets daily (>6 hours).
const TierThreshold = 6 * time.Hour

// detectSubscriptionTier determines the subscription tier based on reset time.
// PRO accounts have hourly quota resets (reset time <= 6 hours from now).
// FREE accounts have daily quota resets (reset time > 6 hours from now).
func detectSubscriptionTier(resetTime time.Time) string {
	if resetTime.IsZero() {
		return string(TierUnknown)
	}

	now := time.Now()
	duration := resetTime.Sub(now)

	// If reset time is in the past, we can't determine tier
	if duration < 0 {
		// Check if it was within the last hour (PRO likely)
		if duration > -1*time.Hour {
			return string(TierPro)
		}
		return string(TierUnknown)
	}

	// PRO tier: resets within 6 hours (hourly reset)
	if duration <= TierThreshold {
		return string(TierPro)
	}

	// FREE tier: resets after 6 hours (daily reset)
	return string(TierFree)
}

// GetTierFromQuotas determines the overall tier from multiple model quotas.
// If any model shows PRO tier, the account is PRO.
func GetTierFromQuotas(quotas []models.ModelQuota) SubscriptionTier {
	hasPro := false
	hasFree := false
	hasValid := false

	for _, q := range quotas {
		tier := SubscriptionTier(q.SubscriptionTier)
		if tier == TierUnknown || q.SubscriptionTier == "" {
			continue
		}

		hasValid = true
		switch tier {
		case TierPro:
			hasPro = true
		case TierFree:
			hasFree = true
		}
	}

	if !hasValid {
		return TierUnknown
	}

	if hasPro {
		return TierPro
	}

	if hasFree {
		return TierFree
	}

	return TierUnknown
}

// TierInfo provides detailed information about a subscription tier.
type TierInfo struct {
	Tier            SubscriptionTier `json:"tier"`
	DisplayName     string           `json:"displayName"`
	ResetInterval   string           `json:"resetInterval"`
	QuotaMultiplier float64          `json:"quotaMultiplier"`
}

// TimeUntilReset calculates the duration until quota reset.
func TimeUntilReset(resetTime time.Time) time.Duration {
	if resetTime.IsZero() {
		return 0
	}
	duration := time.Until(resetTime)
	if duration < 0 {
		return 0
	}
	return duration
}

// FormatResetTime formats the reset time for display.
func FormatResetTime(resetTime time.Time) string {
	if resetTime.IsZero() {
		return "Unknown"
	}

	duration := TimeUntilReset(resetTime)
	if duration <= 0 {
		return "Now"
	}

	if duration < time.Minute {
		return "< 1m"
	}

	if duration < time.Hour {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%dm", minutes)
	}

	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60

	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}

	return fmt.Sprintf("%dh%dm", hours, minutes)
}

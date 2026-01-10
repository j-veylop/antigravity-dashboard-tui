// Package quota provides quota fetching and caching services.
package quota

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/logger"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

var (
	// Antigravity endpoints from reference implementation
	antigravityEndpoints = []string{
		"https://cloudcode-pa.googleapis.com",
		"https://daily-cloudcode-pa.sandbox.googleapis.com",
	}

	// Antigravity headers from reference implementation
	antigravityHeaders = map[string]string{
		"User-Agent":        "antigravity/1.11.5 windows/amd64",
		"X-Goog-Api-Client": "google-cloud-sdk vscode_cloudshelleditor/0.1",
		"Client-Metadata":   `{"ideType":"IDE_UNSPECIFIED","platform":"PLATFORM_UNSPECIFIED","pluginType":"GEMINI"}`,
	}
)

const (
	// Google OAuth token endpoint
	googleOAuthURL = "https://oauth2.googleapis.com/token"

	// UserInfo endpoint
	userInfoEndpoint = "https://www.googleapis.com/oauth2/v2/userinfo"
)

// TokenResponse represents the OAuth token response from Google.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	TokenType    string `json:"token_type"`
	IDToken      string `json:"id_token,omitempty"`
}

// CachedToken represents a cached access token with expiration.
type CachedToken struct {
	AccessToken string
	ExpiresAt   time.Time
}

// IsValid checks if the cached token is still valid.
func (t *CachedToken) IsValid() bool {
	if t == nil || t.AccessToken == "" {
		return false
	}
	// Add 5 minute buffer before expiration
	return time.Now().Add(5 * time.Minute).Before(t.ExpiresAt)
}

// RefreshAccessToken exchanges a refresh token for a new access token.
func RefreshAccessToken(refreshToken, clientID, clientSecret string) (*TokenResponse, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token is empty")
	}

	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(context.Background(), "POST", googleOAuthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error("failed to close response body", "error", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// QuotaResponse represents the full quota API response.
type QuotaResponse struct {
	ModelQuotas []models.ModelQuota `json:"modelQuotas"`
}

// fetchModelsResponse represents the response from fetchAvailableModels API.
type fetchModelsResponse struct {
	Models map[string]struct {
		DisplayName string `json:"displayName"`
		QuotaInfo   struct {
			RemainingFraction float64 `json:"remainingFraction"`
			ResetTime         string  `json:"resetTime"`
		} `json:"quotaInfo"`
	} `json:"models"`
}

func normalizeModelFamily(name string) string {
	nameLower := strings.ToLower(name)
	if strings.Contains(nameLower, "claude") {
		return "claude"
	}
	if strings.Contains(nameLower, "gemini") {
		return "gemini"
	}
	return name
}

// FetchQuota retrieves quota information from the Google Cloud Code API.
func FetchQuota(accessToken string) (*QuotaResponse, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is empty")
	}

	var lastErr error

	// Try each endpoint
	for _, endpoint := range antigravityEndpoints {
		url := endpoint + "/v1internal:fetchAvailableModels"
		req, err := http.NewRequestWithContext(context.Background(), "POST", url, strings.NewReader("{}"))
		if err != nil {
			lastErr = fmt.Errorf("failed to create quota request: %w", err)
			continue
		}

		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")
		for k, v := range antigravityHeaders {
			req.Header.Set(k, v)
		}

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("quota request failed: %w", err)
			continue
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				logger.Error("failed to close response body", "error", err)
			}
		}()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read quota response: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("unauthorized: access token may be expired")
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("quota request failed (status %d): %s", resp.StatusCode, string(body))
			continue
		}

		var modelsResp fetchModelsResponse
		if err := json.Unmarshal(body, &modelsResp); err != nil {
			lastErr = fmt.Errorf("failed to parse quota response: %w", err)
			continue
		}

		// Convert to QuotaResponse format
		var modelQuotas []models.ModelQuota
		now := time.Now()

		for name, data := range modelsResp.Models {
			resetTimeStr := data.QuotaInfo.ResetTime
			var resetTime time.Time
			if resetTimeStr != "" {
				resetTime, _ = time.Parse(time.RFC3339, resetTimeStr)
			}

			remainingFraction := data.QuotaInfo.RemainingFraction
			// Assuming limit is 100 for percentage calculation relative to fraction
			limit := int64(100)
			used := int64(100 - (remainingFraction * 100))
			remaining := int64(remainingFraction * 100)

			var usagePercentage float64
			if used > 0 {
				usagePercentage = float64(used)
			}

			mq := models.ModelQuota{
				ModelFamily:      normalizeModelFamily(name),
				Tier:             string(detectSubscriptionTier(resetTime)),
				Used:             used,
				Limit:            limit,
				ResetTime:        resetTime,
				Remaining:        remaining,
				UsagePercentage:  usagePercentage,
				IsRateLimited:    remaining == 0,
				LastUpdated:      now,
				SubscriptionTier: string(detectSubscriptionTier(resetTime)),
			}
			modelQuotas = append(modelQuotas, mq)
		}

		return &QuotaResponse{ModelQuotas: modelQuotas}, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("failed to fetch quota from any endpoint")
}

// UserInfo represents user information from Google.
type UserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// FetchUserInfo retrieves user information from Google.
func FetchUserInfo(accessToken string) (*UserInfo, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is empty")
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", userInfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error("failed to close response body", "error", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read userinfo response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var userInfo UserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse userinfo response: %w", err)
	}

	return &userInfo, nil
}

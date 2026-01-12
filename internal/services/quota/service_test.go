package quota

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

// MockRoundTripper implements http.RoundTripper for testing
type MockRoundTripper struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

// MockAccountProvider implements AccountProvider for testing
type MockAccountProvider struct {
	Accounts map[string]*models.Account
	mu       sync.Mutex
}

func NewMockAccountProvider() *MockAccountProvider {
	return &MockAccountProvider{
		Accounts: make(map[string]*models.Account),
	}
}

func (m *MockAccountProvider) GetAccounts() []models.Account {
	m.mu.Lock()
	defer m.mu.Unlock()
	var accs []models.Account
	for _, a := range m.Accounts {
		accs = append(accs, *a)
	}
	return accs
}

func (m *MockAccountProvider) GetAccountByEmail(email string) *models.Account {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Accounts[email]
}

func (m *MockAccountProvider) UpdateAccountEmail(oldEmail, newEmail string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if acc, ok := m.Accounts[oldEmail]; ok {
		delete(m.Accounts, oldEmail)
		acc.Email = newEmail
		m.Accounts[newEmail] = acc
		return nil
	}
	return errors.New("account not found")
}

func TestService_GetAccessToken(t *testing.T) {
	provider := NewMockAccountProvider()
	email := "test@example.com"
	provider.Accounts[email] = &models.Account{
		Email:        email,
		RefreshToken: "valid-refresh-token",
	}

	mockTransport := &MockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.String() == googleOAuthURL {
				resp := TokenResponse{
					AccessToken: "new-access-token",
					ExpiresIn:   3600,
				}
				body, _ := json.Marshal(resp)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			}
			return nil, errors.New("unexpected request")
		},
	}

	svc := New(provider, DefaultConfig())
	svc.httpClient = &http.Client{Transport: mockTransport}

	// Test 1: Fetch new token
	token, err := svc.GetAccessToken(email)
	if err != nil {
		t.Fatalf("GetAccessToken failed: %v", err)
	}
	if token != "new-access-token" {
		t.Errorf("expected new-access-token, got %s", token)
	}

	// Test 2: Cache hit
	// Should not trigger HTTP request
	svc.httpClient.Transport = &MockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			t.Fatal("should not make HTTP request on cache hit")
			return nil, nil
		},
	}
	token, err = svc.GetAccessToken(email)
	if err != nil {
		t.Fatalf("GetAccessToken cached failed: %v", err)
	}
	if token != "new-access-token" {
		t.Errorf("expected cached token, got %s", token)
	}
}

func TestService_RefreshQuota(t *testing.T) {
	provider := NewMockAccountProvider()
	email := "test@example.com"
	provider.Accounts[email] = &models.Account{
		Email:        email,
		RefreshToken: "valid-refresh-token",
	}

	mockTransport := &MockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			// Token refresh
			if req.URL.String() == googleOAuthURL {
				resp := TokenResponse{
					AccessToken: "access-token",
					ExpiresIn:   3600,
				}
				body, _ := json.Marshal(resp)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			}
			// Quota fetch
			if strings.Contains(req.URL.String(), "fetchAvailableModels") {
				resp := fetchModelsResponse{
					Models: map[string]struct {
						DisplayName string `json:"displayName"`
						QuotaInfo   struct {
							ResetTime         string  `json:"resetTime"`
							RemainingFraction float64 `json:"remainingFraction"`
						} `json:"quotaInfo"`
					}{
						"claude-3-opus": {
							QuotaInfo: struct {
								ResetTime         string  `json:"resetTime"`
								RemainingFraction float64 `json:"remainingFraction"`
							}{
								ResetTime:         time.Now().Add(1 * time.Hour).Format(time.RFC3339),
								RemainingFraction: 0.8,
							},
						},
					},
				}
				body, _ := json.Marshal(resp)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			}
			// User info check
			if strings.Contains(req.URL.String(), "userinfo") {
				resp := UserInfo{Email: email}
				body, _ := json.Marshal(resp)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			}
			return nil, errors.New("unexpected request: " + req.URL.String())
		},
	}

	svc := New(provider, DefaultConfig())
	svc.httpClient = &http.Client{Transport: mockTransport}

	quota, err := svc.RefreshQuota(email)
	if err != nil {
		t.Fatalf("RefreshQuota failed: %v", err)
	}

	if len(quota.ModelQuotas) != 1 {
		t.Errorf("expected 1 model quota, got %d", len(quota.ModelQuotas))
	}
	if quota.ModelQuotas[0].ModelFamily != "claude" {
		t.Errorf("expected claude family, got %s", quota.ModelQuotas[0].ModelFamily)
	}
}

func TestService_RefreshAllQuotas(t *testing.T) {
	provider := NewMockAccountProvider()
	email1 := "test1@example.com"
	email2 := "test2@example.com"
	provider.Accounts[email1] = &models.Account{Email: email1, RefreshToken: "rt1"}
	provider.Accounts[email2] = &models.Account{Email: email2, RefreshToken: "rt2"}

	mockTransport := &MockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			// Simplified mock: just return success for everything
			if req.URL.String() == googleOAuthURL {
				body, _ := json.Marshal(TokenResponse{AccessToken: "at", ExpiresIn: 3600})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
			}
			if strings.Contains(req.URL.String(), "fetchAvailableModels") {
				body, _ := json.Marshal(fetchModelsResponse{})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
			}
			if strings.Contains(req.URL.String(), "userinfo") {
				// Return error to skip email update logic, ensuring distinct emails remain
				return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("error"))}, nil
			}
			return nil, nil
		},
	}

	svc := New(provider, DefaultConfig())
	svc.httpClient = &http.Client{Transport: mockTransport}

	// This is async inside but we wait
	svc.RefreshAllQuotas()

	stats := svc.GetStats()
	if stats.AccountCount != 2 {
		t.Errorf("expected 2 accounts in stats, got %d", stats.AccountCount)
	}
}

func TestService_Events(t *testing.T) {
	provider := NewMockAccountProvider()
	svc := New(provider, DefaultConfig())

	ch := svc.Events()
	if ch == nil {
		t.Error("Events() returned nil channel")
	}
}

func TestService_Start(t *testing.T) {
	provider := NewMockAccountProvider()
	// Use small interval to trigger poll
	config := DefaultConfig()
	config.PollInterval = 10 * time.Millisecond
	svc := New(provider, config)

	svc.httpClient = &http.Client{Transport: &MockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			// Fail all requests, we just want to see it running
			return nil, errors.New("mock transport")
		},
	}}

	svc.Start()
	defer svc.Close()

	time.Sleep(50 * time.Millisecond)
	// Just verify no panic and goroutine started
}

func TestService_Getters(t *testing.T) {
	provider := NewMockAccountProvider()
	svc := New(provider, DefaultConfig())
	email := "test@example.com"

	// Pre-populate cache
	quota := &models.QuotaInfo{AccountEmail: email, TotalLimit: 100}
	svc.mu.Lock()
	svc.quotaCache[email] = quota
	svc.mu.Unlock()

	// Test GetQuota
	got := svc.GetQuota(email)
	if got != quota {
		t.Errorf("GetQuota() = %v, want %v", got, quota)
	}

	// Test GetAllQuotas
	all := svc.GetAllQuotas()
	if len(all) != 1 {
		t.Errorf("GetAllQuotas() len = %d, want 1", len(all))
	}
	if all[email] != quota {
		t.Errorf("GetAllQuotas()[email] = %v, want %v", all[email], quota)
	}
}

func TestService_HandleQuotaError(t *testing.T) {
	provider := NewMockAccountProvider()
	email := "test@example.com"
	provider.Accounts[email] = &models.Account{Email: email, RefreshToken: "rt"}

	// Mock transport that fails everything
	svc := New(provider, DefaultConfig())
	svc.httpClient = &http.Client{Transport: &MockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}}

	// RefreshQuota should return error and cache it
	_, err := svc.RefreshQuota(email)
	if err == nil {
		t.Error("RefreshQuota() expected error")
	}

	q := svc.GetQuota(email)
	if q == nil {
		t.Fatal("GetQuota() returned nil after error")
	}
	if q.Error == "" {
		t.Error("QuotaInfo.Error should be set")
	}
}

func TestService_CheckEmailUpdate(t *testing.T) {
	provider := NewMockAccountProvider()
	oldEmail := "old@example.com"
	newEmail := "new@example.com"
	provider.Accounts[oldEmail] = &models.Account{Email: oldEmail, RefreshToken: "rt"}

	svc := New(provider, DefaultConfig())
	svc.httpClient = &http.Client{Transport: &MockRoundTripper{
		RoundTripFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "userinfo") {
				body, _ := json.Marshal(UserInfo{Email: newEmail})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
			}
			if req.URL.String() == googleOAuthURL {
				body, _ := json.Marshal(TokenResponse{AccessToken: "at"})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
			}
			// Quota fetch
			body, _ := json.Marshal(fetchModelsResponse{})
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body))}, nil
		},
	}}

	// This should trigger email update
	quota, err := svc.RefreshQuota(oldEmail)
	if err != nil {
		t.Fatalf("RefreshQuota failed: %v", err)
	}

	if quota.AccountEmail != newEmail {
		t.Errorf("expected quota email %s, got %s", newEmail, quota.AccountEmail)
	}

	// Verify provider was updated
	if provider.GetAccountByEmail(newEmail) == nil {
		t.Error("provider should have new email")
	}
	if provider.GetAccountByEmail(oldEmail) != nil {
		t.Error("provider should not have old email")
	}
}

func TestService_SendEvent_Full(t *testing.T) {
	provider := NewMockAccountProvider()
	svc := New(provider, DefaultConfig())
	// eventChan size is 100 in New. We can't change it easily without exposing it or using reflection.
	// But we can fill it.

	// Fill the channel
	for i := 0; i < 110; i++ {
		svc.sendEvent(Event{Type: EventQuotaUpdated})
	}

	// If it didn't block, we are good.
	// Check if we can read 100 events (some might be dropped)
	count := len(svc.Events())
	if count != 100 {
		t.Errorf("expected 100 events in buffer, got %d", count)
	}
}

func TestService_Close(t *testing.T) {
	svc := New(NewMockAccountProvider(), DefaultConfig())
	if err := svc.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

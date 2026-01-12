package quota

import (
	"fmt"
	"maps"
	"sync"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/logger"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

// AccountProvider is an interface for getting account information.
type AccountProvider interface {
	GetAccounts() []models.Account
	GetAccountByEmail(email string) *models.Account
	UpdateAccountEmail(oldEmail, newEmail string) error
}

// Event represents a quota service event.
type Event struct {
	Error        error
	QuotaInfo    *models.QuotaInfo
	AccountEmail string
	Type         EventType
}

// EventType defines the type of quota event.
type EventType int

const (
	// EventQuotaUpdated indicates that quota information has been updated.
	EventQuotaUpdated EventType = iota
	// EventQuotaRefreshing indicates that quota refresh is in progress.
	EventQuotaRefreshing
	// EventQuotaError indicates that an error occurred during quota refresh.
	EventQuotaError
	// EventTokenRefreshed indicates that an access token has been refreshed.
	EventTokenRefreshed
	// EventTokenError indicates that an error occurred during token refresh.
	EventTokenError
)

// Config holds configuration for the quota service.
type Config struct {
	ClientID        string
	ClientSecret    string
	PollInterval    time.Duration
	RefreshInterval time.Duration
	MaxConcurrent   int
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		PollInterval:    30 * time.Second,
		RefreshInterval: 5 * time.Minute,
		MaxConcurrent:   5,
	}
}

// Service manages quota fetching and caching.
type Service struct {
	accountProvider AccountProvider
	quotaCache      map[string]*models.QuotaInfo
	tokenCache      map[string]*CachedToken
	eventChan       chan Event
	stopChan        chan struct{}
	pollTicker      *time.Ticker
	refreshSem      chan struct{}
	config          Config
	mu              sync.RWMutex
}

// New creates a new quota service.
func New(provider AccountProvider, config Config) *Service {
	if config.PollInterval == 0 {
		config = DefaultConfig()
	}

	s := &Service{
		accountProvider: provider,
		quotaCache:      make(map[string]*models.QuotaInfo),
		tokenCache:      make(map[string]*CachedToken),
		eventChan:       make(chan Event, 100),
		stopChan:        make(chan struct{}),
		config:          config,
		refreshSem:      make(chan struct{}, config.MaxConcurrent),
	}

	// Start polling goroutine
	go s.pollQuota()

	return s
}

// Events returns the event channel.
func (s *Service) Events() <-chan Event {
	return s.eventChan
}

// GetAccessToken returns a valid access token for the account, refreshing if needed.
func (s *Service) GetAccessToken(email string) (string, error) {
	s.mu.RLock()
	cached, ok := s.tokenCache[email]
	s.mu.RUnlock()

	if ok && cached.IsValid() {
		return cached.AccessToken, nil
	}

	// Need to refresh
	var refreshToken string
	if s.accountProvider != nil {
		acc := s.accountProvider.GetAccountByEmail(email)
		if acc == nil {
			return "", fmt.Errorf("account not found: %s", email)
		}
		refreshToken = acc.RefreshToken
	}

	if refreshToken == "" {
		return "", fmt.Errorf("no refresh token for account: %s", email)
	}

	var tokenResp *TokenResponse
	var err error

	// Retry with exponential backoff
	backoff := 500 * time.Millisecond
	for i := range 3 {
		tokenResp, err = RefreshAccessToken(refreshToken, s.config.ClientID, s.config.ClientSecret)
		if err == nil {
			break
		}

		if i < 2 {
			time.Sleep(backoff)
			backoff *= 2
		}
	}

	if err != nil {
		s.sendEvent(Event{
			Type:         EventTokenError,
			AccountEmail: email,
			Error:        err,
		})
		return "", fmt.Errorf("failed to refresh token: %w", err)
	}

	// Cache the token
	s.mu.Lock()
	s.tokenCache[email] = &CachedToken{
		AccessToken: tokenResp.AccessToken,
		ExpiresAt:   time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}
	s.mu.Unlock()

	s.sendEvent(Event{
		Type:         EventTokenRefreshed,
		AccountEmail: email,
	})

	return tokenResp.AccessToken, nil
}

// GetQuota returns cached quota for an account.
func (s *Service) GetQuota(email string) *models.QuotaInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.quotaCache[email]
}

// RefreshQuota fetches fresh quota for an account.
func (s *Service) RefreshQuota(email string) (*models.QuotaInfo, error) {
	s.sendEvent(Event{
		Type:         EventQuotaRefreshing,
		AccountEmail: email,
	})

	accessToken, err := s.GetAccessToken(email)
	if err != nil {
		return s.handleQuotaError(email, err)
	}

	if newEmail := s.checkEmailUpdate(email, accessToken); newEmail != "" {
		email = newEmail
	}

	quotaResp, err := FetchQuota(accessToken)
	if err != nil {
		return s.handleQuotaError(email, err)
	}

	return s.processQuotaResponse(email, quotaResp)
}

func (s *Service) handleQuotaError(email string, err error) (*models.QuotaInfo, error) {
	quotaInfo := &models.QuotaInfo{
		AccountEmail: email,
		LastUpdated:  time.Now(),
		Error:        err.Error(),
	}
	s.mu.Lock()
	s.quotaCache[email] = quotaInfo
	s.mu.Unlock()
	s.sendEvent(Event{
		Type:         EventQuotaError,
		AccountEmail: email,
		QuotaInfo:    quotaInfo,
		Error:        err,
	})
	return quotaInfo, err
}

func (s *Service) checkEmailUpdate(email, accessToken string) string {
	if userInfo, err := FetchUserInfo(accessToken); err == nil && userInfo.Email != "" {
		if userInfo.Email != email {
			if err := s.accountProvider.UpdateAccountEmail(email, userInfo.Email); err == nil {
				return userInfo.Email
			}
		}
	}
	return ""
}

func (s *Service) processQuotaResponse(email string, quotaResp *Response) (*models.QuotaInfo, error) {
	quotaInfo := &models.QuotaInfo{
		AccountEmail:     email,
		ModelQuotas:      quotaResp.ModelQuotas,
		LastUpdated:      time.Now(),
		SubscriptionTier: string(GetTierFromQuotas(quotaResp.ModelQuotas)),
	}

	// Calculate aggregates
	for i := range quotaResp.ModelQuotas {
		mq := &quotaResp.ModelQuotas[i]
		quotaInfo.TotalRemaining += mq.Remaining
		quotaInfo.TotalLimit += mq.Limit
	}
	if quotaInfo.TotalLimit > 0 {
		quotaInfo.OverallPercent = float64(quotaInfo.TotalLimit-quotaInfo.TotalRemaining) /
			float64(quotaInfo.TotalLimit) * 100
	}

	s.mu.Lock()
	s.quotaCache[email] = quotaInfo
	s.mu.Unlock()

	s.sendEvent(Event{
		Type:         EventQuotaUpdated,
		AccountEmail: email,
		QuotaInfo:    quotaInfo,
	})

	return quotaInfo, nil
}

// GetAllQuotas returns all cached quotas.
func (s *Service) GetAllQuotas() map[string]*models.QuotaInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*models.QuotaInfo, len(s.quotaCache))
	maps.Copy(result, s.quotaCache)
	return result
}

// RefreshAllQuotas refreshes quota for all accounts.
func (s *Service) RefreshAllQuotas() {
	if s.accountProvider == nil {
		return
	}

	accounts := s.accountProvider.GetAccounts()
	var wg sync.WaitGroup

	for i := range accounts {
		acc := &accounts[i]
		wg.Add(1)
		go func(email string) {
			defer wg.Done()

			// Acquire semaphore
			s.refreshSem <- struct{}{}
			defer func() { <-s.refreshSem }()

			if _, err := s.RefreshQuota(email); err != nil {
				logger.Error("failed to refresh quota", "email", email, "error", err)
			}
		}(acc.Email)
	}

	wg.Wait()
}

// pollQuota runs the background polling goroutine.
func (s *Service) pollQuota() {
	// Initial refresh
	s.RefreshAllQuotas()

	s.pollTicker = time.NewTicker(s.config.PollInterval)
	defer s.pollTicker.Stop()

	for {
		select {
		case <-s.pollTicker.C:
			s.RefreshAllQuotas()
		case <-s.stopChan:
			return
		}
	}
}

// sendEvent sends an event to the event channel non-blocking.
func (s *Service) sendEvent(event Event) {
	select {
	case s.eventChan <- event:
	default:
		// Channel full, drop oldest
		select {
		case <-s.eventChan:
		default:
		}
		select {
		case s.eventChan <- event:
		default:
		}
	}
}

// Close stops the service and cleans up resources.
func (s *Service) Close() error {
	close(s.stopChan)
	return nil
}

// Stats returns statistics about the quota service.
type Stats struct {
	CachedQuotas   int
	CachedTokens   int
	TotalRemaining int64
	TotalLimit     int64
	AccountCount   int
}

// GetStats returns current statistics.
func (s *Service) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := Stats{
		CachedQuotas: len(s.quotaCache),
		CachedTokens: len(s.tokenCache),
	}

	for _, qi := range s.quotaCache {
		if qi.Error == "" {
			stats.AccountCount++
			stats.TotalRemaining += qi.TotalRemaining
			stats.TotalLimit += qi.TotalLimit
		}
	}

	return stats
}

// Package projection provides quota exhaustion time projection calculations
// based on historical usage data. It analyzes consumption rates over time
// to predict when API quotas will be depleted.
package projection

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"math"
	"sync"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/db"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/logger"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

const (
	bucketMinutes    = 5
	lowConfThreshold = 6
	medConfThreshold = 24
)

// Service handles projection calculations.
type Service struct {
	mu sync.RWMutex
	db *db.DB

	lastQuotas      map[string]*quotaState
	sessionIDs      map[string]string
	projectionCache map[string]*models.AccountProjection
}

type quotaState struct {
	claudePercent float64
	geminiPercent float64
	timestamp     time.Time
}

// New creates a new projection service.
func New(database *db.DB) *Service {
	return &Service{
		db:              database,
		lastQuotas:      make(map[string]*quotaState),
		sessionIDs:      make(map[string]string),
		projectionCache: make(map[string]*models.AccountProjection),
	}
}

// CalculateProjections calculates usage projections for an account.
func (s *Service) CalculateProjections(
	email string,
	claudePercent, geminiPercent float64,
	claudeReset, geminiReset time.Time,
) (*models.AccountProjection, error) {
	s.mu.RLock()
	sessionID := s.sessionIDs[email]
	s.mu.RUnlock()

	rates, err := s.db.GetConsumptionRates(email, sessionID)
	if err != nil {
		logger.Error("failed to get consumption rates", "email", email, "error", err)
		rates = &models.ConsumptionRates{Email: email}
	}

	historical, err := s.db.GetHistoricalContext(email)
	if err != nil {
		logger.Error("failed to get historical context", "email", email, "error", err)
	}

	proj := &models.AccountProjection{
		Email:       email,
		LastUpdated: time.Now(),
	}

	proj.Claude = s.calculateModelProjection(
		"claude",
		claudePercent,
		rates.SessionClaudeRate,
		rates.HistoricalClaudeRate,
		claudeReset,
		rates.SessionDataPoints,
		historical,
	)

	proj.Gemini = s.calculateModelProjection(
		"gemini",
		geminiPercent,
		rates.SessionGeminiRate,
		rates.HistoricalGeminiRate,
		geminiReset,
		rates.SessionDataPoints,
		historical,
	)

	s.mu.Lock()
	s.projectionCache[email] = proj
	s.mu.Unlock()

	return proj, nil
}

func (s *Service) calculateModelProjection(
	model string,
	currentPercent float64,
	sessionRate float64,
	historicalRate float64,
	resetTime time.Time,
	dataPoints int,
	historical *models.HistoricalContext,
) *models.ModelProjection {
	proj := s.initProjection(model, currentPercent, sessionRate, historicalRate, resetTime, dataPoints, historical)

	effectiveRate := s.calculateDepletion(proj, currentPercent, sessionRate, historicalRate)
	s.determineStatus(proj, currentPercent, effectiveRate, resetTime)

	if historical != nil {
		proj.VsLastMonth = s.formatComparison(sessionRate, historical.LastMonthRate)
		proj.VsHistorical = s.formatHistoricalComparison(sessionRate, historical.AllTimeAvgRate)
	}

	return proj
}

func (s *Service) initProjection(
	model string,
	currentPercent float64,
	sessionRate float64,
	historicalRate float64,
	resetTime time.Time,
	dataPoints int,
	historical *models.HistoricalContext,
) *models.ModelProjection {
	proj := &models.ModelProjection{
		Model:          model,
		CurrentPercent: currentPercent,
		SessionRate:    sessionRate,
		HistoricalRate: historicalRate,
		ResetTime:      resetTime,
		TimeUntilReset: time.Until(resetTime),
		DataPoints:     dataPoints,
		Historical:     historical,
		Status:         models.ProjectionUnknown,
		Confidence:     "low",
	}

	if proj.TimeUntilReset < 0 {
		proj.TimeUntilReset = 0
	}

	switch {
	case dataPoints < lowConfThreshold:
		proj.Confidence = "low"
	case dataPoints < medConfThreshold:
		proj.Confidence = "medium"
	default:
		proj.Confidence = "high"
	}

	return proj
}

func (s *Service) calculateDepletion(
	proj *models.ModelProjection,
	currentPercent float64,
	sessionRate float64,
	historicalRate float64,
) float64 {
	effectiveRate := sessionRate
	if effectiveRate <= 0 && historicalRate > 0 {
		effectiveRate = historicalRate
	}

	if effectiveRate > 0 {
		proj.SessionHoursLeft = currentPercent / effectiveRate
		proj.SessionDepleteAt = time.Now().Add(time.Duration(proj.SessionHoursLeft * float64(time.Hour)))
	} else {
		proj.SessionHoursLeft = math.Inf(1)
	}

	return effectiveRate
}

func (s *Service) determineStatus(
	proj *models.ModelProjection,
	currentPercent float64,
	effectiveRate float64,
	resetTime time.Time,
) {
	if !resetTime.IsZero() && effectiveRate > 0 {
		hoursUntilReset := proj.TimeUntilReset.Hours()
		neededToSurvive := effectiveRate * hoursUntilReset
		proj.WillDepleteBefore = currentPercent < neededToSurvive

		if proj.WillDepleteBefore {
			if proj.SessionHoursLeft < 1 {
				proj.Status = models.ProjectionCritical
			} else {
				proj.Status = models.ProjectionWarning
			}
		} else {
			proj.Status = models.ProjectionSafe
		}
	}
}

func (s *Service) formatComparison(current, reference float64) string {
	if reference <= 0 {
		return "No prior data"
	}
	diff := ((current - reference) / reference) * 100
	if math.Abs(diff) < 10 {
		return "Similar to last month"
	} else if diff > 0 {
		return fmt.Sprintf("%.0f%% higher than last month", diff)
	}
	return fmt.Sprintf("%.0f%% lower than last month", -diff)
}

func (s *Service) formatHistoricalComparison(current, allTimeAvg float64) string {
	if allTimeAvg <= 0 {
		return "Building history..."
	}
	diff := ((current - allTimeAvg) / allTimeAvg) * 100
	if math.Abs(diff) < 15 {
		return "Typical for you"
	} else if diff > 0 {
		return "Above your average"
	}
	return "Below your average"
}

// AggregateSnapshot aggregates quota data into a snapshot.
func (s *Service) AggregateSnapshot(
	email string,
	claudePercent, geminiPercent float64,
	tier, sessionID string,
) error {
	now := time.Now().UTC()
	bucketTime := now.Truncate(time.Duration(bucketMinutes) * time.Minute)

	s.mu.Lock()
	lastState := s.lastQuotas[email]
	s.mu.Unlock()

	claudeConsumed := 0.0
	geminiConsumed := 0.0

	if lastState != nil {
		timeSinceLast := now.Sub(lastState.timestamp)
		if timeSinceLast < 10*time.Minute {
			claudeDiff := lastState.claudePercent - claudePercent
			if claudeDiff > 0 && claudeDiff < 50 {
				claudeConsumed = claudeDiff
			}
			geminiDiff := lastState.geminiPercent - geminiPercent
			if geminiDiff > 0 && geminiDiff < 50 {
				geminiConsumed = geminiDiff
			}
		}
	}

	snapshot := &models.AggregatedSnapshot{
		Email:          email,
		BucketTime:     bucketTime,
		ClaudeQuotaAvg: claudePercent,
		GeminiQuotaAvg: geminiPercent,
		ClaudeConsumed: claudeConsumed,
		GeminiConsumed: geminiConsumed,
		SampleCount:    1,
		SessionID:      sessionID,
		Tier:           tier,
	}

	if err := s.db.UpsertAggregatedSnapshot(snapshot); err != nil {
		return fmt.Errorf("failed to upsert snapshot: %w", err)
	}

	s.mu.Lock()
	s.lastQuotas[email] = &quotaState{
		claudePercent: claudePercent,
		geminiPercent: geminiPercent,
		timestamp:     now,
	}
	s.mu.Unlock()

	return nil
}

// DetectSessionBoundary checks if a new session has started.
func (s *Service) DetectSessionBoundary(_ string, newPercent, oldPercent float64) bool {
	return newPercent > oldPercent+5
}

// GenerateSessionID creates a unique session ID based on reset time.
func (s *Service) GenerateSessionID(email string, resetTime time.Time) string {
	if resetTime.IsZero() {
		resetTime = time.Now()
	}
	data := fmt.Sprintf("%s:%d", email, resetTime.Unix())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("ses_%x", hash[:8])
}

// GetCachedProjection returns the cached projection for an email.
func (s *Service) GetCachedProjection(email string) *models.AccountProjection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.projectionCache[email]
}

// GetAllProjections returns all cached projections.
func (s *Service) GetAllProjections() map[string]*models.AccountProjection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*models.AccountProjection, len(s.projectionCache))
	maps.Copy(result, s.projectionCache)
	return result
}

// GetOrCreateSessionID gets or creates a session ID.
func (s *Service) GetOrCreateSessionID(email string, resetTime time.Time) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sid, ok := s.sessionIDs[email]; ok {
		return sid
	}

	sid := s.GenerateSessionID(email, resetTime)
	s.sessionIDs[email] = sid
	return sid
}

// ResetSession creates a new session ID for the account.
func (s *Service) ResetSession(email string, resetTime time.Time) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	sid := s.GenerateSessionID(email, resetTime)
	s.sessionIDs[email] = sid
	delete(s.lastQuotas, email)
	return sid
}

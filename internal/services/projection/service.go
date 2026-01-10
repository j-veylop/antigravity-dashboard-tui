package projection

import (
	"crypto/sha256"
	"fmt"
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

func New(database *db.DB) *Service {
	return &Service{
		db:              database,
		lastQuotas:      make(map[string]*quotaState),
		sessionIDs:      make(map[string]string),
		projectionCache: make(map[string]*models.AccountProjection),
	}
}

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

	if dataPoints < lowConfThreshold {
		proj.Confidence = "low"
	} else if dataPoints < medConfThreshold {
		proj.Confidence = "medium"
	} else {
		proj.Confidence = "high"
	}

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

	if historical != nil {
		proj.VsLastMonth = s.formatComparison(sessionRate, historical.LastMonthRate)
		proj.VsHistorical = s.formatHistoricalComparison(sessionRate, historical.AllTimeAvgRate)
	}

	return proj
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

func (s *Service) DetectSessionBoundary(email string, newPercent, oldPercent float64) bool {
	return newPercent > oldPercent+5
}

func (s *Service) GenerateSessionID(email string, resetTime time.Time) string {
	if resetTime.IsZero() {
		resetTime = time.Now()
	}
	data := fmt.Sprintf("%s:%d", email, resetTime.Unix())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("ses_%x", hash[:8])
}

func (s *Service) GetCachedProjection(email string) *models.AccountProjection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.projectionCache[email]
}

func (s *Service) GetAllProjections() map[string]*models.AccountProjection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*models.AccountProjection, len(s.projectionCache))
	for k, v := range s.projectionCache {
		result[k] = v
	}
	return result
}

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

func (s *Service) ResetSession(email string, resetTime time.Time) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	sid := s.GenerateSessionID(email, resetTime)
	s.sessionIDs[email] = sid
	delete(s.lastQuotas, email)
	return sid
}

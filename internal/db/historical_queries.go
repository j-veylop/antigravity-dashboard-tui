package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

var timeFormats = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05Z",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05 -0700 MST",
	"2006-01-02 15:04:05 +0000 UTC",
}

func parseTimeString(s string) (time.Time, bool) {
	for _, format := range timeFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// GetMonthlyStats returns aggregated stats per month.
func (db *DB) GetMonthlyStats(email string, months int) ([]models.PeriodStats, error) {
	query := `
		SELECT
			strftime('%Y-%m', bucket_time) as period,
			SUM(claude_consumed + gemini_consumed) as total_consumed,
			AVG(claude_consumed + gemini_consumed) * 12 as avg_rate,
			MAX(claude_consumed + gemini_consumed) * 12 as peak_rate,
			COUNT(DISTINCT session_id) as session_count,
			COUNT(*) as data_points,
			MIN(bucket_time) as start_time,
			MAX(bucket_time) as end_time
		FROM quota_snapshots_agg
		WHERE email = ? AND bucket_time >= datetime('now', ?)
		GROUP BY period
		ORDER BY period DESC
		LIMIT ?
	`

	windowStr := fmt.Sprintf("-%d months", months)
	rows, err := db.QueryContext(context.Background(), query, email, windowStr, months)
	if err != nil {
		return nil, fmt.Errorf("failed to query monthly stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var stats []models.PeriodStats
	for rows.Next() {
		var s models.PeriodStats
		var period sql.NullString
		var startTimeStr, endTimeStr sql.NullString
		err := rows.Scan(
			&period, &s.TotalConsumed, &s.AvgRatePerHour, &s.PeakRatePerHour,
			&s.SessionCount, &s.DataPoints, &startTimeStr, &endTimeStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monthly stats: %w", err)
		}
		if period.Valid {
			s.Period = period.String
		}
		if startTimeStr.Valid && startTimeStr.String != "" {
			if t, ok := parseTimeString(startTimeStr.String); ok {
				s.StartTime = t
			}
		}
		if endTimeStr.Valid && endTimeStr.String != "" {
			if t, ok := parseTimeString(endTimeStr.String); ok {
				s.EndTime = t
			}
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}

// GetUsagePatterns returns average consumption by hour and day of week.
func (db *DB) GetUsagePatterns(email string) ([]models.UsagePattern, error) {
	query := `
		SELECT
			day_of_week,
			hour,
			AVG(claude_consumed + gemini_consumed) as avg_consumed,
			COUNT(*) as occurrences
		FROM quota_snapshots_agg
		WHERE email = ?
		GROUP BY day_of_week, hour
		ORDER BY day_of_week, hour
	`

	rows, err := db.QueryContext(context.Background(), query, email)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage patterns: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var patterns []models.UsagePattern
	for rows.Next() {
		var p models.UsagePattern
		var dayOfWeek, hour sql.NullInt64
		err := rows.Scan(&dayOfWeek, &hour, &p.AvgConsumed, &p.Occurrences)
		if err != nil {
			return nil, fmt.Errorf("failed to scan usage pattern: %w", err)
		}
		if dayOfWeek.Valid {
			p.DayOfWeek = int(dayOfWeek.Int64)
		}
		if hour.Valid {
			p.Hour = int(hour.Int64)
		}
		patterns = append(patterns, p)
	}

	return patterns, rows.Err()
}

// GetHistoricalContext returns overall historical usage context.
func (db *DB) GetHistoricalContext(email string) (*models.HistoricalContext, error) {
	ctx := &models.HistoricalContext{}

	if err := db.getMonthlyRates(email, ctx); err != nil {
		return nil, err
	}

	if err := db.getAllTimeStats(email, ctx); err != nil {
		return nil, err
	}

	if err := db.getPeakUsage(email, ctx); err != nil {
		return nil, err
	}

	return ctx, nil
}

func (db *DB) getMonthlyRates(email string, ctx *models.HistoricalContext) error {
	currentMonthQuery := `
		SELECT COALESCE(AVG(claude_consumed + gemini_consumed) * 12, 0)
		FROM quota_snapshots_agg
		WHERE email = ? AND strftime('%Y-%m', bucket_time) = strftime('%Y-%m', 'now')
	`
	err := db.QueryRowContext(context.Background(), currentMonthQuery, email).Scan(&ctx.CurrentMonthRate)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to scan current month rate: %w", err)
	}

	lastMonthQuery := `
		SELECT COALESCE(AVG(claude_consumed + gemini_consumed) * 12, 0)
		FROM quota_snapshots_agg
		WHERE email = ? AND strftime('%Y-%m', bucket_time) = strftime('%Y-%m', 'now', '-1 month')
	`
	err = db.QueryRowContext(context.Background(), lastMonthQuery, email).Scan(&ctx.LastMonthRate)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to scan last month rate: %w", err)
	}

	if ctx.LastMonthRate > 0 {
		ctx.MonthOverMonthDiff = ((ctx.CurrentMonthRate - ctx.LastMonthRate) / ctx.LastMonthRate) * 100
	}
	return nil
}

func (db *DB) getAllTimeStats(email string, ctx *models.HistoricalContext) error {
	allTimeQuery := `
		SELECT
			COALESCE(AVG(claude_consumed + gemini_consumed) * 12, 0) as avg_rate,
			COALESCE(MAX(claude_consumed + gemini_consumed) * 12, 0) as peak_rate,
			COUNT(DISTINCT session_id) as total_sessions,
			MIN(bucket_time) as first_data,
			CAST(julianday('now') - julianday(MIN(bucket_time)) AS INTEGER) as total_days
		FROM quota_snapshots_agg
		WHERE email = ?
	`
	var firstDataStr sql.NullString
	var totalDays sql.NullInt64
	if err := db.QueryRowContext(context.Background(), allTimeQuery, email).Scan(
		&ctx.AllTimeAvgRate, &ctx.AllTimePeakRate, &ctx.TotalSessionsEver,
		&firstDataStr, &totalDays,
	); err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to scan all-time stats: %w", err)
	}
	if firstDataStr.Valid && firstDataStr.String != "" {
		if t, ok := parseTimeString(firstDataStr.String); ok {
			ctx.FirstDataPoint = t
		}
	}
	if totalDays.Valid {
		ctx.TotalDataDays = int(totalDays.Int64)
	}
	return nil
}

func (db *DB) getPeakUsage(email string, ctx *models.HistoricalContext) error {
	if err := db.getPeakUsageDay(email, ctx); err != nil {
		return err
	}
	return db.getPeakUsageHour(email, ctx)
}

func (db *DB) getPeakUsageDay(email string, ctx *models.HistoricalContext) error {
	peakDayQuery := `
		SELECT day_of_week
		FROM quota_snapshots_agg
		WHERE email = ?
		GROUP BY day_of_week
		ORDER BY SUM(claude_consumed + gemini_consumed) DESC
		LIMIT 1
	`
	var peakDayNum sql.NullInt64
	if err := db.QueryRowContext(context.Background(), peakDayQuery, email).Scan(&peakDayNum); err != nil &&
		err != sql.ErrNoRows {
		return fmt.Errorf("failed to scan peak day: %w", err)
	}
	if peakDayNum.Valid {
		days := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
		if peakDayNum.Int64 >= 0 && peakDayNum.Int64 < 7 {
			ctx.PeakUsageDay = days[peakDayNum.Int64]
		}
	}
	return nil
}

func (db *DB) getPeakUsageHour(email string, ctx *models.HistoricalContext) error {
	peakHourQuery := `
		SELECT hour
		FROM quota_snapshots_agg
		WHERE email = ?
		GROUP BY hour
		ORDER BY SUM(claude_consumed + gemini_consumed) DESC
		LIMIT 1
	`
	var peakHour sql.NullInt64
	if err := db.QueryRowContext(context.Background(), peakHourQuery, email).Scan(&peakHour); err != nil &&
		err != sql.ErrNoRows {
		return fmt.Errorf("failed to scan peak hour: %w", err)
	}
	if peakHour.Valid {
		ctx.PeakUsageHour = int(peakHour.Int64)
	}
	return nil
}

// GetFirstSnapshotTime returns the timestamp of the first recorded snapshot.
func (db *DB) GetFirstSnapshotTime(email string) (time.Time, error) {
	query := `SELECT MIN(bucket_time) FROM quota_snapshots_agg WHERE email = ?`
	var tStr sql.NullString
	err := db.QueryRowContext(context.Background(), query, email).Scan(&tStr)
	if err != nil {
		return time.Time{}, err
	}
	if tStr.Valid && tStr.String != "" {
		if t, ok := parseTimeString(tStr.String); ok {
			return t, nil
		}
	}
	return time.Time{}, nil
}

// GetRateLimitTransitions counts rate limit transitions (quota going from available to exhausted).
// A transition is detected when consumption goes from <99% to >=99% (near exhaustion).
func (db *DB) GetRateLimitTransitions(email string, days int) (*models.RateLimitStats, error) {
	stats := &models.RateLimitStats{}

	if err := db.getTransitionsInRange(email, days, stats); err != nil {
		return nil, err
	}

	db.getAllTimeTransitions(email, stats)
	db.getRecentTransitions(email, stats)

	if err := db.getTransitionsByDay(email, days, stats); err != nil {
		return nil, err
	}

	return stats, nil
}

func (db *DB) getTransitionsInRange(email string, days int, stats *models.RateLimitStats) error {
	timeFilter := ""
	args := []any{email}
	if days > 0 {
		timeFilter = "AND bucket_time >= datetime('now', ?)"
		args = append(args, fmt.Sprintf("-%d days", days))
	}

	transitionQuery := fmt.Sprintf(`
		WITH ordered_snapshots AS (
			SELECT 
				bucket_time,
				session_id,
				claude_consumed,
				gemini_consumed,
				LAG(claude_consumed) OVER (ORDER BY bucket_time) as prev_claude,
				LAG(gemini_consumed) OVER (ORDER BY bucket_time) as prev_gemini
			FROM quota_snapshots_agg
			WHERE email = ? %s
		)
		SELECT 
			COUNT(*) as total_hits,
			MAX(bucket_time) as last_hit
		FROM ordered_snapshots
		WHERE (claude_consumed >= 99 AND COALESCE(prev_claude, 0) < 99)
		   OR (gemini_consumed >= 99 AND COALESCE(prev_gemini, 0) < 99)
	`, timeFilter)

	var lastHitStr sql.NullString
	err := db.QueryRowContext(context.Background(), transitionQuery, args...).Scan(
		&stats.HitsInRange, &lastHitStr,
	)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to query rate limit transitions: %w", err)
	}
	if lastHitStr.Valid && lastHitStr.String != "" {
		if t, ok := parseTimeString(lastHitStr.String); ok {
			stats.LastHitTime = t
		}
	}
	return nil
}

func (db *DB) getAllTimeTransitions(email string, stats *models.RateLimitStats) {
	allTimeQuery := `
		WITH ordered_snapshots AS (
			SELECT 
				bucket_time,
				claude_consumed,
				gemini_consumed,
				LAG(claude_consumed) OVER (ORDER BY bucket_time) as prev_claude,
				LAG(gemini_consumed) OVER (ORDER BY bucket_time) as prev_gemini
			FROM quota_snapshots_agg
			WHERE email = ?
		)
		SELECT COUNT(*)
		FROM ordered_snapshots
		WHERE (claude_consumed >= 99 AND COALESCE(prev_claude, 0) < 99)
		   OR (gemini_consumed >= 99 AND COALESCE(prev_gemini, 0) < 99)
	`
	err := db.QueryRowContext(context.Background(), allTimeQuery, email).Scan(&stats.TotalHits)
	if err != nil && err != sql.ErrNoRows {
		stats.TotalHits = 0
	}
}

func (db *DB) getRecentTransitions(email string, stats *models.RateLimitStats) {
	last7Query := `
		WITH ordered_snapshots AS (
			SELECT 
				bucket_time,
				claude_consumed,
				gemini_consumed,
				LAG(claude_consumed) OVER (ORDER BY bucket_time) as prev_claude,
				LAG(gemini_consumed) OVER (ORDER BY bucket_time) as prev_gemini
			FROM quota_snapshots_agg
			WHERE email = ? AND bucket_time >= datetime('now', '-7 days')
		)
		SELECT COUNT(*)
		FROM ordered_snapshots
		WHERE (claude_consumed >= 99 AND COALESCE(prev_claude, 0) < 99)
		   OR (gemini_consumed >= 99 AND COALESCE(prev_gemini, 0) < 99)
	`
	if err := db.QueryRowContext(context.Background(), last7Query, email).Scan(&stats.HitsLast7Days); err != nil {
		stats.HitsLast7Days = 0
	}

	last30Query := `
		WITH ordered_snapshots AS (
			SELECT 
				bucket_time,
				claude_consumed,
				gemini_consumed,
				LAG(claude_consumed) OVER (ORDER BY bucket_time) as prev_claude,
				LAG(gemini_consumed) OVER (ORDER BY bucket_time) as prev_gemini
			FROM quota_snapshots_agg
			WHERE email = ? AND bucket_time >= datetime('now', '-30 days')
		)
		SELECT COUNT(*)
		FROM ordered_snapshots
		WHERE (claude_consumed >= 99 AND COALESCE(prev_claude, 0) < 99)
		   OR (gemini_consumed >= 99 AND COALESCE(prev_gemini, 0) < 99)
	`
	if err := db.QueryRowContext(context.Background(), last30Query, email).Scan(&stats.HitsLast30Days); err != nil {
		stats.HitsLast30Days = 0
	}
}

func (db *DB) getTransitionsByDay(email string, days int, stats *models.RateLimitStats) error {
	timeFilter := ""
	args := []any{email}
	if days > 0 {
		timeFilter = "AND bucket_time >= datetime('now', ?)"
		args = append(args, fmt.Sprintf("-%d days", days))
	}

	hitsByDayQuery := fmt.Sprintf(`
		WITH ordered_snapshots AS (
			SELECT 
				date(bucket_time) as hit_date,
				claude_consumed,
				gemini_consumed,
				LAG(claude_consumed) OVER (ORDER BY bucket_time) as prev_claude,
				LAG(gemini_consumed) OVER (ORDER BY bucket_time) as prev_gemini
			FROM quota_snapshots_agg
			WHERE email = ? %s
		)
		SELECT hit_date, COUNT(*) as count
		FROM ordered_snapshots
		WHERE (claude_consumed >= 99 AND COALESCE(prev_claude, 0) < 99)
		   OR (gemini_consumed >= 99 AND COALESCE(prev_gemini, 0) < 99)
		GROUP BY hit_date
		ORDER BY hit_date ASC
	`, timeFilter)

	rows, err := db.QueryContext(context.Background(), hitsByDayQuery, args...)
	if err != nil {
		return fmt.Errorf("failed to query hits by day: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var dateStr string
		var count int
		if err := rows.Scan(&dateStr, &count); err != nil {
			continue
		}
		if t, ok := parseTimeString(dateStr); ok {
			stats.HitsByDay = append(stats.HitsByDay, models.DailyHitCount{
				Date:  t,
				Count: count,
			})
		}
	}
	return nil
}

// GetSessionExhaustionStats calculates time-to-exhaustion statistics per session.
func (db *DB) GetSessionExhaustionStats(email string, days int) (*models.ExhaustionStats, error) {
	stats := &models.ExhaustionStats{}

	rows, err := db.querySessionStats(email, days)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	exhaustedDurations, startPercents := db.processSessionRows(rows, stats)

	db.calculateExhaustionStats(stats, exhaustedDurations, startPercents)

	return stats, nil
}

func (db *DB) querySessionStats(email string, days int) (*sql.Rows, error) {
	timeFilter := ""
	args := []any{email}
	if days > 0 {
		timeFilter = sqlTimeFilterClause
		args = append(args, fmt.Sprintf("-%d days", days))
	}

	sessionQuery := fmt.Sprintf(`
		SELECT 
			session_id,
			MIN(bucket_time) as start_time,
			MAX(bucket_time) as end_time,
			-- First consumption value
			(SELECT claude_consumed + gemini_consumed 
			 FROM quota_snapshots_agg q2 
			 WHERE q2.session_id = q1.session_id AND q2.email = q1.email
			 ORDER BY bucket_time ASC LIMIT 1) as start_consumed,
			-- Last consumption value  
			(SELECT claude_consumed + gemini_consumed 
			 FROM quota_snapshots_agg q2 
			 WHERE q2.session_id = q1.session_id AND q2.email = q1.email
			 ORDER BY bucket_time DESC LIMIT 1) as end_consumed,
			MAX(claude_consumed + gemini_consumed) as peak_consumed,
			COUNT(*) as data_points
		FROM quota_snapshots_agg q1
		WHERE email = ? AND session_id IS NOT NULL AND session_id != '' %s
		GROUP BY session_id
		HAVING COUNT(*) > 1
	`, timeFilter)

	rows, err := db.QueryContext(context.Background(), sessionQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query session stats: %w", err)
	}
	return rows, nil
}

func (db *DB) processSessionRows(rows *sql.Rows, stats *models.ExhaustionStats) (exhaustedDurations []time.Duration, startPercents []float64) {
	for rows.Next() {
		var sessionID string
		var startTimeStr, endTimeStr sql.NullString
		var startConsumed, endConsumed, peakConsumed sql.NullFloat64
		var dataPoints int

		if err := rows.Scan(&sessionID, &startTimeStr, &endTimeStr,
			&startConsumed, &endConsumed, &peakConsumed, &dataPoints); err != nil {
			continue
		}

		stats.TotalSessions++

		var startTime, endTime time.Time
		if startTimeStr.Valid {
			startTime, _ = parseTimeString(startTimeStr.String)
		}
		if endTimeStr.Valid {
			endTime, _ = parseTimeString(endTimeStr.String)
		}

		duration := endTime.Sub(startTime)

		// Session is "exhausted" if peak consumption >= 99%
		if peakConsumed.Valid && peakConsumed.Float64 >= 99 {
			stats.ExhaustedSessions++
			if duration > 0 {
				exhaustedDurations = append(exhaustedDurations, duration)

				// Track min/max
				if stats.MinTimeToExhaust == 0 || duration < stats.MinTimeToExhaust {
					stats.MinTimeToExhaust = duration
				}
				if duration > stats.MaxTimeToExhaust {
					stats.MaxTimeToExhaust = duration
				}
			}
		}

		// Track start percent (100 - consumed = remaining)
		if startConsumed.Valid {
			startPercents = append(startPercents, 100-startConsumed.Float64)
		}
	}
	return exhaustedDurations, startPercents
}

func (db *DB) calculateExhaustionStats(
	stats *models.ExhaustionStats,
	exhaustedDurations []time.Duration,
	startPercents []float64,
) {
	if stats.TotalSessions > 0 {
		if len(exhaustedDurations) > 0 {
			var totalExhausted time.Duration
			for _, d := range exhaustedDurations {
				totalExhausted += d
			}
			stats.AvgTimeToExhaust = totalExhausted / time.Duration(len(exhaustedDurations))

			// Median (simple approach - sort and take middle)
			if len(exhaustedDurations) > 0 {
				// Sort durations
				for i := 0; i < len(exhaustedDurations)-1; i++ {
					for j := i + 1; j < len(exhaustedDurations); j++ {
						if exhaustedDurations[j] < exhaustedDurations[i] {
							exhaustedDurations[i], exhaustedDurations[j] = exhaustedDurations[j], exhaustedDurations[i]
						}
					}
				}
				stats.MedianTimeToExhaust = exhaustedDurations[len(exhaustedDurations)/2]
			}
		}

		stats.ExhaustionRate = float64(stats.ExhaustedSessions) / float64(stats.TotalSessions) * 100

		if len(startPercents) > 0 {
			var totalStart float64
			for _, p := range startPercents {
				totalStart += p
			}
			stats.AvgStartPercent = totalStart / float64(len(startPercents))
		}
	}
}

// GetDailyUsageTrend returns daily consumption data for charts.
func (db *DB) GetDailyUsageTrend(email string, days int) ([]models.DailyUsagePoint, error) {
	// Build time filter
	timeFilter := ""
	args := []any{email}
	if days > 0 {
		timeFilter = sqlTimeFilterClause
		args = append(args, fmt.Sprintf("-%d days", days))
	}

	query := fmt.Sprintf(`
		SELECT 
			date(bucket_time) as day,
			SUM(claude_consumed) as claude_total,
			SUM(gemini_consumed) as gemini_total,
			SUM(claude_consumed + gemini_consumed) as total_consumed,
			COUNT(DISTINCT session_id) as session_count,
			COUNT(*) as data_points
		FROM quota_snapshots_agg
		WHERE email = ? %s
		GROUP BY day
		ORDER BY day ASC
	`, timeFilter)

	rows, err := db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily trend: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var points []models.DailyUsagePoint
	for rows.Next() {
		var p models.DailyUsagePoint
		var dateStr string
		var sessionCount, dataPoints sql.NullInt64

		if err := rows.Scan(&dateStr, &p.ClaudeConsumed, &p.GeminiConsumed,
			&p.TotalConsumed, &sessionCount, &dataPoints); err != nil {
			continue
		}

		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			p.Date = t
		}
		if sessionCount.Valid {
			p.SessionCount = int(sessionCount.Int64)
		}
		if dataPoints.Valid {
			p.DataPoints = int(dataPoints.Int64)
		}

		points = append(points, p)
	}

	return points, rows.Err()
}

// GetHourlyPatterns returns usage patterns by hour of day.
func (db *DB) GetHourlyPatterns(email string, days int) ([]models.HourlyPattern, error) {
	// Build time filter
	timeFilter := ""
	args := []any{email}
	if days > 0 {
		timeFilter = sqlTimeFilterClause
		args = append(args, fmt.Sprintf("-%d days", days))
	}

	query := fmt.Sprintf(`
		SELECT 
			hour,
			AVG(claude_consumed + gemini_consumed) as avg_consumed,
			COUNT(*) as occurrences
		FROM quota_snapshots_agg
		WHERE email = ? %s
		GROUP BY hour
		ORDER BY hour ASC
	`, timeFilter)

	rows, err := db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query hourly patterns: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Initialize all 24 hours
	patterns := make([]models.HourlyPattern, 24)
	for i := range 24 {
		patterns[i] = models.HourlyPattern{Hour: i}
	}

	for rows.Next() {
		var hour sql.NullInt64
		var avgConsumed float64
		var occurrences int

		if err := rows.Scan(&hour, &avgConsumed, &occurrences); err != nil {
			continue
		}

		if hour.Valid && hour.Int64 >= 0 && hour.Int64 < 24 {
			patterns[hour.Int64].AvgConsumed = avgConsumed
			patterns[hour.Int64].Occurrences = occurrences
		}
	}

	return patterns, rows.Err()
}

// GetWeekdayPatterns returns usage patterns by day of week.
func (db *DB) GetWeekdayPatterns(email string, days int) ([]models.WeekdayPattern, error) {
	dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

	// Build time filter
	timeFilter := ""
	args := []any{email}
	if days > 0 {
		timeFilter = sqlTimeFilterClause
		args = append(args, fmt.Sprintf("-%d days", days))
	}

	query := fmt.Sprintf(`
		SELECT 
			day_of_week,
			AVG(claude_consumed + gemini_consumed) as avg_consumed,
			COUNT(*) as occurrences
		FROM quota_snapshots_agg
		WHERE email = ? %s
		GROUP BY day_of_week
		ORDER BY day_of_week ASC
	`, timeFilter)

	rows, err := db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query weekday patterns: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Initialize all 7 days
	patterns := make([]models.WeekdayPattern, 7)
	for i := range 7 {
		patterns[i] = models.WeekdayPattern{
			DayOfWeek: i,
			DayName:   dayNames[i],
		}
	}

	for rows.Next() {
		var dow sql.NullInt64
		var avgConsumed float64
		var occurrences int

		if err := rows.Scan(&dow, &avgConsumed, &occurrences); err != nil {
			continue
		}

		if dow.Valid && dow.Int64 >= 0 && dow.Int64 < 7 {
			patterns[dow.Int64].AvgConsumed = avgConsumed
			patterns[dow.Int64].Occurrences = occurrences
		}
	}

	return patterns, rows.Err()
}

// GetAccountHistoryStats retrieves all history statistics for an account.
func (db *DB) GetAccountHistoryStats(email string, timeRange models.TimeRange) (*models.AccountHistoryStats, error) {
	days := timeRange.Days()

	stats := &models.AccountHistoryStats{
		Email:       email,
		TimeRange:   timeRange,
		LastUpdated: time.Now(),
	}

	// Get rate limit stats
	rateLimits, err := db.GetRateLimitTransitions(email, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit stats: %w", err)
	}
	stats.RateLimits = rateLimits

	// Get exhaustion stats
	exhaustion, err := db.GetSessionExhaustionStats(email, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get exhaustion stats: %w", err)
	}
	stats.Exhaustion = exhaustion

	// Get daily usage trend
	dailyUsage, err := db.GetDailyUsageTrend(email, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily usage: %w", err)
	}
	stats.DailyUsage = dailyUsage

	// Get hourly patterns
	hourlyPatterns, err := db.GetHourlyPatterns(email, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get hourly patterns: %w", err)
	}
	stats.HourlyPatterns = hourlyPatterns

	// Get weekday patterns
	weekdayPatterns, err := db.GetWeekdayPatterns(email, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get weekday patterns: %w", err)
	}
	stats.WeekdayPatterns = weekdayPatterns

	// Get historical context
	historical, err := db.GetHistoricalContext(email)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical context: %w", err)
	}
	stats.Historical = historical

	// Get first/last data points
	firstTime, err := db.GetFirstSnapshotTime(email)
	if err != nil {
		return nil, fmt.Errorf("failed to get first snapshot time: %w", err)
	}
	stats.FirstDataPoint = firstTime
	if len(dailyUsage) > 0 {
		stats.LastDataPoint = dailyUsage[len(dailyUsage)-1].Date
	}

	// Calculate totals
	for _, d := range dailyUsage {
		stats.TotalDataPoints += d.DataPoints
	}
	stats.TotalDataDays = len(dailyUsage)

	return stats, nil
}

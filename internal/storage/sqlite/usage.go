// Package sqlite provides SQLite implementation of API usage tracking operations
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aigateway/internal/models"
)

// ========================================
// API Usage CRUD Operations
// ========================================

// RecordAPIUsage записывает факт использования API
func (s *SQLiteDB) RecordAPIUsage(ctx context.Context, usage *models.APIUsage) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	// Serialize metadata to JSON if present
	var metadataJSON []byte
	var err error
	if usage.Metadata != nil {
		metadataJSON, err = json.Marshal(usage.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO api_usage (
			id, user_id, tenant_id, api_key_id,
			endpoint, method, model,
			status_code, success, error_message,
			prompt_tokens, completion_tokens, total_tokens,
			duration_ms, created_at,
			user_agent, ip_address, conversation_id,
			metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		usage.ID,
		usage.UserID,
		usage.TenantID,
		usage.APIKeyID,
		usage.Endpoint,
		usage.Method,
		usage.Model,
		usage.StatusCode,
		usage.Success,
		usage.ErrorMessage,
		usage.PromptTokens,
		usage.CompletionTokens,
		usage.TotalTokens,
		usage.DurationMS,
		usage.CreatedAt,
		usage.UserAgent,
		usage.IPAddress,
		usage.ConversationID,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert API usage: %w", err)
	}

	// DEBUG: Log successful insert
	s.logger.WithFields(map[string]any{
		"id":                usage.ID,
		"user_id":           usage.UserID,
		"api_key_id":        usage.APIKeyID,
		"model":             usage.Model,
		"endpoint":          usage.Endpoint,
		"prompt_tokens":     usage.PromptTokens,
		"completion_tokens": usage.CompletionTokens,
		"total_tokens":      usage.TotalTokens,
		"success":           usage.Success,
	}).Debug("API usage recorded successfully")

	return nil
}

// GetUserUsageStats возвращает статистику использования для пользователя за период
func (s *SQLiteDB) GetUserUsageStats(ctx context.Context, userID string, period time.Duration) (*models.UsageStats, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Calculate start time
	startTime := time.Now().Add(-period)

	// Query for aggregate stats
	query := `
		SELECT 
			COALESCE(COUNT(*), 0) as total_requests,
			COALESCE(SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END), 0) as successful_requests,
			COALESCE(SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END), 0) as failed_requests,
			COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as completion_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			AVG(duration_ms) as avg_duration_ms,
			MIN(duration_ms) as min_duration_ms,
			MAX(duration_ms) as max_duration_ms
		FROM api_usage
		WHERE user_id = ? AND created_at >= ?
	`

	var stats models.UsageStats
	var avgDuration, minDuration, maxDuration sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, userID, startTime).Scan(
		&stats.TotalRequests,
		&stats.SuccessfulRequests,
		&stats.FailedRequests,
		&stats.PromptTokens,
		&stats.CompletionTokens,
		&stats.TotalTokens,
		&avgDuration,
		&minDuration,
		&maxDuration,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}

	// Handle nullable durations
	if avgDuration.Valid {
		stats.AvgDurationMS = avgDuration.Float64
		stats.AvgDuration = int64(avgDuration.Float64) // Для фронтенда
	}
	if minDuration.Valid {
		stats.MinDurationMS = int64(minDuration.Float64)
	}
	if maxDuration.Valid {
		stats.MaxDurationMS = int64(maxDuration.Float64)
	}

	// Calculate success rate and error rate
	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(stats.SuccessfulRequests) / float64(stats.TotalRequests)
		stats.ErrorRate = float64(stats.FailedRequests) / float64(stats.TotalRequests)
	}

	// Query for model usage breakdown (OPTIMIZED - single query with success count)
	modelQuery := `
		SELECT 
			model,
			COUNT(*) as request_count,
			SUM(total_tokens) as total_tokens,
			AVG(duration_ms) as avg_duration,
			SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as success_count
		FROM api_usage
		WHERE user_id = ? AND created_at >= ?
		GROUP BY model
		ORDER BY request_count DESC
	`

	rows, err := s.db.QueryContext(ctx, modelQuery, userID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get model usage: %w", err)
	}
	defer rows.Close()

	stats.ModelUsage = make(map[string]*models.ModelUsageStats)
	stats.Models = make([]*models.ModelUsageStats, 0) // Инициализируем массив для фронтенда

	for rows.Next() {
		usage := &models.ModelUsageStats{}
		var avgDur sql.NullFloat64
		var successCount int64

		if err := rows.Scan(&usage.Model, &usage.RequestCount, &usage.TotalTokens, &avgDur, &successCount); err != nil {
			return nil, fmt.Errorf("failed to scan model usage: %w", err)
		}

		// Set duplicate fields for frontend
		usage.Requests = usage.RequestCount
		usage.Tokens = usage.TotalTokens

		if avgDur.Valid {
			usage.AvgDurationMS = avgDur.Float64
			usage.AvgDuration = int64(avgDur.Float64)
		}

		// Calculate success rate from aggregated data (NO additional query!)
		if usage.RequestCount > 0 {
			usage.SuccessRate = float64(successCount) / float64(usage.RequestCount)
		}

		stats.ModelUsage[usage.Model] = usage
		stats.Models = append(stats.Models, usage) // Добавляем в массив для фронтенда
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating model usage: %w", err)
	}

	// Query for endpoint usage breakdown (OPTIMIZED - single query with success count)
	endpointQuery := `
		SELECT 
			endpoint,
			COUNT(*) as request_count,
			SUM(total_tokens) as total_tokens,
			SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as success_count
		FROM api_usage
		WHERE user_id = ? AND created_at >= ?
		GROUP BY endpoint
		ORDER BY request_count DESC
	`

	rows, err = s.db.QueryContext(ctx, endpointQuery, userID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint usage: %w", err)
	}
	defer rows.Close()

	stats.EndpointUsage = make(map[string]*models.EndpointUsageStats)
	for rows.Next() {
		usage := &models.EndpointUsageStats{}
		var successCount int64

		if err := rows.Scan(&usage.Endpoint, &usage.RequestCount, &usage.TotalTokens, &successCount); err != nil {
			return nil, fmt.Errorf("failed to scan endpoint usage: %w", err)
		}

		// Calculate success rate from aggregated data (NO additional query!)
		if usage.RequestCount > 0 {
			usage.SuccessRate = float64(successCount) / float64(usage.RequestCount)
		}

		stats.EndpointUsage[usage.Endpoint] = usage
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating endpoint usage: %w", err)
	}

	// Query for API key usage (LIMIT to top 10 keys)
	// COALESCE используется для группировки WebUI (JWT) запросов с api_key_id=NULL
	// MAX(created_at) возвращает raw timestamp - парсим в Go с fallback
	apiKeyQuery := `
		SELECT 
			COALESCE(api_key_id, 'webui-jwt') as api_key_id,
			COUNT(*) as requests,
			SUM(total_tokens) as tokens,
			MAX(created_at) as last_used
		FROM api_usage
		WHERE user_id = ? AND created_at >= ?
		GROUP BY COALESCE(api_key_id, 'webui-jwt')
		ORDER BY requests DESC
		LIMIT 10
	`

	rows, err = s.db.QueryContext(ctx, apiKeyQuery, userID, startTime)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get API key usage")
	} else {
		defer rows.Close()
		stats.APIKeys = make([]*models.APIKeyUsageStats, 0)

		for rows.Next() {
			apiKey := &models.APIKeyUsageStats{}
			var lastUsedStr sql.NullString
			if err := rows.Scan(&apiKey.KeyID, &apiKey.Requests, &apiKey.Tokens, &lastUsedStr); err != nil {
				s.logger.WithError(err).Warn("Failed to scan API key usage")
				continue
			}

			// Parse timestamp with multiple format support
			if lastUsedStr.Valid && lastUsedStr.String != "" {
				apiKey.LastUsed = s.parseTimestamp(lastUsedStr.String)
			} else {
				apiKey.LastUsed = time.Time{} // Zero time if NULL
			}

			stats.APIKeys = append(stats.APIKeys, apiKey)
		}
	}

	// Query for recent requests
	// COALESCE для api_key_id так как WebUI (JWT) запросы имеют NULL
	// created_at raw timestamp - парсим в Go с fallback
	recentQuery := `
		SELECT 
			created_at,
			model,
			COALESCE(api_key_id, 'webui-jwt') as api_key_id,
			total_tokens,
			duration_ms,
			success
		FROM api_usage
		WHERE user_id = ? AND created_at >= ?
		ORDER BY created_at DESC
		LIMIT 20
	`

	rows, err = s.db.QueryContext(ctx, recentQuery, userID, startTime)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get recent requests")
	} else {
		defer rows.Close()
		stats.RecentRequests = make([]*models.RecentRequest, 0)

		for rows.Next() {
			req := &models.RecentRequest{}
			var timestampStr sql.NullString
			if err := rows.Scan(&timestampStr, &req.Model, &req.KeyID, &req.Tokens, &req.Duration, &req.Success); err != nil {
				s.logger.WithError(err).Warn("Failed to scan recent request")
				continue
			}

			// Parse timestamp with multiple format support
			if timestampStr.Valid && timestampStr.String != "" {
				req.Timestamp = s.parseTimestamp(timestampStr.String)
			} else {
				req.Timestamp = time.Now() // Fallback to current time
			}

			stats.RecentRequests = append(stats.RecentRequests, req)
		}
	}

	// Set period information
	stats.StartDate = startTime
	stats.EndDate = time.Now()

	s.logger.WithFields(map[string]any{
		"user_id":        userID,
		"total_requests": stats.TotalRequests,
		"total_tokens":   stats.TotalTokens,
	}).Debug("Retrieved user usage stats")

	return &stats, nil
}

// parseTimestamp пытается распарсить timestamp в разных форматах
func (s *SQLiteDB) parseTimestamp(ts string) time.Time {
	// Список форматов для пробы
	formats := []string{
		"2006-01-02 15:04:05", // SQLite datetime
		time.RFC3339,          // 2006-01-02T15:04:05Z07:00
		time.RFC3339Nano,      // 2006-01-02T15:04:05.999999999Z07:00
		"2006-01-02 15:04:05.999999999 -0700 MST", // Go time.String() format
	}

	for _, format := range formats {
		if t, err := time.Parse(format, ts); err == nil {
			return t
		}
	}

	// Если ничего не подошло, попробуем удалить "m=+XXX" из конца (monotonic clock)
	if idx := strings.Index(ts, " m="); idx > 0 {
		cleaned := ts[:idx]
		return s.parseTimestamp(cleaned) // Рекурсивно с очищенной строкой
	}

	s.logger.Warnf("Failed to parse timestamp in any format: %s", ts)
	return time.Time{} // Zero time as last resort
}

// GetTenantUsageStats возвращает статистику использования для tenant за период
func (s *SQLiteDB) GetTenantUsageStats(ctx context.Context, tenantID string, period time.Duration) (*models.UsageStats, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Calculate start time
	startTime := time.Now().Add(-period)

	// Query for aggregate stats
	query := `
		SELECT 
			COALESCE(COUNT(*), 0) as total_requests,
			COALESCE(SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END), 0) as successful_requests,
			COALESCE(SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END), 0) as failed_requests,
			COALESCE(SUM(prompt_tokens), 0) as prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) as completion_tokens,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			AVG(duration_ms) as avg_duration_ms,
			MIN(duration_ms) as min_duration_ms,
			MAX(duration_ms) as max_duration_ms
		FROM api_usage
		WHERE tenant_id = ? AND created_at >= ?
	`

	var stats models.UsageStats
	var avgDuration, minDuration, maxDuration sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, tenantID, startTime).Scan(
		&stats.TotalRequests,
		&stats.SuccessfulRequests,
		&stats.FailedRequests,
		&stats.PromptTokens,
		&stats.CompletionTokens,
		&stats.TotalTokens,
		&avgDuration,
		&minDuration,
		&maxDuration,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}

	// Handle nullable durations
	if avgDuration.Valid {
		stats.AvgDurationMS = avgDuration.Float64
		stats.AvgDuration = int64(avgDuration.Float64)
	}
	if minDuration.Valid {
		stats.MinDurationMS = int64(minDuration.Float64)
	}
	if maxDuration.Valid {
		stats.MaxDurationMS = int64(maxDuration.Float64)
	}

	// Calculate success rate and error rate
	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(stats.SuccessfulRequests) / float64(stats.TotalRequests)
		stats.ErrorRate = float64(stats.FailedRequests) / float64(stats.TotalRequests)
	}

	// Query for model usage breakdown (OPTIMIZED - single query with success count)
	modelQuery := `
		SELECT 
			model,
			COUNT(*) as request_count,
			SUM(total_tokens) as total_tokens,
			AVG(duration_ms) as avg_duration,
			SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as success_count
		FROM api_usage
		WHERE tenant_id = ? AND created_at >= ?
		GROUP BY model
		ORDER BY request_count DESC
	`

	rows, err := s.db.QueryContext(ctx, modelQuery, tenantID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get model usage: %w", err)
	}
	defer rows.Close()

	stats.ModelUsage = make(map[string]*models.ModelUsageStats)
	stats.Models = make([]*models.ModelUsageStats, 0)

	for rows.Next() {
		usage := &models.ModelUsageStats{}
		var avgDur sql.NullFloat64
		var successCount int64

		if err := rows.Scan(&usage.Model, &usage.RequestCount, &usage.TotalTokens, &avgDur, &successCount); err != nil {
			return nil, fmt.Errorf("failed to scan model usage: %w", err)
		}

		// Set duplicate fields for frontend
		usage.Requests = usage.RequestCount
		usage.Tokens = usage.TotalTokens

		if avgDur.Valid {
			usage.AvgDurationMS = avgDur.Float64
			usage.AvgDuration = int64(avgDur.Float64)
		}

		// Calculate success rate from aggregated data (NO additional query!)
		if usage.RequestCount > 0 {
			usage.SuccessRate = float64(successCount) / float64(usage.RequestCount)
		}

		stats.ModelUsage[usage.Model] = usage
		stats.Models = append(stats.Models, usage)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating model usage: %w", err)
	}

	// Query for endpoint usage breakdown (OPTIMIZED - single query with success count)
	endpointQuery := `
		SELECT 
			endpoint,
			COUNT(*) as request_count,
			SUM(total_tokens) as total_tokens,
			SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as success_count
		FROM api_usage
		WHERE tenant_id = ? AND created_at >= ?
		GROUP BY endpoint
		ORDER BY request_count DESC
	`

	rows, err = s.db.QueryContext(ctx, endpointQuery, tenantID, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint usage: %w", err)
	}
	defer rows.Close()

	stats.EndpointUsage = make(map[string]*models.EndpointUsageStats)
	for rows.Next() {
		usage := &models.EndpointUsageStats{}
		var successCount int64

		if err := rows.Scan(&usage.Endpoint, &usage.RequestCount, &usage.TotalTokens, &successCount); err != nil {
			return nil, fmt.Errorf("failed to scan endpoint usage: %w", err)
		}

		// Calculate success rate from aggregated data (NO additional query!)
		if usage.RequestCount > 0 {
			usage.SuccessRate = float64(successCount) / float64(usage.RequestCount)
		}

		stats.EndpointUsage[usage.Endpoint] = usage
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating endpoint usage: %w", err)
	}

	// Set period information
	stats.StartDate = startTime
	stats.EndDate = time.Now()

	s.logger.WithFields(map[string]any{
		"tenant_id":      tenantID,
		"total_requests": stats.TotalRequests,
		"total_tokens":   stats.TotalTokens,
	}).Debug("Retrieved tenant usage stats")

	return &stats, nil
}

// ========================================
// Reports Statistics Methods (v1.6.3+)
// ========================================

// GetUsageStats возвращает статистику использования за период для отчетов
func (s *SQLiteDB) GetUsageStats(ctx context.Context, start, end time.Time) (*models.UsageReportStats, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	stats := &models.UsageReportStats{}

	// Get aggregate statistics
	query := `
SELECT
COUNT(*) as total_requests,
SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as successful_requests,
SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as failed_requests,
SUM(total_tokens) as total_tokens,
COUNT(DISTINCT user_id) as unique_users,
COUNT(DISTINCT model) as unique_models
FROM api_usage
WHERE created_at >= ? AND created_at <= ?
`

	err := s.db.QueryRowContext(ctx, query, start, end).Scan(
		&stats.TotalRequests,
		&stats.SuccessfulRequests,
		&stats.FailedRequests,
		&stats.TotalTokens,
		&stats.UniqueUsers,
		&stats.UniqueModels,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}

	// Get top models
	modelQuery := `
SELECT model, COUNT(*) as requests, SUM(total_tokens) as tokens
FROM api_usage
WHERE created_at >= ? AND created_at <= ?
GROUP BY model
ORDER BY requests DESC
LIMIT 10
`

	rows, err := s.db.QueryContext(ctx, modelQuery, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get top models: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var model struct {
			Model    string
			Requests int64
			Tokens   int64
		}
		if err := rows.Scan(&model.Model, &model.Requests, &model.Tokens); err != nil {
			continue
		}
		stats.TopModels = append(stats.TopModels, model)
	}

	// Get top users (join with users table to get usernames)
	userQuery := `
SELECT u.username, COUNT(a.id) as requests, SUM(a.total_tokens) as tokens
FROM api_usage a
JOIN users u ON a.user_id = u.id
WHERE a.created_at >= ? AND a.created_at <= ?
GROUP BY u.username
ORDER BY requests DESC
LIMIT 10
`

	rows2, err := s.db.QueryContext(ctx, userQuery, start, end)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var user struct {
				Username string
				Requests int64
				Tokens   int64
			}
			if err := rows2.Scan(&user.Username, &user.Requests, &user.Tokens); err != nil {
				continue
			}
			stats.TopUsers = append(stats.TopUsers, user)
		}
	}

	return stats, nil
}

// GetPerformanceStats возвращает статистику производительности за период для отчетов
func (s *SQLiteDB) GetPerformanceStats(ctx context.Context, start, end time.Time) (*models.PerformanceReportStats, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	stats := &models.PerformanceReportStats{}

	// Get aggregate statistics
	query := `
SELECT
COUNT(*) as total_requests,
AVG(duration_ms) as avg_latency,
MIN(duration_ms) as fastest,
MAX(duration_ms) as slowest,
SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END) as error_count
FROM api_usage
WHERE created_at >= ? AND created_at <= ?
`

	err := s.db.QueryRowContext(ctx, query, start, end).Scan(
		&stats.TotalRequests,
		&stats.AvgLatencyMS,
		&stats.FastestRequest,
		&stats.SlowestRequest,
		&stats.ErrorCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get performance stats: %w", err)
	}

	// Calculate percentiles (simplified - use sorting)
	percQuery := `
SELECT duration_ms
FROM api_usage
WHERE created_at >= ? AND created_at <= ?
ORDER BY duration_ms
`

	rows, err := s.db.QueryContext(ctx, percQuery, start, end)
	if err == nil {
		defer rows.Close()
		durations := []float64{}
		for rows.Next() {
			var d float64
			if err := rows.Scan(&d); err == nil {
				durations = append(durations, d)
			}
		}

		if len(durations) > 0 {
			stats.P50LatencyMS = percentile(durations, 0.50)
			stats.P95LatencyMS = percentile(durations, 0.95)
			stats.P99LatencyMS = percentile(durations, 0.99)
		}
	}

	// Count slow requests (>5 seconds)
	slowQuery := `SELECT COUNT(*) FROM api_usage WHERE created_at >= ? AND created_at <= ? AND duration_ms > 5000`
	s.db.QueryRowContext(ctx, slowQuery, start, end).Scan(&stats.SlowRequestsCount)

	// Calculate requests per second
	duration := end.Sub(start).Seconds()
	if duration > 0 {
		stats.RequestsPerSecond = float64(stats.TotalRequests) / duration
	}

	return stats, nil
}

// CountActiveUsers возвращает количество активных пользователей за период
func (s *SQLiteDB) CountActiveUsers(ctx context.Context, period time.Duration) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("database not connected")
	}

	since := time.Now().Add(-period)
	var count int

	query := `
SELECT COUNT(DISTINCT user_id)
FROM api_usage
WHERE created_at >= ?
`

	err := s.db.QueryRowContext(ctx, query, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active users: %w", err)
	}

	return count, nil
}

// CountTotalUsers возвращает общее количество пользователей
func (s *SQLiteDB) CountTotalUsers(ctx context.Context) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("database not connected")
	}

	var count int
	query := `SELECT COUNT(*) FROM users WHERE status != 'deleted'`

	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count total users: %w", err)
	}

	return count, nil
}

// CountActiveAPIKeys возвращает количество активных API ключей
func (s *SQLiteDB) CountActiveAPIKeys(ctx context.Context) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("database not connected")
	}

	var count int
	query := `SELECT COUNT(*) FROM api_keys WHERE status = 'active' AND (expires_at IS NULL OR expires_at > datetime('now'))`

	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active API keys: %w", err)
	}

	return count, nil
}

// percentile calculates percentile from sorted slice
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := max(int(float64(len(sorted)-1)*p), 0)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

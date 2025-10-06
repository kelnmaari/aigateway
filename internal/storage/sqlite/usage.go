// Package sqlite provides SQLite implementation of API usage tracking operations
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"ollama-openai-proxy/internal/models"
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
	}
	if minDuration.Valid {
		stats.MinDurationMS = int64(minDuration.Float64)
	}
	if maxDuration.Valid {
		stats.MaxDurationMS = int64(maxDuration.Float64)
	}

	// Calculate success rate
	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(stats.SuccessfulRequests) / float64(stats.TotalRequests) * 100
	}

	// Query for model usage breakdown
	modelQuery := `
		SELECT 
			model,
			COUNT(*) as request_count,
			SUM(total_tokens) as total_tokens
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
	for rows.Next() {
		usage := &models.ModelUsageStats{}

		if err := rows.Scan(&usage.Model, &usage.RequestCount, &usage.TotalTokens); err != nil {
			return nil, fmt.Errorf("failed to scan model usage: %w", err)
		}

		// Calculate success rate for model
		var successCount int64
		successQuery := `SELECT COUNT(*) FROM api_usage WHERE user_id = ? AND created_at >= ? AND model = ? AND success = 1`
		if err := s.db.QueryRowContext(ctx, successQuery, userID, startTime, usage.Model).Scan(&successCount); err == nil {
			if usage.RequestCount > 0 {
				usage.SuccessRate = float64(successCount) / float64(usage.RequestCount) * 100
			}
		}

		stats.ModelUsage[usage.Model] = usage
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating model usage: %w", err)
	}

	// Query for endpoint usage breakdown
	endpointQuery := `
		SELECT 
			endpoint,
			COUNT(*) as request_count,
			SUM(total_tokens) as total_tokens
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

		if err := rows.Scan(&usage.Endpoint, &usage.RequestCount, &usage.TotalTokens); err != nil {
			return nil, fmt.Errorf("failed to scan endpoint usage: %w", err)
		}

		// Calculate success rate for endpoint
		var successCount int64
		successQuery := `SELECT COUNT(*) FROM api_usage WHERE user_id = ? AND created_at >= ? AND endpoint = ? AND success = 1`
		if err := s.db.QueryRowContext(ctx, successQuery, userID, startTime, usage.Endpoint).Scan(&successCount); err == nil {
			if usage.RequestCount > 0 {
				usage.SuccessRate = float64(successCount) / float64(usage.RequestCount) * 100
			}
		}

		stats.EndpointUsage[usage.Endpoint] = usage
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating endpoint usage: %w", err)
	}

	// Set period information
	stats.StartDate = startTime
	stats.EndDate = time.Now()

	s.logger.WithFields(map[string]interface{}{
		"user_id":        userID,
		"total_requests": stats.TotalRequests,
		"total_tokens":   stats.TotalTokens,
	}).Debug("Retrieved user usage stats")

	return &stats, nil
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
	}
	if minDuration.Valid {
		stats.MinDurationMS = int64(minDuration.Float64)
	}
	if maxDuration.Valid {
		stats.MaxDurationMS = int64(maxDuration.Float64)
	}

	// Calculate success rate
	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(stats.SuccessfulRequests) / float64(stats.TotalRequests) * 100
	}

	// Query for model usage breakdown
	modelQuery := `
		SELECT 
			model,
			COUNT(*) as request_count,
			SUM(total_tokens) as total_tokens
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
	for rows.Next() {
		usage := &models.ModelUsageStats{}

		if err := rows.Scan(&usage.Model, &usage.RequestCount, &usage.TotalTokens); err != nil {
			return nil, fmt.Errorf("failed to scan model usage: %w", err)
		}

		// Calculate success rate for model
		var successCount int64
		successQuery := `SELECT COUNT(*) FROM api_usage WHERE tenant_id = ? AND created_at >= ? AND model = ? AND success = 1`
		if err := s.db.QueryRowContext(ctx, successQuery, tenantID, startTime, usage.Model).Scan(&successCount); err == nil {
			if usage.RequestCount > 0 {
				usage.SuccessRate = float64(successCount) / float64(usage.RequestCount) * 100
			}
		}

		stats.ModelUsage[usage.Model] = usage
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating model usage: %w", err)
	}

	// Query for endpoint usage breakdown
	endpointQuery := `
		SELECT 
			endpoint,
			COUNT(*) as request_count,
			SUM(total_tokens) as total_tokens
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

		if err := rows.Scan(&usage.Endpoint, &usage.RequestCount, &usage.TotalTokens); err != nil {
			return nil, fmt.Errorf("failed to scan endpoint usage: %w", err)
		}

		// Calculate success rate for endpoint
		var successCount int64
		successQuery := `SELECT COUNT(*) FROM api_usage WHERE tenant_id = ? AND created_at >= ? AND endpoint = ? AND success = 1`
		if err := s.db.QueryRowContext(ctx, successQuery, tenantID, startTime, usage.Endpoint).Scan(&successCount); err == nil {
			if usage.RequestCount > 0 {
				usage.SuccessRate = float64(successCount) / float64(usage.RequestCount) * 100
			}
		}

		stats.EndpointUsage[usage.Endpoint] = usage
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating endpoint usage: %w", err)
	}

	// Set period information
	stats.StartDate = startTime
	stats.EndDate = time.Now()

	s.logger.WithFields(map[string]interface{}{
		"tenant_id":      tenantID,
		"total_requests": stats.TotalRequests,
		"total_tokens":   stats.TotalTokens,
	}).Debug("Retrieved tenant usage stats")

	return &stats, nil
}

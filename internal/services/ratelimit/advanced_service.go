// Package ratelimit provides advanced rate limiting services
// Version 1.12.2+: Advanced Rate Limiting (RATE-02)
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"

	"github.com/sirupsen/logrus"
)

// AdvancedRateLimiter проверяет rate limits с multi-scope support
// Priority: API Key → User → Tenant → Model → Global
type AdvancedRateLimiter struct {
	db            storage.Database
	slidingWindow *SlidingWindowLimiter
	logger        *logrus.Logger

	// Default limits если не настроено в БД
	defaultLimits *DefaultLimits
}

// DefaultLimits дефолтные лимиты если не настроено
type DefaultLimits struct {
	RequestsPerSecond int
	RequestsPerMinute int
	RequestsPerHour   int
	RequestsPerDay    int
}

// NewAdvancedRateLimiter создает новый advanced rate limiter
func NewAdvancedRateLimiter(
	db storage.Database,
	logger *logrus.Logger,
	defaults *DefaultLimits,
) *AdvancedRateLimiter {
	if defaults == nil {
		defaults = &DefaultLimits{
			RequestsPerSecond: 10,
			RequestsPerMinute: 100,
			RequestsPerHour:   1000,
			RequestsPerDay:    10000,
		}
	}

	return &AdvancedRateLimiter{
		db:            db,
		slidingWindow: NewSlidingWindowLimiter(db, logger),
		logger:        logger,
		defaultLimits: defaults,
	}
}

// CheckRateLimit проверяет rate limits по всем scopes с priority
func (r *AdvancedRateLimiter) CheckRateLimit(
	ctx context.Context,
	userID string,
	tenantID *string,
	apiKeyID *string,
	modelName string,
) (*models.RateLimitResult, error) {
	// Priority checking: API Key → User → Tenant → Model → Global

	// 1. Check API key rate limit
	if apiKeyID != nil {
		result, err := r.checkScope(ctx, models.RateLimitScopeAPIKey, *apiKeyID, nil)
		if err != nil {
			return nil, fmt.Errorf("api_key rate limit check failed: %w", err)
		}
		if !result.Allowed {
			result.Scope = string(models.RateLimitScopeAPIKey)
			return result, nil
		}
	}

	// 2. Check user rate limit
	result, err := r.checkScope(ctx, models.RateLimitScopeUser, userID, nil)
	if err != nil {
		return nil, fmt.Errorf("user rate limit check failed: %w", err)
	}
	if !result.Allowed {
		result.Scope = string(models.RateLimitScopeUser)
		return result, nil
	}

	// 3. Check tenant rate limit
	if tenantID != nil {
		result, err := r.checkScope(ctx, models.RateLimitScopeTenant, *tenantID, nil)
		if err != nil {
			return nil, fmt.Errorf("tenant rate limit check failed: %w", err)
		}
		if !result.Allowed {
			result.Scope = string(models.RateLimitScopeTenant)
			return result, nil
		}
	}

	// 4. Check model-specific rate limit
	if modelName != "" {
		result, err := r.checkScope(ctx, models.RateLimitScopeModel, modelName, &modelName)
		if err != nil {
			return nil, fmt.Errorf("model rate limit check failed: %w", err)
		}
		if !result.Allowed {
			result.Scope = string(models.RateLimitScopeModel)
			return result, nil
		}
	}

	// 5. Check global rate limit
	result, err = r.checkScope(ctx, models.RateLimitScopeGlobal, "all", nil)
	if err != nil {
		return nil, fmt.Errorf("global rate limit check failed: %w", err)
	}
	if !result.Allowed {
		result.Scope = string(models.RateLimitScopeGlobal)
		return result, nil
	}

	// All checks passed
	return &models.RateLimitResult{
		Allowed:   true,
		Remaining: result.Remaining,
		ResetAt:   result.ResetAt,
		Limit:     result.Limit,
		Window:    result.Window,
		Scope:     "none",
	}, nil
}

// checkScope проверяет rate limit для конкретного scope
func (r *AdvancedRateLimiter) checkScope(
	ctx context.Context,
	scope models.RateLimitScope,
	targetID string,
	modelName *string,
) (*models.RateLimitResult, error) {
	// Get rate limit config from DB
	config, err := r.getRateLimitConfig(ctx, scope, targetID, modelName)
	if err != nil {
		// No config = no limit, allow
		r.logger.WithFields(logrus.Fields{
			"scope":     scope,
			"target_id": targetID,
		}).Debug("No rate limit config found, allowing")

		return &models.RateLimitResult{
			Allowed:   true,
			Remaining: 9999, // arbitrary high number
			ResetAt:   time.Now().Add(time.Hour),
			Limit:     9999,
			Window:    "none",
		}, nil
	}

	// Check each time window (second, minute, hour, day)
	// Return first that fails

	// Check per-second limit
	if config.RequestsPerSecond != nil {
		allowed, remaining, resetAt, err := r.slidingWindow.CheckLimit(
			ctx,
			config.ID,
			targetID,
			"second",
			*config.RequestsPerSecond,
			time.Second,
		)
		if err != nil || !allowed {
			return &models.RateLimitResult{
				Allowed:   allowed,
				Remaining: remaining,
				ResetAt:   resetAt,
				Limit:     *config.RequestsPerSecond,
				Window:    "second",
			}, err
		}
	}

	// Check per-minute limit
	if config.RequestsPerMinute != nil {
		allowed, remaining, resetAt, err := r.slidingWindow.CheckLimit(
			ctx,
			config.ID,
			targetID,
			"minute",
			*config.RequestsPerMinute,
			time.Minute,
		)
		if err != nil || !allowed {
			return &models.RateLimitResult{
				Allowed:   allowed,
				Remaining: remaining,
				ResetAt:   resetAt,
				Limit:     *config.RequestsPerMinute,
				Window:    "minute",
			}, err
		}
	}

	// Check per-hour limit
	if config.RequestsPerHour != nil {
		allowed, remaining, resetAt, err := r.slidingWindow.CheckLimit(
			ctx,
			config.ID,
			targetID,
			"hour",
			*config.RequestsPerHour,
			time.Hour,
		)
		if err != nil || !allowed {
			return &models.RateLimitResult{
				Allowed:   allowed,
				Remaining: remaining,
				ResetAt:   resetAt,
				Limit:     *config.RequestsPerHour,
				Window:    "hour",
			}, err
		}
	}

	// Check per-day limit
	if config.RequestsPerDay != nil {
		allowed, remaining, resetAt, err := r.slidingWindow.CheckLimit(
			ctx,
			config.ID,
			targetID,
			"day",
			*config.RequestsPerDay,
			24*time.Hour,
		)
		if err != nil || !allowed {
			return &models.RateLimitResult{
				Allowed:   allowed,
				Remaining: remaining,
				ResetAt:   resetAt,
				Limit:     *config.RequestsPerDay,
				Window:    "day",
			}, err
		}
	}

	// All time windows passed
	return &models.RateLimitResult{
		Allowed:   true,
		Remaining: 9999, // TODO: calculate minimum remaining across all windows
		ResetAt:   time.Now().Add(time.Minute),
		Limit:     9999,
		Window:    "none",
	}, nil
}

// getRateLimitConfig получает конфигурацию rate limit из БД
func (r *AdvancedRateLimiter) getRateLimitConfig(
	ctx context.Context,
	scope models.RateLimitScope,
	targetID string,
	modelName *string,
) (*models.RateLimitConfig, error) {
	// TODO: Implement actual DB query
	// For MVP, return nil (no config = no limit)
	return nil, fmt.Errorf("not found")
}

// StartCleanupLoop запускает cleanup для sliding window cache
func (r *AdvancedRateLimiter) StartCleanupLoop(ctx context.Context) {
	go r.slidingWindow.StartCleanupLoop(ctx, 5*time.Minute, 24*time.Hour)
}

// GetStats возвращает статистику rate limiting
func (r *AdvancedRateLimiter) GetStats() map[string]interface{} {
	return r.slidingWindow.GetCacheStats()
}

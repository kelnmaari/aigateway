// Package ratelimit provides advanced rate limiting services
// Version 1.12.2+: Advanced Rate Limiting (RATE-02)
package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"aigateway/internal/models"
	"aigateway/internal/storage"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// RedisRateLimitService interface for Redis rate limiting (v3.0.6+)
type RedisRateLimitService interface {
	CheckLimit(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, remaining int, err error)
	Reset(ctx context.Context, key string) error
}

// SlidingWindowLimiter реализует sliding window rate limiting
// Более точный чем fixed window, предотвращает burst attacks на границе окна
type SlidingWindowLimiter struct {
	db     storage.Database
	logger *logrus.Logger
	mu     sync.RWMutex

	// In-memory cache для быстрого доступа (key = "rateLimit:windowType:targetID")
	cache map[string]*windowState

	// Redis для distributed rate limiting (v3.0.6+)
	redisService RedisRateLimitService
}

// windowState состояние sliding window
type windowState struct {
	timestamps []time.Time
	mu         sync.RWMutex
}

// NewSlidingWindowLimiter создает новый sliding window limiter
func NewSlidingWindowLimiter(db storage.Database, logger *logrus.Logger) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		db:     db,
		logger: logger,
		cache:  make(map[string]*windowState),
	}
}

// SetRedisService sets Redis service for distributed rate limiting (v3.0.6+)
func (l *SlidingWindowLimiter) SetRedisService(redisService RedisRateLimitService) {
	l.redisService = redisService
	if redisService != nil {
		l.logger.Info("Redis rate limiting enabled (distributed mode)")
	}
}

// CheckLimit проверяет rate limit с sliding window algorithm
func (l *SlidingWindowLimiter) CheckLimit(
	ctx context.Context,
	rateLimitID string,
	targetID string,
	windowType string,
	limit int,
	window time.Duration,
) (allowed bool, remaining int, resetAt time.Time, err error) {
	now := time.Now()
	windowStart := now.Add(-window)

	// Cache key
	cacheKey := fmt.Sprintf("%s:%s:%s", rateLimitID, windowType, targetID)

	// Use Redis if available (v3.0.6+: distributed rate limiting)
	if l.redisService != nil {
		allowed, remaining, err := l.redisService.CheckLimit(ctx, cacheKey, limit, window)
		if err != nil {
			l.logger.WithError(err).Warn("Redis rate limit check failed, falling back to in-memory")
			// Fall through to in-memory implementation
		} else {
			resetAt = now.Add(window)
			l.logger.WithFields(logrus.Fields{
				"key":       cacheKey,
				"allowed":   allowed,
				"remaining": remaining,
				"redis":     true,
			}).Debug("Rate limit checked (Redis)")
			return allowed, remaining, resetAt, nil
		}
	}

	// Get or create window state
	l.mu.Lock()
	state, exists := l.cache[cacheKey]
	if !exists {
		state = &windowState{
			timestamps: make([]time.Time, 0, limit+10), // небольшой буфер
		}
		l.cache[cacheKey] = state
	}
	l.mu.Unlock()

	state.mu.Lock()
	defer state.mu.Unlock()

	// Remove timestamps outside sliding window
	newTimestamps := make([]time.Time, 0, len(state.timestamps))
	for _, ts := range state.timestamps {
		if ts.After(windowStart) {
			newTimestamps = append(newTimestamps, ts)
		}
	}
	state.timestamps = newTimestamps

	// Check if limit exceeded
	currentCount := len(state.timestamps)
	if currentCount >= limit {
		// Find oldest timestamp to calculate reset time
		if len(state.timestamps) > 0 {
			oldest := state.timestamps[0]
			resetAt = oldest.Add(window)
		} else {
			resetAt = now.Add(window)
		}

		l.logger.WithFields(logrus.Fields{
			"rate_limit_id": rateLimitID,
			"target_id":     targetID,
			"window_type":   windowType,
			"current_count": currentCount,
			"limit":         limit,
		}).Debug("Rate limit exceeded")

		return false, 0, resetAt, nil
	}

	// Add current request
	state.timestamps = append(state.timestamps, now)

	remaining = limit - len(state.timestamps)
	resetAt = windowStart.Add(window)

	l.logger.WithFields(logrus.Fields{
		"rate_limit_id": rateLimitID,
		"target_id":     targetID,
		"window_type":   windowType,
		"current_count": len(state.timestamps),
		"limit":         limit,
		"remaining":     remaining,
	}).Debug("Rate limit check passed")

	return true, remaining, resetAt, nil
}

// RecordUsage записывает использование rate limit в БД для persistence
// Вызывается асинхронно для минимизации latency
func (l *SlidingWindowLimiter) RecordUsage(
	ctx context.Context,
	rateLimitID string,
	windowType string,
	windowStart time.Time,
) error {
	usage := &models.RateLimitUsage{
		ID:            uuid.New().String(),
		RateLimitID:   rateLimitID,
		WindowType:    windowType,
		WindowStart:   windowStart,
		RequestCount:  1,
		LastRequestAt: time.Now(),
	}

	// TODO: Implement storage layer for rate_limit_usage
	// For MVP, we use in-memory cache only
	_ = usage

	return nil
}

// Cleanup удаляет старые записи из cache (вызывать периодически)
func (l *SlidingWindowLimiter) Cleanup(maxAge time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-maxAge)

	for key, state := range l.cache {
		state.mu.Lock()

		// Remove old timestamps
		newTimestamps := make([]time.Time, 0, len(state.timestamps))
		for _, ts := range state.timestamps {
			if ts.After(cutoff) {
				newTimestamps = append(newTimestamps, ts)
			}
		}
		state.timestamps = newTimestamps

		// Remove empty states
		if len(state.timestamps) == 0 {
			delete(l.cache, key)
		}

		state.mu.Unlock()
	}

	l.logger.WithField("cache_size", len(l.cache)).Debug("Rate limit cache cleaned up")
}

// StartCleanupLoop запускает periodic cleanup
func (l *SlidingWindowLimiter) StartCleanupLoop(ctx context.Context, interval time.Duration, maxAge time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	l.logger.WithFields(logrus.Fields{
		"interval": interval,
		"max_age":  maxAge,
	}).Info("Rate limit cleanup loop started")

	for {
		select {
		case <-ctx.Done():
			l.logger.Info("Rate limit cleanup loop stopped")
			return
		case <-ticker.C:
			l.Cleanup(maxAge)
		}
	}
}

// GetCacheStats возвращает статистику cache (для мониторинга)
func (l *SlidingWindowLimiter) GetCacheStats() map[string]any {
	l.mu.RLock()
	defer l.mu.RUnlock()

	totalTimestamps := 0
	for _, state := range l.cache {
		state.mu.RLock()
		totalTimestamps += len(state.timestamps)
		state.mu.RUnlock()
	}

	return map[string]any{
		"cache_entries":    len(l.cache),
		"total_timestamps": totalTimestamps,
		"avg_per_entry":    float64(totalTimestamps) / float64(max(len(l.cache), 1)),
	}
}

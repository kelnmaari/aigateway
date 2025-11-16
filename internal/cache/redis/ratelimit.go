// Package redis provides Redis-based rate limiting
// Version: v3.0.6+ - Distributed Rate Limiting
package redis

import (
	"context"
	"fmt"
	"time"

	redisLib "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// RateLimitService provides Redis-based distributed rate limiting
type RateLimitService struct {
	client *Client
	logger *logrus.Logger
}

// NewRateLimitService creates a new rate limit service
func NewRateLimitService(client *Client, logger *logrus.Logger) *RateLimitService {
	return &RateLimitService{
		client: client,
		logger: logger,
	}
}

// CheckLimit checks if request is within rate limit using sliding window
// Returns: allowed, remaining, resetAt, error
func (s *RateLimitService) CheckLimit(
	ctx context.Context,
	keyID string,
	limit int64,
	window time.Duration,
) (bool, int64, time.Time, error) {
	now := time.Now()
	windowStart := now.Add(-window)
	
	// Use sorted set with timestamps as scores
	key := fmt.Sprintf("ratelimit:%s", keyID)
	
	// Remove old entries outside the window
	minScore := fmt.Sprintf("%d", windowStart.UnixNano())
	maxScore := "+inf"
	if err := s.client.ZRemRangeByScore(ctx, key, "-inf", minScore); err != nil {
		return false, 0, time.Time{}, fmt.Errorf("failed to clean old entries: %w", err)
	}
	
	// Count current requests in window
	count, err := s.client.ZCard(ctx, key)
	if err != nil {
		return false, 0, time.Time{}, fmt.Errorf("failed to count requests: %w", err)
	}
	
	// Check if limit exceeded
	if count >= limit {
		// Get oldest timestamp to calculate reset time
		oldest, err := s.client.ZRangeByScore(ctx, key, "-inf", maxScore)
		if err != nil || len(oldest) == 0 {
			return false, 0, now.Add(window), nil
		}
		
		// Parse oldest timestamp (nanoseconds)
		var oldestTime int64
		fmt.Sscanf(oldest[0], "%d", &oldestTime)
		resetAt := time.Unix(0, oldestTime).Add(window)
		
		s.logger.WithFields(logrus.Fields{
			"key_id": keyID,
			"count":  count,
			"limit":  limit,
			"reset_at": resetAt,
		}).Debug("Rate limit exceeded")
		
		return false, 0, resetAt, nil
	}
	
	// Add current request timestamp
	score := float64(now.UnixNano())
	member := fmt.Sprintf("%d:%s", now.UnixNano(), keyID)
	
	redisZ := redisLib.Z{Score: score, Member: member}
	if err := s.client.ZAdd(ctx, key, redisZ); err != nil {
		return false, 0, time.Time{}, fmt.Errorf("failed to add request: %w", err)
	}
	
	// Set expiration on key (cleanup after window expires)
	if err := s.client.Expire(ctx, key, window*2); err != nil {
		s.logger.WithError(err).Warn("Failed to set expiration on rate limit key")
	}
	
	remaining := limit - (count + 1)
	resetAt := now.Add(window)
	
	s.logger.WithFields(logrus.Fields{
		"key_id":    keyID,
		"count":     count + 1,
		"limit":     limit,
		"remaining": remaining,
	}).Debug("Rate limit check passed")
	
	return true, remaining, resetAt, nil
}

// Reset resets rate limit for a key
func (s *RateLimitService) Reset(ctx context.Context, keyID string) error {
	key := fmt.Sprintf("ratelimit:%s", keyID)
	return s.client.Delete(ctx, key)
}

// GetCurrentCount gets current request count in window
func (s *RateLimitService) GetCurrentCount(ctx context.Context, keyID string, window time.Duration) (int64, error) {
	key := fmt.Sprintf("ratelimit:%s", keyID)
	windowStart := time.Now().Add(-window)
	minScore := fmt.Sprintf("%d", windowStart.UnixNano())
	
	// Clean old entries
	if err := s.client.ZRemRangeByScore(ctx, key, "-inf", minScore); err != nil {
		return 0, fmt.Errorf("failed to clean old entries: %w", err)
	}
	
	return s.client.ZCard(ctx, key)
}

// IncrementCounter increments a simple counter with expiration (for token usage, etc.)
func (s *RateLimitService) IncrementCounter(
	ctx context.Context,
	keyID string,
	increment int64,
	expiration time.Duration,
) (int64, error) {
	key := fmt.Sprintf("counter:%s", keyID)
	
	// Increment counter
	count, err := s.client.IncrBy(ctx, key, increment)
	if err != nil {
		return 0, fmt.Errorf("failed to increment counter: %w", err)
	}
	
	// Set expiration if this is first increment
	if count == increment {
		if err := s.client.Expire(ctx, key, expiration); err != nil {
			s.logger.WithError(err).Warn("Failed to set expiration on counter")
		}
	}
	
	return count, nil
}

// GetCounter gets counter value
func (s *RateLimitService) GetCounter(ctx context.Context, keyID string) (int64, error) {
	key := fmt.Sprintf("counter:%s", keyID)
	return s.client.GetCounter(ctx, key)
}

// ResetCounter resets a counter
func (s *RateLimitService) ResetCounter(ctx context.Context, keyID string) error {
	key := fmt.Sprintf("counter:%s", keyID)
	return s.client.Delete(ctx, key)
}


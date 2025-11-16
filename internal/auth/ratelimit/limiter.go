// Package ratelimit provides rate limiting functionality per API key
package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"

	"aigateway/internal/config"
	"aigateway/internal/models"
)

// RedisRateLimitService interface for Redis rate limiting (v3.0.6+)
type RedisRateLimitService interface {
	CheckLimit(ctx context.Context, key string, limit int64, window time.Duration) (allowed bool, remaining int64, resetAt time.Time, err error)
	Reset(ctx context.Context, key string) error
}

// Limiter управляет rate limiting для API ключей
type Limiter struct {
	config  *config.Config
	logger  *logrus.Logger
	enabled bool

	// Rate limiters per API key
	mutex    sync.RWMutex
	limiters map[string]*KeyLimiter

	// Cleanup ticker
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
	
	// Redis для distributed rate limiting (v3.0.6+)
	redisService RedisRateLimitService
}

// KeyLimiter содержит rate limiters для одного API ключа
type KeyLimiter struct {
	KeyID           string
	KeyName         string
	RequestsPerMin  *rate.Limiter
	RequestsPerHour *TokenBucketCounter
	RequestsPerDay  *TokenBucketCounter
	TokensPerMin    *TokenBucketCounter
	TokensPerDay    *TokenBucketCounter
	LastUsed        time.Time

	// Статистика
	Stats LimiterStats
}

// LimiterStats статистика rate limiter'а
type LimiterStats struct {
	TotalRequests   int64     `json:"total_requests"`
	AllowedRequests int64     `json:"allowed_requests"`
	BlockedRequests int64     `json:"blocked_requests"`
	TotalTokens     int64     `json:"total_tokens"`
	LastRequest     time.Time `json:"last_request"`
	LastBlock       time.Time `json:"last_block"`
	BlockReason     string    `json:"last_block_reason"`
}

// TokenBucketCounter простая реализация token bucket для подсчета за период
type TokenBucketCounter struct {
	Limit       int64
	Window      time.Duration
	mutex       sync.Mutex
	tokens      int64
	windowStart time.Time
}

// RateLimitResult результат проверки rate limit
type RateLimitResult struct {
	Allowed    bool          `json:"allowed"`
	Reason     string        `json:"reason,omitempty"`
	RetryAfter time.Duration `json:"retry_after,omitempty"`
	Remaining  int64         `json:"remaining,omitempty"`
	ResetTime  time.Time     `json:"reset_time,omitempty"`
}

// SetRedisService sets Redis service for distributed rate limiting (v3.0.6+)
func (l *Limiter) SetRedisService(redisService RedisRateLimitService) {
	l.redisService = redisService
	if redisService != nil {
		l.logger.Info("✅ Redis rate limiting enabled (distributed mode)")
	}
}

// NewLimiter создает новый rate limiter
func NewLimiter(cfg *config.Config, logger *logrus.Logger) *Limiter {
	limiter := &Limiter{
		config:      cfg,
		logger:      logger,
		enabled:     cfg.Auth.RateLimiting.Enabled,
		limiters:    make(map[string]*KeyLimiter),
		stopCleanup: make(chan struct{}),
	}

	// Запускаем периодическую очистку неиспользуемых limiters
	limiter.startCleanupRoutine()

	return limiter
}

// CheckRateLimit проверяет rate limit для API ключа
func (l *Limiter) CheckRateLimit(ctx context.Context, keyID string, keyInfo *models.APIKeyPublic, tokens int64) *RateLimitResult {
	if !l.enabled {
		return &RateLimitResult{Allowed: true}
	}
	
	// Use Redis if available (v3.0.6+: distributed rate limiting)
	if l.redisService != nil && keyInfo != nil {
		// Check per-minute limit in Redis
		redisKey := fmt.Sprintf("apikey:%s:requests:minute", keyID)
		
		// Get limit from config (default or keyInfo override)
		limit := int64(l.config.Auth.RateLimiting.DefaultRequestsPerMinute)
		window := time.Minute
		
		if limit > 0 {
			allowed, remaining, resetAt, err := l.redisService.CheckLimit(ctx, redisKey, limit, window)
			if err != nil {
				l.logger.WithError(err).Warn("Redis rate limit check failed, falling back to in-memory")
				// Fall through to in-memory implementation
			} else {
				l.logger.WithFields(logrus.Fields{
					"key_id":    keyID,
					"allowed":   allowed,
					"remaining": remaining,
					"redis":     true,
					"limit":     limit,
					"window":    "1m",
				}).Debug("Rate limit checked (Redis)")
				
				if !allowed {
					return &RateLimitResult{
						Allowed:    false,
						Reason:     fmt.Sprintf("Rate limit exceeded: %d requests per minute", limit),
						RetryAfter: time.Until(resetAt),
						Remaining:  remaining,
						ResetTime:  resetAt,
					}
				}
				
				return &RateLimitResult{
					Allowed:   true,
					Remaining: remaining,
				}
			}
		}
	}

	// Fallback to in-memory rate limiting
	// Получаем или создаем limiter для ключа
	keyLimiter := l.getOrCreateKeyLimiter(keyID, keyInfo)

	// Проверяем все лимиты
	if result := l.checkRequestLimits(keyLimiter); !result.Allowed {
		return result
	}

	if result := l.checkTokenLimits(keyLimiter, tokens); !result.Allowed {
		return result
	}

	// Все проверки пройдены - разрешаем запрос
	keyLimiter.consume(1, tokens)

	return &RateLimitResult{
		Allowed:   true,
		Remaining: keyLimiter.getRemainingRequests(),
	}
}

// getOrCreateKeyLimiter получает или создает limiter для API ключа
func (l *Limiter) getOrCreateKeyLimiter(keyID string, keyInfo *models.APIKeyPublic) *KeyLimiter {
	l.mutex.RLock()
	keyLimiter, exists := l.limiters[keyID]
	l.mutex.RUnlock()

	if exists {
		keyLimiter.LastUsed = time.Now()
		return keyLimiter
	}

	// Создаем новый limiter
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Double-check после получения write lock
	if keyLimiter, exists := l.limiters[keyID]; exists {
		keyLimiter.LastUsed = time.Now()
		return keyLimiter
	}

	rateLimits := keyInfo.RateLimits

	keyLimiter = &KeyLimiter{
		KeyID:    keyID,
		KeyName:  keyInfo.Name,
		LastUsed: time.Now(),
		Stats:    LimiterStats{},
	}

	// Создаем rate limiters для различных периодов
	if rateLimits.RequestsPerMinute > 0 {
		keyLimiter.RequestsPerMin = rate.NewLimiter(
			rate.Limit(rateLimits.RequestsPerMinute)/60, // requests per second
			rateLimits.RequestsPerMinute,                // burst capacity
		)
	}

	if rateLimits.RequestsPerHour > 0 {
		keyLimiter.RequestsPerHour = NewTokenBucketCounter(
			int64(rateLimits.RequestsPerHour),
			time.Hour,
		)
	}

	if rateLimits.RequestsPerDay > 0 {
		keyLimiter.RequestsPerDay = NewTokenBucketCounter(
			int64(rateLimits.RequestsPerDay),
			24*time.Hour,
		)
	}

	if rateLimits.TokensPerMinute > 0 {
		keyLimiter.TokensPerMin = NewTokenBucketCounter(
			int64(rateLimits.TokensPerMinute),
			time.Minute,
		)
	}

	if rateLimits.TokensPerDay > 0 {
		keyLimiter.TokensPerDay = NewTokenBucketCounter(
			int64(rateLimits.TokensPerDay),
			24*time.Hour,
		)
	}

	l.limiters[keyID] = keyLimiter

	l.logger.WithFields(logrus.Fields{
		"key_id":   keyID,
		"key_name": keyInfo.Name,
		"limits":   rateLimits,
	}).Debug("Created new rate limiter for API key")

	return keyLimiter
}

// checkRequestLimits проверяет лимиты запросов
func (l *Limiter) checkRequestLimits(keyLimiter *KeyLimiter) *RateLimitResult {
	// Проверка requests per minute
	if keyLimiter.RequestsPerMin != nil {
		if !keyLimiter.RequestsPerMin.Allow() {
			keyLimiter.recordBlock("requests_per_minute_exceeded")
			return &RateLimitResult{
				Allowed:    false,
				Reason:     "Too many requests per minute",
				RetryAfter: time.Minute,
			}
		}
	}

	// Проверка requests per hour
	if keyLimiter.RequestsPerHour != nil {
		if !keyLimiter.RequestsPerHour.Allow(1) {
			keyLimiter.recordBlock("requests_per_hour_exceeded")
			return &RateLimitResult{
				Allowed:    false,
				Reason:     "Too many requests per hour",
				RetryAfter: keyLimiter.RequestsPerHour.timeUntilReset(),
			}
		}
	}

	// Проверка requests per day
	if keyLimiter.RequestsPerDay != nil {
		if !keyLimiter.RequestsPerDay.Allow(1) {
			keyLimiter.recordBlock("requests_per_day_exceeded")
			return &RateLimitResult{
				Allowed:    false,
				Reason:     "Daily request limit exceeded",
				RetryAfter: keyLimiter.RequestsPerDay.timeUntilReset(),
			}
		}
	}

	return &RateLimitResult{Allowed: true}
}

// checkTokenLimits проверяет лимиты токенов
func (l *Limiter) checkTokenLimits(keyLimiter *KeyLimiter, tokens int64) *RateLimitResult {
	if tokens <= 0 {
		return &RateLimitResult{Allowed: true}
	}

	// Проверка tokens per minute
	if keyLimiter.TokensPerMin != nil {
		if !keyLimiter.TokensPerMin.Allow(tokens) {
			keyLimiter.recordBlock("tokens_per_minute_exceeded")
			return &RateLimitResult{
				Allowed:    false,
				Reason:     "Too many tokens per minute",
				RetryAfter: keyLimiter.TokensPerMin.timeUntilReset(),
			}
		}
	}

	// Проверка tokens per day
	if keyLimiter.TokensPerDay != nil {
		if !keyLimiter.TokensPerDay.Allow(tokens) {
			keyLimiter.recordBlock("tokens_per_day_exceeded")
			return &RateLimitResult{
				Allowed:    false,
				Reason:     "Daily token limit exceeded",
				RetryAfter: keyLimiter.TokensPerDay.timeUntilReset(),
			}
		}
	}

	return &RateLimitResult{Allowed: true}
}

// GetLimiterStats возвращает статистику всех limiters
func (l *Limiter) GetLimiterStats() map[string]LimiterStats {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	stats := make(map[string]LimiterStats)
	for keyID, limiter := range l.limiters {
		stats[keyID] = limiter.Stats
	}

	return stats
}

// ClearLimiter удаляет limiter для API ключа
func (l *Limiter) ClearLimiter(keyID string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	delete(l.limiters, keyID)
	l.logger.WithField("key_id", keyID).Debug("Cleared rate limiter for API key")
}

// Stop останавливает rate limiter и очищает ресурсы
func (l *Limiter) Stop() {
	if l.cleanupTicker != nil {
		l.cleanupTicker.Stop()
	}
	close(l.stopCleanup)
	l.logger.Info("Rate limiter stopped")
}

// Private methods

// startCleanupRoutine запускает фоновую очистку неиспользуемых limiters
func (l *Limiter) startCleanupRoutine() {
	l.cleanupTicker = time.NewTicker(10 * time.Minute)

	go func() {
		for {
			select {
			case <-l.cleanupTicker.C:
				l.cleanupUnusedLimiters()
			case <-l.stopCleanup:
				return
			}
		}
	}()
}

// cleanupUnusedLimiters очищает неиспользуемые limiters
func (l *Limiter) cleanupUnusedLimiters() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	cutoff := time.Now().Add(-30 * time.Minute) // Очищаем limiters старше 30 минут
	var cleaned []string

	for keyID, limiter := range l.limiters {
		if limiter.LastUsed.Before(cutoff) {
			delete(l.limiters, keyID)
			cleaned = append(cleaned, keyID)
		}
	}

	if len(cleaned) > 0 {
		l.logger.WithFields(logrus.Fields{
			"cleaned_count": len(cleaned),
			"cleaned_keys":  cleaned,
		}).Debug("Cleaned up unused rate limiters")
	}
}

// Methods для KeyLimiter

// consume использует токены/запросы
func (kl *KeyLimiter) consume(requests int64, tokens int64) {
	kl.Stats.TotalRequests += requests
	kl.Stats.AllowedRequests += requests
	kl.Stats.TotalTokens += tokens
	kl.Stats.LastRequest = time.Now()
	kl.LastUsed = time.Now()
}

// recordBlock записывает блокировку запроса
func (kl *KeyLimiter) recordBlock(reason string) {
	kl.Stats.BlockedRequests++
	kl.Stats.LastBlock = time.Now()
	kl.Stats.BlockReason = reason
	kl.LastUsed = time.Now()
}

// getRemainingRequests получает количество оставшихся запросов в минуту
func (kl *KeyLimiter) getRemainingRequests() int64 {
	if kl.RequestsPerMin == nil {
		return -1 // Unlimited
	}

	return int64(kl.RequestsPerMin.Tokens())
}

// TokenBucketCounter implementation

// NewTokenBucketCounter создает новый token bucket counter
func NewTokenBucketCounter(limit int64, window time.Duration) *TokenBucketCounter {
	return &TokenBucketCounter{
		Limit:       limit,
		Window:      window,
		tokens:      limit,
		windowStart: time.Now(),
	}
}

// Allow проверяет можно ли использовать указанное количество токенов
func (tbc *TokenBucketCounter) Allow(tokens int64) bool {
	tbc.mutex.Lock()
	defer tbc.mutex.Unlock()

	now := time.Now()

	// Если окно истекло, сбрасываем счетчик
	if now.Sub(tbc.windowStart) >= tbc.Window {
		tbc.tokens = tbc.Limit
		tbc.windowStart = now
	}

	// Проверяем доступность токенов
	if tbc.tokens >= tokens {
		tbc.tokens -= tokens
		return true
	}

	return false
}

// timeUntilReset возвращает время до сброса окна
func (tbc *TokenBucketCounter) timeUntilReset() time.Duration {
	tbc.mutex.Lock()
	defer tbc.mutex.Unlock()

	elapsed := time.Since(tbc.windowStart)
	if elapsed >= tbc.Window {
		return 0
	}

	return tbc.Window - elapsed
}

// GetRemaining возвращает количество оставшихся токенов
func (tbc *TokenBucketCounter) GetRemaining() int64 {
	tbc.mutex.Lock()
	defer tbc.mutex.Unlock()

	now := time.Now()

	// Если окно истекло, возвращаем полный лимит
	if now.Sub(tbc.windowStart) >= tbc.Window {
		return tbc.Limit
	}

	return tbc.tokens
}


package webfetch

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter управляет rate limiting для доменов
type RateLimiter struct {
	config   RateLimitConfig
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

// NewRateLimiter создает новый rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		config:   config,
		limiters: make(map[string]*rate.Limiter),
	}
}

// Wait ждет разрешения для выполнения запроса к домену
func (r *RateLimiter) Wait(ctx context.Context, rawURL string) error {
	if !r.config.Enabled {
		return nil
	}

	domain := extractDomain(rawURL)
	if domain == "" {
		return fmt.Errorf("invalid URL: cannot extract domain")
	}

	limiter := r.getLimiter(domain)

	return limiter.Wait(ctx)
}

// Allow проверяет можно ли выполнить запрос немедленно
func (r *RateLimiter) Allow(rawURL string) bool {
	if !r.config.Enabled {
		return true
	}

	domain := extractDomain(rawURL)
	if domain == "" {
		return false
	}

	limiter := r.getLimiter(domain)

	return limiter.Allow()
}

// getLimiter возвращает или создает limiter для домена
func (r *RateLimiter) getLimiter(domain string) *rate.Limiter {
	r.mu.RLock()
	limiter, exists := r.limiters[domain]
	r.mu.RUnlock()

	if exists {
		return limiter
	}

	// Create new limiter
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := r.limiters[domain]; exists {
		return limiter
	}

	// Determine rate limit for domain
	requestsPerMinute := r.getRequestsPerMinute(domain)

	// Convert to tokens per second
	tokensPerSecond := float64(requestsPerMinute) / 60.0

	// Burst allows some flexibility (10% of per-minute limit)
	burst := max(1, requestsPerMinute/10)

	limiter = rate.NewLimiter(rate.Limit(tokensPerSecond), burst)
	r.limiters[domain] = limiter

	return limiter
}

// getRequestsPerMinute возвращает лимит для домена
func (r *RateLimiter) getRequestsPerMinute(domain string) int {
	// Check per-domain overrides
	if limit, exists := r.config.PerDomainLimits[domain]; exists {
		return limit
	}

	// Check wildcard matches
	for pattern, limit := range r.config.PerDomainLimits {
		if matchDomain(domain, pattern) {
			return limit
		}
	}

	return r.config.DefaultRequestsPerMinute
}

// extractDomain извлекает домен из URL
func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	hostname := u.Hostname()
	if hostname == "" {
		return ""
	}

	return strings.ToLower(hostname)
}

// matchDomain проверяет соответствие домена паттерну
func matchDomain(domain, pattern string) bool {
	domain = strings.ToLower(domain)
	pattern = strings.ToLower(pattern)

	// Exact match
	if domain == pattern {
		return true
	}

	// Wildcard match (*.example.com)
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:]
		return strings.HasSuffix(domain, "."+suffix) || domain == suffix
	}

	return false
}

// max возвращает максимум из двух int
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

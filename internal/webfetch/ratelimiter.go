package webfetch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter controls the rate of web page fetches per domain
type RateLimiter struct {
	config      RateLimitConfig
	limiterMap  map[string]*rate.Limiter
	burstMap    map[string]int
	mutex       sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		config:     config,
		limiterMap: make(map[string]*rate.Limiter),
		burstMap:   make(map[string]int),
	}
}

// Wait blocks until the domain's rate limit allows a request
func (rl *RateLimiter) Wait(ctx context.Context, domain string) error {
	if !rl.config.Enabled {
		return nil
	}

	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Get or create limiter for this domain
	limiter, exists := rl.limiterMap[domain]
	if !exists {
		// Use per-domain limit if set, otherwise use default
		limit := rl.config.DefaultRequestsPerMin
		if perDomainLimit, ok := rl.config.PerDomainLimits[domain]; ok {
			limit = perDomainLimit
		}

		limiter = rate.NewLimiter(rate.Every(time.Minute/time.Duration(limit)), limit)
		rl.limiterMap[domain] = limiter
		rl.burstMap[domain] = limit
	}

	// Check if we need to wait
	if !limiter.Allow() {
		// Calculate wait time
		wait := limiter.Reserve().DelayFrom(time.Now())
		if wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				return fmt.Errorf("context canceled")
			case <-timer.C:
				break
			}
		}
	}

	return nil
}

// Reset resets the rate limiter for a domain (used for testing)
func (rl *RateLimiter) Reset(domain string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	delete(rl.limiterMap, domain)
	delete(rl.burstMap, domain)
}

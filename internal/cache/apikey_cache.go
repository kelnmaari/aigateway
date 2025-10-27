// Package cache provides in-memory caching for performance optimization
package cache

import (
	"sync"
	"time"

	"aigateway/internal/models"
)

// APIKeyCache provides thread-safe in-memory cache for APIKey hot data
//
// Performance optimization: Stores frequently accessed APIKeyHot in memory
// to avoid database queries on every request validation.
//
// Hot data (64 bytes): ID, Status, ModelsAll, PermissionsAll, IsExpired
// Cold data: Loaded from database only when needed (admin UI, etc)
type APIKeyCache struct {
	mu    sync.RWMutex
	cache map[string]*CacheEntry

	// Configuration
	ttl          time.Duration
	maxSize      int
	cleanupTimer *time.Ticker
	stopCleanup  chan struct{}
}

// CacheEntry wraps APIKeyHot with expiration metadata
type CacheEntry struct {
	Hot       *models.APIKeyHot
	ExpiresAt time.Time
}

// Config for APIKeyCache
type Config struct {
	TTL             time.Duration // Time-to-live for cache entries
	MaxSize         int           // Maximum number of entries
	CleanupInterval time.Duration // How often to cleanup expired entries
}

// DefaultConfig returns default cache configuration
func DefaultConfig() Config {
	return Config{
		TTL:             5 * time.Minute, // Hot data valid for 5 minutes
		MaxSize:         10000,           // Up to 10k API keys in cache
		CleanupInterval: 1 * time.Minute, // Cleanup every minute
	}
}

// NewAPIKeyCache creates new cache instance
func NewAPIKeyCache(cfg Config) *APIKeyCache {
	c := &APIKeyCache{
		cache:       make(map[string]*CacheEntry, cfg.MaxSize),
		ttl:         cfg.TTL,
		maxSize:     cfg.MaxSize,
		stopCleanup: make(chan struct{}),
	}

	// Start background cleanup goroutine
	c.cleanupTimer = time.NewTicker(cfg.CleanupInterval)
	go c.cleanupWorker()

	return c
}

// Get retrieves APIKeyHot from cache
func (c *APIKeyCache) Get(keyID string) (*models.APIKeyHot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[keyID]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry.Hot, true
}

// Set stores APIKeyHot in cache
func (c *APIKeyCache) Set(keyID string, hot *models.APIKeyHot) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict random entry if cache is full
	if len(c.cache) >= c.maxSize {
		// Simple eviction: delete first entry
		for k := range c.cache {
			delete(c.cache, k)
			break
		}
	}

	c.cache[keyID] = &CacheEntry{
		Hot:       hot,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Delete removes entry from cache
func (c *APIKeyCache) Delete(keyID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, keyID)
}

// Invalidate removes all entries from cache
func (c *APIKeyCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CacheEntry, c.maxSize)
}

// Size returns current cache size
func (c *APIKeyCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}

// Stats returns cache statistics
func (c *APIKeyCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now()
	active := 0
	expired := 0

	for _, entry := range c.cache {
		if now.After(entry.ExpiresAt) {
			expired++
		} else {
			active++
		}
	}

	return CacheStats{
		Size:    len(c.cache),
		Active:  active,
		Expired: expired,
		MaxSize: c.maxSize,
	}
}

// CacheStats contains cache statistics
type CacheStats struct {
	Size    int `json:"size"`     // Total entries
	Active  int `json:"active"`   // Non-expired entries
	Expired int `json:"expired"`  // Expired entries
	MaxSize int `json:"max_size"` // Maximum capacity
}

// cleanupWorker periodically removes expired entries
func (c *APIKeyCache) cleanupWorker() {
	for {
		select {
		case <-c.cleanupTimer.C:
			c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

// cleanup removes expired entries
func (c *APIKeyCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removed := 0

	for keyID, entry := range c.cache {
		if now.After(entry.ExpiresAt) {
			delete(c.cache, keyID)
			removed++
		}
	}

	// Optional: log cleanup results
	_ = removed // Can be logged if logger is available
}

// Close stops cleanup goroutine
func (c *APIKeyCache) Close() {
	c.cleanupTimer.Stop()
	close(c.stopCleanup)
}

// GetOrLoad retrieves from cache or loads using provided function
//
// This is a convenience method for load-through caching pattern.
func (c *APIKeyCache) GetOrLoad(keyID string, loader func() (*models.APIKeyHot, error)) (*models.APIKeyHot, error) {
	// Try cache first
	if hot, found := c.Get(keyID); found {
		return hot, nil
	}

	// Cache miss - load from source
	hot, err := loader()
	if err != nil {
		return nil, err
	}

	// Store in cache
	c.Set(keyID, hot)

	return hot, nil
}


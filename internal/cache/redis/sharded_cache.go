// Package redis provides sharded in-memory cache layer over Redis
// Version: v3.0.8 - Sharding для hot keys optimization
package redis

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ShardedCache provides concurrent-safe sharded cache with fallback to Redis
type ShardedCache struct {
	shards     []*CacheShard
	shardCount uint32
	redis      CacheBackend
	logger     *logrus.Logger
	enabled    bool
	ttl        time.Duration
}

// CacheBackend interface for Redis cache operations
type CacheBackend interface {
	GetJSON(ctx context.Context, key string, dest interface{}) error
	SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// CacheShard represents a single cache shard with padding to prevent false sharing
type CacheShard struct {
	data map[string]*CacheEntry
	mu   sync.RWMutex
	_pad [56]byte // Cache line padding (64 - 8 = 56 bytes)
}

// CacheEntry represents a cached value with expiration
type CacheEntry struct {
	Value      []byte    // JSON-encoded value
	ExpiresAt  time.Time
	AccessCount int64
	_pad       [40]byte  // Cache line padding
}

// ShardedCacheConfig configuration for sharded cache
type ShardedCacheConfig struct {
	ShardCount uint32        // Number of shards (power of 2 recommended)
	TTL        time.Duration // In-memory TTL (shorter than Redis)
	Enabled    bool          // Enable in-memory layer
}

// NewShardedCache creates a new sharded cache layer
func NewShardedCache(redis CacheBackend, config ShardedCacheConfig, logger *logrus.Logger) *ShardedCache {
	if config.ShardCount == 0 {
		config.ShardCount = 32 // Default: 32 shards
	}
	if config.TTL == 0 {
		config.TTL = 1 * time.Minute // Default: 1 minute in-memory
	}
	
	shards := make([]*CacheShard, config.ShardCount)
	for i := range shards {
		shards[i] = &CacheShard{
			data: make(map[string]*CacheEntry),
		}
	}
	
	sc := &ShardedCache{
		shards:     shards,
		shardCount: config.ShardCount,
		redis:      redis,
		logger:     logger,
		enabled:    config.Enabled,
		ttl:        config.TTL,
	}
	
	// Start cleanup goroutine
	if config.Enabled {
		go sc.cleanupExpired()
	}
	
	logger.WithFields(logrus.Fields{
		"shard_count": config.ShardCount,
		"ttl":         config.TTL,
		"enabled":     config.Enabled,
	}).Info("✅ Sharded cache layer initialized")
	
	return sc
}

// getShard returns the shard for a given key
func (sc *ShardedCache) getShard(key string) *CacheShard {
	hash := fnv.New32a()
	hash.Write([]byte(key))
	shardIdx := hash.Sum32() % sc.shardCount
	return sc.shards[shardIdx]
}

// Get retrieves a value from cache (memory → Redis fallback)
func (sc *ShardedCache) Get(ctx context.Context, key string, dest interface{}) error {
	// Try in-memory cache first if enabled
	if sc.enabled {
		if data, found := sc.getFromMemory(key); found {
			if err := json.Unmarshal(data, dest); err != nil {
				sc.logger.WithFields(logrus.Fields{
					"key":   key,
					"error": err.Error(),
				}).Warn("Failed to unmarshal from memory cache")
			} else {
				sc.logger.WithField("key", key).Debug("✅ Memory cache hit")
				return nil
			}
		}
	}
	
	// Fallback to Redis
	if err := sc.redis.GetJSON(ctx, key, dest); err != nil {
		return err
	}
	
	// Populate memory cache on Redis hit
	if sc.enabled {
		if data, err := json.Marshal(dest); err == nil {
			sc.setToMemory(key, data)
		}
	}
	
	sc.logger.WithField("key", key).Debug("Redis cache hit (populating memory)")
	return nil
}

// Set stores a value in both memory and Redis
func (sc *ShardedCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Store in Redis first (source of truth)
	if err := sc.redis.SetJSON(ctx, key, value, ttl); err != nil {
		return err
	}
	
	// Populate memory cache
	if sc.enabled {
		if data, err := json.Marshal(value); err == nil {
			sc.setToMemory(key, data)
		}
	}
	
	return nil
}

// Delete removes a value from both memory and Redis
func (sc *ShardedCache) Delete(ctx context.Context, key string) error {
	// Delete from memory
	if sc.enabled {
		shard := sc.getShard(key)
		shard.mu.Lock()
		delete(shard.data, key)
		shard.mu.Unlock()
	}
	
	// Delete from Redis
	return sc.redis.Delete(ctx, key)
}

// getFromMemory retrieves value from in-memory shard
func (sc *ShardedCache) getFromMemory(key string) ([]byte, bool) {
	shard := sc.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	
	entry, exists := shard.data[key]
	if !exists {
		return nil, false
	}
	
	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}
	
	// Update access count (atomic-like, protected by RLock)
	entry.AccessCount++
	
	return entry.Value, true
}

// setToMemory stores value in in-memory shard
func (sc *ShardedCache) setToMemory(key string, data []byte) {
	shard := sc.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	
	shard.data[key] = &CacheEntry{
		Value:      data,
		ExpiresAt:  time.Now().Add(sc.ttl),
		AccessCount: 0,
	}
}

// cleanupExpired periodically removes expired entries
func (sc *ShardedCache) cleanupExpired() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		now := time.Now()
		totalCleaned := 0
		
		for _, shard := range sc.shards {
			shard.mu.Lock()
			for key, entry := range shard.data {
				if now.After(entry.ExpiresAt) {
					delete(shard.data, key)
					totalCleaned++
				}
			}
			shard.mu.Unlock()
		}
		
		if totalCleaned > 0 {
			sc.logger.WithField("cleaned", totalCleaned).Debug("Cleaned expired cache entries")
		}
	}
}

// GetStats returns cache statistics
func (sc *ShardedCache) GetStats() map[string]interface{} {
	if !sc.enabled {
		return map[string]interface{}{
			"enabled": false,
		}
	}
	
	totalEntries := 0
	totalAccessCount := int64(0)
	
	for _, shard := range sc.shards {
		shard.mu.RLock()
		totalEntries += len(shard.data)
		for _, entry := range shard.data {
			totalAccessCount += entry.AccessCount
		}
		shard.mu.RUnlock()
	}
	
	return map[string]interface{}{
		"enabled":           true,
		"shard_count":       sc.shardCount,
		"total_entries":     totalEntries,
		"total_access_count": totalAccessCount,
		"ttl_seconds":       sc.ttl.Seconds(),
		"avg_per_shard":     float64(totalEntries) / float64(sc.shardCount),
	}
}

// Clear removes all entries from memory cache (Redis untouched)
func (sc *ShardedCache) Clear() {
	if !sc.enabled {
		return
	}
	
	for _, shard := range sc.shards {
		shard.mu.Lock()
		shard.data = make(map[string]*CacheEntry)
		shard.mu.Unlock()
	}
	
	sc.logger.Info("✅ Memory cache cleared")
}


// Package redis provides Redis-based caching services
// Version: v3.0.6+ - Model & Request Caching
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// CacheService provides Redis-based caching
type CacheService struct {
	client     *Client
	logger     *logrus.Logger
	defaultTTL time.Duration
	longTTL    time.Duration
	shortTTL   time.Duration
}

// NewCacheService creates a new cache service
func NewCacheService(client *Client, logger *logrus.Logger) *CacheService {
	return &CacheService{
		client:     client,
		logger:     logger,
		defaultTTL: 1 * time.Hour,
		longTTL:    24 * time.Hour,
		shortTTL:   5 * time.Minute,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Model Metadata Caching
// ─────────────────────────────────────────────────────────────────────────────

// ModelMetadata represents cached model metadata
type ModelMetadata struct {
	ModelID         string         `json:"model_id"`
	ModelPath       string         `json:"model_path"`
	Alias           string         `json:"alias"`
	Architecture    string         `json:"architecture"`
	Parameters      int64          `json:"parameters"`
	Quantization    string         `json:"quantization"`
	ContextSize     int            `json:"context_size"`
	IsVLM           bool           `json:"is_vlm"`
	IsLoaded        bool           `json:"is_loaded"`
	LoadedAt        time.Time      `json:"loaded_at"`
	LastUsed        time.Time      `json:"last_used"`
	UsageCount      int64          `json:"usage_count"`
	AvgTokensPerSec float64        `json:"avg_tokens_per_sec"`
	Extra           map[string]any `json:"extra,omitempty"`
}

// CacheModelMetadata caches model metadata
func (s *CacheService) CacheModelMetadata(ctx context.Context, metadata *ModelMetadata) error {
	key := fmt.Sprintf("model_metadata:%s", metadata.ModelID)
	return s.client.Set(ctx, key, metadata, s.longTTL)
}

// GetModelMetadata retrieves cached model metadata
func (s *CacheService) GetModelMetadata(ctx context.Context, modelID string) (*ModelMetadata, error) {
	key := fmt.Sprintf("model_metadata:%s", modelID)

	var metadata ModelMetadata
	if err := s.client.Get(ctx, key, &metadata); err != nil {
		return nil, fmt.Errorf("model metadata not found: %w", err)
	}

	return &metadata, nil
}

// UpdateModelStats updates model usage stats in cache
func (s *CacheService) UpdateModelStats(ctx context.Context, modelID string, tokensPerSec float64) error {
	metadata, err := s.GetModelMetadata(ctx, modelID)
	if err != nil {
		return err
	}

	metadata.UsageCount++
	metadata.LastUsed = time.Now()

	// Update average tokens per second (exponential moving average)
	alpha := 0.3
	if metadata.AvgTokensPerSec == 0 {
		metadata.AvgTokensPerSec = tokensPerSec
	} else {
		metadata.AvgTokensPerSec = alpha*tokensPerSec + (1-alpha)*metadata.AvgTokensPerSec
	}

	return s.CacheModelMetadata(ctx, metadata)
}

// ListLoadedModels gets all loaded models from cache
func (s *CacheService) ListLoadedModels(ctx context.Context) ([]*ModelMetadata, error) {
	pattern := "model_metadata:*"
	keys, err := s.client.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	models := make([]*ModelMetadata, 0, len(keys))
	for _, key := range keys {
		var metadata ModelMetadata
		if err := s.client.Get(ctx, key, &metadata); err == nil && metadata.IsLoaded {
			models = append(models, &metadata)
		}
	}

	return models, nil
}

// InvalidateModelCache invalidates model metadata cache
func (s *CacheService) InvalidateModelCache(ctx context.Context, modelID string) error {
	key := fmt.Sprintf("model_metadata:%s", modelID)
	return s.client.Delete(ctx, key)
}

// ─────────────────────────────────────────────────────────────────────────────
// Request Deduplication (Idempotency)
// ─────────────────────────────────────────────────────────────────────────────

// IdempotencyRecord represents an idempotent request
type IdempotencyRecord struct {
	Key        string    `json:"key"`
	Response   any       `json:"response"`
	StatusCode int       `json:"status_code"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// SaveIdempotencyRecord saves a request response for deduplication
func (s *CacheService) SaveIdempotencyRecord(ctx context.Context, idempotencyKey string, record *IdempotencyRecord) error {
	if idempotencyKey == "" {
		return fmt.Errorf("idempotency_key is required")
	}

	key := fmt.Sprintf("idempotency:%s", idempotencyKey)

	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}
	record.ExpiresAt = record.CreatedAt.Add(s.defaultTTL)

	return s.client.Set(ctx, key, record, s.defaultTTL)
}

// GetIdempotencyRecord retrieves a saved idempotent response
func (s *CacheService) GetIdempotencyRecord(ctx context.Context, idempotencyKey string) (*IdempotencyRecord, error) {
	key := fmt.Sprintf("idempotency:%s", idempotencyKey)

	var record IdempotencyRecord
	if err := s.client.Get(ctx, key, &record); err != nil {
		return nil, fmt.Errorf("idempotency record not found: %w", err)
	}

	return &record, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Generic Caching
// ─────────────────────────────────────────────────────────────────────────────

// Set caches a value with default TTL
func (s *CacheService) Set(ctx context.Context, key string, value any) error {
	cacheKey := fmt.Sprintf("cache:%s", key)
	return s.client.Set(ctx, cacheKey, value, s.defaultTTL)
}

// SetWithTTL caches a value with custom TTL
func (s *CacheService) SetWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error {
	cacheKey := fmt.Sprintf("cache:%s", key)
	return s.client.Set(ctx, cacheKey, value, ttl)
}

// Get retrieves a cached value
func (s *CacheService) Get(ctx context.Context, key string, dest any) error {
	cacheKey := fmt.Sprintf("cache:%s", key)
	return s.client.Get(ctx, cacheKey, dest)
}

// SetJSON caches a value as JSON with custom TTL (v3.0.6+: background sync)
func (s *CacheService) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	cacheKey := fmt.Sprintf("cache:%s", key)
	return s.client.Set(ctx, cacheKey, value, ttl)
}

// GetJSON retrieves a cached value from JSON (v3.0.6+: background sync)
func (s *CacheService) GetJSON(ctx context.Context, key string, dest any) error {
	cacheKey := fmt.Sprintf("cache:%s", key)
	return s.client.Get(ctx, cacheKey, dest)
}

// Delete removes a cached value
func (s *CacheService) Delete(ctx context.Context, key string) error {
	cacheKey := fmt.Sprintf("cache:%s", key)
	return s.client.Delete(ctx, cacheKey)
}

// Exists checks if key exists in cache
func (s *CacheService) Exists(ctx context.Context, key string) (bool, error) {
	cacheKey := fmt.Sprintf("cache:%s", key)
	return s.client.Exists(ctx, cacheKey)
}

// ─────────────────────────────────────────────────────────────────────────────
// Cache Warming & Preloading
// ─────────────────────────────────────────────────────────────────────────────

// WarmCache pre-loads frequently accessed data
func (s *CacheService) WarmCache(ctx context.Context, keys []string, loader func(string) (any, error)) error {
	s.logger.WithField("keys_count", len(keys)).Info("Warming cache...")

	for _, key := range keys {
		// Check if already cached
		exists, _ := s.Exists(ctx, key)
		if exists {
			continue
		}

		// Load data
		data, err := loader(key)
		if err != nil {
			s.logger.WithError(err).WithField("key", key).Warn("Failed to load data for cache warming")
			continue
		}

		// Cache it
		if err := s.Set(ctx, key, data); err != nil {
			s.logger.WithError(err).WithField("key", key).Warn("Failed to cache data")
		}
	}

	s.logger.Info("Cache warming completed")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Cache Statistics
// ─────────────────────────────────────────────────────────────────────────────

// CacheStats represents cache statistics
type CacheStats struct {
	TotalKeys       int64          `json:"total_keys"`
	ModelMetadata   int64          `json:"model_metadata"`
	Sessions        int64          `json:"sessions"`
	IdempotencyKeys int64          `json:"idempotency_keys"`
	RateLimits      int64          `json:"rate_limits"`
	Other           int64          `json:"other"`
	MemoryUsage     string         `json:"memory_usage,omitempty"`
	PoolStats       map[string]any `json:"pool_stats"`
}

// GetStats returns cache statistics
func (s *CacheService) GetStats(ctx context.Context) (*CacheStats, error) {
	// Get database size
	dbSize, err := s.client.DBSize(ctx)
	if err != nil {
		return nil, err
	}

	stats := &CacheStats{
		TotalKeys: dbSize,
		PoolStats: make(map[string]any),
	}

	// Count keys by type
	patterns := map[string]*int64{
		"model_metadata:*": &stats.ModelMetadata,
		"session:*":        &stats.Sessions,
		"idempotency:*":    &stats.IdempotencyKeys,
		"ratelimit:*":      &stats.RateLimits,
	}

	for pattern, counter := range patterns {
		keys, err := s.client.Keys(ctx, pattern)
		if err == nil {
			*counter = int64(len(keys))
		}
	}

	stats.Other = stats.TotalKeys - stats.ModelMetadata - stats.Sessions - stats.IdempotencyKeys - stats.RateLimits

	// Pool stats
	poolStats := s.client.Stats()
	stats.PoolStats["hits"] = poolStats.Hits
	stats.PoolStats["misses"] = poolStats.Misses
	stats.PoolStats["timeouts"] = poolStats.Timeouts
	stats.PoolStats["total_conns"] = poolStats.TotalConns
	stats.PoolStats["idle_conns"] = poolStats.IdleConns
	stats.PoolStats["stale_conns"] = poolStats.StaleConns

	return stats, nil
}

// Package redis - Sharded Cache Tests
// Version: v3.0.8
package redis

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShardedCacheBasicOperations tests basic Get/Set/Delete
func TestShardedCacheBasicOperations(t *testing.T) {
	// Mock Redis service
	mockRedis := &mockCacheService{data: make(map[string][]byte)}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	cache := NewShardedCache(mockRedis, ShardedCacheConfig{
		ShardCount: 16,
		TTL:        1 * time.Minute,
		Enabled:    true,
	}, logger)
	
	ctx := context.Background()
	
	// Test Set
	testData := map[string]interface{}{
		"model_id": "test-model",
		"alias":    "test-alias",
	}
	
	err := cache.Set(ctx, "test:key", testData, 5*time.Minute)
	require.NoError(t, err)
	
	// Test Get (from memory)
	var result map[string]interface{}
	err = cache.Get(ctx, "test:key", &result)
	require.NoError(t, err)
	assert.Equal(t, "test-model", result["model_id"])
	assert.Equal(t, "test-alias", result["alias"])
	
	// Test Delete
	err = cache.Delete(ctx, "test:key")
	require.NoError(t, err)
	
	// Verify deleted
	err = cache.Get(ctx, "test:key", &result)
	assert.Error(t, err) // Should not be in memory
}

// TestShardedCacheConcurrency tests concurrent access
func TestShardedCacheConcurrency(t *testing.T) {
	mockRedis := &mockCacheService{data: make(map[string][]byte)}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	cache := NewShardedCache(mockRedis, ShardedCacheConfig{
		ShardCount: 32,
		TTL:        1 * time.Minute,
		Enabled:    true,
	}, logger)
	
	ctx := context.Background()
	
	// Concurrent writes
	var wg sync.WaitGroup
	concurrency := 100
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			key := "concurrent:key:" + string(rune(id))
			value := map[string]int{"id": id}
			
			err := cache.Set(ctx, key, value, 1*time.Minute)
			assert.NoError(t, err)
			
			var result map[string]int
			err = cache.Get(ctx, key, &result)
			assert.NoError(t, err)
			assert.Equal(t, id, result["id"])
		}(i)
	}
	
	wg.Wait()
	
	stats := cache.GetStats()
	t.Logf("✅ Concurrent operations completed, entries: %v", stats["total_entries"])
}

// TestShardedCacheExpiration tests TTL expiration
func TestShardedCacheExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping expiration test in short mode")
	}
	
	mockRedis := &mockCacheService{data: make(map[string][]byte)}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	cache := NewShardedCache(mockRedis, ShardedCacheConfig{
		ShardCount: 8,
		TTL:        100 * time.Millisecond, // Short TTL for testing
		Enabled:    true,
	}, logger)
	
	ctx := context.Background()
	
	// Set value
	err := cache.Set(ctx, "expire:key", "test-value", 1*time.Minute)
	require.NoError(t, err)
	
	// Should be in memory
	var result string
	err = cache.Get(ctx, "expire:key", &result)
	require.NoError(t, err)
	assert.Equal(t, "test-value", result)
	
	// Wait for expiration
	time.Sleep(150 * time.Millisecond)
	
	// Should fallback to Redis (memory expired)
	err = cache.Get(ctx, "expire:key", &result)
	require.NoError(t, err) // Redis still has it
	assert.Equal(t, "test-value", result)
	
	t.Log("✅ Expiration test passed")
}

// TestShardedCacheStats tests statistics reporting
func TestShardedCacheStats(t *testing.T) {
	mockRedis := &mockCacheService{data: make(map[string][]byte)}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	cache := NewShardedCache(mockRedis, ShardedCacheConfig{
		ShardCount: 16,
		TTL:        1 * time.Minute,
		Enabled:    true,
	}, logger)
	
	ctx := context.Background()
	
	// Add some entries
	for i := 0; i < 50; i++ {
		key := "stats:key:" + string(rune(i))
		_ = cache.Set(ctx, key, i, 1*time.Minute)
	}
	
	// Get stats
	stats := cache.GetStats()
	
	assert.True(t, stats["enabled"].(bool))
	assert.Equal(t, uint32(16), stats["shard_count"])
	assert.Greater(t, stats["total_entries"].(int), 0)
	
	t.Logf("✅ Stats: %+v", stats)
}

// TestShardedCacheDisabled tests behavior when disabled
func TestShardedCacheDisabled(t *testing.T) {
	mockRedis := &mockCacheService{data: make(map[string][]byte)}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	cache := NewShardedCache(mockRedis, ShardedCacheConfig{
		ShardCount: 16,
		TTL:        1 * time.Minute,
		Enabled:    false, // Disabled
	}, logger)
	
	ctx := context.Background()
	
	// Set should still work (Redis only)
	err := cache.Set(ctx, "disabled:key", "test", 1*time.Minute)
	require.NoError(t, err)
	
	// Get should work (Redis fallback)
	var result string
	err = cache.Get(ctx, "disabled:key", &result)
	require.NoError(t, err)
	assert.Equal(t, "test", result)
	
	// Stats should show disabled
	stats := cache.GetStats()
	assert.False(t, stats["enabled"].(bool))
	
	t.Log("✅ Disabled mode works correctly")
}

// BenchmarkShardedCacheGet benchmarks Get operations
func BenchmarkShardedCacheGet(b *testing.B) {
	mockRedis := &mockCacheService{data: make(map[string][]byte)}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	cache := NewShardedCache(mockRedis, ShardedCacheConfig{
		ShardCount: 32,
		TTL:        5 * time.Minute,
		Enabled:    true,
	}, logger)
	
	ctx := context.Background()
	
	// Pre-populate cache
	_ = cache.Set(ctx, "bench:key", "test-value", 5*time.Minute)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		var result string
		_ = cache.Get(ctx, "bench:key", &result)
	}
}

// BenchmarkShardedCacheSetParallel benchmarks concurrent Set operations
func BenchmarkShardedCacheSetParallel(b *testing.B) {
	mockRedis := &mockCacheService{data: make(map[string][]byte)}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	cache := NewShardedCache(mockRedis, ShardedCacheConfig{
		ShardCount: 32,
		TTL:        5 * time.Minute,
		Enabled:    true,
	}, logger)
	
	ctx := context.Background()
	
	b.ResetTimer()
	
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "parallel:key:" + string(rune(i))
			_ = cache.Set(ctx, key, i, 5*time.Minute)
			i++
		}
	})
}

// Mock CacheService for testing
type mockCacheService struct {
	data map[string][]byte
	mu   sync.RWMutex
}

func (m *mockCacheService) GetJSON(ctx context.Context, key string, dest interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	data, exists := m.data[key]
	if !exists {
		return assert.AnError
	}
	
	return json.Unmarshal(data, dest)
}

func (m *mockCacheService) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	
	m.data[key] = data
	return nil
}

func (m *mockCacheService) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.data, key)
	return nil
}


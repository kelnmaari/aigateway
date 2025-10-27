package cache

import (
	"sync"
	"testing"
	"time"

	"aigateway/internal/models"
)

func TestAPIKeyCache_GetSet(t *testing.T) {
	cache := NewAPIKeyCache(Config{
		TTL:             1 * time.Second,
		MaxSize:         100,
		CleanupInterval: 100 * time.Millisecond,
	})
	defer cache.Close()

	// Create test APIKeyHot
	hot := &models.APIKeyHot{
		ID:             "test-key-1",
		Status:         models.APIKeyStatusActive,
		ModelsAll:      true,
		PermissionsAll: true,
		IsExpired:      false,
	}

	// Set
	cache.Set("test-key-1", hot)

	// Get
	retrieved, found := cache.Get("test-key-1")
	if !found {
		t.Fatal("Expected to find key in cache")
	}

	if retrieved.ID != "test-key-1" {
		t.Errorf("Expected ID test-key-1, got %s", retrieved.ID)
	}
}

func TestAPIKeyCache_Expiration(t *testing.T) {
	cache := NewAPIKeyCache(Config{
		TTL:             100 * time.Millisecond, // Very short TTL
		MaxSize:         100,
		CleanupInterval: 50 * time.Millisecond,
	})
	defer cache.Close()

	hot := &models.APIKeyHot{
		ID:     "expire-test",
		Status: models.APIKeyStatusActive,
	}

	cache.Set("expire-test", hot)

	// Should exist immediately
	if _, found := cache.Get("expire-test"); !found {
		t.Fatal("Expected to find key immediately after set")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	if _, found := cache.Get("expire-test"); found {
		t.Fatal("Expected key to be expired")
	}
}

func TestAPIKeyCache_Delete(t *testing.T) {
	cache := NewAPIKeyCache(DefaultConfig())
	defer cache.Close()

	hot := &models.APIKeyHot{ID: "delete-test"}
	cache.Set("delete-test", hot)

	// Verify exists
	if _, found := cache.Get("delete-test"); !found {
		t.Fatal("Key should exist before delete")
	}

	// Delete
	cache.Delete("delete-test")

	// Verify deleted
	if _, found := cache.Get("delete-test"); found {
		t.Fatal("Key should not exist after delete")
	}
}

func TestAPIKeyCache_Eviction(t *testing.T) {
	cache := NewAPIKeyCache(Config{
		TTL:             1 * time.Minute,
		MaxSize:         5, // Very small cache
		CleanupInterval: 1 * time.Second,
	})
	defer cache.Close()

	// Fill cache beyond max size
	for i := 0; i < 10; i++ {
		hot := &models.APIKeyHot{
			ID: string(rune('a' + i)),
		}
		cache.Set(string(rune('a'+i)), hot)
	}

	// Cache should not exceed max size
	size := cache.Size()
	if size > 5 {
		t.Errorf("Cache size %d exceeds max size 5", size)
	}
}

func TestAPIKeyCache_Concurrent(t *testing.T) {
	cache := NewAPIKeyCache(DefaultConfig())
	defer cache.Close()

	var wg sync.WaitGroup
	iterations := 1000

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				hot := &models.APIKeyHot{
					ID: string(rune('a' + id)),
				}
				cache.Set(string(rune('a'+id)), hot)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				cache.Get(string(rune('a' + id)))
			}
		}(i)
	}

	wg.Wait()

	// Should not panic or race
	stats := cache.Stats()
	if stats.Size > cache.maxSize {
		t.Errorf("Cache size %d exceeds max %d", stats.Size, cache.maxSize)
	}
}

func TestAPIKeyCache_GetOrLoad(t *testing.T) {
	cache := NewAPIKeyCache(DefaultConfig())
	defer cache.Close()

	loadCount := 0
	loader := func() (*models.APIKeyHot, error) {
		loadCount++
		return &models.APIKeyHot{
			ID:     "loaded-key",
			Status: models.APIKeyStatusActive,
		}, nil
	}

	// First call should load
	hot1, err := cache.GetOrLoad("loaded-key", loader)
	if err != nil {
		t.Fatalf("GetOrLoad failed: %v", err)
	}
	if hot1.ID != "loaded-key" {
		t.Errorf("Expected ID loaded-key, got %s", hot1.ID)
	}
	if loadCount != 1 {
		t.Errorf("Expected 1 load, got %d", loadCount)
	}

	// Second call should use cache
	hot2, err := cache.GetOrLoad("loaded-key", loader)
	if err != nil {
		t.Fatalf("GetOrLoad failed: %v", err)
	}
	if hot2.ID != "loaded-key" {
		t.Errorf("Expected ID loaded-key, got %s", hot2.ID)
	}
	if loadCount != 1 {
		t.Errorf("Expected still 1 load (cached), got %d", loadCount)
	}
}

func BenchmarkAPIKeyCache_Get(b *testing.B) {
	cache := NewAPIKeyCache(DefaultConfig())
	defer cache.Close()

	hot := &models.APIKeyHot{ID: "bench-key"}
	cache.Set("bench-key", hot)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("bench-key")
	}
}

func BenchmarkAPIKeyCache_Set(b *testing.B) {
	cache := NewAPIKeyCache(DefaultConfig())
	defer cache.Close()

	hot := &models.APIKeyHot{ID: "bench-key"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("bench-key", hot)
	}
}

func BenchmarkAPIKeyCache_Parallel(b *testing.B) {
	cache := NewAPIKeyCache(DefaultConfig())
	defer cache.Close()

	hot := &models.APIKeyHot{ID: "bench-key"}
	cache.Set("bench-key", hot)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Get("bench-key")
		}
	})
}


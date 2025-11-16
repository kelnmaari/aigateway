// Package huggingface provides integration tests for circuit breaker
// Version: v3.0.8 - Circuit breaker integration tests (sony/gobreaker)
package huggingface

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCircuitBreakerOpensAfterFailures verifies circuit opens after consecutive failures
func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	failureCount := int32(0)
	
	// Mock server that fails consistently
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&failureCount, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error": "service unavailable"}`))
	}))
	defer server.Close()
	
	// Create client with mock server URL
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Suppress debug logs for test
	client := NewClient("test-token", logger)
	client.baseURL = server.URL
	
	ctx := context.Background()
	
	// Make requests until circuit opens (5 failures expected)
	for i := 0; i < 10; i++ {
		_, err := client.SearchModels(ctx, ModelFilters{Search: "test"})
		require.Error(t, err)
		
		t.Logf("Request %d error: %v", i+1, err)
	}
	
	// Verify circuit opened (should have stopped at 5 failures)
	finalCount := atomic.LoadInt32(&failureCount)
	assert.LessOrEqual(t, finalCount, int32(6), "Circuit should open after 5 failures")
	
	t.Logf("✅ Circuit breaker opened after %d failures", finalCount)
}

// TestCircuitBreakerGetModelInfo verifies circuit breaker for GetModelInfo
func TestCircuitBreakerGetModelInfo(t *testing.T) {
	requestCount := int32(0)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer server.Close()
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	client := NewClient("test-token", logger)
	client.baseURL = server.URL
	
	ctx := context.Background()
	
	// Trigger circuit breaker
	for i := 0; i < 10; i++ {
		_, err := client.GetModelInfo(ctx, "test-model")
		require.Error(t, err)
	}
	
	finalCount := atomic.LoadInt32(&requestCount)
	assert.LessOrEqual(t, finalCount, int32(6), "Circuit breaker should limit requests")
	
	t.Logf("✅ GetModelInfo circuit breaker stopped after: %d failures", finalCount)
}

// TestCircuitBreakerConcurrentRequests verifies circuit breaker under concurrent load
func TestCircuitBreakerConcurrentRequests(t *testing.T) {
	requestCount := int32(0)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error": "bad gateway"}`))
	}))
	defer server.Close()
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	client := NewClient("test-token", logger)
	client.baseURL = server.URL
	
	ctx := context.Background()
	
	// Launch concurrent requests
	concurrency := 20
	done := make(chan bool, concurrency)
	
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			for j := 0; j < 3; j++ {
				_, err := client.SearchModels(ctx, ModelFilters{Search: "concurrent"})
				if err == nil {
					t.Errorf("Expected error from goroutine %d request %d", id, j)
				}
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < concurrency; i++ {
		<-done
	}
	
	finalCount := atomic.LoadInt32(&requestCount)
	t.Logf("Total requests with %d concurrent goroutines: %d", concurrency, finalCount)
	
	// Circuit breaker should significantly reduce total requests
	// Without CB: 20 * 3 = 60 requests
	// With CB: Should stop around 5-20 requests (concurrent race)
	assert.LessOrEqual(t, finalCount, int32(30), "Circuit breaker should prevent most requests under concurrent load")
	
	t.Logf("✅ Circuit breaker reduced concurrent requests from 60 to %d", finalCount)
}

// TestCircuitBreakerSuccess verifies circuit breaker allows successful requests
func TestCircuitBreakerSuccess(t *testing.T) {
	requestCount := int32(0)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": "test-model", "author": "test"}]`))
	}))
	defer server.Close()
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	client := NewClient("test-token", logger)
	client.baseURL = server.URL
	
	ctx := context.Background()
	
	// Make multiple successful requests
	for i := 0; i < 10; i++ {
		models, err := client.SearchModels(ctx, ModelFilters{Search: "test"})
		require.NoError(t, err)
		assert.Len(t, models, 1)
		assert.Equal(t, "test-model", models[0].ID)
	}
	
	finalCount := atomic.LoadInt32(&requestCount)
	assert.Equal(t, int32(10), finalCount, "All successful requests should pass through circuit breaker")
	
	t.Log("✅ Circuit breaker allows all successful requests")
}

// BenchmarkCircuitBreakerOverhead measures performance overhead of circuit breaker
func BenchmarkCircuitBreakerOverhead(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	client := NewClient("", logger)
	client.baseURL = server.URL
	
	ctx := context.Background()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, _ = client.SearchModels(ctx, ModelFilters{Limit: 1})
	}
}


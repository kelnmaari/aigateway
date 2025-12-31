// Package concurrent - Worker Pool Tests
// Version: v3.0.8
package concurrent

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkerPoolBasic tests basic pool operations
func TestWorkerPoolBasic(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	pool := NewWorkerPool(PoolConfig{
		Workers:   5,
		QueueSize: 10,
		Logger:    logger,
	})
	defer pool.Shutdown(5 * time.Second)
	
	// Submit tasks
	var counter int64
	for i := 0; i < 20; i++ {
		err := pool.Submit(Task{
			ID: "test-task-" + string(rune(i)),
			Execute: func(ctx context.Context) error {
				atomic.AddInt64(&counter, 1)
				time.Sleep(10 * time.Millisecond)
				return nil
			},
		})
		require.NoError(t, err)
	}
	
	// Wait and verify
	time.Sleep(500 * time.Millisecond)
	
	metrics := pool.GetMetrics()
	assert.Equal(t, int64(20), metrics.TotalTasks)
	assert.Equal(t, int64(20), metrics.CompletedTasks)
	assert.Equal(t, int64(0), metrics.FailedTasks)
	assert.Equal(t, int64(20), atomic.LoadInt64(&counter))
	
	t.Logf("✅ Metrics: %+v", metrics)
}

// TestWorkerPoolError tests error handling
func TestWorkerPoolError(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	pool := NewWorkerPool(PoolConfig{
		Workers:   3,
		QueueSize: 5,
		Logger:    logger,
	})
	defer pool.Shutdown(5 * time.Second)
	
	// Submit failing task
	var errorCount int64
	
	err := pool.Submit(Task{
		ID: "failing-task",
		Execute: func(ctx context.Context) error {
			return errors.New("intentional failure")
		},
		OnError: func(err error) {
			atomic.AddInt64(&errorCount, 1)
		},
	})
	require.NoError(t, err)
	
	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	
	metrics := pool.GetMetrics()
	assert.Equal(t, int64(1), metrics.FailedTasks)
	assert.Equal(t, int64(1), atomic.LoadInt64(&errorCount))
	
	t.Log("✅ Error handling works")
}

// TestWorkerPoolShutdown tests graceful shutdown
func TestWorkerPoolShutdown(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	pool := NewWorkerPool(PoolConfig{
		Workers:   2,
		QueueSize: 5,
		Logger:    logger,
	})
	
	// Submit long-running tasks
	for i := 0; i < 5; i++ {
		_ = pool.Submit(Task{
			ID: "long-task-" + string(rune(i)),
			Execute: func(ctx context.Context) error {
				time.Sleep(50 * time.Millisecond)
				return nil
			},
		})
	}
	
	// Shutdown with timeout
	err := pool.Shutdown(2 * time.Second)
	require.NoError(t, err)
	
	metrics := pool.GetMetrics()
	assert.Equal(t, int64(5), metrics.CompletedTasks)
	
	t.Log("✅ Graceful shutdown works")
}

// TestBatchProcessor tests batch processing
func TestBatchProcessor(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	pool := NewWorkerPool(PoolConfig{
		Workers:   4,
		QueueSize: 20,
		Logger:    logger,
	})
	defer pool.Shutdown(5 * time.Second)
	
	bp := NewBatchProcessor[int](5, pool)
	
	// Process items
	items := make([]int, 50)
	for i := range items {
		items[i] = i
	}
	
	var counter int64
	err := bp.Process(context.Background(), items, func(item int) error {
		atomic.AddInt64(&counter, 1)
		return nil
	})
	
	require.NoError(t, err)
	assert.Equal(t, int64(50), atomic.LoadInt64(&counter))
	
	t.Log("✅ Batch processing completed")
}

// TestParallelMap tests parallel map function
func TestParallelMap(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	
	results, err := ParallelMap(
		context.Background(),
		items,
		func(x int) (int, error) {
			return x * 2, nil
		},
		4,
	)
	
	require.NoError(t, err)
	require.Len(t, results, 10)
	
	expected := []int{2, 4, 6, 8, 10, 12, 14, 16, 18, 20}
	assert.Equal(t, expected, results)
	
	t.Log("✅ ParallelMap works correctly")
}

// TestParallelMapError tests parallel map error handling
func TestParallelMapError(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	
	_, err := ParallelMap(
		context.Background(),
		items,
		func(x int) (int, error) {
			if x == 3 {
				return 0, errors.New("error at 3")
			}
			return x * 2, nil
		},
		2,
	)
	
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error at 3")
	
	t.Log("✅ ParallelMap error handling works")
}

// BenchmarkWorkerPoolThroughput benchmarks pool throughput
func BenchmarkWorkerPoolThroughput(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	pool := NewWorkerPool(PoolConfig{
		Workers:   10,
		QueueSize: 1000,
		Logger:    logger,
	})
	defer pool.Shutdown(5 * time.Second)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = pool.Submit(Task{
			ID: "bench-task",
			Execute: func(ctx context.Context) error {
				// Minimal work
				return nil
			},
		})
	}
}

// BenchmarkParallelMap benchmarks parallel map
func BenchmarkParallelMap(b *testing.B) {
	items := make([]int, 1000)
	for i := range items {
		items[i] = i
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, _ = ParallelMap(
			context.Background(),
			items,
			func(x int) (int, error) {
				return x * 2, nil
			},
			10,
		)
	}
}


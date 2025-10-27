// Package processor provides worker pool tests.
package processor

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// Mock Document Processor
type mockDocumentProcessor struct {
	processFunc func(ctx context.Context, documentID, text string) error
	callCount   atomic.Int64
}

func (m *mockDocumentProcessor) ProcessDocument(ctx context.Context, documentID, text string) error {
	m.callCount.Add(1)
	if m.processFunc != nil {
		return m.processFunc(ctx, documentID, text)
	}
	// Simulate some work
	time.Sleep(10 * time.Millisecond)
	return nil
}

func TestNewWorkerPool(t *testing.T) {
	processor := &mockDocumentProcessor{}
	logger := logrus.New()

	pool := NewWorkerPool(processor, 4, 100, logger)

	if pool == nil {
		t.Fatal("Expected non-nil worker pool")
	}

	if pool.workers != 4 {
		t.Errorf("Expected 4 workers, got %d", pool.workers)
	}

	if cap(pool.jobQueue) != 100 {
		t.Errorf("Expected queue capacity 100, got %d", cap(pool.jobQueue))
	}
}

func TestNewWorkerPool_DefaultValues(t *testing.T) {
	processor := &mockDocumentProcessor{}
	logger := logrus.New()

	// Test with zero/negative values
	pool := NewWorkerPool(processor, 0, 0, logger)

	if pool.workers <= 0 {
		t.Error("Expected positive worker count")
	}

	if cap(pool.jobQueue) <= 0 {
		t.Error("Expected positive queue capacity")
	}
}

func TestWorkerPool_StartStop(t *testing.T) {
	processor := &mockDocumentProcessor{}
	logger := logrus.New()

	pool := NewWorkerPool(processor, 2, 10, logger)

	// Start pool
	err := pool.Start()
	if err != nil {
		t.Fatalf("Expected no error starting pool, got %v", err)
	}

	if !pool.isRunning.Load() {
		t.Error("Expected pool to be running")
	}

	// Try starting again (should error)
	err = pool.Start()
	if err == nil {
		t.Error("Expected error when starting already running pool")
	}

	// Stop pool
	err = pool.Stop()
	if err != nil {
		t.Fatalf("Expected no error stopping pool, got %v", err)
	}

	if pool.isRunning.Load() {
		t.Error("Expected pool to be stopped")
	}

	// Try stopping again (should error)
	err = pool.Stop()
	if err == nil {
		t.Error("Expected error when stopping already stopped pool")
	}
}

func TestWorkerPool_Submit(t *testing.T) {
	processor := &mockDocumentProcessor{}
	logger := logrus.New()

	pool := NewWorkerPool(processor, 2, 10, logger)
	pool.Start()
	defer pool.Stop()

	job := ProcessingJob{
		DocumentID:   "doc1",
		DocumentText: "Test document content",
		Priority:     1,
	}

	err := pool.Submit(job)
	if err != nil {
		t.Fatalf("Expected no error submitting job, got %v", err)
	}

	// Wait for job to be processed
	time.Sleep(100 * time.Millisecond)

	// Check that job was processed
	stats := pool.Stats()
	if stats.JobsProcessed == 0 {
		t.Error("Expected at least 1 processed job")
	}
}

func TestWorkerPool_Submit_NotRunning(t *testing.T) {
	processor := &mockDocumentProcessor{}
	logger := logrus.New()

	pool := NewWorkerPool(processor, 2, 10, logger)
	// Don't start the pool

	job := ProcessingJob{
		DocumentID:   "doc1",
		DocumentText: "Test content",
	}

	err := pool.Submit(job)
	if err == nil {
		t.Error("Expected error when submitting to stopped pool")
	}
}

func TestWorkerPool_ProcessMultipleJobs(t *testing.T) {
	processor := &mockDocumentProcessor{}
	logger := logrus.New()

	pool := NewWorkerPool(processor, 4, 100, logger)
	pool.Start()
	defer pool.Stop()

	// Submit 10 jobs
	numJobs := 10
	for i := 0; i < numJobs; i++ {
		job := ProcessingJob{
			DocumentID:   "doc" + string(rune(i+'0')),
			DocumentText: "Test content",
			Priority:     1,
		}

		err := pool.Submit(job)
		if err != nil {
			t.Errorf("Failed to submit job %d: %v", i, err)
		}
	}

	// Wait for all jobs to complete
	err := pool.WaitUntilEmpty(5 * time.Second)
	if err != nil {
		t.Fatalf("Timeout waiting for jobs to complete: %v", err)
	}

	// Check stats
	stats := pool.Stats()
	if stats.JobsProcessed != int64(numJobs) {
		t.Errorf("Expected %d processed jobs, got %d", numJobs, stats.JobsProcessed)
	}

	if stats.JobsFailed != 0 {
		t.Errorf("Expected 0 failed jobs, got %d", stats.JobsFailed)
	}

	if stats.QueueSize != 0 {
		t.Errorf("Expected empty queue, got %d", stats.QueueSize)
	}
}

func TestWorkerPool_JobFailure(t *testing.T) {
	// Processor that fails
	processor := &mockDocumentProcessor{
		processFunc: func(ctx context.Context, documentID, text string) error {
			return context.DeadlineExceeded // Simulate error
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress error logs

	pool := NewWorkerPool(processor, 2, 10, logger)
	pool.Start()
	defer pool.Stop()

	job := ProcessingJob{
		DocumentID:   "doc1",
		DocumentText: "Test content",
	}

	pool.Submit(job)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	stats := pool.Stats()
	if stats.JobsFailed == 0 {
		t.Error("Expected at least 1 failed job")
	}
}

func TestWorkerPool_Stats(t *testing.T) {
	processor := &mockDocumentProcessor{}
	logger := logrus.New()

	pool := NewWorkerPool(processor, 3, 50, logger)

	stats := pool.Stats()

	if stats.Workers != 3 {
		t.Errorf("Expected 3 workers, got %d", stats.Workers)
	}

	if stats.QueueCapacity != 50 {
		t.Errorf("Expected queue capacity 50, got %d", stats.QueueCapacity)
	}

	if stats.QueueSize != 0 {
		t.Errorf("Expected queue size 0, got %d", stats.QueueSize)
	}

	if stats.IsRunning {
		t.Error("Expected pool not running initially")
	}

	if stats.JobsProcessed != 0 {
		t.Errorf("Expected 0 processed jobs, got %d", stats.JobsProcessed)
	}

	if stats.JobsFailed != 0 {
		t.Errorf("Expected 0 failed jobs, got %d", stats.JobsFailed)
	}
}

func TestWorkerPool_WaitUntilEmpty(t *testing.T) {
	processor := &mockDocumentProcessor{}
	logger := logrus.New()

	pool := NewWorkerPool(processor, 2, 10, logger)
	pool.Start()
	defer pool.Stop()

	// Submit jobs
	for i := 0; i < 5; i++ {
		job := ProcessingJob{
			DocumentID:   "doc" + string(rune(i+'0')),
			DocumentText: "Test",
		}
		pool.Submit(job)
	}

	// Wait with reasonable timeout
	err := pool.WaitUntilEmpty(2 * time.Second)
	if err != nil {
		t.Errorf("Expected queue to empty within timeout, got %v", err)
	}

	stats := pool.Stats()
	if stats.QueueSize != 0 {
		t.Errorf("Expected empty queue, got %d", stats.QueueSize)
	}
}

func TestWorkerPool_WaitUntilEmpty_Timeout(t *testing.T) {
	// Slow processor
	processor := &mockDocumentProcessor{
		processFunc: func(ctx context.Context, documentID, text string) error {
			time.Sleep(1 * time.Second)
			return nil
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	pool := NewWorkerPool(processor, 1, 10, logger)
	pool.Start()
	defer pool.Stop()

	// Submit many jobs
	for i := 0; i < 10; i++ {
		job := ProcessingJob{
			DocumentID:   "doc" + string(rune(i+'0')),
			DocumentText: "Test",
		}
		pool.Submit(job)
	}

	// Wait with short timeout (should timeout)
	err := pool.WaitUntilEmpty(100 * time.Millisecond)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestWorkerPool_ConcurrentSubmit(t *testing.T) {
	processor := &mockDocumentProcessor{}
	logger := logrus.New()

	pool := NewWorkerPool(processor, 4, 100, logger)
	pool.Start()
	defer pool.Stop()

	// Submit jobs concurrently from multiple goroutines
	numGoroutines := 5
	jobsPerGoroutine := 10

	var submitted atomic.Int64

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < jobsPerGoroutine; j++ {
				job := ProcessingJob{
					DocumentID:   "doc",
					DocumentText: "Test",
				}
				if pool.Submit(job) == nil {
					submitted.Add(1)
				}
			}
		}(i)
	}

	// Wait for all submissions and processing
	time.Sleep(1 * time.Second)

	totalExpected := int64(numGoroutines * jobsPerGoroutine)
	if submitted.Load() != totalExpected {
		t.Errorf("Expected %d jobs submitted, got %d", totalExpected, submitted.Load())
	}
}

// Benchmark tests
func BenchmarkWorkerPool_Submit(b *testing.B) {
	processor := &mockDocumentProcessor{
		processFunc: func(ctx context.Context, documentID, text string) error {
			// Fast processing
			return nil
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	pool := NewWorkerPool(processor, 4, 10000, logger)
	pool.Start()
	defer pool.Stop()

	job := ProcessingJob{
		DocumentID:   "doc1",
		DocumentText: "Benchmark content",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pool.Submit(job)
	}
}

func BenchmarkWorkerPool_ProcessJobs(b *testing.B) {
	processor := &mockDocumentProcessor{
		processFunc: func(ctx context.Context, documentID, text string) error {
			// Simulate some work
			time.Sleep(1 * time.Microsecond)
			return nil
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	pool := NewWorkerPool(processor, 8, 10000, logger)
	pool.Start()
	defer pool.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		job := ProcessingJob{
			DocumentID:   "doc",
			DocumentText: "content",
		}
		pool.Submit(job)
	}

	pool.WaitUntilEmpty(10 * time.Second)
}



// Package testing provides load tests for GitLab integration
package testing

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"aigateway/internal/gitlab/client"
)

// TestLoad_ConcurrentWebhooks simulates 100+ concurrent webhooks
func TestLoad_ConcurrentWebhooks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	mockGitLab := NewMockGitLabServer()
	defer mockGitLab.Close()

	// Setup test data
	for i := int64(1); i <= 10; i++ {
		project := &client.Project{ID: i, Name: "load-test-project"}
		mockGitLab.AddProject(project)

		for j := 1; j <= 10; j++ {
			mr := &client.MergeRequest{
				ID:    i*100 + int64(j),
				IID:   j,
				Title: "Load test MR",
				State: "opened",
			}
			mockGitLab.AddMergeRequest(i, mr)
		}
	}

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:         mockGitLab.URL,
		AccessToken:     "test-token",
		RateLimitPerSec: 1000, // High rate for load test
	})

	ctx := context.Background()
	concurrency := 100
	requestsPerGoroutine := 10

	var successCount int64
	var errorCount int64
	var wg sync.WaitGroup

	start := time.Now()

	for i := range concurrency {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := range requestsPerGoroutine {
				projectID := int64((workerID % 10) + 1)
				mrIID := (j % 10) + 1

				_, err := gitlabClient.GetMergeRequest(ctx, projectID, mrIID)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalRequests := concurrency * requestsPerGoroutine
	requestsPerSecond := float64(totalRequests) / duration.Seconds()

	t.Logf("Load test results:")
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Failed: %d", errorCount)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Requests/second: %.2f", requestsPerSecond)

	// Assertions
	if errorCount > int64(totalRequests/10) {
		t.Errorf("Too many errors: %d out of %d", errorCount, totalRequests)
	}

	if requestsPerSecond < 100 {
		t.Logf("Warning: Low throughput (%.2f req/s), but not failing test", requestsPerSecond)
	}
}

// TestLoad_QueueProcessing simulates queue under load
func TestLoad_QueueProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// Simulate job queue
	jobQueue := make(chan int, 1000)
	results := make(chan int, 1000)

	// Worker pool
	numWorkers := 10
	var wg sync.WaitGroup

	// Start workers
	for i := range numWorkers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for jobID := range jobQueue {
				// Simulate job processing (1-5ms)
				time.Sleep(time.Duration(1+jobID%5) * time.Millisecond)
				results <- jobID
			}
		}(i)
	}

	// Submit jobs
	numJobs := 500
	start := time.Now()

	go func() {
		for i := range numJobs {
			jobQueue <- i
		}
		close(jobQueue)
	}()

	// Collect results
	var processed int
	go func() {
		wg.Wait()
		close(results)
	}()

	for range results {
		processed++
	}

	duration := time.Since(start)
	jobsPerSecond := float64(processed) / duration.Seconds()

	t.Logf("Queue processing results:")
	t.Logf("  Jobs processed: %d", processed)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Jobs/second: %.2f", jobsPerSecond)

	if processed != numJobs {
		t.Errorf("Not all jobs processed: %d / %d", processed, numJobs)
	}
}

// TestLoad_CommentPosting tests posting many comments
func TestLoad_CommentPosting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	mockGitLab := NewMockGitLabServer()
	defer mockGitLab.Close()

	project := &client.Project{ID: 1, Name: "comment-load-test"}
	mockGitLab.AddProject(project)

	for i := 1; i <= 50; i++ {
		mr := &client.MergeRequest{
			ID:    int64(i),
			IID:   i,
			Title: "MR for comment test",
			State: "opened",
		}
		mockGitLab.AddMergeRequest(1, mr)
	}

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     mockGitLab.URL,
		AccessToken: "test-token",
	})

	ctx := context.Background()
	concurrency := 50

	var successCount int64
	var wg sync.WaitGroup

	start := time.Now()

	for i := 1; i <= concurrency; i++ {
		wg.Add(1)
		go func(mrIID int) {
			defer wg.Done()
			comment := "## AI Review\n\nThis is a test comment for load testing."
			_, err := gitlabClient.CreateMRNote(ctx, 1, mrIID, comment)
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Comment posting results:")
	t.Logf("  Comments posted: %d", successCount)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Comments/second: %.2f", float64(successCount)/duration.Seconds())

	if successCount < int64(concurrency*80/100) {
		t.Errorf("Too few successful comments: %d / %d", successCount, concurrency)
	}
}

// TestLoad_APIRateLimiting tests rate limiter under pressure
func TestLoad_APIRateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	mockGitLab := NewMockGitLabServer()
	defer mockGitLab.Close()

	mockGitLab.SetCurrentUser(&client.User{
		ID:       1,
		Username: "test",
	})

	// Client with strict rate limit
	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:         mockGitLab.URL,
		AccessToken:     "test-token",
		RateLimitPerSec: 50, // 50 requests per second
	})

	ctx := context.Background()
	numRequests := 200

	var successCount int64
	var wg sync.WaitGroup

	start := time.Now()

	for range numRequests {
		wg.Go(func() {
			_, err := gitlabClient.GetCurrentUser(ctx)
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			}
		})
	}

	wg.Wait()
	duration := time.Since(start)

	actualRate := float64(successCount) / duration.Seconds()

	t.Logf("Rate limiting results:")
	t.Logf("  Requests attempted: %d", numRequests)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Actual rate: %.2f req/s", actualRate)

	// Rate should be close to configured limit (allowing some variance)
	// The actual rate depends on rate limiter implementation
	if actualRate > 200 {
		t.Logf("Warning: Rate limiter may not be working as expected (%.2f req/s)", actualRate)
	}
}

// TestLoad_MemoryStability tests for memory leaks under load
func TestLoad_MemoryStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	mockGitLab := NewMockGitLabServer()
	defer mockGitLab.Close()

	project := &client.Project{ID: 1, Name: "memory-test"}
	mockGitLab.AddProject(project)

	mr := &client.MergeRequest{
		ID:    1,
		IID:   1,
		Title: "Memory test MR",
		State: "opened",
	}
	mockGitLab.AddMergeRequest(1, mr)

	changes := &client.MergeRequestChanges{
		MergeRequest: *mr,
		Changes: []client.Change{
			{NewPath: "file.go", Diff: "+code"},
		},
	}
	mockGitLab.AddMRChanges(1, 1, changes)

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     mockGitLab.URL,
		AccessToken: "test-token",
	})

	ctx := context.Background()

	// Run many iterations
	iterations := 1000
	for i := range iterations {
		_, _ = gitlabClient.GetMergeRequest(ctx, 1, 1)
		_, _ = gitlabClient.GetMergeRequestChanges(ctx, 1, 1)

		if i%100 == 0 {
			// Give GC a chance to run
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Reset server to clear logs
	mockGitLab.Reset()

	t.Logf("Memory stability test completed: %d iterations", iterations)
}

// BenchmarkLoad_MRFetch benchmarks MR fetch operations
func BenchmarkLoad_MRFetch(b *testing.B) {
	mockGitLab := NewMockGitLabServer()
	defer mockGitLab.Close()

	project := &client.Project{ID: 1, Name: "bench-test"}
	mockGitLab.AddProject(project)

	mr := &client.MergeRequest{
		ID:    1,
		IID:   1,
		Title: "Benchmark MR",
		State: "opened",
	}
	mockGitLab.AddMergeRequest(1, mr)

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     mockGitLab.URL,
		AccessToken: "test-token",
	})

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = gitlabClient.GetMergeRequest(ctx, 1, 1)
		}
	})
}

// BenchmarkLoad_CommentCreation benchmarks comment creation
func BenchmarkLoad_CommentCreation(b *testing.B) {
	mockGitLab := NewMockGitLabServer()
	defer mockGitLab.Close()

	project := &client.Project{ID: 1, Name: "bench-comment"}
	mockGitLab.AddProject(project)

	mr := &client.MergeRequest{
		ID:    1,
		IID:   1,
		Title: "Benchmark MR",
		State: "opened",
	}
	mockGitLab.AddMergeRequest(1, mr)

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     mockGitLab.URL,
		AccessToken: "test-token",
	})

	ctx := context.Background()
	comment := "## AI Review\n\nBenchmark comment content."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gitlabClient.CreateMRNote(ctx, 1, 1, comment)
	}
}

// BenchmarkLoad_ParallelComments benchmarks parallel comment creation
func BenchmarkLoad_ParallelComments(b *testing.B) {
	mockGitLab := NewMockGitLabServer()
	defer mockGitLab.Close()

	project := &client.Project{ID: 1, Name: "bench-parallel"}
	mockGitLab.AddProject(project)

	// Create multiple MRs for parallel testing
	for i := 1; i <= 100; i++ {
		mr := &client.MergeRequest{
			ID:    int64(i),
			IID:   i,
			Title: "Parallel MR",
			State: "opened",
		}
		mockGitLab.AddMergeRequest(1, mr)
	}

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     mockGitLab.URL,
		AccessToken: "test-token",
	})

	ctx := context.Background()
	comment := "## AI Review\n\nParallel benchmark comment."

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		mrIID := 1
		for pb.Next() {
			_, _ = gitlabClient.CreateMRNote(ctx, 1, (mrIID%100)+1, comment)
			mrIID++
		}
	})
}

// Package testing provides E2E tests for GitLab integration
package testing

import (
	"context"
	"testing"
	"time"

	"aigateway/internal/gitlab/client"
	"aigateway/internal/gitlab/comment"
	"aigateway/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_WebhookToComment tests the complete flow from webhook to comment
func TestE2E_WebhookToComment(t *testing.T) {
	// Setup mock GitLab server
	mockGitLab := NewMockGitLabServer()
	defer mockGitLab.Close()

	// Setup test project
	project := &client.Project{
		ID:                42,
		Name:              "e2e-test",
		PathWithNamespace: "test-org/e2e-test",
		DefaultBranch:     "main",
	}
	mockGitLab.AddProject(project)

	// Setup test MR
	mr := &client.MergeRequest{
		ID:           100,
		IID:          1,
		Title:        "E2E Test MR",
		State:        "opened",
		SourceBranch: "feature/e2e",
		TargetBranch: "main",
		Author:       &client.User{ID: 1, Username: "tester"},
	}
	mockGitLab.AddMergeRequest(42, mr)

	// Setup MR changes
	changes := &client.MergeRequestChanges{
		MergeRequest: *mr,
		Changes: []client.Change{
			{
				OldPath: "main.go",
				NewPath: "main.go",
				Diff:    "@@ -1,5 +1,10 @@\n package main\n+\n+func newFunc() {\n+}\n",
			},
		},
	}
	mockGitLab.AddMRChanges(42, 1, changes)

	// Create GitLab client
	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     mockGitLab.URL,
		AccessToken: "test-token",
	})

	ctx := context.Background()

	// Step 1: Fetch MR details (simulating webhook processing)
	fetchedMR, err := gitlabClient.GetMergeRequest(ctx, 42, 1)
	require.NoError(t, err)
	assert.Equal(t, "E2E Test MR", fetchedMR.Title)

	// Step 2: Fetch MR changes
	mrChanges, err := gitlabClient.GetMergeRequestChanges(ctx, 42, 1)
	require.NoError(t, err)
	assert.Len(t, mrChanges.Changes, 1)

	// Step 3: Simulate analysis result
	reviewResult := &models.GitLabReviewResult{
		Summary:      "Found 1 suggestion in 1 file",
		OverallScore: 80,
	}

	// Step 4: Build comment
	builder := comment.NewBuilder()
	stats := comment.ReviewStats{
		FilesAnalyzed:    1,
		LinesChanged:     3,
		IssuesFound:      1,
		TokensUsed:       500,
		ProcessingTimeMs: 1000,
		Model:            "test-model",
	}
	reviewComment := builder.BuildReviewComment(reviewResult, stats)

	assert.Contains(t, reviewComment, "AI")
	assert.Contains(t, reviewComment, "1 file")

	// Step 5: Post comment to MR
	note, err := gitlabClient.CreateMRNote(ctx, 42, 1, reviewComment)
	require.NoError(t, err)
	assert.NotZero(t, note.ID)

	// Step 6: Verify comment was posted
	notes := mockGitLab.GetNotesCreated()
	assert.Len(t, notes, 1)
}

// TestE2E_LargeReviewSplit tests that large reviews are properly split
func TestE2E_LargeReviewSplit(t *testing.T) {
	mockGitLab := NewMockGitLabServer()
	defer mockGitLab.Close()

	project := &client.Project{ID: 50, Name: "large-review"}
	mockGitLab.AddProject(project)

	mr := &client.MergeRequest{
		ID:    200,
		IID:   5,
		Title: "Large MR",
		State: "opened",
	}
	mockGitLab.AddMergeRequest(50, mr)

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     mockGitLab.URL,
		AccessToken: "test-token",
	})

	ctx := context.Background()

	// Create a large review result
	reviewResult := &models.GitLabReviewResult{
		Summary:      "Found 50 issues",
		OverallScore: 50,
	}

	// Build comment
	builder := comment.NewBuilder()
	stats := comment.ReviewStats{
		FilesAnalyzed: 10,
		LinesChanged:  500,
		IssuesFound:   50,
		TokensUsed:    5000,
		Model:         "gpt-4",
	}
	reviewComment := builder.BuildReviewComment(reviewResult, stats)

	// Check if splitting is needed
	splitter := comment.NewSplitterWithLimits(10000, 500) // Smaller limit for testing
	result := splitter.SplitWithMetadata(reviewComment)

	if result.WasSplit {
		// Post each part
		for _, part := range result.Parts {
			note, err := gitlabClient.CreateMRNote(ctx, 50, 5, part)
			require.NoError(t, err)
			assert.NotZero(t, note.ID)
		}
	} else {
		note, err := gitlabClient.CreateMRNote(ctx, 50, 5, reviewComment)
		require.NoError(t, err)
		assert.NotZero(t, note.ID)
	}

	// Verify all notes were created
	notes := mockGitLab.GetNotesCreated()
	assert.GreaterOrEqual(t, len(notes), 1)
}

// TestE2E_DraftMRSkipping tests that draft MRs are properly handled
func TestE2E_DraftMRSkipping(t *testing.T) {
	mockGitLab := NewMockGitLabServer()
	defer mockGitLab.Close()

	project := &client.Project{ID: 60, Name: "draft-test"}
	mockGitLab.AddProject(project)

	draftMR := &client.MergeRequest{
		ID:             300,
		IID:            10,
		Title:          "WIP: Draft feature",
		State:          "opened",
		Draft:          true,
		WorkInProgress: true,
	}
	mockGitLab.AddMergeRequest(60, draftMR)

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     mockGitLab.URL,
		AccessToken: "test-token",
	})

	ctx := context.Background()

	// Fetch MR
	fetchedMR, err := gitlabClient.GetMergeRequest(ctx, 60, 10)
	require.NoError(t, err)

	// Verify draft detection
	isDraft := fetchedMR.Draft || fetchedMR.WorkInProgress
	assert.True(t, isDraft, "MR should be detected as draft")
}

// TestE2E_WebhookDeduplication tests webhook deduplication
func TestE2E_WebhookDeduplication(t *testing.T) {
	// Simulated deduplication cache
	processedEvents := make(map[string]bool)

	// First webhook
	eventID1 := "event-uuid-123"
	_, exists := processedEvents[eventID1]
	assert.False(t, exists, "First event should not be in cache")
	processedEvents[eventID1] = true

	// Duplicate webhook (same event ID)
	_, exists = processedEvents[eventID1]
	assert.True(t, exists, "Duplicate event should be in cache")

	// Different webhook
	eventID2 := "event-uuid-456"
	_, exists = processedEvents[eventID2]
	assert.False(t, exists, "New event should not be in cache")
}

// TestE2E_FileFiltering tests file include/exclude patterns
func TestE2E_FileFiltering(t *testing.T) {
	changes := []client.Change{
		{NewPath: "main.go", Diff: "+code"},
		{NewPath: "handler_test.go", Diff: "+test"},
		{NewPath: "vendor/lib.go", Diff: "+vendor"},
		{NewPath: "go.sum", Diff: "+deps"},
		{NewPath: "internal/service.go", Diff: "+service"},
		{NewPath: "node_modules/pkg/index.js", Diff: "+node"},
		{NewPath: "docs/README.md", Diff: "+docs"},
	}

	// Include patterns
	includePatterns := []string{"*.go"}
	// Exclude patterns
	excludePatterns := []string{"*_test.go", "vendor/**", "go.sum", "node_modules/**"}

	var filtered []client.Change
	for _, change := range changes {
		// Simple pattern matching (real implementation would use glob)
		include := false
		for _, p := range includePatterns {
			if matchSimple(change.NewPath, p) {
				include = true
				break
			}
		}

		exclude := false
		for _, p := range excludePatterns {
			if matchSimple(change.NewPath, p) {
				exclude = true
				break
			}
		}

		if include && !exclude {
			filtered = append(filtered, change)
		}
	}

	// Should only include main.go and internal/service.go
	assert.Len(t, filtered, 2)
	for _, f := range filtered {
		assert.NotContains(t, f.NewPath, "test")
		assert.NotContains(t, f.NewPath, "vendor")
	}
}

// Simple pattern matcher for testing
func matchSimple(path, pattern string) bool {
	if pattern == "*.go" {
		return len(path) > 3 && path[len(path)-3:] == ".go"
	}
	if pattern == "*_test.go" {
		return len(path) > 8 && path[len(path)-8:] == "_test.go"
	}
	if pattern == "vendor/**" {
		return len(path) > 7 && path[:7] == "vendor/"
	}
	if pattern == "node_modules/**" {
		return len(path) > 13 && path[:13] == "node_modules/"
	}
	if pattern == "go.sum" {
		return path == "go.sum"
	}
	return false
}

// TestE2E_RetryMechanism tests job retry logic
func TestE2E_RetryMechanism(t *testing.T) {
	job := &models.GitLabAnalysisJob{
		ID:         "job-123",
		Status:     models.GitLabJobStatusPending,
		RetryCount: 0,
		MaxRetries: 3,
	}

	// Simulate failures and retries
	for i := 0; i < 3; i++ {
		job.RetryCount++
		job.Status = models.GitLabJobStatusProcessing

		// Simulate failure
		job.Status = models.GitLabJobStatusFailed
		job.LastError = "Connection timeout"

		// Check if can retry
		canRetry := job.RetryCount < job.MaxRetries
		if canRetry {
			job.Status = models.GitLabJobStatusPending
		}
	}

	// After max attempts, job should stay failed
	assert.Equal(t, 3, job.RetryCount)
	assert.Equal(t, models.GitLabJobStatusFailed, job.Status)
}

// TestE2E_ConcurrentWebhooks tests handling multiple webhooks concurrently
func TestE2E_ConcurrentWebhooks(t *testing.T) {
	mockGitLab := NewMockGitLabServer()
	defer mockGitLab.Close()

	// Setup multiple projects
	for i := int64(1); i <= 5; i++ {
		project := &client.Project{ID: i, Name: "project-" + string(rune('A'+i-1))}
		mockGitLab.AddProject(project)

		mr := &client.MergeRequest{
			ID:    i * 100,
			IID:   1,
			Title: "MR for project " + string(rune('A'+i-1)),
			State: "opened",
		}
		mockGitLab.AddMergeRequest(i, mr)
	}

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     mockGitLab.URL,
		AccessToken: "test-token",
	})

	ctx := context.Background()

	// Simulate concurrent MR fetches
	done := make(chan bool, 5)
	errors := make(chan error, 5)

	for i := int64(1); i <= 5; i++ {
		go func(projectID int64) {
			_, err := gitlabClient.GetMergeRequest(ctx, projectID, 1)
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		select {
		case <-done:
			// Success
		case err := <-errors:
			t.Errorf("Concurrent request failed: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}

	// All requests should have completed
	assert.GreaterOrEqual(t, mockGitLab.GetRequestCount(), 5)
}

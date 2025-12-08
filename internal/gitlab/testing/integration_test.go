// Package testing provides integration tests for GitLab integration
package testing

import (
	"context"
	"testing"
	"time"

	"aigateway/internal/gitlab/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_FullMRReviewFlow tests the complete MR review flow
func TestIntegration_FullMRReviewFlow(t *testing.T) {
	// Setup mock GitLab server
	mockServer := NewMockGitLabServer()
	defer mockServer.Close()

	// Setup test data
	project := &client.Project{
		ID:                123,
		Name:              "test-project",
		PathWithNamespace: "group/test-project",
		DefaultBranch:     "main",
		WebURL:            mockServer.URL + "/group/test-project",
	}
	mockServer.AddProject(project)

	author := &client.User{
		ID:       42,
		Username: "developer",
		Name:     "Test Developer",
	}

	mr := &client.MergeRequest{
		ID:           456,
		IID:          1,
		Title:        "Add new feature",
		Description:  "This MR adds a new feature",
		State:        "opened",
		SourceBranch: "feature/new-thing",
		TargetBranch: "main",
		Author:       author,
		Draft:        false,
		WebURL:       mockServer.URL + "/group/test-project/-/merge_requests/1",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	mockServer.AddMergeRequest(123, mr)

	changes := &client.MergeRequestChanges{
		MergeRequest: *mr,
		Changes: []client.Change{
			{
				OldPath:     "main.go",
				NewPath:     "main.go",
				Diff:        "@@ -10,5 +10,10 @@\n func main() {\n+\t// New code\n+\tfmt.Println(\"Hello\")\n }",
				NewFile:     false,
				DeletedFile: false,
			},
			{
				OldPath:     "",
				NewPath:     "feature.go",
				Diff:        "@@ -0,0 +1,20 @@\n+package main\n+\n+func NewFeature() string {\n+\treturn \"feature\"\n+}",
				NewFile:     true,
				DeletedFile: false,
			},
		},
	}
	mockServer.AddMRChanges(123, 1, changes)

	// Create client
	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     mockServer.URL,
		AccessToken: "test-token",
		Timeout:     10 * time.Second,
	})

	ctx := context.Background()

	// Test 1: Get project
	t.Run("GetProject", func(t *testing.T) {
		proj, err := gitlabClient.GetProject(ctx, 123)
		require.NoError(t, err)
		assert.Equal(t, "test-project", proj.Name)
		assert.Equal(t, "group/test-project", proj.PathWithNamespace)
	})

	// Test 2: Get MR
	t.Run("GetMergeRequest", func(t *testing.T) {
		fetchedMR, err := gitlabClient.GetMergeRequest(ctx, 123, 1)
		require.NoError(t, err)
		assert.Equal(t, "Add new feature", fetchedMR.Title)
		assert.Equal(t, "feature/new-thing", fetchedMR.SourceBranch)
		assert.Equal(t, "developer", fetchedMR.Author.Username)
	})

	// Test 3: Get MR Changes
	t.Run("GetMergeRequestChanges", func(t *testing.T) {
		mrChanges, err := gitlabClient.GetMergeRequestChanges(ctx, 123, 1)
		require.NoError(t, err)
		assert.Len(t, mrChanges.Changes, 2)
		assert.Equal(t, "main.go", mrChanges.Changes[0].NewPath)
		assert.True(t, mrChanges.Changes[1].NewFile)
	})

	// Test 4: Post Review Comment
	t.Run("CreateMRNote", func(t *testing.T) {
		reviewComment := "## AI Code Review\n\n**Summary:** Found 2 issues\n\n1. Consider error handling in main.go"
		
		note, err := gitlabClient.CreateMRNote(ctx, 123, 1, reviewComment)
		require.NoError(t, err)
		assert.Equal(t, reviewComment, note.Body)
		assert.NotZero(t, note.ID)
	})

	// Verify notes were created
	notes := mockServer.GetNotesCreated()
	assert.Len(t, notes, 1)
	assert.Contains(t, notes[0].Body, "AI Code Review")
}

// TestIntegration_WebhookCreation tests webhook setup
func TestIntegration_WebhookCreation(t *testing.T) {
	mockServer := NewMockGitLabServer()
	defer mockServer.Close()

	project := &client.Project{
		ID:   456,
		Name: "webhook-test",
	}
	mockServer.AddProject(project)

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     mockServer.URL,
		AccessToken: "test-token",
	})

	ctx := context.Background()

	// Create webhook
	webhook, err := gitlabClient.CreateWebhook(ctx, 456, &client.CreateWebhookRequest{
		URL:                 "https://aigateway.example.com/webhook/gitlab",
		Token:               "secret123",
		MergeRequestsEvents: true,
	})

	require.NoError(t, err)
	assert.NotZero(t, webhook.ID)
	assert.True(t, webhook.MergeRequestsEvents)

	// Delete webhook
	err = gitlabClient.DeleteWebhook(ctx, 456, webhook.ID)
	require.NoError(t, err)
}

// TestIntegration_RateLimiting tests rate limit handling
func TestIntegration_RateLimiting(t *testing.T) {
	mockServer := NewMockGitLabServer()
	mockServer.RateLimitAfter = 3 // Rate limit after 3 requests
	defer mockServer.Close()

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:         mockServer.URL,
		AccessToken:     "test-token",
		RateLimitPerSec: 100, // High rate to test server-side limiting
	})

	ctx := context.Background()

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		_, err := gitlabClient.GetCurrentUser(ctx)
		require.NoError(t, err)
	}

	// 4th request should get rate limited
	_, err := gitlabClient.GetCurrentUser(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "429")
}

// TestIntegration_NotFoundHandling tests 404 handling
func TestIntegration_NotFoundHandling(t *testing.T) {
	mockServer := NewMockGitLabServer()
	defer mockServer.Close()

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     mockServer.URL,
		AccessToken: "test-token",
	})

	ctx := context.Background()

	// Non-existent project
	_, err := gitlabClient.GetProject(ctx, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")

	// Non-existent MR
	_, err = gitlabClient.GetMergeRequest(ctx, 123, 99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

// TestIntegration_MultipleNotesCreation tests creating multiple review notes
func TestIntegration_MultipleNotesCreation(t *testing.T) {
	mockServer := NewMockGitLabServer()
	defer mockServer.Close()

	project := &client.Project{ID: 789, Name: "multi-note-test"}
	mockServer.AddProject(project)

	mr := &client.MergeRequest{
		ID:    100,
		IID:   5,
		Title: "Large MR",
		State: "opened",
	}
	mockServer.AddMergeRequest(789, mr)

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     mockServer.URL,
		AccessToken: "test-token",
	})

	ctx := context.Background()

	// Create multiple notes (simulating split comments)
	comments := []string{
		"## AI Review Part 1/3\n\nFirst set of issues...",
		"## AI Review Part 2/3\n\nMore issues...",
		"## AI Review Part 3/3\n\nFinal issues...",
	}

	for _, comment := range comments {
		note, err := gitlabClient.CreateMRNote(ctx, 789, 5, comment)
		require.NoError(t, err)
		assert.NotZero(t, note.ID)
	}

	// Verify all notes were created
	notes := mockServer.GetNotesCreated()
	assert.Len(t, notes, 3)
}

// TestIntegration_DraftMRDetection tests draft MR detection
func TestIntegration_DraftMRDetection(t *testing.T) {
	mockServer := NewMockGitLabServer()
	defer mockServer.Close()

	project := &client.Project{ID: 111, Name: "draft-test"}
	mockServer.AddProject(project)

	// Draft MR
	draftMR := &client.MergeRequest{
		ID:             200,
		IID:            10,
		Title:          "WIP: Draft feature",
		State:          "opened",
		Draft:          true,
		WorkInProgress: true,
	}
	mockServer.AddMergeRequest(111, draftMR)

	// Non-draft MR
	normalMR := &client.MergeRequest{
		ID:    201,
		IID:   11,
		Title: "Ready feature",
		State: "opened",
		Draft: false,
	}
	mockServer.AddMergeRequest(111, normalMR)

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     mockServer.URL,
		AccessToken: "test-token",
	})

	ctx := context.Background()

	// Check draft MR
	fetchedDraft, err := gitlabClient.GetMergeRequest(ctx, 111, 10)
	require.NoError(t, err)
	assert.True(t, fetchedDraft.Draft)

	// Check normal MR
	fetchedNormal, err := gitlabClient.GetMergeRequest(ctx, 111, 11)
	require.NoError(t, err)
	assert.False(t, fetchedNormal.Draft)
}

// TestIntegration_FileChangesAnalysis tests analysis of different file types
func TestIntegration_FileChangesAnalysis(t *testing.T) {
	mockServer := NewMockGitLabServer()
	defer mockServer.Close()

	project := &client.Project{ID: 222, Name: "file-analysis"}
	mockServer.AddProject(project)

	mr := &client.MergeRequest{
		ID:    300,
		IID:   15,
		Title: "Multi-file change",
		State: "opened",
	}
	mockServer.AddMergeRequest(222, mr)

	changes := &client.MergeRequestChanges{
		MergeRequest: *mr,
		Changes: []client.Change{
			// Go file - should be analyzed
			{NewPath: "handler.go", Diff: "+func Handler() {}", NewFile: true},
			// TypeScript file - should be analyzed
			{NewPath: "app.ts", Diff: "+export const app = () => {}", NewFile: true},
			// Lock file - should be excluded
			{NewPath: "go.sum", Diff: "+github.com/pkg v1.0.0", NewFile: false},
			// Vendor file - should be excluded
			{NewPath: "vendor/lib.go", Diff: "+package lib", NewFile: false},
			// Deleted file
			{NewPath: "old.go", Diff: "-package old", DeletedFile: true},
		},
	}
	mockServer.AddMRChanges(222, 15, changes)

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     mockServer.URL,
		AccessToken: "test-token",
	})

	ctx := context.Background()

	mrChanges, err := gitlabClient.GetMergeRequestChanges(ctx, 222, 15)
	require.NoError(t, err)
	assert.Len(t, mrChanges.Changes, 5)

	// Count analyzable files (excluding lock, vendor, deleted)
	analyzable := 0
	for _, change := range mrChanges.Changes {
		if !change.DeletedFile &&
			change.NewPath != "go.sum" &&
			!contains(change.NewPath, "vendor/") {
			analyzable++
		}
	}
	assert.Equal(t, 2, analyzable) // handler.go and app.ts
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
		   len(s) > len(substr) && containsInner(s, substr)
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}


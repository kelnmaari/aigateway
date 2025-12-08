// Package storage provides data access for GitLab integration
package storage

import (
	"context"

	"aigateway/internal/models"
)

// Store provides data access for GitLab entities
type Store interface {
	IntegrationStore
	ProjectStore
	ReviewStore
	JobStore
	WebhookEventStore
	ModelUsageStore
}

// IntegrationStore manages GitLab integrations
type IntegrationStore interface {
	// CreateIntegration creates a new GitLab integration
	CreateIntegration(ctx context.Context, integration *models.GitLabIntegration) error
	
	// GetIntegration retrieves an integration by ID
	GetIntegration(ctx context.Context, id string) (*models.GitLabIntegration, error)
	
	// ListIntegrations lists integrations with filtering (admin)
	ListIntegrations(ctx context.Context, req *models.GitLabIntegrationListRequest) ([]models.GitLabIntegration, int, error)
	
	// ListIntegrationsByOwner lists integrations owned by a user
	ListIntegrationsByOwner(ctx context.Context, ownerID string) ([]*models.GitLabIntegration, error)
	
	// UpdateIntegration updates an integration
	UpdateIntegration(ctx context.Context, id string, req *models.UpdateGitLabIntegrationRequest) error
	
	// DeleteIntegration deletes an integration
	DeleteIntegration(ctx context.Context, id string) error
	
	// UpdateIntegrationStatus updates the status and last error
	UpdateIntegrationStatus(ctx context.Context, id string, status models.GitLabIntegrationStatus, lastError string) error
	
	// UpdateIntegrationLastSync updates the last sync timestamp
	UpdateIntegrationLastSync(ctx context.Context, id string) error
}

// ProjectStore manages GitLab projects
type ProjectStore interface {
	// CreateProject creates a new GitLab project
	CreateProject(ctx context.Context, project *models.GitLabProject) error
	
	// GetProject retrieves a project by ID
	GetProject(ctx context.Context, id string) (*models.GitLabProject, error)
	
	// GetProjectByGitLabID retrieves a project by GitLab project ID
	GetProjectByGitLabID(ctx context.Context, integrationID string, gitlabProjectID int64) (*models.GitLabProject, error)
	
	// ListProjects lists projects with filtering
	ListProjects(ctx context.Context, req *models.GitLabProjectListRequest) ([]models.GitLabProject, int, error)
	
	// ListProjectsByIntegration lists projects for a specific integration
	ListProjectsByIntegration(ctx context.Context, integrationID string) ([]*models.GitLabProject, error)
	
	// UpdateProject updates a project
	UpdateProject(ctx context.Context, id string, req *models.UpdateGitLabProjectRequest) error
	
	// DeleteProject deletes a project
	DeleteProject(ctx context.Context, id string) error
	
	// UpdateProjectWebhookID updates the webhook ID for a project
	UpdateProjectWebhookID(ctx context.Context, id string, webhookID int64) error
	
	// DeleteProjectsByIntegration deletes all projects for an integration
	DeleteProjectsByIntegration(ctx context.Context, integrationID string) error
}

// ReviewStore manages MR reviews
type ReviewStore interface {
	// CreateReview creates a new MR review
	CreateReview(ctx context.Context, review *models.GitLabMRReview) error
	
	// GetReview retrieves a review by ID
	GetReview(ctx context.Context, id string) (*models.GitLabMRReview, error)
	
	// GetReviewByMR retrieves a review by project and MR IID
	GetReviewByMR(ctx context.Context, projectID string, mrIID int) (*models.GitLabMRReview, error)
	
	// ListReviewsByIntegration lists reviews for an integration
	ListReviewsByIntegration(ctx context.Context, integrationID string) ([]*models.GitLabMRReview, error)
	
	// ListReviews lists reviews with filtering
	ListReviews(ctx context.Context, req *models.GitLabReviewListRequest) ([]models.GitLabMRReview, int, error)
	
	// UpdateReviewStatus updates review status
	UpdateReviewStatus(ctx context.Context, id string, status models.GitLabReviewStatus, err string) error
	
	// UpdateReviewResult updates the review result
	UpdateReviewResult(ctx context.Context, id string, result *models.GitLabReviewResult, noteID int64, discussionID string) error
	
	// UpdateReviewMetrics updates processing metrics
	UpdateReviewMetrics(ctx context.Context, id string, filesAnalyzed, linesChanged, issuesFound int, processingTimeMs int64, tokensUsed int, model string) error
	
	// IncrementReviewRetry increments the retry count
	IncrementReviewRetry(ctx context.Context, id string) error
}

// JobStore manages analysis jobs
type JobStore interface {
	// CreateJob creates a new analysis job
	CreateJob(ctx context.Context, job *models.GitLabAnalysisJob) error
	
	// GetJob retrieves a job by ID
	GetJob(ctx context.Context, id string) (*models.GitLabAnalysisJob, error)
	
	// GetJobByReview retrieves a job by review ID
	GetJobByReview(ctx context.Context, reviewID string) (*models.GitLabAnalysisJob, error)
	
	// GetNextPendingJob retrieves the next pending job (by priority and age)
	GetNextPendingJob(ctx context.Context) (*models.GitLabAnalysisJob, error)
	
	// ListJobs lists jobs with filtering
	ListJobs(ctx context.Context, status *models.GitLabJobStatus, limit, offset int) ([]models.GitLabAnalysisJob, int, error)
	
	// ClaimJob marks a job as being processed by a worker
	ClaimJob(ctx context.Context, jobID, workerID string) error
	
	// CompleteJob marks a job as completed
	CompleteJob(ctx context.Context, jobID string) error
	
	// FailJob marks a job as failed
	FailJob(ctx context.Context, jobID string, err string) error
	
	// RetryJob schedules a job for retry
	RetryJob(ctx context.Context, jobID string, nextRetryAt interface{}) error
	
	// CancelJob cancels a job
	CancelJob(ctx context.Context, jobID string) error
	
	// GetQueueStats retrieves queue statistics
	GetQueueStats(ctx context.Context) (*models.GitLabQueueStats, error)
	
	// CleanupOldJobs removes completed/failed jobs older than duration
	CleanupOldJobs(ctx context.Context, olderThanDays int) (int64, error)
}

// WebhookEventStore manages webhook events for deduplication
type WebhookEventStore interface {
	// CreateWebhookEvent records a webhook event
	CreateWebhookEvent(ctx context.Context, event *models.GitLabWebhookEvent) error
	
	// GetWebhookEvent retrieves an event by deduplication key
	GetWebhookEvent(ctx context.Context, integrationID string, projectID int64, mrIID int, objectID int64) (*models.GitLabWebhookEvent, error)
	
	// MarkWebhookEventProcessed marks an event as processed
	MarkWebhookEventProcessed(ctx context.Context, id string) error
	
	// MarkWebhookEventDeduplicated marks an event as deduplicated
	MarkWebhookEventDeduplicated(ctx context.Context, id string) error
	
	// CleanupOldEvents removes events older than duration
	CleanupOldEvents(ctx context.Context, olderThanDays int) (int64, error)
}

// ModelUsageStore checks model usage in GitLab projects
type ModelUsageStore interface {
	// FindProjectsByModel finds projects using a specific model
	FindProjectsByModel(ctx context.Context, modelID string) ([]models.GitLabProjectRef, error)
	
	// IsModelUsed checks if a model is used by any project
	IsModelUsed(ctx context.Context, modelID string) (bool, error)
}


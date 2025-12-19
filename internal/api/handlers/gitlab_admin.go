// Package handlers provides HTTP handlers for GitLab admin API
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"aigateway/internal/gitlab/client"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/models"
	mainStorage "aigateway/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GitLabAdminHandler handles GitLab admin API requests
// WorkerPoolStats interface for getting worker pool statistics
type WorkerPoolStats interface {
	GetWorkerCounts() (total, active, idle int)
}

type GitLabAdminHandler struct {
	store      storage.Store
	mainDB     mainStorage.Database // For accessing model registry
	workerPool WorkerPoolStats      // Worker pool for queue stats
	logger     *logrus.Logger
}

// NewGitLabAdminHandler creates a new GitLab admin handler
func NewGitLabAdminHandler(store storage.Store, logger *logrus.Logger) *GitLabAdminHandler {
	return &GitLabAdminHandler{
		store:  store,
		logger: logger,
	}
}

// SetWorkerPool sets the worker pool for queue statistics
func (h *GitLabAdminHandler) SetWorkerPool(pool WorkerPoolStats) {
	h.workerPool = pool
}

// SetMainDB sets the main database for model access
func (h *GitLabAdminHandler) SetMainDB(db mainStorage.Database) {
	h.mainDB = db
}

// ============================================================================
// Integration Handlers
// ============================================================================

// ListIntegrations GET /api/admin/gitlab/integrations
func (h *GitLabAdminHandler) ListIntegrations(c *gin.Context) {
	req := models.GitLabIntegrationListRequest{
		Limit:  20,
		Offset: 0,
	}

	if limit, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil && limit > 0 {
		req.Limit = limit
	}
	if offset, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && offset >= 0 {
		req.Offset = offset
	}
	if status := c.Query("status"); status != "" {
		s := models.GitLabIntegrationStatus(status)
		req.Status = &s
	}
	if search := c.Query("search"); search != "" {
		req.Search = &search
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	integrations, total, err := h.store.ListIntegrations(ctx, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list integrations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list integrations"})
		return
	}

	// Mask access tokens
	for i := range integrations {
		integrations[i].AccessToken = maskToken(integrations[i].AccessToken)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  integrations,
		"total": total,
		"pagination": gin.H{
			"limit":  req.Limit,
			"offset": req.Offset,
			"total":  total,
		},
	})
}

// GetIntegration GET /api/admin/gitlab/integrations/:id
func (h *GitLabAdminHandler) GetIntegration(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	integration, err := h.store.GetIntegration(ctx, id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get integration"})
		return
	}
	if integration == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	// Mask access token
	integration.AccessToken = maskToken(integration.AccessToken)

	c.JSON(http.StatusOK, integration)
}

// CreateIntegration POST /api/admin/gitlab/integrations
func (h *GitLabAdminHandler) CreateIntegration(c *gin.Context) {
	var req struct {
		Name          string `json:"name" binding:"required"`
		BaseURL       string `json:"base_url" binding:"required"`
		AccessToken   string `json:"access_token" binding:"required"`
		WebhookSecret string `json:"webhook_secret,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	integration := &models.GitLabIntegration{
		Name:          req.Name,
		BaseURL:       normalizeURL(req.BaseURL),
		AccessToken:   req.AccessToken,
		WebhookSecret: req.WebhookSecret,
		Status:        models.GitLabIntegrationStatusActive,
		Settings:      models.GitLabIntegrationSettings{},
	}

	// Generate webhook secret if not provided
	if integration.WebhookSecret == "" {
		integration.WebhookSecret = generateWebhookSecret()
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := h.store.CreateIntegration(ctx, integration); err != nil {
		h.logger.WithError(err).Error("Failed to create integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create integration"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"integration_id": integration.ID,
		"name":           integration.Name,
	}).Info("GitLab integration created")

	// Mask tokens in response
	integration.AccessToken = maskToken(integration.AccessToken)

	c.JSON(http.StatusCreated, integration)
}

// UpdateIntegration PUT /api/admin/gitlab/integrations/:id
func (h *GitLabAdminHandler) UpdateIntegration(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateGitLabIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Check if integration exists
	existing, err := h.store.GetIntegration(ctx, id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get integration"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	// Normalize URL if provided
	if req.BaseURL != nil {
		normalized := normalizeURL(*req.BaseURL)
		req.BaseURL = &normalized
	}

	if err := h.store.UpdateIntegration(ctx, id, &req); err != nil {
		h.logger.WithError(err).Error("Failed to update integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update integration"})
		return
	}

	h.logger.WithField("integration_id", id).Info("GitLab integration updated")

	// Return updated integration
	updated, _ := h.store.GetIntegration(ctx, id)
	if updated != nil {
		updated.AccessToken = maskToken(updated.AccessToken)
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteIntegration DELETE /api/admin/gitlab/integrations/:id
func (h *GitLabAdminHandler) DeleteIntegration(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Check if integration exists
	existing, err := h.store.GetIntegration(ctx, id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get integration"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	// Delete projects first (cascade)
	if err := h.store.DeleteProjectsByIntegration(ctx, id); err != nil {
		h.logger.WithError(err).Error("Failed to delete projects")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete projects"})
		return
	}

	if err := h.store.DeleteIntegration(ctx, id); err != nil {
		h.logger.WithError(err).Error("Failed to delete integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete integration"})
		return
	}

	h.logger.WithField("integration_id", id).Info("GitLab integration deleted")

	c.JSON(http.StatusOK, gin.H{"message": "Integration deleted successfully"})
}

// TestIntegration POST /api/admin/gitlab/integrations/:id/test
func (h *GitLabAdminHandler) TestIntegration(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	integration, err := h.store.GetIntegration(ctx, id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get integration"})
		return
	}
	if integration == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	// Test connection
	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     integration.BaseURL,
		AccessToken: integration.AccessToken,
	})

	user, err := gitlabClient.GetCurrentUser(ctx)
	if err != nil {
		h.logger.WithError(err).Warn("GitLab connection test failed")
		
		// Update status to error
		h.store.UpdateIntegrationStatus(ctx, id, models.GitLabIntegrationStatusError, err.Error())
		
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Update status to active and last sync
	h.store.UpdateIntegrationStatus(ctx, id, models.GitLabIntegrationStatusActive, "")
	h.store.UpdateIntegrationLastSync(ctx, id)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"name":     user.Name,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		},
	})
}

// ============================================================================
// Project Handlers
// ============================================================================

// ListProjects GET /api/admin/gitlab/integrations/:id/projects
func (h *GitLabAdminHandler) ListProjects(c *gin.Context) {
	integrationID := c.Param("id")

	req := models.GitLabProjectListRequest{
		IntegrationID: &integrationID,
		Limit:         20,
		Offset:        0,
	}

	if limit, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil && limit > 0 {
		req.Limit = limit
	}
	if offset, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && offset >= 0 {
		req.Offset = offset
	}
	if status := c.Query("status"); status != "" {
		s := models.GitLabProjectStatus(status)
		req.Status = &s
	}
	if search := c.Query("search"); search != "" {
		req.Search = &search
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	projects, total, err := h.store.ListProjects(ctx, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list projects")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  projects,
		"total": total,
		"pagination": gin.H{
			"limit":  req.Limit,
			"offset": req.Offset,
			"total":  total,
		},
	})
}

// GetProject GET /api/admin/gitlab/projects/:project_id
func (h *GitLabAdminHandler) GetProject(c *gin.Context) {
	projectID := c.Param("project_id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project"})
		return
	}
	if project == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	c.JSON(http.StatusOK, project)
}

// AddProject POST /api/admin/gitlab/integrations/:id/projects
func (h *GitLabAdminHandler) AddProject(c *gin.Context) {
	integrationID := c.Param("id")

	var req struct {
		GitLabProjectID  int64  `json:"gitlab_project_id" binding:"required"`
		AnalysisModelID  string `json:"analysis_model_id" binding:"required"`
		EmbeddingModelID string `json:"embedding_model_id"`
		AutoReview       bool   `json:"auto_review"`
		ReviewPrompt     string `json:"review_prompt,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Get integration
	integration, err := h.store.GetIntegration(ctx, integrationID)
	if err != nil || integration == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	// Fetch project info from GitLab
	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     integration.BaseURL,
		AccessToken: integration.AccessToken,
	})

	gitlabProject, err := gitlabClient.GetProject(ctx, req.GitLabProjectID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch GitLab project")
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to fetch GitLab project: %v", err)})
		return
	}

	// Check if project already exists
	existing, _ := h.store.GetProjectByGitLabID(ctx, integrationID, req.GitLabProjectID)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Project already added"})
		return
	}

	project := &models.GitLabProject{
		IntegrationID:     integrationID,
		GitLabProjectID:   req.GitLabProjectID,
		Name:              gitlabProject.Name,
		PathWithNamespace: gitlabProject.PathWithNamespace,
		Status:            models.GitLabProjectStatusActive,
		AutoReview:        req.AutoReview,
		AnalysisModelID:   req.AnalysisModelID,
		EmbeddingModelID:  req.EmbeddingModelID,
		ReviewPrompt:      req.ReviewPrompt,
		Settings:          models.GitLabProjectSettings{},
	}

	if err := h.store.CreateProject(ctx, project); err != nil {
		h.logger.WithError(err).Error("Failed to create project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"project_id": project.ID,
		"name":       project.Name,
	}).Info("GitLab project added")

	c.JSON(http.StatusCreated, project)
}

// UpdateProject PUT /api/admin/gitlab/projects/:project_id
func (h *GitLabAdminHandler) UpdateProject(c *gin.Context) {
	projectID := c.Param("project_id")

	var req models.UpdateGitLabProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Check if project exists
	existing, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	if err := h.store.UpdateProject(ctx, projectID, &req); err != nil {
		h.logger.WithError(err).Error("Failed to update project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	h.logger.WithField("project_id", projectID).Info("GitLab project updated")

	// Return updated project
	updated, _ := h.store.GetProject(ctx, projectID)
	c.JSON(http.StatusOK, updated)
}

// DeleteProject DELETE /api/admin/gitlab/projects/:project_id
func (h *GitLabAdminHandler) DeleteProject(c *gin.Context) {
	projectID := c.Param("project_id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Check if project exists
	existing, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	if err := h.store.DeleteProject(ctx, projectID); err != nil {
		h.logger.WithError(err).Error("Failed to delete project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	h.logger.WithField("project_id", projectID).Info("GitLab project deleted")

	c.JSON(http.StatusOK, gin.H{"message": "Project deleted successfully"})
}

// SetupWebhook POST /api/admin/gitlab/projects/:project_id/webhook
func (h *GitLabAdminHandler) SetupWebhook(c *gin.Context) {
	projectID := c.Param("project_id")

	var req struct {
		WebhookURL string `json:"webhook_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Get project
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil || project == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Get integration
	integration, err := h.store.GetIntegration(ctx, project.IntegrationID)
	if err != nil || integration == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	// Create webhook on GitLab
	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     integration.BaseURL,
		AccessToken: integration.AccessToken,
	})

	webhook, err := gitlabClient.CreateProjectWebhook(ctx, project.GitLabProjectID, req.WebhookURL, integration.WebhookSecret)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create GitLab webhook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create webhook: %v", err)})
		return
	}

	// Save webhook ID
	if err := h.store.UpdateProjectWebhookID(ctx, projectID, webhook.ID); err != nil {
		h.logger.WithError(err).Error("Failed to save webhook ID")
	}

	h.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"webhook_id": webhook.ID,
	}).Info("GitLab webhook created")

	c.JSON(http.StatusOK, gin.H{
		"message":    "Webhook created successfully",
		"webhook_id": webhook.ID,
	})
}

// ============================================================================
// Review Handlers
// ============================================================================

// ListReviews GET /api/admin/gitlab/reviews
func (h *GitLabAdminHandler) ListReviews(c *gin.Context) {
	req := models.GitLabReviewListRequest{
		Limit:  20,
		Offset: 0,
	}

	if limit, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil && limit > 0 {
		req.Limit = limit
	}
	if offset, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && offset >= 0 {
		req.Offset = offset
	}
	if projectID := c.Query("project_id"); projectID != "" {
		req.ProjectID = &projectID
	}
	if status := c.Query("status"); status != "" {
		s := models.GitLabReviewStatus(status)
		req.Status = &s
	}
	if search := c.Query("search"); search != "" {
		req.Search = &search
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	reviews, total, err := h.store.ListReviews(ctx, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list reviews")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list reviews"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  reviews,
		"total": total,
		"pagination": gin.H{
			"limit":  req.Limit,
			"offset": req.Offset,
			"total":  total,
		},
	})
}

// GetReview GET /api/admin/gitlab/reviews/:id
func (h *GitLabAdminHandler) GetReview(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	review, err := h.store.GetReview(ctx, id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get review")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get review"})
		return
	}
	if review == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Review not found"})
		return
	}

	c.JSON(http.StatusOK, review)
}

// RetryReview POST /api/admin/gitlab/reviews/:id/retry
func (h *GitLabAdminHandler) RetryReview(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	review, err := h.store.GetReview(ctx, id)
	if err != nil || review == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Review not found"})
		return
	}

	// Reset status to pending
	if err := h.store.UpdateReviewStatus(ctx, id, models.GitLabReviewStatusPending, ""); err != nil {
		h.logger.WithError(err).Error("Failed to retry review")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retry review"})
		return
	}

	// Create new job
	job := &models.GitLabAnalysisJob{
		ReviewID:      id,
		IntegrationID: review.IntegrationID,
		ProjectID:     review.ProjectID,
		MRIID:         review.MRIID,
		MRTitle:       review.MRTitle,
		Status:        models.GitLabJobStatusPending,
		Priority:      models.GitLabReviewPriorityNormal,
		MaxRetries:    3,
	}

	if err := h.store.CreateJob(ctx, job); err != nil {
		h.logger.WithError(err).Error("Failed to create retry job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create retry job"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"review_id": id,
		"job_id":    job.ID,
	}).Info("Review retry scheduled")

	c.JSON(http.StatusOK, gin.H{
		"message": "Review retry scheduled",
		"job_id":  job.ID,
	})
}

// ============================================================================
// Queue Handlers
// ============================================================================

// GetQueueStatus GET /api/admin/gitlab/queue/status
func (h *GitLabAdminHandler) GetQueueStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	stats, err := h.store.GetQueueStats(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get queue stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get queue stats"})
		return
	}

	// Enrich with worker pool stats if available
	if h.workerPool != nil {
		total, active, idle := h.workerPool.GetWorkerCounts()
		stats.TotalWorkers = total
		stats.ActiveWorkers = active
		stats.IdleWorkers = idle
	}

	c.JSON(http.StatusOK, stats)
}

// ListJobs GET /api/admin/gitlab/queue/jobs
func (h *GitLabAdminHandler) ListJobs(c *gin.Context) {
	var status *models.GitLabJobStatus
	if s := c.Query("status"); s != "" {
		st := models.GitLabJobStatus(s)
		status = &st
	}

	limit := 20
	offset := 0
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil && l > 0 {
		limit = l
	}
	if o, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && o >= 0 {
		offset = o
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	jobs, total, err := h.store.ListJobs(ctx, status, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list jobs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list jobs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  jobs,
		"total": total,
		"pagination": gin.H{
			"limit":  limit,
			"offset": offset,
			"total":  total,
		},
	})
}

// CancelJob POST /api/admin/gitlab/queue/jobs/:id/cancel
func (h *GitLabAdminHandler) CancelJob(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := h.store.CancelJob(ctx, id); err != nil {
		h.logger.WithError(err).Error("Failed to cancel job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel job"})
		return
	}

	h.logger.WithField("job_id", id).Info("Job cancelled")

	c.JSON(http.StatusOK, gin.H{"message": "Job cancelled successfully"})
}

// RetryJob POST /api/admin/gitlab/queue/jobs/:id/retry
func (h *GitLabAdminHandler) RetryJob(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := h.store.RetryJob(ctx, id, nil); err != nil {
		h.logger.WithError(err).Error("Failed to retry job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retry job"})
		return
	}

	h.logger.WithField("job_id", id).Info("Job retry scheduled")

	c.JSON(http.StatusOK, gin.H{"message": "Job retry scheduled"})
}

// ============================================================================
// Model Usage Handlers
// ============================================================================

// GetModelUsage GET /api/admin/models/:id/usage
func (h *GitLabAdminHandler) GetModelUsage(c *gin.Context) {
	modelID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	projects, err := h.store.FindProjectsByModel(ctx, modelID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to find projects by model")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get model usage"})
		return
	}

	isUsed, _ := h.store.IsModelUsed(ctx, modelID)

	c.JSON(http.StatusOK, gin.H{
		"model_id":           modelID,
		"is_used_in_gitlab":  isUsed,
		"gitlab_projects":    projects,
		"can_deactivate":     !isUsed,
		"blocking_reason":    getBlockingReason(projects),
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func maskToken(token string) string {
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

func normalizeURL(url string) string {
	// Remove trailing slash
	for len(url) > 0 && url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}
	return url
}

func generateWebhookSecret() string {
	// Generate random 32-byte secret
	b := make([]byte, 32)
	for i := range b {
		b[i] = byte(i + 65) // Simple placeholder, use crypto/rand in production
	}
	return fmt.Sprintf("%x", b)
}

func getBlockingReason(projects []models.GitLabProjectRef) string {
	if len(projects) == 0 {
		return ""
	}
	return fmt.Sprintf("Model is used in %d GitLab project(s). Change model in these projects before deactivating.", len(projects))
}

// SerializeJSON serializes data to JSON string
func SerializeJSON(data interface{}) string {
	b, _ := json.Marshal(data)
	return string(b)
}

// ============================================================================
// Model Selection Handlers
// ============================================================================

// ListActiveModels GET /api/admin/gitlab/models
// Returns list of active models available for selection in GitLab projects
func (h *GitLabAdminHandler) ListActiveModels(c *gin.Context) {
	if h.mainDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Model database not configured"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Get capability filter from query param
	capability := c.Query("capability")
	
	// Build filter for active models
	filter := &models.ModelRegistryFilter{
		Status: models.ModelStatusActive,
	}
	
	if capability != "" {
		filter.Capabilities = []models.ModelCapability{models.ModelCapability(capability)}
	}

	modelsList, err := h.mainDB.ListModelRegistry(ctx, filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list active models")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list models"})
		return
	}

	// Convert to simplified response
	type ModelOption struct {
		ID           string                   `json:"id"`
		ModelID      string                   `json:"model_id"`
		Name         string                   `json:"name"`
		ProviderID   string                   `json:"provider_id"`
		Capabilities []models.ModelCapability `json:"capabilities"`
		Description  string                   `json:"description,omitempty"`
	}

	options := make([]ModelOption, 0, len(modelsList))
	for _, m := range modelsList {
		options = append(options, ModelOption{
			ID:           m.ID,
			ModelID:      m.ModelID,
			Name:         m.ModelName,
			ProviderID:   m.ProviderID,
			Capabilities: m.Capabilities,
			Description:  m.Description,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"models": options,
		"total":  len(options),
	})
}

// ListAnalysisModels GET /api/admin/gitlab/models/analysis
// Returns models suitable for code analysis (chat capability)
func (h *GitLabAdminHandler) ListAnalysisModels(c *gin.Context) {
	if h.mainDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Model database not configured"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	filter := &models.ModelRegistryFilter{
		Status:       models.ModelStatusActive,
		Capabilities: []models.ModelCapability{models.CapabilityChat},
	}

	modelsList, err := h.mainDB.ListModelRegistry(ctx, filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list analysis models")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list models"})
		return
	}

	type ModelOption struct {
		ID          string `json:"id"`
		ModelID     string `json:"model_id"`
		Name        string `json:"name"`
		ProviderID  string `json:"provider_id"`
		Description string `json:"description,omitempty"`
	}

	options := make([]ModelOption, 0, len(modelsList))
	for _, m := range modelsList {
		options = append(options, ModelOption{
			ID:          m.ID,
			ModelID:     m.ModelID,
			Name:        m.ModelName,
			ProviderID:  m.ProviderID,
			Description: m.Description,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"models": options,
		"total":  len(options),
	})
}

// ListEmbeddingModels GET /api/admin/gitlab/models/embedding
// Returns models suitable for embeddings
func (h *GitLabAdminHandler) ListEmbeddingModels(c *gin.Context) {
	if h.mainDB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Model database not configured"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	filter := &models.ModelRegistryFilter{
		Status:       models.ModelStatusActive,
		Capabilities: []models.ModelCapability{models.CapabilityEmbeddings},
	}

	modelsList, err := h.mainDB.ListModelRegistry(ctx, filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list embedding models")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list models"})
		return
	}

	type ModelOption struct {
		ID          string `json:"id"`
		ModelID     string `json:"model_id"`
		Name        string `json:"name"`
		ProviderID  string `json:"provider_id"`
		Description string `json:"description,omitempty"`
	}

	options := make([]ModelOption, 0, len(modelsList))
	for _, m := range modelsList {
		options = append(options, ModelOption{
			ID:          m.ID,
			ModelID:     m.ModelID,
			Name:        m.ModelName,
			ProviderID:  m.ProviderID,
			Description: m.Description,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"models": options,
		"total":  len(options),
	})
}

// ============================================================================
// Settings Handlers
// ============================================================================

// GetSettings GET /api/admin/gitlab/settings
func (h *GitLabAdminHandler) GetSettings(c *gin.Context) {
	// Return default settings for now - can be extended to store in DB
	c.JSON(http.StatusOK, gin.H{
		"auto_review_enabled":     true,
		"default_analysis_model":  "",
		"default_embedding_model": "",
		"max_files_per_mr":        50,
		"max_lines_per_file":      1000,
		"webhook_secret_rotation": false,
		"notification_email":      "",
	})
}

// UpdateSettings PUT /api/admin/gitlab/settings
func (h *GitLabAdminHandler) UpdateSettings(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.WithField("settings", req).Info("GitLab settings update requested")

	// TODO: Persist settings to database
	c.JSON(http.StatusOK, gin.H{"message": "Settings updated", "settings": req})
}

// ============================================================================
// Analytics Handlers
// ============================================================================

// GetAnalytics GET /api/admin/gitlab/analytics
func (h *GitLabAdminHandler) GetAnalytics(c *gin.Context) {
	rangeParam := c.DefaultQuery("range", "7d")

	// Calculate date range
	var days int
	switch rangeParam {
	case "24h":
		days = 1
	case "7d":
		days = 7
	case "30d":
		days = 30
	case "90d":
		days = 90
	default:
		days = 7
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	analytics, err := h.store.GetAnalytics(ctx, days)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get analytics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get analytics"})
		return
	}

	analytics.Range = rangeParam
	c.JSON(http.StatusOK, analytics)
}

// ============================================================================
// Feedback Handlers
// ============================================================================

// ListFeedback GET /api/admin/gitlab/feedback
func (h *GitLabAdminHandler) ListFeedback(c *gin.Context) {
	req := models.GitLabFeedbackListRequest{
		Limit:  20,
		Offset: 0,
	}

	if limit, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil && limit > 0 {
		req.Limit = limit
	}
	if offset, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && offset >= 0 {
		req.Offset = offset
	}
	if reviewID := c.Query("review_id"); reviewID != "" {
		req.ReviewID = &reviewID
	}
	if feedbackType := c.Query("type"); feedbackType != "" {
		req.FeedbackType = &feedbackType
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	feedback, total, err := h.store.ListFeedback(ctx, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list feedback")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list feedback"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  feedback,
		"total": total,
		"pagination": gin.H{
			"limit":  req.Limit,
			"offset": req.Offset,
			"total":  total,
		},
	})
}

// SubmitFeedback POST /api/admin/gitlab/feedback
func (h *GitLabAdminHandler) SubmitFeedback(c *gin.Context) {
	var req struct {
		ReviewID     string  `json:"review_id" binding:"required"`
		Rating       int     `json:"rating" binding:"required,min=1,max=5"`
		FeedbackType string  `json:"feedback_type"`
		Comment      string  `json:"comment"`
		IssueIndex   *int    `json:"issue_index"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default feedback type
	if req.FeedbackType == "" {
		req.FeedbackType = models.FeedbackTypeGeneral
	}

	// Get user ID from context if available
	var userID *string
	if uid, exists := c.Get("user_id"); exists {
		if id, ok := uid.(string); ok {
			userID = &id
		}
	}

	feedback := &models.GitLabReviewFeedback{
		ReviewID:     req.ReviewID,
		UserID:       userID,
		Rating:       req.Rating,
		FeedbackType: req.FeedbackType,
		Comment:      req.Comment,
		IssueIndex:   req.IssueIndex,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	if err := h.store.CreateFeedback(ctx, feedback); err != nil {
		h.logger.WithError(err).Error("Failed to submit feedback")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit feedback"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"feedback_id": feedback.ID,
		"review_id":   req.ReviewID,
		"rating":      req.Rating,
	}).Info("Feedback submitted for review")

	c.JSON(http.StatusCreated, gin.H{
		"message": "Feedback submitted successfully",
		"id":      feedback.ID,
	})
}

// ============================================================================
// Available Projects (GitLab API Integration)
// ============================================================================

// ListAvailableProjects GET /api/admin/gitlab/integrations/:id/available-projects
// Fetches projects from GitLab that the integration has access to
func (h *GitLabAdminHandler) ListAvailableProjects(c *gin.Context) {
	integrationID := c.Param("id")
	search := c.Query("search")
	perPage := 20
	page := 1

	if p, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && p > 0 {
		page = p
	}
	if pp, err := strconv.Atoi(c.DefaultQuery("per_page", "20")); err == nil && pp > 0 && pp <= 100 {
		perPage = pp
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Get integration to access GitLab API
	integration, err := h.store.GetIntegration(ctx, integrationID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get integration"})
		return
	}
	if integration == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	// Create GitLab client
	glClient := client.NewClient(client.ClientConfig{
		BaseURL:     integration.BaseURL,
		AccessToken: integration.AccessToken,
	})

	// Fetch projects from GitLab
	projects, err := glClient.ListProjects(ctx, &client.ListProjectsOptions{
		Search:     search,
		Page:       page,
		PerPage:    perPage,
		Membership: true, // Only show projects user has access to
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to list GitLab projects")
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch projects from GitLab: %v", err)})
		return
	}

	// Get already added project IDs to mark them
	existingProjects, _, err := h.store.ListProjects(ctx, &models.GitLabProjectListRequest{
		IntegrationID: &integrationID,
		Limit:         1000,
	})
	if err != nil {
		h.logger.WithError(err).Warn("Failed to list existing projects")
		existingProjects = nil
	}

	existingIDs := make(map[int64]bool)
	for _, p := range existingProjects {
		existingIDs[int64(p.GitLabProjectID)] = true
	}

	// Format response
	type ProjectInfo struct {
		ID                int64  `json:"id"`
		Name              string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		Description       string `json:"description,omitempty"`
		WebURL            string `json:"web_url"`
		DefaultBranch     string `json:"default_branch"`
		Visibility        string `json:"visibility"`
		AlreadyAdded      bool   `json:"already_added"`
	}

	result := make([]ProjectInfo, 0, len(projects))
	for _, p := range projects {
		result = append(result, ProjectInfo{
			ID:                p.ID,
			Name:              p.Name,
			PathWithNamespace: p.PathWithNamespace,
			Description:       p.Description,
			WebURL:            p.WebURL,
			DefaultBranch:     p.DefaultBranch,
			Visibility:        p.Visibility,
			AlreadyAdded:      existingIDs[p.ID],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"projects": result,
		"total":    len(result),
		"page":     page,
		"per_page": perPage,
	})
}

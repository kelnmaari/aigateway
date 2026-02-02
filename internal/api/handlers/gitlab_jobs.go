// Package handlers provides HTTP handlers for user background jobs
package handlers

import (
	"fmt"
	"net/http"

	"aigateway/internal/gitlab/jobs"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GitLabJobsHandler handles user job API requests
type GitLabJobsHandler struct {
	store      storage.Store
	jobService *jobs.Service
	logger     *logrus.Logger
}

// NewGitLabJobsHandler creates a new jobs handler
func NewGitLabJobsHandler(store storage.Store, jobService *jobs.Service, logger *logrus.Logger) *GitLabJobsHandler {
	return &GitLabJobsHandler{
		store:      store,
		jobService: jobService,
		logger:     logger,
	}
}

// ListUserJobs GET /api/gitlab/jobs
func (h *GitLabJobsHandler) ListUserJobs(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse query params
	var req models.UserJobsRequest
	req.UserID = userID

	if projectID := c.Query("project_id"); projectID != "" {
		req.ProjectID = projectID
	}

	if integrationID := c.Query("integration_id"); integrationID != "" {
		req.IntegrationID = integrationID
	}

	if jobType := c.Query("job_type"); jobType != "" {
		jt := models.UserJobType(jobType)
		req.JobType = &jt
	}

	if status := c.Query("status"); status != "" {
		st := models.UserJobStatus(status)
		req.Status = &st
	}

	// Pagination
	req.Limit = 50
	req.Offset = 0
	if limit := c.Query("limit"); limit != "" {
		var l int
		if _, err := parseIntParam(limit, &l); err == nil && l > 0 && l <= 100 {
			req.Limit = l
		}
	}
	if offset := c.Query("offset"); offset != "" {
		var o int
		if _, err := parseIntParam(offset, &o); err == nil && o >= 0 {
			req.Offset = o
		}
	}

	jobs, total, err := h.store.ListUserJobs(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list user jobs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list jobs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":   jobs,
		"total":  total,
		"limit":  req.Limit,
		"offset": req.Offset,
	})
}

// GetUserJob GET /api/gitlab/jobs/:id
func (h *GitLabJobsHandler) GetUserJob(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	job, err := h.store.GetUserJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Check ownership
	if job.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// GetActiveUserJobs GET /api/gitlab/jobs/active
func (h *GitLabJobsHandler) GetActiveUserJobs(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	jobs, err := h.store.GetActiveUserJobs(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get active jobs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get active jobs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

// CancelUserJob POST /api/gitlab/jobs/:id/cancel
func (h *GitLabJobsHandler) CancelUserJob(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	// Get job to check ownership
	job, err := h.store.GetUserJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Check ownership
	if job.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Cancel the job
	if err := h.jobService.CancelJob(c.Request.Context(), jobID); err != nil {
		h.logger.WithError(err).Error("Failed to cancel job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job cancelled"})
}

// GetJobTypes GET /api/gitlab/jobs/types
func (h *GitLabJobsHandler) GetJobTypes(c *gin.Context) {
	types := []map[string]string{
		{"type": string(models.JobTypeSecretsScn), "name": "Secrets Scan", "description": "Regex-based secrets detection"},
		{"type": string(models.JobTypeDeepSecretsScan), "name": "Deep Secrets Scan", "description": "LLM-powered semantic secrets detection"},
		{"type": string(models.JobTypeSASTScan), "name": "SAST Scan", "description": "Static Application Security Testing"},
		{"type": string(models.JobTypeDependencyScan), "name": "Dependency Scan", "description": "Check for vulnerable dependencies"},
		{"type": string(models.JobTypeQualityScan), "name": "Quality Scan", "description": "Code quality analysis"},
		{"type": string(models.JobTypeDeadCodeScan), "name": "Dead Code Scan", "description": "Detect unused and dead code"},
		{"type": string(models.JobTypeAutoDocsScan), "name": "Auto-Docs Scan", "description": "Find undocumented code"},
		{"type": string(models.JobTypeTestGenScan), "name": "Test Generation Scan", "description": "Find testable functions"},
		{"type": string(models.JobTypeProjectIndex), "name": "Project Index", "description": "Index project for RAG"},
		{"type": string(models.JobTypeGenerateDocs), "name": "Generate Docs", "description": "Auto-generate documentation"},
		{"type": string(models.JobTypeGenerateTests), "name": "Generate Tests", "description": "Auto-generate tests"},
		{"type": string(models.JobTypeCreateMR), "name": "Create MR", "description": "Create merge request with changes"},
	}

	c.JSON(http.StatusOK, gin.H{"types": types})
}

// GetJobStatuses GET /api/gitlab/jobs/statuses
func (h *GitLabJobsHandler) GetJobStatuses(c *gin.Context) {
	statuses := []map[string]string{
		{"status": string(models.UserJobStatusPending), "name": "Pending", "description": "Waiting to start"},
		{"status": string(models.UserJobStatusRunning), "name": "Running", "description": "Currently executing"},
		{"status": string(models.UserJobStatusCompleted), "name": "Completed", "description": "Finished successfully"},
		{"status": string(models.UserJobStatusFailed), "name": "Failed", "description": "Finished with error"},
		{"status": string(models.UserJobStatusCancelled), "name": "Cancelled", "description": "Cancelled by user"},
	}

	c.JSON(http.StatusOK, gin.H{"statuses": statuses})
}

// Helper to parse int from query param
func parseIntParam(s string, v *int) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err == nil {
		*v = n
	}
	return n, err
}

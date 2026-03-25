// Package handlers provides HTTP handlers for user background jobs
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

// getUserID extracts user ID from context (set by auth middleware)
// Strips "user_" prefix if present since JWT claims include it but DB stores raw UUID
func (h *GitLabJobsHandler) getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok {
			// JWT claims store user_id with "user_" prefix, but DB expects raw UUID
			if len(id) > 5 && id[:5] == "user_" {
				return id[5:]
			}
			return id
		}
	}
	return ""
}

// ListUserJobs GET /api/gitlab/jobs
func (h *GitLabJobsHandler) ListUserJobs(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	h.logger.WithField("user_id", userID).Debug("Listing jobs for user")

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

	h.logger.WithFields(logrus.Fields{
		"user_id":        req.UserID,
		"integration_id": req.IntegrationID,
		"project_id":     req.ProjectID,
	}).Debug("ListUserJobs query params")

	jobs, total, err := h.store.ListUserJobs(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list user jobs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list jobs"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"jobs_count": len(jobs),
		"total":      total,
	}).Debug("ListUserJobs results")

	// Ensure jobs is never null in JSON response
	if jobs == nil {
		jobs = []models.UserJob{}
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
	userID := h.getUserID(c)
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
	userID := h.getUserID(c)
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
	userID := h.getUserID(c)
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

// StreamJobUpdates GET /api/gitlab/jobs/:id/stream
// Server-Sent Events endpoint for real-time job progress updates
func (h *GitLabJobsHandler) StreamJobUpdates(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	jobID := c.Param("id")
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
		return
	}

	// Get initial job to check ownership
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

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Get flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	// Send initial state
	h.sendJobSSEEvent(c.Writer, flusher, "init", job)

	// Poll for updates until job completes or client disconnects
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Heartbeat to keep connection alive (prevents proxy timeouts)
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	ctx := c.Request.Context()
	lastProgress := job.Progress
	lastStatus := job.Status

	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			h.logger.WithField("job_id", jobID).Debug("SSE client disconnected")
			return

		case <-heartbeat.C:
			// Send heartbeat comment to keep connection alive
			fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			flusher.Flush()

		case <-ticker.C:
			// Fetch latest job state
			job, err = h.store.GetUserJob(ctx, jobID)
			if err != nil {
				h.sendJobSSEEvent(c.Writer, flusher, "error", map[string]string{"error": "Job not found"})
				return
			}

			// Send update if changed
			if job.Progress != lastProgress || job.Status != lastStatus {
				h.sendJobSSEEvent(c.Writer, flusher, "progress", map[string]any{
					"job_id":       job.ID,
					"status":       job.Status,
					"progress":     job.Progress,
					"progress_msg": job.ProgressMsg,
					"result_id":    job.ResultID,
					"error":        job.Error,
				})
				lastProgress = job.Progress
				lastStatus = job.Status
			}

			// Check if job completed
			if job.Status == models.UserJobStatusCompleted ||
				job.Status == models.UserJobStatusFailed ||
				job.Status == models.UserJobStatusCancelled {
				h.sendJobSSEEvent(c.Writer, flusher, "complete", job)
				h.sendJobSSEEvent(c.Writer, flusher, "done", nil)
				return
			}
		}
	}
}

// sendJobSSEEvent sends a Server-Sent Event
func (h *GitLabJobsHandler) sendJobSSEEvent(w http.ResponseWriter, flusher http.Flusher, event string, data any) {
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			h.logger.WithError(err).Error("Failed to marshal SSE data")
			return
		}
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(jsonData))
	} else {
		fmt.Fprintf(w, "event: %s\ndata: {}\n\n", event)
	}
	flusher.Flush()
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

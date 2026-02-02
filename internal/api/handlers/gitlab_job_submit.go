// Package handlers provides HTTP handlers for submitting background jobs
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"aigateway/internal/gitlab/jobs"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// GitLabJobSubmitHandler handles job submission requests
type GitLabJobSubmitHandler struct {
	store      storage.Store
	jobService *jobs.Service
	logger     *logrus.Logger
}

// NewGitLabJobSubmitHandler creates a new job submit handler
func NewGitLabJobSubmitHandler(store storage.Store, jobService *jobs.Service, logger *logrus.Logger) *GitLabJobSubmitHandler {
	return &GitLabJobSubmitHandler{
		store:      store,
		jobService: jobService,
		logger:     logger,
	}
}

// SubmitJobRequest is the request body for submitting a job
type SubmitJobRequest struct {
	ModelID     string   `json:"model_id,omitempty"`
	Language    string   `json:"language,omitempty"`
	MaxChunks   int      `json:"max_chunks,omitempty"`
	Categories  []string `json:"categories,omitempty"`
	MinSeverity string   `json:"min_severity,omitempty"`
}

// SubmitJobResponse is the response for job submission
type SubmitJobResponse struct {
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	ProjectID string `json:"project_id"`
}

// getUserID extracts user ID from context
// Strips "user_" prefix if present since JWT claims include it but DB stores raw UUID
func (h *GitLabJobSubmitHandler) getUserID(c *gin.Context) string {
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

// canAccessProject checks if user can access the project (via integration ownership)
func (h *GitLabJobSubmitHandler) canAccessProject(c *gin.Context, project *models.GitLabProject) bool {
	userID := h.getUserID(c)
	if userID == "" {
		return false
	}

	// Admins can access everything
	if isAdmin, exists := c.Get("is_admin"); exists {
		if a, ok := isAdmin.(bool); ok && a {
			return true
		}
	}

	// Get integration to check ownership
	ctx := c.Request.Context()
	integration, err := h.store.GetIntegration(ctx, project.IntegrationID)
	if err != nil || integration == nil {
		return false
	}

	// Check ownership
	if integration.OwnerID == userID {
		return true
	}

	// Check tenant membership
	if tids, exists := c.Get("tenant_ids"); exists {
		if ids, ok := tids.([]string); ok {
			for _, tid := range ids {
				if integration.TenantID == tid {
					return true
				}
			}
		}
	}

	return false
}

// MaxActiveJobsPerUser is the maximum number of concurrent jobs per user
const MaxActiveJobsPerUser = 5

// submitJob is a helper that creates and submits a job
func (h *GitLabJobSubmitHandler) submitJob(c *gin.Context, jobType models.UserJobType, req SubmitJobRequest) {
	userID := h.getUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	ctx := c.Request.Context()

	// Rate limiting: check concurrent jobs limit
	activeJobs, err := h.store.GetActiveUserJobs(ctx, userID)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to check active jobs count")
		// Continue anyway - don't block on rate limit check failure
	} else if len(activeJobs) >= MaxActiveJobsPerUser {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":       "Too many active jobs. Please wait for existing jobs to complete.",
			"active_jobs": len(activeJobs),
			"max_allowed": MaxActiveJobsPerUser,
		})
		return
	}

	// Get project
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Check access
	if !h.canAccessProject(c, project) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Check if project is indexed
	if project.GetCollectionName() == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project has no indexed collection. Please index the repository first."})
		return
	}

	// Build job config
	config := models.UserJobConfig{
		ModelID:     req.ModelID,
		Language:    req.Language,
		MaxChunks:   req.MaxChunks,
		Categories:  req.Categories,
		MinSeverity: req.MinSeverity,
	}

	// Use project defaults if not specified
	if config.ModelID == "" {
		config.ModelID = project.AnalysisModelID
	}
	if config.Language == "" {
		config.Language = project.Settings.ReviewLanguage
		if config.Language == "" {
			config.Language = "en"
		}
	}

	configJSON, _ := json.Marshal(config)

	// Create job
	job := &models.UserJob{
		ID:            uuid.New().String(),
		UserID:        userID,
		TenantID:      project.TenantID,
		ProjectID:     projectID,
		IntegrationID: project.IntegrationID,
		JobType:       jobType,
		Status:        models.UserJobStatusPending,
		Config:        string(configJSON),
		Progress:      0,
		ProgressMsg:   "Queued for processing",
		CreatedAt:     time.Now(),
	}

	// Submit to job service
	if err := h.jobService.SubmitJob(ctx, job); err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"project_id": projectID,
			"job_type":   jobType,
			"user_id":    userID,
		}).Error("Failed to submit job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit job: " + err.Error()})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"job_id":     job.ID,
		"project_id": projectID,
		"job_type":   jobType,
		"user_id":    userID,
	}).Info("Job submitted successfully")

	c.JSON(http.StatusAccepted, SubmitJobResponse{
		JobID:     job.ID,
		Status:    string(job.Status),
		Message:   "Job submitted successfully",
		ProjectID: projectID,
	})
}

// SubmitDeepScanJob POST /api/gitlab/projects/:id/jobs/deep-scan
// Submits a deep secrets scan job for background processing
func (h *GitLabJobSubmitHandler) SubmitDeepScanJob(c *gin.Context) {
	var req SubmitJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = SubmitJobRequest{}
	}

	h.submitJob(c, models.JobTypeDeepSecretsScan, req)
}

// SubmitSecretsScanJob POST /api/gitlab/projects/:id/jobs/secrets-scan
// Submits a regex-based secrets scan job for background processing
func (h *GitLabJobSubmitHandler) SubmitSecretsScanJob(c *gin.Context) {
	var req SubmitJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = SubmitJobRequest{}
	}

	h.submitJob(c, models.JobTypeSecretsScn, req)
}

// SubmitSASTJob POST /api/gitlab/projects/:id/jobs/sast-scan
// Submits a SAST scan job for background processing
func (h *GitLabJobSubmitHandler) SubmitSASTJob(c *gin.Context) {
	var req SubmitJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = SubmitJobRequest{}
	}

	h.submitJob(c, models.JobTypeSASTScan, req)
}

// SubmitQualityScanJob POST /api/gitlab/projects/:id/jobs/quality-scan
// Submits a quality scan job for background processing
func (h *GitLabJobSubmitHandler) SubmitQualityScanJob(c *gin.Context) {
	var req SubmitJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = SubmitJobRequest{}
	}

	h.submitJob(c, models.JobTypeQualityScan, req)
}

// SubmitDependencyScanJob POST /api/gitlab/projects/:id/jobs/dependency-scan
// Submits a dependency scan job for background processing
func (h *GitLabJobSubmitHandler) SubmitDependencyScanJob(c *gin.Context) {
	var req SubmitJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = SubmitJobRequest{}
	}

	h.submitJob(c, models.JobTypeDependencyScan, req)
}

// SubmitDeadCodeScanJob POST /api/gitlab/projects/:id/jobs/deadcode-scan
// Submits a dead code scan job for background processing
func (h *GitLabJobSubmitHandler) SubmitDeadCodeScanJob(c *gin.Context) {
	var req SubmitJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = SubmitJobRequest{}
	}

	h.submitJob(c, models.JobTypeDeadCodeScan, req)
}

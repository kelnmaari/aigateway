// Package handlers provides HTTP handlers for GitLab dead code detection API.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"aigateway/internal/gitlab/client"
	"aigateway/internal/gitlab/deadcode"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/models"
	"aigateway/internal/rag/vector"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GitLabDeadCodeHandler handles dead code detection API requests.
type GitLabDeadCodeHandler struct {
	store       storage.Store
	vectorStore *vector.QdrantStore
	llmBaseURL  string
	llmAPIKey   string
	logger      *logrus.Logger
}

// NewGitLabDeadCodeHandler creates a new dead code handler.
func NewGitLabDeadCodeHandler(store storage.Store, vectorStore *vector.QdrantStore, llmBaseURL, llmAPIKey string, logger *logrus.Logger) *GitLabDeadCodeHandler {
	return &GitLabDeadCodeHandler{
		store:       store,
		vectorStore: vectorStore,
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		logger:      logger,
	}
}

// DetectDeadCodeRequest is the request body for dead code detection.
type DetectDeadCodeRequest struct {
	MaxChunks int    `json:"max_chunks,omitempty"`
	Language  string `json:"language,omitempty"`
}

// DetectDeadCode POST /api/admin/gitlab/projects/:id/dead-code
func (h *GitLabDeadCodeHandler) DetectDeadCode(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req DetectDeadCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = DetectDeadCodeRequest{}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Minute)
	defer cancel()

	// Get project
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Failed to get project")
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Check if project has analysis model
	if project.AnalysisModelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project has no analysis model configured"})
		return
	}

	// Check if project is indexed
	collectionName := project.GetCollectionName()
	if collectionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project has no indexed collection. Please index the repository first."})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"collection": collectionName,
		"model":      project.AnalysisModelID,
	}).Info("Starting dead code detection")

	// Create scan history record
	startTime := time.Now()
	scanResult := &models.GitLabScanResult{
		ProjectID:     projectID,
		IntegrationID: project.IntegrationID,
		ScanType:      models.ScanTypeDeadCode,
		Status:        models.ScanStatusRunning,
		ModelID:       project.AnalysisModelID,
		StartedAt:     startTime,
	}

	// Create scanner and run detection
	scanner := deadcode.NewScanner(h.vectorStore, h.llmBaseURL, h.llmAPIKey, h.logger)
	result, err := scanner.Scan(ctx, deadcode.ScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		ModelID:        project.AnalysisModelID,
		MaxChunks:      req.MaxChunks,
		Language:       req.Language,
	})

	// Update scan result
	completedAt := time.Now()
	scanResult.CompletedAt = &completedAt
	scanResult.DurationMs = completedAt.Sub(startTime).Milliseconds()

	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Dead code detection failed")
		scanResult.Status = models.ScanStatusFailed
		scanResult.Error = err.Error()
		if saveErr := h.store.SaveScanResult(ctx, scanResult); saveErr != nil {
			h.logger.WithError(saveErr).Warn("Failed to save scan result")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Dead code detection failed: " + err.Error()})
		return
	}

	// Save successful result
	scanResult.Status = models.ScanStatusCompleted
	scanResult.FindingsCount = len(result.DeadSymbols)
	scanResult.FilesAffected = len(result.Summary.TopAffectedFiles)
	scanResult.TokensUsed = result.TokensUsed
	if resultJSON, jsonErr := json.Marshal(result); jsonErr == nil {
		scanResult.ResultsJSON = string(resultJSON)
	}
	if saveErr := h.store.SaveScanResult(ctx, scanResult); saveErr != nil {
		h.logger.WithError(saveErr).Warn("Failed to save scan result")
	}

	c.JSON(http.StatusOK, result)
}


// DetectUnreachableRequest is the request body for unreachable code detection.
type DetectUnreachableRequest struct {
	MaxChunks int    `json:"max_chunks,omitempty"`
	Language  string `json:"language,omitempty"`
}

// DetectUnreachable POST /api/admin/gitlab/projects/:id/detect-unreachable
func (h *GitLabDeadCodeHandler) DetectUnreachable(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req DetectUnreachableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = DetectUnreachableRequest{}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Failed to get project")
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	if project.AnalysisModelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project has no analysis model configured"})
		return
	}

	collectionName := project.GetCollectionName()
	if collectionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project has no indexed collection"})
		return
	}

	h.logger.WithField("project_id", projectID).Info("Starting unreachable code detection")

	scanner := deadcode.NewScanner(h.vectorStore, h.llmBaseURL, h.llmAPIKey, h.logger)
	result, err := scanner.ScanUnreachable(ctx, deadcode.ScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		ModelID:        project.AnalysisModelID,
		MaxChunks:      req.MaxChunks,
		Language:       req.Language,
	})
	if err != nil {
		h.logger.WithError(err).Error("Unreachable code detection failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Detection failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DetectCommentedCodeRequest is the request body for commented-out code detection.
type DetectCommentedCodeRequest struct {
	MaxChunks int    `json:"max_chunks,omitempty"`
	Language  string `json:"language,omitempty"`
}

// DetectCommentedCode POST /api/admin/gitlab/projects/:id/detect-commented-code
func (h *GitLabDeadCodeHandler) DetectCommentedCode(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req DetectCommentedCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = DetectCommentedCodeRequest{}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Failed to get project")
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	if project.AnalysisModelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project has no analysis model configured"})
		return
	}

	collectionName := project.GetCollectionName()
	if collectionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project has no indexed collection"})
		return
	}

	h.logger.WithField("project_id", projectID).Info("Starting commented-out code detection")

	scanner := deadcode.NewScanner(h.vectorStore, h.llmBaseURL, h.llmAPIKey, h.logger)
	result, err := scanner.ScanCommentedCode(ctx, deadcode.ScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		ModelID:        project.AnalysisModelID,
		MaxChunks:      req.MaxChunks,
		Language:       req.Language,
	})
	if err != nil {
		h.logger.WithError(err).Error("Commented-out code detection failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Detection failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateDeadCodeIssueRequest for creating GitLab issues with dead code
type CreateDeadCodeIssueRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

// CreateDeadCodeIssue POST /api/admin/gitlab/projects/:id/dead-code/create-issue
func (h *GitLabDeadCodeHandler) CreateDeadCodeIssue(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req CreateDeadCodeIssueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx := c.Request.Context()

	// Get project
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Failed to get project")
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Get integration
	integration, err := h.store.GetIntegration(ctx, project.IntegrationID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get GitLab integration"})
		return
	}

	// Create issue using GitLab API
	issueURL, err := h.createGitLabIssue(ctx, integration.BaseURL, integration.AccessToken, project.GitLabProjectID, req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create GitLab issue")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create issue: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Issue created successfully",
		"url":     issueURL,
	})
}

// createGitLabIssue creates an issue in GitLab for dead code.
func (h *GitLabDeadCodeHandler) createGitLabIssue(ctx context.Context, gitlabURL, token string, projectID int64, req CreateDeadCodeIssueRequest) (string, error) {
	h.logger.WithFields(logrus.Fields{
		"gitlab_project_id": projectID,
		"title":             req.Title,
	}).Info("Creating GitLab issue for dead code")

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     gitlabURL,
		AccessToken: token,
		Timeout:     30 * time.Second,
	})

	// Build labels
	labels := req.Labels
	if len(labels) == 0 {
		labels = []string{"dead-code", "cleanup", "ai-generated"}
	}

	issueReq := &client.CreateIssueRequest{
		Title:       req.Title,
		Description: req.Description,
		Labels:      labels,
	}

	issue, err := gitlabClient.CreateIssue(ctx, projectID, issueReq)
	if err != nil {
		return "", fmt.Errorf("create issue: %w", err)
	}

	return issue.WebURL, nil
}

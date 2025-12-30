// Package handlers provides HTTP handlers for GitLab dead code detection API.
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"aigateway/internal/gitlab/deadcode"
	"aigateway/internal/gitlab/storage"
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

	// Create scanner and run detection
	scanner := deadcode.NewScanner(h.vectorStore, h.llmBaseURL, h.llmAPIKey, h.logger)
	result, err := scanner.Scan(ctx, deadcode.ScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		ModelID:        project.AnalysisModelID,
		MaxChunks:      req.MaxChunks,
		Language:       req.Language,
	})
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Dead code detection failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Dead code detection failed: " + err.Error()})
		return
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
	IntegrationID string   `json:"integration_id" binding:"required"`
	SymbolIDs     []string `json:"symbol_ids"`
	Title         string   `json:"title,omitempty"`
	Priority      string   `json:"priority,omitempty"`
	Labels        []string `json:"labels,omitempty"`
	IncludeAll    bool     `json:"include_all,omitempty"`
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	// Get project
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Failed to get project")
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"project_id":  projectID,
		"symbols":     len(req.SymbolIDs),
		"include_all": req.IncludeAll,
	}).Info("Creating GitLab issue for dead code")

	// Build issue content
	title := req.Title
	if title == "" {
		title = fmt.Sprintf("Dead Code Cleanup: %s", project.Name)
	}

	body := "## Dead Code Report\n\n"
	body += fmt.Sprintf("This issue was automatically generated by AIGateway Dead Code Detection.\n\n")
	body += "### Selected Items for Cleanup\n\n"
	
	if req.IncludeAll {
		body += "All detected dead code items are included.\n\n"
	} else {
		body += fmt.Sprintf("%d items selected for cleanup.\n\n", len(req.SymbolIDs))
	}

	body += "### Recommendations\n\n"
	body += "1. Review each item before deletion\n"
	body += "2. Ensure no dynamic references exist\n"
	body += "3. Run tests after cleanup\n"

	// Labels
	labels := req.Labels
	if len(labels) == 0 {
		labels = []string{"dead-code", "cleanup", "ai-generated"}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Issue creation prepared (GitLab integration required)",
		"title":       title,
		"body":        body,
		"labels":      labels,
		"symbol_count": len(req.SymbolIDs),
		"note":        "Full GitLab Issue API integration will be implemented when GitLab client is available in this handler",
	})
}

// Package handlers provides HTTP handlers for GitLab quality API.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"aigateway/internal/gitlab/quality"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/models"
	"aigateway/internal/rag/vector"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GitLabQualityHandler handles code quality analysis API requests.
type GitLabQualityHandler struct {
	store       storage.Store
	vectorStore *vector.QdrantStore
	llmBaseURL  string
	llmAPIKey   string
	logger      *logrus.Logger
}

// NewGitLabQualityHandler creates a new quality handler.
func NewGitLabQualityHandler(store storage.Store, vectorStore *vector.QdrantStore, llmBaseURL, llmAPIKey string, logger *logrus.Logger) *GitLabQualityHandler {
	return &GitLabQualityHandler{
		store:       store,
		vectorStore: vectorStore,
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		logger:      logger,
	}
}

// AnalyzeQualityRequest is the request body for quality analysis.
type AnalyzeQualityRequest struct {
	MaxFiles int    `json:"max_files,omitempty"`
	Language string `json:"language,omitempty"`
}

// AnalyzeQuality POST /api/admin/gitlab/projects/:id/quality-score
func (h *GitLabQualityHandler) AnalyzeQuality(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req AnalyzeQualityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Use defaults
		req = AnalyzeQualityRequest{}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
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
	}).Info("Starting code quality analysis")

	// Create scan history record
	startTime := time.Now()
	scanResult := &models.GitLabScanResult{
		ProjectID:     projectID,
		IntegrationID: project.IntegrationID,
		ScanType:      models.ScanTypeQuality,
		Status:        models.ScanStatusRunning,
		ModelID:       project.AnalysisModelID,
		StartedAt:     startTime,
	}

	// Create analyzer and run analysis
	analyzer := quality.NewAnalyzer(h.vectorStore, h.llmBaseURL, h.llmAPIKey, h.logger)
	result, err := analyzer.Analyze(ctx, quality.AnalysisRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		ModelID:        project.AnalysisModelID,
		MaxFiles:       req.MaxFiles,
		Language:       req.Language,
	})

	// Update scan result
	completedAt := time.Now()
	scanResult.CompletedAt = &completedAt
	scanResult.DurationMs = completedAt.Sub(startTime).Milliseconds()

	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Quality analysis failed")
		scanResult.Status = models.ScanStatusFailed
		scanResult.Error = err.Error()
		if saveErr := h.store.SaveScanResult(ctx, scanResult); saveErr != nil {
			h.logger.WithError(saveErr).Warn("Failed to save scan result")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Quality analysis failed: " + err.Error()})
		return
	}

	// Save successful result
	scanResult.Status = models.ScanStatusCompleted
	scanResult.FindingsCount = result.Summary.IssuesCount
	scanResult.FilesAffected = len(result.FileScores)
	scanResult.TokensUsed = result.TokensUsed
	if resultJSON, jsonErr := json.Marshal(result); jsonErr == nil {
		scanResult.ResultsJSON = string(resultJSON)
	}
	if saveErr := h.store.SaveScanResult(ctx, scanResult); saveErr != nil {
		h.logger.WithError(saveErr).Warn("Failed to save scan result")
	}

	c.JSON(http.StatusOK, result)
}

// DetectDuplicationRequest is the request body for duplication detection.
type DetectDuplicationRequest struct {
	MaxFiles int    `json:"max_files,omitempty"`
	Language string `json:"language,omitempty"`
}

// DetectDuplication POST /api/admin/gitlab/projects/:id/detect-duplication
// Analyzes code for duplicate patterns using LLM.
func (h *GitLabQualityHandler) DetectDuplication(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req DetectDuplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = DetectDuplicationRequest{}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	// Get project
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project has no indexed collection. Please index the repository first."})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"collection": collectionName,
		"model":      project.AnalysisModelID,
	}).Info("Starting code duplication detection")

	analyzer := quality.NewAnalyzer(h.vectorStore, h.llmBaseURL, h.llmAPIKey, h.logger)
	result, err := analyzer.DetectDuplication(ctx, quality.AnalysisRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		ModelID:        project.AnalysisModelID,
		MaxFiles:       req.MaxFiles,
		Language:       req.Language,
	})
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Duplication detection failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Duplication detection failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}


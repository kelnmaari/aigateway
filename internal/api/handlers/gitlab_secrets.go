// Package handlers provides HTTP handlers for GitLab secrets scanning API
package handlers

import (
	"context"
	"net/http"
	"time"

	"aigateway/internal/gitlab/scanner"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/rag/vector"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// QdrantStoreForSecrets interface for secrets scanning
type QdrantStoreForSecrets interface {
	ScrollAll(ctx context.Context, collection string, filters map[string]interface{}, callback func([]vector.VectorDocument) error) error
}

// QdrantStoreInterface type alias for router package
type QdrantStoreInterface = QdrantStoreForSecrets

// GitLabSecretsHandler handles secrets scanning API requests
type GitLabSecretsHandler struct {
	store       storage.Store
	vectorStore *vector.QdrantStore
	scanner     *scanner.Scanner
	llmBaseURL  string
	llmAPIKey   string
	logger      *logrus.Logger
}

// NewGitLabSecretsHandler creates a new secrets handler
func NewGitLabSecretsHandler(store storage.Store, vectorStore *vector.QdrantStore, llmBaseURL, llmAPIKey string, logger *logrus.Logger) *GitLabSecretsHandler {
	return &GitLabSecretsHandler{
		store:       store,
		vectorStore: vectorStore,
		scanner:     scanner.NewScanner(vectorStore, logger),
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		logger:      logger,
	}
}

// NewGitLabSecretsHandlerWithInterface creates a handler with interface-based vector store
func NewGitLabSecretsHandlerWithInterface(store storage.Store, vectorStore QdrantStoreForSecrets, llmBaseURL, llmAPIKey string, logger *logrus.Logger) *GitLabSecretsHandler {
	// Type assert to concrete type for scanner
	qdrantStore, ok := vectorStore.(*vector.QdrantStore)
	if !ok {
		logger.Warn("VectorStore is not a QdrantStore, secrets scanning may not work")
		return &GitLabSecretsHandler{
			store:      store,
			llmBaseURL: llmBaseURL,
			llmAPIKey:  llmAPIKey,
			logger:     logger,
		}
	}
	return &GitLabSecretsHandler{
		store:       store,
		vectorStore: qdrantStore,
		scanner:     scanner.NewScanner(qdrantStore, logger),
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		logger:      logger,
	}
}

// ScanSecretsRequest request body for secrets scanning
type ScanSecretsRequest struct {
	Categories  []string         `json:"categories,omitempty"`
	MinSeverity scanner.Severity `json:"min_severity,omitempty"`
}

// ScanSecrets POST /api/admin/gitlab/projects/:id/scan-secrets
func (h *GitLabSecretsHandler) ScanSecrets(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	// Parse request body
	var req ScanSecretsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Empty body is OK, use defaults
		req = ScanSecretsRequest{}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	// Get project to find collection name
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Failed to get project")
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Determine collection name
	collectionName := project.GetCollectionName()
	if collectionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project has no indexed collection. Please index the repository first."})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"collection": collectionName,
	}).Info("Starting secrets scan")

	// Run scan
	result, err := h.scanner.Scan(ctx, scanner.ScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		Categories:     req.Categories,
		MinSeverity:    req.MinSeverity,
	})
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Secrets scan failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Scan failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetPatterns GET /api/admin/gitlab/secrets/patterns
func (h *GitLabSecretsHandler) GetPatterns(c *gin.Context) {
	patterns := h.scanner.GetPatterns()

	// Convert to response format (without regex)
	response := make([]map[string]interface{}, len(patterns))
	for i, p := range patterns {
		response[i] = map[string]interface{}{
			"id":          p.ID,
			"name":        p.Name,
			"description": p.Description,
			"severity":    p.Severity,
			"category":    p.Category,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"patterns": response,
		"total":    len(patterns),
	})
}

// DeepScanSecretsRequest request body for deep secrets scanning
type DeepScanSecretsRequest struct {
	ModelID   string `json:"model_id,omitempty"`   // Analysis model to use
	MaxChunks int    `json:"max_chunks,omitempty"` // Limit chunks (default 500)
	Language  string `json:"language,omitempty"`   // "en" or "ru"
}

// DeepScanSecrets POST /api/admin/gitlab/projects/:id/deep-scan-secrets
func (h *GitLabSecretsHandler) DeepScanSecrets(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	// Parse request body
	var req DeepScanSecretsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = DeepScanSecretsRequest{}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Minute) // Long timeout for LLM
	defer cancel()

	// Get project
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Failed to get project")
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Determine collection name
	collectionName := project.GetCollectionName()
	if collectionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project has no indexed collection. Please index the repository first."})
		return
	}

	// Use project's analysis model if not specified
	modelID := req.ModelID
	if modelID == "" {
		modelID = project.AnalysisModelID
	}
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No analysis model configured for project"})
		return
	}

	// Use project's review language if not specified
	language := req.Language
	if language == "" {
		language = project.Settings.ReviewLanguage
	}
	if language == "" {
		language = "en"
	}

	h.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"collection": collectionName,
		"model":      modelID,
		"language":   language,
	}).Info("Starting deep secrets scan")

	// Create deep scanner using configured LLM credentials
	deepScanner := scanner.NewDeepScanner(h.vectorStore, h.llmBaseURL, h.llmAPIKey, h.logger)

	// Run deep scan
	result, err := deepScanner.DeepScan(ctx, scanner.DeepScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		ModelID:        modelID,
		MaxChunks:      req.MaxChunks,
		Language:       language,
	})
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Deep secrets scan failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Deep scan failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// SAST (Static Application Security Testing)
// ============================================================================

// SASTScanRequest request body for SAST scanning
type SASTScanRequest struct {
	Types       []scanner.VulnerabilityType `json:"types,omitempty"`
	MinSeverity scanner.Severity            `json:"min_severity,omitempty"`
	Language    string                      `json:"language,omitempty"`
}

// SASTScan POST /api/admin/gitlab/projects/:id/sast-scan
// Performs SAST scanning for security vulnerabilities.
func (h *GitLabSecretsHandler) SASTScan(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req SASTScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = SASTScanRequest{}
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

	// Check if project is indexed
	collectionName := project.GetCollectionName()
	if collectionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project has no indexed collection. Please index the repository first."})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"collection": collectionName,
		"language":   req.Language,
		"types":      req.Types,
	}).Info("Starting SAST scan")

	// Create SAST scanner and run
	sastScanner := scanner.NewSASTScanner(h.vectorStore, h.logger)
	result, err := sastScanner.Scan(ctx, scanner.SASTScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		Types:          req.Types,
		MinSeverity:    req.MinSeverity,
		Language:       req.Language,
	})
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("SAST scan failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "SAST scan failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}


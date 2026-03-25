// Package handlers provides HTTP handlers for GitLab secrets scanning API
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"aigateway/internal/gitlab/client"
	"aigateway/internal/gitlab/scanner"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/models"
	"aigateway/internal/rag/vector"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// QdrantStoreForSecrets interface for secrets scanning
type QdrantStoreForSecrets interface {
	ScrollAll(ctx context.Context, collection string, filters map[string]any, callback func([]vector.VectorDocument) error) error
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

// getUserID extracts user ID from context
func (h *GitLabSecretsHandler) getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

// canAccessProject checks if user can access the project (via integration ownership)
func (h *GitLabSecretsHandler) canAccessProject(c *gin.Context, project *models.GitLabProject) bool {
	userID := h.getUserID(c)
	if userID == "" {
		return false
	}

	// Admins can access everything (if is_admin is set by middleware)
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
			if slices.Contains(ids, integration.TenantID) {
				return true
			}
		}
	}

	return false
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

	// Create scan history record
	startTime := time.Now()
	scanResult := &models.GitLabScanResult{
		ProjectID:     projectID,
		IntegrationID: project.IntegrationID,
		ScanType:      models.ScanTypeSecrets,
		Status:        models.ScanStatusRunning,
		StartedAt:     startTime,
	}

	// Run scan
	result, err := h.scanner.Scan(ctx, scanner.ScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		Categories:     req.Categories,
		MinSeverity:    req.MinSeverity,
	})

	// Update scan result
	completedAt := time.Now()
	scanResult.CompletedAt = &completedAt
	scanResult.DurationMs = completedAt.Sub(startTime).Milliseconds()

	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Secrets scan failed")
		scanResult.Status = models.ScanStatusFailed
		scanResult.Error = err.Error()
		if saveErr := h.store.SaveScanResult(ctx, scanResult); saveErr != nil {
			h.logger.WithError(saveErr).Warn("Failed to save scan result to history")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Scan failed: " + err.Error()})
		return
	}

	// Save successful result
	scanResult.Status = models.ScanStatusCompleted
	scanResult.FindingsCount = len(result.Findings)
	scanResult.FilesAffected = result.Summary.FilesAffected
	if resultJSON, jsonErr := json.Marshal(result); jsonErr == nil {
		scanResult.ResultsJSON = string(resultJSON)
	}
	if saveErr := h.store.SaveScanResult(ctx, scanResult); saveErr != nil {
		h.logger.WithError(saveErr).Warn("Failed to save scan result to history")
	}

	c.JSON(http.StatusOK, result)
}

// ScanMySecrets handles user-level secrets scanning (POST /api/gitlab/projects/:id/scan-secrets)
func (h *GitLabSecretsHandler) ScanMySecrets(c *gin.Context) {
	projectID := c.Param("id")
	ctx := c.Request.Context()

	// Get project
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Check ownership
	if !h.canAccessProject(c, project) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Reuse admin scanner logic
	h.ScanSecrets(c)
}

// GetPatterns GET /api/admin/gitlab/secrets/patterns
func (h *GitLabSecretsHandler) GetPatterns(c *gin.Context) {
	patterns := h.scanner.GetPatterns()

	// Convert to response format (without regex)
	response := make([]map[string]any, len(patterns))
	for i, p := range patterns {
		response[i] = map[string]any{
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
	Stream    bool   `json:"stream,omitempty"`     // Enable SSE streaming for progress
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
		"stream":     req.Stream,
	}).Info("Starting deep secrets scan")

	// Create deep scanner using configured LLM credentials
	deepScanner := scanner.NewDeepScanner(h.vectorStore, h.llmBaseURL, h.llmAPIKey, h.logger)

	// Create scan history record (pending)
	startTime := time.Now()
	scanResult := &models.GitLabScanResult{
		ProjectID:     projectID,
		IntegrationID: project.IntegrationID,
		ScanType:      models.ScanTypeSecretsDeep,
		Status:        models.ScanStatusRunning,
		ModelID:       modelID,
		StartedAt:     startTime,
	}

	// Handle streaming mode
	if req.Stream {
		h.deepScanWithStreaming(c, ctx, deepScanner, projectID, collectionName, modelID, req, scanResult, startTime)
		return
	}

	// Non-streaming mode (original behavior)
	result, err := deepScanner.DeepScan(ctx, scanner.DeepScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		ModelID:        modelID,
		MaxChunks:      req.MaxChunks,
		Language:       language,
	})

	// Update scan result with outcome
	completedAt := time.Now()
	scanResult.CompletedAt = &completedAt
	scanResult.DurationMs = completedAt.Sub(startTime).Milliseconds()

	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Deep secrets scan failed")
		scanResult.Status = models.ScanStatusFailed
		scanResult.Error = err.Error()
		// Save failed result
		if saveErr := h.store.SaveScanResult(ctx, scanResult); saveErr != nil {
			h.logger.WithError(saveErr).Warn("Failed to save scan result to history")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Deep scan failed: " + err.Error()})
		return
	}

	// Save successful result
	scanResult.Status = models.ScanStatusCompleted
	scanResult.FindingsCount = len(result.Findings)
	scanResult.FilesAffected = result.Summary.FilesAffected
	scanResult.TokensUsed = result.TokensUsed

	// Serialize results JSON
	if resultJSON, jsonErr := json.Marshal(result); jsonErr == nil {
		scanResult.ResultsJSON = string(resultJSON)
	}

	if saveErr := h.store.SaveScanResult(ctx, scanResult); saveErr != nil {
		h.logger.WithError(saveErr).Warn("Failed to save scan result to history")
	} else {
		h.logger.WithField("scan_id", scanResult.ID).Info("Deep scan result saved to history")
	}

	c.JSON(http.StatusOK, result)
}

// deepScanWithStreaming handles SSE streaming mode for deep scan
func (h *GitLabSecretsHandler) deepScanWithStreaming(
	c *gin.Context,
	ctx context.Context,
	deepScanner *scanner.DeepScanner,
	projectID, collectionName, modelID string,
	req DeepScanSecretsRequest,
	scanResult *models.GitLabScanResult,
	startTime time.Time,
) {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// Get the flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	// Send initial event
	h.sendSSEEvent(c.Writer, flusher, "start", map[string]any{
		"project_id": projectID,
		"model":      modelID,
		"status":     "starting",
	})

	// Progress callback for streaming updates
	progressCallback := func(progress scanner.ScanProgress) {
		h.sendSSEEvent(c.Writer, flusher, "progress", progress)
	}

	// Run deep scan with progress
	language := req.Language
	if language == "" {
		language = "en"
	}

	result, err := deepScanner.DeepScanWithProgress(ctx, scanner.DeepScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		ModelID:        modelID,
		MaxChunks:      req.MaxChunks,
		Language:       language,
	}, progressCallback)

	// Update scan result with outcome
	completedAt := time.Now()
	scanResult.CompletedAt = &completedAt
	scanResult.DurationMs = completedAt.Sub(startTime).Milliseconds()

	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Deep secrets scan failed")
		scanResult.Status = models.ScanStatusFailed
		scanResult.Error = err.Error()
		if saveErr := h.store.SaveScanResult(ctx, scanResult); saveErr != nil {
			h.logger.WithError(saveErr).Warn("Failed to save scan result to history")
		}
		h.sendSSEEvent(c.Writer, flusher, "error", map[string]string{
			"error": err.Error(),
		})
		h.sendSSEEvent(c.Writer, flusher, "done", nil)
		return
	}

	// Save successful result
	scanResult.Status = models.ScanStatusCompleted
	scanResult.FindingsCount = len(result.Findings)
	scanResult.FilesAffected = result.Summary.FilesAffected
	scanResult.TokensUsed = result.TokensUsed

	if resultJSON, jsonErr := json.Marshal(result); jsonErr == nil {
		scanResult.ResultsJSON = string(resultJSON)
	}

	if saveErr := h.store.SaveScanResult(ctx, scanResult); saveErr != nil {
		h.logger.WithError(saveErr).Warn("Failed to save scan result to history")
	}

	// Send final result
	h.sendSSEEvent(c.Writer, flusher, "complete", result)
	h.sendSSEEvent(c.Writer, flusher, "done", nil)
}

// sendSSEEvent sends a Server-Sent Event
func (h *GitLabSecretsHandler) sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, event string, data any) {
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

// DeepScanMySecrets handles user-level deep secrets scanning (POST /api/gitlab/projects/:id/deep-scan-secrets)
func (h *GitLabSecretsHandler) DeepScanMySecrets(c *gin.Context) {
	projectID := c.Param("id")
	ctx := c.Request.Context()

	// Get project
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Check ownership
	if !h.canAccessProject(c, project) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Reuse admin scanner logic
	h.DeepScanSecrets(c)
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

// SASTScanMySecrets handles user-level SAST scanning
func (h *GitLabSecretsHandler) SASTScanMySecrets(c *gin.Context) {
	projectID := c.Param("id")
	ctx := c.Request.Context()

	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	if !h.canAccessProject(c, project) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	h.SASTScan(c)
}

// CreateSecretsIssueRequest is the request body for creating an issue.
type CreateSecretsIssueRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

// CreateSecretsIssue POST /api/admin/gitlab/projects/:id/secrets/create-issue
func (h *GitLabSecretsHandler) CreateSecretsIssue(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req CreateSecretsIssueRequest
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

// CreateMySecretsIssue handles user-level issue creation
func (h *GitLabSecretsHandler) CreateMySecretsIssue(c *gin.Context) {
	projectID := c.Param("id")
	ctx := c.Request.Context()

	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	if !h.canAccessProject(c, project) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	h.CreateSecretsIssue(c)
}

// createGitLabIssue creates an issue in GitLab for secrets.
func (h *GitLabSecretsHandler) createGitLabIssue(ctx context.Context, gitlabURL, token string, projectID int64, req CreateSecretsIssueRequest) (string, error) {
	h.logger.WithFields(logrus.Fields{
		"gitlab_project_id": projectID,
		"title":             req.Title,
	}).Info("Creating GitLab issue for secrets")

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     gitlabURL,
		AccessToken: token,
		Timeout:     30 * time.Second,
	})

	// Build labels
	labels := req.Labels
	if len(labels) == 0 {
		labels = []string{"security", "secrets", "urgent"}
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

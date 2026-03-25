// Package handlers provides HTTP handlers for GitLab dependencies API.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"aigateway/internal/gitlab/client"
	"aigateway/internal/gitlab/dependencies"
	"aigateway/internal/gitlab/dependencies/changelog"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/models"
	"aigateway/internal/rag/vector"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GitLabDependenciesHandler handles dependency scanning API requests.
type GitLabDependenciesHandler struct {
	store       storage.Store
	vectorStore *vector.QdrantStore
	scanner     *dependencies.Scanner
	llmBaseURL  string
	llmAPIKey   string
	logger      *logrus.Logger
}

// NewGitLabDependenciesHandler creates a new dependencies handler.
func NewGitLabDependenciesHandler(store storage.Store, vectorStore *vector.QdrantStore, llmBaseURL, llmAPIKey string, logger *logrus.Logger) *GitLabDependenciesHandler {
	h := &GitLabDependenciesHandler{
		store:       store,
		vectorStore: vectorStore,
		scanner:     dependencies.NewScanner(vectorStore, logger),
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		logger:      logger,
	}
	h.logger.Debug("GitLabDependenciesHandler initialized")
	return h
}

// getUserID extracts user ID from context
func (h *GitLabDependenciesHandler) getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

// canAccessProject checks if user can access the project (via integration ownership)
func (h *GitLabDependenciesHandler) canAccessProject(c *gin.Context, project *models.GitLabProject) bool {
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

// CheckDependencies POST /api/admin/gitlab/projects/:id/check-dependencies
func (h *GitLabDependenciesHandler) CheckDependencies(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
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
	}).Info("Starting dependency check")

	// Create scan history record
	startTime := time.Now()
	scanResult := &models.GitLabScanResult{
		ProjectID:     projectID,
		IntegrationID: project.IntegrationID,
		ScanType:      models.ScanTypeDependencies,
		Status:        models.ScanStatusRunning,
		StartedAt:     startTime,
	}

	// Run multi-ecosystem scan (supports monorepos with go.mod + package.json + requirements.txt)
	result, err := h.scanner.ScanProjectAllEcosystems(ctx, dependencies.ScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
	})

	// Update scan result
	completedAt := time.Now()
	scanResult.CompletedAt = &completedAt
	scanResult.DurationMs = completedAt.Sub(startTime).Milliseconds()

	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Dependency scan failed")
		scanResult.Status = models.ScanStatusFailed
		scanResult.Error = err.Error()
		if saveErr := h.store.SaveScanResult(ctx, scanResult); saveErr != nil {
			h.logger.WithError(saveErr).Warn("Failed to save scan result")
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Dependency scan failed: " + err.Error()})
		return
	}

	// Save successful result
	scanResult.Status = models.ScanStatusCompleted
	scanResult.FindingsCount = result.TotalSummary.TotalDependencies
	if resultJSON, jsonErr := json.Marshal(result); jsonErr == nil {
		scanResult.ResultsJSON = string(resultJSON)
	}
	if saveErr := h.store.SaveScanResult(ctx, scanResult); saveErr != nil {
		h.logger.WithError(saveErr).Warn("Failed to save scan result")
	}

	c.JSON(http.StatusOK, result)
}

// CheckMyDependencies handles user-level dependency check
func (h *GitLabDependenciesHandler) CheckMyDependencies(c *gin.Context) {
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

	h.CheckDependencies(c)
}

// CreateDependencyIssueRequest is the request body for creating an issue.
type CreateDependencyIssueRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Critical    bool     `json:"critical,omitempty"` // Include only critical/vulnerable deps
}

// CreateDependencyIssue POST /api/admin/gitlab/projects/:id/create-dependency-issue
func (h *GitLabDependenciesHandler) CreateDependencyIssue(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req CreateDependencyIssueRequest
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

// CreateMyDependencyIssue handles user-level dependency issue creation
func (h *GitLabDependenciesHandler) CreateMyDependencyIssue(c *gin.Context) {
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

	h.CreateDependencyIssue(c)
}

// createGitLabIssue creates an issue in GitLab.
func (h *GitLabDependenciesHandler) createGitLabIssue(ctx context.Context, gitlabURL, token string, projectID int64, req CreateDependencyIssueRequest) (string, error) {
	h.logger.WithFields(logrus.Fields{
		"gitlab_project_id": projectID,
		"title":             req.Title,
	}).Info("Creating GitLab issue for dependencies")

	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     gitlabURL,
		AccessToken: token,
		Timeout:     30 * time.Second,
	})

	// Build labels
	labels := req.Labels
	if len(labels) == 0 {
		labels = []string{"dependencies", "security"}
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

// AnalyzeChangelogRequest is the request body for changelog analysis.
type AnalyzeChangelogRequest struct {
	PackageName    string `json:"package_name" binding:"required"`
	CurrentVersion string `json:"current_version" binding:"required"`
	LatestVersion  string `json:"latest_version" binding:"required"`
	Language       string `json:"language" binding:"required"` // "go", "nodejs", "python"
	ModelID        string `json:"model_id,omitempty"`          // Optional, uses project's analysis model if not set
}

// AnalyzeChangelog POST /api/admin/gitlab/projects/:id/analyze-changelog
// Analyzes a specific dependency's changelog using LLM.
func (h *GitLabDependenciesHandler) AnalyzeChangelog(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req AnalyzeChangelogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	// Get project for model ID if not specified
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Failed to get project")
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	modelID := req.ModelID
	if modelID == "" {
		modelID = project.AnalysisModelID
	}
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No model ID specified and project has no default analysis model"})
		return
	}

	// Get LLM settings - use stored values or defaults
	llmBaseURL := h.llmBaseURL
	if llmBaseURL == "" {
		llmBaseURL = "http://localhost:8080" // Self-reference to internal API
	}
	llmAPIKey := h.llmAPIKey
	if llmAPIKey == "" {
		// Try to get from request header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) > 7 {
			llmAPIKey = authHeader[7:] // Remove "Bearer "
		}
	}

	h.logger.WithFields(logrus.Fields{
		"project_id":      projectID,
		"package":         req.PackageName,
		"current_version": req.CurrentVersion,
		"latest_version":  req.LatestVersion,
		"language":        req.Language,
		"model_id":        modelID,
	}).Info("Analyzing changelog")

	analyzer := changelog.NewAnalyzer(llmBaseURL, llmAPIKey, h.logger)

	h.logger.WithField("model_id", modelID).Debug("Starting changelog analysis")

	analysis, err := analyzer.AnalyzeChangelog(ctx, changelog.AnalyzeRequest{
		PackageName:    req.PackageName,
		CurrentVersion: req.CurrentVersion,
		LatestVersion:  req.LatestVersion,
		Language:       req.Language,
		ModelID:        modelID,
	})
	if err != nil {
		h.logger.WithError(err).Error("Changelog analysis failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Changelog analysis failed: " + err.Error()})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"risk_level":  analysis.RiskLevel,
		"confidence":  analysis.Confidence,
		"summary_len": len(analysis.Summary),
	}).Debug("Changelog analysis completed, sending response")

	c.JSON(http.StatusOK, analysis)
}

// AnalyzeMyChangelog handles user-level changelog analysis
func (h *GitLabDependenciesHandler) AnalyzeMyChangelog(c *gin.Context) {
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

	h.AnalyzeChangelog(c)
}

// AnalyzeDependenciesChangelogsRequest is the request for bulk changelog analysis.
type AnalyzeDependenciesChangelogsRequest struct {
	Dependencies []struct {
		Name           string `json:"name"`
		CurrentVersion string `json:"current_version"`
		LatestVersion  string `json:"latest_version"`
	} `json:"dependencies" binding:"required"`
	Language string `json:"language" binding:"required"`
	ModelID  string `json:"model_id,omitempty"`
}

// AnalyzeDependenciesChangelogs POST /api/admin/gitlab/projects/:id/analyze-changelogs
// Analyzes multiple dependencies' changelogs in batch.
func (h *GitLabDependenciesHandler) AnalyzeDependenciesChangelogs(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req AnalyzeDependenciesChangelogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	// Get project for model ID if not specified
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Failed to get project")
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	modelID := req.ModelID
	if modelID == "" {
		modelID = project.AnalysisModelID
	}
	if modelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No model ID specified and project has no default analysis model"})
		return
	}

	// Get LLM settings - use stored values or defaults
	llmBaseURL := h.llmBaseURL
	if llmBaseURL == "" {
		llmBaseURL = "http://localhost:8080" // Self-reference to internal API
	}
	llmAPIKey := h.llmAPIKey
	if llmAPIKey == "" {
		// Try to get from request header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) > 7 {
			llmAPIKey = authHeader[7:] // Remove "Bearer "
		}
	}

	h.logger.WithFields(logrus.Fields{
		"project_id":   projectID,
		"dependencies": len(req.Dependencies),
		"language":     req.Language,
		"model_id":     modelID,
	}).Info("Analyzing multiple changelogs")

	analyzer := changelog.NewAnalyzer(llmBaseURL, llmAPIKey, h.logger)
	results := make([]*changelog.ChangelogAnalysis, 0, len(req.Dependencies))

	for _, dep := range req.Dependencies {
		if dep.CurrentVersion == dep.LatestVersion {
			continue // Skip if already up to date
		}

		analysis, err := analyzer.AnalyzeChangelog(ctx, changelog.AnalyzeRequest{
			PackageName:    dep.Name,
			CurrentVersion: dep.CurrentVersion,
			LatestVersion:  dep.LatestVersion,
			Language:       req.Language,
			ModelID:        modelID,
		})
		if err != nil {
			h.logger.WithError(err).WithField("package", dep.Name).Warn("Failed to analyze changelog")
			// Continue with other dependencies
			continue
		}

		results = append(results, analysis)
	}

	c.JSON(http.StatusOK, gin.H{
		"analyses": results,
		"total":    len(results),
	})
}

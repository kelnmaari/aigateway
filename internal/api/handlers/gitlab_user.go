// Package handlers provides user-level GitLab handlers
package handlers

import (
	"context"
	"net/http"
	"time"

	"aigateway/internal/gitlab/client"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// GitLabUserHandler handles user-level GitLab operations
type GitLabUserHandler struct {
	store  storage.Store
	logger *logrus.Logger
}

// NewGitLabUserHandler creates a new user handler
func NewGitLabUserHandler(store storage.Store, logger *logrus.Logger) *GitLabUserHandler {
	return &GitLabUserHandler{
		store:  store,
		logger: logger,
	}
}

// getUserID extracts user ID from context (set by auth middleware)
func (h *GitLabUserHandler) getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

// ============================================================================
// Integration Management (User's own integrations)
// ============================================================================

// ListMyIntegrations lists user's GitLab integrations
func (h *GitLabUserHandler) ListMyIntegrations(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	integrations, err := h.store.ListIntegrationsByOwner(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list user integrations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list integrations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  integrations,
		"total": len(integrations),
	})
}

// CreateMyIntegration creates a new GitLab integration for the user
func (h *GitLabUserHandler) CreateMyIntegration(c *gin.Context) {
	userID := h.getUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Name          string                           `json:"name" binding:"required"`
		BaseURL       string                           `json:"base_url" binding:"required"`
		AccessToken   string                           `json:"access_token" binding:"required"`
		WebhookSecret string                           `json:"webhook_secret"`
		Settings      models.GitLabIntegrationSettings `json:"settings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	integration := &models.GitLabIntegration{
		ID:            uuid.New().String(),
		OwnerID:       userID,
		Name:          req.Name,
		BaseURL:       req.BaseURL,
		AccessToken:   req.AccessToken,
		WebhookSecret: req.WebhookSecret,
		Status:        models.GitLabIntegrationStatusActive,
		Settings:      req.Settings,
	}

	if err := h.store.CreateIntegration(c.Request.Context(), integration); err != nil {
		h.logger.WithError(err).Error("Failed to create integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create integration"})
		return
	}

	c.JSON(http.StatusCreated, integration)
}

// GetMyIntegration gets a specific integration owned by the user
func (h *GitLabUserHandler) GetMyIntegration(c *gin.Context) {
	userID := h.getUserID(c)
	integrationID := c.Param("id")

	integration, err := h.store.GetIntegration(c.Request.Context(), integrationID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get integration"})
		return
	}

	if integration == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}

	// Check ownership
	if integration.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, integration)
}

// UpdateMyIntegration updates user's integration
func (h *GitLabUserHandler) UpdateMyIntegration(c *gin.Context) {
	userID := h.getUserID(c)
	integrationID := c.Param("id")

	// Check ownership first
	existing, err := h.store.GetIntegration(c.Request.Context(), integrationID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get integration"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}
	if existing.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req models.UpdateGitLabIntegrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if err := h.store.UpdateIntegration(c.Request.Context(), integrationID, &req); err != nil {
		h.logger.WithError(err).Error("Failed to update integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update integration"})
		return
	}

	// Re-fetch updated integration
	updated, _ := h.store.GetIntegration(c.Request.Context(), integrationID)
	c.JSON(http.StatusOK, updated)
}

// DeleteMyIntegration deletes user's integration
func (h *GitLabUserHandler) DeleteMyIntegration(c *gin.Context) {
	userID := h.getUserID(c)
	integrationID := c.Param("id")

	// Check ownership
	existing, err := h.store.GetIntegration(c.Request.Context(), integrationID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get integration"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}
	if existing.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.store.DeleteIntegration(c.Request.Context(), integrationID); err != nil {
		h.logger.WithError(err).Error("Failed to delete integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete integration"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Integration deleted"})
}

// ============================================================================
// Project Management (User's projects)
// ============================================================================

// ListMyProjects lists projects in user's integration
func (h *GitLabUserHandler) ListMyProjects(c *gin.Context) {
	userID := h.getUserID(c)
	integrationID := c.Param("id")

	// Check integration ownership
	integration, err := h.store.GetIntegration(c.Request.Context(), integrationID)
	if err != nil || integration == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}
	if integration.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	projects, err := h.store.ListProjectsByIntegration(c.Request.Context(), integrationID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list projects")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  projects,
		"total": len(projects),
	})
}

// AddMyProject adds a project to user's integration
func (h *GitLabUserHandler) AddMyProject(c *gin.Context) {
	userID := h.getUserID(c)
	integrationID := c.Param("id")

	// Check integration ownership
	integration, err := h.store.GetIntegration(c.Request.Context(), integrationID)
	if err != nil || integration == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		return
	}
	if integration.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req struct {
		GitLabProjectID   int64                        `json:"gitlab_project_id" binding:"required"`
		Name              string                       `json:"name" binding:"required"`
		PathWithNamespace string                       `json:"path_with_namespace"`
		TenantID          string                       `json:"tenant_id"`
		AnalysisModelID   string                       `json:"analysis_model_id" binding:"required"`
		EmbeddingModelID  string                       `json:"embedding_model_id"`
		Settings          models.GitLabProjectSettings `json:"settings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	project := &models.GitLabProject{
		ID:                uuid.New().String(),
		IntegrationID:     integrationID,
		TenantID:          req.TenantID,
		GitLabProjectID:   req.GitLabProjectID,
		Name:              req.Name,
		PathWithNamespace: req.PathWithNamespace,
		AnalysisModelID:   req.AnalysisModelID,
		EmbeddingModelID:  req.EmbeddingModelID,
		AutoReview:        true,
		Status:            models.GitLabProjectStatusActive,
		Settings:          req.Settings,
	}

	if err := h.store.CreateProject(c.Request.Context(), project); err != nil {
		h.logger.WithError(err).Error("Failed to create project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	c.JSON(http.StatusCreated, project)
}

// GetMyProject gets a specific project owned by user
func (h *GitLabUserHandler) GetMyProject(c *gin.Context) {
	userID := h.getUserID(c)
	projectID := c.Param("id")

	project, err := h.store.GetProject(c.Request.Context(), projectID)
	if err != nil || project == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Check integration ownership
	integration, err := h.store.GetIntegration(c.Request.Context(), project.IntegrationID)
	if err != nil || integration == nil || integration.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, project)
}

// UpdateMyProject updates user's project
func (h *GitLabUserHandler) UpdateMyProject(c *gin.Context) {
	userID := h.getUserID(c)
	projectID := c.Param("id")

	project, err := h.store.GetProject(c.Request.Context(), projectID)
	if err != nil || project == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Check integration ownership
	integration, err := h.store.GetIntegration(c.Request.Context(), project.IntegrationID)
	if err != nil || integration == nil || integration.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var req models.UpdateGitLabProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if err := h.store.UpdateProject(c.Request.Context(), projectID, &req); err != nil {
		h.logger.WithError(err).Error("Failed to update project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	// Re-fetch updated project
	updated, _ := h.store.GetProject(c.Request.Context(), projectID)
	c.JSON(http.StatusOK, updated)
}

// DeleteMyProject deletes user's project
func (h *GitLabUserHandler) DeleteMyProject(c *gin.Context) {
	userID := h.getUserID(c)
	projectID := c.Param("id")

	project, err := h.store.GetProject(c.Request.Context(), projectID)
	if err != nil || project == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Check integration ownership
	integration, err := h.store.GetIntegration(c.Request.Context(), project.IntegrationID)
	if err != nil || integration == nil || integration.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.store.DeleteProject(c.Request.Context(), projectID); err != nil {
		h.logger.WithError(err).Error("Failed to delete project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project deleted"})
}

// ============================================================================
// Review History (User's reviews)
// ============================================================================

// ListMyReviews lists reviews for user's projects
func (h *GitLabUserHandler) ListMyReviews(c *gin.Context) {
	userID := h.getUserID(c)

	// Get user's integrations first
	integrations, err := h.store.ListIntegrationsByOwner(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list integrations")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list reviews"})
		return
	}

	var allReviews []*models.GitLabMRReview
	for _, integration := range integrations {
		reviews, err := h.store.ListReviewsByIntegration(c.Request.Context(), integration.ID)
		if err != nil {
			continue
		}
		allReviews = append(allReviews, reviews...)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  allReviews,
		"total": len(allReviews),
	})
}

// GetMyReview gets a specific review
func (h *GitLabUserHandler) GetMyReview(c *gin.Context) {
	userID := h.getUserID(c)
	reviewID := c.Param("id")

	review, err := h.store.GetReview(c.Request.Context(), reviewID)
	if err != nil || review == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Review not found"})
		return
	}

	// Check ownership through integration
	integration, err := h.store.GetIntegration(c.Request.Context(), review.IntegrationID)
	if err != nil || integration == nil || integration.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, review)
}

// SetupMyWebhook registers a webhook in GitLab for the user's project
func (h *GitLabUserHandler) SetupMyWebhook(c *gin.Context) {
	userID := h.getUserID(c)
	projectID := c.Param("id")

	var req struct {
		WebhookURL string `json:"webhook_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
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

	// Check integration ownership
	integration, err := h.store.GetIntegration(ctx, project.IntegrationID)
	if err != nil || integration == nil || integration.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create webhook: " + err.Error()})
		return
	}

	// Save webhook ID
	if err := h.store.UpdateProjectWebhookID(ctx, projectID, webhook.ID); err != nil {
		h.logger.WithError(err).Error("Failed to save webhook ID")
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Webhook created successfully",
		"webhook_id": webhook.ID,
	})
}

// ============================================================================
// Scan History (User's scans)
// ============================================================================

// ListMyScanHistory lists scan results for user's projects
func (h *GitLabUserHandler) ListMyScanHistory(c *gin.Context) {
	userID := h.getUserID(c)
	projectID := c.Query("project_id") // Optional filter

	req := &models.GitLabScanResultsRequest{
		ProjectID: projectID,
		Limit:     50,
		Offset:    0,
	}

	// If no project_id specified, we must filter by user's integrations
	if projectID == "" {
		integrations, err := h.store.ListIntegrationsByOwner(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list integrations"})
			return
		}

		var integrationIDs []string
		for _, integration := range integrations {
			integrationIDs = append(integrationIDs, integration.ID)
		}
		req.IntegrationIDs = integrationIDs
	} else {
		// Verify project ownership if projectID is provided
		project, err := h.store.GetProject(c.Request.Context(), projectID)
		if err != nil || project == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		integration, err := h.store.GetIntegration(c.Request.Context(), project.IntegrationID)
		if err != nil || integration == nil || integration.OwnerID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
	}

	results, total, err := h.store.ListScanResults(c.Request.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list scan history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list scan history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"total": total,
	})
}

// GetMyScanResult gets a specific scan result
func (h *GitLabUserHandler) GetMyScanResult(c *gin.Context) {
	userID := h.getUserID(c)
	scanID := c.Param("id")

	result, err := h.store.GetScanResult(c.Request.Context(), scanID)
	if err != nil || result == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan result not found"})
		return
	}

	// Check ownership through integration
	integration, err := h.store.GetIntegration(c.Request.Context(), result.IntegrationID)
	if err != nil || integration == nil || integration.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetMyScanTypes GET /api/gitlab/scan-history/types
func (h *GitLabUserHandler) GetMyScanTypes(c *gin.Context) {
	types := []gin.H{
		{"value": "secrets", "label": "Secrets Scan", "description": "Regex-based secrets detection"},
		{"value": "secrets_deep", "label": "Deep Secrets Scan", "description": "LLM-powered semantic secrets detection"},
		{"value": "dependencies", "label": "Dependencies Check", "description": "Outdated and vulnerable dependencies"},
		{"value": "quality", "label": "Code Quality", "description": "Code quality analysis"},
		{"value": "deadcode", "label": "Dead Code", "description": "Unused code detection"},
		{"value": "autodocs", "label": "Auto-Documentation", "description": "Undocumented code detection"},
		{"value": "testgen", "label": "Test Generation", "description": "Testable code detection"},
		{"value": "architecture", "label": "Architecture", "description": "Architecture analysis"},
	}

	c.JSON(http.StatusOK, gin.H{"types": types})
}

// Package handlers provides user-level GitLab handlers
package handlers

import (
	"net/http"

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
		Name          string                          `json:"name" binding:"required"`
		BaseURL       string                          `json:"base_url" binding:"required"`
		AccessToken   string                          `json:"access_token" binding:"required"`
		WebhookSecret string                          `json:"webhook_secret"`
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


package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/gitlab/client"
	"aigateway/internal/gitlab/indexer"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/models"
)

// GitLabUserIndexerHandler handles user-level repository indexing requests
// Users can only index projects they own (via integration ownership)
type GitLabUserIndexerHandler struct {
	indexer *indexer.Indexer
	store   storage.Store
	logger  *logrus.Logger
}

// NewGitLabUserIndexerHandler creates a new user-level indexer handler
func NewGitLabUserIndexerHandler(idx *indexer.Indexer, store storage.Store, logger *logrus.Logger) *GitLabUserIndexerHandler {
	return &GitLabUserIndexerHandler{
		indexer: idx,
		store:   store,
		logger:  logger,
	}
}

// getUserFromContext extracts user info from JWT context
func (h *GitLabUserIndexerHandler) getUserFromContext(c *gin.Context) (userID, tenantID string, isAdmin bool) {
	if uid, exists := c.Get("user_id"); exists {
		if u, ok := uid.(string); ok {
			userID = u
		}
	}
	if admin, exists := c.Get("is_admin"); exists {
		if a, ok := admin.(bool); ok {
			isAdmin = a
		}
	}

	// Middleware also sets 'tenant_ids' directly
	if tids, exists := c.Get("tenant_ids"); exists {
		if ids, ok := tids.([]string); ok && len(ids) > 0 {
			tenantID = ids[0]
		}
	}

	return
}

// canAccessProject checks if user can access the project for indexing
func (h *GitLabUserIndexerHandler) canAccessProject(c *gin.Context, project *models.GitLabProject) bool {
	userID, tenantID, isAdmin := h.getUserFromContext(c)

	// Admins can access everything
	if isAdmin {
		return true
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

	// Check tenant membership (if tenant mode)
	if tenantID != "" && integration.TenantID == tenantID {
		return true
	}

	return false
}

// IndexProject handles POST /api/gitlab/projects/:project_id/index
// User-level endpoint - checks ownership before allowing indexing
func (h *GitLabUserIndexerHandler) IndexProject(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID required"})
		return
	}

	var req IndexProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	ctx := c.Request.Context()

	// Get project from store
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get project")
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	// Check access
	if !h.canAccessProject(c, project) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you don't have access to this project"})
		return
	}

	// Determine branch to index
	branch := req.Branch
	if branch == "" {
		branch = project.DefaultBranch
		if branch == "" {
			branch = "main"
		}
	}

	// Build index request
	indexReq := indexer.IndexRequest{
		IntegrationID:    project.IntegrationID,
		ProjectID:        project.ID,
		GitLabProjectID:  project.GitLabProjectID,
		Branch:           branch,
		EmbeddingModelID: project.EmbeddingModelID,
		CollectionName:   project.GetCollectionName(), // Use project-specific Qdrant collection
		Force:            req.Force,
	}

	// Ensure indexer has a client for this integration
	if _, ok := h.indexer.GetClient(project.IntegrationID); !ok {
		integration, err := h.store.GetIntegration(ctx, project.IntegrationID)
		if err != nil || integration == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get integration"})
			return
		}

		gitlabClient := client.NewClient(client.ClientConfig{
			BaseURL:     integration.BaseURL,
			AccessToken: integration.AccessToken,
			Timeout:     60 * time.Second,
		})
		h.indexer.RegisterClient(project.IntegrationID, gitlabClient)
	}

	// Start async indexing
	if err := h.indexer.IndexBranch(ctx, indexReq); err != nil {
		h.logger.WithError(err).Error("Failed to start indexing")
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	userID, _, _ := h.getUserFromContext(c)
	h.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"branch":     branch,
		"force":      req.Force,
		"user_id":    userID,
	}).Info("User started repository indexing")

	c.JSON(http.StatusAccepted, gin.H{
		"message":    "Indexing started",
		"project_id": projectID,
		"branch":     branch,
		"status":     "in_progress",
	})
}

// GetIndexStatus handles GET /api/gitlab/projects/:project_id/index/status
func (h *GitLabUserIndexerHandler) GetIndexStatus(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID required"})
		return
	}

	ctx := c.Request.Context()

	// Get project
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	// Check access
	if !h.canAccessProject(c, project) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you don't have access to this project"})
		return
	}

	branch := c.Query("branch")
	if branch == "" {
		branch = project.DefaultBranch
		if branch == "" {
			branch = "main"
		}
	}

	// Try to get status from indexer (Redis/in-memory cache)
	status := h.indexer.GetStatus(projectID, branch)

	// If indexer has no cached status, use DB as source of truth
	if status == nil {
		status = &indexer.IndexInfo{
			ProjectID:   projectID,
			Branch:      branch,
			Status:      indexer.IndexStatus(project.IndexStatus),
			LastIndexed: project.LastIndexedAt,
		}

		// Handle edge case: if DB has "in_progress" after server restart,
		// the indexing was interrupted - report as failed
		if status.Status == indexer.IndexStatusInProgress {
			status.Status = indexer.IndexStatusFailed
			status.Error = "Indexing was interrupted by server restart"
		}

		// If status is empty string or unknown, treat as pending
		if status.Status == "" {
			status.Status = indexer.IndexStatusPending
		}
	}

	c.JSON(http.StatusOK, status)
}

// DeleteIndex handles DELETE /api/gitlab/projects/:project_id/index
func (h *GitLabUserIndexerHandler) DeleteIndex(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID required"})
		return
	}

	ctx := c.Request.Context()

	// Get project
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return
	}

	// Check access
	if !h.canAccessProject(c, project) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you don't have access to this project"})
		return
	}

	branch := c.Query("branch")
	if branch == "" {
		branch = project.DefaultBranch
		if branch == "" {
			branch = "main"
		}
	}

	if err := h.indexer.DeleteIndex(ctx, projectID, branch); err != nil {
		h.logger.WithError(err).Error("Failed to delete index")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete index"})
		return
	}

	userID, _, _ := h.getUserFromContext(c)
	h.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"branch":     branch,
		"user_id":    userID,
	}).Info("User deleted repository index")

	c.JSON(http.StatusOK, gin.H{
		"message":    "Index deleted",
		"project_id": projectID,
		"branch":     branch,
	})
}

// SetIndexer sets the indexer (for late initialization)
func (h *GitLabUserIndexerHandler) SetIndexer(idx *indexer.Indexer) {
	h.indexer = idx
}

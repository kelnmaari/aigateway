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

// GitLabIndexerHandler handles repository indexing requests
type GitLabIndexerHandler struct {
	indexer *indexer.Indexer
	store   storage.Store
	logger  *logrus.Logger
}

// NewGitLabIndexerHandler creates a new indexer handler
func NewGitLabIndexerHandler(idx *indexer.Indexer, store storage.Store, logger *logrus.Logger) *GitLabIndexerHandler {
	return &GitLabIndexerHandler{
		indexer: idx,
		store:   store,
		logger:  logger,
	}
}

// IndexProjectRequest represents request to index a project
type IndexProjectRequest struct {
	Branch string `json:"branch"` // Target branch to index (default: from project settings)
	Force  bool   `json:"force"`  // Force reindex even if already indexed
}

// IndexProject handles POST /api/admin/gitlab/projects/:project_id/index
// Triggers indexing of the project's target branch for RAG
func (h *GitLabIndexerHandler) IndexProject(c *gin.Context) {
	projectID := c.Param("project_id")
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

	// Verify integration exists
	_, err = h.store.GetIntegration(ctx, project.IntegrationID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get integration"})
		return
	}

	// Determine branch to index
	branch := req.Branch
	if branch == "" {
		branch = project.DefaultBranch
		if branch == "" {
			branch = "main" // fallback
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
		// Get integration to create client
		integration, err := h.store.GetIntegration(ctx, project.IntegrationID)
		if err != nil || integration == nil {
			h.logger.WithError(err).Error("Failed to get integration for client creation")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get integration"})
			return
		}

		// Create and register GitLab client
		gitlabClient := client.NewClient(client.ClientConfig{
			BaseURL:     integration.BaseURL,
			AccessToken: integration.AccessToken,
			Timeout:     60 * time.Second,
		})
		h.indexer.RegisterClient(project.IntegrationID, gitlabClient)
		h.logger.WithField("integration_id", project.IntegrationID).Debug("Registered GitLab client for indexer")
	}

	// Start async indexing
	if err := h.indexer.IndexBranch(ctx, indexReq); err != nil {
		h.logger.WithError(err).Error("Failed to start indexing")
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"branch":     branch,
		"force":      req.Force,
	}).Info("Started repository indexing")

	c.JSON(http.StatusAccepted, gin.H{
		"message":    "Indexing started",
		"project_id": projectID,
		"branch":     branch,
		"status":     "in_progress",
	})
}

// GetIndexStatus handles GET /api/admin/gitlab/projects/:project_id/index/status
// Returns the current indexing status for a project
func (h *GitLabIndexerHandler) GetIndexStatus(c *gin.Context) {
	projectID := c.Param("project_id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID required"})
		return
	}

	branch := c.Query("branch")
	if branch == "" {
		// Get project to determine default branch
		ctx := c.Request.Context()
		project, err := h.store.GetProject(ctx, projectID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
		branch = project.DefaultBranch
		if branch == "" {
			branch = "main"
		}
	}

	status := h.indexer.GetStatus(projectID, branch)
	c.JSON(http.StatusOK, status)
}

// DeleteIndex handles DELETE /api/admin/gitlab/projects/:project_id/index
// Deletes the index for a project's branch
func (h *GitLabIndexerHandler) DeleteIndex(c *gin.Context) {
	projectID := c.Param("project_id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project ID required"})
		return
	}

	branch := c.Query("branch")
	ctx := c.Request.Context()

	if branch == "" {
		// Get project to determine default branch
		project, err := h.store.GetProject(ctx, projectID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
			return
		}
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

	h.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"branch":     branch,
	}).Info("Deleted repository index")

	c.JSON(http.StatusOK, gin.H{
		"message":    "Index deleted",
		"project_id": projectID,
		"branch":     branch,
	})
}

// HandleMergeEvent handles MR merge events to trigger reindexing
// Called by webhook handler when MR is merged
func (h *GitLabIndexerHandler) HandleMergeEvent(integration *models.GitLabIntegration, project *models.GitLabProject, targetBranch string) {
	if h.indexer == nil || integration == nil || project == nil {
		return
	}

	indexReq := indexer.IndexRequest{
		IntegrationID:    project.IntegrationID,
		ProjectID:        project.ID,
		GitLabProjectID:  project.GitLabProjectID,
		Branch:           targetBranch,
		EmbeddingModelID: project.EmbeddingModelID,
		CollectionName:   project.GetCollectionName(), // Use project-specific Qdrant collection
		Force:            true,                        // Always force reindex after merge
	}

	// Ensure indexer has a client for this integration
	if _, ok := h.indexer.GetClient(project.IntegrationID); !ok {
		// Create and register GitLab client from integration
		gitlabClient := client.NewClient(client.ClientConfig{
			BaseURL:     integration.BaseURL,
			AccessToken: integration.AccessToken,
			Timeout:     60 * time.Second,
		})
		h.indexer.RegisterClient(project.IntegrationID, gitlabClient)
		h.logger.WithField("integration_id", project.IntegrationID).Debug("Registered GitLab client for indexer (merge event)")
	}

	// Start async reindexing with background context
	if err := h.indexer.IndexBranch(nil, indexReq); err != nil {
		h.logger.WithError(err).Warn("Failed to trigger reindex after merge")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"project_id":    project.ID,
		"target_branch": targetBranch,
	}).Info("Triggered reindex after MR merge")
}

// SetIndexer sets the indexer (for late initialization)
func (h *GitLabIndexerHandler) SetIndexer(idx *indexer.Indexer) {
	h.indexer = idx
}


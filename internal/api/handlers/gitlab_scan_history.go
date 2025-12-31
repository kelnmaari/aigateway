// Package handlers provides HTTP handlers for GitLab scan history
package handlers

import (
	"net/http"
	"strconv"

	"aigateway/internal/gitlab/storage"
	"aigateway/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GitLabScanHistoryHandler handles scan history endpoints
type GitLabScanHistoryHandler struct {
	store  storage.Store
	logger *logrus.Logger
}

// NewGitLabScanHistoryHandler creates a new scan history handler
func NewGitLabScanHistoryHandler(store storage.Store, logger *logrus.Logger) *GitLabScanHistoryHandler {
	return &GitLabScanHistoryHandler{
		store:  store,
		logger: logger,
	}
}

// ListScanResults GET /api/admin/gitlab/scan-history
func (h *GitLabScanHistoryHandler) ListScanResults(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse query parameters
	req := &models.GitLabScanResultsRequest{
		ProjectID: c.Query("project_id"),
	}

	if scanType := c.Query("scan_type"); scanType != "" {
		st := models.GitLabScanType(scanType)
		req.ScanType = &st
	}

	if status := c.Query("status"); status != "" {
		s := models.GitLabScanStatus(status)
		req.Status = &s
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			req.Limit = limit
		}
	}
	if req.Limit == 0 {
		req.Limit = 50
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			req.Offset = offset
		}
	}

	results, total, err := h.store.ListScanResults(ctx, req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list scan results")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list scan results"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   total,
		"limit":   req.Limit,
		"offset":  req.Offset,
	})
}

// GetScanResult GET /api/admin/gitlab/scan-history/:id
func (h *GitLabScanHistoryHandler) GetScanResult(c *gin.Context) {
	ctx := c.Request.Context()
	id := c.Param("id")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Scan result ID is required"})
		return
	}

	result, err := h.store.GetScanResult(ctx, id)
	if err != nil {
		h.logger.WithError(err).WithField("id", id).Error("Failed to get scan result")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get scan result"})
		return
	}

	if result == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan result not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetScanTypes GET /api/admin/gitlab/scan-history/types
func (h *GitLabScanHistoryHandler) GetScanTypes(c *gin.Context) {
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


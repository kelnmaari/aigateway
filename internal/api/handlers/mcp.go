// Package handlers provides MCP server handlers
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// MCPHandler обрабатывает MCP server endpoints
type MCPHandler struct {
	db     storage.Database
	logger *logrus.Logger
}

// NewMCPHandler создает новый MCP handler
func NewMCPHandler(db storage.Database, logger *logrus.Logger) *MCPHandler {
	return &MCPHandler{
		db:     db,
		logger: logger,
	}
}

// ListMCPServers обрабатывает GET /api/mcp/servers (public)
func (h *MCPHandler) ListMCPServers(c *gin.Context) {
	var req models.MCPServerListRequest

	// Parse query params
	req.Limit = parseIntQuery(c, "limit", 50)
	req.Offset = parseIntQuery(c, "offset", 0)
	req.SortBy = c.DefaultQuery("sort_by", "created_at")
	req.SortOrder = c.DefaultQuery("sort_order", "desc")
	req.ActiveOnly = c.DefaultQuery("active_only", "true") == "true"

	if category := c.Query("category"); category != "" {
		req.Category = &category
	}

	if search := c.Query("search"); search != "" {
		req.Search = &search
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	response, err := h.db.ListMCPServers(ctx, req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list MCP servers")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to retrieve MCP servers",
				"type":    "api_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetMCPServer обрабатывает GET /api/mcp/servers/:id (public)
func (h *MCPHandler) GetMCPServer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Server ID is required",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	server, err := h.db.GetMCPServer(ctx, id)
	if err != nil {
		h.logger.WithError(err).WithField("id", id).Error("Failed to get MCP server")
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message": "MCP server not found",
				"type":    "not_found_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, server)
}

// CreateMCPServer обрабатывает POST /api/admin/mcp/servers (admin only)
func (h *MCPHandler) CreateMCPServer(c *gin.Context) {
	var req models.CreateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request body",
				"type":    "invalid_request_error",
				"details": err.Error(),
			},
		})
		return
	}

	// Generate ID
	server := &models.MCPServer{
		ID:                uuid.New().String(),
		Name:              req.Name,
		Description:       req.Description,
		Category:          req.Category,
		InstallationGuide: req.InstallationGuide,
		WebsiteURL:        req.WebsiteURL,
		GitHubURL:         req.GitHubURL,
		Tags:              req.Tags,
		IsActive:          true,
	}

	// Override IsActive if provided
	if req.IsActive != nil {
		server.IsActive = *req.IsActive
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.db.CreateMCPServer(ctx, server); err != nil {
		h.logger.WithError(err).Error("Failed to create MCP server")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to create MCP server",
				"type":    "api_error",
			},
		})
		return
	}

	h.logger.WithField("mcp_server_id", server.ID).Info("MCP server created")
	c.JSON(http.StatusCreated, server)
}

// UpdateMCPServer обрабатывает PUT /api/admin/mcp/servers/:id (admin only)
func (h *MCPHandler) UpdateMCPServer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Server ID is required",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	var req models.UpdateMCPServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request body",
				"type":    "invalid_request_error",
				"details": err.Error(),
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get existing server
	server, err := h.db.GetMCPServer(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message": "MCP server not found",
				"type":    "not_found_error",
			},
		})
		return
	}

	// Apply updates
	if req.Name != nil {
		server.Name = *req.Name
	}
	if req.Description != nil {
		server.Description = *req.Description
	}
	if req.Category != nil {
		server.Category = *req.Category
	}
	if req.InstallationGuide != nil {
		server.InstallationGuide = *req.InstallationGuide
	}
	if req.WebsiteURL != nil {
		server.WebsiteURL = *req.WebsiteURL
	}
	if req.GitHubURL != nil {
		server.GitHubURL = *req.GitHubURL
	}
	if req.Tags != nil {
		server.Tags = req.Tags
	}
	if req.IsActive != nil {
		server.IsActive = *req.IsActive
	}

	if err := h.db.UpdateMCPServer(ctx, server); err != nil {
		h.logger.WithError(err).Error("Failed to update MCP server")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to update MCP server",
				"type":    "api_error",
			},
		})
		return
	}

	h.logger.WithField("mcp_server_id", server.ID).Info("MCP server updated")
	c.JSON(http.StatusOK, server)
}

// DeleteMCPServer обрабатывает DELETE /api/admin/mcp/servers/:id (admin only)
func (h *MCPHandler) DeleteMCPServer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Server ID is required",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if err := h.db.DeleteMCPServer(ctx, id); err != nil {
		h.logger.WithError(err).WithField("id", id).Error("Failed to delete MCP server")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to delete MCP server",
				"type":    "api_error",
			},
		})
		return
	}

	h.logger.WithField("mcp_server_id", id).Info("MCP server deleted")
	c.JSON(http.StatusOK, gin.H{
		"message": "MCP server deleted successfully",
	})
}

// GetMCPCategories обрабатывает GET /api/mcp/categories (public)
func (h *MCPHandler) GetMCPCategories(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"categories": models.GetMCPCategories(),
	})
}

// Package handlers provides HTTP handlers for Agent API (v2.5.0+, v3.0.6+ YZMA compatibility)
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	agentService "aigateway/internal/services/agent"
)

// AgentHandler handles agent-related API endpoints
type AgentHandler struct {
	agentService *agentService.AgentService
	logger       *logrus.Logger
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(agentService *agentService.AgentService, logger *logrus.Logger) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
		logger:       logger,
	}
}

// HandleListTools returns available tools for client-side execution
// GET /api/agent/tools
func (h *AgentHandler) HandleListTools(c *gin.Context) {
	tools := h.agentService.ListTools()

	c.JSON(http.StatusOK, gin.H{
		"tools": tools,
		"total": len(tools),
	})
}

// HandleListToolsByCategory returns tools filtered by category
// GET /api/agent/tools/:category
func (h *AgentHandler) HandleListToolsByCategory(c *gin.Context) {
	category := c.Param("category")

	// Validate category
	validCategories := map[string]bool{
		"file":     true,
		"terminal": true,
		"mcp":      true,
		"system":   true,
	}

	if !validCategories[category] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid category. Must be one of: file, terminal, mcp, system",
		})
		return
	}

	tools := h.agentService.GetToolsByCategory(models.AgentToolCategory(category))

	c.JSON(http.StatusOK, gin.H{
		"tools":    tools,
		"category": category,
		"total":    len(tools),
	})
}


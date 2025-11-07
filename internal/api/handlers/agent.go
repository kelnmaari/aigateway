// Package handlers provides HTTP handlers for Agent API (v2.5.0+)
package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/services/agent"
)

// AgentHandler handles agent-related HTTP requests
type AgentHandler struct {
	agentService *agent.Service
	logger       *logrus.Logger
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(agentService *agent.Service, logger *logrus.Logger) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
		logger:       logger,
	}
}

// CreateSession handles POST /api/agent/plan
// @Summary Create agent session and generate task plan
// @Description Creates a new agent session, analyzes the task, and generates a step-by-step execution plan
// @Tags agent
// @Accept json
// @Produce json
// @Param request body models.CreateAgentSessionRequest true "Agent session request"
// @Success 200 {object} models.AgentSessionResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 429 {object} models.ErrorResponse "Too many sessions"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/agent/plan [post]
func (h *AgentHandler) CreateSession(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse request
	var req models.CreateAgentSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind agent session request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	h.logger.WithField("task", req.Task).Info("Creating agent session")

	// Get user_id from context (handle both string and *string from different middleware)
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	
	// Extract user_id (can be string or *string depending on middleware)
	var userIDStr string
	switch v := userIDRaw.(type) {
	case string:
		userIDStr = v
	case *string:
		if v != nil {
			userIDStr = *v
		} else {
			h.logger.Error("user_id pointer is nil")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
			return
		}
	default:
		h.logger.WithField("type", fmt.Sprintf("%T", userIDRaw)).Error("Invalid user_id type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user context"})
		return
	}

	// Get tenant_id (optional for API keys)
	var tenantIDPtr *string
	if tid, exists := c.Get("tenant_id"); exists && tid != nil {
		switch v := tid.(type) {
		case string:
			tenantIDPtr = &v
		case *string:
			tenantIDPtr = v
		}
	}

	// Create session and generate plan
	session, plan, err := h.agentService.CreateSession(ctx, userIDStr, tenantIDPtr, req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create agent session")
		
		// Check if it's a rate limit error
		if err.Error() == "maximum concurrent sessions reached" {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agent session"})
		return
	}

	c.JSON(http.StatusOK, models.AgentSessionResponse{
		Session: session,
		Plan:    plan,
	})
}

// GetSession handles GET /api/agent/sessions/:id
// @Summary Get agent session details
// @Description Retrieves the current state of an agent session including progress and results
// @Tags agent
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} models.AgentSessionResponse
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 404 {object} models.ErrorResponse "Session not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/agent/sessions/{id} [get]
func (h *AgentHandler) GetSession(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := c.Param("id")

	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	h.logger.WithField("session_id", sessionID).Debug("Getting agent session")

	// Get session
	session, err := h.agentService.GetSession(ctx, sessionID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get agent session")
		
		if err.Error() == "session not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agent session"})
		return
	}

	c.JSON(http.StatusOK, models.AgentSessionResponse{
		Session: session,
		Plan:    session.Plan,
	})
}

// CancelSession handles POST /api/agent/sessions/:id/cancel
// @Summary Cancel agent session
// @Description Cancels a running agent session and stops all pending tasks
// @Tags agent
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} map[string]interface{} "Success response"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 404 {object} models.ErrorResponse "Session not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/agent/sessions/{id}/cancel [post]
func (h *AgentHandler) CancelSession(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := c.Param("id")

	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	h.logger.WithField("session_id", sessionID).Info("Cancelling agent session")

	// Cancel session
	if err := h.agentService.CancelSession(ctx, sessionID); err != nil {
		h.logger.WithError(err).Error("Failed to cancel agent session")
		
		if err.Error() == "session not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel agent session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Session cancelled successfully",
	})
}


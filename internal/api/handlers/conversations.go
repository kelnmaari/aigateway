// Package handlers provides HTTP handlers for Conversations API
package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// ConversationHandler handles conversation-related HTTP requests
type ConversationHandler struct {
	db storage.Database
}

// NewConversationHandler creates a new conversation handler
func NewConversationHandler(db storage.Database) *ConversationHandler {
	return &ConversationHandler{
		db: db,
	}
}

// ========================================
// Request/Response Models
// ========================================

// CreateConversationRequest represents a request to create a new conversation
type CreateConversationRequest struct {
	Title        string   `json:"title" binding:"required,min=1,max=200"`
	Model        string   `json:"model" binding:"required"`
	TenantID     *string  `json:"tenant_id,omitempty"`
	Temperature  *float64 `json:"temperature,omitempty"`
	SystemPrompt string   `json:"system_prompt,omitempty"`
}

// UpdateConversationRequest represents a request to update a conversation
type UpdateConversationRequest struct {
	Title        *string  `json:"title,omitempty"`
	Temperature  *float64 `json:"temperature,omitempty"`
	SystemPrompt *string  `json:"system_prompt,omitempty"`
	IsArchived   *bool    `json:"is_archived,omitempty"`
	IsPinned     *bool    `json:"is_pinned,omitempty"`
}

// ListConversationsResponse represents the response for listing conversations
type ListConversationsResponse struct {
	Conversations []*models.Conversation `json:"conversations"`
	Total         int                    `json:"total"`
	Limit         int                    `json:"limit,omitempty"`
	Offset        int                    `json:"offset,omitempty"`
}

// ConversationResponse represents the response for a single conversation
type ConversationResponse struct {
	Conversation *models.Conversation `json:"conversation"`
}

// ConversationWithMessagesResponse represents a conversation with all messages
type ConversationWithMessagesResponse struct {
	*models.ConversationWithMessages
}

// ========================================
// HTTP Handlers
// ========================================

// CreateConversation godoc
// @Summary Create a new conversation
// @Description Creates a new conversation for the authenticated user
// @Tags Conversations
// @Accept json
// @Produce json
// @Param request body CreateConversationRequest true "Conversation details"
// @Success 201 {object} ConversationResponse
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/conversations [post]
// @Security BearerAuth
func (h *ConversationHandler) CreateConversation(c *gin.Context) {
	// Get user from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Parse request
	var req CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	// Validate temperature if provided
	if req.Temperature != nil && (*req.Temperature < 0 || *req.Temperature > 2) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "temperature must be between 0 and 2"})
		return
	}

	// Create conversation object
	now := time.Now()
	conv := &models.Conversation{
		ID:           fmt.Sprintf("conv_%s", uuid.New().String()),
		Title:        req.Title,
		UserID:       userID.(string),
		TenantID:     req.TenantID,
		Model:        req.Model,
		Temperature:  req.Temperature,
		SystemPrompt: req.SystemPrompt,
		Status:       models.ConversationStatusActive,
		IsArchived:   false,
		IsPinned:     false,
		CreatedAt:    now,
		UpdatedAt:    now,
		MessageCount: 0,
		TotalTokens:  0,
	}

	// Save to database
	if err := h.db.CreateConversation(c.Request.Context(), conv); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create conversation: %v", err)})
		return
	}

	// Return conversation directly (not wrapped)
	c.JSON(http.StatusCreated, conv)
}

// ListConversations godoc
// @Summary List user conversations
// @Description Gets a list of conversations for the authenticated user
// @Tags Conversations
// @Accept json
// @Produce json
// @Param tenant_id query string false "Filter by tenant ID"
// @Param archived query bool false "Filter by archived status"
// @Param pinned query bool false "Filter by pinned status"
// @Param search query string false "Search in conversation titles"
// @Param limit query int false "Maximum number of results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} ListConversationsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/conversations [get]
// @Security BearerAuth
func (h *ConversationHandler) ListConversations(c *gin.Context) {
	// Get user from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Build filters from query params
	filters := models.ConversationFilters{
		UserID: userID.(string),
		Limit:  50, // Default limit
		Offset: 0,
	}

	// Parse query parameters
	if tenantID := c.Query("tenant_id"); tenantID != "" {
		filters.TenantID = &tenantID
	}

	if archived := c.Query("archived"); archived != "" {
		archivedBool := archived == "true"
		filters.IsArchived = &archivedBool
	}

	if pinned := c.Query("pinned"); pinned != "" {
		pinnedBool := pinned == "true"
		filters.IsPinned = &pinnedBool
	}

	if search := c.Query("search"); search != "" {
		filters.Search = search
	}

	if limit := c.Query("limit"); limit != "" {
		var limitInt int
		if _, err := fmt.Sscanf(limit, "%d", &limitInt); err == nil && limitInt > 0 {
			filters.Limit = limitInt
		}
	}

	if offset := c.Query("offset"); offset != "" {
		var offsetInt int
		if _, err := fmt.Sscanf(offset, "%d", &offsetInt); err == nil && offsetInt >= 0 {
			filters.Offset = offsetInt
		}
	}

	// Get conversations from database
	conversations, err := h.db.ListUserConversations(c.Request.Context(), userID.(string), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to list conversations: %v", err)})
		return
	}

	// Return empty array instead of null if no conversations
	if conversations == nil {
		conversations = []*models.Conversation{}
	}

	c.JSON(http.StatusOK, ListConversationsResponse{
		Conversations: conversations,
		Total:         len(conversations),
		Limit:         filters.Limit,
		Offset:        filters.Offset,
	})
}

// GetConversation godoc
// @Summary Get a conversation
// @Description Gets a conversation by ID with all its messages
// @Tags Conversations
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID"
// @Success 200 {object} ConversationWithMessagesResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/conversations/{id} [get]
// @Security BearerAuth
func (h *ConversationHandler) GetConversation(c *gin.Context) {
	// Get user from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Get conversation ID from path
	convID := c.Param("id")

	// Get conversation from database
	conv, err := h.db.GetConversation(c.Request.Context(), convID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}

	// Check if user owns the conversation
	if conv.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Get messages for the conversation
	messages, err := h.db.ListConversationMessages(c.Request.Context(), convID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to load messages: %v", err)})
		return
	}

	// Return conversation with messages
	if messages == nil {
		messages = []*models.Message{}
	}

	c.JSON(http.StatusOK, ConversationWithMessagesResponse{
		ConversationWithMessages: &models.ConversationWithMessages{
			Conversation: conv,
			Messages:     messages,
		},
	})
}

// UpdateConversation godoc
// @Summary Update a conversation
// @Description Updates conversation metadata (title, settings, etc.)
// @Tags Conversations
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID"
// @Param request body UpdateConversationRequest true "Update fields"
// @Success 200 {object} ConversationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/conversations/{id} [put]
// @Security BearerAuth
func (h *ConversationHandler) UpdateConversation(c *gin.Context) {
	// Get user from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Get conversation ID from path
	convID := c.Param("id")

	// Get existing conversation
	conv, err := h.db.GetConversation(c.Request.Context(), convID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}

	// Check if user owns the conversation
	if conv.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Parse request
	var req UpdateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	// Apply updates
	if req.Title != nil {
		if len(*req.Title) == 0 || len(*req.Title) > 200 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title must be between 1 and 200 characters"})
			return
		}
		conv.Title = *req.Title
	}

	if req.Temperature != nil {
		if *req.Temperature < 0 || *req.Temperature > 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "temperature must be between 0 and 2"})
			return
		}
		conv.Temperature = req.Temperature
	}

	if req.SystemPrompt != nil {
		conv.SystemPrompt = *req.SystemPrompt
	}

	if req.IsArchived != nil {
		conv.IsArchived = *req.IsArchived
		if *req.IsArchived {
			conv.Status = models.ConversationStatusArchived
		} else {
			conv.Status = models.ConversationStatusActive
		}
	}

	if req.IsPinned != nil {
		conv.IsPinned = *req.IsPinned
	}

	// Update timestamp
	conv.UpdatedAt = time.Now()

	// Save to database
	if err := h.db.UpdateConversation(c.Request.Context(), conv); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update conversation: %v", err)})
		return
	}

	// Return conversation directly (not wrapped)
	c.JSON(http.StatusOK, conv)
}

// DeleteConversation godoc
// @Summary Delete a conversation
// @Description Permanently deletes a conversation and all its messages
// @Tags Conversations
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID"
// @Success 204 "No Content"
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/conversations/{id} [delete]
// @Security BearerAuth
func (h *ConversationHandler) DeleteConversation(c *gin.Context) {
	// Get user from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Get conversation ID from path
	convID := c.Param("id")

	// Get existing conversation
	conv, err := h.db.GetConversation(c.Request.Context(), convID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}

	// Check if user owns the conversation
	if conv.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Delete from database (cascade deletes messages)
	if err := h.db.DeleteConversation(c.Request.Context(), convID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete conversation: %v", err)})
		return
	}

	c.Status(http.StatusNoContent)
}

// ========================================
// Messages Handlers
// ========================================

// CreateMessageRequest represents a request to create a message
type CreateMessageRequest struct {
	Role    string `json:"role" binding:"required,oneof=user assistant system tool"`
	Content string `json:"content" binding:"required"`
	Model   string `json:"model,omitempty"`
}

// CreateMessage godoc
// @Summary Create a message in a conversation
// @Description Adds a new message to the conversation
// @Tags Conversations
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID"
// @Param request body CreateMessageRequest true "Message content"
// @Success 201 {object} models.Message
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/conversations/{id}/messages [post]
// @Security BearerAuth
func (h *ConversationHandler) CreateMessage(c *gin.Context) {
	// Get user from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Get conversation ID from path
	convID := c.Param("id")

	// Get existing conversation
	conv, err := h.db.GetConversation(c.Request.Context(), convID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}

	// Check if user owns the conversation
	if conv.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Parse request
	var req CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	// Create message object
	now := time.Now()
	msg := &models.Message{
		ID:             fmt.Sprintf("msg_%s", uuid.New().String()),
		ConversationID: convID,
		Role:           models.MessageRole(req.Role),
		Content:        req.Content,
		Model:          req.Model,
		CreatedAt:      now,
	}

	// Save to database
	if err := h.db.CreateMessage(c.Request.Context(), msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create message: %v", err)})
		return
	}

	// Update conversation stats
	conv.MessageCount++
	conv.LastMessageAt = &now
	conv.UpdatedAt = now
	if err := h.db.UpdateConversation(c.Request.Context(), conv); err != nil {
		// Log but don't fail the request
		fmt.Printf("Warning: failed to update conversation stats: %v\n", err)
	}

	c.JSON(http.StatusCreated, msg)
}

// ListMessages godoc
// @Summary List messages in a conversation
// @Description Gets all messages for a conversation
// @Tags Conversations
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID"
// @Success 200 {array} models.Message
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/conversations/{id}/messages [get]
// @Security BearerAuth
func (h *ConversationHandler) ListMessages(c *gin.Context) {
	// Get user from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Get conversation ID from path
	convID := c.Param("id")

	// Get existing conversation
	conv, err := h.db.GetConversation(c.Request.Context(), convID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "conversation not found"})
		return
	}

	// Check if user owns the conversation
	if conv.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Get messages
	messages, err := h.db.ListConversationMessages(c.Request.Context(), convID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to load messages: %v", err)})
		return
	}

	// Return empty array instead of null
	if messages == nil {
		messages = []*models.Message{}
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

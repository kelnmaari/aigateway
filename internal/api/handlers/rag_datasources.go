// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"aigateway/internal/models"
	"aigateway/internal/services/rag"
	"aigateway/internal/storage"
)

// RAGDataSourcesHandler обрабатывает запросы к RAG data sources API
type RAGDataSourcesHandler struct {
	service *rag.DataSourceService
	logger  *logrus.Logger
}

// NewRAGDataSourcesHandler создает новый handler
func NewRAGDataSourcesHandler(service *rag.DataSourceService, logger *logrus.Logger) *RAGDataSourcesHandler {
	return &RAGDataSourcesHandler{
		service: service,
		logger:  logger,
	}
}

// CreateDataSource создает новый источник данных
// POST /api/rag/sources
func (h *RAGDataSourcesHandler) CreateDataSource(c *gin.Context) {
	var req models.CreateRAGDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get user ID from context (set by auth middleware)
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	
	// Get tenant ID if present
	tenantID := getTenantIDFromContext(c)
	
	// Create data source
	source, err := h.service.CreateDataSource(c.Request.Context(), &req, userID, tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create data source")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create data source"})
		return
	}
	
	c.JSON(http.StatusCreated, source)
}

// GetDataSource получает источник по ID
// GET /api/rag/sources/:id
func (h *RAGDataSourcesHandler) GetDataSource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source ID"})
		return
	}
	
	source, err := h.service.GetDataSource(c.Request.Context(), id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get data source")
		c.JSON(http.StatusNotFound, gin.H{"error": "data source not found"})
		return
	}
	
	// Verify ownership
	userID, _ := getUserIDFromContext(c)
	if source.UserID != userID {
		// TODO: check if shared or tenant member
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	
	c.JSON(http.StatusOK, source)
}

// ListDataSources возвращает список источников с фильтрацией
// GET /api/rag/sources?source_type=file&tags=crm&limit=20&offset=0
func (h *RAGDataSourcesHandler) ListDataSources(c *gin.Context) {
	// Get user ID from context
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	
	// Build filter
	filter := storage.DataSourceFilter{
		UserID: &userID,
		Limit:  20,
		Offset: 0,
	}
	
	// Parse query parameters
	if sourceTypeStr := c.Query("source_type"); sourceTypeStr != "" {
		sourceType := models.SourceType(sourceTypeStr)
		filter.SourceType = &sourceType
	}
	
	if statusStr := c.Query("status"); statusStr != "" {
		status := models.SourceStatus(statusStr)
		filter.Status = &status
	}
	
	if tags := c.QueryArray("tags"); len(tags) > 0 {
		filter.Tags = tags
	}
	
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}
	
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}
	
	// Get tenant ID if filtering by tenant
	if tenantIDStr := c.Query("tenant_id"); tenantIDStr != "" {
		if tenantID, err := uuid.Parse(tenantIDStr); err == nil {
			filter.TenantID = &tenantID
		}
	}
	
	// List sources
	sources, total, err := h.service.ListDataSources(c.Request.Context(), filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list data sources")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list data sources"})
		return
	}
	
	c.JSON(http.StatusOK, models.ListRAGDataSourcesResponse{
		Sources: sources,
		Total:   total,
		Limit:   filter.Limit,
		Offset:  filter.Offset,
	})
}

// UpdateDataSource обновляет источник
// PUT /api/rag/sources/:id
func (h *RAGDataSourcesHandler) UpdateDataSource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source ID"})
		return
	}
	
	var req models.UpdateRAGDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Verify ownership
	source, err := h.service.GetDataSource(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "data source not found"})
		return
	}
	
	userID, _ := getUserIDFromContext(c)
	if source.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	
	// Update
	updated, err := h.service.UpdateDataSource(c.Request.Context(), id, &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update data source")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update data source"})
		return
	}
	
	c.JSON(http.StatusOK, updated)
}

// DeleteDataSource удаляет источник
// DELETE /api/rag/sources/:id
func (h *RAGDataSourcesHandler) DeleteDataSource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source ID"})
		return
	}
	
	// Verify ownership
	source, err := h.service.GetDataSource(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "data source not found"})
		return
	}
	
	userID, _ := getUserIDFromContext(c)
	if source.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	
	// Delete
	if err := h.service.DeleteDataSource(c.Request.Context(), id); err != nil {
		h.logger.WithError(err).Error("Failed to delete data source")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete data source"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "data source deleted"})
}

// TestConnection тестирует подключение к источнику
// POST /api/rag/sources/test-connection
func (h *RAGDataSourcesHandler) TestConnection(c *gin.Context) {
	var req models.TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// TODO: implement connection testing based on source type
	// For now, return success placeholder
	c.JSON(http.StatusOK, models.TestConnectionResponse{
		Success:      true,
		Message:      "Connection test not yet implemented",
		ResponseTime: 0,
	})
}

// SyncSource запускает синхронизацию источника
// POST /api/rag/sources/:id/sync
func (h *RAGDataSourcesHandler) SyncSource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source ID"})
		return
	}
	
	// Verify ownership
	source, err := h.service.GetDataSource(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "data source not found"})
		return
	}
	
	userID, _ := getUserIDFromContext(c)
	if source.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}
	
	// TODO: enqueue sync job
	// For now, return placeholder
	c.JSON(http.StatusAccepted, models.SyncSourceResponse{
		JobID:         1,
		Message:       "Sync job queued (not yet implemented)",
		EstimatedTime: "5m",
	})
}

// getUserIDFromContext извлекает user ID из gin context
func getUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.New("user_id not found in context")
	}
	
	userIDStr, ok := userIDInterface.(string)
	if !ok {
		return uuid.Nil, errors.New("user_id is not a string")
	}
	
	// JWT middleware sets user_id with "user_" prefix, strip it if present
	if len(userIDStr) > 5 && userIDStr[:5] == "user_" {
		userIDStr = userIDStr[5:]
	}
	
	return uuid.Parse(userIDStr)
}

// getTenantIDFromContext извлекает tenant ID из gin context (опционально)
func getTenantIDFromContext(c *gin.Context) *uuid.UUID {
	tenantIDInterface, exists := c.Get("tenant_id")
	if !exists {
		return nil
	}
	
	tenantIDStr, ok := tenantIDInterface.(string)
	if !ok {
		return nil
	}
	
	// Strip "tenant_" prefix if present
	if len(tenantIDStr) > 7 && tenantIDStr[:7] == "tenant_" {
		tenantIDStr = tenantIDStr[7:]
	}
	
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return nil
	}
	
	return &tenantID
}



package handlers

import (
	"context"
	"fmt"
	"net/http"

	"aigateway/internal/models"
	"aigateway/internal/providers"
	"aigateway/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ModelUsageChecker interface for checking model usage in integrations
type ModelUsageChecker interface {
	IsModelUsed(ctx context.Context, modelID string) (bool, error)
	FindProjectsByModel(ctx context.Context, modelID string) ([]models.GitLabProjectRef, error)
}

// RegistryHandler обрабатывает запросы к Model Registry API
type RegistryHandler struct {
	db              storage.Database
	providerManager *providers.ProviderManager
	logger          *logrus.Logger
	usageChecker    ModelUsageChecker // Optional: for GitLab integration checks
}

// NewRegistryHandler создает новый RegistryHandler
func NewRegistryHandler(db storage.Database, providerManager *providers.ProviderManager, logger *logrus.Logger) *RegistryHandler {
	return &RegistryHandler{
		db:              db,
		providerManager: providerManager,
		logger:          logger,
	}
}

// SetModelUsageChecker sets the model usage checker for GitLab integration
func (h *RegistryHandler) SetModelUsageChecker(checker ModelUsageChecker) {
	h.usageChecker = checker
}

// ========================================
// Model Providers API
// ========================================

// CreateProvider создает нового provider
// POST /api/admin/registry/providers
func (h *RegistryHandler) CreateProvider(c *gin.Context) {
	var req models.CreateModelProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	provider := &models.ModelProvider{
		Name:         req.Name,
		ProviderType: req.ProviderType,
		BaseURL:      req.BaseURL,
		APIKey:       req.APIKey,
		Enabled:      req.Enabled != nil && *req.Enabled,
		Priority:     100, // Default priority
		Config:       req.Config,
		HealthStatus: models.HealthStatusUnknown,
	}

	if req.Priority != nil {
		provider.Priority = *req.Priority
	}

	if err := h.db.CreateModelProvider(c.Request.Context(), provider); err != nil {
		h.logger.WithError(err).Error("Failed to create provider")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create provider"})
		return
	}

	// После создания provider, загружаем его в ProviderManager
	if provider.Enabled {
		go h.providerManager.LoadProvidersFromDB(c.Request.Context())
	}

	c.JSON(http.StatusCreated, provider)
}

// ListProviders возвращает список providers
// GET /api/admin/registry/providers
func (h *RegistryHandler) ListProviders(c *gin.Context) {
	enabledOnly := c.Query("enabled") == "true"

	providers, err := h.db.ListModelProviders(c.Request.Context(), enabledOnly)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list providers")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list providers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"total":     len(providers),
	})
}

// GetProvider возвращает provider по ID
// GET /api/admin/registry/providers/:id
func (h *RegistryHandler) GetProvider(c *gin.Context) {
	providerID := c.Param("id")

	provider, err := h.db.GetModelProvider(c.Request.Context(), providerID)
	if err != nil {
		h.logger.WithError(err).Errorf("Failed to get provider: %s", providerID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	c.JSON(http.StatusOK, provider)
}

// UpdateProvider обновляет provider
// PUT /api/admin/registry/providers/:id
func (h *RegistryHandler) UpdateProvider(c *gin.Context) {
	providerID := c.Param("id")

	var req models.UpdateModelProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Получаем существующий provider
	provider, err := h.db.GetModelProvider(c.Request.Context(), providerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	// Обновляем поля
	if req.Name != nil {
		provider.Name = *req.Name
	}
	if req.BaseURL != nil {
		provider.BaseURL = *req.BaseURL
	}
	if req.APIKey != nil {
		provider.APIKey = *req.APIKey
	}
	if req.Enabled != nil {
		provider.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		provider.Priority = *req.Priority
	}
	if req.Config != nil {
		provider.Config = req.Config
	}

	if err := h.db.UpdateModelProvider(c.Request.Context(), provider); err != nil {
		h.logger.WithError(err).Error("Failed to update provider")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update provider"})
		return
	}

	// Перезагружаем providers в manager
	go h.providerManager.LoadProvidersFromDB(c.Request.Context())

	c.JSON(http.StatusOK, provider)
}

// DeleteProvider удаляет provider
// DELETE /api/admin/registry/providers/:id
func (h *RegistryHandler) DeleteProvider(c *gin.Context) {
	providerID := c.Param("id")

	if err := h.db.DeleteModelProvider(c.Request.Context(), providerID); err != nil {
		h.logger.WithError(err).Errorf("Failed to delete provider: %s", providerID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete provider"})
		return
	}

	// Перезагружаем providers в manager
	go h.providerManager.LoadProvidersFromDB(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted successfully"})
}

// HealthCheckProviders проверяет health всех providers
// GET /api/admin/registry/providers/health
func (h *RegistryHandler) HealthCheckProviders(c *gin.Context) {
	results := h.providerManager.HealthCheckAll(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"providers": results,
		"total":     len(results),
	})
}

// ========================================
// Model Registry API
// ========================================

// ListModels возвращает список моделей из registry
// GET /api/models/registry
func (h *RegistryHandler) ListModels(c *gin.Context) {
	// Parse filters from query params
	filter := &models.ModelRegistryFilter{
		ProviderID:   c.Query("provider_id"),
		Status:       models.ModelStatus(c.Query("status")),
		HealthStatus: models.ModelHealthStatus(c.Query("health")),
		Tag:          c.Query("tag"),
	}

	// Parse provider_type
	if providerType := c.Query("provider_type"); providerType != "" {
		filter.ProviderType = models.ModelProviderType(providerType)
	}

	// Parse requires_gpu
	if requiresGPU := c.Query("requires_gpu"); requiresGPU == "true" {
		val := true
		filter.RequiresGPU = &val
	} else if requiresGPU == "false" {
		val := false
		filter.RequiresGPU = &val
	}

	models, err := h.db.ListModelRegistry(c.Request.Context(), filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list models")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list models"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"models": models,
		"total":  len(models),
	})
}

// GetModel возвращает модель по ID
// GET /api/models/registry/:id
func (h *RegistryHandler) GetModel(c *gin.Context) {
	modelID := c.Param("id")

	model, err := h.db.GetModelRegistry(c.Request.Context(), modelID)
	if err != nil {
		h.logger.WithError(err).Errorf("Failed to get model: %s", modelID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	c.JSON(http.StatusOK, model)
}

// RegisterModel регистрирует новую модель вручную
// POST /api/admin/registry/models
func (h *RegistryHandler) RegisterModel(c *gin.Context) {
	var req models.CreateModelRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	model := &models.ModelRegistry{
		ModelID:       req.ModelID,
		ModelName:     req.ModelName,
		ProviderID:    req.ProviderID,
		Capabilities:  req.Capabilities,
		Parameters:    req.Parameters,
		RequiresGPU:   req.RequiresGPU != nil && *req.RequiresGPU,
		MinVRAMGB:     req.MinVRAMGB,
		ContextLength: req.ContextLength,
		Description:   req.Description,
		Tags:          req.Tags,
		Status:        models.ModelStatusActive,
		HealthStatus:  models.HealthStatusUnknown,
	}

	if err := h.db.CreateModelRegistry(c.Request.Context(), model); err != nil {
		h.logger.WithError(err).Error("Failed to register model")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register model"})
		return
	}

	c.JSON(http.StatusCreated, model)
}

// UpdateModel обновляет модель
// PUT /api/admin/registry/models/:id
func (h *RegistryHandler) UpdateModel(c *gin.Context) {
	modelID := c.Param("id")

	var req models.UpdateModelRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Получаем существующую модель
	model, err := h.db.GetModelRegistry(c.Request.Context(), modelID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	// Check if model is being deactivated and is used in GitLab
	if req.Status != nil && *req.Status != models.ModelStatusActive && model.Status == models.ModelStatusActive {
		if h.usageChecker != nil {
			isUsed, err := h.usageChecker.IsModelUsed(c.Request.Context(), modelID)
			if err != nil {
				h.logger.WithError(err).Warn("Failed to check model usage in GitLab")
				// Continue anyway - don't block if check fails
			} else if isUsed {
				// Get projects using this model for error message
				projects, _ := h.usageChecker.FindProjectsByModel(c.Request.Context(), modelID)
				projectCount := len(projects)
				c.JSON(http.StatusConflict, gin.H{
					"error":          "Cannot deactivate model",
					"reason":         fmt.Sprintf("Model is used in %d GitLab project(s)", projectCount),
					"gitlab_usage":   true,
					"project_count":  projectCount,
					"projects":       projects,
					"resolution":     "Change the model in these GitLab projects before deactivating",
				})
				return
			}
		}
	}

	// Обновляем поля
	if req.ModelName != nil {
		model.ModelName = *req.ModelName
	}
	if req.ProviderID != nil {
		model.ProviderID = *req.ProviderID
	}
	if req.Capabilities != nil {
		model.Capabilities = req.Capabilities
	}
	if req.Parameters != nil {
		model.Parameters = req.Parameters
	}
	if req.Status != nil {
		model.Status = *req.Status
	}
	if req.RequiresGPU != nil {
		model.RequiresGPU = *req.RequiresGPU
	}
	if req.MinVRAMGB != nil {
		model.MinVRAMGB = req.MinVRAMGB
	}
	if req.ContextLength != nil {
		model.ContextLength = req.ContextLength
	}
	if req.Description != nil {
		model.Description = *req.Description
	}
	if req.Tags != nil {
		model.Tags = req.Tags
	}

	if err := h.db.UpdateModelRegistry(c.Request.Context(), model); err != nil {
		h.logger.WithError(err).Error("Failed to update model")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update model"})
		return
	}

	c.JSON(http.StatusOK, model)
}

// DeleteModel удаляет модель из registry
// DELETE /api/admin/registry/models/:id
func (h *RegistryHandler) DeleteModel(c *gin.Context) {
	modelID := c.Param("id")

	// Check if model is used in GitLab before deletion
	if h.usageChecker != nil {
		isUsed, err := h.usageChecker.IsModelUsed(c.Request.Context(), modelID)
		if err != nil {
			h.logger.WithError(err).Warn("Failed to check model usage in GitLab")
			// Continue anyway - don't block if check fails
		} else if isUsed {
			// Get projects using this model for error message
			projects, _ := h.usageChecker.FindProjectsByModel(c.Request.Context(), modelID)
			projectCount := len(projects)
			c.JSON(http.StatusConflict, gin.H{
				"error":          "Cannot delete model",
				"reason":         fmt.Sprintf("Model is used in %d GitLab project(s)", projectCount),
				"gitlab_usage":   true,
				"project_count":  projectCount,
				"projects":       projects,
				"resolution":     "Change the model in these GitLab projects before deleting",
			})
			return
		}
	}

	if err := h.db.DeleteModelRegistry(c.Request.Context(), modelID); err != nil {
		h.logger.WithError(err).Errorf("Failed to delete model: %s", modelID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete model"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Model deleted successfully"})
}

// DiscoverModels запускает auto-discovery моделей от providers
// POST /api/admin/registry/discover
func (h *RegistryHandler) DiscoverModels(c *gin.Context) {
	totalDiscovered, err := h.providerManager.DiscoverModels(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to discover models")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to discover models"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Model discovery completed",
		"discovered": totalDiscovered,
	})
}

// GetStats возвращает статистику registry
// GET /api/models/registry/stats
func (h *RegistryHandler) GetStats(c *gin.Context) {
	stats, err := h.db.GetModelRegistryStats(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get registry stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}


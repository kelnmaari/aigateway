// Package ui provides HTMX UI handlers
// Version: v3.0.0 - YZMA-UI-01: yzma model management UI
package ui

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aigateway/internal/storage"
	"aigateway/internal/web/templates"
	"aigateway/internal/yzma"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// YzmaUIHandler handles yzma model management UI
type YzmaUIHandler struct {
	client   *yzma.Client
	db       storage.Database
	renderer *templates.Renderer
	logger   *logrus.Logger
}

// NewYzmaUIHandler creates a new yzma UI handler
func NewYzmaUIHandler(client *yzma.Client, db storage.Database, renderer *templates.Renderer, logger *logrus.Logger) *YzmaUIHandler {
	return &YzmaUIHandler{
		client:   client,
		db:       db,
		renderer: renderer,
		logger:   logger,
	}
}

// GetModelsList returns HTMX fragment with available GGUF models
func (h *YzmaUIHandler) GetModelsList(c *gin.Context) {
	ctx := c.Request.Context()
	
	// List all available models from filesystem
	models, err := h.client.ListAvailableModels()
	if err != nil {
		h.logger.WithError(err).Error("Failed to list available models")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list models"})
		return
	}
	
	// Get loaded models from database (persistent storage)
	loadedModelsDB, err := h.db.ListLoadedModels(ctx, false)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list loaded models from DB")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check loaded status"})
		return
	}
	
	// Create map for quick lookup
	loadedMap := make(map[string]bool)
	for _, model := range loadedModelsDB {
		loadedMap[model.ModelPath] = true
	}
	
	// Prepare model data
	modelsList := make([]map[string]interface{}, 0, len(models))
	for _, modelPath := range models {
		isPartialDownload := strings.HasSuffix(modelPath, ".part")
		modelsList = append(modelsList, map[string]interface{}{
			"path":              modelPath,
			"name":              filepath.Base(modelPath),
			"loaded":            loadedMap[modelPath],
			"provider":          "yzma",
			"is_partial":        isPartialDownload,
		})
	}
	
	data := map[string]interface{}{
		"models": modelsList,
		"count":  len(modelsList),
	}
	
	var buf bytes.Buffer
	if err := h.renderer.RenderInlineTemplate(&buf, "yzma_models_list", data); err != nil {
		h.logger.WithError(err).Error("Failed to render template")
		c.String(http.StatusInternalServerError, "Failed to render models list")
		return
	}
	
	c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}

// GetLoadedModelsList returns HTMX fragment with loaded models
func (h *YzmaUIHandler) GetLoadedModelsList(c *gin.Context) {
	loadedModels := h.client.ListLoadedModels()
	
	modelsList := make([]map[string]interface{}, 0, len(loadedModels))
	for _, modelPath := range loadedModels {
		modelsList = append(modelsList, map[string]interface{}{
			"path": modelPath,
			"name": filepath.Base(modelPath),
		})
	}
	
	data := map[string]interface{}{
		"models": modelsList,
		"count":  len(modelsList),
	}
	
	var buf bytes.Buffer
	if err := h.renderer.RenderInlineTemplate(&buf, "yzma_loaded_models", data); err != nil {
		h.logger.WithError(err).Error("Failed to render template")
		c.String(http.StatusInternalServerError, "Failed to render loaded models")
		return
	}
	
	c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}

// PostLoadModel loads a model into memory (HTMX action)
func (h *YzmaUIHandler) PostLoadModel(c *gin.Context) {
	// Try to get model_path from form data or JSON
	var modelPath string
	
	// First try form data
	modelPath = c.PostForm("model_path")
	
	// If not found, try JSON body
	if modelPath == "" {
		var req struct {
			ModelPath string `json:"model_path"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			modelPath = req.ModelPath
		}
	}
	
	if modelPath == "" {
		h.logger.Error("model_path is required but not provided")
		c.String(http.StatusBadRequest, "model_path is required")
		return
	}
	
	h.logger.WithField("model_path", modelPath).Info("Loading model via UI")
	
	// Load model
	err := h.client.LoadModel(c.Request.Context(), modelPath)
	if err != nil {
		h.logger.WithError(err).Error("Failed to load model")
		c.Header("HX-Trigger", `{"showNotification": {"message": "Failed to load model: `+err.Error()+`", "type": "error"}}`)
		c.String(http.StatusInternalServerError, "Failed to load model")
		return
	}
	
	// Success notification
	c.Header("HX-Trigger", `{"showNotification": {"message": "Model loaded successfully", "type": "success"}, "refreshModels": true}`)
	
	// Return updated model card
	data := map[string]interface{}{
		"path":   modelPath,
		"name":   filepath.Base(modelPath),
		"loaded": true,
	}
	
	var buf bytes.Buffer
	if err := h.renderer.RenderInlineTemplate(&buf, "yzma_model_card", data); err != nil {
		h.logger.WithError(err).Error("Failed to render template")
		c.String(http.StatusInternalServerError, "Failed to render")
		return
	}
	
	c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}

// PostUnloadModel unloads a model from memory (HTMX action)
func (h *YzmaUIHandler) PostUnloadModel(c *gin.Context) {
	// Try to get model_path from form data or JSON
	var modelPath string
	
	// First try form data
	modelPath = c.PostForm("model_path")
	
	// If not found, try JSON body
	if modelPath == "" {
		var req struct {
			ModelPath string `json:"model_path"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			modelPath = req.ModelPath
		}
	}
	
	if modelPath == "" {
		h.logger.Error("model_path is required but not provided")
		c.String(http.StatusBadRequest, "model_path is required")
		return
	}
	
	h.logger.WithField("model_path", modelPath).Info("Unloading model via UI")
	
	// Unload model
	err := h.client.UnloadModel(modelPath)
	if err != nil {
		h.logger.WithError(err).Error("Failed to unload model")
		c.Header("HX-Trigger", `{"showNotification": {"message": "Failed to unload model: `+err.Error()+`", "type": "error"}}`)
		c.String(http.StatusInternalServerError, "Failed to unload model")
		return
	}
	
	// Success notification
	c.Header("HX-Trigger", `{"showNotification": {"message": "Model unloaded successfully", "type": "success"}, "refreshModels": true}`)
	
	// Return updated model card
	data := map[string]interface{}{
		"path":   modelPath,
		"name":   filepath.Base(modelPath),
		"loaded": false,
	}
	
	var buf bytes.Buffer
	if err := h.renderer.RenderInlineTemplate(&buf, "yzma_model_card", data); err != nil {
		h.logger.WithError(err).Error("Failed to render template")
		c.String(http.StatusInternalServerError, "Failed to render")
		return
	}
	
	c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}

// PostDeleteModel permanently deletes a model file from disk (HTMX action)
func (h *YzmaUIHandler) PostDeleteModel(c *gin.Context) {
	// Try to get model_path from form data or JSON
	var modelPath string
	
	// First try form data
	modelPath = c.PostForm("model_path")
	
	// If not found, try JSON body
	if modelPath == "" {
		var req struct {
			ModelPath string `json:"model_path"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			modelPath = req.ModelPath
		}
	}
	
	if modelPath == "" {
		h.logger.Error("model_path is required but not provided")
		c.String(http.StatusBadRequest, "model_path is required")
		return
	}
	
	h.logger.WithField("model_path", modelPath).Info("Deleting model via UI")
	
	// Check if model is currently loaded
	loadedModels := h.client.ListLoadedModels()
	for _, loaded := range loadedModels {
		if loaded == modelPath {
			h.logger.WithField("model_path", modelPath).Warn("Cannot delete loaded model")
			c.Header("HX-Trigger", `{"showNotification": {"message": "Cannot delete model while it is loaded. Unload it first.", "type": "error"}}`)
			c.String(http.StatusBadRequest, "Model is currently loaded")
			return
		}
	}
	
	// Delete model from persistent storage first (if exists)
	ctx := c.Request.Context()
	if err := h.db.RemoveLoadedModel(ctx, modelPath); err != nil {
		h.logger.WithError(err).Warn("Failed to remove model from DB (may not exist in DB)")
		// Continue anyway - file deletion is more important
	}
	
	// Resolve full path (model path might be relative to models directory)
	fullPath := h.client.GetFullModelPath(modelPath)
	
	h.logger.WithFields(logrus.Fields{
		"model_path": modelPath,
		"full_path":  fullPath,
	}).Info("Resolved full path for deletion")
	
	// Delete the file from disk
	if err := os.Remove(fullPath); err != nil {
		h.logger.WithError(err).WithField("full_path", fullPath).Error("Failed to delete model file")
		c.Header("HX-Trigger", `{"showNotification": {"message": "Failed to delete model file: `+err.Error()+`", "type": "error"}}`)
		c.String(http.StatusInternalServerError, "Failed to delete model file")
		return
	}
	
	// Success notification
	fileName := filepath.Base(modelPath)
	c.Header("HX-Trigger", `{"showNotification": {"message": "Model '`+fileName+`' deleted successfully", "type": "success"}, "refreshModels": true}`)
	
	// Return empty HTML to remove the card from UI
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(""))
}

// GetStats returns HTMX fragment with yzma statistics
func (h *YzmaUIHandler) GetStats(c *gin.Context) {
	requests, tokens := h.client.GetStats()
	loadedModels := h.client.ListLoadedModels()
	
	data := map[string]interface{}{
		"total_requests": requests,
		"total_tokens":   tokens,
		"loaded_count":   len(loadedModels),
		"timestamp":      time.Now().Format("15:04:05"),
	}
	
	var buf bytes.Buffer
	if err := h.renderer.RenderInlineTemplate(&buf, "yzma_stats", data); err != nil {
		h.logger.WithError(err).Error("Failed to render stats")
		c.String(http.StatusInternalServerError, "Failed to render stats")
		return
	}
	
	c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}

// GetProviderModels returns models list for chat provider selector
func (h *YzmaUIHandler) GetProviderModels(c *gin.Context) {
	models, err := h.client.ListAvailableModels()
	if err != nil {
		h.logger.WithError(err).Error("Failed to list models for provider")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list models"})
		return
	}
	
	// Return JSON for JavaScript consumption
	c.JSON(http.StatusOK, gin.H{
		"provider": "yzma",
		"models":   models,
	})
}


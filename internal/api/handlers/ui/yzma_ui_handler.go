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
// Also supports JSON response for SvelteKit frontend (Accept: application/json)
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
		
		// Get full path for file operations
		fullPath := h.client.GetFullModelPath(modelPath)
		
		// Get file info for size - skip if file doesn't exist
		info, err := os.Stat(fullPath)
		if err != nil {
			h.logger.WithFields(logrus.Fields{
				"model_path": modelPath,
				"full_path":  fullPath,
				"error":      err.Error(),
			}).Warn("Model file not found, skipping")
			continue // Skip files that don't exist
		}
		
		modelsList = append(modelsList, map[string]interface{}{
			"path":              modelPath,
			"name":              filepath.Base(modelPath),
			"loaded":            loadedMap[modelPath],
			"provider":          "yzma",
			"is_partial":        isPartialDownload,
			"size":              info.Size(),
		})
	}
	
	// Check if JSON response is requested (SvelteKit frontend)
	if strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusOK, gin.H{
			"models": modelsList,
			"count":  len(modelsList),
		})
		return
	}
	
	// Render HTML (HTMX)
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
// Also supports JSON response for SvelteKit frontend
func (h *YzmaUIHandler) GetLoadedModelsList(c *gin.Context) {
	loadedModels := h.client.ListLoadedModels()
	
	modelsList := make([]map[string]interface{}, 0, len(loadedModels))
	for modelPath, modelInfo := range loadedModels {
		modelsList = append(modelsList, map[string]interface{}{
			"path":  modelPath,
			"name":  filepath.Base(modelPath),
			"alias": modelInfo["alias"],
		})
	}
	
	// Check if JSON response is requested (SvelteKit frontend)
	if strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusOK, gin.H{
			"models": modelsList,
			"count":  len(modelsList),
		})
		return
	}
	
	// Render HTML (HTMX)
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
// Also supports JSON response for SvelteKit frontend
func (h *YzmaUIHandler) PostLoadModel(c *gin.Context) {
	// Try to get model from form data or JSON
	var modelPath string
	var modelName string
	isJSONRequest := c.ContentType() == "application/json"
	
	if isJSONRequest {
		var req struct {
			Model     string `json:"model"`
			ModelPath string `json:"model_path"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			modelPath = req.ModelPath
			modelName = req.Model
			if modelPath == "" && modelName != "" {
				modelPath = modelName
			}
		}
	} else {
		modelPath = c.PostForm("model_path")
	}
	
	if modelPath == "" {
		h.logger.Error("model_path is required but not provided")
		if isJSONRequest || strings.Contains(c.GetHeader("Accept"), "application/json") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "model_path is required"})
			return
		}
		c.String(http.StatusBadRequest, "model_path is required")
		return
	}
	
	h.logger.WithField("model_path", modelPath).Info("Loading model via UI")
	
	// Load model
	err := h.client.LoadModel(c.Request.Context(), modelPath)
	if err != nil {
		h.logger.WithError(err).Error("Failed to load model")
		if isJSONRequest || strings.Contains(c.GetHeader("Accept"), "application/json") {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Header("HX-Trigger", `{"showNotification": {"message": "Failed to load model: `+err.Error()+`", "type": "error"}}`)
		c.String(http.StatusInternalServerError, "Failed to load model")
		return
	}
	
	// Check if JSON response is requested (SvelteKit frontend)
	if isJSONRequest || strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusOK, gin.H{
			"message": "Model loaded successfully",
			"path":    modelPath,
			"name":    filepath.Base(modelPath),
		})
		return
	}
	
	// Success notification (HTMX)
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
// Also supports JSON response for SvelteKit frontend
func (h *YzmaUIHandler) PostUnloadModel(c *gin.Context) {
	var modelPath string
	var modelName string
	isJSONRequest := c.ContentType() == "application/json"
	
	if isJSONRequest {
		var req struct {
			Model     string `json:"model"`
			ModelPath string `json:"model_path"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			modelPath = req.ModelPath
			modelName = req.Model
			if modelPath == "" && modelName != "" {
				modelPath = modelName
			}
		}
	} else {
		modelPath = c.PostForm("model_path")
	}
	
	if modelPath == "" {
		h.logger.Error("model_path is required but not provided")
		if isJSONRequest || strings.Contains(c.GetHeader("Accept"), "application/json") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "model_path is required"})
			return
		}
		c.String(http.StatusBadRequest, "model_path is required")
		return
	}
	
	h.logger.WithField("model_path", modelPath).Info("Unloading model via UI")
	
	// Unload model
	err := h.client.UnloadModel(modelPath)
	if err != nil {
		h.logger.WithError(err).Error("Failed to unload model")
		if isJSONRequest || strings.Contains(c.GetHeader("Accept"), "application/json") {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Header("HX-Trigger", `{"showNotification": {"message": "Failed to unload model: `+err.Error()+`", "type": "error"}}`)
		c.String(http.StatusInternalServerError, "Failed to unload model")
		return
	}
	
	// Check if JSON response is requested (SvelteKit frontend)
	if isJSONRequest || strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusOK, gin.H{
			"message": "Model unloaded successfully",
			"path":    modelPath,
			"name":    filepath.Base(modelPath),
		})
		return
	}
	
	// Success notification (HTMX)
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
// Also supports JSON response for SvelteKit frontend
func (h *YzmaUIHandler) PostDeleteModel(c *gin.Context) {
	var modelPath string
	var modelName string
	isJSONRequest := c.ContentType() == "application/json"
	
	if isJSONRequest {
		var req struct {
			Model     string `json:"model"`
			ModelPath string `json:"model_path"`
		}
		if err := c.ShouldBindJSON(&req); err == nil {
			modelPath = req.ModelPath
			modelName = req.Model
			if modelPath == "" && modelName != "" {
				modelPath = modelName
			}
		}
	} else {
		modelPath = c.PostForm("model_path")
	}
	
	if modelPath == "" {
		h.logger.Error("model_path is required but not provided")
		if isJSONRequest || strings.Contains(c.GetHeader("Accept"), "application/json") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "model_path is required"})
			return
		}
		c.String(http.StatusBadRequest, "model_path is required")
		return
	}
	
	h.logger.WithField("model_path", modelPath).Info("Deleting model via UI")
	
	// Check if model is currently loaded
	loadedModels := h.client.ListLoadedModels()
	for loadedPath := range loadedModels {
		if loadedPath == modelPath {
			h.logger.WithField("model_path", modelPath).Warn("Cannot delete loaded model")
			if isJSONRequest || strings.Contains(c.GetHeader("Accept"), "application/json") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete model while it is loaded. Unload it first."})
				return
			}
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
	
	// Delete the file from disk (if it exists)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - that's OK, DB record was already removed
			h.logger.WithField("full_path", fullPath).Info("Model file not found, but DB record removed - considering as success")
		} else {
			// Real error (permissions, etc.)
			h.logger.WithError(err).WithField("full_path", fullPath).Error("Failed to delete model file")
			if isJSONRequest || strings.Contains(c.GetHeader("Accept"), "application/json") {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.Header("HX-Trigger", `{"showNotification": {"message": "Failed to delete model file: `+err.Error()+`", "type": "error"}}`)
			c.String(http.StatusInternalServerError, "Failed to delete model file")
			return
		}
	}
	
	// Check if JSON response is requested (SvelteKit frontend)
	fileName := filepath.Base(modelPath)
	if isJSONRequest || strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusOK, gin.H{
			"message": "Model deleted successfully",
			"name":    fileName,
		})
		return
	}
	
	// Success notification (HTMX)
	c.Header("HX-Trigger", `{"showNotification": {"message": "Model '`+fileName+`' deleted successfully", "type": "success"}, "refreshModels": true}`)
	
	// Return empty HTML to remove the card from UI
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(""))
}

// GetStats returns HTMX fragment with yzma statistics
// Also supports JSON response for SvelteKit frontend
func (h *YzmaUIHandler) GetStats(c *gin.Context) {
	requests, tokens := h.client.GetStats()
	loadedModels := h.client.ListLoadedModels()
	
	// Calculate total models and total size
	models, _ := h.client.ListAvailableModels()
	var totalSize int64
	for _, modelPath := range models {
		if info, err := os.Stat(modelPath); err == nil {
			totalSize += info.Size()
		}
	}
	
	data := map[string]interface{}{
		"total_requests": requests,
		"total_tokens":   tokens,
		"loaded_count":   len(loadedModels),
		"models_count":   len(models),
		"total_size":     totalSize,
		"timestamp":      time.Now().Format("15:04:05"),
	}
	
	// Check if JSON response is requested (SvelteKit frontend)
	if strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusOK, data)
		return
	}
	
	// Render HTML (HTMX)
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

// GetModelMetadata returns detailed metadata for a loaded model
// GET /api/ui/yzma/metadata/*model_path
func (h *YzmaUIHandler) GetModelMetadata(c *gin.Context) {
	modelPath := c.Param("model_path")
	// Remove leading slash if present
	if len(modelPath) > 0 && modelPath[0] == '/' {
		modelPath = modelPath[1:]
	}
	
	// Normalize path separators (Windows paths may have backslashes)
	modelPath = strings.ReplaceAll(modelPath, "\\", "/")
	
	if modelPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model_path is required"})
		return
	}
	
	h.logger.WithField("model_path", modelPath).Debug("Getting model metadata")
	
	// Check if model is loaded
	if !h.client.IsModelLoaded(modelPath) {
		// Model not loaded - return minimal info from file
		fullPath := h.client.GetFullModelPath(modelPath)
		info, err := os.Stat(fullPath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"loaded":          false,
			"path":            modelPath,
			"name":            filepath.Base(modelPath),
			"file_size_bytes": info.Size(),
			"modified_at":     info.ModTime(),
			"message":         "Load model to see full metadata (architecture, parameters, etc.)",
		})
		return
	}
	
	// Get full metadata from loaded model
	metadata, err := h.client.GetModelMetadata(modelPath)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get model metadata")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"loaded":   true,
		"path":     modelPath,
		"name":     filepath.Base(modelPath),
		"metadata": metadata,
	})
}

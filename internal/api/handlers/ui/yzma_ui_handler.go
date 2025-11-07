// Package ui provides HTMX UI handlers
// Version: v3.0.0 - YZMA-UI-01: yzma model management UI
package ui

import (
	"bytes"
	"net/http"
	"path/filepath"
	"time"

	"aigateway/internal/web/templates"
	"aigateway/internal/yzma"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// YzmaUIHandler handles yzma model management UI
type YzmaUIHandler struct {
	client   *yzma.Client
	renderer *templates.Renderer
	logger   *logrus.Logger
}

// NewYzmaUIHandler creates a new yzma UI handler
func NewYzmaUIHandler(client *yzma.Client, renderer *templates.Renderer, logger *logrus.Logger) *YzmaUIHandler {
	return &YzmaUIHandler{
		client:   client,
		renderer: renderer,
		logger:   logger,
	}
}

// GetModelsList returns HTMX fragment with available GGUF models
func (h *YzmaUIHandler) GetModelsList(c *gin.Context) {
	models, err := h.client.ListAvailableModels()
	if err != nil {
		h.logger.WithError(err).Error("Failed to list available models")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list models"})
		return
	}
	
	loadedModels := h.client.ListLoadedModels()
	loadedMap := make(map[string]bool)
	for _, model := range loadedModels {
		loadedMap[model] = true
	}
	
	// Prepare model data
	modelsList := make([]map[string]interface{}, 0, len(models))
	for _, modelPath := range models {
		modelsList = append(modelsList, map[string]interface{}{
			"path":     modelPath,
			"name":     filepath.Base(modelPath),
			"loaded":   loadedMap[modelPath],
			"provider": "yzma",
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
	modelPath := c.PostForm("model_path")
	if modelPath == "" {
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
	modelPath := c.PostForm("model_path")
	if modelPath == "" {
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


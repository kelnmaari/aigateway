// Package ui provides UI handlers for admin panel
// Version: v3.0.0 - Hugging Face model browser
package ui

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"aigateway/internal/huggingface"
	"aigateway/internal/web/templates"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// HuggingFaceUIHandler handles Hugging Face UI endpoints
type HuggingFaceUIHandler struct {
	hfClient   *huggingface.Client
	downloader *huggingface.Downloader
	renderer   *templates.Renderer
	logger     *logrus.Logger
}

// NewHuggingFaceUIHandler creates a new Hugging Face UI handler
func NewHuggingFaceUIHandler(hfClient *huggingface.Client, downloader *huggingface.Downloader, renderer *templates.Renderer, logger *logrus.Logger) *HuggingFaceUIHandler {
	return &HuggingFaceUIHandler{
		hfClient:   hfClient,
		downloader: downloader,
		renderer:   renderer,
		logger:     logger,
	}
}

// GetModelsSearch handles HTMX search for models
// Also supports JSON response for SvelteKit frontend (Accept: application/json)
func (h *HuggingFaceUIHandler) GetModelsSearch(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	// Parse filters
	search := c.Query("search")
	if search == "" {
		search = c.Query("q") // SvelteKit uses 'q' parameter
	}
	author := c.Query("author")
	tagsStr := c.Query("tags")
	providerFilter := c.DefaultQuery("provider", "all")
	sortBy := c.DefaultQuery("sort", "downloads")
	limitStr := c.DefaultQuery("limit", "30")
	
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	
	// Parse tags
	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	}
	
	// Apply provider-specific filters
	var library string
	switch providerFilter {
	case "llama.cpp":
		// GGUF models for llama.cpp - use library filter instead of tag
		// Many GGUF repos don't have the "gguf" tag but are indexed as gguf library
		library = "gguf"
	case "vllm", "sglang", "tgi":
		// Transformer models for vLLM/SGLang/TGI
		tags = append(tags, "text-generation")
		library = "transformers"
	case "embedding":
		// Embedding models (feature-extraction pipeline)
		tags = append(tags, "feature-extraction")
	case "tei":
		// TEI supports both embedding and reranking models
		// sentence-similarity covers both use cases on HuggingFace
		tags = append(tags, "sentence-similarity")
	case "rerank":
		// Reranking models (cross-encoder / reranker)
		tags = append(tags, "text-classification")
	default:
		// "all" - show text-generation models (most common for inference)
		tags = append(tags, "text-generation")
	}

	// Build filters
	filters := huggingface.ModelFilters{
		Search:       search,
		Author:       author,
		Tags:         tags,
		Library:      library,
		Sort:         sortBy,
		Direction:    -1, // Descending
		Limit:        limit,
		FullResponse: true,
		CardData:     true,
	}
	
	h.logger.WithFields(logrus.Fields{
		"search": search,
		"author": author,
		"tags":   tags,
		"sort":   sortBy,
		"limit":  limit,
	}).Debug("Searching Hugging Face models")
	
	// Search models
	models, err := h.hfClient.SearchModels(ctx, filters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to search models")
		statusCode := hfErrorStatusCode(err)
		// Check if JSON response is requested
		if strings.Contains(c.GetHeader("Accept"), "application/json") {
			c.JSON(statusCode, gin.H{"error": err.Error()})
			return
		}
		h.renderError(c, "Failed to search models: "+err.Error())
		return
	}
	
	// For llama.cpp filter, library=gguf was already passed to API
	// All returned models should be GGUF models, mark them as such
	if providerFilter == "llama.cpp" {
		for i := range models {
			models[i].HasGGUF = true // API with library=gguf only returns GGUF models
		}
		h.logger.WithField("count", len(models)).Debug("Marked all models as GGUF (library=gguf filter applied)")
	}
	
	// Check if JSON response is requested (SvelteKit frontend)
	if strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusOK, gin.H{
			"models": models,
			"count":  len(models),
		})
		return
	}
	
	// Render HTML results (HTMX)
	data := map[string]interface{}{
		"Models": models,
		"Count":  len(models),
	}
	
	if err := h.renderer.RenderInlineTemplate(c.Writer, "hf_models_list", data); err != nil {
		h.logger.WithError(err).Error("Failed to render template")
		h.renderError(c, "Failed to render results")
		return
	}
}

// GetModelDetails handles HTMX request for model details
// Also supports JSON response for SvelteKit frontend (Accept: application/json)
func (h *HuggingFaceUIHandler) GetModelDetails(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	
	modelID := c.Param("model_id")
	// Reconstruct model ID (e.g., "TheBloke/Llama-2-7B-GGUF")
	if strings.Contains(c.Request.URL.Path, "/") {
		// Extract from path - handle both /models/ and /model/
		parts := strings.Split(c.Request.URL.Path, "/models/")
		if len(parts) == 2 {
			modelID = parts[1]
		} else {
			parts = strings.Split(c.Request.URL.Path, "/model/")
			if len(parts) == 2 {
				modelID = parts[1]
			}
		}
	}
	
	if modelID == "" {
		if strings.Contains(c.GetHeader("Accept"), "application/json") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Model ID is required"})
			return
		}
		h.renderError(c, "Model ID is required")
		return
	}
	
	h.logger.WithField("model_id", modelID).Debug("Fetching model details")
	
	// Get model info
	model, err := h.hfClient.GetModelInfo(ctx, modelID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get model info")
		if strings.Contains(c.GetHeader("Accept"), "application/json") {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		h.renderError(c, "Failed to load model details: "+err.Error())
		return
	}
	
	// Check if JSON response is requested (SvelteKit frontend)
	if strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusOK, model)
		return
	}
	
	// Render HTML details (HTMX)
	data := map[string]interface{}{
		"Model": model,
	}
	
	if err := h.renderer.RenderInlineTemplate(c.Writer, "hf_model_details", data); err != nil {
		h.logger.WithError(err).Error("Failed to render template")
		h.renderError(c, "Failed to render details")
		return
	}
}

// GetGGUFFilesList handles HTMX request for GGUF files list
func (h *HuggingFaceUIHandler) GetGGUFFilesList(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	
	modelID := c.Param("model_id")
	if strings.Contains(c.Request.URL.Path, "/") {
		parts := strings.Split(c.Request.URL.Path, "/gguf-files/")
		if len(parts) == 2 {
			modelID = parts[1]
		}
	}
	
	if modelID == "" {
		h.renderError(c, "Model ID is required")
		return
	}
	
	// Get model info
	model, err := h.hfClient.GetModelInfo(ctx, modelID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get model info")
		h.renderError(c, "Failed to load GGUF files: "+err.Error())
		return
	}
	
	// Filter only GGUF files
	if !model.HasGGUF {
		h.renderError(c, "Model has no GGUF files")
		return
	}
	
	// Log file sizes for debugging
	h.logger.WithFields(logrus.Fields{
		"model_id":    model.ID,
		"gguf_count":  len(model.GGUFFiles),
		"total_size":  model.TotalSize,
	}).Debug("Rendering GGUF files list")
	
	for i, file := range model.GGUFFiles {
		h.logger.WithFields(logrus.Fields{
			"file_index": i,
			"filename":   file.Filename,
			"size":       file.Size,
			"has_lfs":    file.LFS != nil,
		}).Debug("GGUF file details")
		if file.LFS != nil {
			h.logger.WithFields(logrus.Fields{
				"lfs_size": file.LFS.Size,
				"lfs_oid":  file.LFS.OID,
			}).Debug("LFS details")
		}
	}
	
	// Check if JSON response is requested (SvelteKit frontend)
	if strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusOK, gin.H{
			"model_id": model.ID,
			"files":    model.GGUFFiles,
		})
		return
	}
	
	// Render GGUF files list (HTMX)
	data := map[string]interface{}{
		"ModelID": model.ID,
		"Files":   model.GGUFFiles,
	}
	
	if err := h.renderer.RenderInlineTemplate(c.Writer, "hf_gguf_files", data); err != nil {
		h.logger.WithError(err).Error("Failed to render template")
		h.renderError(c, "Failed to render GGUF files")
		return
	}
}

// GetPopularModels handles HTMX request for popular GGUF models
func (h *HuggingFaceUIHandler) GetPopularModels(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	category := c.DefaultQuery("category", "all")
	providerFilter := c.DefaultQuery("provider", "all")
	
	// Pagination parameters
	limitStr := c.DefaultQuery("limit", "30")
	pageStr := c.DefaultQuery("page", "1")
	limit, _ := strconv.Atoi(limitStr)
	page, _ := strconv.Atoi(pageStr)
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	if page <= 0 {
		page = 1
	}
	
	var search string
	var additionalTags []string
	
	switch category {
	case "llama":
		search = "llama"
		additionalTags = []string{"llama"}
	case "mistral":
		search = "mistral"
		additionalTags = []string{"mistral"}
	case "phi":
		search = "phi"
		additionalTags = []string{"phi"}
	case "gemma":
		search = "gemma"
		additionalTags = []string{"gemma"}
	case "vision":
		search = "vision"
		additionalTags = []string{"vision", "multimodal"}
	default:
		// All popular models
		search = ""
	}
	
	// Apply provider-specific filters
	var baseTags []string
	var library string
	switch providerFilter {
	case "llama.cpp":
		// Use library=gguf instead of tag for better coverage
		library = "gguf"
	case "vllm", "sglang", "tgi":
		baseTags = []string{"text-generation"}
		library = "transformers"
	case "embedding":
		baseTags = []string{"feature-extraction"}
	case "tei":
		// TEI supports both embedding and reranking models
		baseTags = []string{"sentence-similarity"}
	case "rerank":
		// Reranking models (cross-encoder / reranker)
		baseTags = []string{"text-classification"}
	default:
		// "all" - show text-generation models
		baseTags = []string{"text-generation"}
	}
	
	filters := huggingface.ModelFilters{
		Search:       search,
		Tags:         append(baseTags, additionalTags...),
		Library:      library,
		Sort:         "downloads",
		Direction:    -1,
		Limit:        limit,
		Page:         page,
		FullResponse: true,
		CardData:     true,
	}
	
	models, err := h.hfClient.SearchModels(ctx, filters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get popular models")
		statusCode := hfErrorStatusCode(err)
		// Check if JSON response is requested
		if strings.Contains(c.GetHeader("Accept"), "application/json") {
			c.JSON(statusCode, gin.H{"error": err.Error()})
			return
		}
		h.renderError(c, "Failed to load popular models: "+err.Error())
		return
	}
	
	// For llama.cpp filter, library=gguf was already passed to API
	// All returned models are GGUF models, mark them as such
	if providerFilter == "llama.cpp" {
		for i := range models {
			models[i].HasGGUF = true
		}
	}
	
	// Check if JSON response is requested (SvelteKit frontend)
	if strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusOK, gin.H{
			"models":   models,
			"count":    len(models),
			"category": category,
			"provider": providerFilter,
		})
		return
	}
	
	// Render popular models (HTMX)
	data := map[string]interface{}{
		"Models":   models,
		"Category": category,
	}
	
	if err := h.renderer.RenderInlineTemplate(c.Writer, "hf_popular_models", data); err != nil {
		h.logger.WithError(err).Error("Failed to render template")
		h.renderError(c, "Failed to render popular models")
		return
	}
}

// PostDownloadModel handles HTMX request to start model download
// Also supports JSON response for SvelteKit frontend
func (h *HuggingFaceUIHandler) PostDownloadModel(c *gin.Context) {
	type DownloadRequest struct {
		ModelID   string `json:"model_id" form:"model_id" binding:"required"`
		Filename  string `json:"filename" form:"filename" binding:"required"`
		TotalSize int64  `json:"total_size" form:"total_size"`
		SHA256    string `json:"sha256" form:"sha256"`
	}
	
	var req DownloadRequest
	isJSONRequest := c.ContentType() == "application/json"
	
	// Support both JSON and form data for HTMX compatibility
	if isJSONRequest {
		if err := c.ShouldBindJSON(&req); err != nil {
			if strings.Contains(c.GetHeader("Accept"), "application/json") {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			h.renderError(c, "Invalid request: "+err.Error())
			return
		}
	} else {
		if err := c.ShouldBind(&req); err != nil {
			h.renderError(c, "Invalid request: "+err.Error())
			return
		}
	}
	
	h.logger.WithFields(logrus.Fields{
		"model_id": req.ModelID,
		"filename": req.Filename,
		"size":     req.TotalSize,
	}).Info("Model download requested")
	
	// Start download
	download, err := h.downloader.StartDownload(req.ModelID, req.Filename, req.TotalSize, req.SHA256)
	if err != nil {
		h.logger.WithError(err).Error("Failed to start download")
		if strings.Contains(c.GetHeader("Accept"), "application/json") || isJSONRequest {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		h.renderError(c, "Failed to start download: "+err.Error())
		return
	}
	
	// Check if JSON response is requested (SvelteKit frontend)
	if strings.Contains(c.GetHeader("Accept"), "application/json") || isJSONRequest {
		c.JSON(http.StatusOK, gin.H{
			"download": download,
			"message":  "Download started",
		})
		return
	}
	
	// Return download started response with progress (HTMX)
	data := map[string]interface{}{
		"Download": download,
	}
	
	if err := h.renderer.RenderInlineTemplate(c.Writer, "hf_download_started", data); err != nil {
		h.logger.WithError(err).Error("Failed to render template")
		h.renderError(c, "Failed to render download status")
		return
	}
}

// GetDownloadProgress returns download progress for HTMX polling
func (h *HuggingFaceUIHandler) GetDownloadProgress(c *gin.Context) {
	downloadID := c.Param("download_id")
	
	download, exists := h.downloader.GetDownload(downloadID)
	if !exists {
		h.renderError(c, "Download not found")
		return
	}
	
	download.Mu.RLock()
	defer download.Mu.RUnlock()
	
	// Return only progress bar HTML for efficient updates
	html := fmt.Sprintf(`
		<div class="progress-fill" 
		     style="width: %.1f%%"
		     hx-get="/api/ui/huggingface/downloads/%s/progress"
		     hx-trigger="every 1s"
		     hx-swap="outerHTML">
			<span class="progress-text">%.1f%%</span>
		</div>
	`, download.Progress, downloadID, download.Progress)
	
	// If completed, stop polling
	if download.Status == huggingface.DownloadStatusCompleted {
		html = fmt.Sprintf(`
			<div class="progress-fill" style="width: 100%%">
				<span class="progress-text">✅ Completed</span>
			</div>
		`)
	} else if download.Status == huggingface.DownloadStatusFailed {
		html = fmt.Sprintf(`
			<div class="progress-fill" style="width: %.1f%%; background: #dc3545;">
				<span class="progress-text">❌ Failed: %s</span>
			</div>
		`, download.Progress, download.Error)
	}
	
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

// PostPauseDownload pauses an active download
func (h *HuggingFaceUIHandler) PostPauseDownload(c *gin.Context) {
	downloadID := c.Param("download_id")
	
	if err := h.downloader.PauseDownload(downloadID); err != nil {
		h.logger.WithError(err).Error("Failed to pause download")
		h.renderError(c, "Failed to pause download: "+err.Error())
		return
	}
	
	download, _ := h.downloader.GetDownload(downloadID)
	
	// Return updated download card
	data := map[string]interface{}{
		"Download": download,
	}
	
	if err := h.renderer.RenderInlineTemplate(c.Writer, "hf_download_paused", data); err != nil {
		h.logger.WithError(err).Error("Failed to render template")
		h.renderError(c, "Failed to render download status")
		return
	}
}

// PostCancelDownload cancels and removes a download
func (h *HuggingFaceUIHandler) PostCancelDownload(c *gin.Context) {
	downloadID := c.Param("download_id")
	
	if err := h.downloader.CancelDownload(downloadID); err != nil {
		h.logger.WithError(err).Error("Failed to cancel download")
		h.renderError(c, "Failed to cancel download: "+err.Error())
		return
	}
	
	// Return empty div (removes download card)
	html := `<div class="alert alert-info">Download cancelled and removed.</div>`
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

// GetDownloadsList returns all active downloads
func (h *HuggingFaceUIHandler) GetDownloadsList(c *gin.Context) {
	downloads := h.downloader.ListDownloads()
	
	// Check if JSON response is requested (SvelteKit frontend)
	if strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusOK, gin.H{
			"downloads": downloads,
			"count":     len(downloads),
		})
		return
	}
	
	// Render HTML (HTMX)
	data := map[string]interface{}{
		"Downloads": downloads,
		"Count":     len(downloads),
	}
	
	if err := h.renderer.RenderInlineTemplate(c.Writer, "hf_downloads_list", data); err != nil {
		h.logger.WithError(err).Error("Failed to render template")
		h.renderError(c, "Failed to render downloads list")
		return
	}
}

// PostClearCompleted removes all completed, failed, and cancelled downloads
func (h *HuggingFaceUIHandler) PostClearCompleted(c *gin.Context) {
	cleared := h.downloader.ClearCompleted()
	
	// Return JSON for SvelteKit frontend
	if strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.JSON(http.StatusOK, gin.H{
			"cleared": cleared,
			"message": fmt.Sprintf("Cleared %d downloads", cleared),
		})
		return
	}
	
	// Return HTML for HTMX
	html := fmt.Sprintf(`<div class="alert alert-success">Cleared %d completed downloads.</div>`, cleared)
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

// renderError renders error message
func (h *HuggingFaceUIHandler) renderError(c *gin.Context, message string) {
	html := fmt.Sprintf(`
		<div class="alert alert-error">
			<strong>Error:</strong> %s
		</div>
	`, message)
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

// hfErrorStatusCode returns appropriate HTTP status code based on HuggingFace API error.
// Returns 502 Bad Gateway for upstream auth errors (expired/invalid token) to distinguish
// from 401 which triggers user session logout in the frontend client.
func hfErrorStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}
	errLower := strings.ToLower(err.Error())
	if strings.Contains(errLower, "expired") ||
		strings.Contains(errLower, "unauthorized") ||
		strings.Contains(errLower, "invalid token") ||
		strings.Contains(errLower, "401") {
		return http.StatusBadGateway
	}
	return http.StatusInternalServerError
}


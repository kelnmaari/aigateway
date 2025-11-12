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
func (h *HuggingFaceUIHandler) GetModelsSearch(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	// Parse filters
	search := c.Query("search")
	author := c.Query("author")
	tagsStr := c.Query("tags")
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
	
	// Always include GGUF tag
	tags = append(tags, "gguf")
	
	// Build filters
	filters := huggingface.ModelFilters{
		Search:       search,
		Author:       author,
		Tags:         tags,
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
		h.renderError(c, "Failed to search models: "+err.Error())
		return
	}
	
	// Render results
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
func (h *HuggingFaceUIHandler) GetModelDetails(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	
	modelID := c.Param("model_id")
	// Reconstruct model ID (e.g., "TheBloke/Llama-2-7B-GGUF")
	if strings.Contains(c.Request.URL.Path, "/") {
		// Extract from path
		parts := strings.Split(c.Request.URL.Path, "/models/")
		if len(parts) == 2 {
			modelID = parts[1]
		}
	}
	
	if modelID == "" {
		h.renderError(c, "Model ID is required")
		return
	}
	
	h.logger.WithField("model_id", modelID).Debug("Fetching model details")
	
	// Get model info
	model, err := h.hfClient.GetModelInfo(ctx, modelID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get model info")
		h.renderError(c, "Failed to load model details: "+err.Error())
		return
	}
	
	// Render details
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
	
	// Render GGUF files list
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
	
	filters := huggingface.ModelFilters{
		Search:       search,
		Tags:         append([]string{"gguf"}, additionalTags...),
		Sort:         "downloads",
		Direction:    -1,
		Limit:        20,
		FullResponse: true,
		CardData:     true,
	}
	
	models, err := h.hfClient.SearchModels(ctx, filters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get popular models")
		h.renderError(c, "Failed to load popular models: "+err.Error())
		return
	}
	
	// Render popular models
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
func (h *HuggingFaceUIHandler) PostDownloadModel(c *gin.Context) {
	type DownloadRequest struct {
		ModelID   string `json:"model_id" form:"model_id" binding:"required"`
		Filename  string `json:"filename" form:"filename" binding:"required"`
		TotalSize int64  `json:"total_size" form:"total_size"`
		SHA256    string `json:"sha256" form:"sha256"`
	}
	
	var req DownloadRequest
	// Support both JSON and form data for HTMX compatibility
	if c.ContentType() == "application/json" {
		if err := c.ShouldBindJSON(&req); err != nil {
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
		h.renderError(c, "Failed to start download: "+err.Error())
		return
	}
	
	// Return download started response with progress
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


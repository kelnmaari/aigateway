package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
	"aigateway/internal/models"
	"aigateway/internal/storage"
	"aigateway/internal/webfetch"
)

// WebFetchHandler обрабатывает web fetch запросы
type WebFetchHandler struct {
	service *webfetch.Service
	logger  *logrus.Logger
}

// NewWebFetchHandler создает новый handler
func NewWebFetchHandler(cfg *config.Config, db storage.Database, logger *logrus.Logger) *WebFetchHandler {
	// Convert config.WebFetchConfig to webfetch.Config
	webfetchConfig := convertWebFetchConfig(cfg.WebFetch)
	service := webfetch.NewService(webfetchConfig, db, logger)

	return &WebFetchHandler{
		service: service,
		logger:  logger,
	}
}

// convertWebFetchConfig конвертирует config.WebFetchConfig в webfetch.Config
func convertWebFetchConfig(cfg config.WebFetchConfig) webfetch.Config {
	wfc := webfetch.DefaultConfig()

	wfc.Enabled = cfg.Enabled

	// Security
	wfc.Security.BlockPrivateIPs = cfg.BlockPrivateIPs
	wfc.Security.BlockLocalhost = cfg.BlockLocalhost
	wfc.Security.AllowedDomains = cfg.AllowedDomains
	wfc.Security.BlockedDomains = cfg.BlockedDomains

	// Rate limiting
	wfc.RateLimit.Enabled = cfg.RateLimitEnabled
	if cfg.DefaultRequestsPerMin > 0 {
		wfc.RateLimit.DefaultRequestsPerMinute = cfg.DefaultRequestsPerMin
	}

	// Cache
	wfc.Cache.Enabled = cfg.CacheEnabled
	if cfg.CacheTTL != "" {
		if ttl, err := time.ParseDuration(cfg.CacheTTL); err == nil {
			wfc.Cache.DefaultTTL = ttl
		}
	}

	// Timeout
	if cfg.Timeout != "" {
		if timeout, err := time.ParseDuration(cfg.Timeout); err == nil {
			wfc.HTTP.Timeout = timeout
		}
	}

	return wfc
}

// FetchRequest запрос на fetch URL
type FetchRequest struct {
	URL          string `json:"url" binding:"required"`
	ExtractLinks bool   `json:"extract_links"`
	Summarize    bool   `json:"summarize"`
	UseCache     bool   `json:"use_cache"`
}

// FetchResponse ответ с извлеченным контентом
type FetchResponse struct {
	ID          string                 `json:"id,omitempty"`
	URL         string                 `json:"url"`
	Title       string                 `json:"title"`
	Content     string                 `json:"content"`
	Metadata    *webfetch.PageMetadata `json:"metadata,omitempty"`
	Summary     string                 `json:"summary,omitempty"`
	WordCount   int                    `json:"word_count"`
	FetchTimeMS int64                  `json:"fetch_time_ms"`
	Cached      bool                   `json:"cached"`
	FetchedAt   time.Time              `json:"fetched_at"`
}

// HandleFetch обрабатывает POST /api/web/fetch
func (h *WebFetchHandler) HandleFetch(c *gin.Context) {
	var req FetchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid fetch request")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.Error{
				Message: "Invalid request: " + err.Error(),
				Type:    "invalid_request_error",
				Code:    "invalid_json",
			},
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"url":           req.URL,
		"extract_links": req.ExtractLinks,
		"summarize":     req.Summarize,
		"use_cache":     req.UseCache,
	}).Info("Fetching web page")

	// Validate URL first
	if err := h.service.ValidateURL(req.URL); err != nil {
		h.logger.WithError(err).Warn("URL validation failed")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.Error{
				Message: "Invalid URL: " + err.Error(),
				Type:    "invalid_request_error",
				Code:    "invalid_url",
			},
		})
		return
	}

	// Fetch page
	opts := webfetch.FetchOptions{
		Timeout:         30 * time.Second,
		MaxSize:         10 * 1024 * 1024, // 10MB
		FollowRedirects: true,
		ExtractLinks:    req.ExtractLinks,
		Summarize:       req.Summarize,
		CacheEnabled:    req.UseCache,
		CacheTTL:        1 * time.Hour,
	}

	page, err := h.service.FetchURL(c.Request.Context(), req.URL, opts)
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch web page")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.Error{
				Message: "Failed to fetch web page: " + err.Error(),
				Type:    "api_error",
				Code:    "fetch_failed",
			},
		})
		return
	}

	// Count words
	wordCount := len(page.Content) / 5 // Rough estimation

	response := FetchResponse{
		URL:         page.URL,
		Title:       page.Title,
		Content:     page.Content,
		Metadata:    page.Metadata,
		Summary:     page.Summary,
		WordCount:   wordCount,
		FetchTimeMS: page.FetchTimeMS,
		Cached:      page.Cached,
		FetchedAt:   page.FetchedAt,
	}

	h.logger.WithFields(logrus.Fields{
		"url":           page.URL,
		"title":         page.Title,
		"word_count":    wordCount,
		"fetch_time_ms": page.FetchTimeMS,
		"cached":        page.Cached,
	}).Info("Web page fetched successfully")

	c.JSON(http.StatusOK, response)
}

// BatchFetchRequest запрос на batch fetch
type BatchFetchRequest struct {
	URLs      []string `json:"urls" binding:"required"`
	Summarize bool     `json:"summarize"`
}

// BatchFetchResponse ответ с множеством страниц
type BatchFetchResponse struct {
	Results   []BatchResult `json:"results"`
	Total     int           `json:"total"`
	Succeeded int           `json:"succeeded"`
	Failed    int           `json:"failed"`
}

// BatchResult результат для одного URL
type BatchResult struct {
	URL         string                 `json:"url"`
	Status      string                 `json:"status"` // success, failed
	Title       string                 `json:"title,omitempty"`
	Content     string                 `json:"content,omitempty"`
	Metadata    *webfetch.PageMetadata `json:"metadata,omitempty"`
	Error       string                 `json:"error,omitempty"`
	FetchTimeMS int64                  `json:"fetch_time_ms,omitempty"`
}

// HandleBatchFetch обрабатывает POST /api/web/fetch/batch
func (h *WebFetchHandler) HandleBatchFetch(c *gin.Context) {
	var req BatchFetchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid batch fetch request")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.Error{
				Message: "Invalid request: " + err.Error(),
				Type:    "invalid_request_error",
				Code:    "invalid_json",
			},
		})
		return
	}

	if len(req.URLs) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.Error{
				Message: "URLs list cannot be empty",
				Type:    "invalid_request_error",
				Code:    "empty_urls",
			},
		})
		return
	}

	// Limit batch size
	if len(req.URLs) > 10 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.Error{
				Message: "Maximum 10 URLs per batch request",
				Type:    "invalid_request_error",
				Code:    "batch_too_large",
			},
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"urls_count": len(req.URLs),
		"summarize":  req.Summarize,
	}).Info("Fetching multiple web pages")

	opts := webfetch.FetchOptions{
		Timeout:         30 * time.Second,
		MaxSize:         10 * 1024 * 1024,
		FollowRedirects: true,
		ExtractLinks:    false,
		Summarize:       req.Summarize,
		CacheEnabled:    true,
		CacheTTL:        1 * time.Hour,
	}

	pages, err := h.service.FetchBatch(c.Request.Context(), req.URLs, opts)

	// Build response (even if there were errors)
	results := make([]BatchResult, 0, len(req.URLs))
	succeeded := 0
	failed := 0

	if err != nil {
		// Partial failure - some URLs failed
		h.logger.WithError(err).Warn("Batch fetch completed with errors")
		failed = len(req.URLs) - len(pages)
	}

	// Add successful results
	for _, page := range pages {
		results = append(results, BatchResult{
			URL:         page.URL,
			Status:      "success",
			Title:       page.Title,
			Content:     page.Content,
			Metadata:    page.Metadata,
			FetchTimeMS: page.FetchTimeMS,
		})
		succeeded++
	}

	// Add failed URLs (if any)
	if err != nil && len(pages) < len(req.URLs) {
		// For simplicity, mark remaining as failed
		for i := len(pages); i < len(req.URLs); i++ {
			results = append(results, BatchResult{
				URL:    req.URLs[i],
				Status: "failed",
				Error:  "fetch failed",
			})
			failed++
		}
	}

	response := BatchFetchResponse{
		Results:   results,
		Total:     len(req.URLs),
		Succeeded: succeeded,
		Failed:    failed,
	}

	h.logger.WithFields(logrus.Fields{
		"total":     response.Total,
		"succeeded": response.Succeeded,
		"failed":    response.Failed,
	}).Info("Batch fetch completed")

	c.JSON(http.StatusOK, response)
}


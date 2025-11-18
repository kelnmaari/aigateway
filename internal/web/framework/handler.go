// Package framework provides web UI framework HTTP handlers
package framework

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Handler serves framework assets
type Handler struct {
	builder *Builder
	logger  *logrus.Logger
	devMode bool
}

// NewHandler creates a new framework handler
func NewHandler(builder *Builder, logger *logrus.Logger, devMode bool) *Handler {
	return &Handler{
		builder: builder,
		logger:  logger,
		devMode: devMode,
	}
}

// ServeFrameworkJS serves framework.js with caching
func (h *Handler) ServeFrameworkJS(c *gin.Context) {
	h.serveAsset(c, "framework.js", "application/javascript; charset=utf-8")
}

// ServeFrameworkCSS serves framework.css with caching
func (h *Handler) ServeFrameworkCSS(c *gin.Context) {
	h.serveAsset(c, "framework.css", "text/css; charset=utf-8")
}

// serveAsset serves a framework asset with caching headers
func (h *Handler) serveAsset(c *gin.Context, assetName string, contentType string) {
	asset, ok := h.builder.GetAsset(assetName)
	if !ok {
		h.logger.WithField("asset", assetName).Error("Asset not found")
		c.String(http.StatusNotFound, "Asset not found")
		return
	}

	// Set content type
	c.Header("Content-Type", contentType)

	// Cache control headers
	if h.devMode {
		// Dev mode: no cache
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
	} else {
		// Production: aggressive caching with ETag
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("ETag", fmt.Sprintf(`"%s"`, asset.ContentHash))

		// Check if client has cached version
		if match := c.GetHeader("If-None-Match"); match == fmt.Sprintf(`"%s"`, asset.ContentHash) {
			c.Status(http.StatusNotModified)
			return
		}
	}

	// Compression hint
	c.Header("Vary", "Accept-Encoding")

	// Content info headers
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-Build-Time", asset.BuildTime.Format(time.RFC3339))
	c.Header("X-Minified", fmt.Sprintf("%v", asset.Minified))

	// Serve pre-compressed content if available and accepted
	acceptEncoding := c.GetHeader("Accept-Encoding")
	
	// Try Brotli first (better compression)
	if strings.Contains(acceptEncoding, "br") && len(asset.ContentBr) > 0 {
		c.Header("Content-Encoding", "br")
		c.Data(http.StatusOK, contentType, asset.ContentBr)
		h.logServe(assetName, asset, "brotli", len(asset.ContentBr))
		return
	}
	
	// Fallback to Gzip
	if strings.Contains(acceptEncoding, "gzip") && len(asset.ContentGzip) > 0 {
		c.Header("Content-Encoding", "gzip")
		c.Data(http.StatusOK, contentType, asset.ContentGzip)
		h.logServe(assetName, asset, "gzip", len(asset.ContentGzip))
		return
	}

	// Serve uncompressed
	c.Data(http.StatusOK, contentType, asset.Content)
	h.logServe(assetName, asset, "none", len(asset.Content))
}

// logServe logs asset serving
func (h *Handler) logServe(assetName string, asset *Asset, encoding string, size int) {
	h.logger.WithFields(logrus.Fields{
		"asset":    assetName,
		"size":     size,
		"original": asset.Size,
		"hash":     asset.ContentHash[:8],
		"minified": asset.Minified,
		"encoding": encoding,
		"cache":    !h.devMode,
	}).Debug("Framework asset served")
}

// GetFrameworkTags returns HTML tags for framework inclusion
func (h *Handler) GetFrameworkTags() FrameworkTags {
	jsAsset, jsOk := h.builder.GetAsset("framework.js")
	cssAsset, cssOk := h.builder.GetAsset("framework.css")

	tags := FrameworkTags{
		DevMode: h.devMode,
	}

	if jsOk {
		if h.devMode {
			// Dev mode: timestamp query param for cache busting
			tags.JS = fmt.Sprintf(`<script src="/framework/framework.js?v=%d"></script>`, time.Now().Unix())
		} else {
			// Production: hash query param + SRI
			tags.JS = fmt.Sprintf(`<script src="/framework/framework.js?v=%s" integrity="sha256-%s"></script>`, 
				jsAsset.ContentHash[:8], jsAsset.ContentHash)
		}
	}

	if cssOk {
		if h.devMode {
			// Dev mode: timestamp query param for cache busting
			tags.CSS = fmt.Sprintf(`<link rel="stylesheet" href="/framework/framework.css?v=%d">`, time.Now().Unix())
		} else {
			// Production: hash query param + SRI
			tags.CSS = fmt.Sprintf(`<link rel="stylesheet" href="/framework/framework.css?v=%s" integrity="sha256-%s">`, 
				cssAsset.ContentHash[:8], cssAsset.ContentHash)
		}
	}

	return tags
}

// FrameworkTags contains HTML tags for framework inclusion
type FrameworkTags struct {
	JS      string
	CSS     string
	DevMode bool
}

// RegisterRoutes registers framework routes
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	framework := r.Group("/framework")
	{
		// Exact routes for main assets
		framework.GET("/framework.js", h.ServeFrameworkJS)
		framework.GET("/framework.css", h.ServeFrameworkCSS)

		// Dev mode endpoints
		if h.devMode {
			framework.POST("/rebuild", h.Rebuild)
			framework.GET("/stats", h.GetStats)
			framework.GET("/stats.json", h.GetStatsJSON)
		}
	}

	h.logger.Info("Framework routes registered: /framework/framework.{js|css}")
	h.logger.Info("Note: Versioned assets (framework.hash.js) will use same endpoints with cache headers")
}

// Rebuild triggers asset rebuild (dev mode only)
func (h *Handler) Rebuild(c *gin.Context) {
	if !h.devMode {
		c.JSON(http.StatusForbidden, gin.H{"error": "rebuild only available in dev mode"})
		return
	}

	h.logger.Info("Manual rebuild triggered")
	if err := h.builder.Rebuild(); err != nil {
		h.logger.WithError(err).Error("Rebuild failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Framework rebuilt successfully",
		"build_time": time.Now().Format(time.RFC3339),
	})
}

// GetStats returns framework build stats
func (h *Handler) GetStats(c *gin.Context) {
	jsAsset, jsOk := h.builder.GetAsset("framework.js")
	cssAsset, cssOk := h.builder.GetAsset("framework.css")

	stats := gin.H{
		"version": "3.1.0",
		"devMode": h.devMode,
	}

	if jsOk {
		stats["js"] = gin.H{
			"size":       jsAsset.Size,
			"hash":       jsAsset.ContentHash[:16],
			"minified":   jsAsset.Minified,
			"build_time": jsAsset.BuildTime.Format(time.RFC3339),
		}
	}

	if cssOk {
		stats["css"] = gin.H{
			"size":       cssAsset.Size,
			"hash":       cssAsset.ContentHash[:16],
			"minified":   cssAsset.Minified,
			"build_time": cssAsset.BuildTime.Format(time.RFC3339),
		}
	}

	// Add bundle statistics if available
	bundleStats := h.builder.GetStats()
	if len(bundleStats) > 0 {
		stats["bundle_analysis"] = bundleStats
	}

	c.JSON(http.StatusOK, stats)
}

// GetStatsJSON exports detailed stats as JSON
func (h *Handler) GetStatsJSON(c *gin.Context) {
	json, err := h.builder.ExportStatsJSON()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", `attachment; filename="framework-stats.json"`)
	c.String(http.StatusOK, json)
}

// InjectTags middleware injects framework tags into HTML responses
func (h *Handler) InjectTags() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only inject for HTML responses
		if !strings.Contains(c.GetHeader("Accept"), "text/html") {
			c.Next()
			return
		}

		tags := h.GetFrameworkTags()

		// Store tags in context for templates
		c.Set("framework_css", tags.CSS)
		c.Set("framework_js", tags.JS)
		c.Set("framework_devmode", tags.DevMode)

		c.Next()
	}
}


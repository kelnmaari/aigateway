// Package ui provides HTMX UI handlers for dashboard statistics
package ui

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/storage"
)

// DashboardUIHandler handles dashboard statistics UI endpoints
type DashboardUIHandler struct {
	db     storage.Database
	logger *logrus.Logger
}

// NewDashboardUIHandler creates a new dashboard UI handler
func NewDashboardUIHandler(db storage.Database, logger *logrus.Logger) *DashboardUIHandler {
	return &DashboardUIHandler{
		db:     db,
		logger: logger,
	}
}

// GetUsersStatCard returns HTML for Total Users stat card
// GET /api/ui/dashboard/stats/users
func (h *DashboardUIHandler) GetUsersStatCard(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Get total users count
	totalUsers, err := h.db.CountTotalUsers(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get users count")
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `<div class="stat-value error">-</div>`)
		return
	}
	
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, fmt.Sprintf(`<div class="stat-value">%d</div>`, totalUsers))
}

// GetAPIKeysStatCard returns HTML for Total API Keys stat card
// GET /api/ui/dashboard/stats/api-keys
func (h *DashboardUIHandler) GetAPIKeysStatCard(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Get active API keys count
	totalKeys, err := h.db.CountActiveAPIKeys(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get API keys count")
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `<div class="stat-value error">-</div>`)
		return
	}
	
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, fmt.Sprintf(`<div class="stat-value">%d</div>`, totalKeys))
}

// GetModelsStatCard returns HTML for Total Models stat card
// GET /api/ui/dashboard/stats/models
func (h *DashboardUIHandler) GetModelsStatCard(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Get count of loaded models
	models, err := h.db.ListLoadedModels(ctx, false)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get loaded models count")
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `<div class="stat-value error">-</div>`)
		return
	}
	
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, fmt.Sprintf(`<div class="stat-value">%d</div>`, len(models)))
}

// GetRequestsStatCard returns HTML for API Requests (30d) stat card
// GET /api/ui/dashboard/stats/requests
func (h *DashboardUIHandler) GetRequestsStatCard(c *gin.Context) {
	// TODO: Implement proper usage stats aggregation
	// For now, return placeholder
	
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, `<div class="stat-value">-</div>`)
}

// GetSystemHealthTable returns HTML for System Health table
// GET /api/ui/dashboard/system-health
func (h *DashboardUIHandler) GetSystemHealthTable(c *gin.Context) {
	// Mock system health data for now
	// In production, you would integrate with actual system metrics
	
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, `
<tr>
	<td><strong>Database Status</strong></td>
	<td><span class="badge badge-active">Connected</span></td>
</tr>
<tr>
	<td><strong>API Server</strong></td>
	<td><span class="badge badge-active">Running</span></td>
</tr>
<tr>
	<td><strong>Storage</strong></td>
	<td><span class="badge badge-active">Available</span></td>
</tr>
<tr>
	<td><strong>Uptime</strong></td>
	<td class="text-muted">Calculating...</td>
</tr>`)
}


// Package ui provides HTMX UI handlers for monitoring pages
package ui

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/metrics"
	"aigateway/internal/storage"
)

// MonitorUIHandler handles monitoring UI endpoints
type MonitorUIHandler struct {
	logger     *logrus.Logger
	gpuMonitor *metrics.GPUMonitor
	db         storage.Database
	templates  *template.Template
}

// NewMonitorUIHandler creates a new monitor UI handler
func NewMonitorUIHandler(logger *logrus.Logger, gpuMonitor *metrics.GPUMonitor, db storage.Database) *MonitorUIHandler {
	// Load templates
	tmpl, err := template.ParseGlob("internal/web/templates/partials/monitor/*.html")
	if err != nil {
		logger.WithError(err).Warn("Failed to load monitor templates, will create inline")
	}

	return &MonitorUIHandler{
		logger:     logger,
		gpuMonitor: gpuMonitor,
		db:         db,
		templates:  tmpl,
	}
}

// RenderGPUMetrics renders GPU metrics partial for HTMX polling
// GET /ui/monitor/gpu-metrics
func (h *MonitorUIHandler) RenderGPUMetrics(c *gin.Context) {
	if h.gpuMonitor == nil {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `
<div class="alert alert-warning">
	<strong>GPU Monitoring Disabled</strong>
	<p>GPU monitoring is not available on this system.</p>
</div>`)
		return
	}

	metrics, err := h.gpuMonitor.GetMetrics()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get GPU metrics")
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `
<div class="alert alert-error">
	<strong>Error Loading GPU Metrics</strong>
	<p>%s</p>
</div>`, err.Error())
		return
	}

	// Render template
	if h.templates != nil {
		c.Header("Content-Type", "text/html")
		if err := h.templates.ExecuteTemplate(c.Writer, "gpu_metrics.html", metrics); err != nil {
			h.logger.WithError(err).Error("Failed to render GPU metrics template")
			h.renderInlineGPUMetrics(c, metrics)
		}
	} else {
		h.renderInlineGPUMetrics(c, metrics)
	}
}

// renderInlineGPUMetrics renders GPU metrics with inline HTML (fallback)
func (h *MonitorUIHandler) renderInlineGPUMetrics(c *gin.Context, metricsData *metrics.GPUMetrics) {
	c.Header("Content-Type", "text/html")
	
	// Check if we have GPU devices
	if metricsData == nil || len(metricsData.Devices) == 0 {
		c.String(http.StatusOK, `<div class="alert alert-warning">No GPU data available</div>`)
		return
	}

	html := `<div class="gpu-grid">`
	
	for _, device := range metricsData.Devices {
		name := device.Name
		if name == "" {
			name = fmt.Sprintf("GPU %d", device.Index)
		}

		temperature := fmt.Sprintf("%d°C", device.TemperatureC)
		utilization := fmt.Sprintf("%d%%", device.UtilizationGPU)
		
		memoryUsed := fmt.Sprintf("%.2f GB", device.MemoryUsedMB/1024)
		memoryTotal := fmt.Sprintf("%.2f GB", device.MemoryTotalMB/1024)
		memoryPercent := 0.0
		if device.MemoryTotalMB > 0 {
			memoryPercent = (device.MemoryUsedMB / device.MemoryTotalMB) * 100
		}

		powerDraw := fmt.Sprintf("%.1f W", device.PowerUsageW)
		powerLimit := fmt.Sprintf("%.1f W", device.PowerLimitW)

		fanSpeed := fmt.Sprintf("%d%%", device.FanSpeedPercent)

		html += fmt.Sprintf(`
<div class="gpu-card">
	<div class="gpu-header">
		<h3>%s</h3>
		<span class="gpu-status status-online">Active</span>
	</div>
	<div class="gpu-metrics">
		<div class="metric-item">
			<span class="metric-label">Temperature</span>
			<span class="metric-value">%s</span>
		</div>
		<div class="metric-item">
			<span class="metric-label">Utilization</span>
			<span class="metric-value">%s</span>
			<div class="progress-bar">
				<div class="progress-fill" style="width: %s"></div>
			</div>
		</div>
		<div class="metric-item">
			<span class="metric-label">Memory</span>
			<span class="metric-value">%s / %s</span>
			<div class="progress-bar">
				<div class="progress-fill" style="width: %.0f%%"></div>
			</div>
		</div>
		<div class="metric-item">
			<span class="metric-label">Power Draw</span>
			<span class="metric-value">%s / %s</span>
		</div>
		<div class="metric-item">
			<span class="metric-label">Fan Speed</span>
			<span class="metric-value">%s</span>
		</div>
	</div>
	<div class="gpu-footer">
		<small class="text-muted">Last updated: %s</small>
	</div>
</div>
`, name, temperature, utilization, utilization, memoryUsed, memoryTotal, memoryPercent,
			powerDraw, powerLimit, fanSpeed, time.Now().Format("15:04:05"))
	}

	html += `</div>`
	c.String(http.StatusOK, html)
}

// RenderAuditLogRows renders audit log table rows for HTMX polling
// GET /ui/monitor/audit-logs
func (h *MonitorUIHandler) RenderAuditLogRows(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Parse query parameters
	filters := storage.AuditFilters{
		Limit:  20, // Limit for live updates
		Offset: 0,
	}
	
	// Optional filters from query params
	if severity := c.Query("severity"); severity != "" {
		filters.Severity = severity
	}
	if eventType := c.Query("event_type"); eventType != "" {
		filters.EventType = eventType
	}
	
	// Get recent audit events
	events, _, err := h.db.GetAuditEvents(ctx, filters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get audit events for UI")
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `
<tr>
	<td colspan="8" class="table-empty error">
		Failed to load audit events
	</td>
</tr>`)
		return
	}
	
	// Check if no events
	if len(events) == 0 {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `
<tr>
	<td colspan="8" class="table-empty">
		No audit events found
	</td>
</tr>`)
		return
	}
	
	// Render rows
	c.Header("Content-Type", "text/html")
	html := ""
	for _, event := range events {
		severityClass := "severity-info"
		switch event.Severity {
		case "critical":
			severityClass = "severity-critical"
		case "warning":
			severityClass = "severity-warning"
		}
		
		statusClass := "status-success"
		if event.Status == "failure" {
			statusClass = "status-failure"
		}
		
		html += fmt.Sprintf(`
<tr class="audit-row">
	<td>%s</td>
	<td><span class="%s">%s</span></td>
	<td>%s</td>
	<td class="truncate" title="%s">%s</td>
	<td>%s</td>
	<td class="truncate" title="%s">%s</td>
	<td><span class="%s">%s</span></td>
	<td class="truncate">%s</td>
</tr>`,
			event.Timestamp.Format("15:04:05"),
			severityClass, event.Severity,
			event.EventType,
			event.ActorID, truncateString(event.ActorID, 20),
			event.Action,
			event.Resource, truncateString(event.Resource, 30),
			statusClass, event.Status,
			event.IPAddress,
		)
	}
	
	c.String(http.StatusOK, html)
}

// truncateString truncates string to max length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// RenderUsageStats renders usage statistics cards for HTMX polling
// GET /ui/monitor/usage-stats
func (h *MonitorUIHandler) RenderUsageStats(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `<div class="alert alert-error">Unauthorized</div>`)
		return
	}
	userIDStr := userID.(string)
	
	// Get period from query (default: 24h)
	period := 24 * time.Hour
	if periodParam := c.Query("period"); periodParam != "" {
		switch periodParam {
		case "1h":
			period = 1 * time.Hour
		case "24h":
			period = 24 * time.Hour
		case "7d":
			period = 7 * 24 * time.Hour
		}
	}
	
	// Get usage stats for user
	stats, err := h.db.GetUserUsageStats(ctx, userIDStr, period)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get usage stats for UI")
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `<div class="alert alert-error">Failed to load usage statistics</div>`)
		return
	}
	
	// Format numbers
	totalRequests := int64(0)
	totalTokens := int64(0)
	avgResponseTime := 0.0
	
	if stats != nil {
		totalRequests = stats.TotalRequests
		totalTokens = stats.TotalTokens
		avgResponseTime = stats.AvgDurationMS
	}
	
	now := time.Now()
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, fmt.Sprintf(`
<div class="stats-grid">
	<div class="stat-card">
		<div class="stat-icon" style="background: rgba(16, 163, 127, 0.1);">
			<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="var(--accent-primary)">
				<path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" stroke-width="2"/>
			</svg>
		</div>
		<div class="stat-content">
			<div class="stat-label">Total Requests</div>
			<div class="stat-value">%s</div>
		</div>
	</div>
	
	<div class="stat-card">
		<div class="stat-icon" style="background: rgba(168, 85, 247, 0.1);">
			<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#a855f7">
				<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" stroke-width="2"/>
			</svg>
		</div>
		<div class="stat-content">
			<div class="stat-label">Tokens Used</div>
			<div class="stat-value">%s</div>
		</div>
	</div>
	
	<div class="stat-card">
		<div class="stat-icon" style="background: rgba(59, 130, 246, 0.1);">
			<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#3b82f6">
				<circle cx="12" cy="12" r="10" stroke-width="2"/>
				<path d="M12 6v6l4 2" stroke-width="2"/>
			</svg>
		</div>
		<div class="stat-content">
			<div class="stat-label">Avg Response</div>
			<div class="stat-value">%.0fms</div>
		</div>
	</div>
	
	<div class="stat-footer-all">
		<small class="text-muted">Updated: %s</small>
	</div>
</div>
`, formatNumber(int(totalRequests)), formatNumber(int(totalTokens)), avgResponseTime, now.Format("15:04:05")))
}

// formatNumber formats number with thousands separator
func formatNumber(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}


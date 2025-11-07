// Package ui provides HTMX UI handlers for monitoring pages
package ui

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/metrics"
)

// MonitorUIHandler handles monitoring UI endpoints
type MonitorUIHandler struct {
	logger     *logrus.Logger
	gpuMonitor *metrics.GPUMonitor
	templates  *template.Template
}

// NewMonitorUIHandler creates a new monitor UI handler
func NewMonitorUIHandler(logger *logrus.Logger, gpuMonitor *metrics.GPUMonitor) *MonitorUIHandler {
	// Load templates
	tmpl, err := template.ParseGlob("internal/web/templates/partials/monitor/*.html")
	if err != nil {
		logger.WithError(err).Warn("Failed to load monitor templates, will create inline")
	}

	return &MonitorUIHandler{
		logger:     logger,
		gpuMonitor: gpuMonitor,
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

		fanSpeed := fmt.Sprintf("%.0f%%", device.FanSpeedPercent)

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
	// TODO: Implement with actual audit log data from database
	// For now, return placeholder
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, `
<tr>
	<td colspan="8" class="table-empty">
		<div class="loading"></div> Loading audit events...
	</td>
</tr>`)
}

// RenderUsageStats renders usage statistics cards for HTMX polling
// GET /ui/monitor/usage-stats
func (h *MonitorUIHandler) RenderUsageStats(c *gin.Context) {
	// TODO: Implement with actual usage stats from database
	// For now, return placeholder
	c.Header("Content-Type", "text/html")
	now := time.Now()
	c.String(http.StatusOK, fmt.Sprintf(`
<div class="stat-card">
	<div class="stat-icon" style="background: rgba(16, 163, 127, 0.1);">
		<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="var(--accent-primary)">
			<path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" stroke-width="2"/>
		</svg>
	</div>
	<div class="stat-content">
		<div class="stat-label">Total Requests</div>
		<div class="stat-value">1,234</div>
		<div class="stat-trend trend-up">+12%% from last period</div>
	</div>
	<div class="stat-footer">
		<small class="text-muted">Updated: %s</small>
	</div>
</div>
`, now.Format("15:04:05")))
}


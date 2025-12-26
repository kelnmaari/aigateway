// Package handlers provides HTTP handlers for GPU monitoring
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/metrics"
)

// GPUHandler handles GPU monitoring endpoints
type GPUHandler struct {
	logger     *logrus.Logger
	gpuMonitor *metrics.GPUMonitor
}

// NewGPUHandler creates a new GPU handler
func NewGPUHandler(logger *logrus.Logger, gpuMonitor *metrics.GPUMonitor) *GPUHandler {
	return &GPUHandler{
		logger:     logger,
		gpuMonitor: gpuMonitor,
	}
}

// GetGPUMetrics возвращает текущие метрики GPU
// @Summary Get GPU metrics
// @Description Returns current NVIDIA GPU metrics including utilization, temperature, memory usage
// @Tags GPU
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /api/gpu/metrics [get]
func (h *GPUHandler) GetGPUMetrics(c *gin.Context) {
	if h.gpuMonitor == nil {
		c.JSON(http.StatusOK, gin.H{
			"enabled": false,
			"message": "GPU monitoring is not available",
		})
		return
	}

	metrics, err := h.gpuMonitor.GetMetrics()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get GPU metrics")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve GPU metrics",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled": true,
		"data":    metrics,
	})
}

// GPUDevice simplified GPU info for selection
type GPUDevice struct {
	Index       int     `json:"index"`
	Name        string  `json:"name"`
	MemoryMB    float64 `json:"memory_mb"`
	MemoryFreeMB float64 `json:"memory_free_mb"`
}

// GetGPUList returns list of available GPUs for model deployment selection
// @Summary Get GPU list
// @Description Returns list of available NVIDIA GPUs with names and memory info
// @Tags GPU
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/gpu/list [get]
func (h *GPUHandler) GetGPUList(c *gin.Context) {
	if h.gpuMonitor == nil {
		c.JSON(http.StatusOK, gin.H{
			"enabled": false,
			"devices": []GPUDevice{},
		})
		return
	}

	metrics, err := h.gpuMonitor.GetMetrics()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get GPU list")
		c.JSON(http.StatusOK, gin.H{
			"enabled": false,
			"devices": []GPUDevice{},
			"error":   err.Error(),
		})
		return
	}

	devices := make([]GPUDevice, 0, len(metrics.Devices))
	for _, d := range metrics.Devices {
		devices = append(devices, GPUDevice{
			Index:        d.Index,
			Name:         d.Name,
			MemoryMB:     d.MemoryTotalMB,
			MemoryFreeMB: d.MemoryFreeMB,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled": true,
		"devices": devices,
	})
}


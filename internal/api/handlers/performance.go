package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aigateway/internal/observability"
)

// PerformanceHandler handles performance monitoring API endpoints.
type PerformanceHandler struct {
	monitor      *observability.PerformanceMonitor
	leakDetector *observability.LeakDetector
}

// NewPerformanceHandler creates a new PerformanceHandler instance.
func NewPerformanceHandler(monitor *observability.PerformanceMonitor, leakDetector *observability.LeakDetector) *PerformanceHandler {
	return &PerformanceHandler{
		monitor:      monitor,
		leakDetector: leakDetector,
	}
}

// GetMetrics returns current performance metrics.
//
// GET /api/admin/performance/metrics
//
// Response:
//   - 200: Current performance metrics
//   - 503: Performance monitoring disabled
func (h *PerformanceHandler) GetMetrics(c *gin.Context) {
	if h.monitor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Performance monitoring is disabled",
		})
		return
	}

	metrics := h.monitor.GetMetrics()
	baseline := h.monitor.GetBaseline()

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"metrics":  metrics,
		"baseline": baseline,
	})
}

// GetLeakStatus returns current leak detection status and samples.
//
// GET /api/admin/performance/leaks
//
// Response:
//   - 200: Leak detection samples
//   - 503: Leak detection disabled
func (h *PerformanceHandler) GetLeakStatus(c *gin.Context) {
	if h.leakDetector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Leak detection is disabled",
		})
		return
	}

	samples := h.leakDetector.GetSamples()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"samples": samples,
		"count":   len(samples),
	})
}

// ResetBaseline resets the performance baseline to current metrics.
//
// POST /api/admin/performance/reset-baseline
//
// Response:
//   - 200: Baseline reset successfully
//   - 503: Performance monitoring disabled
func (h *PerformanceHandler) ResetBaseline(c *gin.Context) {
	if h.monitor == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Performance monitoring is disabled",
		})
		return
	}

	h.monitor.ResetBaseline()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Performance baseline reset successfully",
	})
}

// ResetLeakDetector clears all leak detection samples.
//
// POST /api/admin/performance/reset-leaks
//
// Response:
//   - 200: Leak detector reset successfully
//   - 503: Leak detection disabled
func (h *PerformanceHandler) ResetLeakDetector(c *gin.Context) {
	if h.leakDetector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Leak detection is disabled",
		})
		return
	}

	h.leakDetector.Reset()

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Leak detector samples reset successfully",
	})
}


package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"

	"aigateway/internal/observability"
)

// PerformanceHandler handles performance monitoring API endpoints.
type PerformanceHandler struct {
	monitor      *observability.PerformanceMonitor
	leakDetector *observability.LeakDetector
	startTime    time.Time
}

// NewPerformanceHandler creates a new PerformanceHandler instance.
func NewPerformanceHandler(monitor *observability.PerformanceMonitor, leakDetector *observability.LeakDetector) *PerformanceHandler {
	return &PerformanceHandler{
		monitor:      monitor,
		leakDetector: leakDetector,
		startTime:    time.Now(),
	}
}

// SystemMetrics represents system-level metrics
type SystemMetrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	CPUCores      int     `json:"cpu_cores"`
	MemoryUsed    uint64  `json:"memory_used"`
	MemoryTotal   uint64  `json:"memory_total"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskUsed      uint64  `json:"disk_used"`
	DiskTotal     uint64  `json:"disk_total"`
	DiskPercent   float64 `json:"disk_percent"`
	Uptime        uint64  `json:"uptime"`
	AppUptime     int64   `json:"app_uptime"`
}

// getSystemMetrics collects system-level metrics using gopsutil
func (h *PerformanceHandler) getSystemMetrics() *SystemMetrics {
	sm := &SystemMetrics{
		AppUptime: int64(time.Since(h.startTime).Seconds()),
	}

	// CPU usage (average across all cores, 100ms sample)
	if cpuPercent, err := cpu.Percent(100*time.Millisecond, false); err == nil && len(cpuPercent) > 0 {
		sm.CPUPercent = cpuPercent[0]
	}

	// CPU cores
	if cores, err := cpu.Counts(true); err == nil {
		sm.CPUCores = cores
	}

	// Memory
	if memInfo, err := mem.VirtualMemory(); err == nil {
		sm.MemoryUsed = memInfo.Used
		sm.MemoryTotal = memInfo.Total
		sm.MemoryPercent = memInfo.UsedPercent
	}

	// Disk (root partition)
	if diskInfo, err := disk.Usage("/"); err == nil {
		sm.DiskUsed = diskInfo.Used
		sm.DiskTotal = diskInfo.Total
		sm.DiskPercent = diskInfo.UsedPercent
	} else if diskInfo, err := disk.Usage("C:"); err == nil {
		// Windows fallback
		sm.DiskUsed = diskInfo.Used
		sm.DiskTotal = diskInfo.Total
		sm.DiskPercent = diskInfo.UsedPercent
	}

	// System uptime
	if hostInfo, err := host.Info(); err == nil {
		sm.Uptime = hostInfo.Uptime
	}

	return sm
}

// GetMetrics returns current performance metrics.
//
// GET /api/admin/performance/metrics
//
// Response:
//   - 200: Current performance metrics (system + app)
//   - 503: Performance monitoring disabled
func (h *PerformanceHandler) GetMetrics(c *gin.Context) {
	// System metrics are always available
	systemMetrics := h.getSystemMetrics()

	response := gin.H{
		"status": "success",
		"system": systemMetrics,
	}

	// App metrics (Go runtime) - optional
	if h.monitor != nil {
		response["app"] = h.monitor.GetMetrics()
		response["baseline"] = h.monitor.GetBaseline()
	}

	c.JSON(http.StatusOK, response)
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


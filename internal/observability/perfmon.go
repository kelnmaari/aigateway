// Package observability provides performance monitoring and observability features.
//
// This file implements continuous performance monitoring with:
//   - Runtime metrics collection (CPU, memory, goroutines)
//   - Memory leak detection
//   - Performance anomaly detection
//   - Baseline comparison
package observability

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// PerformanceMonitor continuously monitors application performance metrics.
type PerformanceMonitor struct {
	logger        *logrus.Logger
	metricsLogger *logrus.Logger // Отдельный логгер для метрик (в metrics.log)
	config        PerfMonConfig
	metrics       *PerformanceMetrics
	baseline      *PerformanceMetrics
	mu            sync.RWMutex
	stopCh        chan struct{}
}

// SetMetricsLogger устанавливает отдельный логгер для метрик
func (pm *PerformanceMonitor) SetMetricsLogger(logger *logrus.Logger) {
	if pm != nil {
		pm.metricsLogger = logger
	}
}

// PerfMonConfig holds configuration for performance monitoring.
type PerfMonConfig struct {
	Enabled              bool          // Enable performance monitoring
	CollectionInterval   time.Duration // How often to collect metrics (default: 30s)
	MemoryThresholdMB    int64         // Alert if heap exceeds this (default: 1024 MB)
	GoroutineThreshold   int           // Alert if goroutines exceed this (default: 1000)
	SlowRequestThreshold time.Duration // Log requests slower than this (default: 5s)
	GCPercentage         int           // GOGC value (default: 100)
}

// PerformanceMetrics represents a snapshot of application performance metrics.
type PerformanceMetrics struct {
	Timestamp     time.Time `json:"timestamp"`
	HeapAllocMB   float64   `json:"heap_alloc_mb"`   // Current heap allocation
	HeapSysMB     float64   `json:"heap_sys_mb"`     // Total heap memory from OS
	NumGC         uint32    `json:"num_gc"`          // Number of GC runs
	NumGoroutines int       `json:"num_goroutines"`  // Current goroutine count
	NumCPU        int       `json:"num_cpu"`         // Number of CPUs
	GCPauseMS     float64   `json:"gc_pause_ms"`     // Average GC pause time
	AllocRate     float64   `json:"alloc_rate_mb_s"` // Memory allocation rate (MB/s)
}

// NewPerformanceMonitor creates a new PerformanceMonitor instance.
//
// If monitoring is disabled in config, returns nil.
// Collects initial baseline metrics on creation.
func NewPerformanceMonitor(logger *logrus.Logger, cfg PerfMonConfig) *PerformanceMonitor {
	if !cfg.Enabled {
		return nil
	}

	// Set default values
	if cfg.CollectionInterval == 0 {
		cfg.CollectionInterval = 30 * time.Second
	}
	if cfg.MemoryThresholdMB == 0 {
		cfg.MemoryThresholdMB = 1024
	}
	if cfg.GoroutineThreshold == 0 {
		cfg.GoroutineThreshold = 1000
	}
	if cfg.SlowRequestThreshold == 0 {
		cfg.SlowRequestThreshold = 5 * time.Second
	}
	if cfg.GCPercentage == 0 {
		cfg.GCPercentage = 100
	}

	// Configure garbage collector
	debug.SetGCPercent(cfg.GCPercentage)

	pm := &PerformanceMonitor{
		logger:  logger,
		config:  cfg,
		metrics: &PerformanceMetrics{},
		stopCh:  make(chan struct{}),
	}

	// Collect initial baseline
	pm.collectMetrics()
	pm.baseline = pm.metrics

	return pm
}

// Start begins continuous performance monitoring.
//
// Runs in a goroutine, collecting metrics at configured intervals.
// Stops when context is canceled or Stop() is called.
func (pm *PerformanceMonitor) Start(ctx context.Context) {
	if pm == nil {
		return
	}

	ticker := time.NewTicker(pm.config.CollectionInterval)
	defer ticker.Stop()

	pm.logger.WithFields(logrus.Fields{
		"interval":            pm.config.CollectionInterval,
		"memory_threshold":    pm.config.MemoryThresholdMB,
		"goroutine_threshold": pm.config.GoroutineThreshold,
		"gc_percentage":       pm.config.GCPercentage,
	}).Info("Performance monitor started")

	for {
		select {
		case <-ticker.C:
			pm.collectMetrics()
			pm.analyzeMetrics()
		case <-ctx.Done():
			pm.logger.Info("Performance monitor stopped by context")
			return
		case <-pm.stopCh:
			pm.logger.Info("Performance monitor stopped")
			return
		}
	}
}

// Stop halts the performance monitoring.
func (pm *PerformanceMonitor) Stop() {
	if pm == nil {
		return
	}
	close(pm.stopCh)
}

// collectMetrics gathers current runtime metrics.
func (pm *PerformanceMonitor) collectMetrics() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	previousMetrics := pm.metrics

	pm.metrics = &PerformanceMetrics{
		Timestamp:     time.Now(),
		HeapAllocMB:   float64(m.HeapAlloc) / 1024 / 1024,
		HeapSysMB:     float64(m.HeapSys) / 1024 / 1024,
		NumGC:         m.NumGC,
		NumGoroutines: runtime.NumGoroutine(),
		NumCPU:        runtime.NumCPU(),
	}

	// Calculate GC pause time (average of last 256 pauses)
	if m.NumGC > 0 {
		totalPause := uint64(0)
		count := int(m.NumGC)
		if count > 256 {
			count = 256
		}
		for i := 0; i < count; i++ {
			totalPause += m.PauseNs[(m.NumGC-uint32(i)+255)%256]
		}
		pm.metrics.GCPauseMS = float64(totalPause) / float64(count) / 1e6
	}

	// Calculate allocation rate
	if previousMetrics != nil && !previousMetrics.Timestamp.IsZero() {
		duration := pm.metrics.Timestamp.Sub(previousMetrics.Timestamp).Seconds()
		if duration > 0 {
			allocDiff := pm.metrics.HeapAllocMB - previousMetrics.HeapAllocMB
			pm.metrics.AllocRate = allocDiff / duration
		}
	}
}

// analyzeMetrics checks for performance anomalies and logs warnings.
func (pm *PerformanceMonitor) analyzeMetrics() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	m := pm.metrics

	// Memory threshold check
	if int64(m.HeapAllocMB) > pm.config.MemoryThresholdMB {
		pm.logger.WithFields(logrus.Fields{
			"heap_alloc_mb": fmt.Sprintf("%.2f", m.HeapAllocMB),
			"threshold_mb":  pm.config.MemoryThresholdMB,
			"goroutines":    m.NumGoroutines,
			"num_gc":        m.NumGC,
		}).Warn("Memory usage exceeds threshold")
	}

	// Goroutine leak detection
	if m.NumGoroutines > pm.config.GoroutineThreshold {
		pm.logger.WithFields(logrus.Fields{
			"num_goroutines": m.NumGoroutines,
			"threshold":      pm.config.GoroutineThreshold,
			"heap_alloc_mb":  fmt.Sprintf("%.2f", m.HeapAllocMB),
		}).Warn("Goroutine count exceeds threshold - possible leak")
	}

	// Memory leak detection (sustained high allocation rate)
	if m.AllocRate > 10 { // >10 MB/s sustained growth
		pm.logger.WithFields(logrus.Fields{
			"alloc_rate_mb_s": fmt.Sprintf("%.2f", m.AllocRate),
			"heap_alloc_mb":   fmt.Sprintf("%.2f", m.HeapAllocMB),
			"heap_sys_mb":     fmt.Sprintf("%.2f", m.HeapSysMB),
		}).Warn("High memory allocation rate - possible memory leak")
	}

	// Log periodic metrics at debug level (в отдельный файл если настроен)
	logTarget := pm.metricsLogger
	if logTarget == nil {
		logTarget = pm.logger
	}
	logTarget.WithFields(logrus.Fields{
		"heap_alloc_mb":   fmt.Sprintf("%.2f", m.HeapAllocMB),
		"heap_sys_mb":     fmt.Sprintf("%.2f", m.HeapSysMB),
		"num_gc":          m.NumGC,
		"gc_pause_ms":     fmt.Sprintf("%.2f", m.GCPauseMS),
		"num_goroutines":  m.NumGoroutines,
		"alloc_rate_mb_s": fmt.Sprintf("%.2f", m.AllocRate),
	}).Debug("Performance metrics collected")
}

// GetMetrics returns a copy of current performance metrics.
//
// Thread-safe, returns a copy to avoid race conditions.
func (pm *PerformanceMonitor) GetMetrics() *PerformanceMetrics {
	if pm == nil {
		return nil
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Return copy to avoid race conditions
	metrics := *pm.metrics
	return &metrics
}

// GetBaseline returns a copy of baseline performance metrics.
func (pm *PerformanceMonitor) GetBaseline() *PerformanceMetrics {
	if pm == nil {
		return nil
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.baseline == nil {
		return nil
	}

	baseline := *pm.baseline
	return &baseline
}

// ResetBaseline resets the baseline to current metrics.
//
// Useful after application warmup or after resolving performance issues.
func (pm *PerformanceMonitor) ResetBaseline() {
	if pm == nil {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.baseline = pm.metrics
	pm.logger.Info("Performance baseline reset")
}


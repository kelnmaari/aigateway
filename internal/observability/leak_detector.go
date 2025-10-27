package observability

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// LeakDetector monitors for memory and goroutine leaks.
//
// It collects periodic samples and analyzes trends to detect
// sustained growth that indicates a leak.
type LeakDetector struct {
	logger     *logrus.Logger
	samples    []MemorySample
	maxSamples int
	mu         sync.RWMutex
}

// MemorySample represents a single memory measurement.
type MemorySample struct {
	Timestamp     time.Time
	HeapAllocMB   float64
	NumGoroutines int
}

// NewLeakDetector creates a new LeakDetector instance.
//
// Parameters:
//   - logger: Logger for warnings and alerts
//
// Returns initialized detector with 10 sample history.
func NewLeakDetector(logger *logrus.Logger) *LeakDetector {
	return &LeakDetector{
		logger:     logger,
		samples:    make([]MemorySample, 0, 10),
		maxSamples: 10,
	}
}

// Start begins continuous leak detection monitoring.
//
// Collects samples every minute and analyzes trends.
// Stops when context is canceled.
func (ld *LeakDetector) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	ld.logger.Info("Leak detector started")

	for {
		select {
		case <-ticker.C:
			ld.collectSample()
			ld.analyzeTrend()
		case <-ctx.Done():
			ld.logger.Info("Leak detector stopped")
			return
		}
	}
}

// collectSample takes a memory and goroutine snapshot.
func (ld *LeakDetector) collectSample() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	sample := MemorySample{
		Timestamp:     time.Now(),
		HeapAllocMB:   float64(m.HeapAlloc) / 1024 / 1024,
		NumGoroutines: runtime.NumGoroutine(),
	}

	ld.mu.Lock()
	defer ld.mu.Unlock()

	ld.samples = append(ld.samples, sample)
	if len(ld.samples) > ld.maxSamples {
		ld.samples = ld.samples[1:] // Remove oldest
	}
}

// analyzeTrend checks for sustained memory growth patterns.
//
// If >80% of recent samples show growth, a leak warning is logged.
func (ld *LeakDetector) analyzeTrend() {
	ld.mu.RLock()
	defer ld.mu.RUnlock()

	if len(ld.samples) < 5 {
		return // Need at least 5 samples for trend analysis
	}

	// Check if memory is consistently growing
	growthCount := 0
	for i := 1; i < len(ld.samples); i++ {
		if ld.samples[i].HeapAllocMB > ld.samples[i-1].HeapAllocMB {
			growthCount++
		}
	}

	// Calculate growth percentage
	growthPercentage := float64(growthCount) / float64(len(ld.samples)-1)

	// If >80% of samples show growth, likely a leak
	if growthPercentage > 0.8 {
		first := ld.samples[0]
		last := ld.samples[len(ld.samples)-1]
		growth := last.HeapAllocMB - first.HeapAllocMB
		duration := last.Timestamp.Sub(first.Timestamp)

		ld.logger.WithFields(logrus.Fields{
			"growth_mb":      fmt.Sprintf("%.2f", growth),
			"duration":       duration,
			"initial_mb":     fmt.Sprintf("%.2f", first.HeapAllocMB),
			"current_mb":     fmt.Sprintf("%.2f", last.HeapAllocMB),
			"goroutines":     last.NumGoroutines,
			"rate_mb_min":    fmt.Sprintf("%.2f", growth/duration.Minutes()),
			"growth_percent": fmt.Sprintf("%.1f%%", growthPercentage*100),
		}).Warn("Possible memory leak detected - sustained memory growth")
	}

	// Check for goroutine growth
	goroutineGrowthCount := 0
	for i := 1; i < len(ld.samples); i++ {
		if ld.samples[i].NumGoroutines > ld.samples[i-1].NumGoroutines {
			goroutineGrowthCount++
		}
	}

	goroutineGrowthPercentage := float64(goroutineGrowthCount) / float64(len(ld.samples)-1)

	if goroutineGrowthPercentage > 0.8 {
		first := ld.samples[0]
		last := ld.samples[len(ld.samples)-1]
		goroutineGrowth := last.NumGoroutines - first.NumGoroutines
		duration := last.Timestamp.Sub(first.Timestamp)

		ld.logger.WithFields(logrus.Fields{
			"goroutine_growth": goroutineGrowth,
			"duration":         duration,
			"initial":          first.NumGoroutines,
			"current":          last.NumGoroutines,
			"rate_per_min":     float64(goroutineGrowth) / duration.Minutes(),
		}).Warn("Possible goroutine leak detected - sustained goroutine growth")
	}
}

// GetSamples returns a copy of current samples for analysis.
func (ld *LeakDetector) GetSamples() []MemorySample {
	ld.mu.RLock()
	defer ld.mu.RUnlock()

	// Return copy to avoid race conditions
	samples := make([]MemorySample, len(ld.samples))
	copy(samples, ld.samples)
	return samples
}

// Reset clears all collected samples.
func (ld *LeakDetector) Reset() {
	ld.mu.Lock()
	defer ld.mu.Unlock()

	ld.samples = make([]MemorySample, 0, ld.maxSamples)
	ld.logger.Info("Leak detector samples reset")
}


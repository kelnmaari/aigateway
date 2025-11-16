// Package health provides Kubernetes health check probes
// Version: v3.0.8
package health

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// ProbeStatus represents the status of a health check probe
type ProbeStatus int32

const (
	StatusHealthy ProbeStatus = iota
	StatusUnhealthy
	StatusUnknown
)

// Probe represents a health check probe
type Probe struct {
	name        string
	checkFunc   func(context.Context) error
	status      ProbeStatus
	lastCheck   time.Time
	lastError   error
	mu          sync.RWMutex
	logger      *logrus.Logger
}

// ProbeResult contains probe check result
type ProbeResult struct {
	Name      string      `json:"name"`
	Status    string      `json:"status"` // "healthy", "unhealthy", "unknown"
	LastCheck time.Time   `json:"last_check"`
	Error     string      `json:"error,omitempty"`
	Duration  int64       `json:"duration_ms"`
}

// NewProbe creates a new health probe
func NewProbe(name string, checkFunc func(context.Context) error, logger *logrus.Logger) *Probe {
	if logger == nil {
		logger = logrus.New()
	}
	
	return &Probe{
		name:      name,
		checkFunc: checkFunc,
		status:    StatusUnknown,
		logger:    logger,
	}
}

// Check runs the probe check
func (p *Probe) Check(ctx context.Context) ProbeResult {
	start := time.Now()
	
	err := p.checkFunc(ctx)
	duration := time.Since(start)
	
	p.mu.Lock()
	p.lastCheck = time.Now()
	p.lastError = err
	if err != nil {
		p.status = StatusUnhealthy
	} else {
		p.status = StatusHealthy
	}
	p.mu.Unlock()
	
	result := ProbeResult{
		Name:      p.name,
		LastCheck: p.lastCheck,
		Duration:  duration.Milliseconds(),
	}
	
	if err != nil {
		result.Status = "unhealthy"
		result.Error = err.Error()
	} else {
		result.Status = "healthy"
	}
	
	return result
}

// GetStatus returns current status (thread-safe)
func (p *Probe) GetStatus() ProbeStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// HealthChecker manages multiple health probes
type HealthChecker struct {
	probes     map[string]*Probe
	mu         sync.RWMutex
	logger     *logrus.Logger
	checkCount int64
	_pad       [56]byte // Cache line padding
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(logger *logrus.Logger) *HealthChecker {
	if logger == nil {
		logger = logrus.New()
	}
	
	return &HealthChecker{
		probes: make(map[string]*Probe),
		logger: logger,
	}
}

// RegisterProbe registers a new health probe
func (hc *HealthChecker) RegisterProbe(name string, checkFunc func(context.Context) error) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	hc.probes[name] = NewProbe(name, checkFunc, hc.logger)
	hc.logger.WithField("probe", name).Info("✅ Health probe registered")
}

// CheckAll runs all probes in parallel
func (hc *HealthChecker) CheckAll(ctx context.Context) map[string]ProbeResult {
	atomic.AddInt64(&hc.checkCount, 1)
	
	hc.mu.RLock()
	probes := make([]*Probe, 0, len(hc.probes))
	for _, probe := range hc.probes {
		probes = append(probes, probe)
	}
	hc.mu.RUnlock()
	
	results := make(map[string]ProbeResult)
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	for _, probe := range probes {
		wg.Add(1)
		go func(p *Probe) {
			defer wg.Done()
			
			result := p.Check(ctx)
			
			mu.Lock()
			results[p.name] = result
			mu.Unlock()
		}(probe)
	}
	
	wg.Wait()
	
	return results
}

// CheckProbe runs a specific probe
func (hc *HealthChecker) CheckProbe(ctx context.Context, name string) (ProbeResult, bool) {
	hc.mu.RLock()
	probe, exists := hc.probes[name]
	hc.mu.RUnlock()
	
	if !exists {
		return ProbeResult{}, false
	}
	
	return probe.Check(ctx), true
}

// IsHealthy returns true if all probes are healthy
func (hc *HealthChecker) IsHealthy(ctx context.Context) bool {
	results := hc.CheckAll(ctx)
	
	for _, result := range results {
		if result.Status != "healthy" {
			return false
		}
	}
	
	return len(results) > 0
}

// GetCheckCount returns total check count
func (hc *HealthChecker) GetCheckCount() int64 {
	return atomic.LoadInt64(&hc.checkCount)
}

// ─────────────────────────────────────────────────────────────────────────────
// Kubernetes Probe Handlers
// ─────────────────────────────────────────────────────────────────────────────

// LivenessProbe checks if the process is alive
type LivenessProbe struct {
	startTime time.Time
}

// NewLivenessProbe creates a liveness probe
func NewLivenessProbe() *LivenessProbe {
	return &LivenessProbe{
		startTime: time.Now(),
	}
}

// Check always returns healthy (process running = alive)
func (lp *LivenessProbe) Check(ctx context.Context) error {
	// Process is running, therefore alive
	return nil
}

// GetUptime returns uptime duration
func (lp *LivenessProbe) GetUptime() time.Duration {
	return time.Since(lp.startTime)
}

// ReadinessProbe checks if dependencies are ready
type ReadinessProbe struct {
	checker *HealthChecker
}

// NewReadinessProbe creates a readiness probe
func NewReadinessProbe(checker *HealthChecker) *ReadinessProbe {
	return &ReadinessProbe{
		checker: checker,
	}
}

// Check returns healthy if all dependencies are ready
func (rp *ReadinessProbe) Check(ctx context.Context) error {
	if rp.checker.IsHealthy(ctx) {
		return nil
	}
	
	return &UnhealthyError{Message: "dependencies not ready"}
}

// UnhealthyError represents an unhealthy state
type UnhealthyError struct {
	Message string
}

func (e *UnhealthyError) Error() string {
	return e.Message
}


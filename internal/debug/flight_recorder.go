// Package debug provides production debugging tools using Go 1.25 features
package debug

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/trace"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// FlightRecorderConfig configuration for production tracing
type FlightRecorderConfig struct {
	// Enabled enables flight recorder
	Enabled bool

	// MinAge minimum age of events in the window (default: 30s)
	MinAge time.Duration

	// MaxBytes maximum size of the trace window in bytes (default: 10MB)
	MaxBytes uint64

	// OutputDir directory to save traces on critical events
	OutputDir string

	// AutoSaveOnPanic automatically save trace on panic recovery
	AutoSaveOnPanic bool

	// AutoSaveOnCriticalError save trace on critical errors (non-panic)
	AutoSaveOnCriticalError bool
}

// FlightRecorderManager manages Go 1.25 FlightRecorder for production debugging
//
// Use cases:
// - Capture trace data around panic events
// - Debug performance issues in production
// - Investigate rare bugs without always-on tracing overhead
type FlightRecorderManager struct {
	config   FlightRecorderConfig
	recorder *trace.FlightRecorder
	logger   *logrus.Logger
	mu       sync.Mutex
	started  bool
}

// NewFlightRecorderManager creates a new flight recorder manager
func NewFlightRecorderManager(cfg FlightRecorderConfig, logger *logrus.Logger) *FlightRecorderManager {
	// Set defaults
	if cfg.MinAge == 0 {
		cfg.MinAge = 30 * time.Second
	}
	if cfg.MaxBytes == 0 {
		cfg.MaxBytes = 10 * 1024 * 1024 // 10MB
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "traces"
	}

	return &FlightRecorderManager{
		config: cfg,
		logger: logger,
	}
}

// Start starts the flight recorder
func (m *FlightRecorderManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.config.Enabled {
		m.logger.Info("FlightRecorder disabled")
		return nil
	}

	if m.started {
		return fmt.Errorf("flight recorder already started")
	}

	// Create Go 1.25 FlightRecorder
	recorderCfg := trace.FlightRecorderConfig{
		MinAge:   m.config.MinAge,
		MaxBytes: m.config.MaxBytes,
	}

	m.recorder = trace.NewFlightRecorder(recorderCfg)

	if err := m.recorder.Start(); err != nil {
		return fmt.Errorf("failed to start flight recorder: %w", err)
	}

	m.started = true

	m.logger.WithFields(logrus.Fields{
		"min_age_sec":  m.config.MinAge.Seconds(),
		"max_bytes_mb": float64(m.config.MaxBytes) / (1024 * 1024),
		"output_dir":   m.config.OutputDir,
	}).Info("FlightRecorder started")

	// Create output directory
	if err := os.MkdirAll(m.config.OutputDir, 0755); err != nil {
		m.logger.WithError(err).Warn("Failed to create trace output directory")
	}

	return nil
}

// Stop stops the flight recorder
func (m *FlightRecorderManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started || m.recorder == nil {
		return
	}

	m.recorder.Stop()
	m.started = false

	m.logger.Info("FlightRecorder stopped")
}

// Enabled returns whether the flight recorder is enabled and running
func (m *FlightRecorderManager) Enabled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.started && m.recorder != nil && m.recorder.Enabled()
}

// SaveTrace saves the current trace window to a file
//
// filename: optional custom filename (without extension)
// reason: reason for saving the trace (e.g., "panic", "timeout", "oom")
func (m *FlightRecorderManager) SaveTrace(ctx context.Context, filename, reason string) error {
	m.mu.Lock()
	recorder := m.recorder
	started := m.started
	m.mu.Unlock()

	if !started || recorder == nil {
		return fmt.Errorf("flight recorder not started")
	}

	// Generate filename if not provided
	if filename == "" {
		timestamp := time.Now().Format("20060102-150405")
		filename = fmt.Sprintf("trace-%s-%s", reason, timestamp)
	}

	tracePath := filepath.Join(m.config.OutputDir, filename+".trace")

	// Create trace file
	f, err := os.Create(tracePath)
	if err != nil {
		return fmt.Errorf("failed to create trace file: %w", err)
	}
	defer f.Close()

	// Write trace data
	start := time.Now()
	n, err := recorder.WriteTo(f)
	if err != nil {
		return fmt.Errorf("failed to write trace: %w", err)
	}

	duration := time.Since(start)

	m.logger.WithFields(logrus.Fields{
		"file":         tracePath,
		"size_mb":      float64(n) / (1024 * 1024),
		"reason":       reason,
		"duration_sec": duration.Seconds(),
	}).Info("FlightRecorder trace saved")

	return nil
}

// SaveTraceOnPanic saves trace and re-panics
//
// Usage in panic recovery:
//
//	defer func() {
//	    if r := recover(); r != nil {
//	        flightRecorder.SaveTraceOnPanic(context.Background(), r)
//	        panic(r) // Re-panic after saving
//	    }
//	}()
func (m *FlightRecorderManager) SaveTraceOnPanic(ctx context.Context, panicValue any) {
	if !m.config.AutoSaveOnPanic {
		return
	}

	m.logger.WithFields(logrus.Fields{
		"panic": panicValue,
	}).Error("Panic detected, saving FlightRecorder trace")

	if err := m.SaveTrace(ctx, "", "panic"); err != nil {
		m.logger.WithError(err).Error("Failed to save panic trace")
	}
}

// SaveTraceOnCriticalError saves trace for non-panic critical errors
func (m *FlightRecorderManager) SaveTraceOnCriticalError(ctx context.Context, errorType string, err error) {
	if !m.config.AutoSaveOnCriticalError {
		return
	}

	m.logger.WithFields(logrus.Fields{
		"error_type": errorType,
		"error":      err,
	}).Error("Critical error detected, saving FlightRecorder trace")

	if saveErr := m.SaveTrace(ctx, "", errorType); saveErr != nil {
		m.logger.WithError(saveErr).Error("Failed to save critical error trace")
	}
}

// WrapHandlerWithPanicRecovery wraps a function with panic recovery and trace saving
//
// Example:
//
//	flightRecorder.WrapHandlerWithPanicRecovery(func() {
//	    // Your code that might panic
//	})()
func (m *FlightRecorderManager) WrapHandlerWithPanicRecovery(fn func()) func() {
	return func() {
		defer func() {
			if r := recover(); r != nil {
				m.SaveTraceOnPanic(context.Background(), r)
				panic(r) // Re-panic to maintain expected behavior
			}
		}()
		fn()
	}
}

// GetTraceWriter returns a writer for manual trace collection
//
// Use when you want to save trace to custom destination (e.g., S3, database)
func (m *FlightRecorderManager) GetTraceWriter(w io.Writer) error {
	m.mu.Lock()
	recorder := m.recorder
	started := m.started
	m.mu.Unlock()

	if !started || recorder == nil {
		return fmt.Errorf("flight recorder not started")
	}

	_, err := recorder.WriteTo(w)
	return err
}

// DefaultConfig returns production-ready defaults
func DefaultConfig() FlightRecorderConfig {
	return FlightRecorderConfig{
		Enabled:                 true,
		MinAge:                  30 * time.Second,
		MaxBytes:                10 * 1024 * 1024, // 10MB
		OutputDir:               "traces",
		AutoSaveOnPanic:         true,
		AutoSaveOnCriticalError: false, // Opt-in for critical errors
	}
}

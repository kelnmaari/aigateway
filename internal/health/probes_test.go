// Package health - Health Probes Tests
// Version: v3.0.8
package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProbeSuccess tests successful probe check
func TestProbeSuccess(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	probe := NewProbe("test-probe", func(ctx context.Context) error {
		return nil
	}, logger)

	result := probe.Check(context.Background())

	assert.Equal(t, "test-probe", result.Name)
	assert.Equal(t, "healthy", result.Status)
	assert.Empty(t, result.Error)
	assert.Equal(t, StatusHealthy, probe.GetStatus())

	t.Log("✅ Successful probe check works")
}

// TestProbeFailure tests failed probe check
func TestProbeFailure(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	expectedErr := errors.New("probe failed")

	probe := NewProbe("failing-probe", func(ctx context.Context) error {
		return expectedErr
	}, logger)

	result := probe.Check(context.Background())

	assert.Equal(t, "failing-probe", result.Name)
	assert.Equal(t, "unhealthy", result.Status)
	assert.Equal(t, expectedErr.Error(), result.Error)
	assert.Equal(t, StatusUnhealthy, probe.GetStatus())

	t.Log("✅ Failed probe check works")
}

// TestHealthChecker tests multiple probes
func TestHealthChecker(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	checker := NewHealthChecker(logger)

	// Register probes
	checker.RegisterProbe("database", func(ctx context.Context) error {
		return nil // Healthy
	})

	checker.RegisterProbe("redis", func(ctx context.Context) error {
		return nil // Healthy
	})

	checker.RegisterProbe("external-api", func(ctx context.Context) error {
		return errors.New("timeout") // Unhealthy
	})

	// Check all
	results := checker.CheckAll(context.Background())

	assert.Len(t, results, 3)
	assert.Equal(t, "healthy", results["database"].Status)
	assert.Equal(t, "healthy", results["redis"].Status)
	assert.Equal(t, "unhealthy", results["external-api"].Status)
	assert.Contains(t, results["external-api"].Error, "timeout")

	// IsHealthy should return false (one probe failed)
	assert.False(t, checker.IsHealthy(context.Background()))

	t.Logf("✅ Health checker with %d probes works", len(results))
}

// TestHealthCheckerConcurrent tests concurrent checks
func TestHealthCheckerConcurrent(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	checker := NewHealthChecker(logger)

	// Register slow probes
	for i := range 10 {
		name := "probe-" + string(rune(i))
		checker.RegisterProbe(name, func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}

	// Check all (should run in parallel)
	start := time.Now()
	results := checker.CheckAll(context.Background())
	duration := time.Since(start)

	// Should complete in ~50ms (parallel), not 500ms (sequential)
	assert.Less(t, duration, 200*time.Millisecond)
	assert.Len(t, results, 10)

	t.Logf("✅ Concurrent checks completed in %v", duration)
}

// TestLivenessProbe tests liveness probe
func TestLivenessProbe(t *testing.T) {
	probe := NewLivenessProbe()

	// Should always be healthy (process running)
	err := probe.Check(context.Background())
	assert.NoError(t, err)

	// Check uptime
	time.Sleep(10 * time.Millisecond)
	uptime := probe.GetUptime()
	assert.Greater(t, uptime, 10*time.Millisecond)

	t.Logf("✅ Liveness probe works (uptime: %v)", uptime)
}

// TestReadinessProbe tests readiness probe
func TestReadinessProbe(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	checker := NewHealthChecker(logger)

	// Healthy dependencies
	checker.RegisterProbe("db", func(ctx context.Context) error {
		return nil
	})

	checker.RegisterProbe("cache", func(ctx context.Context) error {
		return nil
	})

	readinessProbe := NewReadinessProbe(checker)

	// Should be ready (all deps healthy)
	err := readinessProbe.Check(context.Background())
	require.NoError(t, err)

	// Add unhealthy dependency
	checker.RegisterProbe("failing", func(ctx context.Context) error {
		return errors.New("service unavailable")
	})

	// Should NOT be ready
	err = readinessProbe.Check(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")

	t.Log("✅ Readiness probe works")
}

// TestProbeTimeout tests probe timeout handling
func TestProbeTimeout(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	probe := NewProbe("slow-probe", func(ctx context.Context) error {
		select {
		case <-time.After(5 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, logger)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := probe.Check(ctx)

	assert.Equal(t, "unhealthy", result.Status)
	assert.Contains(t, result.Error, "context deadline exceeded")

	t.Log("✅ Probe timeout handling works")
}

// BenchmarkHealthCheckerParallel benchmarks parallel checks
func BenchmarkHealthCheckerParallel(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	checker := NewHealthChecker(logger)

	// Register 20 probes
	for i := range 20 {
		name := "probe-" + string(rune(i))
		checker.RegisterProbe(name, func(ctx context.Context) error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = checker.CheckAll(context.Background())
	}
}

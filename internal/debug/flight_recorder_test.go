package debug

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFlightRecorderManager(t *testing.T) {
	t.Attr("category", "debug")
	t.Attr("type", "unit")
	t.Attr("go_version", "1.25")

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	t.Run("sets defaults", func(t *testing.T) {
		cfg := FlightRecorderConfig{
			Enabled: true,
		}
		manager := NewFlightRecorderManager(cfg, logger)

		assert.NotNil(t, manager)
		assert.Equal(t, 30*time.Second, manager.config.MinAge)
		assert.Equal(t, uint64(10*1024*1024), manager.config.MaxBytes)
		assert.Equal(t, "traces", manager.config.OutputDir)
	})

	t.Run("respects custom config", func(t *testing.T) {
		cfg := FlightRecorderConfig{
			Enabled:  true,
			MinAge:   60 * time.Second,
			MaxBytes: 20 * 1024 * 1024,
			OutputDir: "custom-traces",
		}
		manager := NewFlightRecorderManager(cfg, logger)

		assert.Equal(t, 60*time.Second, manager.config.MinAge)
		assert.Equal(t, uint64(20*1024*1024), manager.config.MaxBytes)
		assert.Equal(t, "custom-traces", manager.config.OutputDir)
	})
}

func TestFlightRecorderManager_StartStop(t *testing.T) {
	t.Attr("category", "debug")
	t.Attr("type", "integration")
	t.Attr("go_version", "1.25")

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	t.Run("disabled config skips start", func(t *testing.T) {
		cfg := FlightRecorderConfig{
			Enabled: false,
		}
		manager := NewFlightRecorderManager(cfg, logger)

		err := manager.Start()
		assert.NoError(t, err)
		assert.False(t, manager.Enabled())
	})

	t.Run("start and stop successfully", func(t *testing.T) {
		cfg := FlightRecorderConfig{
			Enabled:  true,
			OutputDir: filepath.Join(os.TempDir(), "test-traces"),
		}
		manager := NewFlightRecorderManager(cfg, logger)

		err := manager.Start()
		require.NoError(t, err)
		assert.True(t, manager.Enabled())

		manager.Stop()
		assert.False(t, manager.Enabled())

		// Cleanup
		os.RemoveAll(cfg.OutputDir)
	})

	t.Run("prevents double start", func(t *testing.T) {
		cfg := FlightRecorderConfig{
			Enabled:  true,
			OutputDir: filepath.Join(os.TempDir(), "test-traces-2"),
		}
		manager := NewFlightRecorderManager(cfg, logger)

		err := manager.Start()
		require.NoError(t, err)

		// Second start should fail
		err = manager.Start()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already started")

		manager.Stop()
		os.RemoveAll(cfg.OutputDir)
	})
}

func TestFlightRecorderManager_SaveTrace(t *testing.T) {
	t.Attr("category", "debug")
	t.Attr("type", "integration")

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	outputDir := filepath.Join(os.TempDir(), fmt.Sprintf("test-traces-%d", time.Now().UnixNano()))
	defer os.RemoveAll(outputDir)

	cfg := FlightRecorderConfig{
		Enabled:  true,
		OutputDir: outputDir,
	}
	manager := NewFlightRecorderManager(cfg, logger)

	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	t.Run("saves trace with custom filename", func(t *testing.T) {
		ctx := context.Background()
		filename := "test-trace"
		reason := "test"

		err := manager.SaveTrace(ctx, filename, reason)
		require.NoError(t, err)

		tracePath := filepath.Join(outputDir, filename+".trace")
		assert.FileExists(t, tracePath)

		// Check file is not empty
		info, err := os.Stat(tracePath)
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
	})

	t.Run("auto-generates filename", func(t *testing.T) {
		ctx := context.Background()
		reason := "panic"

		err := manager.SaveTrace(ctx, "", reason)
		require.NoError(t, err)

		// Check files in output dir
		files, err := os.ReadDir(outputDir)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(files), 1)
	})
}

func TestFlightRecorderManager_SaveTraceOnPanic(t *testing.T) {
	t.Attr("category", "debug")
	t.Attr("type", "integration")

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	outputDir := filepath.Join(os.TempDir(), fmt.Sprintf("panic-traces-%d", time.Now().UnixNano()))
	defer os.RemoveAll(outputDir)

	t.Run("saves trace on panic when enabled", func(t *testing.T) {
		cfg := FlightRecorderConfig{
			Enabled:         true,
			OutputDir:       outputDir,
			AutoSaveOnPanic: true,
		}
		manager := NewFlightRecorderManager(cfg, logger)

		err := manager.Start()
		require.NoError(t, err)
		defer manager.Stop()

		panicValue := "test panic"
		manager.SaveTraceOnPanic(context.Background(), panicValue)

		// Check trace was saved
		files, err := os.ReadDir(outputDir)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(files), 1)
	})

	t.Run("skips save when AutoSaveOnPanic disabled", func(t *testing.T) {
		outputDir2 := filepath.Join(os.TempDir(), fmt.Sprintf("no-panic-traces-%d", time.Now().UnixNano()))
		defer os.RemoveAll(outputDir2)

		cfg := FlightRecorderConfig{
			Enabled:         true,
			OutputDir:       outputDir2,
			AutoSaveOnPanic: false, // Disabled
		}
		manager := NewFlightRecorderManager(cfg, logger)

		err := manager.Start()
		require.NoError(t, err)
		defer manager.Stop()

		manager.SaveTraceOnPanic(context.Background(), "test")

		// Check no trace was saved
		files, err := os.ReadDir(outputDir2)
		require.NoError(t, err)
		assert.Equal(t, 0, len(files))
	})
}

func TestFlightRecorderManager_WrapHandlerWithPanicRecovery(t *testing.T) {
	t.Attr("category", "debug")
	t.Attr("type", "integration")

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	outputDir := filepath.Join(os.TempDir(), fmt.Sprintf("wrap-traces-%d", time.Now().UnixNano()))
	defer os.RemoveAll(outputDir)

	cfg := FlightRecorderConfig{
		Enabled:         true,
		OutputDir:       outputDir,
		AutoSaveOnPanic: true,
	}
	manager := NewFlightRecorderManager(cfg, logger)

	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	t.Run("captures panic and saves trace", func(t *testing.T) {
		wrapped := manager.WrapHandlerWithPanicRecovery(func() {
			panic("test panic in wrapped handler")
		})

		// Should panic after saving trace
		assert.Panics(t, wrapped)

		// Check trace was saved
		files, err := os.ReadDir(outputDir)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(files), 1)
	})

	t.Run("does not panic if no error", func(t *testing.T) {
		wrapped := manager.WrapHandlerWithPanicRecovery(func() {
			// No panic
		})

		assert.NotPanics(t, wrapped)
	})
}

func TestFlightRecorderManager_GetTraceWriter(t *testing.T) {
	t.Attr("category", "debug")
	t.Attr("type", "integration")

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	outputDir := filepath.Join(os.TempDir(), fmt.Sprintf("writer-traces-%d", time.Now().UnixNano()))
	defer os.RemoveAll(outputDir)

	cfg := FlightRecorderConfig{
		Enabled:  true,
		OutputDir: outputDir,
	}
	manager := NewFlightRecorderManager(cfg, logger)

	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	t.Run("writes trace to custom writer", func(t *testing.T) {
		var buf bytes.Buffer
		err := manager.GetTraceWriter(&buf)
		require.NoError(t, err)

		assert.Greater(t, buf.Len(), 0, "Trace data should be written")
	})

	t.Run("fails if not started", func(t *testing.T) {
		manager2 := NewFlightRecorderManager(FlightRecorderConfig{Enabled: false}, logger)

		var buf bytes.Buffer
		err := manager2.GetTraceWriter(&buf)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not started")
	})
}

func TestDefaultConfig(t *testing.T) {
	t.Attr("category", "debug")
	t.Attr("type", "unit")

	cfg := DefaultConfig()

	assert.True(t, cfg.Enabled)
	assert.Equal(t, 30*time.Second, cfg.MinAge)
	assert.Equal(t, uint64(10*1024*1024), cfg.MaxBytes)
	assert.Equal(t, "traces", cfg.OutputDir)
	assert.True(t, cfg.AutoSaveOnPanic)
	assert.False(t, cfg.AutoSaveOnCriticalError)
}

// Benchmark tests
func BenchmarkFlightRecorderManager_SaveTrace(b *testing.B) {
	b.Attr("category", "benchmarks")
	b.Attr("type", "performance")

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	outputDir := filepath.Join(os.TempDir(), fmt.Sprintf("bench-traces-%d", time.Now().UnixNano()))
	defer os.RemoveAll(outputDir)

	cfg := FlightRecorderConfig{
		Enabled:  true,
		OutputDir: outputDir,
	}
	manager := NewFlightRecorderManager(cfg, logger)
	manager.Start()
	defer manager.Stop()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.SaveTrace(ctx, fmt.Sprintf("bench-%d", i), "benchmark")
	}
}


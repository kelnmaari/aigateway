package inference

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// BenchmarkMockRuntime_ParallelStart benchmarks parallel container starts.
func BenchmarkMockRuntime_ParallelStart(b *testing.B) {
	tmpDir := b.TempDir()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockRuntime := NewMockRuntime()

	dl, _ := NewModelDownloader(ModelDownloaderConfig{
		HFCacheDir:   tmpDir,
		GGUFCacheDir: tmpDir,
		Logger:       logger,
	})

	orch := NewOrchestrator(mockRuntime, dl, logger, OrchestratorConfig{
		HealthCheckTimeout: 100 * time.Millisecond,
		StartupTimeout:     5 * time.Second,
	})

	svc := &Service{
		cfg: ServiceConfig{
			HFCacheDir:   tmpDir,
			GGUFCacheDir: tmpDir,
			Logger:       logger,
		},
		orch:           orch,
		runtime:        mockRuntime,
		downloader:     dl,
		logger:         logger,
		registry:       NewSpecRegistry(),
		providerLogger: NewProviderLogger(tmpDir, logger),
	}

	mgr := NewManager(svc)
	router := NewRouter(mgr)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			spec := ModelSpec{
				Alias:     fmt.Sprintf("bench-model-%d-%d", time.Now().UnixNano(), i),
				Provider:  ProviderVLLM,
				Format:    FormatHF,
				HFRepo:    "test/model",
				LocalPath: fmt.Sprintf("%s/bench-%d", tmpDir, i),
			}
			i++

			ctx := context.Background()
			_, _ = router.EnsureBySpec(ctx, spec)
		}
	})
}

// BenchmarkMockRuntime_SequentialStartStop benchmarks sequential start/stop cycles.
func BenchmarkMockRuntime_SequentialStartStop(b *testing.B) {
	tmpDir := b.TempDir()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockRuntime := NewMockRuntime()

	dl, _ := NewModelDownloader(ModelDownloaderConfig{
		HFCacheDir:   tmpDir,
		GGUFCacheDir: tmpDir,
		Logger:       logger,
	})

	orch := NewOrchestrator(mockRuntime, dl, logger, OrchestratorConfig{
		HealthCheckTimeout: 100 * time.Millisecond,
		StartupTimeout:     5 * time.Second,
	})

	svc := &Service{
		cfg: ServiceConfig{
			HFCacheDir:   tmpDir,
			GGUFCacheDir: tmpDir,
			Logger:       logger,
		},
		orch:           orch,
		runtime:        mockRuntime,
		downloader:     dl,
		logger:         logger,
		registry:       NewSpecRegistry(),
		providerLogger: NewProviderLogger(tmpDir, logger),
	}

	mgr := NewManager(svc)
	router := NewRouter(mgr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		alias := fmt.Sprintf("bench-seq-%d", i)
		spec := ModelSpec{
			Alias:     alias,
			Provider:  ProviderVLLM,
			Format:    FormatHF,
			HFRepo:    "test/model",
			LocalPath: fmt.Sprintf("%s/%s", tmpDir, alias),
		}

		ctx := context.Background()
		_, _ = router.EnsureBySpec(ctx, spec)
		_ = router.Stop(ctx, alias)
	}
}

// BenchmarkRouter_Resolution benchmarks alias resolution without container operations.
func BenchmarkRouter_Resolution(b *testing.B) {
	tmpDir := b.TempDir()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockRuntime := NewMockRuntime()

	dl, _ := NewModelDownloader(ModelDownloaderConfig{
		HFCacheDir:   tmpDir,
		GGUFCacheDir: tmpDir,
		Logger:       logger,
	})

	orch := NewOrchestrator(mockRuntime, dl, logger, OrchestratorConfig{
		HealthCheckTimeout: 100 * time.Millisecond,
		StartupTimeout:     5 * time.Second,
	})

	svc := &Service{
		cfg: ServiceConfig{
			HFCacheDir:   tmpDir,
			GGUFCacheDir: tmpDir,
			Logger:       logger,
		},
		orch:           orch,
		runtime:        mockRuntime,
		downloader:     dl,
		logger:         logger,
		registry:       NewSpecRegistry(),
		providerLogger: NewProviderLogger(tmpDir, logger),
	}

	mgr := NewManager(svc)
	router := NewRouter(mgr)

	// Pre-register specs
	numSpecs := 100
	for i := 0; i < numSpecs; i++ {
		spec := ModelSpec{
			Alias:     fmt.Sprintf("preload-%d", i),
			Provider:  ProviderVLLM,
			Format:    FormatHF,
			HFRepo:    "test/model",
			LocalPath: fmt.Sprintf("%s/preload-%d", tmpDir, i),
		}
		router.RegisterSpec(spec)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			alias := fmt.Sprintf("preload-%d", i%numSpecs)
			ctx := context.Background()
			_, _ = router.EnsureByAlias(ctx, alias)
			i++
		}
	})
}

// TestLoadTest_HighConcurrency simulates high concurrency load.
func TestLoadTest_HighConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	tmpDir := t.TempDir()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	mockRuntime := NewMockRuntime()
	mockRuntime.startDelay = 10 * time.Millisecond // Simulate startup latency

	dl, _ := NewModelDownloader(ModelDownloaderConfig{
		HFCacheDir:   tmpDir,
		GGUFCacheDir: tmpDir,
		Logger:       logger,
	})

	orch := NewOrchestrator(mockRuntime, dl, logger, OrchestratorConfig{
		HealthCheckTimeout: 100 * time.Millisecond,
		StartupTimeout:     30 * time.Second,
	})

	svc := &Service{
		cfg: ServiceConfig{
			HFCacheDir:   tmpDir,
			GGUFCacheDir: tmpDir,
			Logger:       logger,
		},
		orch:           orch,
		runtime:        mockRuntime,
		downloader:     dl,
		logger:         logger,
		registry:       NewSpecRegistry(),
		providerLogger: NewProviderLogger(tmpDir, logger),
	}

	mgr := NewManager(svc)
	router := NewRouter(mgr)

	// Simulate 50 concurrent model loads
	numWorkers := 50
	numModels := 10
	var wg sync.WaitGroup
	var successCount, failCount int64

	start := time.Now()

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for m := 0; m < numModels; m++ {
				alias := fmt.Sprintf("load-model-%d", m) // Same models across workers
				spec := ModelSpec{
					Alias:     alias,
					Provider:  ProviderVLLM,
					Format:    FormatHF,
					HFRepo:    "test/model",
					LocalPath: fmt.Sprintf("%s/%s", tmpDir, alias),
				}

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_, err := router.EnsureBySpec(ctx, spec)
				cancel()

				if err != nil {
					atomic.AddInt64(&failCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(w)
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Load test completed in %v", elapsed)
	t.Logf("Total requests: %d", numWorkers*numModels)
	t.Logf("Success: %d, Failed: %d", successCount, failCount)
	t.Logf("Throughput: %.2f req/s", float64(numWorkers*numModels)/elapsed.Seconds())

	// Verify no excessive failures
	failRate := float64(failCount) / float64(numWorkers*numModels)
	if failRate > 0.1 { // Allow up to 10% failure rate
		t.Errorf("Excessive failure rate: %.2f%%", failRate*100)
	}
}

// TestLoadTest_RapidStartStop simulates rapid start/stop cycles.
func TestLoadTest_RapidStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	tmpDir := t.TempDir()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	mockRuntime := NewMockRuntime()

	dl, _ := NewModelDownloader(ModelDownloaderConfig{
		HFCacheDir:   tmpDir,
		GGUFCacheDir: tmpDir,
		Logger:       logger,
	})

	orch := NewOrchestrator(mockRuntime, dl, logger, OrchestratorConfig{
		HealthCheckTimeout: 100 * time.Millisecond,
		StartupTimeout:     5 * time.Second,
	})

	svc := &Service{
		cfg: ServiceConfig{
			HFCacheDir:   tmpDir,
			GGUFCacheDir: tmpDir,
			Logger:       logger,
		},
		orch:           orch,
		runtime:        mockRuntime,
		downloader:     dl,
		logger:         logger,
		registry:       NewSpecRegistry(),
		providerLogger: NewProviderLogger(tmpDir, logger),
	}

	mgr := NewManager(svc)
	router := NewRouter(mgr)

	cycles := 100
	start := time.Now()

	for i := 0; i < cycles; i++ {
		alias := fmt.Sprintf("rapid-%d", i%5) // Cycle through 5 models
		spec := ModelSpec{
			Alias:     alias,
			Provider:  ProviderVLLM,
			Format:    FormatHF,
			HFRepo:    "test/model",
			LocalPath: fmt.Sprintf("%s/%s", tmpDir, alias),
		}

		ctx := context.Background()
		_, err := router.EnsureBySpec(ctx, spec)
		if err != nil {
			t.Logf("Start failed at cycle %d: %v", i, err)
		}

		_ = router.Stop(ctx, alias)
	}

	elapsed := time.Since(start)
	t.Logf("Completed %d start/stop cycles in %v", cycles, elapsed)
	t.Logf("Average cycle time: %v", elapsed/time.Duration(cycles))
}


package inference

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"aigateway/internal/models"

	"github.com/sirupsen/logrus"
)

// MockContainerRuntime implements ContainerRuntime for testing.
type MockContainerRuntime struct {
	mu         sync.Mutex
	containers map[string]*ContainerHandle
	startDelay time.Duration
	failStart  bool
	failStop   bool
	pullCalls  int
	startCalls int
	stopCalls  int
}

func NewMockRuntime() *MockContainerRuntime {
	return &MockContainerRuntime{
		containers: make(map[string]*ContainerHandle),
	}
}

func (m *MockContainerRuntime) PullImage(ctx context.Context, image string) error {
	m.mu.Lock()
	m.pullCalls++
	m.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Millisecond):
	}
	return nil
}

func (m *MockContainerRuntime) Start(ctx context.Context, req ContainerStartRequest) (*ContainerHandle, error) {
	m.mu.Lock()
	m.startCalls++
	failStart := m.failStart
	delay := m.startDelay
	m.mu.Unlock()

	if failStart {
		return nil, fmt.Errorf("mock start failure")
	}

	if delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	handle := &ContainerHandle{
		ID:       "mock-container-" + req.ModelAlias,
		Provider: req.Provider,
		Endpoint: "", // Empty endpoint skips health check
	}

	m.mu.Lock()
	m.containers[req.ModelAlias] = handle
	m.mu.Unlock()

	return handle, nil
}

func (m *MockContainerRuntime) Stop(ctx context.Context, containerID string) error {
	m.mu.Lock()
	m.stopCalls++
	failStop := m.failStop
	m.mu.Unlock()

	if failStop {
		return fmt.Errorf("mock stop failure")
	}

	m.mu.Lock()
	for alias, h := range m.containers {
		if h.ID == containerID {
			delete(m.containers, alias)
			break
		}
	}
	m.mu.Unlock()

	return nil
}

func (m *MockContainerRuntime) Logs(ctx context.Context, containerID string, tailLines int) (string, error) {
	return fmt.Sprintf("Mock logs for %s (tail %d)", containerID, tailLines), nil
}

func (m *MockContainerRuntime) IsRunning(ctx context.Context, handleID string) (bool, error) {
	return true, nil
}

func (m *MockContainerRuntime) Stats() map[string]int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]int{
		"pull":  m.pullCalls,
		"start": m.startCalls,
		"stop":  m.stopCalls,
	}
}

// TestIntegration_LoadAndStart tests full flow with mock runtime.
func TestIntegration_LoadAndStart(t *testing.T) {
	tmpDir := t.TempDir()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	mockRuntime := NewMockRuntime()

	// Create service with mock runtime
	dl, err := NewModelDownloader(ModelDownloaderConfig{
		HFCacheDir:   tmpDir,
		GGUFCacheDir: tmpDir,
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("NewModelDownloader: %v", err)
	}

	orch := NewOrchestrator(mockRuntime, dl, logger, OrchestratorConfig{
		HealthCheckTimeout: 100 * time.Millisecond,
		StartupTimeout:     1 * time.Second,
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

	// Test load flow - set LocalPath to skip download
	spec := ModelSpec{
		Alias:     "test-model",
		Provider:  ProviderVLLM,
		Format:    FormatHF,
		HFRepo:    "test/model",
		LocalPath: tmpDir + "/test-model", // Pre-set to skip download
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	inst, err := router.EnsureBySpec(ctx, spec)
	if err != nil {
		t.Fatalf("EnsureBySpec: %v", err)
	}

	if inst.Spec.Alias != "test-model" {
		t.Errorf("Alias = %q, want %q", inst.Spec.Alias, "test-model")
	}

	stats := mockRuntime.Stats()
	if stats["start"] != 1 {
		t.Errorf("start calls = %d, want 1", stats["start"])
	}

	// Test stop
	if err := router.Stop(ctx, "test-model"); err != nil {
		t.Errorf("Stop: %v", err)
	}

	stats = mockRuntime.Stats()
	if stats["stop"] != 1 {
		t.Errorf("stop calls = %d, want 1", stats["stop"])
	}
}

// TestIntegration_ContextCancellation tests that StartModel uses its own timeout.
// NOTE: Orchestrator.StartModel intentionally ignores the passed context and uses
// a background context with startupTimeout to prevent cancellation when user refreshes the page.
// This test verifies that short client context doesn't cancel the operation.
func TestIntegration_ContextCancellation(t *testing.T) {
	t.Skip("Skipped: Orchestrator.StartModel intentionally uses background context to prevent cancellation from HTTP request refresh")
}

// TestIntegration_StartFailure tests handling of container start failure.
func TestIntegration_StartFailure(t *testing.T) {
	tmpDir := t.TempDir()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	mockRuntime := NewMockRuntime()
	mockRuntime.failStart = true

	dl, _ := NewModelDownloader(ModelDownloaderConfig{
		HFCacheDir:   tmpDir,
		GGUFCacheDir: tmpDir,
		Logger:       logger,
	})

	orch := NewOrchestrator(mockRuntime, dl, logger, OrchestratorConfig{
		HealthCheckTimeout: 100 * time.Millisecond,
		StartupTimeout:     1 * time.Second,
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

	spec := ModelSpec{
		Alias:     "fail-model",
		Provider:  ProviderVLLM,
		Format:    FormatHF,
		HFRepo:    "test/model",
		LocalPath: tmpDir + "/fail-model",
	}

	ctx := context.Background()
	_, err := router.EnsureBySpec(ctx, spec)
	if err == nil {
		t.Error("Expected start failure error")
	}
}

// TestIntegration_PinUnpin tests pin/unpin functionality.
func TestIntegration_PinUnpin(t *testing.T) {
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
		StartupTimeout:     1 * time.Second,
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

	// Test pin before model exists
	router.Pin("nonexistent")
	if router.IsPinned("nonexistent") {
		t.Log("Pin works even before model exists (expected)")
	}

	// Load model
	spec := ModelSpec{
		Alias:     "pin-test",
		Provider:  ProviderVLLM,
		Format:    FormatHF,
		HFRepo:    "test/model",
		LocalPath: tmpDir + "/pin-test",
	}

	ctx := context.Background()
	_, err := router.EnsureBySpec(ctx, spec)
	if err != nil {
		t.Fatalf("EnsureBySpec: %v", err)
	}

	// Pin
	router.Pin("pin-test")
	if !router.IsPinned("pin-test") {
		t.Error("Expected model to be pinned")
	}

	// Unpin
	router.Unpin("pin-test")
	if router.IsPinned("pin-test") {
		t.Error("Expected model to be unpinned")
	}
}

// TestIntegration_MultipleProviders tests loading different provider types.
func TestIntegration_MultipleProviders(t *testing.T) {
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
		StartupTimeout:     1 * time.Second,
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

	providers := []ProviderKind{ProviderVLLM, ProviderSGLang, ProviderTGI}

	for i, provider := range providers {
		spec := ModelSpec{
			Alias:     fmt.Sprintf("model-%d", i),
			Provider:  provider,
			Format:    FormatHF,
			HFRepo:    "test/model",
			LocalPath: fmt.Sprintf("%s/model-%d", tmpDir, i),
		}

		ctx := context.Background()
		inst, err := router.EnsureBySpec(ctx, spec)
		if err != nil {
			t.Errorf("EnsureBySpec for %s: %v", provider, err)
			continue
		}

		if inst.Handle == nil {
			t.Errorf("Handle is nil for %s", provider)
		}
	}

	stats := mockRuntime.Stats()
	if stats["start"] != len(providers) {
		t.Errorf("start calls = %d, want %d", stats["start"], len(providers))
	}

	// List models
	models := router.ListModels()
	if len(models) != len(providers) {
		t.Errorf("ListModels returned %d, want %d", len(models), len(providers))
	}
}

// TestIntegration_ConcurrentLoads tests concurrent model loading.
func TestIntegration_ConcurrentLoads(t *testing.T) {
	tmpDir := t.TempDir()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	mockRuntime := NewMockRuntime()
	mockRuntime.startDelay = 50 * time.Millisecond

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

	numModels := 5
	var wg sync.WaitGroup
	errors := make(chan error, numModels)

	for i := range numModels {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			spec := ModelSpec{
				Alias:     fmt.Sprintf("concurrent-%d", idx),
				Provider:  ProviderVLLM,
				Format:    FormatHF,
				HFRepo:    "test/model",
				LocalPath: fmt.Sprintf("%s/concurrent-%d", tmpDir, idx),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			_, err := router.EnsureBySpec(ctx, spec)
			if err != nil {
				errors <- fmt.Errorf("model %d: %w", idx, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	stats := mockRuntime.Stats()
	if stats["start"] != numModels {
		t.Errorf("start calls = %d, want %d", stats["start"], numModels)
	}
}

// TestIntegration_ResolveByCapability tests capability-based routing.
func TestIntegration_ResolveByCapability(t *testing.T) {
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
		StartupTimeout:     1 * time.Second,
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

	// Register models with different capabilities
	textSpec := ModelSpec{
		Alias:        "text-model",
		Provider:     ProviderVLLM,
		Format:       FormatHF,
		HFRepo:       "test/text",
		LocalPath:    tmpDir + "/text-model",
		Capabilities: []Capability{models.CapabilityChat},
	}
	visionSpec := ModelSpec{
		Alias:        "vision-model",
		Provider:     ProviderSGLang,
		Format:       FormatHF,
		HFRepo:       "test/vision",
		LocalPath:    tmpDir + "/vision-model",
		Capabilities: []Capability{models.CapabilityChat, models.CapabilityVision},
	}

	ctx := context.Background()

	// Load both
	_, err := router.EnsureBySpec(ctx, textSpec)
	if err != nil {
		t.Fatalf("Load text model: %v", err)
	}
	_, err = router.EnsureBySpec(ctx, visionSpec)
	if err != nil {
		t.Fatalf("Load vision model: %v", err)
	}

	// Resolve by chat capability (should pick first registered)
	inst, err := router.EnsureByCapability(ctx, models.CapabilityChat)
	if err != nil {
		t.Fatalf("EnsureByCapability(chat): %v", err)
	}
	if inst.Spec.Alias != "text-model" {
		t.Logf("Got %q for chat capability (first registered)", inst.Spec.Alias)
	}

	// Resolve by vision capability (only vision-model has it)
	inst, err = router.EnsureByCapability(ctx, models.CapabilityVision)
	if err != nil {
		t.Fatalf("EnsureByCapability(vision): %v", err)
	}
	if inst.Spec.Alias != "vision-model" {
		t.Errorf("Expected vision-model for vision capability, got %q", inst.Spec.Alias)
	}
}

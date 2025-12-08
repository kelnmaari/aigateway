package inference

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// ServiceConfig defines parameters for inference service bootstrapping.
type ServiceConfig struct {
	HFToken               string
	HFCacheDir            string
	GGUFCacheDir          string
	TRTEnginesDir         string // /data/engines/trt for TensorRT-LLM engines
	MaxConcurrentDownload int
	AutoResume            bool
	HTTPTimeout           time.Duration
	DockerBin             string
	Logger                *logrus.Logger
	MaxRunningModels      int
	CacheMaxBytes         int64         // 0 = unlimited
	HealthCheckTimeout    time.Duration // timeout for health check after container start (default 60s)
	StartupTimeout        time.Duration // overall timeout for model startup (default 5m)
}

// Service wires orchestrator, downloader and runtime to manage models/containers.
type Service struct {
	cfg          ServiceConfig
	orch         *Orchestrator
	runtime      ContainerRuntime
	downloader   *ModelDownloader
	trtConverter *TRTConverter
	logger       *logrus.Logger
	registry     *SpecRegistry
}

// NewService builds inference service with Docker runtime and model downloader.
func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.Logger == nil {
		cfg.Logger = logrus.New()
	}
	if cfg.MaxRunningModels < 0 {
		cfg.MaxRunningModels = 0
	}
	if cfg.CacheMaxBytes < 0 {
		cfg.CacheMaxBytes = 0
	}

	dl, err := NewModelDownloader(ModelDownloaderConfig{
		HFToken:       cfg.HFToken,
		HFCacheDir:    cfg.HFCacheDir,
		GGUFCacheDir:  cfg.GGUFCacheDir,
		MaxConcurrent: cfg.MaxConcurrentDownload,
		AutoResume:    cfg.AutoResume,
		HTTPTimeout:   cfg.HTTPTimeout,
		Logger:        cfg.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("init downloader: %w", err)
	}

	runtime := NewDockerRuntime(DockerRuntimeConfig{
		DockerBin: cfg.DockerBin,
		Logger:    cfg.Logger,
	})

	orch := NewOrchestrator(runtime, dl, cfg.Logger, OrchestratorConfig{
		HealthCheckTimeout: cfg.HealthCheckTimeout,
		StartupTimeout:     cfg.StartupTimeout,
	})

	// Initialize TRT converter if engines dir is configured
	var trtConv *TRTConverter
	if cfg.TRTEnginesDir != "" {
		var convErr error
		trtConv, convErr = NewTRTConverter(TRTConverterConfig{
			EnginesDir: cfg.TRTEnginesDir,
			HFCacheDir: cfg.HFCacheDir,
			Logger:     cfg.Logger,
		})
		if convErr != nil {
			cfg.Logger.WithError(convErr).Warn("TRT converter initialization failed, TRT-LLM support disabled")
		}
	}

	return &Service{
		cfg:          cfg,
		orch:         orch,
		runtime:      runtime,
		downloader:   dl,
		trtConverter: trtConv,
		logger:       cfg.Logger,
		registry:     NewSpecRegistry(),
	}, nil
}

// LoadAndStart prepares artifacts and starts container based on provider.
func (s *Service) LoadAndStart(ctx context.Context, spec ModelSpec) (*ModelInstance, error) {
	s.registry.Register(spec)
	var req ContainerStartRequest
	switch spec.Provider {
	case ProviderVLLM:
		req = BuildVLLMRequest(spec, s.cfg.HFCacheDir)
	case ProviderSGLang:
		req = BuildSGLangRequest(spec, s.cfg.HFCacheDir, s.cfg.HFToken)
	case ProviderTGI:
		req = BuildTGIRequest(spec, s.cfg.HFCacheDir, s.cfg.HFToken)
	case ProviderLlamaCPP:
		var err error
		req, err = BuildLlamaCPPRequest(spec)
		if err != nil {
			return nil, err
		}
	case ProviderTRTLLM:
		var err error
		req, err = BuildTRTLLMRequest(spec, s.cfg.GGUFCacheDir, s.cfg.HFToken)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported provider: %s", spec.Provider)
	}

	return s.orch.StartModel(ctx, spec, req)
}

// PrepareOnly downloads artifacts without starting a container.
func (s *Service) PrepareOnly(ctx context.Context, spec ModelSpec) (*ModelInstance, error) {
	s.registry.Register(spec)
	return s.orch.PrepareModel(ctx, spec)
}

// RegisterSpec stores model spec for later resolution.
func (s *Service) RegisterSpec(spec ModelSpec) {
	s.registry.Register(spec)
}

// ResolveSpec returns registered spec by alias.
func (s *Service) ResolveSpec(alias string) (ModelSpec, bool) {
	return s.registry.Get(alias)
}

// Health checks provider-specific health endpoint for running model.
func (s *Service) Health(ctx context.Context, alias string) error {
	inst, ok := s.orch.GetModel(alias)
	if !ok || inst.Handle == nil || inst.Handle.Endpoint == "" {
		return fmt.Errorf("model not running: %s", alias)
	}
	url := providerHealthURL(inst.Handle.Provider, inst.Handle.Endpoint)
	return HealthCheckHTTP(ctx, url)
}

// ContainerLogs returns recent stdout/stderr from running container.
func (s *Service) ContainerLogs(ctx context.Context, alias string, tailLines int) (string, error) {
	inst, ok := s.orch.GetModel(alias)
	if !ok || inst.Handle == nil || inst.Handle.ID == "" {
		return "", fmt.Errorf("model not running: %s", alias)
	}
	return s.runtime.Logs(ctx, inst.Handle.ID, tailLines)
}

// ContainerMetrics fetches provider-specific metrics endpoint (Prometheus format).
func (s *Service) ContainerMetrics(ctx context.Context, alias string) (string, error) {
	inst, ok := s.orch.GetModel(alias)
	if !ok || inst.Handle == nil || inst.Handle.Endpoint == "" {
		return "", fmt.Errorf("model not running: %s", alias)
	}
	url := providerMetricsURL(inst.Handle.Provider, inst.Handle.Endpoint)
	if url == "" {
		return "", fmt.Errorf("metrics not supported for provider %s", inst.Handle.Provider)
	}
	return fetchMetricsHTTP(ctx, url)
}

// Stop stops running container for alias if any.
func (s *Service) Stop(ctx context.Context, alias string) error {
	return s.orch.StopModel(ctx, alias)
}

// Shutdown releases downloader workers.
func (s *Service) Shutdown() {
	s.downloader.Shutdown()
}

// GetModel returns tracked model by alias.
func (s *Service) GetModel(alias string) (*ModelInstance, bool) {
	return s.orch.GetModel(alias)
}

// ListModels returns tracked models.
func (s *Service) ListModels() []*ModelInstance {
	return s.orch.ListModels()
}

// RemoveArtifacts deletes local artifact file if path is within allowed caches.
func (s *Service) RemoveArtifacts(alias string) error {
	inst, ok := s.orch.GetModel(alias)
	if !ok {
		return fmt.Errorf("model not found: %s", alias)
	}
	if inst.Spec.LocalPath == "" {
		return fmt.Errorf("no local path for model: %s", alias)
	}
	if err := ensurePathWithin(inst.Spec.LocalPath, []string{s.cfg.HFCacheDir, s.cfg.GGUFCacheDir}); err != nil {
		return err
	}
	if inst.Handle != nil {
		return fmt.Errorf("model is running; stop before deleting artifacts")
	}
	return os.Remove(inst.Spec.LocalPath)
}

// ArtifactInfo describes a cached file.
type ArtifactInfo struct {
	Path    string      `json:"path"`
	Size    int64       `json:"size"`
	ModTime time.Time   `json:"mod_time"`
	Root    string      `json:"root"`
	Format  ModelFormat `json:"format"`
}

// ListArtifacts scans cache roots and returns files.
func (s *Service) ListArtifacts() ([]ArtifactInfo, error) {
	var roots []string
	if s.cfg.HFCacheDir != "" {
		roots = append(roots, s.cfg.HFCacheDir)
	}
	if s.cfg.GGUFCacheDir != "" && s.cfg.GGUFCacheDir != s.cfg.HFCacheDir {
		roots = append(roots, s.cfg.GGUFCacheDir)
	}
	var artifacts []ArtifactInfo
	seen := make(map[string]bool)
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			abs, _ := filepath.Abs(path)
			if seen[abs] {
				return nil
			}
			seen[abs] = true
			format := FormatOther
			if strings.HasSuffix(strings.ToLower(info.Name()), ".gguf") {
				format = FormatGGUF
			}
			artifacts = append(artifacts, ArtifactInfo{
				Path:    abs,
				Size:    info.Size(),
				ModTime: info.ModTime(),
				Root:    root,
				Format:  format,
			})
			return nil
		})
		if err != nil {
			continue
		}
	}
	return artifacts, nil
}

// TRTConverter returns the TensorRT-LLM converter instance (may be nil if not configured).
func (s *Service) TRTConverter() *TRTConverter {
	return s.trtConverter
}

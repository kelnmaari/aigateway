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
	ContainerLogsDir      string // directory for container log files (default: logs/containers)
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
	cfg            ServiceConfig
	orch           *Orchestrator
	runtime        ContainerRuntime
	dockerRuntime  *DockerRuntime // Direct reference for discovery
	downloader     *ModelDownloader
	trtConverter   *TRTConverter
	logger         *logrus.Logger
	registry       *SpecRegistry
	providerLogger *ProviderLogger
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

	// Docker requires absolute paths for volume mounts
	if cfg.HFCacheDir != "" && !filepath.IsAbs(cfg.HFCacheDir) {
		abs, err := filepath.Abs(cfg.HFCacheDir)
		if err != nil {
			return nil, fmt.Errorf("resolve HFCacheDir path: %w", err)
		}
		cfg.HFCacheDir = abs
	}
	if cfg.GGUFCacheDir != "" && !filepath.IsAbs(cfg.GGUFCacheDir) {
		abs, err := filepath.Abs(cfg.GGUFCacheDir)
		if err != nil {
			return nil, fmt.Errorf("resolve GGUFCacheDir path: %w", err)
		}
		cfg.GGUFCacheDir = abs
	}
	if cfg.TRTEnginesDir != "" && !filepath.IsAbs(cfg.TRTEnginesDir) {
		abs, err := filepath.Abs(cfg.TRTEnginesDir)
		if err != nil {
			return nil, fmt.Errorf("resolve TRTEnginesDir path: %w", err)
		}
		cfg.TRTEnginesDir = abs
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
		LogsDir:   cfg.ContainerLogsDir,
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

	// Determine logs directory for providers
	logsDir := "logs"
	if cfg.ContainerLogsDir != "" {
		logsDir = filepath.Dir(cfg.ContainerLogsDir) // parent of containers dir
	}
	provLogger := NewProviderLogger(logsDir, cfg.Logger)

	svc := &Service{
		cfg:            cfg,
		orch:           orch,
		runtime:        runtime,
		dockerRuntime:  runtime,
		downloader:     dl,
		trtConverter:   trtConv,
		logger:         cfg.Logger,
		registry:       NewSpecRegistry(),
		providerLogger: provLogger,
	}

	// Discover and recover already running containers
	svc.recoverRunningContainers()

	return svc, nil
}

// RegisterSavedSpecs registers model specs from saved models (from ModelStore)
// This should be called after NewService to update recovered instances with full specs
func (s *Service) RegisterSavedSpecs(specs []ModelSpec) {
	updated := 0
	for _, spec := range specs {
		s.registry.Register(spec)
		// Also update any already recovered instances with full spec
		if s.orch.UpdateInstanceSpec(spec.Alias, spec) {
			updated++
			s.logger.WithFields(logrus.Fields{
				"alias":        spec.Alias,
				"provider":     spec.Provider,
				"capabilities": spec.Capabilities,
			}).Debug("Updated recovered instance with full spec")
		}
	}
	if len(specs) > 0 {
		s.logger.WithFields(logrus.Fields{
			"total":   len(specs),
			"updated": updated,
		}).Info("📚 Registered saved model specs")
	}
}

// RecoverRunningContainers discovers already running inference containers and adds them to the registry.
// This allows the server to recover state after restart without stopping running models.
// Call RegisterSavedSpecs before this to ensure full specs (with capabilities) are available.
func (s *Service) RecoverRunningContainers() {
	if s.dockerRuntime == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	discovered, err := s.dockerRuntime.DiscoverRunningContainers(ctx)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to discover running containers")
		return
	}

	if len(discovered) == 0 {
		s.logger.Debug("No running inference containers found")
		return
	}

	s.logger.WithField("count", len(discovered)).Info("🔄 Recovering running inference containers")

	for _, dc := range discovered {
		// Try to get full spec from registry (registered from saved models)
		spec, found := s.registry.Get(dc.ModelAlias)
		if !found {
			// Fallback to minimal spec if not in registry
			spec = ModelSpec{
				Alias:    dc.ModelAlias,
				Provider: dc.Provider,
			}
			s.registry.Register(spec)
			s.logger.WithField("alias", dc.ModelAlias).Debug("Using minimal spec for recovered container (not in saved models)")
		} else {
			s.logger.WithFields(logrus.Fields{
				"alias":        dc.ModelAlias,
				"capabilities": spec.Capabilities,
			}).Debug("Using full spec from saved models for recovered container")
		}

		// Add to orchestrator's running instances
		inst := &ModelInstance{
			Spec:        spec,
			ContainerID: dc.ID,
			Endpoint:    dc.Endpoint,
			Status:      StatusRunning,
			StartedAt:   dc.CreatedAt,
		}
		s.orch.AddRecoveredInstance(dc.ModelAlias, inst)

		s.logger.WithFields(logrus.Fields{
			"alias":        dc.ModelAlias,
			"provider":     spec.Provider,
			"capabilities": spec.Capabilities,
			"endpoint":     dc.Endpoint,
			"container":    dc.ID[:12],
		}).Info("✅ Recovered running model")
	}
}

// recoverRunningContainers is called internally during NewService (before ModelStore is available)
// It creates minimal specs. Full recovery happens via RecoverRunningContainers after ModelStore loads.
func (s *Service) recoverRunningContainers() {
	// Defer to public method - but at this point saved specs are not yet loaded
	// This provides basic recovery; full recovery with capabilities happens later
	s.RecoverRunningContainers()
}

// LoadAndStart prepares artifacts and starts container based on provider.
func (s *Service) LoadAndStart(ctx context.Context, spec ModelSpec) (*ModelInstance, error) {
	s.registry.Register(spec)

	// For providers that require LocalPath (llama.cpp, TRT-LLM), prepare artifacts first
	if spec.Provider == ProviderLlamaCPP || spec.Provider == ProviderTRTLLM {
		if spec.LocalPath == "" {
			s.logger.WithFields(logrus.Fields{
				"alias":    spec.Alias,
				"provider": spec.Provider,
				"hf_repo":  spec.HFRepo,
				"hf_file":  spec.HFFile,
				"gguf_url": spec.GGUFURL,
				"format":   spec.Format,
			}).Debug("Preparing artifacts for provider")
			
			// Download GGUF/TRT artifacts to get LocalPath
			prepInst, err := s.orch.PrepareModel(ctx, spec)
			if err != nil {
				return nil, fmt.Errorf("prepare artifacts: %w", err)
			}
			spec.LocalPath = prepInst.Spec.LocalPath
			s.logger.WithFields(logrus.Fields{
				"alias":      spec.Alias,
				"provider":   spec.Provider,
				"local_path": spec.LocalPath,
			}).Debug("Artifacts prepared, LocalPath set")
			
			// Double-check LocalPath was actually set
			if spec.LocalPath == "" {
				return nil, fmt.Errorf("failed to prepare artifacts: LocalPath is empty (check hf_repo, hf_file, or gguf_url)")
			}
		}
	}

	var req ContainerStartRequest
	switch spec.Provider {
	case ProviderVLLM:
		req = BuildVLLMRequest(spec, s.cfg.HFCacheDir, s.cfg.HFToken)
	case ProviderSGLang:
		req = BuildSGLangRequest(spec, s.cfg.HFCacheDir, s.cfg.HFToken)
	case ProviderTGI:
		req = BuildTGIRequest(spec, s.cfg.HFCacheDir, s.cfg.HFToken)
	case ProviderTEI:
		req = BuildTEIRequest(spec, s.cfg.HFCacheDir, s.cfg.HFToken)
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

	// Log provider launch details
	s.providerLogger.LogLaunch(spec, req)

	inst, err := s.orch.StartModel(ctx, spec, req)
	if err != nil {
		s.providerLogger.LogError(spec.Alias, string(spec.Provider), err)
		return nil, err
	}

	// Log success
	if inst.Handle != nil {
		s.providerLogger.LogSuccess(spec.Alias, string(spec.Provider), inst.Handle.ID, inst.Handle.Endpoint)
	}

	return inst, nil
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

// Forget removes model from in-memory registry (keeps artifacts on disk).
func (s *Service) Forget(alias string) {
	s.orch.ForgetModel(alias)
}

// Shutdown stops all running containers and releases downloader workers.
func (s *Service) Shutdown() {
	s.StopAll()
	s.downloader.Shutdown()
}

// StopAll stops all running model containers.
func (s *Service) StopAll() {
	models := s.orch.ListModels()
	if len(models) == 0 {
		return
	}

	s.logger.WithField("count", len(models)).Info("Stopping all model containers...")
	ctx := context.Background()

	for _, m := range models {
		if m.Status == StatusRunning || m.Status == StatusStarting {
			s.logger.WithField("alias", m.Spec.Alias).Info("Stopping model container")
			if err := s.orch.StopModel(ctx, m.Spec.Alias); err != nil {
				s.logger.WithError(err).WithField("alias", m.Spec.Alias).Warn("Failed to stop model container")
			}
		}
	}
	s.logger.Info("All model containers stopped")
}

// GetModel returns tracked model by alias.
func (s *Service) GetModel(alias string) (*ModelInstance, bool) {
	return s.orch.GetModel(alias)
}

// GetDownloader returns the model downloader instance.
func (s *Service) GetDownloader() *ModelDownloader {
	return s.downloader
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

// ClearCache removes all cached model files from HF and GGUF cache directories.
// Returns the number of bytes freed and any error.
func (s *Service) ClearCache() (int64, error) {
	var totalFreed int64
	var errors []string

	// Get list of all artifacts first
	artifacts, err := s.ListArtifacts()
	if err != nil {
		return 0, fmt.Errorf("failed to list artifacts: %w", err)
	}

	// Calculate total size
	for _, a := range artifacts {
		totalFreed += a.Size
	}

	// Clear HF cache directory
	if s.cfg.HFCacheDir != "" {
		if err := s.clearDirectory(s.cfg.HFCacheDir); err != nil {
			errors = append(errors, fmt.Sprintf("HFCacheDir: %v", err))
		}
	}

	// Clear GGUF cache directory (if different from HF)
	if s.cfg.GGUFCacheDir != "" && s.cfg.GGUFCacheDir != s.cfg.HFCacheDir {
		if err := s.clearDirectory(s.cfg.GGUFCacheDir); err != nil {
			errors = append(errors, fmt.Sprintf("GGUFCacheDir: %v", err))
		}
	}

	s.logger.WithFields(logrus.Fields{
		"freed_bytes": totalFreed,
		"files_count": len(artifacts),
	}).Info("Cache cleared")

	if len(errors) > 0 {
		return totalFreed, fmt.Errorf("partial clear: %s", strings.Join(errors, "; "))
	}
	return totalFreed, nil
}

// clearDirectory removes all contents of a directory but keeps the directory itself.
func (s *Service) clearDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist, nothing to clear
		}
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			s.logger.WithError(err).WithField("path", path).Warn("Failed to remove cache entry")
		}
	}
	return nil
}

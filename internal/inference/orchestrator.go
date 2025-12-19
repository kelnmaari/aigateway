package inference

import (
	"context"
	"fmt"
	"sync"
	"time"

	"aigateway/internal/metrics"
	"github.com/sirupsen/logrus"
)

// Orchestrator coordinates model preparation (download) and container lifecycle.
type Orchestrator struct {
	runtime            ContainerRuntime
	downloader         *ModelDownloader
	logger             *logrus.Logger
	healthCheckTimeout time.Duration
	startupTimeout     time.Duration

	mu      sync.RWMutex
	models  map[string]*ModelInstance // alias -> instance
}

// ModelInstance tracks current state of a model/container.
type ModelInstance struct {
	Spec     ModelSpec
	Status   ModelStatus
	Handle   *ContainerHandle
	LocalDir string
	Error    string
	LastUsed time.Time
}

// OrchestratorConfig holds orchestrator parameters.
type OrchestratorConfig struct {
	HealthCheckTimeout time.Duration
	StartupTimeout     time.Duration
}

// NewOrchestrator creates an orchestrator with given runtime and downloader.
func NewOrchestrator(runtime ContainerRuntime, downloader *ModelDownloader, logger *logrus.Logger, cfg OrchestratorConfig) *Orchestrator {
	hcTimeout := cfg.HealthCheckTimeout
	if hcTimeout <= 0 {
		hcTimeout = 60 * time.Second
	}
	startTimeout := cfg.StartupTimeout
	if startTimeout <= 0 {
		startTimeout = 5 * time.Minute
	}
	return &Orchestrator{
		runtime:            runtime,
		downloader:         downloader,
		logger:             logger,
		healthCheckTimeout: hcTimeout,
		startupTimeout:     startTimeout,
		models:             make(map[string]*ModelInstance),
	}
}

// PrepareModel ensures artifacts are present locally according to spec.
// For HF format without specific file, providers (vLLM, SGLang, TGI) download via HF Hub themselves.
// If model is already running with same basic config, return it.
// If user explicitly loads with different params, replace the model.
func (o *Orchestrator) PrepareModel(ctx context.Context, spec ModelSpec) (*ModelInstance, error) {
	o.mu.Lock()
	if existing, ok := o.models[spec.Alias]; ok {
		// If model is currently running/starting, return it as-is
		// This allows EnsureByAlias to work for chat completions
		if existing.Status == StatusRunning || existing.Status == StatusStarting {
			o.mu.Unlock()
			return existing, nil
		}
		// Model exists but not running - check if user wants different parameters
		// Only replace if there are meaningful parameter changes
		if existing.Spec.Provider != spec.Provider ||
			existing.Spec.VLLMGPUUtilization != spec.VLLMGPUUtilization ||
			existing.Spec.VLLMMaxModelLen != spec.VLLMMaxModelLen ||
			existing.Spec.VLLMTensorParallel != spec.VLLMTensorParallel {
			o.logger.WithFields(logrus.Fields{
				"alias":                    spec.Alias,
				"old_gpu_util":             existing.Spec.VLLMGPUUtilization,
				"new_gpu_util":             spec.VLLMGPUUtilization,
				"old_max_model_len":        existing.Spec.VLLMMaxModelLen,
				"new_max_model_len":        spec.VLLMMaxModelLen,
			}).Info("Replacing model spec with new parameters")
			delete(o.models, spec.Alias)
		} else {
			// Same params - return existing (allows restart of stopped model)
			o.mu.Unlock()
			return existing, nil
		}
	}
	o.mu.Unlock()

	inst := &ModelInstance{
		Spec:   spec,
		Status: StatusPending,
	}

	var localPath string
	var err error

	// Skip download if LocalPath already set (pre-cached or manually specified)
	if spec.LocalPath != "" {
		localPath = spec.LocalPath
	} else {
		switch spec.Format {
		case FormatHF:
			// For HF format: if HFFile is specified, download specific file.
			// Otherwise, providers (vLLM, SGLang, TGI) will download via HF Hub themselves
			// by mounting the HF cache directory and using --model HFRepo argument.
			if spec.HFFile != "" {
				localPath, err = o.downloader.EnsureHFFile(ctx, spec.HFRepo, spec.HFFile, spec.ExpectedSHA)
			} else if spec.HFRepo == "" {
				err = fmt.Errorf("hf format requires either hf_repo or local_path")
			}
			// else: localPath stays empty, provider will download via HFRepo
		case FormatGGUF:
			if spec.GGUFURL != "" {
				localPath, err = o.downloader.EnsureGGUF(ctx, spec.GGUFURL, spec.ExpectedSHA)
			} else if spec.HFRepo != "" && spec.HFFile != "" {
				// GGUF from HF repo
				localPath, err = o.downloader.EnsureHFFile(ctx, spec.HFRepo, spec.HFFile, spec.ExpectedSHA)
			} else {
				err = fmt.Errorf("gguf format requires gguf_url or (hf_repo + hf_file)")
			}
		case FormatTRT:
			if spec.HFRepo != "" && spec.HFFile != "" {
				localPath, err = o.downloader.EnsureHFFile(ctx, spec.HFRepo, spec.HFFile, spec.ExpectedSHA)
			} else if spec.LocalPath == "" {
				err = fmt.Errorf("trt format requires local_path or (hf_repo + hf_file)")
			}
		default:
			err = fmt.Errorf("unsupported format: %s", spec.Format)
		}

		if err != nil {
			inst.Status = StatusFailed
			inst.Error = err.Error()
			o.saveInstance(inst)
			return inst, err
		}
	}

	inst.Spec.LocalPath = localPath
	inst.Status = StatusReady
	inst.LastUsed = time.Now()
	o.saveInstance(inst)
	return inst, nil
}

// StartModel starts provider container (if runtime provided) after ensuring artifacts.
// Uses background context to prevent cancellation from HTTP request refresh.
// Applies startupTimeout for the overall operation.
func (o *Orchestrator) StartModel(ctx context.Context, spec ModelSpec, startReq ContainerStartRequest) (*ModelInstance, error) {
	// Use background context with startup timeout to prevent cancellation from HTTP request
	// This allows model loading to continue even if user refreshes the page
	var opCtx context.Context
	var cancel context.CancelFunc
	if o.startupTimeout > 0 {
		opCtx, cancel = context.WithTimeout(context.Background(), o.startupTimeout)
	} else {
		opCtx, cancel = context.WithTimeout(context.Background(), 60*time.Minute)
	}
	defer cancel()
	_ = ctx // original HTTP context ignored to prevent cancellation on page refresh

	inst, err := o.PrepareModel(opCtx, spec)
	if err != nil {
		return inst, err
	}

	// If already running, return existing handle.
	if inst.Handle != nil && inst.Status == StatusRunning {
		return inst, nil
	}

	if o.runtime == nil {
		return inst, fmt.Errorf("container runtime is not configured")
	}

	inst.Status = StatusStarting
	o.saveInstance(inst)

	handle, err := o.runtime.Start(opCtx, startReq)
	if err != nil {
		inst.Status = StatusFailed
		inst.Error = err.Error()
		o.saveInstance(inst)
		metrics.InferenceStartupFailures.WithLabelValues(string(spec.Provider)).Inc()
		o.logger.WithFields(logrus.Fields{
			"event":    "container_start_failed",
			"alias":    spec.Alias,
			"provider": spec.Provider,
			"error":    err.Error(),
		}).Error("failed to start inference container")
		return inst, err
	}

	inst.Handle = handle
	inst.Status = StatusStarting
	o.saveInstance(inst)

	// Health wait with configurable timeout
	// Use background context to prevent cancellation from HTTP request refresh
	if handle.Endpoint != "" {
		healthCtx, cancel := context.WithTimeout(context.Background(), o.healthCheckTimeout)
		defer cancel()
		healthURL := providerHealthURL(handle.Provider, handle.Endpoint)
		if err := waitForHealth(healthCtx, healthURL, 2*time.Second); err != nil {
			inst.Status = StatusFailed
			inst.Error = fmt.Sprintf("health check failed: %v", err)
			_ = o.runtime.Stop(context.Background(), handle.ID)
			o.saveInstance(inst)
			metrics.InferenceHealthFailures.WithLabelValues(string(spec.Provider)).Inc()
			// Alert: health check timeout/failure
			o.logger.WithFields(logrus.Fields{
				"event":    "health_check_failed",
				"alias":    spec.Alias,
				"provider": spec.Provider,
				"endpoint": handle.Endpoint,
				"timeout":  o.healthCheckTimeout,
				"error":    err.Error(),
			}).Warn("inference container health check failed, container stopped")
			return inst, err
		}
	}

	inst.Status = StatusRunning
	inst.LastUsed = time.Now()
	o.saveInstance(inst)
	metrics.InferenceContainersStarted.WithLabelValues(string(spec.Provider)).Inc()
	o.logger.WithFields(logrus.Fields{
		"event":    "container_started",
		"alias":    spec.Alias,
		"provider": spec.Provider,
		"endpoint": handle.Endpoint,
	}).Info("inference container started successfully")
	return inst, nil
}

// StopModel stops container if running.
func (o *Orchestrator) StopModel(ctx context.Context, alias string) error {
	o.mu.Lock()
	inst, ok := o.models[alias]
	o.mu.Unlock()
	if !ok {
		return fmt.Errorf("model not found: %s", alias)
	}
	if inst.Handle == nil || o.runtime == nil {
		return nil
	}

	if err := o.runtime.Stop(ctx, inst.Handle.ID); err != nil {
		return err
	}

	inst.Status = StatusReady
	inst.Handle = nil
	o.saveInstance(inst)
	return nil
}

// ForgetModel removes model from in-memory registry (after stop).
// Used by Evict to completely remove model from tracking.
func (o *Orchestrator) ForgetModel(alias string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.models, alias)
}

// saveInstance saves/updates instance in registry.
func (o *Orchestrator) saveInstance(inst *ModelInstance) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.models[inst.Spec.Alias] = inst
}

// GetModel returns model instance by alias.
func (o *Orchestrator) GetModel(alias string) (*ModelInstance, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	inst, ok := o.models[alias]
	return inst, ok
}

// ListModels returns all tracked instances.
func (o *Orchestrator) ListModels() []*ModelInstance {
	o.mu.RLock()
	defer o.mu.RUnlock()
	result := make([]*ModelInstance, 0, len(o.models))
	for _, inst := range o.models {
		result = append(result, inst)
	}
	return result
}

// waitForHealth polls health endpoint until success or timeout.
func waitForHealth(ctx context.Context, url string, interval time.Duration) error {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		if err := HealthCheckHTTP(ctx, url); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
}


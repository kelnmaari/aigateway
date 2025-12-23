package inference

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// Router resolves specs by alias or capability and ensures model is prepared/launched.
type Router struct {
	mgr *Manager
}

// NewRouter creates a Router for a Manager.
func NewRouter(mgr *Manager) *Router {
	return &Router{mgr: mgr}
}

// EnsureBySpec registers and launches a spec directly.
func (r *Router) EnsureBySpec(ctx context.Context, spec ModelSpec) (*ModelInstance, error) {
	r.mgr.svc.RegisterSpec(spec)
	inst, err := r.mgr.LoadAndStart(ctx, spec)
	if err == nil {
		r.mgr.touch(spec.Alias)
	}
	return inst, err
}

// EnsureByAlias resolves alias and ensures container is running.
func (r *Router) EnsureByAlias(ctx context.Context, alias string) (*ModelInstance, error) {
	spec, err := r.mgr.ResolveByAlias(alias)
	if err != nil {
		return nil, err
	}
	inst, err := r.mgr.LoadAndStart(ctx, spec)
	if err == nil {
		r.mgr.touch(alias)
	}
	return inst, err
}

// EnsureByCapability picks first spec with capability and ensures it runs.
func (r *Router) EnsureByCapability(ctx context.Context, cap Capability) (*ModelInstance, error) {
	spec, err := r.mgr.ResolveByCapability(cap)
	if err != nil {
		return nil, err
	}
	inst, err := r.mgr.LoadAndStart(ctx, spec)
	if err == nil {
		r.mgr.touch(spec.Alias)
	}
	return inst, err
}

// PrepareBySpec registers and downloads artifacts without starting container.
func (r *Router) PrepareBySpec(ctx context.Context, spec ModelSpec) (*ModelInstance, error) {
	r.mgr.svc.RegisterSpec(spec)
	inst, err := r.mgr.PrepareOnly(ctx, spec)
	if err == nil {
		r.mgr.touch(spec.Alias)
	}
	return inst, err
}

// PrepareByAlias downloads artifacts without starting container.
func (r *Router) PrepareByAlias(ctx context.Context, alias string) (*ModelInstance, error) {
	spec, err := r.mgr.ResolveByAlias(alias)
	if err != nil {
		return nil, err
	}
	inst, err := r.mgr.PrepareOnly(ctx, spec)
	if err == nil {
		r.mgr.touch(alias)
	}
	return inst, err
}

// Stop stops running container by alias.
func (r *Router) Stop(ctx context.Context, alias string) error {
	return r.mgr.Stop(ctx, alias)
}

// Health performs health check for running model by alias.
func (r *Router) Health(ctx context.Context, alias string) error {
	if err := r.mgr.Health(ctx, alias); err != nil {
		return err
	}
	r.mgr.touch(alias)
	return nil
}

// RegisterSpec registers a model spec manually.
func (r *Router) RegisterSpec(spec ModelSpec) {
	r.mgr.svc.RegisterSpec(spec)
}

// ResolveOrError tries alias first, then capability if alias missing and cap provided.
func (r *Router) ResolveOrError(ctx context.Context, alias string, cap *Capability) (*ModelInstance, error) {
	if alias != "" {
		return r.EnsureByAlias(ctx, alias)
	}
	if cap != nil {
		return r.EnsureByCapability(ctx, *cap)
	}
	return nil, fmt.Errorf("no alias or capability provided for resolution")
}

// ListModels exposes underlying service list.
func (r *Router) ListModels() []*ModelInstance {
	return r.mgr.svc.ListModels()
}

// GetModel returns a specific model instance by alias and whether it's running.
// Used for dynamic URL resolution (e.g., embedding model endpoints).
func (r *Router) GetModel(alias string) (endpoint string, running bool) {
	models := r.mgr.svc.ListModels()
	logger := r.mgr.svc.logger

	logger.WithFields(logrus.Fields{
		"requested_alias": alias,
		"total_models":    len(models),
	}).Debug("GetModel: searching for model by alias")

	for _, m := range models {
		// Get endpoint from Handle (primary source) or fallback to direct Endpoint field
		modelEndpoint := ""
		if m.Handle != nil && m.Handle.Endpoint != "" {
			modelEndpoint = m.Handle.Endpoint
		} else if m.Endpoint != "" {
			modelEndpoint = m.Endpoint
		}

		logger.WithFields(logrus.Fields{
			"model_alias":    m.Spec.Alias,
			"model_status":   m.Status,
			"model_endpoint": modelEndpoint,
		}).Debug("GetModel: checking model")

		if m.Spec.Alias == alias {
			if m.Status == StatusRunning && modelEndpoint != "" {
				logger.WithFields(logrus.Fields{
					"alias":    alias,
					"endpoint": modelEndpoint,
				}).Debug("GetModel: found running model")
				return modelEndpoint, true
			}
			logger.WithFields(logrus.Fields{
				"alias":    alias,
				"status":   m.Status,
				"endpoint": modelEndpoint,
			}).Debug("GetModel: model found but not running or no endpoint")
			return "", false
		}
	}

	logger.WithField("alias", alias).Debug("GetModel: model not found")
	return "", false
}

// GetRunningEmbeddingModel finds a running embedding model automatically.
// It looks for models with:
// 1. Provider = TEI (Text Embeddings Inference)
// 2. Or capability "embeddings"
// Returns the first running embedding model found.
func (r *Router) GetRunningEmbeddingModel() (alias string, endpoint string, found bool) {
	models := r.mgr.svc.ListModels()
	logger := r.mgr.svc.logger

	logger.WithField("model_count", len(models)).Debug("Looking for embedding model")

	// Helper to get endpoint from model (Handle.Endpoint or Endpoint field)
	getEndpoint := func(m *ModelInstance) string {
		if m.Handle != nil && m.Handle.Endpoint != "" {
			return m.Handle.Endpoint
		}
		return m.Endpoint
	}

	// First pass: look for TEI provider (dedicated embedding inference)
	for _, m := range models {
		modelEndpoint := getEndpoint(m)
		logger.WithFields(logrus.Fields{
			"alias":        m.Spec.Alias,
			"provider":     m.Spec.Provider,
			"status":       m.Status,
			"endpoint":     modelEndpoint,
			"capabilities": m.Spec.Capabilities,
		}).Debug("Checking model for embedding capability")

		if m.Status == StatusRunning && modelEndpoint != "" && m.Spec.Provider == ProviderTEI {
			logger.WithFields(logrus.Fields{
				"alias":    m.Spec.Alias,
				"endpoint": modelEndpoint,
			}).Debug("Found TEI embedding model")
			return m.Spec.Alias, modelEndpoint, true
		}
	}

	// Second pass: look for models with "embeddings" capability
	for _, m := range models {
		modelEndpoint := getEndpoint(m)
		if m.Status != StatusRunning || modelEndpoint == "" {
			continue
		}
		for _, cap := range m.Spec.Capabilities {
			if cap == "embeddings" {
				logger.WithFields(logrus.Fields{
					"alias":    m.Spec.Alias,
					"endpoint": modelEndpoint,
				}).Debug("Found model with embeddings capability")
				return m.Spec.Alias, modelEndpoint, true
			}
		}
	}

	logger.Debug("No running embedding model found")
	return "", "", false
}

// Evict stops container and removes model from list (keeps artifacts on disk).
func (r *Router) Evict(ctx context.Context, alias string) error {
	return r.mgr.Evict(ctx, alias)
}

// Pin marks model as pinned (skip auto-stop/evict).
func (r *Router) Pin(alias string) {
	r.mgr.Pin(alias)
}

// Unpin removes pin mark.
func (r *Router) Unpin(alias string) {
	r.mgr.Unpin(alias)
}

// IsPinned returns pin status.
func (r *Router) IsPinned(alias string) bool {
	return r.mgr.IsPinned(alias)
}

// DeleteArtifacts removes local files for alias (after stop).
func (r *Router) DeleteArtifacts(alias string) error {
	inst, ok := r.mgr.svc.GetModel(alias)
	if !ok {
		return fmt.Errorf("model not found: %s", alias)
	}
	if inst.Handle != nil {
		return fmt.Errorf("model is running, stop or evict first")
	}
	if inst.Spec.LocalPath == "" {
		return fmt.Errorf("no local path for model: %s", alias)
	}
	if err := ensurePathWithin(inst.Spec.LocalPath, []string{r.mgr.svc.cfg.HFCacheDir, r.mgr.svc.cfg.GGUFCacheDir}); err != nil {
		return err
	}
	if err := os.Remove(inst.Spec.LocalPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove failed: %w", err)
	}
	// Attempt to remove parent dir if empty
	_ = os.Remove(filepath.Dir(inst.Spec.LocalPath))
	return nil
}

// ListArtifacts lists cached files in HF/GGUF roots.
func (r *Router) ListArtifacts() ([]ArtifactInfo, error) {
	return r.mgr.svc.ListArtifacts()
}

// ClearCache removes all cached model files from HF and GGUF cache directories.
// Returns the number of bytes freed and any error.
func (r *Router) ClearCache() (int64, error) {
	return r.mgr.svc.ClearCache()
}

// EvictCacheSize evicts cache to fit under limitBytes (0 = no limit).
func (r *Router) EvictCacheSize(limitBytes int64) error {
	if limitBytes <= 0 {
		return fmt.Errorf("limitBytes must be >0")
	}
	return r.mgr.EvictCacheSize(limitBytes)
}

// ContainerLogs returns recent stdout/stderr from running container.
func (r *Router) ContainerLogs(ctx context.Context, alias string, tailLines int) (string, error) {
	return r.mgr.svc.ContainerLogs(ctx, alias, tailLines)
}

// ContainerMetrics fetches provider metrics (Prometheus format).
func (r *Router) ContainerMetrics(ctx context.Context, alias string) (string, error) {
	return r.mgr.svc.ContainerMetrics(ctx, alias)
}

// TRTConverter returns the TensorRT-LLM converter instance (may be nil if not configured).
func (r *Router) TRTConverter() *TRTConverter {
	return r.mgr.TRTConverter()
}

// GetRuntime returns the underlying DockerRuntime for image management.
// Returns nil if runtime is not DockerRuntime.
func (r *Router) GetRuntime() *DockerRuntime {
	if r.mgr == nil || r.mgr.svc == nil {
		return nil
	}
	dr, ok := r.mgr.svc.runtime.(*DockerRuntime)
	if !ok {
		return nil
	}
	return dr
}

// GetDownloader returns the model downloader for repository downloads.
func (r *Router) GetDownloader() *ModelDownloader {
	if r.mgr == nil || r.mgr.svc == nil {
		return nil
	}
	return r.mgr.svc.downloader
}


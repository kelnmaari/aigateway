package inference

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/metrics"
)

// Manager provides high-level operations over Service with simple resolution by alias/capability.
type Manager struct {
	svc        *Service
	statusMu   sync.RWMutex
	statusMap  map[string]ModelStatus // alias -> status
	lastUsed   sync.RWMutex
	pinsMu     sync.RWMutex
	pins       map[string]bool
	maxRunning int
	cacheMax   int64
}

// NewManager creates a Manager on top of Service.
func NewManager(svc *Service) *Manager {
	m := &Manager{
		svc:        svc,
		statusMap:  make(map[string]ModelStatus),
		pins:       make(map[string]bool),
		maxRunning: svc.cfg.MaxRunningModels,
		cacheMax:   svc.cfg.CacheMaxBytes,
	}
	if svc.cfg.CacheMaxBytes >= 0 {
		metrics.InferenceCacheLimitBytes.Set(float64(svc.cfg.CacheMaxBytes))
	}
	return m
}

// LoadAndStart registers spec, downloads artifacts, starts container.
func (m *Manager) LoadAndStart(ctx context.Context, spec ModelSpec) (*ModelInstance, error) {
	inst, err := m.svc.LoadAndStart(ctx, spec)
	if err != nil {
		m.setStatus(spec.Alias, StatusFailed)
		return inst, err
	}
	m.setStatus(spec.Alias, inst.Status)
	m.touch(spec.Alias)
	m.enforceMaxRunning(spec.Alias)
	m.enforceCacheLimit()
	metrics.ModelsLoaded.Set(float64(len(m.svc.ListModels())))
	return inst, nil
}

// PrepareOnly registers spec and downloads artifacts without starting container.
func (m *Manager) PrepareOnly(ctx context.Context, spec ModelSpec) (*ModelInstance, error) {
	inst, err := m.svc.PrepareOnly(ctx, spec)
	if err != nil {
		m.setStatus(spec.Alias, StatusFailed)
		return inst, err
	}
	m.setStatus(spec.Alias, inst.Status)
	m.touch(spec.Alias)
	m.enforceCacheLimit()
	metrics.ModelsLoaded.Set(float64(len(m.svc.ListModels())))
	return inst, nil
}

// Health checks running model.
func (m *Manager) Health(ctx context.Context, alias string) error {
	return m.svc.Health(ctx, alias)
}

// Stop stops running container (keeps model in list as Ready).
func (m *Manager) Stop(ctx context.Context, alias string) error {
	if m.isPinned(alias) {
		return fmt.Errorf("model is pinned: %s", alias)
	}
	if err := m.svc.Stop(ctx, alias); err != nil {
		return err
	}
	m.setStatus(alias, StatusReady)
	m.touch(alias)
	metrics.ModelsLoaded.Set(float64(len(m.svc.ListModels())))
	return nil
}

// Evict stops container and removes model from in-memory list (artifacts stay on disk).
func (m *Manager) Evict(ctx context.Context, alias string) error {
	if m.isPinned(alias) {
		return fmt.Errorf("model is pinned: %s", alias)
	}
	// 1. Cancel any in-flight StartModel — unblocks health check loops and
	//    releases the alias lock so subsequent Load requests don't deadlock.
	m.svc.orch.CancelStart(alias)

	// 2. Stop the container that the orchestrator has a handle for (best effort).
	if err := m.svc.Stop(ctx, alias); err != nil {
		m.svc.logger.WithFields(logrus.Fields{
			"alias": alias,
			"error": err.Error(),
		}).Debug("Evict: Stop returned error (ignored)")
	}

	// 3. Kill ALL containers with the alias prefix to catch orphans that were
	//    started but whose handle was not yet stored (StatusStarting race).
	if m.svc.orch.runtime != nil {
		if err := m.svc.orch.runtime.StopByAlias(ctx, alias); err != nil {
			m.svc.logger.WithFields(logrus.Fields{
				"alias": alias,
				"error": err.Error(),
			}).Warn("Evict: StopByAlias returned error")
		}
	}

	// 4. Remove from in-memory registry
	m.svc.Forget(alias)
	// Remove from status tracking
	m.statusMu.Lock()
	delete(m.statusMap, alias)
	m.statusMu.Unlock()
	// Remove from pins if any
	m.pinsMu.Lock()
	delete(m.pins, alias)
	m.pinsMu.Unlock()
	metrics.ModelsLoaded.Set(float64(len(m.svc.ListModels())))
	return nil
}

// ResolveByAlias returns registered spec or error.
func (m *Manager) ResolveByAlias(alias string) (ModelSpec, error) {
	if spec, ok := m.svc.ResolveSpec(alias); ok {
		return spec, nil
	}
	return ModelSpec{}, fmt.Errorf("model not found: %s", alias)
}

// ResolveByCapability returns first spec with capability.
func (m *Manager) ResolveByCapability(cap Capability) (ModelSpec, error) {
	specs := m.svc.registry.FindByCapability(cap)
	if len(specs) == 0 {
		return ModelSpec{}, fmt.Errorf("no model with capability: %s", cap)
	}
	return specs[0], nil
}

// Status returns tracked status for alias.
func (m *Manager) Status(alias string) (ModelStatus, bool) {
	m.statusMu.RLock()
	defer m.statusMu.RUnlock()
	s, ok := m.statusMap[alias]
	return s, ok
}

func (m *Manager) setStatus(alias string, st ModelStatus) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	m.statusMap[alias] = st
}

// touch updates last used timestamp for alias if model is tracked.
func (m *Manager) touch(alias string) {
	inst, ok := m.svc.GetModel(alias)
	if !ok {
		return
	}
	inst.LastUsed = time.Now()
	m.svc.orch.saveInstance(inst)
}

// enforceMaxRunning stops least recently used running models if limit exceeded.
func (m *Manager) enforceMaxRunning(currentAlias string) {
	limit := m.maxRunning
	if limit <= 0 {
		return
	}
	models := m.svc.ListModels()
	var running []*ModelInstance
	for _, inst := range models {
		if inst.Status == StatusRunning {
			running = append(running, inst)
		}
	}
	if len(running) <= limit {
		return
	}
	// Sort by LastUsed ascending (oldest first)
	sort.Slice(running, func(i, j int) bool {
		return running[i].LastUsed.Before(running[j].LastUsed)
	})
	// Stop oldest until within limit, skipping current if possible
	for len(running) > limit {
		inst := running[0]
		running = running[1:]
		if m.isPinned(inst.Spec.Alias) {
			continue
		}
		if inst.Spec.Alias == currentAlias && len(running) >= 1 {
			// skip current; pick next
			inst = running[0]
			running = running[1:]
		}
		_ = m.Stop(context.Background(), inst.Spec.Alias)
	}
	metrics.ModelsLoaded.Set(float64(len(m.svc.ListModels())))
}

// enforceCacheLimit trims cache if limit configured.
func (m *Manager) enforceCacheLimit() {
	if m.cacheMax <= 0 {
		return
	}
	_ = m.EvictCacheSize(m.cacheMax)
}

func (m *Manager) refreshCacheMetrics() {
	artifacts, err := m.svc.ListArtifacts()
	if err != nil {
		return
	}
	var total int64
	for _, a := range artifacts {
		total += a.Size
	}
	metrics.InferenceCacheBytes.Set(float64(total))
	if m.cacheMax > 0 {
		m.warnCacheNearLimit(total, m.cacheMax)
	}
}

func (m *Manager) warnCacheNearLimit(total, limit int64) {
	if limit <= 0 || m.svc == nil || m.svc.logger == nil {
		return
	}
	const threshold = 0.9
	if float64(total) >= float64(limit)*threshold {
		m.svc.logger.WithFields(logrus.Fields{
			"event":       "cache_near_limit",
			"total_bytes": total,
			"limit_bytes": limit,
		}).Warn("inference cache usage near limit")
	}
}

// Pin marks alias as pinned (skip auto-stop).
func (m *Manager) Pin(alias string) {
	m.pinsMu.Lock()
	defer m.pinsMu.Unlock()
	m.pins[alias] = true
}

// Unpin removes pin mark.
func (m *Manager) Unpin(alias string) {
	m.pinsMu.Lock()
	defer m.pinsMu.Unlock()
	delete(m.pins, alias)
}

func (m *Manager) isPinned(alias string) bool {
	m.pinsMu.RLock()
	defer m.pinsMu.RUnlock()
	return m.pins[alias]
}

// IsPinned exposes pin status.
func (m *Manager) IsPinned(alias string) bool {
	return m.isPinned(alias)
}

// StartIdleReaper stops running models that were idle longer than idleAfter.
func (m *Manager) StartIdleReaper(ctx context.Context, idleAfter, checkEvery time.Duration) {
	if idleAfter <= 0 || checkEvery <= 0 {
		return
	}
	go func() {
		t := time.NewTicker(checkEvery)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				now := time.Now()
				models := m.svc.ListModels()
				for _, inst := range models {
					if inst.Status != StatusRunning || inst.LastUsed.IsZero() {
						continue
					}
					if m.isPinned(inst.Spec.Alias) {
						continue
					}
					if now.Sub(inst.LastUsed) >= idleAfter {
						_ = m.Stop(context.Background(), inst.Spec.Alias)
					}
				}
			}
		}
	}()
}

// StartCacheGaugeUpdater periodically updates cache size gauge and warns near limit.
func (m *Manager) StartCacheGaugeUpdater(ctx context.Context, every time.Duration) {
	if every <= 0 {
		return
	}
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				m.refreshCacheMetrics()
			}
		}
	}()
}

// EvictCacheSize removes cached artifacts until total size <= limitBytes.
// Skips pinned and running models' artifacts. Evicts by oldest mod time.
func (m *Manager) EvictCacheSize(limitBytes int64) error {
	if limitBytes <= 0 {
		return fmt.Errorf("limitBytes must be >0")
	}
	artifacts, err := m.svc.ListArtifacts()
	if err != nil {
		return err
	}
	var total int64
	for _, a := range artifacts {
		total += a.Size
	}
	metrics.InferenceCacheBytes.Set(float64(total))
	m.warnCacheNearLimit(total, limitBytes)
	if total <= limitBytes {
		return nil
	}
	// Build pinned set: pinned aliases + running instances.
	pinnedPaths := make(map[string]bool)
	models := m.svc.ListModels()
	for _, inst := range models {
		if inst == nil || inst.Spec.LocalPath == "" {
			continue
		}
		if m.isPinned(inst.Spec.Alias) {
			pinnedPaths[inst.Spec.LocalPath] = true
		}
		if inst.Handle != nil {
			pinnedPaths[inst.Spec.LocalPath] = true
		}
	}
	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].ModTime.Before(artifacts[j].ModTime)
	})
	totalBefore := total
	var removedBytes int64
	var removedFiles int
	for _, a := range artifacts {
		if pinnedPaths[a.Path] {
			continue
		}
		if err := os.Remove(a.Path); err != nil && !os.IsNotExist(err) {
			continue
		}
		total -= a.Size
		removedBytes += a.Size
		removedFiles++
		if total <= limitBytes {
			break
		}
	}
	metrics.InferenceCacheEvictedBytes.Add(float64(removedBytes))
	metrics.InferenceCacheEvictedFiles.Add(float64(removedFiles))
	metrics.InferenceCacheBytes.Set(float64(total))
	if removedFiles > 0 && m.svc != nil && m.svc.logger != nil {
		m.svc.logger.WithFields(logrus.Fields{
			"event":          "cache_evicted",
			"removed_files":  removedFiles,
			"removed_bytes":  removedBytes,
			"limit_bytes":    limitBytes,
			"total_before":   totalBefore,
			"total_after":    total,
			"pinned_skipped": len(pinnedPaths),
		}).Info("inference cache eviction completed")
	}
	return nil
}

// TRTConverter returns the TensorRT-LLM converter instance (may be nil).
func (m *Manager) TRTConverter() *TRTConverter {
	return m.svc.TRTConverter()
}

// OnnxExporter returns the ONNX export manager.
func (m *Manager) OnnxExporter() *OnnxExporter {
	return m.svc.OnnxExporter()
}

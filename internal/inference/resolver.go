package inference

import (
	"fmt"
	"sync"
)

// SpecRegistry stores model specs and allows resolution by alias or capability.
type SpecRegistry struct {
	mu    sync.RWMutex
	specs map[string]ModelSpec // alias -> spec
}

// NewSpecRegistry creates an empty registry.
func NewSpecRegistry() *SpecRegistry {
	return &SpecRegistry{
		specs: make(map[string]ModelSpec),
	}
}

// Register adds or replaces a spec by alias.
func (r *SpecRegistry) Register(spec ModelSpec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.specs[spec.Alias] = spec
}

// Get returns a spec by alias.
func (r *SpecRegistry) Get(alias string) (ModelSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	spec, ok := r.specs[alias]
	return spec, ok
}

// FindByCapability returns specs that have requested capability (chat/embeddings/vision).
func (r *SpecRegistry) FindByCapability(cap Capability) []ModelSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []ModelSpec
	for _, spec := range r.specs {
		for _, c := range spec.Capabilities {
			if c == cap {
				out = append(out, spec)
				break
			}
		}
	}
	return out
}

// Require resolves by alias or returns error.
func (r *SpecRegistry) Require(alias string) (ModelSpec, error) {
	if spec, ok := r.Get(alias); ok {
		return spec, nil
	}
	return ModelSpec{}, fmt.Errorf("model alias not registered: %s", alias)
}


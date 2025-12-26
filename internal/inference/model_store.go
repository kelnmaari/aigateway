package inference

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// SavedModel represents a model configuration that can be restored after restart.
type SavedModel struct {
	Alias        string         `json:"alias"`
	Provider     ProviderKind   `json:"provider"`
	Format       ModelFormat    `json:"format"`
	Capabilities []Capability   `json:"capabilities,omitempty"`
	HFRepo       string         `json:"hf_repo,omitempty"`
	HFFile       string         `json:"hf_file,omitempty"`
	HFRevision   string         `json:"hf_revision,omitempty"`
	GGUFURL      string         `json:"gguf_url,omitempty"`
	GPUDevice    string         `json:"gpu_device,omitempty"`

	// vLLM options
	VLLMTensorParallel int     `json:"vllm_tensor_parallel,omitempty"`
	VLLMMaxModelLen    int     `json:"vllm_max_model_len,omitempty"`
	VLLMGPUUtilization float64 `json:"vllm_gpu_utilization,omitempty"`

	// llama.cpp options
	LlamaMainGPU     int    `json:"llama_main_gpu,omitempty"`
	LlamaTensorSplit string `json:"llama_tensor_split,omitempty"`
	LlamaNGPULayers  int    `json:"llama_n_gpu_layers,omitempty"`
	LlamaCtxSize     int    `json:"llama_ctx_size,omitempty"`

	// SGLang options
	SGLangTensorParallel int     `json:"sglang_tensor_parallel,omitempty"`
	SGLangMemFraction    float64 `json:"sglang_mem_fraction,omitempty"`

	// TGI options
	TGINumShard int `json:"tgi_num_shard,omitempty"`

	// Meta
	AutoStart bool      `json:"auto_start"` // Start on server boot
	SavedAt   time.Time `json:"saved_at"`
}

// ModelStore persists model configurations to disk.
type ModelStore struct {
	mu       sync.RWMutex
	path     string
	models   map[string]SavedModel
	logger   *logrus.Logger
}

// NewModelStore creates a new model store.
func NewModelStore(dataDir string, logger *logrus.Logger) (*ModelStore, error) {
	path := filepath.Join(dataDir, "inference_models.json")
	
	store := &ModelStore{
		path:   path,
		models: make(map[string]SavedModel),
		logger: logger,
	}

	// Load existing data
	if err := store.load(); err != nil {
		// File might not exist yet, that's ok
		if !os.IsNotExist(err) {
			logger.WithError(err).Warn("Failed to load saved models")
		}
	}

	return store, nil
}

// Save persists a model configuration.
func (s *ModelStore) Save(model SavedModel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	model.SavedAt = time.Now()
	s.models[model.Alias] = model

	return s.persist()
}

// SaveFromSpec converts ModelSpec to SavedModel and persists it.
func (s *ModelStore) SaveFromSpec(spec ModelSpec, autoStart bool) error {
	saved := SavedModel{
		Alias:              spec.Alias,
		Provider:           spec.Provider,
		Format:             spec.Format,
		Capabilities:       spec.Capabilities,
		HFRepo:             spec.HFRepo,
		HFFile:             spec.HFFile,
		HFRevision:         spec.HFRevision,
		GGUFURL:            spec.GGUFURL,
		GPUDevice:          spec.GPUDevice,
		VLLMTensorParallel: spec.VLLMTensorParallel,
		VLLMMaxModelLen:    spec.VLLMMaxModelLen,
		VLLMGPUUtilization: spec.VLLMGPUUtilization,
		LlamaMainGPU:       spec.LlamaMainGPU,
		LlamaTensorSplit:   spec.LlamaTensorSplit,
		LlamaNGPULayers:    spec.LlamaNGPULayers,
		LlamaCtxSize:       spec.LlamaCtxSize,
		SGLangTensorParallel: spec.SGLangTensorParallel,
		SGLangMemFraction:  spec.SGLangMemFraction,
		TGINumShard:        spec.TGINumShard,
		AutoStart:          autoStart,
	}
	return s.Save(saved)
}

// Delete removes a model configuration.
func (s *ModelStore) Delete(alias string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.models, alias)
	return s.persist()
}

// Update updates an existing saved model configuration.
func (s *ModelStore) Update(alias string, updateFn func(*SavedModel)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.models[alias]
	if !ok {
		return fmt.Errorf("model not found: %s", alias)
	}

	updateFn(&existing)
	s.models[alias] = existing
	return s.persist()
}

// Get returns a saved model by alias.
func (s *ModelStore) Get(alias string) (SavedModel, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.models[alias]
	return m, ok
}

// List returns all saved models.
func (s *ModelStore) List() []SavedModel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SavedModel, 0, len(s.models))
	for _, m := range s.models {
		result = append(result, m)
	}
	return result
}

// ListAutoStart returns models that should start automatically.
func (s *ModelStore) ListAutoStart() []SavedModel {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SavedModel, 0)
	for _, m := range s.models {
		if m.AutoStart {
			result = append(result, m)
		}
	}
	return result
}

// SetAutoStart updates the auto_start flag for a model.
func (s *ModelStore) SetAutoStart(alias string, autoStart bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.models[alias]
	if !ok {
		return nil // Model not saved yet
	}

	m.AutoStart = autoStart
	s.models[alias] = m
	return s.persist()
}

// ToSpec converts SavedModel back to ModelSpec.
func (m SavedModel) ToSpec() ModelSpec {
	return ModelSpec{
		Alias:              m.Alias,
		Provider:           m.Provider,
		Format:             m.Format,
		Capabilities:       m.Capabilities,
		HFRepo:             m.HFRepo,
		HFFile:             m.HFFile,
		HFRevision:         m.HFRevision,
		GGUFURL:            m.GGUFURL,
		GPUDevice:          m.GPUDevice,
		VLLMTensorParallel: m.VLLMTensorParallel,
		VLLMMaxModelLen:    m.VLLMMaxModelLen,
		VLLMGPUUtilization: m.VLLMGPUUtilization,
		LlamaMainGPU:       m.LlamaMainGPU,
		LlamaTensorSplit:   m.LlamaTensorSplit,
		LlamaNGPULayers:    m.LlamaNGPULayers,
		LlamaCtxSize:       m.LlamaCtxSize,
		SGLangTensorParallel: m.SGLangTensorParallel,
		SGLangMemFraction:  m.SGLangMemFraction,
		TGINumShard:        m.TGINumShard,
	}
}

func (s *ModelStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	var models []SavedModel
	if err := json.Unmarshal(data, &models); err != nil {
		return err
	}

	s.models = make(map[string]SavedModel, len(models))
	for _, m := range models {
		s.models[m.Alias] = m
	}

	s.logger.WithField("count", len(models)).Info("Loaded saved model configurations")
	return nil
}

func (s *ModelStore) persist() error {
	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	models := make([]SavedModel, 0, len(s.models))
	for _, m := range s.models {
		models = append(models, m)
	}

	data, err := json.MarshalIndent(models, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}


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
	Alias        string       `json:"alias"`
	Provider     ProviderKind `json:"provider"`
	Format       ModelFormat  `json:"format"`
	Capabilities []Capability `json:"capabilities,omitempty"`
	HFRepo       string       `json:"hf_repo,omitempty"`
	HFFile       string       `json:"hf_file,omitempty"`
	HFRevision   string       `json:"hf_revision,omitempty"`
	GGUFURL      string       `json:"gguf_url,omitempty"`
	GPUDevice    string       `json:"gpu_device,omitempty"`

	// vLLM options
	VLLMTensorParallel int     `json:"vllm_tensor_parallel,omitempty"`
	VLLMMaxModelLen    int     `json:"vllm_max_model_len,omitempty"`
	VLLMGPUUtilization float64 `json:"vllm_gpu_utilization,omitempty"`
	VLLMExtraArgs           string  `json:"vllm_extra_args,omitempty"`
	VLLMQuantization        string  `json:"vllm_quantization,omitempty"`
	VLLMDtype               string  `json:"vllm_dtype,omitempty"`
	VLLMKVCacheDtype        string  `json:"vllm_kv_cache_dtype,omitempty"`
	VLLMMaxNumSeqs          int     `json:"vllm_max_num_seqs,omitempty"`
	VLLMEnforceEager        bool    `json:"vllm_enforce_eager,omitempty"`
	VLLMEnablePrefixCaching bool    `json:"vllm_enable_prefix_caching,omitempty"`
	VLLMEnableChunkedPrefill bool   `json:"vllm_enable_chunked_prefill,omitempty"`
	VLLMSwapSpace           int     `json:"vllm_swap_space,omitempty"`
	VLLMEnableAutoToolChoice bool   `json:"vllm_enable_auto_tool_choice,omitempty"`
	VLLMToolCallParser       string `json:"vllm_tool_call_parser,omitempty"`
	VLLMChatTemplate         string `json:"vllm_chat_template,omitempty"`

	// llama.cpp options
	LlamaMainGPU     int    `json:"llama_main_gpu,omitempty"`
	LlamaTensorSplit string `json:"llama_tensor_split,omitempty"`
	LlamaNGPULayers  int    `json:"llama_n_gpu_layers,omitempty"`
	LlamaCtxSize     int    `json:"llama_ctx_size,omitempty"`
	LlamaNParallel   int    `json:"llama_n_parallel,omitempty"`
	LlamaFlashAttn   bool   `json:"llama_flash_attn,omitempty"`
	LlamaJinja       bool   `json:"llama_jinja,omitempty"`
	LlamaCacheReuse  int    `json:"llama_cache_reuse,omitempty"`
	LlamaExtraArgs   string `json:"llama_extra_args,omitempty"`
	LlamaBatchSize   int    `json:"llama_batch_size,omitempty"`
	LlamaUBatchSize  int    `json:"llama_ubatch_size,omitempty"`
	LlamaCacheTypeK  string `json:"llama_cache_type_k,omitempty"`
	LlamaCacheTypeV  string `json:"llama_cache_type_v,omitempty"`
	LlamaMlock       bool   `json:"llama_mlock,omitempty"`
	LlamaChatTemplate string `json:"llama_chat_template,omitempty"`

	// SGLang options
	SGLangTensorParallel   int     `json:"sglang_tensor_parallel,omitempty"`
	SGLangDataParallel     int     `json:"sglang_data_parallel,omitempty"`
	SGLangMemFraction      float64 `json:"sglang_mem_fraction,omitempty"`
	SGLangContextLen       int     `json:"sglang_context_len,omitempty"`
	SGLangChunkedPrefill   bool    `json:"sglang_chunked_prefill,omitempty"`
	SGLangQuantization     string  `json:"sglang_quantization,omitempty"`
	SGLangAttentionBackend string  `json:"sglang_attention_backend,omitempty"`
	SGLangExtraArgs        string  `json:"sglang_extra_args,omitempty"`
	SGLangToolCallParser   string  `json:"sglang_tool_call_parser,omitempty"`

	// TGI options
	TGINumShard           int     `json:"tgi_num_shard,omitempty"`
	TGIMaxConcurrentReqs  int     `json:"tgi_max_concurrent_reqs,omitempty"`
	TGIMaxInputLen        int     `json:"tgi_max_input_len,omitempty"`
	TGIMaxTotalTokens     int     `json:"tgi_max_total_tokens,omitempty"`
	TGIQuantize           string  `json:"tgi_quantize,omitempty"`
	TGICudaMemoryFraction float64 `json:"tgi_cuda_memory_fraction,omitempty"`
	TGIExtraArgs          string  `json:"tgi_extra_args,omitempty"`

	// TEI options
	TEICPUMode           bool   `json:"tei_cpu_mode,omitempty"`
	TEIMaxBatchTokens    int    `json:"tei_max_batch_tokens,omitempty"`
	TEIMaxConcurrentReqs int    `json:"tei_max_concurrent_reqs,omitempty"`
	TEIPooling           string `json:"tei_pooling,omitempty"`
	TEIDtype             string `json:"tei_dtype,omitempty"`
	TEIExtraArgs         string `json:"tei_extra_args,omitempty"`

	// Meta
	AutoStart bool      `json:"auto_start"` // Start on server boot
	SavedAt   time.Time `json:"saved_at"`
}

// ModelStore persists model configurations to disk.
type ModelStore struct {
	mu     sync.RWMutex
	path   string
	models map[string]SavedModel
	logger *logrus.Logger
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
		Alias:                spec.Alias,
		Provider:             spec.Provider,
		Format:               spec.Format,
		Capabilities:         spec.Capabilities,
		HFRepo:               spec.HFRepo,
		HFFile:               spec.HFFile,
		HFRevision:           spec.HFRevision,
		GGUFURL:              spec.GGUFURL,
		GPUDevice:            spec.GPUDevice,
		VLLMTensorParallel:   spec.VLLMTensorParallel,
		VLLMMaxModelLen:      spec.VLLMMaxModelLen,
		VLLMGPUUtilization:   spec.VLLMGPUUtilization,
		VLLMExtraArgs:        spec.VLLMExtraArgs,
		LlamaMainGPU:         spec.LlamaMainGPU,
		LlamaTensorSplit:     spec.LlamaTensorSplit,
		LlamaNGPULayers:      spec.LlamaNGPULayers,
		LlamaCtxSize:         spec.LlamaCtxSize,
		LlamaNParallel:       spec.LlamaNParallel,
		LlamaFlashAttn:       spec.LlamaFlashAttn,
		LlamaJinja:           spec.LlamaJinja,
		LlamaCacheReuse:      spec.LlamaCacheReuse,
		LlamaExtraArgs:       spec.LlamaExtraArgs,
		SGLangTensorParallel:    spec.SGLangTensorParallel,
		SGLangDataParallel:     spec.SGLangDataParallel,
		SGLangMemFraction:      spec.SGLangMemFraction,
		SGLangContextLen:       spec.SGLangContextLen,
		SGLangChunkedPrefill:   spec.SGLangChunkedPrefill,
		SGLangQuantization:     spec.SGLangQuantization,
		SGLangAttentionBackend: spec.SGLangAttentionBackend,
		SGLangExtraArgs:        spec.SGLangExtraArgs,
		TGINumShard:            spec.TGINumShard,
		TGIMaxConcurrentReqs:   spec.TGIMaxConcurrentReqs,
		TGIMaxInputLen:         spec.TGIMaxInputLen,
		TGIMaxTotalTokens:      spec.TGIMaxTotalTokens,
		TGIQuantize:            spec.TGIQuantize,
		TGICudaMemoryFraction:  spec.TGICudaMemoryFraction,
		TGIExtraArgs:           spec.TGIExtraArgs,
		TEICPUMode:              spec.TEICPUMode,
		TEIMaxBatchTokens:       spec.TEIMaxBatchTokens,
		TEIMaxConcurrentReqs:    spec.TEIMaxConcurrentReqs,
		TEIPooling:              spec.TEIPooling,
		TEIDtype:                spec.TEIDtype,
		TEIExtraArgs:            spec.TEIExtraArgs,
		VLLMQuantization:        spec.VLLMQuantization,
		VLLMDtype:               spec.VLLMDtype,
		VLLMKVCacheDtype:        spec.VLLMKVCacheDtype,
		VLLMMaxNumSeqs:          spec.VLLMMaxNumSeqs,
		VLLMEnforceEager:        spec.VLLMEnforceEager,
		VLLMEnablePrefixCaching: spec.VLLMEnablePrefixCaching,
		VLLMEnableChunkedPrefill: spec.VLLMEnableChunkedPrefill,
		VLLMSwapSpace:           spec.VLLMSwapSpace,
		LlamaBatchSize:          spec.LlamaBatchSize,
		LlamaUBatchSize:         spec.LlamaUBatchSize,
		LlamaCacheTypeK:         spec.LlamaCacheTypeK,
		LlamaCacheTypeV:         spec.LlamaCacheTypeV,
		LlamaMlock:              spec.LlamaMlock,
		LlamaChatTemplate:       spec.LlamaChatTemplate,
		VLLMEnableAutoToolChoice: spec.VLLMEnableAutoToolChoice,
		VLLMToolCallParser:       spec.VLLMToolCallParser,
		VLLMChatTemplate:         spec.VLLMChatTemplate,
		SGLangToolCallParser:     spec.SGLangToolCallParser,
		AutoStart:               autoStart,
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
		Alias:                m.Alias,
		Provider:             m.Provider,
		Format:               m.Format,
		Capabilities:         m.Capabilities,
		HFRepo:               m.HFRepo,
		HFFile:               m.HFFile,
		HFRevision:           m.HFRevision,
		GGUFURL:              m.GGUFURL,
		GPUDevice:            m.GPUDevice,
		VLLMTensorParallel:   m.VLLMTensorParallel,
		VLLMMaxModelLen:      m.VLLMMaxModelLen,
		VLLMGPUUtilization:   m.VLLMGPUUtilization,
		VLLMExtraArgs:        m.VLLMExtraArgs,
		LlamaMainGPU:         m.LlamaMainGPU,
		LlamaTensorSplit:     m.LlamaTensorSplit,
		LlamaNGPULayers:      m.LlamaNGPULayers,
		LlamaCtxSize:         m.LlamaCtxSize,
		LlamaNParallel:       m.LlamaNParallel,
		LlamaFlashAttn:       m.LlamaFlashAttn,
		LlamaJinja:           m.LlamaJinja,
		LlamaCacheReuse:      m.LlamaCacheReuse,
		LlamaExtraArgs:       m.LlamaExtraArgs,
		SGLangTensorParallel:    m.SGLangTensorParallel,
		SGLangDataParallel:     m.SGLangDataParallel,
		SGLangMemFraction:      m.SGLangMemFraction,
		SGLangContextLen:       m.SGLangContextLen,
		SGLangChunkedPrefill:   m.SGLangChunkedPrefill,
		SGLangQuantization:     m.SGLangQuantization,
		SGLangAttentionBackend: m.SGLangAttentionBackend,
		SGLangExtraArgs:        m.SGLangExtraArgs,
		TGINumShard:            m.TGINumShard,
		TGIMaxConcurrentReqs:   m.TGIMaxConcurrentReqs,
		TGIMaxInputLen:         m.TGIMaxInputLen,
		TGIMaxTotalTokens:      m.TGIMaxTotalTokens,
		TGIQuantize:            m.TGIQuantize,
		TGICudaMemoryFraction:  m.TGICudaMemoryFraction,
		TGIExtraArgs:           m.TGIExtraArgs,
		TEICPUMode:              m.TEICPUMode,
		TEIMaxBatchTokens:       m.TEIMaxBatchTokens,
		TEIMaxConcurrentReqs:    m.TEIMaxConcurrentReqs,
		TEIPooling:              m.TEIPooling,
		TEIDtype:                m.TEIDtype,
		TEIExtraArgs:            m.TEIExtraArgs,
		VLLMQuantization:        m.VLLMQuantization,
		VLLMDtype:               m.VLLMDtype,
		VLLMKVCacheDtype:        m.VLLMKVCacheDtype,
		VLLMMaxNumSeqs:          m.VLLMMaxNumSeqs,
		VLLMEnforceEager:        m.VLLMEnforceEager,
		VLLMEnablePrefixCaching: m.VLLMEnablePrefixCaching,
		VLLMEnableChunkedPrefill: m.VLLMEnableChunkedPrefill,
		VLLMSwapSpace:           m.VLLMSwapSpace,
		LlamaBatchSize:          m.LlamaBatchSize,
		LlamaUBatchSize:         m.LlamaUBatchSize,
		LlamaCacheTypeK:         m.LlamaCacheTypeK,
		LlamaCacheTypeV:         m.LlamaCacheTypeV,
		LlamaMlock:              m.LlamaMlock,
		LlamaChatTemplate:       m.LlamaChatTemplate,
		VLLMEnableAutoToolChoice: m.VLLMEnableAutoToolChoice,
		VLLMToolCallParser:       m.VLLMToolCallParser,
		VLLMChatTemplate:         m.VLLMChatTemplate,
		SGLangToolCallParser:     m.SGLangToolCallParser,
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

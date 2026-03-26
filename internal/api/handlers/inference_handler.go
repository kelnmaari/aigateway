package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/inference"
)

// InferenceHandler exposes minimal endpoints to manage inference providers.
type InferenceHandler struct {
	router     *inference.Router
	modelStore *inference.ModelStore
	logger     *logrus.Logger
}

// NewInferenceHandler constructs handler.
func NewInferenceHandler(router *inference.Router, logger *logrus.Logger) *InferenceHandler {
	return &InferenceHandler{
		router: router,
		logger: logger,
	}
}

// SetModelStore sets the model store for persistence.
func (h *InferenceHandler) SetModelStore(store *inference.ModelStore) {
	h.modelStore = store
}

// LoadRequest describes model spec input.
type LoadRequest struct {
	Alias        string                 `json:"alias" binding:"required"`
	Provider     inference.ProviderKind `json:"provider" binding:"required"`
	Format       inference.ModelFormat  `json:"format" binding:"required"`
	Capabilities []inference.Capability `json:"capabilities"`
	HFRepo       string                 `json:"hf_repo"`
	HFFile       string                 `json:"hf_file"`
	HFRevision   string                 `json:"hf_revision"`
	GGUFURL      string                 `json:"gguf_url"`
	ExpectedSHA  string                 `json:"expected_sha"`

	// GPU selection (e.g., "0", "1", "0,1" for multi-GPU)
	GPUDevice string `json:"gpu_device"`

	// vLLM options
	VLLMTensorParallel int     `json:"vllm_tensor_parallel"`
	VLLMMaxModelLen    int     `json:"vllm_max_model_len"`
	VLLMGPUUtilization float64 `json:"vllm_gpu_utilization"`
	VLLMExtraArgs           string  `json:"vllm_extra_args"`
	VLLMQuantization        string  `json:"vllm_quantization"`
	VLLMDtype               string  `json:"vllm_dtype"`
	VLLMKVCacheDtype        string  `json:"vllm_kv_cache_dtype"`
	VLLMMaxNumSeqs          int     `json:"vllm_max_num_seqs"`
	VLLMEnforceEager        bool    `json:"vllm_enforce_eager"`
	VLLMEnablePrefixCaching bool    `json:"vllm_enable_prefix_caching"`
	VLLMEnableChunkedPrefill bool   `json:"vllm_enable_chunked_prefill"`
	VLLMSwapSpace           int     `json:"vllm_swap_space"`
	VLLMEnableAutoToolChoice bool   `json:"vllm_enable_auto_tool_choice"`
	VLLMToolCallParser       string `json:"vllm_tool_call_parser"`
	VLLMChatTemplate         string `json:"vllm_chat_template"`

	// llama.cpp options
	LlamaMainGPU     int    `json:"llama_main_gpu"`
	LlamaTensorSplit string `json:"llama_tensor_split"`
	LlamaNGPULayers  int    `json:"llama_n_gpu_layers"`
	LlamaCtxSize     int    `json:"llama_ctx_size"`    // Context size (default: 2048)
	LlamaNParallel   int    `json:"llama_n_parallel"`  // Parallel slots (concurrent requests)
	LlamaFlashAttn   bool   `json:"llama_flash_attn"`  // Enable Flash Attention
	LlamaJinja       bool   `json:"llama_jinja"`       // Enable Jinja template processing
	LlamaCacheReuse  int    `json:"llama_cache_reuse"` // KV cache reuse (0=default, -1=disable for SWA models)
	LlamaExtraArgs   string `json:"llama_extra_args"`  // Extra CLI args for llama-server
	LlamaBatchSize   int    `json:"llama_batch_size"`
	LlamaUBatchSize  int    `json:"llama_ubatch_size"`
	LlamaCacheTypeK  string `json:"llama_cache_type_k"`
	LlamaCacheTypeV  string `json:"llama_cache_type_v"`
	LlamaMlock       bool   `json:"llama_mlock"`
	LlamaChatTemplate string `json:"llama_chat_template"`

	// SGLang options
	SGLangTensorParallel int     `json:"sglang_tensor_parallel"`
	SGLangDataParallel   int     `json:"sglang_data_parallel"`
	SGLangMemFraction    float64 `json:"sglang_mem_fraction"`
	SGLangContextLen     int     `json:"sglang_context_len"`
	SGLangChunkedPrefill   bool    `json:"sglang_chunked_prefill"`
	SGLangQuantization     string  `json:"sglang_quantization"`
	SGLangAttentionBackend string  `json:"sglang_attention_backend"`
	SGLangExtraArgs        string  `json:"sglang_extra_args"`
	SGLangToolCallParser   string  `json:"sglang_tool_call_parser"`

	// TGI options
	TGINumShard          int `json:"tgi_num_shard"`
	TGIMaxConcurrentReqs int `json:"tgi_max_concurrent_reqs"`
	TGIMaxInputLen       int `json:"tgi_max_input_len"`
	TGIMaxTotalTokens     int     `json:"tgi_max_total_tokens"`
	TGIQuantize           string  `json:"tgi_quantize"`
	TGICudaMemoryFraction float64 `json:"tgi_cuda_memory_fraction"`
	TGIExtraArgs          string  `json:"tgi_extra_args"`

	// TEI options
	TEIMaxBatchTokens    int    `json:"tei_max_batch_tokens"`
	TEIMaxConcurrentReqs int    `json:"tei_max_concurrent_reqs"`
	TEIPooling           string `json:"tei_pooling"`
	TEIDtype             string `json:"tei_dtype"`
	TEIExtraArgs         string `json:"tei_extra_args"`
}

// ModelsResponse describes current tracked models.
type ModelsResponse struct {
	Alias        string                 `json:"alias"`
	Provider     inference.ProviderKind `json:"provider"`
	Format       inference.ModelFormat  `json:"format"`
	Status       inference.ModelStatus  `json:"status"`
	LastError    string                 `json:"last_error,omitempty"`
	Endpoint     string                 `json:"endpoint,omitempty"`
	LocalPath    string                 `json:"local_path,omitempty"`
	LastUsed     string                 `json:"last_used,omitempty"`
	Capabilities []inference.Capability `json:"capabilities,omitempty"`
	Pinned       bool                   `json:"pinned,omitempty"`

	// Source info for restart
	HFRepo      string `json:"hf_repo,omitempty"`
	HFFile      string `json:"hf_file,omitempty"`
	GGUFURL     string `json:"gguf_url,omitempty"`
	ContainerID string `json:"container_id,omitempty"`

	// Provider params (for restart with same settings)
	VLLMTensorParallel   int     `json:"vllm_tensor_parallel,omitempty"`
	VLLMMaxModelLen      int     `json:"vllm_max_model_len,omitempty"`
	VLLMGPUUtilization   float64 `json:"vllm_gpu_utilization,omitempty"`
	VLLMExtraArgs        string  `json:"vllm_extra_args,omitempty"`
	LlamaMainGPU         int     `json:"llama_main_gpu,omitempty"`
	LlamaTensorSplit     string  `json:"llama_tensor_split,omitempty"`
	LlamaNGPULayers      int     `json:"llama_n_gpu_layers,omitempty"`
	LlamaCtxSize         int     `json:"llama_ctx_size,omitempty"`
	LlamaNParallel       int     `json:"llama_n_parallel,omitempty"`
	LlamaFlashAttn       bool    `json:"llama_flash_attn,omitempty"`
	LlamaJinja           bool    `json:"llama_jinja,omitempty"`
	LlamaCacheReuse      int     `json:"llama_cache_reuse,omitempty"`
	LlamaExtraArgs       string  `json:"llama_extra_args,omitempty"`
	SGLangTensorParallel int     `json:"sglang_tensor_parallel,omitempty"`
	SGLangMemFraction    float64 `json:"sglang_mem_fraction,omitempty"`
	TGINumShard          int     `json:"tgi_num_shard,omitempty"`
}

// PostLoad starts container after ensuring artifacts.
func (h *InferenceHandler) PostLoad(c *gin.Context) {
	var req LoadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log received parameters for debugging
	h.logger.WithFields(logrus.Fields{
		"alias":                req.Alias,
		"provider":             req.Provider,
		"format":               req.Format,
		"hf_repo":              req.HFRepo,
		"gpu_device":           req.GPUDevice,
		"vllm_tensor_parallel": req.VLLMTensorParallel,
		"vllm_max_model_len":   req.VLLMMaxModelLen,
		"vllm_gpu_utilization": req.VLLMGPUUtilization,
		"sglang_mem_fraction":  req.SGLangMemFraction,
	}).Info("Received model load request")

	spec := inference.ModelSpec{
		Alias:              req.Alias,
		Provider:           req.Provider,
		Format:             req.Format,
		Capabilities:       req.Capabilities,
		HFRepo:             req.HFRepo,
		HFFile:             req.HFFile,
		HFRevision:         req.HFRevision,
		GGUFURL:            req.GGUFURL,
		ExpectedSHA:        req.ExpectedSHA,
		GPUDevice:          req.GPUDevice,
		VLLMTensorParallel: req.VLLMTensorParallel,
		VLLMMaxModelLen:    req.VLLMMaxModelLen,
		VLLMGPUUtilization: req.VLLMGPUUtilization,
		VLLMExtraArgs:      req.VLLMExtraArgs,
		LlamaMainGPU:       req.LlamaMainGPU,
		LlamaTensorSplit:   req.LlamaTensorSplit,
		LlamaNGPULayers:    req.LlamaNGPULayers,
		LlamaCtxSize:       req.LlamaCtxSize,
		LlamaNParallel:     req.LlamaNParallel,
		LlamaFlashAttn:     req.LlamaFlashAttn,
		LlamaJinja:         req.LlamaJinja,
		LlamaCacheReuse:    req.LlamaCacheReuse,
		LlamaExtraArgs:     req.LlamaExtraArgs,
		// SGLang
		SGLangTensorParallel: req.SGLangTensorParallel,
		SGLangDataParallel:   req.SGLangDataParallel,
		SGLangMemFraction:    req.SGLangMemFraction,
		SGLangContextLen:     req.SGLangContextLen,
		SGLangChunkedPrefill: req.SGLangChunkedPrefill,
		// TGI
		TGINumShard:          req.TGINumShard,
		TGIMaxConcurrentReqs: req.TGIMaxConcurrentReqs,
		TGIMaxInputLen:       req.TGIMaxInputLen,
		TGIMaxTotalTokens:    req.TGIMaxTotalTokens,
		TGIQuantize:          req.TGIQuantize,
		TGICudaMemoryFraction: req.TGICudaMemoryFraction,
		TGIExtraArgs:         req.TGIExtraArgs,
		// TEI
		TEIMaxBatchTokens:    req.TEIMaxBatchTokens,
		TEIMaxConcurrentReqs: req.TEIMaxConcurrentReqs,
		TEIPooling:           req.TEIPooling,
		TEIDtype:             req.TEIDtype,
		TEIExtraArgs:         req.TEIExtraArgs,
		// vLLM new fields
		VLLMQuantization:        req.VLLMQuantization,
		VLLMDtype:               req.VLLMDtype,
		VLLMKVCacheDtype:        req.VLLMKVCacheDtype,
		VLLMMaxNumSeqs:          req.VLLMMaxNumSeqs,
		VLLMEnforceEager:        req.VLLMEnforceEager,
		VLLMEnablePrefixCaching: req.VLLMEnablePrefixCaching,
		VLLMEnableChunkedPrefill: req.VLLMEnableChunkedPrefill,
		VLLMSwapSpace:           req.VLLMSwapSpace,
		VLLMEnableAutoToolChoice: req.VLLMEnableAutoToolChoice,
		VLLMToolCallParser:       req.VLLMToolCallParser,
		VLLMChatTemplate:         req.VLLMChatTemplate,
		// SGLang new fields
		SGLangQuantization:     req.SGLangQuantization,
		SGLangAttentionBackend: req.SGLangAttentionBackend,
		SGLangExtraArgs:        req.SGLangExtraArgs,
		SGLangToolCallParser:   req.SGLangToolCallParser,
		// llama.cpp new fields
		LlamaBatchSize:  req.LlamaBatchSize,
		LlamaUBatchSize: req.LlamaUBatchSize,
		LlamaCacheTypeK: req.LlamaCacheTypeK,
		LlamaCacheTypeV: req.LlamaCacheTypeV,
		LlamaMlock:      req.LlamaMlock,
		LlamaChatTemplate: req.LlamaChatTemplate,
	}

	// Always use the provided spec from the request, not a cached one from registry.
	// This ensures that when user explicitly selects provider=tei, we use tei,
	// not a previously cached spec with provider=vllm.
	inst, err := h.router.EnsureBySpec(c.Request.Context(), spec)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alias":  inst.Spec.Alias,
		"status": inst.Status,
		"endpoint": func() string {
			if inst.Handle != nil {
				return inst.Handle.Endpoint
			}
			return ""
		}(),
	})
}

// PostPrepare only downloads artifacts.
func (h *InferenceHandler) PostPrepare(c *gin.Context) {
	var req LoadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	spec := inference.ModelSpec{
		Alias:        req.Alias,
		Provider:     req.Provider,
		Format:       req.Format,
		Capabilities: req.Capabilities,
		HFRepo:       req.HFRepo,
		HFFile:       req.HFFile,
		HFRevision:   req.HFRevision,
		GGUFURL:      req.GGUFURL,
		ExpectedSHA:  req.ExpectedSHA,
		GPUDevice:    req.GPUDevice,
	}
	inst, err := h.router.PrepareBySpec(c.Request.Context(), spec)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"alias":  inst.Spec.Alias,
		"status": inst.Status,
		"path":   inst.Spec.LocalPath,
	})
}

// PostStop stops running container by alias.
func (h *InferenceHandler) PostStop(c *gin.Context) {
	alias := c.Query("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}
	if err := h.router.Stop(c.Request.Context(), alias); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// PostRestart stops running container and starts it again with same parameters.
func (h *InferenceHandler) PostRestart(c *gin.Context) {
	alias := c.Query("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}

	// Get current instance to capture the full spec before cleanup
	inst := h.router.GetModelInstance(alias)
	if inst == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "model not found: " + alias})
		return
	}
	spec := inst.Spec

	h.logger.WithFields(logrus.Fields{
		"alias":    alias,
		"provider": spec.Provider,
	}).Info("Restarting model")

	// Stop container (ignore error if already stopped)
	_ = h.router.Stop(c.Request.Context(), alias)

	// Forget old instance to force clean re-start
	h.router.ForgetModel(alias)

	// Re-register spec and start fresh
	newInst, err := h.router.EnsureBySpec(c.Request.Context(), spec)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alias":  newInst.Spec.Alias,
		"status": newInst.Status,
		"endpoint": func() string {
			if newInst.Handle != nil {
				return newInst.Handle.Endpoint
			}
			return ""
		}(),
	})
}

// GetHealth performs health check by alias.
func (h *InferenceHandler) GetHealth(c *gin.Context) {
	alias := c.Query("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}
	if err := h.router.Health(c.Request.Context(), alias); err != nil {
		// Return 200 with error info instead of 503 - allows UI to show status gracefully
		c.JSON(http.StatusOK, gin.H{"status": "error", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// PostEvict stops model container (artifacts remain on disk).
func (h *InferenceHandler) PostEvict(c *gin.Context) {
	alias := c.Query("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}
	if err := h.router.Evict(c.Request.Context(), alias); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// PostPin marks model as pinned (skip auto-stop/evict).
func (h *InferenceHandler) PostPin(c *gin.Context) {
	alias := c.Query("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}
	h.router.Pin(alias)
	c.Status(http.StatusNoContent)
}

// PostUnpin removes pin mark.
func (h *InferenceHandler) PostUnpin(c *gin.Context) {
	alias := c.Query("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}
	h.router.Unpin(alias)
	c.Status(http.StatusNoContent)
}

// PostDeleteArtifacts removes local files for alias (model must be stopped/evicted).
func (h *InferenceHandler) PostDeleteArtifacts(c *gin.Context) {
	alias := c.Query("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}
	if err := h.router.DeleteArtifacts(alias); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// GetCache lists cached artifacts.
func (h *InferenceHandler) GetCache(c *gin.Context) {
	artifacts, err := h.router.ListArtifacts()
	if err != nil {
		h.logger.WithError(err).Warn("Failed to list cache artifacts")
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	if len(artifacts) == 0 {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	c.JSON(http.StatusOK, artifacts)
}

// PostEvictCache removes cache files until under limit_bytes.
func (h *InferenceHandler) PostEvictCache(c *gin.Context) {
	limitStr := c.Query("limit_bytes")
	if limitStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit_bytes is required"})
		return
	}
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit_bytes"})
		return
	}
	if err := h.router.EvictCacheSize(limit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// PostClearCache clears all cached model files.
func (h *InferenceHandler) PostClearCache(c *gin.Context) {
	freedBytes, err := h.router.ClearCache()
	if err != nil {
		h.logger.WithError(err).Error("Failed to clear cache")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":       err.Error(),
			"freed_bytes": freedBytes,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":     "Cache cleared successfully",
		"freed_bytes": freedBytes,
	})
}

// GetModels returns list of tracked models.
func (h *InferenceHandler) GetModels(c *gin.Context) {
	models := h.router.ListModels()
	resp := make([]ModelsResponse, 0, len(models))
	for _, m := range models {
		item := ModelsResponse{
			Alias:        m.Spec.Alias,
			Provider:     m.Spec.Provider,
			Format:       m.Spec.Format,
			Status:       m.Status,
			LocalPath:    m.Spec.LocalPath,
			Capabilities: m.Spec.Capabilities,
			LastError:    m.Error,
			// Source info
			HFRepo:  m.Spec.HFRepo,
			HFFile:  m.Spec.HFFile,
			GGUFURL: m.Spec.GGUFURL,
			// Provider params
			VLLMTensorParallel:   m.Spec.VLLMTensorParallel,
			VLLMMaxModelLen:      m.Spec.VLLMMaxModelLen,
			VLLMGPUUtilization:   m.Spec.VLLMGPUUtilization,
			VLLMExtraArgs:        m.Spec.VLLMExtraArgs,
			LlamaMainGPU:         m.Spec.LlamaMainGPU,
			LlamaTensorSplit:     m.Spec.LlamaTensorSplit,
			LlamaNGPULayers:      m.Spec.LlamaNGPULayers,
			LlamaCtxSize:         m.Spec.LlamaCtxSize,
			LlamaNParallel:       m.Spec.LlamaNParallel,
			LlamaFlashAttn:       m.Spec.LlamaFlashAttn,
			LlamaJinja:           m.Spec.LlamaJinja,
			LlamaCacheReuse:      m.Spec.LlamaCacheReuse,
			LlamaExtraArgs:       m.Spec.LlamaExtraArgs,
			SGLangTensorParallel: m.Spec.SGLangTensorParallel,
			SGLangMemFraction:    m.Spec.SGLangMemFraction,
			TGINumShard:          m.Spec.TGINumShard,
		}
		if m.Handle != nil {
			item.Endpoint = m.Handle.Endpoint
			item.ContainerID = m.Handle.ID
		}
		if !m.LastUsed.IsZero() {
			item.LastUsed = m.LastUsed.UTC().Format(time.RFC3339)
		}
		item.Pinned = h.router.IsPinned(m.Spec.Alias)
		resp = append(resp, item)
	}
	c.JSON(http.StatusOK, resp)
}

// GetLogs returns recent container logs for alias.
func (h *InferenceHandler) GetLogs(c *gin.Context) {
	alias := c.Query("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}
	tailStr := c.DefaultQuery("tail", "100")
	tail, err := strconv.Atoi(tailStr)
	if err != nil || tail <= 0 {
		tail = 100
	}
	logs, err := h.router.ContainerLogs(c.Request.Context(), alias, tail)
	if err != nil {
		// Return 200 with empty logs for not-running models
		c.JSON(http.StatusOK, gin.H{"alias": alias, "logs": "", "lines": 0, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alias": alias, "logs": logs, "lines": tail})
}

// GetMetrics returns provider metrics (Prometheus format) for alias.
func (h *InferenceHandler) GetMetrics(c *gin.Context) {
	alias := c.Query("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}
	metrics, err := h.router.ContainerMetrics(c.Request.Context(), alias)
	if err != nil {
		// Return 200 with empty metrics for not-running models
		c.JSON(http.StatusOK, gin.H{"alias": alias, "metrics": "", "content_type": "text/plain", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alias": alias, "metrics": metrics, "content_type": "text/plain"})
}

// ConvertTRT handles TensorRT-LLM model conversion request.
// POST /api/system/inference/convert-trt
func (h *InferenceHandler) ConvertTRT(c *gin.Context) {
	var req struct {
		ModelID        string `json:"model_id" binding:"required"`
		Alias          string `json:"alias"`
		Dtype          string `json:"dtype"`
		MaxBatchSize   int    `json:"max_batch_size"`
		MaxSeqLen      int    `json:"max_seq_len"`
		TensorParallel int    `json:"tensor_parallel"`
		Force          bool   `json:"force"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	converter := h.router.TRTConverter()
	if converter == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "TRT converter not configured"})
		return
	}

	enginePath, err := converter.Convert(c.Request.Context(), inference.ConvertRequest{
		ModelID:        req.ModelID,
		Alias:          req.Alias,
		Dtype:          req.Dtype,
		MaxBatchSize:   req.MaxBatchSize,
		MaxSeqLen:      req.MaxSeqLen,
		TensorParallel: req.TensorParallel,
		Force:          req.Force,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "conversion failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "completed",
		"engine_path": enginePath,
		"alias":       req.Alias,
	})
}

// ListTRTEngines returns all cached TRT engines.
// GET /api/system/inference/trt-engines
func (h *InferenceHandler) ListTRTEngines(c *gin.Context) {
	converter := h.router.TRTConverter()
	if converter == nil {
		// Return empty array when TRT converter not configured
		c.JSON(http.StatusOK, []gin.H{})
		return
	}

	engines, err := converter.ListEngines()
	if err != nil {
		h.logger.WithError(err).Warn("Failed to list TRT engines")
		c.JSON(http.StatusOK, []gin.H{})
		return
	}

	if len(engines) == 0 {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}

	c.JSON(http.StatusOK, engines)
}

// DeleteTRTEngine deletes a cached TRT engine.
// POST /api/system/inference/delete-trt-engine?alias=X
func (h *InferenceHandler) DeleteTRTEngine(c *gin.Context) {
	alias := c.Query("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}

	converter := h.router.TRTConverter()
	if converter == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "TRT converter not configured"})
		return
	}

	if err := converter.DeleteEngine(alias); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted", "alias": alias})
}

// --- Saved Models API ---

// SavedModelResponse describes a saved model configuration.
type SavedModelResponse struct {
	Alias                string                 `json:"alias"`
	Provider             inference.ProviderKind `json:"provider"`
	Format               inference.ModelFormat  `json:"format"`
	Capabilities         []inference.Capability `json:"capabilities,omitempty"`
	HFRepo               string                 `json:"hf_repo,omitempty"`
	HFFile               string                 `json:"hf_file,omitempty"`
	GGUFURL              string                 `json:"gguf_url,omitempty"`
	GPUDevice            string                 `json:"gpu_device,omitempty"`
	AutoStart            bool                   `json:"auto_start"`
	SavedAt              string                 `json:"saved_at,omitempty"`
	VLLMTensorParallel   int                    `json:"vllm_tensor_parallel,omitempty"`
	VLLMMaxModelLen      int                    `json:"vllm_max_model_len,omitempty"`
	VLLMGPUUtilization   float64                `json:"vllm_gpu_utilization,omitempty"`
	LlamaMainGPU         int                    `json:"llama_main_gpu,omitempty"`
	LlamaNGPULayers      int                    `json:"llama_n_gpu_layers,omitempty"`
	LlamaCtxSize         int                    `json:"llama_ctx_size,omitempty"`
	LlamaNParallel       int                    `json:"llama_n_parallel,omitempty"`
	LlamaFlashAttn       bool                   `json:"llama_flash_attn,omitempty"`
	LlamaJinja           bool                   `json:"llama_jinja,omitempty"`
	LlamaTensorSplit     string                 `json:"llama_tensor_split,omitempty"`
	SGLangTensorParallel int                    `json:"sglang_tensor_parallel,omitempty"`
	SGLangMemFraction    float64                `json:"sglang_mem_fraction,omitempty"`
	TGINumShard          int                    `json:"tgi_num_shard,omitempty"`
}

// GetSavedModels returns list of saved model configurations.
// GET /api/system/inference/saved
func (h *InferenceHandler) GetSavedModels(c *gin.Context) {
	if h.modelStore == nil {
		c.JSON(http.StatusOK, []SavedModelResponse{})
		return
	}

	saved := h.modelStore.List()
	resp := make([]SavedModelResponse, 0, len(saved))
	for _, m := range saved {
		resp = append(resp, SavedModelResponse{
			Alias:                m.Alias,
			Provider:             m.Provider,
			Format:               m.Format,
			Capabilities:         m.Capabilities,
			HFRepo:               m.HFRepo,
			HFFile:               m.HFFile,
			GGUFURL:              m.GGUFURL,
			GPUDevice:            m.GPUDevice,
			AutoStart:            m.AutoStart,
			SavedAt:              m.SavedAt.Format(time.RFC3339),
			VLLMTensorParallel:   m.VLLMTensorParallel,
			VLLMMaxModelLen:      m.VLLMMaxModelLen,
			VLLMGPUUtilization:   m.VLLMGPUUtilization,
			LlamaMainGPU:         m.LlamaMainGPU,
			LlamaNGPULayers:      m.LlamaNGPULayers,
			LlamaCtxSize:         m.LlamaCtxSize,
			LlamaNParallel:       m.LlamaNParallel,
			LlamaFlashAttn:       m.LlamaFlashAttn,
			LlamaJinja:           m.LlamaJinja,
			LlamaTensorSplit:     m.LlamaTensorSplit,
			SGLangTensorParallel: m.SGLangTensorParallel,
			SGLangMemFraction:    m.SGLangMemFraction,
			TGINumShard:          m.TGINumShard,
		})
	}
	c.JSON(http.StatusOK, resp)
}

// PostSaveModel saves a model configuration for later use.
// POST /api/system/inference/save
func (h *InferenceHandler) PostSaveModel(c *gin.Context) {
	if h.modelStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model store not configured"})
		return
	}

	var req struct {
		Alias     string `json:"alias" binding:"required"`
		AutoStart bool   `json:"auto_start"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get model spec from running/registered models
	models := h.router.ListModels()
	var found *inference.ModelInstance
	for _, m := range models {
		if m.Spec.Alias == req.Alias {
			found = m
			break
		}
	}

	if found == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "model not found in registry"})
		return
	}

	if err := h.modelStore.SaveFromSpec(found.Spec, req.AutoStart); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "saved", "alias": req.Alias, "auto_start": req.AutoStart})
}

// PostDeleteSaved deletes a saved model configuration.
// POST /api/system/inference/delete-saved?alias=...
func (h *InferenceHandler) PostDeleteSaved(c *gin.Context) {
	if h.modelStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model store not configured"})
		return
	}

	alias := c.Query("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}

	if err := h.modelStore.Delete(alias); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// PostSetAutoStart toggles auto_start flag for a saved model.
// POST /api/system/inference/auto-start?alias=...&enabled=true/false
func (h *InferenceHandler) PostSetAutoStart(c *gin.Context) {
	if h.modelStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model store not configured"})
		return
	}

	alias := c.Query("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}

	enabled := c.Query("enabled") == "true"
	if err := h.modelStore.SetAutoStart(alias, enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"alias": alias, "auto_start": enabled})
}

// UpdateSavedRequest represents request to update a saved model configuration.
type UpdateSavedRequest struct {
	Capabilities         []string `json:"capabilities,omitempty"` // Model capabilities (chat, embeddings, etc.)
	VLLMTensorParallel   *int     `json:"vllm_tensor_parallel,omitempty"`
	VLLMMaxModelLen      *int     `json:"vllm_max_model_len,omitempty"`
	VLLMGPUUtilization   *float64 `json:"vllm_gpu_utilization,omitempty"`
	VLLMExtraArgs        *string  `json:"vllm_extra_args,omitempty"`
	LlamaMainGPU         *int     `json:"llama_main_gpu,omitempty"`
	LlamaNGPULayers      *int     `json:"llama_n_gpu_layers,omitempty"`
	LlamaCtxSize         *int     `json:"llama_ctx_size,omitempty"`
	LlamaNParallel       *int     `json:"llama_n_parallel,omitempty"`
	LlamaFlashAttn       *bool    `json:"llama_flash_attn,omitempty"`
	LlamaJinja           *bool    `json:"llama_jinja,omitempty"`
	LlamaTensorSplit     *string  `json:"llama_tensor_split,omitempty"`
	LlamaCacheReuse      *int     `json:"llama_cache_reuse,omitempty"`
	LlamaExtraArgs       *string  `json:"llama_extra_args,omitempty"`
	SGLangTensorParallel *int     `json:"sglang_tensor_parallel,omitempty"`
	SGLangMemFraction    *float64 `json:"sglang_mem_fraction,omitempty"`
	TGINumShard              *int     `json:"tgi_num_shard,omitempty"`
	TGIQuantize              *string  `json:"tgi_quantize,omitempty"`
	TGICudaMemoryFraction    *float64 `json:"tgi_cuda_memory_fraction,omitempty"`
	TGIExtraArgs             *string  `json:"tgi_extra_args,omitempty"`
	TEIMaxBatchTokens        *int     `json:"tei_max_batch_tokens,omitempty"`
	TEIMaxConcurrentReqs     *int     `json:"tei_max_concurrent_reqs,omitempty"`
	TEIPooling               *string  `json:"tei_pooling,omitempty"`
	TEIDtype                 *string  `json:"tei_dtype,omitempty"`
	TEIExtraArgs             *string  `json:"tei_extra_args,omitempty"`
	VLLMQuantization         *string  `json:"vllm_quantization,omitempty"`
	VLLMDtype                *string  `json:"vllm_dtype,omitempty"`
	VLLMKVCacheDtype         *string  `json:"vllm_kv_cache_dtype,omitempty"`
	VLLMMaxNumSeqs           *int     `json:"vllm_max_num_seqs,omitempty"`
	VLLMEnforceEager         *bool    `json:"vllm_enforce_eager,omitempty"`
	VLLMEnablePrefixCaching  *bool    `json:"vllm_enable_prefix_caching,omitempty"`
	VLLMEnableChunkedPrefill *bool    `json:"vllm_enable_chunked_prefill,omitempty"`
	VLLMSwapSpace            *int     `json:"vllm_swap_space,omitempty"`
	VLLMEnableAutoToolChoice *bool    `json:"vllm_enable_auto_tool_choice,omitempty"`
	VLLMToolCallParser       *string  `json:"vllm_tool_call_parser,omitempty"`
	VLLMChatTemplate         *string  `json:"vllm_chat_template,omitempty"`
	SGLangQuantization       *string  `json:"sglang_quantization,omitempty"`
	SGLangAttentionBackend   *string  `json:"sglang_attention_backend,omitempty"`
	SGLangExtraArgs          *string  `json:"sglang_extra_args,omitempty"`
	LlamaBatchSize           *int     `json:"llama_batch_size,omitempty"`
	LlamaUBatchSize          *int     `json:"llama_ubatch_size,omitempty"`
	LlamaCacheTypeK          *string  `json:"llama_cache_type_k,omitempty"`
	LlamaCacheTypeV          *string  `json:"llama_cache_type_v,omitempty"`
	LlamaMlock               *bool    `json:"llama_mlock,omitempty"`
	LlamaChatTemplate        *string  `json:"llama_chat_template,omitempty"`
	SGLangToolCallParser     *string  `json:"sglang_tool_call_parser,omitempty"`
	GPUDevice                *string  `json:"gpu_device,omitempty"`
	AutoStart                *bool    `json:"auto_start,omitempty"` // Whether to auto-start on boot
}

// PostUpdateSaved updates a saved model configuration.
// POST /api/system/inference/update-saved?alias=...
func (h *InferenceHandler) PostUpdateSaved(c *gin.Context) {
	if h.modelStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model store not configured"})
		return
	}

	alias := c.Query("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}

	var req UpdateSavedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.modelStore.Update(alias, func(m *inference.SavedModel) {
		if req.VLLMTensorParallel != nil {
			m.VLLMTensorParallel = *req.VLLMTensorParallel
		}
		if req.VLLMMaxModelLen != nil {
			m.VLLMMaxModelLen = *req.VLLMMaxModelLen
		}
		if req.VLLMGPUUtilization != nil {
			m.VLLMGPUUtilization = *req.VLLMGPUUtilization
		}
		if req.VLLMExtraArgs != nil {
			m.VLLMExtraArgs = *req.VLLMExtraArgs
		}
		if req.LlamaMainGPU != nil {
			m.LlamaMainGPU = *req.LlamaMainGPU
		}
		if req.LlamaNGPULayers != nil {
			m.LlamaNGPULayers = *req.LlamaNGPULayers
		}
		if req.LlamaCtxSize != nil {
			m.LlamaCtxSize = *req.LlamaCtxSize
		}
		if req.LlamaNParallel != nil {
			m.LlamaNParallel = *req.LlamaNParallel
		}
		if req.LlamaFlashAttn != nil {
			m.LlamaFlashAttn = *req.LlamaFlashAttn
		}
		if req.LlamaJinja != nil {
			m.LlamaJinja = *req.LlamaJinja
		}
		if req.LlamaTensorSplit != nil {
			m.LlamaTensorSplit = *req.LlamaTensorSplit
		}
		if req.LlamaCacheReuse != nil {
			m.LlamaCacheReuse = *req.LlamaCacheReuse
		}
		if req.LlamaExtraArgs != nil {
			m.LlamaExtraArgs = *req.LlamaExtraArgs
		}
		if req.SGLangTensorParallel != nil {
			m.SGLangTensorParallel = *req.SGLangTensorParallel
		}
		if req.SGLangMemFraction != nil {
			m.SGLangMemFraction = *req.SGLangMemFraction
		}
		if req.TGINumShard != nil {
			m.TGINumShard = *req.TGINumShard
		}
		if req.TGIQuantize != nil {
			m.TGIQuantize = *req.TGIQuantize
		}
		if req.TGICudaMemoryFraction != nil {
			m.TGICudaMemoryFraction = *req.TGICudaMemoryFraction
		}
		if req.TGIExtraArgs != nil {
			m.TGIExtraArgs = *req.TGIExtraArgs
		}
		if req.TEIMaxBatchTokens != nil {
			m.TEIMaxBatchTokens = *req.TEIMaxBatchTokens
		}
		if req.TEIMaxConcurrentReqs != nil {
			m.TEIMaxConcurrentReqs = *req.TEIMaxConcurrentReqs
		}
		if req.TEIPooling != nil {
			m.TEIPooling = *req.TEIPooling
		}
		if req.TEIDtype != nil {
			m.TEIDtype = *req.TEIDtype
		}
		if req.TEIExtraArgs != nil {
			m.TEIExtraArgs = *req.TEIExtraArgs
		}
		if req.VLLMQuantization != nil {
			m.VLLMQuantization = *req.VLLMQuantization
		}
		if req.VLLMDtype != nil {
			m.VLLMDtype = *req.VLLMDtype
		}
		if req.VLLMKVCacheDtype != nil {
			m.VLLMKVCacheDtype = *req.VLLMKVCacheDtype
		}
		if req.VLLMMaxNumSeqs != nil {
			m.VLLMMaxNumSeqs = *req.VLLMMaxNumSeqs
		}
		if req.VLLMEnforceEager != nil {
			m.VLLMEnforceEager = *req.VLLMEnforceEager
		}
		if req.VLLMEnablePrefixCaching != nil {
			m.VLLMEnablePrefixCaching = *req.VLLMEnablePrefixCaching
		}
		if req.VLLMEnableChunkedPrefill != nil {
			m.VLLMEnableChunkedPrefill = *req.VLLMEnableChunkedPrefill
		}
		if req.VLLMSwapSpace != nil {
			m.VLLMSwapSpace = *req.VLLMSwapSpace
		}
		if req.VLLMEnableAutoToolChoice != nil {
			m.VLLMEnableAutoToolChoice = *req.VLLMEnableAutoToolChoice
		}
		if req.VLLMToolCallParser != nil {
			m.VLLMToolCallParser = *req.VLLMToolCallParser
		}
		if req.VLLMChatTemplate != nil {
			m.VLLMChatTemplate = *req.VLLMChatTemplate
		}
		if req.SGLangQuantization != nil {
			m.SGLangQuantization = *req.SGLangQuantization
		}
		if req.SGLangAttentionBackend != nil {
			m.SGLangAttentionBackend = *req.SGLangAttentionBackend
		}
		if req.SGLangExtraArgs != nil {
			m.SGLangExtraArgs = *req.SGLangExtraArgs
		}
		if req.LlamaBatchSize != nil {
			m.LlamaBatchSize = *req.LlamaBatchSize
		}
		if req.LlamaUBatchSize != nil {
			m.LlamaUBatchSize = *req.LlamaUBatchSize
		}
		if req.LlamaCacheTypeK != nil {
			m.LlamaCacheTypeK = *req.LlamaCacheTypeK
		}
		if req.LlamaCacheTypeV != nil {
			m.LlamaCacheTypeV = *req.LlamaCacheTypeV
		}
		if req.LlamaMlock != nil {
			m.LlamaMlock = *req.LlamaMlock
		}
		if req.LlamaChatTemplate != nil {
			m.LlamaChatTemplate = *req.LlamaChatTemplate
		}
		if req.SGLangToolCallParser != nil {
			m.SGLangToolCallParser = *req.SGLangToolCallParser
		}
		if req.GPUDevice != nil {
			m.GPUDevice = *req.GPUDevice
		}
		if req.Capabilities != nil {
			// Convert []string to []Capability
			caps := make([]inference.Capability, len(req.Capabilities))
			for i, c := range req.Capabilities {
				caps[i] = inference.Capability(c)
			}
			m.Capabilities = caps
		}
		if req.AutoStart != nil {
			m.AutoStart = *req.AutoStart
		}
	})

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated", "alias": alias})
}

// CreateSavedRequest represents a request to create a saved model configuration directly.
type CreateSavedRequest struct {
	Alias                string   `json:"alias" binding:"required"`
	Provider             string   `json:"provider" binding:"required"`
	Format               string   `json:"format"`
	HFRepo               string   `json:"hf_repo"`
	HFFile               string   `json:"hf_file"`
	HFRevision           string   `json:"hf_revision"`
	GGUFURL              string   `json:"gguf_url"`
	Capabilities         []string `json:"capabilities"`
	GPUDevice            string   `json:"gpu_device"`
	AutoStart            bool     `json:"auto_start"`
	VLLMTensorParallel   int      `json:"vllm_tensor_parallel"`
	VLLMMaxModelLen      int      `json:"vllm_max_model_len"`
	VLLMGPUUtilization   float64  `json:"vllm_gpu_utilization"`
	VLLMExtraArgs        string   `json:"vllm_extra_args"`
	LlamaMainGPU         int      `json:"llama_main_gpu"`
	LlamaTensorSplit     string   `json:"llama_tensor_split"`
	LlamaNGPULayers      int      `json:"llama_n_gpu_layers"`
	LlamaCtxSize         int      `json:"llama_ctx_size"`
	LlamaNParallel       int      `json:"llama_n_parallel"`
	LlamaFlashAttn       bool     `json:"llama_flash_attn"`
	LlamaJinja           bool     `json:"llama_jinja"`
	LlamaCacheReuse      int      `json:"llama_cache_reuse"`
	LlamaExtraArgs       string   `json:"llama_extra_args"`
	SGLangTensorParallel int      `json:"sglang_tensor_parallel"`
	SGLangMemFraction      float64  `json:"sglang_mem_fraction"`
	SGLangQuantization     string   `json:"sglang_quantization"`
	SGLangAttentionBackend string   `json:"sglang_attention_backend"`
	SGLangExtraArgs        string   `json:"sglang_extra_args"`
	TGINumShard            int      `json:"tgi_num_shard"`
	TGIQuantize            string   `json:"tgi_quantize"`
	TGICudaMemoryFraction  float64  `json:"tgi_cuda_memory_fraction"`
	TGIExtraArgs           string   `json:"tgi_extra_args"`
	TEIMaxBatchTokens      int      `json:"tei_max_batch_tokens"`
	TEIMaxConcurrentReqs   int      `json:"tei_max_concurrent_reqs"`
	TEIPooling             string   `json:"tei_pooling"`
	TEIDtype               string   `json:"tei_dtype"`
	TEIExtraArgs           string   `json:"tei_extra_args"`
	VLLMQuantization        string  `json:"vllm_quantization"`
	VLLMDtype               string  `json:"vllm_dtype"`
	VLLMKVCacheDtype        string  `json:"vllm_kv_cache_dtype"`
	VLLMMaxNumSeqs          int     `json:"vllm_max_num_seqs"`
	VLLMEnforceEager        bool    `json:"vllm_enforce_eager"`
	VLLMEnablePrefixCaching bool    `json:"vllm_enable_prefix_caching"`
	VLLMEnableChunkedPrefill bool   `json:"vllm_enable_chunked_prefill"`
	VLLMSwapSpace           int     `json:"vllm_swap_space"`
	VLLMEnableAutoToolChoice bool   `json:"vllm_enable_auto_tool_choice"`
	VLLMToolCallParser       string `json:"vllm_tool_call_parser"`
	VLLMChatTemplate         string `json:"vllm_chat_template"`
	LlamaBatchSize   int    `json:"llama_batch_size"`
	LlamaUBatchSize  int    `json:"llama_ubatch_size"`
	LlamaCacheTypeK  string `json:"llama_cache_type_k"`
	LlamaCacheTypeV  string `json:"llama_cache_type_v"`
	LlamaMlock       bool   `json:"llama_mlock"`
	LlamaChatTemplate string `json:"llama_chat_template"`
	SGLangToolCallParser string `json:"sglang_tool_call_parser"`
}

// PostCreateSaved creates a new saved model configuration directly from form data.
// POST /api/system/inference/create-saved
func (h *InferenceHandler) PostCreateSaved(c *gin.Context) {
	if h.modelStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model store not configured"})
		return
	}

	var req CreateSavedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert capabilities from strings
	var caps []inference.Capability
	for _, c := range req.Capabilities {
		caps = append(caps, inference.Capability(c))
	}

	// Create saved model directly
	saved := inference.SavedModel{
		Alias:                req.Alias,
		Provider:             inference.ProviderKind(req.Provider),
		Format:               inference.ModelFormat(req.Format),
		HFRepo:               req.HFRepo,
		HFFile:               req.HFFile,
		HFRevision:           req.HFRevision,
		GGUFURL:              req.GGUFURL,
		Capabilities:         caps,
		GPUDevice:            req.GPUDevice,
		AutoStart:            req.AutoStart,
		VLLMTensorParallel:   req.VLLMTensorParallel,
		VLLMMaxModelLen:      req.VLLMMaxModelLen,
		VLLMGPUUtilization:   req.VLLMGPUUtilization,
		VLLMExtraArgs:        req.VLLMExtraArgs,
		LlamaMainGPU:         req.LlamaMainGPU,
		LlamaTensorSplit:     req.LlamaTensorSplit,
		LlamaNGPULayers:      req.LlamaNGPULayers,
		LlamaCtxSize:         req.LlamaCtxSize,
		LlamaNParallel:       req.LlamaNParallel,
		LlamaFlashAttn:       req.LlamaFlashAttn,
		LlamaJinja:           req.LlamaJinja,
		LlamaCacheReuse:      req.LlamaCacheReuse,
		LlamaExtraArgs:       req.LlamaExtraArgs,
		SGLangTensorParallel:    req.SGLangTensorParallel,
		SGLangMemFraction:       req.SGLangMemFraction,
		SGLangQuantization:      req.SGLangQuantization,
		SGLangAttentionBackend:  req.SGLangAttentionBackend,
		SGLangExtraArgs:         req.SGLangExtraArgs,
		TGINumShard:             req.TGINumShard,
		TGIQuantize:             req.TGIQuantize,
		TGICudaMemoryFraction:   req.TGICudaMemoryFraction,
		TGIExtraArgs:            req.TGIExtraArgs,
		TEIMaxBatchTokens:       req.TEIMaxBatchTokens,
		TEIMaxConcurrentReqs:    req.TEIMaxConcurrentReqs,
		TEIPooling:              req.TEIPooling,
		TEIDtype:                req.TEIDtype,
		TEIExtraArgs:            req.TEIExtraArgs,
		VLLMQuantization:        req.VLLMQuantization,
		VLLMDtype:               req.VLLMDtype,
		VLLMKVCacheDtype:        req.VLLMKVCacheDtype,
		VLLMMaxNumSeqs:          req.VLLMMaxNumSeqs,
		VLLMEnforceEager:        req.VLLMEnforceEager,
		VLLMEnablePrefixCaching: req.VLLMEnablePrefixCaching,
		VLLMEnableChunkedPrefill: req.VLLMEnableChunkedPrefill,
		VLLMSwapSpace:           req.VLLMSwapSpace,
		VLLMEnableAutoToolChoice: req.VLLMEnableAutoToolChoice,
		VLLMToolCallParser:       req.VLLMToolCallParser,
		VLLMChatTemplate:         req.VLLMChatTemplate,
		LlamaBatchSize:          req.LlamaBatchSize,
		LlamaUBatchSize:         req.LlamaUBatchSize,
		LlamaCacheTypeK:         req.LlamaCacheTypeK,
		LlamaCacheTypeV:         req.LlamaCacheTypeV,
		LlamaMlock:              req.LlamaMlock,
		LlamaChatTemplate:       req.LlamaChatTemplate,
		SGLangToolCallParser:    req.SGLangToolCallParser,
	}

	if err := h.modelStore.Save(saved); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "created", "alias": req.Alias, "auto_start": req.AutoStart})
}

// DockerImageStatus represents the status of a Docker image
type DockerImageStatus struct {
	Provider string `json:"provider"`
	Image    string `json:"image"`
	Exists   bool   `json:"exists"`
	Size     string `json:"size,omitempty"`
	Pulling  bool   `json:"pulling,omitempty"`
}

// GetDockerImages returns the status of Docker images for inference providers
// GET /api/ui/inference/docker-images
func (h *InferenceHandler) GetDockerImages(c *gin.Context) {
	images := []DockerImageStatus{
		{Provider: "vllm", Image: inference.DefaultVLLMImage},
		{Provider: "sglang", Image: inference.DefaultSGLangImage},
		{Provider: "tgi", Image: inference.DefaultTGIImage},
		{Provider: "tei", Image: inference.DefaultTEIImage},
		{Provider: "llama.cpp", Image: inference.DefaultLlamaImage},
		{Provider: "tensorrt-llm", Image: inference.DefaultTRTLLMImage},
	}

	runtime := h.router.GetRuntime()
	if runtime == nil {
		c.JSON(http.StatusOK, gin.H{"images": images, "error": "runtime not available"})
		return
	}

	for i := range images {
		exists, size := runtime.ImageExists(images[i].Image)
		images[i].Exists = exists
		images[i].Size = size
		images[i].Pulling = runtime.IsPulling(images[i].Image)
	}

	c.JSON(http.StatusOK, gin.H{"images": images})
}

// PostPullDockerImage pulls a Docker image
// POST /api/ui/inference/docker-images/pull?image=...
func (h *InferenceHandler) PostPullDockerImage(c *gin.Context) {
	image := c.Query("image")
	if image == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image parameter is required"})
		return
	}

	runtime := h.router.GetRuntime()
	if runtime == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "runtime not available"})
		return
	}

	// Check if already pulling
	if runtime.IsPulling(image) {
		c.JSON(http.StatusConflict, gin.H{"error": "image is already being pulled", "image": image})
		return
	}

	// Pull in background and return immediately
	go func() {
		h.logger.WithField("image", image).Info("Starting Docker image pull")
		startTime := time.Now()
		if err := runtime.PullImage(image); err != nil {
			h.logger.WithError(err).WithFields(map[string]any{
				"image":    image,
				"duration": time.Since(startTime).String(),
			}).Error("Failed to pull Docker image")
		} else {
			h.logger.WithFields(map[string]any{
				"image":    image,
				"duration": time.Since(startTime).String(),
			}).Info("Docker image pulled successfully")
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{"message": "Image pull started", "image": image})
}

// POST /api/system/inference/download-repo
// Downloads files for a HuggingFace repository locally for offline use.
// If filename is specified, downloads only that file. Otherwise downloads all model files.
func (h *InferenceHandler) PostDownloadRepository(c *gin.Context) {
	var req struct {
		ModelID  string `json:"model_id" binding:"required"`
		Filename string `json:"filename"` // Optional: specific file to download
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	downloader := h.router.GetDownloader()
	if downloader == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "downloader not configured"})
		return
	}

	// If specific filename provided, download only that file
	if req.Filename != "" {
		h.logger.WithFields(logrus.Fields{
			"model_id": req.ModelID,
			"filename": req.Filename,
		}).Info("Starting single file download")

		repoDownload, err := downloader.DownloadSingleFile(c.Request.Context(), req.ModelID, req.Filename)
		if err != nil {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"model_id": req.ModelID,
				"filename": req.Filename,
			}).Error("Failed to start file download")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
			"message":     "File download started",
			"model_id":    req.ModelID,
			"filename":    req.Filename,
			"download_id": repoDownload.ID,
			"total_files": repoDownload.TotalFiles,
			"total_size":  repoDownload.TotalSize,
			"local_path":  repoDownload.LocalPath,
		})
		return
	}

	// Download all model files
	h.logger.WithField("model_id", req.ModelID).Info("Starting repository download")

	repoDownload, err := downloader.DownloadRepository(c.Request.Context(), req.ModelID)
	if err != nil {
		h.logger.WithError(err).WithField("model_id", req.ModelID).Error("Failed to start repository download")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "Repository download started",
		"model_id":    req.ModelID,
		"download_id": repoDownload.ID,
		"total_files": repoDownload.TotalFiles,
		"total_size":  repoDownload.TotalSize,
		"local_path":  repoDownload.LocalPath,
	})
}

// GET /api/system/inference/repo-downloads
// Lists all repository downloads with their status.
func (h *InferenceHandler) GetRepoDownloads(c *gin.Context) {
	downloader := h.router.GetDownloader()
	if downloader == nil {
		c.JSON(http.StatusOK, []any{})
		return
	}

	downloads := downloader.ListRepoDownloads()
	c.JSON(http.StatusOK, downloads)
}

// POST /api/system/inference/repo-downloads/status
// Gets status of a specific repository download.
func (h *InferenceHandler) GetRepoDownloadStatus(c *gin.Context) {
	var req struct {
		ModelID string `json:"model_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model_id is required"})
		return
	}

	downloader := h.router.GetDownloader()
	if downloader == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "downloader not configured"})
		return
	}

	repo, found := downloader.GetRepoDownload(req.ModelID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "download not found"})
		return
	}

	c.JSON(http.StatusOK, repo)
}

// POST /api/system/inference/repo-downloads/cancel
// @Summary Cancel repository download
func (h *InferenceHandler) CancelRepoDownload(c *gin.Context) {
	var req struct {
		ModelID string `json:"model_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model_id is required"})
		return
	}

	downloader := h.router.GetDownloader()
	if downloader == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "downloader not configured"})
		return
	}

	if err := downloader.CancelRepoDownload(req.ModelID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "cancelled", "model_id": req.ModelID})
}

// POST /api/system/inference/repo-downloads/remove
// @Summary Remove repository download from list
func (h *InferenceHandler) RemoveRepoDownload(c *gin.Context) {
	var req struct {
		ModelID string `json:"model_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model_id is required"})
		return
	}

	downloader := h.router.GetDownloader()
	if downloader == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "downloader not configured"})
		return
	}

	downloader.RemoveRepoDownload(req.ModelID)
	c.JSON(http.StatusOK, gin.H{"status": "removed", "model_id": req.ModelID})
}

// POST /api/system/inference/refresh-saved
// Refreshes a saved model by re-downloading missing or outdated files.
// This is useful when model files are corrupted or incomplete.
func (h *InferenceHandler) PostRefreshSaved(c *gin.Context) {
	alias := c.Query("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias parameter required"})
		return
	}

	if h.modelStore == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model store not configured"})
		return
	}

	// Get saved model config
	saved, ok := h.modelStore.Get(alias)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "saved model not found"})
		return
	}

	// Check if this is an HF model
	if saved.HFRepo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model has no HuggingFace repository configured, cannot refresh"})
		return
	}

	downloader := h.router.GetDownloader()
	if downloader == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "downloader not configured"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"alias":   alias,
		"hf_repo": saved.HFRepo,
	}).Info("Starting model refresh/re-download")

	// Use DownloadRepository which will check existing files and download missing ones
	repoDownload, err := downloader.DownloadRepository(c.Request.Context(), saved.HFRepo)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"alias":   alias,
			"hf_repo": saved.HFRepo,
		}).Error("Failed to start model refresh")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "Model refresh started",
		"alias":       alias,
		"model_id":    saved.HFRepo,
		"download_id": repoDownload.ID,
		"total_files": repoDownload.TotalFiles,
		"total_size":  repoDownload.TotalSize,
		"local_path":  repoDownload.LocalPath,
	})
}

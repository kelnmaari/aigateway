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
	router *inference.Router
	logger *logrus.Logger
}

// NewInferenceHandler constructs handler.
func NewInferenceHandler(router *inference.Router, logger *logrus.Logger) *InferenceHandler {
	return &InferenceHandler{
		router: router,
		logger: logger,
	}
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

	// vLLM options
	VLLMTensorParallel int     `json:"vllm_tensor_parallel"`
	VLLMMaxModelLen    int     `json:"vllm_max_model_len"`
	VLLMGPUUtilization float64 `json:"vllm_gpu_utilization"`

	// llama.cpp options
	LlamaMainGPU     int    `json:"llama_main_gpu"`
	LlamaTensorSplit string `json:"llama_tensor_split"`
	LlamaNGPULayers  int    `json:"llama_n_gpu_layers"`

	// SGLang options
	SGLangTensorParallel int     `json:"sglang_tensor_parallel"`
	SGLangDataParallel   int     `json:"sglang_data_parallel"`
	SGLangMemFraction    float64 `json:"sglang_mem_fraction"`
	SGLangContextLen     int     `json:"sglang_context_len"`
	SGLangChunkedPrefill bool    `json:"sglang_chunked_prefill"`

	// TGI options
	TGINumShard          int `json:"tgi_num_shard"`
	TGIMaxConcurrentReqs int `json:"tgi_max_concurrent_reqs"`
	TGIMaxInputLen       int `json:"tgi_max_input_len"`
	TGIMaxTotalTokens    int `json:"tgi_max_total_tokens"`
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
}

// PostLoad starts container after ensuring artifacts.
func (h *InferenceHandler) PostLoad(c *gin.Context) {
	var req LoadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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
		VLLMTensorParallel: req.VLLMTensorParallel,
		VLLMMaxModelLen:    req.VLLMMaxModelLen,
		VLLMGPUUtilization: req.VLLMGPUUtilization,
		LlamaMainGPU:       req.LlamaMainGPU,
		LlamaTensorSplit:   req.LlamaTensorSplit,
		LlamaNGPULayers:    req.LlamaNGPULayers,
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
	}

	inst, err := h.router.EnsureByAlias(c.Request.Context(), req.Alias)
	if err != nil {
		// If alias not registered, try with provided spec
		inst, err = h.router.EnsureBySpec(c.Request.Context(), spec)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
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

// GetHealth performs health check by alias.
func (h *InferenceHandler) GetHealth(c *gin.Context) {
	alias := c.Query("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}
	if err := h.router.Health(c.Request.Context(), alias); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		}
		if m.Handle != nil {
			item.Endpoint = m.Handle.Endpoint
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(metrics))
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
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "TRT converter not configured"})
		return
	}

	engines, err := converter.ListEngines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"engines": engines})
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

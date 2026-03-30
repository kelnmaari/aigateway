package handlers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/inference"
)

// InferenceProxyHandler proxies OpenAI-compatible requests to running inference providers.
type InferenceProxyHandler struct {
	router *inference.Router
	logger *logrus.Logger
	client *http.Client
}

// NewInferenceProxyHandler constructs handler.
func NewInferenceProxyHandler(router *inference.Router, logger *logrus.Logger) *InferenceProxyHandler {
	return &InferenceProxyHandler{
		router: router,
		logger: logger,
		client: &http.Client{
			Timeout: 15 * time.Minute, // long timeout for generation
		},
	}
}

// HandleChatCompletions proxies /v1/chat/completions to provider container.
// Automatically converts Anthropic-format tool messages (tool_use / tool_result content blocks)
// to OpenAI format before forwarding.  Purely OpenAI-format requests pass through unchanged.
func (h *InferenceProxyHandler) HandleChatCompletions(c *gin.Context) {
	// Read raw body so we can (a) preserve ALL original fields and (b) normalize message format
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "failed to read request body")
		return
	}

	// Parse only the routing-relevant fields
	var req struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}
	if err := json.Unmarshal(rawBody, &req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	if req.Model == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	// Resolve model alias and ensure container is running
	inst, err := h.router.EnsureByAlias(c.Request.Context(), req.Model)
	if err != nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "model_not_available", err.Error())
		return
	}

	if inst.Handle == nil || inst.Handle.Endpoint == "" {
		h.errorResponse(c, http.StatusServiceUnavailable, "model_not_available", "model container not running")
		return
	}

	endpoint := inst.Handle.Endpoint + "/v1/chat/completions"

	// Rewrite model name to what provider expects
	// vLLM/TGI/SGLang expect HFRepo (e.g., "Qwen/Qwen2.5-3B-Instruct"), not alias
	providerModel := h.resolveProviderModelName(inst)

	h.logger.WithFields(logrus.Fields{
		"model":          req.Model,
		"alias":          inst.Spec.Alias,
		"provider":       inst.Spec.Provider,
		"provider_model": providerModel,
		"endpoint":       endpoint,
		"stream":         req.Stream,
	}).Debug("proxying chat completion request")

	// Normalize Anthropic-format tool messages → OpenAI format (no-op for OpenAI clients)
	normalized := normalizeMessagesInJSON(rawBody)
	// Rewrite model field, preserving all other fields
	bodyBytes := rewriteModelInJSON(normalized, providerModel)

	proxyReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(proxyReq)
	if err != nil {
		h.logger.WithError(err).Error("proxy request failed")
		h.errorResponse(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	defer resp.Body.Close()

	// Handle streaming response
	if req.Stream {
		h.streamResponse(c, resp)
		return
	}

	// Non-streaming: forward response as-is
	c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
}

// HandleCompletions proxies /v1/completions (legacy) to provider container.
func (h *InferenceProxyHandler) HandleCompletions(c *gin.Context) {
	var req struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		Stream bool   `json:"stream"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	if req.Model == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	inst, err := h.router.EnsureByAlias(c.Request.Context(), req.Model)
	if err != nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "model_not_available", err.Error())
		return
	}

	if inst.Handle == nil || inst.Handle.Endpoint == "" {
		h.errorResponse(c, http.StatusServiceUnavailable, "model_not_available", "model container not running")
		return
	}

	endpoint := inst.Handle.Endpoint + "/v1/completions"

	// Rewrite model name to what provider expects
	req.Model = h.resolveProviderModelName(inst)

	bodyBytes, _ := json.Marshal(req)
	proxyReq, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	proxyReq.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(proxyReq)
	if err != nil {
		h.errorResponse(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	defer resp.Body.Close()

	if req.Stream {
		h.streamResponse(c, resp)
		return
	}

	c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
}

// HandleModels returns list of available models from inference registry.
func (h *InferenceProxyHandler) HandleModels(c *gin.Context) {
	models := h.router.ListModels()

	type modelEntry struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}

	data := make([]modelEntry, 0, len(models))
	for _, m := range models {
		if m.Status == inference.StatusRunning || m.Status == inference.StatusReady {
			data = append(data, modelEntry{
				ID:      m.Spec.Alias,
				Object:  "model",
				Created: m.LastUsed.Unix(),
				OwnedBy: string(m.Spec.Provider),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

// HandlePassthrough proxies any OpenAI-compatible request to the inference provider container.
// Supports both JSON and multipart/form-data (audio, images, embeddings, rerank, etc.).
// Body must be pre-loaded into c.Request.Body by the caller (unified handler).
func (h *InferenceProxyHandler) HandlePassthrough(c *gin.Context, path string) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "failed to read request body")
		return
	}

	model := ExtractModelFromRequest(bodyBytes, c.GetHeader("Content-Type"))
	if model == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	inst, err := h.router.EnsureByAlias(c.Request.Context(), model)
	if err != nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "model_not_available", err.Error())
		return
	}

	if inst.Handle == nil || inst.Handle.Endpoint == "" {
		h.errorResponse(c, http.StatusServiceUnavailable, "model_not_available", "model container not running")
		return
	}

	// TEI provider requires path and format translation for rerank
	if inst.Spec.Provider == inference.ProviderTEI && path == "/v1/rerank" {
		h.handleTEIRerank(c, inst, model, bodyBytes)
		return
	}

	// TEI provider: map /v1/embeddings to /v1/embeddings (already OpenAI-compatible)
	// For other paths, apply provider-specific path mapping
	actualPath := h.mapProviderPath(inst.Spec.Provider, path)
	endpoint := inst.Handle.Endpoint + actualPath

	h.logger.WithFields(logrus.Fields{
		"model":       model,
		"alias":       inst.Spec.Alias,
		"provider":    inst.Spec.Provider,
		"endpoint":    endpoint,
		"path":        path,
		"actual_path": actualPath,
	}).Debug("proxying passthrough request")

	proxyReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	// Preserve original Content-Type (important for multipart/form-data with boundary)
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	proxyReq.Header.Set("Content-Type", contentType)

	resp, err := h.client.Do(proxyReq)
	if err != nil {
		h.logger.WithError(err).WithField("path", path).Error("passthrough proxy request failed")
		h.errorResponse(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	defer resp.Body.Close()

	c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
}

// mapProviderPath translates OpenAI-compatible paths to provider-specific paths.
// Most providers support OpenAI paths natively; TEI is the exception for some endpoints.
func (h *InferenceProxyHandler) mapProviderPath(provider inference.ProviderKind, path string) string {
	if provider == inference.ProviderTEI {
		switch path {
		case "/v1/embeddings":
			return "/v1/embeddings" // TEI supports OpenAI-compatible embeddings
		default:
			return path
		}
	}
	return path
}

// handleTEIRerank translates OpenAI-compatible rerank requests to TEI format and back.
//
// OpenAI/Dify format:
//
//	Request:  {"model":"...", "query":"...", "documents":["..."], "top_n":3}
//	Response: {"id":"...", "object":"rerank", "model":"...", "results":[{"index":0, "relevance_score":0.95}]}
//
// TEI format:
//
//	Request:  {"query":"...", "texts":["..."], "return_text":false}
//	Response: [{"index":0, "score":0.95}]
func (h *InferenceProxyHandler) handleTEIRerank(c *gin.Context, inst *inference.ModelInstance, model string, bodyBytes []byte) {
	// Parse OpenAI-compatible rerank request
	var openAIReq struct {
		Model           string   `json:"model"`
		Query           string   `json:"query"`
		Documents       []string `json:"documents"`
		TopN            *int     `json:"top_n,omitempty"`
		ReturnDocuments *bool    `json:"return_documents,omitempty"`
	}
	if err := json.Unmarshal(bodyBytes, &openAIReq); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "invalid rerank request: "+err.Error())
		return
	}

	if openAIReq.Query == "" || len(openAIReq.Documents) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "query and documents are required")
		return
	}

	// Build TEI rerank request
	teiReq := map[string]any{
		"query":       openAIReq.Query,
		"texts":       openAIReq.Documents,
		"return_text": false,
		"raw_scores":  false,
	}

	teiBody, err := json.Marshal(teiReq)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "internal_error", "failed to build TEI request")
		return
	}

	endpoint := inst.Handle.Endpoint + "/rerank"

	h.logger.WithFields(logrus.Fields{
		"model":    model,
		"alias":    inst.Spec.Alias,
		"provider": inst.Spec.Provider,
		"endpoint": endpoint,
		"docs":     len(openAIReq.Documents),
	}).Debug("proxying rerank to TEI (format translation)")

	proxyReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, bytes.NewReader(teiBody))
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(proxyReq)
	if err != nil {
		h.logger.WithError(err).Error("TEI rerank request failed")
		h.errorResponse(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		h.errorResponse(c, http.StatusBadGateway, "upstream_error", "failed to read TEI response")
		return
	}

	// If TEI returned an error, forward it
	if resp.StatusCode != http.StatusOK {
		h.logger.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(respBytes),
		}).Warn("TEI rerank returned non-200")
		c.Data(resp.StatusCode, "application/json", respBytes)
		return
	}

	// Parse TEI response: [{"index":0, "score":0.95}, ...]
	var teiResults []struct {
		Index int     `json:"index"`
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal(respBytes, &teiResults); err != nil {
		h.logger.WithError(err).WithField("body", string(respBytes)).Error("failed to parse TEI rerank response")
		h.errorResponse(c, http.StatusBadGateway, "upstream_error", "failed to parse TEI response")
		return
	}

	// Apply top_n filter
	results := teiResults
	if openAIReq.TopN != nil && *openAIReq.TopN > 0 && *openAIReq.TopN < len(results) {
		results = results[:*openAIReq.TopN]
	}

	// Convert to OpenAI-compatible rerank response
	openAIResults := make([]gin.H, 0, len(results))
	for _, r := range results {
		item := gin.H{
			"index":           r.Index,
			"relevance_score": r.Score,
		}
		// Include document text if requested
		if openAIReq.ReturnDocuments != nil && *openAIReq.ReturnDocuments && r.Index < len(openAIReq.Documents) {
			item["document"] = gin.H{
				"text": openAIReq.Documents[r.Index],
			}
		}
		openAIResults = append(openAIResults, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      "rerank-" + model,
		"object":  "rerank",
		"model":   model,
		"results": openAIResults,
		"usage": gin.H{
			"prompt_tokens": 0,
			"total_tokens":  0,
		},
	})
}

func (h *InferenceProxyHandler) streamResponse(c *gin.Context, resp *http.Response) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Status(resp.StatusCode)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.logger.Error("streaming not supported")
		return
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				h.logger.WithError(err).Error("stream read error")
			}
			break
		}
		_, _ = c.Writer.Write(line)
		flusher.Flush()

		// Check for [DONE] marker
		if strings.Contains(string(line), "[DONE]") {
			break
		}
	}
}

func (h *InferenceProxyHandler) errorResponse(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    errType,
		},
	})
}

// resolveProviderModelName returns the model name that the provider expects.
// vLLM uses --served-model-name (alias), TGI/SGLang expect HFRepo,
// llama.cpp expects the alias.
func (h *InferenceProxyHandler) resolveProviderModelName(inst *inference.ModelInstance) string {
	switch inst.Spec.Provider {
	case inference.ProviderVLLM:
		// vLLM uses --served-model-name which we set to alias
		return inst.Spec.Alias
	case inference.ProviderTGI, inference.ProviderSGLang, inference.ProviderTRTLLM:
		// HF-based providers expect the HF repo name
		if inst.Spec.HFRepo != "" {
			return inst.Spec.HFRepo
		}
		// Fallback to alias if no HFRepo
		return inst.Spec.Alias
	case inference.ProviderLlamaCPP:
		// llama.cpp server uses --alias flag, so alias works
		return inst.Spec.Alias
	default:
		return inst.Spec.Alias
	}
}

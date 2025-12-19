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
	"aigateway/internal/models"
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
			Timeout: 5 * time.Minute, // long timeout for generation
		},
	}
}

// HandleChatCompletions proxies /v1/chat/completions to provider container.
func (h *InferenceProxyHandler) HandleChatCompletions(c *gin.Context) {
	var req models.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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

	// Rewrite model in request to match provider's expected name
	req.Model = providerModel

	// Re-encode request body
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "internal_error", "failed to encode request")
		return
	}

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

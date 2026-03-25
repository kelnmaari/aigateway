package handlers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/providers/converter"
	"aigateway/internal/storage"
)

// ExternalProxyHandler proxies requests to external AI providers
// (OpenAI, Anthropic, Gemini) with format conversion where needed.
type ExternalProxyHandler struct {
	db     storage.Database
	logger *logrus.Logger
	client *http.Client
}

// NewExternalProxyHandler creates a new external proxy handler.
func NewExternalProxyHandler(db storage.Database, logger *logrus.Logger) *ExternalProxyHandler {
	return &ExternalProxyHandler{
		db:     db,
		logger: logger,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

// HandleChatCompletions handles /v1/chat/completions by routing to the appropriate external provider.
func (h *ExternalProxyHandler) HandleChatCompletions(c *gin.Context) {
	var req models.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	if req.Model == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	// Look up model in registry
	modelEntry, err := h.db.GetModelRegistryByModelID(c.Request.Context(), req.Model)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "model_not_found", fmt.Sprintf("model '%s' not found in registry", req.Model))
		return
	}

	// Get the provider
	provider, err := h.db.GetModelProvider(c.Request.Context(), modelEntry.ProviderID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "provider_not_found", fmt.Sprintf("provider '%s' not found", modelEntry.ProviderID))
		return
	}

	if !provider.Enabled {
		h.errorResponse(c, http.StatusServiceUnavailable, "provider_disabled", fmt.Sprintf("provider '%s' is disabled", provider.Name))
		return
	}

	h.logger.WithFields(logrus.Fields{
		"model":         req.Model,
		"provider":      provider.Name,
		"provider_type": provider.ProviderType,
		"stream":        req.Stream,
	}).Debug("routing to external provider")

	switch provider.ProviderType {
	case models.ProviderTypeOpenAI, models.ProviderTypeCustom, models.ProviderTypeVLLM, models.ProviderTypeDeepSeek:
		h.proxyOpenAICompatible(c, &req, provider)
	case models.ProviderTypeAnthropic:
		h.proxyAnthropic(c, &req, provider)
	case models.ProviderTypeGemini:
		h.proxyGemini(c, &req, provider)
	default:
		h.errorResponse(c, http.StatusBadRequest, "unsupported_provider", fmt.Sprintf("provider type '%s' not supported", provider.ProviderType))
	}
}

// proxyOpenAICompatible forwards request to an OpenAI-compatible endpoint.
func (h *ExternalProxyHandler) proxyOpenAICompatible(c *gin.Context, req *models.ChatCompletionRequest, provider *models.ModelProvider) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "internal_error", "failed to encode request")
		return
	}

	endpoint := strings.TrimRight(provider.BaseURL, "/") + "/v1/chat/completions"
	proxyReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")
	if provider.APIKey != "" {
		proxyReq.Header.Set("Authorization", "Bearer "+provider.APIKey)
	}

	resp, err := h.client.Do(proxyReq)
	if err != nil {
		h.logger.WithError(err).WithField("provider", provider.Name).Error("external proxy request failed")
		h.errorResponse(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	defer resp.Body.Close()

	if req.Stream {
		h.streamPassthrough(c, resp)
		return
	}

	c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
}

// proxyAnthropic converts request to Anthropic format, proxies, then converts back.
func (h *ExternalProxyHandler) proxyAnthropic(c *gin.Context, req *models.ChatCompletionRequest, provider *models.ModelProvider) {
	anthropicReq, err := converter.OpenAIToAnthropic(req, req.Model)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "conversion_error", err.Error())
		return
	}

	bodyBytes, err := json.Marshal(anthropicReq)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "internal_error", "failed to encode anthropic request")
		return
	}

	endpoint := strings.TrimRight(provider.BaseURL, "/") + "/v1/messages"
	proxyReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("x-api-key", provider.APIKey)
	proxyReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := h.client.Do(proxyReq)
	if err != nil {
		h.logger.WithError(err).WithField("provider", provider.Name).Error("anthropic proxy request failed")
		h.errorResponse(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		h.errorResponse(c, http.StatusBadGateway, "upstream_error", fmt.Sprintf("anthropic API error %d: %s", resp.StatusCode, string(body)))
		return
	}

	if req.Stream {
		h.streamAnthropicToOpenAI(c, resp, req.Model)
		return
	}

	// Non-streaming: convert response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "internal_error", "failed to read anthropic response")
		return
	}

	var anthropicResp converter.AnthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "conversion_error", "failed to parse anthropic response")
		return
	}

	openaiResp := converter.AnthropicToOpenAI(&anthropicResp, req.Model)
	c.JSON(http.StatusOK, openaiResp)
}

// proxyGemini converts request to Gemini format, proxies, then converts back.
func (h *ExternalProxyHandler) proxyGemini(c *gin.Context, req *models.ChatCompletionRequest, provider *models.ModelProvider) {
	geminiReq, err := converter.OpenAIToGemini(req)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "conversion_error", err.Error())
		return
	}

	bodyBytes, err := json.Marshal(geminiReq)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "internal_error", "failed to encode gemini request")
		return
	}

	// Gemini uses model in the URL path
	action := "generateContent"
	if req.Stream {
		action = "streamGenerateContent?alt=sse"
	}
	endpoint := fmt.Sprintf("%s/v1beta/models/%s:%s?key=%s",
		strings.TrimRight(provider.BaseURL, "/"),
		req.Model,
		action,
		provider.APIKey,
	)

	proxyReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(proxyReq)
	if err != nil {
		h.logger.WithError(err).WithField("provider", provider.Name).Error("gemini proxy request failed")
		h.errorResponse(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		h.errorResponse(c, http.StatusBadGateway, "upstream_error", fmt.Sprintf("gemini API error %d: %s", resp.StatusCode, string(body)))
		return
	}

	if req.Stream {
		h.streamGeminiToOpenAI(c, resp, req.Model)
		return
	}

	// Non-streaming: convert response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "internal_error", "failed to read gemini response")
		return
	}

	var geminiResp converter.GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "conversion_error", "failed to parse gemini response")
		return
	}

	openaiResp := converter.GeminiToOpenAI(&geminiResp, req.Model)
	c.JSON(http.StatusOK, openaiResp)
}

// streamPassthrough passes SSE stream through unchanged (for OpenAI-compatible providers).
func (h *ExternalProxyHandler) streamPassthrough(c *gin.Context, resp *http.Response) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
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

		if strings.Contains(string(line), "[DONE]") {
			break
		}
	}
}

// streamAnthropicToOpenAI converts Anthropic SSE stream to OpenAI format.
func (h *ExternalProxyHandler) streamAnthropicToOpenAI(c *gin.Context, resp *http.Response, requestModel string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.logger.Error("streaming not supported")
		return
	}

	streamID := fmt.Sprintf("chatcmpl-anthropic-%d", time.Now().UnixNano())
	scanner := bufio.NewScanner(resp.Body)

	var currentEventType string
	for scanner.Scan() {
		line := scanner.Text()

		if after, ok0 := strings.CutPrefix(line, "event: "); ok0 {
			currentEventType = after
			continue
		}

		if after, ok0 := strings.CutPrefix(line, "data: "); ok0 {
			data := after

			openaiData, isDone, err := converter.ConvertAnthropicSSE(currentEventType, []byte(data), requestModel, streamID)
			if err != nil {
				h.logger.WithError(err).Error("anthropic SSE conversion error")
				continue
			}

			if openaiData == "" {
				continue
			}

			fmt.Fprintf(c.Writer, "data: %s\n\n", openaiData)
			flusher.Flush()

			if isDone {
				fmt.Fprint(c.Writer, "data: [DONE]\n\n")
				flusher.Flush()
				break
			}
		}
	}
}

// streamGeminiToOpenAI converts Gemini SSE stream to OpenAI format.
func (h *ExternalProxyHandler) streamGeminiToOpenAI(c *gin.Context, resp *http.Response, requestModel string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.logger.Error("streaming not supported")
		return
	}

	streamID := fmt.Sprintf("chatcmpl-gemini-%d", time.Now().UnixNano())
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		openaiData, isDone, err := converter.ConvertGeminiSSE([]byte(data), requestModel, streamID)
		if err != nil {
			h.logger.WithError(err).Error("gemini SSE conversion error")
			continue
		}

		if openaiData == "" {
			continue
		}

		fmt.Fprintf(c.Writer, "data: %s\n\n", openaiData)
		flusher.Flush()

		if isDone {
			fmt.Fprint(c.Writer, "data: [DONE]\n\n")
			flusher.Flush()
			break
		}
	}
}

// HandlePassthrough forwards the request body as-is to the provider's endpoint path.
// Supports both JSON and multipart/form-data requests (audio, images, embeddings, rerank, etc.).
// Body must be pre-loaded into c.Request.Body by the caller (unified handler).
func (h *ExternalProxyHandler) HandlePassthrough(c *gin.Context, provider *models.ModelProvider, path string) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "failed to read request body")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"provider":      provider.Name,
		"provider_type": provider.ProviderType,
		"path":          path,
	}).Debug("proxying request to external provider")

	endpoint := strings.TrimRight(provider.BaseURL, "/") + path
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

	if provider.APIKey != "" {
		proxyReq.Header.Set("Authorization", "Bearer "+provider.APIKey)
	}

	resp, err := h.client.Do(proxyReq)
	if err != nil {
		h.logger.WithError(err).WithField("provider", provider.Name).Error("external proxy request failed")
		h.errorResponse(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	defer resp.Body.Close()

	c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
}

func (h *ExternalProxyHandler) errorResponse(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    errType,
		},
	})
}

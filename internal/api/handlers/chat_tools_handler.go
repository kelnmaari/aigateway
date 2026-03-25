package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/inference"
	"aigateway/internal/tools"
)

// ChatToolsHandler handles chat completions with tool support.
type ChatToolsHandler struct {
	router   *inference.Router
	toolsReg *tools.Registry
	logger   *logrus.Logger
	client   *http.Client
}

// NewChatToolsHandler creates a new handler.
func NewChatToolsHandler(router *inference.Router, toolsReg *tools.Registry, logger *logrus.Logger) *ChatToolsHandler {
	return &ChatToolsHandler{
		router:   router,
		toolsReg: toolsReg,
		logger:   logger,
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// ChatRequest represents a chat request with optional tool support.
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	UseTools    bool          `json:"use_tools,omitempty"` // Enable web search and other tools
}

// ChatMessage represents a message in the conversation.
type ChatMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content"` // string or []ContentPart for multimodal; must not use omitempty to preserve null for tool-calling messages
	ToolCalls  []tools.ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

// HandleChatWithTools handles chat completions with tool calling support.
// POST /api/chat/completions
func (h *ChatToolsHandler) HandleChatWithTools(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if req.Model == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request", "model is required")
		return
	}

	// Resolve model and ensure it's running
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
	providerModel := h.resolveProviderModelName(inst)

	// If tools not requested or not available, forward as-is
	if !req.UseTools || h.toolsReg == nil || !h.toolsReg.HasTavily() {
		h.forwardRequest(c, endpoint, providerModel, req)
		return
	}

	// Handle with tool support
	h.handleWithTools(c, endpoint, providerModel, req, inst)
}

func (h *ChatToolsHandler) handleWithTools(c *gin.Context, endpoint, providerModel string, req ChatRequest, inst *inference.ModelInstance) {
	// Setup streaming
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Status(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.logger.Error("streaming not supported")
		return
	}

	// Agentic loop: call LLM, check for tool calls, execute, repeat
	messages := req.Messages
	toolDefs := h.toolsReg.GetToolDefinitions()

	for range 5 { // Max 5 tool iterations
		// Build LLM request with tools
		llmReq := map[string]any{
			"model":    providerModel,
			"messages": messages,
			"stream":   false, // Non-streaming for tool detection
			"tools":    toolDefs,
		}
		if req.Temperature > 0 {
			llmReq["temperature"] = req.Temperature
		}
		if req.MaxTokens > 0 {
			llmReq["max_tokens"] = req.MaxTokens
		}

		// Call LLM (non-streaming to get tool calls)
		bodyBytes, _ := json.Marshal(llmReq)
		llmResp, err := h.callLLM(c.Request.Context(), endpoint, bodyBytes)
		if err != nil {
			h.sendToolEvent(c, flusher, tools.ToolEvent{Type: "error", Tool: "llm", Query: err.Error()})
			fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
			flusher.Flush()
			return
		}

		// Check if response has tool calls
		choice := llmResp.Choices[0]
		if len(choice.Message.ToolCalls) == 0 {
			// No tool calls - stream the final response
			h.streamFinalResponse(c, flusher, endpoint, providerModel, messages, req)
			return
		}

		// Process tool calls
		assistantMsg := ChatMessage{
			Role:      "assistant",
			Content:   choice.Message.Content,
			ToolCalls: choice.Message.ToolCalls,
		}
		messages = append(messages, assistantMsg)

		for _, tc := range choice.Message.ToolCalls {
			// Send tool start event
			startEvent := tools.ToolEvent{
				Type:  "tool_start",
				Tool:  tc.Function.Name,
				Query: h.extractQueryFromArgs(tc.Function.Arguments),
			}
			h.sendToolEvent(c, flusher, startEvent)

			// Execute tool
			startTime := time.Now()
			result, endEvent, err := h.toolsReg.ExecuteTool(c.Request.Context(), tc)
			elapsed := time.Since(startTime).Milliseconds()

			if err != nil {
				h.sendToolEvent(c, flusher, tools.ToolEvent{
					Type:    "tool_end",
					Tool:    tc.Function.Name,
					Query:   startEvent.Query,
					Result:  "Error: " + err.Error(),
					Elapsed: elapsed,
				})
				messages = append(messages, ChatMessage{
					Role:       "tool",
					Content:    "Error: " + err.Error(),
					ToolCallID: tc.ID,
				})
				continue
			}

			// Send tool end event
			endEvent.Elapsed = elapsed
			h.sendToolEvent(c, flusher, *endEvent)

			// Add tool result to messages
			messages = append(messages, ChatMessage{
				Role:       "tool",
				Content:    result.Content,
				ToolCallID: tc.ID,
			})
		}
	}

	// Max iterations reached - stream final response
	h.streamFinalResponse(c, flusher, endpoint, providerModel, messages, req)
}

type llmResponse struct {
	Choices []struct {
		Message struct {
			Role      string           `json:"role"`
			Content   any              `json:"content"`
			ToolCalls []tools.ToolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

func (h *ChatToolsHandler) callLLM(ctx context.Context, endpoint string, body []byte) (*llmResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM error (status %d): %s", resp.StatusCode, string(body))
	}

	var result llmResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &result, nil
}

func (h *ChatToolsHandler) streamFinalResponse(c *gin.Context, flusher http.Flusher, endpoint, providerModel string, messages []ChatMessage, origReq ChatRequest) {
	// Build streaming request (without tools for final response)
	llmReq := map[string]any{
		"model":    providerModel,
		"messages": messages,
		"stream":   true,
	}
	if origReq.Temperature > 0 {
		llmReq["temperature"] = origReq.Temperature
	}
	if origReq.MaxTokens > 0 {
		llmReq["max_tokens"] = origReq.MaxTokens
	}

	bodyBytes, _ := json.Marshal(llmReq)
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		h.logger.WithError(err).Error("create request failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.WithError(err).Error("LLM request failed")
		h.sendToolEvent(c, flusher, tools.ToolEvent{Type: "error", Tool: "llm", Query: fmt.Sprintf("LLM request failed: %v", err)})
		fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}
	defer resp.Body.Close()

	// Check for non-200 response (LLM may reject tool messages without tool definitions)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		h.logger.WithField("status", resp.StatusCode).WithField("body", string(body)).Error("LLM streaming request failed")
		h.sendToolEvent(c, flusher, tools.ToolEvent{Type: "error", Tool: "llm", Query: fmt.Sprintf("LLM error (status %d)", resp.StatusCode)})
		fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}

	// Stream response
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

func (h *ChatToolsHandler) sendToolEvent(c *gin.Context, flusher http.Flusher, event tools.ToolEvent) {
	eventBytes, _ := json.Marshal(event)
	// Send as SSE with special "tool" event type
	fmt.Fprintf(c.Writer, "event: tool\ndata: %s\n\n", string(eventBytes))
	flusher.Flush()
}

func (h *ChatToolsHandler) forwardRequest(c *gin.Context, endpoint, providerModel string, req ChatRequest) {
	// Forward as-is to the LLM
	llmReq := map[string]any{
		"model":    providerModel,
		"messages": req.Messages,
		"stream":   req.Stream,
	}
	if req.Temperature > 0 {
		llmReq["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		llmReq["max_tokens"] = req.MaxTokens
	}

	bodyBytes, _ := json.Marshal(llmReq)
	proxyReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(proxyReq)
	if err != nil {
		h.errorResponse(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	defer resp.Body.Close()

	if req.Stream {
		h.streamForward(c, resp)
		return
	}

	c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
}

func (h *ChatToolsHandler) streamForward(c *gin.Context, resp *http.Response) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(resp.StatusCode)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			break
		}
		_, _ = c.Writer.Write(line)
		flusher.Flush()

		if strings.Contains(string(line), "[DONE]") {
			break
		}
	}
}

func (h *ChatToolsHandler) extractQueryFromArgs(args string) string {
	var parsed struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(args), &parsed); err != nil {
		return args
	}
	return parsed.Query
}

func (h *ChatToolsHandler) errorResponse(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    errType,
		},
	})
}

func (h *ChatToolsHandler) resolveProviderModelName(inst *inference.ModelInstance) string {
	switch inst.Spec.Provider {
	case inference.ProviderVLLM:
		return inst.Spec.Alias
	case inference.ProviderTGI, inference.ProviderSGLang, inference.ProviderTRTLLM:
		if inst.Spec.HFRepo != "" {
			return inst.Spec.HFRepo
		}
		return inst.Spec.Alias
	case inference.ProviderLlamaCPP:
		return inst.Spec.Alias
	default:
		return inst.Spec.Alias
	}
}

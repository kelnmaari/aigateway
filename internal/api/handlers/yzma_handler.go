// Package handlers provides HTTP handlers for yzma local inference
// Version: v3.0.0 - YZMA-02: OpenAI-compatible inference API
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"aigateway/internal/models"
	agentService "aigateway/internal/services/agent"
	"aigateway/internal/yzma"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ModelListWorker interface for background model list caching (v3.0.6+)
type ModelListWorker interface {
	GetCachedModelList(ctx context.Context) (interface{}, error)
}

// YzmaHandler handles yzma inference requests with OpenAI compatibility
type YzmaHandler struct {
	client         *yzma.Client
	agentService   *agentService.AgentService // v2.5.0+, v3.0.6+: Agent support
	modelWorker    ModelListWorker            // v3.0.6+: Background model list cache
	requestTimeout time.Duration              // v3.0.9+: Configurable request timeout
	logger         *logrus.Logger
}

// NewYzmaHandler creates a new yzma handler
func NewYzmaHandler(client *yzma.Client, logger *logrus.Logger) *YzmaHandler {
	return &YzmaHandler{
		client:         client,
		agentService:   nil, // Will be set later if agent service is available
		requestTimeout: 30 * time.Minute, // Default: 30 minutes (thinking models need more time)
		logger:         logger,
	}
}

// SetRequestTimeout sets the maximum request timeout (0 = no timeout)
func (h *YzmaHandler) SetRequestTimeout(timeout time.Duration) {
	h.requestTimeout = timeout
	h.logger.WithField("timeout", timeout).Info("Yzma request timeout configured")
}

// SetModelWorker sets the model list worker for background caching (v3.0.6+)
func (h *YzmaHandler) SetModelWorker(worker ModelListWorker) {
	h.modelWorker = worker
	h.logger.Info("✅ Model list background caching enabled")
}

// SetAgentService sets the agent service for agent mode support (v2.5.0+, v3.0.6+)
func (h *YzmaHandler) SetAgentService(service *agentService.AgentService) {
	h.agentService = service
	h.logger.Info("Agent service enabled for YZMA handler")
}

// HandleChatCompletion handles /v1/chat/completions (OpenAI compatible)
func (h *YzmaHandler) HandleChatCompletion(c *gin.Context) {
	var req models.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"message": fmt.Sprintf("Invalid request: %v", err),
				"type":    "invalid_request_error",
			},
		})
		return
	}
	
	// Check if agent mode is enabled (v2.5.0+, v3.0.6+: Agent support)
	if req.AgentMode {
		if h.agentService == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"message": "Agent mode is not enabled on this server",
					"type":    "invalid_request_error",
				},
			})
			return
		}
		
		h.handleAgentCompletion(c, req)
		return
	}
	
	// Check if messages contain images (VLM-03: v3.0.4+)
	if h.containsImages(req.Messages) {
		h.handleVLMCompletion(c, req)
		return
	}
	
	// Convert messages to yzma format
	yzmaMessages := make([]yzma.ChatMessage, len(req.Messages))
	for i, msg := range req.Messages {
		// Extract content as string
		content := ""
		if msg.Content != nil {
			switch v := msg.Content.(type) {
			case string:
				content = v
			case []interface{}:
				// For multi-modal, extract text parts
				for _, part := range v {
					if partMap, ok := part.(map[string]interface{}); ok {
						if text, ok := partMap["text"].(string); ok {
							content += text
						}
					}
				}
			}
		}
		
		yzmaMessages[i] = yzma.ChatMessage{
			Role:    msg.Role,
			Content: content,
		}
	}
	
	// Resolve alias to model path (v3.0.5+)
	modelPath := h.client.ResolveModelPath(req.Model)
	
	h.logger.WithFields(logrus.Fields{
		"requested_model": req.Model,
		"resolved_path":   modelPath,
	}).Debug("Model alias resolved")
	
	// Prepare yzma request
	yzmaReq := yzma.GenerateRequest{
		ModelPath: modelPath,  // v3.0.5+: Use resolved path
		Messages:  yzmaMessages,
		MaxTokens: h.getIntValue(req.MaxTokens, 512),
		Temperature: h.getFloatValue(req.Temperature, 0.7),
		TopP:      h.getFloatValue(req.TopP, 0.9),
		TopK:      int32(40), // Default TopK
		Stop:      req.Stop,
		Stream:    req.Stream,
	}
	
	h.logger.WithFields(logrus.Fields{
		"model":       req.Model,
		"model_path":  modelPath,
		"messages":    len(req.Messages),
		"max_tokens":  yzmaReq.MaxTokens,
		"temperature": yzmaReq.Temperature,
		"stream":      yzmaReq.Stream,
	}).Debug("Chat completion request")
	
	if yzmaReq.Stream {
		h.handleStreamingCompletion(c, req, yzmaReq)
	} else {
		h.handleNonStreamingCompletion(c, req, yzmaReq)
	}
}

// handleNonStreamingCompletion handles regular chat completion
func (h *YzmaHandler) handleNonStreamingCompletion(c *gin.Context, req models.ChatCompletionRequest, yzmaReq yzma.GenerateRequest) {
	// Use configurable timeout (default 30m, 0 = no timeout)
	ctx := c.Request.Context()
	var cancel context.CancelFunc
	if h.requestTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, h.requestTimeout)
		defer cancel()
	}
	
	// Generate with yzma
	resp, err := h.client.Generate(ctx, yzmaReq)
	if err != nil {
		h.logger.WithError(err).Error("Generation failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"message": fmt.Sprintf("Generation failed: %v", err),
				"type":    "internal_error",
			},
		})
		return
	}
	
	// Build OpenAI-compatible response
	finishReason := resp.FinishReason
	if finishReason == "" {
		finishReason = "stop"
	}
	
	response := models.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-yzma-%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []models.ChatCompletionChoice{
			{
				Index: 0,
				Message: models.ChatMessage{
					Role:    "assistant",
					Content: resp.Content,
				},
				FinishReason: finishReason,
			},
		},
		Usage: models.Usage{
			PromptTokens:     resp.PromptTokens,
			CompletionTokens: resp.TokensUsed,
			TotalTokens:      resp.PromptTokens + resp.TokensUsed,
		},
	}
	
	h.logger.WithFields(logrus.Fields{
		"model":          req.Model,
		"tokens_used":    resp.TokensUsed,
		"tokens_per_sec": resp.TokensPerSec,
		"duration":       resp.Duration,
	}).Info("Chat completion success")
	
	c.JSON(http.StatusOK, response)
}

// handleStreamingCompletion handles streaming chat completion
func (h *YzmaHandler) handleStreamingCompletion(c *gin.Context, req models.ChatCompletionRequest, yzmaReq yzma.GenerateRequest) {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	
	// Use configurable timeout (default 30m, 0 = no timeout)
	ctx := c.Request.Context()
	var cancel context.CancelFunc
	if h.requestTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, h.requestTimeout)
		defer cancel()
	}
	
	// Create flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.logger.Error("Streaming not supported")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"message": "Streaming not supported",
				"type":    "internal_error",
			},
		})
		return
	}
	
	// Stream callback
	callback := func(token string) error {
		chunk := map[string]interface{}{
			"id":      fmt.Sprintf("chatcmpl-yzma-%d", time.Now().Unix()),
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   req.Model,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]string{
						"content": token,
					},
					"finish_reason": nil,
				},
			},
		}
		
		data, err := json.Marshal(chunk)
		if err != nil {
			return err
		}
		
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		flusher.Flush()
		
		return nil
	}
	
	// Generate streaming
	resp, err := h.client.GenerateStream(ctx, yzmaReq, callback)
	if err != nil {
		h.logger.WithError(err).Error("Streaming generation failed")
		// Send error chunk
		errorChunk := map[string]interface{}{
			"error": map[string]string{
				"message": err.Error(),
				"type":    "internal_error",
			},
		}
		data, _ := json.Marshal(errorChunk)
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		flusher.Flush()
		return
	}
	
	// Send final chunk with finish reason
	finishReason := resp.FinishReason
	if finishReason == "" {
		finishReason = "stop"
	}
	
	finalChunk := map[string]interface{}{
		"id":      fmt.Sprintf("chatcmpl-yzma-%d", time.Now().Unix()),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   req.Model,
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"delta":         map[string]string{},
				"finish_reason": finishReason,
			},
		},
	}
	
	data, _ := json.Marshal(finalChunk)
	fmt.Fprintf(c.Writer, "data: %s\n\n", data)
	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
	
	h.logger.WithFields(logrus.Fields{
		"model":          req.Model,
		"tokens_used":    resp.TokensUsed,
		"tokens_per_sec": resp.TokensPerSec,
	}).Info("Streaming completion success")
}

// HandleModels returns LOADED models (OpenAI compatible)
// v3.0.6+: Only returns models that are currently loaded in memory
// v3.0.6+: Uses Redis cache for instant response
func (h *YzmaHandler) HandleModels(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Try Redis cache first (v3.0.6+: background sync)
	if h.modelWorker != nil {
		cachedModels, err := h.modelWorker.GetCachedModelList(ctx)
		if err == nil {
			h.logger.Debug("✅ Model list served from Redis cache")
			
			response := map[string]interface{}{
				"object": "list",
				"data":   cachedModels,
			}
			
			c.JSON(http.StatusOK, response)
			return
		}
		
		// Cache miss - fallback to direct listing
		h.logger.WithError(err).Debug("⚠️  Model list cache miss, listing directly")
	}
	
	// Fallback: Get ONLY loaded models directly (slower)
	loadedModels := h.client.ListLoadedModels()
	
	// Convert to OpenAI format (v3.0.5+: use aliases)
	var modelObjects []map[string]interface{}
	for modelPath, modelInfo := range loadedModels {
		modelObj := map[string]interface{}{
			"id":      modelInfo["alias"],  // v3.0.5+: Show alias instead of full path
			"object":  "model",
			"created": time.Now().Unix(),
			"owned_by": "local",
			"loaded":  true,       // Always true since we only return loaded models
			"path":    modelPath,  // Keep original path for debugging
		}
		
		// Add size if available
		if size, ok := modelInfo["size"].(int64); ok && size > 0 {
			modelObj["size"] = size
		}
		
		modelObjects = append(modelObjects, modelObj)
	}
	
	response := map[string]interface{}{
		"object": "list",
		"data":   modelObjects,
	}
	
	c.JSON(http.StatusOK, response)
}

// HandleModelInfo returns specific model info
func (h *YzmaHandler) HandleModelInfo(c *gin.Context) {
	modelPath := c.Param("model")
	
	// Check if model is loaded
	isLoaded := h.client.IsModelLoaded(modelPath)
	
	// Check if model exists
	availableModels, err := h.client.ListAvailableModels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"message": "Failed to check model availability",
				"type":    "internal_error",
			},
		})
		return
	}
	
	found := false
	for _, available := range availableModels {
		if available == modelPath {
			found = true
			break
		}
	}
	
	if !found {
		c.JSON(http.StatusNotFound, gin.H{
			"error": map[string]string{
				"message": fmt.Sprintf("Model not found: %s", modelPath),
				"type":    "invalid_request_error",
			},
		})
		return
	}
	
	response := map[string]interface{}{
		"id":       modelPath,
		"object":   "model",
		"created":  time.Now().Unix(),
		"owned_by": "local",
		"loaded":   isLoaded,
		"type":     "gguf",
	}
	
	c.JSON(http.StatusOK, response)
}

// HandleLoadModel manually loads a model
func (h *YzmaHandler) HandleLoadModel(c *gin.Context) {
	var req struct {
		ModelPath string `json:"model_path" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"message": "Invalid request",
				"type":    "invalid_request_error",
			},
		})
		return
	}
	
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()
	
	if err := h.client.LoadModel(ctx, req.ModelPath); err != nil {
		h.logger.WithError(err).Error("Failed to load model")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"message": fmt.Sprintf("Failed to load model: %v", err),
				"type":    "internal_error",
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status":     "loaded",
		"model_path": req.ModelPath,
	})
}

// HandleUnloadModel unloads a model from memory
func (h *YzmaHandler) HandleUnloadModel(c *gin.Context) {
	var req struct {
		ModelPath string `json:"model_path" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"message": "Invalid request",
				"type":    "invalid_request_error",
			},
		})
		return
	}
	
	if err := h.client.UnloadModel(req.ModelPath); err != nil {
		h.logger.WithError(err).Error("Failed to unload model")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"message": fmt.Sprintf("Failed to unload model: %v", err),
				"type":    "internal_error",
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status":     "unloaded",
		"model_path": req.ModelPath,
	})
}

// HandleYzmaStats returns yzma client statistics
func (h *YzmaHandler) HandleYzmaStats(c *gin.Context) {
	requests, tokens := h.client.GetStats()
	
	c.JSON(http.StatusOK, gin.H{
		"total_requests": requests,
		"total_tokens":   tokens,
		"loaded_models":  h.client.ListLoadedModels(),
	})
}

// HandleGPUInfo returns GPU configuration and status (v3.2.2+)
// This endpoint is safe to call before models are loaded
func (h *YzmaHandler) HandleGPUInfo(c *gin.Context) {
	gpuInfo := h.client.GetGPUInfo()
	
	c.JSON(http.StatusOK, gin.H{
		"gpu":               gpuInfo,
		"backend_ready":     gpuInfo.Initialized,
		"models_ready":      gpuInfo.LoadedModelCount > 0,
		"loaded_model_count": gpuInfo.LoadedModelCount,
	})
}

// HandleHealthCheck returns basic health status (v3.2.2+)
// This endpoint always works, even during model loading
func (h *YzmaHandler) HandleHealthCheck(c *gin.Context) {
	initialized := h.client.IsInitialized()
	loadedModels := h.client.ListLoadedModels()
	
	status := "initializing"
	if initialized && len(loadedModels) > 0 {
		status = "ready"
	} else if initialized {
		status = "waiting_for_models"
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status":             status,
		"initialized":        initialized,
		"loaded_model_count": len(loadedModels),
		"loaded_models":      loadedModels,
	})
}

// Helper methods
func (h *YzmaHandler) getIntValue(ptr *int, defaultVal int) int {
	if ptr == nil {
		return defaultVal
	}
	return *ptr
}

func (h *YzmaHandler) getFloatValue(ptr *float64, defaultVal float32) float32 {
	if ptr == nil {
		return defaultVal
	}
	return float32(*ptr)
}

// VLM-related helper methods (v3.0.4+)

// containsImages checks if messages contain any images
func (h *YzmaHandler) containsImages(messages []models.ChatMessage) bool {
	for _, msg := range messages {
		if msg.Content == nil {
			continue
		}
		
		// Check if content is array (multimodal)
		switch content := msg.Content.(type) {
		case []interface{}:
			for _, part := range content {
				if partMap, ok := part.(map[string]interface{}); ok {
					if partMap["type"] == "image_url" {
						return true
					}
				}
			}
		}
	}
	return false
}

// extractTextAndImages extracts text prompt and images from multimodal messages
func (h *YzmaHandler) extractTextAndImages(messages []models.ChatMessage) (string, []string, string) {
	var textParts []string
	var images []string
	var systemPrompt string
	
	for _, msg := range messages {
		if msg.Role == "system" {
			if str, ok := msg.Content.(string); ok {
				systemPrompt = str
			}
			continue
		}
		
		switch content := msg.Content.(type) {
		case string:
			// Simple text content
			textParts = append(textParts, content)
			
		case []interface{}:
			// Multimodal content
			for _, part := range content {
				if partMap, ok := part.(map[string]interface{}); ok {
					switch partMap["type"] {
					case "text":
						if text, ok := partMap["text"].(string); ok {
							textParts = append(textParts, text)
						}
					case "image_url":
						if imageURL, ok := partMap["image_url"].(map[string]interface{}); ok {
							if url, ok := imageURL["url"].(string); ok {
								images = append(images, url)
							}
						}
					}
				}
			}
		}
	}
	
	return strings.Join(textParts, "\n"), images, systemPrompt
}

// handleVLMCompletion handles chat completion with images (VLM-03: v3.0.4+)
func (h *YzmaHandler) handleVLMCompletion(c *gin.Context, req models.ChatCompletionRequest) {
	// Extract text and images from messages
	textPrompt, images, systemPrompt := h.extractTextAndImages(req.Messages)
	
	if len(images) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"message": "No images found in multimodal request",
				"type":    "invalid_request_error",
			},
		})
		return
	}
	
	h.logger.WithFields(logrus.Fields{
		"model":         req.Model,
		"images_count":  len(images),
		"text_length":   len(textPrompt),
		"has_system":    systemPrompt != "",
	}).Info("VLM completion request")
	
	// Check if model is VLM
	if !h.client.IsVLMModel(req.Model) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"message": fmt.Sprintf("Model %s is not a VLM. Load it with mmproj first.", req.Model),
				"type":    "invalid_model_error",
			},
		})
		return
	}
	
	// Prepare VLM request
	vlmReq := yzma.VLMGenerateRequest{
		ModelPath:     req.Model,
		MMProjPath:    "", // Already loaded
		Prompt:        textPrompt,
		Images:        images,
		SystemPrompt:  systemPrompt,
		Temperature:   h.getFloatValue(req.Temperature, 0.7),
		TopK:          int32(40),
		TopP:          h.getFloatValue(req.TopP, 0.9),
		MinP:          h.getFloatValue(nil, 0.1),
		MaxTokens:     h.getIntValue(req.MaxTokens, 2048),
		StopSequences: req.Stop,
	}
	
	// Generate with images
	resp, err := h.client.GenerateWithImages(c.Request.Context(), vlmReq)
	if err != nil {
		h.logger.WithError(err).Error("VLM generation failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"message": fmt.Sprintf("VLM generation failed: %v", err),
				"type":    "api_error",
			},
		})
		return
	}
	
	// Build OpenAI-compatible response
	response := models.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []models.ChatCompletionChoice{
			{
				Index: 0,
				Message: models.ChatMessage{
					Role:    "assistant",
					Content: resp.Text,
				},
				FinishReason: "stop",
			},
		},
		Usage: models.Usage{
			PromptTokens:     resp.PromptTokens,
			CompletionTokens: resp.TokensCount,
			TotalTokens:      resp.PromptTokens + resp.TokensCount,
		},
	}
	
	c.JSON(http.StatusOK, response)
}

// handleAgentCompletion handles agent mode with ReAct loop (v2.5.0+, v3.0.6+)
// Tools are executed CLIENT-SIDE - server only orchestrates the loop
func (h *YzmaHandler) handleAgentCompletion(c *gin.Context, req models.ChatCompletionRequest) {
	// Get or create agent session
	conversationID := req.ConversationID
	if conversationID == "" {
		conversationID = fmt.Sprintf("agent-%d", time.Now().UnixNano())
	}
	
	h.logger.WithFields(logrus.Fields{
		"conversation_id": conversationID,
		"agent_mode":      true,
		"max_iterations":  req.AgentMaxIter,
		"working_dir":     req.AgentWorkingDirectory,
	}).Info("Agent mode: Starting ReAct loop")
	
	// Get or create agent context
	agentCtx, err := h.agentService.GetAgentSession(conversationID)
	if err != nil {
		// Create new session
		task := ""
		if len(req.Messages) > 0 {
			if content, ok := req.Messages[len(req.Messages)-1].Content.(string); ok {
				task = content
			}
		}
		
		agentCtx, err = h.agentService.CreateAgentSession(c.Request.Context(), conversationID, task)
		if err != nil {
			h.logger.WithError(err).Error("Failed to create agent session")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": map[string]string{
					"message": fmt.Sprintf("Failed to create agent session: %v", err),
					"type":    "internal_error",
				},
			})
			return
		}
	}
	
	// Generate ReAct system prompt with available tools
	systemPrompt := h.agentService.GenerateSystemPrompt(req.AgentWorkingDirectory)
	
	// Prepend system prompt to messages
	messages := []models.ChatMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}
	messages = append(messages, req.Messages...)
	
	// Convert to yzma format
	yzmaMessages := make([]yzma.ChatMessage, len(messages))
	for i, msg := range messages {
		content := ""
		if msg.Content != nil {
			if str, ok := msg.Content.(string); ok {
				content = str
			}
		}
		yzmaMessages[i] = yzma.ChatMessage{
			Role:    msg.Role,
			Content: content,
		}
	}
	
	// Resolve model
	modelPath := h.client.ResolveModelPath(req.Model)
	
	// Prepare yzma request
	yzmaReq := yzma.GenerateRequest{
		ModelPath:   modelPath,
		Messages:    yzmaMessages,
		MaxTokens:   h.getIntValue(req.MaxTokens, 2048),
		Temperature: h.getFloatValue(req.Temperature, 0.7),
		TopP:        h.getFloatValue(req.TopP, 0.9),
		TopK:        int32(40),
		Stop:        req.Stop,
		Stream:      req.Stream,
	}
	
	h.logger.WithFields(logrus.Fields{
		"model":         req.Model,
		"model_path":    modelPath,
		"messages":      len(messages),
		"system_prompt": len(systemPrompt),
		"stream":        yzmaReq.Stream,
	}).Debug("Agent request prepared")
	
	// Execute based on streaming mode
	if yzmaReq.Stream {
		h.handleAgentStreamingCompletion(c, req, yzmaReq, agentCtx, conversationID)
	} else {
		h.handleAgentNonStreamingCompletion(c, req, yzmaReq, agentCtx, conversationID)
	}
}

// handleAgentNonStreamingCompletion handles non-streaming agent completion
func (h *YzmaHandler) handleAgentNonStreamingCompletion(c *gin.Context, req models.ChatCompletionRequest, yzmaReq yzma.GenerateRequest, agentCtx *models.AgentContext, conversationID string) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute) // Longer timeout for agent
	defer cancel()
	
	// Generate with yzma
	resp, err := h.client.Generate(ctx, yzmaReq)
	if err != nil {
		h.logger.WithError(err).Error("Agent generation failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"message": fmt.Sprintf("Agent generation failed: %v", err),
				"type":    "internal_error",
			},
		})
		return
	}
	
	// Parse LLM response for ReAct format
	// Expected format: 
	//   Markdown: **Thought**: text\n**Action**: tool.name\n**Parameters**: {...}
	//   or JSON: {"thought": "...", "action": "tool.name", "parameters": {...}}
	//   or Final: **final_answer**: text
	
	agentMessages := []map[string]interface{}{}
	
	// Try to parse ReAct response (both JSON and Markdown)
	reactResponse := h.parseReActResponse(resp.Content)
	
	// Add thinking message
	if thought, ok := reactResponse["thought"].(string); ok && thought != "" {
		agentMessages = append(agentMessages, map[string]interface{}{
			"role":    "agent_thinking",
			"content": thought,
		})
	}
	
	h.logger.WithFields(logrus.Fields{
		"has_thought":      reactResponse["thought"] != nil,
		"has_action":       reactResponse["action"] != nil,
		"has_parameters":   reactResponse["parameters"] != nil,
		"has_final_answer": reactResponse["final_answer"] != nil,
	}).Debug("ReAct response parsed")
	
	// Check for final answer
	if finalAnswer, ok := reactResponse["final_answer"].(string); ok && finalAnswer != "" {
		h.logger.Info("Agent completed with final answer")
		// Task complete
		agentCtx.MarkComplete()
		h.agentService.UpdateAgentSession(conversationID, agentCtx)
			
			// Return final answer
			response := models.ChatCompletionResponse{
				ID:      fmt.Sprintf("chatcmpl-agent-%d", time.Now().Unix()),
				Object:  "chat.completion",
				Created: time.Now().Unix(),
				Model:   req.Model,
				Choices: []models.ChatCompletionChoice{
					{
						Index: 0,
						Message: models.ChatMessage{
							Role:    "assistant",
							Content: finalAnswer,
						},
						FinishReason: "stop",
					},
				},
				Usage: models.Usage{
					PromptTokens:     resp.PromptTokens,
					CompletionTokens: resp.TokensUsed,
					TotalTokens:      resp.PromptTokens + resp.TokensUsed,
				},
				AgentMessages: agentMessages,
			}
			
			c.JSON(http.StatusOK, response)
			return
		}
		
	// Action required - need client to execute
	if action, ok := reactResponse["action"].(string); ok && action != "" {
		parameters, _ := reactResponse["parameters"].(map[string]interface{})
		
		h.logger.WithFields(logrus.Fields{
			"action":     action,
			"parameters": parameters,
		}).Info("Agent requesting tool execution")
		
		// Add action message
		agentMessages = append(agentMessages, map[string]interface{}{
			"role":       "agent_action",
			"content":    action,
			"tool":       action,
			"parameters": parameters,
		})
			
			// Return response indicating client needs to execute tool
			response := models.ChatCompletionResponse{
				ID:      fmt.Sprintf("chatcmpl-agent-%d", time.Now().Unix()),
				Object:  "chat.completion",
				Created: time.Now().Unix(),
				Model:   req.Model,
				Choices: []models.ChatCompletionChoice{
					{
						Index: 0,
						Message: models.ChatMessage{
							Role:    "assistant",
							Content: fmt.Sprintf("Action required: %s", action),
						},
						FinishReason: "tool_calls", // Signal that tool execution is needed
					},
				},
				Usage: models.Usage{
					PromptTokens:     resp.PromptTokens,
					CompletionTokens: resp.TokensUsed,
					TotalTokens:      resp.PromptTokens + resp.TokensUsed,
				},
				AgentMessages: agentMessages,
			}
			
			c.JSON(http.StatusOK, response)
			return
	}
	
	// Fallback: return raw response (no valid ReAct format detected)
	h.logger.Warn("Agent response did not match ReAct format, returning raw content")
	response := models.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-agent-%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []models.ChatCompletionChoice{
			{
				Index: 0,
				Message: models.ChatMessage{
					Role:    "assistant",
					Content: resp.Content,
				},
				FinishReason: "stop",
			},
		},
		Usage: models.Usage{
			PromptTokens:     resp.PromptTokens,
			CompletionTokens: resp.TokensUsed,
			TotalTokens:      resp.PromptTokens + resp.TokensUsed,
		},
		AgentMessages: agentMessages,
	}
	
	c.JSON(http.StatusOK, response)
}

// handleAgentStreamingCompletion handles streaming agent completion
// v3.0.6+: Streams agent messages (agent_thinking, agent_action, agent_observation)
func (h *YzmaHandler) handleAgentStreamingCompletion(c *gin.Context, req models.ChatCompletionRequest, yzmaReq yzma.GenerateRequest, agentCtx *models.AgentContext, conversationID string) {
	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()
	
	responseBuffer := ""
	
	// Streaming callback
	callback := func(token string) error {
		// Accumulate response
		responseBuffer += token
		
		// Send thinking chunk
		thinkingChunk := models.ChatCompletionChunk{
			ID:      fmt.Sprintf("chatcmpl-agent-%d", time.Now().Unix()),
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []models.ChatCompletionChunkChoice{
				{
					Index: 0,
					Delta: models.ChatMessage{
						Role:    "agent_thinking",
						Content: token,
					},
					FinishReason: nil,
				},
			},
		}
		
		data, err := json.Marshal(thinkingChunk)
		if err != nil {
			h.logger.WithError(err).Error("Failed to marshal chunk")
			return err
		}
		
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		c.Writer.Flush()
		
		return nil
	}
	
	// Generate with streaming
	_, err := h.client.GenerateStream(ctx, yzmaReq, callback)
	if err != nil {
		h.logger.WithError(err).Error("Agent streaming generation failed")
		
		errorChunk := map[string]interface{}{
			"error": map[string]string{
				"message": fmt.Sprintf("Agent generation failed: %v", err),
				"type":    "internal_error",
			},
		}
		
		data, _ := json.Marshal(errorChunk)
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		c.Writer.Flush()
		return
	}
	
	// Stream ended - parse final response
	h.logger.Debug("Agent stream ended, parsing response")
	
	// Parse ReAct response (both JSON and Markdown)
	reactResponse := h.parseReActResponse(responseBuffer)
	
	// Check for final answer
	if finalAnswer, ok := reactResponse["final_answer"].(string); ok && finalAnswer != "" {
			// Send final answer chunk
			finalChunk := models.ChatCompletionChunk{
				ID:      fmt.Sprintf("chatcmpl-agent-%d", time.Now().Unix()),
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   req.Model,
				Choices: []models.ChatCompletionChunkChoice{
					{
						Index: 0,
						Delta: models.ChatMessage{
							Role:    "assistant",
							Content: finalAnswer,
						},
						FinishReason: stringPtr("stop"),
					},
				},
			}
			
		data, _ := json.Marshal(finalChunk)
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		c.Writer.Flush()
	} else if action, ok := reactResponse["action"].(string); ok && action != "" {
		// Send action event
		parameters := make(map[string]interface{})
		if params, ok := reactResponse["parameters"].(map[string]interface{}); ok {
			parameters = params
		}
		
		actionChunk := models.ChatCompletionChunk{
			ID:      fmt.Sprintf("chatcmpl-agent-%d", time.Now().Unix()),
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   req.Model,
			Choices: []models.ChatCompletionChunkChoice{
				{
					Index: 0,
					Delta: models.ChatMessage{
						Role:    "agent_action",
						Content: map[string]interface{}{
							"tool":       action,
							"parameters": parameters,
						},
					},
					FinishReason: stringPtr("tool_calls"),
				},
			},
		}
		
		data, _ := json.Marshal(actionChunk)
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		c.Writer.Flush()
	}
	
	// Send [DONE]
	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()
}

// parseReActResponse parses LLM response in both JSON and Markdown formats
// Returns a map with keys: "thought", "action", "parameters", "final_answer"
func (h *YzmaHandler) parseReActResponse(content string) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Try JSON first (direct)
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal([]byte(content), &jsonResponse); err == nil {
		// Valid JSON - use it directly
		return jsonResponse
	}
	
	// Try extracting JSON from markdown code blocks
	// Match ```json ... ``` blocks
	re := regexp.MustCompile("```json\\s*\\n([\\s\\S]*?)\\n```")
	matches := re.FindAllStringSubmatch(content, -1)
	
	if len(matches) > 0 {
		// Try each JSON block (take FIRST valid one to avoid hallucinations)
		for _, match := range matches {
			if len(match) > 1 {
				jsonStr := strings.TrimSpace(match[1])
				var jsonResp map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &jsonResp); err == nil {
					return jsonResp
				}
			}
		}
	}
	
	// Parse Markdown format
	// Format: **Thought**: text\n**Action**: tool.name\n**Parameters**: {...}\n**final_answer**: text
	lines := strings.Split(content, "\n")
	
	currentKey := ""
	currentValue := ""
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Check for section headers
		if strings.HasPrefix(line, "**Thought**:") || strings.HasPrefix(line, "**thought**:") {
			// Save previous section
			if currentKey != "" && currentValue != "" {
				result[currentKey] = strings.TrimSpace(currentValue)
			}
			currentKey = "thought"
			currentValue = strings.TrimPrefix(strings.TrimPrefix(line, "**Thought**:"), "**thought**:")
			currentValue = strings.TrimSpace(currentValue)
		} else if strings.HasPrefix(line, "**Action**:") || strings.HasPrefix(line, "**action**:") {
			// Save previous section
			if currentKey != "" && currentValue != "" {
				result[currentKey] = strings.TrimSpace(currentValue)
			}
			currentKey = "action"
			currentValue = strings.TrimPrefix(strings.TrimPrefix(line, "**Action**:"), "**action**:")
			currentValue = strings.TrimSpace(currentValue)
		} else if strings.HasPrefix(line, "**Parameters**:") || strings.HasPrefix(line, "**parameters**:") {
			// Save previous section
			if currentKey != "" && currentValue != "" {
				result[currentKey] = strings.TrimSpace(currentValue)
			}
			currentKey = "parameters"
			currentValue = strings.TrimPrefix(strings.TrimPrefix(line, "**Parameters**:"), "**parameters**:")
			currentValue = strings.TrimSpace(currentValue)
		} else if strings.HasPrefix(line, "**final_answer**:") || strings.HasPrefix(line, "**Final Answer**:") {
			// Save previous section
			if currentKey != "" && currentValue != "" {
				result[currentKey] = strings.TrimSpace(currentValue)
			}
			currentKey = "final_answer"
			currentValue = strings.TrimPrefix(strings.TrimPrefix(line, "**final_answer**:"), "**Final Answer**:")
			currentValue = strings.TrimSpace(currentValue)
		} else if currentKey != "" {
			// Continue accumulating value
			if currentValue != "" {
				currentValue += "\n" + line
			} else {
				currentValue = line
			}
		}
	}
	
	// Save last section
	if currentKey != "" && currentValue != "" {
		result[currentKey] = strings.TrimSpace(currentValue)
	}
	
	// Parse parameters if it's a JSON string
	if paramsStr, ok := result["parameters"].(string); ok {
		// Try to clean up markdown code blocks
		paramsStr = strings.TrimPrefix(paramsStr, "```json")
		paramsStr = strings.TrimPrefix(paramsStr, "```")
		paramsStr = strings.TrimSuffix(paramsStr, "```")
		paramsStr = strings.TrimSpace(paramsStr)
		
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(paramsStr), &params); err == nil {
			result["parameters"] = params
		}
	}
	
	h.logger.WithFields(logrus.Fields{
		"has_thought":      result["thought"] != nil,
		"has_action":       result["action"] != nil,
		"has_parameters":   result["parameters"] != nil,
		"has_final_answer": result["final_answer"] != nil,
	}).Debug("Parsed ReAct response")
	
	return result
}

// stringPtr returns a pointer to the string value
func stringPtr(s string) *string {
	return &s
}


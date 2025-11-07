// Package handlers provides HTTP handlers for yzma local inference
// Version: v3.0.0 - YZMA-02: OpenAI-compatible inference API
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aigateway/internal/models"
	"aigateway/internal/yzma"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// YzmaHandler handles yzma inference requests with OpenAI compatibility
type YzmaHandler struct {
	client *yzma.Client
	logger *logrus.Logger
}

// NewYzmaHandler creates a new yzma handler
func NewYzmaHandler(client *yzma.Client, logger *logrus.Logger) *YzmaHandler {
	return &YzmaHandler{
		client: client,
		logger: logger,
	}
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
	
	// Prepare yzma request
	yzmaReq := yzma.GenerateRequest{
		ModelPath: req.Model,
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
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()
	
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
	
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()
	
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

// HandleModels returns available models (OpenAI compatible)
func (h *YzmaHandler) HandleModels(c *gin.Context) {
	modelsList, err := h.client.ListAvailableModels()
	if err != nil {
		h.logger.WithError(err).Error("Failed to list models")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"message": fmt.Sprintf("Failed to list models: %v", err),
				"type":    "internal_error",
			},
		})
		return
	}
	
	// Convert to OpenAI format
	var modelObjects []map[string]interface{}
	for _, modelPath := range modelsList {
		isLoaded := h.client.IsModelLoaded(modelPath)
		modelObjects = append(modelObjects, map[string]interface{}{
			"id":      modelPath,
			"object":  "model",
			"created": time.Now().Unix(),
			"owned_by": "local",
			"loaded":  isLoaded,
		})
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


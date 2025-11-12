// Package handlers provides HTTP handlers for yzma local inference
// Version: v3.0.5 - Text Completions API support (legacy OpenAI)
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"aigateway/internal/models"
	"aigateway/internal/yzma"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// HandleCompletions handles /v1/completions (OpenAI legacy API, v3.0.5+)
func (h *YzmaHandler) HandleCompletions(c *gin.Context) {
	var req models.CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"message": fmt.Sprintf("Invalid request: %v", err),
				"type":    "invalid_request_error",
			},
		})
		return
	}
	
	// Convert prompt to string (can be string or array of strings)
	var prompt string
	switch v := req.Prompt.(type) {
	case string:
		prompt = v
	case []interface{}:
		// For array, join with newlines
		for i, item := range v {
			if str, ok := item.(string); ok {
				if i > 0 {
					prompt += "\n"
				}
				prompt += str
			}
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"message": "Invalid prompt format, expected string or array of strings",
				"type":    "invalid_request_error",
			},
		})
		return
	}
	
	if prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"message": "Prompt cannot be empty",
				"type":    "invalid_request_error",
			},
		})
		return
	}
	
	// Resolve model alias
	modelPath := h.client.ResolveModelPath(req.Model)
	
	h.logger.WithFields(logrus.Fields{
		"model":       req.Model,
		"model_path":  modelPath,
		"prompt_len":  len(prompt),
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}).Debug("Text completion request")
	
	// Prepare yzma request
	yzmaReq := yzma.GenerateRequest{
		ModelPath:   modelPath,
		Prompt:      prompt,
		MaxTokens:   h.getIntValue(req.MaxTokens, 256),
		Temperature: h.getFloatValue(req.Temperature, 0.7),
		TopP:        h.getFloatValue(req.TopP, 0.9),
		Stop:        req.Stop,
		Stream:      req.Stream,
	}
	
	// Handle streaming vs non-streaming
	if yzmaReq.Stream {
		h.handleStreamingTextCompletion(c, req, yzmaReq)
	} else {
		h.handleNonStreamingTextCompletion(c, req, yzmaReq)
	}
}

// handleNonStreamingTextCompletion handles non-streaming text completion
func (h *YzmaHandler) handleNonStreamingTextCompletion(c *gin.Context, req models.CompletionRequest, yzmaReq yzma.GenerateRequest) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()
	
	// Generate text
	resp, err := h.client.Generate(ctx, yzmaReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate text completion")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"message": fmt.Sprintf("Failed to generate completion: %v", err),
				"type":    "api_error",
			},
		})
		return
	}
	
	// Get model alias for response
	modelAlias := req.Model
	if modelCtx := h.client.GetModelByAlias(req.Model); modelCtx != nil {
		modelAlias = modelCtx.Alias
	}
	
	// Build OpenAI-compatible response
	response := models.CompletionResponse{
		ID:      fmt.Sprintf("cmpl-%d", time.Now().Unix()),
		Object:  "text_completion",
		Created: time.Now().Unix(),
		Model:   modelAlias,
		Choices: []models.CompletionChoice{
			{
				Text:         resp.Content,
				Index:        0,
				FinishReason: "stop",
			},
		},
		Usage: models.Usage{
			PromptTokens:     resp.PromptTokens,
			CompletionTokens: resp.TokensUsed,
			TotalTokens:      resp.PromptTokens + resp.TokensUsed,
		},
	}
	
	h.logger.WithFields(logrus.Fields{
		"model":  req.Model,
		"tokens": resp.TokensUsed,
	}).Info("Text completion generated successfully")
	
	c.JSON(http.StatusOK, response)
}

// handleStreamingTextCompletion handles streaming text completion
func (h *YzmaHandler) handleStreamingTextCompletion(c *gin.Context, req models.CompletionRequest, yzmaReq yzma.GenerateRequest) {
	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	
	ctx, cancel := context.WithTimeout(c.Request.Context(), 300*time.Second)
	defer cancel()
	
	// Create flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.logger.Error("Streaming not supported")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"message": "Streaming not supported",
				"type":    "api_error",
			},
		})
		return
	}
	
	// Get model alias
	modelAlias := req.Model
	if modelCtx := h.client.GetModelByAlias(req.Model); modelCtx != nil {
		modelAlias = modelCtx.Alias
	}
	
	// Stream callback
	callback := func(token string) error {
		chunk := models.CompletionResponse{
			ID:      fmt.Sprintf("cmpl-%d", time.Now().Unix()),
			Object:  "text_completion.chunk",
			Created: time.Now().Unix(),
			Model:   modelAlias,
			Choices: []models.CompletionChoice{
				{
					Text:         token,
					Index:        0,
					FinishReason: "",
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
		h.logger.WithError(err).Error("Streaming text completion failed")
		// Send error chunk
		errorChunk := gin.H{
			"error": map[string]string{
				"message": fmt.Sprintf("Stream error: %v", err),
				"type":    "api_error",
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
	
	finalChunk := models.CompletionResponse{
		ID:      fmt.Sprintf("cmpl-%d", time.Now().Unix()),
		Object:  "text_completion.chunk",
		Created: time.Now().Unix(),
		Model:   modelAlias,
		Choices: []models.CompletionChoice{
			{
				Text:         "",
				Index:        0,
				FinishReason: finishReason,
			},
		},
	}
	
	data, _ := json.Marshal(finalChunk)
	fmt.Fprintf(c.Writer, "data: %s\n\n", data)
	fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
	flusher.Flush()
}


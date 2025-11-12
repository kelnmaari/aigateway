// Package handlers provides HTTP handlers for yzma local inference
// Version: v3.0.5 - Embeddings API support
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aigateway/internal/models"
	"aigateway/internal/yzma"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// HandleEmbeddings handles /v1/embeddings (OpenAI compatible, v3.0.5+)
func (h *YzmaHandler) HandleEmbeddings(c *gin.Context) {
	var req models.EmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"message": fmt.Sprintf("Invalid request: %v", err),
				"type":    "invalid_request_error",
			},
		})
		return
	}
	
	// Convert input to []string
	var inputs []string
	switch v := req.Input.(type) {
	case string:
		inputs = []string{v}
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok {
				inputs = append(inputs, str)
			}
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"message": "Invalid input format, expected string or array of strings",
				"type":    "invalid_request_error",
			},
		})
		return
	}
	
	if len(inputs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"message": "Input cannot be empty",
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
		"num_inputs":  len(inputs),
	}).Debug("Embeddings request")
	
	// Call yzma client
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()
	
	resp, err := h.client.GenerateEmbedding(ctx, yzma.EmbeddingRequest{
		ModelPath: modelPath,
		Input:     inputs,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate embeddings")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": map[string]string{
				"message": fmt.Sprintf("Failed to generate embeddings: %v", err),
				"type":    "api_error",
			},
		})
		return
	}
	
	// Build OpenAI-compatible response
	var data []map[string]interface{}
	totalTokens := 0
	
	for i, embedding := range resp.Embeddings {
		data = append(data, map[string]interface{}{
			"object":    "embedding",
			"embedding": embedding,
			"index":     i,
		})
		// Estimate tokens (rough approximation)
		totalTokens += len(strings.Split(inputs[i], " "))
	}
	
	// Get model alias for response
	modelAlias := req.Model
	if modelCtx := h.client.GetModelByAlias(req.Model); modelCtx != nil {
		modelAlias = modelCtx.Alias
	}
	
	response := map[string]interface{}{
		"object": "list",
		"data":   data,
		"model":  modelAlias,
		"usage": map[string]interface{}{
			"prompt_tokens": totalTokens,
			"total_tokens":  totalTokens,
		},
	}
	
	h.logger.WithFields(logrus.Fields{
		"model":      req.Model,
		"embeddings": len(resp.Embeddings),
		"dimensions": resp.Dimensions,
	}).Info("Embeddings generated successfully")
	
	c.JSON(http.StatusOK, response)
}


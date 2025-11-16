// Package yzma provides local LLM inference via llama.cpp
// Version: v3.0.6+ - Model List Provider for Background Sync
package yzma

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// ModelListProvider provides model list for background caching
type ModelListProvider struct {
	client *Client
	logger *logrus.Logger
}

// NewModelListProvider creates a new model list provider
func NewModelListProvider(client *Client, logger *logrus.Logger) *ModelListProvider {
	return &ModelListProvider{
		client: client,
		logger: logger,
	}
}

// ModelInfo represents model information for API
type ModelInfo struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	OwnedBy string    `json:"owned_by"`
	Loaded  bool      `json:"loaded"`
	Path    string    `json:"path"`
	Size    int64     `json:"size,omitempty"`  // v3.0.7+: File size in bytes
}

// ListModels collects current model list for caching
func (p *ModelListProvider) ListModels(ctx context.Context) (interface{}, error) {
	// Get currently loaded models
	loadedModels := p.client.ListLoadedModels()
	
	// Convert to API format
	var models []*ModelInfo
	for path, modelInfo := range loadedModels {
		model := &ModelInfo{
			ID:      modelInfo["alias"].(string),  // Use alias as ID
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "local",
			Loaded:  true,
			Path:    path,  // Keep original path for debugging
		}
		
		// Add size if available
		if size, ok := modelInfo["size"].(int64); ok && size > 0 {
			model.Size = size
		}
		
		models = append(models, model)
	}
	
	p.logger.WithField("count", len(models)).Debug("Model list collected for caching")
	
	return models, nil
}


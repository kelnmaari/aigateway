package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aigateway/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDeepSeekProvider(t *testing.T) {
	t.Run("with custom base URL", func(t *testing.T) {
		p := NewDeepSeekProvider("test-deepseek", "https://custom.deepseek.com", "sk-test-key-123")

		assert.NotNil(t, p)
		assert.Equal(t, "test-deepseek", p.name)
		assert.Equal(t, "https://custom.deepseek.com", p.baseURL)
		assert.Equal(t, "sk-test-key-123", p.apiKey)
		assert.NotNil(t, p.client)
	})

	t.Run("with empty base URL defaults to deepseek", func(t *testing.T) {
		p := NewDeepSeekProvider("default-deepseek", "", "sk-test-key")

		assert.Equal(t, "https://api.deepseek.com", p.baseURL)
	})

	t.Run("trims trailing slash from base URL", func(t *testing.T) {
		p := NewDeepSeekProvider("test", "https://api.deepseek.com/", "sk-key")

		assert.Equal(t, "https://api.deepseek.com", p.baseURL)
	})
}

func TestDeepSeekProvider_GetName_GetType(t *testing.T) {
	p := NewDeepSeekProvider("my-deepseek", "https://api.deepseek.com", "sk-key")

	t.Run("GetName returns provider name", func(t *testing.T) {
		assert.Equal(t, "my-deepseek", p.GetName())
	})

	t.Run("GetType returns deepseek provider type", func(t *testing.T) {
		assert.Equal(t, models.ProviderTypeDeepSeek, p.GetType())
	})
}

func TestDeepSeekProvider_HealthCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/models", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer sk-test-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := openaiModelsResponse{
			Object: "list",
			Data: []struct {
				ID      string `json:"id"`
				Object  string `json:"object"`
				Created int64  `json:"created"`
				OwnedBy string `json:"owned_by"`
			}{
				{ID: "deepseek-chat", Object: "model", Created: 1000, OwnedBy: "deepseek"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewDeepSeekProvider("test", server.URL, "sk-test-key")
	err := p.HealthCheck(context.Background())

	require.NoError(t, err)
}

func TestDeepSeekProvider_HealthCheck_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"message": "Invalid API key",
				"type":    "invalid_request_error",
			},
		})
	}))
	defer server.Close()

	p := NewDeepSeekProvider("test", server.URL, "sk-invalid-key")
	err := p.HealthCheck(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestDeepSeekProvider_ListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/models", r.URL.Path)
		assert.Equal(t, "Bearer sk-test-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"object": "list",
			"data": []map[string]any{
				{
					"id":       "deepseek-chat",
					"object":   "model",
					"created":  1700000000,
					"owned_by": "deepseek",
				},
				{
					"id":       "deepseek-reasoner",
					"object":   "model",
					"created":  1700000001,
					"owned_by": "deepseek",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewDeepSeekProvider("test", server.URL, "sk-test-key")
	modelsList, err := p.ListModels(context.Background())

	require.NoError(t, err)
	require.Len(t, modelsList, 2)

	// Verify deepseek-chat
	assert.Equal(t, "deepseek-chat", modelsList[0].ID)
	assert.Equal(t, "deepseek-chat", modelsList[0].Name)
	assert.Contains(t, modelsList[0].Tags, "deepseek")
	assert.Equal(t, "deepseek", modelsList[0].ProviderMeta["owned_by"])
	assert.Contains(t, modelsList[0].Capabilities, models.CapabilityChat)
	assert.Contains(t, modelsList[0].Capabilities, models.CapabilityFunctionCalling)

	// Verify deepseek-reasoner
	assert.Equal(t, "deepseek-reasoner", modelsList[1].ID)
	assert.Contains(t, modelsList[1].Capabilities, models.CapabilityFunctionCalling)
}

func TestDeepSeekProvider_ListModels_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"object": "list",
			"data":   []any{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewDeepSeekProvider("test", server.URL, "sk-test-key")
	modelsList, err := p.ListModels(context.Background())

	require.NoError(t, err)
	assert.Empty(t, modelsList)
}

func TestDeepSeekProvider_GetModelInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/models/deepseek-chat", r.URL.Path)
		assert.Equal(t, "Bearer sk-test-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":       "deepseek-chat",
			"object":   "model",
			"created":  1700000000,
			"owned_by": "deepseek",
		})
	}))
	defer server.Close()

	p := NewDeepSeekProvider("test", server.URL, "sk-test-key")
	model, err := p.GetModelInfo(context.Background(), "deepseek-chat")

	require.NoError(t, err)
	assert.Equal(t, "deepseek-chat", model.ID)
	assert.Contains(t, model.Tags, "deepseek")
	assert.Contains(t, model.Description, "DeepSeek")
}

func TestInferDeepSeekCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		modelID  string
		wantCaps []models.ModelCapability
	}{
		{
			name:     "deepseek-chat has function calling",
			modelID:  "deepseek-chat",
			wantCaps: []models.ModelCapability{models.CapabilityChat, models.CapabilityFunctionCalling},
		},
		{
			name:     "deepseek-reasoner has function calling",
			modelID:  "deepseek-reasoner",
			wantCaps: []models.ModelCapability{models.CapabilityChat, models.CapabilityFunctionCalling},
		},
		{
			name:     "deepseek-v3 has function calling",
			modelID:  "deepseek-v3",
			wantCaps: []models.ModelCapability{models.CapabilityChat, models.CapabilityFunctionCalling},
		},
		{
			name:     "unknown deepseek model has basic chat",
			modelID:  "deepseek-coder",
			wantCaps: []models.ModelCapability{models.CapabilityChat},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := inferDeepSeekCapabilities(tt.modelID)
			for _, wantCap := range tt.wantCaps {
				assert.Contains(t, caps, wantCap)
			}
		})
	}
}

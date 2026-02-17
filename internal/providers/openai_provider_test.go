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

func TestNewOpenAIProvider(t *testing.T) {
	t.Run("with custom base URL", func(t *testing.T) {
		p := NewOpenAIProvider("test-openai", "https://custom.openai.com", "sk-test-key-123")

		assert.NotNil(t, p)
		assert.Equal(t, "test-openai", p.name)
		assert.Equal(t, "https://custom.openai.com", p.baseURL)
		assert.Equal(t, "sk-test-key-123", p.apiKey)
		assert.NotNil(t, p.client)
	})

	t.Run("with empty base URL defaults to openai", func(t *testing.T) {
		p := NewOpenAIProvider("default-openai", "", "sk-test-key")

		assert.Equal(t, "https://api.openai.com", p.baseURL)
	})

	t.Run("trims trailing slash from base URL", func(t *testing.T) {
		p := NewOpenAIProvider("test", "https://api.openai.com/", "sk-key")

		assert.Equal(t, "https://api.openai.com", p.baseURL)
	})
}

func TestOpenAIProvider_GetName_GetType(t *testing.T) {
	p := NewOpenAIProvider("my-openai", "https://api.openai.com", "sk-key")

	t.Run("GetName returns provider name", func(t *testing.T) {
		assert.Equal(t, "my-openai", p.GetName())
	})

	t.Run("GetType returns openai provider type", func(t *testing.T) {
		assert.Equal(t, models.ProviderTypeOpenAI, p.GetType())
	})
}

func TestOpenAIProvider_HealthCheck_Success(t *testing.T) {
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
				{ID: "gpt-4", Object: "model", Created: 1000, OwnedBy: "openai"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOpenAIProvider("test", server.URL, "sk-test-key")
	err := p.HealthCheck(context.Background())

	require.NoError(t, err)
}

func TestOpenAIProvider_HealthCheck_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "Invalid API key",
				"type":    "invalid_request_error",
			},
		})
	}))
	defer server.Close()

	p := NewOpenAIProvider("test", server.URL, "sk-invalid-key")
	err := p.HealthCheck(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestOpenAIProvider_ListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/models", r.URL.Path)
		assert.Equal(t, "Bearer sk-test-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"object": "list",
			"data": []map[string]interface{}{
				{
					"id":       "gpt-4",
					"object":   "model",
					"created":  1687882410,
					"owned_by": "openai",
				},
				{
					"id":       "gpt-3.5-turbo",
					"object":   "model",
					"created":  1677610602,
					"owned_by": "openai",
				},
				{
					"id":       "text-embedding-ada-002",
					"object":   "model",
					"created":  1671217299,
					"owned_by": "openai-internal",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOpenAIProvider("test", server.URL, "sk-test-key")
	modelsList, err := p.ListModels(context.Background())

	require.NoError(t, err)
	require.Len(t, modelsList, 3)

	// Verify gpt-4
	assert.Equal(t, "gpt-4", modelsList[0].ID)
	assert.Equal(t, "gpt-4", modelsList[0].Name)
	assert.Contains(t, modelsList[0].Tags, "openai")
	assert.Equal(t, "openai", modelsList[0].ProviderMeta["owned_by"])
	assert.Contains(t, modelsList[0].Capabilities, models.CapabilityChat)
	assert.Contains(t, modelsList[0].Capabilities, models.CapabilityVision)
	assert.Contains(t, modelsList[0].Capabilities, models.CapabilityFunctionCalling)

	// Verify gpt-3.5-turbo
	assert.Equal(t, "gpt-3.5-turbo", modelsList[1].ID)
	assert.Contains(t, modelsList[1].Capabilities, models.CapabilityFunctionCalling)

	// Verify embedding model
	assert.Equal(t, "text-embedding-ada-002", modelsList[2].ID)
	assert.Contains(t, modelsList[2].Capabilities, models.CapabilityEmbeddings)
}

func TestOpenAIProvider_ListModels_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"object": "list",
			"data":   []interface{}{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewOpenAIProvider("test", server.URL, "sk-test-key")
	modelsList, err := p.ListModels(context.Background())

	require.NoError(t, err)
	assert.Empty(t, modelsList)
}

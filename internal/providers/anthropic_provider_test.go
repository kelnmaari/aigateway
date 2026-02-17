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

func TestNewAnthropicProvider(t *testing.T) {
	t.Run("with custom base URL", func(t *testing.T) {
		p := NewAnthropicProvider("test-anthropic", "https://custom.anthropic.com", "sk-ant-test-key")

		assert.NotNil(t, p)
		assert.Equal(t, "test-anthropic", p.name)
		assert.Equal(t, "https://custom.anthropic.com", p.baseURL)
		assert.Equal(t, "sk-ant-test-key", p.apiKey)
		assert.NotNil(t, p.client)
	})

	t.Run("with empty base URL defaults to anthropic", func(t *testing.T) {
		p := NewAnthropicProvider("default", "", "sk-ant-key")

		assert.Equal(t, anthropicDefaultBaseURL, p.baseURL)
	})

	t.Run("trims trailing slash from base URL", func(t *testing.T) {
		p := NewAnthropicProvider("test", "https://api.anthropic.com/", "key")

		assert.Equal(t, "https://api.anthropic.com", p.baseURL)
	})

	t.Run("GetType returns anthropic", func(t *testing.T) {
		p := NewAnthropicProvider("test", "", "key")

		assert.Equal(t, models.ProviderTypeAnthropic, p.GetType())
	})
}

func TestAnthropicProvider_HealthCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/models", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		// Verify Anthropic-specific headers
		assert.Equal(t, "sk-ant-test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, anthropicAPIVersion, r.Header.Get("anthropic-version"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":           "claude-3-5-sonnet-20241022",
					"display_name": "Claude 3.5 Sonnet",
					"created_at":   "2024-10-22T00:00:00Z",
					"type":         "model",
				},
			},
			"has_more": false,
			"first_id": "claude-3-5-sonnet-20241022",
			"last_id":  "claude-3-5-sonnet-20241022",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewAnthropicProvider("test", server.URL, "sk-ant-test-key")
	err := p.HealthCheck(context.Background())

	require.NoError(t, err)
}

func TestAnthropicProvider_HealthCheck_InvalidKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"type": "authentication_error",
			"error": map[string]string{
				"type":    "authentication_error",
				"message": "invalid x-api-key",
			},
		})
	}))
	defer server.Close()

	p := NewAnthropicProvider("test", server.URL, "sk-ant-invalid")
	err := p.HealthCheck(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestAnthropicProvider_ListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/models", r.URL.Path)
		assert.Equal(t, "sk-ant-key", r.Header.Get("x-api-key"))
		assert.Equal(t, anthropicAPIVersion, r.Header.Get("anthropic-version"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":           "claude-3-5-sonnet-20241022",
					"display_name": "Claude 3.5 Sonnet",
					"created_at":   "2024-10-22T00:00:00Z",
					"type":         "model",
				},
				{
					"id":           "claude-3-opus-20240229",
					"display_name": "Claude 3 Opus",
					"created_at":   "2024-02-29T00:00:00Z",
					"type":         "model",
				},
				{
					"id":           "claude-2.1",
					"display_name": "Claude 2.1",
					"created_at":   "2023-11-21T00:00:00Z",
					"type":         "model",
				},
			},
			"has_more": false,
			"first_id": "claude-3-5-sonnet-20241022",
			"last_id":  "claude-2.1",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewAnthropicProvider("test", server.URL, "sk-ant-key")
	modelsList, err := p.ListModels(context.Background())

	require.NoError(t, err)
	require.Len(t, modelsList, 3)

	// Claude 3.5 Sonnet
	assert.Equal(t, "claude-3-5-sonnet-20241022", modelsList[0].ID)
	assert.Equal(t, "Claude 3.5 Sonnet", modelsList[0].Name)
	assert.Contains(t, modelsList[0].Tags, "anthropic")
	assert.Contains(t, modelsList[0].Tags, "claude")
	assert.Contains(t, modelsList[0].Capabilities, models.CapabilityChat)
	assert.Contains(t, modelsList[0].Capabilities, models.CapabilityFunctionCalling)
	assert.Contains(t, modelsList[0].Capabilities, models.CapabilityVision) // claude-3 model
	assert.Equal(t, "2024-10-22T00:00:00Z", modelsList[0].ProviderMeta["created_at"])
	assert.Equal(t, "model", modelsList[0].ProviderMeta["type"])

	// Claude 3 Opus
	assert.Equal(t, "claude-3-opus-20240229", modelsList[1].ID)
	assert.Equal(t, "Claude 3 Opus", modelsList[1].Name)
	assert.Contains(t, modelsList[1].Capabilities, models.CapabilityVision) // claude-3 model

	// Claude 2.1 (no vision)
	assert.Equal(t, "claude-2.1", modelsList[2].ID)
	assert.Equal(t, "Claude 2.1", modelsList[2].Name)
	assert.NotContains(t, modelsList[2].Capabilities, models.CapabilityVision)
}

func TestAnthropicProvider_HeaderVerification(t *testing.T) {
	var capturedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header.Clone()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]interface{}{
			"data":     []interface{}{},
			"has_more": false,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewAnthropicProvider("test", server.URL, "sk-ant-verify-headers")

	// Trigger a request via ListModels
	_, err := p.ListModels(context.Background())
	require.NoError(t, err)

	// Verify all required Anthropic headers are present
	require.NotNil(t, capturedHeaders)
	assert.Equal(t, "sk-ant-verify-headers", capturedHeaders.Get("x-api-key"), "x-api-key header must be present")
	assert.Equal(t, anthropicAPIVersion, capturedHeaders.Get("anthropic-version"), "anthropic-version header must be present")
	assert.Equal(t, "application/json", capturedHeaders.Get("Content-Type"), "Content-Type header must be present")

	// Verify Authorization header is NOT used (Anthropic uses x-api-key)
	assert.Empty(t, capturedHeaders.Get("Authorization"), "Authorization header must NOT be set for Anthropic")
}

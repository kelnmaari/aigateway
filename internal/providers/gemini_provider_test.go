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

func TestNewGeminiProvider(t *testing.T) {
	t.Run("with custom base URL", func(t *testing.T) {
		p := NewGeminiProvider("test-gemini", "https://custom.googleapis.com", "AIzaSy-test-key")

		assert.NotNil(t, p)
		assert.Equal(t, "test-gemini", p.name)
		assert.Equal(t, "https://custom.googleapis.com", p.baseURL)
		assert.Equal(t, "AIzaSy-test-key", p.apiKey)
		assert.NotNil(t, p.client)
	})

	t.Run("with empty base URL defaults to gemini", func(t *testing.T) {
		p := NewGeminiProvider("default", "", "key")

		assert.Equal(t, geminiDefaultBaseURL, p.baseURL)
	})

	t.Run("trims trailing slash from base URL", func(t *testing.T) {
		p := NewGeminiProvider("test", "https://generativelanguage.googleapis.com/", "key")

		assert.Equal(t, "https://generativelanguage.googleapis.com", p.baseURL)
	})

	t.Run("GetName returns provider name", func(t *testing.T) {
		p := NewGeminiProvider("my-gemini", "", "key")

		assert.Equal(t, "my-gemini", p.GetName())
	})

	t.Run("GetType returns gemini", func(t *testing.T) {
		p := NewGeminiProvider("test", "", "key")

		assert.Equal(t, models.ProviderTypeGemini, p.GetType())
	})
}

func TestGeminiProvider_HealthCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1beta/models", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		// Verify API key is in query params
		assert.Equal(t, "AIzaSy-test-key", r.URL.Query().Get("key"))
		assert.Equal(t, "1", r.URL.Query().Get("pageSize"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"models": []map[string]any{
				{
					"name":                       "models/gemini-1.5-pro",
					"displayName":                "Gemini 1.5 Pro",
					"description":                "Mid-size multimodal model",
					"version":                    "001",
					"inputTokenLimit":            1048576,
					"outputTokenLimit":           8192,
					"supportedGenerationMethods": []string{"generateContent", "countTokens"},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewGeminiProvider("test", server.URL, "AIzaSy-test-key")
	err := p.HealthCheck(context.Background())

	require.NoError(t, err)
}

func TestGeminiProvider_HealthCheck_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    403,
				"message": "API key not valid. Please pass a valid API key.",
				"status":  "PERMISSION_DENIED",
			},
		})
	}))
	defer server.Close()

	p := NewGeminiProvider("test", server.URL, "invalid-key")
	err := p.HealthCheck(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestGeminiProvider_ListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1beta/models", r.URL.Path)
		assert.Equal(t, "AIzaSy-key", r.URL.Query().Get("key"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"models": []map[string]any{
				{
					"name":                       "models/gemini-1.5-pro",
					"displayName":                "Gemini 1.5 Pro",
					"description":                "Mid-size multimodal model that supports up to 1 million tokens",
					"version":                    "001",
					"inputTokenLimit":            1048576,
					"outputTokenLimit":           8192,
					"supportedGenerationMethods": []string{"generateContent", "countTokens"},
				},
				{
					"name":                       "models/gemini-1.5-flash",
					"displayName":                "Gemini 1.5 Flash",
					"description":                "Fast and versatile multimodal model",
					"version":                    "001",
					"inputTokenLimit":            1048576,
					"outputTokenLimit":           8192,
					"supportedGenerationMethods": []string{"generateContent", "countTokens"},
				},
				{
					"name":                       "models/embedding-001",
					"displayName":                "Embedding 001",
					"description":                "Embedding model for text",
					"version":                    "001",
					"inputTokenLimit":            2048,
					"outputTokenLimit":           1,
					"supportedGenerationMethods": []string{"embedContent"},
				},
			},
			"nextPageToken": "",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewGeminiProvider("test", server.URL, "AIzaSy-key")
	modelsList, err := p.ListModels(context.Background())

	require.NoError(t, err)
	require.Len(t, modelsList, 3)

	// Gemini 1.5 Pro - should strip "models/" prefix
	assert.Equal(t, "gemini-1.5-pro", modelsList[0].ID)
	assert.Equal(t, "Gemini 1.5 Pro", modelsList[0].Name)
	assert.Contains(t, modelsList[0].Tags, "gemini")
	assert.Contains(t, modelsList[0].Tags, "google")
	assert.Contains(t, modelsList[0].Capabilities, models.CapabilityChat)
	assert.Contains(t, modelsList[0].Capabilities, models.CapabilityFunctionCalling)
	assert.Contains(t, modelsList[0].Capabilities, models.CapabilityVision)
	assert.NotNil(t, modelsList[0].ContextLength)
	assert.Equal(t, 1048576+8192, *modelsList[0].ContextLength)
	assert.Equal(t, "001", modelsList[0].ProviderMeta["version"])
	assert.Equal(t, 1048576, modelsList[0].ProviderMeta["input_token_limit"])
	assert.Equal(t, 8192, modelsList[0].ProviderMeta["output_token_limit"])

	// Gemini 1.5 Flash
	assert.Equal(t, "gemini-1.5-flash", modelsList[1].ID)
	assert.Equal(t, "Gemini 1.5 Flash", modelsList[1].Name)

	// Embedding model
	assert.Equal(t, "embedding-001", modelsList[2].ID)
	assert.Equal(t, "Embedding 001", modelsList[2].Name)
	assert.Contains(t, modelsList[2].Capabilities, models.CapabilityEmbeddings)
	assert.NotContains(t, modelsList[2].Capabilities, models.CapabilityChat)
}

func TestGeminiProvider_APIKeyInQuery(t *testing.T) {
	var capturedURL string
	var capturedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		capturedHeaders = r.Header.Clone()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"models": []any{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	apiKey := "AIzaSy-verify-query-param"
	p := NewGeminiProvider("test", server.URL, apiKey)

	_, err := p.ListModels(context.Background())
	require.NoError(t, err)

	// Verify API key is present in query parameters
	assert.Contains(t, capturedURL, "key="+apiKey, "API key must be in query parameters")

	// Verify API key is NOT in Authorization header (unlike OpenAI/Anthropic)
	assert.Empty(t, capturedHeaders.Get("Authorization"), "Authorization header must NOT be set for Gemini")
	assert.Empty(t, capturedHeaders.Get("x-api-key"), "x-api-key header must NOT be set for Gemini")
}

func TestGeminiProvider_ListModels_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"models": []any{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewGeminiProvider("test", server.URL, "key")
	modelsList, err := p.ListModels(context.Background())

	require.NoError(t, err)
	assert.Empty(t, modelsList)
}

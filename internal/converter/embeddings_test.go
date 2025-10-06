package converter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ollamaapi "ollama-openai-proxy/internal/client/ollama"
	"ollama-openai-proxy/internal/models"
)

func TestConvertEmbeddingRequest(t *testing.T) {
	t.Run("valid request with string input", func(t *testing.T) {
		openaiReq := &models.EmbeddingRequest{
			Model: "text-embedding-ada-002",
			Input: "Hello, world!",
		}

		ollamaReq, err := ConvertEmbeddingRequest(openaiReq)
		require.NoError(t, err)
		assert.NotNil(t, ollamaReq)
		assert.Equal(t, "text-embedding-ada-002", ollamaReq.Model)
		assert.Equal(t, "Hello, world!", ollamaReq.Input)
	})

	t.Run("valid request with string array input", func(t *testing.T) {
		openaiReq := &models.EmbeddingRequest{
			Model: "text-embedding-ada-002",
			Input: []string{"Hello", "World"},
		}

		ollamaReq, err := ConvertEmbeddingRequest(openaiReq)
		require.NoError(t, err)
		assert.NotNil(t, ollamaReq)
		assert.Equal(t, "text-embedding-ada-002", ollamaReq.Model)
		assert.Equal(t, []string{"Hello", "World"}, ollamaReq.Input)
	})

	t.Run("valid request with dimensions", func(t *testing.T) {
		dimensions := 512
		openaiReq := &models.EmbeddingRequest{
			Model:      "text-embedding-ada-002",
			Input:      "Hello, world!",
			Dimensions: &dimensions,
		}

		ollamaReq, err := ConvertEmbeddingRequest(openaiReq)
		require.NoError(t, err)
		assert.NotNil(t, ollamaReq)
		assert.Equal(t, 512, ollamaReq.Dimensions)
	})

	t.Run("nil request", func(t *testing.T) {
		ollamaReq, err := ConvertEmbeddingRequest(nil)
		assert.Error(t, err)
		assert.Nil(t, ollamaReq)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("empty model", func(t *testing.T) {
		openaiReq := &models.EmbeddingRequest{
			Model: "",
			Input: "Hello, world!",
		}

		ollamaReq, err := ConvertEmbeddingRequest(openaiReq)
		assert.Error(t, err)
		assert.Nil(t, ollamaReq)
		assert.Contains(t, err.Error(), "model is required")
	})

	t.Run("nil input", func(t *testing.T) {
		openaiReq := &models.EmbeddingRequest{
			Model: "text-embedding-ada-002",
			Input: nil,
		}

		ollamaReq, err := ConvertEmbeddingRequest(openaiReq)
		assert.Error(t, err)
		assert.Nil(t, ollamaReq)
		assert.Contains(t, err.Error(), "input is required")
	})

	t.Run("zero dimensions should not be set", func(t *testing.T) {
		dimensions := 0
		openaiReq := &models.EmbeddingRequest{
			Model:      "text-embedding-ada-002",
			Input:      "Hello, world!",
			Dimensions: &dimensions,
		}

		ollamaReq, err := ConvertEmbeddingRequest(openaiReq)
		require.NoError(t, err)
		assert.NotNil(t, ollamaReq)
		assert.Equal(t, 0, ollamaReq.Dimensions) // Zero dimensions не устанавливаются
	})
}

func TestConvertEmbeddingResponse(t *testing.T) {
	t.Run("valid response with single embedding", func(t *testing.T) {
		ollamaResp := &ollamaapi.EmbedResponse{
			Model: "text-embedding-ada-002",
			Embeddings: [][]float32{
				{0.1, 0.2, 0.3, 0.4, 0.5},
			},
			TotalDuration:   1000000000, // 1 second
			LoadDuration:    100000000,  // 0.1 second
			PromptEvalCount: 10,
		}

		openaiResp, err := ConvertEmbeddingResponse(ollamaResp, "text-embedding-ada-002")
		require.NoError(t, err)
		assert.NotNil(t, openaiResp)
		assert.Equal(t, "list", openaiResp.Object)
		assert.Equal(t, "text-embedding-ada-002", openaiResp.Model)
		assert.Len(t, openaiResp.Data, 1)

		// Проверяем первый embedding
		embedding := openaiResp.Data[0]
		assert.Equal(t, "embedding", embedding.Object)
		assert.Equal(t, 0, embedding.Index)
		assert.Len(t, embedding.Embedding, 5)

		// Проверяем конвертацию float32 -> float64
		assert.InDelta(t, 0.1, embedding.Embedding[0], 0.0001)
		assert.InDelta(t, 0.2, embedding.Embedding[1], 0.0001)
		assert.InDelta(t, 0.3, embedding.Embedding[2], 0.0001)
		assert.InDelta(t, 0.4, embedding.Embedding[3], 0.0001)
		assert.InDelta(t, 0.5, embedding.Embedding[4], 0.0001)

		// Проверяем usage
		assert.Equal(t, 10, openaiResp.Usage.PromptTokens)
		assert.Equal(t, 10, openaiResp.Usage.TotalTokens)
	})

	t.Run("valid response with multiple embeddings", func(t *testing.T) {
		ollamaResp := &ollamaapi.EmbedResponse{
			Model: "text-embedding-ada-002",
			Embeddings: [][]float32{
				{0.1, 0.2, 0.3},
				{0.4, 0.5, 0.6},
				{0.7, 0.8, 0.9},
			},
			PromptEvalCount: 30,
		}

		openaiResp, err := ConvertEmbeddingResponse(ollamaResp, "text-embedding-ada-002")
		require.NoError(t, err)
		assert.NotNil(t, openaiResp)
		assert.Len(t, openaiResp.Data, 3)

		// Проверяем индексы
		assert.Equal(t, 0, openaiResp.Data[0].Index)
		assert.Equal(t, 1, openaiResp.Data[1].Index)
		assert.Equal(t, 2, openaiResp.Data[2].Index)

		// Проверяем содержимое
		assert.InDelta(t, 0.1, openaiResp.Data[0].Embedding[0], 0.0001)
		assert.InDelta(t, 0.4, openaiResp.Data[1].Embedding[0], 0.0001)
		assert.InDelta(t, 0.7, openaiResp.Data[2].Embedding[0], 0.0001)

		// Проверяем usage (30 токенов для 3 embeddings)
		assert.Equal(t, 30, openaiResp.Usage.TotalTokens)
	})

	t.Run("valid response without prompt_eval_count", func(t *testing.T) {
		ollamaResp := &ollamaapi.EmbedResponse{
			Model: "text-embedding-ada-002",
			Embeddings: [][]float32{
				{0.1, 0.2, 0.3},
			},
			PromptEvalCount: 0, // Не задано
		}

		openaiResp, err := ConvertEmbeddingResponse(ollamaResp, "text-embedding-ada-002")
		require.NoError(t, err)
		assert.NotNil(t, openaiResp)

		// Должна быть эвристическая оценка
		assert.Greater(t, openaiResp.Usage.TotalTokens, 0)
	})

	t.Run("nil response", func(t *testing.T) {
		openaiResp, err := ConvertEmbeddingResponse(nil, "model")
		assert.Error(t, err)
		assert.Nil(t, openaiResp)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("empty embeddings array", func(t *testing.T) {
		ollamaResp := &ollamaapi.EmbedResponse{
			Model:      "text-embedding-ada-002",
			Embeddings: [][]float32{},
		}

		openaiResp, err := ConvertEmbeddingResponse(ollamaResp, "text-embedding-ada-002")
		require.NoError(t, err)
		assert.NotNil(t, openaiResp)
		assert.Len(t, openaiResp.Data, 0)
	})

	t.Run("large embedding vector", func(t *testing.T) {
		// Создаем большой embedding вектор (1536 dimensions как у OpenAI ada-002)
		largeEmbedding := make([]float32, 1536)
		for i := range largeEmbedding {
			largeEmbedding[i] = float32(i) * 0.001
		}

		ollamaResp := &ollamaapi.EmbedResponse{
			Model:           "text-embedding-ada-002",
			Embeddings:      [][]float32{largeEmbedding},
			PromptEvalCount: 100,
		}

		openaiResp, err := ConvertEmbeddingResponse(ollamaResp, "text-embedding-ada-002")
		require.NoError(t, err)
		assert.NotNil(t, openaiResp)
		assert.Len(t, openaiResp.Data, 1)
		assert.Len(t, openaiResp.Data[0].Embedding, 1536)

		// Проверяем точность конвертации
		assert.InDelta(t, 0.0, openaiResp.Data[0].Embedding[0], 0.0001)
		assert.InDelta(t, 0.001, openaiResp.Data[0].Embedding[1], 0.0001)
		assert.InDelta(t, 1.535, openaiResp.Data[0].Embedding[1535], 0.0001)
	})
}

func TestNormalizeInput(t *testing.T) {
	t.Run("string input", func(t *testing.T) {
		result, err := normalizeInput("Hello, world!")
		require.NoError(t, err)
		assert.Equal(t, []string{"Hello, world!"}, result)
	})

	t.Run("string array input", func(t *testing.T) {
		result, err := normalizeInput([]string{"Hello", "World"})
		require.NoError(t, err)
		assert.Equal(t, []string{"Hello", "World"}, result)
	})

	t.Run("interface array with strings", func(t *testing.T) {
		input := []interface{}{"Hello", "World", "Test"}
		result, err := normalizeInput(input)
		require.NoError(t, err)
		assert.Equal(t, []string{"Hello", "World", "Test"}, result)
	})

	t.Run("interface array with non-string", func(t *testing.T) {
		input := []interface{}{"Hello", 123, "World"}
		result, err := normalizeInput(input)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid input type")
	})

	t.Run("unsupported type", func(t *testing.T) {
		result, err := normalizeInput(12345)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "unsupported input type")
	})

	t.Run("empty string array", func(t *testing.T) {
		result, err := normalizeInput([]string{})
		require.NoError(t, err)
		assert.Equal(t, []string{}, result)
	})
}

// Benchmark тесты
func BenchmarkConvertEmbeddingRequest(b *testing.B) {
	openaiReq := &models.EmbeddingRequest{
		Model: "text-embedding-ada-002",
		Input: "Hello, world! This is a benchmark test.",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ConvertEmbeddingRequest(openaiReq)
	}
}

func BenchmarkConvertEmbeddingResponse(b *testing.B) {
	// Создаем embedding размером как у OpenAI ada-002
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = float32(i) * 0.001
	}

	ollamaResp := &ollamaapi.EmbedResponse{
		Model:           "text-embedding-ada-002",
		Embeddings:      [][]float32{embedding},
		PromptEvalCount: 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ConvertEmbeddingResponse(ollamaResp, "text-embedding-ada-002")
	}
}

func BenchmarkConvertEmbeddingResponseBatch(b *testing.B) {
	// Создаем batch из 10 embeddings
	embeddings := make([][]float32, 10)
	for i := range embeddings {
		embeddings[i] = make([]float32, 1536)
		for j := range embeddings[i] {
			embeddings[i][j] = float32(j) * 0.001
		}
	}

	ollamaResp := &ollamaapi.EmbedResponse{
		Model:           "text-embedding-ada-002",
		Embeddings:      embeddings,
		PromptEvalCount: 1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ConvertEmbeddingResponse(ollamaResp, "text-embedding-ada-002")
	}
}

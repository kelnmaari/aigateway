package converter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aigateway/internal/models"
)

func TestConvertCompletionToChatRequest(t *testing.T) {
	t.Run("valid request with string prompt", func(t *testing.T) {
		maxTokens := 100
		temperature := 0.7
		topP := 0.9

		req := &models.CompletionRequest{
			Model:       "gpt-3.5-turbo",
			Prompt:      "Hello, world!",
			MaxTokens:   &maxTokens,
			Temperature: &temperature,
			TopP:        &topP,
		}

		chatReq, err := ConvertCompletionToChatRequest(req)
		require.NoError(t, err)
		assert.NotNil(t, chatReq)
		assert.Equal(t, "gpt-3.5-turbo", chatReq.Model)
		assert.Len(t, chatReq.Messages, 1)
		assert.Equal(t, "user", chatReq.Messages[0].Role)
		assert.Equal(t, "Hello, world!", chatReq.Messages[0].Content)
		assert.Equal(t, &maxTokens, chatReq.MaxTokens)
		assert.Equal(t, &temperature, chatReq.Temperature)
		assert.Equal(t, &topP, chatReq.TopP)
	})

	t.Run("valid request with array prompt", func(t *testing.T) {
		req := &models.CompletionRequest{
			Model:  "gpt-3.5-turbo",
			Prompt: []string{"Hello", "World", "Test"},
		}

		chatReq, err := ConvertCompletionToChatRequest(req)
		require.NoError(t, err)
		assert.NotNil(t, chatReq)
		assert.Len(t, chatReq.Messages, 3)
		assert.Equal(t, "user", chatReq.Messages[0].Role)
		assert.Equal(t, "Hello", chatReq.Messages[0].Content)
		assert.Equal(t, "user", chatReq.Messages[1].Role)
		assert.Equal(t, "World", chatReq.Messages[1].Content)
		assert.Equal(t, "user", chatReq.Messages[2].Role)
		assert.Equal(t, "Test", chatReq.Messages[2].Content)
	})

	t.Run("valid request with stream", func(t *testing.T) {
		req := &models.CompletionRequest{
			Model:  "gpt-3.5-turbo",
			Prompt: "Hello",
			Stream: true,
		}

		chatReq, err := ConvertCompletionToChatRequest(req)
		require.NoError(t, err)
		assert.True(t, chatReq.Stream)
	})

	t.Run("valid request with stop sequences", func(t *testing.T) {
		req := &models.CompletionRequest{
			Model:  "gpt-3.5-turbo",
			Prompt: "Hello",
			Stop:   []string{"\n", "END"},
		}

		chatReq, err := ConvertCompletionToChatRequest(req)
		require.NoError(t, err)
		assert.Equal(t, []string{"\n", "END"}, chatReq.Stop)
	})

	t.Run("nil request", func(t *testing.T) {
		chatReq, err := ConvertCompletionToChatRequest(nil)
		assert.Error(t, err)
		assert.Nil(t, chatReq)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("empty model", func(t *testing.T) {
		req := &models.CompletionRequest{
			Model:  "",
			Prompt: "Hello",
		}

		chatReq, err := ConvertCompletionToChatRequest(req)
		assert.Error(t, err)
		assert.Nil(t, chatReq)
		assert.Contains(t, err.Error(), "model is required")
	})

	t.Run("nil prompt", func(t *testing.T) {
		req := &models.CompletionRequest{
			Model:  "gpt-3.5-turbo",
			Prompt: nil,
		}

		chatReq, err := ConvertCompletionToChatRequest(req)
		assert.Error(t, err)
		assert.Nil(t, chatReq)
	})

	t.Run("empty array prompt", func(t *testing.T) {
		req := &models.CompletionRequest{
			Model:  "gpt-3.5-turbo",
			Prompt: []string{},
		}

		chatReq, err := ConvertCompletionToChatRequest(req)
		assert.Error(t, err)
		assert.Nil(t, chatReq)
		assert.Contains(t, err.Error(), "empty")
	})
}

func TestConvertChatToCompletionResponse(t *testing.T) {
	t.Run("valid response with single choice", func(t *testing.T) {
		chatResp := &models.ChatCompletionResponse{
			ID:      "chatcmpl-123",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-3.5-turbo",
			Choices: []models.ChatCompletionChoice{
				{
					Index: 0,
					Message: models.ChatMessage{
						Role:    "assistant",
						Content: "Hello! How can I help you?",
					},
					FinishReason: "stop",
				},
			},
			Usage: models.Usage{
				PromptTokens:     10,
				CompletionTokens: 8,
				TotalTokens:      18,
			},
		}

		completionResp, err := ConvertChatToCompletionResponse(chatResp, "text-davinci-003")
		require.NoError(t, err)
		assert.NotNil(t, completionResp)
		assert.Equal(t, "chatcmpl-123", completionResp.ID)
		assert.Equal(t, "text_completion", completionResp.Object)
		assert.Equal(t, int64(1234567890), completionResp.Created)
		assert.Equal(t, "text-davinci-003", completionResp.Model)
		assert.Len(t, completionResp.Choices, 1)

		choice := completionResp.Choices[0]
		assert.Equal(t, "Hello! How can I help you?", choice.Text)
		assert.Equal(t, 0, choice.Index)
		assert.Equal(t, "stop", choice.FinishReason)

		assert.Equal(t, 10, completionResp.Usage.PromptTokens)
		assert.Equal(t, 8, completionResp.Usage.CompletionTokens)
		assert.Equal(t, 18, completionResp.Usage.TotalTokens)
	})

	t.Run("valid response with multiple choices", func(t *testing.T) {
		chatResp := &models.ChatCompletionResponse{
			ID:      "chatcmpl-456",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-4",
			Choices: []models.ChatCompletionChoice{
				{
					Index: 0,
					Message: models.ChatMessage{
						Role:    "assistant",
						Content: "First response",
					},
					FinishReason: "stop",
				},
				{
					Index: 1,
					Message: models.ChatMessage{
						Role:    "assistant",
						Content: "Second response",
					},
					FinishReason: "stop",
				},
			},
			Usage: models.Usage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}

		completionResp, err := ConvertChatToCompletionResponse(chatResp, "text-davinci-003")
		require.NoError(t, err)
		assert.Len(t, completionResp.Choices, 2)
		assert.Equal(t, "First response", completionResp.Choices[0].Text)
		assert.Equal(t, "Second response", completionResp.Choices[1].Text)
		assert.Equal(t, 0, completionResp.Choices[0].Index)
		assert.Equal(t, 1, completionResp.Choices[1].Index)
	})

	t.Run("response with nil content", func(t *testing.T) {
		chatResp := &models.ChatCompletionResponse{
			ID:      "chatcmpl-789",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-3.5-turbo",
			Choices: []models.ChatCompletionChoice{
				{
					Index: 0,
					Message: models.ChatMessage{
						Role:    "assistant",
						Content: nil, // Nil content
					},
					FinishReason: "stop",
				},
			},
			Usage: models.Usage{},
		}

		completionResp, err := ConvertChatToCompletionResponse(chatResp, "text-davinci-003")
		require.NoError(t, err)
		assert.Equal(t, "", completionResp.Choices[0].Text) // Должен быть пустой string
	})

	t.Run("nil response", func(t *testing.T) {
		completionResp, err := ConvertChatToCompletionResponse(nil, "model")
		assert.Error(t, err)
		assert.Nil(t, completionResp)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("empty choices", func(t *testing.T) {
		chatResp := &models.ChatCompletionResponse{
			ID:      "chatcmpl-empty",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-3.5-turbo",
			Choices: []models.ChatCompletionChoice{},
			Usage:   models.Usage{},
		}

		completionResp, err := ConvertChatToCompletionResponse(chatResp, "text-davinci-003")
		require.NoError(t, err)
		assert.Len(t, completionResp.Choices, 0)
	})
}

func TestPromptToMessages(t *testing.T) {
	t.Run("string prompt", func(t *testing.T) {
		messages, err := promptToMessages("Hello, world!")
		require.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Equal(t, "user", messages[0].Role)
		assert.Equal(t, "Hello, world!", messages[0].Content)
	})

	t.Run("string array prompt", func(t *testing.T) {
		messages, err := promptToMessages([]string{"First", "Second", "Third"})
		require.NoError(t, err)
		assert.Len(t, messages, 3)
		assert.Equal(t, "user", messages[0].Role)
		assert.Equal(t, "First", messages[0].Content)
		assert.Equal(t, "user", messages[1].Role)
		assert.Equal(t, "Second", messages[1].Content)
		assert.Equal(t, "user", messages[2].Role)
		assert.Equal(t, "Third", messages[2].Content)
	})

	t.Run("interface array prompt", func(t *testing.T) {
		prompt := []interface{}{"Hello", "World"}
		messages, err := promptToMessages(prompt)
		require.NoError(t, err)
		assert.Len(t, messages, 2)
		assert.Equal(t, "Hello", messages[0].Content)
		assert.Equal(t, "World", messages[1].Content)
	})

	t.Run("interface array with non-string", func(t *testing.T) {
		prompt := []interface{}{"Hello", 123, "World"}
		messages, err := promptToMessages(prompt)
		assert.Error(t, err)
		assert.Nil(t, messages)
		assert.Contains(t, err.Error(), "non-string")
	})

	t.Run("nil prompt", func(t *testing.T) {
		messages, err := promptToMessages(nil)
		assert.Error(t, err)
		assert.Nil(t, messages)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("empty string array", func(t *testing.T) {
		messages, err := promptToMessages([]string{})
		assert.Error(t, err)
		assert.Nil(t, messages)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("empty interface array", func(t *testing.T) {
		messages, err := promptToMessages([]interface{}{})
		assert.Error(t, err)
		assert.Nil(t, messages)
		assert.Contains(t, err.Error(), "empty")
	})

	t.Run("unsupported type", func(t *testing.T) {
		messages, err := promptToMessages(12345)
		assert.Error(t, err)
		assert.Nil(t, messages)
		assert.Contains(t, err.Error(), "unsupported")
	})

	t.Run("complex prompt", func(t *testing.T) {
		prompt := []string{
			"You are a helpful assistant.",
			"User: What is the capital of France?",
			"Assistant: ",
		}
		messages, err := promptToMessages(prompt)
		require.NoError(t, err)
		assert.Len(t, messages, 3)
		for _, msg := range messages {
			assert.Equal(t, "user", msg.Role)
		}
	})
}

func TestConvertCompletionStreamChunk(t *testing.T) {
	t.Run("valid chunk with content", func(t *testing.T) {
		finishReason := "stop"
		chatChunk := &models.ChatCompletionChunk{
			ID:      "chatcmpl-stream-123",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   "gpt-3.5-turbo",
			Choices: []models.ChatCompletionChunkChoice{
				{
					Index: 0,
					Delta: models.ChatMessage{
						Content: "Hello",
					},
					FinishReason: &finishReason,
				},
			},
		}

		chunk, err := ConvertCompletionStreamChunk(chatChunk, "text-davinci-003")
		require.NoError(t, err)
		assert.NotNil(t, chunk)
		assert.Equal(t, "chatcmpl-stream-123", chunk.ID)
		assert.Equal(t, "text_completion", chunk.Object)
		assert.Equal(t, int64(1234567890), chunk.Created)
		assert.Equal(t, "text-davinci-003", chunk.Model)
		assert.Len(t, chunk.Choices, 1)
		assert.Equal(t, "Hello", chunk.Choices[0].Text)
		assert.Equal(t, 0, chunk.Choices[0].Index)
		assert.Equal(t, "stop", chunk.Choices[0].FinishReason)
	})

	t.Run("chunk without finish reason", func(t *testing.T) {
		chatChunk := &models.ChatCompletionChunk{
			ID:      "chatcmpl-stream-456",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   "gpt-4",
			Choices: []models.ChatCompletionChunkChoice{
				{
					Index: 0,
					Delta: models.ChatMessage{
						Content: "World",
					},
					FinishReason: nil, // Нет finish reason
				},
			},
		}

		chunk, err := ConvertCompletionStreamChunk(chatChunk, "text-davinci-003")
		require.NoError(t, err)
		assert.Equal(t, "", chunk.Choices[0].FinishReason)
	})

	t.Run("chunk with nil content", func(t *testing.T) {
		chatChunk := &models.ChatCompletionChunk{
			ID:      "chatcmpl-stream-789",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   "gpt-3.5-turbo",
			Choices: []models.ChatCompletionChunkChoice{
				{
					Index: 0,
					Delta: models.ChatMessage{
						Content: nil, // Nil content
					},
					FinishReason: nil,
				},
			},
		}

		chunk, err := ConvertCompletionStreamChunk(chatChunk, "text-davinci-003")
		require.NoError(t, err)
		assert.Equal(t, "", chunk.Choices[0].Text)
	})

	t.Run("nil chunk", func(t *testing.T) {
		chunk, err := ConvertCompletionStreamChunk(nil, "model")
		assert.Error(t, err)
		assert.Nil(t, chunk)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("chunk with multiple choices", func(t *testing.T) {
		chatChunk := &models.ChatCompletionChunk{
			ID:      "chatcmpl-multi",
			Object:  "chat.completion.chunk",
			Created: 1234567890,
			Model:   "gpt-4",
			Choices: []models.ChatCompletionChunkChoice{
				{
					Index: 0,
					Delta: models.ChatMessage{
						Content: "First",
					},
					FinishReason: nil,
				},
				{
					Index: 1,
					Delta: models.ChatMessage{
						Content: "Second",
					},
					FinishReason: nil,
				},
			},
		}

		chunk, err := ConvertCompletionStreamChunk(chatChunk, "text-davinci-003")
		require.NoError(t, err)
		assert.Len(t, chunk.Choices, 2)
		assert.Equal(t, "First", chunk.Choices[0].Text)
		assert.Equal(t, "Second", chunk.Choices[1].Text)
	})
}

// Benchmark тесты
func BenchmarkConvertCompletionToChatRequest(b *testing.B) {
	req := &models.CompletionRequest{
		Model:  "gpt-3.5-turbo",
		Prompt: "Hello, world! This is a benchmark test for completion conversion.",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ConvertCompletionToChatRequest(req)
	}
}

func BenchmarkConvertChatToCompletionResponse(b *testing.B) {
	chatResp := &models.ChatCompletionResponse{
		ID:      "chatcmpl-bench",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-3.5-turbo",
		Choices: []models.ChatCompletionChoice{
			{
				Index: 0,
				Message: models.ChatMessage{
					Role:    "assistant",
					Content: "This is a benchmark response with some content to test conversion performance.",
				},
				FinishReason: "stop",
			},
		},
		Usage: models.Usage{
			PromptTokens:     20,
			CompletionTokens: 15,
			TotalTokens:      35,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ConvertChatToCompletionResponse(chatResp, "text-davinci-003")
	}
}

func BenchmarkPromptToMessages(b *testing.B) {
	prompt := []string{
		"You are a helpful assistant.",
		"User: What is the capital of France?",
		"User: Please provide a detailed answer.",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = promptToMessages(prompt)
	}
}


package converter

import (
	"encoding/json"
	"testing"

	"aigateway/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIToAnthropic_BasicMessages(t *testing.T) {
	req := &models.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
			{Role: "user", Content: "How are you?"},
		},
	}

	result, err := OpenAIToAnthropic(req, "claude-3-sonnet-20240229")
	require.NoError(t, err)

	assert.Equal(t, "claude-3-sonnet-20240229", result.Model)
	assert.Empty(t, result.System)
	require.Len(t, result.Messages, 3)

	assert.Equal(t, "user", result.Messages[0].Role)
	assert.Equal(t, "Hello", result.Messages[0].Content)

	assert.Equal(t, "assistant", result.Messages[1].Role)
	assert.Equal(t, "Hi there!", result.Messages[1].Content)

	assert.Equal(t, "user", result.Messages[2].Role)
	assert.Equal(t, "How are you?", result.Messages[2].Content)
}

func TestOpenAIToAnthropic_SystemMessage(t *testing.T) {
	req := &models.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []models.ChatMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello"},
		},
	}

	result, err := OpenAIToAnthropic(req, "claude-3-sonnet-20240229")
	require.NoError(t, err)

	assert.Equal(t, "You are a helpful assistant.", result.System)
	require.Len(t, result.Messages, 1)
	assert.Equal(t, "user", result.Messages[0].Role)
	assert.Equal(t, "Hello", result.Messages[0].Content)
}

func TestOpenAIToAnthropic_MultipleSystemMessages(t *testing.T) {
	req := &models.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []models.ChatMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "system", Content: "Always respond in English."},
			{Role: "user", Content: "Hi"},
		},
	}

	result, err := OpenAIToAnthropic(req, "claude-3-sonnet-20240229")
	require.NoError(t, err)

	assert.Equal(t, "You are a helpful assistant.\n\nAlways respond in English.", result.System)
	require.Len(t, result.Messages, 1)
	assert.Equal(t, "user", result.Messages[0].Role)
}

func TestOpenAIToAnthropic_EmptyMessages(t *testing.T) {
	tests := []struct {
		name     string
		messages []models.ChatMessage
		wantErr  string
	}{
		{
			name:     "nil messages",
			messages: nil,
			wantErr:  "messages array is empty",
		},
		{
			name:     "empty slice",
			messages: []models.ChatMessage{},
			wantErr:  "messages array is empty",
		},
		{
			name: "only system messages",
			messages: []models.ChatMessage{
				{Role: "system", Content: "You are a bot."},
			},
			wantErr: "no user or assistant messages after filtering system messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &models.ChatCompletionRequest{
				Model:    "gpt-4",
				Messages: tt.messages,
			}
			_, err := OpenAIToAnthropic(req, "claude-3-sonnet-20240229")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestOpenAIToAnthropic_Parameters(t *testing.T) {
	temp := 0.7
	topP := 0.9
	maxTokens := 1024

	req := &models.ChatCompletionRequest{
		Model:       "gpt-4",
		Messages:    []models.ChatMessage{{Role: "user", Content: "Hi"}},
		Temperature: &temp,
		TopP:        &topP,
		MaxTokens:   &maxTokens,
		Stop:        []string{"END", "STOP"},
		Stream:      true,
	}

	result, err := OpenAIToAnthropic(req, "claude-3-sonnet-20240229")
	require.NoError(t, err)

	assert.Equal(t, 1024, result.MaxTokens)
	require.NotNil(t, result.Temperature)
	assert.InDelta(t, 0.7, *result.Temperature, 0.001)
	require.NotNil(t, result.TopP)
	assert.InDelta(t, 0.9, *result.TopP, 0.001)
	assert.Equal(t, []string{"END", "STOP"}, result.StopSequences)
	assert.True(t, result.Stream)
}

func TestOpenAIToAnthropic_DefaultMaxTokens(t *testing.T) {
	req := &models.ChatCompletionRequest{
		Model:    "gpt-4",
		Messages: []models.ChatMessage{{Role: "user", Content: "Hi"}},
	}

	result, err := OpenAIToAnthropic(req, "claude-3-sonnet-20240229")
	require.NoError(t, err)

	assert.Equal(t, 4096, result.MaxTokens, "default max_tokens should be 4096")
}

func TestAnthropicToOpenAI_TextContent(t *testing.T) {
	resp := &AnthropicResponse{
		ID:   "msg_01XFG",
		Type: "message",
		Role: "assistant",
		Content: []AnthropicContentBlock{
			{Type: "text", Text: "Hello! How can I help you today?"},
		},
		Model:      "claude-3-sonnet-20240229",
		StopReason: "end_turn",
		Usage: AnthropicUsage{
			InputTokens:  10,
			OutputTokens: 25,
		},
	}

	result := AnthropicToOpenAI(resp, "gpt-4")

	assert.Equal(t, "msg_01XFG", result.ID)
	assert.Equal(t, "chat.completion", result.Object)
	assert.Equal(t, "gpt-4", result.Model)
	require.Len(t, result.Choices, 1)
	assert.Equal(t, 0, result.Choices[0].Index)
	assert.Equal(t, "assistant", result.Choices[0].Message.Role)
	assert.Equal(t, "Hello! How can I help you today?", result.Choices[0].Message.Content)
	assert.Equal(t, "stop", result.Choices[0].FinishReason)
}

func TestAnthropicToOpenAI_MultipleContentBlocks(t *testing.T) {
	resp := &AnthropicResponse{
		ID:   "msg_02ABC",
		Type: "message",
		Role: "assistant",
		Content: []AnthropicContentBlock{
			{Type: "text", Text: "First part. "},
			{Type: "text", Text: "Second part."},
		},
		Model:      "claude-3-sonnet-20240229",
		StopReason: "end_turn",
		Usage: AnthropicUsage{
			InputTokens:  5,
			OutputTokens: 10,
		},
	}

	result := AnthropicToOpenAI(resp, "gpt-4")

	require.Len(t, result.Choices, 1)
	assert.Equal(t, "First part. Second part.", result.Choices[0].Message.Content)
}

func TestAnthropicToOpenAI_StopReason(t *testing.T) {
	tests := []struct {
		name           string
		stopReason     string
		expectedFinish string
	}{
		{"end_turn maps to stop", "end_turn", "stop"},
		{"max_tokens maps to length", "max_tokens", "length"},
		{"stop_sequence maps to stop", "stop_sequence", "stop"},
		{"unknown maps to stop", "unknown_reason", "stop"},
		{"empty maps to stop", "", "stop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &AnthropicResponse{
				ID:   "msg_test",
				Type: "message",
				Role: "assistant",
				Content: []AnthropicContentBlock{
					{Type: "text", Text: "response"},
				},
				Model:      "claude-3-sonnet-20240229",
				StopReason: tt.stopReason,
				Usage:      AnthropicUsage{InputTokens: 1, OutputTokens: 1},
			}

			result := AnthropicToOpenAI(resp, "gpt-4")
			assert.Equal(t, tt.expectedFinish, result.Choices[0].FinishReason)
		})
	}
}

func TestAnthropicToOpenAI_Usage(t *testing.T) {
	resp := &AnthropicResponse{
		ID:   "msg_usage",
		Type: "message",
		Role: "assistant",
		Content: []AnthropicContentBlock{
			{Type: "text", Text: "test"},
		},
		Model:      "claude-3-sonnet-20240229",
		StopReason: "end_turn",
		Usage: AnthropicUsage{
			InputTokens:  150,
			OutputTokens: 250,
		},
	}

	result := AnthropicToOpenAI(resp, "gpt-4")

	assert.Equal(t, 150, result.Usage.PromptTokens)
	assert.Equal(t, 250, result.Usage.CompletionTokens)
	assert.Equal(t, 400, result.Usage.TotalTokens)
}

func TestConvertAnthropicSSE_ContentBlockDelta(t *testing.T) {
	event := AnthropicContentBlockDelta{
		Type:  "content_block_delta",
		Index: 0,
	}
	event.Delta.Type = "text_delta"
	event.Delta.Text = "Hello world"

	data, err := json.Marshal(event)
	require.NoError(t, err)

	result, isFinal, err := ConvertAnthropicSSE("content_block_delta", data, "gpt-4", "chatcmpl-123")
	require.NoError(t, err)
	assert.False(t, isFinal)
	assert.NotEmpty(t, result)

	var chunk models.ChatCompletionChunk
	err = json.Unmarshal([]byte(result), &chunk)
	require.NoError(t, err)

	assert.Equal(t, "chatcmpl-123", chunk.ID)
	assert.Equal(t, "chat.completion.chunk", chunk.Object)
	assert.Equal(t, "gpt-4", chunk.Model)
	require.Len(t, chunk.Choices, 1)
	assert.Equal(t, 0, chunk.Choices[0].Index)
	assert.Equal(t, "Hello world", chunk.Choices[0].Delta.Content)
}

func TestConvertAnthropicSSE_MessageStop(t *testing.T) {
	result, isFinal, err := ConvertAnthropicSSE("message_stop", []byte("{}"), "gpt-4", "chatcmpl-123")
	require.NoError(t, err)
	assert.True(t, isFinal)
	assert.Equal(t, "[DONE]", result)
}

func TestConvertAnthropicSSE_MessageStart(t *testing.T) {
	event := AnthropicMessageStart{
		Type: "message_start",
		Message: AnthropicResponse{
			ID:    "msg_start_01",
			Type:  "message",
			Role:  "assistant",
			Model: "claude-3-sonnet-20240229",
			Usage: AnthropicUsage{InputTokens: 10, OutputTokens: 0},
		},
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	result, isFinal, err := ConvertAnthropicSSE("message_start", data, "gpt-4", "chatcmpl-456")
	require.NoError(t, err)
	assert.False(t, isFinal)
	assert.NotEmpty(t, result)

	var chunk models.ChatCompletionChunk
	err = json.Unmarshal([]byte(result), &chunk)
	require.NoError(t, err)

	assert.Equal(t, "chatcmpl-456", chunk.ID)
	assert.Equal(t, "chat.completion.chunk", chunk.Object)
	assert.Equal(t, "gpt-4", chunk.Model)
	require.Len(t, chunk.Choices, 1)
	assert.Equal(t, "assistant", chunk.Choices[0].Delta.Role)
}

func TestConvertAnthropicSSE_MessageDelta(t *testing.T) {
	event := AnthropicMessageDelta{
		Type: "message_delta",
		Usage: AnthropicUsage{
			InputTokens:  0,
			OutputTokens: 50,
		},
	}
	event.Delta.StopReason = "end_turn"

	data, err := json.Marshal(event)
	require.NoError(t, err)

	result, isFinal, err := ConvertAnthropicSSE("message_delta", data, "gpt-4", "chatcmpl-789")
	require.NoError(t, err)
	assert.False(t, isFinal)
	assert.NotEmpty(t, result)

	var chunk models.ChatCompletionChunk
	err = json.Unmarshal([]byte(result), &chunk)
	require.NoError(t, err)

	require.Len(t, chunk.Choices, 1)
	require.NotNil(t, chunk.Choices[0].FinishReason)
	assert.Equal(t, "stop", *chunk.Choices[0].FinishReason)
}

func TestConvertAnthropicSSE_SkippedEvents(t *testing.T) {
	events := []string{"ping", "content_block_start", "content_block_stop"}

	for _, eventType := range events {
		t.Run(eventType, func(t *testing.T) {
			result, isFinal, err := ConvertAnthropicSSE(eventType, []byte("{}"), "gpt-4", "chatcmpl-123")
			require.NoError(t, err)
			assert.False(t, isFinal)
			assert.Empty(t, result)
		})
	}
}

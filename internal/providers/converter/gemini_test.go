package converter

import (
	"encoding/json"
	"testing"

	"aigateway/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIToGemini_BasicMessages(t *testing.T) {
	req := &models.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
			{Role: "user", Content: "How are you?"},
		},
	}

	result, err := OpenAIToGemini(req)
	require.NoError(t, err)

	assert.Nil(t, result.SystemInstruction)
	require.Len(t, result.Contents, 3)

	assert.Equal(t, "user", result.Contents[0].Role)
	require.Len(t, result.Contents[0].Parts, 1)
	assert.Equal(t, "Hello", result.Contents[0].Parts[0].Text)

	assert.Equal(t, "model", result.Contents[1].Role)
	require.Len(t, result.Contents[1].Parts, 1)
	assert.Equal(t, "Hi there!", result.Contents[1].Parts[0].Text)

	assert.Equal(t, "user", result.Contents[2].Role)
	require.Len(t, result.Contents[2].Parts, 1)
	assert.Equal(t, "How are you?", result.Contents[2].Parts[0].Text)
}

func TestOpenAIToGemini_SystemInstruction(t *testing.T) {
	req := &models.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []models.ChatMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello"},
		},
	}

	result, err := OpenAIToGemini(req)
	require.NoError(t, err)

	require.NotNil(t, result.SystemInstruction)
	require.Len(t, result.SystemInstruction.Parts, 1)
	assert.Equal(t, "You are a helpful assistant.", result.SystemInstruction.Parts[0].Text)

	require.Len(t, result.Contents, 1)
	assert.Equal(t, "user", result.Contents[0].Role)
}

func TestOpenAIToGemini_MultipleSystemMessages(t *testing.T) {
	req := &models.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []models.ChatMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "system", Content: "Always respond in English."},
			{Role: "user", Content: "Hi"},
		},
	}

	result, err := OpenAIToGemini(req)
	require.NoError(t, err)

	require.NotNil(t, result.SystemInstruction)
	require.Len(t, result.SystemInstruction.Parts, 2)
	assert.Equal(t, "You are a helpful assistant.", result.SystemInstruction.Parts[0].Text)
	assert.Equal(t, "Always respond in English.", result.SystemInstruction.Parts[1].Text)
}

func TestOpenAIToGemini_RoleMapping(t *testing.T) {
	req := &models.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Question"},
			{Role: "assistant", Content: "Answer"},
		},
	}

	result, err := OpenAIToGemini(req)
	require.NoError(t, err)

	require.Len(t, result.Contents, 2)
	assert.Equal(t, "user", result.Contents[0].Role, "user role should stay 'user'")
	assert.Equal(t, "model", result.Contents[1].Role, "assistant role should map to 'model'")
}

func TestOpenAIToGemini_EmptyMessages(t *testing.T) {
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
			_, err := OpenAIToGemini(req)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestOpenAIToGemini_GenerationConfig(t *testing.T) {
	temp := 0.8
	topP := 0.95
	maxTokens := 2048

	req := &models.ChatCompletionRequest{
		Model:       "gpt-4",
		Messages:    []models.ChatMessage{{Role: "user", Content: "Hi"}},
		Temperature: &temp,
		TopP:        &topP,
		MaxTokens:   &maxTokens,
		Stop:        []string{"END"},
	}

	result, err := OpenAIToGemini(req)
	require.NoError(t, err)

	require.NotNil(t, result.GenerationConfig)
	require.NotNil(t, result.GenerationConfig.Temperature)
	assert.InDelta(t, 0.8, *result.GenerationConfig.Temperature, 0.001)
	require.NotNil(t, result.GenerationConfig.TopP)
	assert.InDelta(t, 0.95, *result.GenerationConfig.TopP, 0.001)
	require.NotNil(t, result.GenerationConfig.MaxOutputTokens)
	assert.Equal(t, 2048, *result.GenerationConfig.MaxOutputTokens)
	assert.Equal(t, []string{"END"}, result.GenerationConfig.StopSequences)
}

func TestGeminiToOpenAI_Candidates(t *testing.T) {
	resp := &GeminiResponse{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Role: "model",
					Parts: []GeminiPart{
						{Text: "Hello! How can I help you?"},
					},
				},
				FinishReason: "STOP",
				Index:        0,
			},
		},
		UsageMetadata: &GeminiUsageMetadata{
			PromptTokenCount:     10,
			CandidatesTokenCount: 20,
			TotalTokenCount:      30,
		},
	}

	result := GeminiToOpenAI(resp, "gpt-4")

	assert.Equal(t, "chat.completion", result.Object)
	assert.Equal(t, "gpt-4", result.Model)
	require.Len(t, result.Choices, 1)
	assert.Equal(t, 0, result.Choices[0].Index)
	assert.Equal(t, "assistant", result.Choices[0].Message.Role)
	assert.Equal(t, "Hello! How can I help you?", result.Choices[0].Message.Content)
	assert.Equal(t, "stop", result.Choices[0].FinishReason)
}

func TestGeminiToOpenAI_MultipleParts(t *testing.T) {
	resp := &GeminiResponse{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Role: "model",
					Parts: []GeminiPart{
						{Text: "Part one. "},
						{Text: "Part two."},
					},
				},
				FinishReason: "STOP",
				Index:        0,
			},
		},
	}

	result := GeminiToOpenAI(resp, "gpt-4")

	require.Len(t, result.Choices, 1)
	assert.Equal(t, "Part one. Part two.", result.Choices[0].Message.Content)
}

func TestGeminiToOpenAI_FinishReason(t *testing.T) {
	tests := []struct {
		name           string
		finishReason   string
		expectedFinish string
	}{
		{"STOP maps to stop", "STOP", "stop"},
		{"MAX_TOKENS maps to length", "MAX_TOKENS", "length"},
		{"SAFETY maps to content_filter", "SAFETY", "content_filter"},
		{"RECITATION maps to content_filter", "RECITATION", "content_filter"},
		{"unknown maps to stop", "UNKNOWN", "stop"},
		{"empty maps to stop", "", "stop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &GeminiResponse{
				Candidates: []GeminiCandidate{
					{
						Content: GeminiContent{
							Role:  "model",
							Parts: []GeminiPart{{Text: "response"}},
						},
						FinishReason: tt.finishReason,
						Index:        0,
					},
				},
			}

			result := GeminiToOpenAI(resp, "gpt-4")
			assert.Equal(t, tt.expectedFinish, result.Choices[0].FinishReason)
		})
	}
}

func TestGeminiToOpenAI_Usage(t *testing.T) {
	resp := &GeminiResponse{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Role:  "model",
					Parts: []GeminiPart{{Text: "test"}},
				},
				FinishReason: "STOP",
				Index:        0,
			},
		},
		UsageMetadata: &GeminiUsageMetadata{
			PromptTokenCount:     100,
			CandidatesTokenCount: 200,
			TotalTokenCount:      300,
		},
	}

	result := GeminiToOpenAI(resp, "gpt-4")

	assert.Equal(t, 100, result.Usage.PromptTokens)
	assert.Equal(t, 200, result.Usage.CompletionTokens)
	assert.Equal(t, 300, result.Usage.TotalTokens)
}

func TestGeminiToOpenAI_NoUsageMetadata(t *testing.T) {
	resp := &GeminiResponse{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Role:  "model",
					Parts: []GeminiPart{{Text: "test"}},
				},
				FinishReason: "STOP",
				Index:        0,
			},
		},
		UsageMetadata: nil,
	}

	result := GeminiToOpenAI(resp, "gpt-4")

	assert.Equal(t, 0, result.Usage.PromptTokens)
	assert.Equal(t, 0, result.Usage.CompletionTokens)
	assert.Equal(t, 0, result.Usage.TotalTokens)
}

func TestConvertGeminiSSE_Candidate(t *testing.T) {
	chunk := GeminiStreamChunk{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Role: "model",
					Parts: []GeminiPart{
						{Text: "Hello streaming"},
					},
				},
				Index: 0,
			},
		},
	}

	data, err := json.Marshal(chunk)
	require.NoError(t, err)

	result, isFinal, err := ConvertGeminiSSE(data, "gpt-4", "chatcmpl-stream-1")
	require.NoError(t, err)
	assert.False(t, isFinal)
	assert.NotEmpty(t, result)

	var openaiChunk models.ChatCompletionChunk
	err = json.Unmarshal([]byte(result), &openaiChunk)
	require.NoError(t, err)

	assert.Equal(t, "chatcmpl-stream-1", openaiChunk.ID)
	assert.Equal(t, "chat.completion.chunk", openaiChunk.Object)
	assert.Equal(t, "gpt-4", openaiChunk.Model)
	require.Len(t, openaiChunk.Choices, 1)
	assert.Equal(t, 0, openaiChunk.Choices[0].Index)
	assert.Equal(t, "Hello streaming", openaiChunk.Choices[0].Delta.Content)
	assert.Nil(t, openaiChunk.Choices[0].FinishReason)
}

func TestConvertGeminiSSE_Done(t *testing.T) {
	chunk := GeminiStreamChunk{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Role:  "model",
					Parts: []GeminiPart{},
				},
				FinishReason: "STOP",
				Index:        0,
			},
		},
	}

	data, err := json.Marshal(chunk)
	require.NoError(t, err)

	result, isFinal, err := ConvertGeminiSSE(data, "gpt-4", "chatcmpl-stream-2")
	require.NoError(t, err)
	assert.True(t, isFinal)
	assert.NotEmpty(t, result)

	var openaiChunk models.ChatCompletionChunk
	err = json.Unmarshal([]byte(result), &openaiChunk)
	require.NoError(t, err)

	require.Len(t, openaiChunk.Choices, 1)
	require.NotNil(t, openaiChunk.Choices[0].FinishReason)
	assert.Equal(t, "stop", *openaiChunk.Choices[0].FinishReason)
}

func TestConvertGeminiSSE_ContentWithFinishReason(t *testing.T) {
	chunk := GeminiStreamChunk{
		Candidates: []GeminiCandidate{
			{
				Content: GeminiContent{
					Role: "model",
					Parts: []GeminiPart{
						{Text: "Final text"},
					},
				},
				FinishReason: "STOP",
				Index:        0,
			},
		},
	}

	data, err := json.Marshal(chunk)
	require.NoError(t, err)

	result, isFinal, err := ConvertGeminiSSE(data, "gpt-4", "chatcmpl-stream-3")
	require.NoError(t, err)
	assert.True(t, isFinal)

	var openaiChunk models.ChatCompletionChunk
	err = json.Unmarshal([]byte(result), &openaiChunk)
	require.NoError(t, err)

	require.Len(t, openaiChunk.Choices, 1)
	assert.Equal(t, "Final text", openaiChunk.Choices[0].Delta.Content)
	require.NotNil(t, openaiChunk.Choices[0].FinishReason)
	assert.Equal(t, "stop", *openaiChunk.Choices[0].FinishReason)
}

func TestConvertGeminiSSE_NoCandidates(t *testing.T) {
	chunk := GeminiStreamChunk{
		Candidates: []GeminiCandidate{},
	}

	data, err := json.Marshal(chunk)
	require.NoError(t, err)

	result, isFinal, err := ConvertGeminiSSE(data, "gpt-4", "chatcmpl-stream-4")
	require.NoError(t, err)
	assert.False(t, isFinal)
	assert.Empty(t, result)
}

func TestConvertGeminiSSE_InvalidJSON(t *testing.T) {
	_, _, err := ConvertGeminiSSE([]byte("not valid json"), "gpt-4", "chatcmpl-stream-5")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse gemini chunk")
}

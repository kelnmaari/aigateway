package converter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aigateway/internal/models"
)

// --- Gemini Request Types ---

// GeminiRequest represents a Gemini generateContent request.
type GeminiRequest struct {
	Contents          []GeminiContent         `json:"contents"`
	SystemInstruction *GeminiContent          `json:"systemInstruction,omitempty"`
	GenerationConfig  *GeminiGenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings    []GeminiSafetySetting   `json:"safetySettings,omitempty"`
}

// GeminiContent represents content in Gemini format.
type GeminiContent struct {
	Role  string       `json:"role,omitempty"` // "user" or "model"
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart represents a part of content in Gemini format.
type GeminiPart struct {
	Text string `json:"text,omitempty"`
}

// GeminiGenerationConfig represents generation parameters.
type GeminiGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	TopK            *int     `json:"topK,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
	CandidateCount  *int     `json:"candidateCount,omitempty"`
}

// GeminiSafetySetting represents a safety setting.
type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// --- Gemini Response Types ---

// GeminiResponse represents a Gemini generateContent response.
type GeminiResponse struct {
	Candidates     []GeminiCandidate     `json:"candidates"`
	PromptFeedback *GeminiPromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  *GeminiUsageMetadata  `json:"usageMetadata,omitempty"`
}

// GeminiCandidate represents a candidate in the Gemini response.
type GeminiCandidate struct {
	Content       GeminiContent        `json:"content"`
	FinishReason  string               `json:"finishReason"` // "STOP", "MAX_TOKENS", "SAFETY", "RECITATION"
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings,omitempty"`
	Index         int                  `json:"index"`
}

// GeminiPromptFeedback represents feedback about the prompt.
type GeminiPromptFeedback struct {
	SafetyRatings []GeminiSafetyRating `json:"safetyRatings,omitempty"`
	BlockReason   string               `json:"blockReason,omitempty"`
}

// GeminiSafetyRating represents a safety rating.
type GeminiSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// GeminiUsageMetadata represents token usage in Gemini format.
type GeminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// --- Gemini Streaming Types ---

// GeminiStreamChunk is the same as GeminiResponse for streaming.
type GeminiStreamChunk = GeminiResponse

// --- Conversion Functions ---

// OpenAIToGemini converts an OpenAI ChatCompletionRequest to a GeminiRequest.
func OpenAIToGemini(req *models.ChatCompletionRequest) (*GeminiRequest, error) {
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages array is empty")
	}

	geminiReq := &GeminiRequest{}

	// Build generation config
	genConfig := &GeminiGenerationConfig{
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}
	if req.MaxTokens != nil {
		genConfig.MaxOutputTokens = req.MaxTokens
	}
	if len(req.Stop) > 0 {
		genConfig.StopSequences = req.Stop
	}
	geminiReq.GenerationConfig = genConfig

	// Separate system messages and convert other messages
	var contents []GeminiContent

	for _, msg := range req.Messages {
		contentStr := extractContentString(msg.Content)

		switch msg.Role {
		case "system":
			// Gemini uses systemInstruction
			if geminiReq.SystemInstruction == nil {
				geminiReq.SystemInstruction = &GeminiContent{
					Parts: []GeminiPart{{Text: contentStr}},
				}
			} else {
				// Append to existing system instruction
				geminiReq.SystemInstruction.Parts = append(
					geminiReq.SystemInstruction.Parts,
					GeminiPart{Text: contentStr},
				)
			}
		case "user":
			contents = append(contents, GeminiContent{
				Role:  "user",
				Parts: []GeminiPart{{Text: contentStr}},
			})
		case "assistant":
			contents = append(contents, GeminiContent{
				Role:  "model",
				Parts: []GeminiPart{{Text: contentStr}},
			})
		case "tool":
			// Tool results become user messages
			contents = append(contents, GeminiContent{
				Role:  "user",
				Parts: []GeminiPart{{Text: fmt.Sprintf("[Tool Result: %s]", contentStr)}},
			})
		}
	}

	if len(contents) == 0 {
		return nil, fmt.Errorf("no user or assistant messages after filtering system messages")
	}

	geminiReq.Contents = contents
	return geminiReq, nil
}

// GeminiToOpenAI converts a GeminiResponse to an OpenAI ChatCompletionResponse.
func GeminiToOpenAI(resp *GeminiResponse, requestModel string) *models.ChatCompletionResponse {
	var choices []models.ChatCompletionChoice

	for i, candidate := range resp.Candidates {
		var textParts []string
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				textParts = append(textParts, part.Text)
			}
		}
		content := strings.Join(textParts, "")

		choices = append(choices, models.ChatCompletionChoice{
			Index: i,
			Message: models.ChatMessage{
				Role:    "assistant",
				Content: content,
			},
			FinishReason: mapGeminiFinishReason(candidate.FinishReason),
		})
	}

	openaiResp := &models.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-gemini-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   requestModel,
		Choices: choices,
	}

	if resp.UsageMetadata != nil {
		openaiResp.Usage = models.Usage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	return openaiResp
}

// ConvertGeminiSSE converts a Gemini streaming chunk to OpenAI SSE format.
// Returns the SSE data string and whether this is the final event.
func ConvertGeminiSSE(data []byte, requestModel string, streamID string) (string, bool, error) {
	var chunk GeminiStreamChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return "", false, fmt.Errorf("parse gemini chunk: %w", err)
	}

	if len(chunk.Candidates) == 0 {
		return "", false, nil
	}

	candidate := chunk.Candidates[0]

	// Extract text from parts
	var text strings.Builder
	for _, part := range candidate.Content.Parts {
		text.WriteString(part.Text)
	}

	// Check if this is the final chunk
	isFinal := candidate.FinishReason != "" && candidate.FinishReason != "FINISH_REASON_UNSPECIFIED"

	if isFinal && text.String() == "" {
		// Final chunk with just finish reason
		finishReason := mapGeminiFinishReason(candidate.FinishReason)
		openaiChunk := models.ChatCompletionChunk{
			ID:      streamID,
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   requestModel,
			Choices: []models.ChatCompletionChunkChoice{
				{
					Index:        0,
					Delta:        models.ChatMessage{},
					FinishReason: &finishReason,
				},
			},
		}
		jsonData, _ := json.Marshal(openaiChunk)
		return string(jsonData), true, nil
	}

	// Content chunk
	openaiChunk := models.ChatCompletionChunk{
		ID:      streamID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   requestModel,
		Choices: []models.ChatCompletionChunkChoice{
			{
				Index: 0,
				Delta: models.ChatMessage{
					Content: text.String(),
				},
			},
		},
	}

	if isFinal {
		finishReason := mapGeminiFinishReason(candidate.FinishReason)
		openaiChunk.Choices[0].FinishReason = &finishReason
	}

	jsonData, _ := json.Marshal(openaiChunk)
	return string(jsonData), isFinal, nil
}

// mapGeminiFinishReason maps Gemini finishReason to OpenAI finish_reason.
func mapGeminiFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	case "RECITATION":
		return "content_filter"
	default:
		return "stop"
	}
}

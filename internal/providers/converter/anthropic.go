// Package converter provides request/response format conversion between OpenAI API
// and other provider APIs (Anthropic, Gemini).
package converter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aigateway/internal/models"
)

// --- Anthropic Request Types ---

// AnthropicRequest represents an Anthropic Messages API request.
type AnthropicRequest struct {
	Model         string             `json:"model"`
	Messages      []AnthropicMessage `json:"messages"`
	System        string             `json:"system,omitempty"`
	MaxTokens     int                `json:"max_tokens"`
	Temperature   *float64           `json:"temperature,omitempty"`
	TopP          *float64           `json:"top_p,omitempty"`
	TopK          *int               `json:"top_k,omitempty"`
	StopSequences []string           `json:"stop_sequences,omitempty"`
	Stream        bool               `json:"stream,omitempty"`
}

// AnthropicMessage represents a message in the Anthropic format.
type AnthropicMessage struct {
	Role    string      `json:"role"` // "user" or "assistant"
	Content interface{} `json:"content"`
}

// AnthropicContentBlock represents a content block in Anthropic format.
type AnthropicContentBlock struct {
	Type string `json:"type"` // "text", "image"
	Text string `json:"text,omitempty"`
}

// --- Anthropic Response Types ---

// AnthropicResponse represents an Anthropic Messages API response.
type AnthropicResponse struct {
	ID           string                  `json:"id"`
	Type         string                  `json:"type"` // "message"
	Role         string                  `json:"role"` // "assistant"
	Content      []AnthropicContentBlock `json:"content"`
	Model        string                  `json:"model"`
	StopReason   string                  `json:"stop_reason"` // "end_turn", "max_tokens", "stop_sequence"
	StopSequence *string                 `json:"stop_sequence,omitempty"`
	Usage        AnthropicUsage          `json:"usage"`
}

// AnthropicUsage represents token usage in the Anthropic format.
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// --- Anthropic SSE Types ---

// AnthropicSSEEvent represents an Anthropic SSE event.
type AnthropicSSEEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"-"` // raw JSON payload after parsing event type
}

// AnthropicMessageStart is the message_start SSE event.
type AnthropicMessageStart struct {
	Type    string            `json:"type"`
	Message AnthropicResponse `json:"message"`
}

// AnthropicContentBlockDelta is the content_block_delta SSE event.
type AnthropicContentBlockDelta struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}

// AnthropicMessageDelta is the message_delta SSE event.
type AnthropicMessageDelta struct {
	Type  string `json:"type"`
	Delta struct {
		StopReason   string  `json:"stop_reason"`
		StopSequence *string `json:"stop_sequence,omitempty"`
	} `json:"delta"`
	Usage AnthropicUsage `json:"usage"`
}

// --- Conversion Functions ---

// OpenAIToAnthropic converts an OpenAI ChatCompletionRequest to an AnthropicRequest.
func OpenAIToAnthropic(req *models.ChatCompletionRequest, targetModel string) (*AnthropicRequest, error) {
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages array is empty")
	}

	anthropicReq := &AnthropicRequest{
		Model:       targetModel,
		MaxTokens:   4096, // Anthropic requires max_tokens
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
	}

	if req.MaxTokens != nil {
		anthropicReq.MaxTokens = *req.MaxTokens
	}

	if len(req.Stop) > 0 {
		anthropicReq.StopSequences = req.Stop
	}

	// Separate system messages from user/assistant messages
	var systemParts []string
	var messages []AnthropicMessage

	for _, msg := range req.Messages {
		contentStr := extractContentString(msg.Content)

		switch msg.Role {
		case "system":
			systemParts = append(systemParts, contentStr)
		case "user":
			messages = append(messages, AnthropicMessage{
				Role:    "user",
				Content: contentStr,
			})
		case "assistant":
			messages = append(messages, AnthropicMessage{
				Role:    "assistant",
				Content: contentStr,
			})
		case "tool":
			// Tool results become user messages in Anthropic format
			messages = append(messages, AnthropicMessage{
				Role:    "user",
				Content: fmt.Sprintf("[Tool Result: %s]", contentStr),
			})
		}
	}

	if len(systemParts) > 0 {
		anthropicReq.System = strings.Join(systemParts, "\n\n")
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no user or assistant messages after filtering system messages")
	}

	anthropicReq.Messages = messages
	return anthropicReq, nil
}

// AnthropicToOpenAI converts an AnthropicResponse to an OpenAI ChatCompletionResponse.
func AnthropicToOpenAI(resp *AnthropicResponse, requestModel string) *models.ChatCompletionResponse {
	// Combine all text content blocks
	var textParts []string
	for _, block := range resp.Content {
		if block.Type == "text" {
			textParts = append(textParts, block.Text)
		}
	}
	content := strings.Join(textParts, "")

	return &models.ChatCompletionResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   requestModel,
		Choices: []models.ChatCompletionChoice{
			{
				Index: 0,
				Message: models.ChatMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: mapAnthropicStopReason(resp.StopReason),
			},
		},
		Usage: models.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

// ConvertAnthropicSSE converts an Anthropic SSE event to OpenAI SSE format.
// Returns the SSE data string (without "data: " prefix) and whether this is the final event.
func ConvertAnthropicSSE(eventType string, data []byte, requestModel string, streamID string) (string, bool, error) {
	switch eventType {
	case "message_start":
		var event AnthropicMessageStart
		if err := json.Unmarshal(data, &event); err != nil {
			return "", false, fmt.Errorf("parse message_start: %w", err)
		}
		chunk := models.ChatCompletionChunk{
			ID:      streamID,
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   requestModel,
			Choices: []models.ChatCompletionChunkChoice{
				{
					Index: 0,
					Delta: models.ChatMessage{
						Role: "assistant",
					},
				},
			},
		}
		jsonData, _ := json.Marshal(chunk)
		return string(jsonData), false, nil

	case "content_block_delta":
		var event AnthropicContentBlockDelta
		if err := json.Unmarshal(data, &event); err != nil {
			return "", false, fmt.Errorf("parse content_block_delta: %w", err)
		}
		chunk := models.ChatCompletionChunk{
			ID:      streamID,
			Object:  "chat.completion.chunk",
			Created: time.Now().Unix(),
			Model:   requestModel,
			Choices: []models.ChatCompletionChunkChoice{
				{
					Index: 0,
					Delta: models.ChatMessage{
						Content: event.Delta.Text,
					},
				},
			},
		}
		jsonData, _ := json.Marshal(chunk)
		return string(jsonData), false, nil

	case "message_delta":
		var event AnthropicMessageDelta
		if err := json.Unmarshal(data, &event); err != nil {
			return "", false, fmt.Errorf("parse message_delta: %w", err)
		}
		finishReason := mapAnthropicStopReason(event.Delta.StopReason)
		chunk := models.ChatCompletionChunk{
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
		jsonData, _ := json.Marshal(chunk)
		return string(jsonData), false, nil

	case "message_stop":
		return "[DONE]", true, nil

	case "ping", "content_block_start", "content_block_stop":
		// Skip these events
		return "", false, nil

	default:
		return "", false, nil
	}
}

// mapAnthropicStopReason maps Anthropic stop_reason to OpenAI finish_reason.
func mapAnthropicStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	default:
		return "stop"
	}
}

// extractContentString extracts a string from ChatMessage.Content which can be
// either a string or an array of content parts.
func extractContentString(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		// Try to marshal and extract text parts
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		var parts []map[string]interface{}
		if err := json.Unmarshal(data, &parts); err != nil {
			return string(data)
		}
		var texts []string
		for _, part := range parts {
			if part["type"] == "text" {
				if text, ok := part["text"].(string); ok {
					texts = append(texts, text)
				}
			}
		}
		return strings.Join(texts, "")
	}
}

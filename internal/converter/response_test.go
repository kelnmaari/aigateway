package converter

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aigateway/internal/client/ollama"
	"aigateway/internal/models"
)

// TestUsageCompletionTokensAlwaysPresent проверяет что completion_tokens всегда есть в JSON
// КРИТИЧНО для совместимости с AI Review и другими OpenAI-совместимыми клиентами
func TestUsageCompletionTokensAlwaysPresent(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockModelManager := &mockModelManager{}
	converter := NewResponseConverter(logger, mockModelManager)

	tests := []struct {
		name                   string
		ollamaResp             *ollama.ChatResponse
		expectCompletionTokens int
	}{
		{
			name: "Normal response with content",
			ollamaResp: &ollama.ChatResponse{
				Model: "llama3.2",
				Message: ollama.ChatMessage{
					Role:    "assistant",
					Content: "Hello, world!",
				},
				Done:            true,
				PromptEvalCount: 10,
				EvalCount:       5,
				CreatedAt:       time.Now(),
			},
			expectCompletionTokens: 5,
		},
		{
			name: "Empty response with zero EvalCount",
			ollamaResp: &ollama.ChatResponse{
				Model: "llama3.2",
				Message: ollama.ChatMessage{
					Role:    "assistant",
					Content: "",
				},
				Done:            true,
				PromptEvalCount: 100,
				EvalCount:       0, // Модель ничего не сгенерировала
				CreatedAt:       time.Now(),
			},
			expectCompletionTokens: 1, // Минимум 1 токен
		},
		{
			name: "Large prompt without response",
			ollamaResp: &ollama.ChatResponse{
				Model: "devstral-tuned:latest",
				Message: ollama.ChatMessage{
					Role:    "assistant",
					Content: "",
				},
				Done:            true,
				PromptEvalCount: 33973, // Очень большой контекст
				EvalCount:       0,
				CreatedAt:       time.Now(),
			},
			expectCompletionTokens: 1, // Даже если ничего не сгенерировано
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &models.ChatCompletionRequest{
				Model: "test-model",
				Messages: []models.ChatMessage{
					{Role: "user", Content: "test"},
				},
			}

			// Конвертируем ответ
			response, err := converter.ConvertChatResponse(tt.ollamaResp, req, "test-123")
			require.NoError(t, err)
			require.NotNil(t, response)

			// Проверяем что usage корректен
			assert.Equal(t, tt.expectCompletionTokens, response.Usage.CompletionTokens)
			assert.Equal(t, tt.ollamaResp.PromptEvalCount, response.Usage.PromptTokens)
			assert.Equal(t, response.Usage.PromptTokens+response.Usage.CompletionTokens, response.Usage.TotalTokens)

			// КРИТИЧНО: Проверяем что completion_tokens присутствует в JSON
			jsonData, err := json.Marshal(response)
			require.NoError(t, err)

			var parsed map[string]interface{}
			err = json.Unmarshal(jsonData, &parsed)
			require.NoError(t, err)

			// Проверяем наличие usage объекта
			usage, ok := parsed["usage"].(map[string]interface{})
			require.True(t, ok, "usage object must be present in JSON")

			// КРИТИЧНО: completion_tokens ДОЛЖЕН присутствовать
			_, hasCompletionTokens := usage["completion_tokens"]
			assert.True(t, hasCompletionTokens, "completion_tokens MUST be present in JSON (required by OpenAI API)")

			// Проверяем все поля usage
			assert.Contains(t, usage, "prompt_tokens")
			assert.Contains(t, usage, "completion_tokens")
			assert.Contains(t, usage, "total_tokens")

			t.Logf("JSON usage: %+v", usage)
		})
	}
}

// mockModelManager для тестов
type mockModelManager struct{}

func (m *mockModelManager) MapOpenAIToOllama(model string) (string, error) {
	return model, nil
}

func (m *mockModelManager) MapOllamaToOpenAI(model string) (string, error) {
	return model, nil
}

func (m *mockModelManager) IsModelSupported(model string, apiType string) bool {
	return true
}

func (m *mockModelManager) GetModelCapabilities(model string) (*models.ModelMappingConfig, error) {
	return nil, nil
}

func (m *mockModelManager) ListSupportedModels(apiType string) []string {
	return []string{}
}

func (m *mockModelManager) GetModelMapping(model string) (*models.ModelMapping, error) {
	return nil, nil
}

func (m *mockModelManager) AddModelMapping(mapping *models.ModelMapping) error {
	return nil
}

func (m *mockModelManager) UpdateModelMapping(model string, mapping *models.ModelMapping) error {
	return nil
}

func (m *mockModelManager) RemoveModelMapping(model string) error {
	return nil
}


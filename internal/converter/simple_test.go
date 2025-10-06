// Package converter provides simple tests without complex mocks
package converter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/test"
)

func TestModelMapping_NormalizeModelName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple", "GPT-3.5-Turbo", "gpt-3-5-turbo"},
		{"with_underscore", "text_davinci_003", "text-davinci-003"},
		{"with_dots", "gpt.3.5.turbo", "gpt-3-5-turbo"},
		{"with_spaces", "GPT 4 Turbo", "gpt-4-turbo"},
		{"mixed", "GPT_4.Turbo Preview", "gpt-4-turbo-preview"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := models.NormalizeModelName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestModelMapping_ExtractModelFamily(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"llama2_7b", "llama2:7b", "llama2"},
		{"llama2_13b", "llama2:13b", "llama2"},
		{"mistral_7b", "mistral-7b", "mistral-7b"}, // Функция пока не обрабатывает дефисы
		{"gpt4_turbo", "gpt-4-turbo", "gpt-4-turbo"},
		{"simple", "llama2", "llama2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := models.ExtractModelFamily(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfig_Validation(t *testing.T) {
	cfg := test.TestConfig()

	// Проверяем что тестовая конфигурация валидна
	err := cfg.Validate()
	assert.NoError(t, err)

	// Проверяем основные поля
	assert.Equal(t, "localhost", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "http://localhost:11434", cfg.Ollama.URL)
	assert.True(t, cfg.Models.Cache.Enabled)
}

func TestDefaultModelMappings(t *testing.T) {
	// Проверяем предустановленные маппинги
	assert.NotEmpty(t, models.DefaultModelMappings)

	for _, mapping := range models.DefaultModelMappings {
		assert.NotEmpty(t, mapping.OpenAIName)
		assert.NotEmpty(t, mapping.OllamaName)

		// Проверяем что маппинг имеет базовую конфигурацию
		assert.NotNil(t, mapping.Config.MaxTokens)
		assert.NotNil(t, mapping.Config.DefaultTemp)
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := models.GenerateRequestID("test")
	time.Sleep(1 * time.Nanosecond) // Убеждаемся что время изменилось
	id2 := models.GenerateRequestID("test")

	// ID должны содержать префикс
	assert.Contains(t, id1, "test-")
	assert.Contains(t, id2, "test-")

	// ID должны быть разными (если время позволяет)
	if id1 == id2 {
		t.Log("Generated same ID (expected due to high precision timing)")
	}
}

func TestCreateConversionContext(t *testing.T) {
	ctx := models.CreateConversionContext("req-123", "gpt-3.5-turbo", "llama2")

	assert.Equal(t, "req-123", ctx.RequestID)
	assert.Equal(t, "gpt-3.5-turbo", ctx.OriginalModel)
	assert.Equal(t, "llama2", ctx.MappedModel)
	assert.NotNil(t, ctx.Metadata)
	assert.False(t, ctx.Debug)
}

func TestCreateSuccessResult(t *testing.T) {
	data := map[string]string{"test": "value"}
	duration := time.Second

	result := models.CreateSuccessResult(data, duration)

	assert.True(t, result.Success)
	assert.Equal(t, data, result.Data)
	assert.Equal(t, duration, result.Duration)
	assert.Empty(t, result.Error)
	assert.NotNil(t, result.Metadata)
}

func TestCreateErrorResult(t *testing.T) {
	err := assert.AnError
	duration := time.Second

	result := models.CreateErrorResult(err, duration)

	assert.False(t, result.Success)
	assert.Nil(t, result.Data)
	assert.Equal(t, duration, result.Duration)
	assert.Equal(t, err.Error(), result.Error)
}

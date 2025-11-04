// Package models provides mapping and conversion models
package models

import (
	"fmt"
	"strings"
	"time"

	"aigateway/internal/utils"
)

// ModelMapping представляет маппинг между именами моделей OpenAI и Ollama
type ModelMapping struct {
	OpenAIName string                 `json:"openai_name"`
	OllamaName string                 `json:"ollama_name"`
	Aliases    []string               `json:"aliases,omitempty"`
	Config     ModelMappingConfig     `json:"config,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ModelMappingConfig содержит конфигурацию для маппинга модели
type ModelMappingConfig struct {
	MaxTokens         *int     `json:"max_tokens,omitempty"`
	DefaultTemp       *float64 `json:"default_temperature,omitempty"`
	SupportsFunctions bool     `json:"supports_functions,omitempty"`
	SupportsVision    bool     `json:"supports_vision,omitempty"`
	SupportsTools     bool     `json:"supports_tools,omitempty"`
	CostPerToken      *float64 `json:"cost_per_token,omitempty"`
}

// ParameterMapping представляет маппинг параметров между API
type ParameterMapping struct {
	OpenAIParam  string      `json:"openai_param"`
	OllamaParam  string      `json:"ollama_param"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Transform    string      `json:"transform,omitempty"` // "direct", "scale", "invert", "custom"
	MinValue     *float64    `json:"min_value,omitempty"`
	MaxValue     *float64    `json:"max_value,omitempty"`
}

// ConversionContext содержит контекст для конвертации запросов/ответов
type ConversionContext struct {
	RequestID      string                 `json:"request_id"`
	OriginalModel  string                 `json:"original_model"`
	MappedModel    string                 `json:"mapped_model"`
	ConversionTime time.Time              `json:"conversion_time"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Debug          bool                   `json:"debug,omitempty"`
}

// ConversionResult представляет результат конвертации
type ConversionResult struct {
	Success  bool                   `json:"success"`
	Data     interface{}            `json:"data,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Warnings []string               `json:"warnings,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Duration time.Duration          `json:"duration"`
}

// ModelManager интерфейс для управления моделями и маппингом
type ModelManager interface {
	// Методы маппинга моделей
	MapOpenAIToOllama(openaiModel string) (string, error)
	MapOllamaToOpenAI(ollamaModel string) (string, error)

	// Методы валидации
	IsModelSupported(model string, apiType string) bool
	GetModelCapabilities(model string) (*ModelMappingConfig, error)

	// Методы получения информации
	ListSupportedModels(apiType string) []string
	GetModelMapping(model string) (*ModelMapping, error)

	// Методы обновления маппинга
	AddModelMapping(mapping *ModelMapping) error
	UpdateModelMapping(model string, mapping *ModelMapping) error
	RemoveModelMapping(model string) error
}

// ParameterConverter интерфейс для конвертации параметров
type ParameterConverter interface {
	// Конвертация параметров OpenAI в Ollama
	ConvertOpenAIParams(params map[string]interface{}) (map[string]interface{}, error)

	// Конвертация параметров Ollama в OpenAI
	ConvertOllamaParams(params map[string]interface{}) (map[string]interface{}, error)

	// Валидация параметров
	ValidateOpenAIParams(params map[string]interface{}) error
	ValidateOllamaParams(params map[string]interface{}) error

	// Получение информации о параметрах
	GetSupportedOpenAIParams() []string
	GetSupportedOllamaParams() []string
	GetParameterMapping(param string) (*ParameterMapping, error)
}

// RequestConverter интерфейс для конвертации запросов
type RequestConverter interface {
	// Конвертация chat completion запросов
	ConvertChatCompletionRequest(req *ChatCompletionRequest, ctx *ConversionContext) (*ConversionResult, error)

	// Конвертация text completion запросов
	ConvertCompletionRequest(req *CompletionRequest, ctx *ConversionContext) (*ConversionResult, error)

	// Конвертация embeddings запросов
	ConvertEmbeddingRequest(req *EmbeddingRequest, ctx *ConversionContext) (*ConversionResult, error)
}

// ResponseConverter интерфейс для конвертации ответов
type ResponseConverter interface {
	// Конвертация chat completion ответов
	ConvertChatCompletionResponse(ollamaResp interface{}, ctx *ConversionContext) (*ConversionResult, error)

	// Конвертация text completion ответов
	ConvertCompletionResponse(ollamaResp interface{}, ctx *ConversionContext) (*ConversionResult, error)

	// Конвертация embeddings ответов
	ConvertEmbeddingResponse(ollamaResp interface{}, ctx *ConversionContext) (*ConversionResult, error)

	// Конвертация streaming ответов
	ConvertStreamingResponse(ollamaChunk interface{}, ctx *ConversionContext) (*ConversionResult, error)
}

// StreamingConverter интерфейс для обработки потоковых ответов
type StreamingConverter interface {
	// Конвертация streaming chunk'ов
	ConvertStreamChunk(ollamaChunk interface{}, ctx *ConversionContext) (*ChatCompletionChunk, error)

	// Обработка завершения потока
	HandleStreamEnd(ctx *ConversionContext) (*ChatCompletionChunk, error)

	// Обработка ошибок в потоке
	HandleStreamError(err error, ctx *ConversionContext) (*ChatCompletionChunk, error)
}

// Вспомогательные функции для маппинга

// NormalizeModelName нормализует имя модели для поиска
func NormalizeModelName(modelName string) string {
	// Приводим к нижнему регистру и убираем пробелы
	normalized := strings.ToLower(strings.TrimSpace(modelName))

	// Заменяем различные разделители на дефис
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, ".", "-")
	normalized = strings.ReplaceAll(normalized, " ", "-")

	return normalized
}

// ExtractModelFamily извлекает семейство модели из полного имени
func ExtractModelFamily(modelName string) string {
	// Убираем размер модели и другие суффиксы
	parts := strings.Split(modelName, ":")
	if len(parts) > 0 {
		return parts[0]
	}

	// Убираем размеры типа "7b", "13b", "70b"
	name := strings.ToLower(modelName)
	sizeSuffixes := []string{"7b", "13b", "30b", "70b", "8b", "22b", "405b"}

	for _, suffix := range sizeSuffixes {
		if strings.HasSuffix(name, suffix) {
			return strings.TrimSuffix(name, suffix)
		}
		if strings.HasSuffix(name, "-"+suffix) {
			return strings.TrimSuffix(name, "-"+suffix)
		}
	}

	return modelName
}

// GenerateRequestID генерирует уникальный ID для запроса
func GenerateRequestID(prefix string) string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s-%d", prefix, timestamp)
}

// CreateConversionContext создает новый контекст конвертации
func CreateConversionContext(requestID, originalModel, mappedModel string) *ConversionContext {
	return &ConversionContext{
		RequestID:      requestID,
		OriginalModel:  originalModel,
		MappedModel:    mappedModel,
		ConversionTime: time.Now(),
		Metadata:       make(map[string]interface{}),
		Debug:          false,
	}
}

// CreateSuccessResult создает успешный результат конвертации
func CreateSuccessResult(data interface{}, duration time.Duration) *ConversionResult {
	return &ConversionResult{
		Success:  true,
		Data:     data,
		Duration: duration,
		Metadata: make(map[string]interface{}),
	}
}

// CreateErrorResult создает результат с ошибкой
func CreateErrorResult(err error, duration time.Duration) *ConversionResult {
	return &ConversionResult{
		Success:  false,
		Error:    err.Error(),
		Duration: duration,
		Metadata: make(map[string]interface{}),
	}
}

// AddWarningToResult добавляет предупреждение к результату
func AddWarningToResult(result *ConversionResult, warning string) {
	if result.Warnings == nil {
		result.Warnings = make([]string, 0)
	}
	result.Warnings = append(result.Warnings, warning)
}

// Константы для типов API
const (
	APITypeOpenAI = "openai"
	APITypeOllama = "ollama"
)

// Константы для типов конвертации
const (
	ConversionTypeChatCompletion = "chat_completion"
	ConversionTypeCompletion     = "completion"
	ConversionTypeEmbedding      = "embedding"
	ConversionTypeModels         = "models"
)

// Константы для статусов конвертации
const (
	ConversionStatusSuccess = "success"
	ConversionStatusError   = "error"
	ConversionStatusWarning = "warning"
)

// Предопределенные маппинги популярных моделей
var DefaultModelMappings = []ModelMapping{
	{
		OpenAIName: "gpt-3.5-turbo",
		OllamaName: "qwen2.5-coder:7b",
		Aliases:    []string{"gpt-3.5-turbo-0613", "gpt-3.5-turbo-16k"},
		Config: ModelMappingConfig{
			MaxTokens:         utils.Ptr(4096),
			DefaultTemp:       utils.Ptr(0.7),
			SupportsFunctions: false,
			SupportsVision:    false,
			SupportsTools:     false,
		},
	},
	{
		OpenAIName: "gpt-4",
		OllamaName: "qwen3-coder:30b",
		Aliases:    []string{"gpt-4-0613", "gpt-4-32k"},
		Config: ModelMappingConfig{
			MaxTokens:         utils.Ptr(8192),
			DefaultTemp:       utils.Ptr(0.7),
			SupportsFunctions: false,
			SupportsVision:    false,
			SupportsTools:     false,
		},
	},
	{
		OpenAIName: "gpt-4-turbo",
		OllamaName: "gpt-oss-c32k:latest",
		Aliases:    []string{"gpt-4-turbo-preview", "gpt-4-0125-preview"},
		Config: ModelMappingConfig{
			MaxTokens:         utils.Ptr(128000),
			DefaultTemp:       utils.Ptr(0.7),
			SupportsFunctions: true,
			SupportsVision:    false,
			SupportsTools:     true,
		},
	},
}

// Вспомогательные функции для указателей replaced with utils.Ptr[T] (Go 1.25 generics)


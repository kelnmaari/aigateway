// Package converter provides interfaces for conversion components
package converter

import "ollama-openai-proxy/internal/models"

// ModelManager интерфейс для управления моделями
type ModelManager interface {
	// Методы маппинга моделей
	MapOpenAIToOllama(openaiModel string) (string, error)
	MapOllamaToOpenAI(ollamaModel string) (string, error)

	// Методы валидации
	IsModelSupported(model string, apiType string) bool
	GetModelCapabilities(model string) (*models.ModelMappingConfig, error)

	// Методы получения информации
	ListSupportedModels(apiType string) []string
	GetModelMapping(model string) (*models.ModelMapping, error)

	// Методы обновления маппинга
	AddModelMapping(mapping *models.ModelMapping) error
	UpdateModelMapping(model string, mapping *models.ModelMapping) error
	RemoveModelMapping(model string) error
}

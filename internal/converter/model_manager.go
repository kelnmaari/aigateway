// Package converter provides model mapping and management
package converter

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
	"aigateway/internal/models"
)

// DefaultModelManager реализация ModelManager с поддержкой конфигурации
type DefaultModelManager struct {
	logger   *logrus.Logger
	config   *config.Config
	mappings map[string]*models.ModelMapping // key: normalized openai name
	reverse  map[string]string               // key: ollama name, value: openai name
	mutex    sync.RWMutex
}

// NewDefaultModelManager создает новый model manager
func NewDefaultModelManager(cfg *config.Config, logger *logrus.Logger) *DefaultModelManager {
	mm := &DefaultModelManager{
		logger:   logger,
		config:   cfg,
		mappings: make(map[string]*models.ModelMapping),
		reverse:  make(map[string]string),
	}

	// Загружаем маппинги по умолчанию
	mm.loadDefaultMappings()

	// Загружаем маппинги из конфигурации
	mm.loadConfigMappings()

	return mm
}

// MapOpenAIToOllama маппинг OpenAI модели в Ollama модель
func (mm *DefaultModelManager) MapOpenAIToOllama(openaiModel string) (string, error) {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	normalized := models.NormalizeModelName(openaiModel)

	// Прямой поиск по маппингу
	if mapping, exists := mm.mappings[normalized]; exists {
		mm.logger.WithFields(logrus.Fields{
			"openai_model": openaiModel,
			"ollama_model": mapping.OllamaName,
		}).Debug("Found exact model mapping")
		return mapping.OllamaName, nil
	}

	// Поиск по алиасам
	for _, mapping := range mm.mappings {
		for _, alias := range mapping.Aliases {
			if models.NormalizeModelName(alias) == normalized {
				mm.logger.WithFields(logrus.Fields{
					"openai_model": openaiModel,
					"alias":        alias,
					"ollama_model": mapping.OllamaName,
				}).Debug("Found model mapping via alias")
				return mapping.OllamaName, nil
			}
		}
	}

	// Поиск по семейству модели
	family := models.ExtractModelFamily(openaiModel)
	normalizedFamily := models.NormalizeModelName(family)

	for _, mapping := range mm.mappings {
		mappingFamily := models.ExtractModelFamily(mapping.OpenAIName)
		if models.NormalizeModelName(mappingFamily) == normalizedFamily {
			mm.logger.WithFields(logrus.Fields{
				"openai_model": openaiModel,
				"model_family": family,
				"mapped_to":    mapping.OllamaName,
			}).Debug("Found model mapping via family match")
			return mapping.OllamaName, nil
		}
	}

	// Fallback: проверяем конфигурацию
	if ollamaModel, exists := mm.config.Models.Mapping[openaiModel]; exists {
		mm.logger.WithFields(logrus.Fields{
			"openai_model": openaiModel,
			"ollama_model": ollamaModel,
		}).Debug("Found model mapping in configuration")
		return ollamaModel, nil
	}

	// Если ничего не найдено, возвращаем ошибку
	return "", fmt.Errorf("no mapping found for OpenAI model: %s", openaiModel)
}

// MapOllamaToOpenAI маппинг Ollama модели в OpenAI модель
func (mm *DefaultModelManager) MapOllamaToOpenAI(ollamaModel string) (string, error) {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	// Прямой поиск в reverse маппинге
	if openaiModel, exists := mm.reverse[ollamaModel]; exists {
		mm.logger.WithFields(logrus.Fields{
			"ollama_model": ollamaModel,
			"openai_model": openaiModel,
		}).Debug("Found reverse model mapping")
		return openaiModel, nil
	}

	// Поиск в основных маппингах
	for _, mapping := range mm.mappings {
		if mapping.OllamaName == ollamaModel {
			mm.logger.WithFields(logrus.Fields{
				"ollama_model": ollamaModel,
				"openai_model": mapping.OpenAIName,
			}).Debug("Found reverse mapping in main mappings")
			return mapping.OpenAIName, nil
		}
	}

	// Fallback: проверяем конфигурацию (обратный поиск)
	for openaiName, ollamaName := range mm.config.Models.Mapping {
		if ollamaName == ollamaModel {
			mm.logger.WithFields(logrus.Fields{
				"ollama_model": ollamaModel,
				"openai_model": openaiName,
			}).Debug("Found reverse mapping in configuration")
			return openaiName, nil
		}
	}

	// Если ничего не найдено, возвращаем исходное имя
	return ollamaModel, nil
}

// IsModelSupported проверяет поддерживается ли модель
func (mm *DefaultModelManager) IsModelSupported(model string, apiType string) bool {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	switch apiType {
	case models.APITypeOpenAI:
		_, err := mm.MapOpenAIToOllama(model)
		return err == nil
	case models.APITypeOllama:
		// Для Ollama считаем что все модели поддерживаются
		return true
	default:
		return false
	}
}

// GetModelCapabilities возвращает возможности модели
func (mm *DefaultModelManager) GetModelCapabilities(model string) (*models.ModelMappingConfig, error) {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	normalized := models.NormalizeModelName(model)

	if mapping, exists := mm.mappings[normalized]; exists {
		return &mapping.Config, nil
	}

	// Поиск по алиасам
	for _, mapping := range mm.mappings {
		for _, alias := range mapping.Aliases {
			if models.NormalizeModelName(alias) == normalized {
				return &mapping.Config, nil
			}
		}
	}

	return nil, fmt.Errorf("no capabilities found for model: %s", model)
}

// ListSupportedModels возвращает список поддерживаемых моделей
func (mm *DefaultModelManager) ListSupportedModels(apiType string) []string {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	var modelsList []string

	switch apiType {
	case models.APITypeOpenAI:
		for _, mapping := range mm.mappings {
			modelsList = append(modelsList, mapping.OpenAIName)
			modelsList = append(modelsList, mapping.Aliases...)
		}

		// Добавляем модели из конфигурации
		for openaiName := range mm.config.Models.Mapping {
			modelsList = append(modelsList, openaiName)
		}

	case models.APITypeOllama:
		for _, mapping := range mm.mappings {
			modelsList = append(modelsList, mapping.OllamaName)
		}

		// Добавляем модели из конфигурации
		for _, ollamaName := range mm.config.Models.Mapping {
			modelsList = append(modelsList, ollamaName)
		}
	}

	// Удаляем дубликаты
	return mm.removeDuplicates(modelsList)
}

// GetModelMapping возвращает маппинг для модели
func (mm *DefaultModelManager) GetModelMapping(model string) (*models.ModelMapping, error) {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	normalized := models.NormalizeModelName(model)

	if mapping, exists := mm.mappings[normalized]; exists {
		return mapping, nil
	}

	return nil, fmt.Errorf("no mapping found for model: %s", model)
}

// AddModelMapping добавляет новый маппинг модели
func (mm *DefaultModelManager) AddModelMapping(mapping *models.ModelMapping) error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	normalized := models.NormalizeModelName(mapping.OpenAIName)

	// Проверяем что маппинг не существует
	if _, exists := mm.mappings[normalized]; exists {
		return fmt.Errorf("mapping for model %s already exists", mapping.OpenAIName)
	}

	mm.mappings[normalized] = mapping
	mm.reverse[mapping.OllamaName] = mapping.OpenAIName

	mm.logger.WithFields(logrus.Fields{
		"openai_model": mapping.OpenAIName,
		"ollama_model": mapping.OllamaName,
	}).Info("Added new model mapping")

	return nil
}

// UpdateModelMapping обновляет существующий маппинг
func (mm *DefaultModelManager) UpdateModelMapping(model string, mapping *models.ModelMapping) error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	normalized := models.NormalizeModelName(model)

	// Проверяем что маппинг существует
	if _, exists := mm.mappings[normalized]; !exists {
		return fmt.Errorf("mapping for model %s does not exist", model)
	}

	// Удаляем старый reverse маппинг
	if oldMapping, exists := mm.mappings[normalized]; exists {
		delete(mm.reverse, oldMapping.OllamaName)
	}

	// Добавляем новый маппинг
	mm.mappings[normalized] = mapping
	mm.reverse[mapping.OllamaName] = mapping.OpenAIName

	mm.logger.WithFields(logrus.Fields{
		"model":        model,
		"openai_model": mapping.OpenAIName,
		"ollama_model": mapping.OllamaName,
	}).Info("Updated model mapping")

	return nil
}

// RemoveModelMapping удаляет маппинг модели
func (mm *DefaultModelManager) RemoveModelMapping(model string) error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	normalized := models.NormalizeModelName(model)

	// Проверяем что маппинг существует
	mapping, exists := mm.mappings[normalized]
	if !exists {
		return fmt.Errorf("mapping for model %s does not exist", model)
	}

	// Удаляем маппинги
	delete(mm.mappings, normalized)
	delete(mm.reverse, mapping.OllamaName)

	mm.logger.WithField("model", model).Info("Removed model mapping")

	return nil
}

// loadDefaultMappings загружает маппинги по умолчанию
func (mm *DefaultModelManager) loadDefaultMappings() {
	for _, mapping := range models.DefaultModelMappings {
		normalized := models.NormalizeModelName(mapping.OpenAIName)
		mm.mappings[normalized] = &mapping
		mm.reverse[mapping.OllamaName] = mapping.OpenAIName
	}

	mm.logger.WithField("count", len(models.DefaultModelMappings)).Debug("Loaded default model mappings")
}

// loadConfigMappings загружает маппинги из конфигурации
func (mm *DefaultModelManager) loadConfigMappings() {
	if mm.config.Models.Mapping == nil {
		return
	}

	count := 0
	for openaiName, ollamaName := range mm.config.Models.Mapping {
		normalized := models.NormalizeModelName(openaiName)

		// Если маппинг уже существует, пропускаем
		if _, exists := mm.mappings[normalized]; exists {
			continue
		}

		mapping := &models.ModelMapping{
			OpenAIName: openaiName,
			OllamaName: ollamaName,
			Config: models.ModelMappingConfig{
				SupportsFunctions: false,
				SupportsVision:    false,
				SupportsTools:     false,
			},
		}

		mm.mappings[normalized] = mapping
		mm.reverse[ollamaName] = openaiName
		count++
	}

	if count > 0 {
		mm.logger.WithField("count", count).Debug("Loaded model mappings from configuration")
	}
}

// removeDuplicates удаляет дубликаты из слайса
func (mm *DefaultModelManager) removeDuplicates(items []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, item := range items {
		if item != "" && !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// GetModelInfo возвращает детальную информацию о модели
func (mm *DefaultModelManager) GetModelInfo(model string) map[string]interface{} {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	info := make(map[string]interface{})

	// Основная информация
	info["model"] = model
	info["normalized"] = models.NormalizeModelName(model)
	info["family"] = models.ExtractModelFamily(model)

	// Проверяем маппинг
	if mapping, err := mm.GetModelMapping(model); err == nil {
		info["has_mapping"] = true
		info["mapped_to"] = mapping.OllamaName
		info["capabilities"] = mapping.Config
		info["aliases"] = mapping.Aliases
	} else {
		info["has_mapping"] = false
	}

	// Проверяем поддержку
	info["supported_openai"] = mm.IsModelSupported(model, models.APITypeOpenAI)
	info["supported_ollama"] = mm.IsModelSupported(model, models.APITypeOllama)

	return info
}

// RefreshMappings обновляет маппинги из конфигурации
func (mm *DefaultModelManager) RefreshMappings() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	// Очищаем текущие маппинги
	mm.mappings = make(map[string]*models.ModelMapping)
	mm.reverse = make(map[string]string)

	// Перезагружаем маппинги
	mm.loadDefaultMappings()
	mm.loadConfigMappings()

	mm.logger.Info("Refreshed model mappings")
}


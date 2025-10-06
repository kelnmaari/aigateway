// Package handlers provides model warming functionality
package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/client/ollama"
	"ollama-openai-proxy/internal/config"
)

// ModelWarmer обеспечивает предварительный прогрев моделей
type ModelWarmer struct {
	config       *config.Config
	logger       *logrus.Logger
	ollamaClient OllamaClientInterface
}

// NewModelWarmer создает новый model warmer
func NewModelWarmer(cfg *config.Config, logger *logrus.Logger, ollamaClient OllamaClientInterface) *ModelWarmer {
	return &ModelWarmer{
		config:       cfg,
		logger:       logger,
		ollamaClient: ollamaClient,
	}
}

// WarmupModel прогревает модель перед использованием
func (w *ModelWarmer) WarmupModel(ctx context.Context, modelName string) error {
	w.logger.WithField("model", modelName).Info("Starting model warmup")

	// Создаем простой warmup запрос
	warmupReq := &ollama.ChatRequest{
		Model: modelName,
		Messages: []ollama.ChatMessage{
			{
				Role:    "user",
				Content: "Hi", // Минимальный запрос для прогрева
			},
		},
		Stream: false, // Не stream для warmup
		Options: &ollama.ChatOptions{
			NumPredict: intPtr(1), // Генерируем только 1 токен
		},
	}

	// Увеличенный timeout для прогрева больших моделей
	warmupTimeout := 180 * time.Second // 3 минуты для warmup
	if isLargeModel(modelName) {
		warmupTimeout = 600 * time.Second // 10 минут для очень больших моделей
	}

	warmupCtx, cancel := context.WithTimeout(ctx, warmupTimeout)
	defer cancel()

	start := time.Now()

	// Отправляем warmup запрос
	_, err := w.ollamaClient.ChatCompletion(warmupCtx, warmupReq)
	duration := time.Since(start)

	if err != nil {
		w.logger.WithError(err).WithFields(logrus.Fields{
			"model":    modelName,
			"duration": duration,
		}).Warn("Model warmup failed")
		return fmt.Errorf("model warmup failed: %w", err)
	}

	w.logger.WithFields(logrus.Fields{
		"model":    modelName,
		"duration": duration,
	}).Info("Model warmup completed successfully")

	return nil
}

// EnsureModelReady убеждается что модель готова к использованию
func (w *ModelWarmer) EnsureModelReady(ctx context.Context, modelName string) error {
	// Проверяем доступность модели
	available, err := w.ollamaClient.IsModelAvailable(ctx, modelName)
	if err != nil {
		return fmt.Errorf("failed to check model availability: %w", err)
	}

	if !available {
		w.logger.WithField("model", modelName).Warn("Model not available, attempting to pull")

		// Пытаемся загрузить модель
		pullCtx, cancel := context.WithTimeout(ctx, 600*time.Second) // 10 минут на загрузку
		defer cancel()

		_, err := w.ollamaClient.PullModel(pullCtx, modelName)
		if err != nil {
			return fmt.Errorf("failed to pull model %s: %w", modelName, err)
		}

		w.logger.WithField("model", modelName).Info("Model pulled successfully")
	}

	// Прогреваем модель если это большая модель
	if isLargeModel(modelName) {
		w.logger.WithField("model", modelName).Info("Large model detected, starting warmup")

		if err := w.WarmupModel(ctx, modelName); err != nil {
			w.logger.WithError(err).Warn("Model warmup failed, proceeding anyway")
			// Не возвращаем ошибку - пытаемся продолжить даже без warmup
		}
	}

	return nil
}

// isLargeModel проверяет является ли модель большой (дублирует функцию из streaming.go)
func isLargeModel(modelName string) bool {
	largeModelPatterns := []string{
		"30b", "70b", "405b", "gpt-oss-c32k", "qwen3-coder:30b", "qwen3-coder-30b",
	}

	modelLower := strings.ToLower(modelName)
	for _, pattern := range largeModelPatterns {
		if strings.Contains(modelLower, pattern) {
			return true
		}
	}

	return false
}

// GetModelSize возвращает примерный размер модели в параметрах
func GetModelSize(modelName string) string {
	modelLower := strings.ToLower(modelName)

	if strings.Contains(modelLower, "30b") {
		return "30B"
	}
	if strings.Contains(modelLower, "70b") {
		return "70B"
	}
	if strings.Contains(modelLower, "7b") {
		return "7B"
	}
	if strings.Contains(modelLower, "13b") {
		return "13B"
	}

	return "Unknown"
}

// intPtr helper function
func intPtr(i int) *int {
	return &i
}

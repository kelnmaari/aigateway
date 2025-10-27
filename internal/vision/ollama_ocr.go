package vision

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/sirupsen/logrus"

	"aigateway/internal/client/ollama"
	"aigateway/internal/config"
)

// OllamaOCR реализация OCREngine через Ollama vision models
type OllamaOCR struct {
	client *ollama.Client
	config *config.Config
	logger *logrus.Logger
}

// NewOllamaOCR создает новый Ollama OCR engine
func NewOllamaOCR(cfg *config.Config, logger *logrus.Logger) (*OllamaOCR, error) {
	client, err := ollama.NewClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	return &OllamaOCR{
		client: client,
		config: cfg,
		logger: logger,
	}, nil
}

// ExtractText извлекает текст из изображения через vision model
func (o *OllamaOCR) ExtractText(ctx context.Context, image io.Reader, opts OCROptions) (*OCRResult, error) {
	// Читаем изображение в память (raw bytes для Ollama API)
	imageData, err := io.ReadAll(image)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %w", err)
	}

	// Формируем промпт для OCR
	prompt := o.buildOCRPrompt(opts)

	// Создаем запрос к Ollama
	temp := opts.Temperature
	maxTokens := opts.MaxTokens

	chatReq := &ollama.ChatRequest{
		Model: opts.Model,
		Messages: []ollama.ChatMessage{
			{
				Role:    "user",
				Content: prompt,
				Images:  [][]byte{imageData}, // Raw bytes, не base64
			},
		},
		Options: &ollama.ChatOptions{
			Temperature: &temp,
			NumPredict:  &maxTokens,
		},
		Stream: false,
	}

	// Отправляем запрос
	o.logger.WithFields(logrus.Fields{
		"model":           opts.Model,
		"image_size_kb":   len(imageData) / 1024,
		"preserve_layout": opts.PreserveLayout,
		"extract_tables":  opts.ExtractTables,
	}).Debug("Sending OCR request to Ollama")

	response, err := o.client.ChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("OCR request failed: %w", err)
	}

	// Извлекаем текст из ответа
	extractedText := response.Message.Content

	// Определяем язык (простая эвристика)
	language := detectLanguage(extractedText)

	o.logger.WithFields(logrus.Fields{
		"model":             opts.Model,
		"extracted_length":  len(extractedText),
		"detected_language": language,
		"prompt_eval_count": response.PromptEvalCount,
		"eval_count":        response.EvalCount,
		"total_duration_ms": response.TotalDuration / 1000000,
	}).Info("OCR completed")

	result := &OCRResult{
		Text:       extractedText,
		Confidence: 0.85, // Vision models не возвращают confidence, используем константу
		Language:   language,
		Tables:     nil, // Table extraction будет реализован позже
		Metadata: map[string]interface{}{
			"model":             opts.Model,
			"prompt_eval_count": response.PromptEvalCount,
			"eval_count":        response.EvalCount,
			"total_duration_ms": response.TotalDuration / 1000000,
		},
	}

	return result, nil
}

// DescribeImage генерирует описание изображения
func (o *OllamaOCR) DescribeImage(ctx context.Context, image io.Reader, prompt string) (string, error) {
	// Читаем изображение (raw bytes)
	imageData, err := io.ReadAll(image)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}

	// Используем llava для описания
	temp := 0.7

	chatReq := &ollama.ChatRequest{
		Model: "llava:7b",
		Messages: []ollama.ChatMessage{
			{
				Role:    "user",
				Content: prompt,
				Images:  [][]byte{imageData}, // Raw bytes
			},
		},
		Options: &ollama.ChatOptions{
			Temperature: &temp,
		},
		Stream: false,
	}

	response, err := o.client.ChatCompletion(ctx, chatReq)
	if err != nil {
		return "", fmt.Errorf("image description failed: %w", err)
	}

	return response.Message.Content, nil
}

// SupportedModels возвращает список поддерживаемых vision моделей
func (o *OllamaOCR) SupportedModels() []string {
	return []string{
		"llava:7b",
		"llava:13b",
		"llava:34b",
		"bakllava",
		"llama3.2-vision:11b",
		"llama3.2-vision:90b",
	}
}

// buildOCRPrompt формирует промпт для OCR в зависимости от опций
func (o *OllamaOCR) buildOCRPrompt(opts OCROptions) string {
	var parts []string

	parts = append(parts, "Extract all text from this image.")

	if opts.PreserveLayout {
		parts = append(parts, "Preserve the original layout and formatting as much as possible.")
	}

	if opts.ExtractTables {
		parts = append(parts, "If the image contains tables, preserve their structure using markdown table format.")
	}

	if opts.Language != "auto" {
		parts = append(parts, fmt.Sprintf("The text is in %s language.", opts.Language))
	}

	parts = append(parts, "Return only the extracted text without any additional comments or explanations.")

	return strings.Join(parts, " ")
}

// detectLanguage простое определение языка по тексту
func detectLanguage(text string) string {
	// Подсчитываем кириллические и латинские символы
	cyrillicCount := 0
	latinCount := 0

	for _, r := range text {
		if r >= 'А' && r <= 'я' || r >= 'Ё' && r <= 'ё' {
			cyrillicCount++
		} else if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
			latinCount++
		}
	}

	if cyrillicCount > latinCount {
		return "ru"
	} else if latinCount > cyrillicCount {
		return "en"
	}

	return "unknown"
}


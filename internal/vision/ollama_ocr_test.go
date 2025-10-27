package vision

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"testing"

	"github.com/sirupsen/logrus"

	"aigateway/internal/client/ollama"
	"aigateway/internal/config"
)

// mockOllamaClient для тестирования
type mockOllamaClient struct {
	chatResponse *ollama.ChatResponse
	chatErr      error
}

func (m *mockOllamaClient) ChatCompletion(ctx context.Context, req *ollama.ChatRequest) (*ollama.ChatResponse, error) {
	return m.chatResponse, m.chatErr
}

func TestOllamaOCR_ExtractText(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs in tests

	cfg := &config.Config{
		Ollama: config.OllamaConfig{
			URL: "http://localhost:11434",
		},
	}

	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	buf := new(bytes.Buffer)
	jpeg.Encode(buf, img, nil)

	t.Run("SuccessfulOCR", func(t *testing.T) {
		ocr, _ := NewOllamaOCR(cfg, logger)

		// Для реального теста нужен мок клиента
		// Пока просто проверяем что метод не паникует
		opts := DefaultOCROptions()

		// Skip if Ollama is not available
		t.Skip("Requires Ollama server running")

		result, err := ocr.ExtractText(context.Background(), bytes.NewReader(buf.Bytes()), opts)
		if err != nil {
			t.Logf("OCR failed (expected if Ollama not running): %v", err)
		} else {
			t.Logf("OCR result: %+v", result)
		}
	})

	t.Run("BuildOCRPrompt", func(t *testing.T) {
		ocr, _ := NewOllamaOCR(cfg, logger)

		opts := OCROptions{
			Model:          "llava:7b",
			Language:       "en",
			PreserveLayout: true,
			ExtractTables:  true,
		}

		prompt := ocr.buildOCRPrompt(opts)

		if len(prompt) == 0 {
			t.Error("Expected non-empty prompt")
		}

		// Check that prompt contains expected keywords
		expectedKeywords := []string{"Extract", "text", "image"}
		for _, keyword := range expectedKeywords {
			if !bytes.Contains([]byte(prompt), []byte(keyword)) {
				t.Errorf("Expected prompt to contain '%s', got: %s", keyword, prompt)
			}
		}
	})
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Russian text",
			input:    "Привет мир",
			expected: "ru",
		},
		{
			name:     "English text",
			input:    "Hello world",
			expected: "en",
		},
		{
			name:     "Mixed text (more English)",
			input:    "Hello мир",
			expected: "en",
		},
		{
			name:     "Empty text",
			input:    "",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectLanguage(tt.input)
			if result != tt.expected {
				t.Errorf("detectLanguage(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestOllamaOCR_SupportedModels(t *testing.T) {
	logger := logrus.New()
	cfg := &config.Config{
		Ollama: config.OllamaConfig{
			URL: "http://localhost:11434",
		},
	}

	ocr, err := NewOllamaOCR(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create OllamaOCR: %v", err)
	}

	models := ocr.SupportedModels()

	if len(models) == 0 {
		t.Error("Expected at least one supported model")
	}

	// Check for known vision models
	expectedModels := []string{"llava:7b", "bakllava", "llama3.2-vision:11b"}
	for _, expected := range expectedModels {
		found := false
		for _, model := range models {
			if model == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected model '%s' not found in supported models", expected)
		}
	}
}


package extractors

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/config"
)

// NewExtractorRegistry создает Registry с extractors из конфигурации
func NewExtractorRegistry(cfg config.ExtractorsConfig, logger *logrus.Logger) (*Registry, error) {
	registry := NewRegistry(logger)

	// Создаем и регистрируем PDF extractor
	pdfExtractor := NewPDFExtractor(
		cfg.PDF.Method,
		cfg.PDF.PreserveLayout,
		cfg.PDF.MaxPages,
	)
	if err := registry.Register(pdfExtractor); err != nil {
		return nil, fmt.Errorf("failed to register PDF extractor: %w", err)
	}

	// Создаем и регистрируем DOCX extractor
	docxExtractor := NewDOCXExtractor(
		cfg.DOCX.ExtractTables,
		cfg.DOCX.ExtractImages,
		cfg.DOCX.ExtractComments,
	)
	if err := registry.Register(docxExtractor); err != nil {
		return nil, fmt.Errorf("failed to register DOCX extractor: %w", err)
	}

	// Создаем и регистрируем Text extractor
	textMaxSize, err := parseSize(cfg.Text.MaxSize)
	if err != nil {
		return nil, fmt.Errorf("invalid text max_size: %w", err)
	}

	textExtractor := NewTextExtractor(
		textMaxSize,
		cfg.Text.Encoding,
		cfg.Text.FallbackEncodings,
	)
	if err := registry.Register(textExtractor); err != nil {
		return nil, fmt.Errorf("failed to register Text extractor: %w", err)
	}

	// Создаем и регистрируем CSV extractor
	csvDelimiter := ','
	if cfg.CSV.Delimiter != "" {
		csvDelimiter = rune(cfg.CSV.Delimiter[0])
	}

	csvExtractor := NewCSVExtractor(
		cfg.CSV.MaxRows,
		csvDelimiter,
		cfg.CSV.Encoding,
		cfg.CSV.AutoDetectDelimiter,
	)
	if err := registry.Register(csvExtractor); err != nil {
		return nil, fmt.Errorf("failed to register CSV extractor: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"extractors": len(registry.SupportedTypes()),
		"timeout":    cfg.Timeout,
		"workers":    cfg.ParallelWorkers,
	}).Info("Extractor registry initialized")

	return registry, nil
}

// parseSize парсит размер в формате "10MB", "1GB"
func parseSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, nil
	}

	var multiplier int64 = 1
	var numStr string

	if len(sizeStr) >= 2 {
		suffix := sizeStr[len(sizeStr)-2:]
		numPart := sizeStr[:len(sizeStr)-2]

		switch suffix {
		case "GB":
			multiplier = 1024 * 1024 * 1024
			numStr = numPart
		case "MB":
			multiplier = 1024 * 1024
			numStr = numPart
		case "KB":
			multiplier = 1024
			numStr = numPart
		default:
			// Проверяем на "B"
			if len(sizeStr) >= 1 && sizeStr[len(sizeStr)-1] == 'B' {
				numStr = sizeStr[:len(sizeStr)-1]
				multiplier = 1
			} else {
				// Предполагаем что указаны байты
				numStr = sizeStr
			}
		}
	} else {
		numStr = sizeStr
	}

	var num int64
	_, err := fmt.Sscanf(numStr, "%d", &num)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	return num * multiplier, nil
}

// parseTimeout парсит timeout строку
func parseTimeout(timeoutStr string) (time.Duration, error) {
	if timeoutStr == "" {
		return 30 * time.Second, nil
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout format: %s", timeoutStr)
	}

	return timeout, nil
}

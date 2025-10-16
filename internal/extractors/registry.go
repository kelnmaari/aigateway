package extractors

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// Registry реестр экстракторов
type Registry struct {
	extractors map[string]DocumentExtractor // MIME type -> extractor
	mu         sync.RWMutex
	logger     *logrus.Logger
}

// NewRegistry создает новый реестр экстракторов
func NewRegistry(logger *logrus.Logger) *Registry {
	return &Registry{
		extractors: make(map[string]DocumentExtractor),
		logger:     logger,
	}
}

// Register регистрирует extractor
func (r *Registry) Register(extractor DocumentExtractor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	supportedTypes := extractor.SupportedTypes()
	if len(supportedTypes) == 0 {
		return fmt.Errorf("extractor supports no MIME types")
	}

	for _, mimeType := range supportedTypes {
		mimeType = strings.ToLower(mimeType)
		r.extractors[mimeType] = extractor
		r.logger.WithFields(logrus.Fields{
			"mime_type": mimeType,
			"extractor": fmt.Sprintf("%T", extractor),
		}).Debug("Registered document extractor")
	}

	return nil
}

// Extract извлекает текст используя подходящий extractor
func (r *Registry) Extract(ctx context.Context, reader io.Reader, opts ExtractOptions) (*ExtractedDocument, error) {
	// Получаем extractor для MIME type
	extractor := r.getExtractor(opts.MimeType)
	if extractor == nil {
		return nil, fmt.Errorf("no extractor found for MIME type: %s", opts.MimeType)
	}

	// Извлекаем текст
	doc, err := extractor.Extract(ctx, reader, opts)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"mime_type":  opts.MimeType,
		"filename":   opts.Filename,
		"word_count": doc.WordCount,
	}).Info("Document extracted successfully")

	return doc, nil
}

// getExtractor возвращает extractor для MIME type
func (r *Registry) getExtractor(mimeType string) DocumentExtractor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mimeType = strings.ToLower(mimeType)
	return r.extractors[mimeType]
}

// Supports проверяет поддержку MIME type
func (r *Registry) Supports(mimeType string) bool {
	return r.getExtractor(mimeType) != nil
}

// SupportedTypes возвращает все поддерживаемые MIME types
func (r *Registry) SupportedTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.extractors))
	for mimeType := range r.extractors {
		types = append(types, mimeType)
	}
	return types
}

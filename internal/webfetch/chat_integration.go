package webfetch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// ChatIntegration интеграция web fetch с чатом
type ChatIntegration struct {
	service  *Service
	detector *URLDetector
	logger   *logrus.Logger
}

// NewChatIntegration создает новую интеграцию
func NewChatIntegration(service *Service, logger *logrus.Logger) *ChatIntegration {
	return &ChatIntegration{
		service:  service,
		detector: NewURLDetector(),
		logger:   logger,
	}
}

// ProcessMessageOptions опции обработки сообщения
type ProcessMessageOptions struct {
	AutoFetch      bool          // Автоматически fetch URLs
	MaxURLs        int           // Максимум URLs для fetch (default: 3)
	IncludeHTML    bool          // Включать HTML контент
	Summarize      bool          // Генерировать summary через LLM
	Timeout        time.Duration // Timeout для fetch (default: 30s)
	TruncateLength int           // Максимальная длина контента на страницу (0 = без ограничений)
}

// ProcessedMessage результат обработки сообщения
type ProcessedMessage struct {
	OriginalMessage string
	EnhancedMessage string // Message с добавленным контекстом
	DetectedURLs    []string
	FetchedPages    []*WebPage
	FetchErrors     map[string]error
	HasWebContent   bool
}

// ProcessMessage обрабатывает сообщение и извлекает web контент
func (ci *ChatIntegration) ProcessMessage(ctx context.Context, message string, opts ProcessMessageOptions) (*ProcessedMessage, error) {
	// Detect URLs
	urls := ci.detector.DetectURLs(message)

	result := &ProcessedMessage{
		OriginalMessage: message,
		EnhancedMessage: message,
		DetectedURLs:    urls,
		FetchedPages:    make([]*WebPage, 0),
		FetchErrors:     make(map[string]error),
		HasWebContent:   false,
	}

	// If no URLs or auto-fetch disabled, return as-is
	if len(urls) == 0 || !opts.AutoFetch {
		return result, nil
	}

	// Limit URLs
	maxURLs := opts.MaxURLs
	if maxURLs == 0 {
		maxURLs = 3 // Default: fetch max 3 URLs
	}
	if len(urls) > maxURLs {
		ci.logger.WithFields(logrus.Fields{
			"detected_urls": len(urls),
			"max_urls":      maxURLs,
		}).Warn("Too many URLs detected, limiting to max")
		urls = urls[:maxURLs]
	}

	// Set timeout
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Fetch URLs
	fetchOpts := FetchOptions{
		Timeout:         timeout,
		MaxSize:         5 * 1024 * 1024, // 5MB max
		FollowRedirects: true,
		ExtractLinks:    false,
		Summarize:       opts.Summarize,
		CacheEnabled:    true,
		CacheTTL:        1 * time.Hour,
	}

	for _, url := range urls {
		ci.logger.WithField("url", url).Debug("Fetching web content for chat context")

		page, err := ci.service.FetchURL(ctx, url, fetchOpts)
		if err != nil {
			ci.logger.WithError(err).WithField("url", url).Warn("Failed to fetch URL for chat context")
			result.FetchErrors[url] = err
			continue
		}

		result.FetchedPages = append(result.FetchedPages, page)
		result.HasWebContent = true
	}

	// Enhance message with web content
	if result.HasWebContent {
		result.EnhancedMessage = ci.buildEnhancedMessage(message, result.FetchedPages, opts)
	}

	ci.logger.WithFields(logrus.Fields{
		"urls_detected": len(urls),
		"urls_fetched":  len(result.FetchedPages),
		"urls_failed":   len(result.FetchErrors),
	}).Info("Processed message with web content")

	return result, nil
}

// buildEnhancedMessage создает улучшенное сообщение с web контентом
func (ci *ChatIntegration) buildEnhancedMessage(originalMessage string, pages []*WebPage, opts ProcessMessageOptions) string {
	var builder strings.Builder

	// Original user message
	builder.WriteString(originalMessage)
	builder.WriteString("\n\n")

	// Add web content
	builder.WriteString("---\n")
	builder.WriteString("📄 **Extracted Web Content:**\n\n")

	for i, page := range pages {
		builder.WriteString(fmt.Sprintf("**[%d] %s**\n", i+1, page.Title))
		builder.WriteString(fmt.Sprintf("URL: %s\n", page.URL))

		if page.Metadata != nil && page.Metadata.Description != "" {
			builder.WriteString(fmt.Sprintf("Description: %s\n", page.Metadata.Description))
		}

		builder.WriteString("\n**Content:**\n")

		// Применяем truncation только если TruncateLength > 0
		content := page.Content
		if opts.TruncateLength > 0 && len(content) > opts.TruncateLength {
			content = content[:opts.TruncateLength] + fmt.Sprintf("... [truncated, original length: %d chars]", len(page.Content))
			ci.logger.WithFields(logrus.Fields{
				"url":             page.URL,
				"original_length": len(page.Content),
				"truncated_to":    opts.TruncateLength,
			}).Debug("Web content truncated for context limits")
		}

		builder.WriteString(content)
		builder.WriteString("\n\n")

		if page.Summary != "" {
			builder.WriteString(fmt.Sprintf("**Summary:** %s\n\n", page.Summary))
		}

		// Показываем статистику если контент большой
		if len(page.Content) > 10000 {
			builder.WriteString(fmt.Sprintf("*[Full content length: %d chars, %d words]*\n\n", 
				len(page.Content), page.WordCount))
		}

		builder.WriteString("---\n\n")
	}

	// Add instruction for LLM
	builder.WriteString("**Note:** Use the above web content to answer the user's question. ")
	builder.WriteString("Reference specific information from the pages when relevant.\n")

	return builder.String()
}

// FormatWebContentAsSystemMessage форматирует web контент как system message
// truncateLength: максимальная длина контента на страницу (0 = без ограничений)
func (ci *ChatIntegration) FormatWebContentAsSystemMessage(pages []*WebPage, truncateLength int) string {
	var builder strings.Builder

	builder.WriteString("You have access to the following web pages:\n\n")

	for i, page := range pages {
		builder.WriteString(fmt.Sprintf("Page %d: %s\n", i+1, page.Title))
		builder.WriteString(fmt.Sprintf("URL: %s\n", page.URL))

		// Применяем truncation только если задано
		content := page.Content
		if truncateLength > 0 && len(content) > truncateLength {
			content = content[:truncateLength] + fmt.Sprintf("... [truncated, full length: %d chars]", len(page.Content))
		}

		builder.WriteString(fmt.Sprintf("Content:\n%s\n\n", content))

		if page.Summary != "" {
			builder.WriteString(fmt.Sprintf("Summary: %s\n\n", page.Summary))
		}
	}

	builder.WriteString("Use this information to answer the user's questions accurately.\n")

	return builder.String()
}

// ExtractWebContentSummary создает краткую сводку извлеченного контента
func (ci *ChatIntegration) ExtractWebContentSummary(pages []*WebPage) string {
	if len(pages) == 0 {
		return ""
	}

	var titles []string
	for _, page := range pages {
		titles = append(titles, fmt.Sprintf("'%s'", page.Title))
	}

	return fmt.Sprintf("Fetched %d page(s): %s", len(pages), strings.Join(titles, ", "))
}


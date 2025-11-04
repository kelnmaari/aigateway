// Package worker provides RAG job processing workers.
package worker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"aigateway/internal/models"
)

// executeWebScrape выполняет web scraping из URL
func (w *RAGWorker) executeWebScrape(ctx context.Context, sourceID string) error {
	w.logger.WithField("source_id", sourceID).Info("Starting web scrape")

	// 1. Получить source configuration
	source, err := w.db.GetRAGDataSource(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("failed to get RAG data source: %w", err)
	}

	// 2. Удалить старые данные перед синхронизацией
	w.logger.WithField("source_id", sourceID).Info("Deleting old chunks before sync")
	if err := w.db.DeleteChunksBySource(ctx, sourceID); err != nil {
		w.logger.WithError(err).Warn("Failed to delete old chunks, continuing anyway")
	}

	// 3. Извлечь URLs из config
	urls := extractURLs(source.Config)
	if len(urls) == 0 {
		return fmt.Errorf("no URLs found in config (expected 'url' or 'urls' field)")
	}

	w.logger.WithFields(logrus.Fields{
		"source_id":  sourceID,
		"urls_count": len(urls),
	}).Info("Scraping URLs")

	var (
		totalChunks int
		totalTokens int64
		errors      []string
	)

	// 3. Обработать каждый URL
	for _, url := range urls {
		chunks, tokens, err := w.scrapeURL(ctx, source, url)
		if err != nil {
			w.logger.WithError(err).WithField("url", url).Error("Failed to scrape URL")
			errors = append(errors, fmt.Sprintf("%s: %v", url, err))
			continue
		}

		totalChunks += chunks
		totalTokens += tokens
	}

	// 4. Обновить source statistics
	now := time.Now()
	source.TotalChunks += totalChunks
	source.TotalTokens += totalTokens
	source.LastChunkCount = totalChunks
	source.LastSyncAt = &now

	if len(errors) == 0 {
		success := models.SyncStatusSuccess
		source.LastSyncStatus = &success
		source.LastError = ""
		source.Status = models.SourceStatusActive
	} else if totalChunks > 0 {
		// Partial success
		partial := models.SyncStatusPartial
		source.LastSyncStatus = &partial
		source.LastError = strings.Join(errors, "; ")
		source.Status = models.SourceStatusActive
	} else {
		// Complete failure
		failed := models.SyncStatusFailed
		source.LastSyncStatus = &failed
		source.LastError = strings.Join(errors, "; ")
		source.Status = models.SourceStatusError
		return fmt.Errorf("all URLs failed to scrape: %s", source.LastError)
	}

	if err := w.db.UpdateRAGDataSource(ctx, source); err != nil {
		return fmt.Errorf("failed to update source stats: %w", err)
	}

	w.logger.WithFields(logrus.Fields{
		"source_id":     source.ID,
		"urls_scraped":  len(urls),
		"urls_failed":   len(errors),
		"total_chunks":  totalChunks,
		"total_tokens":  totalTokens,
	}).Info("Web scrape completed")

	return nil
}

// scrapeURL скрейпит один URL
func (w *RAGWorker) scrapeURL(ctx context.Context, source *models.RAGDataSource, url string) (int, int64, error) {
	w.logger.WithFields(logrus.Fields{
		"source_id": source.ID,
		"url":       url,
	}).Debug("Scraping URL")

	// 1. Выполнить HTTP GET запрос
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent to avoid blocks
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; RAGBot/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, 0, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	// 2. Читать HTML content
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read response body: %w", err)
	}

	// 3. Извлечь текст из HTML
	text := extractTextFromHTML(string(body))
	if text == "" {
		return 0, 0, fmt.Errorf("no text extracted from HTML")
	}

	w.logger.WithFields(logrus.Fields{
		"source_id":   source.ID,
		"url":         url,
		"text_length": len(text),
	}).Debug("Text extracted from HTML")

	// 4. Создать document
	now := time.Now()
	document := &models.RAGDocument{
		ID:                    w.generateDocumentID(),
		SourceID:              source.ID,
		Filename:              sanitizeFilename(url),
		MimeType:              "text/html",
		SizeBytes:             int64(len(body)),
		StorageBackend:        "memory",
		StoragePath:           url, // Store original URL
		Status:                models.DocumentStatusCompleted,
		ProcessingStartedAt:   &now,
		ProcessingCompletedAt: &now,
		Metadata: models.DocumentMetadata{
			"url":         url,
			"status_code": resp.StatusCode,
			"content_type": resp.Header.Get("Content-Type"),
		},
		TotalChunks: 0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := w.db.CreateRAGDocument(ctx, document); err != nil {
		return 0, 0, fmt.Errorf("failed to create document: %w", err)
	}

	// 5. Создать chunks из текста
	chunks, totalTokens := w.createChunksFromText(source, document.ID, text)

	// Add URL to chunk metadata
	for _, chunk := range chunks {
		chunk.Metadata["url"] = url
	}
	
	// 6. Генерируем embeddings если embedder доступен (Version 1.14.0+)
	if err := w.generateAndStoreEmbeddings(ctx, chunks); err != nil {
		w.logger.WithError(err).Warn("Failed to generate embeddings, continuing without them")
	}
	
	// 7. Сохраняем chunks в БД
	for _, chunk := range chunks {
		if err := w.db.CreateRAGChunk(ctx, chunk); err != nil {
			w.logger.WithError(err).Error("Failed to create chunk")
			continue
		}
	}

	// 8. Обновить document statistics
	document.TotalChunks = len(chunks)
	if err := w.db.UpdateRAGDocument(ctx, document); err != nil {
		w.logger.WithError(err).Warn("Failed to update document stats")
	}

	w.logger.WithFields(logrus.Fields{
		"source_id":    source.ID,
		"url":          url,
		"chunks_count": len(chunks),
		"tokens_count": totalTokens,
	}).Info("URL scraped successfully")

	return len(chunks), totalTokens, nil
}

// extractURLs извлекает URLs из config
func extractURLs(config models.SourceConfig) []string {
	var urls []string

	// Check for single URL
	if url, ok := config["url"].(string); ok && url != "" {
		urls = append(urls, url)
	}

	// Check for multiple URLs
	if urlsRaw, ok := config["urls"]; ok {
		switch v := urlsRaw.(type) {
		case []interface{}:
			for _, urlRaw := range v {
				if url, ok := urlRaw.(string); ok && url != "" {
					urls = append(urls, url)
				}
			}
		case []string:
			urls = append(urls, v...)
		}
	}

	return urls
}

// extractTextFromHTML извлекает текст из HTML
func extractTextFromHTML(html string) string {
	// Remove script and style tags
	html = removeTag(html, "script")
	html = removeTag(html, "style")
	html = removeTag(html, "noscript")

	// Remove HTML tags
	html = removeHTMLTags(html)

	// Decode HTML entities
	html = decodeHTMLEntities(html)

	// Clean up whitespace
	html = cleanWhitespace(html)

	return strings.TrimSpace(html)
}

// removeTag удаляет указанный HTML тег и его содержимое
func removeTag(html, tag string) string {
	// Case insensitive regex
	pattern := fmt.Sprintf(`(?i)<%s[^>]*>.*?</%s>`, tag, tag)
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllString(html, "")
}

// removeHTMLTags удаляет все HTML теги
func removeHTMLTags(html string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return re.ReplaceAllString(html, " ")
}

// decodeHTMLEntities декодирует HTML entities
func decodeHTMLEntities(text string) string {
	// Common HTML entities
	replacements := map[string]string{
		"&nbsp;":  " ",
		"&lt;":    "<",
		"&gt;":    ">",
		"&amp;":   "&",
		"&quot;":  "\"",
		"&apos;":  "'",
		"&#39;":   "'",
		"&mdash;": "—",
		"&ndash;": "–",
		"&hellip;":"...",
		"&copy;":  "©",
		"&reg;":   "®",
		"&trade;": "™",
	}

	for entity, replacement := range replacements {
		text = strings.ReplaceAll(text, entity, replacement)
	}

	// Decode numeric entities (&#[0-9]+;)
	numericRe := regexp.MustCompile(`&#(\d+);`)
	text = numericRe.ReplaceAllStringFunc(text, func(match string) string {
		var code int
		if _, err := fmt.Sscanf(match, "&#%d;", &code); err == nil {
			if code < 128 {
				return string(rune(code))
			}
		}
		return match
	})

	return text
}

// cleanWhitespace убирает лишние пробелы и переносы строк
func cleanWhitespace(text string) string {
	// Replace multiple spaces with single space
	re := regexp.MustCompile(`[ \t]+`)
	text = re.ReplaceAllString(text, " ")

	// Replace multiple newlines with double newline
	re = regexp.MustCompile(`\n\s*\n\s*\n+`)
	text = re.ReplaceAllString(text, "\n\n")

	// Trim spaces at start/end of lines
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	text = strings.Join(lines, "\n")

	return text
}

// sanitizeFilename создает безопасное имя файла из URL
func sanitizeFilename(url string) string {
	// Remove protocol
	name := strings.TrimPrefix(url, "http://")
	name = strings.TrimPrefix(name, "https://")

	// Replace special characters
	name = regexp.MustCompile(`[^a-zA-Z0-9_-]+`).ReplaceAllString(name, "_")

	// Limit length
	if len(name) > 100 {
		name = name[:100]
	}

	return name + ".html"
}


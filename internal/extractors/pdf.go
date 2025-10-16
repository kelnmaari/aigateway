package extractors

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
)

// PDFExtractor извлекает текст из PDF файлов
type PDFExtractor struct {
	method         string // "auto", "pdftotext", или "go-pdf"
	preserveLayout bool
	maxPages       int
}

// NewPDFExtractor создает новый PDF extractor
func NewPDFExtractor(method string, preserveLayout bool, maxPages int) *PDFExtractor {
	return &PDFExtractor{
		method:         method,
		preserveLayout: preserveLayout,
		maxPages:       maxPages,
	}
}

// Extract извлекает текст из PDF файла
func (e *PDFExtractor) Extract(ctx context.Context, reader io.Reader, opts ExtractOptions) (*ExtractedDocument, error) {
	// Выбираем метод extraction
	switch e.method {
	case "pdftotext":
		// Пробуем pdftotext, если не работает - fallback на Go-библиотеку
		doc, err := e.extractWithPDFToText(ctx, reader, opts)
		if err == nil {
			return doc, nil
		}
		// Если pdftotext не найден, пробуем Go-библиотеку
		if strings.Contains(err.Error(), "pdftotext not available") {
			return e.extractWithGoPDF(ctx, reader, opts)
		}
		return nil, err
	case "go-pdf":
		return e.extractWithGoPDF(ctx, reader, opts)
	case "auto":
		// Автоматический выбор: пробуем pdftotext, затем Go-библиотеку
		if e.isPDFToTextAvailable() {
			return e.extractWithPDFToText(ctx, reader, opts)
		}
		return e.extractWithGoPDF(ctx, reader, opts)
	default:
		return nil, fmt.Errorf("unsupported PDF extraction method: %s", e.method)
	}
}

// extractWithPDFToText использует утилиту pdftotext
func (e *PDFExtractor) extractWithPDFToText(ctx context.Context, reader io.Reader, opts ExtractOptions) (*ExtractedDocument, error) {
	// Сохраняем во временный файл
	tmpFile, err := e.saveTempFile(reader)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile)

	// Запускаем pdftotext
	args := []string{tmpFile, "-"}
	if !e.preserveLayout {
		args = append(args, "-layout")
	}
	if e.maxPages > 0 {
		args = append(args, "-l", strconv.Itoa(e.maxPages))
	}

	cmd := exec.CommandContext(ctx, "pdftotext", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// Проверяем доступность pdftotext
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Если pdftotext не найден, возвращаем специальную ошибку для fallback
		if strings.Contains(err.Error(), "executable file not found") {
			return nil, fmt.Errorf("pdftotext not available: %w", err)
		}

		return nil, fmt.Errorf("pdftotext failed: %w, stderr: %s", err, stderr.String())
	}

	text := stdout.String()

	// Извлекаем метаданные с помощью pdfinfo
	metadata := e.extractMetadataWithPDFInfo(ctx, tmpFile)

	// Подсчитываем слова
	wordCount := countWords(text)

	// Определяем язык
	language := detectLanguage(text)

	return &ExtractedDocument{
		Text:      text,
		WordCount: wordCount,
		Language:  language,
		Metadata:  metadata,
	}, nil
}

// extractMetadataWithPDFInfo использует pdfinfo для получения метаданных
func (e *PDFExtractor) extractMetadataWithPDFInfo(ctx context.Context, filename string) DocumentMetadata {
	metadata := DocumentMetadata{
		Custom: make(map[string]interface{}),
	}

	// Запускаем pdfinfo
	cmd := exec.CommandContext(ctx, "pdfinfo", filename)
	output, err := cmd.Output()
	if err != nil {
		// Не критично, возвращаем пустые метаданные
		return metadata
	}

	// Парсим вывод pdfinfo
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "Title":
			metadata.Title = value
		case "Author":
			metadata.Author = value
		case "Subject":
			metadata.Subject = value
		case "Keywords":
			metadata.Keywords = strings.Split(value, ",")
		case "CreationDate":
			metadata.CreatedAt = value
		case "ModDate":
			metadata.ModifiedAt = value
		case "Pages":
			if pages, err := strconv.Atoi(value); err == nil {
				metadata.PageCount = pages
			}
		}
	}

	return metadata
}

// saveTempFile сохраняет reader во временный файл
func (e *PDFExtractor) saveTempFile(reader io.Reader) (string, error) {
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("pdf_extract_%d.pdf", os.Getpid()))

	f, err := os.Create(tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, reader)
	if err != nil {
		os.Remove(tmpFile)
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	return tmpFile, nil
}

// isPDFToTextAvailable проверяет доступность утилиты pdftotext
func (e *PDFExtractor) isPDFToTextAvailable() bool {
	_, err := exec.LookPath("pdftotext")
	return err == nil
}

// extractWithGoPDF извлекает текст используя чистую Go-библиотеку
// Работает на всех платформах без внешних зависимостей
func (e *PDFExtractor) extractWithGoPDF(ctx context.Context, reader io.Reader, opts ExtractOptions) (*ExtractedDocument, error) {
	// Сохраняем во временный файл (библиотека требует файл)
	tmpFile, err := e.saveTempFile(reader)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile)

	// Открываем PDF
	f, r, err := pdf.Open(tmpFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	// Извлекаем текст из всех страниц
	var textBuilder strings.Builder
	totalPages := r.NumPage()

	// Ограничиваем количество страниц если задано
	maxPages := totalPages
	if e.maxPages > 0 && e.maxPages < totalPages {
		maxPages = e.maxPages
	}

	for pageNum := 1; pageNum <= maxPages; pageNum++ {
		// Проверяем контекст
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		page := r.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		// Извлекаем текст страницы
		text, err := page.GetPlainText(nil)
		if err != nil {
			// Не критично, пропускаем страницу
			continue
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n\n")
	}

	extractedText := textBuilder.String()

	// Создаем метаданные
	metadata := DocumentMetadata{
		PageCount: totalPages,
		Custom: map[string]interface{}{
			"extraction_method": "go-pdf",
			"extracted_pages":   maxPages,
		},
	}

	// Пробуем извлечь метаданные из PDF
	trailer := r.Trailer()
	if !trailer.IsNull() {
		info := trailer.Key("Info")
		if !info.IsNull() {
			title := info.Key("Title")
			if !title.IsNull() {
				metadata.Title = title.String()
			}
			author := info.Key("Author")
			if !author.IsNull() {
				metadata.Author = author.String()
			}
			subject := info.Key("Subject")
			if !subject.IsNull() {
				metadata.Subject = subject.String()
			}
		}
	}

	// Подсчитываем слова
	wordCount := countWords(extractedText)

	// Определяем язык
	language := detectLanguage(extractedText)

	return &ExtractedDocument{
		Text:      extractedText,
		WordCount: wordCount,
		Language:  language,
		Metadata:  metadata,
	}, nil
}

// SupportedTypes возвращает поддерживаемые MIME types
func (e *PDFExtractor) SupportedTypes() []string {
	return []string{
		"application/pdf",
	}
}

// MaxFileSize возвращает максимальный размер файла
func (e *PDFExtractor) MaxFileSize() int64 {
	// PDF может быть большим
	return 100 * 1024 * 1024 // 100MB
}

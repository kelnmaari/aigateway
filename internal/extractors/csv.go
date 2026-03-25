package extractors

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// CSVExtractor извлекает данные из CSV файлов
type CSVExtractor struct {
	maxRows    int
	delimiter  rune
	encoding   string
	autoDetect bool
}

// NewCSVExtractor создает новый CSV extractor
func NewCSVExtractor(maxRows int, delimiter rune, encoding string, autoDetect bool) *CSVExtractor {
	return &CSVExtractor{
		maxRows:    maxRows,
		delimiter:  delimiter,
		encoding:   encoding,
		autoDetect: autoDetect,
	}
}

// Extract извлекает данные из CSV файла
func (e *CSVExtractor) Extract(ctx context.Context, reader io.Reader, opts ExtractOptions) (*ExtractedDocument, error) {
	// Читаем все данные
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	// Определяем delimiter если autoDetect
	delimiter := e.delimiter
	if e.autoDetect || opts.Delimiter != 0 {
		delimiter = e.detectDelimiter(data, opts.Delimiter)
	}

	// Парсим CSV
	csvReader := csv.NewReader(bytes.NewReader(data))
	csvReader.Comma = delimiter
	csvReader.LazyQuotes = true
	csvReader.TrimLeadingSpace = true

	// Читаем все записи
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	// Проверяем лимит строк
	if e.maxRows > 0 && len(records) > e.maxRows {
		return nil, fmt.Errorf("CSV has %d rows, exceeds maximum %d", len(records), e.maxRows)
	}

	// Извлекаем заголовки (первая строка)
	headers := records[0]
	rows := records[1:]

	// Формируем текстовое представление
	text := e.recordsToText(headers, rows)

	// Создаем таблицу
	table := Table{
		Name:    opts.Filename,
		Headers: headers,
		Rows:    rows,
		Page:    1,
	}

	// Подсчитываем слова
	wordCount := countWords(text)

	return &ExtractedDocument{
		Text:      text,
		WordCount: wordCount,
		Language:  "en", // CSV обычно англоязычный
		Metadata: DocumentMetadata{
			RowCount: len(rows),
			Custom: map[string]any{
				"columns":   len(headers),
				"delimiter": string(delimiter),
			},
		},
		Tables: []Table{table},
	}, nil
}

// detectDelimiter определяет delimiter автоматически
func (e *CSVExtractor) detectDelimiter(data []byte, preferred rune) rune {
	// Если указан предпочтительный, используем его
	if preferred != 0 {
		return preferred
	}

	// Проверяем первую строку
	firstLine := bytes.SplitN(data, []byte("\n"), 2)[0]
	line := string(firstLine)

	// Считаем частоту разных delimiters
	delimiters := []rune{',', ';', '\t', '|'}
	maxCount := 0
	bestDelimiter := ','

	for _, delim := range delimiters {
		count := strings.Count(line, string(delim))
		if count > maxCount {
			maxCount = count
			bestDelimiter = delim
		}
	}

	return bestDelimiter
}

// recordsToText конвертирует CSV records в текст
func (e *CSVExtractor) recordsToText(headers []string, rows [][]string) string {
	var builder strings.Builder

	// Добавляем заголовки
	builder.WriteString("Headers: ")
	builder.WriteString(strings.Join(headers, ", "))
	builder.WriteString("\n\n")

	// Добавляем данные
	for i, row := range rows {
		builder.WriteString(fmt.Sprintf("Row %d:\n", i+1))

		for j, cell := range row {
			if j < len(headers) {
				builder.WriteString(fmt.Sprintf("  %s: %s\n", headers[j], cell))
			}
		}

		builder.WriteString("\n")

		// Ограничиваем вывод для больших CSV
		if i >= 100 {
			builder.WriteString(fmt.Sprintf("... and %d more rows\n", len(rows)-i-1))
			break
		}
	}

	return builder.String()
}

// SupportedTypes возвращает поддерживаемые MIME types
func (e *CSVExtractor) SupportedTypes() []string {
	return []string{
		"text/csv",
		"application/csv",
	}
}

// MaxFileSize возвращает максимальный размер файла
func (e *CSVExtractor) MaxFileSize() int64 {
	// CSV может быть большим, используем 100MB
	return 100 * 1024 * 1024
}

package extractors

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	encodingunicode "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// TextExtractor извлекает текст из plain text файлов
type TextExtractor struct {
	maxSize           int64
	encoding          string
	fallbackEncodings []string
}

// NewTextExtractor создает новый text extractor
func NewTextExtractor(maxSize int64, encoding string, fallbackEncodings []string) *TextExtractor {
	return &TextExtractor{
		maxSize:           maxSize,
		encoding:          encoding,
		fallbackEncodings: fallbackEncodings,
	}
}

// Extract извлекает текст из plain text файла
func (e *TextExtractor) Extract(ctx context.Context, reader io.Reader, opts ExtractOptions) (*ExtractedDocument, error) {
	// Читаем весь файл
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Проверяем размер
	if e.maxSize > 0 && int64(len(data)) > e.maxSize {
		return nil, fmt.Errorf("file size %d exceeds maximum %d", len(data), e.maxSize)
	}

	// Пытаемся декодировать с указанной кодировкой
	text, detectedEncoding, err := e.decodeText(data, opts.Encoding)
	if err != nil {
		return nil, fmt.Errorf("failed to decode text: %w", err)
	}

	// Подсчитываем слова
	wordCount := countWords(text)

	// Определяем язык (простая эвристика)
	language := detectLanguage(text)

	return &ExtractedDocument{
		Text:      text,
		WordCount: wordCount,
		Language:  language,
		Metadata: DocumentMetadata{
			Custom: map[string]interface{}{
				"encoding": detectedEncoding,
			},
		},
	}, nil
}

// decodeText декодирует текст с автоопределением кодировки
func (e *TextExtractor) decodeText(data []byte, preferredEncoding string) (string, string, error) {
	// Проверяем BOM (Byte Order Mark)
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		// UTF-8 BOM
		return string(data[3:]), "utf-8-bom", nil
	}
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		// UTF-16 LE BOM
		text, err := e.tryDecode(data[2:], "utf-16")
		if err == nil {
			return text, "utf-16-le", nil
		}
	}

	// Пробуем предпочтительную кодировку если указана
	if preferredEncoding != "" {
		if text, err := e.tryDecode(data, preferredEncoding); err == nil {
			// Проверяем что результат имеет смысл (не слишком много управляющих символов)
			if isReasonableText(text) {
				return text, preferredEncoding, nil
			}
		}
	}

	// Проверяем валидность UTF-8
	if utf8.Valid(data) {
		text := string(data)
		// Дополнительная проверка: если много байтов >127 и нет кириллицы - возможно это не UTF-8
		if hasHighBytes(data) && !hasCyrillic(text) {
			// Пробуем Windows-1251 (частая кодировка для русских текстов)
			if win1251Text, err := e.tryDecode(data, "windows-1251"); err == nil {
				if hasCyrillic(win1251Text) && isReasonableText(win1251Text) {
					return win1251Text, "windows-1251", nil
				}
			}
		}
		// Если UTF-8 валиден и содержит кириллицу или только ASCII - используем его
		return text, "utf-8", nil
	}

	// Пробуем fallback кодировки по порядку
	for _, enc := range e.fallbackEncodings {
		if text, err := e.tryDecode(data, enc); err == nil {
			if isReasonableText(text) {
				return text, enc, nil
			}
		}
	}

	// Последняя попытка - Windows-1251 (самая частая для русских текстов)
	if text, err := e.tryDecode(data, "windows-1251"); err == nil {
		return text, "windows-1251-fallback", nil
	}

	// Если ничего не сработало, возвращаем как UTF-8 с заменой невалидных символов
	return string(data), "utf-8-fallback", nil
}

// hasHighBytes проверяет наличие байтов >127 (не-ASCII)
func hasHighBytes(data []byte) bool {
	highByteCount := 0
	for _, b := range data {
		if b > 127 {
			highByteCount++
		}
	}
	// Если больше 10% байтов >127, считаем что есть не-ASCII символы
	return highByteCount > len(data)/10
}

// hasCyrillic проверяет наличие кириллицы в тексте
func hasCyrillic(text string) bool {
	for _, r := range text {
		if r >= 0x0400 && r <= 0x04FF {
			return true
		}
	}
	return false
}

// isReasonableText проверяет что текст выглядит разумно (мало управляющих символов)
func isReasonableText(text string) bool {
	if len(text) == 0 {
		return false
	}

	controlCount := 0
	totalChars := 0

	for _, r := range text {
		totalChars++
		// Считаем управляющие символы (кроме \n, \r, \t)
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			controlCount++
		}
		// Проверяем replacement characters (�)
		if r == 0xFFFD {
			controlCount++
		}
	}

	// Если больше 5% управляющих символов - текст подозрительный
	return float64(controlCount)/float64(totalChars) < 0.05
}

// tryDecode пытается декодировать с указанной кодировкой
func (e *TextExtractor) tryDecode(data []byte, encodingName string) (string, error) {
	var decoder *encoding.Decoder

	switch strings.ToLower(encodingName) {
	case "utf-8":
		decoder = encodingunicode.UTF8.NewDecoder()
	case "utf-16":
		decoder = encodingunicode.UTF16(encodingunicode.LittleEndian, encodingunicode.UseBOM).NewDecoder()
	case "windows-1251":
		decoder = charmap.Windows1251.NewDecoder()
	case "iso-8859-1":
		decoder = charmap.ISO8859_1.NewDecoder()
	default:
		return "", fmt.Errorf("unsupported encoding: %s", encodingName)
	}

	result, _, err := transform.Bytes(decoder, data)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

// SupportedTypes возвращает поддерживаемые MIME types
func (e *TextExtractor) SupportedTypes() []string {
	return []string{
		"text/plain",
		"text/markdown",
		"text/rtf",
	}
}

// MaxFileSize возвращает максимальный размер файла
func (e *TextExtractor) MaxFileSize() int64 {
	return e.maxSize
}

// countWords подсчитывает количество слов
func countWords(text string) int {
	words := 0
	inWord := false

	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			inWord = false
		} else if !inWord {
			inWord = true
			words++
		}
	}

	return words
}

// detectLanguage определяет язык текста (простая эвристика)
func detectLanguage(text string) string {
	// Подсчитываем частоту кириллицы
	cyrillicCount := 0
	latinCount := 0
	totalLetters := 0

	for _, r := range text {
		if unicode.IsLetter(r) {
			totalLetters++
			if r >= 0x0400 && r <= 0x04FF {
				cyrillicCount++
			} else if r >= 'A' && r <= 'z' {
				latinCount++
			}
		}
	}

	if totalLetters == 0 {
		return "unknown"
	}

	cyrillicRatio := float64(cyrillicCount) / float64(totalLetters)
	if cyrillicRatio > 0.3 {
		return "ru"
	}

	return "en"
}

// normalizeText нормализует текст (убирает лишние пробелы, переносы)
func normalizeText(text string) string {
	// Заменяем множественные пробелы на одинарные
	var result bytes.Buffer
	prevSpace := false

	for _, r := range text {
		if unicode.IsSpace(r) {
			if !prevSpace {
				result.WriteRune(' ')
				prevSpace = true
			}
		} else {
			result.WriteRune(r)
			prevSpace = false
		}
	}

	return strings.TrimSpace(result.String())
}


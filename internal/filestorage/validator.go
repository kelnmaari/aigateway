package filestorage

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// Validator валидирует файлы перед сохранением
type Validator struct {
	allowedExtensions map[string]bool
	maxFileSize       int64
	validateContent   bool
}

// NewValidator создает новый валидатор
func NewValidator(allowedExts []string, maxSize int64, validateContent bool) *Validator {
	extMap := make(map[string]bool)
	for _, ext := range allowedExts {
		// Нормализуем расширения (добавляем точку если нет, lowercase)
		ext = strings.ToLower(ext)
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extMap[ext] = true
	}

	return &Validator{
		allowedExtensions: extMap,
		maxFileSize:       maxSize,
		validateContent:   validateContent,
	}
}

// ValidateFile проверяет файл перед сохранением
func (v *Validator) ValidateFile(filename string, size int64, reader io.Reader) error {
	// Проверка расширения файла
	ext := strings.ToLower(filepath.Ext(filename))
	if len(v.allowedExtensions) > 0 && !v.allowedExtensions[ext] {
		return fmt.Errorf("file extension %s is not allowed", ext)
	}

	// Проверка размера файла
	if v.maxFileSize > 0 && size > v.maxFileSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", size, v.maxFileSize)
	}

	// Проверка magic number (MIME type validation)
	if v.validateContent && reader != nil {
		if err := v.validateMagicNumber(reader, ext); err != nil {
			return fmt.Errorf("content validation failed: %w", err)
		}
	}

	return nil
}

// ValidateFilename проверяет имя файла на безопасность
func (v *Validator) ValidateFilename(filename string) error {
	// Проверка на пустое имя
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// Проверка на path traversal
	if strings.Contains(filename, "..") {
		return fmt.Errorf("filename contains invalid characters: ..")
	}

	// Проверка на абсолютный путь
	if filepath.IsAbs(filename) {
		return fmt.Errorf("filename cannot be an absolute path")
	}

	// Проверка на слеши
	if strings.ContainsAny(filename, "/\\") {
		return fmt.Errorf("filename cannot contain path separators")
	}

	return nil
}

// validateMagicNumber проверяет magic number файла
func (v *Validator) validateMagicNumber(reader io.Reader, expectedExt string) error {
	// Читаем первые 512 байт для определения типа
	buf := make([]byte, 512)
	n, err := io.ReadFull(reader, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return err
	}
	buf = buf[:n]

	// Определяем тип файла по magic number
	mimeType := detectMimeType(buf)

	// Проверяем соответствие расширению
	expectedMime := extensionToMimeType(expectedExt)
	if expectedMime != "" && mimeType != expectedMime && !isMimeTypeCompatible(mimeType, expectedMime) {
		return fmt.Errorf("file content type %s does not match extension %s", mimeType, expectedExt)
	}

	return nil
}

// detectMimeType определяет MIME type по magic number
func detectMimeType(data []byte) string {
	if len(data) < 4 {
		return "application/octet-stream"
	}

	// PDF
	if len(data) >= 4 && string(data[0:4]) == "%PDF" {
		return "application/pdf"
	}

	// ZIP-based formats (DOCX, XLSX, ODT)
	if len(data) >= 4 && data[0] == 0x50 && data[1] == 0x4B && data[2] == 0x03 && data[3] == 0x04 {
		// Проверяем на DOCX
		if len(data) >= 30 {
			content := string(data)
			if strings.Contains(content, "word/") {
				return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
			}
			if strings.Contains(content, "xl/") {
				return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
			}
		}
		return "application/zip"
	}

	// DOC (older format)
	if len(data) >= 8 && data[0] == 0xD0 && data[1] == 0xCF && data[2] == 0x11 && data[3] == 0xE0 {
		return "application/msword"
	}

	// Text files (простая эвристика)
	isPrintable := true
	for i := 0; i < len(data) && i < 512; i++ {
		b := data[i]
		if b < 0x20 && b != 0x09 && b != 0x0A && b != 0x0D {
			isPrintable = false
			break
		}
	}
	if isPrintable {
		return "text/plain"
	}

	return "application/octet-stream"
}

// extensionToMimeType конвертирует расширение в MIME type
func extensionToMimeType(ext string) string {
	ext = strings.ToLower(ext)
	mimeMap := map[string]string{
		".pdf":  "application/pdf",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".doc":  "application/msword",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".xls":  "application/vnd.ms-excel",
		".txt":  "text/plain",
		".csv":  "text/csv",
		".md":   "text/markdown",
		".rtf":  "application/rtf",
	}
	return mimeMap[ext]
}

// isMimeTypeCompatible проверяет совместимость MIME types
func isMimeTypeCompatible(detected, expected string) bool {
	// ZIP может быть DOCX или XLSX
	if detected == "application/zip" {
		return strings.Contains(expected, "openxmlformats")
	}

	// Text/* типы совместимы между собой
	if strings.HasPrefix(detected, "text/") && strings.HasPrefix(expected, "text/") {
		return true
	}

	return false
}

// Package extractors provides document text extraction (Version 1.10.0+)
// Поддерживает PDF, DOCX, TXT, CSV, XLSX
// RAG-ready: интерфейсы готовы для chunking/embedding в v1.13.0
package extractors

import (
	"context"
	"io"
)

// DocumentExtractor извлекает текст и метаданные из документов
type DocumentExtractor interface {
	// Extract извлекает текст из документа
	Extract(ctx context.Context, reader io.Reader, opts ExtractOptions) (*ExtractedDocument, error)

	// SupportedTypes возвращает MIME types которые поддерживает этот extractor
	SupportedTypes() []string

	// MaxFileSize возвращает максимальный поддерживаемый размер файла
	MaxFileSize() int64
}

// ExtractOptions опции для extraction
type ExtractOptions struct {
	Filename string // Имя файла (для контекста)
	MimeType string // MIME type файла
	Language string // Язык документа (для OCR, future)

	// PDF-specific
	PreserveLayout bool // Сохранять layout (для PDF)
	MaxPages       int  // Максимум страниц для обработки

	// CSV-specific
	Delimiter rune   // Разделитель для CSV
	Encoding  string // Кодировка
}

// ExtractedDocument результат extraction
type ExtractedDocument struct {
	// Основное содержимое
	Text      string // Полный извлеченный текст
	WordCount int    // Количество слов
	Language  string // Определенный язык

	// Метаданные документа
	Metadata DocumentMetadata

	// Структура документа (для RAG в v1.13.0)
	Structure *DocumentStructure

	// Данные таблиц (для CSV, Excel)
	Tables []Table
}

// DocumentMetadata метаданные документа
type DocumentMetadata struct {
	// Общие поля
	Title      string
	Author     string
	Subject    string
	Keywords   []string
	CreatedAt  string // ISO date
	ModifiedAt string

	// Специфичные для типа
	PageCount  int // Для PDF, DOCX
	SheetCount int // Для Excel
	RowCount   int // Для CSV

	// Дополнительные метаданные
	Custom map[string]interface{} // Любые другие метаданные
}

// DocumentStructure структура документа (для RAG)
type DocumentStructure struct {
	Headings []Heading // Заголовки
	Sections []Section // Секции
	Pages    []Page    // Страницы (для PDF)
}

// Heading заголовок документа
type Heading struct {
	Level int // Уровень заголовка (1-6)
	Text  string
	Page  int // Номер страницы (для PDF)
}

// Section секция документа
type Section struct {
	Title   string
	Content string
	Level   int
	Page    int
}

// Page страница документа
type Page struct {
	Number  int
	Content string
}

// Table таблица из документа
type Table struct {
	Name    string     // Название таблицы
	Headers []string   // Заголовки столбцов
	Rows    [][]string // Данные строк
	Page    int        // Номер страницы (для PDF, DOCX)
}

package extractors

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// DOCXExtractor извлекает текст из DOCX файлов
type DOCXExtractor struct {
	extractTables   bool
	extractImages   bool
	extractComments bool
}

// NewDOCXExtractor создает новый DOCX extractor
func NewDOCXExtractor(extractTables, extractImages, extractComments bool) *DOCXExtractor {
	return &DOCXExtractor{
		extractTables:   extractTables,
		extractImages:   extractImages,
		extractComments: extractComments,
	}
}

// Extract извлекает текст из DOCX файла
func (e *DOCXExtractor) Extract(ctx context.Context, reader io.Reader, opts ExtractOptions) (*ExtractedDocument, error) {
	// DOCX это ZIP архив, читаем весь файл
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read DOCX: %w", err)
	}

	// Открываем как ZIP
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open DOCX as ZIP: %w", err)
	}

	// Извлекаем текст из document.xml
	text, err := e.extractText(zipReader)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text: %w", err)
	}

	// Извлекаем метаданные из core.xml
	metadata, err := e.extractMetadata(zipReader)
	if err != nil {
		// Метаданные не критичны, продолжаем
		metadata = DocumentMetadata{}
	}

	// Извлекаем таблицы если нужно
	var tables []Table
	if e.extractTables {
		tables, _ = e.extractTablesFromDocument(zipReader)
	}

	// Подсчитываем слова
	wordCount := countWords(text)

	// Определяем язык
	language := detectLanguage(text)

	return &ExtractedDocument{
		Text:      text,
		WordCount: wordCount,
		Language:  language,
		Metadata:  metadata,
		Tables:    tables,
	}, nil
}

// extractText извлекает текст из document.xml
func (e *DOCXExtractor) extractText(zipReader *zip.Reader) (string, error) {
	// Ищем document.xml
	var docFile *zip.File
	for _, f := range zipReader.File {
		if f.Name == "word/document.xml" {
			docFile = f
			break
		}
	}

	if docFile == nil {
		return "", fmt.Errorf("document.xml not found in DOCX")
	}

	// Открываем и читаем
	rc, err := docFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	// Парсим XML и извлекаем текст
	text, err := e.parseDocumentXML(data)
	if err != nil {
		return "", err
	}

	return text, nil
}

// parseDocumentXML парсит document.xml и извлекает текст
func (e *DOCXExtractor) parseDocumentXML(data []byte) (string, error) {
	// Упрощенный парсинг: ищем <w:t> теги
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var text strings.Builder
	var inText bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "t" {
				inText = true
			}
		case xml.CharData:
			if inText {
				text.Write(t)
				text.WriteRune(' ')
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inText = false
			} else if t.Name.Local == "p" {
				// Конец параграфа - добавляем перенос строки
				text.WriteRune('\n')
			}
		}
	}

	return strings.TrimSpace(text.String()), nil
}

// extractMetadata извлекает метаданные из core.xml
func (e *DOCXExtractor) extractMetadata(zipReader *zip.Reader) (DocumentMetadata, error) {
	metadata := DocumentMetadata{
		Custom: make(map[string]interface{}),
	}

	// Ищем core.xml
	var coreFile *zip.File
	for _, f := range zipReader.File {
		if strings.Contains(f.Name, "core.xml") {
			coreFile = f
			break
		}
	}

	if coreFile == nil {
		return metadata, nil // Не ошибка, просто нет метаданных
	}

	rc, err := coreFile.Open()
	if err != nil {
		return metadata, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return metadata, err
	}

	// Простой парсинг метаданных
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var currentElement string

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			currentElement = t.Name.Local
		case xml.CharData:
			value := strings.TrimSpace(string(t))
			if value == "" {
				continue
			}

			switch currentElement {
			case "title":
				metadata.Title = value
			case "creator":
				metadata.Author = value
			case "subject":
				metadata.Subject = value
			case "created":
				metadata.CreatedAt = value
			case "modified":
				metadata.ModifiedAt = value
			}
		}
	}

	return metadata, nil
}

// extractTablesFromDocument извлекает таблицы из документа
func (e *DOCXExtractor) extractTablesFromDocument(zipReader *zip.Reader) ([]Table, error) {
	// TODO: Implement table extraction
	// Это более сложная задача, требует парсинга <w:tbl> элементов
	return nil, nil
}

// SupportedTypes возвращает поддерживаемые MIME types
func (e *DOCXExtractor) SupportedTypes() []string {
	return []string{
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/msword", // Для .doc тоже попробуем
	}
}

// MaxFileSize возвращает максимальный размер файла
func (e *DOCXExtractor) MaxFileSize() int64 {
	// DOCX может быть довольно большим
	return 50 * 1024 * 1024 // 50MB
}

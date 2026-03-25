// Package chunker provides text chunking implementations for RAG system.
package chunker

import (
	"context"
)

// Chunk represents a text chunk with metadata
type Chunk struct {
	Text        string         // Текст чанка
	Index       int            // Позиция в документе
	Tokens      int            // Количество токенов
	StartOffset int            // Начальная позиция в исходном тексте
	EndOffset   int            // Конечная позиция в исходном тексте
	Metadata    map[string]any // Метаданные (page_number, headers, etc.)
}

// ChunkingOptions опции для chunking
type ChunkingOptions struct {
	MaxTokens     int            // Максимальный размер чанка в токенах
	Overlap       int            // Overlap в токенах
	Strategy      string         // "semantic", "fixed", "hierarchical"
	PreserveLines bool           // Сохранять границы строк
	Metadata      map[string]any // Дополнительные метаданные
}

// Chunker interface для разбиения текста на chunks
type Chunker interface {
	// Chunk разбивает текст на chunks
	Chunk(ctx context.Context, text string, options ChunkingOptions) ([]Chunk, error)

	// EstimateTokens оценивает количество токенов в тексте
	EstimateTokens(text string) int

	// Name возвращает имя chunker'а
	Name() string
}

// DefaultChunkingOptions возвращает опции по умолчанию
func DefaultChunkingOptions() ChunkingOptions {
	return ChunkingOptions{
		MaxTokens:     512,
		Overlap:       50, // 10% overlap
		Strategy:      "semantic",
		PreserveLines: true,
		Metadata:      make(map[string]any),
	}
}

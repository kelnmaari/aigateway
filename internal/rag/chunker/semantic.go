// Package chunker provides text chunking implementations.
package chunker

import (
	"context"
	"maps"
	"strings"
	"unicode"
)

// SemanticChunker разбивает текст на chunks учитывая семантику
type SemanticChunker struct {
	// Средняя длина слова для оценки токенов (1 token ≈ 0.75 words)
	avgWordLength float64
}

// NewSemanticChunker создает новый semantic chunker
func NewSemanticChunker() *SemanticChunker {
	return &SemanticChunker{
		avgWordLength: 4.5, // Средняя длина слова в английском/русском
	}
}

// Name возвращает имя chunker'а
func (c *SemanticChunker) Name() string {
	return "semantic"
}

// Chunk разбивает текст на chunks учитывая семантические границы
func (c *SemanticChunker) Chunk(ctx context.Context, text string, options ChunkingOptions) ([]Chunk, error) {
	if text == "" {
		return []Chunk{}, nil
	}

	// Split текст на параграфы (двойной перенос строки)
	paragraphs := c.splitIntoParagraphs(text)

	var chunks []Chunk
	var currentChunk strings.Builder
	currentStartOffset := 0
	currentIndex := 0

	for _, para := range paragraphs {
		paraTokens := c.EstimateTokens(para)
		currentTokens := c.EstimateTokens(currentChunk.String())

		// Если параграф сам по себе больше max tokens, разбиваем его на предложения
		if paraTokens > options.MaxTokens {
			// Сохраняем текущий chunk если есть
			if currentChunk.Len() > 0 {
				chunks = append(chunks, c.createChunk(
					currentChunk.String(),
					currentIndex,
					currentStartOffset,
					currentStartOffset+currentChunk.Len(),
					options.Metadata,
				))
				currentIndex++
				currentChunk.Reset()
			}

			// Разбиваем большой параграф на предложения
			sentences := c.splitIntoSentences(para)
			sentenceChunks := c.chunkSentences(sentences, options, &currentIndex, currentStartOffset)
			chunks = append(chunks, sentenceChunks...)
			currentStartOffset += len(para) + 1
			continue
		}

		// Если добавление параграфа превысит max tokens, сохраняем текущий chunk
		if currentTokens+paraTokens > options.MaxTokens && currentChunk.Len() > 0 {
			chunkText := currentChunk.String()
			chunks = append(chunks, c.createChunk(
				chunkText,
				currentIndex,
				currentStartOffset,
				currentStartOffset+len(chunkText),
				options.Metadata,
			))
			currentIndex++

			// Добавляем overlap из предыдущего chunk
			if options.Overlap > 0 {
				overlapText := c.getOverlapText(chunkText, options.Overlap)
				currentChunk.Reset()
				currentChunk.WriteString(overlapText)
				currentStartOffset = currentStartOffset + len(chunkText) - len(overlapText)
			} else {
				currentChunk.Reset()
				currentStartOffset += len(chunkText)
			}
		}

		// Добавляем параграф к текущему chunk
		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(para)
	}

	// Сохраняем последний chunk
	if currentChunk.Len() > 0 {
		chunkText := currentChunk.String()
		chunks = append(chunks, c.createChunk(
			chunkText,
			currentIndex,
			currentStartOffset,
			currentStartOffset+len(chunkText),
			options.Metadata,
		))
	}

	return chunks, nil
}

// EstimateTokens оценивает количество токенов в тексте
// Используем эвристику: 1 token ≈ 0.75 words ≈ 4 characters
func (c *SemanticChunker) EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	// Подсчитываем слова
	words := strings.Fields(text)
	wordCount := len(words)

	// 1 token ≈ 0.75 words
	estimatedTokens := int(float64(wordCount) / 0.75)

	return estimatedTokens
}

// splitIntoParagraphs разбивает текст на параграфы
func (c *SemanticChunker) splitIntoParagraphs(text string) []string {
	// Нормализуем переносы строк
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Разбиваем на параграфы (двойной перенос строки)
	paragraphs := strings.Split(text, "\n\n")

	// Удаляем пустые параграфы и trim пробелы
	var result []string
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para != "" {
			result = append(result, para)
		}
	}

	return result
}

// splitIntoSentences разбивает текст на предложения
func (c *SemanticChunker) splitIntoSentences(text string) []string {
	// Простое разбиение по точке, вопросительному и восклицательному знаку
	// В production можно использовать более sophisticated NLP

	var sentences []string
	var currentSentence strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		currentSentence.WriteRune(r)

		// Проверяем конец предложения
		if r == '.' || r == '?' || r == '!' || r == '。' || r == '？' || r == '！' {
			// Проверяем что это не сокращение (след. символ не заглавная буква)
			if i+1 < len(runes) {
				nextRune := runes[i+1]
				if unicode.IsSpace(nextRune) {
					// Пропускаем пробелы
					for i+1 < len(runes) && unicode.IsSpace(runes[i+1]) {
						i++
						currentSentence.WriteRune(runes[i])
					}

					// Если след. символ - заглавная буква, это новое предложение
					if i+1 < len(runes) && unicode.IsUpper(runes[i+1]) {
						sentence := strings.TrimSpace(currentSentence.String())
						if sentence != "" {
							sentences = append(sentences, sentence)
						}
						currentSentence.Reset()
					}
				}
			} else {
				// Конец текста
				sentence := strings.TrimSpace(currentSentence.String())
				if sentence != "" {
					sentences = append(sentences, sentence)
				}
				currentSentence.Reset()
			}
		}
	}

	// Добавляем остаток если есть
	if currentSentence.Len() > 0 {
		sentence := strings.TrimSpace(currentSentence.String())
		if sentence != "" {
			sentences = append(sentences, sentence)
		}
	}

	return sentences
}

// chunkSentences группирует предложения в chunks
func (c *SemanticChunker) chunkSentences(sentences []string, options ChunkingOptions, currentIndex *int, startOffset int) []Chunk {
	var chunks []Chunk
	var currentChunk strings.Builder
	chunkStartOffset := startOffset

	for _, sentence := range sentences {
		sentenceTokens := c.EstimateTokens(sentence)
		currentTokens := c.EstimateTokens(currentChunk.String())

		// Если предложение слишком длинное, режем по словам
		if sentenceTokens > options.MaxTokens {
			// Сохраняем текущий chunk
			if currentChunk.Len() > 0 {
				chunkText := currentChunk.String()
				chunks = append(chunks, c.createChunk(
					chunkText,
					*currentIndex,
					chunkStartOffset,
					chunkStartOffset+len(chunkText),
					options.Metadata,
				))
				*currentIndex++
				currentChunk.Reset()
				chunkStartOffset += len(chunkText)
			}

			// Разбиваем длинное предложение по словам
			words := strings.FieldsSeq(sentence)
			for word := range words {
				if c.EstimateTokens(currentChunk.String()+" "+word) > options.MaxTokens {
					chunkText := currentChunk.String()
					chunks = append(chunks, c.createChunk(
						chunkText,
						*currentIndex,
						chunkStartOffset,
						chunkStartOffset+len(chunkText),
						options.Metadata,
					))
					*currentIndex++
					currentChunk.Reset()
					chunkStartOffset += len(chunkText)
				}
				if currentChunk.Len() > 0 {
					currentChunk.WriteString(" ")
				}
				currentChunk.WriteString(word)
			}
			continue
		}

		// Если добавление предложения превысит max tokens
		if currentTokens+sentenceTokens > options.MaxTokens && currentChunk.Len() > 0 {
			chunkText := currentChunk.String()
			chunks = append(chunks, c.createChunk(
				chunkText,
				*currentIndex,
				chunkStartOffset,
				chunkStartOffset+len(chunkText),
				options.Metadata,
			))
			*currentIndex++

			// Overlap
			if options.Overlap > 0 {
				overlapText := c.getOverlapText(chunkText, options.Overlap)
				currentChunk.Reset()
				currentChunk.WriteString(overlapText)
				chunkStartOffset = chunkStartOffset + len(chunkText) - len(overlapText)
			} else {
				currentChunk.Reset()
				chunkStartOffset += len(chunkText)
			}
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(sentence)
	}

	// Последний chunk
	if currentChunk.Len() > 0 {
		chunkText := currentChunk.String()
		chunks = append(chunks, c.createChunk(
			chunkText,
			*currentIndex,
			chunkStartOffset,
			chunkStartOffset+len(chunkText),
			options.Metadata,
		))
		*currentIndex++
	}

	return chunks
}

// getOverlapText возвращает последние N токенов из текста для overlap
func (c *SemanticChunker) getOverlapText(text string, overlapTokens int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	// Приблизительно overlapTokens ≈ overlapWords * 0.75
	overlapWords := int(float64(overlapTokens) * 0.75)
	if overlapWords >= len(words) {
		return text
	}

	// Берем последние overlapWords слов
	startIndex := len(words) - overlapWords
	return strings.Join(words[startIndex:], " ")
}

// createChunk создает Chunk объект
func (c *SemanticChunker) createChunk(text string, index, startOffset, endOffset int, metadata map[string]any) Chunk {
	// Копируем metadata
	chunkMetadata := make(map[string]any)
	maps.Copy(chunkMetadata, metadata)

	return Chunk{
		Text:        text,
		Index:       index,
		Tokens:      c.EstimateTokens(text),
		StartOffset: startOffset,
		EndOffset:   endOffset,
		Metadata:    chunkMetadata,
	}
}

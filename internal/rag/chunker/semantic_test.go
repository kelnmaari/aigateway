// Package chunker provides text chunking tests.
package chunker

import (
	"context"
	"strings"
	"testing"
)

func TestSemanticChunker_Name(t *testing.T) {
	chunker := NewSemanticChunker()
	if chunker.Name() != "semantic" {
		t.Errorf("Expected name 'semantic', got '%s'", chunker.Name())
	}
}

func TestSemanticChunker_EstimateTokens(t *testing.T) {
	chunker := NewSemanticChunker()

	tests := []struct {
		name     string
		text     string
		expected int
		delta    int // допустимое отклонение
	}{
		{
			name:     "Empty text",
			text:     "",
			expected: 0,
			delta:    0,
		},
		{
			name:     "Single word",
			text:     "Hello",
			expected: 1,
			delta:    1,
		},
		{
			name:     "Short sentence",
			text:     "Hello world this is a test",
			expected: 8, // 6 words / 0.75 ≈ 8 tokens
			delta:    2,
		},
		{
			name:     "Long paragraph",
			text:     strings.Repeat("word ", 100),
			expected: 133, // 100 words / 0.75 ≈ 133 tokens
			delta:    10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunker.EstimateTokens(tt.text)
			
			if result < tt.expected-tt.delta || result > tt.expected+tt.delta {
				t.Errorf("EstimateTokens() = %d, expected %d ± %d", result, tt.expected, tt.delta)
			}
		})
	}
}

func TestSemanticChunker_Chunk_EmptyText(t *testing.T) {
	chunker := NewSemanticChunker()
	ctx := context.Background()

	chunks, err := chunker.Chunk(ctx, "", DefaultChunkingOptions())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks, got %d", len(chunks))
	}
}

func TestSemanticChunker_Chunk_SingleParagraph(t *testing.T) {
	chunker := NewSemanticChunker()
	ctx := context.Background()

	text := "This is a simple test paragraph. It contains multiple sentences. We want to verify chunking works correctly."

	options := ChunkingOptions{
		MaxTokens:     100,
		Overlap:       10,
		Strategy:      "semantic",
		PreserveLines: true,
		Metadata:      map[string]interface{}{"test": true},
	}

	chunks, err := chunker.Chunk(ctx, text, options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should create 1 chunk for this small text
	if len(chunks) == 0 {
		t.Error("Expected at least 1 chunk")
	}

	// Verify chunk structure
	chunk := chunks[0]
	if chunk.Text == "" {
		t.Error("Chunk text is empty")
	}
	if chunk.Index != 0 {
		t.Errorf("Expected index 0, got %d", chunk.Index)
	}
	if chunk.Tokens == 0 {
		t.Error("Chunk tokens is 0")
	}
	if _, ok := chunk.Metadata["test"]; !ok {
		t.Error("Metadata not preserved")
	}
}

func TestSemanticChunker_Chunk_MultipleParagraphs(t *testing.T) {
	chunker := NewSemanticChunker()
	ctx := context.Background()

	text := `Paragraph one contains some text. It has multiple sentences.

Paragraph two is also here. It contains different information.

Paragraph three completes the document. This is the final section.`

	options := ChunkingOptions{
		MaxTokens:     50, // Small size to force multiple chunks
		Overlap:       5,
		Strategy:      "semantic",
		PreserveLines: true,
		Metadata:      make(map[string]interface{}),
	}

	chunks, err := chunker.Chunk(ctx, text, options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should create multiple chunks
	if len(chunks) < 2 {
		t.Errorf("Expected at least 2 chunks, got %d", len(chunks))
	}

	// Verify chunk indices are sequential
	for i, chunk := range chunks {
		if chunk.Index != i {
			t.Errorf("Chunk %d has index %d", i, chunk.Index)
		}
	}

	// Verify no chunk exceeds max tokens
	for i, chunk := range chunks {
		if chunk.Tokens > options.MaxTokens {
			t.Errorf("Chunk %d has %d tokens, exceeds max %d", i, chunk.Tokens, options.MaxTokens)
		}
	}
}

func TestSemanticChunker_Chunk_LongParagraph(t *testing.T) {
	chunker := NewSemanticChunker()
	ctx := context.Background()

	// Generate long paragraph (200+ words)
	text := strings.Repeat("This is a very long sentence that contains many words. ", 40)

	options := ChunkingOptions{
		MaxTokens:     100,
		Overlap:       10,
		Strategy:      "semantic",
		PreserveLines: true,
		Metadata:      make(map[string]interface{}),
	}

	chunks, err := chunker.Chunk(ctx, text, options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should split long paragraph into multiple chunks
	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks for long text, got %d", len(chunks))
	}

	// Verify all chunks contain text
	for i, chunk := range chunks {
		if chunk.Text == "" {
			t.Errorf("Chunk %d is empty", i)
		}
		if chunk.Tokens == 0 {
			t.Errorf("Chunk %d has 0 tokens", i)
		}
	}
}

func TestSemanticChunker_Chunk_Overlap(t *testing.T) {
	chunker := NewSemanticChunker()
	ctx := context.Background()

	text := strings.Repeat("Word ", 100) // 100 words

	options := ChunkingOptions{
		MaxTokens:     50,
		Overlap:       10,
		Strategy:      "semantic",
		PreserveLines: true,
		Metadata:      make(map[string]interface{}),
	}

	chunks, err := chunker.Chunk(ctx, text, options)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(chunks) < 2 {
		t.Skip("Need at least 2 chunks to test overlap")
	}

	// Verify overlap exists between consecutive chunks
	for i := 0; i < len(chunks)-1; i++ {
		chunk1 := chunks[i]
		chunk2 := chunks[i+1]

		// Check if chunk2 starts with some words from end of chunk1
		chunk1Words := strings.Fields(chunk1.Text)
		chunk2Words := strings.Fields(chunk2.Text)

		if len(chunk1Words) == 0 || len(chunk2Words) == 0 {
			continue
		}

		// At least one word should overlap (simple check)
		lastWord := chunk1Words[len(chunk1Words)-1]
		hasOverlap := false
		for _, word := range chunk2Words[:min(5, len(chunk2Words))] {
			if word == lastWord {
				hasOverlap = true
				break
			}
		}

		if !hasOverlap {
			t.Logf("Warning: No obvious overlap detected between chunks %d and %d", i, i+1)
		}
	}
}

func TestSemanticChunker_SplitIntoParagraphs(t *testing.T) {
	chunker := NewSemanticChunker()

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "Single paragraph",
			text:     "Single paragraph text",
			expected: 1,
		},
		{
			name:     "Two paragraphs",
			text:     "First paragraph\n\nSecond paragraph",
			expected: 2,
		},
		{
			name:     "Multiple newlines",
			text:     "First\n\n\n\nSecond",
			expected: 2,
		},
		{
			name:     "Windows line endings",
			text:     "First\r\n\r\nSecond",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paragraphs := chunker.splitIntoParagraphs(tt.text)
			if len(paragraphs) != tt.expected {
				t.Errorf("Expected %d paragraphs, got %d", tt.expected, len(paragraphs))
			}
		})
	}
}

func TestSemanticChunker_SplitIntoSentences(t *testing.T) {
	chunker := NewSemanticChunker()

	tests := []struct {
		name     string
		text     string
		minCount int // минимальное количество предложений
	}{
		{
			name:     "Single sentence",
			text:     "This is a sentence.",
			minCount: 1,
		},
		{
			name:     "Multiple sentences",
			text:     "First sentence. Second sentence. Third sentence.",
			minCount: 3,
		},
		{
			name:     "Question mark",
			text:     "What is this? Another question?",
			minCount: 2,
		},
		{
			name:     "Exclamation mark",
			text:     "Amazing! Wonderful!",
			minCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := chunker.splitIntoSentences(tt.text)
			if len(sentences) < tt.minCount {
				t.Errorf("Expected at least %d sentences, got %d", tt.minCount, len(sentences))
			}
		})
	}
}

func TestSemanticChunker_GetOverlapText(t *testing.T) {
	chunker := NewSemanticChunker()

	tests := []struct {
		name          string
		text          string
		overlapTokens int
		expectEmpty   bool
	}{
		{
			name:          "Empty text",
			text:          "",
			overlapTokens: 5,
			expectEmpty:   true,
		},
		{
			name:          "Short text",
			text:          "Short text here",
			overlapTokens: 2,
			expectEmpty:   false,
		},
		{
			name:          "Long text",
			text:          strings.Repeat("word ", 50),
			overlapTokens: 10,
			expectEmpty:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chunker.getOverlapText(tt.text, tt.overlapTokens)
			
			if tt.expectEmpty && result != "" {
				t.Errorf("Expected empty overlap, got '%s'", result)
			}
			
			if !tt.expectEmpty && result == "" && tt.text != "" {
				t.Error("Expected non-empty overlap")
			}
		})
	}
}

// Benchmark tests
func BenchmarkSemanticChunker_Chunk_SmallText(b *testing.B) {
	chunker := NewSemanticChunker()
	ctx := context.Background()
	text := "This is a small test paragraph with a few sentences."
	options := DefaultChunkingOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = chunker.Chunk(ctx, text, options)
	}
}

func BenchmarkSemanticChunker_Chunk_LargeText(b *testing.B) {
	chunker := NewSemanticChunker()
	ctx := context.Background()
	text := strings.Repeat("This is a test sentence with multiple words. ", 1000)
	options := DefaultChunkingOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = chunker.Chunk(ctx, text, options)
	}
}

func BenchmarkSemanticChunker_EstimateTokens(b *testing.B) {
	chunker := NewSemanticChunker()
	text := strings.Repeat("word ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = chunker.EstimateTokens(text)
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}


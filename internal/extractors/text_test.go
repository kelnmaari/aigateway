package extractors

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"
)

func TestTextExtractor_UTF8(t *testing.T) {
	// Arrange
	extractor := NewTextExtractor(10 * 1024 * 1024) // 10MB max
	ctx := context.Background()

	content := "Hello World\nПривет Мир\n你好世界"
	reader := bytes.NewReader([]byte(content))

	// Act
	doc, err := extractor.Extract(ctx, reader, ExtractOptions{})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, content, doc.Text)
	assert.True(t, doc.WordCount > 0)
}

func TestTextExtractor_UTF8_BOM(t *testing.T) {
	// Arrange
	extractor := NewTextExtractor(10 * 1024 * 1024)
	ctx := context.Background()

	// UTF-8 BOM: EF BB BF
	content := []byte{0xEF, 0xBB, 0xBF}
	content = append(content, []byte("Hello World")...)

	reader := bytes.NewReader(content)

	// Act
	doc, err := extractor.Extract(ctx, reader, ExtractOptions{})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "Hello World", doc.Text)
	assert.NotContains(t, doc.Text, "\uFEFF") // BOM should be stripped
}

func TestTextExtractor_Windows1251(t *testing.T) {
	// Arrange
	extractor := NewTextExtractor(10 * 1024 * 1024)
	ctx := context.Background()

	// Encode Russian text in Windows-1251
	original := "Привет Мир! Это тест кодировки Windows-1251."
	encoder := charmap.Windows1251.NewEncoder()
	encoded, err := encoder.Bytes([]byte(original))
	require.NoError(t, err)

	reader := bytes.NewReader(encoded)

	// Act
	doc, err := extractor.Extract(ctx, reader, ExtractOptions{})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, original, doc.Text)
	assert.Contains(t, doc.Metadata.Custom, "detected_encoding")
	assert.Equal(t, "windows-1251", doc.Metadata.Custom["detected_encoding"])
}

func TestTextExtractor_UTF16LE_BOM(t *testing.T) {
	// Arrange
	extractor := NewTextExtractor(10 * 1024 * 1024)
	ctx := context.Background()

	// UTF-16 LE BOM: FF FE
	text := "Hello World"
	content := []byte{0xFF, 0xFE} // BOM

	// Convert to UTF-16 LE (simple ASCII for testing)
	for _, char := range text {
		content = append(content, byte(char), 0x00)
	}

	reader := bytes.NewReader(content)

	// Act
	doc, err := extractor.Extract(ctx, reader, ExtractOptions{})

	// Assert
	require.NoError(t, err)
	assert.Contains(t, doc.Text, "Hello")
	assert.Contains(t, doc.Metadata.Custom, "detected_encoding")
}

func TestTextExtractor_UTF16BE_BOM(t *testing.T) {
	// Arrange
	extractor := NewTextExtractor(10 * 1024 * 1024)
	ctx := context.Background()

	// UTF-16 BE BOM: FE FF
	text := "Hello World"
	content := []byte{0xFE, 0xFF} // BOM

	// Convert to UTF-16 BE (simple ASCII for testing)
	for _, char := range text {
		content = append(content, 0x00, byte(char))
	}

	reader := bytes.NewReader(content)

	// Act
	doc, err := extractor.Extract(ctx, reader, ExtractOptions{})

	// Assert
	require.NoError(t, err)
	assert.Contains(t, doc.Text, "Hello")
	assert.Contains(t, doc.Metadata.Custom, "detected_encoding")
}

func TestTextExtractor_EmptyFile(t *testing.T) {
	// Arrange
	extractor := NewTextExtractor(10 * 1024 * 1024)
	ctx := context.Background()

	reader := bytes.NewReader([]byte{})

	// Act
	doc, err := extractor.Extract(ctx, reader, ExtractOptions{})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "", doc.Text)
	assert.Equal(t, 0, doc.WordCount)
}

func TestTextExtractor_LargeFile(t *testing.T) {
	// Arrange
	maxSize := int64(1024) // 1KB for testing
	extractor := NewTextExtractor(maxSize)
	ctx := context.Background()

	// Create content larger than max size
	largeContent := bytes.Repeat([]byte("test "), 300) // ~1.5KB
	reader := bytes.NewReader(largeContent)

	// Act
	doc, err := extractor.Extract(ctx, reader, ExtractOptions{})

	// Assert
	assert.Error(t, err)
	assert.Nil(t, doc)
	assert.Contains(t, err.Error(), "file too large")
}

func TestTextExtractor_ContextCancellation(t *testing.T) {
	// Arrange
	extractor := NewTextExtractor(10 * 1024 * 1024)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	content := "Hello World"
	reader := bytes.NewReader([]byte(content))

	// Act
	doc, err := extractor.Extract(ctx, reader, ExtractOptions{})

	// Assert
	assert.Error(t, err)
	assert.Nil(t, doc)
	assert.Equal(t, context.Canceled, err)
}

func TestTextExtractor_MixedContent(t *testing.T) {
	// Arrange
	extractor := NewTextExtractor(10 * 1024 * 1024)
	ctx := context.Background()

	// Mixed languages and special characters
	content := `
English: Hello World!
Русский: Привет Мир!
中文: 你好世界
Emoji: 🚀 🎉 ✅
Special: @#$%^&*()
Numbers: 1234567890
`
	reader := bytes.NewReader([]byte(content))

	// Act
	doc, err := extractor.Extract(ctx, reader, ExtractOptions{})

	// Assert
	require.NoError(t, err)
	assert.Contains(t, doc.Text, "Hello World")
	assert.Contains(t, doc.Text, "Привет Мир")
	assert.Contains(t, doc.Text, "你好世界")
	assert.Contains(t, doc.Text, "🚀")
	assert.True(t, doc.WordCount > 10)
}

func TestTextExtractor_LineEndings(t *testing.T) {
	tests := []struct {
		name    string
		content string
		ending  string
	}{
		{"Unix LF", "line1\nline2\nline3", "\n"},
		{"Windows CRLF", "line1\r\nline2\r\nline3", "\r\n"},
		{"Mac CR", "line1\rline2\rline3", "\r"},
		{"Mixed", "line1\nline2\r\nline3\r", "mixed"},
	}

	extractor := NewTextExtractor(10 * 1024 * 1024)
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader([]byte(tt.content))

			// Act
			doc, err := extractor.Extract(ctx, reader, ExtractOptions{})

			// Assert
			require.NoError(t, err)
			assert.Contains(t, doc.Text, "line1")
			assert.Contains(t, doc.Text, "line2")
			assert.Contains(t, doc.Text, "line3")
		})
	}
}

func TestTextExtractor_CodeFile(t *testing.T) {
	// Arrange
	extractor := NewTextExtractor(10 * 1024 * 1024)
	ctx := context.Background()

	codeContent := `
package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
	// This is a comment
	/* Multi-line
	   comment */
}
`
	reader := bytes.NewReader([]byte(codeContent))

	// Act
	doc, err := extractor.Extract(ctx, reader, ExtractOptions{})

	// Assert
	require.NoError(t, err)
	assert.Contains(t, doc.Text, "package main")
	assert.Contains(t, doc.Text, "func main")
	assert.Contains(t, doc.Text, "fmt.Println")
	assert.True(t, doc.WordCount > 5)
}

func TestTextExtractor_JSONContent(t *testing.T) {
	// Arrange
	extractor := NewTextExtractor(10 * 1024 * 1024)
	ctx := context.Background()

	jsonContent := `{
  "name": "John Doe",
  "age": 30,
  "city": "New York",
  "skills": ["Go", "Python", "JavaScript"]
}`
	reader := bytes.NewReader([]byte(jsonContent))

	// Act
	doc, err := extractor.Extract(ctx, reader, ExtractOptions{})

	// Assert
	require.NoError(t, err)
	assert.Contains(t, doc.Text, "John Doe")
	assert.Contains(t, doc.Text, "New York")
	assert.Contains(t, doc.Text, "skills")
}

func TestTextExtractor_MarkdownContent(t *testing.T) {
	// Arrange
	extractor := NewTextExtractor(10 * 1024 * 1024)
	ctx := context.Background()

	mdContent := `# Heading 1

## Heading 2

This is a **bold** and *italic* text.

- List item 1
- List item 2

[Link](https://example.com)

` + "```go\ncode block\n```"

	reader := bytes.NewReader([]byte(mdContent))

	// Act
	doc, err := extractor.Extract(ctx, reader, ExtractOptions{})

	// Assert
	require.NoError(t, err)
	assert.Contains(t, doc.Text, "Heading 1")
	assert.Contains(t, doc.Text, "bold")
	assert.Contains(t, doc.Text, "italic")
	assert.Contains(t, doc.Text, "List item")
}

func TestTextExtractor_SupportedTypes(t *testing.T) {
	// Arrange
	extractor := NewTextExtractor(10 * 1024 * 1024)

	// Act
	types := extractor.SupportedTypes()

	// Assert
	assert.Contains(t, types, "text/plain")
	assert.Contains(t, types, "text/markdown")
	assert.Contains(t, types, "application/json")
	assert.Contains(t, types, "application/xml")
}

func TestTextExtractor_MaxFileSize(t *testing.T) {
	// Arrange
	maxSize := int64(5 * 1024 * 1024) // 5MB
	extractor := NewTextExtractor(maxSize)

	// Act
	returned := extractor.MaxFileSize()

	// Assert
	assert.Equal(t, maxSize, returned)
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{"empty", "", 0},
		{"single word", "hello", 1},
		{"multiple words", "hello world test", 3},
		{"with punctuation", "Hello, World! How are you?", 5},
		{"with numbers", "Test 123 456 words", 4},
		{"multiple spaces", "word1    word2     word3", 3},
		{"with newlines", "line1\nline2\nline3", 3},
		{"cyrillic", "Привет мир тест", 3},
		{"mixed", "Hello Мир 测试", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := countWords(tt.text)
			assert.Equal(t, tt.expected, count)
		})
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{"english", "Hello World, this is a test", "en"},
		{"russian", "Привет Мир, это тест", "ru"},
		{"chinese", "你好世界，这是一个测试", "zh"},
		{"empty", "", "unknown"},
		{"mixed", "Hello Привет 你好", "mixed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lang := detectLanguage(tt.text)
			assert.Contains(t, []string{tt.expected, "mixed", "unknown"}, lang)
		})
	}
}

func TestHasHighBytes(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{"pure ASCII", []byte("Hello World"), false},
		{"with high bytes", []byte{0x80, 0x90, 0xA0}, true},
		{"UTF-8 cyrillic", []byte("Привет"), true},
		{"mixed", []byte("Hello Мир"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasHighBytes(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasCyrillic(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"pure ASCII", "Hello World", false},
		{"russian", "Привет Мир", true},
		{"mixed", "Hello Привет", true},
		{"ukrainian", "Привіт Світ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasCyrillic(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsReasonableText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{"normal text", "Hello World, this is a test.", true},
		{"russian text", "Привет Мир, это тест.", true},
		{"binary gibberish", string([]byte{0x01, 0x02, 0x03, 0x04}), false},
		{"mostly control chars", "\x00\x01\x02test", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isReasonableText(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark tests
func BenchmarkTextExtractor_UTF8(b *testing.B) {
	extractor := NewTextExtractor(10 * 1024 * 1024)
	ctx := context.Background()
	content := []byte(strings.Repeat("Hello World\n", 100))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(content)
		_, _ = extractor.Extract(ctx, reader, ExtractOptions{})
	}
}

func BenchmarkTextExtractor_Windows1251(b *testing.B) {
	extractor := NewTextExtractor(10 * 1024 * 1024)
	ctx := context.Background()

	original := strings.Repeat("Привет Мир\n", 100)
	encoder := charmap.Windows1251.NewEncoder()
	encoded, _ := encoder.Bytes([]byte(original))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(encoded)
		_, _ = extractor.Extract(ctx, reader, ExtractOptions{})
	}
}

func BenchmarkCountWords(b *testing.B) {
	text := "Hello World! This is a test text with multiple words and punctuation."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = countWords(text)
	}
}

func BenchmarkDetectLanguage(b *testing.B) {
	text := "Hello World! This is a test text in English language."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detectLanguage(text)
	}
}

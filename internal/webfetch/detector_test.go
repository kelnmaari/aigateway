package webfetch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURLDetector_DetectURLs(t *testing.T) {
	detector := NewURLDetector()

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "single https URL",
			text:     "Check out https://example.com for more info",
			expected: []string{"https://example.com"},
		},
		{
			name:     "single http URL",
			text:     "Visit http://example.com/page",
			expected: []string{"http://example.com/page"},
		},
		{
			name:     "www URL (normalized)",
			text:     "See www.example.com for details",
			expected: []string{"https://www.example.com"},
		},
		{
			name:     "multiple URLs",
			text:     "Check https://example.com and https://test.com for info",
			expected: []string{"https://example.com", "https://test.com"},
		},
		{
			name:     "URL with path and query",
			text:     "API docs at https://api.example.com/v1/docs?version=1.0",
			expected: []string{"https://api.example.com/v1/docs?version=1.0"},
		},
		{
			name:     "URL with trailing punctuation",
			text:     "Visit https://example.com. It's great!",
			expected: []string{"https://example.com"},
		},
		{
			name:     "URL with comma",
			text:     "Sites: https://example.com, https://test.com",
			expected: []string{"https://example.com", "https://test.com"},
		},
		{
			name:     "no URLs",
			text:     "This is just plain text without any links",
			expected: []string{},
		},
		{
			name:     "duplicate URLs",
			text:     "Check https://example.com and https://example.com again",
			expected: []string{"https://example.com"}, // Deduplicated
		},
		{
			name:     "URL in quotes",
			text:     `Check "https://example.com" for info`,
			expected: []string{"https://example.com"},
		},
		{
			name:     "GitHub URL",
			text:     "Source code: https://github.com/user/repo",
			expected: []string{"https://github.com/user/repo"},
		},
		{
			name:     "URL with port",
			text:     "Server at http://localhost:8080/api",
			expected: []string{"http://localhost:8080/api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls := detector.DetectURLs(tt.text)
			assert.Equal(t, tt.expected, urls)
		})
	}
}

func TestURLDetector_HasURLs(t *testing.T) {
	detector := NewURLDetector()

	tests := []struct {
		name     string
		text     string
		expected bool
	}{
		{
			name:     "has URL",
			text:     "Check https://example.com",
			expected: true,
		},
		{
			name:     "no URL",
			text:     "Just plain text",
			expected: false,
		},
		{
			name:     "empty string",
			text:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.HasURLs(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestURLDetector_ExtractFirstURL(t *testing.T) {
	detector := NewURLDetector()

	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "single URL",
			text:     "Check https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "multiple URLs - returns first",
			text:     "Check https://example.com and https://test.com",
			expected: "https://example.com",
		},
		{
			name:     "no URL",
			text:     "Just plain text",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.ExtractFirstURL(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func BenchmarkURLDetector_DetectURLs(b *testing.B) {
	detector := NewURLDetector()
	text := "Check out https://example.com and https://test.com for more information about https://docs.example.com/api"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detector.DetectURLs(text)
	}
}


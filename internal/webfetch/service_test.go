package webfetch

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestURLValidator(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		valid   bool
		errorMsg string
	}{
		{"valid HTTP URL", "http://example.com", true, ""},
		{"valid HTTPS URL", "https://example.com", true, ""},
		{"invalid scheme", "ftp://example.com", false, "scheme ftp not allowed"},
		{"private IP", "http://192.168.0.1", false, "private IP addresses not allowed"},
		{"localhost", "http://localhost", false, "localhost not allowed"},
	}

	config := ValidationConfig{
		AllowedSchemes:   []string{"http", "https"},
		BlockPrivateIPs:  true,
		BlockLocalhost:   true,
	}

	validator := NewURLValidator(config)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.url)
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			}
		})
	}
}

func TestRateLimiter(t *testing.T) {
	config := RateLimitConfig{
		Enabled:               true,
		DefaultRequestsPerMin: 5,
	}

	rateLimiter := NewRateLimiter(config)

	// Test that we can make requests within the limit
	for i := 0; i < 5; i++ {
		err := rateLimiter.Wait(context.Background(), "example.com")
		assert.NoError(t, err)
	}

	// Test that we get rate-limited after exceeding the limit
	err := rateLimiter.Wait(context.Background(), "example.com")
	assert.Error(t, err)

	// Reset and try again
	rateLimiter.Reset("example.com")

	for i := 0; i < 5; i++ {
		err := rateLimiter.Wait(context.Background(), "example.com")
		assert.NoError(t, err)
	}
}

func TestHTMLParser(t *testing.T) {
	config := ParserConfig{
		RemoveSelectors: []string{"script", "style"},
		ContentSelectors: []string{"#content", ".article"},
	}

	parser := NewHTMLParser(config)

	// Create a simple HTML document for testing
	html := `
	<html>
	<head><title>Test Page</title></head>
	<body>
		<div id="content">This is the main content.</div>
		<script>alert('test');</script>
		<style>.hidden { display: none; }</style>
		<div class="sidebar">This should be removed.</div>
	</body>
	</html>`

	parsed, err := parser.Parse(strings.NewReader(html))
	assert.NoError(t, err)
	assert.Equal(t, "Test Page", parsed.Title)
	assert.Contains(t, parsed.Content, "This is the main content.")
	assert.NotContains(t, parsed.Content, "This should be removed.")
	assert.NotContains(t, parsed.Content, "alert('test');")
}

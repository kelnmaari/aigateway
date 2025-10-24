package webfetch

import (
	"regexp"
	"strings"
)

// URLDetector детектирует URLs в тексте
type URLDetector struct {
	urlRegex *regexp.Regexp
}

// NewURLDetector создает новый detector
func NewURLDetector() *URLDetector {
	// URL regex pattern
	// Matches: http://..., https://..., www....
	pattern := `https?://[^\s<>"{}|\\^\[\]` + "`" + `]+|www\.[^\s<>"{}|\\^\[\]` + "`" + `]+`

	return &URLDetector{
		urlRegex: regexp.MustCompile(pattern),
	}
}

// DetectURLs находит все URLs в тексте
func (d *URLDetector) DetectURLs(text string) []string {
	matches := d.urlRegex.FindAllString(text, -1)

	// Deduplicate and normalize
	seen := make(map[string]bool)
	urls := make([]string, 0)

	for _, match := range matches {
		// Normalize: add https:// if starts with www.
		url := match
		if strings.HasPrefix(strings.ToLower(url), "www.") {
			url = "https://" + url
		}

		// Remove trailing punctuation
		url = strings.TrimRight(url, ".,!?;:")

		if !seen[url] {
			seen[url] = true
			urls = append(urls, url)
		}
	}

	return urls
}

// HasURLs проверяет есть ли URLs в тексте
func (d *URLDetector) HasURLs(text string) bool {
	return len(d.DetectURLs(text)) > 0
}

// ExtractFirstURL извлекает первый URL из текста
func (d *URLDetector) ExtractFirstURL(text string) string {
	urls := d.DetectURLs(text)
	if len(urls) > 0 {
		return urls[0]
	}
	return ""
}

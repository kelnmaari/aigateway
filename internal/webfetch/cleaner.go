package webfetch

import "strings"

// cleanText removes extra whitespace from the extracted content
func (p *HTMLParser) cleanText(text string) string {
	lines := strings.Split(text, "\n")
	var cleaned []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}

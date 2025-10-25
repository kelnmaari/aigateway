package webfetch

import (
	"io"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// HTMLParser парсит HTML контент
type HTMLParser struct {
	config ParsingConfig
}

// NewHTMLParser создает новый parser
func NewHTMLParser(config ParsingConfig) *HTMLParser {
	return &HTMLParser{
		config: config,
	}
}

// Parse парсит HTML из reader
func (p *HTMLParser) Parse(html io.Reader) (*ParsedContent, error) {
	doc, err := goquery.NewDocumentFromReader(html)
	if err != nil {
		return nil, err
	}

	// Extract title
	title := doc.Find("title").First().Text()
	title = strings.TrimSpace(title)

	// Remove unwanted elements
	for _, selector := range p.config.RemoveSelectors {
		doc.Find(selector).Remove()
	}

	// Remove scripts, styles, comments
	if p.config.RemoveScripts {
		doc.Find("script").Remove()
	}
	if p.config.RemoveStyles {
		doc.Find("style").Remove()
	}

	// Find main content
	var content string
	for _, selector := range p.config.ContentSelectors {
		if el := doc.Find(selector).First(); el.Length() > 0 {
			content = p.extractText(el)
			break
		}
	}

	// Fallback: use body
	if content == "" {
		content = p.extractText(doc.Find("body"))
	}

	// Clean text
	content = p.cleanText(content)

	// Extract metadata
	metadata := p.extractMetadata(doc)

	// Extract links
	var links []string
	if p.config.ExtractLinks {
		doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
			if href, exists := s.Attr("href"); exists {
				links = append(links, strings.TrimSpace(href))
			}
		})
	}

	// Count words
	wordCount := len(strings.Fields(content))

	// Detect language from metadata or content
	language := ""
	if metadata != nil && metadata.Language != "" {
		language = metadata.Language
	}
	// TODO: Implement content-based language detection using lingua-go or similar

	return &ParsedContent{
		Title:     title,
		Content:   content,
		Metadata:  metadata,
		Links:     links,
		WordCount: wordCount,
		Language:  language,
	}, nil
}

// extractText извлекает текст из selection
func (p *HTMLParser) extractText(sel *goquery.Selection) string {
	return sel.Text()
}

// cleanText очищает текст от лишних пробелов
func (p *HTMLParser) cleanText(text string) string {
	// Split by newlines
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

// extractMetadata извлекает метаданные из HTML
func (p *HTMLParser) extractMetadata(doc *goquery.Document) *PageMetadata {
	meta := &PageMetadata{
		StructuredData: make(map[string]interface{}),
	}

	// Meta tags
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		property, _ := s.Attr("property")
		content, _ := s.Attr("content")

		name = strings.ToLower(strings.TrimSpace(name))
		property = strings.ToLower(strings.TrimSpace(property))

		switch {
		// Standard meta tags
		case name == "description":
			meta.Description = content
		case name == "keywords":
			keywords := strings.Split(content, ",")
			for i, kw := range keywords {
				keywords[i] = strings.TrimSpace(kw)
			}
			meta.Keywords = keywords
		case name == "author":
			meta.Author = content
		case name == "language" || name == "lang":
			meta.Language = content

		// Open Graph
		case property == "og:title":
			meta.OGTitle = content
		case property == "og:description":
			meta.OGDescription = content
		case property == "og:image":
			meta.OGImage = content
		case property == "og:type":
			meta.OGType = content

		// Twitter Card
		case name == "twitter:card":
			meta.TwitterCard = content
		case name == "twitter:title":
			meta.TwitterTitle = content
		case name == "twitter:description":
			meta.TwitterDescription = content

		// Article dates
		case property == "article:published_time" || name == "publish_date":
			if t, err := time.Parse(time.RFC3339, content); err == nil {
				meta.PublishedDate = &t
			}
		case property == "article:modified_time" || name == "modified_date":
			if t, err := time.Parse(time.RFC3339, content); err == nil {
				meta.ModifiedDate = &t
			}
		}
	})

	// HTML lang attribute
	if meta.Language == "" {
		if lang, exists := doc.Find("html").Attr("lang"); exists {
			meta.Language = lang
		}
	}

	// Extract JSON-LD structured data
	doc.Find("script[type='application/ld+json']").Each(func(i int, s *goquery.Selection) {
		jsonLD := s.Text()
		// TODO: Parse JSON-LD если потребуется
		meta.StructuredData["json-ld"] = jsonLD
	})

	return meta
}

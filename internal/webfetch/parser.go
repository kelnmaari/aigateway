package webfetch

import (
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// HTMLParser extracts content and metadata from HTML
type HTMLParser struct {
	config ParserConfig
}

// NewHTMLParser creates a new HTML parser
func NewHTMLParser(config ParserConfig) *HTMLParser {
	return &HTMLParser{
		config: config,
	}
}

// Parse extracts text, metadata, and links from HTML
func (p *HTMLParser) Parse(html io.Reader) (*ParsedContent, error) {
	doc, err := goquery.NewDocumentFromReader(html)
	if err != nil {
		return nil, err
	}

	// Extract title
	title := doc.Find("title").First().Text()

	// Remove unwanted elements
	for _, selector := range p.config.RemoveSelectors {
		doc.Find(selector).Remove()
	}

	// Find main content
	var content string
	for _, selector := range p.config.ContentSelectors {
		if el := doc.Find(selector).First(); el.Length() > 0 {
			content = p.extractText(el)
			break
		}
	}

	// Fallback: use body if no content selectors match
	if content == "" {
		content = p.extractText(doc.Find("body"))
	}

	// Extract metadata
	metadata := &PageMetadata{}
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		property, _ := s.Attr("property")
		content, _ := s.Attr("content")

		switch {
		case name == "description":
			metadata.Description = content
		case name == "keywords":
			metadata.Keywords = strings.Split(content, ",")
		case name == "author":
			metadata.Author = content
		case property == "og:title":
			metadata.OGTitle = content
		case property == "og:description":
			metadata.OGDescription = content
		case property == "og:image":
			metadata.OGImage = content
		case property == "og:type":
			metadata.OGType = content
		}
	})

	// Extract links
	var links []string
	if p.config.ExtractLinks {
		doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
			if href, exists := s.Attr("href"); exists {
				links = append(links, href)
			}
		})
	}

	return &ParsedContent{
		Title:    strings.TrimSpace(title),
		Content:  content,
		Metadata: metadata,
		Links:    links,
	}, nil
}

// extractText removes scripts and styles from a selection and returns the text content
func (p *HTMLParser) extractText(sel *goquery.Selection) string {
	// Remove script and style tags
	sel.Find("script, style").Remove()

	// Get text
	return sel.Text()
}

// ParsedContent represents the parsed content from an HTML document
type ParsedContent struct {
	Title    string
	Content  string
	Metadata *PageMetadata
	Links    []string
}

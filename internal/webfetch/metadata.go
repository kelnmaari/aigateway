package webfetch

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// extractMetadata extracts metadata from HTML meta tags
func (p *HTMLParser) extractMetadata(doc *goquery.Document) *PageMetadata {
	meta := &PageMetadata{}

	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		property, _ := s.Attr("property")
		content, _ := s.Attr("content")

		switch {
		case name == "description":
			meta.Description = content
		case name == "keywords":
			meta.Keywords = strings.Split(content, ",")
		case name == "author":
			meta.Author = content
		case property == "og:title":
			meta.OGTitle = content
		case property == "og:description":
			meta.OGDescription = content
		case property == "og:image":
			meta.OGImage = content
		case property == "og:type":
			meta.OGType = content
		}
	})

	return meta
}

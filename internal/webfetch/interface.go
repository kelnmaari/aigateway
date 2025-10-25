package webfetch

import (
	"context"
	"time"
)

// WebFetcher интерфейс для извлечения веб-контента
type WebFetcher interface {
	// Fetch retrieves and parses web page
	Fetch(ctx context.Context, url string, opts FetchOptions) (*WebPage, error)

	// FetchBatch fetches multiple URLs concurrently
	FetchBatch(ctx context.Context, urls []string, opts FetchOptions) ([]*WebPage, error)

	// ValidateURL checks if URL is safe to fetch
	ValidateURL(url string) error
}

// FetchOptions опции для fetching
type FetchOptions struct {
	Timeout         time.Duration
	MaxSize         int64 // Maximum response size
	FollowRedirects bool
	ExtractLinks    bool // Extract all links from page
	Summarize       bool // Generate LLM summary
	CacheEnabled    bool // Use cached result if available
	CacheTTL        time.Duration
	UserAgent       string
}

// WebPage извлеченная веб-страница
type WebPage struct {
	URL         string
	Title       string
	Content     string // Plain text
	HTMLContent string // Raw HTML (optional)
	Metadata    *PageMetadata
	Links       []string // Extracted links
	Summary     string   // LLM summary (optional)
	FetchTimeMS int64
	FetchedAt   time.Time
	ExpiresAt   time.Time
	Cached      bool
	WordCount   int    // Number of words in content
	Language    string // Detected language
}

// PageMetadata метаданные страницы
type PageMetadata struct {
	Description   string
	Keywords      []string
	Author        string
	PublishedDate *time.Time
	ModifiedDate  *time.Time
	Language      string

	// Open Graph
	OGTitle       string
	OGDescription string
	OGImage       string
	OGType        string

	// Twitter Card
	TwitterCard        string
	TwitterTitle       string
	TwitterDescription string

	// JSON-LD structured data
	StructuredData map[string]interface{}
}

// ParsedContent результат парсинга HTML
type ParsedContent struct {
	Title     string
	Content   string
	Metadata  *PageMetadata
	Links     []string
	WordCount int
	Language  string // Detected language code (e.g., "en", "ru")
}

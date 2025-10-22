package webfetch

import (
	"context"
	"time"
)

// WebFetcher interface defines methods for fetching and parsing web pages
type WebFetcher interface {
	// Fetch retrieves and parses a web page
	Fetch(ctx context.Context, url string, opts FetchOptions) (*WebPage, error)

	// FetchBatch fetches multiple URLs concurrently
	FetchBatch(ctx context.Context, urls []string, opts FetchOptions) ([]*WebPage, error)

	// ValidateURL checks if URL is safe to fetch
	ValidateURL(url string) error
}

// FetchOptions contains configuration for the web page fetch operation
type FetchOptions struct {
	Timeout         time.Duration
	MaxSize         int64   // Maximum response size in bytes
	FollowRedirects bool
	ExtractLinks    bool    // Extract all links from page
	Summarize       bool    // Generate LLM summary
	CacheEnabled    bool    // Use cached result if available
	CacheTTL        time.Duration
}

// WebPage represents the parsed content of a web page
type WebPage struct {
	URL             string
	Title           string
	Content         string                 // Plain text
	HTMLContent     string                 // Raw HTML (optional)
	Metadata        *PageMetadata          // Extracted metadata
	Links           []string               // Extracted links
	Summary         string                 // LLM summary (optional)
	FetchTimeMS     int64
	FetchedAt       time.Time
	ExpiresAt       time.Time
}

// PageMetadata contains metadata extracted from the web page
type PageMetadata struct {
	Description     string
	Keywords        []string
	Author          string
	PublishedDate   time.Time
	ModifiedDate    time.Time
	Language        string

	// Open Graph
	OGTitle         string
	OGDescription   string
	OGImage         string
	OGType          string

	// Twitter Card
	TwitterCard     string
	TwitterTitle    string
	TwitterDescription string

	// JSON-LD structured data
	StructuredData  map[string]interface{}
}

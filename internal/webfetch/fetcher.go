package webfetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Fetcher implements the WebFetcher interface for fetching web pages
type Fetcher struct {
	client     *http.Client
	validator  *URLValidator
	parser     *HTMLParser
	rateLimiter *RateLimiter
}

// NewFetcher creates a new fetcher with the given configuration
func NewFetcher(config Config) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: config.HTTP.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= config.HTTP.MaxRedirects {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		validator:   NewURLValidator(config.Security),
		parser:      NewHTMLParser(config.Parsing),
		rateLimiter: NewRateLimiter(config.RateLimit),
	}
}

// Fetch retrieves and parses a web page
func (f *Fetcher) Fetch(ctx context.Context, rawURL string, opts FetchOptions) (*WebPage, error) {
	start := time.Now()

	// Validate URL
	if err := f.validator.Validate(rawURL); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check rate limit
	domain := extractDomain(rawURL)
	if err := f.rateLimiter.Wait(ctx, domain); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Make HTTP request
	req, _ := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	req.Header.Set("User-Agent", "OllamaProxy-Bot/1.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Limit response size
	limitedReader := io.LimitReader(resp.Body, opts.MaxSize)

	// Parse HTML
	parsed, err := f.parser.Parse(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("parse failed: %w", err)
	}

	result := &WebPage{
		URL:         rawURL,
		Title:       parsed.Title,
		Content:     parsed.Content,
		Metadata:    parsed.Metadata,
		Links:       parsed.Links,
		FetchTimeMS: time.Since(start).Milliseconds(),
		FetchedAt:   time.Now(),
	}

	// Extract HTML content if needed
	if opts.ExtractLinks {
		doc, _ := goquery.NewDocumentFromReader(resp.Body)
		for _, selector := range f.parser.config.ContentSelectors {
			if el := doc.Find(selector).First(); el.Length() > 0 {
				result.HTMLContent = el.Html()
				break
			}
		}
	}

	return result, nil
}

// FetchBatch fetches multiple URLs concurrently
func (f *Fetcher) FetchBatch(ctx context.Context, urls []string, opts FetchOptions) ([]*WebPage, error) {
	type result struct {
		page *WebPage
		err  error
	}

	resultCh := make(chan result, len(urls))
	sem := make(chan struct{}, 5) // Limit concurrency

	for _, url := range urls {
		url := url // Capture range variable
		go func() {
			sem <- struct{}{}
			page, err := f.Fetch(ctx, url, opts)
			resultCh <- result{page: page, err: err}
			<-sem
		}()
	}

	pages := make([]*WebPage, 0, len(urls))
	for i := 0; i < len(urls); i++ {
		res := <-resultCh
		if res.err != nil {
			return nil, res.err
		}
		pages = append(pages, res.page)
	}

	return pages, nil
}

// extractDomain extracts the domain from a URL string
func extractDomain(url string) string {
	// Simple extraction for rate limiting purposes
	parts := strings.Split(url, "/")
	if len(parts) > 2 {
		return parts[2]
	}
	return url
}

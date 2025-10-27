package webfetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Fetcher выполняет HTTP запросы с защитой от SSRF
type Fetcher struct {
	client      *http.Client
	validator   *URLValidator
	rateLimiter *RateLimiter
	parser      *HTMLParser
	config      Config
	logger      *logrus.Logger
}

// NewFetcher создает новый fetcher
func NewFetcher(config Config, logger *logrus.Logger) *Fetcher {
	client := &http.Client{
		Timeout: config.HTTP.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !config.HTTP.FollowRedirects {
				return http.ErrUseLastResponse
			}
			if len(via) >= config.HTTP.MaxRedirects {
				return fmt.Errorf("too many redirects: %d", len(via))
			}
			return nil
		},
	}

	return &Fetcher{
		client:      client,
		validator:   NewURLValidator(config.Security),
		rateLimiter: NewRateLimiter(config.RateLimit),
		parser:      NewHTMLParser(config.Parsing),
		config:      config,
		logger:      logger,
	}
}

// Fetch извлекает веб-страницу
func (f *Fetcher) Fetch(ctx context.Context, rawURL string, opts FetchOptions) (*WebPage, error) {
	start := time.Now()

	// Validate URL
	if err := f.validator.Validate(rawURL); err != nil {
		f.logger.WithFields(logrus.Fields{
			"url":   rawURL,
			"error": err,
		}).Warn("URL validation failed")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check rate limit
	if err := f.rateLimiter.Wait(ctx, rawURL); err != nil {
		f.logger.WithFields(logrus.Fields{
			"url":   rawURL,
			"error": err,
		}).Warn("Rate limit exceeded")
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Fetch with retries
	resp, err := f.fetchWithRetry(ctx, rawURL, opts)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		f.logger.WithFields(logrus.Fields{
			"url":         rawURL,
			"status_code": resp.StatusCode,
		}).Warn("Non-200 status code")
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !f.isAllowedContentType(contentType) {
		return nil, fmt.Errorf("content type %s not allowed", contentType)
	}

	// Determine max size
	maxSize := opts.MaxSize
	if maxSize == 0 {
		maxSize = f.config.HTTP.MaxResponseSize
	}

	// Limit response size
	limitedReader := io.LimitReader(resp.Body, maxSize+1) // +1 to detect oversized

	// Read body
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	// Check if content was truncated
	if int64(len(body)) > maxSize {
		return nil, fmt.Errorf("response too large: exceeds %d bytes", maxSize)
	}

	fetchTimeMS := time.Since(start).Milliseconds()

	// Parse HTML
	parsed, err := f.parser.Parse(strings.NewReader(string(body)))
	if err != nil {
		f.logger.WithFields(logrus.Fields{
			"url":   rawURL,
			"error": err,
		}).Error("HTML parsing failed")
		return nil, fmt.Errorf("parse HTML failed: %w", err)
	}

	f.logger.WithFields(logrus.Fields{
		"url":           rawURL,
		"status":        resp.StatusCode,
		"content_type":  contentType,
		"size_bytes":    len(body),
		"word_count":    parsed.WordCount,
		"links_found":   len(parsed.Links),
		"fetch_time_ms": fetchTimeMS,
	}).Info("Web page fetched and parsed successfully")

	return &WebPage{
		URL:         rawURL,
		Title:       parsed.Title,
		Content:     parsed.Content,
		HTMLContent: string(body),
		Metadata:    parsed.Metadata,
		Links:       parsed.Links,
		FetchTimeMS: fetchTimeMS,
		FetchedAt:   time.Now(),
		Cached:      false,
		WordCount:   parsed.WordCount,
		Language:    parsed.Language,
	}, nil
}

// FetchBatch извлекает несколько URLs параллельно
func (f *Fetcher) FetchBatch(ctx context.Context, urls []string, opts FetchOptions) ([]*WebPage, error) {
	type result struct {
		page *WebPage
		err  error
		url  string
	}

	results := make(chan result, len(urls))

	// Fetch concurrently
	for _, url := range urls {
		go func(u string) {
			page, err := f.Fetch(ctx, u, opts)
			results <- result{page: page, err: err, url: u}
		}(url)
	}

	// Collect results
	pages := make([]*WebPage, 0, len(urls))
	var errors []error

	for i := 0; i < len(urls); i++ {
		res := <-results
		if res.err != nil {
			f.logger.WithFields(logrus.Fields{
				"url":   res.url,
				"error": res.err,
			}).Error("Batch fetch failed for URL")
			errors = append(errors, fmt.Errorf("%s: %w", res.url, res.err))
		} else {
			pages = append(pages, res.page)
		}
	}

	if len(errors) > 0 {
		return pages, fmt.Errorf("batch fetch completed with %d errors", len(errors))
	}

	return pages, nil
}

// ValidateURL проверяет URL на безопасность
func (f *Fetcher) ValidateURL(rawURL string) error {
	return f.validator.Validate(rawURL)
}

// fetchWithRetry выполняет HTTP запрос с повторами
func (f *Fetcher) fetchWithRetry(ctx context.Context, rawURL string, opts FetchOptions) (*http.Response, error) {
	var lastErr error
	attempts := f.config.RateLimit.RetryAttempts
	if attempts == 0 {
		attempts = 1
	}

	delay := f.config.RateLimit.RetryDelay
	maxDelay := f.config.RateLimit.RetryMaxDelay

	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create request failed: %w", err)
		}

		// Set headers
		f.setHeaders(req, opts)

		// Execute request
		resp, err := f.client.Do(req)
		if err == nil {
			// Success
			return resp, nil
		}

		lastErr = err

		// Log retry
		f.logger.WithFields(logrus.Fields{
			"url":     rawURL,
			"attempt": attempt,
			"error":   err,
		}).Warn("Fetch attempt failed, retrying")

		// Check if should retry
		if attempt < attempts {
			// Exponential backoff
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				delay *= 2
				if delay > maxDelay {
					delay = maxDelay
				}
			}
		}
	}

	return nil, fmt.Errorf("fetch failed after %d attempts: %w", attempts, lastErr)
}

// setHeaders устанавливает HTTP заголовки
func (f *Fetcher) setHeaders(req *http.Request, opts FetchOptions) {
	// User-Agent
	userAgent := opts.UserAgent
	if userAgent == "" {
		userAgent = f.config.HTTP.UserAgent
	}
	req.Header.Set("User-Agent", userAgent)

	// Additional headers from config
	for key, value := range f.config.HTTP.Headers {
		req.Header.Set(key, value)
	}
}

// isAllowedContentType проверяет допустимость content type
func (f *Fetcher) isAllowedContentType(contentType string) bool {
	if len(f.config.Security.AllowedContentTypes) == 0 {
		return true
	}

	// Extract base content type (before ;)
	contentType = strings.ToLower(strings.Split(contentType, ";")[0])
	contentType = strings.TrimSpace(contentType)

	for _, allowed := range f.config.Security.AllowedContentTypes {
		if strings.ToLower(allowed) == contentType {
			return true
		}
	}

	return false
}


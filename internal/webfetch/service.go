package webfetch

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/storage"
)

// Service high-level web fetch service
type Service struct {
	fetcher WebFetcher
	db      storage.Database
	config  Config
	logger  *logrus.Logger
}

// NewService создает новый service
func NewService(config Config, db storage.Database, logger *logrus.Logger) *Service {
	return &Service{
		fetcher: NewFetcher(config, logger),
		db:      db,
		config:  config,
		logger:  logger,
	}
}

// FetchURL извлекает URL с кешированием
func (s *Service) FetchURL(ctx context.Context, url string, opts FetchOptions) (*WebPage, error) {
	// Calculate URL hash for cache lookup
	urlHash := calculateURLHash(url)

	// Check cache if enabled
	if opts.CacheEnabled && s.config.Cache.Enabled {
		if cached, err := s.getCachedPage(ctx, urlHash); err == nil && cached != nil {
			// Check if cache is still valid
			if !cached.ExpiresAt.IsZero() && time.Now().Before(cached.ExpiresAt) {
				s.logger.WithFields(logrus.Fields{
					"url":        url,
					"expires_at": cached.ExpiresAt,
				}).Debug("Returning cached web page")

				cached.Cached = true
				return cached, nil
			}
		}
	}

	// Fetch fresh content
	page, err := s.fetcher.Fetch(ctx, url, opts)
	if err != nil {
		return nil, err
	}

	// Determine cache TTL
	cacheTTL := opts.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = s.config.Cache.DefaultTTL
	}

	// Set expiration
	if cacheTTL > 0 {
		page.ExpiresAt = time.Now().Add(cacheTTL)
	}

	// Store in cache (best effort, don't fail on cache errors)
	if s.config.Cache.Enabled {
		if err := s.storePage(ctx, url, urlHash, page); err != nil {
			s.logger.WithError(err).Warn("Failed to store page in cache")
		}
	}

	return page, nil
}

// FetchBatch извлекает множество URLs
func (s *Service) FetchBatch(ctx context.Context, urls []string, opts FetchOptions) ([]*WebPage, error) {
	return s.fetcher.FetchBatch(ctx, urls, opts)
}

// ValidateURL проверяет URL
func (s *Service) ValidateURL(url string) error {
	return s.fetcher.ValidateURL(url)
}

// calculateURLHash вычисляет hash для URL
func calculateURLHash(url string) string {
	hash := sha256.Sum256([]byte(url))
	return fmt.Sprintf("%x", hash)
}

// getCachedPage пытается получить страницу из кеша
func (s *Service) getCachedPage(ctx context.Context, urlHash string) (*WebPage, error) {
	// TODO: Implement database query when web_fetches CRUD is added
	// For now, return nil (cache miss)
	return nil, fmt.Errorf("cache not implemented yet")
}

// storePage сохраняет страницу в кеш
func (s *Service) storePage(ctx context.Context, url, urlHash string, page *WebPage) error {
	// TODO: Implement database insert when web_fetches CRUD is added
	// For now, just log
	s.logger.WithFields(logrus.Fields{
		"url":        url,
		"url_hash":   urlHash,
		"title":      page.Title,
		"word_count": len(page.Content),
	}).Debug("Would store page in cache (not implemented yet)")

	return nil
}

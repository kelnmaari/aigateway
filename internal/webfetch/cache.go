package webfetch

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Cache stores fetched web pages for reuse
type Cache struct {
	config     CacheConfig
	store      map[string]*WebPage
	expirations map[string]time.Time
	mutex      sync.RWMutex
	createdAt  map[string]time.Time // Track creation time for LRU eviction
}

// NewCache creates a new cache with the given configuration
func NewCache(config CacheConfig) *Cache {
	return &Cache{
		config:     config,
		store:      make(map[string]*WebPage),
		expirations: make(map[string]time.Time),
		createdAt:  make(map[string]time.Time),
	}
}

// Get retrieves a cached web page by URL hash
func (c *Cache) Get(urlHash string) (*WebPage, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	page, exists := c.store[urlHash]
	if !exists {
		return nil, false
	}

	// Check if the cache entry has expired
	if expiresAt, ok := c.expirations[urlHash]; ok && time.Now().After(expiresAt) {
		return nil, false
	}

	return page, true
}

// Set stores a web page in the cache with an expiration time
func (c *Cache) Set(page *WebPage) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	urlHash := computeURLHash(page.URL)
	ttl := c.config.DefaultTTL

	// Check for domain-specific TTL overrides
	if override, ok := c.config.TTLOverrides[extractDomain(page.URL)]; ok {
		ttl = override
	}

	expiresAt := time.Now().Add(ttl)
	page.ExpiresAt = expiresAt

	c.store[urlHash] = page
	c.expirations[urlHash] = expiresAt
	c.createdAt[urlHash] = time.Now()

	// If we've exceeded the max entries, remove the oldest one
	if len(c.store) > c.config.MaxEntries {
		c.removeOldest()
	}
}

// removeOldest removes the oldest entry from the cache (by creation time)
func (c *Cache) removeOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, createdAt := range c.createdAt {
		if oldestKey == "" || createdAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = createdAt
		}
	}

	if oldestKey != "" {
		delete(c.store, oldestKey)
		delete(c.expirations, oldestKey)
		delete(c.createdAt, oldestKey)
	}
}

// computeURLHash creates a hash for a URL string
func computeURLHash(url string) string {
	// Use a simple hash function for demonstration purposes
	// In production, consider using a cryptographic hash
	u := uuid.NewSHA1(uuid.Nil, []byte(url))
	return u.String()
}

package webfetch

import "time"

// Config represents the web fetch service configuration
type Config struct {
	HTTP HTTPConfig

	Security ValidationConfig

	RateLimit RateLimitConfig

	Parsing ParserConfig

	Summarization SummarizationConfig

	Cache CacheConfig
}

// HTTPConfig contains HTTP client settings
type HTTPConfig struct {
	Timeout         time.Duration
	UserAgent       string
	FollowRedirects bool
	MaxRedirects    int
	MaxResponseSize int64
	Headers         map[string]string
}

// ValidationConfig contains URL validation settings
type ValidationConfig struct {
	AllowedSchemes   []string
	BlockedDomains   []string
	AllowedDomains   []string
	BlockPrivateIPs  bool
	BlockLocalhost   bool
}

// RateLimitConfig contains rate limiting settings
type RateLimitConfig struct {
	Enabled               bool
	DefaultRequestsPerMin int
	PerDomainLimits       map[string]int
	RetryAttempts         int
	RetryDelay            time.Duration
	RetryMaxDelay         time.Duration
}

// ParserConfig contains HTML parsing settings
type ParserConfig struct {
	RemoveSelectors   []string
	ContentSelectors  []string
	ExtractLinks      bool
	ExtractImages     bool
	PreserveFormatting bool
}

// SummarizationConfig contains LLM summarization settings
type SummarizationConfig struct {
	Enabled    bool
	Provider   string
	Model      string
	Prompt     string
	MaxInputTokens  int
	MaxOutputTokens int
}

// CacheConfig contains caching settings
type CacheConfig struct {
	Enabled       bool
	DefaultTTL    time.Duration
	MaxEntries    int
	TTLOverrides map[string]time.Duration
}

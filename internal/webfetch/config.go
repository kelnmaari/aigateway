// Package webfetch provides web content fetching and parsing capabilities
package webfetch

import "time"

// Config конфигурация web fetcher
type Config struct {
	Enabled       bool                `mapstructure:"enabled"`
	HTTP          HTTPConfig          `mapstructure:"http"`
	Security      SecurityConfig      `mapstructure:"security"`
	RateLimit     RateLimitConfig     `mapstructure:"rate_limit"`
	Parsing       ParsingConfig       `mapstructure:"parsing"`
	Summarization SummarizationConfig `mapstructure:"summarization"`
	Cache         CacheConfig         `mapstructure:"cache"`
}

// HTTPConfig настройки HTTP клиента
type HTTPConfig struct {
	Timeout         time.Duration     `mapstructure:"timeout"`
	UserAgent       string            `mapstructure:"user_agent"`
	FollowRedirects bool              `mapstructure:"follow_redirects"`
	MaxRedirects    int               `mapstructure:"max_redirects"`
	MaxResponseSize int64             `mapstructure:"max_response_size"` // в байтах
	Headers         map[string]string `mapstructure:"headers"`
}

// SecurityConfig настройки безопасности
type SecurityConfig struct {
	AllowedSchemes      []string `mapstructure:"allowed_schemes"`
	BlockedDomains      []string `mapstructure:"blocked_domains"`
	AllowedDomains      []string `mapstructure:"allowed_domains"`
	BlockPrivateIPs     bool     `mapstructure:"block_private_ips"`
	BlockLocalhost      bool     `mapstructure:"block_localhost"`
	MaxContentSize      int64    `mapstructure:"max_content_size"`
	AllowedContentTypes []string `mapstructure:"allowed_content_types"`
}

// RateLimitConfig настройки rate limiting
type RateLimitConfig struct {
	Enabled                  bool           `mapstructure:"enabled"`
	DefaultRequestsPerMinute int            `mapstructure:"default_requests_per_minute"`
	PerDomainLimits          map[string]int `mapstructure:"per_domain_limits"`
	RetryAttempts            int            `mapstructure:"retry_attempts"`
	RetryDelay               time.Duration  `mapstructure:"retry_delay"`
	RetryMaxDelay            time.Duration  `mapstructure:"retry_max_delay"`
}

// ParsingConfig настройки парсинга HTML
type ParsingConfig struct {
	ExtractLinks       bool     `mapstructure:"extract_links"`
	ExtractImages      bool     `mapstructure:"extract_images"`
	PreserveFormatting bool     `mapstructure:"preserve_formatting"`
	RemoveScripts      bool     `mapstructure:"remove_scripts"`
	RemoveStyles       bool     `mapstructure:"remove_styles"`
	RemoveComments     bool     `mapstructure:"remove_comments"`
	RemoveSelectors    []string `mapstructure:"remove_selectors"`
	ContentSelectors   []string `mapstructure:"content_selectors"`
}

// SummarizationConfig настройки суммаризации через LLM
type SummarizationConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	Provider        string `mapstructure:"provider"`
	Model           string `mapstructure:"model"`
	Prompt          string `mapstructure:"prompt"`
	MaxInputTokens  int    `mapstructure:"max_input_tokens"`
	MaxOutputTokens int    `mapstructure:"max_output_tokens"`
}

// CacheConfig настройки кеширования
type CacheConfig struct {
	Enabled      bool                     `mapstructure:"enabled"`
	DefaultTTL   time.Duration            `mapstructure:"default_ttl"`
	MaxEntries   int                      `mapstructure:"max_entries"`
	TTLOverrides map[string]time.Duration `mapstructure:"ttl_overrides"`
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() Config {
	return Config{
		Enabled: true,
		HTTP: HTTPConfig{
			Timeout:         30 * time.Second,
			UserAgent:       "AIGateway-Bot/1.0",
			FollowRedirects: true,
			MaxRedirects:    5,
			MaxResponseSize: 10 * 1024 * 1024, // 10MB
			Headers: map[string]string{
				"Accept":          "text/html,application/xhtml+xml",
				"Accept-Language": "en-US,en;q=0.9",
			},
		},
		Security: SecurityConfig{
			AllowedSchemes:  []string{"http", "https"},
			BlockedDomains:  []string{},
			AllowedDomains:  []string{},
			BlockPrivateIPs: true,
			BlockLocalhost:  true,
			MaxContentSize:  10 * 1024 * 1024, // 10MB
			AllowedContentTypes: []string{
				"text/html",
				"application/xhtml+xml",
			},
		},
		RateLimit: RateLimitConfig{
			Enabled:                  true,
			DefaultRequestsPerMinute: 10,
			PerDomainLimits:          map[string]int{},
			RetryAttempts:            3,
			RetryDelay:               1 * time.Second,
			RetryMaxDelay:            10 * time.Second,
		},
		Parsing: ParsingConfig{
			ExtractLinks:       true,
			ExtractImages:      false,
			PreserveFormatting: false,
			RemoveScripts:      true,
			RemoveStyles:       true,
			RemoveComments:     true,
			RemoveSelectors: []string{
				"nav", "header", "footer",
				".advertisement", "#sidebar",
			},
			ContentSelectors: []string{
				"article", "main",
				".post-content", ".article-body",
				"#content",
			},
		},
		Summarization: SummarizationConfig{
			Enabled:         false, // По умолчанию выключено
			Provider:        "openai",
			Model:           "llama3.1:8b",
			MaxInputTokens:  4096,
			MaxOutputTokens: 256,
			Prompt: `Summarize the following web page content in 3-5 sentences.
Focus on the main points and key information.

Content:
{content}

Summary:`,
		},
		Cache: CacheConfig{
			Enabled:      true,
			DefaultTTL:   1 * time.Hour,
			MaxEntries:   10000,
			TTLOverrides: map[string]time.Duration{},
		},
	}
}


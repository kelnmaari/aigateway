// Package config provides configuration management for Ollama-OpenAI Proxy
// Поддерживает Go 1.25 features и современные паттерны
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config представляет конфигурацию всего приложения
type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Ollama        OllamaConfig        `mapstructure:"ollama"`
	Auth          AuthConfig          `mapstructure:"auth"`
	Database      DatabaseConfig      `mapstructure:"database"` // Version 1.3.0+: Database abstraction
	Logging       LoggingConfig       `mapstructure:"logging"`
	Models        ModelsConfig        `mapstructure:"models"`
	Tools         ToolsConfig         `mapstructure:"tools"`
	Prompts       PromptsConfig       `mapstructure:"prompts"` // Настройки промптов
	Metrics       MetricsConfig       `mapstructure:"metrics"`
	TUI           TUIConfig           `mapstructure:"tui"`
	Development   DevelopmentConfig   `mapstructure:"development"`
	Observability ObservabilityConfig `mapstructure:"observability"` // Version 1.6.0+: Tracing and monitoring
	FileStorage   FileStorageConfig   `mapstructure:"file_storage"`  // Version 1.10.0+: File storage and processing
	Extractors    ExtractorsConfig    `mapstructure:"extractors"`    // Version 1.10.0+: Document extractors
	WebFetch      WebFetchConfig      `mapstructure:"web_fetch"`     // Version 1.10.4+: Web content fetching
}

// ServerConfig конфигурация HTTP сервера
type ServerConfig struct {
	Host           string        `mapstructure:"host"`
	Port           int           `mapstructure:"port"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	IdleTimeout    time.Duration `mapstructure:"idle_timeout"`
	MaxHeaderBytes int           `mapstructure:"max_header_bytes"`

	// TLS настройки
	TLS struct {
		Enabled  bool   `mapstructure:"enabled"`
		CertFile string `mapstructure:"cert_file"`
		KeyFile  string `mapstructure:"key_file"`
		AutoCert bool   `mapstructure:"auto_cert"`
	} `mapstructure:"tls"`

	// CORS настройки
	CORS struct {
		Enabled        bool     `mapstructure:"enabled"`
		AllowedOrigins []string `mapstructure:"allowed_origins"`
		AllowedMethods []string `mapstructure:"allowed_methods"`
		AllowedHeaders []string `mapstructure:"allowed_headers"`
		MaxAge         int      `mapstructure:"max_age"`
	} `mapstructure:"cors"`
}

// OllamaConfig конфигурация подключения к Ollama
type OllamaConfig struct {
	URL                string        `mapstructure:"url"`
	Timeout            time.Duration `mapstructure:"timeout"`
	RetryAttempts      int           `mapstructure:"retry_attempts"`
	RetryDelay         time.Duration `mapstructure:"retry_delay"`
	ConnectionPoolSize int           `mapstructure:"connection_pool_size"`
	KeepAlive          bool          `mapstructure:"keep_alive"`

	// Health check настройки
	HealthCheck struct {
		Enabled  bool          `mapstructure:"enabled"`
		Interval time.Duration `mapstructure:"interval"`
		Timeout  time.Duration `mapstructure:"timeout"`
	} `mapstructure:"health_check"`

	// Circuit breaker настройки
	CircuitBreaker struct {
		Enabled      bool          `mapstructure:"enabled"`
		MaxFailures  int           `mapstructure:"max_failures"`
		ResetTimeout time.Duration `mapstructure:"reset_timeout"`
	} `mapstructure:"circuit_breaker"`
}

// AuthConfig конфигурация аутентификации
type AuthConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	StorageType string `mapstructure:"storage_type"` // json, sqlite
	StoragePath string `mapstructure:"storage_path"`
	AdminKey    string `mapstructure:"admin_key"`

	// Rate limiting настройки
	RateLimiting struct {
		Enabled                  bool `mapstructure:"enabled"`
		DefaultRequestsPerMinute int  `mapstructure:"default_requests_per_minute"`
		DefaultRequestsPerHour   int  `mapstructure:"default_requests_per_hour"`

		// Redis для distributed rate limiting
		Redis struct {
			Enabled   bool   `mapstructure:"enabled"`
			URL       string `mapstructure:"url"`
			KeyPrefix string `mapstructure:"key_prefix"`
		} `mapstructure:"redis"`
	} `mapstructure:"rate_limiting"`

	// JWT настройки (Version 1.3.0+: User Authentication)
	JWT struct {
		Secret             string        `mapstructure:"secret"`
		Expiry             time.Duration `mapstructure:"expiry"` // Deprecated: use AccessTokenExpiry
		AccessTokenExpiry  time.Duration `mapstructure:"access_token_expiry"`
		RefreshTokenExpiry time.Duration `mapstructure:"refresh_token_expiry"`
	} `mapstructure:"jwt"`

	// OIDC настройки (Version 1.11.1+: Keycloak SSO Integration)
	OIDC OIDCConfig `mapstructure:"oidc"`
}

// OIDCConfig конфигурация OpenID Connect для SSO
type OIDCConfig struct {
	// Enabled включить OIDC авториз ацию
	Enabled bool `mapstructure:"enabled"`

	// Provider провайдер OIDC (keycloak, google, azure, okta)
	Provider string `mapstructure:"provider"`

	// Issuer URL OIDC issuer (e.g., https://keycloak.example.com/realms/myrealm)
	Issuer string `mapstructure:"issuer" validate:"required_if=Enabled true,url"`

	// ClientID OIDC client ID
	ClientID string `mapstructure:"client_id" validate:"required_if=Enabled true"`

	// ClientSecret OIDC client secret
	ClientSecret string `mapstructure:"client_secret" validate:"required_if=Enabled true"`

	// RedirectURI redirect URI after authentication (e.g., https://proxy.example.com/auth/oidc/callback)
	RedirectURI string `mapstructure:"redirect_uri" validate:"required_if=Enabled true,url"`

	// Scopes OIDC scopes (default: openid, profile, email)
	Scopes []string `mapstructure:"scopes"`

	// Claims mapping конфигурация маппинга claims
	Claims ClaimsMapping `mapstructure:"claims"`

	// AutoCreateUser автоматически создавать пользователя при первом логине
	AutoCreateUser bool `mapstructure:"auto_create_user"`

	// AutoUpdateUser автоматически обновлять информацию о пользователе при каждом логине
	AutoUpdateUser bool `mapstructure:"auto_update_user"`

	// DefaultRole роль по умолчанию для новых пользователей (user, admin)
	DefaultRole string `mapstructure:"default_role"`

	// SessionStore хранилище сессий для state parameter (memory, redis)
	SessionStore string `mapstructure:"session_store"`

	// SessionTTL время жизни сессии для OIDC flow
	SessionTTL time.Duration `mapstructure:"session_ttl"`
}

// ClaimsMapping маппинг OIDC claims на поля пользователя
type ClaimsMapping struct {
	// UserID claim для user ID (default: "sub")
	UserID string `mapstructure:"user_id"`

	// Username claim для username (default: "preferred_username")
	Username string `mapstructure:"username"`

	// Email claim для email (default: "email")
	Email string `mapstructure:"email"`

	// Name claim для full name (default: "name")
	Name string `mapstructure:"name"`

	// Groups claim для групп (default: "groups")
	Groups string `mapstructure:"groups"`

	// Roles claim для ролей (optional)
	Roles string `mapstructure:"roles"`
}

// LoggingConfig конфигурация логирования
type LoggingConfig struct {
	Level      string `mapstructure:"level"`  // debug, info, warn, error
	Format     string `mapstructure:"format"` // text, json
	Output     string `mapstructure:"output"` // stdout, file
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"` // MB
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"` // days
	Compress   bool   `mapstructure:"compress"`

	// Structured fields для JSON логирования
	StructuredFields map[string]string `mapstructure:"structured_fields"`
}

// ModelsConfig конфигурация управления моделями
type ModelsConfig struct {
	// Маппинг имен моделей OpenAI -> Ollama
	Mapping map[string]string `mapstructure:"mapping"`

	// Алиасы моделей
	Aliases map[string]string `mapstructure:"aliases"`

	// Скрытые модели (не показывать в /v1/models)
	Hidden []string `mapstructure:"hidden"`

	// DefaultEmbeddingModel модель по умолчанию для embeddings
	// Если в запросе не указана модель или указана OpenAI модель (text-embedding-*),
	// будет использована эта модель
	// Рекомендуемые модели: nomic-embed-text, mxbai-embed-large, all-minilm
	DefaultEmbeddingModel string `mapstructure:"default_embedding_model"`

	// Настройки кеширования
	Cache struct {
		Enabled         bool          `mapstructure:"enabled"`
		TTL             time.Duration `mapstructure:"ttl"`
		RefreshInterval time.Duration `mapstructure:"refresh_interval"`
	} `mapstructure:"cache"`
}

// ToolsConfig конфигурация обработки tools (function calling)
type ToolsConfig struct {
	// ForceUsage автоматически применять tool_choice: "required" для всех запросов с tools
	// Когда true, модель будет обязана вызывать функции вместо текстового ответа
	ForceUsage bool `mapstructure:"force_usage"`

	// DefaultChoice значение tool_choice по умолчанию, если клиент не указал
	// Возможные значения: "auto", "required", "none"
	DefaultChoice string `mapstructure:"default_choice"`

	// FallbackModel модель для автоматического переключения при наличии tools
	// Когда указана и в запросе есть tools, запрос будет перенаправлен на эту модель
	// Пример: "llama3.1:latest" - модель с хорошей поддержкой function calling
	FallbackModel string `mapstructure:"fallback_model"`

	// Optimizer конфигурация оптимизации промптов для локальных моделей
	Optimizer OptimizerConfig `mapstructure:"optimizer"`
}

// OptimizerConfig конфигурация Smart Prompt Optimizer
type OptimizerConfig struct {
	// Enabled включить оптимизацию промптов
	Enabled bool `mapstructure:"enabled"`

	// SimplifySystemMessage упрощать system message убирая избыточные инструкции
	SimplifySystemMessage bool `mapstructure:"simplify_system_message"`

	// SmartToolFiltering фильтровать tools по релевантности к запросу
	SmartToolFiltering bool `mapstructure:"smart_tool_filtering"`

	// MaxToolsPerRequest максимум tools в одном запросе (0 = без лимита)
	MaxToolsPerRequest int `mapstructure:"max_tools_per_request"`

	// PreserveInstructions дополнительные инструкции для сохранения
	PreserveInstructions []string `mapstructure:"preserve_instructions"`
}

// MetricsConfig конфигурация метрик
type MetricsConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	PrometheusPath string `mapstructure:"prometheus_path"`

	// Внутренние метрики
	Collection struct {
		Enabled       bool          `mapstructure:"enabled"`
		BufferSize    int           `mapstructure:"buffer_size"`
		FlushInterval time.Duration `mapstructure:"flush_interval"`
	} `mapstructure:"collection"`

	// Экспорт метрик
	Export struct {
		Prometheus struct {
			Enabled      bool          `mapstructure:"enabled"`
			PushGateway  string        `mapstructure:"push_gateway"`
			JobName      string        `mapstructure:"job_name"`
			PushInterval time.Duration `mapstructure:"push_interval"`
		} `mapstructure:"prometheus"`
	} `mapstructure:"export"`
}

// TUIConfig конфигурация Terminal UI
type TUIConfig struct {
	Enabled     bool          `mapstructure:"enabled"`
	RefreshRate time.Duration `mapstructure:"refresh_rate"`
	Theme       string        `mapstructure:"theme"`

	// Настройки экранов
	Dashboard struct {
		ChartsHistory int  `mapstructure:"charts_history"`
		AutoRefresh   bool `mapstructure:"auto_refresh"`
	} `mapstructure:"dashboard"`

	RequestMonitor struct {
		MaxRequests int  `mapstructure:"max_requests"`
		AutoScroll  bool `mapstructure:"auto_scroll"`
	} `mapstructure:"request_monitor"`
}

// DevelopmentConfig настройки для разработки
type DevelopmentConfig struct {
	HotReload      bool `mapstructure:"hot_reload"`
	DebugMode      bool `mapstructure:"debug_mode"`
	ProfileEnabled bool `mapstructure:"profile_enabled"`
	PProfEnabled   bool `mapstructure:"pprof_enabled"`
	RaceDetection  bool `mapstructure:"race_detection"`

	// Mock настройки
	MockOllama struct {
		Enabled       bool          `mapstructure:"enabled"`
		ResponseDelay time.Duration `mapstructure:"response_delay"`
		RandomErrors  bool          `mapstructure:"random_errors"`
		ErrorRate     float64       `mapstructure:"error_rate"`
	} `mapstructure:"mock_ollama"`
}

// ObservabilityConfig настройки для observability и tracing (Version 1.6.0+)
type ObservabilityConfig struct {
	Tracing     TracingConfig     `mapstructure:"tracing"`
	Performance PerformanceConfig `mapstructure:"performance"`
}

// TracingConfig конфигурация OpenTelemetry distributed tracing
type TracingConfig struct {
	Enabled      bool    `mapstructure:"enabled"`
	Provider     string  `mapstructure:"provider"`      // "jaeger" или "zipkin"
	ServiceName  string  `mapstructure:"service_name"`  // Название сервиса в traces
	SamplingRate float64 `mapstructure:"sampling_rate"` // 0.0 - 1.0 (1.0 = 100%)

	// Jaeger specific
	Jaeger struct {
		Endpoint string `mapstructure:"endpoint"` // http://localhost:14268/api/traces
	} `mapstructure:"jaeger"`

	// Zipkin specific
	Zipkin struct {
		Endpoint string `mapstructure:"endpoint"` // http://localhost:9411/api/v2/spans
	} `mapstructure:"zipkin"`
}

// PerformanceConfig конфигурация performance monitoring (Version 1.6.2+)
type PerformanceConfig struct {
	Enabled              bool   `mapstructure:"enabled"`                // Включить performance monitoring
	CollectionInterval   string `mapstructure:"collection_interval"`    // Интервал сбора метрик (default: "30s")
	MemoryThresholdMB    int64  `mapstructure:"memory_threshold_mb"`    // Alert если heap > этого (default: 1024)
	GoroutineThreshold   int    `mapstructure:"goroutine_threshold"`    // Alert если goroutines > этого (default: 1000)
	SlowRequestThreshold string `mapstructure:"slow_request_threshold"` // Log запросы медленнее этого (default: "5s")
	GCPercentage         int    `mapstructure:"gc_percentage"`          // GOGC value (default: 100)
	LeakDetection        bool   `mapstructure:"leak_detection"`         // Включить leak detection (default: true)
	PprofEnabled         bool   `mapstructure:"pprof_enabled"`          // Включить pprof endpoints (default: true)
}

// FileStorageConfig конфигурация хранения файлов (Version 1.10.0+)
type FileStorageConfig struct {
	Backend string `mapstructure:"backend"` // "local" или "s3"

	// Local filesystem storage
	Local struct {
		BasePath        string   `mapstructure:"base_path"`      // ./data/files
		MaxFileSize     string   `mapstructure:"max_file_size"`  // 100MB
		MaxTotalSize    string   `mapstructure:"max_total_size"` // 10GB per user
		AllowedExts     []string `mapstructure:"allowed_extensions"`
		ScanViruses     bool     `mapstructure:"scan_viruses"`     // ClamAV integration (future)
		ValidateContent bool     `mapstructure:"validate_content"` // Magic number check
	} `mapstructure:"local"`

	// S3-compatible storage (MinIO)
	S3 struct {
		Endpoint        string `mapstructure:"endpoint"` // http://minio:9000
		Bucket          string `mapstructure:"bucket"`   // user-files
		AccessKey       string `mapstructure:"access_key"`
		SecretKey       string `mapstructure:"secret_key"`
		UseSSL          bool   `mapstructure:"use_ssl"`
		Region          string `mapstructure:"region"`            // us-east-1
		PublicBucket    string `mapstructure:"public_bucket"`     // For shared files
		SignedURLExpiry string `mapstructure:"signed_url_expiry"` // 1h
		MaxFileSize     string `mapstructure:"max_file_size"`     // 500MB
	} `mapstructure:"s3"`
}

// ExtractorsConfig конфигурация извлечения текста из документов (Version 1.10.0+)
type ExtractorsConfig struct {
	// PDF extraction
	PDF struct {
		Method         string `mapstructure:"method"` // "pdftotext" или "go-fitz"
		PreserveLayout bool   `mapstructure:"preserve_layout"`
		ExtractImages  bool   `mapstructure:"extract_images"` // For future IMAGE-01
		OCREnabled     bool   `mapstructure:"ocr_enabled"`    // For scanned PDFs (future)
		MaxPages       int    `mapstructure:"max_pages"`
	} `mapstructure:"pdf"`

	// DOCX extraction
	DOCX struct {
		ExtractTables   bool `mapstructure:"extract_tables"`
		ExtractImages   bool `mapstructure:"extract_images"`
		ExtractComments bool `mapstructure:"extract_comments"`
	} `mapstructure:"docx"`

	// Text files
	Text struct {
		MaxSize           string   `mapstructure:"max_size"` // 10MB
		Encoding          string   `mapstructure:"encoding"` // utf-8
		FallbackEncodings []string `mapstructure:"fallback_encodings"`
	} `mapstructure:"text"`

	// CSV files
	CSV struct {
		Delimiter           string `mapstructure:"delimiter"` // ,
		MaxRows             int    `mapstructure:"max_rows"`
		Encoding            string `mapstructure:"encoding"`
		AutoDetectDelimiter bool   `mapstructure:"auto_detect_delimiter"`
	} `mapstructure:"csv"`

	// General
	Timeout         string `mapstructure:"timeout"`          // 30s per file
	ParallelWorkers int    `mapstructure:"parallel_workers"` // 4 for batch processing
}

// Load загружает конфигурацию из файла и переменных окружения
func Load(configPath string) (*Config, error) {
	viper := viper.New()

	// Настройка поиска конфигурационных файлов
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		// Auto-discover config file: try config.yaml (production) or dev.yaml (development)
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath("../configs")
		viper.AddConfigPath("/opt/ollama-openai-proxy/configs")
		viper.AddConfigPath("/app/configs")

		// Try config.yaml first (production), fallback to dev.yaml (development)
		viper.SetConfigName("config")
	}

	// Настройка переменных окружения
	viper.SetEnvPrefix("PROXY")
	viper.AutomaticEnv()

	// Установка значений по умолчанию
	setDefaults(viper)

	// Чтение конфигурации
	if configPath != "" {
		fmt.Printf("🔍 Attempting to read config from: %s\n", configPath)
	} else {
		fmt.Printf("🔍 Auto-discovering config file (config.yaml or dev.yaml)...\n")
	}

	err := viper.ReadInConfig()
	if err != nil {
		// If config.yaml not found and we're auto-discovering, try dev.yaml
		if configPath == "" {
			viper.SetConfigName("dev")
			err = viper.ReadInConfig()
		}

		if err != nil {
			// Логируем предупреждение, но продолжаем с defaults
			fmt.Printf("⚠️  Config file not found or error reading config: %v\n", err)
			fmt.Printf("🔧 Using default configuration values\n")
			if configPath != "" {
				fmt.Printf("   Expected config path: %s\n", configPath)
			}
		} else {
			fmt.Printf("✅ Config file successfully read: %s\n", viper.ConfigFileUsed())
		}
	} else {
		fmt.Printf("✅ Config file successfully read: %s\n", viper.ConfigFileUsed())
	}

	// Unmarshaling в структуру Config
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Debug: показать источник конфигурации для server.host
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 SERVER CONFIGURATION DEBUG INFO:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if envHost := os.Getenv("PROXY_SERVER_HOST"); envHost != "" {
		fmt.Printf("⚠️  SERVER HOST OVERRIDDEN by environment variable:\n")
		fmt.Printf("   PROXY_SERVER_HOST = %s\n", envHost)
		fmt.Printf("   (This overrides config file!)\n")
	} else if viper.ConfigFileUsed() != "" {
		fmt.Printf("✅ Configuration loaded from file:\n")
		fmt.Printf("   File: %s\n", viper.ConfigFileUsed())
		fmt.Printf("   server.host from config: %s\n", viper.GetString("server.host"))
		fmt.Printf("   Actual config.Server.Host: %s\n", config.Server.Host)
	} else {
		fmt.Printf("⚠️  Using DEFAULT configuration:\n")
		fmt.Printf("   No config file loaded!\n")
		fmt.Printf("   Default server.host: %s\n", config.Server.Host)
	}

	fmt.Printf("\n🚀 Server will bind to: %s:%d\n", config.Server.Host, config.Server.Port)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Валидация конфигурации
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults устанавливает значения по умолчанию
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.idle_timeout", "60s")
	v.SetDefault("server.max_header_bytes", 1048576)

	// Ollama defaults
	v.SetDefault("ollama.url", "http://localhost:11434")
	v.SetDefault("ollama.timeout", "30s")
	v.SetDefault("ollama.retry_attempts", 3)
	v.SetDefault("ollama.retry_delay", "1s")
	v.SetDefault("ollama.connection_pool_size", 10)
	v.SetDefault("ollama.keep_alive", true)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.output", "stdout")

	// Auth defaults
	v.SetDefault("auth.enabled", false)
	v.SetDefault("auth.storage_type", "json")
	v.SetDefault("auth.storage_path", "data/api_keys.json")

	// Metrics defaults
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.prometheus_path", "/metrics")

	// TUI defaults
	v.SetDefault("tui.enabled", true)
	v.SetDefault("tui.refresh_rate", "1s")
	v.SetDefault("tui.theme", "default")

	// Observability defaults (Version 1.6.0+)
	v.SetDefault("observability.tracing.enabled", false)
	v.SetDefault("observability.tracing.provider", "jaeger")
	v.SetDefault("observability.tracing.service_name", "ollama-proxy")
	v.SetDefault("observability.tracing.sampling_rate", 1.0)
	v.SetDefault("observability.tracing.jaeger.endpoint", "http://localhost:14268/api/traces")
	v.SetDefault("observability.tracing.zipkin.endpoint", "http://localhost:9411/api/v2/spans")

	// Performance monitoring defaults (Version 1.6.2+)
	v.SetDefault("observability.performance.enabled", false)
	v.SetDefault("observability.performance.collection_interval", "30s")
	v.SetDefault("observability.performance.memory_threshold_mb", 1024)
	v.SetDefault("observability.performance.goroutine_threshold", 1000)
	v.SetDefault("observability.performance.slow_request_threshold", "5s")
	v.SetDefault("observability.performance.gc_percentage", 100)
	v.SetDefault("observability.performance.leak_detection", true)
	v.SetDefault("observability.performance.pprof_enabled", true)
}

// PromptsConfig конфигурация промптов и системных сообщений
type PromptsConfig struct {
	// AdditionalSystemMessage дополнительное системное сообщение, добавляемое ко всем запросам
	AdditionalSystemMessage string `mapstructure:"additional_system_message"`

	// PrependToSystem добавлять в начало system message (true) или в конец (false)
	PrependToSystem bool `mapstructure:"prepend_to_system"`
}

// Validate валидирует конфигурацию
func (c *Config) Validate() error {
	// Валидация порта
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// Валидация Ollama URL
	if c.Ollama.URL == "" {
		return fmt.Errorf("ollama URL cannot be empty")
	}

	// Валидация storage type
	if c.Auth.Enabled {
		switch c.Auth.StorageType {
		case "json", "sqlite":
			// OK
		default:
			return fmt.Errorf("invalid auth storage type: %s", c.Auth.StorageType)
		}

		if c.Auth.StoragePath == "" {
			return fmt.Errorf("auth storage path cannot be empty when auth is enabled")
		}
	}

	// Валидация logging level
	switch c.Logging.Level {
	case "debug", "info", "warn", "error":
		// OK
	default:
		return fmt.Errorf("invalid logging level: %s", c.Logging.Level)
	}

	return nil
}

// WebFetchConfig конфигурация web content fetcher (Version 1.10.4+)
type WebFetchConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Timeout string `mapstructure:"timeout"` // e.g., "30s"

	// Security settings
	BlockPrivateIPs bool     `mapstructure:"block_private_ips"`
	BlockLocalhost  bool     `mapstructure:"block_localhost"`
	AllowedDomains  []string `mapstructure:"allowed_domains"`
	BlockedDomains  []string `mapstructure:"blocked_domains"`

	// Rate limiting
	RateLimitEnabled      bool `mapstructure:"rate_limit_enabled"`
	DefaultRequestsPerMin int  `mapstructure:"default_requests_per_min"`

	// Cache
	CacheEnabled bool   `mapstructure:"cache_enabled"`
	CacheTTL     string `mapstructure:"cache_ttl"` // e.g., "1h"
}

// GetServerAddr возвращает адрес сервера в формате host:port
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// IsDevelopment возвращает true, если приложение запущено в режиме разработки
func (c *Config) IsDevelopment() bool {
	return c.Development.DebugMode || c.Logging.Level == "debug"
}

// Package config provides configuration management for Ollama-OpenAI Proxy
// Поддерживает Go 1.25 features и современные паттерны
package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config представляет конфигурацию всего приложения
type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Ollama      OllamaConfig      `mapstructure:"ollama"`
	Auth        AuthConfig        `mapstructure:"auth"`
	Database    DatabaseConfig    `mapstructure:"database"` // Version 1.3.0+: Database abstraction
	Logging     LoggingConfig     `mapstructure:"logging"`
	Models      ModelsConfig      `mapstructure:"models"`
	Tools       ToolsConfig       `mapstructure:"tools"`
	Prompts     PromptsConfig     `mapstructure:"prompts"` // Настройки промптов
	Metrics     MetricsConfig     `mapstructure:"metrics"`
	TUI         TUIConfig         `mapstructure:"tui"`
	Development DevelopmentConfig `mapstructure:"development"`
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

// Load загружает конфигурацию из файла и переменных окружения
func Load(configPath string) (*Config, error) {
	viper := viper.New()

	// Настройка поиска конфигурационных файлов
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("dev")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath("../configs")
		viper.AddConfigPath("/app/configs")
	}

	// Настройка переменных окружения
	viper.SetEnvPrefix("PROXY")
	viper.AutomaticEnv()

	// Установка значений по умолчанию
	setDefaults(viper)

	// Чтение конфигурации
	err := viper.ReadInConfig()
	if err != nil {
		// Логируем предупреждение, но продолжаем с defaults
		fmt.Printf("⚠️ Config file not found or error reading config: %v\n", err)
		fmt.Println("🔧 Using default configuration values")
	}

	// Unmarshaling в структуру Config
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

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

// GetServerAddr возвращает адрес сервера в формате host:port
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// IsDevelopment возвращает true, если приложение запущено в режиме разработки
func (c *Config) IsDevelopment() bool {
	return c.Development.DebugMode || c.Logging.Level == "debug"
}

// Package handlers provides HTTP handlers for API endpoints
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
)

// ConfigHandler обрабатывает запросы конфигурации
type ConfigHandler struct {
	config *config.Config
	logger *logrus.Logger
}

// NewConfigHandler создает новый handler конфигурации
func NewConfigHandler(cfg *config.Config, logger *logrus.Logger) *ConfigHandler {
	return &ConfigHandler{
		config: cfg,
		logger: logger,
	}
}

// ConfigResponse структура ответа с конфигурацией (без секретов)
type ConfigResponse struct {
	Server struct {
		Host           string `json:"host"`
		Port           int    `json:"port"`
		ReadTimeout    string `json:"read_timeout"`
		WriteTimeout   string `json:"write_timeout"`
		IdleTimeout    string `json:"idle_timeout"`
		MaxHeaderBytes int    `json:"max_header_bytes"`
	} `json:"server"`

	Ollama struct {
		URL                string `json:"url"`
		Timeout            string `json:"timeout"`
		RetryAttempts      int    `json:"retry_attempts"`
		RetryDelay         string `json:"retry_delay"`
		ConnectionPoolSize int    `json:"connection_pool_size"`
		KeepAlive          bool   `json:"keep_alive"`
	} `json:"ollama"`

	Auth struct {
		Enabled      bool   `json:"enabled"`
		StorageType  string `json:"storage_type"`
		StoragePath  string `json:"storage_path"`
		RateLimiting struct {
			Enabled                  bool `json:"enabled"`
			DefaultRequestsPerMinute int  `json:"default_requests_per_minute"`
			DefaultRequestsPerHour   int  `json:"default_requests_per_hour"`
		} `json:"rate_limiting"`
	} `json:"auth"`

	Logging struct {
		Level      string `json:"level"`
		Format     string `json:"format"`
		Output     string `json:"output"`
		FilePath   string `json:"file_path,omitempty"`
		MaxSize    int    `json:"max_size,omitempty"`
		MaxBackups int    `json:"max_backups,omitempty"`
		MaxAge     int    `json:"max_age,omitempty"`
		Compress   bool   `json:"compress,omitempty"`
	} `json:"logging"`

	Models struct {
		Mapping map[string]string `json:"mapping"`
		Aliases map[string]string `json:"aliases"`
		Hidden  []string          `json:"hidden"`
		Cache   struct {
			Enabled         bool   `json:"enabled"`
			TTL             string `json:"ttl"`
			RefreshInterval string `json:"refresh_interval"`
		} `json:"cache"`
	} `json:"models"`

	Tools struct {
		ForceUsage    bool   `json:"force_usage"`
		DefaultChoice string `json:"default_choice"`
		FallbackModel string `json:"fallback_model,omitempty"`
		Optimizer     struct {
			Enabled               bool     `json:"enabled"`
			SimplifySystemMessage bool     `json:"simplify_system_message"`
			SmartToolFiltering    bool     `json:"smart_tool_filtering"`
			MaxToolsPerRequest    int      `json:"max_tools_per_request"`
			PreserveInstructions  []string `json:"preserve_instructions"`
		} `json:"optimizer"`
	} `json:"tools"`

	Metrics struct {
		Enabled        bool   `json:"enabled"`
		PrometheusPath string `json:"prometheus_path"`
	} `json:"metrics"`

	TUI struct {
		Enabled     bool   `json:"enabled"`
		RefreshRate string `json:"refresh_rate"`
		Theme       string `json:"theme"`
	} `json:"tui"`

	Development struct {
		HotReload      bool `json:"hot_reload"`
		DebugMode      bool `json:"debug_mode"`
		ProfileEnabled bool `json:"profile_enabled"`
		PProfEnabled   bool `json:"pprof_enabled"`
		RaceDetection  bool `json:"race_detection"`
	} `json:"development"`
}

// GetConfig возвращает текущую конфигурацию (без секретов)
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	h.logger.Debug("Fetching current configuration")

	response := ConfigResponse{}

	// Server config
	response.Server.Host = h.config.Server.Host
	response.Server.Port = h.config.Server.Port
	response.Server.ReadTimeout = h.config.Server.ReadTimeout.String()
	response.Server.WriteTimeout = h.config.Server.WriteTimeout.String()
	response.Server.IdleTimeout = h.config.Server.IdleTimeout.String()
	response.Server.MaxHeaderBytes = h.config.Server.MaxHeaderBytes

	// Ollama config
	response.Ollama.URL = h.config.Ollama.URL
	response.Ollama.Timeout = h.config.Ollama.Timeout.String()
	response.Ollama.RetryAttempts = h.config.Ollama.RetryAttempts
	response.Ollama.RetryDelay = h.config.Ollama.RetryDelay.String()
	response.Ollama.ConnectionPoolSize = h.config.Ollama.ConnectionPoolSize
	response.Ollama.KeepAlive = h.config.Ollama.KeepAlive

	// Auth config (без секретов!)
	response.Auth.Enabled = h.config.Auth.Enabled
	response.Auth.StorageType = h.config.Auth.StorageType
	response.Auth.StoragePath = h.config.Auth.StoragePath
	response.Auth.RateLimiting.Enabled = h.config.Auth.RateLimiting.Enabled
	response.Auth.RateLimiting.DefaultRequestsPerMinute = h.config.Auth.RateLimiting.DefaultRequestsPerMinute
	response.Auth.RateLimiting.DefaultRequestsPerHour = h.config.Auth.RateLimiting.DefaultRequestsPerHour

	// Logging config
	response.Logging.Level = h.config.Logging.Level
	response.Logging.Format = h.config.Logging.Format
	response.Logging.Output = h.config.Logging.Output
	response.Logging.FilePath = h.config.Logging.FilePath
	response.Logging.MaxSize = h.config.Logging.MaxSize
	response.Logging.MaxBackups = h.config.Logging.MaxBackups
	response.Logging.MaxAge = h.config.Logging.MaxAge
	response.Logging.Compress = h.config.Logging.Compress

	// Models config
	response.Models.Mapping = h.config.Models.Mapping
	response.Models.Aliases = h.config.Models.Aliases
	response.Models.Hidden = h.config.Models.Hidden
	response.Models.Cache.Enabled = h.config.Models.Cache.Enabled
	response.Models.Cache.TTL = h.config.Models.Cache.TTL.String()
	response.Models.Cache.RefreshInterval = h.config.Models.Cache.RefreshInterval.String()

	// Tools config
	response.Tools.ForceUsage = h.config.Tools.ForceUsage
	response.Tools.DefaultChoice = h.config.Tools.DefaultChoice
	response.Tools.FallbackModel = h.config.Tools.FallbackModel
	response.Tools.Optimizer.Enabled = h.config.Tools.Optimizer.Enabled
	response.Tools.Optimizer.SimplifySystemMessage = h.config.Tools.Optimizer.SimplifySystemMessage
	response.Tools.Optimizer.SmartToolFiltering = h.config.Tools.Optimizer.SmartToolFiltering
	response.Tools.Optimizer.MaxToolsPerRequest = h.config.Tools.Optimizer.MaxToolsPerRequest
	response.Tools.Optimizer.PreserveInstructions = h.config.Tools.Optimizer.PreserveInstructions

	// Metrics config
	response.Metrics.Enabled = h.config.Metrics.Enabled
	response.Metrics.PrometheusPath = h.config.Metrics.PrometheusPath

	// TUI config
	response.TUI.Enabled = h.config.TUI.Enabled
	response.TUI.RefreshRate = h.config.TUI.RefreshRate.String()
	response.TUI.Theme = h.config.TUI.Theme

	// Development config
	response.Development.HotReload = h.config.Development.HotReload
	response.Development.DebugMode = h.config.Development.DebugMode
	response.Development.ProfileEnabled = h.config.Development.ProfileEnabled
	response.Development.PProfEnabled = h.config.Development.PProfEnabled
	response.Development.RaceDetection = h.config.Development.RaceDetection

	c.JSON(http.StatusOK, response)
}


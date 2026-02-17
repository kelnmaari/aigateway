// Package config - Hybrid configuration source with DB-first, YAML fallback
// Version: v3.1.0
package config

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// SettingsManager interface для избежания циклических зависимостей
// settings.Manager реализует этот интерфейс
type SettingsManager interface {
	GetString(ctx context.Context, id string) (string, error)
	GetInt(ctx context.Context, id string) (int, error)
	GetBool(ctx context.Context, id string) (bool, error)
	GetDuration(ctx context.Context, id string) (time.Duration, error)
	InvalidateCache()
}

// HybridConfigSource implements ConfigSource with cascading lookup:
// 1. Try database first (via SettingsManager)
// 2. Fallback to YAML config if DB lookup fails
// 3. Return error only if both fail
type HybridConfigSource struct {
	db       SettingsManager
	fallback *Config // Original YAML config
	logger   *logrus.Logger
	ctx      context.Context
}

// NewHybridConfigSource creates a new hybrid config source
func NewHybridConfigSource(db SettingsManager, fallback *Config, logger *logrus.Logger) *HybridConfigSource {
	if logger == nil {
		logger = logrus.New()
	}
	
	return &HybridConfigSource{
		db:       db,
		fallback: fallback,
		logger:   logger,
		ctx:      context.Background(),
	}
}

// GetString retrieves a string value (DB first, YAML fallback)
func (h *HybridConfigSource) GetString(ctx context.Context, key string) (string, error) {
	// Try database first
	if h.db != nil {
		if val, err := h.db.GetString(ctx, key); err == nil {
			h.logger.WithFields(logrus.Fields{
				"key":    key,
				"value":  val,
				"source": "database",
			}).Debug("Config loaded from database")
			return val, nil
		}
	}
	
	// Fallback to YAML
	if h.fallback != nil {
		if val := h.getFromYAML(key); val != "" {
			h.logger.WithFields(logrus.Fields{
				"key":    key,
				"value":  val,
				"source": "yaml",
			}).Debug("Config loaded from YAML fallback")
			return val, nil
		}
	}
	
	return "", fmt.Errorf("config key not found: %s", key)
}

// GetStringWithDefault retrieves a string with default fallback
func (h *HybridConfigSource) GetStringWithDefault(ctx context.Context, key, defaultValue string) string {
	val, err := h.GetString(ctx, key)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetInt retrieves an integer value
func (h *HybridConfigSource) GetInt(ctx context.Context, key string) (int, error) {
	// Try database first
	if h.db != nil {
		if val, err := h.db.GetInt(ctx, key); err == nil {
			return val, nil
		}
	}
	
	// Fallback to YAML
	if h.fallback != nil {
		if val := h.getIntFromYAML(key); val != 0 {
			return val, nil
		}
	}
	
	return 0, fmt.Errorf("config key not found: %s", key)
}

// GetIntWithDefault retrieves an int with default fallback
func (h *HybridConfigSource) GetIntWithDefault(ctx context.Context, key string, defaultValue int) int {
	val, err := h.GetInt(ctx, key)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetInt64 retrieves an int64 value
func (h *HybridConfigSource) GetInt64(ctx context.Context, key string) (int64, error) {
	val, err := h.GetInt(ctx, key)
	return int64(val), err
}

// GetInt64WithDefault retrieves an int64 with default fallback
func (h *HybridConfigSource) GetInt64WithDefault(ctx context.Context, key string, defaultValue int64) int64 {
	val, err := h.GetInt64(ctx, key)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetFloat64 retrieves a float64 value
func (h *HybridConfigSource) GetFloat64(ctx context.Context, key string) (float64, error) {
	strVal, err := h.GetString(ctx, key)
	if err != nil {
		return 0, err
	}
	
	val, err := strconv.ParseFloat(strVal, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse float for key %s: %w", key, err)
	}
	
	return val, nil
}

// GetFloat64WithDefault retrieves a float64 with default fallback
func (h *HybridConfigSource) GetFloat64WithDefault(ctx context.Context, key string, defaultValue float64) float64 {
	val, err := h.GetFloat64(ctx, key)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetBool retrieves a boolean value
func (h *HybridConfigSource) GetBool(ctx context.Context, key string) (bool, error) {
	// Try database first
	if h.db != nil {
		if val, err := h.db.GetBool(ctx, key); err == nil {
			return val, nil
		}
	}
	
	// Fallback to YAML
	if h.fallback != nil {
		if val := h.getBoolFromYAML(key); val {
			return val, nil
		}
	}
	
	return false, fmt.Errorf("config key not found: %s", key)
}

// GetBoolWithDefault retrieves a bool with default fallback
func (h *HybridConfigSource) GetBoolWithDefault(ctx context.Context, key string, defaultValue bool) bool {
	val, err := h.GetBool(ctx, key)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetDuration retrieves a duration value
func (h *HybridConfigSource) GetDuration(ctx context.Context, key string) (time.Duration, error) {
	// Try database first
	if h.db != nil {
		if val, err := h.db.GetDuration(ctx, key); err == nil {
			return val, nil
		}
	}
	
	// Fallback to YAML
	if h.fallback != nil {
		if val := h.getDurationFromYAML(key); val != 0 {
			return val, nil
		}
	}
	
	return 0, fmt.Errorf("config key not found: %s", key)
}

// GetDurationWithDefault retrieves a duration with default fallback
func (h *HybridConfigSource) GetDurationWithDefault(ctx context.Context, key string, defaultValue time.Duration) time.Duration {
	val, err := h.GetDuration(ctx, key)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetStringSlice retrieves a string slice value
func (h *HybridConfigSource) GetStringSlice(ctx context.Context, key string) ([]string, error) {
	strVal, err := h.GetString(ctx, key)
	if err != nil {
		return nil, err
	}
	
	// Parse comma-separated string
	if strVal == "" {
		return []string{}, nil
	}
	
	parts := strings.Split(strVal, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	
	return result, nil
}

// GetStringSliceWithDefault retrieves a string slice with default fallback
func (h *HybridConfigSource) GetStringSliceWithDefault(ctx context.Context, key string, defaultValue []string) []string {
	val, err := h.GetStringSlice(ctx, key)
	if err != nil {
		return defaultValue
	}
	return val
}

// Reload triggers cache invalidation for the database manager
func (h *HybridConfigSource) Reload(ctx context.Context) error {
	if h.db != nil {
		h.db.InvalidateCache()
		h.logger.Info("Config cache invalidated, reloading from database")
	}
	return nil
}

// getFromYAML extracts value from YAML config using dot notation
// Example: "server.port" → config.Server.Port
func (h *HybridConfigSource) getFromYAML(key string) string {
	if h.fallback == nil {
		return ""
	}
	
	parts := strings.Split(key, ".")
	if len(parts) < 2 {
		return ""
	}
	
	category := parts[0]
	field := strings.Join(parts[1:], ".")
	
	switch category {
	case "server":
		return h.getServerField(field)
	case "database":
		return h.getDatabaseField(field)
	case "auth":
		return h.getAuthField(field)
	case "logging":
		return h.getLoggingField(field)
	case "metrics":
		return h.getMetricsField(field)
	case "inference":
		return h.getInferenceField(field)
	default:
		return ""
	}
}

// getServerField extracts server config fields
func (h *HybridConfigSource) getServerField(field string) string {
	switch field {
	case "host":
		return h.fallback.Server.Host
	case "port":
		return strconv.Itoa(h.fallback.Server.Port)
	case "read_timeout":
		return h.fallback.Server.ReadTimeout.String()
	case "write_timeout":
		return h.fallback.Server.WriteTimeout.String()
	case "idle_timeout":
		return h.fallback.Server.IdleTimeout.String()
	case "max_header_bytes":
		return strconv.Itoa(h.fallback.Server.MaxHeaderBytes)
	case "tls.enabled":
		return strconv.FormatBool(h.fallback.Server.TLS.Enabled)
	case "tls.common_name":
		return h.fallback.Server.TLS.CommonName
	case "tls.cert_file":
		return h.fallback.Server.TLS.CertFile
	case "tls.key_file":
		return h.fallback.Server.TLS.KeyFile
	case "tls.valid_days":
		return strconv.Itoa(h.fallback.Server.TLS.ValidDays)
	default:
		return ""
	}
}

// getDatabaseField extracts database config fields
func (h *HybridConfigSource) getDatabaseField(field string) string {
	switch field {
	case "type":
		return string(h.fallback.Database.Type)
	case "postgresql.host":
		return h.fallback.Database.PostgreSQL.Host
	case "postgresql.port":
		return strconv.Itoa(h.fallback.Database.PostgreSQL.Port)
	case "postgresql.database":
		return h.fallback.Database.PostgreSQL.Database
	case "postgresql.user":
		return h.fallback.Database.PostgreSQL.User
	case "postgresql.password":
		return h.fallback.Database.PostgreSQL.Password
	case "postgresql.ssl_mode":
		return h.fallback.Database.PostgreSQL.SSLMode
	case "postgresql.max_open_conns":
		return strconv.Itoa(h.fallback.Database.PostgreSQL.MaxOpenConns)
	case "postgresql.max_idle_conns":
		return strconv.Itoa(h.fallback.Database.PostgreSQL.MaxIdleConns)
	default:
		return ""
	}
}

// getAuthField extracts auth config fields
func (h *HybridConfigSource) getAuthField(field string) string {
	switch field {
	case "jwt.secret":
		return h.fallback.Auth.JWT.Secret
	case "jwt.access_token_expiry":
		return h.fallback.Auth.JWT.AccessTokenExpiry.String()
	case "jwt.refresh_token_expiry":
		return h.fallback.Auth.JWT.RefreshTokenExpiry.String()
	default:
		return ""
	}
}

// getLoggingField extracts logging config fields
func (h *HybridConfigSource) getLoggingField(field string) string {
	switch field {
	case "level":
		return h.fallback.Logging.Level
	case "format":
		return h.fallback.Logging.Format
	case "output":
		return h.fallback.Logging.Output
	case "file_path":
		return h.fallback.Logging.FilePath
	default:
		return ""
	}
}

// getMetricsField extracts metrics config fields
func (h *HybridConfigSource) getMetricsField(field string) string {
	switch field {
	case "enabled":
		return strconv.FormatBool(h.fallback.Metrics.Enabled)
	case "prometheus_path":
		return h.fallback.Metrics.PrometheusPath
	default:
		return ""
	}
}

// getInferenceField extracts inference config fields
func (h *HybridConfigSource) getInferenceField(field string) string {
	switch field {
	case "provider":
		return "docker"
	case "backend":
		return h.fallback.Inference.Backend
	case "gpu_layers":
		return strconv.Itoa(h.fallback.Inference.GPULayers)
	case "max_loaded_models":
		return strconv.Itoa(h.fallback.Inference.MaxLoadedModels)
	default:
		return ""
	}
}

// Helper functions for YAML extraction with type conversion

func (h *HybridConfigSource) getIntFromYAML(key string) int {
	strVal := h.getFromYAML(key)
	if strVal == "" {
		return 0
	}
	
	val, err := strconv.Atoi(strVal)
	if err != nil {
		h.logger.WithError(err).WithField("key", key).Warn("Failed to parse int from YAML")
		return 0
	}
	
	return val
}

func (h *HybridConfigSource) getBoolFromYAML(key string) bool {
	strVal := h.getFromYAML(key)
	if strVal == "" {
		return false
	}
	
	val, err := strconv.ParseBool(strVal)
	if err != nil {
		h.logger.WithError(err).WithField("key", key).Warn("Failed to parse bool from YAML")
		return false
	}
	
	return val
}

func (h *HybridConfigSource) getDurationFromYAML(key string) time.Duration {
	strVal := h.getFromYAML(key)
	if strVal == "" {
		return 0
	}
	
	val, err := time.ParseDuration(strVal)
	if err != nil {
		h.logger.WithError(err).WithField("key", key).Warn("Failed to parse duration from YAML")
		return 0
	}
	
	return val
}


// Package settings - Configuration Seeder
// Version: v3.0.9 - Phase 2
package settings

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
)

// ConfigSeeder loads settings from YAML config into database
type ConfigSeeder struct {
	storage Storage
	logger  *logrus.Logger
}

// NewConfigSeeder creates a new config seeder
func NewConfigSeeder(storage Storage, logger *logrus.Logger) *ConfigSeeder {
	return &ConfigSeeder{
		storage: storage,
		logger:  logger,
	}
}

// SeedFromYAML populates database with settings from YAML config
// Returns number of settings seeded and any error
func (s *ConfigSeeder) SeedFromYAML(ctx context.Context, cfg *config.Config) (int, error) {
	return s.SeedFromYAMLForce(ctx, cfg, false)
}

// SeedFromYAMLForce populates database with optional force overwrite
func (s *ConfigSeeder) SeedFromYAMLForce(ctx context.Context, cfg *config.Config, force bool) (int, error) {
	s.logger.Info("🌱 Starting settings seed from YAML config...")

	// Check if already seeded
	existing, err := s.storage.GetAllSettings(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to check existing settings: %w", err)
	}

	if len(existing) > 0 && !force {
		s.logger.WithField("count", len(existing)).Info("Settings already seeded, skipping")
		return 0, nil
	}

	// Map YAML config to settings
	settings := s.MapConfigToSettings(cfg)

	// Save all settings
	count := 0
	for _, setting := range settings {
		if err := s.storage.UpsertSetting(ctx, &setting); err != nil {
			s.logger.WithError(err).WithField("id", setting.ID).Warn("Failed to save setting, continuing...")
			continue
		}
		count++
	}

	s.logger.WithField("seeded", count).WithField("total", len(settings)).Info("✅ Settings seeded from YAML")
	return count, nil
}

// MapConfigToSettings converts config.Config to []Setting (public for CLI)
func (s *ConfigSeeder) MapConfigToSettings(cfg *config.Config) []Setting {
	settings := make([]Setting, 0, 100) // Прогнозируем ~100 настроек

	// === Server Settings ===
	settings = append(settings, Setting{
		ID:           "server.host",
		Category:     CategoryServer,
		Key:          "host",
		Value:        cfg.Server.Host,
		Type:         TypeString,
		DefaultValue: "0.0.0.0",
		Description:  "Server host address to bind",
		IsEditable:   false, // Requires restart
		IsRequired:   true,
		IsMigrated:   true,
		UpdatedAt:    time.Now(),
	})

	settings = append(settings, Setting{
		ID:           "server.port",
		Category:     CategoryServer,
		Key:          "port",
		Value:        fmt.Sprintf("%d", cfg.Server.Port),
		Type:         TypeInt,
		DefaultValue: "8085",
		Description:  "Server port to listen on",
		IsEditable:   false, // Requires restart
		IsRequired:   true,
		IsMigrated:   true,
		UpdatedAt:    time.Now(),
	})

	settings = append(settings, Setting{
		ID:              "server.read_timeout",
		Category:        CategoryServer,
		Key:             "read_timeout",
		Value:           time.Duration(cfg.Server.ReadTimeout).String(),
		Type:            TypeDuration,
		DefaultValue:    "30s",
		Description:     "HTTP read timeout",
		IsEditable:      true,  // Can be changed (affects new connections)
		RequiresRestart: false, // Phase 4: Hot-reload (applies to new connections)
		IsRequired:      false,
		IsMigrated:      true,
		UpdatedAt:       time.Now(),
	})

	settings = append(settings, Setting{
		ID:              "server.write_timeout",
		Category:        CategoryServer,
		Key:             "write_timeout",
		Value:           time.Duration(cfg.Server.WriteTimeout).String(),
		Type:            TypeDuration,
		DefaultValue:    "30s",
		Description:     "HTTP write timeout",
		IsEditable:      true,  // Can be changed (affects new connections)
		RequiresRestart: false, // Phase 4: Hot-reload (applies to new connections)
		IsRequired:      false,
		IsMigrated:      true,
		UpdatedAt:       time.Now(),
	})

	// TLS settings
	settings = append(settings, Setting{
		ID:           "server.tls.enabled",
		Category:     CategoryServer,
		Key:          "tls_enabled",
		Value:        fmt.Sprintf("%t", cfg.Server.TLS.Enabled),
		Type:         TypeBool,
		DefaultValue: "false",
		Description:  "Enable HTTPS/TLS server",
		IsEditable:   false, // Requires restart
		IsRequired:   false,
		IsMigrated:   true,
		UpdatedAt:    time.Now(),
	})

	// === Auth Settings ===
	settings = append(settings, Setting{
		ID:           "auth.enabled",
		Category:     CategoryAuth,
		Key:          "enabled",
		Value:        fmt.Sprintf("%t", cfg.Auth.Enabled),
		Type:         TypeBool,
		DefaultValue: "true",
		Description:  "Enable authentication system",
		IsEditable:   false, // Critical security setting
		IsRequired:   true,
		IsMigrated:   true,
		UpdatedAt:    time.Now(),
	})

	settings = append(settings, Setting{
		ID:              "auth.jwt.expiration",
		Category:        CategoryAuth,
		Key:             "jwt_expiration",
		Value:           cfg.Auth.JWT.Expiry.String(),
		Type:            TypeDuration,
		DefaultValue:    "24h",
		Description:     "JWT token expiration time",
		IsEditable:      true,  // Can be changed
		RequiresRestart: false, // Phase 4: Applies to new tokens
		IsRequired:      false,
		IsMigrated:      true,
		UpdatedAt:       time.Now(),
	})

	// Rate limiting
	settings = append(settings, Setting{
		ID:              "auth.rate_limiting.default_requests_per_minute",
		Category:        CategoryAuth,
		Key:             "rate_limit_rpm",
		Value:           fmt.Sprintf("%d", cfg.Auth.RateLimiting.DefaultRequestsPerMinute),
		Type:            TypeInt,
		DefaultValue:    "30",
		Description:     "Default requests per minute limit",
		IsEditable:      true,  // Hot-reloadable
		RequiresRestart: false, // Phase 4: Can be updated live
		IsRequired:      false,
		IsMigrated:      true,
		UpdatedAt:       time.Now(),
	})

	settings = append(settings, Setting{
		ID:              "auth.rate_limiting.default_requests_per_hour",
		Category:        CategoryAuth,
		Key:             "rate_limit_rph",
		Value:           fmt.Sprintf("%d", cfg.Auth.RateLimiting.DefaultRequestsPerHour),
		Type:            TypeInt,
		DefaultValue:    "500",
		Description:     "Default requests per hour limit",
		IsEditable:      true,  // Hot-reloadable
		RequiresRestart: false, // Phase 4: Can be updated live
		IsRequired:      false,
		IsMigrated:      true,
		UpdatedAt:       time.Now(),
	})

	// === Logging Settings ===
	settings = append(settings, Setting{
		ID:              "logging.level",
		Category:        CategoryLogging,
		Key:             "level",
		Value:           cfg.Logging.Level,
		Type:            TypeString,
		DefaultValue:    "info",
		Description:     "Log level (debug, info, warn, error)",
		IsEditable:      true,  // Hot-reloadable
		RequiresRestart: false, // Phase 4: Can update logger level live
		IsRequired:      false,
		IsMigrated:      true,
		ValidationRule:  "^(debug|info|warn|error)$",
		UpdatedAt:       time.Now(),
	})

	settings = append(settings, Setting{
		ID:              "logging.format",
		Category:        CategoryLogging,
		Key:             "format",
		Value:           cfg.Logging.Format,
		Type:            TypeString,
		DefaultValue:    "json",
		Description:     "Log format (json or text)",
		IsEditable:      true,
		RequiresRestart: true, // Requires restart to change formatter
		IsRequired:      false,
		IsMigrated:      true,
		ValidationRule:  "^(json|text)$",
		UpdatedAt:       time.Now(),
	})

	// === Inference Settings (yzma) ===
	settings = append(settings, Setting{
		ID:           "inference.yzma.enabled",
		Category:     CategoryInference,
		Key:          "yzma_enabled",
		Value:        fmt.Sprintf("%t", cfg.Yzma.Enabled),
		Type:         TypeBool,
		DefaultValue: "true",
		Description:  "Enable yzma local inference engine",
		IsEditable:   false, // Requires restart
		IsRequired:   false,
		IsMigrated:   true,
		UpdatedAt:    time.Now(),
	})

	settings = append(settings, Setting{
		ID:           "inference.yzma.models_dir",
		Category:     CategoryInference,
		Key:          "models_directory",
		Value:        cfg.Yzma.ModelsDir,
		Type:         TypeString,
		DefaultValue: "./models",
		Description:  "Directory for GGUF models",
		IsEditable:   false, // Requires restart
		IsRequired:   true,
		IsMigrated:   true,
		UpdatedAt:    time.Now(),
	})

	// === Database Settings ===
	settings = append(settings, Setting{
		ID:           "database.type",
		Category:     CategoryDatabase,
		Key:          "type",
		Value:        string(cfg.Database.Type),
		Type:         TypeString,
		DefaultValue: "postgresql",
		Description:  "Database type (postgresql or sqlite)",
		IsEditable:   false, // Critical, requires restart
		IsRequired:   true,
		IsMigrated:   true,
		UpdatedAt:    time.Now(),
	})

	// === Metrics Settings ===
	settings = append(settings, Setting{
		ID:              "metrics.enabled",
		Category:        CategoryMetrics,
		Key:             "enabled",
		Value:           fmt.Sprintf("%t", cfg.Metrics.Enabled),
		Type:            TypeBool,
		DefaultValue:    "true",
		Description:     "Enable Prometheus metrics export",
		IsEditable:      true,  // Hot-reloadable
		RequiresRestart: false, // Phase 4: Can toggle metrics live
		IsRequired:      false,
		IsMigrated:      true,
		UpdatedAt:       time.Now(),
	})

	// === RAG Settings ===
	if cfg.RAG.Enabled {
		settings = append(settings, Setting{
			ID:           "rag.enabled",
			Category:     CategoryRAG,
			Key:          "enabled",
			Value:        "true",
			Type:         TypeBool,
			DefaultValue: "false",
			Description:  "Enable RAG (Retrieval-Augmented Generation)",
			IsEditable:   false, // Requires restart
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})
	}

	s.logger.WithField("mapped", len(settings)).Debug("Mapped config to settings")
	return settings
}


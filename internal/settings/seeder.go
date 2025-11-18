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

	// Convert to []*Setting for BulkUpsertSettings
	settingPtrs := make([]*Setting, len(settings))
	for i := range settings {
		settingPtrs[i] = &settings[i]
	}

	// Use BulkUpsertSettings for initial seed (bypasses Phase 1 check)
	if err := s.storage.BulkUpsertSettings(ctx, settingPtrs); err != nil {
		return 0, fmt.Errorf("failed to bulk upsert settings: %w", err)
	}

	s.logger.WithField("seeded", len(settings)).WithField("total", len(settings)).Info("✅ Settings seeded from YAML")
	return len(settings), nil
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

	// === Additional Server Settings ===
	settings = append(settings, Setting{
		ID:           "server.idle_timeout",
		Category:     CategoryServer,
		Key:          "idle_timeout",
		Value:        time.Duration(cfg.Server.IdleTimeout).String(),
		Type:         TypeDuration,
		DefaultValue: "120s",
		Description:  "HTTP idle timeout",
		IsEditable:   false,
		IsRequired:   false,
		IsMigrated:   true,
		UpdatedAt:    time.Now(),
	})

	// TLS advanced settings
	if cfg.Server.TLS.Enabled {
		settings = append(settings, Setting{
			ID:           "server.tls.common_name",
			Category:     CategoryServer,
			Key:          "tls_common_name",
			Value:        cfg.Server.TLS.CommonName,
			Type:         TypeString,
			DefaultValue: "AIGateway Local",
			Description:  "TLS certificate common name",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "server.tls.valid_days",
			Category:     CategoryServer,
			Key:          "tls_valid_days",
			Value:        fmt.Sprintf("%d", cfg.Server.TLS.ValidDays),
			Type:         TypeInt,
			DefaultValue: "365",
			Description:  "TLS certificate validity period in days",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "server.tls.cert_file",
			Category:     CategoryServer,
			Key:          "tls_cert_file",
			Value:        cfg.Server.TLS.CertFile,
			Type:         TypeString,
			DefaultValue: "certs/cert.pem",
			Description:  "Path to TLS certificate file",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "server.tls.key_file",
			Category:     CategoryServer,
			Key:          "tls_key_file",
			Value:        cfg.Server.TLS.KeyFile,
			Type:         TypeString,
			DefaultValue: "certs/key.pem",
			Description:  "Path to TLS private key file",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})
	}

	// === Additional Auth Settings ===
	settings = append(settings, Setting{
		ID:           "auth.storage_type",
		Category:     CategoryAuth,
		Key:          "storage_type",
		Value:        cfg.Auth.StorageType,
		Type:         TypeString,
		DefaultValue: "json",
		Description:  "Auth storage type (json, sqlite)",
		IsEditable:   false,
		IsRequired:   false,
		IsMigrated:   true,
		UpdatedAt:    time.Now(),
	})

	settings = append(settings, Setting{
		ID:           "auth.storage_path",
		Category:     CategoryAuth,
		Key:          "storage_path",
		Value:        cfg.Auth.StoragePath,
		Type:         TypeString,
		DefaultValue: "./data/users.json",
		Description:  "Path to auth storage file",
		IsEditable:   false,
		IsRequired:   false,
		IsMigrated:   true,
		UpdatedAt:    time.Now(),
	})

	// JWT advanced
	settings = append(settings, Setting{
		ID:           "auth.jwt.secret",
		Category:     CategoryAuth,
		Key:          "jwt_secret",
		Value:        cfg.Auth.JWT.Secret,
		Type:         TypeString,
		DefaultValue: "",
		Description:  "JWT signing secret key",
		IsEditable:   false,
		IsRequired:   true,
		IsMigrated:   true,
		UpdatedAt:    time.Now(),
	})

	settings = append(settings, Setting{
		ID:           "auth.jwt.access_token_expiry",
		Category:     CategoryAuth,
		Key:          "jwt_access_token_expiry",
		Value:        cfg.Auth.JWT.AccessTokenExpiry.String(),
		Type:         TypeDuration,
		DefaultValue: "15m",
		Description:  "JWT access token expiration time",
		IsEditable:   true,
		RequiresRestart: false,
		IsRequired:   false,
		IsMigrated:   true,
		UpdatedAt:    time.Now(),
	})

	settings = append(settings, Setting{
		ID:           "auth.jwt.refresh_token_expiry",
		Category:     CategoryAuth,
		Key:          "jwt_refresh_token_expiry",
		Value:        cfg.Auth.JWT.RefreshTokenExpiry.String(),
		Type:         TypeDuration,
		DefaultValue: "168h",
		Description:  "JWT refresh token expiration time",
		IsEditable:   true,
		RequiresRestart: false,
		IsRequired:   false,
		IsMigrated:   true,
		UpdatedAt:    time.Now(),
	})

	// OIDC
	if cfg.Auth.OIDC.Enabled {
		settings = append(settings, Setting{
			ID:           "auth.oidc.enabled",
			Category:     CategoryAuth,
			Key:          "oidc_enabled",
			Value:        "true",
			Type:         TypeBool,
			DefaultValue: "false",
			Description:  "Enable OIDC authentication",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "auth.oidc.provider",
			Category:     CategoryAuth,
			Key:          "oidc_provider",
			Value:        cfg.Auth.OIDC.Provider,
			Type:         TypeString,
			DefaultValue: "keycloak",
			Description:  "OIDC provider (keycloak, google, azure, okta)",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "auth.oidc.client_id",
			Category:     CategoryAuth,
			Key:          "oidc_client_id",
			Value:        cfg.Auth.OIDC.ClientID,
			Type:         TypeString,
			DefaultValue: "",
			Description:  "OIDC client ID",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "auth.oidc.issuer",
			Category:     CategoryAuth,
			Key:          "oidc_issuer",
			Value:        cfg.Auth.OIDC.Issuer,
			Type:         TypeString,
			DefaultValue: "",
			Description:  "OIDC issuer URL",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})
	}

	// LDAP
	if cfg.Auth.LDAP.Enabled {
		settings = append(settings, Setting{
			ID:           "auth.ldap.enabled",
			Category:     CategoryAuth,
			Key:          "ldap_enabled",
			Value:        "true",
			Type:         TypeBool,
			DefaultValue: "false",
			Description:  "Enable LDAP authentication",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "auth.ldap.url",
			Category:     CategoryAuth,
			Key:          "ldap_url",
			Value:        cfg.Auth.LDAP.URL,
			Type:         TypeString,
			DefaultValue: "",
			Description:  "LDAP server URL (ldap:// or ldaps://)",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "auth.ldap.bind_dn",
			Category:     CategoryAuth,
			Key:          "ldap_bind_dn",
			Value:        cfg.Auth.LDAP.BindDN,
			Type:         TypeString,
			DefaultValue: "",
			Description:  "LDAP bind DN for authentication",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})
	}

	// Registration & Invitations
	settings = append(settings, Setting{
		ID:           "auth.registration.mode",
		Category:     CategoryAuth,
		Key:          "registration_mode",
		Value:        cfg.Auth.Registration.Mode,
		Type:         TypeString,
		DefaultValue: "disabled",
		Description:  "Registration mode (open, invitation_only, disabled)",
		IsEditable:   true,
		RequiresRestart: false,
		IsRequired:   false,
		IsMigrated:   true,
		ValidationRule: "^(open|invitation_only|disabled)$",
		UpdatedAt:    time.Now(),
	})

	if cfg.Auth.Invitations.Enabled {
		settings = append(settings, Setting{
			ID:           "auth.invitations.enabled",
			Category:     CategoryAuth,
			Key:          "invitations_enabled",
			Value:        "true",
			Type:         TypeBool,
			DefaultValue: "false",
			Description:  "Enable invitation system",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "auth.invitations.default_expiry_days",
			Category:     CategoryAuth,
			Key:          "invitations_expiry",
			Value:        fmt.Sprintf("%d", cfg.Auth.Invitations.DefaultExpiryDays),
			Type:         TypeInt,
			DefaultValue: "7",
			Description:  "Default invitation expiry in days (0 = never)",
			IsEditable:   true,
			RequiresRestart: false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})
	}

	// === Database Settings (detailed) ===
	if cfg.Database.Type == "sqlite" {
		settings = append(settings, Setting{
			ID:           "database.sqlite.path",
			Category:     CategoryDatabase,
			Key:          "sqlite_path",
			Value:        cfg.Database.SQLite.Path,
			Type:         TypeString,
			DefaultValue: "./data/aigateway.db",
			Description:  "SQLite database file path",
			IsEditable:   false,
			IsRequired:   true,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "database.sqlite.cache_size",
			Category:     CategoryDatabase,
			Key:          "sqlite_cache_size",
			Value:        fmt.Sprintf("%d", cfg.Database.SQLite.CacheSize),
			Type:         TypeInt,
			DefaultValue: "10000",
			Description:  "SQLite cache size in KB",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "database.sqlite.journal_mode",
			Category:     CategoryDatabase,
			Key:          "sqlite_journal_mode",
			Value:        cfg.Database.SQLite.JournalMode,
			Type:         TypeString,
			DefaultValue: "WAL",
			Description:  "SQLite journal mode (DELETE, TRUNCATE, PERSIST, MEMORY, WAL)",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			ValidationRule: "^(DELETE|TRUNCATE|PERSIST|MEMORY|WAL)$",
			UpdatedAt:    time.Now(),
		})
	}

	if cfg.Database.Type == "postgresql" {
		settings = append(settings, Setting{
			ID:           "database.postgresql.host",
			Category:     CategoryDatabase,
			Key:          "postgresql_host",
			Value:        cfg.Database.PostgreSQL.Host,
			Type:         TypeString,
			DefaultValue: "localhost",
			Description:  "PostgreSQL host address",
			IsEditable:   false,
			IsRequired:   true,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "database.postgresql.port",
			Category:     CategoryDatabase,
			Key:          "postgresql_port",
			Value:        fmt.Sprintf("%d", cfg.Database.PostgreSQL.Port),
			Type:         TypeInt,
			DefaultValue: "5432",
			Description:  "PostgreSQL port",
			IsEditable:   false,
			IsRequired:   true,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "database.postgresql.database",
			Category:     CategoryDatabase,
			Key:          "postgresql_database",
			Value:        cfg.Database.PostgreSQL.Database,
			Type:         TypeString,
			DefaultValue: "aigateway",
			Description:  "PostgreSQL database name",
			IsEditable:   false,
			IsRequired:   true,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "database.postgresql.user",
			Category:     CategoryDatabase,
			Key:          "postgresql_user",
			Value:        cfg.Database.PostgreSQL.User,
			Type:         TypeString,
			DefaultValue: "postgres",
			Description:  "PostgreSQL username",
			IsEditable:   false,
			IsRequired:   true,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "database.postgresql.ssl_mode",
			Category:     CategoryDatabase,
			Key:          "postgresql_ssl_mode",
			Value:        cfg.Database.PostgreSQL.SSLMode,
			Type:         TypeString,
			DefaultValue: "disable",
			Description:  "PostgreSQL SSL mode (disable, require, verify-ca, verify-full)",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			ValidationRule: "^(disable|require|verify-ca|verify-full)$",
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "database.postgresql.max_open_conns",
			Category:     CategoryDatabase,
			Key:          "postgresql_max_open_conns",
			Value:        fmt.Sprintf("%d", cfg.Database.PostgreSQL.MaxOpenConns),
			Type:         TypeInt,
			DefaultValue: "25",
			Description:  "PostgreSQL maximum open connections",
			IsEditable:   true,
			RequiresRestart: false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "database.postgresql.max_idle_conns",
			Category:     CategoryDatabase,
			Key:          "postgresql_max_idle_conns",
			Value:        fmt.Sprintf("%d", cfg.Database.PostgreSQL.MaxIdleConns),
			Type:         TypeInt,
			DefaultValue: "5",
			Description:  "PostgreSQL maximum idle connections",
			IsEditable:   true,
			RequiresRestart: false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})
	}

	// === Logging Settings (extended) ===
	settings = append(settings, Setting{
		ID:           "logging.output",
		Category:     CategoryLogging,
		Key:          "output",
		Value:        cfg.Logging.Output,
		Type:         TypeString,
		DefaultValue: "stdout",
		Description:  "Log output destination (stdout, file)",
		IsEditable:   false,
		IsRequired:   false,
		IsMigrated:   true,
		ValidationRule: "^(stdout|file)$",
		UpdatedAt:    time.Now(),
	})

	if cfg.Logging.Output == "file" {
		settings = append(settings, Setting{
			ID:           "logging.file_path",
			Category:     CategoryLogging,
			Key:          "file_path",
			Value:        cfg.Logging.FilePath,
			Type:         TypeString,
			DefaultValue: "./logs/aigateway.log",
			Description:  "Log file path",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "logging.max_size",
			Category:     CategoryLogging,
			Key:          "max_size",
			Value:        fmt.Sprintf("%d", cfg.Logging.MaxSize),
			Type:         TypeInt,
			DefaultValue: "100",
			Description:  "Maximum log file size in MB",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "logging.max_backups",
			Category:     CategoryLogging,
			Key:          "max_backups",
			Value:        fmt.Sprintf("%d", cfg.Logging.MaxBackups),
			Type:         TypeInt,
			DefaultValue: "3",
			Description:  "Maximum number of old log files to retain",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "logging.max_age",
			Category:     CategoryLogging,
			Key:          "max_age",
			Value:        fmt.Sprintf("%d", cfg.Logging.MaxAge),
			Type:         TypeInt,
			DefaultValue: "28",
			Description:  "Maximum age in days to retain old log files",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})
	}

	// === Inference.Yzma (extended) ===
	if cfg.Inference.Yzma.Enabled {
		settings = append(settings, Setting{
			ID:           "inference.yzma.lib_path",
			Category:     CategoryInference,
			Key:          "yzma_lib_path",
			Value:        cfg.Inference.Yzma.LibPath,
			Type:         TypeString,
			DefaultValue: "",
			Description:  "Path to yzma shared library",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "inference.yzma.context_size",
			Category:     CategoryInference,
			Key:          "yzma_context_size",
			Value:        fmt.Sprintf("%d", cfg.Inference.Yzma.ContextSize),
			Type:         TypeInt,
			DefaultValue: "4096",
			Description:  "Yzma context window size",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "inference.yzma.batch_size",
			Category:     CategoryInference,
			Key:          "yzma_batch_size",
			Value:        fmt.Sprintf("%d", cfg.Inference.Yzma.BatchSize),
			Type:         TypeInt,
			DefaultValue: "512",
			Description:  "Yzma batch size",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "inference.yzma.ubatch_size",
			Category:     CategoryInference,
			Key:          "yzma_ubatch_size",
			Value:        fmt.Sprintf("%d", cfg.Inference.Yzma.UBatchSize),
			Type:         TypeInt,
			DefaultValue: "128",
			Description:  "Yzma micro-batch size",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "inference.yzma.temperature",
			Category:     CategoryInference,
			Key:          "yzma_temperature",
			Value:        fmt.Sprintf("%.2f", cfg.Inference.Yzma.Temperature),
			Type:         TypeFloat,
			DefaultValue: "0.8",
			Description:  "Yzma sampling temperature",
			IsEditable:   true,
			RequiresRestart: false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "inference.yzma.top_k",
			Category:     CategoryInference,
			Key:          "yzma_top_k",
			Value:        fmt.Sprintf("%d", cfg.Inference.Yzma.TopK),
			Type:         TypeInt,
			DefaultValue: "40",
			Description:  "Yzma top-k sampling",
			IsEditable:   true,
			RequiresRestart: false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "inference.yzma.top_p",
			Category:     CategoryInference,
			Key:          "yzma_top_p",
			Value:        fmt.Sprintf("%.2f", cfg.Inference.Yzma.TopP),
			Type:         TypeFloat,
			DefaultValue: "0.95",
			Description:  "Yzma top-p (nucleus) sampling",
			IsEditable:   true,
			RequiresRestart: false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "inference.yzma.min_p",
			Category:     CategoryInference,
			Key:          "yzma_min_p",
			Value:        fmt.Sprintf("%.2f", cfg.Inference.Yzma.MinP),
			Type:         TypeFloat,
			DefaultValue: "0.05",
			Description:  "Yzma minimum probability threshold",
			IsEditable:   true,
			RequiresRestart: false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "inference.yzma.gpu_layers",
			Category:     CategoryInference,
			Key:          "yzma_gpu_layers",
			Value:        fmt.Sprintf("%d", cfg.Inference.GPULayers),
			Type:         TypeInt,
			DefaultValue: "-1",
			Description:  "Number of layers to offload to GPU (-1 = auto, 0 = CPU only, >0 = specific count)",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "inference.yzma.verbose",
			Category:     CategoryInference,
			Key:          "yzma_verbose",
			Value:        fmt.Sprintf("%t", cfg.Inference.Yzma.Verbose),
			Type:         TypeBool,
			DefaultValue: "false",
			Description:  "Enable verbose Yzma logging",
			IsEditable:   true,
			RequiresRestart: false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})
	}

	// === Metrics (extended) ===
	settings = append(settings, Setting{
		ID:           "metrics.prometheus_path",
		Category:     CategoryMetrics,
		Key:          "prometheus_path",
		Value:        cfg.Metrics.PrometheusPath,
		Type:         TypeString,
		DefaultValue: "/metrics",
		Description:  "Prometheus metrics endpoint path",
		IsEditable:   false,
		IsRequired:   false,
		IsMigrated:   true,
		UpdatedAt:    time.Now(),
	})

	// === RAG Settings (extended) ===
	if cfg.RAG.Enabled {
		settings = append(settings, Setting{
			ID:           "rag.enabled",
			Category:     CategoryRAG,
			Key:          "enabled",
			Value:        "true",
			Type:         TypeBool,
			DefaultValue: "false",
			Description:  "Enable RAG (Retrieval-Augmented Generation)",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "rag.embeddings.provider",
			Category:     CategoryRAG,
			Key:          "embeddings_provider",
			Value:        cfg.RAG.Embeddings.Provider,
			Type:         TypeString,
			DefaultValue: "ollama",
			Description:  "Embeddings provider (ollama, openai)",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "rag.embeddings.model",
			Category:     CategoryRAG,
			Key:          "embeddings_model",
			Value:        cfg.RAG.Embeddings.Model,
			Type:         TypeString,
			DefaultValue: "nomic-embed-text",
			Description:  "Embeddings model name",
			IsEditable:   true,
			RequiresRestart: false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "rag.embeddings.dimensions",
			Category:     CategoryRAG,
			Key:          "embeddings_dimensions",
			Value:        fmt.Sprintf("%d", cfg.RAG.Embeddings.Dimensions),
			Type:         TypeInt,
			DefaultValue: "768",
			Description:  "Embeddings vector dimensions",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "rag.vector_store.type",
			Category:     CategoryRAG,
			Key:          "vector_store_type",
			Value:        cfg.RAG.VectorStore.Type,
			Type:         TypeString,
			DefaultValue: "pgvector",
			Description:  "Vector store type (pgvector, qdrant)",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "rag.vector_store.connection_string",
			Category:     CategoryRAG,
			Key:          "vector_store_connection",
			Value:        cfg.RAG.VectorStore.ConnectionString,
			Type:         TypeString,
			DefaultValue: "",
			Description:  "Vector store connection string",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "rag.security.encryption_key",
			Category:     CategoryRAG,
			Key:          "encryption_key",
			Value:        cfg.RAG.Security.EncryptionKey,
			Type:         TypeString,
			DefaultValue: "",
			Description:  "Encryption key for RAG credentials (32 bytes for AES-256)",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})
	}

	// === Observability Settings ===
	if cfg.Observability.Tracing.Enabled {
		settings = append(settings, Setting{
			ID:           "observability.tracing.enabled",
			Category:     CategoryObservability,
			Key:          "tracing_enabled",
			Value:        "true",
			Type:         TypeBool,
			DefaultValue: "false",
			Description:  "Enable OpenTelemetry distributed tracing",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "observability.tracing.provider",
			Category:     CategoryObservability,
			Key:          "tracing_provider",
			Value:        cfg.Observability.Tracing.Provider,
			Type:         TypeString,
			DefaultValue: "jaeger",
			Description:  "Tracing provider (jaeger, zipkin)",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			ValidationRule: "^(jaeger|zipkin)$",
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "observability.tracing.service_name",
			Category:     CategoryObservability,
			Key:          "tracing_service_name",
			Value:        cfg.Observability.Tracing.ServiceName,
			Type:         TypeString,
			DefaultValue: "aigateway",
			Description:  "Service name in traces",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "observability.tracing.sampling_rate",
			Category:     CategoryObservability,
			Key:          "tracing_sampling_rate",
			Value:        fmt.Sprintf("%.2f", cfg.Observability.Tracing.SamplingRate),
			Type:         TypeFloat,
			DefaultValue: "1.0",
			Description:  "Tracing sampling rate (0.0 - 1.0)",
			IsEditable:   true,
			RequiresRestart: false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		if cfg.Observability.Tracing.Provider == "jaeger" {
			settings = append(settings, Setting{
				ID:           "observability.tracing.jaeger.endpoint",
				Category:     CategoryObservability,
				Key:          "jaeger_endpoint",
				Value:        cfg.Observability.Tracing.Jaeger.Endpoint,
				Type:         TypeString,
				DefaultValue: "http://localhost:14268/api/traces",
				Description:  "Jaeger collector endpoint",
				IsEditable:   false,
				IsRequired:   false,
				IsMigrated:   true,
				UpdatedAt:    time.Now(),
			})
		}

		if cfg.Observability.Tracing.Provider == "zipkin" {
			settings = append(settings, Setting{
				ID:           "observability.tracing.zipkin.endpoint",
				Category:     CategoryObservability,
				Key:          "zipkin_endpoint",
				Value:        cfg.Observability.Tracing.Zipkin.Endpoint,
				Type:         TypeString,
				DefaultValue: "http://localhost:9411/api/v2/spans",
				Description:  "Zipkin collector endpoint",
				IsEditable:   false,
				IsRequired:   false,
				IsMigrated:   true,
				UpdatedAt:    time.Now(),
			})
		}
	}

	if cfg.Observability.Performance.Enabled {
		settings = append(settings, Setting{
			ID:           "observability.performance.enabled",
			Category:     CategoryObservability,
			Key:          "performance_enabled",
			Value:        "true",
			Type:         TypeBool,
			DefaultValue: "false",
			Description:  "Enable performance monitoring",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "observability.performance.collection_interval",
			Category:     CategoryObservability,
			Key:          "performance_interval",
			Value:        cfg.Observability.Performance.CollectionInterval,
			Type:         TypeDuration,
			DefaultValue: "30s",
			Description:  "Performance metrics collection interval",
			IsEditable:   true,
			RequiresRestart: false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "observability.performance.memory_threshold_mb",
			Category:     CategoryObservability,
			Key:          "performance_memory_threshold",
			Value:        fmt.Sprintf("%d", cfg.Observability.Performance.MemoryThresholdMB),
			Type:         TypeInt,
			DefaultValue: "1024",
			Description:  "Memory threshold for alerts (MB)",
			IsEditable:   true,
			RequiresRestart: false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "observability.performance.goroutine_threshold",
			Category:     CategoryObservability,
			Key:          "performance_goroutine_threshold",
			Value:        fmt.Sprintf("%d", cfg.Observability.Performance.GoroutineThreshold),
			Type:         TypeInt,
			DefaultValue: "1000",
			Description:  "Goroutine count threshold for alerts",
			IsEditable:   true,
			RequiresRestart: false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})
	}

	// === Model Registry ===
	if cfg.ModelRegistry.Enabled {
		settings = append(settings, Setting{
			ID:           "model_registry.enabled",
			Category:     CategoryModelRegistry,
			Key:          "enabled",
			Value:        "true",
			Type:         TypeBool,
			DefaultValue: "false",
			Description:  "Enable Model Registry system",
			IsEditable:   false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})

		settings = append(settings, Setting{
			ID:           "model_registry.auto_discovery.enabled",
			Category:     CategoryModelRegistry,
			Key:          "auto_discovery_enabled",
			Value:        fmt.Sprintf("%t", cfg.ModelRegistry.AutoDiscovery.Enabled),
			Type:         TypeBool,
			DefaultValue: "true",
			Description:  "Enable automatic model discovery at startup",
			IsEditable:   true,
			RequiresRestart: false,
			IsRequired:   false,
			IsMigrated:   true,
			UpdatedAt:    time.Now(),
		})
	}

	// === Development Settings ===
	settings = append(settings, Setting{
		ID:           "development.hot_reload",
		Category:     CategoryDevelopment,
		Key:          "hot_reload",
		Value:        fmt.Sprintf("%t", cfg.Development.HotReload),
		Type:         TypeBool,
		DefaultValue: "false",
		Description:  "Enable hot reload for templates in development",
		IsEditable:   false,
		IsRequired:   false,
		IsMigrated:   true,
		UpdatedAt:    time.Now(),
	})

	s.logger.WithField("mapped", len(settings)).Debug("Mapped config to settings")
	return settings
}


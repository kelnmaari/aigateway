// Package config - Database-first configuration loader with bootstrap support
// Version: v3.1.0
package config

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// LoadMode определяет режим загрузки конфигурации
type LoadMode int

const (
	// LoadModeLegacy - полная загрузка из YAML (backward compatibility)
	LoadModeLegacy LoadMode = iota
	
	// LoadModeBootstrap - минимальный bootstrap + БД
	LoadModeBootstrap
	
	// LoadModeHybrid - YAML + БД overlay (для постепенной миграции)
	LoadModeHybrid
)

// LoadOptions определяет опции загрузки конфигурации
type LoadOptions struct {
	ConfigPath     string
	Logger         *logrus.Logger
	SettingsManager SettingsManager // Опционально: для database-first режима
	Mode           LoadMode
}

// LoadWithMode загружает конфигурацию в указанном режиме
func LoadWithMode(opts LoadOptions) (*Config, ConfigSource, error) {
	if opts.Logger == nil {
		opts.Logger = logrus.New()
	}
	
	// Detect mode if not specified
	if opts.Mode == LoadModeLegacy && opts.ConfigPath != "" {
		if isBootstrapConfig(opts.ConfigPath) {
			opts.Mode = LoadModeBootstrap
			opts.Logger.Info("Detected bootstrap config, using database-first mode")
		}
	}
	
	// Load base config from YAML
	baseConfig, err := loadYAMLConfig(opts.ConfigPath, opts.Logger)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load YAML config: %w", err)
	}
	
	// Apply mode-specific logic
	switch opts.Mode {
	case LoadModeLegacy:
		// Legacy mode: just return YAML config
		opts.Logger.Info("Using legacy mode: full YAML configuration")
		return baseConfig, nil, nil
		
	case LoadModeBootstrap:
		// Bootstrap mode: minimal YAML + database
		if opts.SettingsManager == nil {
			return nil, nil, fmt.Errorf("bootstrap mode requires SettingsManager")
		}
		
		opts.Logger.Info("Using bootstrap mode: minimal YAML + database settings")
		source := NewHybridConfigSource(opts.SettingsManager, baseConfig, opts.Logger)
		return baseConfig, source, nil
		
	case LoadModeHybrid:
		// Hybrid mode: YAML + database overlay
		if opts.SettingsManager != nil {
			opts.Logger.Info("Using hybrid mode: YAML with database overlay")
			source := NewHybridConfigSource(opts.SettingsManager, baseConfig, opts.Logger)
			return baseConfig, source, nil
		}
		// Fallback to legacy if no DB
		opts.Logger.Warn("Hybrid mode requested but no SettingsManager, falling back to legacy")
		return baseConfig, nil, nil
		
	default:
		return nil, nil, fmt.Errorf("unknown load mode: %d", opts.Mode)
	}
}

// isBootstrapConfig проверяет, является ли конфиг минимальным bootstrap
func isBootstrapConfig(path string) bool {
	// Check filename
	filename := filepath.Base(path)
	if filename == "bootstrap.yaml" || filename == "bootstrap.yml" {
		return true
	}
	
	// Could also check file size or content, but filename is enough for now
	return false
}

// loadYAMLConfig загружает базовый YAML конфиг (legacy функция)
func loadYAMLConfig(path string, logger *logrus.Logger) (*Config, error) {
	v := viper.New()
	
	// Setup viper
	if path != "" {
		v.SetConfigFile(path)
	} else {
		// Auto-detect
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/aigateway")
	}
	
	// Environment variables
	v.SetEnvPrefix("AIGATEWAY")
	v.AutomaticEnv()
	
	// Read config
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Apply defaults
	setDefaults(v)
	
	// Unmarshal to struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	logger.WithField("file", v.ConfigFileUsed()).Info("Configuration loaded successfully")
	
	return &cfg, nil
}

// LoadLegacy загружает конфигурацию в legacy режиме (backward compatibility)
// Deprecated: Use LoadWithMode or LoadWithSettings instead
func LoadLegacy(configPath string) (*Config, error) {
	cfg, _, err := LoadWithMode(LoadOptions{
		ConfigPath: configPath,
		Mode:       LoadModeLegacy,
	})
	return cfg, err
}

// LoadWithSettings загружает конфигурацию с поддержкой database-first
func LoadWithSettings(configPath string, settingsManager SettingsManager, logger *logrus.Logger) (*Config, ConfigSource, error) {
	return LoadWithMode(LoadOptions{
		ConfigPath:      configPath,
		SettingsManager: settingsManager,
		Logger:          logger,
		Mode:            LoadModeHybrid, // Auto-detect в функции
	})
}

// ConfigWrapper оборачивает Config для плавной миграции
// Предоставляет как старый интерфейс (cfg.Server.Port), так и новый (cfg.GetInt)
type ConfigWrapper struct {
	*Config              // Embedded для backward compatibility
	Source  ConfigSource // Новый интерфейс
	ctx     context.Context
	logger  *logrus.Logger
}

// NewConfigWrapper создает обертку с поддержкой обоих интерфейсов
func NewConfigWrapper(cfg *Config, source ConfigSource, logger *logrus.Logger) *ConfigWrapper {
	if logger == nil {
		logger = logrus.New()
	}
	
	return &ConfigWrapper{
		Config: cfg,
		Source: source,
		ctx:    context.Background(),
		logger: logger,
	}
}

// GetString получает строковое значение (database-first если доступно)
func (w *ConfigWrapper) GetString(key string) string {
	if w.Source != nil {
		ctx, cancel := context.WithTimeout(w.ctx, 5*time.Second)
		defer cancel()
		
		if val, err := w.Source.GetString(ctx, key); err == nil {
			return val
		}
	}
	
	// Fallback to struct access
	w.logger.WithField("key", key).Debug("Using legacy struct access")
	return "" // Caller должен использовать cfg.Server.Port напрямую
}

// GetInt получает целочисленное значение
func (w *ConfigWrapper) GetInt(key string) int {
	if w.Source != nil {
		ctx, cancel := context.WithTimeout(w.ctx, 5*time.Second)
		defer cancel()
		
		if val, err := w.Source.GetInt(ctx, key); err == nil {
			return val
		}
	}
	
	return 0
}

// GetBool получает булево значение
func (w *ConfigWrapper) GetBool(key string) bool {
	if w.Source != nil {
		ctx, cancel := context.WithTimeout(w.ctx, 5*time.Second)
		defer cancel()
		
		if val, err := w.Source.GetBool(ctx, key); err == nil {
			return val
		}
	}
	
	return false
}

// GetDuration получает duration значение
func (w *ConfigWrapper) GetDuration(key string) time.Duration {
	if w.Source != nil {
		ctx, cancel := context.WithTimeout(w.ctx, 5*time.Second)
		defer cancel()
		
		if val, err := w.Source.GetDuration(ctx, key); err == nil {
			return val
		}
	}
	
	return 0
}

// IsBootstrapMode проверяет, работает ли конфиг в bootstrap режиме
func (w *ConfigWrapper) IsBootstrapMode() bool {
	return w.Source != nil
}

// Reload перезагружает конфигурацию из источника
func (w *ConfigWrapper) Reload() error {
	if w.Source != nil {
		ctx, cancel := context.WithTimeout(w.ctx, 10*time.Second)
		defer cancel()
		return w.Source.Reload(ctx)
	}
	
	w.logger.Warn("Reload called but no ConfigSource available")
	return nil
}


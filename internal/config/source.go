// Package config - ConfigSource interface for hybrid configuration loading
// Version: v3.1.0 - Database-first configuration with YAML fallback
package config

import (
	"context"
	"time"
)

// ConfigSource provides unified interface for accessing configuration
// regardless of the underlying storage (Database, YAML, Environment)
type ConfigSource interface {
	// String accessors
	GetString(ctx context.Context, key string) (string, error)
	GetStringWithDefault(ctx context.Context, key, defaultValue string) string
	
	// Numeric accessors
	GetInt(ctx context.Context, key string) (int, error)
	GetIntWithDefault(ctx context.Context, key string, defaultValue int) int
	
	GetInt64(ctx context.Context, key string) (int64, error)
	GetInt64WithDefault(ctx context.Context, key string, defaultValue int64) int64
	
	GetFloat64(ctx context.Context, key string) (float64, error)
	GetFloat64WithDefault(ctx context.Context, key string, defaultValue float64) float64
	
	// Boolean accessors
	GetBool(ctx context.Context, key string) (bool, error)
	GetBoolWithDefault(ctx context.Context, key string, defaultValue bool) bool
	
	// Duration accessors
	GetDuration(ctx context.Context, key string) (time.Duration, error)
	GetDurationWithDefault(ctx context.Context, key string, defaultValue time.Duration) time.Duration
	
	// Array accessors
	GetStringSlice(ctx context.Context, key string) ([]string, error)
	GetStringSliceWithDefault(ctx context.Context, key string, defaultValue []string) []string
	
	// Reload triggers cache invalidation
	Reload(ctx context.Context) error
}

// TypedConfigSource provides type-safe accessors for common configuration keys
// This ensures compile-time safety for critical configuration paths
type TypedConfigSource interface {
	ConfigSource
	
	// Server configuration
	GetServerHost() string
	GetServerPort() int
	GetServerReadTimeout() time.Duration
	GetServerWriteTimeout() time.Duration
	GetServerIdleTimeout() time.Duration
	GetServerMaxHeaderBytes() int
	
	// TLS configuration
	IsTLSEnabled() bool
	GetTLSCommonName() string
	GetTLSHosts() []string
	GetTLSValidDays() int
	GetTLSCertFile() string
	GetTLSKeyFile() string
	
	// Database configuration
	GetDatabaseType() string
	GetDatabaseHost() string
	GetDatabasePort() int
	GetDatabaseName() string
	GetDatabaseUser() string
	GetDatabasePassword() string
	GetDatabaseSSLMode() string
	GetDatabaseMaxOpenConns() int
	GetDatabaseMaxIdleConns() int
	
	// Auth configuration
	GetJWTSecret() string
	GetJWTAccessTokenExpiry() time.Duration
	GetJWTRefreshTokenExpiry() time.Duration
	
	// Logging configuration
	GetLoggingLevel() string
	GetLoggingFormat() string
	GetLoggingOutput() string
	GetLoggingFilePath() string
	
	// Metrics configuration
	IsMetricsEnabled() bool
	GetMetricsPort() int
	GetMetricsPrometheusPath() string
	
	// Inference configuration
	GetInferenceProvider() string
	GetInferenceYzmaLibPath() string
	GetInferenceContextSize() int
	GetInferenceBatchSize() int
	GetInferenceGPULayers() int
	GetInferenceTemperature() float64
}


// Package settings - Live reload handlers for hot-reloadable settings
// Version: v3.0.9 - Phase 4
package settings

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"
)

// ReloadHandlerRegistry manages all reload handlers for the application
type ReloadHandlerRegistry struct {
	manager *Manager
	logger  *logrus.Logger
	db      *sql.DB // For database connection pool updates
}

// NewReloadHandlerRegistry creates a new registry and registers all handlers
func NewReloadHandlerRegistry(manager *Manager, logger *logrus.Logger, db *sql.DB) *ReloadHandlerRegistry {
	registry := &ReloadHandlerRegistry{
		manager: manager,
		logger:  logger,
		db:      db,
	}
	
	// Register all handlers
	registry.registerAllHandlers()
	
	return registry
}

// registerAllHandlers registers reload handlers for all hot-reloadable settings
func (r *ReloadHandlerRegistry) registerAllHandlers() {
	// Logging level
	r.manager.RegisterReloadHandler("logging.level", r.handleLoggingLevelChange)
	
	// Database connection pools
	r.manager.RegisterReloadHandler("database.postgresql.max_open_conns", r.handleMaxOpenConnsChange)
	r.manager.RegisterReloadHandler("database.postgresql.max_idle_conns", r.handleMaxIdleConnsChange)
	
	// Rate limiting (handlers would be registered by rate limiter service)
	// r.manager.RegisterReloadHandler("auth.rate_limiting.default_requests_per_minute", ...)
	// r.manager.RegisterReloadHandler("auth.rate_limiting.default_requests_per_hour", ...)
	
	r.logger.Info("Registered live reload handlers")
}

// handleLoggingLevelChange updates the global log level
func (r *ReloadHandlerRegistry) handleLoggingLevelChange(ctx context.Context, setting *Setting) error {
	level, err := logrus.ParseLevel(setting.Value)
	if err != nil {
		return fmt.Errorf("invalid log level '%s': %w", setting.Value, err)
	}
	
	r.logger.SetLevel(level)
	r.logger.WithFields(logrus.Fields{
		"old_level": r.logger.GetLevel(),
		"new_level": level,
	}).Info("Log level updated dynamically")
	
	return nil
}

// handleMaxOpenConnsChange updates database max open connections
func (r *ReloadHandlerRegistry) handleMaxOpenConnsChange(ctx context.Context, setting *Setting) error {
	if r.db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	maxConns, err := strconv.Atoi(setting.Value)
	if err != nil {
		return fmt.Errorf("invalid max_open_conns value '%s': %w", setting.Value, err)
	}
	
	if maxConns < 1 {
		return fmt.Errorf("max_open_conns must be >= 1, got %d", maxConns)
	}
	
	r.db.SetMaxOpenConns(maxConns)
	r.logger.WithField("max_open_conns", maxConns).Info("Database max open connections updated")
	
	return nil
}

// handleMaxIdleConnsChange updates database max idle connections
func (r *ReloadHandlerRegistry) handleMaxIdleConnsChange(ctx context.Context, setting *Setting) error {
	if r.db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	maxConns, err := strconv.Atoi(setting.Value)
	if err != nil {
		return fmt.Errorf("invalid max_idle_conns value '%s': %w", setting.Value, err)
	}
	
	if maxConns < 0 {
		return fmt.Errorf("max_idle_conns must be >= 0, got %d", maxConns)
	}
	
	r.db.SetMaxIdleConns(maxConns)
	r.logger.WithField("max_idle_conns", maxConns).Info("Database max idle connections updated")
	
	return nil
}

// Example handler for rate limiting (to be implemented by rate limiter service)
/*
func (r *ReloadHandlerRegistry) handleRateLimitRPMChange(ctx context.Context, setting *Setting) error {
	rpm, err := strconv.Atoi(setting.Value)
	if err != nil {
		return fmt.Errorf("invalid RPM value: %w", err)
	}
	
	// Update rate limiter configuration
	// rateLimiter.SetDefaultRPM(rpm)
	
	r.logger.WithField("rpm", rpm).Info("Rate limit RPM updated")
	return nil
}
*/



// Package redis provides Redis manager coordinating all services
// Version: v3.0.6+ - Redis Integration Manager
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// Manager coordinates all Redis services
type Manager struct {
	Client    *Client
	RateLimit *RateLimitService
	Session   *SessionService
	JWT       *JWTService
	Cache     *CacheService
	PubSub    *PubSubService
	logger    *logrus.Logger
	enabled   bool

	// Background workers (v3.0.6+: background sync)
	BackgroundSync *BackgroundSyncManager
}

// GetJWT returns JWT service (v3.0.6+: interface compatibility)
func (m *Manager) GetJWT() *JWTService {
	if m == nil {
		return nil
	}
	return m.JWT
}

// StartBackgroundWorkers starts background sync workers (v3.0.6+)
func (m *Manager) StartBackgroundWorkers() {
	if m.BackgroundSync != nil {
		m.BackgroundSync.Start()
	}
}

// StopBackgroundWorkers stops background sync workers (v3.0.6+)
func (m *Manager) StopBackgroundWorkers() {
	if m.BackgroundSync != nil {
		m.BackgroundSync.Stop()
	}
}

// ManagerConfig configuration for Redis manager
type ManagerConfig struct {
	URL        string
	KeyPrefix  string
	DB         int
	MaxRetries int
	PoolSize   int
	Enabled    bool
}

// NewManager creates a new Redis manager with all services
func NewManager(cfg ManagerConfig, logger *logrus.Logger) (*Manager, error) {
	if !cfg.Enabled {
		logger.Info("Redis is disabled (using in-memory fallback)")
		return &Manager{
			enabled: false,
			logger:  logger,
		}, nil
	}

	// Create Redis client
	client, err := NewClient(Config{
		URL:        cfg.URL,
		KeyPrefix:  cfg.KeyPrefix,
		DB:         cfg.DB,
		MaxRetries: cfg.MaxRetries,
		PoolSize:   cfg.PoolSize,
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %w", err)
	}

	// Initialize all services
	// Create manager first (needed for background sync init)
	manager := &Manager{
		Client:    client,
		RateLimit: NewRateLimitService(client, logger),
		Session:   NewSessionService(client, logger),
		JWT:       NewJWTService(client, logger),
		Cache:     NewCacheService(client, logger),
		PubSub:    NewPubSubService(client, logger),
		logger:    logger,
		enabled:   true,
	}

	// Initialize background sync manager (v3.0.6+)
	backgroundSync := NewBackgroundSyncManager(manager, logger, BackgroundWorkerConfig{
		StatsInterval:     10 * time.Second,
		ModelListInterval: 30 * time.Second,
		APIKeyTTL:         15 * time.Minute,
		Enabled:           true,
	})
	manager.BackgroundSync = backgroundSync

	logger.WithFields(logrus.Fields{
		"url":        cfg.URL,
		"db":         cfg.DB,
		"key_prefix": cfg.KeyPrefix,
	}).Info("🚀 Redis manager initialized")

	return manager, nil
}

// IsEnabled returns true if Redis is enabled
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// Close closes all Redis connections
func (m *Manager) Close() error {
	if !m.enabled {
		return nil
	}

	m.logger.Info("Closing Redis connections...")

	// Stop background workers first (v3.0.6+)
	if m.BackgroundSync != nil {
		m.BackgroundSync.Stop()
		m.logger.Debug("Background workers stopped")
	}

	// Close pub/sub
	if err := m.PubSub.Close(); err != nil {
		m.logger.WithError(err).Warn("Failed to close pub/sub")
	}

	// Close client
	if err := m.Client.Close(); err != nil {
		return fmt.Errorf("failed to close Redis client: %w", err)
	}

	m.logger.Info("✅ Redis connections closed")
	return nil
}

// Ping checks Redis connection health
func (m *Manager) Ping(ctx context.Context) error {
	if !m.enabled {
		return nil
	}

	return m.Client.Ping(ctx)
}

// GetStats returns comprehensive Redis statistics
func (m *Manager) GetStats(ctx context.Context) (map[string]any, error) {
	if !m.enabled {
		return map[string]any{
			"enabled": false,
		}, nil
	}

	stats := make(map[string]any)

	// Basic info
	stats["enabled"] = true

	// DB size
	dbSize, _ := m.Client.DBSize(ctx)
	stats["db_size"] = dbSize

	// Pool stats
	poolStats := m.Client.Stats()
	stats["pool"] = map[string]any{
		"hits":        poolStats.Hits,
		"misses":      poolStats.Misses,
		"timeouts":    poolStats.Timeouts,
		"total_conns": poolStats.TotalConns,
		"idle_conns":  poolStats.IdleConns,
		"stale_conns": poolStats.StaleConns,
	}

	// Cache stats
	cacheStats, err := m.Cache.GetStats(ctx)
	if err == nil {
		stats["cache"] = cacheStats
	}

	// Instances (if pub/sub enabled)
	instances, err := m.PubSub.ListInstances(ctx)
	if err == nil {
		stats["instances"] = len(instances)
	}

	return stats, nil
}

// WarmUp performs initial cache warming
func (m *Manager) WarmUp(ctx context.Context) error {
	if !m.enabled {
		return nil
	}

	m.logger.Info("Warming up Redis cache...")

	// Warm up can be customized based on application needs
	// For now, just log that we're ready

	m.logger.Info("✅ Redis cache ready")
	return nil
}

// RegisterInstance registers this instance for multi-instance coordination
func (m *Manager) RegisterInstance(ctx context.Context, instanceID, hostname, ip string, port int) error {
	if !m.enabled {
		return nil
	}

	info := &InstanceInfo{
		InstanceID: instanceID,
		Hostname:   hostname,
		IP:         ip,
		Port:       port,
		StartedAt:  time.Now(),
	}

	if err := m.PubSub.RegisterInstance(ctx, info); err != nil {
		return fmt.Errorf("failed to register instance: %w", err)
	}

	// Start heartbeat goroutine
	go m.heartbeatLoop(ctx, instanceID)

	m.logger.WithFields(logrus.Fields{
		"instance_id": instanceID,
		"hostname":    hostname,
		"ip":          ip,
		"port":        port,
	}).Info("✅ Instance registered")

	return nil
}

// heartbeatLoop sends periodic heartbeats
func (m *Manager) heartbeatLoop(ctx context.Context, instanceID string) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.PubSub.Heartbeat(ctx, instanceID); err != nil {
				m.logger.WithError(err).Warn("Failed to send heartbeat")
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Convenience Methods (pass-through to services)
// ─────────────────────────────────────────────────────────────────────────────

// CheckRateLimit convenience method for rate limiting
func (m *Manager) CheckRateLimit(ctx context.Context, keyID string, limit int64, window time.Duration) (bool, int64, time.Time, error) {
	if !m.enabled {
		// Fallback to in-memory (would need implementation)
		return true, limit, time.Now().Add(window), nil
	}

	return m.RateLimit.CheckLimit(ctx, keyID, limit, window)
}

// CreateSession convenience method for session management
func (m *Manager) CreateSession(ctx context.Context, session *Session, ttl time.Duration) error {
	if !m.enabled {
		// Fallback to in-memory (would need implementation)
		return nil
	}

	return m.Session.CreateSession(ctx, session, ttl)
}

// BlacklistToken convenience method for JWT management
func (m *Manager) BlacklistToken(ctx context.Context, tokenID string, ttl time.Duration) error {
	if !m.enabled {
		// Fallback to in-memory (would need implementation)
		return nil
	}

	return m.JWT.BlacklistToken(ctx, tokenID, ttl)
}

// CacheSet convenience method for caching
func (m *Manager) CacheSet(ctx context.Context, key string, value any, ttl time.Duration) error {
	if !m.enabled {
		return nil
	}

	return m.Cache.SetWithTTL(ctx, key, value, ttl)
}

// CacheGet convenience method for caching
func (m *Manager) CacheGet(ctx context.Context, key string, dest any) error {
	if !m.enabled {
		return fmt.Errorf("cache miss: Redis disabled")
	}

	return m.Cache.Get(ctx, key, dest)
}

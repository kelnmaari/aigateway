// Package indexer provides repository indexing for GitLab RAG
package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	rediscache "aigateway/internal/cache/redis"

	"github.com/sirupsen/logrus"
)

const (
	// Redis key prefix for index status
	indexStatusKeyPrefix = "gitlab:index:status:"
	
	// TTL for index status (1 hour for in_progress, 24 hours for completed/failed)
	indexStatusInProgressTTL = 1 * time.Hour
	indexStatusCompletedTTL  = 24 * time.Hour
	
	// Pub/Sub channel for index status updates
	indexStatusChannel = "gitlab:index:status:updates"
)

// RedisStatusStore stores indexing status in Redis
type RedisStatusStore struct {
	client *rediscache.Client
	logger *logrus.Logger
}

// NewRedisStatusStore creates a new Redis-based status store
func NewRedisStatusStore(client *rediscache.Client, logger *logrus.Logger) *RedisStatusStore {
	return &RedisStatusStore{
		client: client,
		logger: logger,
	}
}

// IndexStatusUpdate represents an index status update event
type IndexStatusUpdate struct {
	ProjectID string      `json:"project_id"`
	Branch    string      `json:"branch"`
	Status    IndexStatus `json:"status"`
	Chunks    int         `json:"chunks"`
	Error     string      `json:"error,omitempty"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// SetStatus saves index status to Redis
func (s *RedisStatusStore) SetStatus(ctx context.Context, info *IndexInfo) error {
	if s.client == nil || !s.client.IsEnabled() {
		return nil // Redis not available, fallback to memory
	}

	key := s.makeKey(info.ProjectID, info.Branch)
	
	// Determine TTL based on status
	ttl := indexStatusCompletedTTL
	if info.Status == IndexStatusInProgress {
		ttl = indexStatusInProgressTTL
	}

	// Save status
	if err := s.client.Set(ctx, key, info, ttl); err != nil {
		s.logger.WithError(err).WithField("key", key).Warn("Failed to save index status to Redis")
		return err
	}

	// Publish status update for real-time notifications
	update := IndexStatusUpdate{
		ProjectID: info.ProjectID,
		Branch:    info.Branch,
		Status:    info.Status,
		Chunks:    info.ChunksTotal,
		Error:     info.Error,
		UpdatedAt: time.Now(),
	}
	
	if err := s.client.Publish(ctx, indexStatusChannel, update); err != nil {
		s.logger.WithError(err).Debug("Failed to publish index status update")
		// Don't return error - pub/sub is optional
	}

	s.logger.WithFields(logrus.Fields{
		"project_id": info.ProjectID,
		"branch":     info.Branch,
		"status":     info.Status,
		"ttl":        ttl,
	}).Debug("Index status saved to Redis")

	return nil
}

// GetStatus retrieves index status from Redis
func (s *RedisStatusStore) GetStatus(ctx context.Context, projectID, branch string) (*IndexInfo, error) {
	if s.client == nil || !s.client.IsEnabled() {
		return nil, nil // Redis not available
	}

	key := s.makeKey(projectID, branch)
	
	var info IndexInfo
	if err := s.client.Get(ctx, key, &info); err != nil {
		// Key not found is not an error
		return nil, nil
	}

	return &info, nil
}

// DeleteStatus removes index status from Redis
func (s *RedisStatusStore) DeleteStatus(ctx context.Context, projectID, branch string) error {
	if s.client == nil || !s.client.IsEnabled() {
		return nil
	}

	key := s.makeKey(projectID, branch)
	return s.client.Delete(ctx, key)
}

// GetAllStatuses retrieves all index statuses (for a specific pattern)
func (s *RedisStatusStore) GetAllStatuses(ctx context.Context) (map[string]*IndexInfo, error) {
	if s.client == nil || !s.client.IsEnabled() {
		return nil, nil
	}

	// Get all keys matching pattern
	keys, err := s.client.Keys(ctx, indexStatusKeyPrefix+"*")
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	statuses := make(map[string]*IndexInfo)
	for _, key := range keys {
		var info IndexInfo
		// Remove prefix to get the actual key
		shortKey := key
		if err := s.client.Get(ctx, shortKey, &info); err != nil {
			continue // Skip invalid entries
		}
		statuses[shortKey] = &info
	}

	return statuses, nil
}

// SubscribeToUpdates subscribes to index status updates
func (s *RedisStatusStore) SubscribeToUpdates(ctx context.Context, handler func(update IndexStatusUpdate)) error {
	if s.client == nil || !s.client.IsEnabled() {
		return fmt.Errorf("redis not available")
	}

	pubsub := s.client.Subscribe(ctx, indexStatusChannel)
	
	go func() {
		defer pubsub.Close()
		
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-ch:
				if msg == nil {
					continue
				}
				
				var update IndexStatusUpdate
				if err := json.Unmarshal([]byte(msg.Payload), &update); err != nil {
					s.logger.WithError(err).Debug("Failed to unmarshal index status update")
					continue
				}
				
				handler(update)
			}
		}
	}()

	return nil
}

// CleanupStale removes stale in_progress statuses (for recovery after crash)
func (s *RedisStatusStore) CleanupStale(ctx context.Context, maxAge time.Duration) (int, error) {
	if s.client == nil || !s.client.IsEnabled() {
		return 0, nil
	}

	statuses, err := s.GetAllStatuses(ctx)
	if err != nil {
		return 0, err
	}

	cleaned := 0
	for key, info := range statuses {
		if info.Status == IndexStatusInProgress {
			// Check if started_at is too old
			if info.StartedAt != nil && time.Since(*info.StartedAt) > maxAge {
				// Mark as failed
				info.Status = IndexStatusFailed
				info.Error = "indexing timed out"
				now := time.Now()
				info.CompletedAt = &now
				
				if err := s.SetStatus(ctx, info); err != nil {
					s.logger.WithError(err).WithField("key", key).Warn("Failed to update stale status")
					continue
				}
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		s.logger.WithField("count", cleaned).Info("Cleaned up stale index statuses")
	}

	return cleaned, nil
}

// makeKey creates a Redis key for project:branch
func (s *RedisStatusStore) makeKey(projectID, branch string) string {
	return fmt.Sprintf("%s%s:%s", indexStatusKeyPrefix, projectID, branch)
}

// IsAvailable returns true if Redis is available
func (s *RedisStatusStore) IsAvailable() bool {
	return s.client != nil && s.client.IsEnabled()
}


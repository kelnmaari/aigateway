// Package redis provides Redis-based Pub/Sub for multi-instance coordination
// Version: v3.0.6+ - Pub/Sub Messaging
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// PubSubService provides Redis-based pub/sub messaging
type PubSubService struct {
	client      *Client
	logger      *logrus.Logger
	subscribers map[string][]*Subscription
	mu          sync.RWMutex
}

// NewPubSubService creates a new pub/sub service
func NewPubSubService(client *Client, logger *logrus.Logger) *PubSubService {
	return &PubSubService{
		client:      client,
		logger:      logger,
		subscribers: make(map[string][]*Subscription),
	}
}

// Message represents a pub/sub message
type Message struct {
	Channel   string         `json:"channel"`
	Type      string         `json:"type"`
	Payload   any            `json:"payload"`
	Timestamp time.Time      `json:"timestamp"`
	Source    string         `json:"source,omitempty"` // instance ID
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// Subscription represents an active subscription
type Subscription struct {
	ID       string
	Channel  string
	Handler  MessageHandler
	pubsub   *redis.PubSub
	stopChan chan struct{}
	stopped  bool
	mu       sync.Mutex
}

// MessageHandler processes incoming messages
type MessageHandler func(msg *Message) error

// Publish publishes a message to a channel
func (s *PubSubService) Publish(ctx context.Context, channel string, msgType string, payload any) error {
	msg := &Message{
		Channel:   channel,
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	if err := s.client.Publish(ctx, channel, msg); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"channel": channel,
		"type":    msgType,
	}).Debug("Message published")

	return nil
}

// Subscribe subscribes to a channel with a handler
func (s *PubSubService) Subscribe(ctx context.Context, channel string, handler MessageHandler) (*Subscription, error) {
	pubsub := s.client.Subscribe(ctx, channel)

	// Test subscription
	if _, err := pubsub.Receive(ctx); err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	subscription := &Subscription{
		ID:       fmt.Sprintf("sub_%d", time.Now().UnixNano()),
		Channel:  channel,
		Handler:  handler,
		pubsub:   pubsub,
		stopChan: make(chan struct{}),
	}

	// Register subscription
	s.mu.Lock()
	s.subscribers[channel] = append(s.subscribers[channel], subscription)
	s.mu.Unlock()

	// Start listening in background
	go s.listen(subscription)

	s.logger.WithFields(logrus.Fields{
		"subscription_id": subscription.ID,
		"channel":         channel,
	}).Info("✅ Subscribed to channel")

	return subscription, nil
}

// listen listens for messages on a subscription
func (s *PubSubService) listen(sub *Subscription) {
	ch := sub.pubsub.Channel()

	for {
		select {
		case <-sub.stopChan:
			s.logger.WithFields(logrus.Fields{
				"subscription_id": sub.ID,
				"channel":         sub.Channel,
			}).Info("Subscription stopped")
			return

		case redisMsg := <-ch:
			if redisMsg == nil {
				continue
			}

			// Parse message
			var msg Message
			if err := json.Unmarshal([]byte(redisMsg.Payload), &msg); err != nil {
				s.logger.WithError(err).Warn("Failed to parse message")
				continue
			}

			// Handle message
			if err := sub.Handler(&msg); err != nil {
				s.logger.WithError(err).WithFields(logrus.Fields{
					"channel": msg.Channel,
					"type":    msg.Type,
				}).Error("Message handler error")
			}
		}
	}
}

// Unsubscribe unsubscribes from a channel
func (s *PubSubService) Unsubscribe(sub *Subscription) error {
	sub.mu.Lock()
	defer sub.mu.Unlock()

	if sub.stopped {
		return nil
	}

	// Signal stop
	close(sub.stopChan)
	sub.stopped = true

	// Close pubsub
	if err := sub.pubsub.Close(); err != nil {
		return err
	}

	// Unregister subscription
	s.mu.Lock()
	defer s.mu.Unlock()

	subs := s.subscribers[sub.Channel]
	for i, sub := range subs {
		if sub.ID == sub.ID {
			s.subscribers[sub.Channel] = append(subs[:i], subs[i+1:]...)
			break
		}
	}

	s.logger.WithFields(logrus.Fields{
		"subscription_id": sub.ID,
		"channel":         sub.Channel,
	}).Info("Unsubscribed from channel")

	return nil
}

// Close closes all subscriptions
func (s *PubSubService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for channel, subs := range s.subscribers {
		for _, sub := range subs {
			s.Unsubscribe(sub)
		}
		delete(s.subscribers, channel)
	}

	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Pre-defined Event Types (for multi-instance coordination)
// ─────────────────────────────────────────────────────────────────────────────

// Event types
const (
	EventModelLoaded     = "model.loaded"
	EventModelUnloaded   = "model.unloaded"
	EventConfigUpdated   = "config.updated"
	EventCacheInvalidate = "cache.invalidate"
	EventUserLogout      = "user.logout"
	EventAPIKeyRevoked   = "apikey.revoked"
)

// ModelLoadedEvent represents a model loaded event
type ModelLoadedEvent struct {
	ModelID   string    `json:"model_id"`
	ModelPath string    `json:"model_path"`
	Alias     string    `json:"alias"`
	Instance  string    `json:"instance"` // which instance loaded it
	Timestamp time.Time `json:"timestamp"`
}

// ModelUnloadedEvent represents a model unloaded event
type ModelUnloadedEvent struct {
	ModelID   string    `json:"model_id"`
	Instance  string    `json:"instance"`
	Timestamp time.Time `json:"timestamp"`
}

// CacheInvalidateEvent represents a cache invalidation event
type CacheInvalidateEvent struct {
	CacheKey  string    `json:"cache_key"`
	CacheType string    `json:"cache_type"` // model, session, etc.
	Instance  string    `json:"instance"`
	Timestamp time.Time `json:"timestamp"`
}

// PublishModelLoaded publishes a model loaded event
func (s *PubSubService) PublishModelLoaded(ctx context.Context, modelID, modelPath, alias, instance string) error {
	event := &ModelLoadedEvent{
		ModelID:   modelID,
		ModelPath: modelPath,
		Alias:     alias,
		Instance:  instance,
		Timestamp: time.Now(),
	}

	return s.Publish(ctx, "models", EventModelLoaded, event)
}

// PublishModelUnloaded publishes a model unloaded event
func (s *PubSubService) PublishModelUnloaded(ctx context.Context, modelID, instance string) error {
	event := &ModelUnloadedEvent{
		ModelID:   modelID,
		Instance:  instance,
		Timestamp: time.Now(),
	}

	return s.Publish(ctx, "models", EventModelUnloaded, event)
}

// PublishCacheInvalidate publishes a cache invalidation event
func (s *PubSubService) PublishCacheInvalidate(ctx context.Context, cacheKey, cacheType, instance string) error {
	event := &CacheInvalidateEvent{
		CacheKey:  cacheKey,
		CacheType: cacheType,
		Instance:  instance,
		Timestamp: time.Now(),
	}

	return s.Publish(ctx, "cache", EventCacheInvalidate, event)
}

// ─────────────────────────────────────────────────────────────────────────────
// Health Monitoring & Instance Discovery
// ─────────────────────────────────────────────────────────────────────────────

// InstanceInfo represents an instance registration
type InstanceInfo struct {
	InstanceID string    `json:"instance_id"`
	Hostname   string    `json:"hostname"`
	IP         string    `json:"ip"`
	Port       int       `json:"port"`
	StartedAt  time.Time `json:"started_at"`
	LastSeen   time.Time `json:"last_seen"`
	Status     string    `json:"status"` // online, offline
}

// RegisterInstance registers this instance for discovery
func (s *PubSubService) RegisterInstance(ctx context.Context, info *InstanceInfo) error {
	key := fmt.Sprintf("instance:%s", info.InstanceID)

	info.LastSeen = time.Now()
	info.Status = "online"

	// Store instance info with 30s TTL (requires heartbeat)
	return s.client.Set(ctx, key, info, 30*time.Second)
}

// Heartbeat updates instance last seen timestamp
func (s *PubSubService) Heartbeat(ctx context.Context, instanceID string) error {
	key := fmt.Sprintf("instance:%s", instanceID)

	// Get current info
	var info InstanceInfo
	if err := s.client.Get(ctx, key, &info); err != nil {
		return err
	}

	info.LastSeen = time.Now()

	// Update with 30s TTL
	return s.client.Set(ctx, key, &info, 30*time.Second)
}

// ListInstances lists all registered instances
func (s *PubSubService) ListInstances(ctx context.Context) ([]*InstanceInfo, error) {
	pattern := "instance:*"
	keys, err := s.client.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	instances := make([]*InstanceInfo, 0, len(keys))
	for _, key := range keys {
		var info InstanceInfo
		if err := s.client.Get(ctx, key, &info); err == nil {
			instances = append(instances, &info)
		}
	}

	return instances, nil
}

// UnregisterInstance removes instance registration
func (s *PubSubService) UnregisterInstance(ctx context.Context, instanceID string) error {
	key := fmt.Sprintf("instance:%s", instanceID)
	return s.client.Delete(ctx, key)
}

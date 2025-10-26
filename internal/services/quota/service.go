// Package quota provides quota checking and usage tracking service
// Version: 1.11.7+ (Enterprise Suite - Usage Quotas System)
package quota

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// Service provides quota checking and usage tracking
type Service struct {
	db     storage.Database
	logger *logrus.Logger
	mu     sync.RWMutex
}

// NewService creates a new quota service
func NewService(db storage.Database, logger *logrus.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger,
	}
}

// CheckQuota проверяет может ли user/tenant сделать запрос с заданными параметрами
func (s *Service) CheckQuota(
	ctx context.Context,
	userID string,
	tenantID *string,
	estimatedTokens int64,
	model string,
) (*models.QuotaCheck, error) {
	// Определяем scope и targetID
	scope, targetID := s.getQuotaTarget(userID, tenantID)

	// Получаем quota и usage
	quota, usage, err := s.db.GetQuotaWithUsage(ctx, scope, targetID)
	if err != nil {
		// No quota = unlimited access
		return &models.QuotaCheck{
			Allowed:     true,
			QuotaExists: false,
		}, nil
	}

	// Auto-reset если необходимо
	if err := s.autoReset(ctx, usage); err != nil {
		s.logger.WithError(err).Warn("Failed to auto-reset quota usage")
	}

	// Check if quota is enabled
	if !quota.Enabled {
		return &models.QuotaCheck{
			Allowed:     true,
			QuotaExists: true,
			Message:     "quota disabled",
		}, nil
	}

	// Check model restrictions
	if len(quota.AllowedModels) > 0 {
		modelAllowed := false
		for _, allowedModel := range quota.AllowedModels {
			if allowedModel == model || allowedModel == "*" {
				modelAllowed = true
				break
			}
		}
		if !modelAllowed {
			return &models.QuotaCheck{
				Allowed:     false,
				QuotaExists: true,
				ErrorType:   "model_not_allowed",
				Message:     fmt.Sprintf("model '%s' not allowed by quota", model),
			}, nil
		}
	}

	// Check daily token limit
	if quota.TokensPerDay != nil {
		if usage.TokensUsedToday+estimatedTokens > *quota.TokensPerDay {
			return &models.QuotaCheck{
				Allowed:     false,
				QuotaExists: true,
				ErrorType:   "daily_tokens",
				Limit:       *quota.TokensPerDay,
				Used:        usage.TokensUsedToday,
				Remaining:   *quota.TokensPerDay - usage.TokensUsedToday,
				Message:     fmt.Sprintf("daily token quota exceeded: %d/%d used", usage.TokensUsedToday, *quota.TokensPerDay),
			}, nil
		}
	}

	// Check monthly token limit
	if quota.TokensPerMonth != nil {
		if usage.TokensUsedMonth+estimatedTokens > *quota.TokensPerMonth {
			return &models.QuotaCheck{
				Allowed:     false,
				QuotaExists: true,
				ErrorType:   "monthly_tokens",
				Limit:       *quota.TokensPerMonth,
				Used:        usage.TokensUsedMonth,
				Remaining:   *quota.TokensPerMonth - usage.TokensUsedMonth,
				Message:     fmt.Sprintf("monthly token quota exceeded: %d/%d used", usage.TokensUsedMonth, *quota.TokensPerMonth),
			}, nil
		}
	}

	// Check daily request limit
	if quota.RequestsPerDay != nil {
		if usage.RequestsToday+1 > *quota.RequestsPerDay {
			return &models.QuotaCheck{
				Allowed:     false,
				QuotaExists: true,
				ErrorType:   "daily_requests",
				Limit:       *quota.RequestsPerDay,
				Used:        usage.RequestsToday,
				Remaining:   *quota.RequestsPerDay - usage.RequestsToday,
				Message:     fmt.Sprintf("daily request quota exceeded: %d/%d used", usage.RequestsToday, *quota.RequestsPerDay),
			}, nil
		}
	}

	// Check monthly request limit
	if quota.RequestsPerMonth != nil {
		if usage.RequestsMonth+1 > *quota.RequestsPerMonth {
			return &models.QuotaCheck{
				Allowed:     false,
				QuotaExists: true,
				ErrorType:   "monthly_requests",
				Limit:       *quota.RequestsPerMonth,
				Used:        usage.RequestsMonth,
				Remaining:   *quota.RequestsPerMonth - usage.RequestsMonth,
				Message:     fmt.Sprintf("monthly request quota exceeded: %d/%d used", usage.RequestsMonth, *quota.RequestsPerMonth),
			}, nil
		}
	}

	// Check concurrent request limit
	if quota.MaxConcurrent != nil {
		if usage.CurrentConcurrent >= *quota.MaxConcurrent {
			return &models.QuotaCheck{
				Allowed:     false,
				QuotaExists: true,
				ErrorType:   "concurrent",
				Limit:       int64(*quota.MaxConcurrent),
				Used:        int64(usage.CurrentConcurrent),
				Remaining:   int64(*quota.MaxConcurrent - usage.CurrentConcurrent),
				Message:     fmt.Sprintf("concurrent request limit reached: %d/%d", usage.CurrentConcurrent, *quota.MaxConcurrent),
			}, nil
		}
	}

	// All checks passed
	return &models.QuotaCheck{
		Allowed:     true,
		QuotaExists: true,
	}, nil
}

// RecordUsage записывает фактическое использование после запроса
func (s *Service) RecordUsage(
	ctx context.Context,
	userID string,
	tenantID *string,
	promptTokens int64,
	completionTokens int64,
) error {
	scope, targetID := s.getQuotaTarget(userID, tenantID)

	// Get quota and usage
	quota, usage, err := s.db.GetQuotaWithUsage(ctx, scope, targetID)
	if err != nil {
		// No quota = no tracking needed
		return nil
	}

	if !quota.Enabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	totalTokens := promptTokens + completionTokens

	// Update usage counters
	usage.TokensUsedToday += totalTokens
	usage.TokensUsedMonth += totalTokens
	usage.RequestsToday += 1
	usage.RequestsMonth += 1

	return s.db.UpdateQuotaUsage(ctx, usage)
}

// IncrementConcurrent увеличивает счетчик concurrent requests
func (s *Service) IncrementConcurrent(ctx context.Context, userID string, tenantID *string) error {
	_, targetID := s.getQuotaTarget(userID, tenantID)

	usage, err := s.db.GetQuotaUsageByTarget(ctx, targetID)
	if err != nil {
		// No quota = no tracking
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	usage.CurrentConcurrent += 1
	return s.db.UpdateQuotaUsage(ctx, usage)
}

// DecrementConcurrent уменьшает счетчик concurrent requests
func (s *Service) DecrementConcurrent(ctx context.Context, userID string, tenantID *string) error {
	_, targetID := s.getQuotaTarget(userID, tenantID)

	usage, err := s.db.GetQuotaUsageByTarget(ctx, targetID)
	if err != nil {
		// No quota = no tracking
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if usage.CurrentConcurrent > 0 {
		usage.CurrentConcurrent -= 1
	}
	return s.db.UpdateQuotaUsage(ctx, usage)
}

// GetQuotaStats возвращает статистику использования квоты (для UI)
func (s *Service) GetQuotaStats(ctx context.Context, userID string, tenantID *string) (*models.QuotaStats, error) {
	scope, targetID := s.getQuotaTarget(userID, tenantID)

	quota, usage, err := s.db.GetQuotaWithUsage(ctx, scope, targetID)
	if err != nil {
		return nil, fmt.Errorf("quota not found")
	}

	stats := &models.QuotaStats{
		QuotaID:            quota.ID,
		Scope:              quota.Scope,
		TargetID:           quota.TargetID,
		TokensUsedToday:    usage.TokensUsedToday,
		TokensUsedMonth:    usage.TokensUsedMonth,
		RequestsToday:      usage.RequestsToday,
		RequestsMonth:      usage.RequestsMonth,
		CurrentConcurrent:  usage.CurrentConcurrent,
		StorageUsedBytes:   usage.StorageUsedBytes,
		LastDailyReset:     usage.LastDailyReset,
		LastMonthlyReset:   usage.LastMonthlyReset,
	}

	// Calculate percentages if limits exist
	if quota.TokensPerDay != nil {
		stats.TokensPerDay = quota.TokensPerDay
		stats.TokensPercentDay = float64(usage.TokensUsedToday) / float64(*quota.TokensPerDay) * 100
	}

	if quota.TokensPerMonth != nil {
		stats.TokensPerMonth = quota.TokensPerMonth
		stats.TokensPercentMonth = float64(usage.TokensUsedMonth) / float64(*quota.TokensPerMonth) * 100
	}

	if quota.RequestsPerDay != nil {
		stats.RequestsPerDay = quota.RequestsPerDay
		stats.RequestsPercentDay = float64(usage.RequestsToday) / float64(*quota.RequestsPerDay) * 100
	}

	if quota.RequestsPerMonth != nil {
		stats.RequestsPerMonth = quota.RequestsPerMonth
		stats.RequestsPercentMonth = float64(usage.RequestsMonth) / float64(*quota.RequestsPerMonth) * 100
	}

	if quota.MaxConcurrent != nil {
		stats.MaxConcurrent = quota.MaxConcurrent
		stats.ConcurrentPercent = float64(usage.CurrentConcurrent) / float64(*quota.MaxConcurrent) * 100
	}

	if quota.MaxStorageBytes != nil {
		stats.MaxStorageBytes = quota.MaxStorageBytes
		stats.StoragePercentUsed = float64(usage.StorageUsedBytes) / float64(*quota.MaxStorageBytes) * 100
	}

	return stats, nil
}

// autoReset автоматически сбрасывает daily/monthly счетчики если необходимо
func (s *Service) autoReset(ctx context.Context, usage *models.QuotaUsage) error {
	now := time.Now()
	needsUpdate := false

	// Daily reset (если last reset был вчера или раньше)
	if now.Sub(usage.LastDailyReset) >= 24*time.Hour {
		usage.TokensUsedToday = 0
		usage.RequestsToday = 0
		usage.LastDailyReset = now
		needsUpdate = true

		s.logger.WithFields(logrus.Fields{
			"target_id": usage.TargetID,
			"quota_id":  usage.QuotaID,
		}).Debug("Auto-reset daily quota usage")
	}

	// Monthly reset (если different month)
	if now.Month() != usage.LastMonthlyReset.Month() || now.Year() != usage.LastMonthlyReset.Year() {
		usage.TokensUsedMonth = 0
		usage.RequestsMonth = 0
		usage.LastMonthlyReset = now
		needsUpdate = true

		s.logger.WithFields(logrus.Fields{
			"target_id": usage.TargetID,
			"quota_id":  usage.QuotaID,
		}).Debug("Auto-reset monthly quota usage")
	}

	if needsUpdate {
		return s.db.UpdateQuotaUsage(ctx, usage)
	}

	return nil
}

// getQuotaTarget определяет scope и target_id для quota lookup
// Приоритет: tenant > user
func (s *Service) getQuotaTarget(userID string, tenantID *string) (models.QuotaScope, string) {
	if tenantID != nil && *tenantID != "" {
		return models.QuotaScopeTenant, *tenantID
	}
	return models.QuotaScopeUser, userID
}


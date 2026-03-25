// Package request provides request tracking and monitoring functionality
package request

import (
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// StorageConfig конфигурация для request storage
type StorageConfig struct {
	// MaxRequests максимальное количество хранимых запросов
	MaxRequests int

	// RetentionPeriod период хранения завершенных запросов
	RetentionPeriod time.Duration

	// CleanupInterval интервал очистки старых запросов
	CleanupInterval time.Duration
}

// DefaultStorageConfig возвращает конфигурацию по умолчанию
func DefaultStorageConfig() StorageConfig {
	return StorageConfig{
		MaxRequests:     1000,
		RetentionPeriod: 24 * time.Hour,
		CleanupInterval: 5 * time.Minute,
	}
}

// Storage хранит информацию о запросах в памяти
type Storage struct {
	mu     sync.RWMutex
	config StorageConfig
	logger *logrus.Logger

	// Requests хранятся в slice (ring buffer)
	requests []*RequestInfo
	index    int  // текущий индекс для записи
	full     bool // true если buffer заполнен хотя бы раз

	// Канал для остановки background задач
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewStorage создает новое хранилище запросов
func NewStorage(config StorageConfig, logger *logrus.Logger) *Storage {
	if logger == nil {
		logger = logrus.New()
	}

	s := &Storage{
		config:   config,
		logger:   logger,
		requests: make([]*RequestInfo, config.MaxRequests),
		stopChan: make(chan struct{}),
	}

	return s
}

// Start запускает background задачи
func (s *Storage) Start() {
	s.logger.Info("Starting request storage")

	// Запускаем cleanup задачу
	s.wg.Add(1)
	go s.cleanupWorker()
}

// Stop останавливает background задачи
func (s *Storage) Stop() {
	s.logger.Info("Stopping request storage")
	close(s.stopChan)
	s.wg.Wait()
	s.logger.Info("Request storage stopped")
}

// Add добавляет новый запрос в storage
func (s *Storage) Add(req *RequestInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests[s.index] = req
	s.index++

	if s.index >= s.config.MaxRequests {
		s.index = 0
		s.full = true
	}

	s.logger.WithFields(logrus.Fields{
		"request_id": req.ID,
		"endpoint":   req.Endpoint,
		"status":     req.Status,
	}).Debug("Request added to storage")
}

// Update обновляет существующий запрос
func (s *Storage) Update(req *RequestInfo) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ищем запрос по ID
	for i := range s.requests {
		if s.requests[i] != nil && s.requests[i].ID == req.ID {
			s.requests[i] = req
			return true
		}
	}

	return false
}

// Get возвращает запрос по ID
func (s *Storage) Get(id string) *RequestInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, req := range s.requests {
		if req != nil && req.ID == id {
			return req
		}
	}

	return nil
}

// List возвращает все запросы (копию)
func (s *Storage) List() []*RequestInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*RequestInfo, 0, s.config.MaxRequests)

	if s.full {
		// Если buffer заполнен, начинаем с текущего индекса (oldest)
		for i := s.index; i < len(s.requests); i++ {
			if s.requests[i] != nil {
				result = append(result, s.requests[i])
			}
		}
	}

	// Добавляем запросы до текущего индекса
	for i := 0; i < s.index; i++ {
		if s.requests[i] != nil {
			result = append(result, s.requests[i])
		}
	}

	return result
}

// Filter возвращает запросы, соответствующие фильтру
func (s *Storage) Filter(filters ...FilterFunc) []*RequestInfo {
	allRequests := s.List()

	if len(filters) == 0 {
		return allRequests
	}

	result := make([]*RequestInfo, 0, len(allRequests))

	for _, req := range allRequests {
		match := true
		for _, filter := range filters {
			if !filter(req) {
				match = false
				break
			}
		}
		if match {
			result = append(result, req)
		}
	}

	return result
}

// Sort сортирует запросы по указанному полю
func (s *Storage) Sort(requests []*RequestInfo, field SortField, descending bool) []*RequestInfo {
	sorted := make([]*RequestInfo, len(requests))
	copy(sorted, requests)

	sort.Slice(sorted, func(i, j int) bool {
		var less bool

		switch field {
		case SortByTime:
			less = sorted[i].Timestamp < sorted[j].Timestamp
		case SortByDuration:
			less = sorted[i].Duration < sorted[j].Duration
		case SortByStatus:
			less = sorted[i].Status < sorted[j].Status
		case SortByEndpoint:
			less = sorted[i].Endpoint < sorted[j].Endpoint
		default:
			less = sorted[i].Timestamp < sorted[j].Timestamp
		}

		if descending {
			return !less
		}
		return less
	})

	return sorted
}

// GetStats возвращает статистику по запросам
func (s *Storage) GetStats() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := 0
	pending := 0
	success := 0
	errors := 0
	var totalDuration int64

	for _, req := range s.requests {
		if req == nil {
			continue
		}

		total++

		switch req.Status {
		case StatusPending:
			pending++
		case StatusSuccess:
			success++
		case StatusError:
			errors++
		}

		if req.IsComplete() {
			totalDuration += req.Duration
		}
	}

	avgDuration := int64(0)
	if total-pending > 0 {
		avgDuration = totalDuration / int64(total-pending)
	}

	return map[string]any{
		"total":        total,
		"pending":      pending,
		"success":      success,
		"errors":       errors,
		"avg_duration": avgDuration,
	}
}

// Clear очищает все запросы
func (s *Storage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests = make([]*RequestInfo, s.config.MaxRequests)
	s.index = 0
	s.full = false

	s.logger.Info("Request storage cleared")
}

// cleanupWorker периодически очищает старые запросы
func (s *Storage) cleanupWorker() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

// cleanup удаляет старые завершенные запросы
func (s *Storage) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-s.config.RetentionPeriod).Unix()
	cleaned := 0

	for i := range s.requests {
		req := s.requests[i]
		if req != nil && req.IsComplete() && req.Timestamp < cutoff {
			s.requests[i] = nil
			cleaned++
		}
	}

	if cleaned > 0 {
		s.logger.WithField("cleaned_count", cleaned).Debug("Cleaned old requests")
	}
}

// Package circuit provides circuit breaker implementation for Ollama-OpenAI Proxy
package circuit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/config"
)

// State представляет состояние circuit breaker
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Breaker реализует circuit breaker паттерн
type Breaker struct {
	name         string
	maxFailures  int
	resetTimeout time.Duration
	logger       *logrus.Logger

	mutex           sync.RWMutex
	state           State
	failures        int
	lastFailure     time.Time
	lastStateChange time.Time

	// Статистика
	stats Stats
}

// Stats содержит статистику работы circuit breaker
type Stats struct {
	TotalRequests      int64     `json:"total_requests"`
	SuccessfulRequests int64     `json:"successful_requests"`
	FailedRequests     int64     `json:"failed_requests"`
	RejectedRequests   int64     `json:"rejected_requests"`
	StateChanges       int64     `json:"state_changes"`
	LastStateChange    time.Time `json:"last_state_change"`
	CurrentState       string    `json:"current_state"`
	FailureRate        float64   `json:"failure_rate"`
}

// Operation представляет операцию для выполнения через circuit breaker
type Operation func(ctx context.Context) (interface{}, error)

// NewBreaker создает новый circuit breaker
func NewBreaker(name string, cfg *config.Config, logger *logrus.Logger) *Breaker {
	maxFailures := 5
	resetTimeout := 60 * time.Second

	// Используем конфигурацию если доступна
	if cfg.Ollama.CircuitBreaker.Enabled {
		if cfg.Ollama.CircuitBreaker.MaxFailures > 0 {
			maxFailures = cfg.Ollama.CircuitBreaker.MaxFailures
		}
		if cfg.Ollama.CircuitBreaker.ResetTimeout > 0 {
			resetTimeout = cfg.Ollama.CircuitBreaker.ResetTimeout
		}
	}

	breaker := &Breaker{
		name:            name,
		maxFailures:     maxFailures,
		resetTimeout:    resetTimeout,
		logger:          logger,
		state:           StateClosed,
		lastStateChange: time.Now(),
		stats: Stats{
			CurrentState: StateClosed.String(),
		},
	}

	logger.WithFields(logrus.Fields{
		"name":          name,
		"max_failures":  maxFailures,
		"reset_timeout": resetTimeout,
	}).Info("Circuit breaker initialized")

	return breaker
}

// Execute выполняет операцию через circuit breaker
func (b *Breaker) Execute(ctx context.Context, operation Operation) (interface{}, error) {
	// Проверяем можем ли выполнить операцию
	if err := b.allowRequest(); err != nil {
		return nil, err
	}

	// Выполняем операцию
	start := time.Now()
	result, err := operation(ctx)
	duration := time.Since(start)

	// Обновляем статистику и состояние
	b.recordResult(err, duration)

	return result, err
}

// allowRequest проверяет можно ли выполнить запрос
func (b *Breaker) allowRequest() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.stats.TotalRequests++

	switch b.state {
	case StateClosed:
		// Закрытое состояние - разрешаем все запросы
		return nil

	case StateOpen:
		// Открытое состояние - проверяем можно ли перейти в half-open
		if time.Since(b.lastFailure) > b.resetTimeout {
			b.setState(StateHalfOpen)
			b.logger.WithField("name", b.name).Info("Circuit breaker transitioned to half-open")
			return nil
		}

		// Отклоняем запрос
		b.stats.RejectedRequests++
		return NewCircuitBreakerError(b.name, "circuit breaker is open")

	case StateHalfOpen:
		// Half-open состояние - разрешаем один запрос для проверки
		return nil

	default:
		return NewCircuitBreakerError(b.name, "unknown circuit breaker state")
	}
}

// recordResult записывает результат операции
func (b *Breaker) recordResult(err error, duration time.Duration) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if err != nil {
		b.failures++
		b.lastFailure = time.Now()
		b.stats.FailedRequests++

		b.logger.WithFields(logrus.Fields{
			"name":     b.name,
			"failures": b.failures,
			"state":    b.state.String(),
			"duration": duration,
		}).WithError(err).Debug("Circuit breaker recorded failure")

		// Проверяем нужно ли открыть circuit breaker
		if b.state == StateClosed && b.failures >= b.maxFailures {
			b.setState(StateOpen)
			b.logger.WithFields(logrus.Fields{
				"name":         b.name,
				"failures":     b.failures,
				"max_failures": b.maxFailures,
			}).Warn("Circuit breaker opened due to failures")
		} else if b.state == StateHalfOpen {
			// В half-open состоянии любая ошибка возвращает нас в open
			b.setState(StateOpen)
			b.logger.WithField("name", b.name).Warn("Circuit breaker reopened after half-open failure")
		}
	} else {
		// Успешная операция
		b.stats.SuccessfulRequests++

		b.logger.WithFields(logrus.Fields{
			"name":     b.name,
			"state":    b.state.String(),
			"duration": duration,
		}).Debug("Circuit breaker recorded success")

		if b.state == StateHalfOpen {
			// Успешная операция в half-open состоянии закрывает circuit breaker
			b.failures = 0
			b.setState(StateClosed)
			b.logger.WithField("name", b.name).Info("Circuit breaker closed after successful half-open request")
		} else if b.state == StateClosed {
			// Сбрасываем счетчик неудач при успешной операции
			b.failures = 0
		}
	}

	// Обновляем failure rate
	b.updateFailureRate()
}

// setState изменяет состояние circuit breaker
func (b *Breaker) setState(newState State) {
	oldState := b.state
	b.state = newState
	b.lastStateChange = time.Now()
	b.stats.StateChanges++
	b.stats.CurrentState = newState.String()
	b.stats.LastStateChange = b.lastStateChange

	b.logger.WithFields(logrus.Fields{
		"name":      b.name,
		"old_state": oldState.String(),
		"new_state": newState.String(),
	}).Info("Circuit breaker state changed")
}

// updateFailureRate обновляет статистику failure rate
func (b *Breaker) updateFailureRate() {
	if b.stats.TotalRequests > 0 {
		b.stats.FailureRate = float64(b.stats.FailedRequests) / float64(b.stats.TotalRequests) * 100
	} else {
		b.stats.FailureRate = 0
	}
}

// GetState возвращает текущее состояние
func (b *Breaker) GetState() State {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.state
}

// GetStats возвращает статистику
func (b *Breaker) GetStats() Stats {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	stats := b.stats
	stats.CurrentState = b.state.String()
	b.updateFailureRate()

	return stats
}

// Reset сбрасывает circuit breaker в закрытое состояние
func (b *Breaker) Reset() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	oldState := b.state
	b.failures = 0
	b.setState(StateClosed)

	b.logger.WithFields(logrus.Fields{
		"name":      b.name,
		"old_state": oldState.String(),
	}).Info("Circuit breaker manually reset")
}

// IsOpen проверяет открыт ли circuit breaker
func (b *Breaker) IsOpen() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.state == StateOpen
}

// CircuitBreakerError представляет ошибку circuit breaker
type CircuitBreakerError struct {
	BreakerName string
	Message     string
}

func (e *CircuitBreakerError) Error() string {
	return fmt.Sprintf("circuit breaker '%s': %s", e.BreakerName, e.Message)
}

// NewCircuitBreakerError создает новую ошибку circuit breaker
func NewCircuitBreakerError(name, message string) *CircuitBreakerError {
	return &CircuitBreakerError{
		BreakerName: name,
		Message:     message,
	}
}

// Manager управляет несколькими circuit breaker'ами
type Manager struct {
	config   *config.Config
	logger   *logrus.Logger
	breakers map[string]*Breaker
	mutex    sync.RWMutex
}

// NewManager создает новый manager для circuit breaker'ов
func NewManager(cfg *config.Config, logger *logrus.Logger) *Manager {
	return &Manager{
		config:   cfg,
		logger:   logger,
		breakers: make(map[string]*Breaker),
	}
}

// GetBreaker возвращает circuit breaker по имени (создает если не существует)
func (m *Manager) GetBreaker(name string) *Breaker {
	m.mutex.RLock()
	breaker, exists := m.breakers[name]
	m.mutex.RUnlock()

	if exists {
		return breaker
	}

	// Создаем новый breaker
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Double-check после получения write lock
	if breaker, exists := m.breakers[name]; exists {
		return breaker
	}

	breaker = NewBreaker(name, m.config, m.logger)
	m.breakers[name] = breaker

	return breaker
}

// GetAllStats возвращает статистику всех circuit breaker'ов
func (m *Manager) GetAllStats() map[string]Stats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := make(map[string]Stats)
	for name, breaker := range m.breakers {
		stats[name] = breaker.GetStats()
	}

	return stats
}

// ResetAll сбрасывает все circuit breaker'ы
func (m *Manager) ResetAll() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for name, breaker := range m.breakers {
		breaker.Reset()
		m.logger.WithField("breaker", name).Info("Circuit breaker reset")
	}
}

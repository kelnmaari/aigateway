// Package metrics provides Prometheus metrics collection and export
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics содержит все Prometheus метрики
type Metrics struct {
	// HTTP Request metrics
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight *prometheus.GaugeVec
	HTTPResponseSize     *prometheus.HistogramVec

	// Ollama Client metrics
	OllamaRequestsTotal     *prometheus.CounterVec
	OllamaRequestDuration   *prometheus.HistogramVec
	OllamaErrorsTotal       *prometheus.CounterVec
	OllamaConnectionsActive prometheus.Gauge

	// API Key metrics
	APIKeyRequestsTotal     *prometheus.CounterVec
	APIKeyRateLimitExceeded *prometheus.CounterVec
	APIKeysActiveTotal      prometheus.Gauge
	APIKeyTokensUsedTotal   *prometheus.CounterVec

	// Circuit Breaker metrics
	CircuitBreakerState      *prometheus.GaugeVec
	CircuitBreakerTripsTotal *prometheus.CounterVec
}

var (
	// DefaultMetrics глобальный экземпляр метрик
	DefaultMetrics *Metrics
)

// NewMetrics создает новый набор метрик
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		// HTTP Request metrics
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status_code"},
		),

		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request latencies in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "endpoint"},
		),

		HTTPRequestsInFlight: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_requests_in_flight",
				Help:      "Current number of HTTP requests being served",
			},
			[]string{"endpoint"},
		),

		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_response_size_bytes",
				Help:      "HTTP response sizes in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8), // 100B to ~100MB
			},
			[]string{"method", "endpoint"},
		),

		// Ollama Client metrics
		OllamaRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "ollama_requests_total",
				Help:      "Total number of requests to Ollama",
			},
			[]string{"model", "operation", "status"},
		),

		OllamaRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "ollama_request_duration_seconds",
				Help:      "Ollama request latencies in seconds",
				Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60, 120},
			},
			[]string{"model", "operation"},
		),

		OllamaErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "ollama_errors_total",
				Help:      "Total number of Ollama errors",
			},
			[]string{"model", "operation", "error_type"},
		),

		OllamaConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "ollama_connections_active",
				Help:      "Current number of active connections to Ollama",
			},
		),

		// API Key metrics
		APIKeyRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "api_key_requests_total",
				Help:      "Total number of requests per API key",
			},
			[]string{"key_id", "endpoint"},
		),

		APIKeyRateLimitExceeded: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "api_key_rate_limit_exceeded_total",
				Help:      "Total number of rate limit exceeded events per API key",
			},
			[]string{"key_id"},
		),

		APIKeysActiveTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "api_keys_active_total",
				Help:      "Current number of active API keys",
			},
		),

		APIKeyTokensUsedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "api_key_tokens_used_total",
				Help:      "Total tokens used per API key and model",
			},
			[]string{"key_id", "model"},
		),

		// Circuit Breaker metrics
		CircuitBreakerState: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "circuit_breaker_state",
				Help:      "Circuit breaker state (0=closed, 1=half-open, 2=open)",
			},
			[]string{"breaker_name"},
		),

		CircuitBreakerTripsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "circuit_breaker_trips_total",
				Help:      "Total number of circuit breaker trips",
			},
			[]string{"breaker_name"},
		),
	}

	return m
}

// Init инициализирует глобальные метрики
func Init(namespace string) {
	DefaultMetrics = NewMetrics(namespace)
}

// RecordHTTPRequest записывает метрику HTTP запроса
func (m *Metrics) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration, responseSize int) {
	m.HTTPRequestsTotal.WithLabelValues(method, endpoint, statusCodeToString(statusCode)).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	m.HTTPResponseSize.WithLabelValues(method, endpoint).Observe(float64(responseSize))
}

// IncHTTPRequestsInFlight увеличивает счетчик активных запросов
func (m *Metrics) IncHTTPRequestsInFlight(endpoint string) {
	m.HTTPRequestsInFlight.WithLabelValues(endpoint).Inc()
}

// DecHTTPRequestsInFlight уменьшает счетчик активных запросов
func (m *Metrics) DecHTTPRequestsInFlight(endpoint string) {
	m.HTTPRequestsInFlight.WithLabelValues(endpoint).Dec()
}

// RecordOllamaRequest записывает метрику запроса к Ollama
func (m *Metrics) RecordOllamaRequest(model, operation, status string, duration time.Duration) {
	m.OllamaRequestsTotal.WithLabelValues(model, operation, status).Inc()
	m.OllamaRequestDuration.WithLabelValues(model, operation).Observe(duration.Seconds())
}

// RecordOllamaError записывает ошибку Ollama
func (m *Metrics) RecordOllamaError(model, operation, errorType string) {
	m.OllamaErrorsTotal.WithLabelValues(model, operation, errorType).Inc()
}

// IncOllamaConnectionsActive увеличивает счетчик активных подключений
func (m *Metrics) IncOllamaConnectionsActive() {
	m.OllamaConnectionsActive.Inc()
}

// DecOllamaConnectionsActive уменьшает счетчик активных подключений
func (m *Metrics) DecOllamaConnectionsActive() {
	m.OllamaConnectionsActive.Dec()
}

// RecordAPIKeyRequest записывает запрос для API ключа
func (m *Metrics) RecordAPIKeyRequest(keyID, endpoint string) {
	m.APIKeyRequestsTotal.WithLabelValues(keyID, endpoint).Inc()
}

// RecordAPIKeyRateLimitExceeded записывает превышение rate limit
func (m *Metrics) RecordAPIKeyRateLimitExceeded(keyID string) {
	m.APIKeyRateLimitExceeded.WithLabelValues(keyID).Inc()
}

// SetAPIKeysActiveTotal устанавливает количество активных ключей
func (m *Metrics) SetAPIKeysActiveTotal(count float64) {
	m.APIKeysActiveTotal.Set(count)
}

// RecordAPIKeyTokensUsed записывает использованные токены
func (m *Metrics) RecordAPIKeyTokensUsed(keyID, model string, tokens int) {
	m.APIKeyTokensUsedTotal.WithLabelValues(keyID, model).Add(float64(tokens))
}

// SetCircuitBreakerState устанавливает состояние circuit breaker
// 0 = closed, 1 = half-open, 2 = open
func (m *Metrics) SetCircuitBreakerState(name string, state int) {
	m.CircuitBreakerState.WithLabelValues(name).Set(float64(state))
}

// RecordCircuitBreakerTrip записывает trip circuit breaker
func (m *Metrics) RecordCircuitBreakerTrip(name string) {
	m.CircuitBreakerTripsTotal.WithLabelValues(name).Inc()
}

// statusCodeToString конвертирует status code в string для labels
func statusCodeToString(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

// Package metrics provides Prometheus metrics export for monitoring and observability
// Version: 1.11.6+ (Enterprise Suite - Prometheus Metrics Export)
package metrics

import (
	"context"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"

	"aigateway/internal/storage"
)

// Prometheus metrics namespace
const (
	namespace = "ollama_proxy"
)

// ========================================
// HTTP Metrics
// ========================================

var (
	// HTTPRequestsTotal counts total HTTP requests by method, endpoint, and status
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTPRequestDuration measures HTTP request duration in seconds
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10},
		},
		[]string{"method", "endpoint"},
	)

	// HTTPResponseSize measures HTTP response size in bytes
	HTTPResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 7), // 100B to 100MB
		},
		[]string{"method", "endpoint"},
	)

	// ActiveConnections tracks current active HTTP connections
	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "http_active_connections",
			Help:      "Number of active HTTP connections",
		},
	)
)

// ========================================
// API Usage Metrics
// ========================================

var (
	// APITokensUsed counts total API tokens used by api_key_id, model, and type (prompt/completion)
	APITokensUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "api_tokens_used_total",
			Help:      "Total tokens used",
		},
		[]string{"api_key_id", "model", "type"},
	)

	// APIRequestsTotal counts total API requests by model and status (success/error)
	APIRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "api_requests_total",
			Help:      "Total API requests",
		},
		[]string{"model", "status"},
	)

	// APICost tracks total cost by api_key_id and model (if pricing available)
	APICost = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "api_cost_total",
			Help:      "Total API cost (if pricing enabled)",
		},
		[]string{"api_key_id", "model"},
	)
)

// ========================================
// Model Metrics
// ========================================

var (
	// ModelRequestDuration measures model request duration in seconds
	ModelRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "model_request_duration_seconds",
			Help:      "Model request duration in seconds",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
		},
		[]string{"model"},
	)

	// ModelsLoaded tracks number of currently loaded models
	ModelsLoaded = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "models_loaded",
			Help:      "Number of currently loaded models",
		},
	)

	// ModelErrors counts model errors by model and error_type
	ModelErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "model_errors_total",
			Help:      "Total model errors",
		},
		[]string{"model", "error_type"},
	)

	// InferenceCacheBytes tracks current size of inference artifact cache.
	InferenceCacheBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "inference_cache_bytes",
			Help:      "Current size of inference artifact cache in bytes",
		},
	)

	// InferenceCacheLimitBytes tracks configured cache limit (0 = unlimited).
	InferenceCacheLimitBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "inference_cache_limit_bytes",
			Help:      "Configured cache size limit in bytes (0 = unlimited)",
		},
	)

	// InferenceCacheEvictedBytes counts evicted bytes.
	InferenceCacheEvictedBytes = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "inference_cache_evicted_bytes_total",
			Help:      "Total bytes evicted from inference cache",
		},
	)

	// InferenceCacheEvictedFiles counts evicted files.
	InferenceCacheEvictedFiles = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "inference_cache_evicted_files_total",
			Help:      "Total files evicted from inference cache",
		},
	)

	// InferenceStartupFailures counts container startup failures by provider.
	InferenceStartupFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "inference_startup_failures_total",
			Help:      "Total inference container startup failures",
		},
		[]string{"provider"},
	)

	// InferenceHealthFailures counts health check failures by provider.
	InferenceHealthFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "inference_health_failures_total",
			Help:      "Total inference container health check failures",
		},
		[]string{"provider"},
	)

	// InferenceContainersStarted counts successfully started containers by provider.
	InferenceContainersStarted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "inference_containers_started_total",
			Help:      "Total inference containers started successfully",
		},
		[]string{"provider"},
	)
)

// ========================================
// Quota Metrics
// ========================================

var (
	// QuotaUsage tracks current quota usage by target_id and type (tokens_daily, requests_daily, etc.)
	QuotaUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "quota_usage",
			Help:      "Current quota usage",
		},
		[]string{"target_id", "type"},
	)

	// QuotaExceeded counts quota exceeded events by target_id and type
	QuotaExceeded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "quota_exceeded_total",
			Help:      "Total quota exceeded events",
		},
		[]string{"target_id", "type"},
	)

	// QuotaLimit tracks current quota limits by target_id and type
	QuotaLimit = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "quota_limit",
			Help:      "Current quota limit",
		},
		[]string{"target_id", "type"},
	)
)

// ========================================
// System Metrics
// ========================================

var (
	// DBConnections tracks active database connections
	DBConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "db_connections",
			Help:      "Number of active database connections",
		},
	)

	// GoroutinesCount tracks number of goroutines
	GoroutinesCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "goroutines",
			Help:      "Number of goroutines",
		},
	)

	// MemoryUsage tracks memory usage in bytes by type (alloc, sys, heap_alloc, etc.)
	MemoryUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "memory_bytes",
			Help:      "Memory usage in bytes",
		},
		[]string{"type"},
	)

	// APIKeysTotal tracks total number of API keys
	APIKeysTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "api_keys_total",
			Help:      "Total number of API keys",
		},
	)

	// UsersTotal tracks total number of users
	UsersTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "users_total",
			Help:      "Total number of users",
		},
	)

	// TenantsTotal tracks total number of tenants
	TenantsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "tenants_total",
			Help:      "Total number of tenants",
		},
	)
)

// ========================================
// Authentication Metrics (v1.11+)
// ========================================

var (
	// AuthAttempts counts authentication attempts by type (jwt, oidc, ldap, api_key) and status
	AuthAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "auth_attempts_total",
			Help:      "Total authentication attempts",
		},
		[]string{"type", "status"},
	)

	// AuthDuration measures authentication duration by type
	AuthDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "auth_duration_seconds",
			Help:      "Authentication duration in seconds",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2},
		},
		[]string{"type"},
	)
)

// ========================================
// MetricsCollector
// ========================================

// MetricsCollector periodically collects system-level metrics
type MetricsCollector struct {
	db        storage.Database
	logger    *logrus.Logger
	interval  time.Duration
	ctx       context.Context
	cancelFn  context.CancelFunc
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(db storage.Database, logger *logrus.Logger, interval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		db:       db,
		logger:   logger,
		interval: interval,
	}
}

// Start начинает periodic сбор метрик
func (m *MetricsCollector) Start(parentCtx context.Context) {
	m.ctx, m.cancelFn = context.WithCancel(parentCtx)
	
	m.logger.WithField("interval", m.interval).Info("Metrics collector started")
	
	// Collect immediately on start
	m.collect()
	
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			m.logger.Info("Metrics collector stopped")
			return
		case <-ticker.C:
			m.collect()
		}
	}
}

// Stop останавливает collector
func (m *MetricsCollector) Stop() {
	if m.cancelFn != nil {
		m.cancelFn()
	}
}

// collect собирает все system metrics
func (m *MetricsCollector) collect() {
	m.collectSystemMetrics()
	m.collectDatabaseMetrics()
}

// collectSystemMetrics собирает system-level метрики
func (m *MetricsCollector) collectSystemMetrics() {
	// Goroutines count
	GoroutinesCount.Set(float64(runtime.NumGoroutine()))
	
	// Memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	MemoryUsage.WithLabelValues("alloc").Set(float64(memStats.Alloc))
	MemoryUsage.WithLabelValues("sys").Set(float64(memStats.Sys))
	MemoryUsage.WithLabelValues("heap_alloc").Set(float64(memStats.HeapAlloc))
	MemoryUsage.WithLabelValues("heap_sys").Set(float64(memStats.HeapSys))
	MemoryUsage.WithLabelValues("heap_inuse").Set(float64(memStats.HeapInuse))
	MemoryUsage.WithLabelValues("stack_inuse").Set(float64(memStats.StackInuse))
}

// collectDatabaseMetrics собирает database-level метрики
func (m *MetricsCollector) collectDatabaseMetrics() {
	if m.db == nil {
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Count API keys (if storage available)
	if m.db != nil {
		if keys, err := m.db.ListAPIKeys(ctx); err == nil {
			APIKeysTotal.Set(float64(len(keys)))
		}
	}
	
	// Count users (v1.11+)
	// Note: Добавим позже при наличии User management
	
	// Count tenants (v1.11+)
	// Note: Добавим позже при наличии Tenant management
	
	// Collect quota usage (v1.11.7+)
	if m.db != nil {
		// Get all quotas and their usage
		quotas, err := m.db.ListQuotas(ctx, nil)
		if err == nil {
			for _, quota := range quotas {
				if !quota.Enabled {
					continue
				}
				
				usage, err := m.db.GetQuotaUsage(ctx, quota.ID)
				if err != nil {
					continue
				}
				
				targetID := quota.TargetID
				
				// Record daily token usage
				if quota.TokensPerDay != nil {
					QuotaUsage.WithLabelValues(targetID, "tokens_daily").Set(float64(usage.TokensUsedToday))
					QuotaLimit.WithLabelValues(targetID, "tokens_daily").Set(float64(*quota.TokensPerDay))
				}
				
				// Record monthly token usage
				if quota.TokensPerMonth != nil {
					QuotaUsage.WithLabelValues(targetID, "tokens_monthly").Set(float64(usage.TokensUsedMonth))
					QuotaLimit.WithLabelValues(targetID, "tokens_monthly").Set(float64(*quota.TokensPerMonth))
				}
				
				// Record daily request usage
				if quota.RequestsPerDay != nil {
					QuotaUsage.WithLabelValues(targetID, "requests_daily").Set(float64(usage.RequestsToday))
					QuotaLimit.WithLabelValues(targetID, "requests_daily").Set(float64(*quota.RequestsPerDay))
				}
				
				// Record monthly request usage
				if quota.RequestsPerMonth != nil {
					QuotaUsage.WithLabelValues(targetID, "requests_monthly").Set(float64(usage.RequestsMonth))
					QuotaLimit.WithLabelValues(targetID, "requests_monthly").Set(float64(*quota.RequestsPerMonth))
				}
			}
		}
	}
}

// ========================================
// Utility Functions
// ========================================

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(method, endpoint, status string, duration time.Duration, responseSize int64) {
	HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	if responseSize > 0 {
		HTTPResponseSize.WithLabelValues(method, endpoint).Observe(float64(responseSize))
	}
}

// RecordAPIUsage records API usage metrics
func RecordAPIUsage(apiKeyID, model string, promptTokens, completionTokens int, success bool) {
	if promptTokens > 0 {
		APITokensUsed.WithLabelValues(apiKeyID, model, "prompt").Add(float64(promptTokens))
	}
	if completionTokens > 0 {
		APITokensUsed.WithLabelValues(apiKeyID, model, "completion").Add(float64(completionTokens))
	}
	
	status := "success"
	if !success {
		status = "error"
	}
	APIRequestsTotal.WithLabelValues(model, status).Inc()
}

// RecordModelRequest records model request metrics
func RecordModelRequest(model string, duration time.Duration) {
	ModelRequestDuration.WithLabelValues(model).Observe(duration.Seconds())
}

// RecordModelError records model error
func RecordModelError(model, errorType string) {
	ModelErrors.WithLabelValues(model, errorType).Inc()
}

// RecordAuthAttempt records authentication attempt
func RecordAuthAttempt(authType, status string, duration time.Duration) {
	AuthAttempts.WithLabelValues(authType, status).Inc()
	AuthDuration.WithLabelValues(authType).Observe(duration.Seconds())
}

// RecordQuotaExceeded records quota exceeded event
func RecordQuotaExceeded(targetID, quotaType string) {
	QuotaExceeded.WithLabelValues(targetID, quotaType).Inc()
}

// UpdateQuotaUsage updates current quota usage
func UpdateQuotaUsage(targetID, quotaType string, current, limit float64) {
	QuotaUsage.WithLabelValues(targetID, quotaType).Set(current)
	QuotaLimit.WithLabelValues(targetID, quotaType).Set(limit)
}



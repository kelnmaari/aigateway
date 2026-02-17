// Package metrics provides compatibility wrapper for old DefaultMetrics pattern
// This file bridges old metrics code with new Prometheus implementation
package metrics

import "time"

// Metrics provides backward compatibility for old DefaultMetrics pattern
type Metrics struct{}

// DefaultMetrics is a global instance for backward compatibility
var DefaultMetrics *Metrics = &Metrics{}

// RecordAPIKeyTokensUsed records API key token usage
func (m *Metrics) RecordAPIKeyTokensUsed(apiKeyID, model string, tokens int) {
	if m == nil {
		return
	}
	APITokensUsed.WithLabelValues(apiKeyID, model, "total").Add(float64(tokens))
}

// RecordAPIKeyRateLimitExceeded records rate limit exceeded events
func (m *Metrics) RecordAPIKeyRateLimitExceeded(apiKeyID string) {
	// Rate limiting metrics handled elsewhere
}

// RecordProviderRequest records provider request
func (m *Metrics) RecordProviderRequest(model, operation, status string, duration time.Duration) {
	if m == nil {
		return
	}
	ModelRequestDuration.WithLabelValues(model).Observe(duration.Seconds())
}

// RecordProviderError records provider error
func (m *Metrics) RecordProviderError(model, operation, errorType string) {
	if m == nil {
		return
	}
	ModelErrors.WithLabelValues(model, errorType).Inc()
}

// IncHTTPRequestsInFlight increments active HTTP requests
func (m *Metrics) IncHTTPRequestsInFlight(endpoint string) {
	if m == nil {
		return
	}
	ActiveConnections.Inc()
}

// DecHTTPRequestsInFlight decrements active HTTP requests
func (m *Metrics) DecHTTPRequestsInFlight(endpoint string) {
	if m == nil {
		return
	}
	ActiveConnections.Dec()
}

// RecordHTTPRequest records HTTP request metrics
func (m *Metrics) RecordHTTPRequest(method, endpoint string, statusCode int, duration time.Duration, responseSize int) {
	if m == nil {
		return
	}
	HTTPRequestsTotal.WithLabelValues(method, endpoint, statusCodeToString(statusCode)).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	HTTPResponseSize.WithLabelValues(method, endpoint).Observe(float64(responseSize))
}

// RecordAPIKeyRequest records API key request
func (m *Metrics) RecordAPIKeyRequest(apiKeyID, model, endpoint string, tokens int) {
	if m == nil {
		return
	}
	if tokens > 0 {
		APITokensUsed.WithLabelValues(apiKeyID, model, "total").Add(float64(tokens))
	}
}

func statusCodeToString(code int) string {
	if code >= 200 && code < 300 {
		return "2xx"
	} else if code >= 300 && code < 400 {
		return "3xx"
	} else if code >= 400 && code < 500 {
		return "4xx"
	} else if code >= 500 {
		return "5xx"
	}
	return "unknown"
}



// Package middleware provides Prometheus metrics middleware for Gin
// Version: 1.11.6+ (Enterprise Suite - Prometheus Metrics Export)
package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ollama-openai-proxy/internal/metrics"
)

// PrometheusMiddleware собирает HTTP metrics для Prometheus
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		
		// Increment active connections
		metrics.ActiveConnections.Inc()
		defer metrics.ActiveConnections.Dec()
		
		// Get request details
		path := c.Request.URL.Path
		method := c.Request.Method
		
		// Process request
		c.Next()
		
		// Calculate metrics
		duration := time.Since(start)
		status := strconv.Itoa(c.Writer.Status())
		responseSize := int64(c.Writer.Size())
		
		// Normalize endpoint path (remove IDs for better cardinality)
		endpoint := normalizePath(path)
		
		// Record metrics
		metrics.RecordHTTPRequest(method, endpoint, status, duration, responseSize)
	}
}

// normalizePath нормализует путь для снижения cardinality метрик
// Заменяет ID/UUID в путях на placeholder
func normalizePath(path string) string {
	// Keep common prefixes as-is
	// For now, return path as-is for simplicity
	// In production, you might want to replace IDs:
	// "/api/users/abc123" -> "/api/users/:id"
	// "/api/keys/def456" -> "/api/keys/:id"
	
	// This can be implemented with regex or path parsing
	// For MVP, we keep original path but be aware of high cardinality
	
	return path
}


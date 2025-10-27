package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SlowRequestLogger logs HTTP requests that exceed the specified duration threshold.
//
// This middleware measures request processing time and logs detailed information
// about slow requests to help identify performance bottlenecks.
//
// Parameters:
//   - logger: Logger for slow request warnings
//   - threshold: Duration threshold for slow requests (e.g., 5 seconds)
//
// Returns:
//   - Gin middleware function
//
// Example:
//
//	router.Use(middleware.SlowRequestLogger(logger, 5*time.Second))
func SlowRequestLogger(logger *logrus.Logger, threshold time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		duration := time.Since(start)

		// Log slow requests
		if duration > threshold {
			logger.WithFields(logrus.Fields{
				"method":       c.Request.Method,
				"path":         c.Request.URL.Path,
				"query":        c.Request.URL.RawQuery,
				"duration_ms":  duration.Milliseconds(),
				"duration":     duration.String(),
				"status":       c.Writer.Status(),
				"user_agent":   c.Request.UserAgent(),
				"remote_addr":  c.ClientIP(),
				"threshold_ms": threshold.Milliseconds(),
			}).Warn("Slow request detected")
		}
	}
}


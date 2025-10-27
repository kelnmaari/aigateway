package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware adds OpenTelemetry distributed tracing to Gin HTTP requests.
//
// It creates a span for each HTTP request, extracts trace context from headers,
// and records HTTP metadata (method, URL, status code, etc.).
// The span is marked as error if status code >= 400.
//
// Parameters:
//   - tracer: OpenTelemetry tracer instance
//
// Returns:
//   - Gin middleware function
func TracingMiddleware(tracer trace.Tracer) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract trace context from incoming request headers (W3C Trace Context)
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Start new span for this HTTP request
		ctx, span := tracer.Start(ctx, c.Request.Method+" "+c.Request.URL.Path,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.scheme", c.Request.URL.Scheme),
				attribute.String("http.host", c.Request.Host),
				attribute.String("http.target", c.Request.URL.Path),
				attribute.String("http.user_agent", c.Request.UserAgent()),
				attribute.String("http.remote_addr", c.Request.RemoteAddr),
			),
		)
		defer span.End()

		// Update context in Gin
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// Record response status code
		span.SetAttributes(
			attribute.Int("http.status_code", c.Writer.Status()),
		)

		// Mark span as error if status code >= 400
		if c.Writer.Status() >= 400 {
			span.SetStatus(codes.Error, http.StatusText(c.Writer.Status()))
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}


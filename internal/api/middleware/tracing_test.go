package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func TestTracingMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	middleware := TracingMiddleware(tracer)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		// Check span is in context
		span := trace.SpanFromContext(c.Request.Context())
		assert.NotNil(t, span)
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())
}

func TestTracingMiddleware_ContextPropagation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	middleware := TracingMiddleware(tracer)

	router := gin.New()
	router.Use(middleware)
	router.POST("/api/test", func(c *gin.Context) {
		// Span should be available in context
		span := trace.SpanFromContext(c.Request.Context())
		assert.NotNil(t, span)

		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("POST", "/api/test", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestTracingMiddleware_ErrorStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	middleware := TracingMiddleware(tracer)

	router := gin.New()
	router.Use(middleware)
	router.GET("/error", func(c *gin.Context) {
		c.String(http.StatusBadRequest, "Bad Request")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "Bad Request", rr.Body.String())
}

func TestTracingMiddleware_ServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	middleware := TracingMiddleware(tracer)

	router := gin.New()
	router.Use(middleware)
	router.POST("/fail", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "Internal Server Error")
	})

	req := httptest.NewRequest("POST", "/fail", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestTracingMiddleware_WithTraceHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	middleware := TracingMiddleware(tracer)

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		assert.NotNil(t, span)
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	// Add W3C Trace Context headers
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestTracingMiddleware_MultipleRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	middleware := TracingMiddleware(tracer)

	spanCount := 0
	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		span := trace.SpanFromContext(c.Request.Context())
		if span != nil {
			spanCount++
		}
		c.String(http.StatusOK, "OK")
	})

	// Make multiple requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	assert.Equal(t, 5, spanCount)
}

func TestTracingMiddleware_SuccessStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	middleware := TracingMiddleware(tracer)

	router := gin.New()
	router.Use(middleware)
	router.GET("/success", func(c *gin.Context) {
		c.String(http.StatusOK, "Success")
	})

	req := httptest.NewRequest("GET", "/success", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "Success", rr.Body.String())
}


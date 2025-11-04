package middleware

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCrossOriginProtection(t *testing.T) {
	t.Attr("category", "middleware")
	t.Attr("type", "unit")
	t.Attr("feature", "csrf-protection")
	t.Attr("go_version", "1.25")

	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	t.Run("allows GET requests (safe method)", func(t *testing.T) {
		cfg := CrossOriginProtectionConfig{
			Enabled: true,
		}

		router := gin.New()
		router.Use(CrossOriginProtection(cfg, logger))
		router.GET("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "OK", w.Body.String())
	})

	t.Run("allows same-origin POST", func(t *testing.T) {
		cfg := CrossOriginProtectionConfig{
			Enabled: true,
		}

		router := gin.New()
		router.Use(CrossOriginProtection(cfg, logger))
		router.POST("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/test", nil)
		req.Host = "example.com"
		// Sec-Fetch-Site: same-origin indicates browser same-origin request
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("blocks cross-origin POST without trusted origin", func(t *testing.T) {
		cfg := CrossOriginProtectionConfig{
			Enabled:        true,
			TrustedOrigins: []string{}, // No trusted origins
		}

		router := gin.New()
		router.Use(CrossOriginProtection(cfg, logger))
		router.POST("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/test", nil)
		req.Host = "example.com"
		// Sec-Fetch-Site: cross-site indicates browser cross-origin request
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		req.Header.Set("Origin", "https://evil.com")
		router.ServeHTTP(w, req)

		assert.Equal(t, 403, w.Code)
		assert.Contains(t, w.Body.String(), "blocked")
	})

	t.Run("allows cross-origin POST from trusted origin", func(t *testing.T) {
		cfg := CrossOriginProtectionConfig{
			Enabled:        true,
			TrustedOrigins: []string{"https://trusted.com"},
		}

		router := gin.New()
		router.Use(CrossOriginProtection(cfg, logger))
		router.POST("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/test", nil)
		req.Host = "example.com"
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		req.Header.Set("Origin", "https://trusted.com")
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("respects disabled config", func(t *testing.T) {
		cfg := CrossOriginProtectionConfig{
			Enabled: false,
		}

		router := gin.New()
		router.Use(CrossOriginProtection(cfg, logger))
		router.POST("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		req.Header.Set("Origin", "https://evil.com")
		// Should pass because disabled
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("allows POST without Sec-Fetch-Site (non-browser)", func(t *testing.T) {
		cfg := CrossOriginProtectionConfig{
			Enabled: true,
		}

		router := gin.New()
		router.Use(CrossOriginProtection(cfg, logger))
		router.POST("/test", func(c *gin.Context) {
			c.String(200, "OK")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/test", nil)
		req.Host = "example.com"
		// No Sec-Fetch-Site or Origin header = non-browser request (e.g., curl, API client)
		// CrossOriginProtection allows these by default
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})
}

func TestIsSafeMethod(t *testing.T) {
	t.Attr("category", "middleware")
	t.Attr("type", "unit")

	tests := []struct {
		method string
		safe   bool
	}{
		{"GET", true},
		{"HEAD", true},
		{"OPTIONS", true},
		{"TRACE", true},
		{"POST", false},
		{"PUT", false},
		{"DELETE", false},
		{"PATCH", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			result := isSafeMethod(tt.method)
			assert.Equal(t, tt.safe, result)
		})
	}
}

func TestIsSameOrigin(t *testing.T) {
	t.Attr("category", "middleware")
	t.Attr("type", "unit")

	tests := []struct {
		name        string
		origin      string
		requestHost string
		expected    bool
	}{
		{
			name:        "same origin with https",
			origin:      "https://example.com",
			requestHost: "example.com",
			expected:    true,
		},
		{
			name:        "same origin with http",
			origin:      "http://example.com",
			requestHost: "example.com",
			expected:    true,
		},
		{
			name:        "same origin with port",
			origin:      "https://example.com:8080",
			requestHost: "example.com:8080",
			expected:    true,
		},
		{
			name:        "different origin",
			origin:      "https://evil.com",
			requestHost: "example.com",
			expected:    false,
		},
		{
			name:        "empty origin",
			origin:      "",
			requestHost: "example.com",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSameOrigin(tt.origin, tt.requestHost)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsOriginTrusted(t *testing.T) {
	t.Attr("category", "middleware")
	t.Attr("type", "unit")

	tests := []struct {
		name           string
		origin         string
		trustedOrigins []string
		requestHost    string
		expected       bool
	}{
		{
			name:           "same origin trusted",
			origin:         "https://example.com",
			trustedOrigins: []string{},
			requestHost:    "example.com",
			expected:       true,
		},
		{
			name:           "in trusted list",
			origin:         "https://trusted.com",
			trustedOrigins: []string{"https://trusted.com"},
			requestHost:    "example.com",
			expected:       true,
		},
		{
			name:           "wildcard trusted",
			origin:         "https://anything.com",
			trustedOrigins: []string{"*"},
			requestHost:    "example.com",
			expected:       true,
		},
		{
			name:           "not trusted",
			origin:         "https://evil.com",
			trustedOrigins: []string{"https://trusted.com"},
			requestHost:    "example.com",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOriginTrusted(tt.origin, tt.trustedOrigins, tt.requestHost)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultCrossOriginProtectionConfig(t *testing.T) {
	t.Attr("category", "middleware")
	t.Attr("type", "unit")

	cfg := DefaultCrossOriginProtectionConfig()

	assert.True(t, cfg.Enabled, "Should be enabled by default")
	assert.Empty(t, cfg.TrustedOrigins, "No default trusted origins")
	assert.False(t, cfg.AllowCredentials, "Credentials disabled by default")
	assert.True(t, cfg.RequireOriginHeader, "Require origin by default")
}

// Benchmark tests
func BenchmarkCrossOriginProtection_SafeMethod(b *testing.B) {
	b.Attr("category", "benchmarks")
	b.Attr("type", "performance")

	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	cfg := CrossOriginProtectionConfig{Enabled: true}
	router := gin.New()
	router.Use(CrossOriginProtection(cfg, logger))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}
}

func BenchmarkCrossOriginProtection_SameOriginPOST(b *testing.B) {
	b.Attr("category", "benchmarks")
	b.Attr("type", "performance")

	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	cfg := CrossOriginProtectionConfig{
		Enabled:             true,
		RequireOriginHeader: false,
	}
	router := gin.New()
	router.Use(CrossOriginProtection(cfg, logger))
	router.POST("/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/test", nil)
		req.Host = "example.com"
		req.Header.Set("Origin", "https://example.com")
		router.ServeHTTP(w, req)
	}
}


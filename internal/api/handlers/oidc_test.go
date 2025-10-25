// Package handlers provides tests for OIDC handlers
package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oidcauth "ollama-openai-proxy/internal/auth/oidc"
	"ollama-openai-proxy/internal/config"
)

// setupTestRouter creates a test Gin router with session middleware
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// Setup session middleware
	store := cookie.NewStore([]byte("test-secret-key-32-chars-long!!"))
	router.Use(sessions.Sessions("test_session", store))
	
	return router
}

func TestOIDCHandler_HandleLogin_OIDCDisabled(t *testing.T) {
	// Setup
	router := setupTestRouter()
	cfg := &config.Config{
		Auth: config.AuthConfig{
			OIDC: config.OIDCConfig{
				Enabled: false,
			},
		},
	}
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs

	handler := &OIDCHandler{
		config: cfg,
		logger: logger,
	}

	router.GET("/auth/oidc/login", handler.HandleLogin)

	// Execute
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/auth/oidc/login", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "OIDC authentication is not enabled")
}

func TestOIDCHandler_HandleLogin_GeneratesState(t *testing.T) {
	t.Skip("Requires OIDC provider mock")
	// This test would require:
	// 1. Mock OIDC provider
	// 2. Mock database
	// 3. Mock JWT manager
	// 4. Verify state is stored in session
	// 5. Verify redirect URL is generated
}

func TestOIDCHandler_HandleCallback_MissingState(t *testing.T) {
	// Setup
	router := setupTestRouter()
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	handler := &OIDCHandler{
		config: cfg,
		logger: logger,
	}

	router.GET("/auth/oidc/callback", handler.HandleCallback)

	// Execute - callback without state in session
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/auth/oidc/callback?state=test&code=test", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid session")
}

func TestOIDCHandler_HandleCallback_StateMismatch(t *testing.T) {
	t.Skip("Requires proper session handling in tests")
	// This test needs proper Gin session setup which is complex in unit tests
}

func TestOIDCHandler_HandleCallback_ErrorFromProvider(t *testing.T) {
	t.Skip("Requires proper session handling in tests")
}

func TestOIDCHandler_HandleCallback_MissingCode(t *testing.T) {
	t.Skip("Requires proper session handling in tests")
}

func TestOIDCHandler_HandleLogout_ClearSession(t *testing.T) {
	// Setup
	router := setupTestRouter()
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	handler := &OIDCHandler{
		config: cfg,
		logger: logger,
	}

	router.POST("/auth/oidc/logout", handler.HandleLogout)

	// Execute
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/oidc/logout", nil)
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Logged out successfully", response["message"])
}

func TestGenerateUsername_PreferredUsername(t *testing.T) {
	handler := &OIDCHandler{}
	
	claims := &oidcauth.KeycloakClaims{
		StandardClaims: oidcauth.StandardClaims{
			PreferredUsername: "john.doe",
			Email:             "john@example.com",
			Subject:           "123456",
		},
	}
	
	username := handler.generateUsername(claims)
	assert.Equal(t, "john.doe", username)
}

func TestGenerateUsername_EmailFallback(t *testing.T) {
	handler := &OIDCHandler{}
	
	claims := &oidcauth.KeycloakClaims{
		StandardClaims: oidcauth.StandardClaims{
			PreferredUsername: "",
			Email:             "john@example.com",
			Subject:           "123456",
		},
	}
	
	username := handler.generateUsername(claims)
	assert.Equal(t, "john", username)
}

func TestGenerateUsername_SubjectFallback(t *testing.T) {
	handler := &OIDCHandler{}
	
	claims := &oidcauth.KeycloakClaims{
		StandardClaims: oidcauth.StandardClaims{
			PreferredUsername: "",
			Email:             "",
			Subject:           "12345678-abcd-efgh",
		},
	}
	
	username := handler.generateUsername(claims)
	assert.Equal(t, "user_12345678", username)
}

func TestIsAdminRole_RealmRoles(t *testing.T) {
	handler := &OIDCHandler{}
	
	claims := &oidcauth.KeycloakClaims{
		RealmRoles: []string{"user", "admin", "developer"},
	}
	
	isAdmin := handler.isAdminRole(claims)
	assert.True(t, isAdmin)
}

func TestIsAdminRole_RealmAccess(t *testing.T) {
	handler := &OIDCHandler{}
	
	claims := &oidcauth.KeycloakClaims{}
	claims.RealmAccess.Roles = []string{"administrator"}
	
	isAdmin := handler.isAdminRole(claims)
	assert.True(t, isAdmin)
}

func TestIsAdminRole_Groups(t *testing.T) {
	handler := &OIDCHandler{}
	
	claims := &oidcauth.KeycloakClaims{
		Groups: []string{"/users", "/admin", "/developers"},
	}
	
	isAdmin := handler.isAdminRole(claims)
	assert.True(t, isAdmin)
}

func TestIsAdminRole_NoAdminRole(t *testing.T) {
	handler := &OIDCHandler{}
	
	claims := &oidcauth.KeycloakClaims{
		RealmRoles: []string{"user", "developer"},
		Groups:     []string{"/users", "/developers"},
	}
	
	isAdmin := handler.isAdminRole(claims)
	assert.False(t, isAdmin)
}

// Integration tests (require full setup)
func TestOIDCHandler_FullFlow_Integration(t *testing.T) {
	t.Skip("Integration test - requires OIDC provider, database, and JWT manager")
	// This would test the complete flow:
	// 1. Login -> redirect to OIDC provider
	// 2. Callback -> exchange code, verify token, provision user, return JWT
	// 3. Verify JWT works for protected endpoints
}

// Benchmark tests
func BenchmarkOIDCHandler_HandleLogout(b *testing.B) {
	router := setupTestRouter()
	handler := &OIDCHandler{
		config: &config.Config{},
		logger: logrus.New(),
	}
	router.POST("/auth/oidc/logout", handler.HandleLogout)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/auth/oidc/logout", nil)
		router.ServeHTTP(w, req)
	}
}


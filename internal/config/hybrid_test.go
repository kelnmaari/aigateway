package config

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSettingsManager для тестирования
type MockSettingsManager struct {
	mock.Mock
}

func (m *MockSettingsManager) GetString(ctx context.Context, id string) (string, error) {
	args := m.Called(ctx, id)
	return args.String(0), args.Error(1)
}

func (m *MockSettingsManager) GetInt(ctx context.Context, id string) (int, error) {
	args := m.Called(ctx, id)
	return args.Int(0), args.Error(1)
}

func (m *MockSettingsManager) GetBool(ctx context.Context, id string) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *MockSettingsManager) GetDuration(ctx context.Context, id string) (time.Duration, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *MockSettingsManager) InvalidateCache() {
	m.Called()
}

// Helper для создания тестовой конфигурации
func createTestConfig() *Config {
	cfg := &Config{
		Server: ServerConfig{
			Host:           "localhost",
			Port:           8080,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			IdleTimeout:    60 * time.Second,
			MaxHeaderBytes: 1 << 20,
		},
		Database: DatabaseConfig{
			Type: DatabaseTypePostgreSQL,
			PostgreSQL: PostgreSQLConfig{
				Host:          "localhost",
				Port:          5432,
				Database:      "testdb",
				User:          "testuser",
				Password:      "testpass",
				SSLMode:       "disable",
				MaxOpenConns:  25,
				MaxIdleConns:  5,
			},
		},
		Auth: AuthConfig{
			Enabled: true,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Metrics: MetricsConfig{
			Enabled:        true,
			PrometheusPath: "/metrics",
		},
	}
	
	// JWT nested struct
	cfg.Auth.JWT.Secret = "test-secret"
	cfg.Auth.JWT.AccessTokenExpiry = 15 * time.Minute
	cfg.Auth.JWT.RefreshTokenExpiry = 7 * 24 * time.Hour
	
	return cfg
}

func TestHybridConfigSource_GetString_FromDatabase(t *testing.T) {
	// Arrange
	mockSettings := new(MockSettingsManager)
	mockSettings.On("GetString", mock.Anything, "server.host").
		Return("db-host", nil)
	
	fallback := createTestConfig()
	source := NewHybridConfigSource(mockSettings, fallback, nil)
	
	// Act
	result, err := source.GetString(context.Background(), "server.host")
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "db-host", result, "Should return value from database")
	mockSettings.AssertExpectations(t)
}

func TestHybridConfigSource_GetString_FromYAMLFallback(t *testing.T) {
	// Arrange
	mockSettings := new(MockSettingsManager)
	mockSettings.On("GetString", mock.Anything, "server.host").
		Return("", assert.AnError) // DB lookup fails
	
	fallback := createTestConfig()
	source := NewHybridConfigSource(mockSettings, fallback, nil)
	
	// Act
	result, err := source.GetString(context.Background(), "server.host")
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "localhost", result, "Should fallback to YAML config")
	mockSettings.AssertExpectations(t)
}

func TestHybridConfigSource_GetInt_FromDatabase(t *testing.T) {
	// Arrange
	mockSettings := new(MockSettingsManager)
	mockSettings.On("GetInt", mock.Anything, "server.port").
		Return(9090, nil)
	
	fallback := createTestConfig()
	source := NewHybridConfigSource(mockSettings, fallback, nil)
	
	// Act
	result, err := source.GetInt(context.Background(), "server.port")
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 9090, result, "Should return value from database")
	mockSettings.AssertExpectations(t)
}

func TestHybridConfigSource_GetInt_FromYAMLFallback(t *testing.T) {
	// Arrange
	mockSettings := new(MockSettingsManager)
	mockSettings.On("GetInt", mock.Anything, "server.port").
		Return(0, assert.AnError)
	
	fallback := createTestConfig()
	source := NewHybridConfigSource(mockSettings, fallback, nil)
	
	// Act
	result, err := source.GetInt(context.Background(), "server.port")
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 8080, result, "Should fallback to YAML config")
	mockSettings.AssertExpectations(t)
}

func TestHybridConfigSource_GetBool_FromDatabase(t *testing.T) {
	// Arrange
	mockSettings := new(MockSettingsManager)
	mockSettings.On("GetBool", mock.Anything, "metrics.enabled").
		Return(false, nil)
	
	fallback := createTestConfig()
	source := NewHybridConfigSource(mockSettings, fallback, nil)
	
	// Act
	result, err := source.GetBool(context.Background(), "metrics.enabled")
	
	// Assert
	assert.NoError(t, err)
	assert.False(t, result, "Should return value from database")
	mockSettings.AssertExpectations(t)
}

func TestHybridConfigSource_GetDuration_FromDatabase(t *testing.T) {
	// Arrange
	mockSettings := new(MockSettingsManager)
	mockSettings.On("GetDuration", mock.Anything, "server.read_timeout").
		Return(60*time.Second, nil)
	
	fallback := createTestConfig()
	source := NewHybridConfigSource(mockSettings, fallback, nil)
	
	// Act
	result, err := source.GetDuration(context.Background(), "server.read_timeout")
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 60*time.Second, result, "Should return value from database")
	mockSettings.AssertExpectations(t)
}

func TestHybridConfigSource_GetDuration_FromYAMLFallback(t *testing.T) {
	// Arrange
	mockSettings := new(MockSettingsManager)
	mockSettings.On("GetDuration", mock.Anything, "server.read_timeout").
		Return(time.Duration(0), assert.AnError)
	
	fallback := createTestConfig()
	source := NewHybridConfigSource(mockSettings, fallback, nil)
	
	// Act
	result, err := source.GetDuration(context.Background(), "server.read_timeout")
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 30*time.Second, result, "Should fallback to YAML config")
	mockSettings.AssertExpectations(t)
}

func TestHybridConfigSource_GetStringWithDefault(t *testing.T) {
	// Arrange
	mockSettings := new(MockSettingsManager)
	mockSettings.On("GetString", mock.Anything, "nonexistent.key").
		Return("", assert.AnError)
	
	fallback := createTestConfig()
	source := NewHybridConfigSource(mockSettings, fallback, nil)
	
	// Act
	result := source.GetStringWithDefault(context.Background(), "nonexistent.key", "default-value")
	
	// Assert
	assert.Equal(t, "default-value", result, "Should return default when key not found")
	mockSettings.AssertExpectations(t)
}

func TestHybridConfigSource_GetIntWithDefault(t *testing.T) {
	// Arrange
	mockSettings := new(MockSettingsManager)
	mockSettings.On("GetInt", mock.Anything, "nonexistent.key").
		Return(0, assert.AnError)
	
	fallback := createTestConfig()
	source := NewHybridConfigSource(mockSettings, fallback, nil)
	
	// Act
	result := source.GetIntWithDefault(context.Background(), "nonexistent.key", 12345)
	
	// Assert
	assert.Equal(t, 12345, result, "Should return default when key not found")
	mockSettings.AssertExpectations(t)
}

func TestHybridConfigSource_GetStringSlice(t *testing.T) {
	// Arrange
	mockSettings := new(MockSettingsManager)
	mockSettings.On("GetString", mock.Anything, "server.tls.hosts").
		Return("localhost,127.0.0.1,::1", nil)
	
	fallback := createTestConfig()
	source := NewHybridConfigSource(mockSettings, fallback, nil)
	
	// Act
	result, err := source.GetStringSlice(context.Background(), "server.tls.hosts")
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, []string{"localhost", "127.0.0.1", "::1"}, result)
	mockSettings.AssertExpectations(t)
}

func TestHybridConfigSource_GetStringSlice_EmptyString(t *testing.T) {
	// Arrange
	mockSettings := new(MockSettingsManager)
	mockSettings.On("GetString", mock.Anything, "empty.key").
		Return("", nil)
	
	fallback := createTestConfig()
	source := NewHybridConfigSource(mockSettings, fallback, nil)
	
	// Act
	result, err := source.GetStringSlice(context.Background(), "empty.key")
	
	// Assert
	assert.NoError(t, err)
	assert.Empty(t, result, "Should return empty slice for empty string")
	mockSettings.AssertExpectations(t)
}

func TestHybridConfigSource_Reload(t *testing.T) {
	// Arrange
	mockSettings := new(MockSettingsManager)
	mockSettings.On("InvalidateCache").Return()
	
	fallback := createTestConfig()
	source := NewHybridConfigSource(mockSettings, fallback, nil)
	
	// Act
	err := source.Reload(context.Background())
	
	// Assert
	assert.NoError(t, err)
	mockSettings.AssertExpectations(t)
}

func TestHybridConfigSource_NoDatabaseSource(t *testing.T) {
	// Arrange: No SettingsManager, только YAML fallback
	fallback := createTestConfig()
	source := NewHybridConfigSource(nil, fallback, nil)
	
	// Act
	result, err := source.GetString(context.Background(), "server.host")
	
	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "localhost", result, "Should work with YAML only")
}

func TestHybridConfigSource_NoFallback(t *testing.T) {
	// Arrange: SettingsManager returns error, no YAML fallback
	mockSettings := new(MockSettingsManager)
	mockSettings.On("GetString", mock.Anything, "nonexistent.key").
		Return("", assert.AnError)
	
	source := NewHybridConfigSource(mockSettings, nil, nil)
	
	// Act
	_, err := source.GetString(context.Background(), "nonexistent.key")
	
	// Assert
	assert.Error(t, err, "Should return error when both DB and fallback fail")
	mockSettings.AssertExpectations(t)
}

func TestHybridConfigSource_DatabaseConfig(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"Database Type", "database.type", "postgresql"},
		{"Database Host", "database.postgresql.host", "localhost"},
		{"Database Port", "database.postgresql.port", "5432"},
		{"Database Name", "database.postgresql.database", "testdb"},
		{"Database User", "database.postgresql.user", "testuser"},
		{"Database SSL Mode", "database.postgresql.ssl_mode", "disable"},
	}
	
	fallback := createTestConfig()
	source := NewHybridConfigSource(nil, fallback, nil)
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := source.GetString(context.Background(), tt.key)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHybridConfigSource_AuthConfig(t *testing.T) {
	fallback := createTestConfig()
	source := NewHybridConfigSource(nil, fallback, nil)
	
	// Test JWT Secret
	secret, err := source.GetString(context.Background(), "auth.jwt.secret")
	assert.NoError(t, err)
	assert.Equal(t, "test-secret", secret)
	
	// Test JWT Access Token Expiry
	expiry, err := source.GetDuration(context.Background(), "auth.jwt.access_token_expiry")
	assert.NoError(t, err)
	assert.Equal(t, 15*time.Minute, expiry)
}

func TestHybridConfigSource_LoggingConfig(t *testing.T) {
	fallback := createTestConfig()
	source := NewHybridConfigSource(nil, fallback, nil)
	
	// Test logging level
	level, err := source.GetString(context.Background(), "logging.level")
	assert.NoError(t, err)
	assert.Equal(t, "info", level)
	
	// Test logging format
	format, err := source.GetString(context.Background(), "logging.format")
	assert.NoError(t, err)
	assert.Equal(t, "json", format)
}

func TestConfigWrapper_GetString(t *testing.T) {
	mockSettings := new(MockSettingsManager)
	mockSettings.On("GetString", mock.Anything, "test.key").
		Return("db-value", nil)
	
	fallback := createTestConfig()
	source := NewHybridConfigSource(mockSettings, fallback, logrus.New())
	wrapper := NewConfigWrapper(fallback, source, logrus.New())
	
	result := wrapper.GetString("test.key")
	assert.Equal(t, "db-value", result)
	mockSettings.AssertExpectations(t)
}

func TestConfigWrapper_IsBootstrapMode(t *testing.T) {
	// With ConfigSource
	wrapper1 := NewConfigWrapper(createTestConfig(), &HybridConfigSource{}, nil)
	assert.True(t, wrapper1.IsBootstrapMode())
	
	// Without ConfigSource
	wrapper2 := NewConfigWrapper(createTestConfig(), nil, nil)
	assert.False(t, wrapper2.IsBootstrapMode())
}

func TestConfigWrapper_Reload(t *testing.T) {
	mockSettings := new(MockSettingsManager)
	mockSettings.On("InvalidateCache").Return()
	
	source := NewHybridConfigSource(mockSettings, createTestConfig(), nil)
	wrapper := NewConfigWrapper(createTestConfig(), source, nil)
	
	err := wrapper.Reload()
	assert.NoError(t, err)
	mockSettings.AssertExpectations(t)
}

// Benchmark tests
func BenchmarkHybridConfigSource_GetString_Database(b *testing.B) {
	mockSettings := new(MockSettingsManager)
	mockSettings.On("GetString", mock.Anything, "server.host").
		Return("db-host", nil)
	
	source := NewHybridConfigSource(mockSettings, createTestConfig(), nil)
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		source.GetString(ctx, "server.host")
	}
}

func BenchmarkHybridConfigSource_GetString_YAMLFallback(b *testing.B) {
	source := NewHybridConfigSource(nil, createTestConfig(), nil)
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		source.GetString(ctx, "server.host")
	}
}


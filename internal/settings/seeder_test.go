// Package settings - Configuration Seeder Tests
// Version: v3.0.9 - Phase 2
package settings

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"aigateway/internal/config"
)

// MockStorage is a mock implementation of Storage interface
type MockStorage struct {
	mock.Mock
	settings    map[string]*Setting // For simple tests
	useSimpleMode bool // Flag to indicate simple mode vs mock mode
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		settings:    make(map[string]*Setting),
		useSimpleMode: true, // CLI tests use simple mode
	}
}

func (m *MockStorage) GetAllSettings(ctx context.Context) ([]*Setting, error) {
	// If in simple mode, return from settings map
	if m.useSimpleMode {
		result := make([]*Setting, 0, len(m.settings))
		for _, s := range m.settings {
			result = append(result, s)
		}
		return result, nil
	}
	
	// Otherwise use mock.Called (for seeder tests)
	args := m.Called(ctx)
	return args.Get(0).([]*Setting), args.Error(1)
}

func (m *MockStorage) GetSettingsByCategory(ctx context.Context, category SettingCategory) ([]*Setting, error) {
	args := m.Called(ctx, category)
	return args.Get(0).([]*Setting), args.Error(1)
}

func (m *MockStorage) GetSetting(ctx context.Context, id string) (*Setting, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Setting), args.Error(1)
}

func (m *MockStorage) UpsertSetting(ctx context.Context, setting *Setting) error {
	args := m.Called(ctx, setting)
	return args.Error(0)
}

func (m *MockStorage) UpdateSettingValue(ctx context.Context, id, value, updatedBy string) error {
	args := m.Called(ctx, id, value, updatedBy)
	return args.Error(0)
}

func (m *MockStorage) DeleteSetting(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockStorage) BulkUpsertSettings(ctx context.Context, settings []*Setting) error {
	args := m.Called(ctx, settings)
	return args.Error(0)
}

func TestNewConfigSeeder(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during test
	mockStorage := new(MockStorage)

	seeder := NewConfigSeeder(mockStorage, logger)

	assert.NotNil(t, seeder)
	assert.Equal(t, mockStorage, seeder.storage)
	assert.Equal(t, logger, seeder.logger)
}

func TestSeedFromYAML_SkipIfAlreadySeeded(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	mockStorage := new(MockStorage)

	// Mock: settings table already has data
	existingSettings := []*Setting{
		{ID: "server.host", Category: CategoryServer, Key: "host", Value: "0.0.0.0"},
	}
	mockStorage.On("GetAllSettings", mock.Anything).Return(existingSettings, nil)

	seeder := NewConfigSeeder(mockStorage, logger)
	cfg := &config.Config{}

	ctx := context.Background()
	count, err := seeder.SeedFromYAML(ctx, cfg)

	assert.NoError(t, err)
	assert.Equal(t, 0, count, "Should skip seeding if settings already exist")
	mockStorage.AssertExpectations(t)
}

func TestSeedFromYAML_SuccessfulSeed(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	mockStorage := new(MockStorage)

	// Mock: settings table is empty
	mockStorage.On("GetAllSettings", mock.Anything).Return([]*Setting{}, nil)

	// Mock: UpsertSetting succeeds for all settings
	mockStorage.On("UpsertSetting", mock.Anything, mock.AnythingOfType("*settings.Setting")).Return(nil)

	seeder := NewConfigSeeder(mockStorage, logger)

	// Create test config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "localhost",
			Port:         8085,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Auth: config.AuthConfig{
			Enabled: true,
			JWT: struct {
				Secret             string        `mapstructure:"secret"`
				Expiry             time.Duration `mapstructure:"expiry"`
				AccessTokenExpiry  time.Duration `mapstructure:"access_token_expiry"`
				RefreshTokenExpiry time.Duration `mapstructure:"refresh_token_expiry"`
			}{
				Expiry: 24 * time.Hour,
			},
			RateLimiting: struct {
				Enabled                  bool `mapstructure:"enabled"`
				DefaultRequestsPerMinute int  `mapstructure:"default_requests_per_minute"`
				DefaultRequestsPerHour   int  `mapstructure:"default_requests_per_hour"`
				Redis                    struct {
					Enabled   bool   `mapstructure:"enabled"`
					URL       string `mapstructure:"url"`
					KeyPrefix string `mapstructure:"key_prefix"`
				} `mapstructure:"redis"`
			}{
				Enabled:                  true,
				DefaultRequestsPerMinute: 30,
				DefaultRequestsPerHour:   500,
			},
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Yzma: config.YzmaConfig{
			Enabled:   true,
			ModelsDir: "./models",
		},
		Database: config.DatabaseConfig{
			Type: "postgresql",
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
		},
	}

	ctx := context.Background()
	count, err := seeder.SeedFromYAML(ctx, cfg)

	assert.NoError(t, err)
	assert.Greater(t, count, 10, "Should seed at least 10 settings")
	mockStorage.AssertExpectations(t)
}

func TestMapConfigToSettings_ServerSettings(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	mockStorage := new(MockStorage)
	seeder := NewConfigSeeder(mockStorage, logger)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:         "0.0.0.0",
			Port:         8085,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 45 * time.Second,
		},
	}

	settings := seeder.MapConfigToSettings(cfg)

	// Verify server settings
	serverHost := findSetting(settings, "server.host")
	assert.NotNil(t, serverHost)
	assert.Equal(t, "0.0.0.0", serverHost.Value)
	assert.Equal(t, CategoryServer, serverHost.Category)
	assert.True(t, serverHost.IsMigrated)
	assert.False(t, serverHost.IsEditable, "Host should not be editable (requires restart)")

	serverPort := findSetting(settings, "server.port")
	assert.NotNil(t, serverPort)
	assert.Equal(t, "8085", serverPort.Value)
	assert.Equal(t, TypeInt, serverPort.Type)

	readTimeout := findSetting(settings, "server.read_timeout")
	assert.NotNil(t, readTimeout)
	assert.Equal(t, "30s", readTimeout.Value)
	assert.Equal(t, TypeDuration, readTimeout.Type)
	assert.True(t, readTimeout.IsEditable, "Timeout should be editable")

	writeTimeout := findSetting(settings, "server.write_timeout")
	assert.NotNil(t, writeTimeout)
	assert.Equal(t, "45s", writeTimeout.Value)

	tlsEnabled := findSetting(settings, "server.tls.enabled")
	assert.NotNil(t, tlsEnabled)
	assert.Equal(t, "false", tlsEnabled.Value)
	assert.Equal(t, TypeBool, tlsEnabled.Type)
}

func TestMapConfigToSettings_AuthSettings(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	mockStorage := new(MockStorage)
	seeder := NewConfigSeeder(mockStorage, logger)

	cfg := &config.Config{
		Auth: config.AuthConfig{
			Enabled: true,
			JWT: struct {
				Secret             string        `mapstructure:"secret"`
				Expiry             time.Duration `mapstructure:"expiry"`
				AccessTokenExpiry  time.Duration `mapstructure:"access_token_expiry"`
				RefreshTokenExpiry time.Duration `mapstructure:"refresh_token_expiry"`
			}{
				Expiry: 24 * time.Hour,
			},
			RateLimiting: struct {
				Enabled                  bool `mapstructure:"enabled"`
				DefaultRequestsPerMinute int  `mapstructure:"default_requests_per_minute"`
				DefaultRequestsPerHour   int  `mapstructure:"default_requests_per_hour"`
				Redis                    struct {
					Enabled   bool   `mapstructure:"enabled"`
					URL       string `mapstructure:"url"`
					KeyPrefix string `mapstructure:"key_prefix"`
				} `mapstructure:"redis"`
			}{
				Enabled:                  true,
				DefaultRequestsPerMinute: 60,
				DefaultRequestsPerHour:   1000,
			},
		},
	}

	settings := seeder.MapConfigToSettings(cfg)

	authEnabled := findSetting(settings, "auth.enabled")
	assert.NotNil(t, authEnabled)
	assert.Equal(t, "true", authEnabled.Value)
	assert.False(t, authEnabled.IsEditable, "Auth enabled should not be editable")

	jwtExpiry := findSetting(settings, "auth.jwt.expiration")
	assert.NotNil(t, jwtExpiry)
	assert.Equal(t, "24h0m0s", jwtExpiry.Value)
	assert.True(t, jwtExpiry.IsEditable, "JWT expiry should be editable")

	rateLimitRPM := findSetting(settings, "auth.rate_limiting.default_requests_per_minute")
	assert.NotNil(t, rateLimitRPM)
	assert.Equal(t, "60", rateLimitRPM.Value)
	assert.True(t, rateLimitRPM.IsEditable, "Rate limit should be hot-reloadable")

	rateLimitRPH := findSetting(settings, "auth.rate_limiting.default_requests_per_hour")
	assert.NotNil(t, rateLimitRPH)
	assert.Equal(t, "1000", rateLimitRPH.Value)
}

func TestMapConfigToSettings_LoggingSettings(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	mockStorage := new(MockStorage)
	seeder := NewConfigSeeder(mockStorage, logger)

	cfg := &config.Config{
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "text",
		},
	}

	settings := seeder.MapConfigToSettings(cfg)

	logLevel := findSetting(settings, "logging.level")
	assert.NotNil(t, logLevel)
	assert.Equal(t, "debug", logLevel.Value)
	assert.Equal(t, "^(debug|info|warn|error)$", logLevel.ValidationRule)
	assert.True(t, logLevel.IsEditable, "Log level should be hot-reloadable")

	logFormat := findSetting(settings, "logging.format")
	assert.NotNil(t, logFormat)
	assert.Equal(t, "text", logFormat.Value)
	assert.Equal(t, "^(json|text)$", logFormat.ValidationRule)
}

func TestMapConfigToSettings_AllMigratedFlag(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	mockStorage := new(MockStorage)
	seeder := NewConfigSeeder(mockStorage, logger)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8085,
		},
		Auth: config.AuthConfig{
			Enabled: true,
		},
		Logging: config.LoggingConfig{
			Level: "info",
		},
	}

	settings := seeder.MapConfigToSettings(cfg)

	// All settings should be marked as migrated
	for _, setting := range settings {
		assert.True(t, setting.IsMigrated, "Setting %s should be marked as migrated", setting.ID)
	}
}

// Helper function to find setting by ID
func findSetting(settings []Setting, id string) *Setting {
	for i := range settings {
		if settings[i].ID == id {
			return &settings[i]
		}
	}
	return nil
}

// ============================================================================
// CLI Commands Tests
// ============================================================================

func TestExportCommand(t *testing.T) {
	ctx := context.Background()
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockStorage := NewMockStorage()

	// Populate with test data
	testSettings := []*Setting{
		{
			ID:          "server.host",
			Category:    CategoryServer,
			Key:         "host",
			Value:       "0.0.0.0",
			Type:        TypeString,
			Description: "Server host",
			IsEditable:  true,
		},
		{
			ID:          "server.port",
			Category:    CategoryServer,
			Key:         "port",
			Value:       "8080",
			Type:        TypeInt,
			Description: "Server port",
			IsEditable:  true,
		},
	}

	for _, s := range testSettings {
		mockStorage.settings[s.ID] = s
	}

	// Create temp file for export
	tmpFile := "test_export.yaml"
	defer os.Remove(tmpFile)

	// Test export
	err := ExportCommand(ctx, mockStorage, logger, tmpFile)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(tmpFile)
	assert.NoError(t, err, "Export file should exist")

	// Verify file content
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "server.host")
	assert.Contains(t, string(content), "0.0.0.0")
}

func TestListCommand_EmptyDatabase(t *testing.T) {
	ctx := context.Background()
	mockStorage := NewMockStorage()

	// Test with empty database
	err := ListCommand(ctx, mockStorage, "")
	assert.NoError(t, err)
}

func TestListCommand_WithCategory(t *testing.T) {
	ctx := context.Background()
	mockStorage := NewMockStorage()

	// Add settings
	mockStorage.settings["server.host"] = &Setting{
		ID:       "server.host",
		Category: CategoryServer,
		Value:    "0.0.0.0",
	}
	mockStorage.settings["auth.enabled"] = &Setting{
		ID:       "auth.enabled",
		Category: CategoryAuth,
		Value:    "true",
	}

	// Test category filter
	err := ListCommand(ctx, mockStorage, "server")
	assert.NoError(t, err)
}


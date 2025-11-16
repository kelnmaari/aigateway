// Package ui - Settings Handler Tests
// Version: v3.0.9 - Phase 3
package ui

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"aigateway/internal/settings"
)

// Unit tests for validation functions

func TestValidateSettingValue_Duration(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	handler := &SettingsHandler{
		manager: nil, // Not needed for validation tests
		logger:  logger,
	}
	
	setting := &settings.Setting{
		Type: settings.TypeDuration,
	}
	
	// Valid durations
	assert.NoError(t, handler.validateSettingValue(setting, "30s"))
	assert.NoError(t, handler.validateSettingValue(setting, "5m"))
	assert.NoError(t, handler.validateSettingValue(setting, "1h"))
	assert.NoError(t, handler.validateSettingValue(setting, "1h30m"))
	assert.NoError(t, handler.validateSettingValue(setting, "24h"))
	
	// Invalid duration
	assert.Error(t, handler.validateSettingValue(setting, "invalid"))
	assert.Error(t, handler.validateSettingValue(setting, "30"))
	assert.Error(t, handler.validateSettingValue(setting, "abc"))
}

func TestValidateSettingValue_Int(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	handler := &SettingsHandler{
		manager: nil,
		logger:  logger,
	}
	
	setting := &settings.Setting{
		Type: settings.TypeInt,
	}
	
	// Valid integers
	assert.NoError(t, handler.validateSettingValue(setting, "0"))
	assert.NoError(t, handler.validateSettingValue(setting, "123"))
	assert.NoError(t, handler.validateSettingValue(setting, "-456"))
	assert.NoError(t, handler.validateSettingValue(setting, "9999"))
	
	// Invalid integers
	assert.Error(t, handler.validateSettingValue(setting, "abc"))
	assert.Error(t, handler.validateSettingValue(setting, "12.34"))
	assert.Error(t, handler.validateSettingValue(setting, "12.5"))
}

func TestValidateSettingValue_Float(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	handler := &SettingsHandler{
		manager: nil,
		logger:  logger,
	}
	
	setting := &settings.Setting{
		Type: settings.TypeFloat,
	}
	
	// Valid floats
	assert.NoError(t, handler.validateSettingValue(setting, "0.5"))
	assert.NoError(t, handler.validateSettingValue(setting, "123.456"))
	assert.NoError(t, handler.validateSettingValue(setting, "-78.9"))
	assert.NoError(t, handler.validateSettingValue(setting, "0"))
	assert.NoError(t, handler.validateSettingValue(setting, "123"))
	
	// Invalid floats
	assert.Error(t, handler.validateSettingValue(setting, "abc"))
	assert.Error(t, handler.validateSettingValue(setting, "12.34.56"))
}

func TestValidateSettingValue_Bool(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	handler := &SettingsHandler{
		manager: nil,
		logger:  logger,
	}
	
	setting := &settings.Setting{
		Type: settings.TypeBool,
	}
	
	// Valid booleans
	assert.NoError(t, handler.validateSettingValue(setting, "true"))
	assert.NoError(t, handler.validateSettingValue(setting, "false"))
	
	// Invalid booleans
	assert.Error(t, handler.validateSettingValue(setting, "yes"))
	assert.Error(t, handler.validateSettingValue(setting, "no"))
	assert.Error(t, handler.validateSettingValue(setting, "1"))
	assert.Error(t, handler.validateSettingValue(setting, "0"))
	assert.Error(t, handler.validateSettingValue(setting, "TRUE"))
	assert.Error(t, handler.validateSettingValue(setting, "False"))
}

func TestValidateSettingValue_String(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	handler := &SettingsHandler{
		manager: nil,
		logger:  logger,
	}
	
	setting := &settings.Setting{
		Type: settings.TypeString,
	}
	
	// Any string is valid
	assert.NoError(t, handler.validateSettingValue(setting, "hello"))
	assert.NoError(t, handler.validateSettingValue(setting, ""))
	assert.NoError(t, handler.validateSettingValue(setting, "123"))
}

func TestValidateSettingValue_RegexValidation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	handler := &SettingsHandler{
		manager: nil,
		logger:  logger,
	}
	
	setting := &settings.Setting{
		Type:           settings.TypeString,
		ValidationRule: "^(debug|info|warn|error)$", // Log level regex
	}
	
	// Valid values matching regex
	assert.NoError(t, handler.validateSettingValue(setting, "debug"))
	assert.NoError(t, handler.validateSettingValue(setting, "info"))
	assert.NoError(t, handler.validateSettingValue(setting, "warn"))
	assert.NoError(t, handler.validateSettingValue(setting, "error"))
	
	// Invalid values not matching regex
	assert.Error(t, handler.validateSettingValue(setting, "trace"))
	assert.Error(t, handler.validateSettingValue(setting, "fatal"))
	assert.Error(t, handler.validateSettingValue(setting, "DEBUG"))
	assert.Error(t, handler.validateSettingValue(setting, "info "))
}


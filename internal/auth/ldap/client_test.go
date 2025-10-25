// Package ldap provides LDAP/Active Directory authentication client tests
// Version: 1.11.3+ (Enterprise Suite - LDAP Integration)
package ldap

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"ollama-openai-proxy/internal/config"
)

func TestNewClient_DisabledLDAP(t *testing.T) {
	cfg := &config.LDAPConfig{
		Enabled: false,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	_, err := NewClient(cfg, logger)
	if err == nil {
		t.Error("Expected error when LDAP is disabled")
	}
	if err.Error() != "LDAP is not enabled" {
		t.Errorf("Expected 'LDAP is not enabled' error, got: %v", err)
	}
}

func TestNewClient_MissingURL(t *testing.T) {
	cfg := &config.LDAPConfig{
		Enabled:      true,
		URL:          "", // Missing
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "password",
		UserBaseDN:   "ou=users,dc=example,dc=com",
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	_, err := NewClient(cfg, logger)
	if err == nil {
		t.Error("Expected error when URL is missing")
	}
	if err.Error() != "LDAP URL is required" {
		t.Errorf("Expected 'LDAP URL is required' error, got: %v", err)
	}
}

func TestNewClient_MissingBindDN(t *testing.T) {
	cfg := &config.LDAPConfig{
		Enabled:      true,
		URL:          "ldap://localhost:389",
		BindDN:       "", // Missing
		BindPassword: "password",
		UserBaseDN:   "ou=users,dc=example,dc=com",
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	_, err := NewClient(cfg, logger)
	if err == nil {
		t.Error("Expected error when BindDN is missing")
	}
	if err.Error() != "LDAP Bind DN is required" {
		t.Errorf("Expected 'LDAP Bind DN is required' error, got: %v", err)
	}
}

func TestNewClient_MissingBindPassword(t *testing.T) {
	cfg := &config.LDAPConfig{
		Enabled:      true,
		URL:          "ldap://localhost:389",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "", // Missing
		UserBaseDN:   "ou=users,dc=example,dc=com",
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	_, err := NewClient(cfg, logger)
	if err == nil {
		t.Error("Expected error when BindPassword is missing")
	}
	if err.Error() != "LDAP Bind Password is required" {
		t.Errorf("Expected 'LDAP Bind Password is required' error, got: %v", err)
	}
}

func TestNewClient_MissingUserBaseDN(t *testing.T) {
	cfg := &config.LDAPConfig{
		Enabled:      true,
		URL:          "ldap://localhost:389",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "password",
		UserBaseDN:   "", // Missing
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	_, err := NewClient(cfg, logger)
	if err == nil {
		t.Error("Expected error when UserBaseDN is missing")
	}
	if err.Error() != "LDAP User Base DN is required" {
		t.Errorf("Expected 'LDAP User Base DN is required' error, got: %v", err)
	}
}

func TestNewClient_ValidConfigWithDefaults(t *testing.T) {
	cfg := &config.LDAPConfig{
		Enabled:      true,
		URL:          "ldap://localhost:389",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "password",
		UserBaseDN:   "ou=users,dc=example,dc=com",
		// Defaults should be set by NewClient
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	client, err := NewClient(cfg, logger)
	if err != nil {
		t.Fatalf("Expected no error with valid config, got: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be initialized")
	}

	// Check defaults
	if cfg.UserFilter != "(uid={username})" {
		t.Errorf("Expected default UserFilter to be '(uid={username})', got: %s", cfg.UserFilter)
	}
	if cfg.UserIDAttribute != "uid" {
		t.Errorf("Expected default UserIDAttribute to be 'uid', got: %s", cfg.UserIDAttribute)
	}
	if cfg.UserEmailAttribute != "mail" {
		t.Errorf("Expected default UserEmailAttribute to be 'mail', got: %s", cfg.UserEmailAttribute)
	}
	if cfg.UserNameAttribute != "cn" {
		t.Errorf("Expected default UserNameAttribute to be 'cn', got: %s", cfg.UserNameAttribute)
	}
	if cfg.GroupFilter != "(member={userdn})" {
		t.Errorf("Expected default GroupFilter to be '(member={userdn})', got: %s", cfg.GroupFilter)
	}
	if cfg.GroupNameAttribute != "cn" {
		t.Errorf("Expected default GroupNameAttribute to be 'cn', got: %s", cfg.GroupNameAttribute)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Expected default Timeout to be 30s, got: %v", cfg.Timeout)
	}
}

func TestNewClient_ValidConfigWithCustomValues(t *testing.T) {
	cfg := &config.LDAPConfig{
		Enabled:            true,
		URL:                "ldap://localhost:389",
		BindDN:             "cn=admin,dc=example,dc=com",
		BindPassword:       "password",
		UserBaseDN:         "ou=users,dc=example,dc=com",
		UserFilter:         "(sAMAccountName={username})", // Active Directory
		UserIDAttribute:    "sAMAccountName",
		UserEmailAttribute: "mail",
		UserNameAttribute:  "displayName",
		GroupBaseDN:        "ou=groups,dc=example,dc=com",
		GroupFilter:        "(memberOf={userdn})",
		GroupNameAttribute: "cn",
		Timeout:            60 * time.Second,
		StartTLS:           true,
		SkipVerify:         false,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	client, err := NewClient(cfg, logger)
	if err != nil {
		t.Fatalf("Expected no error with valid config, got: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be initialized")
	}

	// Check that custom values are preserved
	if cfg.UserFilter != "(sAMAccountName={username})" {
		t.Errorf("Expected UserFilter to be '(sAMAccountName={username})', got: %s", cfg.UserFilter)
	}
	if cfg.UserIDAttribute != "sAMAccountName" {
		t.Errorf("Expected UserIDAttribute to be 'sAMAccountName', got: %s", cfg.UserIDAttribute)
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("Expected Timeout to be 60s, got: %v", cfg.Timeout)
	}
}

// TestAuthenticate_EmptyCredentials tests that Authenticate returns error for empty username or password
func TestAuthenticate_EmptyCredentials(t *testing.T) {
	cfg := &config.LDAPConfig{
		Enabled:      true,
		URL:          "ldap://localhost:389",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "password",
		UserBaseDN:   "ou=users,dc=example,dc=com",
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	client, err := NewClient(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name     string
		username string
		password string
	}{
		{"EmptyUsername", "", "password"},
		{"EmptyPassword", "user", ""},
		{"BothEmpty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.Authenticate(tt.username, tt.password)
			if err == nil {
				t.Error("Expected error for empty credentials")
			}
			if err.Error() != "username and password are required" {
				t.Errorf("Expected 'username and password are required' error, got: %v", err)
			}
		})
	}
}

// Note: Full authentication tests require a real LDAP server or mock LDAP server
// These tests only cover configuration validation and basic input validation


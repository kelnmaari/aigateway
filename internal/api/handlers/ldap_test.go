// Package handlers provides LDAP authentication HTTP handlers tests
// Version: 1.11.3+ (Enterprise Suite - LDAP Integration)
package handlers

import (
	"testing"

	"github.com/sirupsen/logrus"
	"aigateway/internal/config"
)

func TestLDAPHandler_isAdminGroup(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			LDAP: config.LDAPConfig{
				Enabled: true,
				TenantProvisioning: config.TenantProvisioningConfig{
					GroupMapping: config.GroupMappingConfig{
						AdminGroups: []string{
							"Domain Admins",
							"Administrators",
							"Engineering-Admins",
						},
					},
				},
			},
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	handler := &LDAPHandler{
		config: cfg,
		logger: logger,
	}

	tests := []struct {
		name     string
		groups   []string
		expected bool
	}{
		{
			name:     "ExactMatch_DomainAdmins",
			groups:   []string{"Domain Admins"},
			expected: true,
		},
		{
			name:     "ExactMatch_Administrators",
			groups:   []string{"Administrators"},
			expected: true,
		},
		{
			name:     "ExactMatch_EngineeringAdmins",
			groups:   []string{"Engineering-Admins"},
			expected: true,
		},
		{
			name:     "CaseInsensitive_DomainAdmins",
			groups:   []string{"domain admins"},
			expected: true,
		},
		{
			name:     "CaseInsensitive_Administrators",
			groups:   []string{"ADMINISTRATORS"},
			expected: true,
		},
		{
			name:     "PartialMatch_DomainAdminsInPath",
			groups:   []string{"CN=Domain Admins,OU=Groups,DC=example,DC=com"},
			expected: true,
		},
		{
			name:     "PartialMatch_EngineeringInGroup",
			groups:   []string{"Engineering-Admins-Team"},
			expected: true,
		},
		{
			name:     "MultipleGroups_OneAdmin",
			groups:   []string{"Users", "Developers", "Domain Admins"},
			expected: true,
		},
		{
			name:     "MultipleGroups_NoAdmin",
			groups:   []string{"Users", "Developers", "Engineers"},
			expected: false,
		},
		{
			name:     "EmptyGroups",
			groups:   []string{},
			expected: false,
		},
		{
			name:     "NilGroups",
			groups:   nil,
			expected: false,
		},
		{
			name:     "NoMatch",
			groups:   []string{"Users", "Guests"},
			expected: false,
		},
		{
			name:     "PartialMatchNotAdmin",
			groups:   []string{"Domain Users", "Engineering"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.isAdminGroup(tt.groups)
			if result != tt.expected {
				t.Errorf("isAdminGroup(%v) = %v, expected %v", tt.groups, result, tt.expected)
			}
		})
	}
}

func TestLDAPHandler_isAdminGroup_EmptyAdminGroups(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			LDAP: config.LDAPConfig{
				Enabled: true,
				TenantProvisioning: config.TenantProvisioningConfig{
					GroupMapping: config.GroupMappingConfig{
						AdminGroups: []string{}, // Empty admin groups
					},
				},
			},
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	handler := &LDAPHandler{
		config: cfg,
		logger: logger,
	}

	// With empty admin groups, no user should be considered admin
	result := handler.isAdminGroup([]string{"Domain Admins", "Administrators"})
	if result != false {
		t.Errorf("Expected false when admin groups list is empty, got: %v", result)
	}
}

func TestLDAPHandler_isAdminGroup_ActiveDirectoryFormat(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			LDAP: config.LDAPConfig{
				Enabled: true,
				TenantProvisioning: config.TenantProvisioningConfig{
					GroupMapping: config.GroupMappingConfig{
						AdminGroups: []string{
							"APP-Admins",
						},
					},
				},
			},
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	handler := &LDAPHandler{
		config: cfg,
		logger: logger,
	}

	tests := []struct {
		name     string
		groups   []string
		expected bool
	}{
		{
			name:     "AD_FullDN",
			groups:   []string{"CN=APP-Admins,OU=Application Groups,OU=Groups,DC=company,DC=com"},
			expected: true,
		},
		{
			name:     "AD_PartialDN",
			groups:   []string{"CN=APP-Admins,OU=Groups"},
			expected: true,
		},
		{
			name:     "AD_JustCN",
			groups:   []string{"CN=APP-Admins"},
			expected: true,
		},
		{
			name:     "AD_NoMatch",
			groups:   []string{"CN=APP-Users,OU=Groups,DC=company,DC=com"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.isAdminGroup(tt.groups)
			if result != tt.expected {
				t.Errorf("isAdminGroup(%v) = %v, expected %v", tt.groups, result, tt.expected)
			}
		})
	}
}

// Note: Full handler tests (HandleLogin, HandleTestConnection) require:
// - Mock LDAP client
// - Mock database
// - Mock JWT manager
// - Gin test context
// These integration tests can be added when needed



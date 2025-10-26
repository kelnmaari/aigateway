// Package oidc provides tests for tenant provisioning from OIDC groups
package oidc

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/models"
)

// TestParseGroups_DirectMode tests direct 1:1 group mapping
func TestParseGroups_DirectMode(t *testing.T) {
	groups := []string{
		"engineering",
		"sales",
		"support",
	}

	cfg := &config.GroupMappingConfig{
		Mode:        "direct",
		AdminGroups: []string{"admins"},
	}

	mappings := ParseGroups(groups, cfg)

	assert.Len(t, mappings, 3)
	assert.Equal(t, "engineering", mappings[0].TenantName)
	assert.Equal(t, "sales", mappings[1].TenantName)
	assert.Equal(t, "support", mappings[2].TenantName)
	assert.Equal(t, models.TenantRoleMember, mappings[0].Role)
}

// TestParseGroups_PrefixMode tests prefix-based group mapping
func TestParseGroups_PrefixMode(t *testing.T) {
	groups := []string{
		"/engineering/backend",
		"/engineering/frontend",
		"/sales/emea",
		"/other/team",
	}

	cfg := &config.GroupMappingConfig{
		Mode:        "prefix",
		Prefix:      "/engineering/",
		AdminGroups: []string{},
	}

	mappings := ParseGroups(groups, cfg)

	assert.Len(t, mappings, 2)
	assert.Equal(t, "backend", mappings[0].TenantName)
	assert.Equal(t, "frontend", mappings[1].TenantName)
}

// TestParseGroups_AdminRoles tests admin role assignment
func TestParseGroups_AdminRoles(t *testing.T) {
	groups := []string{
		"engineering",
		"engineering-admins",
		"sales",
	}

	cfg := &config.GroupMappingConfig{
		Mode:        "direct",
		AdminGroups: []string{"engineering-admins", "/admins"},
	}

	mappings := ParseGroups(groups, cfg)

	assert.Len(t, mappings, 3)

	// Find engineering-admins mapping
	var adminMapping *TenantMapping
	for i := range mappings {
		if mappings[i].TenantName == "engineering-admins" {
			adminMapping = &mappings[i]
			break
		}
	}

	assert.NotNil(t, adminMapping)
	assert.Equal(t, models.TenantRoleAdmin, adminMapping.Role)
}

// TestApplyGroupMapping_Direct tests direct mapping
func TestApplyGroupMapping_Direct(t *testing.T) {
	cfg := &config.GroupMappingConfig{
		Mode: "direct",
	}

	tests := []struct {
		group    string
		expected string
	}{
		{"engineering", "engineering"},
		{"sales-team", "sales-team"},
		{"/admins", "/admins"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.group, func(t *testing.T) {
			result := applyGroupMapping(tt.group, cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestApplyGroupMapping_Prefix tests prefix-based mapping
func TestApplyGroupMapping_Prefix(t *testing.T) {
	cfg := &config.GroupMappingConfig{
		Mode:   "prefix",
		Prefix: "/engineering/",
	}

	tests := []struct {
		group    string
		expected string
	}{
		{"/engineering/backend", "backend"},
		{"/engineering/frontend", "frontend"},
		{"/engineering/backend/api", "backend"}, // Nested: takes first segment
		{"/sales/emea", ""},                     // No match - different prefix
		{"engineering", ""},                     // No match - missing prefix
		{"/engineering/", ""},                   // Empty after prefix
	}

	for _, tt := range tests {
		t.Run(tt.group, func(t *testing.T) {
			result := applyGroupMapping(tt.group, cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsAdminGroup tests admin group matching
func TestIsAdminGroup(t *testing.T) {
	adminGroups := []string{
		"admins",
		"engineering-admins",
		"/admin",
	}

	tests := []struct {
		group    string
		expected bool
	}{
		{"admins", true},
		{"engineering-admins", true},
		{"/admin", true},
		{"/admin/users", true},            // Prefix match
		{"backend-admins", true},          // Suffix match
		{"/engineering/admins", true},     // Contains match
		{"users", false},                  // No match
		{"engineering", false},            // No match
		{"administrator", false},          // Partial match but not exact
	}

	for _, tt := range tests {
		t.Run(tt.group, func(t *testing.T) {
			result := isAdminGroup(tt.group, adminGroups)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNormalizeTenantName tests tenant name normalization
func TestNormalizeTenantName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Backend Team", "backend-team"},    // Spaces → hyphens
		{"Sales-EMEA", "sales-emea"},
		{"  Engineering  ", "engineering"},
		{"Frontend__API", "frontend__api"},
		{"Support---Team", "support-team"}, // Multiple hyphens → single
		{"/admin/", "admin"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeTenantName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetTenantNameFromGroup convenience function test
func TestGetTenantNameFromGroup(t *testing.T) {
	cfg := &config.GroupMappingConfig{
		Mode:   "prefix",
		Prefix: "/org/",
	}

	tests := []struct {
		group    string
		expected string
	}{
		{"/org/engineering", "engineering"},
		{"/org/Sales Team", "sales-team"}, // Normalized: spaces → hyphens
		{"/other/team", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.group, func(t *testing.T) {
			result := GetTenantNameFromGroup(tt.group, cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestUniqueTenantMappings tests deduplication with role priority
func TestUniqueTenantMappings(t *testing.T) {
	mappings := []TenantMapping{
		{TenantName: "engineering", Role: models.TenantRoleMember, SourceGroup: "/engineering"},
		{TenantName: "engineering", Role: models.TenantRoleAdmin, SourceGroup: "/engineering-admins"},
		{TenantName: "sales", Role: models.TenantRoleMember, SourceGroup: "/sales"},
		{TenantName: "support", Role: models.TenantRoleMember, SourceGroup: "/support"},
		{TenantName: "sales", Role: models.TenantRoleMember, SourceGroup: "/sales-team"}, // Duplicate member
	}

	unique := UniqueTenantMappings(mappings)

	assert.Len(t, unique, 3) // engineering, sales, support

	// Find engineering mapping - should be admin (highest role)
	var engineeringMapping *TenantMapping
	for i := range unique {
		if unique[i].TenantName == "engineering" {
			engineeringMapping = &unique[i]
			break
		}
	}

	assert.NotNil(t, engineeringMapping)
	assert.Equal(t, models.TenantRoleAdmin, engineeringMapping.Role)

	// Find sales mapping - should be member (no admin)
	var salesMapping *TenantMapping
	for i := range unique {
		if unique[i].TenantName == "sales" {
			salesMapping = &unique[i]
			break
		}
	}

	assert.NotNil(t, salesMapping)
	assert.Equal(t, models.TenantRoleMember, salesMapping.Role)
}

// TestParseGroups_EmptyGroups tests empty groups input
func TestParseGroups_EmptyGroups(t *testing.T) {
	cfg := &config.GroupMappingConfig{
		Mode: "direct",
	}

	mappings := ParseGroups([]string{}, cfg)
	assert.Empty(t, mappings)

	mappings = ParseGroups(nil, cfg)
	assert.Empty(t, mappings)
}

// TestParseGroups_ComplexKeycloakScenario tests realistic Keycloak scenario
func TestParseGroups_ComplexKeycloakScenario(t *testing.T) {
	// Realistic Keycloak groups structure
	groups := []string{
		"/organizations/acme",
		"/organizations/acme/engineering",
		"/organizations/acme/engineering/backend",
		"/organizations/globex",
		"/admin",
	}

	cfg := &config.GroupMappingConfig{
		Mode:        "prefix",
		Prefix:      "/organizations/",
		AdminGroups: []string{"/admin"},
	}

	mappings := ParseGroups(groups, cfg)

	// Should extract:
	// - acme (from "/organizations/acme")
	// - acme (from "/organizations/acme/engineering" - takes first segment)
	// - acme (from "/organizations/acme/engineering/backend" - takes first segment)
	// - globex (from "/organizations/globex")
	// "/admin" doesn't match prefix, so not included
	assert.Len(t, mappings, 4)

	// Count acme tenants (should be 3 before deduplication)
	var acmeCount int
	for _, m := range mappings {
		if m.TenantName == "acme" {
			acmeCount++
		}
	}
	assert.Equal(t, 3, acmeCount) // Multiple groups map to same tenant (before deduplication)

	// Test deduplication
	unique := UniqueTenantMappings(mappings)
	assert.Len(t, unique, 2) // After dedup: acme, globex

	// Check that admin role works with "/admin" group (though it's not in final mappings due to prefix)
	// Let's test with a group that matches admin
	groupsWithAdmin := []string{
		"/organizations/admins",
	}
	cfgWithAdmin := &config.GroupMappingConfig{
		Mode:        "prefix",
		Prefix:      "/organizations/",
		AdminGroups: []string{"admins", "/admin"},
	}
	adminMappings := ParseGroups(groupsWithAdmin, cfgWithAdmin)
	assert.Len(t, adminMappings, 1)
	assert.Equal(t, models.TenantRoleAdmin, adminMappings[0].Role)
}

// Benchmark tests
func BenchmarkParseGroups_Direct(b *testing.B) {
	groups := []string{
		"engineering", "sales", "support", "marketing",
		"product", "design", "operations", "finance",
	}
	cfg := &config.GroupMappingConfig{
		Mode:        "direct",
		AdminGroups: []string{"admins"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseGroups(groups, cfg)
	}
}

func BenchmarkParseGroups_Prefix(b *testing.B) {
	groups := []string{
		"/organizations/engineering",
		"/organizations/sales",
		"/organizations/support",
		"/organizations/marketing",
	}
	cfg := &config.GroupMappingConfig{
		Mode:        "prefix",
		Prefix:      "/organizations/",
		AdminGroups: []string{"/admin"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseGroups(groups, cfg)
	}
}


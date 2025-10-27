// Package oidc provides tenant provisioning logic from OIDC groups
// Version: 1.11.2+ (Auto-tenant Provisioning from OIDC Groups)
package oidc

import (
	"strings"

	"aigateway/internal/config"
	"aigateway/internal/models"
)

// TenantMapping представляет mapping от OIDC group к tenant
type TenantMapping struct {
	TenantName string             // Имя tenant
	Role       models.TenantRole  // Роль пользователя в tenant (member, admin)
	SourceGroup string            // Исходная OIDC группа
}

// ParseGroups извлекает tenant mappings из OIDC groups claims
// Применяет правила mapping (direct/prefix mode) и определяет роли
func ParseGroups(groups []string, config *config.GroupMappingConfig) []TenantMapping {
	var mappings []TenantMapping
	
	for _, group := range groups {
		// Apply mapping rules
		tenantName := applyGroupMapping(group, config)
		if tenantName == "" {
			continue // Skip unmapped groups
		}
		
		// Normalize tenant name (lowercase, replace spaces with hyphens)
		tenantName = normalizeTenantName(tenantName)
		
		// Determine role based on admin groups
		role := models.TenantRoleMember
		if isAdminGroup(group, config.AdminGroups) {
			role = models.TenantRoleAdmin
		}
		
		mappings = append(mappings, TenantMapping{
			TenantName:  tenantName,
			Role:        role,
			SourceGroup: group,
		})
	}
	
	return mappings
}

// applyGroupMapping применяет правила маппинга группы в tenant name
func applyGroupMapping(group string, config *config.GroupMappingConfig) string {
	switch config.Mode {
	case "direct":
		// Direct 1:1 mapping: group name = tenant name
		// Group: "backend-team" → Tenant: "backend-team"
		return group
		
	case "prefix":
		// Prefix-based mapping: extract tenant name after prefix
		// Group: "/engineering/backend" → Tenant: "backend"
		// Group: "/sales/emea" → Tenant: "emea"
		if config.Prefix != "" && strings.HasPrefix(group, config.Prefix) {
			tenantName := strings.TrimPrefix(group, config.Prefix)
			
			// Remove leading/trailing slashes
			tenantName = strings.Trim(tenantName, "/")
			
			// For nested groups, take the first segment
			// "/engineering/backend/api" → "backend"
			parts := strings.Split(tenantName, "/")
			if len(parts) > 0 && parts[0] != "" {
				return parts[0]
			}
		}
		return "" // Skip groups that don't match prefix
		
	default:
		// Default to direct mapping if mode is unknown
		return group
	}
}

// isAdminGroup проверяет, является ли группа admin группой
func isAdminGroup(group string, adminGroups []string) bool {
	for _, adminGroup := range adminGroups {
		// Exact match
		if group == adminGroup {
			return true
		}
		
		// Suffix match for nested groups
		// Group: "/engineering/backend-admins" matches admin group "*-admins"
		if strings.HasSuffix(group, adminGroup) {
			return true
		}
		
		// Prefix match
		// Group: "/admins/engineering" matches admin group "/admins"
		if strings.HasPrefix(group, adminGroup) {
			return true
		}
		
		// Contains match for flexibility
		if strings.Contains(group, adminGroup) {
			return true
		}
	}
	
	return false
}

// normalizeTenantName нормализует имя tenant для использования в системе
func normalizeTenantName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)
	
	// Replace spaces with hyphens
	name = strings.ReplaceAll(name, " ", "-")
	
	// Remove leading/trailing whitespace and special characters
	name = strings.Trim(name, " \t\n\r-_/")
	
	// Replace multiple hyphens with single hyphen
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}
	
	return name
}

// GetTenantNameFromGroup извлекает tenant name из OIDC группы с применением правил mapping
// Convenience function для единичной группы
func GetTenantNameFromGroup(group string, config *config.GroupMappingConfig) string {
	tenantName := applyGroupMapping(group, config)
	if tenantName == "" {
		return ""
	}
	return normalizeTenantName(tenantName)
}

// UniqueT enantMappings удаляет дубликаты tenant mappings
// При наличии дубликатов сохраняется mapping с максимальной ролью (admin > member)
func UniqueTenantMappings(mappings []TenantMapping) []TenantMapping {
	// Map: tenant name → highest role
	tenantRoles := make(map[string]models.TenantRole)
	tenantSources := make(map[string]string) // Track source group
	
	for _, mapping := range mappings {
		existingRole, exists := tenantRoles[mapping.TenantName]
		
		if !exists {
			// First occurrence
			tenantRoles[mapping.TenantName] = mapping.Role
			tenantSources[mapping.TenantName] = mapping.SourceGroup
		} else {
			// Duplicate - keep highest role
			if mapping.Role == models.TenantRoleAdmin && existingRole != models.TenantRoleAdmin {
				tenantRoles[mapping.TenantName] = models.TenantRoleAdmin
				tenantSources[mapping.TenantName] = mapping.SourceGroup
			}
		}
	}
	
	// Convert back to slice
	var uniqueMappings []TenantMapping
	for tenantName, role := range tenantRoles {
		uniqueMappings = append(uniqueMappings, TenantMapping{
			TenantName:  tenantName,
			Role:        role,
			SourceGroup: tenantSources[tenantName],
		})
	}
	
	return uniqueMappings
}



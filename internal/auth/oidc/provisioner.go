// Package oidc provides tenant provisioner service for auto-provisioning from OIDC groups
// Version: 1.11.2+ (Auto-tenant Provisioning from OIDC Groups)
package oidc

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// TenantProvisioner управляет автоматическим созданием tenants и добавлением пользователей
type TenantProvisioner struct {
	db     storage.Database
	config *config.TenantProvisioningConfig
	logger *logrus.Logger
}

// NewTenantProvisioner создает новый provisioner
func NewTenantProvisioner(db storage.Database, config *config.TenantProvisioningConfig, logger *logrus.Logger) *TenantProvisioner {
	if logger == nil {
		logger = logrus.New()
	}
	
	return &TenantProvisioner{
		db:     db,
		config: config,
		logger: logger,
	}
}

// ProvisionTenantsForUser автоматически создает tenants и добавляет пользователя в них
// на основе OIDC groups claims
func (p *TenantProvisioner) ProvisionTenantsForUser(
	ctx context.Context,
	userID string,
	mappings []TenantMapping,
) error {
	if !p.config.Enabled {
		p.logger.Debug("Tenant provisioning disabled, skipping")
		return nil
	}
	
	if len(mappings) == 0 {
		p.logger.WithField("user_id", userID).Debug("No tenant mappings to provision")
		return nil
	}
	
	// Remove duplicates (keep highest role)
	mappings = UniqueTenantMappings(mappings)
	
	p.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"mappings_count": len(mappings),
	}).Info("Provisioning tenants for user from OIDC groups")
	
	successCount := 0
	errorCount := 0
	
	for _, mapping := range mappings {
		if err := p.provisionSingleTenant(ctx, userID, mapping); err != nil {
			p.logger.WithError(err).WithFields(logrus.Fields{
				"user_id":      userID,
				"tenant_name":  mapping.TenantName,
				"source_group": mapping.SourceGroup,
			}).Error("Failed to provision tenant")
			errorCount++
			continue
		}
		successCount++
	}
	
	// Optionally remove orphaned memberships
	if p.config.RemoveOrphanedMemberships {
		if err := p.removeOrphanedMemberships(ctx, userID, mappings); err != nil {
			p.logger.WithError(err).WithField("user_id", userID).Warn("Failed to remove orphaned memberships")
		}
	}
	
	p.logger.WithFields(logrus.Fields{
		"user_id":       userID,
		"success_count": successCount,
		"error_count":   errorCount,
	}).Info("Tenant provisioning completed")
	
	if errorCount > 0 && successCount == 0 {
		return fmt.Errorf("failed to provision all tenants: %d errors", errorCount)
	}
	
	return nil
}

// provisionSingleTenant создает tenant (если нужно) и добавляет пользователя
func (p *TenantProvisioner) provisionSingleTenant(
	ctx context.Context,
	userID string,
	mapping TenantMapping,
) error {
	// Get or create tenant
	tenant, err := p.getOrCreateTenant(ctx, mapping.TenantName, mapping.SourceGroup)
	if err != nil {
		return fmt.Errorf("failed to get/create tenant: %w", err)
	}
	
	// Add user to tenant with specified role
	if err := p.addUserToTenant(ctx, userID, tenant.ID, mapping.Role); err != nil {
		return fmt.Errorf("failed to add user to tenant: %w", err)
	}
	
	p.logger.WithFields(logrus.Fields{
		"user_id":      userID,
		"tenant_id":    tenant.ID,
		"tenant_name":  tenant.Name,
		"role":         mapping.Role,
		"source_group": mapping.SourceGroup,
	}).Info("User provisioned to tenant")
	
	return nil
}

// getOrCreateTenant получает существующий tenant или создает новый
func (p *TenantProvisioner) getOrCreateTenant(
	ctx context.Context,
	name string,
	sourceGroup string,
) (*models.Tenant, error) {
	// Try to get existing tenant by name
	tenant, err := p.db.GetTenantByName(ctx, name)
	if err == nil {
		p.logger.WithField("tenant_id", tenant.ID).Debugf("Found existing tenant: %s", name)
		return tenant, nil
	}
	
	// Tenant doesn't exist - create if auto-create is enabled
	if !p.config.AutoCreateTenants {
		return nil, fmt.Errorf("tenant '%s' not found and auto-create is disabled", name)
	}
	
	// Create new tenant
	now := time.Now()
	tenant = &models.Tenant{
		ID:          uuid.New().String(),
		Name:        name,
		Slug:        name, // Use name as slug (can be improved with slugification)
		Description: fmt.Sprintf("Auto-provisioned from OIDC group: %s", sourceGroup),
		Type:        models.TenantTypeOrganization,
		Status:      models.TenantStatusActive,
		IsActive:    true,
		Settings: models.TenantSettings{
			MaxAPIKeys:       50,   // Default limit
			MaxConversations: 1000, // Default limit
			ChatEnabled:      true,
			APIAccessEnabled: true,
		},
		Metadata:  make(map[string]interface{}), // Empty map for JSONB
		CreatedAt: now,
		UpdatedAt: now,
	}
	
	if err := p.db.CreateTenant(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}
	
	p.logger.WithFields(logrus.Fields{
		"tenant_id":    tenant.ID,
		"tenant_name":  name,
		"source_group": sourceGroup,
	}).Info("Auto-created tenant from OIDC group")
	
	return tenant, nil
}

// addUserToTenant добавляет пользователя в tenant или обновляет его роль
func (p *TenantProvisioner) addUserToTenant(
	ctx context.Context,
	userID, tenantID string,
	role models.TenantRole,
) error {
	// Check if already a member
	member, err := p.db.GetTenantMember(ctx, tenantID, userID)
	if err == nil && member != nil {
		// User is already a member - update role if different
		if member.Role != role {
			p.logger.WithFields(logrus.Fields{
				"user_id":   userID,
				"tenant_id": tenantID,
				"old_role":  member.Role,
				"new_role":  role,
			}).Info("Updating tenant member role")
			
			member.Role = role
			member.UpdatedAt = time.Now()
			return p.db.UpdateTenantMember(ctx, member)
		}
		
		// Role is the same - no action needed
		p.logger.WithFields(logrus.Fields{
			"user_id":   userID,
			"tenant_id": tenantID,
			"role":      role,
		}).Debug("User is already a tenant member with correct role")
		
		return nil
	}
	
	// Add as new member
	now := time.Now()
	member = &models.TenantMember{
		TenantID:  tenantID,
		UserID:    userID,
		Role:      role,
		JoinedAt:  now,
		UpdatedAt: now,
		Metadata:  make(map[string]interface{}), // Empty map for JSONB
	}
	
	if err := p.db.AddTenantMember(ctx, member); err != nil {
		return fmt.Errorf("failed to add tenant member: %w", err)
	}
	
	p.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"tenant_id": tenantID,
		"role":      role,
	}).Info("Added user as tenant member")
	
	return nil
}

// removeOrphanedMemberships удаляет membership из tenants, которые больше не в OIDC groups
func (p *TenantProvisioner) removeOrphanedMemberships(
	ctx context.Context,
	userID string,
	validMappings []TenantMapping,
) error {
	// Get all current tenants for user
	tenants, err := p.db.ListUserTenants(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to list user tenants: %w", err)
	}
	
	if len(tenants) == 0 {
		return nil // No tenants to check
	}
	
	// Build set of valid tenant names
	validTenantNames := make(map[string]bool)
	for _, mapping := range validMappings {
		validTenantNames[mapping.TenantName] = true
	}
	
	// Remove memberships for tenants not in valid set
	removedCount := 0
	for _, tenant := range tenants {
		if !validTenantNames[tenant.Name] {
			if err := p.db.RemoveTenantMember(ctx, tenant.ID, userID); err != nil {
				p.logger.WithError(err).WithFields(logrus.Fields{
					"user_id":     userID,
					"tenant_id":   tenant.ID,
					"tenant_name": tenant.Name,
				}).Warn("Failed to remove orphaned tenant membership")
				continue
			}
			
			p.logger.WithFields(logrus.Fields{
				"user_id":     userID,
				"tenant_id":   tenant.ID,
				"tenant_name": tenant.Name,
			}).Info("Removed orphaned tenant membership")
			
			removedCount++
		}
	}
	
	if removedCount > 0 {
		p.logger.WithFields(logrus.Fields{
			"user_id":       userID,
			"removed_count": removedCount,
		}).Info("Removed orphaned tenant memberships")
	}
	
	return nil
}

// formatTenantDisplayName форматирует display name из имени tenant
func formatTenantDisplayName(name string) string {
	// Replace hyphens with spaces and capitalize first letter of each word
	// "backend-team" → "Backend Team"
	// "sales" → "Sales"
	
	if name == "" {
		return name
	}
	
	// Replace hyphens and underscores with spaces
	displayName := name
	displayName = fmt.Sprintf("%s%s", 
		string([]rune(displayName)[0:1]), // First letter (uppercase handled below)
		displayName[1:])
	
	// Simple capitalization (first letter only for simplicity)
	if len(displayName) > 0 {
		firstLetter := []rune(displayName)[0]
		if firstLetter >= 'a' && firstLetter <= 'z' {
			displayName = string(firstLetter-32) + displayName[1:]
		}
	}
	
	return displayName
}



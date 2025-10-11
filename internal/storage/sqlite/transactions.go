// Package sqlite provides transaction delegation for SQLite database operations
package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// ========================================
// Transaction Control Methods
// ========================================

// Connect cannot be called on transaction
func (tx *sqliteTx) Connect(ctx context.Context) error {
	return fmt.Errorf("cannot call Connect on transaction")
}

// Close cannot be called on transaction, use Commit or Rollback
func (tx *sqliteTx) Close() error {
	return fmt.Errorf("cannot call Close on transaction, use Commit or Rollback")
}

// Ping cannot be called on transaction
func (tx *sqliteTx) Ping(ctx context.Context) error {
	return fmt.Errorf("cannot call Ping on transaction")
}

// Migrate cannot be called on transaction
func (tx *sqliteTx) Migrate(ctx context.Context) error {
	return fmt.Errorf("cannot call Migrate on transaction")
}

// GetMigrationVersion cannot be called on transaction
func (tx *sqliteTx) GetMigrationVersion(ctx context.Context) (int, error) {
	return 0, fmt.Errorf("cannot call GetMigrationVersion on transaction")
}

// BeginTx - nested transactions not supported
func (tx *sqliteTx) BeginTx(ctx context.Context) (storage.Tx, error) {
	return nil, fmt.Errorf("nested transactions not supported")
}

// ========================================
// Users Methods (Using Transaction)
// ========================================

func (tx *sqliteTx) CreateUser(ctx context.Context, user *models.User) error {
	tx.logger.WithField("username", user.Username).Debug("Creating user in transaction")

	// Serialize preferences to JSON
	preferencesJSON, err := json.Marshal(user.Preferences)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	// Serialize metadata to JSON if present
	var metadataJSON []byte
	if user.Metadata != nil {
		metadataJSON, err = json.Marshal(user.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO users (
			id, username, email, full_name, password_hash,
			status, is_admin, is_active, verified, verified_at,
			created_at, updated_at, last_login,
			preferences, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.tx.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.FullName,
		user.PasswordHash,
		user.Status,
		user.IsAdmin,
		user.IsActive,
		user.Verified,
		user.VerifiedAt,
		user.CreatedAt,
		user.UpdatedAt,
		user.LastLogin,
		preferencesJSON,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert user in transaction: %w", err)
	}

	tx.logger.WithField("user_id", user.ID).Info("User created in transaction")
	return nil
}

func (tx *sqliteTx) GetUser(ctx context.Context, id string) (*models.User, error) {
	return tx.db.GetUser(ctx, id)
}

func (tx *sqliteTx) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	return tx.db.GetUserByUsername(ctx, username)
}

func (tx *sqliteTx) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return tx.db.GetUserByEmail(ctx, email)
}

func (tx *sqliteTx) UpdateUser(ctx context.Context, user *models.User) error {
	return tx.db.UpdateUser(ctx, user)
}

func (tx *sqliteTx) UpdateUserPassword(ctx context.Context, userID string, passwordHash string) error {
	return tx.db.UpdateUserPassword(ctx, userID, passwordHash)
}

func (tx *sqliteTx) DeleteUser(ctx context.Context, id string) error {
	return tx.db.DeleteUser(ctx, id)
}

func (tx *sqliteTx) ListUsers(ctx context.Context, filters models.UserFilters) ([]*models.User, error) {
	return tx.db.ListUsers(ctx, filters)
}

// ========================================
// Tenants Methods (Using Transaction)
// ========================================

func (tx *sqliteTx) CreateTenant(ctx context.Context, tenant *models.Tenant) error {
	tx.logger.WithField("name", tenant.Name).Debug("Creating tenant in transaction")

	// Serialize settings to JSON
	settingsJSON, err := json.Marshal(tenant.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	query := `
		INSERT INTO tenants (
			id, name, slug, description, owner_id, type,
			status, settings, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.tx.ExecContext(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.Slug,
		tenant.Description,
		tenant.OwnerID,
		tenant.Type,
		tenant.Status,
		settingsJSON,
		tenant.CreatedAt,
		tenant.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert tenant in transaction: %w", err)
	}

	tx.logger.WithField("tenant_id", tenant.ID).Info("Tenant created in transaction")
	return nil
}

func (tx *sqliteTx) GetTenant(ctx context.Context, id string) (*models.Tenant, error) {
	return tx.db.GetTenant(ctx, id)
}

func (tx *sqliteTx) GetTenantBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	return tx.db.GetTenantBySlug(ctx, slug)
}

func (tx *sqliteTx) UpdateTenant(ctx context.Context, tenant *models.Tenant) error {
	return tx.db.UpdateTenant(ctx, tenant)
}

func (tx *sqliteTx) DeleteTenant(ctx context.Context, id string) error {
	return tx.db.DeleteTenant(ctx, id)
}

func (tx *sqliteTx) ListUserTenants(ctx context.Context, userID string) ([]*models.Tenant, error) {
	return tx.db.ListUserTenants(ctx, userID)
}

// ========================================
// Tenant Members Methods (Using Transaction)
// ========================================

func (tx *sqliteTx) AddTenantMember(ctx context.Context, member *models.TenantMember) error {
	tx.logger.WithFields(map[string]interface{}{
		"tenant_id": member.TenantID,
		"user_id":   member.UserID,
		"role":      member.Role,
	}).Debug("Adding tenant member in transaction")

	query := `
		INSERT INTO tenant_members (
			tenant_id, user_id, role, joined_at, updated_at
		) VALUES (?, ?, ?, ?, ?)
	`

	_, err := tx.tx.ExecContext(ctx, query,
		member.TenantID,
		member.UserID,
		member.Role,
		member.JoinedAt,
		member.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to add tenant member in transaction: %w", err)
	}

	tx.logger.Info("Tenant member added in transaction")
	return nil
}

func (tx *sqliteTx) GetTenantMember(ctx context.Context, tenantID, userID string) (*models.TenantMember, error) {
	return tx.db.GetTenantMember(ctx, tenantID, userID)
}

func (tx *sqliteTx) UpdateTenantMember(ctx context.Context, member *models.TenantMember) error {
	return tx.db.UpdateTenantMember(ctx, member)
}

func (tx *sqliteTx) RemoveTenantMember(ctx context.Context, tenantID, userID string) error {
	return tx.db.RemoveTenantMember(ctx, tenantID, userID)
}

func (tx *sqliteTx) ListTenantMembers(ctx context.Context, tenantID string) ([]*models.TenantMember, error) {
	return tx.db.ListTenantMembers(ctx, tenantID)
}

// ========================================
// API Keys Methods (Delegation)
// ========================================

func (tx *sqliteTx) CreateAPIKey(ctx context.Context, key *models.APIKey) error {
	return tx.db.CreateAPIKey(ctx, key)
}

func (tx *sqliteTx) GetAPIKey(ctx context.Context, id string) (*models.APIKey, error) {
	return tx.db.GetAPIKey(ctx, id)
}

func (tx *sqliteTx) GetAPIKeyByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	return tx.db.GetAPIKeyByHash(ctx, hash)
}

func (tx *sqliteTx) UpdateAPIKey(ctx context.Context, key *models.APIKey) error {
	return tx.db.UpdateAPIKey(ctx, key)
}

func (tx *sqliteTx) DeleteAPIKey(ctx context.Context, id string) error {
	return tx.db.DeleteAPIKey(ctx, id)
}

func (tx *sqliteTx) RevokeAPIKey(ctx context.Context, id string, reason string) error {
	return tx.db.RevokeAPIKey(ctx, id, reason)
}

func (tx *sqliteTx) EnableAPIKey(ctx context.Context, id string) error {
	return tx.db.EnableAPIKey(ctx, id)
}

func (tx *sqliteTx) ListAPIKeys(ctx context.Context) ([]*models.APIKey, error) {
	return tx.db.ListAPIKeys(ctx)
}

func (tx *sqliteTx) ListPersonalAPIKeys(ctx context.Context, userID string) ([]*models.APIKey, error) {
	return tx.db.ListPersonalAPIKeys(ctx, userID)
}

func (tx *sqliteTx) ListTenantAPIKeys(ctx context.Context, tenantID string) ([]*models.APIKey, error) {
	return tx.db.ListTenantAPIKeys(ctx, tenantID)
}

// ========================================
// Conversations Methods (Delegation)
// ========================================

func (tx *sqliteTx) CreateConversation(ctx context.Context, conv *models.Conversation) error {
	return tx.db.CreateConversation(ctx, conv)
}

func (tx *sqliteTx) GetConversation(ctx context.Context, id string) (*models.Conversation, error) {
	return tx.db.GetConversation(ctx, id)
}

func (tx *sqliteTx) UpdateConversation(ctx context.Context, conv *models.Conversation) error {
	return tx.db.UpdateConversation(ctx, conv)
}

func (tx *sqliteTx) DeleteConversation(ctx context.Context, id string) error {
	return tx.db.DeleteConversation(ctx, id)
}

func (tx *sqliteTx) ListUserConversations(ctx context.Context, userID string, filters models.ConversationFilters) ([]*models.Conversation, error) {
	return tx.db.ListUserConversations(ctx, userID, filters)
}

// ========================================
// Messages Methods (Delegation)
// ========================================

func (tx *sqliteTx) CreateMessage(ctx context.Context, msg *models.Message) error {
	return tx.db.CreateMessage(ctx, msg)
}

func (tx *sqliteTx) GetMessage(ctx context.Context, id string) (*models.Message, error) {
	return tx.db.GetMessage(ctx, id)
}

func (tx *sqliteTx) ListConversationMessages(ctx context.Context, convID string) ([]*models.Message, error) {
	return tx.db.ListConversationMessages(ctx, convID)
}

func (tx *sqliteTx) DeleteConversationMessages(ctx context.Context, convID string) error {
	return tx.db.DeleteConversationMessages(ctx, convID)
}

// ========================================
// Usage Stats Methods (Delegation)
// ========================================

func (tx *sqliteTx) RecordAPIUsage(ctx context.Context, usage *models.APIUsage) error {
	return tx.db.RecordAPIUsage(ctx, usage)
}

func (tx *sqliteTx) GetUserUsageStats(ctx context.Context, userID string, period time.Duration) (*models.UsageStats, error) {
	return tx.db.GetUserUsageStats(ctx, userID, period)
}

func (tx *sqliteTx) GetTenantUsageStats(ctx context.Context, tenantID string, period time.Duration) (*models.UsageStats, error) {
	return tx.db.GetTenantUsageStats(ctx, tenantID, period)
}

// ========================================
// MCP Servers Methods (Delegation - v1.4.5)
// ========================================

func (tx *sqliteTx) CreateMCPServer(ctx context.Context, server *models.MCPServer) error {
	return tx.db.CreateMCPServer(ctx, server)
}

func (tx *sqliteTx) GetMCPServer(ctx context.Context, id string) (*models.MCPServer, error) {
	return tx.db.GetMCPServer(ctx, id)
}

func (tx *sqliteTx) UpdateMCPServer(ctx context.Context, server *models.MCPServer) error {
	return tx.db.UpdateMCPServer(ctx, server)
}

func (tx *sqliteTx) DeleteMCPServer(ctx context.Context, id string) error {
	return tx.db.DeleteMCPServer(ctx, id)
}

func (tx *sqliteTx) ListMCPServers(ctx context.Context, req models.MCPServerListRequest) (*models.MCPServerListResponse, error) {
	return tx.db.ListMCPServers(ctx, req)
}

// ========================================
// Changelog Methods (Delegation) (v1.4.11)
// ========================================

func (tx *sqliteTx) GetChangelog(ctx context.Context, version string) (*models.Changelog, error) {
	return tx.db.GetChangelog(ctx, version)
}

func (tx *sqliteTx) ListChangelogs(ctx context.Context) ([]*models.Changelog, error) {
	return tx.db.ListChangelogs(ctx)
}

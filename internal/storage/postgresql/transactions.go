// Package postgresql provides transaction delegation for PostgreSQL database operations
package postgresql

import (
	"context"
	"fmt"
	"time"

	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// ========================================
// Transaction Control Methods
// ========================================

// Connect cannot be called on transaction
func (tx *postgresqlTx) Connect(ctx context.Context) error {
	return fmt.Errorf("cannot call Connect on transaction")
}

// Close cannot be called on transaction, use Commit or Rollback
func (tx *postgresqlTx) Close() error {
	return fmt.Errorf("cannot call Close on transaction, use Commit or Rollback")
}

// Ping cannot be called on transaction
func (tx *postgresqlTx) Ping(ctx context.Context) error {
	return fmt.Errorf("cannot call Ping on transaction")
}

// Migrate cannot be called on transaction
func (tx *postgresqlTx) Migrate(ctx context.Context) error {
	return fmt.Errorf("cannot call Migrate on transaction")
}

// GetMigrationVersion cannot be called on transaction
func (tx *postgresqlTx) GetMigrationVersion(ctx context.Context) (int, error) {
	return 0, fmt.Errorf("cannot call GetMigrationVersion on transaction")
}

// BeginTx - nested transactions not supported
func (tx *postgresqlTx) BeginTx(ctx context.Context) (storage.Tx, error) {
	return nil, fmt.Errorf("nested transactions not supported")
}

// ========================================
// Users Methods (Delegation)
// ========================================

func (tx *postgresqlTx) CreateUser(ctx context.Context, user *models.User) error {
	return tx.db.CreateUser(ctx, user)
}

func (tx *postgresqlTx) GetUser(ctx context.Context, id string) (*models.User, error) {
	return tx.db.GetUser(ctx, id)
}

func (tx *postgresqlTx) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	return tx.db.GetUserByUsername(ctx, username)
}

func (tx *postgresqlTx) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return tx.db.GetUserByEmail(ctx, email)
}

func (tx *postgresqlTx) GetUserByOIDCSubject(ctx context.Context, issuer, subject string) (*models.User, error) {
	return tx.db.GetUserByOIDCSubject(ctx, issuer, subject)
}

func (tx *postgresqlTx) UpdateUser(ctx context.Context, user *models.User) error {
	return tx.db.UpdateUser(ctx, user)
}

func (tx *postgresqlTx) UpdateUserPassword(ctx context.Context, userID string, passwordHash string) error {
	return tx.db.UpdateUserPassword(ctx, userID, passwordHash)
}

func (tx *postgresqlTx) DeleteUser(ctx context.Context, id string) error {
	return tx.db.DeleteUser(ctx, id)
}

func (tx *postgresqlTx) ListUsers(ctx context.Context, filters models.UserFilters) ([]*models.User, error) {
	return tx.db.ListUsers(ctx, filters)
}

// ========================================
// Tenants Methods (Delegation)
// ========================================

func (tx *postgresqlTx) CreateTenant(ctx context.Context, tenant *models.Tenant) error {
	return tx.db.CreateTenant(ctx, tenant)
}

func (tx *postgresqlTx) GetTenant(ctx context.Context, id string) (*models.Tenant, error) {
	return tx.db.GetTenant(ctx, id)
}

func (tx *postgresqlTx) GetTenantBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	return tx.db.GetTenantBySlug(ctx, slug)
}

func (tx *postgresqlTx) GetTenantByName(ctx context.Context, name string) (*models.Tenant, error) {
	return tx.db.GetTenantByName(ctx, name)
}

func (tx *postgresqlTx) GetUserByLDAPDN(ctx context.Context, ldapDN string) (*models.User, error) {
	return tx.db.GetUserByLDAPDN(ctx, ldapDN)
}

// Audit Events (Version 1.11.4+: Enhanced Audit Logging)
func (tx *postgresqlTx) CreateAuditEvent(ctx context.Context, event *models.AuditEvent) error {
	return tx.db.CreateAuditEvent(ctx, event)
}

func (tx *postgresqlTx) GetAuditEvents(ctx context.Context, filters storage.AuditFilters) ([]*models.AuditEvent, int, error) {
	return tx.db.GetAuditEvents(ctx, filters)
}

func (tx *postgresqlTx) DeleteOldAuditEvents(ctx context.Context, olderThan time.Time) (int, error) {
	return tx.db.DeleteOldAuditEvents(ctx, olderThan)
}

func (tx *postgresqlTx) UpdateTenant(ctx context.Context, tenant *models.Tenant) error {
	return tx.db.UpdateTenant(ctx, tenant)
}

func (tx *postgresqlTx) DeleteTenant(ctx context.Context, id string) error {
	return tx.db.DeleteTenant(ctx, id)
}

func (tx *postgresqlTx) ListUserTenants(ctx context.Context, userID string) ([]*models.Tenant, error) {
	return tx.db.ListUserTenants(ctx, userID)
}

// ========================================
// Tenant Members Methods (Delegation)
// ========================================

func (tx *postgresqlTx) AddTenantMember(ctx context.Context, member *models.TenantMember) error {
	return tx.db.AddTenantMember(ctx, member)
}

func (tx *postgresqlTx) GetTenantMember(ctx context.Context, tenantID, userID string) (*models.TenantMember, error) {
	return tx.db.GetTenantMember(ctx, tenantID, userID)
}

func (tx *postgresqlTx) UpdateTenantMember(ctx context.Context, member *models.TenantMember) error {
	return tx.db.UpdateTenantMember(ctx, member)
}

func (tx *postgresqlTx) RemoveTenantMember(ctx context.Context, tenantID, userID string) error {
	return tx.db.RemoveTenantMember(ctx, tenantID, userID)
}

func (tx *postgresqlTx) ListTenantMembers(ctx context.Context, tenantID string) ([]*models.TenantMember, error) {
	return tx.db.ListTenantMembers(ctx, tenantID)
}

// ========================================
// API Keys Methods (Delegation)
// ========================================

func (tx *postgresqlTx) CreateAPIKey(ctx context.Context, key *models.APIKey) error {
	return tx.db.CreateAPIKey(ctx, key)
}

func (tx *postgresqlTx) GetAPIKey(ctx context.Context, id string) (*models.APIKey, error) {
	return tx.db.GetAPIKey(ctx, id)
}

func (tx *postgresqlTx) GetAPIKeyByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	return tx.db.GetAPIKeyByHash(ctx, hash)
}

func (tx *postgresqlTx) UpdateAPIKey(ctx context.Context, key *models.APIKey) error {
	return tx.db.UpdateAPIKey(ctx, key)
}

func (tx *postgresqlTx) DeleteAPIKey(ctx context.Context, id string) error {
	return tx.db.DeleteAPIKey(ctx, id)
}

func (tx *postgresqlTx) RevokeAPIKey(ctx context.Context, id string, reason string) error {
	return tx.db.RevokeAPIKey(ctx, id, reason)
}

func (tx *postgresqlTx) EnableAPIKey(ctx context.Context, id string) error {
	return tx.db.EnableAPIKey(ctx, id)
}

func (tx *postgresqlTx) ListAPIKeys(ctx context.Context) ([]*models.APIKey, error) {
	return tx.db.ListAPIKeys(ctx)
}

func (tx *postgresqlTx) ListPersonalAPIKeys(ctx context.Context, userID string) ([]*models.APIKey, error) {
	return tx.db.ListPersonalAPIKeys(ctx, userID)
}

func (tx *postgresqlTx) ListTenantAPIKeys(ctx context.Context, tenantID string) ([]*models.APIKey, error) {
	return tx.db.ListTenantAPIKeys(ctx, tenantID)
}

// ========================================
// Conversations Methods (Delegation)
// ========================================

func (tx *postgresqlTx) CreateConversation(ctx context.Context, conv *models.Conversation) error {
	return tx.db.CreateConversation(ctx, conv)
}

func (tx *postgresqlTx) GetConversation(ctx context.Context, id string) (*models.Conversation, error) {
	return tx.db.GetConversation(ctx, id)
}

func (tx *postgresqlTx) UpdateConversation(ctx context.Context, conv *models.Conversation) error {
	return tx.db.UpdateConversation(ctx, conv)
}

func (tx *postgresqlTx) DeleteConversation(ctx context.Context, id string) error {
	return tx.db.DeleteConversation(ctx, id)
}

func (tx *postgresqlTx) ListUserConversations(ctx context.Context, userID string, filters models.ConversationFilters) ([]*models.Conversation, error) {
	return tx.db.ListUserConversations(ctx, userID, filters)
}

// ========================================
// Messages Methods (Delegation)
// ========================================

func (tx *postgresqlTx) CreateMessage(ctx context.Context, msg *models.Message) error {
	return tx.db.CreateMessage(ctx, msg)
}

func (tx *postgresqlTx) GetMessage(ctx context.Context, id string) (*models.Message, error) {
	return tx.db.GetMessage(ctx, id)
}

func (tx *postgresqlTx) ListConversationMessages(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return tx.db.ListConversationMessages(ctx, conversationID)
}

func (tx *postgresqlTx) DeleteConversationMessages(ctx context.Context, convID string) error {
	return tx.db.DeleteConversationMessages(ctx, convID)
}

// ========================================
// Usage Stats Methods (Delegation)
// ========================================

func (tx *postgresqlTx) RecordAPIUsage(ctx context.Context, usage *models.APIUsage) error {
	return tx.db.RecordAPIUsage(ctx, usage)
}

func (tx *postgresqlTx) GetUserUsageStats(ctx context.Context, userID string, period time.Duration) (*models.UsageStats, error) {
	return tx.db.GetUserUsageStats(ctx, userID, period)
}

func (tx *postgresqlTx) GetTenantUsageStats(ctx context.Context, tenantID string, period time.Duration) (*models.UsageStats, error) {
	return tx.db.GetTenantUsageStats(ctx, tenantID, period)
}

// ========================================
// MCP Servers Methods (Delegation - v1.4.5)
// ========================================

func (tx *postgresqlTx) CreateMCPServer(ctx context.Context, server *models.MCPServer) error {
	return tx.db.CreateMCPServer(ctx, server)
}

func (tx *postgresqlTx) GetMCPServer(ctx context.Context, id string) (*models.MCPServer, error) {
	return tx.db.GetMCPServer(ctx, id)
}

func (tx *postgresqlTx) UpdateMCPServer(ctx context.Context, server *models.MCPServer) error {
	return tx.db.UpdateMCPServer(ctx, server)
}

func (tx *postgresqlTx) DeleteMCPServer(ctx context.Context, id string) error {
	return tx.db.DeleteMCPServer(ctx, id)
}

func (tx *postgresqlTx) ListMCPServers(ctx context.Context, req models.MCPServerListRequest) (*models.MCPServerListResponse, error) {
	return tx.db.ListMCPServers(ctx, req)
}

// ========================================
// Changelog Methods (Delegation) (v1.4.11)
// ========================================

func (tx *postgresqlTx) GetChangelog(ctx context.Context, version string) (*models.Changelog, error) {
	return tx.db.GetChangelog(ctx, version)
}

func (tx *postgresqlTx) ListChangelogs(ctx context.Context) ([]*models.Changelog, error) {
	return tx.db.ListChangelogs(ctx)
}

// ========================================
// Files Methods (Delegation) (v1.10.0+)
// ========================================

func (tx *postgresqlTx) CreateFile(ctx context.Context, req models.CreateFileRequest) (*models.File, error) {
	return tx.db.CreateFile(ctx, req)
}

func (tx *postgresqlTx) GetFileByID(ctx context.Context, fileID string) (*models.File, error) {
	return tx.db.GetFileByID(ctx, fileID)
}

func (tx *postgresqlTx) UpdateFile(ctx context.Context, fileID string, req models.UpdateFileRequest) (*models.File, error) {
	return tx.db.UpdateFile(ctx, fileID, req)
}

func (tx *postgresqlTx) DeleteFile(ctx context.Context, fileID string) error {
	return tx.db.DeleteFile(ctx, fileID)
}

func (tx *postgresqlTx) ListFiles(ctx context.Context, req models.ListFilesRequest) ([]*models.File, int, error) {
	return tx.db.ListFiles(ctx, req)
}

func (tx *postgresqlTx) ListFilesWithUserInfo(ctx context.Context, req models.ListFilesRequest) ([]*models.FileWithUser, int, error) {
	return tx.db.ListFilesWithUserInfo(ctx, req)
}

func (tx *postgresqlTx) IncrementDownloadCount(ctx context.Context, fileID string) error {
	return tx.db.IncrementDownloadCount(ctx, fileID)
}

func (tx *postgresqlTx) LogFileAccess(ctx context.Context, log models.FileAccessLog) error {
	return tx.db.LogFileAccess(ctx, log)
}

func (tx *postgresqlTx) GetFileAccessLogs(ctx context.Context, fileID string, limit int) ([]*models.FileAccessLog, error) {
	return tx.db.GetFileAccessLogs(ctx, fileID, limit)
}

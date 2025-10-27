// Package storage provides database abstraction layer for Ollama-OpenAI Proxy.
//
// This package implements a unified interface for working with different database backends
// (SQLite and PostgreSQL), providing type-safe CRUD operations, transaction support,
// and automatic migrations.
//
// Key features:
//   - Database abstraction (SQLite + PostgreSQL support)
//   - Automatic migrations on startup
//   - Connection pooling and health checks
//   - Transaction support with rollback
//   - Type-safe CRUD operations
//
// Usage:
//
//	cfg := config.DatabaseConfig{Type: config.DatabaseTypeSQLite, ...}
//	db, err := storage.NewDatabase(cfg, logger)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
//
//	// Auto-migrate
//	if err := db.Migrate(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
package storage

import (
	"context"
	"time"

	"aigateway/internal/models"
)

// Database представляет unified интерфейс для работы с БД
type Database interface {
	// ========================================
	// Connection Management
	// ========================================

	// Connect устанавливает соединение с БД
	Connect(ctx context.Context) error

	// Close закрывает соединение с БД
	Close() error

	// Ping проверяет доступность БД
	Ping(ctx context.Context) error

	// ========================================
	// Migrations
	// ========================================

	// Migrate применяет все pending миграции
	Migrate(ctx context.Context) error

	// GetMigrationVersion возвращает текущую версию схемы
	GetMigrationVersion(ctx context.Context) (int, error)

	// ========================================
	// Transaction Support
	// ========================================

	// BeginTx начинает новую транзакцию
	BeginTx(ctx context.Context) (Tx, error)

	// ========================================
	// Users (AUTH-05: User Authentication)
	// ========================================

	// CreateUser создает нового пользователя
	CreateUser(ctx context.Context, user *models.User) error

	// GetUser получает пользователя по ID
	GetUser(ctx context.Context, id string) (*models.User, error)

	// GetUserByUsername получает пользователя по username
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)

	// GetUserByEmail получает пользователя по email
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)

	// GetUserByOIDCSubject получает пользователя по OIDC subject и issuer (Version 1.11.1+)
	GetUserByOIDCSubject(ctx context.Context, issuer, subject string) (*models.User, error)
	
	// GetUserByLDAPDN получает пользователя по LDAP DN (Version 1.11.3+: LDAP Integration)
	GetUserByLDAPDN(ctx context.Context, ldapDN string) (*models.User, error)

	// UpdateUser обновляет данные пользователя
	UpdateUser(ctx context.Context, user *models.User) error

	// UpdateUserPassword обновляет пароль пользователя
	UpdateUserPassword(ctx context.Context, userID string, passwordHash string) error

	// DeleteUser удаляет пользователя (soft delete)
	DeleteUser(ctx context.Context, id string) error

	// ListUsers возвращает список пользователей с фильтрацией
	ListUsers(ctx context.Context, filters models.UserFilters) ([]*models.User, error)

	// ========================================
	// Tenants (AUTH-05: Multi-Tenancy)
	// ========================================

	// CreateTenant создает новый tenant
	CreateTenant(ctx context.Context, tenant *models.Tenant) error

	// GetTenant получает tenant по ID
	GetTenant(ctx context.Context, id string) (*models.Tenant, error)

	// GetTenantBySlug получает tenant по slug
	GetTenantBySlug(ctx context.Context, slug string) (*models.Tenant, error)

	// GetTenantByName получает tenant по имени (Version 1.11.2+: для OIDC auto-provisioning)
	GetTenantByName(ctx context.Context, name string) (*models.Tenant, error)

	// UpdateTenant обновляет данные tenant
	UpdateTenant(ctx context.Context, tenant *models.Tenant) error

	// DeleteTenant удаляет tenant (soft delete)
	DeleteTenant(ctx context.Context, id string) error

	// ListUserTenants возвращает список tenants пользователя
	ListUserTenants(ctx context.Context, userID string) ([]*models.Tenant, error)
	
	// ListAllTenants возвращает список всех tenants в системе (для админов)
	ListAllTenants(ctx context.Context) ([]*models.Tenant, error)

	// ========================================
	// Tenant Members (AUTH-05: Multi-Tenancy)
	// ========================================

	// AddTenantMember добавляет участника в tenant
	AddTenantMember(ctx context.Context, member *models.TenantMember) error

	// GetTenantMember получает информацию об участнике
	GetTenantMember(ctx context.Context, tenantID, userID string) (*models.TenantMember, error)

	// UpdateTenantMember обновляет роль участника
	UpdateTenantMember(ctx context.Context, member *models.TenantMember) error

	// RemoveTenantMember удаляет участника из tenant
	RemoveTenantMember(ctx context.Context, tenantID, userID string) error

	// ListTenantMembers возвращает список участников tenant
	ListTenantMembers(ctx context.Context, tenantID string) ([]*models.TenantMember, error)

	// ========================================
	// API Keys (обновленные для multi-tenancy)
	// ========================================

	// CreateAPIKey создает новый API ключ
	CreateAPIKey(ctx context.Context, key *models.APIKey) error

	// GetAPIKey получает API ключ по ID
	GetAPIKey(ctx context.Context, id string) (*models.APIKey, error)

	// GetAPIKeyByHash получает API ключ по hash
	GetAPIKeyByHash(ctx context.Context, hash string) (*models.APIKey, error)

	// UpdateAPIKey обновляет API ключ
	UpdateAPIKey(ctx context.Context, key *models.APIKey) error

	// DeleteAPIKey удаляет API ключ
	DeleteAPIKey(ctx context.Context, id string) error

	// RevokeAPIKey отзывает API ключ
	RevokeAPIKey(ctx context.Context, id string, reason string) error

	// EnableAPIKey включает отозванный API ключ
	EnableAPIKey(ctx context.Context, id string) error

	// ListAPIKeys возвращает список всех API ключей
	ListAPIKeys(ctx context.Context) ([]*models.APIKey, error)

	// ListPersonalAPIKeys возвращает личные API ключи пользователя
	ListPersonalAPIKeys(ctx context.Context, userID string) ([]*models.APIKey, error)

	// ListTenantAPIKeys возвращает API ключи tenant
	ListTenantAPIKeys(ctx context.Context, tenantID string) ([]*models.APIKey, error)

	// ========================================
	// Conversations (WEBUI-03: Chat Interface)
	// ========================================

	// CreateConversation создает новый диалог
	CreateConversation(ctx context.Context, conv *models.Conversation) error

	// GetConversation получает диалог по ID
	GetConversation(ctx context.Context, id string) (*models.Conversation, error)

	// UpdateConversation обновляет диалог
	UpdateConversation(ctx context.Context, conv *models.Conversation) error

	// DeleteConversation удаляет диалог (soft delete)
	DeleteConversation(ctx context.Context, id string) error

	// ListUserConversations возвращает диалоги пользователя
	ListUserConversations(ctx context.Context, userID string, filters models.ConversationFilters) ([]*models.Conversation, error)

	// ========================================
	// Messages (WEBUI-03: Chat Interface)
	// ========================================

	// CreateMessage создает новое сообщение
	CreateMessage(ctx context.Context, msg *models.Message) error

	// GetMessage получает сообщение по ID
	GetMessage(ctx context.Context, id string) (*models.Message, error)

	// ListConversationMessages возвращает все сообщения диалога
	ListConversationMessages(ctx context.Context, convID string) ([]*models.Message, error)

	// DeleteConversationMessages удаляет все сообщения диалога
	DeleteConversationMessages(ctx context.Context, convID string) error

	// ========================================
	// Usage Statistics (WEBUI-04: Dashboard)
	// ========================================

	// RecordAPIUsage записывает использование API
	RecordAPIUsage(ctx context.Context, usage *models.APIUsage) error

	// GetUserUsageStats возвращает статистику пользователя за период
	GetUserUsageStats(ctx context.Context, userID string, period time.Duration) (*models.UsageStats, error)

	// GetTenantUsageStats возвращает статистику tenant за период
	GetTenantUsageStats(ctx context.Context, tenantID string, period time.Duration) (*models.UsageStats, error)

	// ========================================
	// MCP Servers (WEBUI-07: v1.4.5)
	// ========================================

	// CreateMCPServer создает новую запись MCP сервера
	CreateMCPServer(ctx context.Context, server *models.MCPServer) error

	// GetMCPServer получает MCP сервер по ID
	GetMCPServer(ctx context.Context, id string) (*models.MCPServer, error)

	// UpdateMCPServer обновляет MCP сервер
	UpdateMCPServer(ctx context.Context, server *models.MCPServer) error

	// DeleteMCPServer удаляет MCP сервер
	DeleteMCPServer(ctx context.Context, id string) error

	// ListMCPServers возвращает список MCP серверов с фильтрацией
	ListMCPServers(ctx context.Context, req models.MCPServerListRequest) (*models.MCPServerListResponse, error)

	// ========================================
	// Changelog Methods (v1.4.11+)
	// ========================================

	// GetChangelog возвращает changelog для конкретной версии
	GetChangelog(ctx context.Context, version string) (*models.Changelog, error)

	// ListChangelogs возвращает список всех changelog записей
	ListChangelogs(ctx context.Context) ([]*models.Changelog, error)

	// ========================================
	// Reports Statistics (v1.6.3+)
	// ========================================

	// GetUsageStats возвращает статистику использования за период
	GetUsageStats(ctx context.Context, start, end time.Time) (*models.UsageReportStats, error)

	// GetPerformanceStats возвращает статистику производительности за период
	GetPerformanceStats(ctx context.Context, start, end time.Time) (*models.PerformanceReportStats, error)

	// CountActiveUsers возвращает количество активных пользователей за период
	CountActiveUsers(ctx context.Context, period time.Duration) (int, error)

	// CountTotalUsers возвращает общее количество пользователей
	CountTotalUsers(ctx context.Context) (int, error)

	// CountActiveAPIKeys возвращает количество активных API ключей
	CountActiveAPIKeys(ctx context.Context) (int, error)

	// ========================================
	// Model Configurations (v1.9.1+)
	// ========================================

	// CreateModelConfig создает новую конфигурацию модели
	CreateModelConfig(ctx context.Context, config *models.ModelConfig) error

	// GetModelConfig получает конфигурацию модели по ID
	GetModelConfig(ctx context.Context, id string) (*models.ModelConfig, error)

	// GetModelConfigByScope получает конфигурацию модели по scope и ID
	// modelName: имя модели (e.g., "llama3.1:latest")
	// scope: "global", "tenant", "user"
	// scopeID: nil для global, tenant_id для tenant, user_id для user
	GetModelConfigByScope(ctx context.Context, modelName, scope string, scopeID *string) (*models.ModelConfig, error)

	// ListModelConfigs возвращает список конфигураций по фильтру
	ListModelConfigs(ctx context.Context, scope string, scopeID *string) ([]*models.ModelConfig, error)

	// UpdateModelConfig обновляет конфигурацию модели
	UpdateModelConfig(ctx context.Context, config *models.ModelConfig) error

	// DeleteModelConfig удаляет конфигурацию модели
	DeleteModelConfig(ctx context.Context, id string) error

	// GetEffectiveModelConfig возвращает effective конфигурацию с приоритетом:
	// user config > tenant config > global config > defaults
	GetEffectiveModelConfig(ctx context.Context, modelName, userID, tenantID string) (*models.ModelParameters, error)

	// ========================================
	// Files (FILE-STORAGE-01: v1.10.0+)
	// ========================================

	// CreateFile создает новую запись о файле
	CreateFile(ctx context.Context, req models.CreateFileRequest) (*models.File, error)

	// GetFileByID получает файл по ID
	GetFileByID(ctx context.Context, fileID string) (*models.File, error)

	// UpdateFile обновляет информацию о файле
	UpdateFile(ctx context.Context, fileID string, req models.UpdateFileRequest) (*models.File, error)

	// DeleteFile удаляет файл (soft delete)
	DeleteFile(ctx context.Context, fileID string) error

	// ListFiles возвращает список файлов с фильтрацией и пагинацией
	ListFiles(ctx context.Context, req models.ListFilesRequest) ([]*models.File, int, error)

	// ListFilesWithUserInfo возвращает файлы с информацией о владельцах (для админки)
	ListFilesWithUserInfo(ctx context.Context, req models.ListFilesRequest) ([]*models.FileWithUser, int, error)

	// IncrementDownloadCount увеличивает счетчик скачиваний
	IncrementDownloadCount(ctx context.Context, fileID string) error

	// LogFileAccess записывает лог доступа к файлу
	LogFileAccess(ctx context.Context, log models.FileAccessLog) error

	// GetFileAccessLogs возвращает историю доступа к файлу
	GetFileAccessLogs(ctx context.Context, fileID string, limit int) ([]*models.FileAccessLog, error)

	// ========================================
	// Audit Events (Version 1.11.4+: Enhanced Audit Logging)
	// ========================================

	// CreateAuditEvent создает новое событие аудита
	CreateAuditEvent(ctx context.Context, event *models.AuditEvent) error

	// GetAuditEvents возвращает список событий аудита с фильтрами
	GetAuditEvents(ctx context.Context, filters AuditFilters) ([]*models.AuditEvent, int, error)

	// DeleteOldAuditEvents удаляет события аудита старше указанного времени (retention policy)
	DeleteOldAuditEvents(ctx context.Context, olderThan time.Time) (int, error)

	// ========================================
	// RBAC (Roles & Permissions) (Version 1.11.5+: Custom Roles & Permissions)
	// ========================================

	// Permissions
	CreatePermission(ctx context.Context, permission *models.RBACPermission) error
	GetPermission(ctx context.Context, id string) (*models.RBACPermission, error)
	GetPermissionByName(ctx context.Context, name string) (*models.RBACPermission, error)
	ListPermissions(ctx context.Context) ([]*models.RBACPermission, error)

	// Roles
	CreateRole(ctx context.Context, role *models.Role) error
	GetRole(ctx context.Context, id string) (*models.Role, error)
	GetRoleByName(ctx context.Context, name string, tenantID *string) (*models.Role, error)
	ListRoles(ctx context.Context, tenantID *string) ([]*models.Role, error)
	UpdateRole(ctx context.Context, role *models.Role) error
	DeleteRole(ctx context.Context, id string) error

	// Role-Permission mapping
	AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error
	RemovePermissionFromRole(ctx context.Context, roleID, permissionID string) error
	GetRolePermissions(ctx context.Context, roleID string) ([]*models.RBACPermission, error)

	// User-Role assignments
	AssignRoleToUser(ctx context.Context, userRole *models.UserRole) error
	RemoveRoleFromUser(ctx context.Context, userID, roleID string, tenantID *string) error
	GetUserRoles(ctx context.Context, userID string) ([]*models.UserRole, error)
	GetRoleUsers(ctx context.Context, roleID string) ([]*models.User, error)

	// ========================================
	// Quotas (Version 1.11.7+: Usage Quotas System)
	// ========================================

	// Quota Management
	CreateQuota(ctx context.Context, quota *models.Quota) error
	GetQuota(ctx context.Context, id string) (*models.Quota, error)
	GetQuotaByTarget(ctx context.Context, scope models.QuotaScope, targetID string) (*models.Quota, error)
	ListQuotas(ctx context.Context, scope *models.QuotaScope) ([]*models.Quota, error)
	UpdateQuota(ctx context.Context, quota *models.Quota) error
	DeleteQuota(ctx context.Context, id string) error

	// Quota Usage Tracking
	GetQuotaUsage(ctx context.Context, quotaID string) (*models.QuotaUsage, error)
	GetQuotaUsageByTarget(ctx context.Context, targetID string) (*models.QuotaUsage, error)
	UpdateQuotaUsage(ctx context.Context, usage *models.QuotaUsage) error
	ResetQuotaUsage(ctx context.Context, quotaID string, resetType string) error // "daily" or "monthly"

	// Quota & Usage Combined (for checking)
	GetQuotaWithUsage(ctx context.Context, scope models.QuotaScope, targetID string) (*models.Quota, *models.QuotaUsage, error)

	// ========================================
	// RAG System (Version 1.13.0+: RAG System)
	// ========================================

	// RAG Data Sources
	CreateRAGDataSource(ctx context.Context, source *models.RAGDataSource) error
	GetRAGDataSource(ctx context.Context, id string) (*models.RAGDataSource, error)
	UpdateRAGDataSource(ctx context.Context, source *models.RAGDataSource) error
	DeleteRAGDataSource(ctx context.Context, id string) error
	ListRAGDataSources(ctx context.Context, filter *RAGDataSourceFilter) ([]*models.RAGDataSource, int, error)

	// RAG Documents
	CreateRAGDocument(ctx context.Context, doc *models.RAGDocument) error
	GetRAGDocument(ctx context.Context, id string) (*models.RAGDocument, error)
	UpdateRAGDocument(ctx context.Context, doc *models.RAGDocument) error
	DeleteRAGDocument(ctx context.Context, id string) error
	
	// ========================================
	// Invitations (AUTH-03: Invitation-Only Registration System, v2.2.0)
	// ========================================

	// CreateInvitation создает новое приглашение
	CreateInvitation(ctx context.Context, invitation *models.Invitation) error

	// GetInvitation получает приглашение по ID
	GetInvitation(ctx context.Context, id string) (*models.Invitation, error)

	// GetInvitationWithUsers получает приглашение с информацией о пользователях
	GetInvitationWithUsers(ctx context.Context, id string) (*models.InvitationWithUsers, error)

	// GetInvitationByToken получает приглашение по токену
	GetInvitationByToken(ctx context.Context, token string) (*models.Invitation, error)

	// ListInvitations возвращает список приглашений с фильтрацией
	ListInvitations(ctx context.Context, filter models.InvitationListFilter) ([]*models.Invitation, error)

	// UseInvitation увеличивает счетчик использований приглашения
	UseInvitation(ctx context.Context, token string, userID string) error

	// RevokeInvitation отзывает приглашение
	RevokeInvitation(ctx context.Context, id string, revokedByUserID string, reason string) error

	// GetInvitationStats возвращает статистику по приглашениям
	GetInvitationStats(ctx context.Context) (*models.InvitationStats, error)
	ListRAGDocuments(ctx context.Context, filter *RAGDocumentFilter) ([]*models.RAGDocument, error)

	// RAG Chunks
	CreateRAGChunk(ctx context.Context, chunk *models.RAGChunk) error
	GetRAGChunk(ctx context.Context, id string) (*models.RAGChunk, error)
	ListRAGChunksByDocument(ctx context.Context, documentID string) ([]*models.RAGChunk, error)
	ListRAGChunksBySource(ctx context.Context, sourceID string, limit, offset int) ([]*models.RAGChunk, error)
	DeleteRAGChunksByDocument(ctx context.Context, documentID string) error

	// RAG Jobs Queue
	CreateRAGJob(ctx context.Context, job *models.RAGJob) error
	GetRAGJob(ctx context.Context, id string) (*models.RAGJob, error)
	UpdateRAGJob(ctx context.Context, job *models.RAGJob) error
	GetNextPendingRAGJob(ctx context.Context) (*models.RAGJob, error)
	CountRAGJobsByStatus(ctx context.Context, status string) (int, error)
	DeleteOldRAGJobs(ctx context.Context, cutoffTime time.Time, statuses []string) (int, error)
	UnlockExpiredRAGJobs(ctx context.Context, now time.Time) (int, error)

	// RAG Query Logs
	CreateRAGQueryLog(ctx context.Context, log *models.RAGQueryLog) error
	GetRAGQueryLog(ctx context.Context, id int64) (*models.RAGQueryLog, error)
	ListRAGQueryLogsByUser(ctx context.Context, userID string, limit, offset int) ([]*models.RAGQueryLog, error)
}

// AuditFilters фильтры для запроса audit events (Version 1.11.4+)
type AuditFilters struct {
	EventType  string
	ActorID    string
	Resource   string
	Severity   string
	Status     string
	FromDate   time.Time
	ToDate     time.Time
	Limit      int
	Offset     int
}

// RAGDataSourceFilter фильтр для списка RAG data sources (Version 1.13.0+)
type RAGDataSourceFilter struct {
	UserID     *string
	TenantID   *string
	SourceType *string
	Status     *string
	Tags       []string
	Limit      int
	Offset     int
}

// RAGDocumentFilter фильтр для списка RAG documents (Version 1.13.0+)
type RAGDocumentFilter struct {
	SourceID *string
	Status   *string
	Limit    int
	Offset   int
}

// Tx представляет транзакцию БД
type Tx interface {
	// Commit фиксирует транзакцию
	Commit() error

	// Rollback откатывает транзакцию
	Rollback() error

	// Database - все методы Database доступны в транзакции
	Database
}


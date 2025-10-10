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

	"ollama-openai-proxy/internal/models"
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

	// UpdateUser обновляет данные пользователя
	UpdateUser(ctx context.Context, user *models.User) error

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

	// UpdateTenant обновляет данные tenant
	UpdateTenant(ctx context.Context, tenant *models.Tenant) error

	// DeleteTenant удаляет tenant (soft delete)
	DeleteTenant(ctx context.Context, id string) error

	// ListUserTenants возвращает список tenants пользователя
	ListUserTenants(ctx context.Context, userID string) ([]*models.Tenant, error)

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

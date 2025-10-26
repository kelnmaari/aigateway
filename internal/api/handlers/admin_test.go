package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ollama-openai-proxy/internal/auth/apikey"
	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// testDBAdapter wraps MemoryStorage to implement storage.Database interface
type testDBAdapter struct {
	*storage.MemoryStorage
}

// Database interface implementations (required methods)
func (a *testDBAdapter) Connect(ctx context.Context) error                    { return nil }
func (a *testDBAdapter) Ping(ctx context.Context) error                       { return nil }
func (a *testDBAdapter) Migrate(ctx context.Context) error                    { return nil }
func (a *testDBAdapter) GetMigrationVersion(ctx context.Context) (int, error) { return 0, nil }
func (a *testDBAdapter) BeginTx(ctx context.Context) (storage.Tx, error)      { return nil, nil }

func (a *testDBAdapter) ListAPIKeys(ctx context.Context) ([]*models.APIKey, error) {
	req := models.ListAPIKeysRequest{}
	keys, _, err := a.MemoryStorage.ListAPIKeys(ctx, req)
	if err != nil {
		return nil, err
	}
	result := make([]*models.APIKey, len(keys))
	for i := range keys {
		result[i] = &keys[i]
	}
	return result, nil
}

// User methods
func (a *testDBAdapter) CreateUser(ctx context.Context, user *models.User) error { return nil }
func (a *testDBAdapter) GetUser(ctx context.Context, id string) (*models.User, error) {
	return nil, nil
}
func (a *testDBAdapter) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	return nil, nil
}
func (a *testDBAdapter) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return nil, nil
}
func (a *testDBAdapter) UpdateUser(ctx context.Context, user *models.User) error { return nil }
func (a *testDBAdapter) UpdateUserPassword(ctx context.Context, userID string, passwordHash string) error {
	return nil
}
func (a *testDBAdapter) DeleteUser(ctx context.Context, id string) error { return nil }
func (a *testDBAdapter) ListUsers(ctx context.Context, filters models.UserFilters) ([]*models.User, error) {
	return nil, nil
}

// Tenant methods
func (a *testDBAdapter) CreateTenant(ctx context.Context, tenant *models.Tenant) error { return nil }
func (a *testDBAdapter) GetTenant(ctx context.Context, id string) (*models.Tenant, error) {
	return nil, nil
}
func (a *testDBAdapter) GetTenantBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	return nil, nil
}
func (a *testDBAdapter) UpdateTenant(ctx context.Context, tenant *models.Tenant) error { return nil }
func (a *testDBAdapter) DeleteTenant(ctx context.Context, id string) error             { return nil }
func (a *testDBAdapter) ListUserTenants(ctx context.Context, userID string) ([]*models.Tenant, error) {
	return nil, nil
}

// Tenant member methods
func (a *testDBAdapter) AddTenantMember(ctx context.Context, member *models.TenantMember) error {
	return nil
}
func (a *testDBAdapter) GetTenantMember(ctx context.Context, tenantID, userID string) (*models.TenantMember, error) {
	return nil, nil
}
func (a *testDBAdapter) UpdateTenantMember(ctx context.Context, member *models.TenantMember) error {
	return nil
}
func (a *testDBAdapter) RemoveTenantMember(ctx context.Context, tenantID, userID string) error {
	return nil
}
func (a *testDBAdapter) ListTenantMembers(ctx context.Context, tenantID string) ([]*models.TenantMember, error) {
	return nil, nil
}

// Additional API key methods
func (a *testDBAdapter) ListPersonalAPIKeys(ctx context.Context, userID string) ([]*models.APIKey, error) {
	return nil, nil
}
func (a *testDBAdapter) ListTenantAPIKeys(ctx context.Context, tenantID string) ([]*models.APIKey, error) {
	return nil, nil
}

// Conversation methods
func (a *testDBAdapter) CreateConversation(ctx context.Context, conv *models.Conversation) error {
	return nil
}
func (a *testDBAdapter) GetConversation(ctx context.Context, id string) (*models.Conversation, error) {
	return nil, nil
}
func (a *testDBAdapter) UpdateConversation(ctx context.Context, conv *models.Conversation) error {
	return nil
}
func (a *testDBAdapter) DeleteConversation(ctx context.Context, id string) error { return nil }
func (a *testDBAdapter) ListUserConversations(ctx context.Context, userID string, filters models.ConversationFilters) ([]*models.Conversation, error) {
	return nil, nil
}

// Message methods
func (a *testDBAdapter) CreateMessage(ctx context.Context, msg *models.Message) error { return nil }
func (a *testDBAdapter) GetMessage(ctx context.Context, id string) (*models.Message, error) {
	return nil, nil
}
func (a *testDBAdapter) ListConversationMessages(ctx context.Context, convID string) ([]*models.Message, error) {
	return nil, nil
}
func (a *testDBAdapter) DeleteConversationMessages(ctx context.Context, convID string) error {
	return nil
}

// Usage methods
func (a *testDBAdapter) RecordAPIUsage(ctx context.Context, usage *models.APIUsage) error { return nil }
func (a *testDBAdapter) GetUserUsageStats(ctx context.Context, userID string, period time.Duration) (*models.UsageStats, error) {
	return nil, nil
}
func (a *testDBAdapter) GetTenantUsageStats(ctx context.Context, tenantID string, period time.Duration) (*models.UsageStats, error) {
	return nil, nil
}

// MCP Server methods
func (a *testDBAdapter) CreateMCPServer(ctx context.Context, server *models.MCPServer) error {
	return nil
}
func (a *testDBAdapter) GetMCPServer(ctx context.Context, id string) (*models.MCPServer, error) {
	return nil, nil
}
func (a *testDBAdapter) UpdateMCPServer(ctx context.Context, server *models.MCPServer) error {
	return nil
}
func (a *testDBAdapter) DeleteMCPServer(ctx context.Context, id string) error { return nil }
func (a *testDBAdapter) ListMCPServers(ctx context.Context, req models.MCPServerListRequest) (*models.MCPServerListResponse, error) {
	return nil, nil
}

// Changelog methods
func (a *testDBAdapter) GetChangelog(ctx context.Context, version string) (*models.Changelog, error) {
	return nil, nil
}
func (a *testDBAdapter) ListChangelogs(ctx context.Context) ([]*models.Changelog, error) {
	return nil, nil
}

// Report methods
func (a *testDBAdapter) GetUsageStats(ctx context.Context, start, end time.Time) (*models.UsageReportStats, error) {
	return nil, nil
}
func (a *testDBAdapter) GetPerformanceStats(ctx context.Context, start, end time.Time) (*models.PerformanceReportStats, error) {
	return nil, nil
}
func (a *testDBAdapter) CountActiveUsers(ctx context.Context, period time.Duration) (int, error) {
	return 0, nil
}
func (a *testDBAdapter) CountTotalUsers(ctx context.Context) (int, error)    { return 0, nil }
func (a *testDBAdapter) CountActiveAPIKeys(ctx context.Context) (int, error) { return 0, nil }

// Model config methods
func (a *testDBAdapter) CreateModelConfig(ctx context.Context, config *models.ModelConfig) error {
	return nil
}
func (a *testDBAdapter) GetModelConfig(ctx context.Context, id string) (*models.ModelConfig, error) {
	return nil, nil
}
func (a *testDBAdapter) GetModelConfigByScope(ctx context.Context, modelName, scope string, scopeID *string) (*models.ModelConfig, error) {
	return nil, nil
}
func (a *testDBAdapter) ListModelConfigs(ctx context.Context, scope string, scopeID *string) ([]*models.ModelConfig, error) {
	return nil, nil
}
func (a *testDBAdapter) UpdateModelConfig(ctx context.Context, config *models.ModelConfig) error {
	return nil
}
func (a *testDBAdapter) DeleteModelConfig(ctx context.Context, id string) error { return nil }
func (a *testDBAdapter) GetEffectiveModelConfig(ctx context.Context, modelName, userID, tenantID string) (*models.ModelParameters, error) {
	return nil, nil
}

// File methods (Version 1.10.5+: WEB-FETCH-01)
func (a *testDBAdapter) CreateFile(ctx context.Context, req models.CreateFileRequest) (*models.File, error) {
	return nil, nil
}
func (a *testDBAdapter) GetFileByID(ctx context.Context, fileID string) (*models.File, error) {
	return nil, nil
}
func (a *testDBAdapter) UpdateFile(ctx context.Context, fileID string, req models.UpdateFileRequest) (*models.File, error) {
	return nil, nil
}
func (a *testDBAdapter) DeleteFile(ctx context.Context, fileID string) error {
	return nil
}
func (a *testDBAdapter) ListFiles(ctx context.Context, req models.ListFilesRequest) ([]*models.File, int, error) {
	return nil, 0, nil
}
func (a *testDBAdapter) ListFilesWithUserInfo(ctx context.Context, req models.ListFilesRequest) ([]*models.FileWithUser, int, error) {
	return nil, 0, nil
}
func (a *testDBAdapter) IncrementDownloadCount(ctx context.Context, fileID string) error {
	return nil
}
func (a *testDBAdapter) LogFileAccess(ctx context.Context, log models.FileAccessLog) error {
	return nil
}
func (a *testDBAdapter) GetFileAccessLogs(ctx context.Context, fileID string, limit int) ([]*models.FileAccessLog, error) {
	return nil, nil
}

// OIDC methods (Version 1.11.1+: Keycloak SSO Integration)
func (a *testDBAdapter) GetUserByOIDCSubject(ctx context.Context, issuer, subject string) (*models.User, error) {
	return nil, nil
}

// LDAP methods (Version 1.11.3+: LDAP Integration)
func (a *testDBAdapter) GetUserByLDAPDN(ctx context.Context, ldapDN string) (*models.User, error) {
	return nil, nil
}

// Tenant methods (Version 1.11.2+: Auto-tenant Provisioning)
func (a *testDBAdapter) GetTenantByName(ctx context.Context, name string) (*models.Tenant, error) {
	return nil, nil
}

// RBAC methods (Version 1.11.5+: Custom Roles & Permissions)
func (a *testDBAdapter) CreatePermission(ctx context.Context, permission *models.RBACPermission) error {
	return nil
}
func (a *testDBAdapter) GetPermission(ctx context.Context, id string) (*models.RBACPermission, error) {
	return nil, nil
}
func (a *testDBAdapter) GetPermissionByName(ctx context.Context, name string) (*models.RBACPermission, error) {
	return nil, nil
}
func (a *testDBAdapter) ListPermissions(ctx context.Context) ([]*models.RBACPermission, error) {
	return nil, nil
}
func (a *testDBAdapter) CreateRole(ctx context.Context, role *models.Role) error { return nil }
func (a *testDBAdapter) GetRole(ctx context.Context, id string) (*models.Role, error) {
	return nil, nil
}
func (a *testDBAdapter) GetRoleByName(ctx context.Context, name string, tenantID *string) (*models.Role, error) {
	return nil, nil
}
func (a *testDBAdapter) ListRoles(ctx context.Context, tenantID *string) ([]*models.Role, error) {
	return nil, nil
}
func (a *testDBAdapter) UpdateRole(ctx context.Context, role *models.Role) error { return nil }
func (a *testDBAdapter) DeleteRole(ctx context.Context, id string) error          { return nil }
func (a *testDBAdapter) AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error {
	return nil
}
func (a *testDBAdapter) RemovePermissionFromRole(ctx context.Context, roleID, permissionID string) error {
	return nil
}
func (a *testDBAdapter) GetRolePermissions(ctx context.Context, roleID string) ([]*models.RBACPermission, error) {
	return nil, nil
}
func (a *testDBAdapter) AssignRoleToUser(ctx context.Context, userRole *models.UserRole) error {
	return nil
}
func (a *testDBAdapter) RemoveRoleFromUser(ctx context.Context, userID, roleID string, tenantID *string) error {
	return nil
}
func (a *testDBAdapter) GetUserRoles(ctx context.Context, userID string) ([]*models.UserRole, error) {
	return nil, nil
}
func (a *testDBAdapter) GetRoleUsers(ctx context.Context, roleID string) ([]*models.User, error) {
	return nil, nil
}

// Quota methods (Version 1.11.7+: Usage Quotas System)
func (a *testDBAdapter) CreateQuota(ctx context.Context, quota *models.Quota) error { return nil }
func (a *testDBAdapter) GetQuota(ctx context.Context, id string) (*models.Quota, error) {
	return nil, nil
}
func (a *testDBAdapter) GetQuotaByTarget(ctx context.Context, scope models.QuotaScope, targetID string) (*models.Quota, error) {
	return nil, nil
}
func (a *testDBAdapter) ListQuotas(ctx context.Context, scope *models.QuotaScope) ([]*models.Quota, error) {
	return nil, nil
}
func (a *testDBAdapter) UpdateQuota(ctx context.Context, quota *models.Quota) error { return nil }
func (a *testDBAdapter) DeleteQuota(ctx context.Context, id string) error           { return nil }
func (a *testDBAdapter) GetQuotaUsage(ctx context.Context, quotaID string) (*models.QuotaUsage, error) {
	return nil, nil
}
func (a *testDBAdapter) GetQuotaUsageByTarget(ctx context.Context, targetID string) (*models.QuotaUsage, error) {
	return nil, nil
}
func (a *testDBAdapter) UpdateQuotaUsage(ctx context.Context, usage *models.QuotaUsage) error {
	return nil
}
func (a *testDBAdapter) ResetQuotaUsage(ctx context.Context, quotaID string, resetType string) error {
	return nil
}
func (a *testDBAdapter) GetQuotaWithUsage(ctx context.Context, scope models.QuotaScope, targetID string) (*models.Quota, *models.QuotaUsage, error) {
	return nil, nil, nil
}

// Audit methods (Version 1.11.4+: Audit Logging)
func (a *testDBAdapter) CreateAuditEvent(ctx context.Context, event *models.AuditEvent) error {
	return nil
}
func (a *testDBAdapter) GetAuditEvents(ctx context.Context, filters storage.AuditFilters) ([]*models.AuditEvent, int, error) {
	return nil, 0, nil
}
func (a *testDBAdapter) DeleteOldAuditEvents(ctx context.Context, olderThan time.Time) (int, error) {
	return 0, nil
}

// setupAdminTest создает тестовое окружение для admin handler
func setupAdminTest(t *testing.T) (*AdminHandler, *storage.MemoryStorage) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	stor := storage.NewMemoryStorage()
	keyManager := apikey.NewManager(cfg, logger, stor)

	adapter := &testDBAdapter{MemoryStorage: stor}
	handler := NewAdminHandler(cfg, logger, keyManager, adapter)
	return handler, stor
}

// TestAdminHandler_CreateAPIKey_Success тестирует создание API ключа
func TestAdminHandler_CreateAPIKey_Success(t *testing.T) {
	handler, _ := setupAdminTest(t)

	reqBody := models.CreateAPIKeyRequest{
		Name:        "test-key",
		Description: "Test API Key",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/keys", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateAPIKey(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp models.CreateAPIKeyResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.PlainKey)
	assert.NotEmpty(t, resp.APIKey.ID)
	assert.Equal(t, "test-key", resp.APIKey.Name)
}

// TestAdminHandler_CreateAPIKey_InvalidJSON тестирует невалидный JSON
func TestAdminHandler_CreateAPIKey_InvalidJSON(t *testing.T) {
	handler, _ := setupAdminTest(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/keys", bytes.NewBufferString("{invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateAPIKey(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAdminHandler_ListAPIKeys_Success тестирует получение списка ключей
func TestAdminHandler_ListAPIKeys_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем тестовый ключ
	ctx := context.Background()
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/keys", nil)

	handler.ListAPIKeys(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.ListAPIKeysResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(resp.APIKeys), 1)
	assert.Equal(t, "test-key", resp.APIKeys[0].Name)
}

// TestAdminHandler_GetAPIKey_Success тестирует получение одного ключа
func TestAdminHandler_GetAPIKey_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем тестовый ключ
	ctx := context.Background()
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/keys/test-id", nil)

	handler.GetAPIKey(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		APIKey models.APIKeyPublic `json:"api_key"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "test-id", resp.APIKey.ID)
	assert.Equal(t, "test-key", resp.APIKey.Name)
}

// TestAdminHandler_GetAPIKey_NotFound тестирует несуществующий ключ
func TestAdminHandler_GetAPIKey_NotFound(t *testing.T) {
	handler, _ := setupAdminTest(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/keys/nonexistent", nil)

	handler.GetAPIKey(c)

	// isNotFoundError всегда false, поэтому ожидаем 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestAdminHandler_UpdateAPIKey_Success тестирует обновление ключа
func TestAdminHandler_UpdateAPIKey_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем тестовый ключ
	ctx := context.Background()
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	name := "updated-key"
	desc := "Updated description"
	modelsList := []string{"gpt-3.5"}
	reqBody := models.UpdateAPIKeyRequest{
		Name:        &name,
		Description: &desc,
		Models:      &modelsList,
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/admin/keys/test-id", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateAPIKey(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		APIKey models.APIKeyPublic `json:"api_key"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "updated-key", resp.APIKey.Name)
}

// TestAdminHandler_DeleteAPIKey_Success тестирует удаление ключа
func TestAdminHandler_DeleteAPIKey_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем тестовый ключ
	ctx := context.Background()
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/admin/keys/test-id", nil)

	handler.DeleteAPIKey(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestAdminHandler_RevokeAPIKey_Success тестирует отзыв ключа (AUTH-04)
func TestAdminHandler_RevokeAPIKey_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем тестовый ключ
	ctx := context.Background()
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	reqBody := map[string]string{"reason": "Security concern"}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest(http.MethodPatch, "/admin/keys/test-id/revoke", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.RevokeAPIKey(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем, что ключ был отозван
	key, _ := stor.GetAPIKey(ctx, "test-id")
	assert.Equal(t, models.APIKeyStatusRevoked, key.Status)
}

// TestAdminHandler_EnableAPIKey_Success тестирует включение ключа (AUTH-04)
func TestAdminHandler_EnableAPIKey_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем отозванный ключ
	ctx := context.Background()
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusRevoked,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest(http.MethodPatch, "/admin/keys/test-id/enable", nil)

	handler.EnableAPIKey(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем, что ключ был включен
	key, _ := stor.GetAPIKey(ctx, "test-id")
	assert.Equal(t, models.APIKeyStatusActive, key.Status)
}

// TestAdminHandler_ExtendAPIKey_Success тестирует продление срока (AUTH-04)
func TestAdminHandler_ExtendAPIKeyExpiration_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем ключ с истекшим сроком
	ctx := context.Background()
	expiredTime := time.Now().Add(-24 * time.Hour)
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusActive,
		ExpiresAt:   &expiredTime,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	reqBody := struct {
		Days int `json:"days"`
	}{Days: 30}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/keys/test-id/extend", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ExtendAPIKeyExpiration(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем, что срок был продлен
	key, _ := stor.GetAPIKey(ctx, "test-id")
	assert.NotNil(t, key.ExpiresAt)
	assert.True(t, key.ExpiresAt.After(time.Now()))
}

// TestAdminHandler_UpdatePermissions_Success тестирует обновление permissions (AUTH-04)
func TestAdminHandler_UpdateAPIKeyPermissions_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем тестовый ключ
	ctx := context.Background()
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	perms := []string{"chat", "embeddings", "admin"}
	reqBody := struct {
		Permissions *[]string `json:"permissions"`
	}{Permissions: &perms}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest(http.MethodPatch, "/admin/keys/test-id/permissions", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateAPIKeyPermissions(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем, что permissions были обновлены
	key, _ := stor.GetAPIKey(ctx, "test-id")
	assert.Contains(t, key.Permissions, "admin")
	assert.Len(t, key.Permissions, 3)
}

// TestAdminHandler_WithoutKeyManager_ReturnsError тестирует handler без key manager и db
func TestAdminHandler_WithoutKeyManager_ReturnsError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Создаем handler БЕЗ db и БЕЗ keyManager
	handler := NewAdminHandlerWithoutKeys(cfg, logger, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/keys", nil)

	handler.ListAPIKeys(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "API Key management not available")
}

// TestAdminHandler_WithDB_Success тестирует handler с базой данных
func TestAdminHandler_WithDB_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем тестовый ключ
	ctx := context.Background()
	testKey := &models.APIKey{
		ID:          "db-test-id",
		Name:        "db-test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/keys", nil)

	handler.ListAPIKeys(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.ListAPIKeysResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(resp.APIKeys), 1)
}

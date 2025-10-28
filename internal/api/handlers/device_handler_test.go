package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// mockDeviceStorage implements minimal storage.Database interface for testing
type mockDeviceStorage struct {
	apiKeys map[string]*models.APIKey
}

func newMockDeviceStorage() *mockDeviceStorage {
	return &mockDeviceStorage{
		apiKeys: make(map[string]*models.APIKey),
	}
}

func (m *mockDeviceStorage) FindAPIKeyByDeviceFingerprint(ctx context.Context, userID, fingerprint string) (*models.APIKey, error) {
	for _, key := range m.apiKeys {
		if key.UserID != nil && *key.UserID == userID &&
			key.DeviceFingerprint != nil && *key.DeviceFingerprint == fingerprint {
			return key, nil
		}
	}
	return nil, storage.ErrNotFound
}

func (m *mockDeviceStorage) CreateAPIKey(ctx context.Context, key *models.APIKey) error {
	m.apiKeys[key.ID] = key
	return nil
}

func (m *mockDeviceStorage) UpdateAPIKeyLastSeen(ctx context.Context, keyID string) error {
	if key, exists := m.apiKeys[keyID]; exists {
		now := time.Now()
		key.LastSeenAt = &now
		return nil
	}
	return storage.ErrNotFound
}

// Stub implementations for storage.Database interface (not used in device tests)
func (m *mockDeviceStorage) Connect(ctx context.Context) error { return nil }
func (m *mockDeviceStorage) Close() error                      { return nil }
func (m *mockDeviceStorage) Ping(ctx context.Context) error    { return nil }
func (m *mockDeviceStorage) Migrate(ctx context.Context) error { return nil }
func (m *mockDeviceStorage) BeginTx(ctx context.Context) (storage.Tx, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetMigrationVersion(ctx context.Context) (int, error) { return 0, nil }
func (m *mockDeviceStorage) GetAPIKey(ctx context.Context, id string) (*models.APIKey, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetAPIKeyByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateAPIKey(ctx context.Context, key *models.APIKey) error { return nil }
func (m *mockDeviceStorage) DeleteAPIKey(ctx context.Context, id string) error          { return nil }
func (m *mockDeviceStorage) RevokeAPIKey(ctx context.Context, id string, reason string) error {
	return nil
}
func (m *mockDeviceStorage) EnableAPIKey(ctx context.Context, id string) error        { return nil }
func (m *mockDeviceStorage) ListAPIKeys(ctx context.Context) ([]*models.APIKey, error) { return nil, nil }
func (m *mockDeviceStorage) ListPersonalAPIKeys(ctx context.Context, userID string) ([]*models.APIKey, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListTenantAPIKeys(ctx context.Context, tenantID string) ([]*models.APIKey, error) {
	return nil, nil
}
func (m *mockDeviceStorage) CreateConversation(ctx context.Context, conv *models.Conversation) error {
	return nil
}
func (m *mockDeviceStorage) GetConversation(ctx context.Context, id string) (*models.Conversation, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListConversations(ctx context.Context, userID string) ([]*models.Conversation, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateConversation(ctx context.Context, conv *models.Conversation) error {
	return nil
}
func (m *mockDeviceStorage) DeleteConversation(ctx context.Context, id string) error { return nil }
func (m *mockDeviceStorage) CreateMessage(ctx context.Context, msg *models.Message) error {
	return nil
}
func (m *mockDeviceStorage) GetMessage(ctx context.Context, id string) (*models.Message, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListMessages(ctx context.Context, conversationID string) ([]*models.Message, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateMessage(ctx context.Context, msg *models.Message) error { return nil }
func (m *mockDeviceStorage) DeleteMessage(ctx context.Context, id string) error           { return nil }
func (m *mockDeviceStorage) CreateUser(ctx context.Context, user *models.User) error      { return nil }
func (m *mockDeviceStorage) GetUser(ctx context.Context, id string) (*models.User, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetUserByOIDCSubject(ctx context.Context, issuer, subject string) (*models.User, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetUserByLDAPDN(ctx context.Context, dn string) (*models.User, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateUser(ctx context.Context, user *models.User) error { return nil }
func (m *mockDeviceStorage) DeleteUser(ctx context.Context, id string) error         { return nil }
func (m *mockDeviceStorage) ListUsers(ctx context.Context, filters models.UserFilters) ([]*models.User, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetUsersWithDetails(ctx context.Context, filters models.UserFilters) ([]*models.UserWithDetails, error) {
	return nil, nil
}
func (m *mockDeviceStorage) CreateTenant(ctx context.Context, tenant *models.Tenant) error {
	return nil
}
func (m *mockDeviceStorage) GetTenant(ctx context.Context, id string) (*models.Tenant, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetTenantBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetTenantByName(ctx context.Context, name string) (*models.Tenant, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateTenant(ctx context.Context, tenant *models.Tenant) error { return nil }
func (m *mockDeviceStorage) DeleteTenant(ctx context.Context, id string) error             { return nil }
func (m *mockDeviceStorage) ListTenants(ctx context.Context) ([]*models.Tenant, error) {
	return nil, nil
}
func (m *mockDeviceStorage) AddTenantMember(ctx context.Context, member *models.TenantMember) error {
	return nil
}
func (m *mockDeviceStorage) RemoveTenantMember(ctx context.Context, tenantID, userID string) error {
	return nil
}
func (m *mockDeviceStorage) GetTenantMember(ctx context.Context, tenantID, userID string) (*models.TenantMember, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListTenantMembers(ctx context.Context, tenantID string) ([]*models.TenantMember, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListUserTenants(ctx context.Context, userID string) ([]*models.Tenant, error) {
	return nil, nil
}
func (m *mockDeviceStorage) RecordUsage(ctx context.Context, usage *models.Usage) error { return nil }
func (m *mockDeviceStorage) GetUserUsageStats(ctx context.Context, userID string, start, end time.Time) (*models.UsageStats, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetTenantUsageStats(ctx context.Context, tenantID string, start, end time.Time) (*models.UsageStats, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetUsageStats(ctx context.Context, start, end time.Time) (*models.UsageReport, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetPerformanceStats(ctx context.Context, start, end time.Time) (*models.PerformanceReport, error) {
	return nil, nil
}
func (m *mockDeviceStorage) CreateInvitation(ctx context.Context, inv *models.Invitation) error {
	return nil
}
func (m *mockDeviceStorage) GetInvitation(ctx context.Context, id string) (*models.Invitation, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetInvitationWithUsers(ctx context.Context, id string) (*models.InvitationWithUsers, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetInvitationByToken(ctx context.Context, token string) (*models.Invitation, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListInvitations(ctx context.Context, filters models.InvitationListFilter) ([]*models.Invitation, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UseInvitation(ctx context.Context, invID, userID string) error {
	return nil
}
func (m *mockDeviceStorage) RevokeInvitation(ctx context.Context, invID, revokedByUserID, reason string) error {
	return nil
}
func (m *mockDeviceStorage) GetInvitationStats(ctx context.Context) (*models.InvitationStats, error) {
	return nil, nil
}
func (m *mockDeviceStorage) CreatePermission(ctx context.Context, perm *models.RBACPermission) error {
	return nil
}
func (m *mockDeviceStorage) GetPermission(ctx context.Context, id string) (*models.RBACPermission, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetPermissionByName(ctx context.Context, name string) (*models.RBACPermission, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdatePermission(ctx context.Context, perm *models.RBACPermission) error {
	return nil
}
func (m *mockDeviceStorage) DeletePermission(ctx context.Context, id string) error { return nil }
func (m *mockDeviceStorage) ListPermissions(ctx context.Context) ([]*models.RBACPermission, error) {
	return nil, nil
}
func (m *mockDeviceStorage) CreateRole(ctx context.Context, role *models.Role) error { return nil }
func (m *mockDeviceStorage) GetRole(ctx context.Context, id string) (*models.Role, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetRoleByName(ctx context.Context, name string, tenantID *string) (*models.Role, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateRole(ctx context.Context, role *models.Role) error { return nil }
func (m *mockDeviceStorage) DeleteRole(ctx context.Context, id string) error         { return nil }
func (m *mockDeviceStorage) ListRoles(ctx context.Context, tenantID *string) ([]*models.Role, error) {
	return nil, nil
}
func (m *mockDeviceStorage) AddRolePermission(ctx context.Context, rp *models.RolePermission) error {
	return nil
}
func (m *mockDeviceStorage) RemoveRolePermission(ctx context.Context, roleID, permID string) error {
	return nil
}
func (m *mockDeviceStorage) GetRolePermissions(ctx context.Context, roleID string) ([]*models.RBACPermission, error) {
	return nil, nil
}
func (m *mockDeviceStorage) AssignUserRole(ctx context.Context, ur *models.UserRole) error { return nil }
func (m *mockDeviceStorage) RevokeUserRole(ctx context.Context, userID, roleID, scopeType, scopeID string) error {
	return nil
}
func (m *mockDeviceStorage) GetUserRoles(ctx context.Context, userID string) ([]*models.UserRole, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetRoleUsers(ctx context.Context, roleID string) ([]*models.UserRole, error) {
	return nil, nil
}
func (m *mockDeviceStorage) CheckUserPermission(ctx context.Context, userID, permission, scopeType, scopeID string) (bool, error) {
	return false, nil
}
func (m *mockDeviceStorage) CreateQuota(ctx context.Context, quota *models.Quota) error { return nil }
func (m *mockDeviceStorage) GetQuota(ctx context.Context, id string) (*models.Quota, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetQuotaByTarget(ctx context.Context, scope models.QuotaScope, targetID string) (*models.Quota, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateQuota(ctx context.Context, quota *models.Quota) error { return nil }
func (m *mockDeviceStorage) DeleteQuota(ctx context.Context, id string) error           { return nil }
func (m *mockDeviceStorage) ListQuotas(ctx context.Context, scope *models.QuotaScope) ([]*models.Quota, error) {
	return nil, nil
}
func (m *mockDeviceStorage) CreateQuotaUsage(ctx context.Context, usage *models.QuotaUsage) error {
	return nil
}
func (m *mockDeviceStorage) GetQuotaUsage(ctx context.Context, quotaID string) (*models.QuotaUsage, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetQuotaUsageByTarget(ctx context.Context, scope models.QuotaScope, targetID string) (*models.QuotaUsage, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateQuotaUsage(ctx context.Context, usage *models.QuotaUsage) error {
	return nil
}
func (m *mockDeviceStorage) ResetQuotaUsage(ctx context.Context, quotaID string) error { return nil }
func (m *mockDeviceStorage) GetQuotaWithUsage(ctx context.Context, id string) (*models.Quota, *models.QuotaUsage, error) {
	return nil, nil, nil
}
func (m *mockDeviceStorage) CreateModelConfig(ctx context.Context, mc *models.ModelConfig) error {
	return nil
}
func (m *mockDeviceStorage) GetModelConfig(ctx context.Context, id string) (*models.ModelConfig, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetModelConfigByScope(ctx context.Context, model string, scope models.ModelConfigScope, scopeID string) (*models.ModelConfig, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListModelConfigs(ctx context.Context, model string, scope *models.ModelConfigScope) ([]*models.ModelConfig, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateModelConfig(ctx context.Context, mc *models.ModelConfig) error {
	return nil
}
func (m *mockDeviceStorage) DeleteModelConfig(ctx context.Context, id string) error { return nil }
func (m *mockDeviceStorage) GetEffectiveModelConfig(ctx context.Context, model, userID, tenantID string) (*models.ModelConfig, error) {
	return nil, nil
}
func (m *mockDeviceStorage) CreateMCPServer(ctx context.Context, srv *models.MCPServer) error {
	return nil
}
func (m *mockDeviceStorage) GetMCPServer(ctx context.Context, id string) (*models.MCPServer, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListMCPServers(ctx context.Context, enabledOnly bool) ([]*models.MCPServer, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateMCPServer(ctx context.Context, srv *models.MCPServer) error {
	return nil
}
func (m *mockDeviceStorage) DeleteMCPServer(ctx context.Context, id string) error { return nil }
func (m *mockDeviceStorage) CreateFileUpload(ctx context.Context, f *models.File) error {
	return nil
}
func (m *mockDeviceStorage) GetFileByID(ctx context.Context, id string) (*models.File, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListFiles(ctx context.Context, filters models.FileFilters) ([]*models.File, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateFileMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	return nil
}
func (m *mockDeviceStorage) DeleteFile(ctx context.Context, id string) error { return nil }
func (m *mockDeviceStorage) RecordFileAccess(ctx context.Context, log *models.FileAccessLog) error {
	return nil
}
func (m *mockDeviceStorage) GetFileAccessLogs(ctx context.Context, fileID string) ([]*models.FileAccessLog, error) {
	return nil, nil
}
func (m *mockDeviceStorage) CreateRAGDataSource(ctx context.Context, ds *models.RAGDataSource) error {
	return nil
}
func (m *mockDeviceStorage) GetRAGDataSource(ctx context.Context, id string) (*models.RAGDataSource, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListRAGDataSources(ctx context.Context, filters models.RAGDataSourceFilters) ([]*models.RAGDataSource, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateRAGDataSource(ctx context.Context, ds *models.RAGDataSource) error {
	return nil
}
func (m *mockDeviceStorage) DeleteRAGDataSource(ctx context.Context, id string) error { return nil }
func (m *mockDeviceStorage) CreateRAGDocument(ctx context.Context, doc *models.RAGDocument) error {
	return nil
}
func (m *mockDeviceStorage) GetRAGDocument(ctx context.Context, id string) (*models.RAGDocument, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListRAGDocuments(ctx context.Context, dsID string) ([]*models.RAGDocument, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateRAGDocument(ctx context.Context, doc *models.RAGDocument) error {
	return nil
}
func (m *mockDeviceStorage) DeleteRAGDocument(ctx context.Context, id string) error { return nil }
func (m *mockDeviceStorage) CreateRAGChunk(ctx context.Context, chunk *models.RAGChunk) error {
	return nil
}
func (m *mockDeviceStorage) GetRAGChunk(ctx context.Context, id string) (*models.RAGChunk, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListRAGChunks(ctx context.Context, docID string) ([]*models.RAGChunk, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateRAGChunk(ctx context.Context, chunk *models.RAGChunk) error {
	return nil
}
func (m *mockDeviceStorage) DeleteRAGChunk(ctx context.Context, id string) error { return nil }
func (m *mockDeviceStorage) SearchRAGChunks(ctx context.Context, dsID string, embedding []float64, limit int) ([]*models.RAGChunk, error) {
	return nil, nil
}
func (m *mockDeviceStorage) CreateRAGJob(ctx context.Context, job *models.RAGJob) error { return nil }
func (m *mockDeviceStorage) GetRAGJob(ctx context.Context, id string) (*models.RAGJob, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListRAGJobs(ctx context.Context, dsID string) ([]*models.RAGJob, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateRAGJob(ctx context.Context, job *models.RAGJob) error { return nil }
func (m *mockDeviceStorage) DeleteRAGJob(ctx context.Context, id string) error           { return nil }
func (m *mockDeviceStorage) GetNextPendingRAGJob(ctx context.Context) (*models.RAGJob, error) {
	return nil, nil
}
func (m *mockDeviceStorage) CountRAGJobsByStatus(ctx context.Context, status models.RAGJobStatus) (int, error) {
	return 0, nil
}
func (m *mockDeviceStorage) CreateRAGQueryLog(ctx context.Context, log *models.RAGQueryLog) error {
	return nil
}
func (m *mockDeviceStorage) GetRAGQueryLog(ctx context.Context, id string) (*models.RAGQueryLog, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListRAGQueryLogs(ctx context.Context, filters models.RAGQueryLogFilters) ([]*models.RAGQueryLog, error) {
	return nil, nil
}
func (m *mockDeviceStorage) CreateAuditEvent(ctx context.Context, event *models.AuditEvent) error {
	return nil
}
func (m *mockDeviceStorage) GetAuditEvents(ctx context.Context, filters models.AuditFilters) ([]*models.AuditEvent, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetChangelog(ctx context.Context, version string) (*models.Changelog, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListChangelogs(ctx context.Context, limit int) ([]*models.Changelog, error) {
	return nil, nil
}
func (m *mockDeviceStorage) CreateModelProvider(ctx context.Context, prov *models.ModelProvider) error {
	return nil
}
func (m *mockDeviceStorage) GetModelProvider(ctx context.Context, id string) (*models.ModelProvider, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetModelProviderByName(ctx context.Context, name string) (*models.ModelProvider, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListModelProviders(ctx context.Context) ([]*models.ModelProvider, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateModelProvider(ctx context.Context, prov *models.ModelProvider) error {
	return nil
}
func (m *mockDeviceStorage) DeleteModelProvider(ctx context.Context, id string) error { return nil }
func (m *mockDeviceStorage) CreateModelRegistry(ctx context.Context, mod *models.ModelRegistry) error {
	return nil
}
func (m *mockDeviceStorage) GetModelRegistry(ctx context.Context, id string) (*models.ModelRegistry, error) {
	return nil, nil
}
func (m *mockDeviceStorage) GetModelRegistryByModelID(ctx context.Context, modelID string) (*models.ModelRegistry, error) {
	return nil, nil
}
func (m *mockDeviceStorage) ListModelRegistry(ctx context.Context, filters models.ModelRegistryFilters) ([]*models.ModelRegistry, error) {
	return nil, nil
}
func (m *mockDeviceStorage) UpdateModelRegistry(ctx context.Context, mod *models.ModelRegistry) error {
	return nil
}
func (m *mockDeviceStorage) DeleteModelRegistry(ctx context.Context, id string) error { return nil }
func (m *mockDeviceStorage) UpsertModelFromDiscovery(ctx context.Context, mod *models.ModelRegistry) error {
	return nil
}
func (m *mockDeviceStorage) GetModelRegistryStats(ctx context.Context) (*models.ModelRegistryStats, error) {
	return nil, nil
}

// setupDeviceTestRouter sets up Gin router with DeviceHandler for testing
func setupDeviceTestRouter(db storage.Database) (*gin.Engine, *DeviceHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	logger := logrus.New()
	logger.SetOutput(logrus.StandardLogger().Out)

	handler := NewDeviceHandler(db, logger)

	// Setup route with JWT middleware simulation
	auth := router.Group("/api/auth")
	{
		// Simulate JWT middleware - set user_id in context
		auth.Use(func(c *gin.Context) {
			c.Set("user_id", "test-user-123")
			c.Next()
		})
		auth.POST("/devices/register", handler.RegisterDevice)
	}

	return router, handler
}

func TestDeviceHandler_RegisterDevice_NewDevice(t *testing.T) {
	db := newMockDeviceStorage()
	router, _ := setupDeviceTestRouter(db)

	req := models.DeviceRegistrationRequest{
		DeviceName:        "Test Laptop",
		DeviceOS:          "windows",
		DeviceHostname:    "TEST-PC",
		DeviceVersion:     "1.0.0",
		DeviceFingerprint: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
		AutoExpireDays:    90,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/devices/register", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp models.DeviceRegistrationResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.NotEmpty(t, resp.APIKey)
	assert.NotEmpty(t, resp.KeyID)
	assert.Equal(t, "Test Laptop", resp.DeviceName)
	assert.Equal(t, "windows", resp.DeviceOS)
	assert.True(t, resp.IsNewDevice)
	assert.NotNil(t, resp.ExpiresAt)
}

func TestDeviceHandler_RegisterDevice_ExistingDevice(t *testing.T) {
	db := newMockDeviceStorage()
	router, _ := setupDeviceTestRouter(db)

	fingerprint := "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2"
	userID := "test-user-123"

	// Create existing device
	deviceName := "Existing Laptop"
	deviceOS := "windows"
	existingKey := &models.APIKey{
		ID:                "existing-key-id",
		UserID:            &userID,
		DeviceName:        &deviceName,
		DeviceOS:          &deviceOS,
		DeviceFingerprint: &fingerprint,
		CreatedAt:         time.Now().Add(-7 * 24 * time.Hour),
		Status:            models.APIKeyStatusActive,
	}
	db.apiKeys["existing-key-id"] = existingKey

	req := models.DeviceRegistrationRequest{
		DeviceName:        "Test Laptop",
		DeviceOS:          "windows",
		DeviceHostname:    "TEST-PC",
		DeviceVersion:     "1.0.1", // Different version
		DeviceFingerprint: fingerprint,
		AutoExpireDays:    90,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/devices/register", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.DeviceRegistrationResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "existing-key-id", resp.KeyID)
	assert.Equal(t, "Existing Laptop", resp.DeviceName)
	assert.False(t, resp.IsNewDevice)
	assert.Contains(t, resp.Message, "already registered")
}

func TestDeviceHandler_RegisterDevice_InvalidOS(t *testing.T) {
	db := newMockDeviceStorage()
	router, _ := setupDeviceTestRouter(db)

	req := models.DeviceRegistrationRequest{
		DeviceName:        "Test Laptop",
		DeviceOS:          "invalid-os", // Invalid OS
		DeviceHostname:    "TEST-PC",
		DeviceVersion:     "1.0.0",
		DeviceFingerprint: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
		AutoExpireDays:    90,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/devices/register", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeviceHandler_RegisterDevice_ShortFingerprint(t *testing.T) {
	db := newMockDeviceStorage()
	router, _ := setupDeviceTestRouter(db)

	req := models.DeviceRegistrationRequest{
		DeviceName:        "Test Laptop",
		DeviceOS:          "windows",
		DeviceHostname:    "TEST-PC",
		DeviceVersion:     "1.0.0",
		DeviceFingerprint: "short", // Too short
		AutoExpireDays:    90,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/devices/register", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeviceHandler_RegisterDevice_AutoGenerateDeviceName(t *testing.T) {
	db := newMockDeviceStorage()
	router, _ := setupDeviceTestRouter(db)

	req := models.DeviceRegistrationRequest{
		DeviceName:        "", // Empty name - should auto-generate
		DeviceOS:          "darwin",
		DeviceHostname:    "MacBook-Pro",
		DeviceVersion:     "1.0.0",
		DeviceFingerprint: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
		AutoExpireDays:    90,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/devices/register", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp models.DeviceRegistrationResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.NotEmpty(t, resp.DeviceName)
	assert.Contains(t, resp.DeviceName, "macOS")
	assert.Contains(t, resp.DeviceName, "MacBook-Pro")
}

func TestDeviceHandler_RegisterDevice_NoUserID(t *testing.T) {
	db := newMockDeviceStorage()
	gin.SetMode(gin.TestMode)
	router := gin.New()

	logger := logrus.New()
	handler := NewDeviceHandler(db, logger)

	// NO JWT middleware - no user_id in context
	router.POST("/api/auth/devices/register", handler.RegisterDevice)

	req := models.DeviceRegistrationRequest{
		DeviceName:        "Test Laptop",
		DeviceOS:          "windows",
		DeviceHostname:    "TEST-PC",
		DeviceVersion:     "1.0.0",
		DeviceFingerprint: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2",
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/devices/register", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGenerateDeviceFingerprint(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		os         string
		hostname   string
		macAddr    string
		cpuModel   string
		diskSerial string
		wantLen    int
	}{
		{
			name:       "Complete fingerprint",
			userID:     "user-123",
			os:         "windows",
			hostname:   "DESKTOP-PC",
			macAddr:    "AA:BB:CC:DD:EE:FF",
			cpuModel:   "Intel Core i7",
			diskSerial: "ABC123",
			wantLen:    64, // SHA256 hex = 64 chars
		},
		{
			name:       "Partial data",
			userID:     "user-456",
			os:         "linux",
			hostname:   "ubuntu-server",
			macAddr:    "",
			cpuModel:   "",
			diskSerial: "",
			wantLen:    64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := GenerateDeviceFingerprint(tt.userID, tt.os, tt.hostname, tt.macAddr, tt.cpuModel, tt.diskSerial)
			assert.Len(t, fp, tt.wantLen)

			// Same inputs should produce same fingerprint
			fp2 := GenerateDeviceFingerprint(tt.userID, tt.os, tt.hostname, tt.macAddr, tt.cpuModel, tt.diskSerial)
			assert.Equal(t, fp, fp2)

			// Different inputs should produce different fingerprint
			fp3 := GenerateDeviceFingerprint(tt.userID, tt.os, tt.hostname+"different", tt.macAddr, tt.cpuModel, tt.diskSerial)
			assert.NotEqual(t, fp, fp3)
		})
	}
}

func TestModels_GenerateDeviceName(t *testing.T) {
	tests := []struct {
		name     string
		os       string
		hostname string
		wantOS   string
	}{
		{
			name:     "Windows with hostname",
			os:       "windows",
			hostname: "DESKTOP-PC",
			wantOS:   "Windows",
		},
		{
			name:     "macOS with hostname",
			os:       "darwin",
			hostname: "MacBook-Pro",
			wantOS:   "macOS",
		},
		{
			name:     "Linux with hostname",
			os:       "linux",
			hostname: "ubuntu-server",
			wantOS:   "Linux",
		},
		{
			name:     "No hostname",
			os:       "windows",
			hostname: "",
			wantOS:   "Windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := models.GenerateDeviceName(tt.os, tt.hostname)
			assert.NotEmpty(t, name)
			assert.Contains(t, name, tt.wantOS)
			if tt.hostname != "" {
				assert.Contains(t, name, tt.hostname)
			}
		})
	}
}


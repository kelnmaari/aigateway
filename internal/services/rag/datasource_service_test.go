// Package rag provides data source service tests.
package rag

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// Mock Database
type mockDatabase struct {
	sources   map[string]*models.RAGDataSource
	testConn  error
	syncError error
}

func newMockDatabase() *mockDatabase {
	return &mockDatabase{
		sources: make(map[string]*models.RAGDataSource),
	}
}

func (m *mockDatabase) CreateRAGDataSource(ctx context.Context, source *models.RAGDataSource) error {
	m.sources[source.ID] = source
	return nil
}

func (m *mockDatabase) GetRAGDataSource(ctx context.Context, id string) (*models.RAGDataSource, error) {
	if source, ok := m.sources[id]; ok {
		return source, nil
	}
	return nil, &storage.StorageError{Type: storage.StorageErrorTypeNotFound}
}

func (m *mockDatabase) UpdateRAGDataSource(ctx context.Context, source *models.RAGDataSource) error {
	if _, ok := m.sources[source.ID]; !ok {
		return &storage.StorageError{Type: storage.StorageErrorTypeNotFound}
	}
	m.sources[source.ID] = source
	return nil
}

func (m *mockDatabase) DeleteRAGDataSource(ctx context.Context, id string) error {
	if _, ok := m.sources[id]; !ok {
		return &storage.StorageError{Type: storage.StorageErrorTypeNotFound}
	}
	delete(m.sources, id)
	return nil
}

func (m *mockDatabase) ListRAGDataSources(ctx context.Context, filter *storage.RAGDataSourceFilter) ([]*models.RAGDataSource, int, error) {
	var result []*models.RAGDataSource
	for _, source := range m.sources {
		result = append(result, source)
	}
	return result, len(result), nil
}

// Implement minimal interface methods (not used in tests but required)
func (m *mockDatabase) Connect(ctx context.Context) error                                 { return nil }
func (m *mockDatabase) Close() error                                                      { return nil }
func (m *mockDatabase) Ping(ctx context.Context) error                                    { return nil }
func (m *mockDatabase) Migrate(ctx context.Context) error                                 { return nil }
func (m *mockDatabase) GetMigrationVersion(ctx context.Context) (int, error)              { return 0, nil }
func (m *mockDatabase) BeginTx(ctx context.Context) (storage.Tx, error)                   { return nil, nil }

func TestNewDataSourceService(t *testing.T) {
	db := newMockDatabase()
	encryptionKey := "12345678901234567890123456789012" // 32 bytes
	logger := logrus.New()

	service, err := NewDataSourceService(db, encryptionKey, logger)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if service == nil {
		t.Fatal("Expected non-nil service")
	}
}

func TestNewDataSourceService_InvalidKey(t *testing.T) {
	db := newMockDatabase()
	invalidKey := "short" // < 32 bytes
	logger := logrus.New()

	_, err := NewDataSourceService(db, invalidKey, logger)
	if err == nil {
		t.Error("Expected error for invalid encryption key")
	}
}

func TestDataSourceService_CreateDataSource(t *testing.T) {
	db := newMockDatabase()
	encryptionKey := "12345678901234567890123456789012"
	logger := logrus.New()

	service, _ := NewDataSourceService(db, encryptionKey, logger)
	ctx := context.Background()

	req := CreateDataSourceRequest{
		UserID:      "user1",
		TenantID:    stringPtr("tenant1"),
		Name:        "Test Source",
		Description: "Test description",
		SourceType:  "api",
		Config: map[string]interface{}{
			"url": "https://api.example.com",
		},
		Credentials: map[string]string{
			"api_key": "secret-key-123",
		},
		SyncFrequency: stringPtr("1h"),
		Tags:          []string{"test", "api"},
	}

	source, err := service.CreateDataSource(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if source == nil {
		t.Fatal("Expected non-nil source")
	}

	if source.ID == "" {
		t.Error("Expected non-empty ID")
	}

	if source.Name != req.Name {
		t.Errorf("Expected name %s, got %s", req.Name, source.Name)
	}

	if source.Status != "active" {
		t.Errorf("Expected status 'active', got %s", source.Status)
	}

	if source.CredentialsEncrypted == nil {
		t.Error("Expected credentials to be encrypted")
	}

	// Verify credentials were encrypted
	if *source.CredentialsEncrypted == "secret-key-123" {
		t.Error("Credentials should be encrypted, not plain text")
	}
}

func TestDataSourceService_GetDataSource(t *testing.T) {
	db := newMockDatabase()
	encryptionKey := "12345678901234567890123456789012"
	logger := logrus.New()

	service, _ := NewDataSourceService(db, encryptionKey, logger)
	ctx := context.Background()

	// Create a source first
	createReq := CreateDataSourceRequest{
		UserID:     "user1",
		Name:       "Test Source",
		SourceType: "api",
		Config:     map[string]interface{}{"url": "test"},
	}

	created, _ := service.CreateDataSource(ctx, createReq)

	// Get the source
	source, err := service.GetDataSource(ctx, created.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if source == nil {
		t.Fatal("Expected non-nil source")
	}

	if source.ID != created.ID {
		t.Errorf("Expected ID %s, got %s", created.ID, source.ID)
	}
}

func TestDataSourceService_GetDataSource_NotFound(t *testing.T) {
	db := newMockDatabase()
	encryptionKey := "12345678901234567890123456789012"
	logger := logrus.New()

	service, _ := NewDataSourceService(db, encryptionKey, logger)
	ctx := context.Background()

	_, err := service.GetDataSource(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent source")
	}
}

func TestDataSourceService_UpdateDataSource(t *testing.T) {
	db := newMockDatabase()
	encryptionKey := "12345678901234567890123456789012"
	logger := logrus.New()

	service, _ := NewDataSourceService(db, encryptionKey, logger)
	ctx := context.Background()

	// Create a source
	createReq := CreateDataSourceRequest{
		UserID:     "user1",
		Name:       "Original Name",
		SourceType: "api",
		Config:     map[string]interface{}{},
	}

	created, _ := service.CreateDataSource(ctx, createReq)

	// Update the source
	updateReq := UpdateDataSourceRequest{
		Name:        stringPtr("Updated Name"),
		Description: stringPtr("Updated description"),
		Status:      stringPtr("inactive"),
	}

	updated, err := service.UpdateDataSource(ctx, created.ID, updateReq)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if updated.Name != "Updated Name" {
		t.Errorf("Expected name 'Updated Name', got %s", updated.Name)
	}

	if updated.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got %s", updated.Description)
	}

	if updated.Status != "inactive" {
		t.Errorf("Expected status 'inactive', got %s", updated.Status)
	}
}

func TestDataSourceService_UpdateDataSource_NotFound(t *testing.T) {
	db := newMockDatabase()
	encryptionKey := "12345678901234567890123456789012"
	logger := logrus.New()

	service, _ := NewDataSourceService(db, encryptionKey, logger)
	ctx := context.Background()

	updateReq := UpdateDataSourceRequest{
		Name: stringPtr("New Name"),
	}

	_, err := service.UpdateDataSource(ctx, "non-existent-id", updateReq)
	if err == nil {
		t.Error("Expected error for non-existent source")
	}
}

func TestDataSourceService_DeleteDataSource(t *testing.T) {
	db := newMockDatabase()
	encryptionKey := "12345678901234567890123456789012"
	logger := logrus.New()

	service, _ := NewDataSourceService(db, encryptionKey, logger)
	ctx := context.Background()

	// Create a source
	createReq := CreateDataSourceRequest{
		UserID:     "user1",
		Name:       "Test Source",
		SourceType: "api",
		Config:     map[string]interface{}{},
	}

	created, _ := service.CreateDataSource(ctx, createReq)

	// Delete the source
	err := service.DeleteDataSource(ctx, created.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify it's deleted
	_, err = service.GetDataSource(ctx, created.ID)
	if err == nil {
		t.Error("Expected error when getting deleted source")
	}
}

func TestDataSourceService_ListDataSources(t *testing.T) {
	db := newMockDatabase()
	encryptionKey := "12345678901234567890123456789012"
	logger := logrus.New()

	service, _ := NewDataSourceService(db, encryptionKey, logger)
	ctx := context.Background()

	// Create multiple sources
	for i := 0; i < 5; i++ {
		req := CreateDataSourceRequest{
			UserID:     "user1",
			Name:       "Source " + string(rune(i+'0')),
			SourceType: "api",
			Config:     map[string]interface{}{},
		}
		_, _ = service.CreateDataSource(ctx, req)
	}

	// List sources
	listReq := ListDataSourcesRequest{
		UserID: stringPtr("user1"),
		Limit:  10,
		Offset: 0,
	}

	sources, total, err := service.ListDataSources(ctx, listReq)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if total != 5 {
		t.Errorf("Expected 5 sources, got %d", total)
	}

	if len(sources) != 5 {
		t.Errorf("Expected 5 sources in result, got %d", len(sources))
	}
}

func TestDataSourceService_EncryptDecryptCredentials(t *testing.T) {
	db := newMockDatabase()
	encryptionKey := "12345678901234567890123456789012"
	logger := logrus.New()

	service, _ := NewDataSourceService(db, encryptionKey, logger)

	credentials := map[string]string{
		"api_key":  "secret-key-123",
		"username": "testuser",
		"password": "testpass",
	}

	// Encrypt
	encrypted, err := service.encryptCredentials(credentials)
	if err != nil {
		t.Fatalf("Expected no error encrypting, got %v", err)
	}

	if encrypted == "" {
		t.Error("Expected non-empty encrypted string")
	}

	// Decrypt
	decrypted, err := service.decryptCredentials(encrypted)
	if err != nil {
		t.Fatalf("Expected no error decrypting, got %v", err)
	}

	// Verify
	if decrypted["api_key"] != "secret-key-123" {
		t.Errorf("Expected api_key 'secret-key-123', got %s", decrypted["api_key"])
	}

	if decrypted["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", decrypted["username"])
	}

	if decrypted["password"] != "testpass" {
		t.Errorf("Expected password 'testpass', got %s", decrypted["password"])
	}
}

func TestDataSourceService_TestConnection(t *testing.T) {
	db := newMockDatabase()
	encryptionKey := "12345678901234567890123456789012"
	logger := logrus.New()

	service, _ := NewDataSourceService(db, encryptionKey, logger)
	ctx := context.Background()

	// This will depend on implementation
	// For now, just test that it doesn't panic
	req := TestConnectionRequest{
		SourceType: "api",
		Config: map[string]interface{}{
			"url": "https://api.example.com",
		},
	}

	result := service.TestConnection(ctx, req)
	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestDataSourceService_SyncSource(t *testing.T) {
	db := newMockDatabase()
	encryptionKey := "12345678901234567890123456789012"
	logger := logrus.New()

	service, _ := NewDataSourceService(db, encryptionKey, logger)
	ctx := context.Background()

	// Create a source
	createReq := CreateDataSourceRequest{
		UserID:     "user1",
		Name:       "Test Source",
		SourceType: "api",
		Config:     map[string]interface{}{},
	}

	created, _ := service.CreateDataSource(ctx, createReq)

	// Sync (this is a stub that will enqueue job)
	err := service.SyncSource(ctx, created.ID)
	if err != nil {
		t.Logf("Sync returned error (expected if queue not implemented): %v", err)
	}
}

// Benchmark tests
func BenchmarkDataSourceService_CreateDataSource(b *testing.B) {
	db := newMockDatabase()
	encryptionKey := "12345678901234567890123456789012"
	logger := logrus.New()

	service, _ := NewDataSourceService(db, encryptionKey, logger)
	ctx := context.Background()

	req := CreateDataSourceRequest{
		UserID:     "user1",
		Name:       "Benchmark Source",
		SourceType: "api",
		Config:     map[string]interface{}{"url": "test"},
		Credentials: map[string]string{
			"api_key": "secret",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.CreateDataSource(ctx, req)
	}
}

func BenchmarkDataSourceService_EncryptDecrypt(b *testing.B) {
	db := newMockDatabase()
	encryptionKey := "12345678901234567890123456789012"
	logger := logrus.New()

	service, _ := NewDataSourceService(db, encryptionKey, logger)

	credentials := map[string]string{
		"api_key": "secret-key-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encrypted, _ := service.encryptCredentials(credentials)
		_, _ = service.decryptCredentials(encrypted)
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}


// Package sources implements REST API data source.
package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// APIDataSource implements DataSource for REST APIs
type APIDataSource struct {
	name       string
	baseURL    string
	method     string
	headers    map[string]string
	auth       *APIAuth
	httpClient *http.Client
	logger     *logrus.Logger
	
	// Response parsing
	dataPath       string   // JSON path to documents array
	titleField     string   // Field for title
	contentField   string   // Field for content
	idField        string   // Field for ID
}

// APIAuth authentication config
type APIAuth struct {
	Type   string // "bearer", "basic", "api_key", "oauth2"
	Token  string // Bearer token or API key
	Header string // Header name for API key
	
	// Basic auth
	Username string
	Password string
}

// APIDataSourceConfig конфигурация для API source
type APIDataSourceConfig struct {
	BaseURL      string
	Method       string
	Headers      map[string]string
	Auth         *APIAuth
	DataPath     string // JSON path к массиву документов (e.g., "data.items")
	TitleField   string // Поле для title (default: "title")
	ContentField string // Поле для content (default: "content")
	IDField      string // Поле для ID (default: "id")
	Timeout      int    // Timeout в секундах
}

// NewAPIDataSource creates new API data source
func NewAPIDataSource(name string, config APIDataSourceConfig, logger *logrus.Logger) *APIDataSource {
	if logger == nil {
		logger = logrus.New()
	}

	if config.Method == "" {
		config.Method = "GET"
	}
	if config.TitleField == "" {
		config.TitleField = "title"
	}
	if config.ContentField == "" {
		config.ContentField = "content"
	}
	if config.IDField == "" {
		config.IDField = "id"
	}
	if config.Timeout == 0 {
		config.Timeout = 30
	}

	httpClient := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}

	return &APIDataSource{
		name:         name,
		baseURL:      config.BaseURL,
		method:       config.Method,
		headers:      config.Headers,
		auth:         config.Auth,
		httpClient:   httpClient,
		logger:       logger,
		dataPath:     config.DataPath,
		titleField:   config.TitleField,
		contentField: config.ContentField,
		idField:      config.IDField,
	}
}

// Type возвращает тип источника
func (s *APIDataSource) Type() DataSourceType {
	return SourceTypeAPI
}

// Name возвращает имя источника
func (s *APIDataSource) Name() string {
	return s.name
}

// GetMetadata возвращает метаданные
func (s *APIDataSource) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"type":    string(s.Type()),
		"name":    s.name,
		"base_url": s.baseURL,
		"method":  s.method,
	}
}

// TestConnection проверяет доступность API
func (s *APIDataSource) TestConnection(ctx context.Context) (*ConnectionTestResult, error) {
	startTime := time.Now()

	req, err := http.NewRequestWithContext(ctx, s.method, s.baseURL, nil)
	if err != nil {
		return &ConnectionTestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create request: %v", err),
		}, nil
	}

	// Add headers
	for key, value := range s.headers {
		req.Header.Set(key, value)
	}

	// Add auth
	if s.auth != nil {
		s.applyAuth(req)
	}

	resp, err := s.httpClient.Do(req)
	latency := time.Since(startTime)

	if err != nil {
		return &ConnectionTestResult{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
			Latency: latency,
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &ConnectionTestResult{
			Success: true,
			Message: fmt.Sprintf("Connection successful (HTTP %d)", resp.StatusCode),
			Latency: latency,
			Details: map[string]interface{}{
				"status_code": resp.StatusCode,
			},
		}, nil
	}

	return &ConnectionTestResult{
		Success: false,
		Message: fmt.Sprintf("HTTP error: %d", resp.StatusCode),
		Latency: latency,
		Details: map[string]interface{}{
			"status_code": resp.StatusCode,
		},
	}, nil
}

// Fetch загружает документы из API
func (s *APIDataSource) Fetch(ctx context.Context) ([]Document, error) {
	s.logger.WithField("url", s.baseURL).Info("Fetching documents from API")

	req, err := http.NewRequestWithContext(ctx, s.method, s.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range s.headers {
		req.Header.Set(key, value)
	}

	// Add auth
	if s.auth != nil {
		s.applyAuth(req)
	}

	startTime := time.Now()

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned error: status=%d body=%s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var responseData interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract documents from JSON path
	documentsData, err := s.extractDocuments(responseData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract documents: %w", err)
	}

	documents := make([]Document, 0, len(documentsData))

	for _, docData := range documentsData {
		doc, err := s.parseDocument(docData)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to parse document, skipping")
			continue
		}

		documents = append(documents, doc)
	}

	duration := time.Since(startTime)

	s.logger.WithFields(logrus.Fields{
		"count":       len(documents),
		"duration_ms": duration.Milliseconds(),
	}).Info("Documents fetched successfully")

	return documents, nil
}

// Sync синхронизирует данные
func (s *APIDataSource) Sync(ctx context.Context) (*SyncResult, error) {
	startTime := time.Now()

	documents, err := s.Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch documents: %w", err)
	}

	// В production здесь нужно сравнивать с существующими документами
	// и определять новые/обновленные

	result := &SyncResult{
		TotalDocuments:   len(documents),
		NewDocuments:     len(documents), // Пока все считаем новыми
		UpdatedDocuments: 0,
		Errors:           []string{},
		SyncDuration:     time.Since(startTime),
		LastSyncAt:       time.Now(),
	}

	return result, nil
}

// applyAuth применяет authentication к запросу
func (s *APIDataSource) applyAuth(req *http.Request) {
	switch s.auth.Type {
	case "bearer":
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.auth.Token))
	case "api_key":
		header := s.auth.Header
		if header == "" {
			header = "X-API-Key"
		}
		req.Header.Set(header, s.auth.Token)
	case "basic":
		req.SetBasicAuth(s.auth.Username, s.auth.Password)
	}
}

// extractDocuments извлекает документы из JSON response по dataPath
func (s *APIDataSource) extractDocuments(data interface{}) ([]interface{}, error) {
	if s.dataPath == "" {
		// Если path не указан, ожидаем что data - это массив
		if arr, ok := data.([]interface{}); ok {
			return arr, nil
		}
		return nil, fmt.Errorf("expected array at root")
	}

	// Simple JSON path extraction (e.g., "data.items")
	// В production использовать полноценную библиотеку для JSON path
	current := data
	// Заглушка - пока не реализовано полностью
	// TODO: implement proper JSON path extraction
	
	if arr, ok := current.([]interface{}); ok {
		return arr, nil
	}

	return nil, fmt.Errorf("failed to extract documents array")
}

// parseDocument парсит один document из JSON
func (s *APIDataSource) parseDocument(data interface{}) (Document, error) {
	docMap, ok := data.(map[string]interface{})
	if !ok {
		return Document{}, fmt.Errorf("document must be an object")
	}

	// Extract fields
	id := s.getString(docMap, s.idField)
	title := s.getString(docMap, s.titleField)
	content := s.getString(docMap, s.contentField)

	if id == "" {
		return Document{}, fmt.Errorf("document ID is required")
	}
	if content == "" {
		return Document{}, fmt.Errorf("document content is required")
	}

	doc := Document{
		ID:          id,
		Title:       title,
		Content:     content,
		ContentType: "application/json",
		Metadata:    docMap,
		FetchedAt:   time.Now(),
	}

	return doc, nil
}

// getString извлекает строку из map
func (s *APIDataSource) getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// Ensure APIDataSource implements DataSource interface
var _ DataSource = (*APIDataSource)(nil)


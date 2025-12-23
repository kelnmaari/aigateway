// Package rag provides RAG (Retrieval-Augmented Generation) capabilities for GitLab code review
package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// QdrantConfig holds configuration for Qdrant vector database
type QdrantConfig struct {
	URL           string        `yaml:"url" json:"url"`
	APIKey        string        `yaml:"api_key" json:"api_key"`
	Collection    string        `yaml:"collection" json:"collection"`
	VectorSize    int           `yaml:"vector_size" json:"vector_size"`
	Timeout       time.Duration `yaml:"timeout" json:"timeout"`
	Enabled       bool          `yaml:"enabled" json:"enabled"`
}

// DefaultQdrantConfig returns default Qdrant configuration
func DefaultQdrantConfig() QdrantConfig {
	return QdrantConfig{
		URL:        "http://localhost:6333",
		Collection: "gitlab_code_embeddings",
		VectorSize: 1536, // OpenAI ada-002 default
		Timeout:    30 * time.Second,
		Enabled:    false,
	}
}

// QdrantClient provides interface to Qdrant vector database
type QdrantClient struct {
	config     QdrantConfig
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewQdrantClient creates a new Qdrant client
func NewQdrantClient(config QdrantConfig, logger *logrus.Logger) *QdrantClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.VectorSize == 0 {
		config.VectorSize = 1536
	}
	if config.Collection == "" {
		config.Collection = "gitlab_code_embeddings"
	}

	return &QdrantClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger: logger,
	}
}

// Point represents a vector point in Qdrant
type Point struct {
	ID      string                 `json:"id"`
	Vector  []float64              `json:"vector"` // Use float64 for JSON compatibility
	Payload map[string]interface{} `json:"payload"`
}

// SearchResult represents a search result from Qdrant
type SearchResult struct {
	ID      string                 `json:"id"`
	Score   float32                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

// CodeChunkPayload represents metadata for a code chunk
type CodeChunkPayload struct {
	ProjectID     string `json:"project_id"`
	FilePath      string `json:"file_path"`
	ChunkIndex    int    `json:"chunk_index"`
	Language      string `json:"language"`
	Content       string `json:"content"`
	StartLine     int    `json:"start_line"`
	EndLine       int    `json:"end_line"`
	FunctionName  string `json:"function_name,omitempty"`
	ClassName     string `json:"class_name,omitempty"`
	CommitSHA     string `json:"commit_sha,omitempty"`
	BranchName    string `json:"branch_name,omitempty"`
	LastUpdated   int64  `json:"last_updated"`
}

// EnsureCollection creates the collection if it doesn't exist
func (c *QdrantClient) EnsureCollection(ctx context.Context) error {
	// Check if collection exists
	exists, err := c.collectionExists(ctx)
	if err != nil {
		return fmt.Errorf("check collection exists: %w", err)
	}

	if exists {
		c.logger.WithField("collection", c.config.Collection).Debug("Collection already exists")
		return nil
	}

	// Create collection
	body := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     c.config.VectorSize,
			"distance": "Cosine",
		},
	}

	_, err = c.request(ctx, "PUT", fmt.Sprintf("/collections/%s", c.config.Collection), body)
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}

	c.logger.WithField("collection", c.config.Collection).Info("Created Qdrant collection")
	return nil
}

// collectionExists checks if the collection exists
func (c *QdrantClient) collectionExists(ctx context.Context) (bool, error) {
	resp, err := c.request(ctx, "GET", fmt.Sprintf("/collections/%s", c.config.Collection), nil)
	if err != nil {
		// 404 means collection doesn't exist
		return false, nil
	}
	
	return resp != nil, nil
}

// UpsertPoints inserts or updates points in the collection
func (c *QdrantClient) UpsertPoints(ctx context.Context, points []Point) error {
	if len(points) == 0 {
		return nil
	}

	body := map[string]interface{}{
		"points": points,
	}

	_, err := c.request(ctx, "PUT", fmt.Sprintf("/collections/%s/points", c.config.Collection), body)
	if err != nil {
		return fmt.Errorf("upsert points: %w", err)
	}

	c.logger.WithField("count", len(points)).Debug("Upserted points to Qdrant")
	return nil
}

// Search performs a vector similarity search
func (c *QdrantClient) Search(ctx context.Context, vector []float32, limit int, filter map[string]interface{}) ([]SearchResult, error) {
	// Convert float32 to float64 for better JSON precision
	vector64 := make([]float64, len(vector))
	for i, v := range vector {
		vector64[i] = float64(v)
	}

	body := map[string]interface{}{
		"vector":       vector64, // Use float64 for better JSON compatibility
		"limit":        limit,
		"with_payload": true,
	}

	if filter != nil {
		body["filter"] = filter
	}

	resp, err := c.request(ctx, "POST", fmt.Sprintf("/collections/%s/points/search", c.config.Collection), body)
	if err != nil {
		// If search fails with unnamed vector format, the collection might have been created differently
		// Log the error for debugging
		c.logger.WithError(err).Debug("Search failed, collection may need recreation")
		return nil, fmt.Errorf("search: %w", err)
	}

	var searchResp struct {
		Result []SearchResult `json:"result"`
	}
	if err := json.Unmarshal(resp, &searchResp); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	return searchResp.Result, nil
}

// SearchByProject searches for similar code chunks within a project
func (c *QdrantClient) SearchByProject(ctx context.Context, vector []float32, projectID string, limit int) ([]SearchResult, error) {
	filter := map[string]interface{}{
		"must": []map[string]interface{}{
			{
				"key":   "project_id",
				"match": map[string]interface{}{"value": projectID},
			},
		},
	}

	return c.Search(ctx, vector, limit, filter)
}

// DeleteByProject deletes all points for a project
func (c *QdrantClient) DeleteByProject(ctx context.Context, projectID string) error {
	body := map[string]interface{}{
		"filter": map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key":   "project_id",
					"match": map[string]interface{}{"value": projectID},
				},
			},
		},
	}

	_, err := c.request(ctx, "POST", fmt.Sprintf("/collections/%s/points/delete", c.config.Collection), body)
	if err != nil {
		return fmt.Errorf("delete by project: %w", err)
	}

	c.logger.WithField("project_id", projectID).Info("Deleted project embeddings from Qdrant")
	return nil
}

// DeleteByFile deletes all points for a specific file
func (c *QdrantClient) DeleteByFile(ctx context.Context, projectID, filePath string) error {
	body := map[string]interface{}{
		"filter": map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key":   "project_id",
					"match": map[string]interface{}{"value": projectID},
				},
				{
					"key":   "file_path",
					"match": map[string]interface{}{"value": filePath},
				},
			},
		},
	}

	_, err := c.request(ctx, "POST", fmt.Sprintf("/collections/%s/points/delete", c.config.Collection), body)
	if err != nil {
		return fmt.Errorf("delete by file: %w", err)
	}

	return nil
}

// GetCollectionInfo returns information about the collection
func (c *QdrantClient) GetCollectionInfo(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.request(ctx, "GET", fmt.Sprintf("/collections/%s", c.config.Collection), nil)
	if err != nil {
		return nil, fmt.Errorf("get collection info: %w", err)
	}

	var info map[string]interface{}
	if err := json.Unmarshal(resp, &info); err != nil {
		return nil, fmt.Errorf("parse collection info: %w", err)
	}

	return info, nil
}

// HealthCheck checks if Qdrant is healthy
func (c *QdrantClient) HealthCheck(ctx context.Context) error {
	_, err := c.request(ctx, "GET", "/", nil)
	if err != nil {
		return fmt.Errorf("qdrant health check failed: %w", err)
	}
	return nil
}

// request makes an HTTP request to Qdrant
func (c *QdrantClient) request(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	url := c.config.URL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("api-key", c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("qdrant error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// IsEnabled returns whether Qdrant integration is enabled
func (c *QdrantClient) IsEnabled() bool {
	return c.config.Enabled
}

// Float32ToFloat64 converts float32 slice to float64 slice
// Used for Qdrant API compatibility
func Float32ToFloat64(f32 []float32) []float64 {
	f64 := make([]float64, len(f32))
	for i, v := range f32 {
		f64[i] = float64(v)
	}
	return f64
}


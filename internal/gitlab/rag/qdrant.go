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
	URL        string        `yaml:"url" json:"url"`
	APIKey     string        `yaml:"api_key" json:"api_key"`
	Collection string        `yaml:"collection" json:"collection"`
	VectorSize int           `yaml:"vector_size" json:"vector_size"`
	Timeout    time.Duration `yaml:"timeout" json:"timeout"`
	Enabled    bool          `yaml:"enabled" json:"enabled"`
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
	ID      string         `json:"id"`
	Vector  []float64      `json:"vector"` // Use float64 for JSON compatibility
	Payload map[string]any `json:"payload"`
}

// SearchResult represents a search result from Qdrant
type SearchResult struct {
	ID      string         `json:"id"`
	Score   float32        `json:"score"`
	Payload map[string]any `json:"payload"`
}

// CodeChunkPayload represents metadata for a code chunk
type CodeChunkPayload struct {
	ProjectID    string `json:"project_id"`
	FilePath     string `json:"file_path"`
	ChunkIndex   int    `json:"chunk_index"`
	Language     string `json:"language"`
	Content      string `json:"content"`
	StartLine    int    `json:"start_line"`
	EndLine      int    `json:"end_line"`
	FunctionName string `json:"function_name,omitempty"`
	ClassName    string `json:"class_name,omitempty"`
	CommitSHA    string `json:"commit_sha,omitempty"`
	BranchName   string `json:"branch_name,omitempty"`
	LastUpdated  int64  `json:"last_updated"`
}

// EnsureCollection creates the collection if it doesn't exist (uses default collection)
func (c *QdrantClient) EnsureCollection(ctx context.Context) error {
	return c.EnsureCollectionNamed(ctx, c.config.Collection)
}

// EnsureCollectionNamed creates the specified collection if it doesn't exist
func (c *QdrantClient) EnsureCollectionNamed(ctx context.Context, collectionName string) error {
	// Check if collection exists
	exists, err := c.collectionExistsNamed(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("check collection exists: %w", err)
	}

	if exists {
		c.logger.WithField("collection", collectionName).Debug("Collection already exists")
		return nil
	}

	// Create collection
	body := map[string]any{
		"vectors": map[string]any{
			"size":     c.config.VectorSize,
			"distance": "Cosine",
		},
	}

	_, err = c.request(ctx, "PUT", fmt.Sprintf("/collections/%s", collectionName), body)
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}

	c.logger.WithField("collection", collectionName).Info("Created Qdrant collection")
	return nil
}

// collectionExists checks if the default collection exists
func (c *QdrantClient) collectionExists(ctx context.Context) (bool, error) {
	return c.collectionExistsNamed(ctx, c.config.Collection)
}

// collectionExistsNamed checks if the specified collection exists
func (c *QdrantClient) collectionExistsNamed(ctx context.Context, collectionName string) (bool, error) {
	resp, err := c.request(ctx, "GET", fmt.Sprintf("/collections/%s", collectionName), nil)
	if err != nil {
		// 404 means collection doesn't exist
		return false, nil
	}

	return resp != nil, nil
}

// UpsertPoints inserts or updates points in the default collection
func (c *QdrantClient) UpsertPoints(ctx context.Context, points []Point) error {
	return c.UpsertPointsToCollection(ctx, c.config.Collection, points)
}

// UpsertPointsToCollection inserts or updates points in the specified collection
func (c *QdrantClient) UpsertPointsToCollection(ctx context.Context, collectionName string, points []Point) error {
	if len(points) == 0 {
		return nil
	}

	body := map[string]any{
		"points": points,
	}

	_, err := c.request(ctx, "PUT", fmt.Sprintf("/collections/%s/points", collectionName), body)
	if err != nil {
		return fmt.Errorf("upsert points: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"collection": collectionName,
		"count":      len(points),
	}).Debug("Upserted points to Qdrant")
	return nil
}

// Search performs a vector similarity search in the default collection
func (c *QdrantClient) Search(ctx context.Context, vector []float32, limit int, filter map[string]any) ([]SearchResult, error) {
	return c.SearchInCollection(ctx, c.config.Collection, vector, limit, filter)
}

// SearchInCollection performs a vector similarity search in the specified collection
func (c *QdrantClient) SearchInCollection(ctx context.Context, collectionName string, vector []float32, limit int, filter map[string]any) ([]SearchResult, error) {
	// Validate input vector
	if len(vector) == 0 {
		c.logger.Warn("Qdrant search: empty vector provided, skipping")
		return nil, nil
	}

	c.logger.WithFields(logrus.Fields{
		"collection":       collectionName,
		"input_vector_len": len(vector),
		"limit":            limit,
		"has_filter":       filter != nil,
	}).Debug("Qdrant search: starting")

	// Convert float32 to float64 for better JSON precision
	vector64 := make([]float64, len(vector))
	for i, v := range vector {
		vector64[i] = float64(v)
	}

	c.logger.WithField("output_vector_len", len(vector64)).Debug("Qdrant search: vector converted")

	body := map[string]any{
		"vector":       vector64, // Use float64 for better JSON compatibility
		"limit":        limit,
		"with_payload": true,
	}

	if filter != nil {
		body["filter"] = filter
	}

	resp, err := c.request(ctx, "POST", fmt.Sprintf("/collections/%s/points/search", collectionName), body)
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

// SearchByProject searches for similar code chunks within a project (default collection)
func (c *QdrantClient) SearchByProject(ctx context.Context, vector []float32, projectID string, limit int) ([]SearchResult, error) {
	return c.SearchByProjectInCollection(ctx, c.config.Collection, vector, projectID, limit)
}

// SearchByProjectInCollection searches for similar code chunks within a project in specified collection
func (c *QdrantClient) SearchByProjectInCollection(ctx context.Context, collectionName string, vector []float32, projectID string, limit int) ([]SearchResult, error) {
	filter := map[string]any{
		"must": []map[string]any{
			{
				"key":   "project_id",
				"match": map[string]any{"value": projectID},
			},
		},
	}

	return c.SearchInCollection(ctx, collectionName, vector, limit, filter)
}

// DeleteByProject deletes all points for a project (default collection)
func (c *QdrantClient) DeleteByProject(ctx context.Context, projectID string) error {
	return c.DeleteByProjectInCollection(ctx, c.config.Collection, projectID)
}

// DeleteByProjectInCollection deletes all points for a project in specified collection
func (c *QdrantClient) DeleteByProjectInCollection(ctx context.Context, collectionName, projectID string) error {
	body := map[string]any{
		"filter": map[string]any{
			"must": []map[string]any{
				{
					"key":   "project_id",
					"match": map[string]any{"value": projectID},
				},
			},
		},
	}

	_, err := c.request(ctx, "POST", fmt.Sprintf("/collections/%s/points/delete", collectionName), body)
	if err != nil {
		return fmt.Errorf("delete by project: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"collection": collectionName,
		"project_id": projectID,
	}).Info("Deleted project embeddings from Qdrant")
	return nil
}

// DeleteByFile deletes all points for a specific file (default collection)
func (c *QdrantClient) DeleteByFile(ctx context.Context, projectID, filePath string) error {
	return c.DeleteByFileInCollection(ctx, c.config.Collection, projectID, filePath)
}

// DeleteByFileInCollection deletes all points for a specific file in specified collection
func (c *QdrantClient) DeleteByFileInCollection(ctx context.Context, collectionName, projectID, filePath string) error {
	body := map[string]any{
		"filter": map[string]any{
			"must": []map[string]any{
				{
					"key":   "project_id",
					"match": map[string]any{"value": projectID},
				},
				{
					"key":   "file_path",
					"match": map[string]any{"value": filePath},
				},
			},
		},
	}

	_, err := c.request(ctx, "POST", fmt.Sprintf("/collections/%s/points/delete", collectionName), body)
	if err != nil {
		return fmt.Errorf("delete by file: %w", err)
	}

	return nil
}

// DeleteByProjectBranch deletes all points for a specific project+branch combination
func (c *QdrantClient) DeleteByProjectBranch(ctx context.Context, projectID, branch string) error {
	body := map[string]any{
		"filter": map[string]any{
			"must": []map[string]any{
				{
					"key":   "project_id",
					"match": map[string]any{"value": projectID},
				},
				{
					"key":   "branch_name",
					"match": map[string]any{"value": branch},
				},
			},
		},
	}

	_, err := c.request(ctx, "POST", fmt.Sprintf("/collections/%s/points/delete", c.config.Collection), body)
	if err != nil {
		return fmt.Errorf("delete by project+branch: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"branch":     branch,
	}).Info("Deleted project+branch embeddings from Qdrant")
	return nil
}

// GetCollectionInfo returns information about the default collection
func (c *QdrantClient) GetCollectionInfo(ctx context.Context) (map[string]any, error) {
	return c.GetCollectionInfoNamed(ctx, c.config.Collection)
}

// GetCollectionInfoNamed returns information about a specific collection
func (c *QdrantClient) GetCollectionInfoNamed(ctx context.Context, collectionName string) (map[string]any, error) {
	resp, err := c.request(ctx, "GET", fmt.Sprintf("/collections/%s", collectionName), nil)
	if err != nil {
		return nil, fmt.Errorf("get collection info: %w", err)
	}

	var info map[string]any
	if err := json.Unmarshal(resp, &info); err != nil {
		return nil, fmt.Errorf("parse collection info: %w", err)
	}

	return info, nil
}

// CollectionStats holds collection statistics
type CollectionStats struct {
	PointsCount   int64  `json:"points_count"`
	VectorsCount  int64  `json:"vectors_count"`
	SegmentsCount int    `json:"segments_count"`
	Status        string `json:"status"`
}

// GetCollectionStats returns statistics for a collection
func (c *QdrantClient) GetCollectionStats(ctx context.Context, collectionName string) (*CollectionStats, error) {
	info, err := c.GetCollectionInfoNamed(ctx, collectionName)
	if err != nil {
		return nil, err
	}

	stats := &CollectionStats{}

	// Parse result.points_count, result.vectors_count, etc.
	if result, ok := info["result"].(map[string]any); ok {
		if pc, ok := result["points_count"].(float64); ok {
			stats.PointsCount = int64(pc)
		}
		if vc, ok := result["vectors_count"].(float64); ok {
			stats.VectorsCount = int64(vc)
		}
		if sc, ok := result["segments_count"].(float64); ok {
			stats.SegmentsCount = int(sc)
		}
		if st, ok := result["status"].(string); ok {
			stats.Status = st
		}
	}

	return stats, nil
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
func (c *QdrantClient) request(ctx context.Context, method, path string, body any) ([]byte, error) {
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

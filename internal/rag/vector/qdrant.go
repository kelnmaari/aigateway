// Package vector implements Qdrant vector storage.
package vector

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

// QdrantStore implements VectorStore using Qdrant vector database
type QdrantStore struct {
	client     *http.Client
	baseURL    string
	collection string
	apiKey     string
	dimensions int
	logger     *logrus.Logger
}

// QdrantStoreConfig configuration for Qdrant store
type QdrantStoreConfig struct {
	URL        string
	Collection string
	APIKey     string
	Dimensions int
	Timeout    time.Duration
}

// Qdrant API structures
type qdrantPoint struct {
	ID      string         `json:"id"`
	Vector  []float64      `json:"vector"`
	Payload map[string]any `json:"payload"`
}

type qdrantUpsertRequest struct {
	Points []qdrantPoint `json:"points"`
}

type qdrantSearchRequest struct {
	Vector         []float64     `json:"vector"`
	Limit          int           `json:"limit"`
	WithPayload    bool          `json:"with_payload"`
	WithVector     bool          `json:"with_vector"`
	Filter         *qdrantFilter `json:"filter,omitempty"`
	ScoreThreshold float64       `json:"score_threshold,omitempty"`
}

type qdrantFilter struct {
	Must []qdrantCondition `json:"must,omitempty"`
}

type qdrantCondition struct {
	Key   string `json:"key"`
	Match any    `json:"match"`
}

type qdrantSearchResult struct {
	ID      string         `json:"id"`
	Version int            `json:"version"`
	Score   float64        `json:"score"`
	Payload map[string]any `json:"payload"`
	Vector  []float64      `json:"vector,omitempty"`
}

type qdrantSearchResponse struct {
	Result []qdrantSearchResult `json:"result"`
	Status string               `json:"status"`
	Time   float64              `json:"time"`
}

type qdrantCollectionInfo struct {
	Result struct {
		Status        string `json:"status"`
		PointsCount   int64  `json:"points_count"` // Primary counter for indexed points
		SegmentsCount int    `json:"segments_count"`
		// VectorsCount can be int64 (legacy) or object (for named vectors in Qdrant 1.x+)
		VectorsCount json.RawMessage `json:"vectors_count"`
		Config       struct {
			Params struct {
				// Vectors can be a single vector config or named vectors map
				Vectors json.RawMessage `json:"vectors"`
			} `json:"params"`
		} `json:"config"`
	} `json:"result"`
	Status string  `json:"status"`
	Time   float64 `json:"time"`
}

type qdrantCreateCollection struct {
	Vectors struct {
		Size     int    `json:"size"`
		Distance string `json:"distance"`
	} `json:"vectors"`
}

// NewQdrantStore creates new Qdrant store
func NewQdrantStore(config QdrantStoreConfig, logger *logrus.Logger) (*QdrantStore, error) {
	if logger == nil {
		logger = logrus.New()
	}

	if config.URL == "" {
		config.URL = "http://localhost:6333"
	}
	if config.Collection == "" {
		config.Collection = "aigateway_rag"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.Dimensions == 0 {
		config.Dimensions = 1024
	}

	logger.WithFields(logrus.Fields{
		"url":        config.URL,
		"collection": config.Collection,
		"dimensions": config.Dimensions,
	}).Info("Initializing Qdrant vector store")

	store := &QdrantStore{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		baseURL:    config.URL,
		collection: config.Collection,
		apiKey:     config.APIKey,
		dimensions: config.Dimensions,
		logger:     logger,
	}

	// Check connection and create collection if needed
	if err := store.ensureCollection(); err != nil {
		logger.WithError(err).Error("Failed to ensure Qdrant collection")
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"url":        config.URL,
		"collection": config.Collection,
	}).Info("Qdrant vector store initialized successfully")

	return store, nil
}

// Name returns the provider name
func (s *QdrantStore) Name() string {
	return "qdrant"
}

// ensureCollection creates collection if it doesn't exist
func (s *QdrantStore) ensureCollection() error {
	// Check if collection exists
	url := fmt.Sprintf("%s/collections/%s", s.baseURL, s.collection)

	s.logger.WithField("url", url).Debug("Checking if Qdrant collection exists")

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.WithError(err).Error("Failed to connect to Qdrant")
		return fmt.Errorf("failed to connect to Qdrant: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		s.logger.WithField("collection", s.collection).Debug("Qdrant collection already exists")
		return nil
	}

	// Create collection
	s.logger.WithFields(logrus.Fields{
		"collection": s.collection,
		"dimensions": s.dimensions,
	}).Info("Creating Qdrant collection")

	createReq := qdrantCreateCollection{}
	createReq.Vectors.Size = s.dimensions
	createReq.Vectors.Distance = "Cosine"

	body, _ := json.Marshal(createReq)

	req, err = http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	s.setHeaders(req)

	resp, err = s.client.Do(req)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create Qdrant collection")
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		s.logger.WithFields(logrus.Fields{
			"status":   resp.StatusCode,
			"response": string(respBody),
		}).Error("Failed to create Qdrant collection")
		return fmt.Errorf("failed to create collection: %s", string(respBody))
	}

	s.logger.WithField("collection", s.collection).Info("Qdrant collection created successfully")
	return nil
}

// setHeaders sets common headers for Qdrant requests
func (s *QdrantStore) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("api-key", s.apiKey)
	}
}

// Insert adds a single vector
func (s *QdrantStore) Insert(ctx context.Context, doc VectorDocument) error {
	s.logger.WithFields(logrus.Fields{
		"id":          doc.ID,
		"vector_dims": len(doc.Vector),
	}).Debug("Inserting vector into Qdrant")

	return s.InsertBatch(ctx, []VectorDocument{doc})
}

// InsertBatch adds multiple vectors
func (s *QdrantStore) InsertBatch(ctx context.Context, docs []VectorDocument) error {
	if len(docs) == 0 {
		return nil
	}

	startTime := time.Now()

	s.logger.WithField("count", len(docs)).Debug("Inserting batch of vectors into Qdrant")

	points := make([]qdrantPoint, len(docs))
	for i, doc := range docs {
		payload := doc.Metadata
		if payload == nil {
			payload = make(map[string]any)
		}
		payload["text"] = doc.Text
		payload["created_at"] = doc.CreatedAt.Format(time.RFC3339)

		points[i] = qdrantPoint{
			ID:      doc.ID,
			Vector:  doc.Vector,
			Payload: payload,
		}
	}

	upsertReq := qdrantUpsertRequest{Points: points}
	body, err := json.Marshal(upsertReq)
	if err != nil {
		s.logger.WithError(err).Error("Failed to marshal upsert request")
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points?wait=true", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.WithError(err).Error("Failed to upsert points to Qdrant")
		return fmt.Errorf("failed to upsert points: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		s.logger.WithFields(logrus.Fields{
			"status":   resp.StatusCode,
			"response": string(respBody),
		}).Error("Qdrant upsert failed")
		return fmt.Errorf("upsert failed: %s", string(respBody))
	}

	duration := time.Since(startTime)
	s.logger.WithFields(logrus.Fields{
		"count":    len(docs),
		"duration": duration.String(),
	}).Info("Batch vectors inserted into Qdrant successfully")

	return nil
}

// Search performs vector similarity search
func (s *QdrantStore) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	if len(req.Query) == 0 {
		return nil, fmt.Errorf("query vector is empty")
	}

	if req.TopK <= 0 {
		req.TopK = 10
	}

	startTime := time.Now()

	s.logger.WithFields(logrus.Fields{
		"top_k":       req.TopK,
		"min_score":   req.MinScore,
		"vector_dims": len(req.Query),
	}).Debug("Executing Qdrant vector search")

	searchReq := qdrantSearchRequest{
		Vector:         req.Query,
		Limit:          req.TopK,
		WithPayload:    true,
		WithVector:     req.IncludeVectors,
		ScoreThreshold: req.MinScore,
	}

	// Add filters
	if len(req.Filters) > 0 {
		conditions := make([]qdrantCondition, 0, len(req.Filters))
		for key, value := range req.Filters {
			conditions = append(conditions, qdrantCondition{
				Key:   key,
				Match: map[string]any{"value": value},
			})
		}
		searchReq.Filter = &qdrantFilter{Must: conditions}
	}

	body, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", s.baseURL, s.collection)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	s.setHeaders(httpReq)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		s.logger.WithError(err).Error("Qdrant search request failed")
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		s.logger.WithFields(logrus.Fields{
			"status":   resp.StatusCode,
			"response": string(respBody),
		}).Error("Qdrant search failed")
		return nil, fmt.Errorf("search failed: %s", string(respBody))
	}

	var searchResp qdrantSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		s.logger.WithError(err).Error("Failed to decode Qdrant search response")
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to VectorDocument
	documents := make([]VectorDocument, len(searchResp.Result))
	for i, result := range searchResp.Result {
		doc := VectorDocument{
			ID:       result.ID,
			Score:    result.Score,
			Metadata: result.Payload,
			Vector:   result.Vector,
		}

		// Extract text and created_at from payload
		if text, ok := result.Payload["text"].(string); ok {
			doc.Text = text
		}
		if createdAt, ok := result.Payload["created_at"].(string); ok {
			doc.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		}

		documents[i] = doc
	}

	searchTime := time.Since(startTime)

	s.logger.WithFields(logrus.Fields{
		"results":     len(documents),
		"search_time": searchTime.String(),
		"qdrant_time": fmt.Sprintf("%.3fms", searchResp.Time*1000),
		"top_k":       req.TopK,
	}).Debug("Qdrant vector search completed")

	return &SearchResponse{
		Documents:  documents,
		TotalFound: len(documents),
		SearchTime: int(searchTime.Milliseconds()),
	}, nil
}

// Delete removes a vector by ID
func (s *QdrantStore) Delete(ctx context.Context, id string) error {
	s.logger.WithField("id", id).Debug("Deleting vector from Qdrant")

	body, _ := json.Marshal(map[string]any{
		"points": []string{id},
	})

	url := fmt.Sprintf("%s/collections/%s/points/delete?wait=true", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.WithError(err).Error("Failed to delete point from Qdrant")
		return fmt.Errorf("failed to delete point: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		s.logger.WithFields(logrus.Fields{
			"status":   resp.StatusCode,
			"response": string(respBody),
		}).Error("Qdrant delete failed")
		return fmt.Errorf("delete failed: %s", string(respBody))
	}

	s.logger.WithField("id", id).Debug("Vector deleted from Qdrant successfully")
	return nil
}

// DeleteByMetadata deletes vectors by metadata filters
func (s *QdrantStore) DeleteByMetadata(ctx context.Context, filters map[string]any) (int, error) {
	if len(filters) == 0 {
		return 0, fmt.Errorf("filters are required")
	}

	s.logger.WithField("filters", filters).Debug("Deleting vectors by metadata from Qdrant")

	conditions := make([]qdrantCondition, 0, len(filters))
	for key, value := range filters {
		conditions = append(conditions, qdrantCondition{
			Key:   key,
			Match: map[string]any{"value": value},
		})
	}

	body, _ := json.Marshal(map[string]any{
		"filter": map[string]any{
			"must": conditions,
		},
	})

	url := fmt.Sprintf("%s/collections/%s/points/delete?wait=true", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.WithError(err).Error("Failed to delete points by metadata from Qdrant")
		return 0, fmt.Errorf("failed to delete by metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("delete failed: %s", string(respBody))
	}

	s.logger.WithField("filters", filters).Info("Vectors deleted by metadata from Qdrant")
	// Qdrant doesn't return count of deleted points in this API
	return 0, nil
}

// Update updates a vector
func (s *QdrantStore) Update(ctx context.Context, doc VectorDocument) error {
	s.logger.WithField("id", doc.ID).Debug("Updating vector in Qdrant")
	// In Qdrant, upsert is idempotent - same as insert
	return s.Insert(ctx, doc)
}

// GetByID retrieves a vector by ID
func (s *QdrantStore) GetByID(ctx context.Context, id string) (*VectorDocument, error) {
	s.logger.WithField("id", id).Debug("Getting vector by ID from Qdrant")

	url := fmt.Sprintf("%s/collections/%s/points/%s", s.baseURL, s.collection, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get point from Qdrant")
		return nil, fmt.Errorf("failed to get point: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("vector not found")
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get failed: %s", string(respBody))
	}

	var result struct {
		Result qdrantSearchResult `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	doc := &VectorDocument{
		ID:       result.Result.ID,
		Vector:   result.Result.Vector,
		Metadata: result.Result.Payload,
	}

	if text, ok := result.Result.Payload["text"].(string); ok {
		doc.Text = text
	}
	if createdAt, ok := result.Result.Payload["created_at"].(string); ok {
		doc.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	}

	return doc, nil
}

// CreateIndex creates an index (Qdrant manages indexes automatically)
func (s *QdrantStore) CreateIndex(ctx context.Context, indexType string, params map[string]any) error {
	s.logger.WithFields(logrus.Fields{
		"index_type": indexType,
		"params":     params,
	}).Info("Qdrant manages indexes automatically, skipping manual index creation")
	// Qdrant creates indexes automatically
	return nil
}

// GetIndexStats returns index statistics
func (s *QdrantStore) GetIndexStats(ctx context.Context) (*IndexStats, error) {
	s.logger.Debug("Getting Qdrant collection stats")

	url := fmt.Sprintf("%s/collections/%s", s.baseURL, s.collection)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get collection info from Qdrant")
		return nil, fmt.Errorf("failed to get collection info: %w", err)
	}
	defer resp.Body.Close()

	var info qdrantCollectionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse vectors_count - can be int64 (legacy) or object (named vectors)
	var vectorsCount int64
	if len(info.Result.VectorsCount) > 0 {
		// Try as int64 first
		if err := json.Unmarshal(info.Result.VectorsCount, &vectorsCount); err != nil {
			// Try as object (named vectors: {"": count, "name1": count})
			var vectorsMap map[string]int64
			if err := json.Unmarshal(info.Result.VectorsCount, &vectorsMap); err == nil {
				// Sum all vector counts
				for _, count := range vectorsMap {
					vectorsCount += count
				}
			}
		}
	}
	// Fallback to points_count if vectors_count is 0 or missing
	if vectorsCount == 0 {
		vectorsCount = info.Result.PointsCount
	}

	// Parse vectors config - can be single config or named vectors map
	var dimensions int
	if len(info.Result.Config.Params.Vectors) > 0 {
		// Try as single vector config first
		var singleConfig struct {
			Size     int    `json:"size"`
			Distance string `json:"distance"`
		}
		if err := json.Unmarshal(info.Result.Config.Params.Vectors, &singleConfig); err == nil && singleConfig.Size > 0 {
			dimensions = singleConfig.Size
		} else {
			// Try as named vectors map
			var vectorsMap map[string]struct {
				Size     int    `json:"size"`
				Distance string `json:"distance"`
			}
			if err := json.Unmarshal(info.Result.Config.Params.Vectors, &vectorsMap); err == nil {
				// Get first vector config
				for _, v := range vectorsMap {
					dimensions = v.Size
					break
				}
			}
		}
	}
	// Fallback to configured dimensions
	if dimensions == 0 {
		dimensions = s.dimensions
	}

	stats := &IndexStats{
		TotalVectors: vectorsCount,
		Dimensions:   dimensions,
		IndexType:    "hnsw", // Qdrant uses HNSW by default
	}

	s.logger.WithFields(logrus.Fields{
		"vectors_count": stats.TotalVectors,
		"points_count":  info.Result.PointsCount,
		"dimensions":    stats.Dimensions,
	}).Debug("Qdrant collection stats retrieved")

	return stats, nil
}

// HealthCheck checks Qdrant availability
func (s *QdrantStore) HealthCheck(ctx context.Context) error {
	s.logger.Debug("Performing Qdrant health check")

	url := fmt.Sprintf("%s/healthz", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.WithError(err).Error("Qdrant health check failed")
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.WithField("status", resp.StatusCode).Error("Qdrant health check returned non-OK status")
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	s.logger.Debug("Qdrant health check passed")
	return nil
}

// Close closes the Qdrant store (no-op for HTTP client)
func (s *QdrantStore) Close() error {
	s.logger.Debug("Closing Qdrant store")
	return nil
}

// ScrollRequest for scrolling through all points in a collection
type ScrollRequest struct {
	Collection string         // Collection name (empty = use default)
	Filters    map[string]any // Metadata filters
	Limit      int            // Points per page
	Offset     *string        // Offset for pagination (nil = start from beginning)
	WithVector bool           // Include vectors in response
}

// ScrollResponse contains scroll results
type ScrollResponse struct {
	Documents  []VectorDocument
	NextOffset *string // Nil if no more results
}

// Scroll iterates through all points in collection
func (s *QdrantStore) Scroll(ctx context.Context, req ScrollRequest) (*ScrollResponse, error) {
	collection := req.Collection
	if collection == "" {
		collection = s.collection
	}

	if req.Limit <= 0 {
		req.Limit = 100
	}

	s.logger.WithFields(logrus.Fields{
		"collection": collection,
		"limit":      req.Limit,
		"has_offset": req.Offset != nil,
	}).Debug("Scrolling Qdrant collection")

	// Build scroll request body
	scrollReq := map[string]any{
		"limit":        req.Limit,
		"with_payload": true,
		"with_vector":  req.WithVector,
	}

	if req.Offset != nil {
		scrollReq["offset"] = *req.Offset
	}

	// Add filters if any
	if len(req.Filters) > 0 {
		conditions := make([]qdrantCondition, 0, len(req.Filters))
		for key, value := range req.Filters {
			conditions = append(conditions, qdrantCondition{
				Key:   key,
				Match: map[string]any{"value": value},
			})
		}
		scrollReq["filter"] = map[string]any{
			"must": conditions,
		}
	}

	body, err := json.Marshal(scrollReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scroll request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/scroll", s.baseURL, collection)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	s.setHeaders(httpReq)

	resp, err := s.client.Do(httpReq)
	if err != nil {
		s.logger.WithError(err).Error("Qdrant scroll request failed")
		return nil, fmt.Errorf("scroll request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("scroll failed: %s", string(respBody))
	}

	var scrollResp struct {
		Result struct {
			Points     []qdrantSearchResult `json:"points"`
			NextPageID any                  `json:"next_page_offset"`
		} `json:"result"`
		Status string  `json:"status"`
		Time   float64 `json:"time"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&scrollResp); err != nil {
		return nil, fmt.Errorf("failed to decode scroll response: %w", err)
	}

	// Convert to VectorDocuments
	documents := make([]VectorDocument, len(scrollResp.Result.Points))
	for i, point := range scrollResp.Result.Points {
		doc := VectorDocument{
			ID:       point.ID,
			Score:    point.Score,
			Metadata: point.Payload,
			Vector:   point.Vector,
		}

		if text, ok := point.Payload["text"].(string); ok {
			doc.Text = text
		}
		if createdAt, ok := point.Payload["created_at"].(string); ok {
			doc.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		}

		documents[i] = doc
	}

	// Handle next offset
	var nextOffset *string
	if scrollResp.Result.NextPageID != nil {
		switch v := scrollResp.Result.NextPageID.(type) {
		case string:
			nextOffset = &v
		case float64:
			s := fmt.Sprintf("%.0f", v)
			nextOffset = &s
		}
	}

	s.logger.WithFields(logrus.Fields{
		"documents": len(documents),
		"has_more":  nextOffset != nil,
	}).Debug("Qdrant scroll completed")

	return &ScrollResponse{
		Documents:  documents,
		NextOffset: nextOffset,
	}, nil
}

// ScrollAll scrolls through ALL points in collection using callback
func (s *QdrantStore) ScrollAll(ctx context.Context, collection string, filters map[string]any, callback func([]VectorDocument) error) error {
	var offset *string
	batchNum := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := s.Scroll(ctx, ScrollRequest{
			Collection: collection,
			Filters:    filters,
			Limit:      100,
			Offset:     offset,
			WithVector: false,
		})
		if err != nil {
			return fmt.Errorf("scroll batch %d: %w", batchNum, err)
		}

		if len(resp.Documents) > 0 {
			if err := callback(resp.Documents); err != nil {
				return fmt.Errorf("callback batch %d: %w", batchNum, err)
			}
		}

		if resp.NextOffset == nil {
			break
		}
		offset = resp.NextOffset
		batchNum++
	}

	return nil
}

// Ensure QdrantStore implements VectorStore interface
var _ VectorStore = (*QdrantStore)(nil)

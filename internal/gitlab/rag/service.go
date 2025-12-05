// Package rag provides RAG (Retrieval-Augmented Generation) capabilities for GitLab code review
package rag

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// EmbeddingProvider interface for generating embeddings
type EmbeddingProvider interface {
	// GenerateEmbedding creates a vector embedding for the given text
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	
	// GenerateEmbeddings creates vector embeddings for multiple texts
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}

// RAGService provides RAG capabilities for code review
type RAGService struct {
	qdrant    *QdrantClient
	embedder  EmbeddingProvider
	logger    *logrus.Logger
	enabled   bool
}

// RAGConfig holds RAG service configuration
type RAGConfig struct {
	Qdrant  QdrantConfig `yaml:"qdrant" json:"qdrant"`
	Enabled bool         `yaml:"enabled" json:"enabled"`
}

// NewRAGService creates a new RAG service
func NewRAGService(config RAGConfig, embedder EmbeddingProvider, logger *logrus.Logger) *RAGService {
	var qdrantClient *QdrantClient
	if config.Qdrant.Enabled {
		qdrantClient = NewQdrantClient(config.Qdrant, logger)
	}

	return &RAGService{
		qdrant:   qdrantClient,
		embedder: embedder,
		logger:   logger,
		enabled:  config.Enabled && config.Qdrant.Enabled,
	}
}

// Initialize sets up the RAG service (creates collections, etc.)
func (s *RAGService) Initialize(ctx context.Context) error {
	if !s.enabled || s.qdrant == nil {
		return nil
	}

	if err := s.qdrant.EnsureCollection(ctx); err != nil {
		return fmt.Errorf("ensure qdrant collection: %w", err)
	}

	return nil
}

// IndexCodeChunk indexes a code chunk for later retrieval
func (s *RAGService) IndexCodeChunk(ctx context.Context, chunk CodeChunk) error {
	if !s.enabled {
		return nil
	}

	// Generate embedding for the chunk
	embedding, err := s.embedder.GenerateEmbedding(ctx, chunk.Content)
	if err != nil {
		return fmt.Errorf("generate embedding: %w", err)
	}

	// Create point ID from project + file + chunk index
	pointID := generatePointID(chunk.ProjectID, chunk.FilePath, chunk.ChunkIndex)

	point := Point{
		ID:     pointID,
		Vector: embedding,
		Payload: map[string]interface{}{
			"project_id":    chunk.ProjectID,
			"file_path":     chunk.FilePath,
			"chunk_index":   chunk.ChunkIndex,
			"language":      chunk.Language,
			"content":       chunk.Content,
			"start_line":    chunk.StartLine,
			"end_line":      chunk.EndLine,
			"function_name": chunk.FunctionName,
			"class_name":    chunk.ClassName,
			"commit_sha":    chunk.CommitSHA,
			"branch_name":   chunk.BranchName,
			"last_updated":  chunk.LastUpdated,
		},
	}

	if err := s.qdrant.UpsertPoints(ctx, []Point{point}); err != nil {
		return fmt.Errorf("upsert point: %w", err)
	}

	return nil
}

// IndexCodeChunks indexes multiple code chunks in batch
func (s *RAGService) IndexCodeChunks(ctx context.Context, chunks []CodeChunk) error {
	if !s.enabled || len(chunks) == 0 {
		return nil
	}

	// Generate embeddings in batch
	contents := make([]string, len(chunks))
	for i, chunk := range chunks {
		contents[i] = chunk.Content
	}

	embeddings, err := s.embedder.GenerateEmbeddings(ctx, contents)
	if err != nil {
		return fmt.Errorf("generate embeddings: %w", err)
	}

	// Create points
	points := make([]Point, len(chunks))
	for i, chunk := range chunks {
		pointID := generatePointID(chunk.ProjectID, chunk.FilePath, chunk.ChunkIndex)
		points[i] = Point{
			ID:     pointID,
			Vector: embeddings[i],
			Payload: map[string]interface{}{
				"project_id":    chunk.ProjectID,
				"file_path":     chunk.FilePath,
				"chunk_index":   chunk.ChunkIndex,
				"language":      chunk.Language,
				"content":       chunk.Content,
				"start_line":    chunk.StartLine,
				"end_line":      chunk.EndLine,
				"function_name": chunk.FunctionName,
				"class_name":    chunk.ClassName,
				"commit_sha":    chunk.CommitSHA,
				"branch_name":   chunk.BranchName,
				"last_updated":  chunk.LastUpdated,
			},
		}
	}

	if err := s.qdrant.UpsertPoints(ctx, points); err != nil {
		return fmt.Errorf("upsert points: %w", err)
	}

	s.logger.WithField("count", len(chunks)).Info("Indexed code chunks")
	return nil
}

// FindSimilarCode finds code chunks similar to the given query
func (s *RAGService) FindSimilarCode(ctx context.Context, query string, projectID string, limit int) ([]CodeChunk, error) {
	if !s.enabled {
		return nil, nil
	}

	// Generate embedding for query
	embedding, err := s.embedder.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("generate query embedding: %w", err)
	}

	// Search in Qdrant
	results, err := s.qdrant.SearchByProject(ctx, embedding, projectID, limit)
	if err != nil {
		return nil, fmt.Errorf("search qdrant: %w", err)
	}

	// Convert results to CodeChunks
	chunks := make([]CodeChunk, 0, len(results))
	for _, result := range results {
		chunk := CodeChunk{
			ProjectID:    getString(result.Payload, "project_id"),
			FilePath:     getString(result.Payload, "file_path"),
			ChunkIndex:   getInt(result.Payload, "chunk_index"),
			Language:     getString(result.Payload, "language"),
			Content:      getString(result.Payload, "content"),
			StartLine:    getInt(result.Payload, "start_line"),
			EndLine:      getInt(result.Payload, "end_line"),
			FunctionName: getString(result.Payload, "function_name"),
			ClassName:    getString(result.Payload, "class_name"),
			CommitSHA:    getString(result.Payload, "commit_sha"),
			BranchName:   getString(result.Payload, "branch_name"),
			Score:        result.Score,
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// FindRelatedCode finds code related to the given file path
func (s *RAGService) FindRelatedCode(ctx context.Context, projectID, filePath string, content string, limit int) ([]CodeChunk, error) {
	if !s.enabled {
		return nil, nil
	}

	// Use the file content as query
	return s.FindSimilarCode(ctx, content, projectID, limit)
}

// DeleteProjectIndex removes all indexed code for a project
func (s *RAGService) DeleteProjectIndex(ctx context.Context, projectID string) error {
	if !s.enabled {
		return nil
	}

	return s.qdrant.DeleteByProject(ctx, projectID)
}

// DeleteFileIndex removes indexed code for a specific file
func (s *RAGService) DeleteFileIndex(ctx context.Context, projectID, filePath string) error {
	if !s.enabled {
		return nil
	}

	return s.qdrant.DeleteByFile(ctx, projectID, filePath)
}

// GetContextForReview retrieves relevant code context for MR review
func (s *RAGService) GetContextForReview(ctx context.Context, projectID string, changedFiles []ChangedFile, limit int) ([]CodeChunk, error) {
	if !s.enabled || len(changedFiles) == 0 {
		return nil, nil
	}

	allChunks := make([]CodeChunk, 0)
	seenIDs := make(map[string]bool)

	for _, file := range changedFiles {
		// Find similar code for the changed content
		chunks, err := s.FindSimilarCode(ctx, file.Diff, projectID, limit/len(changedFiles)+1)
		if err != nil {
			s.logger.WithError(err).WithField("file", file.Path).Warn("Failed to find similar code")
			continue
		}

		for _, chunk := range chunks {
			// Skip chunks from the same file being reviewed
			if chunk.FilePath == file.Path {
				continue
			}

			// Deduplicate
			id := generatePointID(chunk.ProjectID, chunk.FilePath, chunk.ChunkIndex)
			if seenIDs[id] {
				continue
			}
			seenIDs[id] = true

			allChunks = append(allChunks, chunk)
		}
	}

	// Limit total results
	if len(allChunks) > limit {
		allChunks = allChunks[:limit]
	}

	return allChunks, nil
}

// IsEnabled returns whether RAG is enabled
func (s *RAGService) IsEnabled() bool {
	return s.enabled
}

// HealthCheck checks if RAG service is healthy
func (s *RAGService) HealthCheck(ctx context.Context) error {
	if !s.enabled {
		return nil
	}
	return s.qdrant.HealthCheck(ctx)
}

// CodeChunk represents a chunk of code for indexing
type CodeChunk struct {
	ProjectID    string  `json:"project_id"`
	FilePath     string  `json:"file_path"`
	ChunkIndex   int     `json:"chunk_index"`
	Language     string  `json:"language"`
	Content      string  `json:"content"`
	StartLine    int     `json:"start_line"`
	EndLine      int     `json:"end_line"`
	FunctionName string  `json:"function_name,omitempty"`
	ClassName    string  `json:"class_name,omitempty"`
	CommitSHA    string  `json:"commit_sha,omitempty"`
	BranchName   string  `json:"branch_name,omitempty"`
	LastUpdated  int64   `json:"last_updated,omitempty"`
	Score        float32 `json:"score,omitempty"` // Similarity score (for search results)
}

// ChangedFile represents a file changed in an MR
type ChangedFile struct {
	Path string `json:"path"`
	Diff string `json:"diff"`
}

// Helper functions

func generatePointID(projectID, filePath string, chunkIndex int) string {
	data := fmt.Sprintf("%s:%s:%d", projectID, filePath, chunkIndex)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16])
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return 0
}

// FormatContextForPrompt formats code context for inclusion in LLM prompt
func FormatContextForPrompt(chunks []CodeChunk) string {
	if len(chunks) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Related Code Context\n\n")
	sb.WriteString("The following code snippets from the codebase may be relevant:\n\n")

	for i, chunk := range chunks {
		sb.WriteString(fmt.Sprintf("### Context %d: `%s` (lines %d-%d)\n", i+1, chunk.FilePath, chunk.StartLine, chunk.EndLine))
		if chunk.FunctionName != "" {
			sb.WriteString(fmt.Sprintf("Function: `%s`\n", chunk.FunctionName))
		}
		if chunk.ClassName != "" {
			sb.WriteString(fmt.Sprintf("Class: `%s`\n", chunk.ClassName))
		}
		sb.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", chunk.Language, chunk.Content))
	}

	return sb.String()
}


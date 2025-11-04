// Package orchestrator provides main RAG orchestration logic.
package orchestrator

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/rag/embeddings"
	"aigateway/internal/rag/vector"
	"aigateway/internal/storage"
)

// RAGOrchestrator координирует все компоненты RAG системы
type RAGOrchestrator struct {
	db           storage.Database
	embedder     embeddings.Embedder
	vectorStore  vector.VectorStore
	logger       *logrus.Logger
	
	// Configuration
	defaultTopK     int
	defaultMinScore float64
	maxContextSize  int // Maximum tokens in assembled context
}

// NewRAGOrchestrator creates new orchestrator
func NewRAGOrchestrator(
	db storage.Database,
	embedder embeddings.Embedder,
	vectorStore vector.VectorStore,
	logger *logrus.Logger,
) *RAGOrchestrator {
	if logger == nil {
		logger = logrus.New()
	}

	return &RAGOrchestrator{
		db:              db,
		embedder:        embedder,
		vectorStore:     vectorStore,
		logger:          logger,
		defaultTopK:     10,
		defaultMinScore: 0.7,
		maxContextSize:  4000, // Tokens
	}
}

// RAGRequest запрос для RAG
type RAGRequest struct {
	Query      string   // User query
	SourceIDs  []string // Фильтр по источникам (optional)
	TopK       int      // Количество chunks для retrieval
	MinScore   float64  // Минимальный similarity score
	UserID     string   // ID пользователя для логирования
	ConvID     string   // ID разговора
	Rerank     bool     // Применять reranking
}

// RAGResponse результат RAG
type RAGResponse struct {
	Context        string                 // Собранный context для LLM
	SourceChunks   []RetrievedChunk       // Использованные chunks
	TotalChunks    int                    // Всего найденных chunks
	SearchTime     time.Duration          // Время поиска
	ContextTokens  int                    // Токенов в context
	Metadata       map[string]interface{} // Дополнительные метаданные
}

// RetrievedChunk информация об одном chunk
type RetrievedChunk struct {
	ChunkID    string                 // ID chunk
	SourceID   string                 // ID источника
	DocumentID string                 // ID документа
	Text       string                 // Текст chunk
	Score      float64                // Similarity score
	Metadata   map[string]interface{} // Метаданные
}

// Query выполняет RAG запрос
func (o *RAGOrchestrator) Query(ctx context.Context, req RAGRequest) (*RAGResponse, error) {
	startTime := time.Now()

	// Validate request
	if req.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	if req.TopK <= 0 {
		req.TopK = o.defaultTopK
	}
	if req.MinScore <= 0 {
		req.MinScore = o.defaultMinScore
	}

	o.logger.WithFields(logrus.Fields{
		"query":      req.Query,
		"top_k":      req.TopK,
		"source_ids": req.SourceIDs,
	}).Info("Processing RAG query")

	// ⚠️ Fallback: If embedder or vectorStore not configured, use simple query without vector search
	if o.embedder == nil || o.vectorStore == nil {
		o.logger.Warn("Embedder or VectorStore not configured, using simple query without similarity search")
		return o.simpleQuery(ctx, req, startTime)
	}

	// 1. Generate query embedding
	queryEmbedding, err := o.embedder.Embed(ctx, embeddings.EmbeddingRequest{
		Text: req.Query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// 2. Vector search
	searchReq := vector.SearchRequest{
		Query:    queryEmbedding.Vector,
		TopK:     req.TopK * 2, // Получаем больше для reranking
		MinScore: req.MinScore,
		Filters:  make(map[string]interface{}),
	}

	// Add source filters
	if len(req.SourceIDs) > 0 {
		// В production здесь нужна поддержка IN clause в vector store
		// Пока фильтруем после получения результатов
	}

	searchResults, err := o.vectorStore.Search(ctx, searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	// 3. Filter by source IDs (если указаны)
	chunks := o.filterBySourceIDs(searchResults.Documents, req.SourceIDs)

	// 4. Reranking (если включен)
	if req.Rerank {
		chunks = o.rerankChunks(req.Query, chunks)
	}

	// Ограничиваем до TopK
	if len(chunks) > req.TopK {
		chunks = chunks[:req.TopK]
	}

	// 5. Convert to RetrievedChunk
	retrievedChunks := o.convertToRetrievedChunks(chunks)

	// 6. Assemble context
	context, contextTokens := o.assembleContext(retrievedChunks, req.Query)

	// 7. Log query
	if req.UserID != "" {
		o.logQuery(ctx, req, retrievedChunks, time.Since(startTime), contextTokens)
	}

	response := &RAGResponse{
		Context:       context,
		SourceChunks:  retrievedChunks,
		TotalChunks:   len(retrievedChunks),
		SearchTime:    time.Since(startTime),
		ContextTokens: contextTokens,
		Metadata: map[string]interface{}{
			"query_embedding_dims": len(queryEmbedding.Vector),
			"rerank_enabled":       req.Rerank,
		},
	}

	o.logger.WithFields(logrus.Fields{
		"chunks":         len(retrievedChunks),
		"context_tokens": contextTokens,
		"search_time_ms": response.SearchTime.Milliseconds(),
	}).Info("RAG query completed")

	return response, nil
}

// filterBySourceIDs фильтрует chunks по source IDs
func (o *RAGOrchestrator) filterBySourceIDs(docs []vector.VectorDocument, sourceIDs []string) []vector.VectorDocument {
	if len(sourceIDs) == 0 {
		return docs
	}

	sourceSet := make(map[string]bool)
	for _, id := range sourceIDs {
		sourceSet[id] = true
	}

	filtered := []vector.VectorDocument{}
	for _, doc := range docs {
		if sourceID, ok := doc.Metadata["source_id"].(string); ok {
			if sourceSet[sourceID] {
				filtered = append(filtered, doc)
			}
		}
	}

	return filtered
}

// rerankChunks переранжирует chunks используя более сложные метрики
func (o *RAGOrchestrator) rerankChunks(query string, docs []vector.VectorDocument) []vector.VectorDocument {
	// Simple reranking based on:
	// - Similarity score (already calculated)
	// - Query keyword overlap
	// - Document metadata (e.g., freshness)

	queryLower := strings.ToLower(query)
	queryTokens := strings.Fields(queryLower)

	type scoredDoc struct {
		doc   vector.VectorDocument
		score float64
	}

	scored := make([]scoredDoc, len(docs))

	for i, doc := range docs {
		// Base score from vector similarity
		score := doc.Score

		// Boost for query keyword overlap
		textLower := strings.ToLower(doc.Text)
		matchCount := 0
		for _, token := range queryTokens {
			if strings.Contains(textLower, token) {
				matchCount++
			}
		}
		keywordBoost := float64(matchCount) / float64(len(queryTokens)) * 0.1
		score += keywordBoost

		// Можно добавить boost за freshness, source quality, etc.

		scored[i] = scoredDoc{doc: doc, score: score}
	}

	// Sort by reranked score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Convert back to docs
	result := make([]vector.VectorDocument, len(scored))
	for i, s := range scored {
		result[i] = s.doc
		result[i].Score = s.score // Update with reranked score
	}

	return result
}

// convertToRetrievedChunks конвертирует VectorDocument в RetrievedChunk
func (o *RAGOrchestrator) convertToRetrievedChunks(docs []vector.VectorDocument) []RetrievedChunk {
	chunks := make([]RetrievedChunk, len(docs))

	for i, doc := range docs {
		chunk := RetrievedChunk{
			ChunkID:  doc.ID,
			Text:     doc.Text,
			Score:    doc.Score,
			Metadata: doc.Metadata,
		}

		// Extract source_id и document_id from metadata
		if sourceID, ok := doc.Metadata["source_id"].(string); ok {
			chunk.SourceID = sourceID
		}
		if docID, ok := doc.Metadata["document_id"].(string); ok {
			chunk.DocumentID = docID
		}

		chunks[i] = chunk
	}

	return chunks
}

// assembleContext собирает context из chunks для LLM
func (o *RAGOrchestrator) assembleContext(chunks []RetrievedChunk, query string) (string, int) {
	if len(chunks) == 0 {
		return "", 0
	}

	var builder strings.Builder

	// Header
	builder.WriteString("# Relevant Context\n\n")
	builder.WriteString(fmt.Sprintf("User Query: %s\n\n", query))
	builder.WriteString("## Retrieved Information:\n\n")

	totalTokens := o.estimateTokens(builder.String())

	// Add chunks до maxContextSize
	for i, chunk := range chunks {
		chunkText := fmt.Sprintf("### Source %d (Score: %.3f)\n%s\n\n", i+1, chunk.Score, chunk.Text)
		chunkTokens := o.estimateTokens(chunkText)

		if totalTokens+chunkTokens > o.maxContextSize {
			// Достигли лимита токенов
			break
		}

		builder.WriteString(chunkText)
		totalTokens += chunkTokens
	}

	// Footer
	footer := "\n---\nPlease answer the user's query using the information above. If the context doesn't contain relevant information, say so.\n"
	builder.WriteString(footer)
	totalTokens += o.estimateTokens(footer)

	return builder.String(), totalTokens
}

// estimateTokens оценивает количество токенов (1 token ≈ 4 chars)
func (o *RAGOrchestrator) estimateTokens(text string) int {
	return len(text) / 4
}

// logQuery логирует RAG запрос в БД
func (o *RAGOrchestrator) logQuery(
	ctx context.Context,
	req RAGRequest,
	chunks []RetrievedChunk,
	searchTime time.Duration,
	contextTokens int,
) {
	// Extract source IDs
	sourceIDs := make([]string, 0, len(chunks))
	seen := make(map[string]bool)
	for _, chunk := range chunks {
		if chunk.SourceID != "" && !seen[chunk.SourceID] {
			sourceIDs = append(sourceIDs, chunk.SourceID)
			seen[chunk.SourceID] = true
		}
	}

	// TODO: Implement CreateRAGQueryLog
	// log := &models.RAGQueryLog{
	// 	UserID:            &req.UserID,
	// 	ConversationID:    &req.ConvID,
	// 	QueryText:         req.Query,
	// 	SourceIDs:         sourceIDs,
	// 	ChunksRetrieved:   len(chunks),
	// 	SearchTimeMs:      int(searchTime.Milliseconds()),
	// 	TotalTokensUsed:   &contextTokens,
	// 	CreatedAt:         time.Now(),
	// }
	//
	// if err := o.db.CreateRAGQueryLog(ctx, log); err != nil {
	// 	o.logger.WithError(err).Warn("Failed to log RAG query")
	// }
}

// simpleQuery выполняет простой поиск chunks БЕЗ embeddings (fallback)
func (o *RAGOrchestrator) simpleQuery(ctx context.Context, req RAGRequest, startTime time.Time) (*RAGResponse, error) {
	// Получаем все chunks из указанных sources
	var allChunks []*models.RAGChunk
	
	for _, sourceID := range req.SourceIDs {
		chunks, err := o.db.ListRAGChunksBySource(ctx, sourceID, req.TopK, 0)
		if err != nil {
			o.logger.WithError(err).WithField("source_id", sourceID).Warn("Failed to get chunks from source")
			continue
		}
		allChunks = append(allChunks, chunks...)
	}

	o.logger.WithField("total_chunks", len(allChunks)).Info("Retrieved chunks from sources (simple query)")

	// Если chunks не найдены
	if len(allChunks) == 0 {
		return &RAGResponse{
			Context:       "",
			SourceChunks:  []RetrievedChunk{},
			TotalChunks:   0,
			SearchTime:    time.Since(startTime),
			ContextTokens: 0,
			Metadata: map[string]interface{}{
				"method": "simple_query_no_embeddings",
			},
		}, nil
	}

	// Ограничиваем до TopK
	if len(allChunks) > req.TopK {
		allChunks = allChunks[:req.TopK]
	}

	// Собираем context
	var contextParts []string
	retrievedChunks := make([]RetrievedChunk, 0, len(allChunks))
	totalTokens := 0

	for i, chunk := range allChunks {
		retrievedChunks = append(retrievedChunks, RetrievedChunk{
			ChunkID:    chunk.ID,
			SourceID:   chunk.SourceID,
			DocumentID: chunk.DocumentID,
			Text:       chunk.ChunkText,
			Score:      1.0, // Фиктивный score (нет similarity search)
			Metadata:   chunk.Metadata,
		})

		contextParts = append(contextParts, fmt.Sprintf("[Chunk %d]\n%s", i+1, chunk.ChunkText))
		totalTokens += chunk.ChunkTokens
	}

	context := strings.Join(contextParts, "\n\n---\n\n")
	searchTime := time.Since(startTime)

	o.logger.WithFields(logrus.Fields{
		"chunks_used":    len(retrievedChunks),
		"context_tokens": totalTokens,
		"search_time_ms": searchTime.Milliseconds(),
	}).Info("Simple RAG query completed")

	return &RAGResponse{
		Context:       context,
		SourceChunks:  retrievedChunks,
		TotalChunks:   len(retrievedChunks),
		SearchTime:    searchTime,
		ContextTokens: totalTokens,
		Metadata: map[string]interface{}{
			"method": "simple_query_no_embeddings",
			"note":   "This is a simplified RAG without vector search. For production use, implement embeddings.",
		},
	}, nil
}


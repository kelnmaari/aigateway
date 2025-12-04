// Package handlers provides HTTP handlers for RAG statistics
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/rag/vector"
)

// RAGStatsHandler handles RAG statistics endpoints
type RAGStatsHandler struct {
	vectorStore vector.VectorStore
	logger      *logrus.Logger
}

// NewRAGStatsHandler creates a new RAGStatsHandler
func NewRAGStatsHandler(vectorStore vector.VectorStore, logger *logrus.Logger) *RAGStatsHandler {
	return &RAGStatsHandler{
		vectorStore: vectorStore,
		logger:      logger,
	}
}

// RAGStatsResponse represents RAG system statistics
type RAGStatsResponse struct {
	Enabled      bool            `json:"enabled"`
	Provider     string          `json:"provider"`
	VectorStats  *VectorStats    `json:"vector_stats,omitempty"`
	HealthStatus string          `json:"health_status"`
	LoadedAt     time.Time       `json:"loaded_at"`
}

// VectorStats represents vector store statistics
type VectorStats struct {
	TotalVectors  int64  `json:"total_vectors"`
	Dimensions    int    `json:"dimensions"`
	IndexType     string `json:"index_type"`
	CollectionURL string `json:"collection_url,omitempty"`
}

// GetRAGStats returns RAG system statistics
// GET /api/admin/rag/stats
func (h *RAGStatsHandler) GetRAGStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	h.logger.Debug("Loading RAG statistics")

	response := RAGStatsResponse{
		Enabled:      h.vectorStore != nil,
		LoadedAt:     time.Now(),
		HealthStatus: "disabled",
	}

	if h.vectorStore == nil {
		h.logger.Debug("Vector store not configured")
		c.JSON(http.StatusOK, response)
		return
	}

	response.Provider = h.vectorStore.Name()

	// Health check
	if err := h.vectorStore.HealthCheck(ctx); err != nil {
		h.logger.WithError(err).Warn("Vector store health check failed")
		response.HealthStatus = "unhealthy"
		c.JSON(http.StatusOK, response)
		return
	}

	response.HealthStatus = "healthy"

	// Get index stats
	stats, err := h.vectorStore.GetIndexStats(ctx)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to get vector store stats")
		c.JSON(http.StatusOK, response)
		return
	}

	response.VectorStats = &VectorStats{
		TotalVectors: stats.TotalVectors,
		Dimensions:   stats.Dimensions,
		IndexType:    stats.IndexType,
	}

	h.logger.WithFields(logrus.Fields{
		"provider":      response.Provider,
		"total_vectors": stats.TotalVectors,
		"dimensions":    stats.Dimensions,
		"health":        response.HealthStatus,
	}).Debug("RAG statistics loaded")

	c.JSON(http.StatusOK, response)
}


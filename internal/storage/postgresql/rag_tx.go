// Package postgresql provides RAG transaction methods for PostgreSQL.
package postgresql

import (
	"context"
	"time"

	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// RAG Data Sources (v1.13.0+)

func (tx *postgresqlTx) CreateRAGDataSource(ctx context.Context, source *models.RAGDataSource) error {
	return tx.db.CreateRAGDataSource(ctx, source)
}

func (tx *postgresqlTx) GetRAGDataSource(ctx context.Context, id string) (*models.RAGDataSource, error) {
	return tx.db.GetRAGDataSource(ctx, id)
}

func (tx *postgresqlTx) UpdateRAGDataSource(ctx context.Context, source *models.RAGDataSource) error {
	return tx.db.UpdateRAGDataSource(ctx, source)
}

func (tx *postgresqlTx) DeleteRAGDataSource(ctx context.Context, id string) error {
	return tx.db.DeleteRAGDataSource(ctx, id)
}

func (tx *postgresqlTx) ListRAGDataSources(ctx context.Context, filter *storage.RAGDataSourceFilter) ([]*models.RAGDataSource, int, error) {
	return tx.db.ListRAGDataSources(ctx, filter)
}

// RAG Documents (v1.13.0+)

func (tx *postgresqlTx) CreateRAGDocument(ctx context.Context, doc *models.RAGDocument) error {
	return tx.db.CreateRAGDocument(ctx, doc)
}

func (tx *postgresqlTx) GetRAGDocument(ctx context.Context, id string) (*models.RAGDocument, error) {
	return tx.db.GetRAGDocument(ctx, id)
}

func (tx *postgresqlTx) UpdateRAGDocument(ctx context.Context, doc *models.RAGDocument) error {
	return tx.db.UpdateRAGDocument(ctx, doc)
}

func (tx *postgresqlTx) DeleteRAGDocument(ctx context.Context, id string) error {
	return tx.db.DeleteRAGDocument(ctx, id)
}

func (tx *postgresqlTx) ListRAGDocuments(ctx context.Context, filter *storage.RAGDocumentFilter) ([]*models.RAGDocument, error) {
	return tx.db.ListRAGDocuments(ctx, filter)
}

// RAG Chunks (v1.13.0+)

func (tx *postgresqlTx) CreateRAGChunk(ctx context.Context, chunk *models.RAGChunk) error {
	return tx.db.CreateRAGChunk(ctx, chunk)
}

func (tx *postgresqlTx) GetRAGChunk(ctx context.Context, id string) (*models.RAGChunk, error) {
	return tx.db.GetRAGChunk(ctx, id)
}

func (tx *postgresqlTx) ListRAGChunksByDocument(ctx context.Context, documentID string) ([]*models.RAGChunk, error) {
	return tx.db.ListRAGChunksByDocument(ctx, documentID)
}

func (tx *postgresqlTx) ListRAGChunksBySource(ctx context.Context, sourceID string, limit, offset int) ([]*models.RAGChunk, error) {
	return tx.db.ListRAGChunksBySource(ctx, sourceID, limit, offset)
}

func (tx *postgresqlTx) DeleteRAGChunksByDocument(ctx context.Context, documentID string) error {
	return tx.db.DeleteRAGChunksByDocument(ctx, documentID)
}

// RAG Jobs Queue (v1.13.0+)

func (tx *postgresqlTx) CreateRAGJob(ctx context.Context, job *models.RAGJob) error {
	return tx.db.CreateRAGJob(ctx, job)
}

func (tx *postgresqlTx) GetRAGJob(ctx context.Context, id string) (*models.RAGJob, error) {
	return tx.db.GetRAGJob(ctx, id)
}

func (tx *postgresqlTx) UpdateRAGJob(ctx context.Context, job *models.RAGJob) error {
	return tx.db.UpdateRAGJob(ctx, job)
}

func (tx *postgresqlTx) GetNextPendingRAGJob(ctx context.Context) (*models.RAGJob, error) {
	return tx.db.GetNextPendingRAGJob(ctx)
}

func (tx *postgresqlTx) CountRAGJobsByStatus(ctx context.Context, status string) (int, error) {
	return tx.db.CountRAGJobsByStatus(ctx, status)
}

func (tx *postgresqlTx) DeleteOldRAGJobs(ctx context.Context, cutoffTime time.Time, statuses []string) (int, error) {
	return tx.db.DeleteOldRAGJobs(ctx, cutoffTime, statuses)
}

func (tx *postgresqlTx) UnlockExpiredRAGJobs(ctx context.Context, now time.Time) (int, error) {
	return tx.db.UnlockExpiredRAGJobs(ctx, now)
}

// RAG Query Logs (v1.13.0+)

func (tx *postgresqlTx) CreateRAGQueryLog(ctx context.Context, log *models.RAGQueryLog) error {
	return tx.db.CreateRAGQueryLog(ctx, log)
}

func (tx *postgresqlTx) GetRAGQueryLog(ctx context.Context, id int64) (*models.RAGQueryLog, error) {
	return tx.db.GetRAGQueryLog(ctx, id)
}

func (tx *postgresqlTx) ListRAGQueryLogsByUser(ctx context.Context, userID string, limit, offset int) ([]*models.RAGQueryLog, error) {
	return tx.db.ListRAGQueryLogsByUser(ctx, userID, limit, offset)
}


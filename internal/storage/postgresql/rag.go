// Package postgresql implements RAG database operations stubs for PostgreSQL.
package postgresql

import (
	"context"
	"fmt"
	"time"

	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// RAG Data Sources (v1.13.0+) - Stubs for PostgreSQL

func (p *PostgreSQLDB) CreateRAGDataSource(ctx context.Context, source *models.RAGDataSource) error {
	return fmt.Errorf("RAG data sources not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) GetRAGDataSource(ctx context.Context, id string) (*models.RAGDataSource, error) {
	return nil, fmt.Errorf("RAG data sources not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) UpdateRAGDataSource(ctx context.Context, source *models.RAGDataSource) error {
	return fmt.Errorf("RAG data sources not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) DeleteRAGDataSource(ctx context.Context, id string) error {
	return fmt.Errorf("RAG data sources not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) ListRAGDataSources(ctx context.Context, filter *storage.RAGDataSourceFilter) ([]*models.RAGDataSource, int, error) {
	return nil, 0, fmt.Errorf("RAG data sources not yet implemented for PostgreSQL")
}

// RAG Documents (v1.13.0+)

func (p *PostgreSQLDB) CreateRAGDocument(ctx context.Context, doc *models.RAGDocument) error {
	return fmt.Errorf("RAG documents not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) GetRAGDocument(ctx context.Context, id string) (*models.RAGDocument, error) {
	return nil, fmt.Errorf("RAG documents not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) UpdateRAGDocument(ctx context.Context, doc *models.RAGDocument) error {
	return fmt.Errorf("RAG documents not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) DeleteRAGDocument(ctx context.Context, id string) error {
	return fmt.Errorf("RAG documents not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) ListRAGDocuments(ctx context.Context, filter *storage.RAGDocumentFilter) ([]*models.RAGDocument, error) {
	return nil, fmt.Errorf("RAG documents not yet implemented for PostgreSQL")
}

// RAG Chunks (v1.13.0+)

func (p *PostgreSQLDB) CreateRAGChunk(ctx context.Context, chunk *models.RAGChunk) error {
	return fmt.Errorf("RAG chunks not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) GetRAGChunk(ctx context.Context, id string) (*models.RAGChunk, error) {
	return nil, fmt.Errorf("RAG chunks not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) ListRAGChunksByDocument(ctx context.Context, documentID string) ([]*models.RAGChunk, error) {
	return nil, fmt.Errorf("RAG chunks not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) ListRAGChunksBySource(ctx context.Context, sourceID string, limit, offset int) ([]*models.RAGChunk, error) {
	return nil, fmt.Errorf("RAG chunks not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) DeleteRAGChunksByDocument(ctx context.Context, documentID string) error {
	return fmt.Errorf("RAG chunks not yet implemented for PostgreSQL")
}

// RAG Jobs Queue (v1.13.0+)

func (p *PostgreSQLDB) CreateRAGJob(ctx context.Context, job *models.RAGJob) error {
	return fmt.Errorf("RAG jobs not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) GetRAGJob(ctx context.Context, id string) (*models.RAGJob, error) {
	return nil, fmt.Errorf("RAG jobs not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) UpdateRAGJob(ctx context.Context, job *models.RAGJob) error {
	return fmt.Errorf("RAG jobs not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) GetNextPendingRAGJob(ctx context.Context) (*models.RAGJob, error) {
	return nil, fmt.Errorf("RAG jobs not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) CountRAGJobsByStatus(ctx context.Context, status string) (int, error) {
	return 0, fmt.Errorf("RAG jobs not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) DeleteOldRAGJobs(ctx context.Context, cutoffTime time.Time, statuses []string) (int, error) {
	return 0, fmt.Errorf("RAG jobs not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) UnlockExpiredRAGJobs(ctx context.Context, now time.Time) (int, error) {
	return 0, fmt.Errorf("RAG jobs not yet implemented for PostgreSQL")
}

// RAG Query Logs (v1.13.0+)

func (p *PostgreSQLDB) CreateRAGQueryLog(ctx context.Context, log *models.RAGQueryLog) error {
	return fmt.Errorf("RAG query logs not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) GetRAGQueryLog(ctx context.Context, id int64) (*models.RAGQueryLog, error) {
	return nil, fmt.Errorf("RAG query logs not yet implemented for PostgreSQL")
}

func (p *PostgreSQLDB) ListRAGQueryLogsByUser(ctx context.Context, userID string, limit, offset int) ([]*models.RAGQueryLog, error) {
	return nil, fmt.Errorf("RAG query logs not yet implemented for PostgreSQL")
}


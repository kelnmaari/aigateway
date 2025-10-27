// Package storage defines RAG data storage interfaces.
package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
	"aigateway/internal/models"
)

// RAGDataSourceRepository интерфейс для работы с RAG data sources
type RAGDataSourceRepository interface {
	// Create создает новый источник данных
	CreateDataSource(ctx context.Context, source *models.RAGDataSource) error
	
	// Get получает источник по ID
	GetDataSourceByID(ctx context.Context, id uuid.UUID) (*models.RAGDataSource, error)
	
	// List возвращает список источников с фильтрацией
	ListDataSources(ctx context.Context, filter DataSourceFilter) ([]models.RAGDataSource, int, error)
	
	// Update обновляет источник
	UpdateDataSource(ctx context.Context, source *models.RAGDataSource) error
	
	// Delete удаляет источник
	DeleteDataSource(ctx context.Context, id uuid.UUID) error
	
	// UpdateStatus обновляет статус источника
	UpdateSourceStatus(ctx context.Context, id uuid.UUID, status models.SourceStatus, err string) error
	
	// UpdateSyncInfo обновляет информацию о синхронизации
	UpdateSyncInfo(ctx context.Context, id uuid.UUID, status models.SyncStatus, chunkCount int) error
	
	// UpdateStatistics обновляет статистику источника
	UpdateStatistics(ctx context.Context, id uuid.UUID, totalChunks int, totalTokens int64) error
}

// DataSourceFilter фильтр для списка источников
type DataSourceFilter struct {
	UserID     *uuid.UUID
	TenantID   *uuid.UUID
	SourceType *models.SourceType
	Status     *models.SourceStatus
	Tags       []string
	Limit      int
	Offset     int
}

// RAGDocumentRepository интерфейс для работы с RAG documents
type RAGDocumentRepository interface {
	// Create создает новый документ
	CreateDocument(ctx context.Context, doc *models.RAGDocument) error
	
	// Get получает документ по ID
	GetDocumentByID(ctx context.Context, id uuid.UUID) (*models.RAGDocument, error)
	
	// ListBySource возвращает список документов источника
	ListDocumentsBySource(ctx context.Context, sourceID uuid.UUID, limit, offset int) ([]models.RAGDocument, int, error)
	
	// Update обновляет документ
	UpdateDocument(ctx context.Context, doc *models.RAGDocument) error
	
	// UpdateStatus обновляет статус обработки
	UpdateDocumentStatus(ctx context.Context, id uuid.UUID, status models.DocumentStatus, err string) error
	
	// Delete удаляет документ
	DeleteDocument(ctx context.Context, id uuid.UUID) error
}

// RAGChunkRepository интерфейс для работы с RAG chunks
type RAGChunkRepository interface {
	// CreateBatch создает batch чанков
	CreateChunksBatch(ctx context.Context, chunks []models.RAGChunk) error
	
	// GetByDocument возвращает чанки документа
	GetChunksByDocument(ctx context.Context, documentID uuid.UUID) ([]models.RAGChunk, error)
	
	// GetBySource возвращает чанки источника
	GetChunksBySource(ctx context.Context, sourceID uuid.UUID, limit, offset int) ([]models.RAGChunk, int, error)
	
	// DeleteByDocument удаляет чанки документа
	DeleteChunksByDocument(ctx context.Context, documentID uuid.UUID) error
	
	// DeleteBySource удаляет чанки источника
	DeleteChunksBySource(ctx context.Context, sourceID uuid.UUID) error
}

// RAGJobRepository интерфейс для работы с RAG jobs queue
type RAGJobRepository interface {
	// Create создает новую задачу
	CreateJob(ctx context.Context, job *models.RAGJob) error
	
	// GetNext получает следующую pending задачу (с visibility timeout)
	GetNextJob(ctx context.Context, jobTypes []string) (*models.RAGJob, error)
	
	// GetByID получает задачу по ID
	GetJobByID(ctx context.Context, id int64) (*models.RAGJob, error)
	
	// UpdateStatus обновляет статус задачи
	UpdateJobStatus(ctx context.Context, id int64, status models.JobStatus, result, err string) error
	
	// MarkStarted помечает задачу как начатую
	MarkJobStarted(ctx context.Context, id int64) error
	
	// MarkCompleted помечает задачу как завершенную
	MarkJobCompleted(ctx context.Context, id int64, result string) error
	
	// MarkFailed помечает задачу как провалившуюся
	MarkJobFailed(ctx context.Context, id int64, err string) error
	
	// IncrementAttempts увеличивает счетчик попыток
	IncrementJobAttempts(ctx context.Context, id int64) error
	
	// CleanupCompleted удаляет завершенные задачи старше заданного времени
	CleanupCompletedJobs(ctx context.Context, olderThanHours int) error
}

// RAGQueryLogRepository интерфейс для работы с RAG query logs
type RAGQueryLogRepository interface {
	// Create создает лог запроса
	CreateQueryLog(ctx context.Context, log *models.RAGQueryLog) error
	
	// GetByUser возвращает логи пользователя
	GetQueryLogsByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.RAGQueryLog, int, error)
	
	// GetStats возвращает статистику по логам
	GetQueryStats(ctx context.Context, userID *uuid.UUID, from, to *time.Time) (*models.RAGQueryStats, error)
}



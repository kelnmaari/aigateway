// Package sources provides external data source implementations.
package sources

import (
	"context"
	"time"
)

// DataSourceType тип источника данных
type DataSourceType string

const (
	SourceTypeFile     DataSourceType = "file"
	SourceTypeAPI      DataSourceType = "api"
	SourceTypeDatabase DataSourceType = "database"
	SourceTypeWeb      DataSourceType = "web"
)

// Document представляет документ из источника
type Document struct {
	ID          string         // Уникальный ID
	Title       string         // Название
	Content     string         // Содержимое
	ContentType string         // MIME type
	Metadata    map[string]any // Метаданные
	FetchedAt   time.Time      // Время загрузки
}

// SyncResult результат синхронизации
type SyncResult struct {
	TotalDocuments   int           // Всего документов
	NewDocuments     int           // Новых документов
	UpdatedDocuments int           // Обновленных документов
	Errors           []string      // Ошибки при синхронизации
	SyncDuration     time.Duration // Длительность синхронизации
	LastSyncAt       time.Time     // Время последней синхронизации
}

// ConnectionTestResult результат проверки подключения
type ConnectionTestResult struct {
	Success bool           // Успешно ли подключение
	Message string         // Сообщение (ошибка или статус)
	Latency time.Duration  // Задержка подключения
	Details map[string]any // Дополнительные детали
}

// DataSource interface для внешних источников данных
type DataSource interface {
	// Fetch загружает документы из источника
	Fetch(ctx context.Context) ([]Document, error)

	// Sync синхронизирует данные (добавляет новые/обновляет существующие)
	Sync(ctx context.Context) (*SyncResult, error)

	// TestConnection проверяет доступность источника
	TestConnection(ctx context.Context) (*ConnectionTestResult, error)

	// GetMetadata возвращает метаданные источника
	GetMetadata() map[string]any

	// Type возвращает тип источника
	Type() DataSourceType

	// Name возвращает имя источника
	Name() string
}

// DataSourceConfig конфигурация источника данных
type DataSourceConfig struct {
	Type   DataSourceType
	Name   string
	Config map[string]any
}

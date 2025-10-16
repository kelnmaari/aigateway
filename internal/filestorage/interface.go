// Package filestorage provides universal file storage abstraction (Version 1.10.0+)
// Поддерживает multiple backends (Local FS, S3/MinIO) с единым интерфейсом
package filestorage

import (
	"context"
	"io"
	"time"
)

// StorageBackend определяет интерфейс для хранения файлов
// Используется в v1.10.0 для простого хранения и в v1.13.0 для RAG
type StorageBackend interface {
	// Store сохраняет файл и возвращает storage path
	Store(ctx context.Context, file io.Reader, opts StoreOptions) (string, error)

	// Retrieve получает содержимое файла по path
	Retrieve(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete удаляет файл
	Delete(ctx context.Context, path string) error

	// Exists проверяет существование файла
	Exists(ctx context.Context, path string) (bool, error)

	// GetURL возвращает публичный или подписанный URL (для S3)
	// Для local storage возвращает относительный путь
	GetURL(ctx context.Context, path string, expiry time.Duration) (string, error)

	// GetSize возвращает размер файла в байтах
	GetSize(ctx context.Context, path string) (int64, error)

	// Type возвращает тип backend ("local", "s3")
	Type() string
}

// StoreOptions опции для сохранения файла
type StoreOptions struct {
	UserID   string // ID пользователя (для организации файлов)
	TenantID string // ID tenant (для multi-tenancy)
	Filename string // Оригинальное имя файла
	MimeType string // MIME type файла

	// Metadata дополнительные метаданные (для будущего использования)
	Metadata map[string]interface{}

	// Public флаг публичного доступа (для S3 ACL)
	Public bool

	// Checksum SHA-256 для проверки целостности (optional)
	Checksum string
}

// FileInfo информация о сохраненном файле
type FileInfo struct {
	Path      string                 // Storage path
	Size      int64                  // Размер в байтах
	MimeType  string                 // MIME type
	Checksum  string                 // SHA-256 checksum
	Metadata  map[string]interface{} // Метаданные
	CreatedAt time.Time              // Время создания
}

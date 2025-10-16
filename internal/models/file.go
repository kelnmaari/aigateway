package models

import (
	"time"
)

// File представляет файл в системе (Version 1.10.0+)
type File struct {
	ID       string  `json:"id" db:"id"`
	UserID   string  `json:"user_id" db:"user_id"`
	TenantID *string `json:"tenant_id,omitempty" db:"tenant_id"`

	// File information
	Filename         string  `json:"filename" db:"filename"`
	OriginalFilename string  `json:"original_filename" db:"original_filename"`
	MimeType         string  `json:"mime_type" db:"mime_type"`
	SizeBytes        int64   `json:"size_bytes" db:"size_bytes"`
	ChecksumSHA256   *string `json:"checksum_sha256,omitempty" db:"checksum_sha256"`

	// Storage
	StorageBackend string  `json:"storage_backend" db:"storage_backend"` // 'local', 's3'
	StoragePath    string  `json:"storage_path" db:"storage_path"`
	StorageBucket  *string `json:"storage_bucket,omitempty" db:"storage_bucket"`

	// Extracted content
	ExtractedText    *string `json:"extracted_text,omitempty" db:"extracted_text"`
	ExtractionStatus string  `json:"extraction_status" db:"extraction_status"` // 'pending', 'completed', 'failed'
	ExtractionError  *string `json:"extraction_error,omitempty" db:"extraction_error"`

	// Metadata (JSON)
	Metadata *string `json:"metadata,omitempty" db:"metadata"` // Serialized JSON

	// Document info
	PageCount *int    `json:"page_count,omitempty" db:"page_count"`
	WordCount *int    `json:"word_count,omitempty" db:"word_count"`
	Language  *string `json:"language,omitempty" db:"language"`

	// Access control
	IsPublic   bool    `json:"is_public" db:"is_public"`
	SharedWith *string `json:"shared_with,omitempty" db:"shared_with"` // JSON array

	// Usage tracking
	DownloadCount  int        `json:"download_count" db:"download_count"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty" db:"last_accessed_at"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"` // Soft delete
}

// FileMetadata метаданные файла (будет сериализована в JSON)
type FileMetadata struct {
	PageCount  int                    `json:"page_count,omitempty"`
	SheetCount int                    `json:"sheet_count,omitempty"`
	RowCount   int                    `json:"row_count,omitempty"`
	Author     string                 `json:"author,omitempty"`
	Title      string                 `json:"title,omitempty"`
	Subject    string                 `json:"subject,omitempty"`
	Keywords   []string               `json:"keywords,omitempty"`
	CreatedAt  string                 `json:"created_at,omitempty"`
	ModifiedAt string                 `json:"modified_at,omitempty"`
	Custom     map[string]interface{} `json:"custom,omitempty"`
}

// CreateFileRequest запрос на создание файла
type CreateFileRequest struct {
	UserID           string
	TenantID         *string
	Filename         string
	OriginalFilename string
	MimeType         string
	SizeBytes        int64
	ChecksumSHA256   *string
	StorageBackend   string
	StoragePath      string
	StorageBucket    *string
	ExtractedText    *string
	ExtractionStatus *string
	ExtractionError  *string
	PageCount        *int
	WordCount        *int
	Language         *string
	Metadata         *FileMetadata
}

// UpdateFileRequest запрос на обновление файла
type UpdateFileRequest struct {
	ExtractedText    *string
	ExtractionStatus *string
	ExtractionError  *string
	PageCount        *int
	WordCount        *int
	Language         *string
	Metadata         *FileMetadata
}

// ListFilesRequest запрос на получение списка файлов
type ListFilesRequest struct {
	UserID           *string
	TenantID         *string
	MimeType         *string
	ExtractionStatus *string
	Limit            int
	Offset           int
	SortBy           string // created_at, size_bytes, filename
	Order            string // asc, desc
}

// FileAccessLog лог доступа к файлу
type FileAccessLog struct {
	ID        int64     `json:"id" db:"id"`
	FileID    string    `json:"file_id" db:"file_id"`
	UserID    *string   `json:"user_id,omitempty" db:"user_id"`
	Action    string    `json:"action" db:"action"` // 'upload', 'download', 'delete', 'view'
	IPAddress *string   `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent *string   `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// FileWithUser файл с информацией о владельце (для админки)
type FileWithUser struct {
	*File
	OwnerEmail    *string `json:"owner_email,omitempty"`
	OwnerUsername *string `json:"owner_username,omitempty"`
}

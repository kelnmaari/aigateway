package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"aigateway/internal/models"
)

// ========================================
// Files CRUD Operations (FILE-STORAGE-01: v1.10.0+)
// ========================================

// CreateFile создает новую запись о файле
func (s *SQLiteDB) CreateFile(ctx context.Context, req models.CreateFileRequest) (*models.File, error) {
	// Генерируем ID
	id := generateID()

	// Сериализуем metadata если есть
	var metadataJSON *string
	if req.Metadata != nil {
		data, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		jsonStr := string(data)
		metadataJSON = &jsonStr
	}

	// Выполняем INSERT
	query := `
		INSERT INTO files (
			id, user_id, tenant_id, filename, original_filename, mime_type, size_bytes,
			checksum_sha256, storage_backend, storage_path, storage_bucket,
			extracted_text, extraction_status, metadata, page_count, word_count, language,
			created_at, updated_at
		) VALUES (
			?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?, ?, ?,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)
	`

	// Определяем extraction status
	extractionStatus := "pending"
	if req.ExtractionStatus != nil {
		extractionStatus = *req.ExtractionStatus
	} else if req.ExtractedText != nil {
		extractionStatus = "completed"
	}

	_, err := s.db.ExecContext(ctx, query,
		id, req.UserID, req.TenantID, req.Filename, req.OriginalFilename, req.MimeType, req.SizeBytes,
		req.ChecksumSHA256, req.StorageBackend, req.StoragePath, req.StorageBucket,
		req.ExtractedText, extractionStatus, metadataJSON, req.PageCount, req.WordCount, req.Language,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create file record: %w", err)
	}

	// Возвращаем созданный файл
	return s.GetFileByID(ctx, id)
}

// GetFileByID получает файл по ID
func (s *SQLiteDB) GetFileByID(ctx context.Context, fileID string) (*models.File, error) {
	query := `
		SELECT id, user_id, tenant_id, filename, original_filename, mime_type, size_bytes,
			checksum_sha256, storage_backend, storage_path, storage_bucket,
			extracted_text, extraction_status, extraction_error, metadata,
			page_count, word_count, language, is_public, shared_with,
			download_count, last_accessed_at, created_at, updated_at, deleted_at
		FROM files
		WHERE id = ? AND deleted_at IS NULL
	`

	file := &models.File{}
	err := s.db.QueryRowContext(ctx, query, fileID).Scan(
		&file.ID, &file.UserID, &file.TenantID, &file.Filename, &file.OriginalFilename, &file.MimeType, &file.SizeBytes,
		&file.ChecksumSHA256, &file.StorageBackend, &file.StoragePath, &file.StorageBucket,
		&file.ExtractedText, &file.ExtractionStatus, &file.ExtractionError, &file.Metadata,
		&file.PageCount, &file.WordCount, &file.Language, &file.IsPublic, &file.SharedWith,
		&file.DownloadCount, &file.LastAccessedAt, &file.CreatedAt, &file.UpdatedAt, &file.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not found: %s", fileID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	return file, nil
}

// UpdateFile обновляет информацию о файле
func (s *SQLiteDB) UpdateFile(ctx context.Context, fileID string, req models.UpdateFileRequest) (*models.File, error) {
	// Сериализуем metadata если есть
	var metadataJSON *string
	if req.Metadata != nil {
		data, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		jsonStr := string(data)
		metadataJSON = &jsonStr
	}

	query := `
		UPDATE files
		SET extracted_text = COALESCE(?, extracted_text),
			extraction_status = COALESCE(?, extraction_status),
			extraction_error = COALESCE(?, extraction_error),
			page_count = COALESCE(?, page_count),
			word_count = COALESCE(?, word_count),
			language = COALESCE(?, language),
			metadata = COALESCE(?, metadata),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query,
		req.ExtractedText, req.ExtractionStatus, req.ExtractionError,
		req.PageCount, req.WordCount, req.Language, metadataJSON,
		fileID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update file: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, fmt.Errorf("file not found: %s", fileID)
	}

	return s.GetFileByID(ctx, fileID)
}

// DeleteFile удаляет файл (soft delete)
func (s *SQLiteDB) DeleteFile(ctx context.Context, fileID string) error {
	query := `
		UPDATE files
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, fileID)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("file not found: %s", fileID)
	}

	return nil
}

// ListFilesWithUserInfo возвращает файлы с информацией о владельцах (для админки)
func (s *SQLiteDB) ListFilesWithUserInfo(ctx context.Context, req models.ListFilesRequest) ([]*models.FileWithUser, int, error) {
	// Base query с JOIN к users
	baseQuery := `
		FROM files f
		LEFT JOIN users u ON f.user_id = u.id
		WHERE f.deleted_at IS NULL
	`

	args := []interface{}{}

	// Фильтры
	if req.UserID != nil {
		baseQuery += " AND f.user_id = ?"
		args = append(args, *req.UserID)
	}
	if req.TenantID != nil {
		baseQuery += " AND f.tenant_id = ?"
		args = append(args, *req.TenantID)
	}
	if req.MimeType != nil {
		baseQuery += " AND f.mime_type = ?"
		args = append(args, *req.MimeType)
	}
	if req.ExtractionStatus != nil {
		baseQuery += " AND f.extraction_status = ?"
		args = append(args, *req.ExtractionStatus)
	}

	// Подсчет total
	countQuery := "SELECT COUNT(*) " + baseQuery
	var total int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count files: %w", err)
	}

	// Сортировка
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	order := req.Order
	if order == "" {
		order = "desc"
	}

	// Query для получения файлов
	query := `
		SELECT f.id, f.user_id, f.tenant_id, f.filename, f.original_filename, f.mime_type, f.size_bytes,
			f.checksum_sha256, f.storage_backend, f.storage_path, f.storage_bucket,
			f.extracted_text, f.extraction_status, f.extraction_error, f.metadata,
			f.page_count, f.word_count, f.language, f.is_public, f.shared_with,
			f.download_count, f.last_accessed_at, f.created_at, f.updated_at, f.deleted_at,
			u.email, u.username
	` + baseQuery + `
		ORDER BY f.` + sortBy + ` ` + order + `
		LIMIT ? OFFSET ?
	`

	args = append(args, req.Limit, req.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list files: %w", err)
	}
	defer rows.Close()

	var files []*models.FileWithUser
	for rows.Next() {
		file := &models.File{}
		var ownerEmail, ownerUsername sql.NullString

		err := rows.Scan(
			&file.ID, &file.UserID, &file.TenantID, &file.Filename, &file.OriginalFilename, &file.MimeType, &file.SizeBytes,
			&file.ChecksumSHA256, &file.StorageBackend, &file.StoragePath, &file.StorageBucket,
			&file.ExtractedText, &file.ExtractionStatus, &file.ExtractionError, &file.Metadata,
			&file.PageCount, &file.WordCount, &file.Language, &file.IsPublic, &file.SharedWith,
			&file.DownloadCount, &file.LastAccessedAt, &file.CreatedAt, &file.UpdatedAt, &file.DeletedAt,
			&ownerEmail, &ownerUsername,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan file row: %w", err)
		}

		fileWithUser := &models.FileWithUser{
			File: file,
		}

		if ownerEmail.Valid {
			fileWithUser.OwnerEmail = &ownerEmail.String
		}
		if ownerUsername.Valid {
			fileWithUser.OwnerUsername = &ownerUsername.String
		}

		files = append(files, fileWithUser)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating files: %w", err)
	}

	return files, total, nil
}

// ListFiles возвращает список файлов с фильтрацией и пагинацией
func (s *SQLiteDB) ListFiles(ctx context.Context, req models.ListFilesRequest) ([]*models.File, int, error) {
	// Base query
	baseQuery := `
		FROM files
		WHERE deleted_at IS NULL
	`

	args := []interface{}{}

	// Фильтры
	if req.UserID != nil {
		baseQuery += " AND user_id = ?"
		args = append(args, *req.UserID)
	}
	if req.TenantID != nil {
		baseQuery += " AND tenant_id = ?"
		args = append(args, *req.TenantID)
	}
	if req.MimeType != nil {
		baseQuery += " AND mime_type = ?"
		args = append(args, *req.MimeType)
	}
	if req.ExtractionStatus != nil {
		baseQuery += " AND extraction_status = ?"
		args = append(args, *req.ExtractionStatus)
	}

	// Считаем total
	var total int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) "+baseQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count files: %w", err)
	}

	// Получаем файлы
	sortBy := "created_at"
	if req.SortBy != "" {
		sortBy = req.SortBy
	}
	order := "DESC"
	if req.Order == "asc" {
		order = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, tenant_id, filename, original_filename, mime_type, size_bytes,
			checksum_sha256, storage_backend, storage_path, storage_bucket,
			extracted_text, extraction_status, extraction_error, metadata,
			page_count, word_count, language, is_public, shared_with,
			download_count, last_accessed_at, created_at, updated_at, deleted_at
		%s
		ORDER BY %s %s
		LIMIT ? OFFSET ?
	`, baseQuery, sortBy, order)

	args = append(args, req.Limit, req.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list files: %w", err)
	}
	defer rows.Close()

	files := make([]*models.File, 0)
	for rows.Next() {
		file := &models.File{}
		err := rows.Scan(
			&file.ID, &file.UserID, &file.TenantID, &file.Filename, &file.OriginalFilename, &file.MimeType, &file.SizeBytes,
			&file.ChecksumSHA256, &file.StorageBackend, &file.StoragePath, &file.StorageBucket,
			&file.ExtractedText, &file.ExtractionStatus, &file.ExtractionError, &file.Metadata,
			&file.PageCount, &file.WordCount, &file.Language, &file.IsPublic, &file.SharedWith,
			&file.DownloadCount, &file.LastAccessedAt, &file.CreatedAt, &file.UpdatedAt, &file.DeletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan file: %w", err)
		}
		files = append(files, file)
	}

	return files, total, nil
}

// IncrementDownloadCount увеличивает счетчик скачиваний
func (s *SQLiteDB) IncrementDownloadCount(ctx context.Context, fileID string) error {
	query := `
		UPDATE files
		SET download_count = download_count + 1,
			last_accessed_at = CURRENT_TIMESTAMP
		WHERE id = ? AND deleted_at IS NULL
	`

	_, err := s.db.ExecContext(ctx, query, fileID)
	return err
}

// LogFileAccess записывает лог доступа к файлу
func (s *SQLiteDB) LogFileAccess(ctx context.Context, log models.FileAccessLog) error {
	query := `
		INSERT INTO file_access_logs (file_id, user_id, action, ip_address, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	_, err := s.db.ExecContext(ctx, query, log.FileID, log.UserID, log.Action, log.IPAddress, log.UserAgent)
	if err != nil {
		return fmt.Errorf("failed to log file access: %w", err)
	}

	return nil
}

// GetFileAccessLogs возвращает историю доступа к файлу
func (s *SQLiteDB) GetFileAccessLogs(ctx context.Context, fileID string, limit int) ([]*models.FileAccessLog, error) {
	query := `
		SELECT id, file_id, user_id, action, ip_address, user_agent, created_at
		FROM file_access_logs
		WHERE file_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, fileID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get file access logs: %w", err)
	}
	defer rows.Close()

	logs := make([]*models.FileAccessLog, 0)
	for rows.Next() {
		log := &models.FileAccessLog{}
		err := rows.Scan(&log.ID, &log.FileID, &log.UserID, &log.Action, &log.IPAddress, &log.UserAgent, &log.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file access log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// generateID генерирует уникальный ID (можно использовать UUID или другой метод)
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}


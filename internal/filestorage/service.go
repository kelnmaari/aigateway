package filestorage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
)

// Service предоставляет высокоуровневые операции с файлами
type Service struct {
	storage   StorageBackend
	validator *Validator
	config    config.FileStorageConfig
	logger    *logrus.Logger
}

// NewService создает новый файловый сервис
func NewService(storage StorageBackend, cfg config.FileStorageConfig, logger *logrus.Logger) *Service {
	// Создаем валидатор на основе конфигурации
	var allowedExts []string
	var maxSize int64

	if cfg.Backend == "local" {
		allowedExts = cfg.Local.AllowedExts
		// maxSize будет парситься в storage backend
	} else if cfg.Backend == "s3" {
		// Для S3 можем использовать те же расширения
		// В production можно добавить отдельную секцию
	}

	validator := NewValidator(allowedExts, maxSize, cfg.Local.ValidateContent)

	return &Service{
		storage:   storage,
		validator: validator,
		config:    cfg,
		logger:    logger,
	}
}

// UploadRequest запрос на загрузку файла
type UploadRequest struct {
	Reader                io.Reader
	Filename              string
	MimeType              string
	Size                  int64
	UserID                string
	TenantID              string
	Public                bool
	Metadata              map[string]any
	Extract               bool // Извлекать ли текст (для Phase 2)
	SkipContentValidation bool // Пропустить валидацию magic number (для HTTP uploads)
}

// UploadResponse ответ на загрузку файла
type UploadResponse struct {
	Path          string
	Size          int64
	Checksum      string
	StorageType   string
	UploadedAt    time.Time
	ExtractedText string // Будет использоваться в Phase 2
}

// Upload загружает файл в хранилище
func (s *Service) Upload(ctx context.Context, req UploadRequest) (*UploadResponse, error) {
	// Валидация имени файла
	if err := s.validator.ValidateFilename(req.Filename); err != nil {
		return nil, fmt.Errorf("invalid filename: %w", err)
	}

	// Валидация размера и содержимого
	// Если SkipContentValidation=true, передаем nil вместо reader чтобы не читать содержимое
	var readerForValidation io.Reader
	if !req.SkipContentValidation {
		readerForValidation = req.Reader
	}

	if err := s.validator.ValidateFile(req.Filename, req.Size, readerForValidation); err != nil {
		return nil, fmt.Errorf("file validation failed: %w", err)
	}

	// Подсчитываем checksum
	var checksum string
	if req.Reader != nil {
		// Читаем данные для checksum
		hasher := sha256.New()
		tee := io.TeeReader(req.Reader, hasher)

		// Сохраняем в storage
		storeOpts := StoreOptions{
			UserID:   req.UserID,
			TenantID: req.TenantID,
			Filename: req.Filename,
			MimeType: req.MimeType,
			Metadata: req.Metadata,
			Public:   req.Public,
		}

		path, err := s.storage.Store(ctx, tee, storeOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to store file: %w", err)
		}

		checksum = hex.EncodeToString(hasher.Sum(nil))

		s.logger.WithFields(logrus.Fields{
			"path":     path,
			"user_id":  req.UserID,
			"filename": req.Filename,
			"size":     req.Size,
		}).Info("File uploaded successfully")

		return &UploadResponse{
			Path:        path,
			Size:        req.Size,
			Checksum:    checksum,
			StorageType: s.storage.Type(),
			UploadedAt:  time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("reader is nil")
}

// Download скачивает файл из хранилища
func (s *Service) Download(ctx context.Context, path, userID string) (io.ReadCloser, error) {
	// TODO: Добавить проверку прав доступа (user owns this file)

	reader, err := s.storage.Retrieve(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve file: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"path":    path,
		"user_id": userID,
	}).Debug("File downloaded")

	return reader, nil
}

// Delete удаляет файл из хранилища
func (s *Service) Delete(ctx context.Context, path, userID string) error {
	// TODO: Добавить проверку прав доступа

	if err := s.storage.Delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"path":    path,
		"user_id": userID,
	}).Info("File deleted")

	return nil
}

// GetURL возвращает URL для доступа к файлу
func (s *Service) GetURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	url, err := s.storage.GetURL(ctx, path, expiry)
	if err != nil {
		return "", fmt.Errorf("failed to get file URL: %w", err)
	}

	return url, nil
}

// Exists проверяет существование файла
func (s *Service) Exists(ctx context.Context, path string) (bool, error) {
	return s.storage.Exists(ctx, path)
}

// GetSize возвращает размер файла
func (s *Service) GetSize(ctx context.Context, path string) (int64, error) {
	return s.storage.GetSize(ctx, path)
}

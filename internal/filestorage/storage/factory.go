package storage

import (
	"fmt"
	"strings"
	"time"

	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/filestorage"
)

// NewStorageBackend создает storage backend на основе конфигурации
func NewStorageBackend(cfg config.FileStorageConfig) (filestorage.StorageBackend, error) {
	backend := strings.ToLower(cfg.Backend)

	switch backend {
	case "local":
		return newLocalFromConfig(cfg)
	case "s3":
		return newS3FromConfig(cfg)
	default:
		return nil, fmt.Errorf("unsupported storage backend: %s", cfg.Backend)
	}
}

// newLocalFromConfig создает LocalStorage из конфигурации
func newLocalFromConfig(cfg config.FileStorageConfig) (*LocalStorage, error) {
	// Парсим max file size
	maxFileSize, err := parseSize(cfg.Local.MaxFileSize)
	if err != nil {
		return nil, fmt.Errorf("invalid max_file_size: %w", err)
	}

	localCfg := LocalStorageConfig{
		BasePath:        cfg.Local.BasePath,
		MaxFileSize:     maxFileSize,
		ValidateContent: cfg.Local.ValidateContent,
	}

	return NewLocalStorage(localCfg)
}

// newS3FromConfig создает S3Storage из конфигурации
func newS3FromConfig(cfg config.FileStorageConfig) (*S3Storage, error) {
	// Парсим max file size
	maxFileSize, err := parseSize(cfg.S3.MaxFileSize)
	if err != nil {
		return nil, fmt.Errorf("invalid max_file_size: %w", err)
	}

	// Парсим URL expiry
	urlExpiry, err := time.ParseDuration(cfg.S3.SignedURLExpiry)
	if err != nil {
		return nil, fmt.Errorf("invalid signed_url_expiry: %w", err)
	}

	s3Cfg := S3StorageConfig{
		Endpoint:     cfg.S3.Endpoint,
		Bucket:       cfg.S3.Bucket,
		PublicBucket: cfg.S3.PublicBucket,
		AccessKey:    cfg.S3.AccessKey,
		SecretKey:    cfg.S3.SecretKey,
		UseSSL:       cfg.S3.UseSSL,
		Region:       cfg.S3.Region,
		URLExpiry:    urlExpiry,
		MaxFileSize:  maxFileSize,
	}

	return NewS3Storage(s3Cfg)
}

// parseSize парсит размер в формате "100MB", "10GB" и т.д.
func parseSize(sizeStr string) (int64, error) {
	if sizeStr == "" {
		return 0, nil
	}

	// Убираем пробелы
	sizeStr = strings.TrimSpace(sizeStr)

	// Определяем множитель по последним символам
	var multiplier int64 = 1
	var numStr string

	upper := strings.ToUpper(sizeStr)

	if strings.HasSuffix(upper, "GB") {
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(upper, "GB")
	} else if strings.HasSuffix(upper, "MB") {
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(upper, "MB")
	} else if strings.HasSuffix(upper, "KB") {
		multiplier = 1024
		numStr = strings.TrimSuffix(upper, "KB")
	} else if strings.HasSuffix(upper, "B") {
		multiplier = 1
		numStr = strings.TrimSuffix(upper, "B")
	} else {
		// Предполагаем что указаны байты
		numStr = sizeStr
	}

	numStr = strings.TrimSpace(numStr)

	var num int64
	_, err := fmt.Sscanf(numStr, "%d", &num)
	if err != nil {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	return num * multiplier, nil
}

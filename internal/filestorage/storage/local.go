// Package storage implements storage backends for file storage
package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"aigateway/internal/filestorage"
)

// LocalStorage реализует хранение файлов в локальной файловой системе
type LocalStorage struct {
	basePath        string
	maxFileSize     int64
	validateContent bool
}

// LocalStorageConfig конфигурация для локального хранилища
type LocalStorageConfig struct {
	BasePath        string // Базовый путь для хранения файлов
	MaxFileSize     int64  // Максимальный размер файла в байтах
	ValidateContent bool   // Проверять magic number файла
}

// NewLocalStorage создает новый экземпляр локального хранилища
func NewLocalStorage(config LocalStorageConfig) (*LocalStorage, error) {
	// Создаем базовый каталог если не существует
	if err := os.MkdirAll(config.BasePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &LocalStorage{
		basePath:        config.BasePath,
		maxFileSize:     config.MaxFileSize,
		validateContent: config.ValidateContent,
	}, nil
}

// Store сохраняет файл в локальной файловой системе
// Структура: basePath/userID/uuid_filename
func (s *LocalStorage) Store(ctx context.Context, file io.Reader, opts filestorage.StoreOptions) (string, error) {
	// Генерируем уникальный ID для файла
	fileID := uuid.New().String()

	// Формируем путь: userID/fileID_filename
	relativePath := filepath.Join(opts.UserID, fileID+"_"+opts.Filename)
	fullPath := filepath.Join(s.basePath, relativePath)

	// Создаем директорию пользователя
	userDir := filepath.Join(s.basePath, opts.UserID)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create user directory: %w", err)
	}

	// Создаем временный файл для записи
	tmpPath := fullPath + ".tmp"
	dst, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	// Копируем содержимое с подсчетом размера и checksum
	var written int64
	hasher := sha256.New()
	multiWriter := io.MultiWriter(dst, hasher)

	written, err = io.Copy(multiWriter, file)
	dst.Close()

	if err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Проверяем размер файла
	if s.maxFileSize > 0 && written > s.maxFileSize {
		os.Remove(tmpPath)
		return "", fmt.Errorf("file size %d exceeds limit %d", written, s.maxFileSize)
	}

	// Проверяем checksum если указан
	if opts.Checksum != "" {
		actualChecksum := hex.EncodeToString(hasher.Sum(nil))
		if actualChecksum != opts.Checksum {
			os.Remove(tmpPath)
			return "", fmt.Errorf("checksum mismatch: expected %s, got %s", opts.Checksum, actualChecksum)
		}
	}

	// Переименовываем временный файл в финальный
	if err := os.Rename(tmpPath, fullPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to rename temp file: %w", err)
	}

	return relativePath, nil
}

// Retrieve получает содержимое файла
func (s *LocalStorage) Retrieve(ctx context.Context, path string) (io.ReadCloser, error) {
	// Нормализуем путь для текущей ОС (конвертируем слэши)
	path = filepath.FromSlash(path)
	cleanPath := filepath.Clean(path)

	// Проверяем на абсолютный путь
	if filepath.IsAbs(cleanPath) {
		return nil, fmt.Errorf("absolute paths not allowed: %s", path)
	}

	// Проверяем на path traversal (..)
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("path traversal not allowed: %s", path)
	}

	fullPath := filepath.Join(s.basePath, cleanPath)

	// Убедимся что итоговый путь находится внутри basePath (используем Abs для корректного сравнения)
	absBasePath, err := filepath.Abs(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base path: %w", err)
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve file path: %w", err)
	}

	// Нормализуем для сравнения (на Windows учитываем регистр)
	absBasePath = filepath.Clean(absBasePath) + string(filepath.Separator)
	absFullPath = filepath.Clean(absFullPath)

	if !strings.HasPrefix(absFullPath+string(filepath.Separator), absBasePath) && absFullPath != strings.TrimSuffix(absBasePath, string(filepath.Separator)) {
		return nil, fmt.Errorf("path outside base directory: %s", path)
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete удаляет файл
func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	// Нормализуем путь
	path = filepath.FromSlash(path)
	cleanPath := filepath.Clean(path)

	// Проверяем на абсолютный путь
	if filepath.IsAbs(cleanPath) {
		return fmt.Errorf("absolute paths not allowed: %s", path)
	}

	// Проверяем на path traversal
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed: %s", path)
	}

	fullPath := filepath.Join(s.basePath, cleanPath)

	// Убедимся что путь внутри basePath
	absBasePath, err := filepath.Abs(s.basePath)
	if err != nil {
		return fmt.Errorf("failed to resolve base path: %w", err)
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("failed to resolve file path: %w", err)
	}

	if !strings.HasPrefix(absFullPath, absBasePath) {
		return fmt.Errorf("path traversal attempt: %s", path)
	}

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", path)
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Exists проверяет существование файла
func (s *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
	cleanPath := filepath.Clean(path)
	fullPath := filepath.Join(s.basePath, cleanPath)

	if !filepath.HasPrefix(fullPath, s.basePath) {
		return false, fmt.Errorf("path traversal attempt: %s", path)
	}

	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// GetURL возвращает относительный путь (для local storage нет публичных URL)
func (s *LocalStorage) GetURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	// Для локального хранилища просто возвращаем путь
	// API endpoint будет обрабатывать /api/files/{id}/download
	return path, nil
}

// GetSize возвращает размер файла в байтах
func (s *LocalStorage) GetSize(ctx context.Context, path string) (int64, error) {
	cleanPath := filepath.Clean(path)
	fullPath := filepath.Join(s.basePath, cleanPath)

	if !filepath.HasPrefix(fullPath, s.basePath) {
		return 0, fmt.Errorf("path traversal attempt: %s", path)
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("file not found: %s", path)
		}
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}

	return info.Size(), nil
}

// Type возвращает тип backend
func (s *LocalStorage) Type() string {
	return "local"
}


package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"ollama-openai-proxy/internal/filestorage"
)

// S3Storage реализует хранение файлов в S3-compatible storage (MinIO)
type S3Storage struct {
	client       *minio.Client
	bucket       string
	publicBucket string
	urlExpiry    time.Duration
	maxFileSize  int64
}

// S3StorageConfig конфигурация для S3 хранилища
type S3StorageConfig struct {
	Endpoint     string // http://minio:9000
	Bucket       string // user-files
	PublicBucket string // public-files (for shared)
	AccessKey    string
	SecretKey    string
	UseSSL       bool
	Region       string
	URLExpiry    time.Duration // 1 hour
	MaxFileSize  int64
}

// NewS3Storage создает новый экземпляр S3 хранилища
func NewS3Storage(config S3StorageConfig) (*S3Storage, error) {
	// Создаем MinIO client
	client, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.UseSSL,
		Region: config.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	storage := &S3Storage{
		client:       client,
		bucket:       config.Bucket,
		publicBucket: config.PublicBucket,
		urlExpiry:    config.URLExpiry,
		maxFileSize:  config.MaxFileSize,
	}

	// Создаем bucket если не существует
	ctx := context.Background()
	if err := storage.ensureBucket(ctx, config.Bucket); err != nil {
		return nil, err
	}

	// Создаем public bucket если указан
	if config.PublicBucket != "" {
		if err := storage.ensureBucket(ctx, config.PublicBucket); err != nil {
			return nil, err
		}
	}

	return storage, nil
}

// ensureBucket создает bucket если он не существует
func (s *S3Storage) ensureBucket(ctx context.Context, bucketName string) error {
	exists, err := s.client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		if err := s.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", bucketName, err)
		}
	}

	return nil
}

// Store сохраняет файл в S3
// Структура: userID/fileID_filename
func (s *S3Storage) Store(ctx context.Context, file io.Reader, opts filestorage.StoreOptions) (string, error) {
	// Генерируем уникальный ID для файла
	fileID := uuid.New().String()

	// Формируем object key: userID/fileID_filename
	objectKey := filepath.Join(opts.UserID, fileID+"_"+opts.Filename)

	// Выбираем bucket
	bucket := s.bucket
	if opts.Public && s.publicBucket != "" {
		bucket = s.publicBucket
	}

	// Читаем данные в буфер для расчета размера и checksum
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file data: %w", err)
	}

	// Проверяем размер
	if s.maxFileSize > 0 && int64(len(data)) > s.maxFileSize {
		return "", fmt.Errorf("file size %d exceeds limit %d", len(data), s.maxFileSize)
	}

	// Проверяем checksum если указан
	if opts.Checksum != "" {
		hasher := sha256.New()
		hasher.Write(data)
		actualChecksum := hex.EncodeToString(hasher.Sum(nil))
		if actualChecksum != opts.Checksum {
			return "", fmt.Errorf("checksum mismatch: expected %s, got %s", opts.Checksum, actualChecksum)
		}
	}

	// Загружаем в S3
	reader := bytes.NewReader(data)
	uploadOpts := minio.PutObjectOptions{
		ContentType: opts.MimeType,
	}

	// Добавляем user metadata
	if opts.Metadata != nil {
		uploadOpts.UserMetadata = make(map[string]string)
		for k, v := range opts.Metadata {
			uploadOpts.UserMetadata[k] = fmt.Sprintf("%v", v)
		}
	}

	_, err = s.client.PutObject(ctx, bucket, objectKey, reader, int64(len(data)), uploadOpts)
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Возвращаем путь в формате bucket/objectKey
	return fmt.Sprintf("%s/%s", bucket, objectKey), nil
}

// Retrieve получает содержимое файла из S3
func (s *S3Storage) Retrieve(ctx context.Context, path string) (io.ReadCloser, error) {
	// Парсим путь: bucket/objectKey
	bucket, objectKey, err := s.parsePath(path)
	if err != nil {
		return nil, err
	}

	object, err := s.client.GetObject(ctx, bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}

	// Проверяем что объект существует
	_, err = object.Stat()
	if err != nil {
		object.Close()
		return nil, fmt.Errorf("object not found: %w", err)
	}

	return object, nil
}

// Delete удаляет файл из S3
func (s *S3Storage) Delete(ctx context.Context, path string) error {
	bucket, objectKey, err := s.parsePath(path)
	if err != nil {
		return err
	}

	if err := s.client.RemoveObject(ctx, bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("failed to delete object from S3: %w", err)
	}

	return nil
}

// Exists проверяет существование файла в S3
func (s *S3Storage) Exists(ctx context.Context, path string) (bool, error) {
	bucket, objectKey, err := s.parsePath(path)
	if err != nil {
		return false, err
	}

	_, err = s.client.StatObject(ctx, bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		// Проверяем тип ошибки
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// GetURL возвращает подписанный URL для скачивания
func (s *S3Storage) GetURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	bucket, objectKey, err := s.parsePath(path)
	if err != nil {
		return "", err
	}

	// Используем expiry из аргумента или дефолтный
	if expiry == 0 {
		expiry = s.urlExpiry
	}

	url, err := s.client.PresignedGetObject(ctx, bucket, objectKey, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url.String(), nil
}

// GetSize возвращает размер файла в байтах
func (s *S3Storage) GetSize(ctx context.Context, path string) (int64, error) {
	bucket, objectKey, err := s.parsePath(path)
	if err != nil {
		return 0, err
	}

	stat, err := s.client.StatObject(ctx, bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		errResponse := minio.ToErrorResponse(err)
		if errResponse.Code == "NoSuchKey" {
			return 0, fmt.Errorf("file not found: %s", path)
		}
		return 0, fmt.Errorf("failed to get object stat: %w", err)
	}

	return stat.Size, nil
}

// Type возвращает тип backend
func (s *S3Storage) Type() string {
	return "s3"
}

// parsePath парсит путь в формате bucket/objectKey
func (s *S3Storage) parsePath(path string) (bucket, objectKey string, err error) {
	// Путь должен быть в формате: bucket/userID/fileID_filename
	// Например: user-files/user123/abc-def-ghi_document.pdf

	// Простой парсинг: ищем первый слеш
	parts := filepath.SplitList(filepath.ToSlash(path))
	if len(parts) < 2 {
		// Если нет bucket в пути, используем дефолтный
		bucket = s.bucket
		objectKey = path
		return bucket, objectKey, nil
	}

	// Иначе первая часть - bucket, остальное - object key
	bucket = parts[0]
	objectKey = filepath.Join(parts[1:]...)

	return bucket, objectKey, nil
}

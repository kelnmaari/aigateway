// Package logger provides structured logging setup for Ollama-OpenAI Proxy
package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"ollama-openai-proxy/internal/config"
)

// Setup настраивает логгер согласно конфигурации
func Setup(cfg *config.Config) *logrus.Logger {
	logger := logrus.New()

	// Настройка уровня логирования
	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		logger.Warn("Invalid log level, using info")
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Настройка формата логов
	switch cfg.Logging.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	default:
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// Настройка вывода логов
	output := setupOutput(cfg.Logging)
	logger.SetOutput(output)

	// Добавляем structured fields если настроены
	if len(cfg.Logging.StructuredFields) > 0 {
		fields := make(logrus.Fields)
		for k, v := range cfg.Logging.StructuredFields {
			fields[k] = v
		}
		logger = logger.WithFields(fields).Logger
	}

	// Добавляем базовые поля для всех логов
	logger = logger.WithFields(logrus.Fields{
		"service": "ollama-openai-proxy",
		"version": "dev", // TODO: Получать из build информации
	}).Logger

	return logger
}

// rotatePreviousLog переименовывает существующий лог-файл перед запуском
func rotatePreviousLog(filePath string) error {
	// Проверяем существует ли текущий лог-файл
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Файла нет, ничего не делаем
		return nil
	}

	// Формируем имя для предыдущего лога
	dir := filepath.Dir(filePath)
	ext := filepath.Ext(filePath)
	nameWithoutExt := filepath.Base(filePath[:len(filePath)-len(ext)])
	previousLogPath := filepath.Join(dir, nameWithoutExt+"-previous"+ext)

	// Удаляем старый previous файл если существует
	if _, err := os.Stat(previousLogPath); err == nil {
		if err := os.Remove(previousLogPath); err != nil {
			return err
		}
	}

	// Переименовываем текущий лог в previous
	if err := os.Rename(filePath, previousLogPath); err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"old_file": filePath,
		"new_file": previousLogPath,
	}).Info("Rotated previous log file on startup")

	return nil
}

// setupOutput настраивает вывод логов (файл, stdout, или оба)
func setupOutput(cfg config.LoggingConfig) io.Writer {
	var writers []io.Writer

	// Всегда добавляем stdout если не указано иное
	if cfg.Output == "stdout" || cfg.Output == "" {
		writers = append(writers, os.Stdout)
	}

	// Добавляем файл если настроен
	if cfg.Output == "file" || cfg.Output == "both" {
		if cfg.FilePath == "" {
			logrus.Warn("File path not specified, using stdout only")
			return os.Stdout
		}

		// Создаем директорию если не существует
		dir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			logrus.WithError(err).Warn("Failed to create log directory, using stdout")
			return os.Stdout
		}

		// 🔄 Переименовываем старый лог-файл перед запуском
		if err := rotatePreviousLog(cfg.FilePath); err != nil {
			logrus.WithError(err).Warn("Failed to rotate previous log file, continuing with current file")
		}

		// Настраиваем log rotation с lumberjack
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,    // MB
			MaxBackups: cfg.MaxBackups, // Количество старых файлов
			MaxAge:     cfg.MaxAge,     // Дней
			Compress:   cfg.Compress,   // Сжимать старые логи
		}

		writers = append(writers, fileWriter)

		logrus.WithFields(logrus.Fields{
			"file_path":   cfg.FilePath,
			"max_size":    cfg.MaxSize,
			"max_backups": cfg.MaxBackups,
			"max_age":     cfg.MaxAge,
			"compress":    cfg.Compress,
		}).Info("File logging configured with rotation")
	}

	// Если настроен "both", добавляем и stdout
	if cfg.Output == "both" {
		writers = append([]io.Writer{os.Stdout}, writers...) // stdout первым
	}

	// Возвращаем MultiWriter если несколько выходов
	if len(writers) > 1 {
		return io.MultiWriter(writers...)
	}

	if len(writers) == 1 {
		return writers[0]
	}

	return os.Stdout
}

// WithRequestID добавляет request ID к logger
func WithRequestID(logger *logrus.Logger, requestID string) *logrus.Entry {
	return logger.WithField("request_id", requestID)
}

// WithError добавляет ошибку к logger
func WithError(logger *logrus.Logger, err error) *logrus.Entry {
	return logger.WithError(err)
}

// WithFields добавляет дополнительные поля к logger
func WithFields(logger *logrus.Logger, fields map[string]interface{}) *logrus.Entry {
	return logger.WithFields(logrus.Fields(fields))
}

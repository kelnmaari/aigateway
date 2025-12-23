// Package logger provides structured logging setup for Ollama-OpenAI Proxy
package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"aigateway/internal/config"
	"aigateway/internal/version"
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

	// Настройка отдельного файла для ошибок и warnings
	if cfg.Logging.ErrorLogEnabled && cfg.Logging.ErrorLogFilePath != "" {
		errorHook := setupErrorLogHook(cfg.Logging, logger.Formatter)
		if errorHook != nil {
			logger.AddHook(errorHook)
			logger.WithFields(logrus.Fields{
				"error_log_file": cfg.Logging.ErrorLogFilePath,
				"max_size":       cfg.Logging.ErrorLogMaxSize,
				"max_backups":    cfg.Logging.ErrorLogMaxBackups,
			}).Info("Separate error log file configured")
		}
	}

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
		"version": version.Version,
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

// setupErrorLogHook настраивает hook для записи error/warning логов в отдельный файл
func setupErrorLogHook(cfg config.LoggingConfig, formatter logrus.Formatter) *ErrorLogHook {
	if cfg.ErrorLogFilePath == "" {
		logrus.Warn("Error log file path not specified, skipping error log hook")
		return nil
	}

	// Создаем директорию если не существует
	dir := filepath.Dir(cfg.ErrorLogFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logrus.WithError(err).Warn("Failed to create error log directory, skipping error log hook")
		return nil
	}

	// Используем defaults из основной конфигурации если не заданы отдельно
	maxSize := cfg.ErrorLogMaxSize
	if maxSize == 0 {
		maxSize = cfg.MaxSize
		if maxSize == 0 {
			maxSize = 100 // default 100MB
		}
	}

	maxBackups := cfg.ErrorLogMaxBackups
	if maxBackups == 0 {
		maxBackups = cfg.MaxBackups
		if maxBackups == 0 {
			maxBackups = 3 // default 3 backups
		}
	}

	maxAge := cfg.ErrorLogMaxAge
	if maxAge == 0 {
		maxAge = cfg.MaxAge
		if maxAge == 0 {
			maxAge = 7 // default 7 days
		}
	}

	compress := cfg.ErrorLogCompress || cfg.Compress

	// 🔄 Переименовываем старый error log перед запуском
	if err := rotatePreviousLog(cfg.ErrorLogFilePath); err != nil {
		logrus.WithError(err).Warn("Failed to rotate previous error log file, continuing with current file")
	}

	return NewErrorLogHook(
		cfg.ErrorLogFilePath,
		maxSize,
		maxBackups,
		maxAge,
		compress,
		formatter,
	)
}

// NewFileLogger создает отдельный logger для вывода в конкретный файл
// Поддерживает ротацию при старте (переименование в -previous) и по размеру (lumberjack)
func NewFileLogger(filePath string, levelStr string) *logrus.Logger {
	logger := logrus.New()
	
	// Настройка уровня логирования
	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		logger.Warn("Invalid log level, using info")
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)
	
	// Форматирование логов
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	
	// Создаем директорию если не существует
	if dir := filepath.Dir(filePath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.WithError(err).Error("Failed to create log directory")
			logger.SetOutput(os.Stdout)
			return logger
		}
	}
	
	// 🔄 Ротация предыдущего лога при старте (как у основного логгера)
	if err := rotatePreviousLog(filePath); err != nil {
		// Не критично - продолжаем работу
		logrus.WithError(err).WithField("file", filePath).Debug("Failed to rotate previous log")
	}
	
	// Настройка ротации логов по размеру
	fileWriter := &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    50,   // MB - ротация при достижении 50MB
		MaxBackups: 3,    // Хранить 3 старых файла
		MaxAge:     14,   // Удалять старше 14 дней
		Compress:   true, // Сжимать старые логи
	}
	
	// Multi-writer: file + stdout
	multiWriter := io.MultiWriter(fileWriter, os.Stdout)
	logger.SetOutput(multiWriter)
	
	return logger
}

// Package logger provides error logging hook for separate error file
package logger

import (
	"io"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// ErrorLogHook перехватывает error и warning сообщения и пишет в отдельный файл
type ErrorLogHook struct {
	writer    io.Writer
	formatter logrus.Formatter
	levels    []logrus.Level
}

// NewErrorLogHook создает новый hook для error логов
func NewErrorLogHook(filePath string, maxSize, maxBackups, maxAge int, compress bool, formatter logrus.Formatter) *ErrorLogHook {
	// Настраиваем rotation для error log file
	errorWriter := &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   compress,
	}

	return &ErrorLogHook{
		writer:    errorWriter,
		formatter: formatter,
		levels: []logrus.Level{
			logrus.ErrorLevel,
			logrus.FatalLevel,
			logrus.PanicLevel,
			logrus.WarnLevel, // Включаем warnings тоже
		},
	}
}

// Levels определяет на каких уровнях срабатывает hook
func (hook *ErrorLogHook) Levels() []logrus.Level {
	return hook.levels
}

// Fire вызывается при логировании сообщения нужного уровня
func (hook *ErrorLogHook) Fire(entry *logrus.Entry) error {
	// Форматируем сообщение
	serialized, err := hook.formatter.Format(entry)
	if err != nil {
		return err
	}

	// Пишем в error log file
	_, err = hook.writer.Write(serialized)
	return err
}


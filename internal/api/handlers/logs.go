package handlers

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LogsHandler обрабатывает запросы к логам системы
type LogsHandler struct {
	logsDir string
	logger  *logrus.Logger
}

// NewLogsHandler создает новый Logs Handler
func NewLogsHandler(logsDir string, logger *logrus.Logger) *LogsHandler {
	return &LogsHandler{
		logsDir: logsDir,
		logger:  logger,
	}
}

// LogFile представляет информацию о лог-файле
type LogFile struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Modified  time.Time `json:"modified"`
	IsCurrent bool      `json:"is_current"`
}

// LogEntry представляет одну запись в логе
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Raw       string `json:"raw"`
}

// ListLogFiles возвращает список всех лог-файлов
// GET /api/admin/logs
func (h *LogsHandler) ListLogFiles(c *gin.Context) {
	h.logger.Debug("Listing log files")

	files, err := os.ReadDir(h.logsDir)
	if err != nil {
		h.logger.WithError(err).Error("Failed to read logs directory")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read logs directory",
		})
		return
	}

	var logFiles []LogFile
	currentLogName := "proxy.log" // Текущий активный лог

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Фильтруем только .log файлы
		if !strings.HasSuffix(file.Name(), ".log") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			h.logger.WithError(err).Warn("Failed to get file info", "file", file.Name())
			continue
		}

		logFiles = append(logFiles, LogFile{
			Name:      file.Name(),
			Size:      info.Size(),
			Modified:  info.ModTime(),
			IsCurrent: file.Name() == currentLogName,
		})
	}

	// Сортируем по дате модификации (новые первые)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].Modified.After(logFiles[j].Modified)
	})

	c.JSON(http.StatusOK, gin.H{
		"files": logFiles,
		"total": len(logFiles),
	})
}

// GetLogFile возвращает содержимое конкретного лог-файла
// GET /api/admin/logs/:filename
func (h *LogsHandler) GetLogFile(c *gin.Context) {
	filename := c.Param("filename")

	h.logger.WithField("filename", filename).Debug("Reading log file")

	// Валидация имени файла (security)
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid filename",
		})
		return
	}

	// Query параметры
	limitStr := c.DefaultQuery("limit", "500") // Последние N строк
	offsetStr := c.DefaultQuery("offset", "0") // Пропустить первые N строк
	level := c.Query("level")                  // Фильтр по уровню

	var limit, offset int
	fmt.Sscanf(limitStr, "%d", &limit)
	fmt.Sscanf(offsetStr, "%d", &offset)

	if limit <= 0 || limit > 10000 {
		limit = 500
	}

	filePath := filepath.Join(h.logsDir, filename)

	// Проверка существования файла
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Log file not found",
		})
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		h.logger.WithError(err).Error("Failed to open log file")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to open log file",
		})
		return
	}
	defer file.Close()

	// Читаем строки
	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// Фильтр по уровню
		if level != "" && !strings.Contains(strings.ToLower(line), "level="+strings.ToLower(level)) {
			continue
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		h.logger.WithError(err).Error("Failed to read log file")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read log file",
		})
		return
	}

	// Применяем offset и limit
	totalLines := len(lines)

	if offset >= totalLines {
		offset = 0
	}

	end := offset + limit
	if end > totalLines {
		end = totalLines
	}

	// Берем последние строки (обратный порядок для свежих логов)
	if offset == 0 && limit > 0 {
		start := totalLines - limit
		if start < 0 {
			start = 0
		}
		lines = lines[start:]
	} else {
		lines = lines[offset:end]
	}

	// Парсим строки в структурированные записи
	entries := make([]LogEntry, 0, len(lines))
	for _, line := range lines {
		entry := h.parseLogLine(line)
		entries = append(entries, entry)
	}

	c.JSON(http.StatusOK, gin.H{
		"filename": filename,
		"entries":  entries,
		"total":    totalLines,
		"returned": len(entries),
	})
}

// StreamLogs предоставляет real-time stream логов через SSE
// GET /api/admin/logs/stream?file=proxy.log
func (h *LogsHandler) StreamLogs(c *gin.Context) {
	h.logger.Debug("Starting log stream")

	// Устанавливаем SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Получаем имя файла из query параметра (по умолчанию proxy.log)
	filename := c.DefaultQuery("file", "proxy.log")

	// Защита от directory traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		h.logger.WithField("filename", filename).Warn("Attempted directory traversal in log stream request")
		c.SSEvent("error", gin.H{"message": "Invalid filename"})
		return
	}

	// Текущий лог файл
	logFile := filepath.Join(h.logsDir, filename)

	// Открываем файл
	file, err := os.Open(logFile)
	if err != nil {
		h.logger.WithError(err).WithField("file", logFile).Error("Failed to open log file for streaming")
		c.SSEvent("error", gin.H{"message": fmt.Sprintf("Failed to open log file: %s", filename)})
		return
	}
	defer file.Close()

	// Переходим в конец файла
	file.Seek(0, io.SeekEnd)

	// Context для отслеживания отключения клиента
	ctx := c.Request.Context()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	scanner := bufio.NewScanner(file)

	for {
		select {
		case <-ctx.Done():
			h.logger.Debug("Client disconnected from log stream")
			return

		case <-ticker.C:
			// Проверяем новые строки
			for scanner.Scan() {
				line := scanner.Text()
				entry := h.parseLogLine(line)

				// Отправляем событие
				c.SSEvent("log", entry)
				c.Writer.Flush()
			}

			if err := scanner.Err(); err != nil {
				h.logger.WithError(err).Error("Error reading log stream")
				c.SSEvent("error", gin.H{"message": "Error reading logs"})
				return
			}
		}
	}
}

// parseLogLine парсит строку лога в структурированную запись
func (h *LogsHandler) parseLogLine(line string) LogEntry {
	entry := LogEntry{
		Raw: line,
	}

	// Logrus формат: time="2025-10-10 19:06:57" level=debug msg="..." key1=value1 key2=value2

	// Извлекаем поля через regex-подобный парсинг
	fields := h.parseLogrusFields(line)

	// Извлекаем стандартные поля
	if timestamp, ok := fields["time"]; ok {
		entry.Timestamp = timestamp
		delete(fields, "time")
	}

	if level, ok := fields["level"]; ok {
		entry.Level = strings.ToUpper(level)
		delete(fields, "level")
	} else {
		entry.Level = "UNKNOWN"
	}

	if msg, ok := fields["msg"]; ok {
		entry.Message = msg
		delete(fields, "msg")
	} else {
		entry.Message = line
	}

	// Все оставшиеся поля добавляем в сообщение как контекст
	if len(fields) > 0 {
		var extras []string
		for key, value := range fields {
			extras = append(extras, fmt.Sprintf("%s=%s", key, value))
		}
		if len(extras) > 0 {
			entry.Message = entry.Message + " | " + strings.Join(extras, " ")
		}
	}

	return entry
}

// parseLogrusFields парсит logrus формат в map полей
func (h *LogsHandler) parseLogrusFields(line string) map[string]string {
	fields := make(map[string]string)

	// Простой парсер для формата key=value и key="quoted value"
	i := 0
	for i < len(line) {
		// Пропускаем пробелы
		for i < len(line) && line[i] == ' ' {
			i++
		}
		if i >= len(line) {
			break
		}

		// Находим ключ
		keyStart := i
		for i < len(line) && line[i] != '=' {
			i++
		}
		if i >= len(line) {
			break
		}
		key := line[keyStart:i]
		i++ // Пропускаем '='

		// Находим значение
		var value string
		if i < len(line) && line[i] == '"' {
			// Quoted value
			i++ // Пропускаем открывающую кавычку
			valueStart := i
			for i < len(line) && line[i] != '"' {
				if line[i] == '\\' && i+1 < len(line) {
					i += 2 // Пропускаем escaped символ
				} else {
					i++
				}
			}
			value = line[valueStart:i]
			if i < len(line) {
				i++ // Пропускаем закрывающую кавычку
			}
		} else {
			// Unquoted value (до пробела)
			valueStart := i
			for i < len(line) && line[i] != ' ' {
				i++
			}
			value = line[valueStart:i]
		}

		if key != "" {
			fields[key] = value
		}
	}

	return fields
}

// DownloadLogFile позволяет скачать лог-файл
// GET /api/admin/logs/:filename/download
func (h *LogsHandler) DownloadLogFile(c *gin.Context) {
	filename := c.Param("filename")

	h.logger.WithField("filename", filename).Debug("Downloading log file")

	// Валидация имени файла
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid filename",
		})
		return
	}

	filePath := filepath.Join(h.logsDir, filename)

	// Проверка существования
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Log file not found",
		})
		return
	}

	// Отправляем файл
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.File(filePath)
}


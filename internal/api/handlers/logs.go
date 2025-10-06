package handlers

import (
	"bufio"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/config"
)

// LogsHandler обрабатывает запросы к логам
type LogsHandler struct {
	config *config.Config
	logger *logrus.Logger
}

// NewLogsHandler создает новый logs handler
func NewLogsHandler(cfg *config.Config, logger *logrus.Logger) *LogsHandler {
	return &LogsHandler{
		config: cfg,
		logger: logger,
	}
}

// LogEntry представляет одну запись лога
type LogEntry struct {
	Line  string `json:"line"`
	Level string `json:"level"`
}

// GetLogs возвращает последние N строк логов
func (h *LogsHandler) GetLogs(c *gin.Context) {
	// Параметры
	limitStr := c.DefaultQuery("limit", "100")
	filterLevel := c.Query("level") // debug, info, warning, error

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000 // Max 1000 lines
	}

	// Путь к файлу логов
	logPath := h.config.Logging.FilePath
	if logPath == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Log file path not configured",
		})
		return
	}

	// Читаем файл
	file, err := os.Open(logPath)
	if err != nil {
		h.logger.WithError(err).Error("Failed to open log file")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read logs",
		})
		return
	}
	defer file.Close()

	// Читаем все строки
	var allLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		h.logger.WithError(err).Error("Failed to scan log file")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read logs",
		})
		return
	}

	// Берем последние N строк
	start := 0
	if len(allLines) > limit {
		start = len(allLines) - limit
	}

	lines := allLines[start:]

	// Фильтрация по уровню
	var entries []LogEntry
	for _, line := range lines {
		level := detectLogLevel(line)

		// Применяем фильтр
		if filterLevel != "" && !strings.EqualFold(level, filterLevel) {
			continue
		}

		entries = append(entries, LogEntry{
			Line:  line,
			Level: level,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  entries,
		"total": len(entries),
		"limit": limit,
	})
}

// detectLogLevel определяет уровень лога по содержимому строки
func detectLogLevel(line string) string {
	lowerLine := strings.ToLower(line)

	if strings.Contains(lowerLine, "level=error") || strings.Contains(lowerLine, "level=fatal") || strings.Contains(lowerLine, "level=panic") {
		return "error"
	}
	if strings.Contains(lowerLine, "level=warning") || strings.Contains(lowerLine, "level=warn") {
		return "warning"
	}
	if strings.Contains(lowerLine, "level=info") {
		return "info"
	}
	if strings.Contains(lowerLine, "level=debug") {
		return "debug"
	}

	// Fallback - определяем по ключевым словам
	if strings.Contains(lowerLine, "error") || strings.Contains(lowerLine, "fail") {
		return "error"
	}
	if strings.Contains(lowerLine, "warn") {
		return "warning"
	}

	return "info"
}

package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/filestorage"
	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// AdminFilesHandler обрабатывает административные операции с файлами (v1.10.0+)
type AdminFilesHandler struct {
	fileService *filestorage.Service
	db          storage.Database
	logger      *logrus.Logger
}

// NewAdminFilesHandler создает новый admin files handler
func NewAdminFilesHandler(
	fileService *filestorage.Service,
	db storage.Database,
	logger *logrus.Logger,
) *AdminFilesHandler {
	return &AdminFilesHandler{
		fileService: fileService,
		db:          db,
		logger:      logger,
	}
}

// ListAllFiles возвращает список всех файлов в системе (только для админов)
// GET /api/admin/files
func (h *AdminFilesHandler) ListAllFiles(c *gin.Context) {
	// Парсим параметры пагинации и фильтрации
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	sortBy := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")

	// Фильтры
	userID := c.Query("user_id")
	mimeType := c.Query("mime_type")
	extractionStatus := c.Query("extraction_status")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	// Получаем файлы
	req := models.ListFilesRequest{
		Limit:  limit,
		Offset: offset,
		SortBy: sortBy,
		Order:  order,
	}

	if userID != "" {
		req.UserID = &userID
	}
	if mimeType != "" {
		req.MimeType = &mimeType
	}
	if extractionStatus != "" {
		req.ExtractionStatus = &extractionStatus
	}

	// Используем метод с информацией о пользователях
	filesWithUser, total, err := h.db.ListFilesWithUserInfo(c.Request.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list all files")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list files"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"files":  filesWithUser,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// DeleteFile удаляет файл любого пользователя (только для админов)
// DELETE /api/admin/files/:id
func (h *AdminFilesHandler) DeleteFile(c *gin.Context) {
	adminID := c.GetString("user_id")
	fileID := c.Param("id")

	// Получаем информацию о файле
	file, err := h.db.GetFileByID(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// Удаляем файл из storage
	if err := h.fileService.Delete(c.Request.Context(), file.StoragePath, file.UserID); err != nil {
		h.logger.WithError(err).Error("Failed to delete file from storage")
		// Продолжаем, чтобы удалить из БД даже если storage failed
	}

	// Удаляем запись из БД
	if err := h.db.DeleteFile(c.Request.Context(), fileID); err != nil {
		h.logger.WithError(err).Error("Failed to delete file from database")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete file"})
		return
	}

	// Логируем действие администратора
	_ = h.db.LogFileAccess(c.Request.Context(), models.FileAccessLog{
		FileID:    fileID,
		UserID:    &adminID,
		Action:    "admin_delete",
		IPAddress: stringPtr(c.ClientIP()),
		UserAgent: stringPtr(c.Request.UserAgent()),
	})

	h.logger.WithFields(logrus.Fields{
		"file_id":  fileID,
		"admin_id": adminID,
		"owner_id": file.UserID,
		"filename": file.Filename,
	}).Info("File deleted by admin")

	c.JSON(http.StatusOK, gin.H{"message": "file deleted successfully"})
}

// GetFileStats возвращает статистику по файлам (только для админов)
// GET /api/admin/files/stats
func (h *AdminFilesHandler) GetFileStats(c *gin.Context) {
	// Получаем все файлы для подсчета статистики
	allFiles, _, err := h.db.ListFiles(c.Request.Context(), models.ListFilesRequest{
		Limit:  10000, // Большой лимит для статистики
		Offset: 0,
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to get file stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}

	// Подсчитываем статистику
	stats := struct {
		TotalFiles     int            `json:"total_files"`
		TotalSize      int64          `json:"total_size_bytes"`
		TotalSizeHuman string         `json:"total_size_human"`
		FilesByType    map[string]int `json:"files_by_type"`
		FilesByStatus  map[string]int `json:"files_by_status"`
		FilesByUser    map[string]int `json:"files_by_user"`
		ExtractedCount int            `json:"extracted_count"`
		PendingCount   int            `json:"pending_count"`
		FailedCount    int            `json:"failed_count"`
	}{
		FilesByType:   make(map[string]int),
		FilesByStatus: make(map[string]int),
		FilesByUser:   make(map[string]int),
	}

	for _, file := range allFiles {
		stats.TotalFiles++
		stats.TotalSize += file.SizeBytes

		// По типу
		stats.FilesByType[file.MimeType]++

		// По статусу
		stats.FilesByStatus[file.ExtractionStatus]++
		if file.ExtractionStatus == "completed" {
			stats.ExtractedCount++
		} else if file.ExtractionStatus == "pending" {
			stats.PendingCount++
		} else if file.ExtractionStatus == "failed" {
			stats.FailedCount++
		}

		// По пользователю
		stats.FilesByUser[file.UserID]++
	}

	// Форматируем размер
	stats.TotalSizeHuman = formatBytes(stats.TotalSize)

	c.JSON(http.StatusOK, stats)
}

// formatBytes форматирует размер в человекочитаемый формат
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return strconv.FormatInt(bytes, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return strconv.FormatFloat(float64(bytes)/float64(div), 'f', 1, 64) + " " + units[exp]
}


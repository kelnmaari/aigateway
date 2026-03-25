package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/extractors"
	"aigateway/internal/filestorage"
	"aigateway/internal/models"
	"aigateway/internal/storage"
	"aigateway/internal/utils"
)

// FileHandler обрабатывает HTTP запросы для работы с файлами (v1.10.0+)
type FileHandler struct {
	fileService       *filestorage.Service
	extractorRegistry *extractors.Registry
	db                storage.Database
	logger            *logrus.Logger
	wsBroadcaster     FileWSBroadcaster // WS-01 v1.10.2: WebSocket events
}

// FileWSBroadcaster интерфейс для file processing WebSocket events
type FileWSBroadcaster interface {
	BroadcastFileUploadStart(fileID, filename string, size int64) error
	BroadcastFileUploadProgress(fileID string, bytesUploaded, totalBytes int64, percent float64) error
	BroadcastFileUploadComplete(fileID, filename string, downloadURL string) error
	BroadcastFileUploadError(fileID, filename string, errorMsg string) error
	BroadcastFileProcessingStart(fileID, filename string, processingType string) error
	BroadcastFileProcessingProgress(fileID string, stage string, percent float64) error
	BroadcastFileProcessingComplete(fileID string, result map[string]any) error
	BroadcastFileProcessingError(fileID string, errorMsg string) error
}

// NewFileHandler создает новый file handler
func NewFileHandler(
	fileService *filestorage.Service,
	extractorRegistry *extractors.Registry,
	db storage.Database,
	logger *logrus.Logger,
) *FileHandler {
	return &FileHandler{
		fileService:       fileService,
		extractorRegistry: extractorRegistry,
		db:                db,
		logger:            logger,
		wsBroadcaster:     nil, // Optional WebSocket support
	}
}

// SetWSBroadcaster устанавливает WebSocket broadcaster (WS-01 v1.10.2)
func (h *FileHandler) SetWSBroadcaster(broadcaster FileWSBroadcaster) {
	h.wsBroadcaster = broadcaster
}

// UploadFile обрабатывает загрузку файла
// POST /api/files/upload
func (h *FileHandler) UploadFile(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Получаем файл из multipart form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		h.logger.WithError(err).Error("Failed to get file from form")
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file provided"})
		return
	}
	defer file.Close()

	// Опциональные параметры
	description := c.PostForm("description")
	isPublic := c.PostForm("is_public") == "true"
	extract := c.PostForm("extract") != "false" // По умолчанию true

	// Получаем tenant_id если есть
	var tenantID *string
	tenantIDStr := c.PostForm("tenant_id")
	if tenantIDStr != "" {
		tenantID = &tenantIDStr
	}

	// Определяем правильный MIME-type по расширению файла
	mimeType := detectMimeType(header.Filename, header.Header.Get("Content-Type"))

	// Загружаем файл в storage
	// NOTE: Не валидируем содержимое (validateContent=false) т.к. multipart.File не поддерживает Seek
	// и чтение для валидации сдвинет указатель, обрезав начало файла
	uploadResp, err := h.fileService.Upload(c.Request.Context(), filestorage.UploadRequest{
		Reader:                file,
		Filename:              header.Filename,
		MimeType:              mimeType,
		Size:                  header.Size,
		UserID:                userID,
		TenantID:              tenantIDStr,
		Public:                isPublic,
		Extract:               extract,
		SkipContentValidation: true, // Пропускаем валидацию содержимого для HTTP uploads
		Metadata: map[string]any{
			"description": description,
		},
	})
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"user_id":  userID,
			"filename": header.Filename,
			"size":     header.Size,
		}).Error("Failed to upload file")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file", "details": err.Error()})
		return
	}

	// Создаем запись в БД
	var extractedText *string
	var pageCount, wordCount *int
	var language *string

	// Извлекаем текст если требуется
	extractionStatus := "pending"
	if extract {
		h.logger.WithFields(logrus.Fields{
			"mime_type": mimeType,
			"filename":  header.Filename,
			"supports":  h.extractorRegistry.Supports(mimeType),
		}).Debug("Checking extraction support")

		if h.extractorRegistry.Supports(mimeType) {
			// Открываем файл для extraction
			reader, err := h.fileService.Download(c.Request.Context(), uploadResp.Path, userID)
			if err == nil {
				defer reader.Close()

				doc, err := h.extractorRegistry.Extract(c.Request.Context(), reader, extractors.ExtractOptions{
					Filename: header.Filename,
					MimeType: mimeType,
				})
				if err == nil {
					extractedText = &doc.Text
					pageCount = &doc.Metadata.PageCount
					wordCount = &doc.WordCount
					language = &doc.Language
					extractionStatus = "completed"

					h.logger.WithFields(logrus.Fields{
						"filename":   header.Filename,
						"word_count": doc.WordCount,
						"language":   doc.Language,
					}).Info("Text extraction completed")
				} else {
					extractionStatus = "failed"
					h.logger.WithError(err).Warn("Failed to extract text from file")
				}
			} else {
				extractionStatus = "failed"
				h.logger.WithError(err).Error("Failed to open file for extraction")
			}
		} else {
			extractionStatus = "unsupported"
			h.logger.WithField("mime_type", mimeType).Warn("Extractor not found for MIME type")
		}
	}

	// Сохраняем метаданные в БД
	dbFile, err := h.db.CreateFile(c.Request.Context(), models.CreateFileRequest{
		UserID:           userID,
		TenantID:         tenantID,
		Filename:         header.Filename,
		OriginalFilename: header.Filename,
		MimeType:         mimeType,
		SizeBytes:        header.Size,
		ChecksumSHA256:   &uploadResp.Checksum,
		StorageBackend:   uploadResp.StorageType,
		StoragePath:      uploadResp.Path,
		ExtractedText:    extractedText,
		ExtractionStatus: &extractionStatus,
		PageCount:        pageCount,
		WordCount:        wordCount,
		Language:         language,
		Metadata: &models.FileMetadata{
			PageCount: utils.Value(pageCount, 0),
		},
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to save file metadata to database")
		// Пытаемся удалить файл из storage
		_ = h.fileService.Delete(c.Request.Context(), uploadResp.Path, userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file metadata"})
		return
	}

	// Логируем доступ
	_ = h.db.LogFileAccess(c.Request.Context(), models.FileAccessLog{
		FileID:    dbFile.ID,
		UserID:    &userID,
		Action:    "upload",
		IPAddress: new(c.ClientIP()),
		UserAgent: new(c.Request.UserAgent()),
	})

	// WS-01 v1.10.2: Уведомляем о завершении загрузки и обработки
	if h.wsBroadcaster != nil {
		downloadURL := fmt.Sprintf("/api/files/%s/download", dbFile.ID)
		if err := h.wsBroadcaster.BroadcastFileUploadComplete(dbFile.ID, header.Filename, downloadURL); err != nil {
			h.logger.WithError(err).Debug("Failed to broadcast upload complete")
		}

		// Если файл обрабатывался, отправляем результат
		if extract && extractionStatus == "completed" {
			processingResult := map[string]any{
				"extracted_text_length": len(utils.Value(extractedText, "")),
				"word_count":            utils.Value(wordCount, 0),
				"language":              utils.Value(language, ""),
				"page_count":            utils.Value(pageCount, 0),
			}
			if err := h.wsBroadcaster.BroadcastFileProcessingComplete(dbFile.ID, processingResult); err != nil {
				h.logger.WithError(err).Debug("Failed to broadcast processing complete")
			}
		} else if extract {
			// Если extraction не удалось
			if err := h.wsBroadcaster.BroadcastFileProcessingError(dbFile.ID, "extraction failed or not supported"); err != nil {
				h.logger.WithError(err).Debug("Failed to broadcast processing error")
			}
		}
	}

	h.logger.WithFields(logrus.Fields{
		"file_id":  dbFile.ID,
		"user_id":  userID,
		"filename": header.Filename,
		"size":     header.Size,
	}).Info("File uploaded successfully")

	c.JSON(http.StatusCreated, gin.H{
		"id":                dbFile.ID,
		"filename":          dbFile.Filename,
		"mime_type":         dbFile.MimeType,
		"size_bytes":        dbFile.SizeBytes,
		"storage_backend":   dbFile.StorageBackend,
		"extraction_status": dbFile.ExtractionStatus,
		"metadata":          dbFile.Metadata,
		"download_url":      fmt.Sprintf("/api/files/%s/download", dbFile.ID),
		"created_at":        dbFile.CreatedAt,
	})
}

// GetFile получает информацию о файле
// GET /api/files/:id
func (h *FileHandler) GetFile(c *gin.Context) {
	userID := c.GetString("user_id")
	fileID := c.Param("id")

	file, err := h.db.GetFileByID(c.Request.Context(), fileID)
	if err != nil {
		h.logger.WithError(err).WithField("file_id", fileID).Error("File not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// Проверяем доступ
	if file.UserID != userID && !file.IsPublic {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, file)
}

// DownloadFile скачивает файл
// GET /api/files/:id/download
func (h *FileHandler) DownloadFile(c *gin.Context) {
	userID := c.GetString("user_id")
	fileID := c.Param("id")

	file, err := h.db.GetFileByID(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// Проверяем доступ
	if file.UserID != userID && !file.IsPublic {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Получаем файл из storage
	reader, err := h.fileService.Download(c.Request.Context(), file.StoragePath, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to download file from storage")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download file"})
		return
	}
	defer reader.Close()

	// Увеличиваем счетчик скачиваний
	_ = h.db.IncrementDownloadCount(c.Request.Context(), fileID)

	// Логируем доступ
	_ = h.db.LogFileAccess(c.Request.Context(), models.FileAccessLog{
		FileID:    fileID,
		UserID:    &userID,
		Action:    "download",
		IPAddress: new(c.ClientIP()),
		UserAgent: new(c.Request.UserAgent()),
	})

	// Устанавливаем заголовки для скачивания
	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", file.Filename))
	c.Header("Content-Length", strconv.FormatInt(file.SizeBytes, 10))

	// Копируем содержимое файла в response
	if _, err := io.Copy(c.Writer, reader); err != nil {
		h.logger.WithError(err).Error("Failed to stream file content")
	}
}

// GetFileText получает извлеченный текст файла
// GET /api/files/:id/text
func (h *FileHandler) GetFileText(c *gin.Context) {
	userID := c.GetString("user_id")
	fileID := c.Param("id")

	file, err := h.db.GetFileByID(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// Проверяем доступ
	if file.UserID != userID && !file.IsPublic {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Проверяем статус extraction
	if file.ExtractionStatus == "pending" {
		c.JSON(http.StatusAccepted, gin.H{
			"status":  "pending",
			"message": "text extraction is in progress",
		})
		return
	}

	if file.ExtractionStatus == "failed" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "failed",
			"error":  file.ExtractionError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"file_id":    fileID,
		"text":       file.ExtractedText,
		"page_count": file.PageCount,
		"word_count": file.WordCount,
		"language":   file.Language,
	})
}

// ListFiles получает список файлов пользователя
// GET /api/files
func (h *FileHandler) ListFiles(c *gin.Context) {
	userID := c.GetString("user_id")

	// Параметры запроса
	mimeType := c.Query("mime_type")
	extractionStatus := c.Query("extraction_status")
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")
	sortBy := c.DefaultQuery("sort", "created_at")
	order := c.DefaultQuery("order", "desc")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	req := models.ListFilesRequest{
		UserID: &userID,
		Limit:  limit,
		Offset: offset,
		SortBy: sortBy,
		Order:  order,
	}

	if mimeType != "" {
		req.MimeType = &mimeType
	}
	if extractionStatus != "" {
		req.ExtractionStatus = &extractionStatus
	}

	files, total, err := h.db.ListFiles(c.Request.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list files")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list files"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"files":  files,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// DeleteFile удаляет файл
// DELETE /api/files/:id
func (h *FileHandler) DeleteFile(c *gin.Context) {
	userID := c.GetString("user_id")
	fileID := c.Param("id")

	file, err := h.db.GetFileByID(c.Request.Context(), fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// Проверяем доступ (только владелец может удалить)
	if file.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Удаляем файл из storage
	if err := h.fileService.Delete(c.Request.Context(), file.StoragePath, userID); err != nil {
		h.logger.WithError(err).Error("Failed to delete file from storage")
		// Продолжаем, чтобы удалить из БД даже если storage failed
	}

	// Удаляем запись из БД (soft delete)
	if err := h.db.DeleteFile(c.Request.Context(), fileID); err != nil {
		h.logger.WithError(err).Error("Failed to delete file from database")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete file"})
		return
	}

	// Логируем доступ
	_ = h.db.LogFileAccess(c.Request.Context(), models.FileAccessLog{
		FileID:    fileID,
		UserID:    &userID,
		Action:    "delete",
		IPAddress: new(c.ClientIP()),
		UserAgent: new(c.Request.UserAgent()),
	})

	h.logger.WithFields(logrus.Fields{
		"file_id": fileID,
		"user_id": userID,
	}).Info("File deleted successfully")

	c.Status(http.StatusNoContent)
}

// SearchFiles ищет файлы по содержимому (для будущей реализации с full-text search)
// POST /api/files/search
func (h *FileHandler) SearchFiles(c *gin.Context) {
	_ = c.GetString("user_id") // Для будущего использования

	var req struct {
		Query     string   `json:"query"`
		MimeTypes []string `json:"mime_types"`
		DateFrom  string   `json:"date_from"`
		DateTo    string   `json:"date_to"`
		Tags      []string `json:"tags"`
		Limit     int      `json:"limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// TODO: Implement full-text search using SQLite FTS5 or PostgreSQL ts_vector
	// For now, return simple substring search

	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "search not yet implemented",
		"message": "full-text search will be implemented in future version",
	})
}

// Helper functions

// detectMimeType определяет MIME-тип по расширению файла
// detectProcessingType определяет тип обработки файла (WS-01 v1.10.2)
func detectProcessingType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	processingTypes := map[string]string{
		".pdf":  "pdf_extract",
		".docx": "docx_extract",
		".doc":  "doc_extract",
		".csv":  "csv_parse",
		".xlsx": "xlsx_parse",
		".txt":  "text_extract",
		".md":   "markdown_parse",
		".rtf":  "rtf_extract",
		".png":  "ocr",
		".jpg":  "ocr",
		".jpeg": "ocr",
		".gif":  "ocr",
		".webp": "ocr",
		".bmp":  "ocr",
		".tiff": "ocr",
	}

	if procType, ok := processingTypes[ext]; ok {
		return procType
	}

	return "unknown"
}

// detectMimeType определяет MIME-тип по расширению файла
func detectMimeType(filename, browserMimeType string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	// Маппинг расширений на MIME-типы
	mimeTypes := map[string]string{
		".txt":  "text/plain",
		".md":   "text/markdown",
		".rtf":  "text/rtf",
		".csv":  "text/csv",
		".pdf":  "application/pdf",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".doc":  "application/msword",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".xls":  "application/vnd.ms-excel",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".ppt":  "application/vnd.ms-powerpoint",
		".odt":  "application/vnd.oasis.opendocument.text",
		".ods":  "application/vnd.oasis.opendocument.spreadsheet",
	}

	// Если нашли по расширению - используем его
	if mimeType, ok := mimeTypes[ext]; ok {
		return mimeType
	}

	// Иначе используем то что прислал браузер
	if browserMimeType != "" && browserMimeType != "application/octet-stream" {
		return browserMimeType
	}

	// Fallback на текст/plain для неизвестных файлов
	return "application/octet-stream"
}

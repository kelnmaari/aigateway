package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/filestorage"
	"aigateway/internal/imageproc"
	"aigateway/internal/models"
	"aigateway/internal/storage"
	"aigateway/internal/vision"
)

// ImageHandler обрабатывает операции с изображениями (v1.10.3)
type ImageHandler struct {
	fileService   *filestorage.Service
	imageProc     *imageproc.ImageProcessor
	ocrEngine     vision.OCREngine
	db            storage.Database
	logger        *logrus.Logger
	wsBroadcaster FileWSBroadcaster
}

// NewImageHandler создает новый image handler
func NewImageHandler(
	fileService *filestorage.Service,
	imageProc *imageproc.ImageProcessor,
	ocrEngine vision.OCREngine,
	db storage.Database,
	logger *logrus.Logger,
) *ImageHandler {
	return &ImageHandler{
		fileService:   fileService,
		imageProc:     imageProc,
		ocrEngine:     ocrEngine,
		db:            db,
		logger:        logger,
		wsBroadcaster: nil,
	}
}

// SetWSBroadcaster устанавливает WebSocket broadcaster
func (h *ImageHandler) SetWSBroadcaster(broadcaster FileWSBroadcaster) {
	h.wsBroadcaster = broadcaster
}

// UploadImage загружает изображение с OCR обработкой
// POST /api/images/upload
func (h *ImageHandler) UploadImage(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Получаем файл из multipart form
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		h.logger.WithError(err).Error("Failed to get image from form")
		c.JSON(http.StatusBadRequest, gin.H{"error": "no image provided"})
		return
	}
	defer file.Close()

	// Опциональные параметры
	description := c.PostForm("description")
	performOCR := c.PostForm("ocr") != "false" // По умолчанию true
	generateThumb := c.PostForm("thumbnail") != "false"
	ocrModel := c.DefaultPostForm("ocr_model", "llava:7b")

	// Читаем изображение в память для валидации
	imageData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read image"})
		return
	}

	// Валидация изображения
	if err := h.imageProc.ValidateImage(bytes.NewReader(imageData)); err != nil {
		h.logger.WithError(err).Warn("Invalid image upload attempt")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image format", "details": err.Error()})
		return
	}

	// Получаем метаданные изображения
	imageInfo, err := h.imageProc.GetImageInfo(bytes.NewReader(imageData))
	if err != nil {
		h.logger.WithError(err).Error("Failed to get image info")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process image"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"filename": header.Filename,
		"width":    imageInfo.Width,
		"height":   imageInfo.Height,
		"format":   imageInfo.Format,
		"size_kb":  imageInfo.SizeBytes / 1024,
	}).Info("Image upload started")

	// Определяем MIME type
	mimeType := fmt.Sprintf("image/%s", strings.ToLower(imageInfo.Format))

	// Загружаем оригинал в storage
	uploadResp, err := h.fileService.Upload(c.Request.Context(), filestorage.UploadRequest{
		Reader:                bytes.NewReader(imageData),
		Filename:              header.Filename,
		MimeType:              mimeType,
		Size:                  imageInfo.SizeBytes,
		UserID:                userID,
		Public:                false,
		Extract:               false, // Для изображений используем OCR, не text extraction
		SkipContentValidation: true,
		Metadata: map[string]any{
			"description": description,
			"width":       imageInfo.Width,
			"height":      imageInfo.Height,
			"format":      imageInfo.Format,
		},
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to upload image to storage")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload image"})
		return
	}

	// WebSocket: upload complete notification
	if h.wsBroadcaster != nil {
		tempFileID := fmt.Sprintf("img_%d", time.Now().UnixNano())
		downloadURL := fmt.Sprintf("/api/images/%s", tempFileID)
		if err := h.wsBroadcaster.BroadcastFileUploadComplete(tempFileID, header.Filename, downloadURL); err != nil {
			h.logger.WithError(err).Debug("Failed to broadcast image upload complete")
		}
	}

	// Генерируем thumbnail если требуется
	var thumbnailPath *string
	if generateThumb {
		thumbOpts := imageproc.DefaultThumbnailOptions()
		thumbReader, err := h.imageProc.GenerateThumbnail(bytes.NewReader(imageData), thumbOpts)
		if err == nil {
			// Загружаем thumbnail
			thumbFilename := "thumb_" + header.Filename
			thumbResp, err := h.fileService.Upload(c.Request.Context(), filestorage.UploadRequest{
				Reader:                thumbReader,
				Filename:              thumbFilename,
				MimeType:              "image/jpeg",
				Size:                  0, // Size будет вычислен автоматически
				UserID:                userID,
				Public:                false,
				SkipContentValidation: true,
			})
			if err == nil {
				thumbnailPath = &thumbResp.Path
				h.logger.WithField("thumb_path", thumbResp.Path).Debug("Thumbnail generated")
			} else {
				h.logger.WithError(err).Warn("Failed to upload thumbnail")
			}
		} else {
			h.logger.WithError(err).Warn("Failed to generate thumbnail")
		}
	}

	// Выполняем OCR если требуется
	var ocrText *string
	var ocrLanguage *string
	var ocrConfidence *float64
	if performOCR {
		// WebSocket: processing start
		if h.wsBroadcaster != nil {
			tempFileID := fmt.Sprintf("img_%d", time.Now().UnixNano())
			if err := h.wsBroadcaster.BroadcastFileProcessingStart(tempFileID, header.Filename, "ocr"); err != nil {
				h.logger.WithError(err).Debug("Failed to broadcast OCR start")
			}
		}

		ocrOpts := vision.DefaultOCROptions()
		ocrOpts.Model = ocrModel

		ocrResult, err := h.ocrEngine.ExtractText(c.Request.Context(), bytes.NewReader(imageData), ocrOpts)
		if err == nil {
			ocrText = &ocrResult.Text
			ocrLanguage = &ocrResult.Language
			ocrConfidence = &ocrResult.Confidence

			h.logger.WithFields(logrus.Fields{
				"ocr_text_length": len(ocrResult.Text),
				"language":        ocrResult.Language,
				"confidence":      ocrResult.Confidence,
			}).Info("OCR completed")

			// WebSocket: processing complete
			if h.wsBroadcaster != nil {
				tempFileID := fmt.Sprintf("img_%d", time.Now().UnixNano())
				result := map[string]any{
					"text_length": len(ocrResult.Text),
					"language":    ocrResult.Language,
					"confidence":  ocrResult.Confidence,
				}
				if err := h.wsBroadcaster.BroadcastFileProcessingComplete(tempFileID, result); err != nil {
					h.logger.WithError(err).Debug("Failed to broadcast OCR complete")
				}
			}
		} else {
			h.logger.WithError(err).Warn("OCR failed")
			// WebSocket: processing error
			if h.wsBroadcaster != nil {
				tempFileID := fmt.Sprintf("img_%d", time.Now().UnixNano())
				if err := h.wsBroadcaster.BroadcastFileProcessingError(tempFileID, err.Error()); err != nil {
					h.logger.WithError(err).Debug("Failed to broadcast OCR error")
				}
			}
		}
	}

	// Создаем запись в БД
	dbFile, err := h.db.CreateFile(c.Request.Context(), models.CreateFileRequest{
		UserID:           userID,
		Filename:         header.Filename,
		OriginalFilename: header.Filename,
		MimeType:         mimeType,
		SizeBytes:        imageInfo.SizeBytes,
		ChecksumSHA256:   &uploadResp.Checksum,
		StorageBackend:   uploadResp.StorageType,
		StoragePath:      uploadResp.Path,
		ExtractedText:    ocrText,
		ExtractionStatus: new("completed"),
		Language:         ocrLanguage,
		Metadata: &models.FileMetadata{
			PageCount: 0, // Для изображений не применимо
		},
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to save image metadata to database")
		// Пытаемся удалить файл из storage
		_ = h.fileService.Delete(c.Request.Context(), uploadResp.Path, userID)
		if thumbnailPath != nil {
			_ = h.fileService.Delete(c.Request.Context(), *thumbnailPath, userID)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save image metadata"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"file_id":   dbFile.ID,
		"user_id":   userID,
		"filename":  header.Filename,
		"has_ocr":   ocrText != nil,
		"has_thumb": thumbnailPath != nil,
	}).Info("Image uploaded successfully")

	c.JSON(http.StatusCreated, gin.H{
		"id":             dbFile.ID,
		"filename":       dbFile.Filename,
		"mime_type":      dbFile.MimeType,
		"size_bytes":     dbFile.SizeBytes,
		"width":          imageInfo.Width,
		"height":         imageInfo.Height,
		"format":         imageInfo.Format,
		"ocr_text":       ocrText,
		"ocr_language":   ocrLanguage,
		"ocr_confidence": ocrConfidence,
		"thumbnail_url":  getThumbnailURL(dbFile.ID, thumbnailPath),
		"download_url":   fmt.Sprintf("/api/images/%s/download", dbFile.ID),
		"created_at":     dbFile.CreatedAt,
	})
}

// GetImage получает информацию об изображении
// GET /api/images/:id
func (h *ImageHandler) GetImage(c *gin.Context) {
	userID := c.GetString("user_id")
	imageID := c.Param("id")

	file, err := h.db.GetFileByID(c.Request.Context(), imageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	// Проверка прав доступа (пока без Public)
	if file.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             file.ID,
		"filename":       file.Filename,
		"mime_type":      file.MimeType,
		"size_bytes":     file.SizeBytes,
		"metadata":       file.Metadata,
		"extracted_text": file.ExtractedText,
		"language":       file.Language,
		"download_url":   fmt.Sprintf("/api/images/%s/download", file.ID),
		"created_at":     file.CreatedAt,
	})
}

// DownloadImage скачивает оригинал изображения
// GET /api/images/:id/download
func (h *ImageHandler) DownloadImage(c *gin.Context) {
	userID := c.GetString("user_id")
	imageID := c.Param("id")

	file, err := h.db.GetFileByID(c.Request.Context(), imageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	// Проверка прав доступа (пока без Public)
	if file.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Скачиваем из storage
	reader, err := h.fileService.Download(c.Request.Context(), file.StoragePath, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to download image")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download image"})
		return
	}
	defer reader.Close()

	// Устанавливаем заголовки
	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", file.Filename))

	// Копируем данные
	io.Copy(c.Writer, reader)
}

// GetThumbnail возвращает thumbnail изображения
// GET /api/images/:id/thumbnail
func (h *ImageHandler) GetThumbnail(c *gin.Context) {
	userID := c.GetString("user_id")
	imageID := c.Param("id")

	file, err := h.db.GetFileByID(c.Request.Context(), imageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found"})
		return
	}

	// Проверка прав доступа (пока без Public)
	if file.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// TODO: Добавить поддержку thumbnail в БД
	// Пока что генерируем thumbnail on-the-fly из оригинала
	originalReader, err := h.fileService.Download(c.Request.Context(), file.StoragePath, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to download original image")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to download image"})
		return
	}
	defer originalReader.Close()

	// Генерируем thumbnail on-the-fly
	thumbOpts := imageproc.DefaultThumbnailOptions()
	thumbReader, err := h.imageProc.GenerateThumbnail(originalReader, thumbOpts)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate thumbnail")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate thumbnail"})
		return
	}

	// Устанавливаем заголовки
	c.Header("Content-Type", "image/jpeg")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=thumb_%s", file.Filename))

	// Копируем данные
	io.Copy(c.Writer, thumbReader.(io.Reader))
}

// getThumbnailURL формирует URL для thumbnail
func getThumbnailURL(fileID string, thumbnailPath *string) *string {
	if thumbnailPath == nil {
		return nil
	}
	url := fmt.Sprintf("/api/images/%s/thumbnail", fileID)
	return &url
}

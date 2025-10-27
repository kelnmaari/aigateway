// Package handlers provides backup and restore handlers
package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
	auditService "aigateway/internal/services/audit"
	"aigateway/internal/storage"
)

// BackupHandler обрабатывает backup и restore операции
type BackupHandler struct {
	config      *config.Config
	logger      *logrus.Logger
	db          storage.Database
	backupDir   string
	maxBackups  int // Максимальное количество хранимых бэкапов
	auditLogger *auditService.AuditLogger
}

// NewBackupHandler создает новый backup handler
func NewBackupHandler(cfg *config.Config, logger *logrus.Logger, db storage.Database, auditLogger *auditService.AuditLogger) *BackupHandler {
	// Определяем директорию для бэкапов
	backupDir := "./backups"
	if cfg.Database.SQLite.Path != "" {
		// Размещаем бэкапы рядом с основной БД
		dbDir := filepath.Dir(cfg.Database.SQLite.Path)
		backupDir = filepath.Join(dbDir, "backups")
	}

	// Создаем директорию если не существует
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		logger.WithError(err).Error("Failed to create backup directory")
	}

	return &BackupHandler{
		config:      cfg,
		logger:      logger,
		db:          db,
		backupDir:   backupDir,
		maxBackups:  10, // Храним последние 10 бэкапов
		auditLogger: auditLogger,
	}
}

// CreateBackup создает backup базы данных
// POST /api/admin/backup
func (h *BackupHandler) CreateBackup(c *gin.Context) {
	h.logger.Info("Creating database backup")

	// Генерируем имя файла с timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupFilename := fmt.Sprintf("backup_%s.db", timestamp)
	backupPath := filepath.Join(h.backupDir, backupFilename)

	// Создаем backup
	if err := h.performBackup(backupPath); err != nil {
		h.logger.WithError(err).Error("Failed to create backup")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create backup",
		})
		return
	}

	// Получаем информацию о созданном файле
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		h.logger.WithError(err).Error("Failed to stat backup file")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Backup created but failed to get file info",
		})
		return
	}

	// Очищаем старые бэкапы
	h.cleanupOldBackups()

	h.logger.WithFields(logrus.Fields{
		"filename": backupFilename,
		"size":     fileInfo.Size(),
	}).Info("Backup created successfully")

	// Audit log: Backup created (CRITICAL)
	userID, _ := c.Get("user_id")
	if h.auditLogger != nil && userID != nil {
		_ = h.auditLogger.LogBackupCreated(c.Request.Context(), userID.(string), backupFilename, c.ClientIP())
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Backup created successfully",
		"filename": backupFilename,
		"size":     fileInfo.Size(),
		"created":  fileInfo.ModTime(),
		"path":     backupPath,
	})
}

// ListBackups возвращает список доступных бэкапов
// GET /api/admin/backups
func (h *BackupHandler) ListBackups(c *gin.Context) {
	h.logger.Info("Listing available backups")

	// Читаем директорию с бэкапами
	files, err := os.ReadDir(h.backupDir)
	if err != nil {
		h.logger.WithError(err).Error("Failed to read backup directory")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list backups",
		})
		return
	}

	// Собираем информацию о бэкапах
	backups := make([]gin.H, 0)
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".db") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			h.logger.WithError(err).WithField("filename", file.Name()).Warn("Failed to get file info")
			continue
		}

		backups = append(backups, gin.H{
			"filename": file.Name(),
			"size":     info.Size(),
			"created":  info.ModTime(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"backups": backups,
		"count":   len(backups),
		"dir":     h.backupDir,
	})
}

// DownloadBackup скачивает backup файл
// GET /api/admin/backup/:filename
func (h *BackupHandler) DownloadBackup(c *gin.Context) {
	filename := c.Param("filename")

	// Проверка безопасности: filename не должен содержать path separators
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid filename",
		})
		return
	}

	backupPath := filepath.Join(h.backupDir, filename)

	// Проверяем существование файла
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Backup not found",
		})
		return
	}

	h.logger.WithField("filename", filename).Info("Downloading backup file")

	// Отправляем файл
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/octet-stream")
	c.File(backupPath)
}

// RestoreBackup восстанавливает базу данных из backup
// POST /api/admin/restore/:filename
func (h *BackupHandler) RestoreBackup(c *gin.Context) {
	filename := c.Param("filename")

	// Проверка безопасности
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid filename",
		})
		return
	}

	backupPath := filepath.Join(h.backupDir, filename)

	// Проверяем существование файла
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Backup not found",
		})
		return
	}

	h.logger.WithField("filename", filename).Warn("Starting database restore")

	// Выполняем restore
	if err := h.performRestore(backupPath); err != nil {
		h.logger.WithError(err).Error("Failed to restore backup")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to restore backup",
		})
		return
	}

	h.logger.WithField("filename", filename).Info("Backup restored successfully")

	// Audit log: Backup restored (CRITICAL - data loss risk!)
	userID, _ := c.Get("user_id")
	if h.auditLogger != nil && userID != nil {
		_ = h.auditLogger.LogBackupRestored(c.Request.Context(), userID.(string), filename, c.ClientIP())
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Backup restored successfully. Please restart the server for changes to take effect.",
		"filename": filename,
	})
}

// DeleteBackup удаляет backup файл
// DELETE /api/admin/backup/:filename
func (h *BackupHandler) DeleteBackup(c *gin.Context) {
	filename := c.Param("filename")

	// Проверка безопасности
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid filename",
		})
		return
	}

	backupPath := filepath.Join(h.backupDir, filename)

	// Удаляем файл
	if err := os.Remove(backupPath); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Backup not found",
			})
			return
		}

		h.logger.WithError(err).Error("Failed to delete backup")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete backup",
		})
		return
	}

	h.logger.WithField("filename", filename).Info("Backup deleted successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Backup deleted successfully",
	})
}

// performBackup выполняет резервное копирование базы данных
func (h *BackupHandler) performBackup(destPath string) error {
	// Получаем путь к текущей БД
	dbPath := h.config.Database.SQLite.Path
	if dbPath == "" {
		return fmt.Errorf("database path not configured")
	}

	// Открываем исходный файл
	sourceFile, err := os.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer sourceFile.Close()

	// Создаем файл бэкапа
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer destFile.Close()

	// Копируем данные
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		os.Remove(destPath) // Удаляем неполный бэкап
		return fmt.Errorf("failed to copy database: %w", err)
	}

	// Синхронизируем с диском
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync backup file: %w", err)
	}

	return nil
}

// performRestore восстанавливает базу данных из backup
func (h *BackupHandler) performRestore(backupPath string) error {
	dbPath := h.config.Database.SQLite.Path
	if dbPath == "" {
		return fmt.Errorf("database path not configured")
	}

	// Создаем временный backup текущей БД перед restore
	tempBackup := dbPath + ".before_restore"
	if err := h.copyFile(dbPath, tempBackup); err != nil {
		return fmt.Errorf("failed to create safety backup: %w", err)
	}

	// Восстанавливаем из бэкапа
	if err := h.copyFile(backupPath, dbPath); err != nil {
		// Откатываем изменения
		h.copyFile(tempBackup, dbPath)
		os.Remove(tempBackup)
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	// Удаляем временный backup после успешного restore
	os.Remove(tempBackup)

	return nil
}

// copyFile копирует файл
func (h *BackupHandler) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return destFile.Sync()
}

// cleanupOldBackups удаляет старые бэкапы, оставляя только последние N
func (h *BackupHandler) cleanupOldBackups() {
	files, err := os.ReadDir(h.backupDir)
	if err != nil {
		h.logger.WithError(err).Error("Failed to read backup directory for cleanup")
		return
	}

	// Собираем файлы бэкапов с их временем модификации
	type backupFile struct {
		name    string
		modTime time.Time
	}

	backups := make([]backupFile, 0)
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".db") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		backups = append(backups, backupFile{
			name:    file.Name(),
			modTime: info.ModTime(),
		})
	}

	// Если бэкапов меньше или равно лимиту, ничего не делаем
	if len(backups) <= h.maxBackups {
		return
	}

	// Сортируем по времени (старые первыми)
	for i := 0; i < len(backups)-1; i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].modTime.After(backups[j].modTime) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	// Удаляем самые старые
	deleteCount := len(backups) - h.maxBackups
	for i := 0; i < deleteCount; i++ {
		backupPath := filepath.Join(h.backupDir, backups[i].name)
		if err := os.Remove(backupPath); err != nil {
			h.logger.WithError(err).WithField("filename", backups[i].name).Warn("Failed to delete old backup")
		} else {
			h.logger.WithField("filename", backups[i].name).Info("Deleted old backup")
		}
	}
}


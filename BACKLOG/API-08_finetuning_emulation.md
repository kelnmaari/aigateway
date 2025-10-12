# API-08: Fine-tuning API Emulation

**Версия:** v1.7.1  
**Приоритет:** LOW  
**Оценка времени:** 8-10 часов  
**Статус:** 📋 Не начато  

---

## 📋 Описание

Эмуляция OpenAI Fine-tuning API для совместимости с существующими клиентами и инструментами. Поскольку Ollama не поддерживает настоящий fine-tuning, эта реализация будет **симулировать** процесс fine-tuning для совместимости.

## 🎯 Цели

1. **OpenAI API Compatibility** - полная совместимость с fine-tuning endpoints
2. **Job Tracking** - отслеживание статуса "fine-tuning" задач
3. **Model Aliasing** - создание алиасов для "fine-tuned" моделей
4. **File Upload** - загрузка training файлов (для совместимости)

## 🔧 Технические детали

### 1. Fine-tuning Models

**Файл:** `internal/models/finetuning.go`

```go
package models

import "time"

// FineTuningJob представляет задачу fine-tuning
type FineTuningJob struct {
	ID                string                 `json:"id" db:"id"`                         // ft-xxx
	Object            string                 `json:"object" db:"object"`                 // "fine_tuning.job"
	Model             string                 `json:"model" db:"model"`                   // Базовая модель (llama2, etc.)
	CreatedAt         int64                  `json:"created_at" db:"created_at"`         // Unix timestamp
	FinishedAt        *int64                 `json:"finished_at,omitempty" db:"finished_at"`
	FineTunedModel    *string                `json:"fine_tuned_model,omitempty" db:"fine_tuned_model"` // ft:llama2:suffix
	OrganizationID    string                 `json:"organization_id" db:"organization_id"`
	ResultFiles       []string               `json:"result_files" db:"result_files"`
	Status            FineTuningStatus       `json:"status" db:"status"` // queued, running, succeeded, failed, cancelled
	ValidationFile    *string                `json:"validation_file,omitempty" db:"validation_file"`
	TrainingFile      string                 `json:"training_file" db:"training_file"`
	Hyperparameters   FineTuningHyperparams  `json:"hyperparameters" db:"hyperparameters"`
	TrainedTokens     *int                   `json:"trained_tokens,omitempty" db:"trained_tokens"`
	Error             *FineTuningError       `json:"error,omitempty" db:"error"`
	UserID            string                 `json:"-" db:"user_id"`
	TenantID          *string                `json:"-" db:"tenant_id"`
}

type FineTuningStatus string

const (
	FineTuningStatusQueued     FineTuningStatus = "queued"
	FineTuningStatusValidating FineTuningStatus = "validating_files"
	FineTuningStatusRunning    FineTuningStatus = "running"
	FineTuningStatusSucceeded  FineTuningStatus = "succeeded"
	FineTuningStatusFailed     FineTuningStatus = "failed"
	FineTuningStatusCancelled  FineTuningStatus = "cancelled"
)

type FineTuningHyperparams struct {
	NEpochs          int     `json:"n_epochs" db:"n_epochs"`               // Default: auto
	BatchSize        *int    `json:"batch_size,omitempty" db:"batch_size"` // Default: auto
	LearningRateMultiplier *float64 `json:"learning_rate_multiplier,omitempty" db:"learning_rate_multiplier"`
}

type FineTuningError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Param   string `json:"param,omitempty"`
}

// FineTuningEvent представляет событие в процессе fine-tuning
type FineTuningEvent struct {
	ID        string    `json:"id"`
	Object    string    `json:"object"` // "fine_tuning.job.event"
	CreatedAt int64     `json:"created_at"`
	Level     string    `json:"level"` // info, warning, error
	Message   string    `json:"message"`
	Data      *EventData `json:"data,omitempty"`
	Type      string    `json:"type"` // message, metrics
}

type EventData struct {
	Step              int     `json:"step,omitempty"`
	TrainLoss         float64 `json:"train_loss,omitempty"`
	TrainAccuracy     float64 `json:"train_accuracy,omitempty"`
	ValidLoss         float64 `json:"valid_loss,omitempty"`
	ValidMeanTokenAccuracy float64 `json:"valid_mean_token_accuracy,omitempty"`
}

// UploadedFile представляет загруженный файл
type UploadedFile struct {
	ID        string  `json:"id" db:"id"`           // file-xxx
	Object    string  `json:"object" db:"object"`   // "file"
	Bytes     int64   `json:"bytes" db:"bytes"`     // Размер файла
	CreatedAt int64   `json:"created_at" db:"created_at"`
	Filename  string  `json:"filename" db:"filename"`
	Purpose   string  `json:"purpose" db:"purpose"` // "fine-tune", "fine-tune-results"
	Status    string  `json:"status" db:"status"`   // "uploaded", "processed", "error"
	UserID    string  `json:"-" db:"user_id"`
	TenantID  *string `json:"-" db:"tenant_id"`
	FilePath  string  `json:"-" db:"file_path"` // Путь на диске
}
```

### 2. Fine-tuning Handlers

**Файл:** `internal/api/handlers/finetuning.go`

```go
package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	
	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

type FineTuningHandler struct {
	db     storage.Database
	logger *logrus.Logger
}

func NewFineTuningHandler(db storage.Database, logger *logrus.Logger) *FineTuningHandler {
	return &FineTuningHandler{
		db:     db,
		logger: logger,
	}
}

// CreateFineTuningJob создает новую задачу fine-tuning (симуляция)
// POST /v1/fine_tuning/jobs
func (h *FineTuningHandler) CreateFineTuningJob(c *gin.Context) {
	var req struct {
		Model           string                       `json:"model" binding:"required"`
		TrainingFile    string                       `json:"training_file" binding:"required"`
		ValidationFile  *string                      `json:"validation_file,omitempty"`
		Hyperparameters *models.FineTuningHyperparams `json:"hyperparameters,omitempty"`
		Suffix          *string                      `json:"suffix,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user from context
	userID, _ := c.Get("user_id")

	// Generate fine-tuned model name
	suffix := "custom"
	if req.Suffix != nil {
		suffix = *req.Suffix
	}
	fineTunedModel := fmt.Sprintf("ft:%s:%s:%s", req.Model, userID, suffix)

	// Create job
	job := &models.FineTuningJob{
		ID:             fmt.Sprintf("ftjob-%s", uuid.New().String()),
		Object:         "fine_tuning.job",
		Model:          req.Model,
		CreatedAt:      time.Now().Unix(),
		Status:         models.FineTuningStatusQueued,
		TrainingFile:   req.TrainingFile,
		ValidationFile: req.ValidationFile,
		FineTunedModel: &fineTunedModel,
		UserID:         userID.(string),
		Hyperparameters: models.FineTuningHyperparams{
			NEpochs: 4, // Default
		},
	}

	if req.Hyperparameters != nil {
		job.Hyperparameters = *req.Hyperparameters
	}

	// Save to database
	if err := h.db.CreateFineTuningJob(c.Request.Context(), job); err != nil {
		h.logger.WithError(err).Error("Failed to create fine-tuning job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Start simulation in background
	go h.simulateFineTuning(job.ID)

	c.JSON(http.StatusOK, job)
}

// ListFineTuningJobs возвращает список fine-tuning задач
// GET /v1/fine_tuning/jobs
func (h *FineTuningHandler) ListFineTuningJobs(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	jobs, err := h.db.ListFineTuningJobs(c.Request.Context(), userID.(string))
	if err != nil {
		h.logger.WithError(err).Error("Failed to list fine-tuning jobs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   jobs,
	})
}

// GetFineTuningJob возвращает информацию о задаче
// GET /v1/fine_tuning/jobs/:job_id
func (h *FineTuningHandler) GetFineTuningJob(c *gin.Context) {
	jobID := c.Param("job_id")
	
	job, err := h.db.GetFineTuningJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// CancelFineTuningJob отменяет задачу
// POST /v1/fine_tuning/jobs/:job_id/cancel
func (h *FineTuningHandler) CancelFineTuningJob(c *gin.Context) {
	jobID := c.Param("job_id")
	
	job, err := h.db.GetFineTuningJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Only queued or running jobs can be cancelled
	if job.Status != models.FineTuningStatusQueued && job.Status != models.FineTuningStatusRunning {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Job cannot be cancelled"})
		return
	}

	job.Status = models.FineTuningStatusCancelled
	if err := h.db.UpdateFineTuningJob(c.Request.Context(), job); err != nil {
		h.logger.WithError(err).Error("Failed to cancel job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, job)
}

// simulateFineTuning симулирует процесс fine-tuning
func (h *FineTuningHandler) simulateFineTuning(jobID string) {
	ctx := context.Background()

	// Wait a bit (queued state)
	time.Sleep(5 * time.Second)

	// Update to running
	job, _ := h.db.GetFineTuningJob(ctx, jobID)
	if job.Status == models.FineTuningStatusCancelled {
		return
	}

	job.Status = models.FineTuningStatusRunning
	h.db.UpdateFineTuningJob(ctx, job)

	// Simulate training (30 seconds)
	time.Sleep(30 * time.Second)

	// Check if cancelled
	job, _ = h.db.GetFineTuningJob(ctx, jobID)
	if job.Status == models.FineTuningStatusCancelled {
		return
	}

	// Update to succeeded
	now := time.Now().Unix()
	job.Status = models.FineTuningStatusSucceeded
	job.FinishedAt = &now
	trainedTokens := 1000000 // Fake number
	job.TrainedTokens = &trainedTokens

	h.db.UpdateFineTuningJob(ctx, job)

	// Create model alias in database
	// Fine-tuned model теперь доступен через обычный chat completions API
	h.logger.WithField("model", *job.FineTunedModel).Info("Fine-tuning completed (simulated)")
}
```

### 3. Database Schema

```sql
-- Fine-tuning jobs table
CREATE TABLE IF NOT EXISTS fine_tuning_jobs (
    id TEXT PRIMARY KEY,
    object TEXT NOT NULL DEFAULT 'fine_tuning.job',
    model TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    finished_at INTEGER,
    fine_tuned_model TEXT,
    organization_id TEXT,
    status TEXT NOT NULL,
    validation_file TEXT,
    training_file TEXT NOT NULL,
    n_epochs INTEGER DEFAULT 4,
    batch_size INTEGER,
    learning_rate_multiplier REAL,
    trained_tokens INTEGER,
    error_message TEXT,
    error_code TEXT,
    user_id TEXT NOT NULL,
    tenant_id TEXT,
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE INDEX idx_fine_tuning_jobs_user ON fine_tuning_jobs(user_id);
CREATE INDEX idx_fine_tuning_jobs_status ON fine_tuning_jobs(status);
CREATE INDEX idx_fine_tuning_jobs_created ON fine_tuning_jobs(created_at DESC);

-- Uploaded files table
CREATE TABLE IF NOT EXISTS uploaded_files (
    id TEXT PRIMARY KEY,
    object TEXT NOT NULL DEFAULT 'file',
    bytes INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    filename TEXT NOT NULL,
    purpose TEXT NOT NULL,
    status TEXT NOT NULL,
    file_path TEXT NOT NULL,
    user_id TEXT NOT NULL,
    tenant_id TEXT,
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE INDEX idx_uploaded_files_user ON uploaded_files(user_id);
CREATE INDEX idx_uploaded_files_purpose ON uploaded_files(purpose);
```

### 4. Router Integration

```go
// internal/api/router/router.go

// Fine-tuning endpoints (OpenAI compatibility)
fineTuning := r.engine.Group("/v1/fine_tuning/jobs")
fineTuning.Use(middleware.AuthMiddleware(r.authService))
{
    fineTuning.POST("", r.fineTuningHandler.CreateFineTuningJob)
    fineTuning.GET("", r.fineTuningHandler.ListFineTuningJobs)
    fineTuning.GET("/:job_id", r.fineTuningHandler.GetFineTuningJob)
    fineTuning.POST("/:job_id/cancel", r.fineTuningHandler.CancelFineTuningJob)
    fineTuning.GET("/:job_id/events", r.fineTuningHandler.ListFineTuningEvents)
}

// Files API (для загрузки training files)
files := r.engine.Group("/v1/files")
files.Use(middleware.AuthMiddleware(r.authService))
{
    files.POST("", r.filesHandler.UploadFile)
    files.GET("", r.filesHandler.ListFiles)
    files.GET("/:file_id", r.filesHandler.GetFile)
    files.DELETE("/:file_id", r.filesHandler.DeleteFile)
    files.GET("/:file_id/content", r.filesHandler.GetFileContent)
}
```

## 🧪 Тестирование

### Unit Tests

```go
func TestFineTuningHandler_CreateJob(t *testing.T) {
	db := &MockDatabase{}
	handler := NewFineTuningHandler(db, logrus.New())
	
	req := CreateFineTuningRequest{
		Model:        "llama2",
		TrainingFile: "file-abc123",
		Suffix:       strPtr("custom"),
	}
	
	// Test job creation
	job, err := handler.CreateJob(context.Background(), req, "user-123")
	require.NoError(t, err)
	assert.Equal(t, "llama2", job.Model)
	assert.Equal(t, models.FineTuningStatusQueued, job.Status)
	assert.Contains(t, *job.FineTunedModel, "ft:llama2")
}

func TestFineTuningSimulation(t *testing.T) {
	// Test that job progresses through states
	// queued -> running -> succeeded
}
```

### Integration Tests

```bash
# Create fine-tuning job
curl -X POST http://localhost:8080/v1/fine_tuning/jobs \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama2",
    "training_file": "file-abc123",
    "suffix": "my-model"
  }'

# List jobs
curl http://localhost:8080/v1/fine_tuning/jobs \
  -H "Authorization: Bearer sk-xxx"

# Get specific job
curl http://localhost:8080/v1/fine_tuning/jobs/ftjob-xxx \
  -H "Authorization: Bearer sk-xxx"

# Cancel job
curl -X POST http://localhost:8080/v1/fine_tuning/jobs/ftjob-xxx/cancel \
  -H "Authorization: Bearer sk-xxx"

# Use fine-tuned model (после completion)
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "ft:llama2:user123:my-model",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## 📁 Структура файлов

```
internal/
├── models/
│   └── finetuning.go          # Fine-tuning models
├── api/
│   └── handlers/
│       ├── finetuning.go      # Fine-tuning handlers
│       ├── finetuning_test.go
│       ├── files.go           # File upload handlers
│       └── files_test.go
├── storage/
│   ├── database.go            # Add fine-tuning methods to interface
│   ├── sqlite/
│   │   └── finetuning.go      # SQLite implementation
│   └── postgresql/
│       └── finetuning.go      # PostgreSQL implementation
```

## 📝 Конфигурация

```yaml
# configs/dev.yaml
fine_tuning:
  enabled: true
  max_jobs_per_user: 10
  upload_dir: "./data/uploads"
  max_file_size_mb: 512
  allowed_formats: ["jsonl", "json"]
  simulation_duration_seconds: 30  # Для тестирования
```

## 🎯 Критерии успеха

- [ ] OpenAI fine-tuning API endpoints работают
- [ ] Job creation, listing, retrieval, cancellation
- [ ] File upload API функционирует
- [ ] Simulation процесса fine-tuning
- [ ] Model aliasing для fine-tuned моделей
- [ ] Database migrations для новых таблиц
- [ ] Unit tests покрывают >90%
- [ ] Integration tests с реальными запросами
- [ ] Документация обновлена

## 📖 Документация

**Обновить:**
- `docs/API_DOCUMENTATION.md` - Fine-tuning API section
- `README.md` - упомянуть fine-tuning emulation
- Добавить примеры использования

## ⚠️ Важные заметки

1. **Это эмуляция, не настоящий fine-tuning**
   - Ollama не поддерживает fine-tuning
   - "Fine-tuned" модели - это алиасы базовых моделей
   - Для совместимости с OpenAI клиентами

2. **Model Aliasing**
   - `ft:llama2:user123:custom` → `llama2`
   - Просто routing к базовой модели

3. **File Storage**
   - Файлы сохраняются на диске (не используются)
   - Для полной совместимости API

## 💡 Future Enhancements

- Реальная интеграция с Ollama adapters (если появится)
- LoRA fine-tuning через внешние инструменты
- Training metrics visualization в WebUI
- Export fine-tuned models
- Integration с HuggingFace для реального fine-tuning

---

**Оценка времени:** 8-10 часов  
**Сложность:** MEDIUM  
**Зависимости:** Нет  
**Блокирует:** Нет


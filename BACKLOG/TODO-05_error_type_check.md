# TODO-05: Error Type Checking

**Версия:** v1.5.4  
**Приоритет:** LOW  
**Оценка:** 1-2 часа  
**Зависимости:** Нет  
**Статус:** 📋 Planned  
**Тип:** Code Quality / Error Handling

---

## 📋 Описание

Улучшить обработку ошибок с проверкой типа ошибки вместо использования `errors.Is()` или строковых сравнений.

## 📊 Current Issue

**Файл:** `internal/api/handlers/admin.go` (строка 792)

```go
func (h *AdminHandlers) handleSomeError(err error) {
    // TODO: Реализовать проверку типа ошибки
    
    // Текущий код (плохо):
    if err != nil {
        if err.Error() == "not found" {  // Строковое сравнение ❌
            // handle not found
        }
    }
}
```

## 🔧 Solution: Typed Errors

### 1. Define Custom Error Types

**Файл:** `internal/errors/errors.go` (новый)

```go
package errors

import (
    "errors"
    "fmt"
)

// Custom error types
var (
    ErrNotFound          = errors.New("resource not found")
    ErrUnauthorized      = errors.New("unauthorized")
    ErrForbidden         = errors.New("forbidden")
    ErrInvalidInput      = errors.New("invalid input")
    ErrAlreadyExists     = errors.New("resource already exists")
    ErrInternalServer    = errors.New("internal server error")
    ErrDatabaseError     = errors.New("database error")
    ErrValidationFailed  = errors.New("validation failed")
)

// NotFoundError для ресурсов не найденных
type NotFoundError struct {
    Resource string
    ID       string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

// ValidationError для ошибок валидации
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}

// DatabaseError для ошибок БД
type DatabaseError struct {
    Operation string
    Err       error
}

func (e *DatabaseError) Error() string {
    return fmt.Sprintf("database error during %s: %v", e.Operation, e.Err)
}

func (e *DatabaseError) Unwrap() error {
    return e.Err
}

// Helper functions
func NewNotFoundError(resource, id string) error {
    return &NotFoundError{Resource: resource, ID: id}
}

func NewValidationError(field, message string) error {
    return &ValidationError{Field: field, Message: message}
}

func NewDatabaseError(operation string, err error) error {
    return &DatabaseError{Operation: operation, Err: err}
}

// Type checking helpers
func IsNotFound(err error) bool {
    var notFoundErr *NotFoundError
    return errors.As(err, &notFoundErr) || errors.Is(err, ErrNotFound)
}

func IsValidationError(err error) bool {
    var validationErr *ValidationError
    return errors.As(err, &validationErr) || errors.Is(err, ErrValidationFailed)
}

func IsDatabaseError(err error) bool {
    var dbErr *DatabaseError
    return errors.As(err, &dbErr) || errors.Is(err, ErrDatabaseError)
}
```

### 2. Update Admin Handlers

**Файл:** `internal/api/handlers/admin.go`

**До (плохо):**

```go
func (h *AdminHandlers) GetModel(c *gin.Context) {
    modelName := c.Param("name")
    
    model, err := h.storage.GetModel(c.Request.Context(), modelName)
    if err != nil {
        // TODO: Реализовать проверку типа ошибки
        if err.Error() == "not found" {  // ❌ Строковое сравнение
            c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
        return
    }
    
    c.JSON(http.StatusOK, model)
}
```

**После (хорошо):**

```go
import (
    "your-project/internal/errors"
)

func (h *AdminHandlers) GetModel(c *gin.Context) {
    modelName := c.Param("name")
    
    model, err := h.storage.GetModel(c.Request.Context(), modelName)
    if err != nil {
        // Typed error checking ✅
        if errors.IsNotFound(err) {
            c.JSON(http.StatusNotFound, gin.H{
                "error": "Model not found",
                "model": modelName,
            })
            return
        }
        
        if errors.IsDatabaseError(err) {
            h.logger.WithError(err).Error("Database error getting model")
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": "Database error",
            })
            return
        }
        
        // Generic error
        h.logger.WithError(err).Error("Failed to get model")
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Internal server error",
        })
        return
    }
    
    c.JSON(http.StatusOK, model)
}
```

### 3. Update Storage Layer

**Файл:** `internal/storage/sqlite/models.go`

**До:**

```go
func (s *SQLiteStorage) GetModel(ctx context.Context, name string) (*Model, error) {
    var model Model
    err := s.db.GetContext(ctx, &model, "SELECT * FROM models WHERE name = ?", name)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("not found") // ❌ Generic string
    }
    if err != nil {
        return nil, err
    }
    return &model, nil
}
```

**После:**

```go
import (
    "your-project/internal/errors"
)

func (s *SQLiteStorage) GetModel(ctx context.Context, name string) (*Model, error) {
    var model Model
    err := s.db.GetContext(ctx, &model, "SELECT * FROM models WHERE name = ?", name)
    if err == sql.ErrNoRows {
        return nil, errors.NewNotFoundError("model", name) // ✅ Typed error
    }
    if err != nil {
        return nil, errors.NewDatabaseError("get model", err) // ✅ Typed error
    }
    return &model, nil
}
```

### 4. Error Response Helper

**Файл:** `internal/api/handlers/error_handler.go` (новый)

```go
package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "your-project/internal/errors"
)

// HandleError обрабатывает ошибки и возвращает соответствующий HTTP response
func HandleError(c *gin.Context, err error, logger Logger) {
    if err == nil {
        return
    }
    
    // Not found errors -> 404
    if errors.IsNotFound(err) {
        c.JSON(http.StatusNotFound, gin.H{
            "error": err.Error(),
        })
        return
    }
    
    // Validation errors -> 400
    if errors.IsValidationError(err) {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": err.Error(),
        })
        return
    }
    
    // Database errors -> 500 (with logging)
    if errors.IsDatabaseError(err) {
        logger.WithError(err).Error("Database error")
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Internal server error",
        })
        return
    }
    
    // Check for standard errors
    if errors.Is(err, errors.ErrUnauthorized) {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error": "Unauthorized",
        })
        return
    }
    
    if errors.Is(err, errors.ErrForbidden) {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "Forbidden",
        })
        return
    }
    
    if errors.Is(err, errors.ErrAlreadyExists) {
        c.JSON(http.StatusConflict, gin.H{
            "error": err.Error(),
        })
        return
    }
    
    // Default: internal server error
    logger.WithError(err).Error("Unhandled error")
    c.JSON(http.StatusInternalServerError, gin.H{
        "error": "Internal server error",
    })
}

// Usage example:
func (h *AdminHandlers) GetModel(c *gin.Context) {
    modelName := c.Param("name")
    
    model, err := h.storage.GetModel(c.Request.Context(), modelName)
    if err != nil {
        HandleError(c, err, h.logger)
        return
    }
    
    c.JSON(http.StatusOK, model)
}
```

## 📝 Implementation Plan

### Step 1: Create Error Package (30 мин)

- [ ] Создать `internal/errors/errors.go`
- [ ] Определить custom error types
- [ ] Создать helper functions

### Step 2: Update Storage Layer (30 мин)

- [ ] Обновить storage методы
- [ ] Заменить generic errors на typed
- [ ] Тестирование

### Step 3: Update Handlers (30 мин)

- [ ] Обновить admin.go
- [ ] Создать error handler helper
- [ ] Удалить TODO

### Step 4: Testing (30 мин)

- [ ] Unit tests для error types
- [ ] Integration tests
- [ ] Error handling tests

## ✅ Acceptance Criteria

- [ ] Custom error types определены
- [ ] Storage layer использует typed errors
- [ ] Handlers используют errors.Is() и errors.As()
- [ ] Error handler helper создан
- [ ] TODO комментарий удален
- [ ] Нет строковых сравнений ошибок
- [ ] Unit tests покрывают error handling
- [ ] Documentation обновлена

## 🧪 Testing

### Unit Tests

```go
func TestErrorTypes(t *testing.T) {
    // Test NotFoundError
    err := errors.NewNotFoundError("model", "llama3")
    assert.True(t, errors.IsNotFound(err))
    assert.Contains(t, err.Error(), "model not found: llama3")
    
    // Test ValidationError
    err = errors.NewValidationError("email", "invalid format")
    assert.True(t, errors.IsValidationError(err))
    
    // Test wrapping
    baseErr := fmt.Errorf("connection failed")
    dbErr := errors.NewDatabaseError("insert user", baseErr)
    assert.True(t, errors.IsDatabaseError(dbErr))
    assert.True(t, errors.Is(dbErr, baseErr))
}

func TestHandleError(t *testing.T) {
    // Test 404 response
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    
    err := errors.NewNotFoundError("user", "123")
    HandleError(c, err, logger)
    
    assert.Equal(t, http.StatusNotFound, w.Code)
    assert.Contains(t, w.Body.String(), "not found")
}
```

## 📚 References

- [Go errors package](https://pkg.go.dev/errors)
- [Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors)
- [Error handling best practices](https://dave.cheney.net/2016/04/27/dont-just-check-errors-handle-them-gracefully)

## 🔄 Benefits

**До:**

- Хрупкий код (строковые сравнения)
- Сложно поддерживать
- Нет типобезопасности

**После:**

- Type-safe error handling
- Легко поддерживать
- Централизованная обработка
- Лучше logging и debugging

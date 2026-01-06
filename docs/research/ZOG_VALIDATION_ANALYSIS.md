# Zog Validation Library Analysis

> Анализ библиотеки [Zog](https://github.com/Oudwins/zog) для улучшения валидации в проекте.

**Дата анализа:** 2026-01-05  
**Версия Zog:** v0.22.0  
**Лицензия:** MIT  
**Документация:** [zog.dev](https://zog.dev)

## Обзор

Zog — Zod-подобная библиотека валидации для Go:
- Декларативные схемы с method chaining
- Zero dependencies
- Быстрая (близка к goplayground/validator)
- Встроенная coercion (приведение типов)
- Rich errors с контекстом
- i18n поддержка

## Установка

```bash
go get github.com/Oudwins/zog
```

## Ключевые концепции

### 1. Базовые схемы

```go
import z "github.com/Oudwins/zog"

// Примитивы
z.String().Min(3).Max(100).Required()
z.Int().GTE(0).LTE(1000)
z.Bool()
z.Float64().Positive()
z.Time().After(time.Now())

// Структуры
z.Struct(z.Shape{
    "name": z.String().Required(),
    "age":  z.Int().GT(0),
})

// Слайсы
z.Slice(z.String().Email())

// Опциональность
z.String().Optional()  // по умолчанию все optional
z.String().Required()  // делает обязательным
```

### 2. Parse vs Validate

```go
// Parse: из map/json/form в struct (с coercion)
var user User
errs := userSchema.Parse(inputMap, &user)

// Validate: проверка существующего struct
user := User{Name: "John", Age: 25}
errs := userSchema.Validate(&user)
```

### 3. Helper Packages

```go
// HTTP Forms & Query Params
import zhttp "github.com/Oudwins/zog/zhttp"
errs := schema.Parse(zhttp.Request(r), &data)

// JSON
import zjson "github.com/Oudwins/zog/zjson"
errs := schema.Parse(zjson.Decode(reader), &data)

// Environment Variables
import zenv "github.com/Oudwins/zog/zenv"
errs := schema.Parse(zenv.NewDataProvider(), &config)
```

### 4. Transformations

```go
// Preprocess - трансформация до валидации
schema := z.Preprocess(func(data any, ctx z.Ctx) ([]string, error) {
    s := data.(string)
    return strings.Split(s, ","), nil
}, z.Slice(z.String().Email()))

// Trim, ToLower, etc.
z.String().Trim().ToLower().Email()
```

### 5. Custom Validators

```go
z.String().Test(z.TestFunc("custom", func(val string, ctx z.Ctx) bool {
    return strings.HasPrefix(val, "sk-")
}), z.Message("must start with sk-"))
```

### 6. Error Handling

```go
errs := schema.Parse(input, &dest)
if errs != nil {
    for field, issues := range errs {
        for _, issue := range issues {
            fmt.Printf("%s: %s\n", field, issue.Message)
        }
    }
}
```

## План внедрения в проект

### Phase 1: Новые endpoints (сразу)

Использовать Zog для всех новых API handlers:

```go
// internal/api/schemas/apikey.go
package schemas

import z "github.com/Oudwins/zog"

var CreateAPIKey = z.Struct(z.Shape{
    "name":        z.String().Min(1).Max(255).Required(),
    "description": z.String().Max(1000).Optional(),
    "models":      z.Slice(z.String()).Optional(),
    "permissions": z.Slice(z.String()).Optional(),
    "rate_limits": z.Struct(z.Shape{
        "requests_per_minute": z.Int().GTE(0).LTE(10000).Optional(),
        "requests_per_hour":   z.Int().GTE(0).Optional(),
        "requests_per_day":    z.Int().GTE(0).Optional(),
    }).Optional(),
    "expires_at": z.Time().After(time.Now()).Optional(),
})

var UpdateAPIKey = z.Struct(z.Shape{
    "name":        z.String().Min(1).Max(255).Optional(),
    "description": z.String().Max(1000).Optional(),
    "status":      z.String().OneOf([]string{"active", "inactive", "revoked"}).Optional(),
})
```

### Phase 2: Config validation

```go
// internal/config/schema.go
package config

import z "github.com/Oudwins/zog"

var ConfigSchema = z.Struct(z.Shape{
    "server": z.Struct(z.Shape{
        "port":          z.Int().GTE(1).LTE(65535).Default(8080),
        "host":          z.String().Default("0.0.0.0"),
        "read_timeout":  z.String().Default("30s"),
        "write_timeout": z.String().Default("30s"),
    }),
    "database": z.Struct(z.Shape{
        "driver": z.String().OneOf([]string{"sqlite", "postgres"}).Required(),
        "dsn":    z.String().Required(),
    }),
    "auth": z.Struct(z.Shape{
        "jwt_secret":     z.String().Min(32).Required(),
        "token_duration": z.String().Default("24h"),
    }),
    "ollama": z.Struct(z.Shape{
        "base_url": z.String().URL().Default("http://localhost:11434"),
        "timeout":  z.String().Default("30s"),
    }),
})
```

### Phase 3: Рефакторинг существующих handlers

Приоритет по частоте использования:

| Handler | Файл | Приоритет |
|---------|------|-----------|
| Chat Completions | `inference_proxy_handler.go` | Высокий |
| Create API Key | `admin.go` | Высокий |
| User Registration | `auth.go` | Высокий |
| GitLab Project | `gitlab_admin.go` | Средний |
| Model Config | `inference_handler.go` | Средний |

### Phase 4: Environment validation

```go
// internal/config/env_schema.go
import zenv "github.com/Oudwins/zog/zenv"

var EnvSchema = z.Struct(z.Shape{
    "PORT":           z.Int().Default(8080),
    "DATABASE_URL":   z.String().Required(),
    "JWT_SECRET":     z.String().Min(32).Required(),
    "OLLAMA_URL":     z.String().URL().Default("http://localhost:11434"),
    "LOG_LEVEL":      z.String().OneOf([]string{"debug", "info", "warn", "error"}).Default("info"),
    "REDIS_URL":      z.String().Optional(),
    "QDRANT_URL":     z.String().Optional(),
})

func LoadEnv() (*EnvConfig, error) {
    var cfg EnvConfig
    if errs := EnvSchema.Parse(zenv.NewDataProvider(), &cfg); errs != nil {
        return nil, fmt.Errorf("invalid environment: %v", errs)
    }
    return &cfg, nil
}
```

## Примеры для наших типов

### API Key Request

```go
var CreateAPIKeyRequest = z.Struct(z.Shape{
    "name": z.String().
        Min(1, z.Message("Имя обязательно")).
        Max(255, z.Message("Имя слишком длинное")).
        Required(),
    "models": z.Slice(z.String()).
        Optional(),
    "permissions": z.Slice(z.String().
        OneOf([]string{"chat", "embeddings", "models", "admin"})).
        Optional(),
})
```

### GitLab Project

```go
var CreateGitLabProject = z.Struct(z.Shape{
    "integration_id":    z.String().UUID().Required(),
    "gitlab_project_id": z.Int().GT(0).Required(),
    "name":              z.String().Min(1).Max(255).Required(),
    "default_branch":    z.String().Default("main"),
    "embedding_model_id": z.String().Optional(),
})
```

### Model Configuration

```go
var ModelSpec = z.Struct(z.Shape{
    "alias":       z.String().Min(1).Max(100).Required(),
    "format":      z.String().OneOf([]string{"gguf", "safetensors", "pytorch"}).Required(),
    "provider":    z.String().OneOf([]string{"llama.cpp", "vllm", "ollama"}).Required(),
    "hf_repo":     z.String().Optional(),
    "hf_file":     z.String().Optional(),
    "ctx_size":    z.Int().GTE(512).LTE(131072).Default(4096),
    "gpu_layers":  z.Int().GTE(-1).Default(-1),
    "flash_attn":  z.Bool().Default(true),
})
```

## Интеграция с Gin

```go
// internal/api/middleware/validation.go
package middleware

import (
    "github.com/gin-gonic/gin"
    z "github.com/Oudwins/zog"
    zhttp "github.com/Oudwins/zog/zhttp"
)

// ValidateBody validates request body against schema
func ValidateBody[T any](schema z.ZogSchema) gin.HandlerFunc {
    return func(c *gin.Context) {
        var dest T
        if errs := schema.Parse(zhttp.Request(c.Request), &dest); errs != nil {
            c.JSON(400, gin.H{
                "error":   "validation_error",
                "details": formatErrors(errs),
            })
            c.Abort()
            return
        }
        c.Set("validated_body", dest)
        c.Next()
    }
}

// GetValidatedBody retrieves validated body from context
func GetValidatedBody[T any](c *gin.Context) T {
    return c.MustGet("validated_body").(T)
}

func formatErrors(errs z.ZogIssueMap) map[string][]string {
    result := make(map[string][]string)
    for field, issues := range errs {
        for _, issue := range issues {
            result[field] = append(result[field], issue.Message)
        }
    }
    return result
}
```

**Использование:**

```go
router.POST("/api/keys", 
    middleware.ValidateBody[CreateAPIKeyRequest](schemas.CreateAPIKey),
    h.CreateAPIKey,
)

func (h *Handler) CreateAPIKey(c *gin.Context) {
    req := middleware.GetValidatedBody[CreateAPIKeyRequest](c)
    // req уже валиден, можно использовать
}
```

## i18n для ошибок

```go
import zi18n "github.com/Oudwins/zog/i18n"

// Русские сообщения
zi18n.SetLanguage("ru", map[string]string{
    "required":   "Поле обязательно",
    "min_length": "Минимальная длина: {{min}}",
    "max_length": "Максимальная длина: {{max}}",
    "email":      "Некорректный email",
    "url":        "Некорректный URL",
})
```

## Сравнение с текущим кодом

### До (текущий подход)

```go
func (h *Handler) CreateAPIKey(c *gin.Context) {
    var req CreateAPIKeyRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
        return
    }
    
    if req.Name == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
        return
    }
    
    if len(req.Name) > 255 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Name too long"})
        return
    }
    
    for _, perm := range req.Permissions {
        if !isValidPermission(perm) {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid permission: " + perm})
            return
        }
    }
    
    // ... business logic
}
```

### После (с Zog)

```go
func (h *Handler) CreateAPIKey(c *gin.Context) {
    var req CreateAPIKeyRequest
    if errs := schemas.CreateAPIKey.Parse(zhttp.Request(c.Request), &req); errs != nil {
        c.JSON(http.StatusBadRequest, gin.H{"errors": errs})
        return
    }
    
    // req уже валиден, сразу business logic
}
```

## Файловая структура

```
internal/api/
├── schemas/
│   ├── apikey.go       # API Key schemas
│   ├── auth.go         # Auth schemas (login, register)
│   ├── gitlab.go       # GitLab project schemas
│   ├── inference.go    # Model/inference schemas
│   ├── chat.go         # Chat completion schemas
│   └── common.go       # Shared schemas (pagination, etc.)
├── middleware/
│   └── validation.go   # Generic validation middleware
└── handlers/
    └── ...             # Handlers use schemas
```

## Checklist для рефакторинга

- [ ] Добавить `github.com/Oudwins/zog` в go.mod
- [ ] Создать `internal/api/schemas/` директорию
- [ ] Написать базовые схемы для ключевых endpoints
- [ ] Создать validation middleware для Gin
- [ ] Настроить i18n (русский + английский)
- [ ] Мигрировать handlers по приоритету
- [ ] Добавить тесты для схем
- [ ] Обновить документацию API

## Риски

1. **API не стабильно** — версия 0.x, могут быть breaking changes
2. **Время на миграцию** — нужно рефакторить существующие handlers
3. **Learning curve** — команде нужно изучить новый подход

## Альтернативы

- [go-playground/validator](https://github.com/go-playground/validator) — struct tags (более популярно, менее выразительно)
- [ozzo-validation](https://github.com/go-ozzo/ozzo-validation) — похожий подход на Zog
- Текущий подход — ручная валидация (работает, но verbose)

## Ссылки

- [GitHub](https://github.com/Oudwins/zog)
- [Documentation](https://zog.dev)
- [Benchmarks](https://github.com/Oudwins/zog#benchmarks)


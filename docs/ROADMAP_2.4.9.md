# Roadmap v2.4.9: Go 1.18-1.25 Modernization

## Цель релиза

Модернизация кодовой базы с использованием современных возможностей Go 1.18-1.25:
- **Generics** (Go 1.18)
- **slices/maps packages** (Go 1.21)
- **Custom iterators** (Go 1.23)
- **Generic type aliases** (Go 1.24)
- **New stdlib features** (Go 1.25): WaitGroup.Go, FlightRecorder, CrossOriginProtection

**Ожидаемый результат**:
- ✅ Сокращение кода на ~500 строк
- ✅ Улучшение type safety
- ✅ Повышение читаемости
- ✅ Нулевой overhead в runtime

---

## Задачи (Priority Order)

### 🔴 **Priority 1: Generic Pointer Helpers**

**Задача**: Создать `internal/utils/ptr.go` с generic функциями

**Файлы**:
- `internal/utils/ptr.go` (новый)
- Рефакторинг:
  - `internal/api/handlers/file_handler.go` (stringPtr, intValue)
  - `internal/converter/*.go`
  - `internal/models/*.go`
  - ~20+ файлов с дублированием

**API**:
```go
package utils

// Ptr создает pointer из value
func Ptr[T any](v T) *T {
    return &v
}

// Value извлекает value из pointer с fallback
func Value[T any](ptr *T, defaultVal T) T {
    if ptr == nil {
        return defaultVal
    }
    return *ptr
}

// PtrOrNil возвращает pointer или nil если value равен zero value
func PtrOrNil[T comparable](v T) *T {
    var zero T
    if v == zero {
        return nil
    }
    return &v
}
```

**Пример использования**:
```go
// До:
temp := float64Ptr(0.7)
maxTokens := intPtr(2048)
prompt := stringPtr("system prompt")

// После:
temp := utils.Ptr(0.7)
maxTokens := utils.Ptr(2048)
prompt := utils.Ptr("system prompt")

// Value extraction:
actualTemp := utils.Value(config.Temperature, 0.5)
```

**Оценка**:
- Строк удаляется: ~200
- Строк добавляется: ~50
- Файлов затронуто: ~25
- Время: 2-3 часа

---

### 🟠 **Priority 2: slices Package Integration**

**Задача**: Заменить ручные циклы на `slices` package функции

**Файлы для рефакторинга**:
- `internal/models/*.go` (фильтрация моделей)
- `internal/api/handlers/*.go` (поиск в списках)
- `internal/storage/*.go` (работа со слайсами)
- `internal/rag/worker/*.go` (обработка chunks)

**Замены**:

1. **Поиск элемента**:
```go
// До:
func contains(models []string, target string) bool {
    for _, m := range models {
        if m == target {
            return true
        }
    }
    return false
}

// После:
import "slices"

func contains(models []string, target string) bool {
    return slices.Contains(models, target)
}
```

2. **Фильтрация**:
```go
// До:
func filterActive(models []*Model) []*Model {
    var active []*Model
    for _, m := range models {
        if m.Status == StatusActive {
            active = append(active, m)
        }
    }
    return active
}

// После:
func filterActive(models []*Model) []*Model {
    return slices.DeleteFunc(slices.Clone(models), func(m *Model) bool {
        return m.Status != StatusActive
    })
}
```

3. **Индекс элемента**:
```go
// До:
func indexOf(keys []*APIKey, id string) int {
    for i, k := range keys {
        if k.ID == id {
            return i
        }
    }
    return -1
}

// После:
func indexOf(keys []*APIKey, id string) int {
    return slices.IndexFunc(keys, func(k *APIKey) bool {
        return k.ID == id
    })
}
```

**Целевые функции**:
- `slices.Contains` - проверка наличия
- `slices.Index` / `slices.IndexFunc` - поиск индекса
- `slices.DeleteFunc` - фильтрация
- `slices.Sort` / `slices.SortFunc` - сортировка
- `slices.Clone` - копирование
- `slices.Equal` - сравнение

**Оценка**:
- Строк удаляется: ~150
- Файлов затронуто: ~30
- Время: 3-4 часа

---

### 🟠 **Priority 3: maps Package Integration**

**Задача**: Использовать `maps` для безопасных операций с map

**Файлы**:
- `internal/config/config.go` (слияние конфигов)
- `internal/models/*.go` (metadata операции)
- `internal/rag/worker/*.go` (config слияние)

**Замены**:

1. **Копирование map**:
```go
// До:
func copyMetadata(src map[string]interface{}) map[string]interface{} {
    dst := make(map[string]interface{})
    for k, v := range src {
        dst[k] = v
    }
    return dst
}

// После:
import "maps"

func copyMetadata(src map[string]interface{}) map[string]interface{} {
    return maps.Clone(src)
}
```

2. **Слияние config**:
```go
// До:
func mergeConfigs(base, overrides map[string]string) map[string]string {
    result := make(map[string]string)
    for k, v := range base {
        result[k] = v
    }
    for k, v := range overrides {
        result[k] = v
    }
    return result
}

// После:
func mergeConfigs(base, overrides map[string]string) map[string]string {
    result := maps.Clone(base)
    maps.Copy(result, overrides)
    return result
}
```

3. **Получение ключей**:
```go
// До:
func getConfigKeys(cfg map[string]interface{}) []string {
    keys := make([]string, 0, len(cfg))
    for k := range cfg {
        keys = append(keys, k)
    }
    return keys
}

// После:
func getConfigKeys(cfg map[string]interface{}) []string {
    return slices.Collect(maps.Keys(cfg))
}
```

**Целевые функции**:
- `maps.Clone` - копирование
- `maps.Copy` - слияние
- `maps.Keys` - итератор ключей
- `maps.Values` - итератор значений
- `maps.Equal` - сравнение
- `maps.DeleteFunc` - фильтрация

**Оценка**:
- Строк удаляется: ~80
- Файлов затронуто: ~15
- Время: 2 часа

---

### 🟡 **Priority 4: Generic API Response**

**Задача**: Создать type-safe API response wrapper

**Файлы**:
- `internal/api/response/response.go` (новый)
- Рефакторинг всех handlers в `internal/api/handlers/*.go`

**API**:
```go
package response

type Response[T any] struct {
    Data    T                      `json:"data,omitempty"`
    Error   *ErrorDetail           `json:"error,omitempty"`
    Meta    map[string]interface{} `json:"meta,omitempty"`
    Success bool                   `json:"success"`
}

type ErrorDetail struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

// Success создает успешный response
func Success[T any](data T) Response[T] {
    return Response[T]{
        Data:    data,
        Success: true,
    }
}

// SuccessWithMeta создает успешный response с metadata
func SuccessWithMeta[T any](data T, meta map[string]interface{}) Response[T] {
    return Response[T]{
        Data:    data,
        Meta:    meta,
        Success: true,
    }
}

// Error создает error response
func Error[T any](code, message, details string) Response[T] {
    return Response[T]{
        Error: &ErrorDetail{
            Code:    code,
            Message: message,
            Details: details,
        },
        Success: false,
    }
}
```

**Использование**:
```go
// До:
func (h *ModelHandler) ListModels(c *gin.Context) {
    models, err := h.service.ListModels(c.Request.Context())
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"models": models, "count": len(models)})
}

// После:
func (h *ModelHandler) ListModels(c *gin.Context) {
    models, err := h.service.ListModels(c.Request.Context())
    if err != nil {
        c.JSON(500, response.Error[[]*Model]("internal_error", "Failed to list models", err.Error()))
        return
    }
    c.JSON(200, response.SuccessWithMeta(models, map[string]interface{}{"count": len(models)}))
}
```

**Оценка**:
- Строк удаляется: ~120
- Строк добавляется: ~80
- Файлов затронуто: ~40 handlers
- Время: 4-5 часов

---

### 🟢 **Priority 5: Custom Iterators (Go 1.23)**

**Задача**: Реализовать custom iterators для пагинации

**Файлы**:
- `internal/storage/iterator.go` (новый)
- `internal/rag/iterator.go` (новый)

**API**:

1. **Database Pagination**:
```go
package storage

// IterateUsers итерирует по всем пользователям батчами
func (db *PostgreSQLDB) IterateUsers(
    ctx context.Context,
    batchSize int,
) func(func(*models.User) bool) {
    return func(yield func(*models.User) bool) {
        offset := 0
        for {
            users, err := db.ListUsers(ctx, batchSize, offset)
            if err != nil || len(users) == 0 {
                return
            }
            for _, user := range users {
                if !yield(user) {
                    return // Early exit
                }
            }
            offset += batchSize
        }
    }
}

// Использование:
for user := range db.IterateUsers(ctx, 100) {
    if processUser(user) {
        break // Early exit возможен
    }
}
```

2. **RAG Chunk Processing**:
```go
package rag

// IterateChunks итерирует по chunks источника
func (o *Orchestrator) IterateChunks(
    ctx context.Context,
    sourceID string,
    batchSize int,
) func(func(*models.RAGChunk) bool) {
    return func(yield func(*models.RAGChunk) bool) {
        offset := 0
        for {
            chunks, _ := o.db.ListRAGChunksBySource(ctx, sourceID, batchSize, offset)
            if len(chunks) == 0 {
                return
            }
            for _, chunk := range chunks {
                if !yield(chunk) {
                    return
                }
            }
            offset += batchSize
        }
    }
}
```

**Оценка**:
- Строк добавляется: ~100
- Код становится чище: High
- Время: 2-3 часа

---

### 🟢 **Priority 6: Generic Type Aliases (Go 1.24)**

**Задача**: Типизировать config maps и metadata

**Файлы**:
- `internal/types/aliases.go` (новый)
- Рефакторинг config и models

**API**:
```go
package types

// ConfigMap - типизированная конфигурационная map
type ConfigMap[T any] = map[string]T

// StringConfig - строковые настройки
type StringConfig = ConfigMap[string]

// IntConfig - числовые настройки
type IntConfig = ConfigMap[int]

// BoolConfig - булевы настройки
type BoolConfig = ConfigMap[bool]

// Metadata - типизированные метаданные
type Metadata[T any] = map[string]T

// StringMetadata - строковые метаданные
type StringMetadata = Metadata[string]

// NumericMetadata - числовые метаданные
type NumericMetadata = Metadata[float64]
```

**Использование**:
```go
// До:
type AppConfig struct {
    Database map[string]string
    Limits   map[string]int
    Features map[string]bool
}

// После:
type AppConfig struct {
    Database types.StringConfig
    Limits   types.IntConfig
    Features types.BoolConfig
}

// Сразу видно типы значений!
```

**Оценка**:
- Строк добавляется: ~30
- Улучшение clarity: Very High
- Время: 1-2 часа

---

### 🔵 **Priority 7: Generic Vector Operations**

**Задача**: Type-safe векторные операции для RAG

**Файлы**:
- `internal/rag/vector/types.go` (обновить)

**API**:
```go
package vector

// Vector - generic vector для float32/float64
type Vector[T ~float32 | ~float64] struct {
    Data       []T
    Dimensions int
}

// NewVector создает новый вектор
func NewVector[T ~float32 | ~float64](data []T) Vector[T] {
    return Vector[T]{
        Data:       data,
        Dimensions: len(data),
    }
}

// CosineSimilarity вычисляет косинусное сходство
func (v Vector[T]) CosineSimilarity(other Vector[T]) T {
    if v.Dimensions != other.Dimensions {
        return 0
    }
    
    var dotProduct, normA, normB T
    for i := 0; i < v.Dimensions; i++ {
        dotProduct += v.Data[i] * other.Data[i]
        normA += v.Data[i] * v.Data[i]
        normB += other.Data[i] * other.Data[i]
    }
    
    if normA == 0 || normB == 0 {
        return 0
    }
    
    return dotProduct / (T(math.Sqrt(float64(normA))) * T(math.Sqrt(float64(normB))))
}

// L2Distance вычисляет евклидово расстояние
func (v Vector[T]) L2Distance(other Vector[T]) T {
    if v.Dimensions != other.Dimensions {
        return T(math.MaxFloat64)
    }
    
    var sum T
    for i := 0; i < v.Dimensions; i++ {
        diff := v.Data[i] - other.Data[i]
        sum += diff * diff
    }
    
    return T(math.Sqrt(float64(sum)))
}
```

**Оценка**:
- Строк добавляется: ~80
- Type safety: Very High
- Время: 2 часа

---

### 🟣 **Priority 8: sync.WaitGroup.Go() (Go 1.25)**

**Задача**: Упростить pattern запуска goroutines

**Файлы для рефакторинга**:
- `internal/rag/worker/*.go`
- `internal/api/handlers/*.go`
- `internal/services/*.go`

**API**:
```go
// До (Go 1.24):
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    processItem(item)
}()
wg.Wait()

// После (Go 1.25):
var wg sync.WaitGroup
wg.Go(func() {
    processItem(item)
})
wg.Wait()
```

**Оценка**:
- Строк удаляется: ~50
- Code clarity: High
- Время: 1-2 часа

---

### 🟣 **Priority 9: runtime/trace.FlightRecorder (Go 1.25)**

**Задача**: Lightweight tracing для редких событий

**Файлы**:
- `internal/debug/flight_recorder.go` (новый)
- `cmd/server/main.go` (интеграция)

**API**:
```go
package debug

import "runtime/trace"

var globalRecorder *trace.FlightRecorder

func InitFlightRecorder() error {
    var err error
    globalRecorder, err = trace.NewFlightRecorder(trace.FlightRecorderConfig{
        Period: 10 * time.Second,  // Последние 10 секунд
    })
    return err
}

// CapturePanic сохраняет trace при панике
func CapturePanic(panicValue interface{}) {
    if globalRecorder != nil {
        filename := fmt.Sprintf("panic-trace-%d.trace", time.Now().Unix())
        f, _ := os.Create(filename)
        defer f.Close()
        globalRecorder.WriteTo(f)
        log.Printf("Trace captured to %s", filename)
    }
    panic(panicValue)  // Re-panic
}
```

**Использование**:
```go
// В critical sections
defer func() {
    if r := recover(); r != nil {
        debug.CapturePanic(r)
    }
}()
```

**Оценка**:
- Строк добавляется: ~100
- Debug capability: Very High
- Время: 2-3 часа

---

### 🟣 **Priority 10: net/http.CrossOriginProtection (Go 1.25)**

**Задача**: CSRF защита без токенов/cookies

**Файлы**:
- `internal/api/middleware/csrf.go` (новый)
- `internal/api/router/router.go` (интеграция)

**API**:
```go
package middleware

import "net/http"

func CSRF() gin.HandlerFunc {
    protection := http.CrossOriginProtection{
        // Разрешаем запросы с нашего домена
        AllowOrigin: []string{
            "https://aigateway.example.com",
        },
        // Разрешаем безопасные методы без проверки
        SafeMethods: []string{"GET", "HEAD", "OPTIONS"},
    }
    
    return func(c *gin.Context) {
        if err := protection.Check(c.Request); err != nil {
            c.AbortWithStatusJSON(403, gin.H{
                "error": "Cross-origin request rejected",
                "code":  "csrf_protection",
            })
            return
        }
        c.Next()
    }
}
```

**Интеграция**:
```go
// В router.go
r.Use(middleware.CSRF())  // Защита всех endpoints
```

**Оценка**:
- Строк добавляется: ~60
- Security improvement: High
- Время: 1-2 часа

---

### 🟣 **Priority 11: testing.T.Attr() (Go 1.25)**

**Задача**: Метаданные для тестов

**Файлы для обновления**:
- Все `*_test.go` файлы

**API**:
```go
func TestAPIKey(t *testing.T) {
    // Добавляем атрибуты для observability
    t.Attr("category", "security")
    t.Attr("priority", "high")
    t.Attr("go_version", runtime.Version())
    t.Attr("test_type", "unit")
    
    // Тест...
}

// С -json флагом:
// {"Time":"...","Action":"attr","Test":"TestAPIKey","Key":"category","Value":"security"}
```

**Преимущества**:
- Фильтрация тестов по category
- Анализ в CI/CD
- Метрики по test types

**Оценка**:
- Тестов обновляется: ~200
- Observability: High
- Время: 2-3 часа

---

### 🟣 **Priority 12: Experimental GreenTea GC (Go 1.25)**

**Задача**: Измерить GC overhead reduction

**Файлы**:
- `docs/GREENTEA_GC_BENCHMARK.md` (новый)
- `scripts/benchmark_gc.sh` (новый)

**Процесс**:

1. **Baseline benchmark** (текущий GC):
```bash
go test -bench=. -benchmem -benchtime=30s ./... > baseline.txt
```

2. **GreenTea GC benchmark**:
```bash
GOEXPERIMENT=greenteagc go build -o server-greentea cmd/server/main.go
# Run load tests
wrk -t4 -c100 -d60s http://localhost:8080/api/models
```

3. **Сравнение метрик**:
- GC pause time (target: -30%)
- Total GC time (target: -20%)
- Memory overhead
- Throughput

**ВАЖНО**: 
- ⚠️ Экспериментальная фича
- Только для testing/staging
- Не для production (пока)

**Оценка**:
- Research time: 3-4 часа
- Documentation: 1 час

---

## Порядок реализации

### Week 1: Foundation (Priorities 1-3)
- День 1-2: Generic Pointer Helpers
- День 3-4: slices package integration
- День 5: maps package integration

### Week 2: Advanced (Priorities 4-5)
- День 1-3: Generic API Response
- День 4-5: Custom Iterators

### Week 3: Generics & Go 1.25 (Priorities 6-8)
- День 1-2: Generic Type Aliases + Vector Operations
- День 3: sync.WaitGroup.Go() refactoring
- День 4: FlightRecorder + CSRF protection
- День 5: Test attributes + documentation

### Week 4: Performance & Polish (Priorities 9-12)
- День 1-2: GreenTea GC benchmarking
- День 3-4: Code review, final testing
- День 5: CHANGELOG.md, documentation finalization

---

## Критерии успеха

- ✅ Все тесты проходят (go test -race ./...)
- ✅ Компиляция без ошибок
- ✅ Сокращение кода минимум на 400 строк
- ✅ Нет performance regression (benchmarks)
- ✅ Документация обновлена
- ✅ CHANGELOG.md заполнен

---

## Риски и mitigation

| Риск | Вероятность | Impact | Mitigation |
|------|-------------|--------|------------|
| Breaking changes в API | Low | High | Incremental rollout, версионирование |
| Performance regression | Very Low | Medium | Benchmarks перед/после, generics zero-cost |
| Complexity increase | Low | Low | Хорошая документация, примеры |
| Go version requirement | Medium | Low | Уже используем Go 1.25.3 |

---

## Зависимости

- Go 1.21+ (для slices/maps)
- Go 1.23+ (для custom iterators)
- Go 1.24+ (для generic type aliases)
- **Go 1.25+** (для WaitGroup.Go, FlightRecorder, CrossOriginProtection)

**Текущая версия**: Go 1.25.3 ✅

**Официальная документация**:
- [Go 1.25 Release Notes](https://tip.golang.org/doc/go1.25)
- [Go 1.25.1 Issues](https://github.com/golang/go/issues?q=milestone%3AGo1.25.1+label%3ACherryPickApproved)
- [Go 1.25.2 Issues](https://github.com/golang/go/issues?q=milestone%3AGo1.25.2+label%3ACherryPickApproved)
- [Go 1.25.3 Issues](https://github.com/golang/go/issues?q=milestone%3AGo1.25.3+label%3ACherryPickApproved)

---

## После релиза

- 📊 Метрики:
  - Измерить code reduction
  - Benchmark performance
  - Собрать feedback от команды
- 📚 Документация:
  - Обновить development guidelines
  - Добавить примеры в README
  - Создать migration guide
- 🎯 Следующие шаги:
  - v2.5.0: Error handling improvements
  - v2.5.1: Context propagation patterns
  - v2.5.2: Structured logging enhancements


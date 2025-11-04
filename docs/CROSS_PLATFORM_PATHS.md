# Cross-Platform Path Handling in Go

## 🎯 Best Practices для работы с путями

### 1. **embed.FS - всегда forward slashes**

```go
import (
    "embed"
    "path" // НЕ path/filepath!
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// ✅ ПРАВИЛЬНО - path.Join создает forward slashes
content, _ := migrationsFS.ReadFile(path.Join("migrations", "001_schema.sql"))

// ❌ НЕПРАВИЛЬНО - filepath.Join создает OS-специфичные пути
// На Windows: migrations\001_schema.sql → ошибка в embed.FS
content, _ := migrationsFS.ReadFile(filepath.Join("migrations", "001_schema.sql"))
```

**Почему `path`, а не `filepath`?**
- `path` - для URL-подобных путей (всегда `/`)
- `filepath` - для файловой системы OS (на Windows `\`)
- `embed.FS` использует виртуальную FS с forward slashes

### 2. **Реальная файловая система - filepath**

```go
import "path/filepath"

// ✅ ПРАВИЛЬНО - filepath.Join учитывает OS
configPath := filepath.Join("configs", "dev.yaml")
// Windows: configs\dev.yaml
// Linux:   configs/dev.yaml
```

### 3. **Универсальная нормализация путей**

```go
import "path/filepath"

// Конвертирует в forward slashes для embed.FS
func normalizeForEmbed(osPath string) string {
    return filepath.ToSlash(osPath)
}

// Конвертирует в OS-специфичный путь
func normalizeForOS(embedPath string) string {
    return filepath.FromSlash(embedPath)
}
```

## 📊 Comparison Table

| Сценарий | Используй | Пример |
|----------|-----------|--------|
| embed.FS | `path.Join()` | `path.Join("migrations", file)` |
| Реальные файлы | `filepath.Join()` | `filepath.Join("data", "db.sqlite")` |
| URLs | `path.Join()` | `path.Join("/api", endpoint)` |
| Нормализация для embed | `filepath.ToSlash()` | `filepath.ToSlash(osPath)` |
| Нормализация для OS | `filepath.FromSlash()` | `filepath.FromSlash(embedPath)` |

## 🔧 Практические примеры из проекта

### Миграции (embed.FS)

```go
// internal/storage/postgresql/migrations.go
import "path"

func (db *PostgreSQLDB) getMigrations() []migration {
    entries, _ := migrationsFS.ReadDir("migrations")
    
    for _, entry := range entries {
        filename := entry.Name()
        
        // ✅ path.Join для embed.FS
        content, _ := migrationsFS.ReadFile(path.Join("migrations", filename))
    }
}
```

### Конфигурационные файлы (реальная FS)

```go
import "path/filepath"

func loadConfig(env string) (*Config, error) {
    // ✅ filepath.Join для реальных файлов
    configPath := filepath.Join("configs", env+".yaml")
    return readConfig(configPath)
}
```

### Веб-ресурсы (embed.FS)

```go
//go:embed web/static/*
var staticFS embed.FS

http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
    // ✅ path.Join для URL путей
    file := path.Join("web", r.URL.Path[1:])
    data, _ := staticFS.ReadFile(file)
    w.Write(data)
})
```

## 🐛 Типичные ошибки

### ❌ Ошибка 1: filepath.Join с embed.FS

```go
// Windows: migrations\001_schema.sql
// embed.FS: НЕ НАЙДЕТ!
content, _ := migrationsFS.ReadFile(filepath.Join("migrations", "001_schema.sql"))
```

### ❌ Ошибка 2: Хардкод обратных слешей

```go
// Не работает на Linux/macOS
configPath := "configs\dev.yaml"
```

### ❌ Ошибка 3: path.Join для реальных файлов на Windows

```go
// Windows: создаст data/db.sqlite, но система ожидает data\db.sqlite
dbPath := path.Join("data", "db.sqlite") 
os.Open(dbPath) // Может не работать
```

## ✅ Правильные решения

### 1. Всегда используй правильный пакет

```go
import (
    "path"          // Для embed.FS и URLs
    "path/filepath" // Для реальной файловой системы
)
```

### 2. Добавь комментарии

```go
// Use path.Join (not filepath.Join) for embed.FS - always forward slashes
content, err := migrationsFS.ReadFile(path.Join("migrations", filename))

// Use filepath.Join for OS filesystem
configPath := filepath.Join("configs", "dev.yaml")
```

### 3. Создай helper функции

```go
// For embed.FS paths
func embedPath(parts ...string) string {
    return path.Join(parts...)
}

// For OS filesystem
func osPath(parts ...string) string {
    return filepath.Join(parts...)
}
```

## 🎓 Дополнительные ресурсы

- [Go embed package](https://pkg.go.dev/embed)
- [path vs path/filepath](https://stackoverflow.com/questions/43667200/difference-between-path-and-path-filepath)
- [Cross-platform file paths](https://freshman.tech/snippets/go/cross-platform-file-paths/)

## 📝 Checklist для Code Review

- [ ] Используется `path.Join` для embed.FS?
- [ ] Используется `filepath.Join` для реальных файлов?
- [ ] Нет хардкода путей с `\` или `/`?
- [ ] Добавлены комментарии о выборе пакета?
- [ ] Протестировано на Windows и Linux?


# Database-First Configuration - Usage Guide

**Version:** v3.1.0  
**Status:** ✅ Implemented (Gradual Migration Ready)

## Overview

Новая система конфигурации поддерживает **три режима работы**:

1. **Legacy Mode** (v3.0.x) - полный YAML, без БД
2. **Hybrid Mode** (v3.1.0) - YAML + БД overlay, постепенная миграция
3. **Bootstrap Mode** (v3.1.0+) - минимальный YAML + полная БД

---

## 🚀 Quick Start

### Mode 1: Legacy (Backward Compatible)

```go
// Старый код продолжает работать
cfg, err := config.Load("configs/dev.yaml")
if err != nil {
    log.Fatal(err)
}

// Используем напрямую
server := &http.Server{
    Addr: cfg.GetServerAddr(), // cfg.Server.Host:Port
}
```

**Без изменений! Полная обратная совместимость.**

---

### Mode 2: Hybrid (Recommended for Migration)

```go
// 1. Load config with settings manager
cfg, source, err := config.LoadWithSettings(
    "configs/dev.yaml",
    settingsManager,  // *settings.Manager
    logger,
)

// 2. Create wrapper for dual interface
wrapper := config.NewConfigWrapper(cfg, source, logger)

// 3. Use NEW interface (database-first)
ctx := context.Background()
port := wrapper.Source.GetInt(ctx, "server.port")

// 4. Or OLD interface (backward compat)
port := cfg.Server.Port

// 5. Check mode
if wrapper.IsBootstrapMode() {
    log.Info("Running in database-first mode")
}
```

**Преимущества:**
- ✅ Постепенный переход
- ✅ Работает старый код
- ✅ Новый код использует БД
- ✅ Нет breaking changes

---

### Mode 3: Bootstrap (Full Database-First)

**Step 1: Generate bootstrap config**
```bash
./bin/server.exe -generate-bootstrap=configs/bootstrap.yaml
```

**Step 2: Migrate settings to DB**
```bash
./bin/server.exe -migrate-config
```

**Step 3: Use bootstrap mode**
```go
// configs/bootstrap.yaml - только 8 строк!
cfg, source, err := config.LoadWithSettings(
    "configs/bootstrap.yaml",  // Auto-detects bootstrap mode
    settingsManager,
    logger,
)

// MUST use ConfigSource interface
ctx := context.Background()
port, _ := source.GetInt(ctx, "server.port")
host, _ := source.GetString(ctx, "server.host")
```

**Требования:**
- ⚠️ Старый код (`cfg.Server.Port`) НЕ работает в bootstrap mode
- ⚠️ Нужен рефакторинг на `source.GetInt()`
- ✅ Все настройки из БД

---

## 📖 API Reference

### LoadWithMode

```go
type LoadOptions struct {
    ConfigPath      string          // Path to YAML file
    Logger          *logrus.Logger
    SettingsManager SettingsManager // For database-first
    Mode            LoadMode        // Legacy|Hybrid|Bootstrap
}

cfg, source, err := config.LoadWithMode(LoadOptions{
    ConfigPath:      "configs/dev.yaml",
    SettingsManager: settingsManager,
    Logger:          logger,
    Mode:            config.LoadModeHybrid,
})
```

### ConfigWrapper

```go
wrapper := config.NewConfigWrapper(cfg, source, logger)

// New interface (database-first)
wrapper.GetString("server.host")
wrapper.GetInt("server.port")
wrapper.GetBool("metrics.enabled")
wrapper.GetDuration("server.read_timeout")

// Check mode
wrapper.IsBootstrapMode() // true if bootstrap.yaml

// Reload from database
wrapper.Reload()

// Legacy access (backward compat)
wrapper.Server.Port  // Works if not bootstrap mode
```

### ConfigSource Interface

```go
type ConfigSource interface {
    GetString(ctx context.Context, key string) (string, error)
    GetInt(ctx context.Context, key string) (int, error)
    GetBool(ctx context.Context, key string) (bool, error)
    GetDuration(ctx context.Context, key string) (time.Duration, error)
    GetStringSlice(ctx context.Context, key string) ([]string, error)
    
    // WithDefault variants
    GetStringWithDefault(ctx, key, defaultValue string) string
    GetIntWithDefault(ctx, key string, defaultValue int) int
    // ... etc
    
    Reload(ctx context.Context) error
}
```

---

## 🔄 Migration Path

### Phase 1: Current State (v3.0.9)
```go
// All code uses direct access
cfg.Server.Port
cfg.Auth.JWT.Secret
cfg.Database.PostgreSQL.Host
```

### Phase 2: Add Hybrid Support (v3.1.0)
```go
// Load with hybrid mode
cfg, source, _ := config.LoadWithSettings(configPath, settingsManager, logger)

// Old code still works
cfg.Server.Port

// New code uses source
source.GetInt(ctx, "server.port")
```

### Phase 3: Gradual Refactoring
```go
// Refactor critical paths first
// Before:
server := &http.Server{Addr: cfg.GetServerAddr()}

// After:
ctx := context.Background()
host, _ := source.GetString(ctx, "server.host")
port, _ := source.GetInt(ctx, "server.port")
server := &http.Server{Addr: fmt.Sprintf("%s:%d", host, port)}
```

### Phase 4: Switch to Bootstrap
```bash
# Generate minimal config
./bin/server.exe -generate-bootstrap=bootstrap.yaml

# Backup old config
mv configs/dev.yaml configs/dev.yaml.backup

# Use bootstrap
./bin/server.exe -config=configs/bootstrap.yaml
```

---

## 🎯 Configuration Keys

### Server Settings
```
server.host
server.port
server.read_timeout
server.write_timeout
server.idle_timeout
server.max_header_bytes
server.tls.enabled
server.tls.common_name
server.tls.cert_file
server.tls.key_file
```

### Database Settings
```
database.type
database.postgresql.host
database.postgresql.port
database.postgresql.database
database.postgresql.user
database.postgresql.password
database.postgresql.ssl_mode
database.postgresql.max_open_conns
database.postgresql.max_idle_conns
```

### Auth Settings
```
auth.jwt.secret
auth.jwt.access_token_expiry
auth.jwt.refresh_token_expiry
```

### Logging Settings
```
logging.level
logging.format
logging.output
logging.file_path
```

### Metrics Settings
```
metrics.enabled
metrics.prometheus_path
```

### Inference Settings
```
inference.backend
inference.gpu_layers
inference.max_loaded_models
inference.yzma.lib_path
inference.yzma.context_size
inference.yzma.batch_size
inference.yzma.temperature
```

---

## 🔧 Troubleshooting

### Error: "config key not found"
**Причина**: Ключ не найден ни в БД, ни в YAML

**Решение**:
```go
// Use WithDefault variant
port := source.GetIntWithDefault(ctx, "server.port", 8080)
```

### Error: "bootstrap mode requires SettingsManager"
**Причина**: Используется `bootstrap.yaml` без БД подключения

**Решение**:
```go
// Ensure SettingsManager is provided
cfg, source, err := config.LoadWithSettings(
    "bootstrap.yaml",
    settingsManager, // MUST be non-nil
    logger,
)
```

### Warning: "Using legacy struct access"
**Причина**: В bootstrap mode используется `cfg.Server.Port`

**Решение**: Используйте `source.GetInt(ctx, "server.port")` вместо прямого доступа.

---

## 📊 Performance

### HybridConfigSource Lookup
1. **Database**: ~1-5ms (cached in settings.Manager)
2. **YAML fallback**: ~0.1ms (in-memory struct)
3. **Total**: ~1-5ms per GetString/GetInt call

### Optimization
```go
// Cache frequently accessed values
type AppConfig struct {
    ServerPort int
    JWTSecret  string
}

// Load once at startup
appCfg := AppConfig{
    ServerPort: source.GetInt(ctx, "server.port"),
    JWTSecret:  source.GetString(ctx, "auth.jwt.secret"),
}
```

---

## 🧪 Testing

### Unit Test Example
```go
func TestHybridConfig(t *testing.T) {
    mockSettings := &MockSettingsManager{}
    mockSettings.On("GetInt", mock.Anything, "server.port").
        Return(9090, nil)
    
    cfg := &config.Config{
        Server: config.ServerConfig{Port: 8080},
    }
    
    source := config.NewHybridConfigSource(mockSettings, cfg, nil)
    
    port, _ := source.GetInt(context.Background(), "server.port")
    assert.Equal(t, 9090, port) // Database takes precedence
}
```

---

## 📚 Related Documentation

- [BOOTSTRAP_MIGRATION.md](./BOOTSTRAP_MIGRATION.md) - Migration roadmap
- [SETTINGS_CLI.md](./SETTINGS_CLI.md) - CLI management
- [CONFIGURATION_MIGRATION.md](./CONFIGURATION_MIGRATION.md) - Initial plan
- `internal/config/loader.go` - Implementation
- `internal/config/hybrid.go` - HybridConfigSource
- `internal/config/source.go` - ConfigSource interface

---

## 💡 Best Practices

1. **Use Hybrid Mode for Migration**
   - Minimizes breaking changes
   - Allows gradual refactoring
   - Old code continues working

2. **Context with Timeout**
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()
   val, err := source.GetString(ctx, "server.host")
   ```

3. **Cache at Startup**
   - Don't call `GetInt()` on every request
   - Load once, cache in struct

4. **Use WithDefault Variants**
   - Prevents panics from missing keys
   - Graceful degradation

5. **Test Both Modes**
   - Test legacy mode (full YAML)
   - Test hybrid mode (YAML + DB)
   - Test bootstrap mode (minimal YAML)

---

## 🎉 Summary

**v3.1.0 предоставляет:**
- ✅ Полная обратная совместимость
- ✅ Постепенная миграция (hybrid mode)
- ✅ Bootstrap mode для новых деплойментов
- ✅ Database-first с YAML fallback
- ✅ Нет breaking changes

**Рекомендуемый путь:**
1. Используй hybrid mode
2. Постепенно рефактори критичные пути
3. Переключись на bootstrap когда готов


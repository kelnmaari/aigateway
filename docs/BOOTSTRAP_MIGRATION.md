# Bootstrap Config Migration Guide

**Version:** v3.1.0  
**Status:** ✅ Partially Implemented (Hybrid Mode Ready)

## Overview

Новая система конфигурации **v3.1.0** реализована с поддержкой **трёх режимов**:
- ✅ **Legacy Mode**: Полный YAML (backward compatible)
- ✅ **Hybrid Mode**: YAML + БД overlay (постепенная миграция)
- ✅ **Bootstrap Mode**: Минимальный YAML + полная БД (требует рефакторинга)

**Текущий статус**: Инфраструктура готова, можно использовать Hybrid Mode **без breaking changes**.

---

## 🎯 Current State (v3.1.0)

### Что реализовано ✅

```go
// 1. ConfigSource interface
type ConfigSource interface {
    GetString(ctx context.Context, key string) (string, error)
    GetInt(ctx context.Context, key string) (int, error)
    GetBool(ctx context.Context, key string) (bool, error)
    GetDuration(ctx context.Context, key string) (time.Duration, error)
    // + WithDefault variants
}

// 2. HybridConfigSource (DB-first, YAML fallback)
source := config.NewHybridConfigSource(settingsManager, yamlConfig, logger)
port, _ := source.GetInt(ctx, "server.port") // From DB or YAML

// 3. Config Loader (3 modes)
cfg, source, _ := config.LoadWithSettings(
    "configs/dev.yaml",     // or bootstrap.yaml
    settingsManager,
    logger,
)

// 4. ConfigWrapper (dual interface)
wrapper := config.NewConfigWrapper(cfg, source, logger)
wrapper.Server.Port         // Legacy access ✅
wrapper.GetInt("server.port") // New access ✅

// 5. Bootstrap Generator
./bin/server.exe -generate-bootstrap=configs/bootstrap.yaml
```

### Что работает сейчас:

```bash
# Mode 1: Legacy (no changes required)
./bin/server.exe -config=configs/dev.yaml

# Mode 2: Hybrid (backward compatible, database overlay)
./bin/server.exe -config=configs/dev.yaml
# Settings from DB take precedence when available

# Mode 3: Bootstrap (requires code refactoring)
./bin/server.exe -config=configs/bootstrap.yaml
# MUST use source.GetInt() instead of cfg.Server.Port
```

---

## 🚀 Target State (v3.1.0)

### Minimal Bootstrap Config

**`configs/bootstrap.yaml`** (8 строк):
```yaml
database:
  type: postgresql
  postgresql:
    host: 192.168.1.101
    port: 5432
    database: aigateway
    user: postgres
    password: ${POSTGRES_PASSWORD}
```

### Database-First Loading

```go
// Config loaded from DB, not YAML ✅
cfg := config.LoadFromDatabase(db, logger)

server := &http.Server{
    Addr:        cfg.GetString("server.host") + ":" + cfg.GetInt("server.port"),
    ReadTimeout: cfg.GetDuration("server.read_timeout"),
}
```

---

## 📋 Implementation Phases

### Phase 1: Bootstrap Generator ✅ DONE (v3.0.9+)

**CLI Command:**
```bash
./bin/server.exe -generate-bootstrap=configs/bootstrap.yaml
```

**Output:**
- Minimal YAML with DB connection only
- Environment variable placeholders
- Instructional comments

**Status**: ✅ Implemented in `internal/settings/bootstrap.go`

---

### Phase 2: ConfigSource Interface (v3.1.0)

**Goal**: Abstract configuration access behind interface.

**Design:**
```go
// internal/config/source.go
type ConfigSource interface {
    GetString(key string) string
    GetInt(key string) int
    GetBool(key string) bool
    GetDuration(key string) time.Duration
    GetStringSlice(key string) []string
}

// Cascading loader: DB → Fallback to YAML
type HybridConfigSource struct {
    db       *settings.Manager
    fallback *Config // Original YAML
}

func (h *HybridConfigSource) GetString(key string) string {
    // Try database first
    if val, err := h.db.GetString(context.Background(), key); err == nil {
        return val
    }
    
    // Fallback to YAML (for backward compatibility)
    return h.fallback.GetNestedString(key)
}
```

**Key Feature**: Gradual migration without breaking existing code.

---

### Phase 3: Code Refactoring (v3.1.x)

**Replace direct struct access** with interface calls.

#### Before:
```go
server := &http.Server{
    Addr:        cfg.GetServerAddr(),
    ReadTimeout: cfg.Server.ReadTimeout,
}
```

#### After:
```go
server := &http.Server{
    Addr:        cfg.GetString("server.host") + ":" + strconv.Itoa(cfg.GetInt("server.port")),
    ReadTimeout: cfg.GetDuration("server.read_timeout"),
}
```

**Scope**: ~50 locations in codebase using `cfg.*` directly.

**Strategy**:
1. Refactor `cmd/server/main.go` first
2. Refactor `internal/api/router/router.go`
3. Refactor handlers and middleware
4. Add tests for each refactored component

---

### Phase 4: Database-First Mode (v3.1.x)

**New Config Loading Flow:**

```go
// 1. Load minimal bootstrap
bootstrap := config.LoadBootstrap("bootstrap.yaml")

// 2. Connect to database
db := connectDatabase(bootstrap.Database)

// 3. Load settings from database
settingsManager := settings.NewManager(db, logger)

// 4. Create hybrid config (DB-first, YAML fallback)
cfg := config.NewHybridConfig(settingsManager, bootstrap)

// 5. All cfg.GetString() calls → read from DB first
```

**Backward Compatibility**:
- If `dev.yaml` exists → use it (old behavior)
- If only `bootstrap.yaml` → load from DB
- Gradual adoption path for existing deployments

---

## 🛠️ Migration Workflow for Users

### Step 1: Current Setup (v3.0.9)
```bash
# Full YAML config required
./bin/server.exe -config=configs/dev.yaml
```

### Step 2: Migrate to Database
```bash
# One-time migration
./bin/server.exe -migrate-config
✅ Successfully migrated 59 settings to database
```

### Step 3: Generate Bootstrap
```bash
./bin/server.exe -generate-bootstrap=configs/bootstrap.yaml
✅ Bootstrap config generated
```

### Step 4: Backup Full Config
```bash
# Keep old config as backup
mv configs/dev.yaml configs/dev.yaml.backup
```

### Step 5: Use Bootstrap Mode (v3.1.0+)
```bash
# Minimal config, all settings from DB
export POSTGRES_PASSWORD=your_password
./bin/server.exe -config=configs/bootstrap.yaml
```

---

## 🔄 Benefits

### For Development:
- ✅ No need to commit sensitive configs to Git
- ✅ Change settings via UI without editing files
- ✅ Live reload for most settings (no restart)
- ✅ Per-environment configs stored in DB, not files

### For Production:
- ✅ Centralized config management
- ✅ Audit trail (who changed what, when)
- ✅ Rollback capability (export/import)
- ✅ Secrets in env vars, not config files
- ✅ Hot-swap settings without redeployment

### For Multi-Instance Deployments:
- ✅ Single source of truth (shared database)
- ✅ Instant config sync across instances
- ✅ No config drift between servers

---

## 🚧 Challenges & Solutions

### Challenge 1: Circular Dependency
**Problem**: Need DB to load config, but need config to connect to DB.

**Solution**: Bootstrap config contains only DB connection info.

---

### Challenge 2: Startup Performance
**Problem**: Database query for each `cfg.GetString()` call is slow.

**Solution**: 
- `settings.Manager` already has in-memory cache
- Load all settings once on startup
- Cache invalidation on updates

---

### Challenge 3: Type Safety
**Problem**: Interface methods lose compile-time type safety.

**Solution**:
```go
// Generate typed accessors
func (c *Config) GetServerPort() int {
    return c.GetInt("server.port")
}

// Still type-safe at call sites
port := cfg.GetServerPort()
```

---

## 📋 Implementation Status

| Phase | Description | Time | Status |
|-------|-------------|------|--------|
| **Phase 1** | Bootstrap Generator | 2h | ✅ **DONE** (v3.0.9) |
| **Phase 2** | ConfigSource Interface | 4h | ✅ **DONE** (v3.1.0) |
| **Phase 3** | HybridConfigSource | 4h | ✅ **DONE** (v3.1.0) |
| **Phase 4** | Config Loader (3 modes) | 4h | ✅ **DONE** (v3.1.0) |
| **Phase 5** | Code Refactoring | 8h | ⏳ **OPTIONAL** |
| **Phase 6** | Full Bootstrap Mode | 2h | ⏳ **OPTIONAL** |
| **Total** | - | **24h** | **67% Complete** |

### Phase 1-4: Infrastructure ✅ COMPLETE

**Реализовано:**
- `internal/config/source.go` - ConfigSource interface
- `internal/config/hybrid.go` - HybridConfigSource (DB-first)
- `internal/config/loader.go` - LoadWithMode, LoadWithSettings
- `internal/settings/bootstrap.go` - Bootstrap generator CLI

**Можно использовать сейчас:**
```go
// Hybrid mode без изменений в коде!
cfg, source, err := config.LoadWithSettings(
    "configs/dev.yaml",
    settingsManager,
    logger,
)
// DB settings take precedence, YAML is fallback
```

### Phase 5-6: Code Refactoring ⏳ OPTIONAL

**Зачем:** Полностью убрать зависимость от `dev.yaml`, использовать только `bootstrap.yaml`.

**Требует:**
- Замена ~50 мест: `cfg.Server.Port` → `source.GetInt(ctx, "server.port")`
- Рефакторинг `cmd/server/main.go`
- Рефакторинг handlers и middleware

**Но:** Необязательно! Hybrid mode работает **без breaking changes**.

---

## 🎯 Roadmap

- [x] **v3.0.9**: Settings в БД + UI/CLI management
- [x] **v3.0.9**: Live reload hooks
- [x] **v3.0.9**: Bootstrap generator CLI
- [x] **v3.1.0**: ConfigSource interface ✅
- [x] **v3.1.0**: HybridConfigSource (DB-first, YAML fallback) ✅
- [x] **v3.1.0**: Config Loader (3 modes) ✅
- [x] **v3.1.0**: ConfigWrapper (dual interface) ✅
- [ ] **v3.1.1**: Refactor main.go to use ConfigSource (optional)
- [ ] **v3.1.2**: Refactor router and handlers (optional)
- [ ] **v3.2.0**: Full database-first deployment (optional)

**Текущая версия v3.1.0 готова к использованию!** 🎉

Hybrid mode позволяет постепенную миграцию без breaking changes.

---

## 📖 Related Documentation

- [DATABASE_FIRST_USAGE.md](./DATABASE_FIRST_USAGE.md) - **Полное руководство по использованию v3.1.0**
- [SETTINGS_CLI.md](./SETTINGS_CLI.md) - CLI commands
- [CONFIGURATION_MIGRATION.md](./CONFIGURATION_MIGRATION.md) - Initial migration plan
- `internal/config/loader.go` - Config loader implementation
- `internal/config/hybrid.go` - HybridConfigSource implementation
- `internal/config/source.go` - ConfigSource interface
- `internal/settings/bootstrap.go` - Bootstrap generator

---

## 🧪 Testing Strategy

### Unit Tests:
- ConfigSource interface implementations
- Cascading fallback logic
- Type conversion edge cases

### Integration Tests:
- Full startup with bootstrap config only
- Settings reload without restart
- Multi-instance config sync

### Migration Tests:
- Backward compatibility with full YAML
- Bootstrap-only startup
- Rollback scenarios

---

## 💡 Future Enhancements (v3.2.0+)

- [ ] Remote config backend (etcd, Consul)
- [ ] Config versioning (history, rollback UI)
- [ ] Config templates for common deployments
- [ ] Schema validation (JSON Schema)
- [ ] Config diffing between environments


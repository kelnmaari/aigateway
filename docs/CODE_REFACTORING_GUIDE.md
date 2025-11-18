# Code Refactoring Guide - v3.1.0 Hybrid Config Migration

**Status:** Phase 7 In Progress (main.go completed)  
**Remaining:** ~58 locations in router, handlers, middleware

## ✅ Completed

### cmd/server/main.go ✅
- HybridConfigSource integration
- Settings Manager initialization
- Auto-seeding from YAML
- Live reload handlers
- ConfigWrapper setup
- Bootstrap mode detection

## 🔄 In Progress / TODO

### Locations Requiring Refactoring

#### internal/api/router/router.go (38 occurrences)
```go
// Current (direct access):
cfg.Server.Port
cfg.Auth.JWT.Secret

// Target (ConfigSource):
port, _ := configSource.GetInt(ctx, "server.port")
secret, _ := configSource.GetString(ctx, "auth.jwt.secret")
```

#### internal/api/middleware/*.go (10 occurrences)
- `middleware/auth.go` (2)
- `middleware/cors.go` (10)
- `middleware/apikey_db_auth.go` (1)
- `middleware/apikey_db_auth_optimized.go` (1)

#### internal/api/handlers/*.go (10 occurrences)
- `handlers/ldap.go` (2)
- `handlers/oidc.go` (2)
- `handlers/backup.go` (2)

## 📋 Refactoring Pattern

### Step 1: Add ConfigSource Parameter

```go
// Before:
func NewRouter(cfg *config.Config, db storage.Database, logger *logrus.Logger) *Router {
    // ...
}

// After:
func NewRouter(
    cfg *config.Config, 
    configSource config.ConfigSource, // Add this
    db storage.Database, 
    logger *logrus.Logger,
) *Router {
    // ...
}
```

### Step 2: Replace Direct Access

```go
// Before:
if cfg.Metrics.Enabled {
    // ...
}

// After:
ctx := context.Background()
if enabled, err := configSource.GetBool(ctx, "metrics.enabled"); err == nil && enabled {
    // ...
}

// Or with default:
if configSource.GetBoolWithDefault(ctx, "metrics.enabled", false) {
    // ...
}
```

### Step 3: Update Callers

```go
// In cmd/server/main.go:
// Before:
appRouter := router.NewRouter(cfg, db, logger)

// After:
appRouter := router.NewRouter(cfg, configWrapper.Source, db, logger)
```

## 🎯 Priority Order

### High Priority (Bootstrap Mode Required)
1. ✅ `cmd/server/main.go` - DONE
2. ⏳ `internal/api/router/router.go` - Critical для server initialization
3. ⏳ `internal/api/middleware/auth.go` - JWT secret access
4. ⏳ `internal/api/middleware/cors.go` - CORS configuration

### Medium Priority (Hybrid Mode Works Fine)
5. ⏳ `internal/api/handlers/ldap.go`
6. ⏳ `internal/api/handlers/oidc.go`
7. ⏳ `internal/api/handlers/backup.go`

### Low Priority (Rarely Used)
8. ⏳ `internal/api/middleware/apikey_db_auth*.go`

## 💡 Quick Win: Hybrid Mode без рефакторинга

**Хорошая новость:** Hybrid mode работает **БЕЗ** рефакторинга router/handlers!

### Почему?

```go
// После LoadWithSettings в main.go:
cfg = cfgNew  // cfg теперь содержит значения из БД

// Когда router делает:
cfg.Server.Port  // Получает значение из БД (через LoadWithSettings)

// HybridConfigSource уже "прошит" в cfg через LoadWithSettings!
```

### Что это означает:

- ✅ **БД настройки работают** даже без рефакторинга
- ✅ **Изменения в UI** отражаются в `cfg.*` после рестарта
- ✅ **Live reload** работает для зарегистрированных handlers
- ⚠️ **Bootstrap mode** требует полного рефакторинга

## 🔧 When to Refactor?

### Сейчас (v3.1.0):
- **Hybrid mode** полностью функционален
- Рефакторинг **опционален** для улучшения кода
- Bootstrap mode **не работает** без рефакторинга

### Когда нужен полный рефакторинг:
1. Переход на **bootstrap.yaml** (minimal config)
2. **Hot-reload без рестарта** для server port, timeouts
3. **Runtime config changes** через API без рестарта

## 📊 Effort Estimate

| Component | Locations | Time | Dependency |
|-----------|-----------|------|------------|
| main.go | 23 | ✅ 2h DONE | None |
| router.go | 38 | ⏳ 3h | main.go |
| middleware/* | 14 | ⏳ 2h | router.go |
| handlers/* | 6 | ⏳ 1h | router.go |
| **Total** | **81** | **8h** | Sequential |

## 🎯 Recommended Approach

### Option 1: Gradual Migration (Recommended)
```
1. ✅ main.go - DONE
2. Use Hybrid mode (БД работает!)
3. Refactor по мере необходимости
4. Full bootstrap mode когда все готово
```

**Pros:**
- ✅ Immediate benefits from БД
- ✅ No breaking changes
- ✅ Flexible timeline

### Option 2: Complete Now
```
1. ✅ main.go - DONE
2. ⏳ router.go (3h)
3. ⏳ middleware/* (2h)
4. ⏳ handlers/* (1h)
5. ✅ Full bootstrap mode enabled
```

**Pros:**
- ✅ Full bootstrap mode now
- ✅ Clean architecture
- ✅ Hot-reload everywhere

**Cons:**
- ⏰ Requires 6 more hours
- 🔧 More testing needed

## 📚 Reference Implementation

See `cmd/server/main.go:163-199` for complete example of:
- Settings Manager initialization
- ConfigSeeder usage
- Reload handlers registration
- HybridConfigSource creation
- Bootstrap mode detection

## 🧪 Testing Strategy

### Before Refactoring:
```bash
# Test hybrid mode
./bin/server.exe -config=configs/dev.yaml
# → БД settings override YAML ✅
```

### After Each Component:
```bash
# Test compilation
go build -o bin/server.exe cmd/server/main.go

# Test runtime
./bin/server.exe -config=configs/dev.yaml

# Test bootstrap mode (if router refactored)
./bin/server.exe -config=configs/bootstrap.yaml
```

## 💡 Key Insight

**v3.1.0 Hybrid Mode уже полностью функционален!**

Дальнейший рефакторинг **улучшает** код, но **не требуется** для работы database-first конфигурации.

Выбор за пользователем:
- **Hybrid mode сейчас** → Immediate value, minimal work
- **Full refactor** → Clean architecture, bootstrap mode

Both are valid paths forward.


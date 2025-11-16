# Configuration Migration Plan (v3.0.9+)

## Overview

Постепенная миграция конфигурации из YAML файлов в PostgreSQL database для централизованного управления и возможности изменения без перезапуска.

---

## Architecture

### Phase 1: Read-Only UI ✅ (v3.0.9)
**Status:** Implemented

**Goal:** Отображение current configuration в Admin Panel

**Components:**
- `internal/settings/manager.go` - Settings manager с кешированием
- `internal/settings/storage_postgresql.go` - PostgreSQL storage
- `internal/settings/migration.go` - Database schema
- `internal/api/handlers/ui/settings_handler.go` - UI handler
- `web/admin.html` - Settings tab (accordion UI)
- `web/js/admin-settings.js` - Frontend logic
- `web/css/settings.css` - Styles

**UI:**
- Admin Panel → Settings tab
- Accordion по категориям (Server, Auth, Inference, etc.)
- Badge: `DB` (migrated) vs `YAML` (not migrated)
- Read-only display с типами данных

---

### Phase 2: Initial Seed from YAML (Planned)
**Status:** 📋 Not started

**Goal:** Автоматическая загрузка настроек из YAML в БД при первом запуске

**Tasks:**
1. Create `ConfigSeeder` service
   ```go
   func SeedFromYAML(ctx context.Context, cfg *config.Config, storage settings.Storage) error
   ```

2. Map YAML to settings:
   - `server.port` → Setting{Category: "server", Key: "port", Value: "8085", Type: "int"}
   - `auth.enabled` → Setting{Category: "auth", Key: "enabled", Value: "true", Type: "bool"}
   - etc.

3. Run on startup if `settings` table is empty

**Priority:** High (needed for Phase 3)

---

### Phase 3: Selective Editing (Planned)
**Status:** 📋 Not started

**Goal:** Возможность редактирования safe settings через UI

**Editable Settings:**
- ✅ `server.read_timeout`, `server.write_timeout` (performance tuning)
- ✅ `logging.level` (debug/info/warn/error)
- ✅ `auth.rate_limiting.*` (rate limits)
- ✅ `metrics.enabled` (monitoring toggle)
- ❌ `server.port` (requires restart)
- ❌ `database.*` (critical)
- ❌ `auth.jwt.secret` (security critical)

**UI Changes:**
- Add "Edit" button for editable settings
- Inline editing with validation
- Confirm modal for changes
- Real-time apply (where possible) or "Restart Required" badge

**Tasks:**
1. Update UI: edit icons, input fields, save button
2. Backend validation rules
3. `ApplyChange()` hook для hot-reload
4. Audit log: who changed what when

---

### Phase 4: Live Reload (Planned)
**Status:** 📋 Not started

**Goal:** Применение изменений без перезапуска сервера (где возможно)

**Hot-Reloadable:**
- Log level
- Rate limits
- Timeouts (new connections)
- Feature flags

**Requires Restart:**
- Ports, TLS config
- Database connection
- JWT secrets

**Implementation:**
```go
type Reloadable interface {
    ApplyChange(ctx context.Context, setting *Setting) error
}

// Example
func (r *RateLimiter) ApplyChange(ctx context.Context, setting *Setting) error {
    if setting.Key == "default_requests_per_minute" {
        newLimit := parseInt(setting.Value)
        r.UpdateLimit(newLimit)
        return nil
    }
}
```

---

### Phase 5: YAML → DB Migration Tool (Planned)
**Status:** 📋 Not started

**Goal:** CLI tool для массовой миграции настроек

**CLI Commands:**
```bash
# Migrate all settings from YAML to DB
./bin/server.exe -migrate-config

# Migrate specific category
./bin/server.exe -migrate-config -category auth

# Export DB settings to YAML (for backup)
./bin/server.exe -export-config -output config-backup.yaml

# Validate DB settings against YAML schema
./bin/server.exe -validate-config
```

---

## Database Schema

```sql
CREATE TABLE settings (
    id VARCHAR(255) PRIMARY KEY,          -- "server.port", "auth.enabled"
    category VARCHAR(50) NOT NULL,        -- server, auth, inference, etc.
    key VARCHAR(255) NOT NULL,            -- port, enabled, timeout, etc.
    value TEXT NOT NULL,                  -- Actual value (as string)
    type VARCHAR(20) NOT NULL,            -- string, int, bool, float, json, array, duration
    default_value TEXT,                   -- Default from YAML
    description TEXT,                     -- Human-readable description
    is_editable BOOLEAN DEFAULT FALSE,    -- Can be changed via UI
    is_required BOOLEAN DEFAULT FALSE,    -- Required for system operation
    is_migrated BOOLEAN DEFAULT FALSE,    -- Migrated from YAML to DB
    validation_rule TEXT,                 -- Regex or validation logic
    updated_at TIMESTAMP DEFAULT NOW(),
    updated_by VARCHAR(255),              -- User ID who last updated
    
    UNIQUE(category, key)
);

CREATE INDEX idx_settings_category ON settings(category);
CREATE INDEX idx_settings_migrated ON settings(is_migrated);
CREATE INDEX idx_settings_editable ON settings(is_editable);
```

---

## API Endpoints

### Admin Panel (UI)
```
GET    /api/admin/settings              - Get all settings (grouped by category)
GET    /api/admin/settings/:category    - Get settings for specific category
PUT    /api/admin/settings/:id          - Update setting value (Phase 3+)
```

### Internal API (for services)
```go
manager.GetString(ctx, "server.host") string
manager.GetInt(ctx, "server.port") int
manager.GetBool(ctx, "auth.enabled") bool
manager.GetDuration(ctx, "server.read_timeout") time.Duration
manager.GetJSON(ctx, "rag.embedding_config", &config)

manager.SetString(ctx, "logging.level", "debug", userID)
manager.SetBool(ctx, "metrics.enabled", true, userID)
```

---

## Migration Strategy

### Step-by-Step Process:

**Week 1:** Phase 1 (Read-Only UI)
- ✅ Implement settings manager
- ✅ Create UI tab
- ✅ Display YAML config (no DB yet)

**Week 2:** Phase 2 (YAML Seed)
- Auto-populate DB from YAML on startup
- Badge distinction: DB vs YAML

**Week 3:** Phase 3 (Selective Editing)
- Enable editing for safe settings
- Validation & audit log

**Week 4:** Phase 4 (Live Reload)
- Hot-reload for non-critical settings
- "Restart Required" indicator

**Week 5:** Phase 5 (Migration Tool)
- CLI commands
- Export/import functionality

---

## Rollback Plan

If migration fails or causes issues:

1. **Emergency:** Keep YAML as source of truth (Phase 1-2)
2. **Fallback:** Use YAML if DB settings fail to load
3. **Feature Flag:** `server.config.use_database: false` to disable DB config

---

## Testing Plan

### Unit Tests
- ✅ Settings manager (Get/Set operations)
- ✅ PostgreSQL storage (CRUD)
- 📋 Config seeder (YAML → DB)
- 📋 Validation rules

### Integration Tests
- 📋 Full workflow: YAML → DB → UI → Edit → Apply
- 📋 Hot-reload scenarios
- 📋 Concurrent updates

### UI Tests
- 📋 Settings tab rendering
- 📋 Edit form validation
- 📋 Audit log display

---

## Security Considerations

1. **Sensitive Settings:** 
   - JWT secrets, API keys → **NOT** editable via UI
   - Encrypted storage for secrets (future)

2. **RBAC:**
   - Only admins can view/edit settings
   - Audit log for all changes

3. **Validation:**
   - Type checking (int, bool, duration)
   - Range validation (port 1-65535)
   - Regex patterns (email, URL)

---

## Performance

- **Cache:** In-memory cache в settings manager (invalidate on update)
- **DB Load:** Minimal (settings read once, cached)
- **Hot-Reload:** Only changed settings trigger reload

---

## Future Enhancements

- 📋 **Tenant-specific settings:** Override global settings per tenant
- 📋 **Environment profiles:** Dev/Staging/Prod configs
- 📋 **Version control:** Track changes over time
- 📋 **Config templates:** Pre-defined setting bundles
- 📋 **Secret management:** Integration with HashiCorp Vault

---

## Dependencies

- `github.com/jmoiron/sqlx` - PostgreSQL queries
- `github.com/sirupsen/logrus` - Logging
- `github.com/gin-gonic/gin` - HTTP handlers

---

## Related Files

- `internal/settings/*` - Core settings system
- `internal/config/config.go` - Current YAML config
- `web/admin.html` - UI (Settings tab)
- `web/js/admin-settings.js` - Frontend logic
- `docs/CONFIGURATION_MIGRATION.md` - This file

---

## Contributors

v3.0.9 implementation by: [Your Team]


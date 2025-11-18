# Settings CLI Commands

**Version:** v3.0.9 - Phase 5 + Optional Features  
**Status:** ✅ Implemented

## Overview

CLI tools для управления настройками, хранящимися в PostgreSQL.

## Available Commands

### 1. Migrate Settings from YAML

Миграция всех настроек из `configs/dev.yaml` в PostgreSQL:

```bash
# Обычная миграция с подтверждением
./bin/server.exe -migrate-config

# Preview changes (dry-run) без применения
./bin/server.exe -migrate-config -dry-run
```

**Behavior:**
- Проверяет, есть ли уже настройки в БД
- Если настройки уже есть → запрашивает подтверждение перезаписи
- Создает записи в таблице `settings` для всех параметров из YAML
- Устанавливает `is_migrated=true` для всех настроек
- Устанавливает `requires_restart` согласно конфигурации

**Dry-run mode (`-dry-run`):**
- Показывает preview изменений без применения
- Отображает что будет СОЗДАНО, ОБНОВЛЕНО, и что останется без изменений
- Не требует подтверждения (read-only)

**Output (normal mode):**
```
⚠️  Warning: Database already contains 42 settings
Overwrite existing settings? (yes/no): yes
✅ Successfully migrated 42 settings from YAML to database
```

**Output (dry-run mode):**
```
🔍 DRY RUN MODE: Preview only, no changes will be applied

📊 Migration Preview:
Total settings in YAML: 42
Total settings in DB: 42

✨ Would CREATE (2):
  + server.new_feature = 'enabled'
  + auth.oauth_timeout = '30s'

🔄 Would UPDATE (3):
  ~ server.port: '8080' → '9000'
  ~ auth.jwt_expiration: '24h' → '48h'
  ~ metrics.enabled: 'false' → 'true'

✓ Unchanged (37 settings)

💡 Run without --dry-run to apply these changes
```

### 2. Export Settings to File

Экспорт всех настроек из БД в файл (YAML или JSON):

```bash
# Export as YAML (default)
./bin/server.exe -export-config=backup.yaml

# Export as JSON
./bin/server.exe -export-config=settings.json -format=json

# Explicit YAML format
./bin/server.exe -export-config=backup.yaml -format=yaml
```

**Output file structure (YAML):**
```yaml
exported_at: "2025-11-16T15:30:00Z"
total_settings: 42
settings:
  server:
    - id: server.host
      key: host
      value: "0.0.0.0"
      type: string
      default_value: "0.0.0.0"
      description: "Server bind address"
      is_editable: true
      requires_restart: true
      validation_rule: "^[0-9]{1,3}(\\.[0-9]{1,3}){3}$|^[a-zA-Z0-9.-]+$"
  auth:
    - id: auth.enabled
      key: enabled
      value: "true"
      type: bool
      ...
```

**Use cases:**
- ✅ Backup before making bulk changes
- ✅ Review current settings configuration
- ✅ Audit changes over time
- ✅ Documentation generation

### 3. Validate Settings

Сравнение настроек в БД с YAML конфигом (для диагностики):

```bash
# Simple validation report
./bin/server.exe -validate-config

# Detailed diff with side-by-side comparison
./bin/server.exe -validate-config -diff
```

**Output (simple mode):**
```
📊 Validation Report:
Total in DB: 42
Total in YAML: 42

❌ Missing in DB (2):
  - server.new_feature_flag
  - auth.oauth_timeout

⚠️  Not in YAML config (1):
  - deprecated.old_setting

⚠️  Value mismatches (3):
  - server.port (DB: 9000, YAML: 8080)
  - auth.jwt_expiration (DB: 48h, YAML: 24h)
  - metrics.enabled (DB: false, YAML: true)
```

**Interpretation:**
- **Missing in DB** → Нужно добавить в seeder или применить `-migrate-config`
- **Not in YAML config** → Настройка была добавлена вручную через UI или устарела
- **Value mismatches** → БД и YAML рассинхронизированы (ожидаемо если редактировали через UI)

**Diff mode (`-diff`):**
- Side-by-side сравнение значений
- Подробный вывод с категоризацией изменений
- Табличное представление для value mismatches
- Рекомендуемые действия для синхронизации

### 4. List Settings

Показать все настройки в табличном виде:

```bash
# All settings
./bin/server.exe -list-settings

# Filter by category
./bin/server.exe -list-settings -category auth
```

**Output:**
```
ID                        CATEGORY  VALUE           EDITABLE  RESTART
---                       ---       ---             ---       ---
server.host               server    0.0.0.0         ✅        ⚠️
server.port               server    8080            ✅        ⚠️
auth.enabled              auth      true            ✅        ❌
auth.jwt_expiration       auth      24h             ✅        ❌
logging.level             logging   info            ✅        ❌
rate_limit_rpm            rate_limit 60             ✅        ❌

Total: 42 settings
```

**Legend:**
- ✅ Editable via UI
- ❌ Read-only (requires config file change)
- ⚠️ Requires server restart after change

## Workflow Examples

### Initial Setup

При первом запуске сервера (v3.0.9+) миграция происходит автоматически:

```bash
./bin/server.exe -config configs/dev.yaml
# [INFO] Settings table is empty, auto-seeding from YAML...
# [INFO] ✅ Settings seeded from YAML config: count=42
```

### Manual Re-sync

Если YAML файл был обновлен и нужно перенести изменения в БД:

```bash
# 1. Validate current state
./bin/server.exe -validate-config

# 2. Export current settings (backup)
./bin/server.exe -export-config=backup-$(date +%Y%m%d).yaml

# 3. Migrate updated settings
./bin/server.exe -migrate-config
# Overwrite existing settings? (yes/no): yes

# 4. Validate again
./bin/server.exe -validate-config
# ✅ All settings are in sync!
```

### Backup Before Major Change

```bash
# Export before editing via UI
./bin/server.exe -export-config=backup-before-rate-limit-change.yaml

# Edit via UI (e.g., change rate limits)

# Export after change for comparison
./bin/server.exe -export-config=backup-after-rate-limit-change.yaml

# Diff the changes
diff backup-before-rate-limit-change.yaml backup-after-rate-limit-change.yaml
```

### Rollback to YAML Defaults

Если настройки через UI были изменены и нужно вернуть YAML defaults:

```bash
# 1. Backup current state
./bin/server.exe -export-config=backup-before-rollback.yaml

# 2. Force re-migrate from YAML
./bin/server.exe -migrate-config
# Overwrite existing settings? (yes/no): yes

# 3. Restart server to apply
./bin/server.exe -config configs/dev.yaml
```

## Implementation Details

### File Structure

```
internal/settings/
  cli.go              - CLI command implementations
  seeder.go           - ConfigSeeder with force flag
  storage_adapter.go  - Adapter for storage.Database interface
  manager.go          - Settings manager (unchanged)
  migration.go        - DB schema (unchanged)

cmd/server/main.go
  - CLI flags parsing
  - handleSettingsCommands() - Router for CLI commands
```

### Key Functions

**cli.go:**
- `MigrateCommand(ctx, cfg, storage, logger, categoryFilter, dryRun) error`
- `ExportCommand(ctx, storage, logger, outputPath, format) error`
- `ValidateCommand(ctx, cfg, storage, logger, diffMode) error`
- `ListCommand(ctx, storage, categoryFilter) error`

**seeder.go:**
- `SeedFromYAML(ctx, cfg) (int, error)` - Default (skip if exists)
- `SeedFromYAMLForce(ctx, cfg, force) (int, error)` - With force flag
- `MapConfigToSettings(cfg) []Setting` - Public for validation

### Testing

```bash
# Run tests
go test ./internal/settings/... -v

# Test dry-run
./bin/server.exe -migrate-config -dry-run

# Test diff mode
./bin/server.exe -validate-config -diff

# Test JSON export
./bin/server.exe -export-config=settings.json -format=json
cat settings.json | jq
```

## Error Handling

**Common errors:**

1. **Database not configured:**
   ```
   ❌ Settings commands require database to be configured
   ```
   → Add `database:` section to config file

2. **Migration failed:**
   ```
   ❌ Migration failed: failed to upsert setting server.host: pq: connection refused
   ```
   → Check PostgreSQL is running and connection string is correct

3. **Export to read-only location:**
   ```
   ❌ Export failed: failed to write file: permission denied
   ```
   → Use writable directory (e.g., `./exports/backup.yaml`)

4. **Validation timeout:**
   ```
   ❌ Validation failed: context deadline exceeded
   ```
   → Database query timeout, check DB load

## Optional Features (v3.0.9+)

### ✅ Dry-Run Mode for Migration
```bash
./bin/server.exe -migrate-config -dry-run
```
- Preview changes before applying
- Shows CREATE/UPDATE/Unchanged summary
- No confirmation required (read-only operation)

### ✅ Diff Mode for Validation
```bash
./bin/server.exe -validate-config -diff
```
- Side-by-side comparison of DB vs YAML
- Tabular output for value mismatches
- Detailed breakdown by change type
- Recommended actions for sync

### ✅ JSON Export Format
```bash
./bin/server.exe -export-config=settings.json -format=json
```
- Export as JSON or YAML (default: yaml)
- Structured data for programmatic access
- Same metadata as YAML format

### ✅ Import from Exported File
```bash
# Import from YAML backup
./bin/server.exe -import-config=backup.yaml

# Import from JSON
./bin/server.exe -import-config=settings.json
```
- Restore settings from previously exported file
- Auto-detects YAML or JSON format
- Confirms before overwriting existing settings
- Shows import summary with success/failed counts

### ✅ Delete Deprecated Settings
```bash
# Delete specific settings by ID
./bin/server.exe -delete-settings 'old.setting1,deprecated.param2'

# Force delete without confirmation
./bin/server.exe -delete-settings 'unused.config' -force
```

**Behavior:**
- Accepts comma-separated list of setting IDs
- Shows preview table before deletion
- Interactive confirmation (unless `--force`)
- Validates all settings exist before proceeding
- Removes from storage + cache
- Auto-unregisters reload handlers
- Batch deletion with summary report

**Output:**
```
📋 Settings to be deleted (2):

ID                  Category    Key           Value         Migrated
──                  ────────    ───           ─────         ────────
old.setting1        auth        old_flag      true          Yes
deprecated.param2   server      old_timeout   30s           Yes

⚠️  Are you sure you want to DELETE these settings? (yes/no): yes
✅ Deleted: old.setting1
✅ Deleted: deprecated.param2

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Deletion Summary:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Deleted:  2
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Use Cases:**
- Remove deprecated settings after refactoring
- Clean up test/debug settings in production
- Consolidate duplicate/obsolete configuration

## Live Reload Handlers (Phase 4)

Settings with `requires_restart=false` support hot-reload through registered handlers:

### Supported Hot-Reloadable Settings

| Setting ID | Handler | Description |
|-----------|---------|-------------|
| `logging.level` | `logrus.SetLevel()` | Dynamic log level change |
| `database.postgresql.max_open_conns` | `db.SetMaxOpenConns()` | Connection pool resize |
| `database.postgresql.max_idle_conns` | `db.SetMaxIdleConns()` | Idle connection limit |

**Example:**
```bash
# Change log level via UI
PUT /api/admin/settings/logging.level
Body: {"value": "debug"}

# → logrus.SetLevel(debug) applied instantly!
# → No server restart required
```

### Extensibility

Reload handlers are registered in `internal/settings/reload_handlers.go`:

```go
// Register custom handler
manager.RegisterReloadHandler("inference.yzma.temperature", func(ctx context.Context, setting *Setting) error {
    temp, err := strconv.ParseFloat(setting.Value, 64)
    if err != nil {
        return err
    }
    yzmaClient.SetDefaultTemperature(temp)
    return nil
})
```

**Future candidates:**
- `auth.rate_limiting.default_requests_per_minute`
- `inference.yzma.temperature/top_k/top_p`
- `metrics.enabled` (toggle Prometheus)

## Future Enhancements (v3.1.0+)

- [ ] Interactive mode with prompts for bulk editing
- [ ] Web UI for CLI-equivalent operations
- [ ] `--category` filter for validate and export commands
- [ ] Soft delete with `deleted_at` timestamp

## Related Documentation

- [Configuration Migration Plan](./CONFIGURATION_MIGRATION.md) - Overall migration strategy
- [Settings Manager API](./SETTINGS_API.md) - Programmatic access
- [Database Schema](../internal/settings/migration.go) - `settings` table structure


# Settings CLI Commands

**Version:** v3.0.9 - Phase 5
**Status:** ✅ Implemented

## Overview

CLI tools для управления настройками, хранящимися в PostgreSQL.

## Available Commands

### 1. Migrate Settings from YAML

Миграция всех настроек из `configs/dev.yaml` в PostgreSQL:

```bash
./bin/server.exe -migrate-config
```

**Behavior:**
- Проверяет, есть ли уже настройки в БД
- Если настройки уже есть → запрашивает подтверждение перезаписи
- Создает записи в таблице `settings` для всех параметров из YAML
- Устанавливает `is_migrated=true` для всех настроек
- Устанавливает `requires_restart` согласно конфигурации

**Output:**
```
⚠️  Warning: Database already contains 42 settings
Overwrite existing settings? (yes/no): yes
✅ Successfully migrated 42 settings from YAML to database
```

### 2. Export Settings to YAML

Экспорт всех настроек из БД в YAML файл (для backup или review):

```bash
./bin/server.exe -export-config=backup.yaml
```

**Output file structure:**
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
./bin/server.exe -validate-config
```

**Output:**
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
- `MigrateCommand(ctx, cfg, storage, logger, categoryFilter) error`
- `ExportCommand(ctx, storage, logger, outputPath) error`
- `ValidateCommand(ctx, cfg, storage, logger) error`
- `ListCommand(ctx, storage, categoryFilter) error`

**seeder.go:**
- `SeedFromYAML(ctx, cfg) (int, error)` - Default (skip if exists)
- `SeedFromYAMLForce(ctx, cfg, force) (int, error)` - With force flag
- `MapConfigToSettings(cfg) []Setting` - Public for validation

### Testing

```bash
# Run tests
go test ./internal/settings/... -v

# CLI integration test
./bin/server.exe -migrate-config
./bin/server.exe -validate-config
./bin/server.exe -export-config=test.yaml
cat test.yaml
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

## Future Enhancements (v3.1.0+)

- [ ] `--dry-run` flag for migrate-config (preview changes without applying)
- [ ] `--category` filter for validate and export commands
- [ ] `--diff` mode for validate (show side-by-side comparison)
- [ ] `--import-config` command (restore from exported YAML)
- [ ] Interactive mode with prompts for bulk editing
- [ ] Web UI for CLI-equivalent operations

## Related Documentation

- [Configuration Migration Plan](./CONFIGURATION_MIGRATION.md) - Overall migration strategy
- [Settings Manager API](./SETTINGS_API.md) - Programmatic access
- [Database Schema](../internal/settings/migration.go) - `settings` table structure


# REFACTOR-01: Migration System Refactoring

**Задача:** REFACTOR-01  
**Версия:** v2.4.7  
**Статус:** 📋 Запланирована  
**Приоритет:** MEDIUM  
**Сложность:** Medium  
**Оценка:** 4-6 часов (с rollback системой)  
**Dependencies:** None

---

## 📋 Описание

Рефакторинг системы миграций базы данных путем разделения монолитного файла `sqlite.go` (4560+ строк) на отдельные SQL файлы с использованием Go 1.16+ `embed.FS` + добавление rollback системы для безопасного отката миграций.

### Текущая проблема

**Файл `internal/storage/sqlite/sqlite.go`:**
- 4560+ строк кода
- Содержит 71 миграцию в виде Go функций
- Постоянно растет с каждой версией
- Сложно ревьюить изменения в Git
- Невозможно использовать SQL форматтеры/linters
- Нарушение хронологического порядка при поиске

**Пример текущей структуры:**
```go
func (s *SQLiteDB) getMigrations() []migration {
    return []migration{
        {Version: 1, Name: "initial_schema", SQL: s.getInitialSchemaMigration()},
        {Version: 2, Name: "add_mcp_servers", SQL: s.getMCPServersMigration()},
        // ... еще 69 миграций
    }
}

// 71 функция вида:
func (s *SQLiteDB) getInitialSchemaMigration() string {
    return `CREATE TABLE ... 500 lines of SQL ...`
}
```

---

## 🎯 Цели

1. **Разделение на файлы** - каждая миграция в отдельном SQL файле (.up.sql и .down.sql)
2. **Встраивание с embed.FS** - использование нативного Go механизма
3. **Автоматическая загрузка** - loader читает все файлы из директории
4. **Rollback система** - возможность отката миграций до любой версии
5. **Reversible migrations** - все миграции должны иметь возможность отката
6. **Обновление правил** - новые Cursor Rules для работы с SQL файлами
7. **Документация** - обновление руководств и примеров

---

## 📁 Новая структура (с Rollback Support)

```
internal/storage/
  sqlite/
    sqlite.go              (~300 строк - connection management)
    migrations.go          (~200 строк - loader & rollback system)
    migrations/
      001_initial_schema.up.sql      ← CREATE TABLES
      001_initial_schema.down.sql    ← DROP TABLES
      002_add_mcp_servers.up.sql
      002_add_mcp_servers.down.sql
      003_add_changelogs_with_data.up.sql
      003_add_changelogs_with_data.down.sql
      ...
      071_add_changelog_v2_4_6.up.sql
      071_add_changelog_v2_4_6.down.sql
  postgresql/
    postgresql.go          (~200 строк)
    migrations.go          (~150 строк)
    migrations/
      001_initial_schema.up.sql
      001_initial_schema.down.sql
      ...
```

**Ключевые изменения:**
- ✅ Каждая миграция теперь имеет `.up.sql` (forward) и `.down.sql` (rollback)
- ✅ Loader читает оба типа файлов
- ✅ Возможность отката миграций до любой версии
- ✅ Автоматические бэкапы перед применением миграций

---

## 🔧 Техническая реализация

### 1. Создание директории миграций

```bash
mkdir -p internal/storage/sqlite/migrations
mkdir -p internal/storage/postgresql/migrations
```

### 2. Конвертация миграций в SQL файлы

**Скрипт автоматизации (опционально):**
```bash
# scripts/extract-migrations.sh
#!/bin/bash

# Извлечь SQL из Go функций и создать отдельные файлы
# Парсинг функций вида getXXXMigration() и сохранение в .sql
```

**Ручная конвертация (если скрипт не нужен):**

Пример миграции `001_initial_schema.sql`:
```sql
-- ========================================
-- Users Table (AUTH-05)
-- ========================================
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	email TEXT NOT NULL UNIQUE,
	full_name TEXT NOT NULL,
	password_hash TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'active',
	...
);

CREATE INDEX idx_users_username ON users(username);
-- ... rest of SQL
```

### 3. Новый loader в migrations.go (с Rollback Support)

```go
package sqlite

import (
	"context"
	"embed"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// migration представляет одну миграцию (up + down)
type migration struct {
	Version  int
	Name     string
	UpSQL    string  // Forward migration
	DownSQL  string  // Rollback migration
}

// getMigrations возвращает список всех миграций из embedded FS
func (s *SQLiteDB) getMigrations() []migration {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		s.logger.WithError(err).Fatal("Failed to read migrations directory")
	}

	// Map для сбора up/down пар
	migrationsMap := make(map[int]*migration)
	versionRegex := regexp.MustCompile(`^(\d+)_(.+)\.(up|down)\.sql$`)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		matches := versionRegex.FindStringSubmatch(filename)
		if matches == nil {
			s.logger.WithField("file", filename).Warn("Skipping migration file with invalid format")
			continue
		}

		version, err := strconv.Atoi(matches[1])
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"file":  filename,
				"error": err,
			}).Warn("Skipping migration file with invalid version number")
			continue
		}

		name := matches[2]
		direction := matches[3]  // "up" или "down"

		content, err := migrationsFS.ReadFile(filepath.Join("migrations", filename))
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"file":  filename,
				"error": err,
			}).Error("Failed to read migration file")
			continue
		}

		// Создаем migration если еще не существует
		if migrationsMap[version] == nil {
			migrationsMap[version] = &migration{
				Version: version,
				Name:    name,
			}
		}

		// Заполняем up или down SQL
		if direction == "up" {
			migrationsMap[version].UpSQL = string(content)
		} else {
			migrationsMap[version].DownSQL = string(content)
		}
	}

	// Convert map to sorted slice
	migrations := make([]migration, 0, len(migrationsMap))
	for _, m := range migrationsMap {
		migrations = append(migrations, *m)
	}
	
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	s.logger.WithField("count", len(migrations)).Info("Loaded migrations from embedded FS")
	return migrations
}

// Rollback откатывает миграции до указанной версии
func (s *SQLiteDB) Rollback(ctx context.Context, targetVersion int) error {
	currentVersion, err := s.GetMigrationVersion(ctx)
	if err != nil {
		return err
	}

	if targetVersion >= currentVersion {
		return fmt.Errorf("target version (%d) must be less than current version (%d)", targetVersion, currentVersion)
	}

	s.logger.WithFields(logrus.Fields{
		"current": currentVersion,
		"target":  targetVersion,
	}).Info("Starting migration rollback")

	// Создаем backup перед rollback
	backupPath := fmt.Sprintf("data/backups/pre-rollback-%d-%d.db", time.Now().Unix(), currentVersion)
	if err := s.createBackup(ctx, backupPath); err != nil {
		s.logger.WithError(err).Warn("Failed to create backup before rollback")
	}

	migrations := s.getMigrations()

	// Откатываем миграции в обратном порядке
	for i := len(migrations) - 1; i >= 0; i-- {
		m := migrations[i]
		
		// Пропускаем миграции которые не нужно откатывать
		if m.Version <= targetVersion || m.Version > currentVersion {
			continue
		}

		s.logger.WithField("version", m.Version).Info("Rolling back migration")

		// Проверяем наличие down SQL
		if m.DownSQL == "" {
			return fmt.Errorf("migration %d (%s) has no down migration - rollback impossible", m.Version, m.Name)
		}

		// Проверяем на пометку IRREVERSIBLE
		if strings.Contains(m.DownSQL, "IRREVERSIBLE MIGRATION") {
			s.logger.WithField("version", m.Version).Warn("Migration marked as irreversible")
			// Можно либо прервать, либо продолжить с warning
		}

		if err := s.rollbackMigration(ctx, m); err != nil {
			return fmt.Errorf("failed to rollback migration %d: %w", m.Version, err)
		}

		s.logger.WithField("version", m.Version).Info("Migration rolled back successfully")
	}

	finalVersion, _ := s.GetMigrationVersion(ctx)
	s.logger.WithFields(logrus.Fields{
		"from": currentVersion,
		"to":   finalVersion,
	}).Info("Rollback completed successfully")

	return nil
}

func (s *SQLiteDB) rollbackMigration(ctx context.Context, m migration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute down SQL
	if _, err := tx.ExecContext(ctx, m.DownSQL); err != nil {
		return fmt.Errorf("failed to execute down SQL: %w", err)
	}

	// Remove migration record from migrations table
	if _, err := tx.ExecContext(ctx, `DELETE FROM migrations WHERE version = ?`, m.Version); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	return tx.Commit()
}

// createBackup создает backup БД перед критичными операциями
func (s *SQLiteDB) createBackup(ctx context.Context, path string) error {
	// Implementation using SQLite BACKUP API or file copy
	// ...
	return nil
}
```

### 4. Обновление sqlite.go

**До (4560 строк):**
```go
// 71 функция вида getXXXMigration()
func (s *SQLiteDB) getInitialSchemaMigration() string { ... }
func (s *SQLiteDB) getMCPServersMigration() string { ... }
// ... 69 more
```

**После (~300 строк):**
```go
// Только connection management, transaction support
// Migrations loader вынесен в migrations.go
```

---

## 📝 Обновление Cursor Rules

### 1. Файл: `.cursor/rules/changelog-migration.mdc`

**Старое правило:**
```
#### Где находится:
- Файл: `internal/storage/sqlite/sqlite.go`
- Функция: `getChangelogsMigrationWithData()`

#### Что делать:
1. Найди функцию `getChangelogsMigrationWithData()`
2. Добавь новую запись `INSERT OR REPLACE`
```

**Новое правило:**
```
#### Где находится:
- Директория: `internal/storage/sqlite/migrations/`
- Формат файла: `{VERSION}_add_changelog_v{X_Y_Z}.sql`

#### Что делать:
1. Создай новый SQL файл с номером следующей версии
2. Имя файла: `072_add_changelog_v2_4_7.sql` (пример)
3. Содержимое:

```sql
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('2.4.7', 'YYYY-MM-DD', '## [2.4.7] - YYYY-MM-DD

### Added
- **Feature Name**: Описание

### Technical
- Детали');
```

**Важно:**
- Файл автоматически будет загружен через embed.FS
- Номер версии должен быть уникальным и последовательным
- Формат SQL остается прежним (INSERT OR REPLACE)
```

### 2. Файл: `.cursor/rules/changelog-migration-workflow.mdc`

**Обновить раздел "Шаг 3.1 и 3.2":**

**Старое:**
```
#### Шаг 3.1: Добавь миграцию в getMigrations()
Найди функцию `getMigrations()` и добавь новую запись...

#### Шаг 3.2: Создай функцию миграции
В конце файла перед `// NOTE: CRUD Operations` добавь...
```

**Новое:**
```
### 3. Создание SQL миграции

#### Шаг 3.1: Определи номер следующей версии миграции

Посмотри в директории `internal/storage/sqlite/migrations/`:
- Последний файл: `071_add_changelog_v2_4_6.sql`
- Следующий номер: `072`

#### Шаг 3.2: Создай новый SQL файл

**Файл:** `internal/storage/sqlite/migrations/072_add_changelog_v2_4_7.sql`

**Содержимое:**
```sql
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('2.4.7', 'YYYY-MM-DD', '## [2.4.7] - YYYY-MM-DD

### Added
- **Feature Name**: Полное описание
  - Детали...

### Changed
- **Configuration**: Изменения в конфигурации
  
### Technical
- Технические детали
- Новые зависимости
- Архитектурные изменения');
```

**ВАЖНО:**
- Escape одинарные кавычки в тексте как `''`
- Markdown форматирование должно быть корректным
- Файл будет автоматически загружен через embed.FS
- Не нужно обновлять `getMigrations()` - loader сам найдет файл
```

---

## ✅ Acceptance Criteria

### SQLite Migrations

- [ ] Создана директория `internal/storage/sqlite/migrations/`
- [ ] Все 71 миграций конвертированы в `.up.sql` и `.down.sql` файлы (001-071)
- [ ] Создан `migrations.go` с embed.FS loader (поддержка up/down)
- [ ] Файл `sqlite.go` уменьшен до ~300 строк
- [ ] Все миграции применяются корректно (тест на чистой БД)
- [ ] Обратная совместимость: старые БД мигрируются без проблем

### Rollback System

- [ ] Реализован метод `Rollback(ctx, targetVersion)` в SQLiteDB
- [ ] Rollback откатывает миграции в обратном порядке
- [ ] Автоматический backup перед rollback операциями
- [ ] CLI команды для rollback (`-rollback`, `-rollback-to`)
- [ ] Проверка на IRREVERSIBLE миграции с warning
- [ ] Логирование всех rollback операций
- [ ] Тесты на rollback функциональность

### PostgreSQL Migrations

- [ ] Создана директория `internal/storage/postgresql/migrations/`
- [ ] Конвертированы все PostgreSQL миграции (.up.sql + .down.sql)
- [ ] Создан `migrations.go` аналогичный SQLite
- [ ] Rollback система для PostgreSQL
- [ ] Тесты проходят

### Documentation & Rules

- [ ] Обновлен `.cursor/rules/changelog-migration.mdc` (with rollback guide)
- [ ] Обновлен `.cursor/rules/changelog-migration-workflow.mdc` (with rollback CLI)
- [ ] Создан `docs/MIGRATIONS_GUIDE.md` с примерами up/down миграций
- [ ] Добавлены примеры reversible/irreversible миграций
- [ ] Обновлен README.md (если упоминаются миграции)

### Testing

- [ ] Unit test для migrations loader (up/down parsing)
- [ ] Integration test для всех 71 миграций
- [ ] Test на идемпотентность (повторное применение)
- [ ] Test на чистой БД (from scratch)
- [ ] **Rollback tests:**
  - [ ] Rollback одной миграции
  - [ ] Rollback нескольких миграций
  - [ ] Rollback до версии 0 (полный сброс)
  - [ ] Rollback с IRREVERSIBLE миграцией (warning handling)
  - [ ] Backup создается перед rollback

---

## 🚀 План выполнения

### Phase 1: Подготовка (30 мин)

1. Создать директории migrations/
2. Написать скрипт извлечения SQL (опционально)
3. Создать структуру migrations.go с поддержкой up/down

### Phase 2: Конвертация SQLite (2 часа)

1. Извлечь все 71 миграции из sqlite.go
2. Создать 71 пар SQL файлов (001-071.up.sql + 001-071.down.sql)
3. Проверить синтаксис SQL в каждом файле
4. Реализовать loader с embed.FS (поддержка up/down)
5. Удалить старые функции getXXXMigration()

**Подход к DOWN миграциям:**
- Changelog миграции: `DELETE FROM changelogs WHERE version = 'X.Y.Z'`
- CREATE TABLE: `DROP TABLE IF EXISTS table_name`
- CREATE INDEX: `DROP INDEX IF EXISTS index_name`
- ALTER TABLE ADD COLUMN: `ALTER TABLE DROP COLUMN` (с предупреждением о потере данных)
- Сложные миграции: Пометка IRREVERSIBLE с объяснением

### Phase 3: Rollback System (1-1.5 часа)

1. Реализовать метод `Rollback(ctx, targetVersion)`
2. Реализовать метод `rollbackMigration(ctx, migration)`
3. Добавить автоматический backup перед rollback
4. Добавить CLI флаги:
   - `-rollback N` - откатить последние N миграций
   - `-rollback-to VERSION` - откатить до конкретной версии
   - `-migration-version` - показать текущую версию
   - `-migrations-list` - список всех миграций
5. Обработка IRREVERSIBLE миграций

### Phase 4: Конвертация PostgreSQL (30 мин)

1. Повторить для PostgreSQL migrations (.up.sql + .down.sql)
2. Создать аналогичный loader
3. Реализовать rollback для PostgreSQL

### Phase 5: Тестирование (1 час)

1. Unit tests для loader (up/down parsing)
2. Integration tests для всех миграций
3. Тест на чистой БД
4. Тест на существующей БД (upgrade path)
5. **Rollback tests:**
   - Rollback single migration
   - Rollback multiple migrations
   - Rollback to version 0
   - IRREVERSIBLE handling
   - Backup creation

### Phase 6: Документация (45 мин)

1. Обновить Cursor Rules (2 файла) с rollback примерами
2. Создать MIGRATIONS_GUIDE.md с:
   - Примеры up/down миграций
   - Руководство по reversible migrations
   - CLI команды для rollback
   - Best practices
3. Обновить README при необходимости

---

## 🔍 Риски и митигация

| Риск | Вероятность | Влияние | Митигация |
|------|-------------|---------|-----------|
| Ошибка при конвертации SQL | Средняя | Высокое | Тщательная проверка синтаксиса, тесты |
| Проблемы с embed.FS | Низкая | Среднее | Документация Go 1.16+, примеры |
| Нарушение порядка миграций | Низкая | Высокое | Автосортировка по номеру версии |
| Escape проблемы в SQL | Средняя | Среднее | SQL linter, syntax check |
| **DOWN миграции с потерей данных** | Средняя | Высокое | Обязательный backup перед rollback, IRREVERSIBLE пометки |
| **Rollback невозможен для некоторых миграций** | Средняя | Среднее | Явные IRREVERSIBLE пометки, документация |
| **Rollback применен на production без backup** | Низкая | Критическое | Автоматический backup, confirmation prompt |

---

## 📊 Метрики успеха

- **Размер sqlite.go:** 4560 строк → ~300 строк ✅ (93% reduction)
- **Читаемость:** Каждая миграция в отдельном файле ✅
- **Rollback support:** 100% миграций имеют `.down.sql` файлы ✅
- **Reversibility:** >80% миграций могут быть откачены без потери данных ✅
- **CLI tools:** 4 команды для управления миграциями ✅
- **Git diff:** Изменения видны в одном файле ✅
- **Время добавления новой миграции:** 5 мин → 1 мин ✅
- **Поддержка SQL tools:** Нет → Да ✅

---

## 📚 References

- [Go embed package documentation](https://pkg.go.dev/embed)
- [Go 1.16 embed.FS examples](https://blog.golang.org/go1.16)
- [Migration best practices](https://github.com/golang-migrate/migrate)
- Current implementation: `internal/storage/sqlite/sqlite.go`

---

## ✨ Примеры миграций

### Пример 1: Initial Schema (001_initial_schema.sql)

```sql
-- ========================================
-- Users Table (AUTH-05)
-- ========================================
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	email TEXT NOT NULL UNIQUE,
	full_name TEXT NOT NULL,
	password_hash TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'active',
	is_admin BOOLEAN NOT NULL DEFAULT 0,
	is_active BOOLEAN NOT NULL DEFAULT 1,
	verified BOOLEAN NOT NULL DEFAULT 0,
	verified_at TIMESTAMP,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	last_login TIMESTAMP,
	preferences TEXT NOT NULL DEFAULT '{}',
	metadata TEXT
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
-- ... rest of tables
```

### Пример 2: Changelog Migration (072_add_changelog_v2_4_7.sql)

```sql
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('2.4.7', '2025-10-30', '## [2.4.7] - 2025-10-30

### Added
- **Migration System Refactoring**: Разбиение sqlite.go на отдельные SQL файлы
  - 71 миграция в отдельных .sql файлах
  - Использование Go 1.16+ embed.FS
  - Автоматический loader из директории
  - Уменьшен размер sqlite.go с 4560 до ~300 строк

### Changed
- **Cursor Rules**: Обновлены правила для работы с новой системой миграций
- **Developer Experience**: Теперь добавление миграции = создание одного SQL файла

### Technical
- internal/storage/sqlite/migrations/ директория
- internal/storage/postgresql/migrations/ директория
- migrations.go с embed.FS loader
- Regex-based version extraction из имен файлов');
```

---

## 🎯 Next Steps After Completion

После завершения REFACTOR-01:

1. **Все новые миграции** создаются как отдельные SQL файлы
2. **Changelog миграции** следуют новому workflow (создать файл вместо функции)
3. **Code review** становится проще (один файл вместо большого diff)
4. **SQL качество** улучшается (можно использовать linters)

---

**Estimated Total Time:** 3-4 часа  
**Complexity:** Medium (requires careful SQL extraction)  
**Impact:** High (improves developer experience significantly)


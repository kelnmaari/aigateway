INSERT INTO changelogs (version, release_date, content) VALUES
('2.4.7', '2025-11-03', '## [2.4.7] - 2025-11-03

### Added

- **PostgreSQL Full Support**: Complete PostgreSQL database backend implementation
  - All 12 CRUD modules ported from SQLite (8,000+ lines of code)
  - 142 migration files (71 up + 71 down) with rollback support
  - Automatic placeholder conversion (? → $1, $2, $3...)
  - Production-ready with full feature parity to SQLite

### Changed

- **Migration System Refactoring**: Split monolithic sqlite.go (4,560 lines) into separate SQL files
  - Reduced sqlite.go to ~300 lines (connection management only)
  - New structure: internal/storage/{sqlite,postgresql}/migrations/
  - File naming: 001_initial_schema.up.sql + 001_initial_schema.down.sql
  - Automatic loader via Go 1.16+ embed.FS

### Fixed

- **PostgreSQL Migration Syntax**: Converted 47 changelog migrations from SQLite to PostgreSQL
  - INSERT OR REPLACE → INSERT ... ON CONFLICT (version) DO UPDATE
  - DATETIME → TIMESTAMP
  - INTEGER PRIMARY KEY AUTOINCREMENT → BIGSERIAL PRIMARY KEY
  - randomblob() → gen_random_bytes() (requires pgcrypto extension)
  - CREATE TRIGGER IF NOT EXISTS → CREATE OR REPLACE FUNCTION + CREATE TRIGGER
  - Fixed cross-platform path handling (path.Join for embed.FS)
  - Fixed build.ps1 PowerShell script (emoji parser errors, VCS status)
  - Added required PostgreSQL extensions: pgcrypto, vector

### Technical

- **Migration Loader**:
  - Regex-based version extraction from filenames
  - Automatic sorting by version number
  - Embedded FS for zero-config deployment
  - Rollback support with .down.sql files

- **PostgreSQL Specifics**:
  - JSONB for all JSON fields (better performance)
  - Native UUID support via gen_random_uuid()
  - Proper boolean types (vs INTEGER in SQLite)
  - Timestamp with timezone support
  - Full-text search capabilities (future)
  - PL/pgSQL trigger functions for updated_at

- **Code Quality**:
  - 100% method coverage across all CRUD operations
  - Type-safe placeholder conversion
  - Consistent error handling patterns
  - Thread-safe transaction support')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;

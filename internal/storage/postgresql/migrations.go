package postgresql

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path"          // For embed.FS paths (always forward slashes)
	"path/filepath" // For OS filesystem paths
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/storage"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// migration represents a database migration (forward and rollback)
type migration struct {
	Version int
	Name    string
	UpSQL   string // Forward migration
	DownSQL string // Rollback migration
}

// getMigrations loads all migrations from embedded FS
func (db *PostgreSQLDB) getMigrations() []migration {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		db.logger.WithError(err).Fatal("Failed to read migrations directory")
	}

	// Map to collect up/down pairs
	migrationsMap := make(map[int]*migration)
	versionRegex := regexp.MustCompile(`^(\d+)_(.+)\.(up|down)\.sql$`)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		matches := versionRegex.FindStringSubmatch(filename)
		if matches == nil {
			db.logger.WithField("file", filename).Warn("Skipping migration file with invalid format")
			continue
		}

		version, err := strconv.Atoi(matches[1])
		if err != nil {
			db.logger.WithFields(logrus.Fields{
				"file":  filename,
				"error": err,
			}).Warn("Skipping migration file with invalid version number")
			continue
		}

		name := matches[2]
		direction := matches[3] // "up" or "down"

		// Use path.Join (not filepath.Join) for embed.FS - always forward slashes
		content, err := migrationsFS.ReadFile(path.Join("migrations", filename))
		if err != nil {
			db.logger.WithFields(logrus.Fields{
				"file":  filename,
				"error": err,
			}).Error("Failed to read migration file")
			continue
		}

		// Create migration if not exists
		if migrationsMap[version] == nil {
			migrationsMap[version] = &migration{
				Version: version,
				Name:    name,
			}
		}

		// Fill up or down SQL
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

	db.logger.WithField("count", len(migrations)).Info("Loaded migrations from embedded FS")
	return migrations
}

// RunMigrations applies all pending migrations
func (db *PostgreSQLDB) RunMigrations(ctx context.Context) error {
	// Create migrations table if not exists
	if err := db.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	appliedVersions, err := db.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Load all migrations
	migrations := db.getMigrations()

	// Apply pending migrations
	for _, m := range migrations {
		if appliedVersions[m.Version] {
			continue // Already applied
		}

		db.logger.WithFields(logrus.Fields{
			"version": m.Version,
			"name":    m.Name,
		}).Info("Applying migration...")

		if err := db.applyMigration(ctx, m); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", m.Version, m.Name, err)
		}

		db.logger.WithFields(logrus.Fields{
			"version": m.Version,
			"name":    m.Name,
		}).Info("Migration applied successfully")
	}

	return nil
}

// Migrate applies all pending migrations
func (db *PostgreSQLDB) Migrate(ctx context.Context) error {
	db.logger.Info("Starting database migration")

	// Create migrations table if not exists
	if err := db.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	currentVersion, err := db.GetMigrationVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	db.logger.WithField("current_version", currentVersion).Info("Current migration version")

	// Apply migrations
	migrations := db.getMigrations()
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue // Already applied
		}

		db.logger.WithField("version", migration.Version).Info("Applying migration")

		if err := db.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}

		db.logger.WithField("version", migration.Version).Info("Migration applied successfully")
	}

	finalVersion, _ := db.GetMigrationVersion(ctx)
	db.logger.WithField("version", finalVersion).Info("Database migration completed")

	return nil
}

// createMigrationsTable creates the migrations tracking table
func (db *PostgreSQLDB) createMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`
	_, err := db.db.ExecContext(ctx, query)
	return err
}

// getAppliedMigrations returns a map of applied migration versions
func (db *PostgreSQLDB) getAppliedMigrations(ctx context.Context) (map[int]bool, error) {
	query := `SELECT version FROM schema_migrations ORDER BY version`
	rows, err := db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		versions[version] = true
	}

	return versions, rows.Err()
}

// applyMigration applies a single migration within a transaction
func (db *PostgreSQLDB) applyMigration(ctx context.Context, m migration) error {
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.ExecContext(ctx, m.UpSQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration as applied
	query := `INSERT INTO schema_migrations (version, name, applied_at) VALUES ($1, $2, $3)`
	if _, err := tx.ExecContext(ctx, query, m.Version, m.Name, time.Now()); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Rollback rolls back migrations to a specific version
func (db *PostgreSQLDB) Rollback(ctx context.Context, targetVersion int) error {
	// Create backup before rollback
	if err := db.createBackup(); err != nil {
		db.logger.WithError(err).Warn("Failed to create backup before rollback, continuing anyway...")
	}

	// Get applied migrations
	appliedVersions, err := db.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Load all migrations
	migrations := db.getMigrations()

	// Find migrations to rollback (in reverse order)
	var toRollback []migration
	for i := len(migrations) - 1; i >= 0; i-- {
		m := migrations[i]
		if m.Version > targetVersion && appliedVersions[m.Version] {
			toRollback = append(toRollback, m)
		}
	}

	if len(toRollback) == 0 {
		db.logger.WithField("target_version", targetVersion).Info("No migrations to rollback")
		return nil
	}

	// Rollback migrations
	for _, m := range toRollback {
		db.logger.WithFields(logrus.Fields{
			"version": m.Version,
			"name":    m.Name,
		}).Info("Rolling back migration...")

		if err := db.rollbackMigration(ctx, m); err != nil {
			return fmt.Errorf("failed to rollback migration %d (%s): %w", m.Version, m.Name, err)
		}

		db.logger.WithFields(logrus.Fields{
			"version": m.Version,
			"name":    m.Name,
		}).Info("Migration rolled back successfully")
	}

	return nil
}

// rollbackMigration rolls back a single migration
func (db *PostgreSQLDB) rollbackMigration(ctx context.Context, m migration) error {
	if m.DownSQL == "" {
		return fmt.Errorf("migration %d (%s) has no rollback SQL", m.Version, m.Name)
	}

	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute rollback SQL
	if _, err := tx.ExecContext(ctx, m.DownSQL); err != nil {
		return fmt.Errorf("failed to execute rollback SQL: %w", err)
	}

	// Remove migration record
	query := `DELETE FROM schema_migrations WHERE version = $1`
	if _, err := tx.ExecContext(ctx, query, m.Version); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetCurrentVersion returns the current schema version
func (db *PostgreSQLDB) GetCurrentVersion(ctx context.Context) (int, error) {
	query := `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`
	var version int
	err := db.db.QueryRowContext(ctx, query).Scan(&version)
	return version, err
}

// GetMigrationVersion returns the current migration version (alias for GetCurrentVersion)
func (db *PostgreSQLDB) GetMigrationVersion(ctx context.Context) (int, error) {
	return db.GetCurrentVersion(ctx)
}

// ListMigrations returns all available migrations with their status
func (db *PostgreSQLDB) ListMigrations(ctx context.Context) ([]storage.MigrationInfo, error) {
	appliedVersions, err := db.getAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	migrations := db.getMigrations()
	result := make([]storage.MigrationInfo, len(migrations))

	for i, m := range migrations {
		result[i] = storage.MigrationInfo{
			Version: m.Version,
			Name:    m.Name,
			Applied: appliedVersions[m.Version],
		}
	}

	return result, nil
}

// RollbackMigrations rolls back migrations to a target version
func (db *PostgreSQLDB) RollbackMigrations(ctx context.Context, targetVersion int) error {
	currentVersion, err := db.GetMigrationVersion(ctx)
	if err != nil {
		return err
	}

	if targetVersion >= currentVersion {
		return fmt.Errorf("target version (%d) must be less than current version (%d)", targetVersion, currentVersion)
	}

	db.logger.WithFields(logrus.Fields{
		"current": currentVersion,
		"target":  targetVersion,
	}).Info("Starting migration rollback")

	// Create backup before rollback
	if err := db.createBackup(); err != nil {
		db.logger.WithError(err).Warn("Failed to create backup before rollback")
	}

	migrations := db.getMigrations()

	// Rollback migrations in reverse order
	for i := len(migrations) - 1; i >= 0; i-- {
		m := migrations[i]

		// Skip migrations that don't need rollback
		if m.Version <= targetVersion || m.Version > currentVersion {
			continue
		}

		db.logger.WithFields(logrus.Fields{
			"version": m.Version,
			"name":    m.Name,
		}).Info("Rolling back migration")

		// Check for IRREVERSIBLE marker
		if strings.Contains(m.DownSQL, "IRREVERSIBLE MIGRATION") {
			db.logger.WithField("version", m.Version).Warn("Migration marked as irreversible - proceeding with caution")
		}

		if err := db.rollbackMigration(ctx, m); err != nil {
			return fmt.Errorf("failed to rollback migration %d: %w", m.Version, err)
		}

		db.logger.WithField("version", m.Version).Info("Migration rolled back successfully")
	}

	finalVersion, _ := db.GetMigrationVersion(ctx)
	db.logger.WithFields(logrus.Fields{
		"from": currentVersion,
		"to":   finalVersion,
	}).Info("Rollback completed successfully")

	return nil
}

// createBackup creates a database backup before rollback operations
func (db *PostgreSQLDB) createBackup() error {
	// For PostgreSQL, we'll use pg_dump if available
	// This is a simple implementation - production should use proper backup strategy

	// Use filepath.Join for OS filesystem (not embed.FS)
	backupDir := filepath.Join("data", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	currentVersion, err := db.GetCurrentVersion(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	backupFile := filepath.Join(backupDir, fmt.Sprintf("pre-rollback-%s-v%d.sql", timestamp, currentVersion))

	db.logger.WithField("backup_file", backupFile).Info("Creating backup before rollback")

	// For PostgreSQL, we would use pg_dump command
	// This is a placeholder - actual implementation would depend on PostgreSQL connection details
	// For now, just log the backup location
	db.logger.WithField("backup_file", backupFile).Warn("PostgreSQL backup requires pg_dump - ensure you have proper backup strategy")

	return nil
}

// DestroyDatabase полностью удаляет все таблицы из базы данных
func (db *PostgreSQLDB) DestroyDatabase(ctx context.Context) error {
	db.logger.Warn("⚠️  DESTROYING DATABASE - ALL DATA WILL BE LOST")

	// Create backup before destruction
	backupDir := filepath.Join("data", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		db.logger.WithError(err).Warn("Failed to create backup directory")
	}

	timestamp := time.Now().Format("20060102-150405")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("pre-destroy-%s.sql", timestamp))
	db.logger.WithField("backup_file", backupFile).Info("Backup location logged (use pg_dump manually)")

	// Get all tables in the current schema (excluding system tables)
	query := `
		SELECT tablename 
		FROM pg_tables 
		WHERE schemaname = 'public'
		ORDER BY tablename
	`

	rows, err := db.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	if len(tables) == 0 {
		db.logger.Info("No tables to drop - database is already empty")
		return nil
	}

	db.logger.WithField("table_count", len(tables)).Info("Found tables to drop")

	// Drop all tables (CASCADE will drop dependent objects)
	for _, table := range tables {
		db.logger.WithField("table", table).Debug("Dropping table")
		
		dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table)
		if _, err := db.db.ExecContext(ctx, dropSQL); err != nil {
			db.logger.WithError(err).Errorf("Failed to drop table: %s", table)
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	// Drop extensions if needed
	db.logger.Info("Dropping PostgreSQL extensions")
	_, _ = db.db.ExecContext(ctx, "DROP EXTENSION IF EXISTS vector CASCADE")
	_, _ = db.db.ExecContext(ctx, "DROP EXTENSION IF EXISTS pgcrypto CASCADE")

	db.logger.Info("✅ Database destroyed successfully - all tables dropped")
	return nil
}

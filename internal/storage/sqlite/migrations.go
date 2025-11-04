package sqlite

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path" // For embed.FS paths (always forward slashes)
	"path/filepath"
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
func (s *SQLiteDB) getMigrations() []migration {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		s.logger.WithError(err).Fatal("Failed to read migrations directory")
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
		direction := matches[3] // "up" or "down"

		// Use path.Join (not filepath.Join) for embed.FS - always forward slashes
		content, err := migrationsFS.ReadFile(path.Join("migrations", filename))
		if err != nil {
			s.logger.WithFields(logrus.Fields{
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

	s.logger.WithField("count", len(migrations)).Info("Loaded migrations from embedded FS")
	return migrations
}

// GetMigrationVersion returns current database migration version
func (s *SQLiteDB) GetMigrationVersion(ctx context.Context) (int, error) {
	var version int
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0) FROM migrations
	`).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get migration version: %w", err)
	}
	return version, nil
}

// RollbackMigrations rolls back migrations to target version
func (s *SQLiteDB) RollbackMigrations(ctx context.Context, targetVersion int) error {
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

	// Create backup before rollback
	backupPath := fmt.Sprintf("data/backups/pre-rollback-%d-%d.db", time.Now().Unix(), currentVersion)
	if err := s.createBackup(backupPath); err != nil {
		s.logger.WithError(err).Warn("Failed to create backup before rollback")
	} else {
		s.logger.WithField("path", backupPath).Info("Backup created successfully")
	}

	migrations := s.getMigrations()

	// Rollback migrations in reverse order
	for i := len(migrations) - 1; i >= 0; i-- {
		m := migrations[i]

		// Skip migrations that don't need rollback
		if m.Version <= targetVersion || m.Version > currentVersion {
			continue
		}

		s.logger.WithFields(logrus.Fields{
			"version": m.Version,
			"name":    m.Name,
		}).Info("Rolling back migration")

		// Check if down SQL exists
		if m.DownSQL == "" {
			return fmt.Errorf("migration %d (%s) has no down migration - rollback impossible", m.Version, m.Name)
		}

		// Check for IRREVERSIBLE marker
		if strings.Contains(m.DownSQL, "IRREVERSIBLE MIGRATION") {
			s.logger.WithField("version", m.Version).Warn("Migration marked as irreversible - proceeding with caution")
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

// rollbackMigration executes a single migration rollback
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

// createBackup creates a backup of the database
func (s *SQLiteDB) createBackup(backupPath string) error {
	// Ensure backup directory exists
	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Get current database file path
	var dbPath string
	err := s.db.QueryRow("PRAGMA database_list").Scan(nil, nil, &dbPath)
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	// Use SQLite VACUUM INTO for atomic backup (SQLite 3.27.0+)
	_, err = s.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		// Fallback to file copy if VACUUM INTO not supported
		return s.copyFile(dbPath, backupPath)
	}

	return nil
}

// copyFile copies a file (fallback for backup)
func (s *SQLiteDB) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := destFile.ReadFrom(sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return destFile.Sync()
}

// ListMigrations returns list of all available migrations
func (s *SQLiteDB) ListMigrations(ctx context.Context) ([]storage.MigrationInfo, error) {
	migrations := s.getMigrations()
	currentVersion, err := s.GetMigrationVersion(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]storage.MigrationInfo, len(migrations))
	for i, m := range migrations {
		result[i] = storage.MigrationInfo{
			Version:      m.Version,
			Name:         m.Name,
			Applied:      m.Version <= currentVersion,
			HasRollback:  m.DownSQL != "",
			Irreversible: strings.Contains(m.DownSQL, "IRREVERSIBLE MIGRATION"),
		}
	}

	return result, nil
}

// DestroyDatabase полностью удаляет все таблицы из базы данных
func (s *SQLiteDB) DestroyDatabase(ctx context.Context) error {
	s.logger.Warn("⚠️  DESTROYING DATABASE - ALL DATA WILL BE LOST")

	// Create backup before destruction
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("data/backups/pre-destroy-%s.db", timestamp)
	
	if err := s.createBackup(backupPath); err != nil {
		s.logger.WithError(err).Warn("Failed to create backup before destruction")
	} else {
		s.logger.WithField("backup_path", backupPath).Info("✅ Backup created successfully")
	}

	// Get all tables (excluding sqlite_* system tables)
	query := `
		SELECT name FROM sqlite_master 
		WHERE type='table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`

	rows, err := s.db.QueryContext(ctx, query)
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
		s.logger.Info("No tables to drop - database is already empty")
		return nil
	}

	s.logger.WithField("table_count", len(tables)).Info("Found tables to drop")

	// Disable foreign key constraints temporarily
	if _, err := s.db.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("failed to disable foreign keys: %w", err)
	}

	// Drop all tables
	for _, table := range tables {
		s.logger.WithField("table", table).Debug("Dropping table")
		
		dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", table)
		if _, err := s.db.ExecContext(ctx, dropSQL); err != nil {
			s.logger.WithError(err).Errorf("Failed to drop table: %s", table)
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	// Re-enable foreign key constraints
	if _, err := s.db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		s.logger.WithError(err).Warn("Failed to re-enable foreign keys")
	}

	// Vacuum to reclaim space
	s.logger.Info("Running VACUUM to reclaim space")
	if _, err := s.db.ExecContext(ctx, "VACUUM"); err != nil {
		s.logger.WithError(err).Warn("Failed to vacuum database")
	}

	s.logger.Info("✅ Database destroyed successfully - all tables dropped")
	return nil
}

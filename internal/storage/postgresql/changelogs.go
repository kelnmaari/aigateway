// Package sqlite provides SQLite implementation of changelog storage
package postgresql

import (
	"context"
	"database/sql"
	"fmt"

	"aigateway/internal/models"
)

// GetChangelog возвращает changelog для конкретной версии
func (db *PostgreSQLDB) GetChangelog(ctx context.Context, version string) (*models.Changelog, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT version, release_date, content, created_at
		FROM changelogs
		WHERE version = $1
	`

	var changelog models.Changelog
	err := db.db.QueryRowContext(ctx, query, version).Scan(
		&changelog.Version,
		&changelog.ReleaseDate,
		&changelog.Content,
		&changelog.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("changelog not found for version %s", version)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get changelog: %w", err)
	}

	db.logger.WithField("version", version).Debug("Retrieved changelog")
	return &changelog, nil
}

// ListChangelogs возвращает список всех changelog записей
func (db *PostgreSQLDB) ListChangelogs(ctx context.Context) ([]*models.Changelog, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT version, release_date, content, created_at
		FROM changelogs
		ORDER BY release_date DESC, version DESC
	`

	rows, err := db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list changelogs: %w", err)
	}
	defer rows.Close()

	var changelogs []*models.Changelog
	for rows.Next() {
		var changelog models.Changelog
		if err := rows.Scan(
			&changelog.Version,
			&changelog.ReleaseDate,
			&changelog.Content,
			&changelog.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan changelog: %w", err)
		}
		changelogs = append(changelogs, &changelog)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating changelogs: %w", err)
	}

	db.logger.WithField("count", len(changelogs)).Debug("Listed changelogs")
	return changelogs, nil
}

// Package sqlite provides SQLite implementation of changelog storage
package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"ollama-openai-proxy/internal/models"
)

// GetChangelog возвращает changelog для конкретной версии
func (s *SQLiteDB) GetChangelog(ctx context.Context, version string) (*models.Changelog, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT version, release_date, content, created_at
		FROM changelogs
		WHERE version = ?
	`

	var changelog models.Changelog
	err := s.db.QueryRowContext(ctx, query, version).Scan(
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

	s.logger.WithField("version", version).Debug("Retrieved changelog")
	return &changelog, nil
}

// ListChangelogs возвращает список всех changelog записей
func (s *SQLiteDB) ListChangelogs(ctx context.Context) ([]*models.Changelog, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT version, release_date, content, created_at
		FROM changelogs
		ORDER BY release_date DESC, version DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
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

	s.logger.WithField("count", len(changelogs)).Debug("Listed changelogs")
	return changelogs, nil
}

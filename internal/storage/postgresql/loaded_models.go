package postgresql

import (
	"context"
	"database/sql"
	"fmt"

	"aigateway/internal/models"

	"github.com/google/uuid"
)

// SaveLoadedModel сохраняет информацию о загруженной модели (v3.0.6+)
func (db *PostgreSQLDB) SaveLoadedModel(ctx context.Context, model *models.LoadedModel) error {
	if model.ID == "" {
		model.ID = "lm_" + uuid.New().String()
	}
	
	query := `
		INSERT INTO loaded_models (
			id, model_path, alias, loaded_at, auto_load, context_size, batch_size, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT(model_path) DO UPDATE SET
			alias = EXCLUDED.alias,
			loaded_at = EXCLUDED.loaded_at,
			auto_load = EXCLUDED.auto_load,
			context_size = EXCLUDED.context_size,
			batch_size = EXCLUDED.batch_size
	`
	
	_, err := db.db.ExecContext(ctx, query,
		model.ID,
		model.ModelPath,
		model.Alias,
		model.LoadedAt,
		model.AutoLoad,
		model.ContextSize,
		model.BatchSize,
		model.CreatedAt,
	)
	
	if err != nil {
		return fmt.Errorf("failed to save loaded model: %w", err)
	}
	
	return nil
}

// RemoveLoadedModel удаляет информацию о модели из persistence (v3.0.6+)
func (db *PostgreSQLDB) RemoveLoadedModel(ctx context.Context, modelPath string) error {
	query := `DELETE FROM loaded_models WHERE model_path = $1`
	
	result, err := db.db.ExecContext(ctx, query, modelPath)
	if err != nil {
		return fmt.Errorf("failed to remove loaded model: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		return fmt.Errorf("loaded model not found: %s", modelPath)
	}
	
	return nil
}

// GetLoadedModel получает информацию о загруженной модели (v3.0.6+)
func (db *PostgreSQLDB) GetLoadedModel(ctx context.Context, modelPath string) (*models.LoadedModel, error) {
	query := `
		SELECT id, model_path, alias, loaded_at, auto_load, context_size, batch_size, created_at
		FROM loaded_models
		WHERE model_path = $1
	`
	
	var model models.LoadedModel
	err := db.db.QueryRowContext(ctx, query, modelPath).Scan(
		&model.ID,
		&model.ModelPath,
		&model.Alias,
		&model.LoadedAt,
		&model.AutoLoad,
		&model.ContextSize,
		&model.BatchSize,
		&model.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("loaded model not found: %s", modelPath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get loaded model: %w", err)
	}
	
	return &model, nil
}

// ListLoadedModels возвращает список всех сохраненных моделей (v3.0.6+)
func (db *PostgreSQLDB) ListLoadedModels(ctx context.Context, autoLoadOnly bool) ([]*models.LoadedModel, error) {
	query := `
		SELECT id, model_path, alias, loaded_at, auto_load, context_size, batch_size, created_at
		FROM loaded_models
	`
	
	args := []interface{}{}
	if autoLoadOnly {
		query += ` WHERE auto_load = $1`
		args = append(args, true)
	}
	
	query += ` ORDER BY loaded_at DESC`
	
	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list loaded models: %w", err)
	}
	defer rows.Close()
	
	var loadedModels []*models.LoadedModel
	for rows.Next() {
		var model models.LoadedModel
		err := rows.Scan(
			&model.ID,
			&model.ModelPath,
			&model.Alias,
			&model.LoadedAt,
			&model.AutoLoad,
			&model.ContextSize,
			&model.BatchSize,
			&model.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan loaded model: %w", err)
		}
		loadedModels = append(loadedModels, &model)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating loaded models: %w", err)
	}
	
	return loadedModels, nil
}

// Transaction methods (v3.0.6+)

func (tx *postgresqlTx) SaveLoadedModel(ctx context.Context, model *models.LoadedModel) error {
	if model.ID == "" {
		model.ID = "lm_" + uuid.New().String()
	}
	
	query := `
		INSERT INTO loaded_models (
			id, model_path, alias, loaded_at, auto_load, context_size, batch_size, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT(model_path) DO UPDATE SET
			alias = EXCLUDED.alias,
			loaded_at = EXCLUDED.loaded_at,
			auto_load = EXCLUDED.auto_load,
			context_size = EXCLUDED.context_size,
			batch_size = EXCLUDED.batch_size
	`
	
	_, err := tx.tx.ExecContext(ctx, query,
		model.ID,
		model.ModelPath,
		model.Alias,
		model.LoadedAt,
		model.AutoLoad,
		model.ContextSize,
		model.BatchSize,
		model.CreatedAt,
	)
	
	if err != nil {
		return fmt.Errorf("failed to save loaded model: %w", err)
	}
	
	return nil
}

func (tx *postgresqlTx) RemoveLoadedModel(ctx context.Context, modelPath string) error {
	query := `DELETE FROM loaded_models WHERE model_path = $1`
	
	result, err := tx.tx.ExecContext(ctx, query, modelPath)
	if err != nil {
		return fmt.Errorf("failed to remove loaded model: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		return fmt.Errorf("loaded model not found: %s", modelPath)
	}
	
	return nil
}

func (tx *postgresqlTx) GetLoadedModel(ctx context.Context, modelPath string) (*models.LoadedModel, error) {
	query := `
		SELECT id, model_path, alias, loaded_at, auto_load, context_size, batch_size, created_at
		FROM loaded_models
		WHERE model_path = $1
	`
	
	var model models.LoadedModel
	err := tx.tx.QueryRowContext(ctx, query, modelPath).Scan(
		&model.ID,
		&model.ModelPath,
		&model.Alias,
		&model.LoadedAt,
		&model.AutoLoad,
		&model.ContextSize,
		&model.BatchSize,
		&model.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("loaded model not found: %s", modelPath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get loaded model: %w", err)
	}
	
	return &model, nil
}

func (tx *postgresqlTx) ListLoadedModels(ctx context.Context, autoLoadOnly bool) ([]*models.LoadedModel, error) {
	query := `
		SELECT id, model_path, alias, loaded_at, auto_load, context_size, batch_size, created_at
		FROM loaded_models
	`
	
	args := []interface{}{}
	if autoLoadOnly {
		query += ` WHERE auto_load = $1`
		args = append(args, true)
	}
	
	query += ` ORDER BY loaded_at DESC`
	
	rows, err := tx.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list loaded models: %w", err)
	}
	defer rows.Close()
	
	var loadedModels []*models.LoadedModel
	for rows.Next() {
		var model models.LoadedModel
		err := rows.Scan(
			&model.ID,
			&model.ModelPath,
			&model.Alias,
			&model.LoadedAt,
			&model.AutoLoad,
			&model.ContextSize,
			&model.BatchSize,
			&model.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan loaded model: %w", err)
		}
		loadedModels = append(loadedModels, &model)
	}
	
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating loaded models: %w", err)
	}
	
	return loadedModels, nil
}


package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aigateway/internal/models"
)

// ========================================
// Model Providers CRUD Operations
// ========================================

// CreateModelProvider создает нового model provider
func (s *SQLiteDB) CreateModelProvider(ctx context.Context, provider *models.ModelProvider) error {
	// Generate ID if not set
	if provider.ID == "" {
		provider.ID = generateID()
	}

	// Serialize Config to JSON
	if err := provider.MarshalConfigToDB(); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		INSERT INTO model_providers (
			id, name, provider_type, base_url, api_key,
			enabled, priority, config,
			health_status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	provider.CreatedAt = now
	provider.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, query,
		provider.ID,
		provider.Name,
		provider.ProviderType,
		provider.BaseURL,
		provider.APIKey,
		provider.Enabled,
		provider.Priority,
		provider.ConfigDB,
		provider.HealthStatus,
		provider.CreatedAt,
		provider.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create model provider: %w", err)
	}

	return nil
}

// GetModelProvider получает provider по ID
func (s *SQLiteDB) GetModelProvider(ctx context.Context, id string) (*models.ModelProvider, error) {
	query := `
		SELECT id, name, provider_type, base_url, api_key,
		       enabled, priority, config,
		       health_status, last_health_check, error_message,
		       created_at, updated_at
		FROM model_providers
		WHERE id = ?
	`

	provider := &models.ModelProvider{}
	var configJSON string
	var lastHealthCheck sql.NullTime
	var apiKey sql.NullString
	var errorMessage sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&provider.ID,
		&provider.Name,
		&provider.ProviderType,
		&provider.BaseURL,
		&apiKey,
		&provider.Enabled,
		&provider.Priority,
		&configJSON,
		&provider.HealthStatus,
		&lastHealthCheck,
		&errorMessage,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("model provider not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model provider: %w", err)
	}

	// Deserialize Config
	provider.ConfigDB = configJSON
	if err := provider.UnmarshalConfigFromDB(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Handle nullable fields
	if apiKey.Valid {
		provider.APIKey = apiKey.String
	}
	if errorMessage.Valid {
		provider.ErrorMessage = errorMessage.String
	}
	if lastHealthCheck.Valid {
		provider.LastHealthCheck = &lastHealthCheck.Time
	}

	return provider, nil
}

// GetModelProviderByName получает provider по имени
func (s *SQLiteDB) GetModelProviderByName(ctx context.Context, name string) (*models.ModelProvider, error) {
	query := `
		SELECT id, name, provider_type, base_url, api_key,
		       enabled, priority, config,
		       health_status, last_health_check, error_message,
		       created_at, updated_at
		FROM model_providers
		WHERE name = ?
	`

	provider := &models.ModelProvider{}
	var configJSON string
	var lastHealthCheck sql.NullTime
	var apiKey sql.NullString
	var errorMessage sql.NullString

	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&provider.ID,
		&provider.Name,
		&provider.ProviderType,
		&provider.BaseURL,
		&apiKey,
		&provider.Enabled,
		&provider.Priority,
		&configJSON,
		&provider.HealthStatus,
		&lastHealthCheck,
		&errorMessage,
		&provider.CreatedAt,
		&provider.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("model provider not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model provider: %w", err)
	}

	// Deserialize Config
	provider.ConfigDB = configJSON
	if err := provider.UnmarshalConfigFromDB(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Handle nullable fields
	if apiKey.Valid {
		provider.APIKey = apiKey.String
	}
	if errorMessage.Valid {
		provider.ErrorMessage = errorMessage.String
	}
	if lastHealthCheck.Valid {
		provider.LastHealthCheck = &lastHealthCheck.Time
	}

	return provider, nil
}

// UpdateModelProvider обновляет provider
func (s *SQLiteDB) UpdateModelProvider(ctx context.Context, provider *models.ModelProvider) error {
	// Serialize Config to JSON
	if err := provider.MarshalConfigToDB(); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		UPDATE model_providers SET
			name = ?, provider_type = ?, base_url = ?, api_key = ?,
			enabled = ?, priority = ?, config = ?,
			health_status = ?, error_message = ?, updated_at = ?
		WHERE id = ?
	`

	provider.UpdatedAt = time.Now()

	result, err := s.db.ExecContext(ctx, query,
		provider.Name,
		provider.ProviderType,
		provider.BaseURL,
		provider.APIKey,
		provider.Enabled,
		provider.Priority,
		provider.ConfigDB,
		provider.HealthStatus,
		provider.ErrorMessage,
		provider.UpdatedAt,
		provider.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update model provider: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("model provider not found: %s", provider.ID)
	}

	return nil
}

// DeleteModelProvider удаляет provider (cascade delete всех моделей)
func (s *SQLiteDB) DeleteModelProvider(ctx context.Context, id string) error {
	query := `DELETE FROM model_providers WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete model provider: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("model provider not found: %s", id)
	}

	return nil
}

// ListModelProviders возвращает список providers
func (s *SQLiteDB) ListModelProviders(ctx context.Context, enabledOnly bool) ([]*models.ModelProvider, error) {
	query := `
		SELECT id, name, provider_type, base_url, api_key,
		       enabled, priority, config,
		       health_status, last_health_check, error_message,
		       created_at, updated_at
		FROM model_providers
	`

	if enabledOnly {
		query += " WHERE enabled = 1"
	}

	query += " ORDER BY priority DESC, name ASC"

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list model providers: %w", err)
	}
	defer rows.Close()

	var providers []*models.ModelProvider
	for rows.Next() {
		provider := &models.ModelProvider{}
		var configJSON string
		var lastHealthCheck sql.NullTime
		var apiKey sql.NullString
		var errorMessage sql.NullString

		err := rows.Scan(
			&provider.ID,
			&provider.Name,
			&provider.ProviderType,
			&provider.BaseURL,
			&apiKey,
			&provider.Enabled,
			&provider.Priority,
			&configJSON,
			&provider.HealthStatus,
			&lastHealthCheck,
			&errorMessage,
			&provider.CreatedAt,
			&provider.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}

		// Deserialize Config
		provider.ConfigDB = configJSON
		if err := provider.UnmarshalConfigFromDB(); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		// Handle nullable fields
		if apiKey.Valid {
			provider.APIKey = apiKey.String
		}
		if errorMessage.Valid {
			provider.ErrorMessage = errorMessage.String
		}
		if lastHealthCheck.Valid {
			provider.LastHealthCheck = &lastHealthCheck.Time
		}

		providers = append(providers, provider)
	}

	return providers, nil
}

// UpdateModelProviderHealth обновляет health status provider
func (s *SQLiteDB) UpdateModelProviderHealth(ctx context.Context, id string, health models.ModelHealthStatus, errorMsg string) error {
	query := `
		UPDATE model_providers SET
			health_status = ?,
			last_health_check = ?,
			error_message = ?,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now()

	result, err := s.db.ExecContext(ctx, query,
		health,
		now,
		errorMsg,
		now,
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to update provider health: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("model provider not found: %s", id)
	}

	return nil
}

// ========================================
// Model Registry CRUD Operations
// ========================================

// CreateModelRegistry регистрирует новую модель
func (s *SQLiteDB) CreateModelRegistry(ctx context.Context, model *models.ModelRegistry) error {
	// Generate ID if not set
	if model.ID == "" {
		model.ID = generateID()
	}

	// Serialize JSON fields
	if err := model.MarshalToDB(); err != nil {
		return fmt.Errorf("failed to marshal model data: %w", err)
	}

	// Tags: convert []string to comma-separated string for SQLite
	tagsStr := strings.Join(model.Tags, ",")

	query := `
		INSERT INTO model_registry (
			id, model_id, model_name, provider_id,
			capabilities, parameters,
			requires_gpu, min_vram_gb, context_length,
			status, health_status,
			description, tags,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	model.CreatedAt = now
	model.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, query,
		model.ID,
		model.ModelID,
		model.ModelName,
		model.ProviderID,
		model.CapabilitiesDB,
		model.ParametersDB,
		model.RequiresGPU,
		model.MinVRAMGB,
		model.ContextLength,
		model.Status,
		model.HealthStatus,
		model.Description,
		tagsStr,
		model.CreatedAt,
		model.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create model registry: %w", err)
	}

	return nil
}

// GetModelRegistry получает модель по ID
func (s *SQLiteDB) GetModelRegistry(ctx context.Context, id string) (*models.ModelRegistry, error) {
	query := `
		SELECT id, model_id, model_name, provider_id,
		       capabilities, parameters,
		       requires_gpu, min_vram_gb, context_length,
		       status, health_status, last_health_check,
		       description, tags,
		       avg_latency_ms, tokens_per_second, total_requests,
		       created_at, updated_at
		FROM model_registry
		WHERE id = ?
	`

	model := &models.ModelRegistry{}
	var capsJSON, paramsJSON, tagsStr string
	var lastHealthCheck sql.NullTime
	var minVRAM, contextLength sql.NullInt64
	var avgLatency, tokensPerSec sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&model.ID,
		&model.ModelID,
		&model.ModelName,
		&model.ProviderID,
		&capsJSON,
		&paramsJSON,
		&model.RequiresGPU,
		&minVRAM,
		&contextLength,
		&model.Status,
		&model.HealthStatus,
		&lastHealthCheck,
		&model.Description,
		&tagsStr,
		&avgLatency,
		&tokensPerSec,
		&model.TotalRequests,
		&model.CreatedAt,
		&model.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("model not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	// Deserialize JSON fields
	model.CapabilitiesDB = capsJSON
	model.ParametersDB = paramsJSON
	if err := model.UnmarshalFromDB(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal model data: %w", err)
	}

	// Parse tags
	if tagsStr != "" {
		model.Tags = strings.Split(tagsStr, ",")
	}

	// Nullable fields
	if minVRAM.Valid {
		val := int(minVRAM.Int64)
		model.MinVRAMGB = &val
	}
	if contextLength.Valid {
		val := int(contextLength.Int64)
		model.ContextLength = &val
	}
	if lastHealthCheck.Valid {
		model.LastHealthCheck = &lastHealthCheck.Time
	}
	if avgLatency.Valid {
		model.AvgLatencyMs = &avgLatency.Float64
	}
	if tokensPerSec.Valid {
		model.TokensPerSecond = &tokensPerSec.Float64
	}

	return model, nil
}

// GetModelRegistryByModelID получает модель по model_id (уникальному идентификатору модели)
func (s *SQLiteDB) GetModelRegistryByModelID(ctx context.Context, modelID string) (*models.ModelRegistry, error) {
	query := `
		SELECT id, model_id, model_name, provider_id,
		       capabilities, parameters,
		       requires_gpu, min_vram_gb, context_length,
		       status, health_status, last_health_check,
		       description, tags,
		       avg_latency_ms, tokens_per_second, total_requests,
		       created_at, updated_at
		FROM model_registry
		WHERE model_id = ?
	`

	model := &models.ModelRegistry{}
	var capsJSON, paramsJSON, tagsStr string
	var lastHealthCheck sql.NullTime
	var minVRAM, contextLength sql.NullInt64
	var avgLatency, tokensPerSec sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, modelID).Scan(
		&model.ID,
		&model.ModelID,
		&model.ModelName,
		&model.ProviderID,
		&capsJSON,
		&paramsJSON,
		&model.RequiresGPU,
		&minVRAM,
		&contextLength,
		&model.Status,
		&model.HealthStatus,
		&lastHealthCheck,
		&model.Description,
		&tagsStr,
		&avgLatency,
		&tokensPerSec,
		&model.TotalRequests,
		&model.CreatedAt,
		&model.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	// Deserialize JSON fields
	model.CapabilitiesDB = capsJSON
	model.ParametersDB = paramsJSON
	if err := model.UnmarshalFromDB(); err != nil {
		return nil, fmt.Errorf("failed to unmarshal model data: %w", err)
	}

	// Parse tags
	if tagsStr != "" {
		model.Tags = strings.Split(tagsStr, ",")
	}

	// Nullable fields
	if minVRAM.Valid {
		val := int(minVRAM.Int64)
		model.MinVRAMGB = &val
	}
	if contextLength.Valid {
		val := int(contextLength.Int64)
		model.ContextLength = &val
	}
	if lastHealthCheck.Valid {
		model.LastHealthCheck = &lastHealthCheck.Time
	}
	if avgLatency.Valid {
		model.AvgLatencyMs = &avgLatency.Float64
	}
	if tokensPerSec.Valid {
		model.TokensPerSecond = &tokensPerSec.Float64
	}

	return model, nil
}

// UpdateModelRegistry обновляет модель
func (s *SQLiteDB) UpdateModelRegistry(ctx context.Context, model *models.ModelRegistry) error {
	// Serialize JSON fields
	if err := model.MarshalToDB(); err != nil {
		return fmt.Errorf("failed to marshal model data: %w", err)
	}

	// Tags: convert []string to comma-separated string
	tagsStr := strings.Join(model.Tags, ",")

	query := `
		UPDATE model_registry SET
			model_name = ?, provider_id = ?,
			capabilities = ?, parameters = ?,
			requires_gpu = ?, min_vram_gb = ?, context_length = ?,
			status = ?, description = ?, tags = ?,
			updated_at = ?
		WHERE id = ?
	`

	model.UpdatedAt = time.Now()

	result, err := s.db.ExecContext(ctx, query,
		model.ModelName,
		model.ProviderID,
		model.CapabilitiesDB,
		model.ParametersDB,
		model.RequiresGPU,
		model.MinVRAMGB,
		model.ContextLength,
		model.Status,
		model.Description,
		tagsStr,
		model.UpdatedAt,
		model.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update model: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("model not found: %s", model.ID)
	}

	return nil
}

// DeleteModelRegistry удаляет модель из реестра
func (s *SQLiteDB) DeleteModelRegistry(ctx context.Context, id string) error {
	query := `DELETE FROM model_registry WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("model not found: %s", id)
	}

	return nil
}

// ListModelRegistry возвращает список моделей с фильтрацией
func (s *SQLiteDB) ListModelRegistry(ctx context.Context, filter *models.ModelRegistryFilter) ([]*models.ModelRegistry, error) {
	query := `
		SELECT m.id, m.model_id, m.model_name, m.provider_id,
		       m.capabilities, m.parameters,
		       m.requires_gpu, m.min_vram_gb, m.context_length,
		       m.status, m.health_status, m.last_health_check,
		       m.description, m.tags,
		       m.avg_latency_ms, m.tokens_per_second, m.total_requests,
		       m.created_at, m.updated_at
		FROM model_registry m
		WHERE 1=1
	`

	args := []interface{}{}

	// Apply filters
	if filter != nil {
		if filter.ProviderID != "" {
			query += " AND m.provider_id = ?"
			args = append(args, filter.ProviderID)
		}

		if filter.ProviderType != "" {
			query += " AND EXISTS (SELECT 1 FROM model_providers mp WHERE mp.id = m.provider_id AND mp.provider_type = ?)"
			args = append(args, filter.ProviderType)
		}

		if filter.Status != "" {
			query += " AND m.status = ?"
			args = append(args, filter.Status)
		}

		if filter.HealthStatus != "" {
			query += " AND m.health_status = ?"
			args = append(args, filter.HealthStatus)
		}

		if filter.RequiresGPU != nil {
			query += " AND m.requires_gpu = ?"
			args = append(args, *filter.RequiresGPU)
		}

		if filter.Tag != "" {
			query += " AND m.tags LIKE ?"
			args = append(args, "%"+filter.Tag+"%")
		}

		// Capabilities filter (check if JSON contains capability)
		if len(filter.Capabilities) > 0 {
			for _, cap := range filter.Capabilities {
				query += " AND m.capabilities LIKE ?"
				args = append(args, "%"+string(cap)+"%")
			}
		}
	}

	query += " ORDER BY m.model_name ASC"

	// Pagination
	if filter != nil {
		if filter.Limit > 0 {
			query += " LIMIT ?"
			args = append(args, filter.Limit)
		}
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	defer rows.Close()

	var result []*models.ModelRegistry
	for rows.Next() {
		model := &models.ModelRegistry{}
		var capsJSON, paramsJSON, tagsStr string
		var lastHealthCheck sql.NullTime
		var minVRAM, contextLength sql.NullInt64
		var avgLatency, tokensPerSec sql.NullFloat64

		err := rows.Scan(
			&model.ID,
			&model.ModelID,
			&model.ModelName,
			&model.ProviderID,
			&capsJSON,
			&paramsJSON,
			&model.RequiresGPU,
			&minVRAM,
			&contextLength,
			&model.Status,
			&model.HealthStatus,
			&lastHealthCheck,
			&model.Description,
			&tagsStr,
			&avgLatency,
			&tokensPerSec,
			&model.TotalRequests,
			&model.CreatedAt,
			&model.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model: %w", err)
		}

		// Deserialize JSON fields
		model.CapabilitiesDB = capsJSON
		model.ParametersDB = paramsJSON
		if err := model.UnmarshalFromDB(); err != nil {
			return nil, fmt.Errorf("failed to unmarshal model data: %w", err)
		}

		// Parse tags
		if tagsStr != "" {
			model.Tags = strings.Split(tagsStr, ",")
		}

		// Nullable fields
		if minVRAM.Valid {
			val := int(minVRAM.Int64)
			model.MinVRAMGB = &val
		}
		if contextLength.Valid {
			val := int(contextLength.Int64)
			model.ContextLength = &val
		}
		if lastHealthCheck.Valid {
			model.LastHealthCheck = &lastHealthCheck.Time
		}
		if avgLatency.Valid {
			model.AvgLatencyMs = &avgLatency.Float64
		}
		if tokensPerSec.Valid {
			model.TokensPerSecond = &tokensPerSec.Float64
		}

		result = append(result, model)
	}

	return result, nil
}

// UpdateModelRegistryHealth обновляет health status модели
func (s *SQLiteDB) UpdateModelRegistryHealth(ctx context.Context, id string, health models.ModelHealthStatus) error {
	query := `
		UPDATE model_registry SET
			health_status = ?,
			last_health_check = ?,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now()

	result, err := s.db.ExecContext(ctx, query,
		health,
		now,
		now,
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to update model health: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("model not found: %s", id)
	}

	return nil
}

// UpdateModelRegistryMetrics обновляет performance metrics модели
func (s *SQLiteDB) UpdateModelRegistryMetrics(ctx context.Context, id string, latency float64, tokensPerSec float64) error {
	query := `
		UPDATE model_registry SET
			avg_latency_ms = ?,
			tokens_per_second = ?,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now()

	result, err := s.db.ExecContext(ctx, query,
		latency,
		tokensPerSec,
		now,
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to update model metrics: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("model not found: %s", id)
	}

	return nil
}

// IncrementModelRequests увеличивает счетчик запросов к модели
func (s *SQLiteDB) IncrementModelRequests(ctx context.Context, id string) error {
	query := `
		UPDATE model_registry SET
			total_requests = total_requests + 1,
			updated_at = ?
		WHERE id = ?
	`

	now := time.Now()

	result, err := s.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to increment model requests: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("model not found: %s", id)
	}

	return nil
}

// GetModelRegistryStats возвращает статистику registry
func (s *SQLiteDB) GetModelRegistryStats(ctx context.Context) (*models.ModelRegistryStats, error) {
	stats := &models.ModelRegistryStats{
		ModelsByProvider:   make(map[string]int),
		ModelsByCapability: make(map[models.ModelCapability]int),
	}

	// Total models
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM model_registry").Scan(&stats.TotalModels)
	if err != nil {
		return nil, fmt.Errorf("failed to count total models: %w", err)
	}

	// Active models
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM model_registry WHERE status = 'active'").Scan(&stats.ActiveModels)
	if err != nil {
		return nil, fmt.Errorf("failed to count active models: %w", err)
	}

	// Healthy models
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM model_registry WHERE health_status = 'healthy'").Scan(&stats.HealthyModels)
	if err != nil {
		return nil, fmt.Errorf("failed to count healthy models: %w", err)
	}

	// Total providers
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM model_providers").Scan(&stats.TotalProviders)
	if err != nil {
		return nil, fmt.Errorf("failed to count total providers: %w", err)
	}

	// Enabled providers
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM model_providers WHERE enabled = 1").Scan(&stats.EnabledProviders)
	if err != nil {
		return nil, fmt.Errorf("failed to count enabled providers: %w", err)
	}

	// Models by provider
	rows, err := s.db.QueryContext(ctx, `
		SELECT mp.name, COUNT(mr.id)
		FROM model_providers mp
		LEFT JOIN model_registry mr ON mp.id = mr.provider_id
		GROUP BY mp.name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to count models by provider: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var providerName string
		var count int
		if err := rows.Scan(&providerName, &count); err != nil {
			return nil, fmt.Errorf("failed to scan provider stats: %w", err)
		}
		stats.ModelsByProvider[providerName] = count
	}

	// Models by capability (approximation via JSON parsing)
	// Note: SQLite не поддерживает native JSONB queries, так что это будет простой подсчет моделей с capabilities != []
	// Для точных данных нужно парсить JSON в коде
	capabilityRows, err := s.db.QueryContext(ctx, "SELECT capabilities FROM model_registry WHERE capabilities != '[]'")
	if err != nil {
		return nil, fmt.Errorf("failed to query capabilities: %w", err)
	}
	defer capabilityRows.Close()

	for capabilityRows.Next() {
		var capsJSON string
		if err := capabilityRows.Scan(&capsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan capabilities: %w", err)
		}

		var caps []models.ModelCapability
		if err := json.Unmarshal([]byte(capsJSON), &caps); err != nil {
			continue // Skip malformed JSON
		}

		for _, cap := range caps {
			stats.ModelsByCapability[cap]++
		}
	}

	return stats, nil
}


package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"aigateway/internal/models"
	"github.com/lib/pq"
)

// ========================================
// Model Providers CRUD Operations
// ========================================

// CreateModelProvider создает нового model provider
func (db *PostgreSQLDB) CreateModelProvider(ctx context.Context, provider *models.ModelProvider) error {
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
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	now := time.Now()
	provider.CreatedAt = now
	provider.UpdatedAt = now

	_, err := db.db.ExecContext(ctx, query,
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
func (db *PostgreSQLDB) GetModelProvider(ctx context.Context, id string) (*models.ModelProvider, error) {
	query := `
		SELECT id, name, provider_type, base_url, api_key,
		       enabled, priority, config,
		       health_status, last_health_check, error_message,
		       created_at, updated_at
		FROM model_providers
		WHERE id = $1
	`

	provider := &models.ModelProvider{}
	var configJSON string
	var lastHealthCheck sql.NullTime
	var apiKey sql.NullString
	var errorMessage sql.NullString

	err := db.db.QueryRowContext(ctx, query, id).Scan(
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
func (db *PostgreSQLDB) GetModelProviderByName(ctx context.Context, name string) (*models.ModelProvider, error) {
	query := `
		SELECT id, name, provider_type, base_url, api_key,
		       enabled, priority, config,
		       health_status, last_health_check, error_message,
		       created_at, updated_at
		FROM model_providers
		WHERE name = $1
	`

	provider := &models.ModelProvider{}
	var configJSON string
	var lastHealthCheck sql.NullTime
	var apiKey sql.NullString
	var errorMessage sql.NullString

	err := db.db.QueryRowContext(ctx, query, name).Scan(
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
func (db *PostgreSQLDB) UpdateModelProvider(ctx context.Context, provider *models.ModelProvider) error {
	// Serialize Config to JSON
	if err := provider.MarshalConfigToDB(); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		UPDATE model_providers SET
			name = $1, provider_type = $2, base_url = $3, api_key = $4,
			enabled = $5, priority = $6, config = $7,
			health_status = $8, error_message = $9, updated_at = $10
		WHERE id = $11
	`

	provider.UpdatedAt = time.Now()

	result, err := db.db.ExecContext(ctx, query,
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
func (db *PostgreSQLDB) DeleteModelProvider(ctx context.Context, id string) error {
	query := `DELETE FROM model_providers WHERE id = $1`

	result, err := db.db.ExecContext(ctx, query, id)
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
func (db *PostgreSQLDB) ListModelProviders(ctx context.Context, enabledOnly bool) ([]*models.ModelProvider, error) {
	query := `
		SELECT id, name, provider_type, base_url, api_key,
		       enabled, priority, config,
		       health_status, last_health_check, error_message,
		       created_at, updated_at
		FROM model_providers
	`

	if enabledOnly {
		query += " WHERE enabled = true"
	}

	query += " ORDER BY priority DESC, name ASC"

	rows, err := db.db.QueryContext(ctx, query)
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
func (db *PostgreSQLDB) UpdateModelProviderHealth(ctx context.Context, id string, health models.ModelHealthStatus, errorMsg string) error {
	query := `
		UPDATE model_providers SET
			health_status = $1,
			last_health_check = $2,
			error_message = $3,
			updated_at = $4
		WHERE id = $5
	`

	now := time.Now()

	result, err := db.db.ExecContext(ctx, query,
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
func (db *PostgreSQLDB) CreateModelRegistry(ctx context.Context, model *models.ModelRegistry) error {
	// Generate ID if not set
	if model.ID == "" {
		model.ID = generateID()
	}

	// For PostgreSQL JSONB columns, marshal to []byte
	var capabilitiesJSON, parametersJSON []byte
	var err error

	// Capabilities
	if model.Capabilities == nil {
		capabilitiesJSON = []byte("[]")
	} else {
		capabilitiesJSON, err = json.Marshal(model.Capabilities)
		if err != nil {
			return fmt.Errorf("failed to marshal capabilities: %w", err)
		}
	}

	// Parameters
	if model.Parameters == nil {
		parametersJSON = []byte("{}")
	} else {
		parametersJSON, err = json.Marshal(model.Parameters)
		if err != nil {
			return fmt.Errorf("failed to marshal parameters: %w", err)
		}
	}

	// Tags: for PostgreSQL, use TEXT[] array
	// pq driver supports pq.Array(model.Tags)
	var tagsArray interface{}
	if model.Tags == nil || len(model.Tags) == 0 {
		tagsArray = pq.Array([]string{}) // Empty array
	} else {
		tagsArray = pq.Array(model.Tags)
	}

	query := `
		INSERT INTO model_registry (
			id, model_id, model_name, provider_id,
			capabilities, parameters,
			requires_gpu, min_vram_gb, context_length,
			status, health_status,
			description, tags,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	now := time.Now()
	model.CreatedAt = now
	model.UpdatedAt = now

	_, err = db.db.ExecContext(ctx, query,
		model.ID,
		model.ModelID,
		model.ModelName,
		model.ProviderID,
		capabilitiesJSON, // []byte for JSONB
		parametersJSON,   // []byte for JSONB
		model.RequiresGPU,
		model.MinVRAMGB,
		model.ContextLength,
		model.Status,
		model.HealthStatus,
		model.Description,
		tagsArray, // pq.Array for TEXT[]
		model.CreatedAt,
		model.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create model registry: %w", err)
	}

	return nil
}

// GetModelRegistry получает модель по ID
func (db *PostgreSQLDB) GetModelRegistry(ctx context.Context, id string) (*models.ModelRegistry, error) {
	query := `
		SELECT id, model_id, model_name, provider_id,
		       capabilities, parameters,
		       requires_gpu, min_vram_gb, context_length,
		       status, health_status, last_health_check,
		       description, tags,
		       avg_latency_ms, tokens_per_second, total_requests,
		       created_at, updated_at
		FROM model_registry
		WHERE id = $1
	`

	model := &models.ModelRegistry{}
	var capabilitiesJSON, parametersJSON []byte
	var tags []string
	var lastHealthCheck sql.NullTime
	var minVRAM, contextLength sql.NullInt64
	var avgLatency, tokensPerSec sql.NullFloat64

	err := db.db.QueryRowContext(ctx, query, id).Scan(
		&model.ID,
		&model.ModelID,
		&model.ModelName,
		&model.ProviderID,
		&capabilitiesJSON,
		&parametersJSON,
		&model.RequiresGPU,
		&minVRAM,
		&contextLength,
		&model.Status,
		&model.HealthStatus,
		&lastHealthCheck,
		&model.Description,
		pq.Array(&tags), // Read TEXT[] array
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

	// Deserialize JSONB fields
	if len(capabilitiesJSON) > 0 {
		if err := json.Unmarshal(capabilitiesJSON, &model.Capabilities); err != nil {
			return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
		}
	} else {
		model.Capabilities = []models.ModelCapability{}
	}

	if len(parametersJSON) > 0 {
		if err := json.Unmarshal(parametersJSON, &model.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
	} else {
		model.Parameters = make(map[string]interface{})
	}

	// Set tags
	model.Tags = tags

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
func (db *PostgreSQLDB) GetModelRegistryByModelID(ctx context.Context, modelID string) (*models.ModelRegistry, error) {
	query := `
		SELECT id, model_id, model_name, provider_id,
		       capabilities, parameters,
		       requires_gpu, min_vram_gb, context_length,
		       status, health_status, last_health_check,
		       description, tags,
		       avg_latency_ms, tokens_per_second, total_requests,
		       created_at, updated_at
		FROM model_registry
		WHERE model_id = $1
	`

	model := &models.ModelRegistry{}
	var capabilitiesJSON, parametersJSON []byte
	var tags []string
	var lastHealthCheck sql.NullTime
	var minVRAM, contextLength sql.NullInt64
	var avgLatency, tokensPerSec sql.NullFloat64

	err := db.db.QueryRowContext(ctx, query, modelID).Scan(
		&model.ID,
		&model.ModelID,
		&model.ModelName,
		&model.ProviderID,
		&capabilitiesJSON,
		&parametersJSON,
		&model.RequiresGPU,
		&minVRAM,
		&contextLength,
		&model.Status,
		&model.HealthStatus,
		&lastHealthCheck,
		&model.Description,
		pq.Array(&tags), // Read TEXT[] array
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

	// Deserialize JSONB fields
	if len(capabilitiesJSON) > 0 {
		if err := json.Unmarshal(capabilitiesJSON, &model.Capabilities); err != nil {
			return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
		}
	} else {
		model.Capabilities = []models.ModelCapability{}
	}

	if len(parametersJSON) > 0 {
		if err := json.Unmarshal(parametersJSON, &model.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
	} else {
		model.Parameters = make(map[string]interface{})
	}

	// Set tags
	model.Tags = tags

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
func (db *PostgreSQLDB) UpdateModelRegistry(ctx context.Context, model *models.ModelRegistry) error {
	// For PostgreSQL JSONB columns, marshal to []byte
	var capabilitiesJSON, parametersJSON []byte
	var err error

	// Capabilities
	if model.Capabilities == nil {
		capabilitiesJSON = []byte("[]")
	} else {
		capabilitiesJSON, err = json.Marshal(model.Capabilities)
		if err != nil {
			return fmt.Errorf("failed to marshal capabilities: %w", err)
		}
	}

	// Parameters
	if model.Parameters == nil {
		parametersJSON = []byte("{}")
	} else {
		parametersJSON, err = json.Marshal(model.Parameters)
		if err != nil {
			return fmt.Errorf("failed to marshal parameters: %w", err)
		}
	}

	// Tags: for PostgreSQL, use TEXT[] array
	var tagsArray interface{}
	if model.Tags == nil || len(model.Tags) == 0 {
		tagsArray = pq.Array([]string{})
	} else {
		tagsArray = pq.Array(model.Tags)
	}

	query := `
		UPDATE model_registry SET
			model_name = $1, provider_id = $2,
			capabilities = $3, parameters = $4,
			requires_gpu = $5, min_vram_gb = $6, context_length = $7,
			status = $8, description = $9, tags = $10,
			updated_at = $11
		WHERE id = $12
	`

	model.UpdatedAt = time.Now()

	result, err := db.db.ExecContext(ctx, query,
		model.ModelName,
		model.ProviderID,
		capabilitiesJSON, // []byte for JSONB
		parametersJSON,   // []byte for JSONB
		model.RequiresGPU,
		model.MinVRAMGB,
		model.ContextLength,
		model.Status,
		model.Description,
		tagsArray, // pq.Array for TEXT[]
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
func (db *PostgreSQLDB) DeleteModelRegistry(ctx context.Context, id string) error {
	query := `DELETE FROM model_registry WHERE id = $1`

	result, err := db.db.ExecContext(ctx, query, id)
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

// DeleteModelRegistryByProviderID удаляет все модели провайдера из registry
func (db *PostgreSQLDB) DeleteModelRegistryByProviderID(ctx context.Context, providerID string) error {
	query := `DELETE FROM model_registry WHERE provider_id = $1`
	_, err := db.db.ExecContext(ctx, query, providerID)
	if err != nil {
		return fmt.Errorf("failed to delete registry models for provider %s: %w", providerID, err)
	}
	return nil
}

// ListModelRegistry возвращает список моделей с фильтрацией
func (db *PostgreSQLDB) ListModelRegistry(ctx context.Context, filter *models.ModelRegistryFilter) ([]*models.ModelRegistry, error) {
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
	paramIndex := 1 // PostgreSQL uses $1, $2, $3, etc

	// Apply filters
	if filter != nil {
		if filter.ProviderID != "" {
			query += fmt.Sprintf(" AND m.provider_id = $%d", paramIndex)
			args = append(args, filter.ProviderID)
			paramIndex++
		}

		if filter.ProviderType != "" {
			query += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM model_providers mp WHERE mp.id = m.provider_id AND mp.provider_type = $%d)", paramIndex)
			args = append(args, filter.ProviderType)
			paramIndex++
		}

		if filter.Status != "" {
			query += fmt.Sprintf(" AND m.status = $%d", paramIndex)
			args = append(args, filter.Status)
			paramIndex++
		}

		if filter.HealthStatus != "" {
			query += fmt.Sprintf(" AND m.health_status = $%d", paramIndex)
			args = append(args, filter.HealthStatus)
			paramIndex++
		}

		if filter.RequiresGPU != nil {
			query += fmt.Sprintf(" AND m.requires_gpu = $%d", paramIndex)
			args = append(args, *filter.RequiresGPU)
			paramIndex++
		}

		if filter.Tag != "" {
			// PostgreSQL: TEXT[] @> ARRAY['tag'] or 'tag' = ANY(tags)
			query += fmt.Sprintf(" AND $%d = ANY(m.tags)", paramIndex)
			args = append(args, filter.Tag)
			paramIndex++
		}

		// Capabilities filter (check if JSONB contains capability)
		if len(filter.Capabilities) > 0 {
			for _, cap := range filter.Capabilities {
				// PostgreSQL JSONB containment: capabilities @> '"chat"'::jsonb
				query += fmt.Sprintf(" AND m.capabilities::jsonb @> $%d::jsonb", paramIndex)
				args = append(args, fmt.Sprintf(`"%s"`, string(cap)))
				paramIndex++
			}
		}
	}

	query += " ORDER BY m.model_name ASC"

	// Pagination
	if filter != nil {
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", paramIndex)
			args = append(args, filter.Limit)
			paramIndex++
		}
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", paramIndex)
			args = append(args, filter.Offset)
			paramIndex++
		}
	}

	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	defer rows.Close()

	var result []*models.ModelRegistry
	for rows.Next() {
		model := &models.ModelRegistry{}
		var capabilitiesJSON, parametersJSON []byte
		var tags []string
		var lastHealthCheck sql.NullTime
		var minVRAM, contextLength sql.NullInt64
		var avgLatency, tokensPerSec sql.NullFloat64

		err := rows.Scan(
			&model.ID,
			&model.ModelID,
			&model.ModelName,
			&model.ProviderID,
			&capabilitiesJSON,
			&parametersJSON,
			&model.RequiresGPU,
			&minVRAM,
			&contextLength,
			&model.Status,
			&model.HealthStatus,
			&lastHealthCheck,
			&model.Description,
			pq.Array(&tags), // Read TEXT[] array
			&avgLatency,
			&tokensPerSec,
			&model.TotalRequests,
			&model.CreatedAt,
			&model.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model: %w", err)
		}

		// Deserialize JSONB fields
		if len(capabilitiesJSON) > 0 {
			if err := json.Unmarshal(capabilitiesJSON, &model.Capabilities); err != nil {
				return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
			}
		} else {
			model.Capabilities = []models.ModelCapability{}
		}

		if len(parametersJSON) > 0 {
			if err := json.Unmarshal(parametersJSON, &model.Parameters); err != nil {
				return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
			}
		} else {
			model.Parameters = make(map[string]interface{})
		}

		// Set tags
		model.Tags = tags

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
func (db *PostgreSQLDB) UpdateModelRegistryHealth(ctx context.Context, id string, health models.ModelHealthStatus) error {
	query := `
		UPDATE model_registry SET
			health_status = $1,
			last_health_check = $2,
			updated_at = $3
		WHERE id = $4
	`

	now := time.Now()

	result, err := db.db.ExecContext(ctx, query,
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
func (db *PostgreSQLDB) UpdateModelRegistryMetrics(ctx context.Context, id string, latency float64, tokensPerSec float64) error {
	query := `
		UPDATE model_registry SET
			avg_latency_ms = $1,
			tokens_per_second = $2,
			updated_at = $3
		WHERE id = $4
	`

	now := time.Now()

	result, err := db.db.ExecContext(ctx, query,
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
func (db *PostgreSQLDB) IncrementModelRequests(ctx context.Context, id string) error {
	query := `
		UPDATE model_registry SET
			total_requests = total_requests + 1,
			updated_at = $1
		WHERE id = $2
	`

	now := time.Now()

	result, err := db.db.ExecContext(ctx, query, now, id)
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
func (db *PostgreSQLDB) GetModelRegistryStats(ctx context.Context) (*models.ModelRegistryStats, error) {
	stats := &models.ModelRegistryStats{
		ModelsByProvider:   make(map[string]int),
		ModelsByCapability: make(map[models.ModelCapability]int),
	}

	// Total models
	err := db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM model_registry").Scan(&stats.TotalModels)
	if err != nil {
		return nil, fmt.Errorf("failed to count total models: %w", err)
	}

	// Active models
	err = db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM model_registry WHERE status = 'active'").Scan(&stats.ActiveModels)
	if err != nil {
		return nil, fmt.Errorf("failed to count active models: %w", err)
	}

	// Healthy models
	err = db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM model_registry WHERE health_status = 'healthy'").Scan(&stats.HealthyModels)
	if err != nil {
		return nil, fmt.Errorf("failed to count healthy models: %w", err)
	}

	// Total providers
	err = db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM model_providers").Scan(&stats.TotalProviders)
	if err != nil {
		return nil, fmt.Errorf("failed to count total providers: %w", err)
	}

	// Enabled providers
	err = db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM model_providers WHERE enabled = true").Scan(&stats.EnabledProviders)
	if err != nil {
		return nil, fmt.Errorf("failed to count enabled providers: %w", err)
	}

	// Models by provider
	rows, err := db.db.QueryContext(ctx, `
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
	capabilityRows, err := db.db.QueryContext(ctx, "SELECT capabilities FROM model_registry WHERE capabilities != '[]'")
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

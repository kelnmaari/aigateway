// Package storage provides PostgreSQL implementation for GitLab data access
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aigateway/internal/models"

	"github.com/google/uuid"
)

// PostgresStore implements Store for PostgreSQL
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgreSQL store
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// ============================================================================
// Integration Store Implementation
// ============================================================================

func (s *PostgresStore) CreateIntegration(ctx context.Context, integration *models.GitLabIntegration) error {
	if integration.ID == "" {
		integration.ID = uuid.New().String()
	}
	
	now := time.Now()
	integration.CreatedAt = now
	integration.UpdatedAt = now

	settingsJSON, err := json.Marshal(integration.Settings)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	query := `
		INSERT INTO gitlab_integrations (id, name, base_url, access_token, webhook_secret, status, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = s.db.ExecContext(ctx, query,
		integration.ID,
		integration.Name,
		integration.BaseURL,
		integration.AccessToken,
		integration.WebhookSecret,
		integration.Status,
		string(settingsJSON),
		integration.CreatedAt,
		integration.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert integration: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetIntegration(ctx context.Context, id string) (*models.GitLabIntegration, error) {
	query := `
		SELECT id, name, base_url, access_token, webhook_secret, status, last_sync_at, last_error, settings, created_at, updated_at,
		       (SELECT COUNT(*) FROM gitlab_projects WHERE integration_id = gitlab_integrations.id) as project_count
		FROM gitlab_integrations
		WHERE id = $1
	`

	var integration models.GitLabIntegration
	var lastSyncAt sql.NullTime
	var lastError sql.NullString
	var settingsJSON string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&integration.ID,
		&integration.Name,
		&integration.BaseURL,
		&integration.AccessToken,
		&integration.WebhookSecret,
		&integration.Status,
		&lastSyncAt,
		&lastError,
		&settingsJSON,
		&integration.CreatedAt,
		&integration.UpdatedAt,
		&integration.ProjectCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query integration: %w", err)
	}

	if lastSyncAt.Valid {
		integration.LastSyncAt = &lastSyncAt.Time
	}
	if lastError.Valid {
		integration.LastError = lastError.String
	}
	if err := json.Unmarshal([]byte(settingsJSON), &integration.Settings); err != nil {
		// Ignore JSON errors, use default settings
	}

	return &integration, nil
}

func (s *PostgresStore) ListIntegrations(ctx context.Context, req *models.GitLabIntegrationListRequest) ([]models.GitLabIntegration, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if req.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *req.Status)
		argNum++
	}
	if req.Search != nil && *req.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR base_url ILIKE $%d)", argNum, argNum+1))
		search := "%" + *req.Search + "%"
		args = append(args, search, search)
		argNum += 2
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM gitlab_integrations %s", whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count integrations: %w", err)
	}

	// Fetch items
	query := fmt.Sprintf(`
		SELECT id, name, base_url, access_token, webhook_secret, status, last_sync_at, last_error, settings, created_at, updated_at,
		       (SELECT COUNT(*) FROM gitlab_projects WHERE integration_id = gitlab_integrations.id) as project_count
		FROM gitlab_integrations
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query integrations: %w", err)
	}
	defer rows.Close()

	var integrations []models.GitLabIntegration
	for rows.Next() {
		var integration models.GitLabIntegration
		var lastSyncAt sql.NullTime
		var lastError sql.NullString
		var settingsJSON string

		if err := rows.Scan(
			&integration.ID,
			&integration.Name,
			&integration.BaseURL,
			&integration.AccessToken,
			&integration.WebhookSecret,
			&integration.Status,
			&lastSyncAt,
			&lastError,
			&settingsJSON,
			&integration.CreatedAt,
			&integration.UpdatedAt,
			&integration.ProjectCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan integration: %w", err)
		}

		if lastSyncAt.Valid {
			integration.LastSyncAt = &lastSyncAt.Time
		}
		if lastError.Valid {
			integration.LastError = lastError.String
		}
		if err := json.Unmarshal([]byte(settingsJSON), &integration.Settings); err != nil {
			// Ignore JSON errors
		}

		integrations = append(integrations, integration)
	}

	return integrations, total, nil
}

func (s *PostgresStore) ListIntegrationsByOwner(ctx context.Context, ownerID string) ([]*models.GitLabIntegration, error) {
	query := `
		SELECT id, name, base_url, access_token, webhook_secret, status, last_sync_at, last_error, settings, created_at, updated_at
		FROM gitlab_integrations
		WHERE owner_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("query integrations by owner: %w", err)
	}
	defer rows.Close()

	var integrations []*models.GitLabIntegration
	for rows.Next() {
		var integration models.GitLabIntegration
		var lastSyncAt sql.NullTime
		var lastError sql.NullString
		var settingsJSON string

		if err := rows.Scan(
			&integration.ID,
			&integration.Name,
			&integration.BaseURL,
			&integration.AccessToken,
			&integration.WebhookSecret,
			&integration.Status,
			&lastSyncAt,
			&lastError,
			&settingsJSON,
			&integration.CreatedAt,
			&integration.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan integration: %w", err)
		}

		if lastSyncAt.Valid {
			integration.LastSyncAt = &lastSyncAt.Time
		}
		if lastError.Valid {
			integration.LastError = lastError.String
		}
		if settingsJSON != "" {
			if err := json.Unmarshal([]byte(settingsJSON), &integration.Settings); err != nil {
				// Ignore JSON errors
			}
		}

		integrations = append(integrations, &integration)
	}

	return integrations, nil
}

func (s *PostgresStore) UpdateIntegration(ctx context.Context, id string, req *models.UpdateGitLabIntegrationRequest) error {
	var sets []string
	var args []interface{}
	argNum := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argNum))
		args = append(args, *req.Name)
		argNum++
	}
	if req.BaseURL != nil {
		sets = append(sets, fmt.Sprintf("base_url = $%d", argNum))
		args = append(args, *req.BaseURL)
		argNum++
	}
	if req.AccessToken != nil {
		sets = append(sets, fmt.Sprintf("access_token = $%d", argNum))
		args = append(args, *req.AccessToken)
		argNum++
	}
	if req.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *req.Status)
		argNum++
	}
	if req.Settings != nil {
		settingsJSON, err := json.Marshal(req.Settings)
		if err != nil {
			return fmt.Errorf("marshal settings: %w", err)
		}
		sets = append(sets, fmt.Sprintf("settings = $%d", argNum))
		args = append(args, string(settingsJSON))
		argNum++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++
	args = append(args, id)

	query := fmt.Sprintf("UPDATE gitlab_integrations SET %s WHERE id = $%d", strings.Join(sets, ", "), argNum)
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update integration: %w", err)
	}

	return nil
}

func (s *PostgresStore) DeleteIntegration(ctx context.Context, id string) error {
	// Projects will be cascade deleted due to foreign key
	_, err := s.db.ExecContext(ctx, "DELETE FROM gitlab_integrations WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete integration: %w", err)
	}
	return nil
}

func (s *PostgresStore) UpdateIntegrationStatus(ctx context.Context, id string, status models.GitLabIntegrationStatus, lastError string) error {
	query := "UPDATE gitlab_integrations SET status = $1, last_error = $2, updated_at = $3 WHERE id = $4"
	_, err := s.db.ExecContext(ctx, query, status, lastError, time.Now(), id)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

func (s *PostgresStore) UpdateIntegrationLastSync(ctx context.Context, id string) error {
	query := "UPDATE gitlab_integrations SET last_sync_at = $1, updated_at = $2 WHERE id = $3"
	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, now, now, id)
	if err != nil {
		return fmt.Errorf("update last_sync: %w", err)
	}
	return nil
}

// ============================================================================
// Project Store Implementation
// ============================================================================

func (s *PostgresStore) CreateProject(ctx context.Context, project *models.GitLabProject) error {
	if project.ID == "" {
		project.ID = uuid.New().String()
	}
	
	now := time.Now()
	project.CreatedAt = now
	project.UpdatedAt = now

	settingsJSON, err := json.Marshal(project.Settings)
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	query := `
		INSERT INTO gitlab_projects (
			id, integration_id, gitlab_project_id, name, path_with_namespace, webhook_id,
			status, auto_review, analysis_model_id, embedding_model_id, review_prompt, settings,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err = s.db.ExecContext(ctx, query,
		project.ID,
		project.IntegrationID,
		project.GitLabProjectID,
		project.Name,
		project.PathWithNamespace,
		project.WebhookID,
		project.Status,
		project.AutoReview,
		project.AnalysisModelID,
		project.EmbeddingModelID,
		project.ReviewPrompt,
		string(settingsJSON),
		project.CreatedAt,
		project.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert project: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetProject(ctx context.Context, id string) (*models.GitLabProject, error) {
	query := `
		SELECT p.id, p.integration_id, p.gitlab_project_id, p.name, p.path_with_namespace, p.default_branch, p.webhook_id,
		       p.status, p.auto_review, p.analysis_model_id, p.embedding_model_id, p.review_prompt, p.settings,
		       p.index_status, p.last_indexed_at,
		       p.created_at, p.updated_at,
		       i.name as integration_name,
		       (SELECT COUNT(*) FROM gitlab_mr_reviews WHERE project_id = p.id) as review_count
		FROM gitlab_projects p
		LEFT JOIN gitlab_integrations i ON i.id = p.integration_id
		WHERE p.id = $1
	`

	var project models.GitLabProject
	var webhookID sql.NullInt64
	var reviewPrompt sql.NullString
	var settingsJSON string
	var integrationName sql.NullString
	var indexStatus sql.NullString
	var lastIndexedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&project.ID,
		&project.IntegrationID,
		&project.GitLabProjectID,
		&project.Name,
		&project.PathWithNamespace,
		&project.DefaultBranch,
		&webhookID,
		&project.Status,
		&project.AutoReview,
		&project.AnalysisModelID,
		&project.EmbeddingModelID,
		&reviewPrompt,
		&settingsJSON,
		&indexStatus,
		&lastIndexedAt,
		&project.CreatedAt,
		&project.UpdatedAt,
		&integrationName,
		&project.ReviewCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query project: %w", err)
	}

	if webhookID.Valid {
		wid := webhookID.Int64
		project.WebhookID = &wid
	}
	if reviewPrompt.Valid {
		project.ReviewPrompt = reviewPrompt.String
	}
	if integrationName.Valid {
		project.IntegrationName = integrationName.String
	}
	if indexStatus.Valid {
		project.IndexStatus = indexStatus.String
	}
	if lastIndexedAt.Valid {
		project.LastIndexedAt = &lastIndexedAt.Time
	}
	if err := json.Unmarshal([]byte(settingsJSON), &project.Settings); err != nil {
		// Ignore JSON errors
	}

	return &project, nil
}

func (s *PostgresStore) GetProjectByGitLabID(ctx context.Context, integrationID string, gitlabProjectID int64) (*models.GitLabProject, error) {
	query := `
		SELECT p.id, p.integration_id, p.gitlab_project_id, p.name, p.path_with_namespace, p.webhook_id,
		       p.status, p.auto_review, p.analysis_model_id, p.embedding_model_id, p.review_prompt, p.settings,
		       p.created_at, p.updated_at
		FROM gitlab_projects p
		WHERE p.integration_id = $1 AND p.gitlab_project_id = $2
	`

	var project models.GitLabProject
	var webhookID sql.NullInt64
	var reviewPrompt sql.NullString
	var settingsJSON string

	err := s.db.QueryRowContext(ctx, query, integrationID, gitlabProjectID).Scan(
		&project.ID,
		&project.IntegrationID,
		&project.GitLabProjectID,
		&project.Name,
		&project.PathWithNamespace,
		&webhookID,
		&project.Status,
		&project.AutoReview,
		&project.AnalysisModelID,
		&project.EmbeddingModelID,
		&reviewPrompt,
		&settingsJSON,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query project: %w", err)
	}

	if webhookID.Valid {
		wid := webhookID.Int64
		project.WebhookID = &wid
	}
	if reviewPrompt.Valid {
		project.ReviewPrompt = reviewPrompt.String
	}
	if err := json.Unmarshal([]byte(settingsJSON), &project.Settings); err != nil {
		// Ignore JSON errors
	}

	return &project, nil
}

func (s *PostgresStore) ListProjects(ctx context.Context, req *models.GitLabProjectListRequest) ([]models.GitLabProject, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if req.IntegrationID != nil {
		conditions = append(conditions, fmt.Sprintf("p.integration_id = $%d", argNum))
		args = append(args, *req.IntegrationID)
		argNum++
	}
	if req.Status != nil {
		conditions = append(conditions, fmt.Sprintf("p.status = $%d", argNum))
		args = append(args, *req.Status)
		argNum++
	}
	if req.Search != nil && *req.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(p.name ILIKE $%d OR p.path_with_namespace ILIKE $%d)", argNum, argNum+1))
		search := "%" + *req.Search + "%"
		args = append(args, search, search)
		argNum += 2
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM gitlab_projects p %s", whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count projects: %w", err)
	}

	// Fetch items
	query := fmt.Sprintf(`
		SELECT p.id, p.integration_id, p.gitlab_project_id, p.name, p.path_with_namespace, p.default_branch, p.webhook_id,
		       p.status, p.auto_review, p.analysis_model_id, p.embedding_model_id, p.review_prompt, p.settings,
		       p.index_status, p.last_indexed_at,
		       p.created_at, p.updated_at,
		       i.name as integration_name,
		       (SELECT COUNT(*) FROM gitlab_mr_reviews WHERE project_id = p.id) as review_count
		FROM gitlab_projects p
		LEFT JOIN gitlab_integrations i ON i.id = p.integration_id
		%s
		ORDER BY p.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query projects: %w", err)
	}
	defer rows.Close()

	var projects []models.GitLabProject
	for rows.Next() {
		var project models.GitLabProject
		var webhookID sql.NullInt64
		var reviewPrompt sql.NullString
		var settingsJSON string
		var integrationName sql.NullString
		var indexStatus sql.NullString
		var lastIndexedAt sql.NullTime

		if err := rows.Scan(
			&project.ID,
			&project.IntegrationID,
			&project.GitLabProjectID,
			&project.Name,
			&project.PathWithNamespace,
			&project.DefaultBranch,
			&webhookID,
			&project.Status,
			&project.AutoReview,
			&project.AnalysisModelID,
			&project.EmbeddingModelID,
			&reviewPrompt,
			&settingsJSON,
			&indexStatus,
			&lastIndexedAt,
			&project.CreatedAt,
			&project.UpdatedAt,
			&integrationName,
			&project.ReviewCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan project: %w", err)
		}

		if webhookID.Valid {
			wid := webhookID.Int64
			project.WebhookID = &wid
		}
		if reviewPrompt.Valid {
			project.ReviewPrompt = reviewPrompt.String
		}
		if integrationName.Valid {
			project.IntegrationName = integrationName.String
		}
		if indexStatus.Valid {
			project.IndexStatus = indexStatus.String
		}
		if lastIndexedAt.Valid {
			project.LastIndexedAt = &lastIndexedAt.Time
		}
		if err := json.Unmarshal([]byte(settingsJSON), &project.Settings); err != nil {
			// Ignore JSON errors
		}

		projects = append(projects, project)
	}

	return projects, total, nil
}

func (s *PostgresStore) ListProjectsByIntegration(ctx context.Context, integrationID string) ([]*models.GitLabProject, error) {
	query := `
		SELECT id, integration_id, gitlab_project_id, name, path_with_namespace,
		       auto_review, webhook_id, status, analysis_model_id, embedding_model_id, review_prompt,
		       settings, created_at, updated_at
		FROM gitlab_projects
		WHERE integration_id = $1
		ORDER BY name ASC
	`

	rows, err := s.db.QueryContext(ctx, query, integrationID)
	if err != nil {
		return nil, fmt.Errorf("query projects by integration: %w", err)
	}
	defer rows.Close()

	var projects []*models.GitLabProject
	for rows.Next() {
		var project models.GitLabProject
		var webhookID sql.NullInt64
		var settingsJSON string
		var analysisModelID, embeddingModelID sql.NullString
		var reviewPrompt sql.NullString

		if err := rows.Scan(
			&project.ID,
			&project.IntegrationID,
			&project.GitLabProjectID,
			&project.Name,
			&project.PathWithNamespace,
			&project.AutoReview,
			&webhookID,
			&project.Status,
			&analysisModelID,
			&embeddingModelID,
			&reviewPrompt,
			&settingsJSON,
			&project.CreatedAt,
			&project.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}

		if webhookID.Valid {
			wid := webhookID.Int64
			project.WebhookID = &wid
		}
		if analysisModelID.Valid {
			project.AnalysisModelID = analysisModelID.String
		}
		if embeddingModelID.Valid {
			project.EmbeddingModelID = embeddingModelID.String
		}
		if reviewPrompt.Valid {
			project.ReviewPrompt = reviewPrompt.String
		}
		if settingsJSON != "" {
			if err := json.Unmarshal([]byte(settingsJSON), &project.Settings); err != nil {
				// Ignore JSON errors
			}
		}

		projects = append(projects, &project)
	}

	return projects, nil
}

func (s *PostgresStore) UpdateProject(ctx context.Context, id string, req *models.UpdateGitLabProjectRequest) error {
	var sets []string
	var args []interface{}
	argNum := 1

	if req.AutoReview != nil {
		sets = append(sets, fmt.Sprintf("auto_review = $%d", argNum))
		args = append(args, *req.AutoReview)
		argNum++
	}
	if req.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *req.Status)
		argNum++
	}
	if req.AnalysisModelID != nil {
		sets = append(sets, fmt.Sprintf("analysis_model_id = $%d", argNum))
		args = append(args, *req.AnalysisModelID)
		argNum++
	}
	if req.EmbeddingModelID != nil {
		sets = append(sets, fmt.Sprintf("embedding_model_id = $%d", argNum))
		args = append(args, *req.EmbeddingModelID)
		argNum++
	}
	if req.ReviewPrompt != nil {
		sets = append(sets, fmt.Sprintf("review_prompt = $%d", argNum))
		args = append(args, *req.ReviewPrompt)
		argNum++
	}
	if req.Settings != nil {
		settingsJSON, err := json.Marshal(req.Settings)
		if err != nil {
			return fmt.Errorf("marshal settings: %w", err)
		}
		sets = append(sets, fmt.Sprintf("settings = $%d", argNum))
		args = append(args, string(settingsJSON))
		argNum++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, fmt.Sprintf("updated_at = $%d", argNum))
	args = append(args, time.Now())
	argNum++
	args = append(args, id)

	query := fmt.Sprintf("UPDATE gitlab_projects SET %s WHERE id = $%d", strings.Join(sets, ", "), argNum)
	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}

	return nil
}

func (s *PostgresStore) DeleteProject(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM gitlab_projects WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}

func (s *PostgresStore) UpdateProjectWebhookID(ctx context.Context, id string, webhookID int64) error {
	query := "UPDATE gitlab_projects SET webhook_id = $1, updated_at = $2 WHERE id = $3"
	_, err := s.db.ExecContext(ctx, query, webhookID, time.Now(), id)
	if err != nil {
		return fmt.Errorf("update webhook_id: %w", err)
	}
	return nil
}

func (s *PostgresStore) DeleteProjectsByIntegration(ctx context.Context, integrationID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM gitlab_projects WHERE integration_id = $1", integrationID)
	if err != nil {
		return fmt.Errorf("delete projects: %w", err)
	}
	return nil
}

// UpdateProjectIndexStatus updates the index status and last indexed timestamp
func (s *PostgresStore) UpdateProjectIndexStatus(ctx context.Context, projectID, status string, chunksCount int64) error {
	now := time.Now()
	query := `UPDATE gitlab_projects 
		SET index_status = $1, last_indexed_at = $2, updated_at = $3 
		WHERE id = $4`
	_, err := s.db.ExecContext(ctx, query, status, now, now, projectID)
	if err != nil {
		return fmt.Errorf("update index status: %w", err)
	}
	return nil
}

// ============================================================================
// Model Usage Store Implementation
// ============================================================================

func (s *PostgresStore) FindProjectsByModel(ctx context.Context, modelID string) ([]models.GitLabProjectRef, error) {
	query := `
		SELECT p.id, p.name, p.integration_id, i.name,
		       CASE WHEN p.analysis_model_id = $1 THEN 'analysis' ELSE '' END as analysis_usage,
		       CASE WHEN p.embedding_model_id = $2 THEN 'embedding' ELSE '' END as embedding_usage
		FROM gitlab_projects p
		JOIN gitlab_integrations i ON i.id = p.integration_id
		WHERE p.analysis_model_id = $3 OR p.embedding_model_id = $4
	`

	rows, err := s.db.QueryContext(ctx, query, modelID, modelID, modelID, modelID)
	if err != nil {
		return nil, fmt.Errorf("query projects by model: %w", err)
	}
	defer rows.Close()

	var refs []models.GitLabProjectRef
	for rows.Next() {
		var ref models.GitLabProjectRef
		var analysisUsage, embeddingUsage string

		if err := rows.Scan(
			&ref.ProjectID,
			&ref.ProjectName,
			&ref.IntegrationID,
			&ref.IntegrationName,
			&analysisUsage,
			&embeddingUsage,
		); err != nil {
			return nil, fmt.Errorf("scan ref: %w", err)
		}

		// Determine usage type
		if analysisUsage != "" && embeddingUsage != "" {
			ref.UsageType = "analysis+embedding"
		} else if analysisUsage != "" {
			ref.UsageType = "analysis"
		} else {
			ref.UsageType = "embedding"
		}

		refs = append(refs, ref)
	}

	return refs, nil
}

func (s *PostgresStore) IsModelUsed(ctx context.Context, modelID string) (bool, error) {
	query := `SELECT COUNT(*) FROM gitlab_projects WHERE analysis_model_id = $1 OR embedding_model_id = $2`
	var count int
	if err := s.db.QueryRowContext(ctx, query, modelID, modelID).Scan(&count); err != nil {
		return false, fmt.Errorf("count model usage: %w", err)
	}
	return count > 0, nil
}


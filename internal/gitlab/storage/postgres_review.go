// Package storage provides PostgreSQL implementation for reviews, jobs, and webhook events
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

// ============================================================================
// Review Store Implementation
// ============================================================================

func (s *PostgresStore) CreateReview(ctx context.Context, review *models.GitLabMRReview) error {
	if review.ID == "" {
		review.ID = uuid.New().String()
	}
	
	now := time.Now()
	review.CreatedAt = now
	review.UpdatedAt = now

	var resultJSON []byte
	if review.ReviewResult != nil {
		var err error
		resultJSON, err = json.Marshal(review.ReviewResult)
		if err != nil {
			return fmt.Errorf("marshal review result: %w", err)
		}
	}

	query := `
		INSERT INTO gitlab_mr_reviews (
			id, project_id, mr_iid, mr_title, mr_author, source_branch, target_branch,
			status, files_analyzed, lines_changed, issues_found, review_result,
			note_id, discussion_id, processing_time_ms, tokens_used, model,
			retry_count, max_retries, error, created_at, updated_at, completed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17,
			$18, $19, $20, $21, $22, $23
		)
	`

	_, err := s.db.ExecContext(ctx, query,
		review.ID,
		review.ProjectID,
		review.MRIID,
		review.MRTitle,
		review.MRAuthor,
		review.SourceBranch,
		review.TargetBranch,
		review.Status,
		review.FilesAnalyzed,
		review.LinesChanged,
		review.IssuesFound,
		resultJSON,
		review.NoteID,
		review.DiscussionID,
		review.ProcessingTimeMs,
		review.TokensUsed,
		review.ModelUsed,
		review.RetryCount,
		review.MaxRetries,
		review.Error,
		review.CreatedAt,
		review.UpdatedAt,
		review.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert review: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetReview(ctx context.Context, id string) (*models.GitLabMRReview, error) {
	query := `
		SELECT id, project_id, mr_iid, mr_title, mr_author, source_branch, target_branch,
		       status, files_analyzed, lines_changed, issues_found, review_result,
		       note_id, discussion_id, processing_time_ms, tokens_used, model,
		       retry_count, max_retries, error, created_at, updated_at, completed_at
		FROM gitlab_mr_reviews
		WHERE id = $1
	`

	return s.scanReview(s.db.QueryRowContext(ctx, query, id))
}

func (s *PostgresStore) GetReviewByMR(ctx context.Context, projectID string, mrIID int) (*models.GitLabMRReview, error) {
	query := `
		SELECT id, project_id, mr_iid, mr_title, mr_author, source_branch, target_branch,
		       status, files_analyzed, lines_changed, issues_found, review_result,
		       note_id, discussion_id, processing_time_ms, tokens_used, model,
		       retry_count, max_retries, error, created_at, updated_at, completed_at
		FROM gitlab_mr_reviews
		WHERE project_id = $1 AND mr_iid = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	return s.scanReview(s.db.QueryRowContext(ctx, query, projectID, mrIID))
}

func (s *PostgresStore) scanReview(row *sql.Row) (*models.GitLabMRReview, error) {
	var review models.GitLabMRReview
	var resultJSON sql.NullString
	var noteID sql.NullInt64
	var discussionID sql.NullString
	var processingTime sql.NullInt64
	var tokensUsed sql.NullInt32
	var model sql.NullString
	var errStr sql.NullString
	var completedAt sql.NullTime

	err := row.Scan(
		&review.ID,
		&review.ProjectID,
		&review.MRIID,
		&review.MRTitle,
		&review.MRAuthor,
		&review.SourceBranch,
		&review.TargetBranch,
		&review.Status,
		&review.FilesAnalyzed,
		&review.LinesChanged,
		&review.IssuesFound,
		&resultJSON,
		&noteID,
		&discussionID,
		&processingTime,
		&tokensUsed,
		&model,
		&review.RetryCount,
		&review.MaxRetries,
		&errStr,
		&review.CreatedAt,
		&review.UpdatedAt,
		&completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan review: %w", err)
	}

	if resultJSON.Valid && resultJSON.String != "" {
		if err := json.Unmarshal([]byte(resultJSON.String), &review.ReviewResult); err != nil {
			// Ignore JSON errors
		}
	}
	if noteID.Valid {
		review.NoteID = &noteID.Int64
	}
	if discussionID.Valid {
		review.DiscussionID = &discussionID.String
	}
	if processingTime.Valid {
		review.ProcessingTimeMs = processingTime.Int64
	}
	if tokensUsed.Valid {
		review.TokensUsed = int(tokensUsed.Int32)
	}
	if model.Valid {
		review.ModelUsed = model.String
	}
	if errStr.Valid {
		review.Error = errStr.String
	}
	if completedAt.Valid {
		review.CompletedAt = &completedAt.Time
	}

	return &review, nil
}

func (s *PostgresStore) ListReviews(ctx context.Context, req *models.GitLabReviewListRequest) ([]models.GitLabMRReview, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if req.ProjectID != nil {
		conditions = append(conditions, fmt.Sprintf("r.project_id = $%d", argNum))
		args = append(args, *req.ProjectID)
		argNum++
	}
	if req.Status != nil {
		conditions = append(conditions, fmt.Sprintf("r.status = $%d", argNum))
		args = append(args, *req.Status)
		argNum++
	}
	if req.MRIID != nil {
		conditions = append(conditions, fmt.Sprintf("r.mr_iid = $%d", argNum))
		args = append(args, *req.MRIID)
		argNum++
	}
	if req.Search != nil && *req.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(r.mr_title ILIKE $%d OR r.mr_author ILIKE $%d)", argNum, argNum+1))
		search := "%" + *req.Search + "%"
		args = append(args, search, search)
		argNum += 2
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM gitlab_mr_reviews r %s", whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count reviews: %w", err)
	}

	// Fetch items
	query := fmt.Sprintf(`
		SELECT r.id, r.project_id, r.mr_iid, r.mr_title, r.mr_author, r.source_branch, r.target_branch,
		       r.status, r.files_analyzed, r.lines_changed, r.issues_found, r.review_result,
		       r.note_id, r.discussion_id, r.processing_time_ms, r.tokens_used, r.model,
		       r.retry_count, r.max_retries, r.error, r.created_at, r.updated_at, r.completed_at,
		       p.name as project_name, p.path_with_namespace
		FROM gitlab_mr_reviews r
		LEFT JOIN gitlab_projects p ON p.id = r.project_id
		%s
		ORDER BY r.created_at DESC
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
		return nil, 0, fmt.Errorf("query reviews: %w", err)
	}
	defer rows.Close()

	var reviews []models.GitLabMRReview
	for rows.Next() {
		var review models.GitLabMRReview
		var resultJSON sql.NullString
		var noteID sql.NullInt64
		var discussionID sql.NullString
		var processingTime sql.NullInt64
		var tokensUsed sql.NullInt32
		var model sql.NullString
		var errStr sql.NullString
		var completedAt sql.NullTime
		var projectName sql.NullString
		var pathWithNS sql.NullString

		if err := rows.Scan(
			&review.ID,
			&review.ProjectID,
			&review.MRIID,
			&review.MRTitle,
			&review.MRAuthor,
			&review.SourceBranch,
			&review.TargetBranch,
			&review.Status,
			&review.FilesAnalyzed,
			&review.LinesChanged,
			&review.IssuesFound,
			&resultJSON,
			&noteID,
			&discussionID,
			&processingTime,
			&tokensUsed,
			&model,
			&review.RetryCount,
			&review.MaxRetries,
			&errStr,
			&review.CreatedAt,
			&review.UpdatedAt,
			&completedAt,
			&projectName,
			&pathWithNS,
		); err != nil {
			return nil, 0, fmt.Errorf("scan review: %w", err)
		}

		if resultJSON.Valid && resultJSON.String != "" {
			if err := json.Unmarshal([]byte(resultJSON.String), &review.ReviewResult); err != nil {
				// Ignore JSON errors
			}
		}
		if noteID.Valid {
			review.NoteID = &noteID.Int64
		}
		if discussionID.Valid {
			review.DiscussionID = discussionID.String
		}
		if processingTime.Valid {
			review.ProcessingTimeMs = processingTime.Int64
		}
		if tokensUsed.Valid {
			review.TokensUsed = int(tokensUsed.Int32)
		}
		if model.Valid {
			review.ModelUsed = model.String
		}
		if errStr.Valid {
			review.Error = errStr.String
		}
		if completedAt.Valid {
			review.CompletedAt = &completedAt.Time
		}
		if projectName.Valid {
			review.ProjectName = projectName.String
		}
		if pathWithNS.Valid {
			review.ProjectPath = pathWithNS.String
		}

		reviews = append(reviews, review)
	}

	return reviews, total, nil
}

func (s *PostgresStore) UpdateReviewStatus(ctx context.Context, id string, status models.GitLabReviewStatus, errStr string) error {
	query := "UPDATE gitlab_mr_reviews SET status = $1, error = $2, updated_at = $3 WHERE id = $4"
	_, err := s.db.ExecContext(ctx, query, status, errStr, time.Now(), id)
	if err != nil {
		return fmt.Errorf("update review status: %w", err)
	}
	return nil
}

func (s *PostgresStore) UpdateReviewResult(ctx context.Context, id string, result *models.GitLabReviewResult, noteID int64, discussionID string) error {
	resultJSON, _ := json.Marshal(result)
	query := "UPDATE gitlab_mr_reviews SET review_result = $1, note_id = $2, discussion_id = $3, status = $4, completed_at = $5, updated_at = $6 WHERE id = $7"
	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, resultJSON, noteID, discussionID, models.GitLabReviewStatusCompleted, now, now, id)
	if err != nil {
		return fmt.Errorf("update review result: %w", err)
	}
	return nil
}

func (s *PostgresStore) UpdateReviewMetrics(ctx context.Context, id string, filesAnalyzed, linesChanged, issuesFound int, processingTimeMs int64, tokensUsed int, model string) error {
	query := "UPDATE gitlab_mr_reviews SET files_analyzed = $1, lines_changed = $2, issues_found = $3, processing_time_ms = $4, tokens_used = $5, model = $6, updated_at = $7 WHERE id = $8"
	_, err := s.db.ExecContext(ctx, query, filesAnalyzed, linesChanged, issuesFound, processingTimeMs, tokensUsed, model, time.Now(), id)
	return err
}

func (s *PostgresStore) IncrementReviewRetry(ctx context.Context, id string) error {
	query := "UPDATE gitlab_mr_reviews SET retry_count = retry_count + 1, updated_at = $1 WHERE id = $2"
	_, err := s.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("increment retry: %w", err)
	}
	return nil
}

// ============================================================================
// Job Store Implementation
// ============================================================================

func (s *PostgresStore) CreateJob(ctx context.Context, job *models.GitLabAnalysisJob) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	
	now := time.Now()
	job.CreatedAt = now
	job.UpdatedAt = now

	var configJSON []byte
	if job.Config != nil {
		var err error
		configJSON, err = json.Marshal(job.Config)
		if err != nil {
			return fmt.Errorf("marshal config: %w", err)
		}
	}

	query := `
		INSERT INTO gitlab_analysis_jobs (
			id, review_id, integration_id, project_id, mr_iid,
			status, priority, worker_id, config,
			retry_count, max_retries, next_retry_at, error,
			created_at, updated_at, started_at, completed_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16, $17
		)
	`

	_, err := s.db.ExecContext(ctx, query,
		job.ID,
		job.ReviewID,
		job.IntegrationID,
		job.ProjectID,
		job.MRIID,
		job.Status,
		job.Priority,
		job.WorkerID,
		configJSON,
		job.RetryCount,
		job.MaxRetries,
		job.NextRetryAt,
		job.Error,
		job.CreatedAt,
		job.UpdatedAt,
		job.StartedAt,
		job.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert job: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetJob(ctx context.Context, id string) (*models.GitLabAnalysisJob, error) {
	query := `
		SELECT j.id, j.review_id, j.integration_id, j.project_id, j.mr_iid,
		       j.status, j.priority, j.worker_id, j.config,
		       j.retry_count, j.max_retries, j.next_retry_at, j.error,
		       j.created_at, j.updated_at, j.started_at, j.completed_at,
		       r.mr_title, r.mr_author, r.source_branch, r.target_branch,
		       p.name as project_name
		FROM gitlab_analysis_jobs j
		LEFT JOIN gitlab_mr_reviews r ON r.id = j.review_id
		LEFT JOIN gitlab_projects p ON p.id = j.project_id
		WHERE j.id = $1
	`

	var job models.GitLabAnalysisJob
	var workerID sql.NullString
	var configJSON sql.NullString
	var nextRetryAt sql.NullTime
	var errStr sql.NullString
	var startedAt sql.NullTime
	var completedAt sql.NullTime
	var mrTitle sql.NullString
	var mrAuthor sql.NullString
	var sourceBranch sql.NullString
	var targetBranch sql.NullString
	var projectName sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID,
		&job.ReviewID,
		&job.IntegrationID,
		&job.ProjectID,
		&job.MRIID,
		&job.Status,
		&job.Priority,
		&workerID,
		&configJSON,
		&job.RetryCount,
		&job.MaxRetries,
		&nextRetryAt,
		&errStr,
		&job.CreatedAt,
		&job.UpdatedAt,
		&startedAt,
		&completedAt,
		&mrTitle,
		&mrAuthor,
		&sourceBranch,
		&targetBranch,
		&projectName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query job: %w", err)
	}

	if workerID.Valid {
		job.WorkerID = workerID.String
	}
	if configJSON.Valid && configJSON.String != "" {
		if err := json.Unmarshal([]byte(configJSON.String), &job.Config); err != nil {
			// Ignore JSON errors
		}
	}
	if nextRetryAt.Valid {
		job.NextRetryAt = &nextRetryAt.Time
	}
	if errStr.Valid {
		job.Error = errStr.String
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if mrTitle.Valid {
		job.MRTitle = mrTitle.String
	}
	if mrAuthor.Valid {
		job.MRAuthor = mrAuthor.String
	}
	if sourceBranch.Valid {
		job.SourceBranch = sourceBranch.String
	}
	if targetBranch.Valid {
		job.TargetBranch = targetBranch.String
	}
	if projectName.Valid {
		job.ProjectName = projectName.String
	}

	return &job, nil
}

func (s *PostgresStore) GetJobByReview(ctx context.Context, reviewID string) (*models.GitLabAnalysisJob, error) {
	query := `
		SELECT j.id, j.review_id, j.integration_id, j.project_id, j.mr_iid,
		       j.status, j.priority, j.worker_id, j.config,
		       j.retry_count, j.max_retries, j.next_retry_at, j.error,
		       j.created_at, j.updated_at, j.started_at, j.completed_at
		FROM gitlab_analysis_jobs j
		WHERE j.review_id = $1
		ORDER BY j.created_at DESC
		LIMIT 1
	`

	var job models.GitLabAnalysisJob
	var workerID sql.NullString
	var configJSON sql.NullString
	var nextRetryAt sql.NullTime
	var errStr sql.NullString
	var startedAt sql.NullTime
	var completedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, reviewID).Scan(
		&job.ID,
		&job.ReviewID,
		&job.IntegrationID,
		&job.ProjectID,
		&job.MRIID,
		&job.Status,
		&job.Priority,
		&workerID,
		&configJSON,
		&job.RetryCount,
		&job.MaxRetries,
		&nextRetryAt,
		&errStr,
		&job.CreatedAt,
		&job.UpdatedAt,
		&startedAt,
		&completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query job: %w", err)
	}

	if workerID.Valid {
		job.WorkerID = workerID.String
	}
	if configJSON.Valid && configJSON.String != "" {
		if err := json.Unmarshal([]byte(configJSON.String), &job.Config); err != nil {
			// Ignore JSON errors
		}
	}
	if nextRetryAt.Valid {
		job.NextRetryAt = &nextRetryAt.Time
	}
	if errStr.Valid {
		job.Error = errStr.String
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	return &job, nil
}

func (s *PostgresStore) GetNextPendingJob(ctx context.Context) (*models.GitLabAnalysisJob, error) {
	// Use FOR UPDATE SKIP LOCKED for concurrent access
	query := `
		SELECT j.id, j.review_id, j.integration_id, j.project_id, j.mr_iid,
		       j.status, j.priority, j.worker_id, j.config,
		       j.retry_count, j.max_retries, j.next_retry_at, j.error,
		       j.created_at, j.updated_at, j.started_at, j.completed_at,
		       r.mr_title, r.mr_author, r.source_branch, r.target_branch,
		       p.name as project_name, p.analysis_model_id, p.embedding_model_id, p.review_prompt
		FROM gitlab_analysis_jobs j
		LEFT JOIN gitlab_mr_reviews r ON r.id = j.review_id
		LEFT JOIN gitlab_projects p ON p.id = j.project_id
		WHERE j.status = $1 AND (j.next_retry_at IS NULL OR j.next_retry_at <= $2)
		ORDER BY j.priority DESC, j.created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	var job models.GitLabAnalysisJob
	var workerID sql.NullString
	var configJSON sql.NullString
	var nextRetryAt sql.NullTime
	var errStr sql.NullString
	var startedAt sql.NullTime
	var completedAt sql.NullTime
	var mrTitle sql.NullString
	var mrAuthor sql.NullString
	var sourceBranch sql.NullString
	var targetBranch sql.NullString
	var projectName sql.NullString
	var analysisModelID sql.NullString
	var embeddingModelID sql.NullString
	var reviewPrompt sql.NullString

	err := s.db.QueryRowContext(ctx, query, models.GitLabJobStatusPending, time.Now()).Scan(
		&job.ID,
		&job.ReviewID,
		&job.IntegrationID,
		&job.ProjectID,
		&job.MRIID,
		&job.Status,
		&job.Priority,
		&workerID,
		&configJSON,
		&job.RetryCount,
		&job.MaxRetries,
		&nextRetryAt,
		&errStr,
		&job.CreatedAt,
		&job.UpdatedAt,
		&startedAt,
		&completedAt,
		&mrTitle,
		&mrAuthor,
		&sourceBranch,
		&targetBranch,
		&projectName,
		&analysisModelID,
		&embeddingModelID,
		&reviewPrompt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query next job: %w", err)
	}

	if workerID.Valid {
		job.WorkerID = workerID.String
	}
	if configJSON.Valid && configJSON.String != "" {
		if err := json.Unmarshal([]byte(configJSON.String), &job.Config); err != nil {
			// Ignore JSON errors
		}
	}
	if nextRetryAt.Valid {
		job.NextRetryAt = &nextRetryAt.Time
	}
	if errStr.Valid {
		job.Error = errStr.String
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if mrTitle.Valid {
		job.MRTitle = mrTitle.String
	}
	if mrAuthor.Valid {
		job.MRAuthor = mrAuthor.String
	}
	if sourceBranch.Valid {
		job.SourceBranch = sourceBranch.String
	}
	if targetBranch.Valid {
		job.TargetBranch = targetBranch.String
	}
	if projectName.Valid {
		job.ProjectName = projectName.String
	}
	if analysisModelID.Valid {
		job.AnalysisModelID = analysisModelID.String
	}
	if embeddingModelID.Valid {
		job.EmbeddingModelID = embeddingModelID.String
	}
	if reviewPrompt.Valid {
		job.ReviewPrompt = reviewPrompt.String
	}

	return &job, nil
}

func (s *PostgresStore) ListJobs(ctx context.Context, status *models.GitLabJobStatus, limit, offset int) ([]models.GitLabAnalysisJob, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if status != nil {
		conditions = append(conditions, fmt.Sprintf("j.status = $%d", argNum))
		args = append(args, *status)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM gitlab_analysis_jobs j %s", whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count jobs: %w", err)
	}

	// Fetch items
	query := fmt.Sprintf(`
		SELECT j.id, j.review_id, j.integration_id, j.project_id, j.mr_iid,
		       j.status, j.priority, j.worker_id, j.config,
		       j.retry_count, j.max_retries, j.next_retry_at, j.error,
		       j.created_at, j.updated_at, j.started_at, j.completed_at,
		       r.mr_title, r.mr_author, r.source_branch, r.target_branch,
		       p.name as project_name
		FROM gitlab_analysis_jobs j
		LEFT JOIN gitlab_mr_reviews r ON r.id = j.review_id
		LEFT JOIN gitlab_projects p ON p.id = j.project_id
		%s
		ORDER BY j.priority DESC, j.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []models.GitLabAnalysisJob
	for rows.Next() {
		var job models.GitLabAnalysisJob
		var workerID sql.NullString
		var configJSON sql.NullString
		var nextRetryAt sql.NullTime
		var errStr sql.NullString
		var startedAt sql.NullTime
		var completedAt sql.NullTime
		var mrTitle sql.NullString
		var mrAuthor sql.NullString
		var sourceBranch sql.NullString
		var targetBranch sql.NullString
		var projectName sql.NullString

		if err := rows.Scan(
			&job.ID,
			&job.ReviewID,
			&job.IntegrationID,
			&job.ProjectID,
			&job.MRIID,
			&job.Status,
			&job.Priority,
			&workerID,
			&configJSON,
			&job.RetryCount,
			&job.MaxRetries,
			&nextRetryAt,
			&errStr,
			&job.CreatedAt,
			&job.UpdatedAt,
			&startedAt,
			&completedAt,
			&mrTitle,
			&mrAuthor,
			&sourceBranch,
			&targetBranch,
			&projectName,
		); err != nil {
			return nil, 0, fmt.Errorf("scan job: %w", err)
		}

		if workerID.Valid {
			job.WorkerID = workerID.String
		}
		if configJSON.Valid && configJSON.String != "" {
			if err := json.Unmarshal([]byte(configJSON.String), &job.Config); err != nil {
				// Ignore JSON errors
			}
		}
		if nextRetryAt.Valid {
			job.NextRetryAt = &nextRetryAt.Time
		}
		if errStr.Valid {
			job.Error = errStr.String
		}
		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}
		if mrTitle.Valid {
			job.MRTitle = mrTitle.String
		}
		if mrAuthor.Valid {
			job.MRAuthor = mrAuthor.String
		}
		if sourceBranch.Valid {
			job.SourceBranch = sourceBranch.String
		}
		if targetBranch.Valid {
			job.TargetBranch = targetBranch.String
		}
		if projectName.Valid {
			job.ProjectName = projectName.String
		}

		jobs = append(jobs, job)
	}

	return jobs, total, nil
}

func (s *PostgresStore) ClaimJob(ctx context.Context, jobID, workerID string) error {
	query := "UPDATE gitlab_analysis_jobs SET status = $1, worker_id = $2, started_at = $3, updated_at = $4 WHERE id = $5"
	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, models.GitLabJobStatusProcessing, workerID, now, now, jobID)
	if err != nil {
		return fmt.Errorf("claim job: %w", err)
	}
	return nil
}

func (s *PostgresStore) CompleteJob(ctx context.Context, jobID string) error {
	query := "UPDATE gitlab_analysis_jobs SET status = $1, completed_at = $2, updated_at = $3 WHERE id = $4"
	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, models.GitLabJobStatusCompleted, now, now, jobID)
	return err
}

func (s *PostgresStore) FailJob(ctx context.Context, jobID string, errStr string) error {
	query := "UPDATE gitlab_analysis_jobs SET status = $1, error = $2, completed_at = $3, updated_at = $4 WHERE id = $5"
	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, models.GitLabJobStatusFailed, errStr, now, now, jobID)
	return err
}

func (s *PostgresStore) RetryJob(ctx context.Context, jobID string, nextRetryAt interface{}) error {
	query := "UPDATE gitlab_analysis_jobs SET status = $1, retry_count = retry_count + 1, next_retry_at = $2, updated_at = $3 WHERE id = $4"
	_, err := s.db.ExecContext(ctx, query, models.GitLabJobStatusPending, nextRetryAt, time.Now(), jobID)
	return err
}

func (s *PostgresStore) CancelJob(ctx context.Context, jobID string) error {
	query := "UPDATE gitlab_analysis_jobs SET status = $1, completed_at = $2, updated_at = $3 WHERE id = $4"
	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, models.GitLabJobStatusCancelled, now, now, jobID)
	return err
}

func (s *PostgresStore) GetQueueStats(ctx context.Context) (*models.GitLabQueueStats, error) {
	stats := &models.GitLabQueueStats{}

	// Get counts by status
	query := `
		SELECT status, COUNT(*) 
		FROM gitlab_analysis_jobs 
		GROUP BY status
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status models.GitLabJobStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}

		switch status {
		case models.GitLabJobStatusPending:
			stats.Pending = count
		case models.GitLabJobStatusProcessing:
			stats.Processing = count
		case models.GitLabJobStatusCompleted:
			stats.Completed = count
		case models.GitLabJobStatusFailed:
			stats.Failed = count
		case models.GitLabJobStatusCancelled:
			stats.Cancelled = count
		}
	}

	// Get today's completed count
	todayQuery := `
		SELECT COUNT(*) FROM gitlab_analysis_jobs 
		WHERE status = $1 AND completed_at >= $2
	`
	today := time.Now().Truncate(24 * time.Hour)
	if err := s.db.QueryRowContext(ctx, todayQuery, models.GitLabJobStatusCompleted, today).Scan(&stats.CompletedToday); err != nil {
		return nil, fmt.Errorf("query today stats: %w", err)
	}

	// Get average processing time (last 100 completed jobs)
	avgQuery := `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at)) * 1000), 0)
		FROM (
			SELECT completed_at, started_at 
			FROM gitlab_analysis_jobs 
			WHERE status = $1 AND started_at IS NOT NULL AND completed_at IS NOT NULL
			ORDER BY completed_at DESC
			LIMIT 100
		) sub
	`
	if err := s.db.QueryRowContext(ctx, avgQuery, models.GitLabJobStatusCompleted).Scan(&stats.AvgProcessingTimeMs); err != nil {
		// Ignore error, keep 0
	}

	return stats, nil
}

func (s *PostgresStore) CleanupOldJobs(ctx context.Context, olderThanDays int) (int64, error) {
	query := `
		DELETE FROM gitlab_analysis_jobs 
		WHERE status IN ($1, $2, $3) AND completed_at < $4
	`
	cutoff := time.Now().AddDate(0, 0, -olderThanDays)
	result, err := s.db.ExecContext(ctx, query, 
		models.GitLabJobStatusCompleted, 
		models.GitLabJobStatusFailed, 
		models.GitLabJobStatusCancelled,
		cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleanup jobs: %w", err)
	}
	return result.RowsAffected()
}

// ============================================================================
// Webhook Event Store Implementation
// ============================================================================

func (s *PostgresStore) CreateWebhookEvent(ctx context.Context, event *models.GitLabWebhookEvent) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	event.ReceivedAt = time.Now()

	payloadJSON, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	query := `
		INSERT INTO gitlab_webhook_events (
			id, integration_id, event_type, project_id, mr_iid, object_id,
			payload, status, received_at, processed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = s.db.ExecContext(ctx, query,
		event.ID,
		event.IntegrationID,
		event.EventType,
		event.ProjectID,
		event.MRIID,
		event.ObjectID,
		payloadJSON,
		event.Status,
		event.ReceivedAt,
		event.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetWebhookEvent(ctx context.Context, integrationID string, projectID int64, mrIID int, objectID int64) (*models.GitLabWebhookEvent, error) {
	query := `
		SELECT id, integration_id, event_type, project_id, mr_iid, object_id,
		       payload, status, received_at, processed_at
		FROM gitlab_webhook_events
		WHERE integration_id = $1 AND project_id = $2 AND mr_iid = $3 AND object_id = $4
		ORDER BY received_at DESC
		LIMIT 1
	`

	var event models.GitLabWebhookEvent
	var payloadJSON []byte
	var processedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, integrationID, projectID, mrIID, objectID).Scan(
		&event.ID,
		&event.IntegrationID,
		&event.EventType,
		&event.ProjectID,
		&event.MRIID,
		&event.ObjectID,
		&payloadJSON,
		&event.Status,
		&event.ReceivedAt,
		&processedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query event: %w", err)
	}

	if len(payloadJSON) > 0 {
		if err := json.Unmarshal(payloadJSON, &event.Payload); err != nil {
			// Ignore JSON errors
		}
	}
	if processedAt.Valid {
		event.ProcessedAt = &processedAt.Time
	}

	return &event, nil
}

func (s *PostgresStore) MarkWebhookEventProcessed(ctx context.Context, id string) error {
	query := "UPDATE gitlab_webhook_events SET status = $1, processed_at = $2 WHERE id = $3"
	_, err := s.db.ExecContext(ctx, query, models.GitLabWebhookStatusProcessed, time.Now(), id)
	if err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}
	return nil
}

func (s *PostgresStore) MarkWebhookEventDeduplicated(ctx context.Context, id string) error {
	query := "UPDATE gitlab_webhook_events SET status = $1, processed_at = $2 WHERE id = $3"
	_, err := s.db.ExecContext(ctx, query, models.GitLabWebhookStatusDeduplicated, time.Now(), id)
	if err != nil {
		return fmt.Errorf("mark deduplicated: %w", err)
	}
	return nil
}

func (s *PostgresStore) CleanupOldEvents(ctx context.Context, olderThanDays int) (int64, error) {
	query := `DELETE FROM gitlab_webhook_events WHERE received_at < $1`
	cutoff := time.Now().AddDate(0, 0, -olderThanDays)
	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleanup events: %w", err)
	}
	return result.RowsAffected()
}


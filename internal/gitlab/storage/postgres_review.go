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

	review.CreatedAt = time.Now()

	var resultJSON interface{} = nil
	if review.ReviewResult != nil {
		data, err := json.Marshal(review.ReviewResult)
		if err != nil {
			return fmt.Errorf("marshal review result: %w", err)
		}
		resultJSON = data
	}

	query := `
		INSERT INTO gitlab_mr_reviews (
			id, project_id, integration_id, mr_iid, mr_title, mr_author, mr_author_id,
			source_branch, target_branch, mr_url, status, priority,
			files_analyzed, lines_changed, issues_found, review_result,
			note_id, discussion_id, processing_time_ms, tokens_used, model_used,
			retry_count, error, created_at, completed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16,
			$17, $18, $19, $20, $21,
			$22, $23, $24, $25
		)
	`

	_, err := s.db.ExecContext(ctx, query,
		review.ID,
		review.ProjectID,
		review.IntegrationID,
		review.MRIID,
		review.MRTitle,
		review.MRAuthor,
		review.MRAuthorID,
		review.SourceBranch,
		review.TargetBranch,
		review.MRURL,
		review.Status,
		review.Priority,
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
		review.Error,
		review.CreatedAt,
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
		       note_id, discussion_id, processing_time_ms, tokens_used, model_used,
		       retry_count, error, created_at, completed_at
		FROM gitlab_mr_reviews
		WHERE id = $1
	`

	return s.scanReview(s.db.QueryRowContext(ctx, query, id))
}

func (s *PostgresStore) GetReviewByMR(ctx context.Context, projectID string, mrIID int) (*models.GitLabMRReview, error) {
	query := `
		SELECT id, project_id, mr_iid, mr_title, mr_author, source_branch, target_branch,
		       status, files_analyzed, lines_changed, issues_found, review_result,
		       note_id, discussion_id, processing_time_ms, tokens_used, model_used,
		       retry_count, error, created_at, completed_at
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
		&errStr,
		&review.CreatedAt,
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

func (s *PostgresStore) ListReviewsByIntegration(ctx context.Context, integrationID string) ([]*models.GitLabMRReview, error) {
	query := `
		SELECT r.id, r.project_id, r.mr_iid, r.mr_title, r.mr_author, r.source_branch, r.target_branch,
		       r.status, r.files_analyzed, r.lines_changed, r.issues_found, r.review_result,
		       r.note_id, r.discussion_id, r.processing_time_ms, r.tokens_used, r.model_used,
		       r.retry_count, r.error, r.created_at, r.completed_at
		FROM gitlab_mr_reviews r
		INNER JOIN gitlab_projects p ON r.project_id = p.id
		WHERE p.integration_id = $1
		ORDER BY r.created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, integrationID)
	if err != nil {
		return nil, fmt.Errorf("query reviews by integration: %w", err)
	}
	defer rows.Close()

	var reviews []*models.GitLabMRReview
	for rows.Next() {
		var review models.GitLabMRReview
		var resultJSON []byte
		var noteID sql.NullInt64
		var discussionID sql.NullString
		var completedAt sql.NullTime
		var errorStr sql.NullString

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
			&review.ProcessingTimeMs,
			&review.TokensUsed,
			&review.ModelUsed,
			&review.RetryCount,
			&errorStr,
			&review.CreatedAt,
			&completedAt,
		); err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}

		if noteID.Valid {
			nid := noteID.Int64
			review.NoteID = &nid
		}
		if discussionID.Valid {
			did := discussionID.String
			review.DiscussionID = &did
		}
		if errorStr.Valid {
			review.Error = errorStr.String
		}
		if completedAt.Valid {
			review.CompletedAt = &completedAt.Time
		}
		if len(resultJSON) > 0 {
			var result models.GitLabReviewResult
			if err := json.Unmarshal(resultJSON, &result); err == nil {
				review.ReviewResult = &result
			}
		}

		reviews = append(reviews, &review)
	}

	return reviews, nil
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
		       r.note_id, r.discussion_id, r.processing_time_ms, r.tokens_used, r.model_used,
		       r.retry_count, r.error, r.created_at, r.completed_at,
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
			&errStr,
			&review.CreatedAt,
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
		if projectName.Valid {
			review.ProjectName = projectName.String
		}
		// pathWithNS is not stored in review

		reviews = append(reviews, review)
	}

	return reviews, total, nil
}

func (s *PostgresStore) UpdateReviewStatus(ctx context.Context, id string, status models.GitLabReviewStatus, errStr string) error {
	query := "UPDATE gitlab_mr_reviews SET status = $1, last_error = $2, updated_at = $3 WHERE id = $4"
	_, err := s.db.ExecContext(ctx, query, status, errStr, time.Now(), id)
	if err != nil {
		return fmt.Errorf("update review status: %w", err)
	}
	return nil
}

func (s *PostgresStore) UpdateReviewResult(ctx context.Context, id string, result *models.GitLabReviewResult, noteID int64, discussionID string) error {
	var resultJSON interface{} = nil
	if result != nil {
		data, _ := json.Marshal(result)
		resultJSON = data
	}
	query := "UPDATE gitlab_mr_reviews SET review_result = $1, note_id = $2, discussion_id = $3, status = $4, completed_at = $5, updated_at = $6 WHERE id = $7"
	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, resultJSON, noteID, discussionID, models.GitLabReviewStatusCompleted, now, now, id)
	if err != nil {
		return fmt.Errorf("update review result: %w", err)
	}
	return nil
}

func (s *PostgresStore) UpdateReviewMetrics(ctx context.Context, id string, filesAnalyzed, linesChanged, issuesFound int, processingTimeMs int64, tokensUsed int, model string) error {
	query := "UPDATE gitlab_mr_reviews SET files_analyzed = $1, lines_changed = $2, issues_found = $3, processing_time_ms = $4, tokens_used = $5, model_used = $6, updated_at = $7 WHERE id = $8"
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

	// Validate required UUID fields
	if job.ReviewID == "" {
		return fmt.Errorf("review_id is required")
	}
	if job.IntegrationID == "" {
		return fmt.Errorf("integration_id is required")
	}
	if job.ProjectID == "" {
		return fmt.Errorf("project_id is required")
	}

	now := time.Now()
	job.CreatedAt = now
	job.UpdatedAt = now

	var configJSON interface{} = nil
	if job.Config != nil {
		data, err := json.Marshal(job.Config)
		if err != nil {
			return fmt.Errorf("marshal config: %w", err)
		}
		configJSON = data
	}

	query := `
		INSERT INTO gitlab_analysis_jobs (
			id, review_id, integration_id, project_id, mr_iid, mr_title,
			status, priority, worker_id, config,
			retry_count, max_retries, next_retry_at, last_error,
			created_at, updated_at, started_at, completed_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14,
			$15, $16, $17, $18
		)
	`

	_, err := s.db.ExecContext(ctx, query,
		job.ID,
		job.ReviewID,
		job.IntegrationID,
		job.ProjectID,
		job.MRIID,
		job.MRTitle,
		job.Status,
		job.Priority,
		job.WorkerID,
		configJSON,
		job.RetryCount,
		job.MaxRetries,
		job.NextRetryAt,
		job.LastError,
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
		       j.status, j.priority, j.worker_id, j.mr_title, j.config,
		       j.retry_count, j.max_retries, j.next_retry_at, j.last_error,
		       j.created_at, j.updated_at, j.started_at, j.completed_at
		FROM gitlab_analysis_jobs j
		WHERE j.id = $1
	`

	var job models.GitLabAnalysisJob
	var workerID sql.NullString
	var mrTitle sql.NullString
	var configJSON []byte
	var nextRetryAt sql.NullTime
	var errStr sql.NullString
	var updatedAt sql.NullTime
	var startedAt sql.NullTime
	var completedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID,
		&job.ReviewID,
		&job.IntegrationID,
		&job.ProjectID,
		&job.MRIID,
		&job.Status,
		&job.Priority,
		&workerID,
		&mrTitle,
		&configJSON,
		&job.RetryCount,
		&job.MaxRetries,
		&nextRetryAt,
		&errStr,
		&job.CreatedAt,
		&updatedAt,
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
		job.WorkerID = &workerID.String
	}
	if mrTitle.Valid {
		job.MRTitle = mrTitle.String
	}
	if len(configJSON) > 0 {
		job.Config = &models.GitLabJobConfig{}
		_ = json.Unmarshal(configJSON, job.Config)
	}
	if nextRetryAt.Valid {
		job.NextRetryAt = &nextRetryAt.Time
	}
	if errStr.Valid {
		job.LastError = errStr.String
	}
	if updatedAt.Valid {
		job.UpdatedAt = updatedAt.Time
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	return &job, nil
}

func (s *PostgresStore) GetJobByReview(ctx context.Context, reviewID string) (*models.GitLabAnalysisJob, error) {
	query := `
		SELECT j.id, j.review_id, j.integration_id, j.project_id, j.mr_iid,
		       j.status, j.priority, j.worker_id, j.mr_title, j.config,
		       j.retry_count, j.max_retries, j.next_retry_at, j.last_error,
		       j.created_at, j.updated_at, j.started_at, j.completed_at
		FROM gitlab_analysis_jobs j
		WHERE j.review_id = $1
		ORDER BY j.created_at DESC
		LIMIT 1
	`

	var job models.GitLabAnalysisJob
	var workerID sql.NullString
	var mrTitle sql.NullString
	var configJSON []byte
	var nextRetryAt sql.NullTime
	var errStr sql.NullString
	var updatedAt sql.NullTime
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
		&mrTitle,
		&configJSON,
		&job.RetryCount,
		&job.MaxRetries,
		&nextRetryAt,
		&errStr,
		&job.CreatedAt,
		&updatedAt,
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
		job.WorkerID = &workerID.String
	}
	if mrTitle.Valid {
		job.MRTitle = mrTitle.String
	}
	if len(configJSON) > 0 {
		job.Config = &models.GitLabJobConfig{}
		_ = json.Unmarshal(configJSON, job.Config)
	}
	if nextRetryAt.Valid {
		job.NextRetryAt = &nextRetryAt.Time
	}
	if errStr.Valid {
		job.LastError = errStr.String
	}
	if updatedAt.Valid {
		job.UpdatedAt = updatedAt.Time
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
	// Use a transaction with SELECT FOR UPDATE + immediate status update
	// This ensures atomic claim - no two workers can get the same job
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Step 1: Find and lock a pending job
	selectQuery := `
		SELECT j.id, j.review_id, j.integration_id, j.project_id, j.mr_iid,
		       j.status, j.priority, j.worker_id, j.mr_title, j.config,
		       j.retry_count, j.max_retries, j.next_retry_at, j.last_error,
		       j.created_at, j.updated_at, j.started_at, j.completed_at,
		       r.mr_author, r.source_branch, r.target_branch,
		       p.name as project_name, p.analysis_model_id, p.embedding_model_id, p.review_prompt
		FROM gitlab_analysis_jobs j
		LEFT JOIN gitlab_mr_reviews r ON r.id = j.review_id
		LEFT JOIN gitlab_projects p ON p.id = j.project_id
		WHERE j.status = $1 AND (j.next_retry_at IS NULL OR j.next_retry_at <= $2)
		ORDER BY j.priority DESC, j.created_at ASC
		LIMIT 1
		FOR UPDATE OF j SKIP LOCKED
	`

	var job models.GitLabAnalysisJob
	var workerID sql.NullString
	var mrTitle sql.NullString
	var configJSON []byte
	var nextRetryAt sql.NullTime
	var errStr sql.NullString
	var updatedAt sql.NullTime
	var startedAt sql.NullTime
	var completedAt sql.NullTime
	var mrAuthor sql.NullString
	var sourceBranch sql.NullString
	var targetBranch sql.NullString
	var projectName sql.NullString
	var analysisModelID sql.NullString
	var embeddingModelID sql.NullString
	var reviewPrompt sql.NullString

	err = tx.QueryRowContext(ctx, selectQuery, models.GitLabJobStatusPending, time.Now()).Scan(
		&job.ID,
		&job.ReviewID,
		&job.IntegrationID,
		&job.ProjectID,
		&job.MRIID,
		&job.Status,
		&job.Priority,
		&workerID,
		&mrTitle,
		&configJSON,
		&job.RetryCount,
		&job.MaxRetries,
		&nextRetryAt,
		&errStr,
		&job.CreatedAt,
		&updatedAt,
		&startedAt,
		&completedAt,
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

	// Step 2: Immediately mark as processing (claim the job atomically)
	updateQuery := `UPDATE gitlab_analysis_jobs SET status = $1, started_at = $2, updated_at = $2 WHERE id = $3`
	now := time.Now()
	_, err = tx.ExecContext(ctx, updateQuery, models.GitLabJobStatusProcessing, now, job.ID)
	if err != nil {
		return nil, fmt.Errorf("claim job: %w", err)
	}

	// Step 3: Commit transaction - only now the job is claimed
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit claim: %w", err)
	}

	// Update job fields from scan results
	job.Status = models.GitLabJobStatusProcessing
	job.StartedAt = &now

	if workerID.Valid {
		job.WorkerID = &workerID.String
	}
	if mrTitle.Valid {
		job.MRTitle = mrTitle.String
	}
	if len(configJSON) > 0 {
		job.Config = &models.GitLabJobConfig{}
		_ = json.Unmarshal(configJSON, job.Config)
	}
	if nextRetryAt.Valid {
		job.NextRetryAt = &nextRetryAt.Time
	}
	if errStr.Valid {
		job.LastError = errStr.String
	}
	if updatedAt.Valid {
		job.UpdatedAt = updatedAt.Time
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
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
		       j.status, j.priority, j.worker_id, j.mr_title, j.config,
		       j.retry_count, j.max_retries, j.next_retry_at, j.last_error,
		       j.created_at, j.updated_at, j.started_at, j.completed_at,
		       r.mr_author, r.source_branch, r.target_branch,
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
		var mrTitle sql.NullString
		var configJSON []byte
		var nextRetryAt sql.NullTime
		var errStr sql.NullString
		var updatedAt sql.NullTime
		var startedAt sql.NullTime
		var completedAt sql.NullTime
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
			&mrTitle,
			&configJSON,
			&job.RetryCount,
			&job.MaxRetries,
			&nextRetryAt,
			&errStr,
			&job.CreatedAt,
			&updatedAt,
			&startedAt,
			&completedAt,
			&mrAuthor,
			&sourceBranch,
			&targetBranch,
			&projectName,
		); err != nil {
			return nil, 0, fmt.Errorf("scan job: %w", err)
		}

		if workerID.Valid {
			job.WorkerID = &workerID.String
		}
		if mrTitle.Valid {
			job.MRTitle = mrTitle.String
		}
		if len(configJSON) > 0 {
			job.Config = &models.GitLabJobConfig{}
			_ = json.Unmarshal(configJSON, job.Config)
		}
		if nextRetryAt.Valid {
			job.NextRetryAt = &nextRetryAt.Time
		}
		if errStr.Valid {
			job.LastError = errStr.String
		}
		if updatedAt.Valid {
			job.UpdatedAt = updatedAt.Time
		}
		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
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
	query := "UPDATE gitlab_analysis_jobs SET status = $1, last_error = $2, completed_at = $3, updated_at = $4 WHERE id = $5"
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

	// Payload is already a JSON string, pass it directly or nil if empty
	var payloadJSON interface{} = nil
	if event.Payload != "" {
		// Validate it's valid JSON before storing
		if json.Valid([]byte(event.Payload)) {
			payloadJSON = []byte(event.Payload)
		} else {
			// Wrap non-JSON string as JSON string
			data, _ := json.Marshal(event.Payload)
			payloadJSON = data
		}
	}

	query := `
		INSERT INTO gitlab_webhook_events (
			id, integration_id, event_type, project_id, mr_iid, object_id,
			payload, status, received_at, processed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.db.ExecContext(ctx, query,
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
		// Store raw JSON as string
		event.Payload = string(payloadJSON)
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

// ============================================================================
// Feedback Store Implementation
// ============================================================================

func (s *PostgresStore) CreateFeedback(ctx context.Context, feedback *models.GitLabReviewFeedback) error {
	if feedback.ID == "" {
		feedback.ID = uuid.New().String()
	}
	feedback.CreatedAt = time.Now()

	query := `
		INSERT INTO gitlab_review_feedback (
			id, review_id, user_id, rating, feedback_type, comment, issue_index, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := s.db.ExecContext(ctx, query,
		feedback.ID,
		feedback.ReviewID,
		feedback.UserID,
		feedback.Rating,
		feedback.FeedbackType,
		feedback.Comment,
		feedback.IssueIndex,
		feedback.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert feedback: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListFeedback(ctx context.Context, req *models.GitLabFeedbackListRequest) ([]models.GitLabReviewFeedback, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	if req.ReviewID != nil {
		conditions = append(conditions, fmt.Sprintf("f.review_id = $%d", argNum))
		args = append(args, *req.ReviewID)
		argNum++
	}
	if req.FeedbackType != nil {
		conditions = append(conditions, fmt.Sprintf("f.feedback_type = $%d", argNum))
		args = append(args, *req.FeedbackType)
		argNum++
	}
	if req.MinRating != nil {
		conditions = append(conditions, fmt.Sprintf("f.rating >= $%d", argNum))
		args = append(args, *req.MinRating)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM gitlab_review_feedback f %s", whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count feedback: %w", err)
	}

	// Fetch items
	query := fmt.Sprintf(`
		SELECT f.id, f.review_id, f.user_id, f.rating, f.feedback_type, f.comment, f.issue_index, f.created_at
		FROM gitlab_review_feedback f
		%s
		ORDER BY f.created_at DESC
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
		return nil, 0, fmt.Errorf("query feedback: %w", err)
	}
	defer rows.Close()

	var feedback []models.GitLabReviewFeedback
	for rows.Next() {
		var f models.GitLabReviewFeedback
		var userID sql.NullString
		var comment sql.NullString
		var issueIndex sql.NullInt32

		if err := rows.Scan(
			&f.ID,
			&f.ReviewID,
			&userID,
			&f.Rating,
			&f.FeedbackType,
			&comment,
			&issueIndex,
			&f.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan feedback: %w", err)
		}

		if userID.Valid {
			f.UserID = &userID.String
		}
		if comment.Valid {
			f.Comment = comment.String
		}
		if issueIndex.Valid {
			idx := int(issueIndex.Int32)
			f.IssueIndex = &idx
		}

		feedback = append(feedback, f)
	}

	return feedback, total, nil
}

// ============================================================================
// Analytics Store Implementation
// ============================================================================

func (s *PostgresStore) GetAnalytics(ctx context.Context, days int) (*models.GitLabAnalytics, error) {
	cutoff := time.Now().AddDate(0, 0, -days)

	analytics := &models.GitLabAnalytics{
		Days:            days,
		ReviewsByStatus: make(map[string]int),
	}

	// Get totals
	totalsQuery := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COUNT(*) FILTER (WHERE status = 'pending' OR status = 'queued' OR status = 'analyzing') as pending,
			COALESCE(AVG(processing_time_ms) FILTER (WHERE status = 'completed'), 0) as avg_processing,
			COALESCE(SUM(issues_found), 0) as total_issues,
			COALESCE(SUM(tokens_used), 0) as total_tokens
		FROM gitlab_mr_reviews
		WHERE created_at >= $1
	`

	err := s.db.QueryRowContext(ctx, totalsQuery, cutoff).Scan(
		&analytics.TotalReviews,
		&analytics.CompletedReviews,
		&analytics.FailedReviews,
		&analytics.PendingReviews,
		&analytics.AvgProcessingMs,
		&analytics.TotalIssuesFound,
		&analytics.TotalTokensUsed,
	)
	if err != nil {
		return nil, fmt.Errorf("query totals: %w", err)
	}

	// Get reviews by day
	dayQuery := `
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COALESCE(SUM(issues_found), 0) as issues
		FROM gitlab_mr_reviews
		WHERE created_at >= $1
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`

	dayRows, err := s.db.QueryContext(ctx, dayQuery, cutoff)
	if err != nil {
		return nil, fmt.Errorf("query by day: %w", err)
	}
	defer dayRows.Close()

	for dayRows.Next() {
		var ds models.DayStats
		var date time.Time
		if err := dayRows.Scan(&date, &ds.TotalReviews, &ds.CompletedReviews, &ds.FailedReviews, &ds.IssuesFound); err != nil {
			return nil, fmt.Errorf("scan day stats: %w", err)
		}
		ds.Date = date.Format("2006-01-02")
		analytics.ReviewsByDay = append(analytics.ReviewsByDay, ds)
	}

	// Get reviews by project
	projectQuery := `
		SELECT 
			r.project_id,
			p.name,
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE r.status = 'completed') as completed,
			COALESCE(SUM(r.issues_found), 0) as issues
		FROM gitlab_mr_reviews r
		LEFT JOIN gitlab_projects p ON p.id = r.project_id
		WHERE r.created_at >= $1
		GROUP BY r.project_id, p.name
		ORDER BY total DESC
		LIMIT 10
	`

	projectRows, err := s.db.QueryContext(ctx, projectQuery, cutoff)
	if err != nil {
		return nil, fmt.Errorf("query by project: %w", err)
	}
	defer projectRows.Close()

	for projectRows.Next() {
		var ps models.ProjectStats
		var projectName sql.NullString
		if err := projectRows.Scan(&ps.ProjectID, &projectName, &ps.TotalReviews, &ps.CompletedReviews, &ps.IssuesFound); err != nil {
			return nil, fmt.Errorf("scan project stats: %w", err)
		}
		if projectName.Valid {
			ps.ProjectName = projectName.String
		} else {
			ps.ProjectName = "Unknown"
		}
		analytics.ReviewsByProject = append(analytics.ReviewsByProject, ps)
	}

	// Get reviews by status
	statusQuery := `
		SELECT status, COUNT(*) FROM gitlab_mr_reviews WHERE created_at >= $1 GROUP BY status
	`
	statusRows, err := s.db.QueryContext(ctx, statusQuery, cutoff)
	if err != nil {
		return nil, fmt.Errorf("query by status: %w", err)
	}
	defer statusRows.Close()

	for statusRows.Next() {
		var status string
		var count int
		if err := statusRows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan status: %w", err)
		}
		analytics.ReviewsByStatus[status] = count
	}

	// Get feedback stats
	feedbackQuery := `
		SELECT 
			COALESCE(AVG(rating), 0),
			COUNT(*)
		FROM gitlab_review_feedback
		WHERE created_at >= $1
	`
	err = s.db.QueryRowContext(ctx, feedbackQuery, cutoff).Scan(&analytics.AvgRating, &analytics.FeedbackCount)
	if err != nil && err != sql.ErrNoRows {
		// Ignore error if table doesn't exist yet
		analytics.AvgRating = 0
		analytics.FeedbackCount = 0
	}

	return analytics, nil
}

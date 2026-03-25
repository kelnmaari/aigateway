// Package postgresql implements RAG Jobs and Query Logs operations for PostgreSQL.
package postgresql

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
// RAG Jobs Queue
// ========================================

// CreateRAGJob создает новую задачу в очереди
func (db *PostgreSQLDB) CreateRAGJob(ctx context.Context, job *models.RAGJob) error {
	query := `
		INSERT INTO rag_jobs (
			job_type, status, payload, priority, attempts, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	err := db.db.QueryRowContext(ctx, query,
		job.JobType,
		job.Status,
		job.Payload,
		job.Priority,
		job.Attempts,
		job.CreatedAt,
	).Scan(&job.ID)

	return err
}

// GetRAGJob получает задачу по ID
func (db *PostgreSQLDB) GetRAGJob(ctx context.Context, id string) (*models.RAGJob, error) {
	query := `
		SELECT 
			id, job_type, status, payload, result, priority, attempts,
			created_at, started_at, completed_at, error, locked_until
		FROM rag_jobs
		WHERE id = $1
	`

	var job models.RAGJob
	var startedAt, completedAt, errorMsg, lockedUntil sql.NullString
	var resultData []byte

	err := db.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID,
		&job.JobType,
		&job.Status,
		&job.Payload,
		&resultData,
		&job.Priority,
		&job.Attempts,
		&job.CreatedAt,
		&startedAt,
		&completedAt,
		&errorMsg,
		&lockedUntil,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if len(resultData) > 0 {
		result := new(models.JobResult)
		json.Unmarshal(resultData, result)
		job.Result = result
	}
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339, startedAt.String)
		job.StartedAt = &t
	}
	if completedAt.Valid {
		t, _ := time.Parse(time.RFC3339, completedAt.String)
		job.CompletedAt = &t
	}
	if errorMsg.Valid {
		job.Error = errorMsg.String
	}
	if lockedUntil.Valid {
		t, _ := time.Parse(time.RFC3339, lockedUntil.String)
		job.LockedUntil = &t
	}

	return &job, nil
}

// UpdateRAGJob обновляет задачу
func (db *PostgreSQLDB) UpdateRAGJob(ctx context.Context, job *models.RAGJob) error {
	var resultJSON sql.NullString
	if job.Result != nil {
		data, err := json.Marshal(job.Result)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
		resultJSON = sql.NullString{String: string(data), Valid: true}
	} else {
		resultJSON = sql.NullString{Valid: false}
	}

	query := `
		UPDATE rag_jobs SET
			status = $1, result = $2, attempts = $3, started_at = $4,
			completed_at = $5, error = $6, locked_until = $7
		WHERE id = $8
	`

	_, err := db.db.ExecContext(ctx, query,
		job.Status,
		resultJSON,
		job.Attempts,
		job.StartedAt,
		job.CompletedAt,
		job.Error,
		job.LockedUntil,
		job.ID,
	)

	return err
}

// GetNextPendingRAGJob получает следующую pending задачу с блокировкой
func (db *PostgreSQLDB) GetNextPendingRAGJob(ctx context.Context) (*models.RAGJob, error) {
	// PostgreSQL: используем FOR UPDATE SKIP LOCKED для эффективной блокировки
	query := `
		SELECT 
			id, job_type, status, payload, result, priority, attempts,
			created_at, started_at, completed_at, error, locked_until
		FROM rag_jobs
		WHERE status = 'pending' 
		AND (locked_until IS NULL OR locked_until < NOW())
		ORDER BY priority DESC, created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	var job models.RAGJob
	var startedAt, completedAt, errorMsg, lockedUntil sql.NullString
	var resultData []byte

	err := db.db.QueryRowContext(ctx, query).Scan(
		&job.ID,
		&job.JobType,
		&job.Status,
		&job.Payload,
		&resultData,
		&job.Priority,
		&job.Attempts,
		&job.CreatedAt,
		&startedAt,
		&completedAt,
		&errorMsg,
		&lockedUntil,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get next job: %w", err)
	}

	if len(resultData) > 0 {
		result := new(models.JobResult)
		json.Unmarshal(resultData, result)
		job.Result = result
	}
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339, startedAt.String)
		job.StartedAt = &t
	}
	if completedAt.Valid {
		t, _ := time.Parse(time.RFC3339, completedAt.String)
		job.CompletedAt = &t
	}
	if errorMsg.Valid {
		job.Error = errorMsg.String
	}
	if lockedUntil.Valid {
		t, _ := time.Parse(time.RFC3339, lockedUntil.String)
		job.LockedUntil = &t
	}

	// Блокируем задачу на 5 минут
	lockUntil := time.Now().Add(5 * time.Minute)
	_, err = db.db.ExecContext(ctx, "UPDATE rag_jobs SET locked_until = $1 WHERE id = $2", lockUntil, job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to lock job: %w", err)
	}
	job.LockedUntil = &lockUntil

	return &job, nil
}

// CountRAGJobsByStatus подсчитывает задачи по статусу
func (db *PostgreSQLDB) CountRAGJobsByStatus(ctx context.Context, status string) (int, error) {
	query := "SELECT COUNT(*) FROM rag_jobs WHERE status = $1"

	var count int
	err := db.db.QueryRowContext(ctx, query, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count jobs: %w", err)
	}

	return count, nil
}

// DeleteOldRAGJobs удаляет старые задачи
func (db *PostgreSQLDB) DeleteOldRAGJobs(ctx context.Context, cutoffTime time.Time, statuses []string) (int, error) {
	if len(statuses) == 0 {
		return 0, nil
	}

	placeholders := make([]string, len(statuses))
	args := []any{cutoffTime}
	for i, status := range statuses {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, status)
	}

	query := fmt.Sprintf(`
		DELETE FROM rag_jobs
		WHERE created_at < $1 AND status IN (%s)
	`, strings.Join(placeholders, ","))

	result, err := db.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old jobs: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return int(affected), nil
}

// UnlockExpiredRAGJobs разблокирует задачи с истекшим locked_until
func (db *PostgreSQLDB) UnlockExpiredRAGJobs(ctx context.Context, now time.Time) (int, error) {
	query := `
		UPDATE rag_jobs
		SET locked_until = NULL, status = 'pending'
		WHERE status = 'processing' AND locked_until IS NOT NULL AND locked_until < $1
	`

	result, err := db.db.ExecContext(ctx, query, now)
	if err != nil {
		return 0, fmt.Errorf("failed to unlock expired jobs: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return int(affected), nil
}

// ========================================
// RAG Query Logs
// ========================================

// CreateRAGQueryLog создает лог запроса
func (db *PostgreSQLDB) CreateRAGQueryLog(ctx context.Context, log *models.RAGQueryLog) error {
	sourceIDsJSON, _ := json.Marshal(log.SourceIDs)

	query := `
		INSERT INTO rag_query_logs (
			user_id, conversation_id, query_text, source_ids,
			chunks_retrieved, chunks_used, search_time_ms,
			total_tokens_used, response_quality_score, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	err := db.db.QueryRowContext(ctx, query,
		log.UserID,
		log.ConversationID,
		log.QueryText,
		sourceIDsJSON,
		log.ChunksRetrieved,
		log.ChunksUsed,
		log.SearchTimeMs,
		log.TotalTokensUsed,
		log.ResponseQualityScore,
		log.CreatedAt,
	).Scan(&log.ID)

	return err
}

// GetRAGQueryLog получает лог по ID
func (db *PostgreSQLDB) GetRAGQueryLog(ctx context.Context, id int64) (*models.RAGQueryLog, error) {
	query := `
		SELECT 
			id, user_id, conversation_id, query_text, source_ids,
			chunks_retrieved, chunks_used, search_time_ms,
			total_tokens_used, response_quality_score, created_at
		FROM rag_query_logs
		WHERE id = $1
	`

	var log models.RAGQueryLog
	var userID, conversationID sql.NullString
	var sourceIDsJSON []byte
	var chunksUsed, totalTokensUsed sql.NullInt64
	var responseQualityScore sql.NullFloat64

	err := db.db.QueryRowContext(ctx, query, id).Scan(
		&log.ID,
		&userID,
		&conversationID,
		&log.QueryText,
		&sourceIDsJSON,
		&log.ChunksRetrieved,
		&chunksUsed,
		&log.SearchTimeMs,
		&totalTokensUsed,
		&responseQualityScore,
		&log.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("query log not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get query log: %w", err)
	}

	if userID.Valid {
		uidStr := userID.String
		log.UserID = &uidStr
	}
	if conversationID.Valid {
		cidStr := conversationID.String
		log.ConversationID = &cidStr
	}
	if chunksUsed.Valid {
		log.ChunksUsed = int(chunksUsed.Int64)
	}
	if totalTokensUsed.Valid {
		log.TotalTokensUsed = int(totalTokensUsed.Int64)
	}
	if responseQualityScore.Valid {
		score := float64(responseQualityScore.Float64)
		log.ResponseQualityScore = &score
	}

	json.Unmarshal(sourceIDsJSON, &log.SourceIDs)

	return &log, nil
}

// ListRAGQueryLogsByUser возвращает логи пользователя
func (db *PostgreSQLDB) ListRAGQueryLogsByUser(ctx context.Context, userID string, limit, offset int) ([]*models.RAGQueryLog, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT 
			id, user_id, conversation_id, query_text, source_ids,
			chunks_retrieved, chunks_used, search_time_ms,
			total_tokens_used, response_quality_score, created_at
		FROM rag_query_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.RAGQueryLog
	for rows.Next() {
		var log models.RAGQueryLog
		var userIDField, conversationID sql.NullString
		var sourceIDsJSON []byte
		var chunksUsed, totalTokensUsed sql.NullInt64
		var responseQualityScore sql.NullFloat64

		err := rows.Scan(
			&log.ID,
			&userIDField,
			&conversationID,
			&log.QueryText,
			&sourceIDsJSON,
			&log.ChunksRetrieved,
			&chunksUsed,
			&log.SearchTimeMs,
			&totalTokensUsed,
			&responseQualityScore,
			&log.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan log: %w", err)
		}

		if userIDField.Valid {
			uidStr := userIDField.String
			log.UserID = &uidStr
		}
		if conversationID.Valid {
			cidStr := conversationID.String
			log.ConversationID = &cidStr
		}
		if chunksUsed.Valid {
			log.ChunksUsed = int(chunksUsed.Int64)
		}
		if totalTokensUsed.Valid {
			log.TotalTokensUsed = int(totalTokensUsed.Int64)
		}
		if responseQualityScore.Valid {
			score := float64(responseQualityScore.Float64)
			log.ResponseQualityScore = &score
		}

		json.Unmarshal(sourceIDsJSON, &log.SourceIDs)

		logs = append(logs, &log)
	}

	return logs, nil
}

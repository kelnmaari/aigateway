package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// JobType тип задачи RAG
type JobType string

const (
	JobTypeFileUpload JobType = "file_upload"
	JobTypeAPISync    JobType = "api_sync"
	JobTypeDBQuery    JobType = "db_query"
	JobTypeWebScrape  JobType = "web_scrape"
)

// JobStatus статус задачи
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// RAGJob представляет задачу в queue
type RAGJob struct {
	ID          int64        `json:"id" db:"id"`
	JobType     JobType      `json:"job_type" db:"job_type"`
	Status      JobStatus    `json:"status" db:"status"`
	
	// Данные
	Payload     JobPayload   `json:"payload" db:"payload"`
	Result      *JobResult   `json:"result,omitempty" db:"result"`
	
	// Приоритет и повторы
	Priority    int          `json:"priority" db:"priority"`
	Attempts    int          `json:"attempts" db:"attempts"`
	MaxAttempts int          `json:"max_attempts" db:"max_attempts"`
	
	// Timestamps
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	StartedAt   *time.Time   `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time   `json:"completed_at,omitempty" db:"completed_at"`
	
	// Ошибки
	Error       string       `json:"error,omitempty" db:"error"`
	
	// Для visibility timeout
	LockedUntil *time.Time   `json:"locked_until,omitempty" db:"locked_until"`
}

// JobPayload данные задачи (varies by job type)
type JobPayload map[string]interface{}

// Scan реализует sql.Scanner для JobPayload
func (p *JobPayload) Scan(value interface{}) error {
	if value == nil {
		*p = make(JobPayload)
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan JobPayload: expected []byte, got %T", value)
	}
	
	return json.Unmarshal(bytes, p)
}

// Value реализует driver.Valuer для JobPayload
func (p JobPayload) Value() (driver.Value, error) {
	if p == nil {
		return json.Marshal(make(JobPayload))
	}
	return json.Marshal(p)
}

// JobResult результат выполнения задачи
type JobResult map[string]interface{}

// Scan реализует sql.Scanner для JobResult
func (r *JobResult) Scan(value interface{}) error {
	if value == nil {
		*r = make(JobResult)
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan JobResult: expected []byte, got %T", value)
	}
	
	return json.Unmarshal(bytes, r)
}

// Value реализует driver.Valuer для JobResult
func (r JobResult) Value() (driver.Value, error) {
	if r == nil {
		return nil, nil
	}
	return json.Marshal(r)
}

// CreateJobRequest запрос на создание задачи
type CreateJobRequest struct {
	JobType     JobType                `json:"job_type" binding:"required"`
	Payload     map[string]interface{} `json:"payload" binding:"required"`
	Priority    int                    `json:"priority,omitempty"`
	MaxAttempts int                    `json:"max_attempts,omitempty"`
}



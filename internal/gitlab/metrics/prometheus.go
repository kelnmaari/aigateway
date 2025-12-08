// Package metrics provides Prometheus metrics for GitLab MR Review integration
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace = "aigateway"
	subsystem = "gitlab"
)

// ========================================
// Queue Metrics
// ========================================

var (
	// QueueJobsTotal counts total jobs by status (pending, processing, completed, failed, cancelled)
	QueueJobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "queue_jobs_total",
			Help:      "Total number of GitLab review jobs processed",
		},
		[]string{"status"},
	)

	// QueueJobsCurrent tracks current number of jobs by status
	QueueJobsCurrent = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "queue_jobs_current",
			Help:      "Current number of jobs by status",
		},
		[]string{"status"},
	)

	// QueueJobDuration measures job processing duration in seconds
	QueueJobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "queue_job_duration_seconds",
			Help:      "Job processing duration in seconds",
			Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600}, // 1s to 10min
		},
		[]string{"project_id"},
	)

	// QueueJobRetries counts job retry attempts
	QueueJobRetries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "queue_job_retries_total",
			Help:      "Total number of job retry attempts",
		},
		[]string{"project_id"},
	)

	// QueueWaitTime measures job wait time in queue before processing
	QueueWaitTime = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "queue_wait_time_seconds",
			Help:      "Time jobs spend waiting in queue before processing",
			Buckets:   []float64{0.1, 0.5, 1, 5, 10, 30, 60, 120, 300},
		},
	)
)

// ========================================
// Worker Metrics
// ========================================

var (
	// WorkersActive tracks number of active workers
	WorkersActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "workers_active",
			Help:      "Number of active workers processing jobs",
		},
	)

	// WorkersTotal tracks total number of configured workers
	WorkersTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "workers_total",
			Help:      "Total number of configured workers",
		},
	)

	// WorkerJobsProcessed counts jobs processed per worker
	WorkerJobsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "worker_jobs_processed_total",
			Help:      "Total jobs processed by each worker",
		},
		[]string{"worker_id"},
	)
)

// ========================================
// Webhook Metrics
// ========================================

var (
	// WebhooksReceived counts total webhooks received by event type
	WebhooksReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "webhooks_received_total",
			Help:      "Total number of webhooks received",
		},
		[]string{"event_type", "integration_id"},
	)

	// WebhooksDeduplicated counts deduplicated webhooks
	WebhooksDeduplicated = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "webhooks_deduplicated_total",
			Help:      "Total number of deduplicated webhooks",
		},
	)

	// WebhooksProcessed counts successfully processed webhooks
	WebhooksProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "webhooks_processed_total",
			Help:      "Total number of processed webhooks",
		},
		[]string{"status"}, // success, error
	)

	// WebhookProcessingDuration measures webhook processing duration
	WebhookProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "webhook_processing_duration_seconds",
			Help:      "Webhook processing duration in seconds",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
		},
	)
)

// ========================================
// Review Metrics
// ========================================

var (
	// ReviewsCreated counts reviews created by project
	ReviewsCreated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "reviews_created_total",
			Help:      "Total number of reviews created",
		},
		[]string{"project_id", "integration_id"},
	)

	// ReviewsCompleted counts completed reviews by status
	ReviewsCompleted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "reviews_completed_total",
			Help:      "Total number of completed reviews",
		},
		[]string{"status"}, // completed, failed, cancelled
	)

	// ReviewAnalysisDuration measures total review analysis duration
	ReviewAnalysisDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "review_analysis_duration_seconds",
			Help:      "Review analysis duration in seconds",
			Buckets:   []float64{5, 10, 30, 60, 120, 300, 600, 1200},
		},
		[]string{"model_id"},
	)

	// ReviewFilesAnalyzed tracks files analyzed per review
	ReviewFilesAnalyzed = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "review_files_analyzed",
			Help:      "Number of files analyzed per review",
			Buckets:   []float64{1, 5, 10, 20, 50, 100, 200},
		},
	)

	// ReviewLinesChanged tracks lines changed per review
	ReviewLinesChanged = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "review_lines_changed",
			Help:      "Number of lines changed per review",
			Buckets:   []float64{10, 50, 100, 500, 1000, 5000, 10000},
		},
	)

	// ReviewIssuesFound tracks issues found per review
	ReviewIssuesFound = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "review_issues_found",
			Help:      "Number of issues found per review",
			Buckets:   []float64{0, 1, 2, 5, 10, 20, 50},
		},
	)
)

// ========================================
// LLM Metrics
// ========================================

var (
	// LLMTokensUsed counts tokens used by model
	LLMTokensUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "llm_tokens_used_total",
			Help:      "Total LLM tokens used for code analysis",
		},
		[]string{"model_id", "type"}, // type: prompt, completion
	)

	// LLMRequestDuration measures LLM API request duration
	LLMRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "llm_request_duration_seconds",
			Help:      "LLM API request duration in seconds",
			Buckets:   []float64{1, 5, 10, 30, 60, 120, 300},
		},
		[]string{"model_id"},
	)

	// LLMErrors counts LLM API errors
	LLMErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "llm_errors_total",
			Help:      "Total LLM API errors",
		},
		[]string{"model_id", "error_type"},
	)
)

// ========================================
// Integration Metrics
// ========================================

var (
	// IntegrationsActive tracks active integrations
	IntegrationsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "integrations_active",
			Help:      "Number of active GitLab integrations",
		},
	)

	// ProjectsMonitored tracks monitored projects
	ProjectsMonitored = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "projects_monitored",
			Help:      "Number of monitored GitLab projects",
		},
	)

	// GitLabAPIRequests counts GitLab API requests
	GitLabAPIRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "api_requests_total",
			Help:      "Total GitLab API requests",
		},
		[]string{"integration_id", "endpoint", "status"},
	)

	// GitLabAPILatency measures GitLab API latency
	GitLabAPILatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "api_latency_seconds",
			Help:      "GitLab API request latency in seconds",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
		},
		[]string{"integration_id", "endpoint"},
	)

	// GitLabRateLimitRemaining tracks rate limit remaining
	GitLabRateLimitRemaining = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "rate_limit_remaining",
			Help:      "GitLab API rate limit remaining",
		},
		[]string{"integration_id"},
	)
)

// ========================================
// Comment Metrics
// ========================================

var (
	// CommentsPosted counts comments posted to GitLab
	CommentsPosted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "comments_posted_total",
			Help:      "Total comments posted to GitLab MRs",
		},
		[]string{"project_id", "type"}, // type: note, discussion
	)

	// CommentsSplit counts times comments were split due to size limit
	CommentsSplit = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "comments_split_total",
			Help:      "Total times comments were split due to size limit",
		},
	)

	// CommentSize tracks comment sizes
	CommentSize = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "comment_size_bytes",
			Help:      "Comment size in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 2, 10), // 100B to 100KB
		},
	)
)

// ========================================
// Utility Functions
// ========================================

// RecordJobCompleted records job completion metrics
func RecordJobCompleted(projectID string, status string, duration time.Duration) {
	QueueJobsTotal.WithLabelValues(status).Inc()
	QueueJobDuration.WithLabelValues(projectID).Observe(duration.Seconds())
}

// RecordJobQueued records job queued
func RecordJobQueued() {
	QueueJobsTotal.WithLabelValues("queued").Inc()
}

// RecordJobRetry records job retry
func RecordJobRetry(projectID string) {
	QueueJobRetries.WithLabelValues(projectID).Inc()
}

// RecordWebhookReceived records webhook received
func RecordWebhookReceived(eventType, integrationID string) {
	WebhooksReceived.WithLabelValues(eventType, integrationID).Inc()
}

// RecordWebhookDeduplicated records deduplicated webhook
func RecordWebhookDeduplicated() {
	WebhooksDeduplicated.Inc()
}

// RecordWebhookProcessed records webhook processing result
func RecordWebhookProcessed(success bool, duration time.Duration) {
	status := "success"
	if !success {
		status = "error"
	}
	WebhooksProcessed.WithLabelValues(status).Inc()
	WebhookProcessingDuration.Observe(duration.Seconds())
}

// RecordReviewCreated records review creation
func RecordReviewCreated(projectID, integrationID string) {
	ReviewsCreated.WithLabelValues(projectID, integrationID).Inc()
}

// RecordReviewCompleted records review completion
func RecordReviewCompleted(status string, modelID string, duration time.Duration, filesAnalyzed, linesChanged, issuesFound int) {
	ReviewsCompleted.WithLabelValues(status).Inc()
	ReviewAnalysisDuration.WithLabelValues(modelID).Observe(duration.Seconds())
	ReviewFilesAnalyzed.Observe(float64(filesAnalyzed))
	ReviewLinesChanged.Observe(float64(linesChanged))
	ReviewIssuesFound.Observe(float64(issuesFound))
}

// RecordLLMUsage records LLM usage
func RecordLLMUsage(modelID string, promptTokens, completionTokens int, duration time.Duration) {
	LLMTokensUsed.WithLabelValues(modelID, "prompt").Add(float64(promptTokens))
	LLMTokensUsed.WithLabelValues(modelID, "completion").Add(float64(completionTokens))
	LLMRequestDuration.WithLabelValues(modelID).Observe(duration.Seconds())
}

// RecordLLMError records LLM error
func RecordLLMError(modelID, errorType string) {
	LLMErrors.WithLabelValues(modelID, errorType).Inc()
}

// RecordGitLabAPIRequest records GitLab API request
func RecordGitLabAPIRequest(integrationID, endpoint string, success bool, duration time.Duration) {
	status := "success"
	if !success {
		status = "error"
	}
	GitLabAPIRequests.WithLabelValues(integrationID, endpoint, status).Inc()
	GitLabAPILatency.WithLabelValues(integrationID, endpoint).Observe(duration.Seconds())
}

// RecordCommentPosted records comment posted
func RecordCommentPosted(projectID, commentType string, size int, wasSplit bool) {
	CommentsPosted.WithLabelValues(projectID, commentType).Inc()
	CommentSize.Observe(float64(size))
	if wasSplit {
		CommentsSplit.Inc()
	}
}

// UpdateQueueStats updates queue gauge metrics
func UpdateQueueStats(pending, processing, completed, failed int) {
	QueueJobsCurrent.WithLabelValues("pending").Set(float64(pending))
	QueueJobsCurrent.WithLabelValues("processing").Set(float64(processing))
	QueueJobsCurrent.WithLabelValues("completed").Set(float64(completed))
	QueueJobsCurrent.WithLabelValues("failed").Set(float64(failed))
}

// UpdateWorkerStats updates worker metrics
func UpdateWorkerStats(active, total int) {
	WorkersActive.Set(float64(active))
	WorkersTotal.Set(float64(total))
}

// UpdateIntegrationStats updates integration metrics
func UpdateIntegrationStats(integrations, projects int) {
	IntegrationsActive.Set(float64(integrations))
	ProjectsMonitored.Set(float64(projects))
}


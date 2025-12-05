// Package analytics provides statistics and analytics for GitLab code reviews
package analytics

import (
	"context"
	"time"
)

// Service provides analytics for code reviews
type Service struct {
	store Store
}

// Store interface for analytics data access
type Store interface {
	// Review stats
	GetReviewStats(ctx context.Context, filter *StatsFilter) (*ReviewStats, error)
	GetReviewsByProject(ctx context.Context, filter *StatsFilter) ([]ProjectReviewStats, error)
	GetReviewsByModel(ctx context.Context, filter *StatsFilter) ([]ModelReviewStats, error)
	GetReviewsByUser(ctx context.Context, filter *StatsFilter) ([]UserReviewStats, error)
	
	// Trend data
	GetReviewTrend(ctx context.Context, filter *StatsFilter, interval string) ([]TrendPoint, error)
	GetIssueTrend(ctx context.Context, filter *StatsFilter, interval string) ([]TrendPoint, error)
	
	// Issue breakdown
	GetIssuesByCategory(ctx context.Context, filter *StatsFilter) ([]CategoryStats, error)
	GetIssuesBySeverity(ctx context.Context, filter *StatsFilter) ([]SeverityStats, error)
	
	// Performance
	GetProcessingTimeStats(ctx context.Context, filter *StatsFilter) (*ProcessingTimeStats, error)
	GetTokenUsageStats(ctx context.Context, filter *StatsFilter) (*TokenUsageStats, error)
}

// NewService creates a new analytics service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// StatsFilter filters analytics data
type StatsFilter struct {
	IntegrationID string    `json:"integration_id,omitempty"`
	ProjectID     string    `json:"project_id,omitempty"`
	ModelID       string    `json:"model_id,omitempty"`
	UserID        string    `json:"user_id,omitempty"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
}

// ReviewStats contains overall review statistics
type ReviewStats struct {
	TotalReviews      int     `json:"total_reviews"`
	CompletedReviews  int     `json:"completed_reviews"`
	FailedReviews     int     `json:"failed_reviews"`
	PendingReviews    int     `json:"pending_reviews"`
	TotalFilesReviewed int    `json:"total_files_reviewed"`
	TotalLinesChanged int     `json:"total_lines_changed"`
	TotalIssuesFound  int     `json:"total_issues_found"`
	TotalTokensUsed   int64   `json:"total_tokens_used"`
	AvgScore          float64 `json:"avg_score"`
	AvgProcessingTime float64 `json:"avg_processing_time_ms"`
	SuccessRate       float64 `json:"success_rate"`
}

// ProjectReviewStats contains per-project statistics
type ProjectReviewStats struct {
	ProjectID         string  `json:"project_id"`
	ProjectName       string  `json:"project_name"`
	TotalReviews      int     `json:"total_reviews"`
	CompletedReviews  int     `json:"completed_reviews"`
	TotalIssuesFound  int     `json:"total_issues_found"`
	AvgScore          float64 `json:"avg_score"`
	AvgProcessingTime float64 `json:"avg_processing_time_ms"`
}

// ModelReviewStats contains per-model statistics
type ModelReviewStats struct {
	ModelID           string  `json:"model_id"`
	ModelName         string  `json:"model_name"`
	TotalReviews      int     `json:"total_reviews"`
	TotalTokensUsed   int64   `json:"total_tokens_used"`
	AvgTokensPerReview float64 `json:"avg_tokens_per_review"`
	AvgProcessingTime float64 `json:"avg_processing_time_ms"`
	AvgScore          float64 `json:"avg_score"`
}

// UserReviewStats contains per-user statistics
type UserReviewStats struct {
	UserID            string  `json:"user_id"`
	Username          string  `json:"username"`
	TotalMRs          int     `json:"total_mrs"`
	TotalIssuesFound  int     `json:"total_issues_found"`
	AvgIssuesPerMR    float64 `json:"avg_issues_per_mr"`
	AvgScore          float64 `json:"avg_score"`
}

// TrendPoint represents a point in a time series
type TrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Count     int       `json:"count"`
}

// CategoryStats contains statistics by issue category
type CategoryStats struct {
	Category   string `json:"category"`
	Count      int    `json:"count"`
	Percentage float64 `json:"percentage"`
}

// SeverityStats contains statistics by severity
type SeverityStats struct {
	Severity   string `json:"severity"`
	Count      int    `json:"count"`
	Percentage float64 `json:"percentage"`
}

// ProcessingTimeStats contains processing time statistics
type ProcessingTimeStats struct {
	MinMs    int64   `json:"min_ms"`
	MaxMs    int64   `json:"max_ms"`
	AvgMs    float64 `json:"avg_ms"`
	MedianMs int64   `json:"median_ms"`
	P95Ms    int64   `json:"p95_ms"`
	P99Ms    int64   `json:"p99_ms"`
}

// TokenUsageStats contains token usage statistics
type TokenUsageStats struct {
	TotalTokens      int64   `json:"total_tokens"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	AvgPerReview     float64 `json:"avg_per_review"`
	MaxPerReview     int64   `json:"max_per_review"`
}

// Dashboard provides dashboard data
type Dashboard struct {
	Overview          *ReviewStats          `json:"overview"`
	TopProjects       []ProjectReviewStats  `json:"top_projects"`
	TopModels         []ModelReviewStats    `json:"top_models"`
	ReviewTrend       []TrendPoint          `json:"review_trend"`
	IssuesByCategory  []CategoryStats       `json:"issues_by_category"`
	IssuesBySeverity  []SeverityStats       `json:"issues_by_severity"`
	RecentActivity    []ActivityItem        `json:"recent_activity"`
}

// ActivityItem represents recent activity
type ActivityItem struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // review_started, review_completed, review_failed
	ProjectName string    `json:"project_name"`
	MRTitle     string    `json:"mr_title"`
	MRIID       int       `json:"mr_iid"`
	Timestamp   time.Time `json:"timestamp"`
	Details     string    `json:"details,omitempty"`
}

// GetDashboard returns dashboard data
func (s *Service) GetDashboard(ctx context.Context, filter *StatsFilter) (*Dashboard, error) {
	dashboard := &Dashboard{}
	
	// Get overview stats
	overview, err := s.store.GetReviewStats(ctx, filter)
	if err != nil {
		return nil, err
	}
	dashboard.Overview = overview
	
	// Get top projects (by review count)
	projects, err := s.store.GetReviewsByProject(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(projects) > 5 {
		projects = projects[:5]
	}
	dashboard.TopProjects = projects
	
	// Get top models
	models, err := s.store.GetReviewsByModel(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(models) > 5 {
		models = models[:5]
	}
	dashboard.TopModels = models
	
	// Get review trend (daily)
	trend, err := s.store.GetReviewTrend(ctx, filter, "day")
	if err != nil {
		return nil, err
	}
	dashboard.ReviewTrend = trend
	
	// Get issues by category
	categories, err := s.store.GetIssuesByCategory(ctx, filter)
	if err != nil {
		return nil, err
	}
	dashboard.IssuesByCategory = categories
	
	// Get issues by severity
	severities, err := s.store.GetIssuesBySeverity(ctx, filter)
	if err != nil {
		return nil, err
	}
	dashboard.IssuesBySeverity = severities
	
	return dashboard, nil
}

// GetProjectAnalytics returns detailed analytics for a project
func (s *Service) GetProjectAnalytics(ctx context.Context, projectID string, filter *StatsFilter) (*ProjectAnalytics, error) {
	filter.ProjectID = projectID
	
	analytics := &ProjectAnalytics{}
	
	// Get overall stats
	stats, err := s.store.GetReviewStats(ctx, filter)
	if err != nil {
		return nil, err
	}
	analytics.Stats = stats
	
	// Get processing time stats
	procTime, err := s.store.GetProcessingTimeStats(ctx, filter)
	if err != nil {
		return nil, err
	}
	analytics.ProcessingTime = procTime
	
	// Get token usage
	tokens, err := s.store.GetTokenUsageStats(ctx, filter)
	if err != nil {
		return nil, err
	}
	analytics.TokenUsage = tokens
	
	// Get user breakdown
	users, err := s.store.GetReviewsByUser(ctx, filter)
	if err != nil {
		return nil, err
	}
	analytics.UserBreakdown = users
	
	// Get issue trends
	issueTrend, err := s.store.GetIssueTrend(ctx, filter, "day")
	if err != nil {
		return nil, err
	}
	analytics.IssueTrend = issueTrend
	
	return analytics, nil
}

// ProjectAnalytics contains detailed project analytics
type ProjectAnalytics struct {
	Stats          *ReviewStats          `json:"stats"`
	ProcessingTime *ProcessingTimeStats  `json:"processing_time"`
	TokenUsage     *TokenUsageStats      `json:"token_usage"`
	UserBreakdown  []UserReviewStats     `json:"user_breakdown"`
	IssueTrend     []TrendPoint          `json:"issue_trend"`
}

// GetModelComparison compares model performance
func (s *Service) GetModelComparison(ctx context.Context, filter *StatsFilter) (*ModelComparison, error) {
	models, err := s.store.GetReviewsByModel(ctx, filter)
	if err != nil {
		return nil, err
	}
	
	comparison := &ModelComparison{
		Models: models,
	}
	
	// Calculate best performers
	var bestScore *ModelReviewStats
	var fastestModel *ModelReviewStats
	var mostEfficient *ModelReviewStats
	
	for i := range models {
		m := &models[i]
		if bestScore == nil || m.AvgScore > bestScore.AvgScore {
			bestScore = m
		}
		if fastestModel == nil || m.AvgProcessingTime < fastestModel.AvgProcessingTime {
			fastestModel = m
		}
		if mostEfficient == nil || m.AvgTokensPerReview < mostEfficient.AvgTokensPerReview {
			mostEfficient = m
		}
	}
	
	comparison.BestQuality = bestScore
	comparison.Fastest = fastestModel
	comparison.MostEfficient = mostEfficient
	
	return comparison, nil
}

// ModelComparison contains model comparison data
type ModelComparison struct {
	Models         []ModelReviewStats `json:"models"`
	BestQuality    *ModelReviewStats  `json:"best_quality,omitempty"`
	Fastest        *ModelReviewStats  `json:"fastest,omitempty"`
	MostEfficient  *ModelReviewStats  `json:"most_efficient,omitempty"`
}

// ExportReport exports analytics as a report
func (s *Service) ExportReport(ctx context.Context, filter *StatsFilter, format string) ([]byte, error) {
	// Get all data
	dashboard, err := s.GetDashboard(ctx, filter)
	if err != nil {
		return nil, err
	}
	
	switch format {
	case "json":
		return s.exportJSON(dashboard)
	case "csv":
		return s.exportCSV(dashboard)
	default:
		return s.exportJSON(dashboard)
	}
}

func (s *Service) exportJSON(data interface{}) ([]byte, error) {
	// Implementation would use encoding/json
	return nil, nil
}

func (s *Service) exportCSV(data *Dashboard) ([]byte, error) {
	// Implementation would generate CSV
	return nil, nil
}


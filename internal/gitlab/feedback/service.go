// Package feedback provides learning from user feedback on reviews
package feedback

import (
	"context"
	"time"
)

// Service handles feedback collection and learning
type Service struct {
	store Store
}

// Store interface for feedback data
type Store interface {
	// Feedback CRUD
	SaveFeedback(ctx context.Context, feedback *Feedback) error
	GetFeedback(ctx context.Context, id string) (*Feedback, error)
	GetFeedbackByReview(ctx context.Context, reviewID string) ([]*Feedback, error)
	GetFeedbackByIssue(ctx context.Context, issueID string) (*Feedback, error)
	ListFeedback(ctx context.Context, filter *FeedbackFilter) ([]*Feedback, error)
	
	// Aggregated stats
	GetFeedbackStats(ctx context.Context, filter *FeedbackFilter) (*FeedbackStats, error)
	GetAccuracyByCategory(ctx context.Context, filter *FeedbackFilter) ([]CategoryAccuracy, error)
	GetAccuracyByModel(ctx context.Context, filter *FeedbackFilter) ([]ModelAccuracy, error)
	
	// Training data export
	GetApprovedIssues(ctx context.Context, filter *FeedbackFilter) ([]*ApprovedIssue, error)
	GetRejectedIssues(ctx context.Context, filter *FeedbackFilter) ([]*RejectedIssue, error)
}

// NewService creates a new feedback service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// FeedbackType represents the type of feedback
type FeedbackType string

const (
	FeedbackTypeApprove  FeedbackType = "approve"
	FeedbackTypeReject   FeedbackType = "reject"
	FeedbackTypeEdit     FeedbackType = "edit"
	FeedbackTypeIgnore   FeedbackType = "ignore"
)

// Feedback represents user feedback on a review issue
type Feedback struct {
	ID           string       `json:"id" db:"id"`
	ReviewID     string       `json:"review_id" db:"review_id"`
	IssueID      string       `json:"issue_id" db:"issue_id"`
	ProjectID    string       `json:"project_id" db:"project_id"`
	ModelID      string       `json:"model_id" db:"model_id"`
	
	// Original issue data
	IssueCategory string     `json:"issue_category" db:"issue_category"`
	IssueSeverity string     `json:"issue_severity" db:"issue_severity"`
	IssueMessage  string     `json:"issue_message" db:"issue_message"`
	IssueLine     int        `json:"issue_line" db:"issue_line"`
	IssueFile     string     `json:"issue_file" db:"issue_file"`
	
	// Feedback
	Type          FeedbackType `json:"type" db:"type"`
	Comment       string       `json:"comment,omitempty" db:"comment"`
	EditedMessage string       `json:"edited_message,omitempty" db:"edited_message"`
	
	// User info
	UserID        string       `json:"user_id" db:"user_id"`
	Username      string       `json:"username" db:"username"`
	
	// Timestamps
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
}

// FeedbackFilter filters feedback queries
type FeedbackFilter struct {
	ProjectID     string       `json:"project_id,omitempty"`
	ModelID       string       `json:"model_id,omitempty"`
	Type          FeedbackType `json:"type,omitempty"`
	Category      string       `json:"category,omitempty"`
	StartDate     time.Time    `json:"start_date"`
	EndDate       time.Time    `json:"end_date"`
	Limit         int          `json:"limit"`
	Offset        int          `json:"offset"`
}

// FeedbackStats contains aggregated feedback statistics
type FeedbackStats struct {
	TotalFeedback    int     `json:"total_feedback"`
	ApprovedCount    int     `json:"approved_count"`
	RejectedCount    int     `json:"rejected_count"`
	EditedCount      int     `json:"edited_count"`
	IgnoredCount     int     `json:"ignored_count"`
	ApprovalRate     float64 `json:"approval_rate"`     // Approved / (Approved + Rejected)
	AccuracyRate     float64 `json:"accuracy_rate"`     // (Approved + Edited) / Total
	UsefulnessRate   float64 `json:"usefulness_rate"`   // (Approved + Edited + Rejected) / Total
}

// CategoryAccuracy shows accuracy per issue category
type CategoryAccuracy struct {
	Category      string  `json:"category"`
	TotalIssues   int     `json:"total_issues"`
	ApprovedCount int     `json:"approved_count"`
	RejectedCount int     `json:"rejected_count"`
	AccuracyRate  float64 `json:"accuracy_rate"`
}

// ModelAccuracy shows accuracy per model
type ModelAccuracy struct {
	ModelID       string  `json:"model_id"`
	ModelName     string  `json:"model_name"`
	TotalIssues   int     `json:"total_issues"`
	ApprovedCount int     `json:"approved_count"`
	RejectedCount int     `json:"rejected_count"`
	AccuracyRate  float64 `json:"accuracy_rate"`
}

// ApprovedIssue represents an approved issue for training
type ApprovedIssue struct {
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	CodeContext string `json:"code_context"`
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion"`
	Approved    bool   `json:"approved"`
}

// RejectedIssue represents a rejected issue for negative examples
type RejectedIssue struct {
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	CodeContext string `json:"code_context"`
	Message     string `json:"message"`
	Reason      string `json:"reason,omitempty"`
}

// SubmitFeedback submits feedback on a review issue
func (s *Service) SubmitFeedback(ctx context.Context, feedback *Feedback) error {
	feedback.CreatedAt = time.Now()
	return s.store.SaveFeedback(ctx, feedback)
}

// GetFeedbackForReview gets all feedback for a review
func (s *Service) GetFeedbackForReview(ctx context.Context, reviewID string) ([]*Feedback, error) {
	return s.store.GetFeedbackByReview(ctx, reviewID)
}

// GetFeedbackStats gets aggregated statistics
func (s *Service) GetFeedbackStats(ctx context.Context, filter *FeedbackFilter) (*FeedbackStats, error) {
	return s.store.GetFeedbackStats(ctx, filter)
}

// GetCategoryAccuracy gets accuracy breakdown by category
func (s *Service) GetCategoryAccuracy(ctx context.Context, filter *FeedbackFilter) ([]CategoryAccuracy, error) {
	return s.store.GetAccuracyByCategory(ctx, filter)
}

// GetModelAccuracy gets accuracy breakdown by model
func (s *Service) GetModelAccuracy(ctx context.Context, filter *FeedbackFilter) ([]ModelAccuracy, error) {
	return s.store.GetAccuracyByModel(ctx, filter)
}

// ExportTrainingData exports approved/rejected issues for model training
func (s *Service) ExportTrainingData(ctx context.Context, filter *FeedbackFilter) (*TrainingData, error) {
	approved, err := s.store.GetApprovedIssues(ctx, filter)
	if err != nil {
		return nil, err
	}
	
	rejected, err := s.store.GetRejectedIssues(ctx, filter)
	if err != nil {
		return nil, err
	}
	
	return &TrainingData{
		ApprovedExamples: approved,
		RejectedExamples: rejected,
		ExportedAt:       time.Now(),
	}, nil
}

// TrainingData contains exported training data
type TrainingData struct {
	ApprovedExamples []*ApprovedIssue `json:"approved_examples"`
	RejectedExamples []*RejectedIssue `json:"rejected_examples"`
	ExportedAt       time.Time        `json:"exported_at"`
}

// GenerateFewShotExamples generates few-shot examples from feedback
func (s *Service) GenerateFewShotExamples(ctx context.Context, category string, count int) ([]FewShotExample, error) {
	filter := &FeedbackFilter{
		Type:     FeedbackTypeApprove,
		Category: category,
		Limit:    count,
	}
	
	approved, err := s.store.GetApprovedIssues(ctx, filter)
	if err != nil {
		return nil, err
	}
	
	examples := make([]FewShotExample, 0, len(approved))
	for _, issue := range approved {
		examples = append(examples, FewShotExample{
			Input:    issue.CodeContext,
			Output:   issue.Message,
			Category: issue.Category,
		})
	}
	
	return examples, nil
}

// FewShotExample represents a few-shot learning example
type FewShotExample struct {
	Input    string `json:"input"`
	Output   string `json:"output"`
	Category string `json:"category"`
}

// CalculateModelQuality calculates model quality based on feedback
func (s *Service) CalculateModelQuality(ctx context.Context, modelID string) (*ModelQuality, error) {
	filter := &FeedbackFilter{ModelID: modelID}
	
	stats, err := s.store.GetFeedbackStats(ctx, filter)
	if err != nil {
		return nil, err
	}
	
	categories, err := s.store.GetAccuracyByCategory(ctx, filter)
	if err != nil {
		return nil, err
	}
	
	quality := &ModelQuality{
		ModelID:          modelID,
		OverallAccuracy:  stats.AccuracyRate,
		ApprovalRate:     stats.ApprovalRate,
		TotalFeedback:    stats.TotalFeedback,
		CategoryAccuracy: categories,
	}
	
	// Find best and worst categories
	for _, cat := range categories {
		if quality.BestCategory == "" || cat.AccuracyRate > quality.BestAccuracy {
			quality.BestCategory = cat.Category
			quality.BestAccuracy = cat.AccuracyRate
		}
		if quality.WorstCategory == "" || cat.AccuracyRate < quality.WorstAccuracy {
			quality.WorstCategory = cat.Category
			quality.WorstAccuracy = cat.AccuracyRate
		}
	}
	
	return quality, nil
}

// ModelQuality represents model quality metrics
type ModelQuality struct {
	ModelID          string            `json:"model_id"`
	OverallAccuracy  float64           `json:"overall_accuracy"`
	ApprovalRate     float64           `json:"approval_rate"`
	TotalFeedback    int               `json:"total_feedback"`
	CategoryAccuracy []CategoryAccuracy `json:"category_accuracy"`
	BestCategory     string            `json:"best_category"`
	BestAccuracy     float64           `json:"best_accuracy"`
	WorstCategory    string            `json:"worst_category"`
	WorstAccuracy    float64           `json:"worst_accuracy"`
}


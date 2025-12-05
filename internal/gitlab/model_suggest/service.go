// Package model_suggest provides automatic model suggestion based on codebase language
package model_suggest

import (
	"context"
	"sort"
	"strings"
)

// Service suggests optimal models for code review
type Service struct {
	modelStore    ModelStore
	feedbackStore FeedbackStore
}

// ModelStore interface for model data
type ModelStore interface {
	GetActiveModels(ctx context.Context) ([]Model, error)
	GetModelCapabilities(ctx context.Context, modelID string) ([]string, error)
}

// FeedbackStore interface for feedback data
type FeedbackStore interface {
	GetModelAccuracyByLanguage(ctx context.Context, modelID, language string) (float64, error)
	GetModelSuccessRate(ctx context.Context, modelID string) (float64, error)
}

// Model represents a model for suggestion
type Model struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Provider     string   `json:"provider"`
	Capabilities []string `json:"capabilities"`
	ContextSize  int      `json:"context_size"`
	Speed        string   `json:"speed"` // fast, medium, slow
	Quality      string   `json:"quality"` // standard, high, premium
}

// NewService creates a new model suggestion service
func NewService(models ModelStore, feedback FeedbackStore) *Service {
	return &Service{
		modelStore:    models,
		feedbackStore: feedback,
	}
}

// SuggestModels suggests models for given languages and requirements
func (s *Service) SuggestModels(ctx context.Context, req *SuggestionRequest) (*SuggestionResponse, error) {
	// Get all active models
	models, err := s.modelStore.GetActiveModels(ctx)
	if err != nil {
		return nil, err
	}

	// Score each model
	scored := make([]ScoredModel, 0, len(models))
	for _, model := range models {
		score := s.scoreModel(ctx, model, req)
		if score.TotalScore > 0 {
			scored = append(scored, score)
		}
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].TotalScore > scored[j].TotalScore
	})

	// Take top N
	limit := req.Limit
	if limit <= 0 || limit > len(scored) {
		limit = len(scored)
	}
	if limit > 5 {
		limit = 5
	}

	response := &SuggestionResponse{
		Languages:     req.Languages,
		Suggestions:   scored[:limit],
	}

	if len(scored) > 0 {
		response.Recommended = &scored[0]
		response.RecommendedReason = s.generateReason(scored[0], req)
	}

	return response, nil
}

// SuggestionRequest contains suggestion parameters
type SuggestionRequest struct {
	Languages      []string `json:"languages"`
	FileCount      int      `json:"file_count"`
	TotalLines     int      `json:"total_lines"`
	Priority       string   `json:"priority"` // speed, quality, balanced
	RequireContext int      `json:"require_context"` // minimum context size
	Limit          int      `json:"limit"`
}

// SuggestionResponse contains suggested models
type SuggestionResponse struct {
	Languages         []string      `json:"languages"`
	Suggestions       []ScoredModel `json:"suggestions"`
	Recommended       *ScoredModel  `json:"recommended,omitempty"`
	RecommendedReason string        `json:"recommended_reason,omitempty"`
}

// ScoredModel contains a model with its score
type ScoredModel struct {
	Model           Model   `json:"model"`
	TotalScore      float64 `json:"total_score"`
	LanguageScore   float64 `json:"language_score"`
	QualityScore    float64 `json:"quality_score"`
	SpeedScore      float64 `json:"speed_score"`
	ContextScore    float64 `json:"context_score"`
	FeedbackScore   float64 `json:"feedback_score"`
	Reasoning       string  `json:"reasoning,omitempty"`
}

func (s *Service) scoreModel(ctx context.Context, model Model, req *SuggestionRequest) ScoredModel {
	scored := ScoredModel{Model: model}

	// Language score (based on model specialization)
	scored.LanguageScore = s.calculateLanguageScore(model, req.Languages)

	// Quality score
	scored.QualityScore = s.calculateQualityScore(model, req.Priority)

	// Speed score
	scored.SpeedScore = s.calculateSpeedScore(model, req.Priority)

	// Context score
	scored.ContextScore = s.calculateContextScore(model, req.RequireContext, req.TotalLines)

	// Feedback score (from historical data)
	if s.feedbackStore != nil {
		score, _ := s.feedbackStore.GetModelSuccessRate(ctx, model.ID)
		scored.FeedbackScore = score
	}

	// Calculate total score based on priority
	switch req.Priority {
	case "speed":
		scored.TotalScore = scored.LanguageScore*0.2 + 
			scored.SpeedScore*0.4 + 
			scored.QualityScore*0.2 + 
			scored.ContextScore*0.1 + 
			scored.FeedbackScore*0.1
	case "quality":
		scored.TotalScore = scored.LanguageScore*0.3 + 
			scored.SpeedScore*0.1 + 
			scored.QualityScore*0.4 + 
			scored.ContextScore*0.1 + 
			scored.FeedbackScore*0.1
	default: // balanced
		scored.TotalScore = scored.LanguageScore*0.25 + 
			scored.SpeedScore*0.2 + 
			scored.QualityScore*0.25 + 
			scored.ContextScore*0.15 + 
			scored.FeedbackScore*0.15
	}

	return scored
}

func (s *Service) calculateLanguageScore(model Model, languages []string) float64 {
	if len(languages) == 0 {
		return 0.5 // Neutral if no languages specified
	}

	// Models known to be good for specific languages
	modelLanguageStrength := map[string]map[string]float64{
		"gpt-4":           {"all": 0.9},
		"gpt-4-turbo":     {"all": 0.9},
		"claude-3-opus":   {"all": 0.9, "python": 0.95, "javascript": 0.95},
		"claude-3-sonnet": {"all": 0.85, "python": 0.9, "javascript": 0.9},
		"codellama":       {"python": 0.9, "go": 0.8, "rust": 0.8, "cpp": 0.85},
		"deepseek-coder":  {"python": 0.9, "javascript": 0.85, "typescript": 0.85, "go": 0.85},
		"starcoder":       {"python": 0.85, "javascript": 0.85, "typescript": 0.85},
		"mistral":         {"all": 0.7},
		"llama3":          {"all": 0.75},
	}

	// Check model strengths
	modelKey := strings.ToLower(model.Name)
	for key, strengths := range modelLanguageStrength {
		if strings.Contains(modelKey, key) {
			// Check for specific language strength
			maxStrength := 0.0
			for _, lang := range languages {
				if strength, ok := strengths[strings.ToLower(lang)]; ok {
					if strength > maxStrength {
						maxStrength = strength
					}
				} else if allStrength, ok := strengths["all"]; ok {
					if allStrength > maxStrength {
						maxStrength = allStrength
					}
				}
			}
			if maxStrength > 0 {
				return maxStrength
			}
		}
	}

	// Default score based on capabilities
	hasCodeCap := false
	for _, cap := range model.Capabilities {
		if cap == "code" || cap == "code-completion" {
			hasCodeCap = true
			break
		}
	}
	if hasCodeCap {
		return 0.6
	}

	return 0.4
}

func (s *Service) calculateQualityScore(model Model, priority string) float64 {
	qualityScores := map[string]float64{
		"premium":  1.0,
		"high":     0.8,
		"standard": 0.6,
	}

	if score, ok := qualityScores[model.Quality]; ok {
		return score
	}

	// Infer from model name
	name := strings.ToLower(model.Name)
	if strings.Contains(name, "gpt-4") || strings.Contains(name, "opus") {
		return 1.0
	}
	if strings.Contains(name, "sonnet") || strings.Contains(name, "turbo") {
		return 0.8
	}
	if strings.Contains(name, "haiku") || strings.Contains(name, "mini") {
		return 0.6
	}

	return 0.5
}

func (s *Service) calculateSpeedScore(model Model, priority string) float64 {
	speedScores := map[string]float64{
		"fast":   1.0,
		"medium": 0.7,
		"slow":   0.4,
	}

	if score, ok := speedScores[model.Speed]; ok {
		return score
	}

	// Infer from model name
	name := strings.ToLower(model.Name)
	if strings.Contains(name, "mini") || strings.Contains(name, "haiku") || strings.Contains(name, "flash") {
		return 1.0
	}
	if strings.Contains(name, "turbo") || strings.Contains(name, "sonnet") {
		return 0.8
	}
	if strings.Contains(name, "opus") || strings.Contains(name, "gpt-4") {
		return 0.5
	}

	return 0.6
}

func (s *Service) calculateContextScore(model Model, required, totalLines int) float64 {
	if required <= 0 {
		required = totalLines * 10 // Rough estimate: 10 tokens per line
	}
	if required <= 0 {
		return 0.5
	}

	contextSize := model.ContextSize
	if contextSize <= 0 {
		// Estimate from model name
		name := strings.ToLower(model.Name)
		switch {
		case strings.Contains(name, "128k"):
			contextSize = 128000
		case strings.Contains(name, "32k"):
			contextSize = 32000
		case strings.Contains(name, "16k"):
			contextSize = 16000
		case strings.Contains(name, "gpt-4"):
			contextSize = 8000
		default:
			contextSize = 4000
		}
	}

	if contextSize >= required*2 {
		return 1.0 // Plenty of room
	}
	if contextSize >= required {
		return 0.8 // Enough
	}
	if contextSize >= required/2 {
		return 0.5 // Might work with chunking
	}
	return 0.2 // Too small
}

func (s *Service) generateReason(scored ScoredModel, req *SuggestionRequest) string {
	var reasons []string

	if scored.LanguageScore > 0.8 {
		reasons = append(reasons, "excellent for "+strings.Join(req.Languages, ", "))
	}

	if req.Priority == "speed" && scored.SpeedScore > 0.8 {
		reasons = append(reasons, "fast processing")
	}
	if req.Priority == "quality" && scored.QualityScore > 0.8 {
		reasons = append(reasons, "high quality output")
	}

	if scored.FeedbackScore > 0.8 {
		reasons = append(reasons, "high user satisfaction")
	}

	if len(reasons) == 0 {
		return "Best overall match for your requirements"
	}

	return "Recommended: " + strings.Join(reasons, ", ")
}

// GetModelForLanguage returns the best model for a specific language
func (s *Service) GetModelForLanguage(ctx context.Context, language string) (*Model, error) {
	resp, err := s.SuggestModels(ctx, &SuggestionRequest{
		Languages: []string{language},
		Priority:  "balanced",
		Limit:     1,
	})
	if err != nil {
		return nil, err
	}

	if resp.Recommended != nil {
		return &resp.Recommended.Model, nil
	}

	return nil, nil
}

// GetFastModel returns the fastest model with acceptable quality
func (s *Service) GetFastModel(ctx context.Context) (*Model, error) {
	resp, err := s.SuggestModels(ctx, &SuggestionRequest{
		Priority: "speed",
		Limit:    1,
	})
	if err != nil {
		return nil, err
	}

	if resp.Recommended != nil {
		return &resp.Recommended.Model, nil
	}

	return nil, nil
}

// GetQualityModel returns the highest quality model
func (s *Service) GetQualityModel(ctx context.Context) (*Model, error) {
	resp, err := s.SuggestModels(ctx, &SuggestionRequest{
		Priority: "quality",
		Limit:    1,
	})
	if err != nil {
		return nil, err
	}

	if resp.Recommended != nil {
		return &resp.Recommended.Model, nil
	}

	return nil, nil
}


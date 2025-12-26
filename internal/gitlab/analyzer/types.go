// Package analyzer provides code analysis service for GitLab MR reviews
package analyzer

import (
	"time"
)

// AnalysisConfig конфигурация анализа
type AnalysisConfig struct {
	// LLM Settings
	AnalysisModel   string  `json:"analysis_model"`
	EmbeddingModel  string  `json:"embedding_model"`
	Temperature     float64 `json:"temperature"`
	MaxTokens       int     `json:"max_tokens"`
	
	// Review Settings
	CustomPrompt    string   `json:"custom_prompt,omitempty"`
	FileFilters     []string `json:"file_filters,omitempty"`
	MaxFiles        int      `json:"max_files"`
	MaxLinesPerFile int      `json:"max_lines_per_file"`
	
	// Chunking Settings
	ChunkSize       int `json:"chunk_size"`       // Tokens per chunk
	ChunkOverlap    int `json:"chunk_overlap"`    // Overlap between chunks
	
	// Vector Settings
	UseEmbeddings   bool    `json:"use_embeddings"`
	TopKChunks      int     `json:"top_k_chunks"`
	MinSimilarity   float64 `json:"min_similarity"`
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() AnalysisConfig {
	return AnalysisConfig{
		Temperature:     0.3,
		MaxTokens:       4096,
		MaxFiles:        50,
		MaxLinesPerFile: 1000,
		ChunkSize:       1000,
		ChunkOverlap:    100,
		UseEmbeddings:   true,
		TopKChunks:      10,
		MinSimilarity:   0.7,
	}
}

// AnalysisRequest запрос на анализ MR
type AnalysisRequest struct {
	// MR Info
	ProjectID      string `json:"project_id"`
	MRIID          int    `json:"mr_iid"`
	MRTitle        string `json:"mr_title"`
	MRDescription  string `json:"mr_description"`
	MRAuthor       string `json:"mr_author"`
	SourceBranch   string `json:"source_branch"`
	TargetBranch   string `json:"target_branch"`
	
	// Files
	Changes        []FileChange `json:"changes"`
	
	// Config
	Config         AnalysisConfig `json:"config"`
}

// FileChange изменения в файле
type FileChange struct {
	FilePath    string `json:"file_path"`
	OldPath     string `json:"old_path,omitempty"` // Для переименований
	NewFile     bool   `json:"new_file"`
	DeletedFile bool   `json:"deleted_file"`
	RenamedFile bool   `json:"renamed_file"`
	Diff        string `json:"diff"`
	NewContent  string `json:"new_content,omitempty"` // Полный контент нового файла
	Language    string `json:"language,omitempty"`
	LinesAdded  int    `json:"lines_added"`
	LinesRemoved int   `json:"lines_removed"`
}

// AnalysisResult результат анализа
type AnalysisResult struct {
	Summary       string           `json:"summary"`
	OverallScore  int              `json:"overall_score"` // 0-100
	Categories    []CategoryResult `json:"categories"`
	FileReviews   []FileReview     `json:"file_reviews"`
	Suggestions   []Suggestion     `json:"suggestions"`
	
	// Metrics
	FilesAnalyzed   int           `json:"files_analyzed"`
	LinesChanged    int           `json:"lines_changed"`
	IssuesFound     int           `json:"issues_found"`
	ProcessingTime  time.Duration `json:"processing_time"`
	TokensUsed      int           `json:"tokens_used"`
	Model           string        `json:"model"`
}

// CategoryResult результат по категории
type CategoryResult struct {
	Name     string `json:"name"`      // security, bugs, style, performance
	Score    int    `json:"score"`     // 0-100
	Issues   int    `json:"issues"`
	Details  string `json:"details"`
}

// FileReview результат анализа файла
type FileReview struct {
	FilePath    string      `json:"file_path"`
	Score       int         `json:"score"`       // 0-100
	LineIssues  []LineIssue `json:"line_issues"`
	Summary     string      `json:"summary"`
	Language    string      `json:"language,omitempty"`
}

// LineIssue проблема на конкретной строке
type LineIssue struct {
	Line        int    `json:"line"`
	EndLine     int    `json:"end_line,omitempty"` // Для multi-line issues
	Severity    string `json:"severity"`           // critical, warning, info, suggestion
	Category    string `json:"category"`           // security, bug, style, performance
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion,omitempty"` // Предложение исправления
	CodeSnippet string `json:"code_snippet,omitempty"`
}

// Suggestion общее предложение по MR
type Suggestion struct {
	Category    string `json:"category"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"` // high, medium, low
}

// Severity levels
const (
	SeverityCritical   = "critical"
	SeverityWarning    = "warning"
	SeverityInfo       = "info"
	SeveritySuggestion = "suggestion"
)

// Category types
const (
	CategorySecurity    = "security"
	CategoryBugs        = "bugs"
	CategoryStyle       = "style"
	CategoryPerformance = "performance"
	CategoryBestPractice = "best_practice"
)

// LLMRequest запрос к LLM
type LLMRequest struct {
	Model       string       `json:"model"`
	Messages    []LLMMessage `json:"messages"`
	Temperature float64      `json:"temperature,omitempty"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
	Stream      bool         `json:"stream"`
}

// LLMMessage сообщение для LLM
type LLMMessage struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

// LLMResponse ответ от LLM
type LLMResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// CodeChunk chunk кода для embedding
type CodeChunk struct {
	ID          string            `json:"id"`
	FilePath    string            `json:"file_path"`
	ChangeType  string            `json:"change_type"` // added, modified, deleted
	LineStart   int               `json:"line_start"`
	LineEnd     int               `json:"line_end"`
	Content     string            `json:"content"`
	Language    string            `json:"language"`
	ChunkIndex  int               `json:"chunk_index"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Issue represents a code issue found during analysis
type Issue struct {
	FilePath   string `json:"file_path"`
	Line       int    `json:"line"`
	EndLine    int    `json:"end_line,omitempty"`
	Severity   string `json:"severity"`
	Category   string `json:"category"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// SuggestionItem represents an improvement suggestion
type SuggestionItem struct {
	FilePath    string `json:"file_path,omitempty"`
	Line        int    `json:"line,omitempty"`
	Type        string `json:"type"`
	Category    string `json:"category,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
}

// AnalysisResultParsed simplified result for processor
type AnalysisResultParsed struct {
	Summary     string           `json:"summary"`
	Score       int              `json:"overall_score"`
	Issues      []Issue          `json:"issues"`
	Suggestions []SuggestionItem `json:"suggestions"`
}


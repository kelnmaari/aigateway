// Package ollama provides data models for Ollama API
package ollama

import "time"

// ModelsResponse представляет ответ на запрос списка моделей (/api/tags)
type ModelsResponse struct {
	Models []Model `json:"models"`
}

// Model представляет информацию о модели в Ollama
type Model struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    Details   `json:"details"`
}

// Details содержит детальную информацию о модели
type Details struct {
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// ChatRequest представляет запрос для /api/chat
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`           // КРИТИЧНО: Без omitempty - всегда отправляем явно stream:false
	Format   string        `json:"format,omitempty"` // "json" for structured output
	Options  *ChatOptions  `json:"options,omitempty"`
	Think    *bool         `json:"think,omitempty"` // Для thinking моделей: включить/выключить рассуждения
	Tools    []Tool        `json:"tools,omitempty"` // Tools для function calling
}

// ChatMessage представляет сообщение в чате
type ChatMessage struct {
	Role      string     `json:"role"` // "system", "user", "assistant", "tool"
	Content   string     `json:"content"`
	Thinking  string     `json:"thinking,omitempty"`   // Для gpt-oss моделей с reasoning
	ToolCalls []ToolCall `json:"tool_calls,omitempty"` // Вызовы инструментов от модели
	ToolName  string     `json:"tool_name,omitempty"`  // Имя инструмента для role="tool"
	Images    [][]byte   `json:"images,omitempty"`     // Изображения для vision models (raw bytes)
}

// Tool представляет инструмент доступный для модели
type Tool struct {
	Type     string   `json:"type"` // "function"
	Function Function `json:"function"`
}

// Function представляет функцию доступную для вызова
type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"` // JSON Schema
}

// ToolCall представляет вызов инструмента
type ToolCall struct {
	Function FunctionCall `json:"function"`
}

// FunctionCall представляет вызов функции
type FunctionCall struct {
	Name      string      `json:"name"`
	Arguments interface{} `json:"arguments"` // map[string]interface{} или string
}

// ChatOptions содержит опции для генерации
type ChatOptions struct {
	Temperature   *float64 `json:"temperature,omitempty"`    // 0.0 to 1.0
	TopK          *int     `json:"top_k,omitempty"`          // Top-K sampling
	TopP          *float64 `json:"top_p,omitempty"`          // Top-P sampling
	RepeatPenalty *float64 `json:"repeat_penalty,omitempty"` // Penalty for repetition
	Seed          *int     `json:"seed,omitempty"`           // Random seed
	NumCtx        *int     `json:"num_ctx,omitempty"`        // Context length
	NumPredict    *int     `json:"num_predict,omitempty"`    // Max tokens to predict
	Stop          []string `json:"stop,omitempty"`           // Stop sequences
}

// ChatResponse представляет ответ от /api/chat
type ChatResponse struct {
	Model              string      `json:"model"`
	CreatedAt          time.Time   `json:"created_at"`
	Message            ChatMessage `json:"message"`
	Done               bool        `json:"done"`
	TotalDuration      int64       `json:"total_duration,omitempty"`
	LoadDuration       int64       `json:"load_duration,omitempty"`
	PromptEvalCount    int         `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64       `json:"prompt_eval_duration,omitempty"`
	EvalCount          int         `json:"eval_count,omitempty"`
	EvalDuration       int64       `json:"eval_duration,omitempty"`
	ToolCalls          []ToolCall  `json:"tool_calls,omitempty"` // 🔧 КРИТИЧНО: tool_calls могут быть на уровне ответа (как в официальном API)!
}

// GenerateRequest представляет запрос для /api/generate
type GenerateRequest struct {
	Model   string           `json:"model"`
	Prompt  string           `json:"prompt"`
	Stream  bool             `json:"stream,omitempty"`
	Format  string           `json:"format,omitempty"`
	Options *GenerateOptions `json:"options,omitempty"`
	System  string           `json:"system,omitempty"`
	Context []int            `json:"context,omitempty"`
}

// GenerateOptions содержит опции для генерации текста
type GenerateOptions struct {
	Temperature   *float64 `json:"temperature,omitempty"`
	TopK          *int     `json:"top_k,omitempty"`
	TopP          *float64 `json:"top_p,omitempty"`
	RepeatPenalty *float64 `json:"repeat_penalty,omitempty"`
	Seed          *int     `json:"seed,omitempty"`
	NumCtx        *int     `json:"num_ctx,omitempty"`
	NumPredict    *int     `json:"num_predict,omitempty"`
	Stop          []string `json:"stop,omitempty"`
}

// GenerateResponse представляет ответ от /api/generate
type GenerateResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	Context            []int     `json:"context,omitempty"`
	TotalDuration      int64     `json:"total_duration,omitempty"`
	LoadDuration       int64     `json:"load_duration,omitempty"`
	PromptEvalCount    int       `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64     `json:"prompt_eval_duration,omitempty"`
	EvalCount          int       `json:"eval_count,omitempty"`
	EvalDuration       int64     `json:"eval_duration,omitempty"`
}

// EmbedRequest представляет запрос для /api/embed (batch embeddings)
// Поддерживает множественные входы и расширенные опции
type EmbedRequest struct {
	Model      string         `json:"model"`
	Input      interface{}    `json:"input"`                // string, []string, []int, или [][]int
	Truncate   *bool          `json:"truncate,omitempty"`   // Обрезать ввод до макс. длины
	Dimensions int            `json:"dimensions,omitempty"` // Размерность output embedding
	KeepAlive  string         `json:"keep_alive,omitempty"` // "5m", "10m", "-1" (forever)
	Options    map[string]any `json:"options,omitempty"`    // Model-specific options
}

// EmbedResponse представляет ответ от /api/embed
type EmbedResponse struct {
	Model           string      `json:"model"`
	Embeddings      [][]float32 `json:"embeddings"` // Batch embeddings (float32)
	TotalDuration   int64       `json:"total_duration,omitempty"`
	LoadDuration    int64       `json:"load_duration,omitempty"`
	PromptEvalCount int         `json:"prompt_eval_count,omitempty"`
}

// EmbeddingsRequest представляет запрос для /api/embeddings (legacy single embedding)
type EmbeddingsRequest struct {
	Model     string         `json:"model"`
	Prompt    string         `json:"prompt"`
	KeepAlive string         `json:"keep_alive,omitempty"`
	Options   map[string]any `json:"options,omitempty"`
}

// EmbeddingsResponse представляет ответ от /api/embeddings (legacy)
type EmbeddingsResponse struct {
	Embedding []float64 `json:"embedding"` // Single embedding (float64)
}

// ShowRequest представляет запрос для /api/show (информация о модели)
type ShowRequest struct {
	Name string `json:"name"`
}

// ShowResponse представляет ответ от /api/show
type ShowResponse struct {
	License    string                 `json:"license,omitempty"`
	Modelfile  string                 `json:"modelfile,omitempty"`
	Parameters string                 `json:"parameters,omitempty"`
	Template   string                 `json:"template,omitempty"`
	System     string                 `json:"system,omitempty"`
	Details    *Details               `json:"details,omitempty"`
	ModelInfo  map[string]interface{} `json:"modelinfo,omitempty"`
	ModifiedAt time.Time              `json:"modified_at"`
}

// PullRequest представляет запрос для /api/pull (загрузка модели)
type PullRequest struct {
	Name     string `json:"name"`
	Insecure bool   `json:"insecure,omitempty"`
	Stream   bool   `json:"stream,omitempty"`
}

// PullResponse представляет ответ от /api/pull
type PullResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

// ErrorResponse представляет ошибку от Ollama API
type ErrorResponse struct {
	Error string `json:"error"`
}

// IsTemporaryError проверяет, является ли ошибка временной
func (e *ErrorResponse) IsTemporaryError() bool {
	// Некоторые ошибки можно ретраить
	return e.Error == "model not found" ||
		e.Error == "model not loaded" ||
		e.Error == "service unavailable"
}

// Package models provides data models for OpenAI API compatibility
package models

import "time"

// OpenAI Chat Completion Models

// ChatCompletionRequest представляет запрос на создание chat completion в OpenAI API
type ChatCompletionRequest struct {
	Model            string             `json:"model" binding:"required"`
	Messages         []ChatMessage      `json:"messages" binding:"required"`
	Temperature      *float64           `json:"temperature,omitempty"`
	TopP             *float64           `json:"top_p,omitempty"`
	N                *int               `json:"n,omitempty"`
	Stream           bool               `json:"stream,omitempty"`
	Stop             []string           `json:"stop,omitempty"`
	MaxTokens        *int               `json:"max_tokens,omitempty"`
	PresencePenalty  *float64           `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64           `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]float64 `json:"logit_bias,omitempty"`
	User             string             `json:"user,omitempty"`
	Tools            []Tool             `json:"tools,omitempty"`
	ToolChoice       interface{}        `json:"tool_choice,omitempty"`
	ResponseFormat   *ResponseFormat    `json:"response_format,omitempty"`
	Seed             *int               `json:"seed,omitempty"`
	// Ollama-specific options (v1.9.1+)
	Options map[string]interface{} `json:"options,omitempty"`
	// Дополнительные поля для функций
	Functions    []Function  `json:"functions,omitempty"`     // Deprecated
	FunctionCall interface{} `json:"function_call,omitempty"` // Deprecated
	
	// RAG System (v1.13.0+)
	RAGEnabled  bool     `json:"rag_enabled,omitempty"`  // Включить RAG
	RAGSourceIDs []string `json:"rag_source_ids,omitempty"` // Фильтр по источникам
	RAGTopK     int      `json:"rag_top_k,omitempty"`    // Количество chunks для retrieval
	RAGMinScore float64  `json:"rag_min_score,omitempty"` // Минимальный similarity score
	RAGRerank   bool     `json:"rag_rerank,omitempty"`   // Применять reranking
}

// ChatMessage представляет сообщение в чате
type ChatMessage struct {
	Role         string        `json:"role" binding:"required"` // system, user, assistant, tool
	Content      interface{}   `json:"content"`                 // string или array для multi-modal
	Name         string        `json:"name,omitempty"`
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID   string        `json:"tool_call_id,omitempty"`
	FunctionCall *FunctionCall `json:"function_call,omitempty"` // Deprecated
	FileIDs      []string      `json:"file_ids,omitempty"`      // FILE-STORAGE-01: Phase 4, v1.10.0+
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
	Parameters  interface{} `json:"parameters,omitempty"`
}

// ToolCall представляет вызов инструмента
type ToolCall struct {
	Index    int          `json:"index"` // ОБЯЗАТЕЛЬНО для streaming chunks (OpenAI требует)
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall представляет вызов функции
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ResponseFormat определяет формат ответа
type ResponseFormat struct {
	Type string `json:"type"` // "text" или "json_object"
}

// ChatCompletionResponse представляет ответ chat completion
type ChatCompletionResponse struct {
	ID                string                 `json:"id"`
	Object            string                 `json:"object"`
	Created           int64                  `json:"created"`
	Model             string                 `json:"model"`
	SystemFingerprint string                 `json:"system_fingerprint,omitempty"`
	Choices           []ChatCompletionChoice `json:"choices"`
	Usage             Usage                  `json:"usage"`
}

// ChatCompletionChoice представляет выбор в ответе
type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	LogProbs     *LogProbs   `json:"logprobs,omitempty"`
	FinishReason string      `json:"finish_reason"` // stop, length, tool_calls, content_filter, function_call
}

// ChatCompletionChunk представляет chunk в streaming ответе
type ChatCompletionChunk struct {
	ID                string                      `json:"id"`
	Object            string                      `json:"object"`
	Created           int64                       `json:"created"`
	Model             string                      `json:"model"`
	SystemFingerprint string                      `json:"system_fingerprint,omitempty"`
	Choices           []ChatCompletionChunkChoice `json:"choices"`
}

// ChatCompletionChunkChoice представляет выбор в streaming chunk
type ChatCompletionChunkChoice struct {
	Index        int         `json:"index"`
	Delta        ChatMessage `json:"delta"`
	LogProbs     *LogProbs   `json:"logprobs,omitempty"`
	FinishReason *string     `json:"finish_reason"`
}

// OpenAI Models API

// ModelsResponse представляет ответ на запрос списка моделей
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// Model представляет информацию о модели
type Model struct {
	ID         string       `json:"id"`
	Object     string       `json:"object"`
	Created    int64        `json:"created"`
	OwnedBy    string       `json:"owned_by"`
	Permission []Permission `json:"permission"`

	// Extended Ollama-specific fields (optional, won't break OpenAI compatibility)
	Size              *int64  `json:"size,omitempty"`               // Model size in bytes
	Digest            *string `json:"digest,omitempty"`             // Model digest/hash
	Format            *string `json:"format,omitempty"`             // Model format (e.g., "gguf")
	Family            *string `json:"family,omitempty"`             // Model family (e.g., "llama", "qwen")
	ParameterSize     *string `json:"parameter_size,omitempty"`     // Parameter size (e.g., "7B", "30B")
	QuantizationLevel *string `json:"quantization_level,omitempty"` // Quantization (e.g., "Q4_K_M")
	ModifiedAt        *int64  `json:"modified_at,omitempty"`        // Last modified timestamp
}

// Permission представляет разрешение для модели
type Permission struct {
	ID                 string  `json:"id"`
	Object             string  `json:"object"`
	Created            int64   `json:"created"`
	AllowCreateEngine  bool    `json:"allow_create_engine"`
	AllowSampling      bool    `json:"allow_sampling"`
	AllowLogProbs      bool    `json:"allow_logprobs"`
	AllowSearchIndices bool    `json:"allow_search_indices"`
	AllowView          bool    `json:"allow_view"`
	AllowFineTuning    bool    `json:"allow_fine_tuning"`
	Organization       string  `json:"organization"`
	Group              *string `json:"group"`
	IsBlocking         bool    `json:"is_blocking"`
}

// OpenAI Completions API (Legacy)

// CompletionRequest представляет запрос text completion
type CompletionRequest struct {
	Model            string             `json:"model" binding:"required"`
	Prompt           interface{}        `json:"prompt"` // string, array of strings, array of tokens, or array of token arrays
	Suffix           string             `json:"suffix,omitempty"`
	MaxTokens        *int               `json:"max_tokens,omitempty"`
	Temperature      *float64           `json:"temperature,omitempty"`
	TopP             *float64           `json:"top_p,omitempty"`
	N                *int               `json:"n,omitempty"`
	Stream           bool               `json:"stream,omitempty"`
	LogProbs         *int               `json:"logprobs,omitempty"`
	Echo             bool               `json:"echo,omitempty"`
	Stop             []string           `json:"stop,omitempty"`
	PresencePenalty  *float64           `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64           `json:"frequency_penalty,omitempty"`
	BestOf           *int               `json:"best_of,omitempty"`
	LogitBias        map[string]float64 `json:"logit_bias,omitempty"`
	User             string             `json:"user,omitempty"`
}

// CompletionResponse представляет ответ text completion
type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   Usage              `json:"usage"`
}

// CompletionChoice представляет выбор в text completion
type CompletionChoice struct {
	Text         string    `json:"text"`
	Index        int       `json:"index"`
	LogProbs     *LogProbs `json:"logprobs,omitempty"`
	FinishReason string    `json:"finish_reason"`
}

// CompletionStreamChunk представляет streaming chunk для text completion
type CompletionStreamChunk struct {
	ID      string                   `json:"id"`
	Object  string                   `json:"object"`
	Created int64                    `json:"created"`
	Model   string                   `json:"model"`
	Choices []CompletionStreamChoice `json:"choices"`
}

// CompletionStreamChoice представляет choice в streaming completion
type CompletionStreamChoice struct {
	Text         string `json:"text"`
	Index        int    `json:"index"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// OpenAI Embeddings API

// EmbeddingRequest представляет запрос на создание embeddings
type EmbeddingRequest struct {
	Input          interface{} `json:"input" binding:"required"` // string, array of strings, array of integers, or array of arrays
	Model          string      `json:"model" binding:"required"`
	EncodingFormat string      `json:"encoding_format,omitempty"` // "float" или "base64"
	Dimensions     *int        `json:"dimensions,omitempty"`
	User           string      `json:"user,omitempty"`
}

// EmbeddingResponse представляет ответ с embeddings
type EmbeddingResponse struct {
	Object string      `json:"object"`
	Data   []Embedding `json:"data"`
	Model  string      `json:"model"`
	Usage  Usage       `json:"usage"`
}

// Embedding представляет одно embedding
type Embedding struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

// Общие структуры

// Usage представляет информацию об использовании токенов
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"` // ОБЯЗАТЕЛЬНО для OpenAI API совместимости
	TotalTokens      int `json:"total_tokens"`
}

// LogProbs представляет log probabilities
type LogProbs struct {
	Tokens        []string             `json:"tokens"`
	TokenLogProbs []float64            `json:"token_logprobs"`
	TopLogProbs   []map[string]float64 `json:"top_logprobs"`
	TextOffset    []int                `json:"text_offset"`
}

// Error представляет ошибку в OpenAI API формате
type Error struct {
	Message string      `json:"message"`
	Type    string      `json:"type"`
	Param   *string     `json:"param,omitempty"`
	Code    interface{} `json:"code,omitempty"` // string или int
}

// ErrorResponse представляет ответ с ошибкой
type ErrorResponse struct {
	Error Error `json:"error"`
}

// Вспомогательные функции

// GetContentAsString извлекает содержимое сообщения как строку
func (m *ChatMessage) GetContentAsString() string {
	if content, ok := m.Content.(string); ok {
		return content
	}
	return ""
}

// SetContentAsString устанавливает содержимое сообщения как строку
func (m *ChatMessage) SetContentAsString(content string) {
	m.Content = content
}

// IsSystemMessage проверяет является ли сообщение системным
func (m *ChatMessage) IsSystemMessage() bool {
	return m.Role == "system"
}

// IsUserMessage проверяет является ли сообщение пользователя
func (m *ChatMessage) IsUserMessage() bool {
	return m.Role == "user"
}

// IsAssistantMessage проверяет является ли сообщение ассистента
func (m *ChatMessage) IsAssistantMessage() bool {
	return m.Role == "assistant"
}

// HasToolCalls проверяет есть ли вызовы инструментов в сообщении
func (m *ChatMessage) HasToolCalls() bool {
	return len(m.ToolCalls) > 0
}

// NewChatCompletionResponse создает новый базовый ответ chat completion
func NewChatCompletionResponse(id, model string) *ChatCompletionResponse {
	return &ChatCompletionResponse{
		ID:      id,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: make([]ChatCompletionChoice, 0),
		Usage:   Usage{},
	}
}

// NewChatCompletionChunk создает новый streaming chunk
func NewChatCompletionChunk(id, model string) *ChatCompletionChunk {
	return &ChatCompletionChunk{
		ID:      id,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: make([]ChatCompletionChunkChoice, 0),
	}
}


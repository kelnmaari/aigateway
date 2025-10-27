// Package models provides data structures for Ollama-OpenAI Proxy
package models

import "time"

// OllamaModel represents a model from Ollama /api/tags endpoint
type OllamaModel struct {
	Name       string    `json:"name"`
	Model      string    `json:"model"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
	Details    struct {
		ParentModel       string   `json:"parent_model"`
		Format            string   `json:"format"`
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
	} `json:"details"`
}

// OllamaModelDetails represents detailed model information from /api/show endpoint
type OllamaModelDetails struct {
	ModelInfo  ModelInfo  `json:"modelinfo"`
	License    string     `json:"license"`
	Modelfile  string     `json:"modelfile"`
	Parameters string     `json:"parameters"`
	Template   string     `json:"template"`
	Details    ModelSpecs `json:"details"`
}

// ModelInfo contains architecture and parameter details
type ModelInfo struct {
	Architecture              string  `json:"general.architecture"`
	FileType                  string  `json:"general.file_type"`
	ParameterCount            int64   `json:"general.parameter_count"`
	QuantizationLevel         string  `json:"general.quantization_version"`
	Tokenizer                 string  `json:"tokenizer.ggml.model"`
	ContextLength             int     `json:"llama.context_length"`
	EmbeddingLength           int     `json:"llama.embedding_length"`
	BlockCount                int     `json:"llama.block_count"`
	FeedForwardLength         int     `json:"llama.feed_forward_length"`
	AttentionHeadCount        int     `json:"llama.attention.head_count"`
	AttentionHeadCountKV      int     `json:"llama.attention.head_count_kv"`
	AttentionLayerNormEpsilon float64 `json:"llama.attention.layer_norm_rms_epsilon"`
	RopeFreqBase              float64 `json:"llama.rope.freq_base"`
	BOS_TokenID               int     `json:"tokenizer.ggml.bos_token_id"`
	EOS_TokenID               int     `json:"tokenizer.ggml.eos_token_id"`
	VocabSize                 int     `json:"tokenizer.ggml.vocab_size"`
}

// ModelSpecs contains model specifications
type ModelSpecs struct {
	ParentModel       string   `json:"parent_model"`
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// ModelDetailsResponse is the response for /api/admin/models/:name/details
type ModelDetailsResponse struct {
	Name          string                 `json:"name"`
	Size          int64                  `json:"size"`
	ModifiedAt    time.Time              `json:"modified_at"`
	Digest        string                 `json:"digest"`
	Format        string                 `json:"format"`
	Family        string                 `json:"family"`
	ParameterSize string                 `json:"parameter_size"`
	Quantization  string                 `json:"quantization"`
	Architecture  string                 `json:"architecture"`
	ContextLength int                    `json:"context_length"`
	EmbeddingSize int                    `json:"embedding_size"`
	Layers        int                    `json:"layers"`
	Heads         int                    `json:"attention_heads"`
	VocabSize     int                    `json:"vocab_size"`
	License       string                 `json:"license,omitempty"`
	Template      string                 `json:"template,omitempty"`
	Modelfile     string                 `json:"modelfile,omitempty"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
}


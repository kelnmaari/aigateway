// Package yzma provides local LLM inference using yzma library
// Version: v3.0.0 - YZMA-01: Local inference without Ollama
package yzma

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hybridgroup/yzma/pkg/llama"
	"github.com/sirupsen/logrus"
)

// Client wraps yzma for local inference
type Client struct {
	modelsDir string
	logger    *logrus.Logger
	libPath   string
	
	// Loaded models cache
	models      map[string]*ModelContext
	modelsMu    sync.RWMutex
	
	// Default parameters
	contextSize  uint32
	batchSize    uint32
	uBatchSize   uint32
	temperature  float32
	topK         int32
	topP         float32
	minP         float32
	
	// Stats
	totalRequests int64
	totalTokens   int64
	statsMu       sync.RWMutex
	
	// Initialized flag
	initialized bool
}

// ModelContext holds loaded model, context, and sampler
type ModelContext struct {
	Model    llama.Model
	Vocab    llama.Vocab
	Ctx      llama.Context
	Sampler  llama.Sampler
	Path     string
	LoadedAt time.Time
	
	// VLM Support (v3.0.4+)
	IsVLM      bool
	MtmdCtx    uint64  // Multimodal context for VLM (mtmd.Context stored as uint64)
	MMProjPath string  // Path to mmproj file
}

// ClientConfig represents yzma client configuration
type ClientConfig struct {
	ModelsDir   string
	LibPath     string  // Path to llama.cpp shared library
	ContextSize uint32  // Context window size (default: 4096)
	BatchSize   uint32  // Logical batch size (default: 2048)
	UBatchSize  uint32  // Physical batch size (default: 2048)
	Temperature float32 // Default sampling temperature (default: 0.7)
	TopK        int32   // Top-K sampling (default: 40)
	TopP        float32 // Top-P sampling (default: 0.9)
	MinP        float32 // Min-P sampling (default: 0.1)
	Verbose     bool    // Enable llama.cpp logging
}

// NewClient creates a new yzma client
func NewClient(config ClientConfig, logger *logrus.Logger) (*Client, error) {
	if config.ModelsDir == "" {
		return nil, fmt.Errorf("models directory is required")
	}
	
	// Ensure models directory exists
	if err := os.MkdirAll(config.ModelsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create models directory: %w", err)
	}
	
	// Get library path from env or config
	libPath := config.LibPath
	if libPath == "" {
		libPath = os.Getenv("YZMA_LIB")
	}
	if libPath == "" {
		return nil, fmt.Errorf("YZMA_LIB environment variable or LibPath must be set")
	}
	
	// Set defaults
	if config.ContextSize == 0 {
		config.ContextSize = 4096
	}
	if config.BatchSize == 0 {
		config.BatchSize = 2048
	}
	if config.UBatchSize == 0 {
		config.UBatchSize = 2048
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}
	if config.TopK == 0 {
		config.TopK = 40
	}
	if config.TopP == 0 {
		config.TopP = 0.9
	}
	if config.MinP == 0 {
		config.MinP = 0.1
	}
	
	client := &Client{
		modelsDir:   config.ModelsDir,
		logger:      logger,
		libPath:     libPath,
		models:      make(map[string]*ModelContext),
		contextSize: config.ContextSize,
		batchSize:   config.BatchSize,
		uBatchSize:  config.UBatchSize,
		temperature: config.Temperature,
		topK:        config.TopK,
		topP:        config.TopP,
		minP:        config.MinP,
	}
	
	// Load llama.cpp library
	if err := llama.Load(libPath); err != nil {
		return nil, fmt.Errorf("failed to load llama.cpp library: %w", err)
	}
	
	// Suppress logs if not verbose
	if !config.Verbose {
		llama.LogSet(llama.LogSilent())
	}
	
	// Initialize backend
	llama.Init()
	client.initialized = true
	
	logger.WithFields(logrus.Fields{
		"models_dir":   config.ModelsDir,
		"lib_path":     libPath,
		"context_size": config.ContextSize,
		"batch_size":   config.BatchSize,
	}).Info("yzma client initialized successfully")
	
	return client, nil
}

// LoadModel loads a GGUF model from disk
func (c *Client) LoadModel(ctx context.Context, modelPath string) error {
	c.modelsMu.Lock()
	defer c.modelsMu.Unlock()
	
	// Check if already loaded
	if _, exists := c.models[modelPath]; exists {
		c.logger.WithField("model_path", modelPath).Debug("Model already loaded")
		return nil
	}
	
	// Verify model file exists
	fullPath := filepath.Join(c.modelsDir, modelPath)
	if _, err := os.Stat(fullPath); err != nil {
		return fmt.Errorf("model file not found: %s", fullPath)
	}
	
	c.logger.WithField("model_path", fullPath).Info("Loading model...")
	startTime := time.Now()
	
	// Load model
	mParams := llama.ModelDefaultParams()
	model := llama.ModelLoadFromFile(fullPath, mParams)
	if model == 0 {
		return fmt.Errorf("failed to load model from file: %s", fullPath)
	}
	
	vocab := llama.ModelGetVocab(model)
	
	// Initialize context
	ctxParams := llama.ContextDefaultParams()
	ctxParams.NCtx = c.contextSize
	ctxParams.NBatch = c.batchSize
	ctxParams.NUbatch = c.uBatchSize
	
	lctx := llama.InitFromModel(model, ctxParams)
	if lctx == 0 {
		llama.ModelFree(model)
		return fmt.Errorf("failed to initialize context for model: %s", fullPath)
	}
	
	// Initialize sampler chain
	sampler := llama.SamplerChainInit(llama.SamplerChainDefaultParams())
	llama.SamplerChainAdd(sampler, llama.SamplerInitTopK(c.topK))
	llama.SamplerChainAdd(sampler, llama.SamplerInitTopP(c.topP, 1))
	llama.SamplerChainAdd(sampler, llama.SamplerInitMinP(c.minP, 1))
	llama.SamplerChainAdd(sampler, llama.SamplerInitTempExt(c.temperature, 0, 1.0))
	llama.SamplerChainAdd(sampler, llama.SamplerInitDist(llama.DefaultSeed))
	
	// Cache model context
	c.models[modelPath] = &ModelContext{
		Model:    model,
		Vocab:    vocab,
		Ctx:      lctx,
		Sampler:  sampler,
		Path:     fullPath,
		LoadedAt: time.Now(),
	}
	
	c.logger.WithFields(logrus.Fields{
		"model_path": modelPath,
		"load_time":  time.Since(startTime),
	}).Info("Model loaded successfully")
	
	return nil
}

// UnloadModel unloads a model from memory
func (c *Client) UnloadModel(modelPath string) error {
	c.modelsMu.Lock()
	defer c.modelsMu.Unlock()
	
	modelCtx, exists := c.models[modelPath]
	if !exists {
		return fmt.Errorf("model not loaded: %s", modelPath)
	}
	
	// Free resources
	llama.SamplerFree(modelCtx.Sampler)
	llama.Free(modelCtx.Ctx)
	llama.ModelFree(modelCtx.Model)
	
	delete(c.models, modelPath)
	
	c.logger.WithField("model_path", modelPath).Info("Model unloaded")
	
	return nil
}

// IsModelLoaded checks if a model is loaded
func (c *Client) IsModelLoaded(modelPath string) bool {
	c.modelsMu.RLock()
	defer c.modelsMu.RUnlock()
	
	_, exists := c.models[modelPath]
	return exists
}

// GetModelContext retrieves loaded model context
func (c *Client) GetModelContext(modelPath string) (*ModelContext, error) {
	c.modelsMu.RLock()
	defer c.modelsMu.RUnlock()
	
	modelCtx, exists := c.models[modelPath]
	if !exists {
		return nil, fmt.Errorf("model not loaded: %s", modelPath)
	}
	
	return modelCtx, nil
}

// ListLoadedModels returns all loaded models
func (c *Client) ListLoadedModels() []string {
	c.modelsMu.RLock()
	defer c.modelsMu.RUnlock()
	
	models := make([]string, 0, len(c.models))
	for path := range c.models {
		models = append(models, path)
	}
	
	return models
}

// ListAvailableModels scans models directory for GGUF files
func (c *Client) ListAvailableModels() ([]string, error) {
	var models []string
	
	err := filepath.Walk(c.modelsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && filepath.Ext(path) == ".gguf" {
			// Get relative path
			relPath, err := filepath.Rel(c.modelsDir, path)
			if err != nil {
				return err
			}
			models = append(models, relPath)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to scan models directory: %w", err)
	}
	
	return models, nil
}

// GenerateRequest represents a text generation request
type GenerateRequest struct {
	ModelPath     string   // Path to GGUF model (relative to models dir)
	Messages      []ChatMessage // Chat messages for chat format
	Prompt        string   // Raw prompt (alternative to Messages)
	MaxTokens     int      // Maximum tokens to generate
	Temperature   float32  // Sampling temperature (0.0 to 2.0)
	TopP          float32  // Nucleus sampling threshold
	TopK          int32    // Top-K sampling
	MinP          float32  // Min-P sampling
	Stop          []string // Stop sequences
	Stream        bool     // Enable streaming
	SystemPrompt  string   // System prompt for chat
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string // "system", "user", "assistant"
	Content string
}

// GenerateResponse represents a text generation response
type GenerateResponse struct {
	Content       string        // Generated text
	TokensUsed    int           // Tokens used in generation
	PromptTokens  int           // Tokens in prompt
	Duration      time.Duration // Generation duration
	TokensPerSec  float64       // Tokens per second
	FinishReason  string        // Reason for stopping ("stop", "length", "error")
}

// Generate performs text generation
func (c *Client) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	startTime := time.Now()
	
	// Load model if not already loaded
	if !c.IsModelLoaded(req.ModelPath) {
		if err := c.LoadModel(ctx, req.ModelPath); err != nil {
			return nil, fmt.Errorf("failed to load model: %w", err)
		}
	}
	
	modelCtx, err := c.GetModelContext(req.ModelPath)
	if err != nil {
		return nil, err
	}
	
	// Prepare prompt
	var promptText string
	if len(req.Messages) > 0 {
		// Use chat template
		promptText = c.formatChatMessages(modelCtx.Model, req.Messages, req.SystemPrompt)
	} else {
		promptText = req.Prompt
	}
	
	// Set defaults
	if req.MaxTokens <= 0 {
		req.MaxTokens = int(c.contextSize)
	}
	
	c.logger.WithFields(logrus.Fields{
		"model_path": req.ModelPath,
		"prompt_len": len(promptText),
		"max_tokens": req.MaxTokens,
	}).Debug("Generating text")
	
	// Tokenize prompt
	tokens := llama.Tokenize(modelCtx.Vocab, promptText, true, true)
	promptTokens := len(tokens)
	
	batch := llama.BatchGetOne(tokens)
	
	// Generate response
	var response strings.Builder
	tokensGenerated := 0
	finishReason := "stop"
	
	for pos := int32(0); pos < int32(req.MaxTokens); pos += batch.NTokens {
		// Check context cancellation
		select {
		case <-ctx.Done():
			finishReason = "cancelled"
			goto done
		default:
		}
		
		llama.Decode(modelCtx.Ctx, batch)
		token := llama.SamplerSample(modelCtx.Sampler, modelCtx.Ctx, -1)
		
		// Check for end of generation
		if llama.VocabIsEOG(modelCtx.Vocab, token) {
			finishReason = "stop"
			break
		}
		
		// Convert token to text
		buf := make([]byte, 256)
		length := llama.TokenToPiece(modelCtx.Vocab, token, buf, 0, false)
		text := string(buf[:length])
		
		response.WriteString(text)
		tokensGenerated++
		
		// Check stop sequences
		if c.containsStopSequence(response.String(), req.Stop) {
			finishReason = "stop"
			break
		}
		
		batch = llama.BatchGetOne([]llama.Token{token})
	}
	
	if tokensGenerated >= req.MaxTokens {
		finishReason = "length"
	}
	
done:
	duration := time.Since(startTime)
	tokensPerSec := 0.0
	if duration.Seconds() > 0 {
		tokensPerSec = float64(tokensGenerated) / duration.Seconds()
	}
	
	// Update stats
	c.statsMu.Lock()
	c.totalRequests++
	c.totalTokens += int64(promptTokens + tokensGenerated)
	c.statsMu.Unlock()
	
	c.logger.WithFields(logrus.Fields{
		"model_path":     req.ModelPath,
		"tokens_used":    tokensGenerated,
		"prompt_tokens":  promptTokens,
		"duration":       duration,
		"tokens_per_sec": tokensPerSec,
	}).Info("Generation completed")
	
	return &GenerateResponse{
		Content:      response.String(),
		TokensUsed:   tokensGenerated,
		PromptTokens: promptTokens,
		Duration:     duration,
		TokensPerSec: tokensPerSec,
		FinishReason: finishReason,
	}, nil
}

// StreamingCallback handles streaming tokens
type StreamingCallback func(token string) error

// GenerateStream performs streaming text generation
func (c *Client) GenerateStream(ctx context.Context, req GenerateRequest, callback StreamingCallback) (*GenerateResponse, error) {
	startTime := time.Now()
	
	// Load model if not already loaded
	if !c.IsModelLoaded(req.ModelPath) {
		if err := c.LoadModel(ctx, req.ModelPath); err != nil {
			return nil, fmt.Errorf("failed to load model: %w", err)
		}
	}
	
	modelCtx, err := c.GetModelContext(req.ModelPath)
	if err != nil {
		return nil, err
	}
	
	// Prepare prompt
	var promptText string
	if len(req.Messages) > 0 {
		promptText = c.formatChatMessages(modelCtx.Model, req.Messages, req.SystemPrompt)
	} else {
		promptText = req.Prompt
	}
	
	// Set defaults
	if req.MaxTokens <= 0 {
		req.MaxTokens = int(c.contextSize)
	}
	
	c.logger.WithFields(logrus.Fields{
		"model_path": req.ModelPath,
		"streaming":  true,
	}).Debug("Starting streaming generation")
	
	// Tokenize prompt
	tokens := llama.Tokenize(modelCtx.Vocab, promptText, true, true)
	promptTokens := len(tokens)
	
	batch := llama.BatchGetOne(tokens)
	
	// Stream response
	var fullResponse strings.Builder
	tokensGenerated := 0
	finishReason := "stop"
	
	for pos := int32(0); pos < int32(req.MaxTokens); pos += batch.NTokens {
		// Check context cancellation
		select {
		case <-ctx.Done():
			finishReason = "cancelled"
			goto done
		default:
		}
		
		llama.Decode(modelCtx.Ctx, batch)
		token := llama.SamplerSample(modelCtx.Sampler, modelCtx.Ctx, -1)
		
		// Check for end of generation
		if llama.VocabIsEOG(modelCtx.Vocab, token) {
			finishReason = "stop"
			break
		}
		
		// Convert token to text
		buf := make([]byte, 256)
		length := llama.TokenToPiece(modelCtx.Vocab, token, buf, 0, false)
		text := string(buf[:length])
		
		fullResponse.WriteString(text)
		tokensGenerated++
		
		// Stream token to callback
		if err := callback(text); err != nil {
			return nil, fmt.Errorf("streaming callback error: %w", err)
		}
		
		// Check stop sequences
		if c.containsStopSequence(fullResponse.String(), req.Stop) {
			finishReason = "stop"
			break
		}
		
		batch = llama.BatchGetOne([]llama.Token{token})
	}
	
	if tokensGenerated >= req.MaxTokens {
		finishReason = "length"
	}
	
done:
	duration := time.Since(startTime)
	tokensPerSec := 0.0
	if duration.Seconds() > 0 {
		tokensPerSec = float64(tokensGenerated) / duration.Seconds()
	}
	
	// Update stats
	c.statsMu.Lock()
	c.totalRequests++
	c.totalTokens += int64(promptTokens + tokensGenerated)
	c.statsMu.Unlock()
	
	return &GenerateResponse{
		Content:      fullResponse.String(),
		TokensUsed:   tokensGenerated,
		PromptTokens: promptTokens,
		Duration:     duration,
		TokensPerSec: tokensPerSec,
		FinishReason: finishReason,
	}, nil
}

// formatChatMessages formats messages using chat template
func (c *Client) formatChatMessages(model llama.Model, messages []ChatMessage, systemPrompt string) string {
	// Build llama.ChatMessage array
	llamaMsgs := make([]llama.ChatMessage, 0, len(messages)+1)
	
	if systemPrompt != "" {
		llamaMsgs = append(llamaMsgs, llama.NewChatMessage("system", systemPrompt))
	}
	
	for _, msg := range messages {
		llamaMsgs = append(llamaMsgs, llama.NewChatMessage(msg.Role, msg.Content))
	}
	
	// Get chat template from model
	template := llama.ModelChatTemplate(model, "")
	if template == "" {
		template = "chatml" // Default to chatml
	}
	
	// Apply template
	buf := make([]byte, 8192)
	length := llama.ChatApplyTemplate(template, llamaMsgs, true, buf)
	
	return string(buf[:length])
}

// containsStopSequence checks if response contains any stop sequence
func (c *Client) containsStopSequence(response string, stopSequences []string) bool {
	for _, stop := range stopSequences {
		if strings.Contains(response, stop) {
			return true
		}
	}
	return false
}

// GetStats returns client statistics
func (c *Client) GetStats() (requests int64, tokens int64) {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	return c.totalRequests, c.totalTokens
}

// Shutdown gracefully shuts down the client
func (c *Client) Shutdown() error {
	c.logger.Info("Shutting down yzma client")
	
	c.modelsMu.Lock()
	defer c.modelsMu.Unlock()
	
	// Unload all models
	for path, modelCtx := range c.models {
		llama.SamplerFree(modelCtx.Sampler)
		llama.Free(modelCtx.Ctx)
		llama.ModelFree(modelCtx.Model)
		c.logger.WithField("model_path", path).Debug("Model freed")
	}
	
	c.models = make(map[string]*ModelContext)
	
	// Free backend
	if c.initialized {
		llama.BackendFree()
	}
	
	c.logger.Info("yzma client shut down successfully")
	return nil
}

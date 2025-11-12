// Package yzma provides local LLM inference using yzma library
// Version: v3.0.0 - YZMA-01: Local inference without Ollama
package yzma

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"aigateway/internal/models"

	"github.com/ebitengine/purego"
	"github.com/hybridgroup/yzma/pkg/llama"
	"github.com/sirupsen/logrus"
)

// stringFromPtr converts a C string pointer to Go string
func stringFromPtr(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	
	// Read bytes until null terminator
	var bytes []byte
	for i := uintptr(0); ; i++ {
		b := *(*byte)(unsafe.Pointer(ptr + i))
		if b == 0 {
			break
		}
		bytes = append(bytes, b)
	}
	
	return string(bytes)
}

// StorageInterface defines DB methods needed by yzma Client (v3.0.6+)
type StorageInterface interface {
	SaveLoadedModel(ctx context.Context, model *models.LoadedModel) error
	RemoveLoadedModel(ctx context.Context, modelPath string) error
	ListLoadedModels(ctx context.Context, autoLoadOnly bool) ([]*models.LoadedModel, error)
}

// Client wraps yzma for local inference
type Client struct {
	modelsDir          string
	logger             *logrus.Logger
	conversationLogger *logrus.Logger  // v3.0.6+: for user requests/responses
	libPath            string
	db                 StorageInterface  // v3.0.6+: for model persistence
	
	// Loaded models cache
	models      map[string]*ModelContext  // path -> ModelContext
	aliases     map[string]string         // alias -> path (v3.0.5+)
	modelsMu    sync.RWMutex
	
	// Default parameters
	contextSize  uint32
	batchSize    uint32
	uBatchSize   uint32
	temperature  float32
	topK         int32
	topP         float32
	minP         float32
	nGpuLayers   int32  // GPU offloading: -1 = all layers, 0 = CPU only (v3.0.6+)
	
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
	Alias    string    // v3.0.5+: Short name for OpenAI compatibility (e.g. "llama3.2-3b")
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
	NGpuLayers  int32   // Number of layers to offload to GPU (0 = CPU only, -1 = all layers, default: -1)
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
	// Default: offload all layers to GPU (-1 = auto-detect max layers)
	// User can set 0 for CPU-only mode
	if config.NGpuLayers == 0 {
		config.NGpuLayers = -1 // Auto-detect GPU and offload all layers
	}
	
	client := &Client{
		modelsDir:   config.ModelsDir,
		logger:      logger,
		libPath:     libPath,
		db:          nil,  // v3.0.6+: Set via SetDB later
		models:      make(map[string]*ModelContext),
		aliases:     make(map[string]string),  // v3.0.5+: alias -> path map
		contextSize: config.ContextSize,
		batchSize:   config.BatchSize,
		uBatchSize:  config.UBatchSize,
		temperature: config.Temperature,
		topK:        config.TopK,
		topP:        config.TopP,
		minP:        config.MinP,
		nGpuLayers:  config.NGpuLayers,  // v3.0.6+: GPU offloading
	}
	
	// Load llama.cpp library
	if err := llama.Load(libPath); err != nil {
		return nil, fmt.Errorf("failed to load llama.cpp library: %w", err)
	}
	
	// Configure llama.cpp logging
	if !config.Verbose {
		llama.LogSet(llama.LogSilent())
	} else {
		// Redirect llama.cpp logs to our logger
		// Callback signature: func(level int32, text, data uintptr) uintptr
		logCallback := purego.NewCallback(func(level int32, text, data uintptr) uintptr {
			// Convert C string to Go string
			msg := stringFromPtr(text)
			
			// llama.cpp log levels:
			// 0 = ERROR, 1 = WARN, 2 = INFO, 3+ = DEBUG
			switch {
			case level <= 0:
				logger.Error("[llama.cpp] " + msg)
			case level == 1:
				logger.Warn("[llama.cpp] " + msg)
			case level == 2:
				logger.Info("[llama.cpp] " + msg)
			default:
				logger.Debug("[llama.cpp] " + msg)
			}
			
			return 0
		})
		
		llama.LogSet(logCallback)
		logger.Info("✅ llama.cpp log callback configured")
	}
	
	// Initialize backend
	logger.WithFields(logrus.Fields{
		"lib_path":        libPath,
		"YZMA_LIB_env":    os.Getenv("YZMA_LIB"),
	}).Info("🔄 Calling llama.Init() to initialize backends...")
	
	// Ensure YZMA_LIB is set for llama.Init()
	if os.Getenv("YZMA_LIB") == "" {
		logger.WithField("lib_path", libPath).Warn("⚠️ YZMA_LIB not set, setting it now")
		os.Setenv("YZMA_LIB", libPath)
	}
	
	// Call llama.Init() which should call:
	// 1. BackendInit()
	// 2. GGMLBackendLoadAllFromPath(YZMA_LIB) if YZMA_LIB is set
	llama.Init()
	
	logger.Info("✅ llama.Init() completed")
	
	// Manually try to load backends from path as fallback
	logger.WithField("lib_path", libPath).Info("🔄 Manually calling GGMLBackendLoadAllFromPath...")
	llama.GGMLBackendLoadAllFromPath(libPath)
	logger.Info("✅ GGMLBackendLoadAllFromPath completed")
	
	client.initialized = true
	
	logger.WithFields(logrus.Fields{
		"models_dir":   config.ModelsDir,
		"lib_path":     libPath,
		"context_size": config.ContextSize,
		"batch_size":   config.BatchSize,
	}).Info("yzma client initialized successfully")
	
	return client, nil
}

// GetModelByAlias resolves alias to ModelContext (v3.0.5+)
// Returns ModelContext if found by alias or by path, nil otherwise
func (c *Client) GetModelByAlias(nameOrAlias string) *ModelContext {
	c.modelsMu.RLock()
	defer c.modelsMu.RUnlock()
	
	// Try alias first
	if path, ok := c.aliases[nameOrAlias]; ok {
		return c.models[path]
	}
	
	// Fallback to direct path lookup
	return c.models[nameOrAlias]
}

// ResolveModelPath resolves alias to full path (v3.0.5+)
// Returns the model path, or the input if no alias found
func (c *Client) ResolveModelPath(nameOrAlias string) string {
	c.modelsMu.RLock()
	
	// Try alias first
	if path, ok := c.aliases[nameOrAlias]; ok {
		c.modelsMu.RUnlock()
		return path
	}
	
	// If not found as alias, check if it's already a loaded model path
	if _, ok := c.models[nameOrAlias]; ok {
		c.modelsMu.RUnlock()
		return nameOrAlias
	}
	
	// Check if any loaded model has this alias
	var foundPath string
	for path, modelCtx := range c.models {
		if modelCtx.Alias == nameOrAlias {
			foundPath = path
			break
		}
	}
	
	c.modelsMu.RUnlock()
	
	// If found, re-register the alias for faster lookup next time
	if foundPath != "" {
		c.modelsMu.Lock()
		c.aliases[nameOrAlias] = foundPath
		c.modelsMu.Unlock()
		
		c.logger.WithFields(logrus.Fields{
			"alias": nameOrAlias,
			"path":  foundPath,
		}).Debug("🔧 Re-registered alias from loaded model")
		return foundPath
	}
	
	return nameOrAlias
}

// LoadModel loads a GGUF model from disk with optional alias (v3.0.5+)
// alias parameter is variadic for backward compatibility
func (c *Client) LoadModel(ctx context.Context, modelPath string, alias ...string) error {
	// Extract alias if provided
	modelAlias := ""
	if len(alias) > 0 {
		modelAlias = alias[0]
	}
	c.modelsMu.Lock()
	defer c.modelsMu.Unlock()
	
	c.logger.WithFields(map[string]interface{}{
		"model_path": modelPath,
		"alias":      modelAlias,
		"models_dir": c.modelsDir,
	}).Info("📥 LoadModel called")
	
	// Check if already loaded
	if _, exists := c.models[modelPath]; exists {
		c.logger.WithField("model_path", modelPath).Debug("Model already loaded")
		return nil
	}
	
	// Build full path
	var fullPath string
	if filepath.IsAbs(modelPath) {
		// If absolute path provided, use it directly
		fullPath = modelPath
	} else {
		// If relative path, join with models dir
		fullPath = filepath.Join(c.modelsDir, modelPath)
	}
	
	c.logger.WithFields(map[string]interface{}{
		"model_path": modelPath,
		"full_path":  fullPath,
		"models_dir": c.modelsDir,
	}).Info("🔍 Resolved full path for model")
	
	// Verify model file exists
	if _, err := os.Stat(fullPath); err != nil {
		c.logger.WithError(err).WithField("full_path", fullPath).Error("❌ Model file not found")
		return fmt.Errorf("model file not found: %s", fullPath)
	}
	
	c.logger.WithField("full_path", fullPath).Info("✅ Model file exists, loading...")
	
	// Get file info for debugging
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		c.logger.WithError(err).Error("❌ Failed to stat model file")
		return fmt.Errorf("failed to stat model file: %w", err)
	}
	
	c.logger.WithFields(logrus.Fields{
		"file_size": fileInfo.Size(),
		"file_mode": fileInfo.Mode(),
		"mod_time":  fileInfo.ModTime(),
	}).Info("📊 Model file details")
	
	startTime := time.Now()
	
	// Load model
	c.logger.Info("🔄 Calling llama.ModelLoadFromFile...")
	mParams := llama.ModelDefaultParams()
	
	// Enable GPU offloading
	mParams.NGpuLayers = c.nGpuLayers
	
	c.logger.WithFields(logrus.Fields{
		"n_gpu_layers": mParams.NGpuLayers,
		"split_mode":   mParams.SplitMode,
		"main_gpu":     mParams.MainGpu,
		"vocab_only":   mParams.VocabOnly,
	}).Info("🔧 Model parameters (GPU offloading enabled)")
	
	model := llama.ModelLoadFromFile(fullPath, mParams)
	if model == 0 {
		c.logger.WithField("full_path", fullPath).Error("❌ llama.ModelLoadFromFile returned 0 (failed)")
		return fmt.Errorf("failed to load model from file: %s (llama.cpp returned null pointer)", fullPath)
	}
	
	c.logger.WithField("model_handle", model).Info("✅ llama.ModelLoadFromFile succeeded")
	
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
	
	// Generate alias if not provided (v3.0.5+)
	if modelAlias == "" {
		// Auto-generate alias from filename: "Llama-3.2-3B-Instruct-Q8_0.gguf" -> "llama-3.2-3b-instruct-q8_0"
		modelAlias = strings.ToLower(strings.TrimSuffix(filepath.Base(fullPath), ".gguf"))
	}
	
	// Cache model context
	c.models[modelPath] = &ModelContext{
		Model:    model,
		Vocab:    vocab,
		Ctx:      lctx,
		Sampler:  sampler,
		Path:     fullPath,
		Alias:    modelAlias,
		LoadedAt: time.Now(),
	}
	
	// Register alias (v3.0.5+)
	c.aliases[modelAlias] = modelPath
	
	c.logger.WithFields(logrus.Fields{
		"alias": modelAlias,
		"path":  modelPath,
	}).Info("✅ Model alias registered")
	
	c.logger.WithFields(logrus.Fields{
		"model_path": modelPath,
		"load_time":  time.Since(startTime),
	}).Info("Model loaded successfully")
	
	// Save to DB for persistence (v3.0.6+)
	if c.db != nil {
		dbModel := &models.LoadedModel{
			ModelPath:  modelPath,
			Alias:      modelAlias,
			LoadedAt:   time.Now(),
			AutoLoad:   true,
			CreatedAt:  time.Now(),
		}
		
		if err := c.db.SaveLoadedModel(ctx, dbModel); err != nil {
			c.logger.WithError(err).Warn("Failed to persist loaded model to DB")
			// Don't fail the load, just log the warning
		} else {
			c.logger.WithField("alias", modelAlias).Debug("Model persisted to DB")
		}
	}
	
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
	
	c.logger.WithFields(logrus.Fields{
		"model_path": modelPath,
		"loaded_at":  modelCtx.LoadedAt,
	}).Info("🗑️ Starting model unload...")
	
	// Free resources in correct order
	c.logger.Debug("Freeing sampler...")
	llama.SamplerFree(modelCtx.Sampler)
	
	c.logger.Debug("Freeing context...")
	llama.Free(modelCtx.Ctx)
	
	c.logger.Debug("Freeing model...")
	llama.ModelFree(modelCtx.Model)
	
	// Remove from cache and aliases (v3.0.5+)
	if modelCtx.Alias != "" {
		delete(c.aliases, modelCtx.Alias)
		c.logger.WithField("alias", modelCtx.Alias).Debug("Alias removed")
	}
	delete(c.models, modelPath)
	
	c.logger.WithField("model_path", modelPath).Info("✅ Model resources freed from llama.cpp")
	
	// Force garbage collection to help release GPU memory
	c.logger.Debug("🔄 Running garbage collection...")
	runtime.GC()
	c.logger.Debug("✅ Garbage collection completed")
	
	c.logger.WithField("model_path", modelPath).Info("✅ Model unloaded successfully")
	
	// Remove from DB persistence (v3.0.6+)
	if c.db != nil {
		ctx := context.Background()
		if err := c.db.RemoveLoadedModel(ctx, modelPath); err != nil {
			c.logger.WithError(err).Warn("Failed to remove model from DB persistence")
			// Don't fail the unload, just log the warning
		} else {
			c.logger.Debug("Model removed from DB persistence")
		}
	}
	
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

// ListLoadedModels returns all loaded models with their aliases
func (c *Client) ListLoadedModels() map[string]string {
	c.modelsMu.RLock()
	defer c.modelsMu.RUnlock()
	
	models := make(map[string]string, len(c.models))
	for path, ctx := range c.models {
		models[path] = ctx.Alias
	}
	
	return models
}

// ListAvailableModels scans models directory for GGUF files (including .part files)
func (c *Client) ListAvailableModels() ([]string, error) {
	var models []string
	
	err := filepath.Walk(c.modelsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			ext := filepath.Ext(path)
			// Include both .gguf and .part files (incomplete downloads)
			if ext == ".gguf" || ext == ".part" {
				// Get relative path
				relPath, err := filepath.Rel(c.modelsDir, path)
				if err != nil {
					return err
				}
				models = append(models, relPath)
			}
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to scan models directory: %w", err)
	}
	
	return models, nil
}

// GetFullModelPath resolves relative model path to absolute path
// Uses the same logic as LoadModel to ensure consistency
func (c *Client) GetFullModelPath(modelPath string) string {
	if filepath.IsAbs(modelPath) {
		return modelPath
	}
	return filepath.Join(c.modelsDir, modelPath)
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

// EmbeddingRequest represents an embedding generation request (v3.0.5+)
type EmbeddingRequest struct {
	ModelPath string   // Path to GGUF model (relative to models dir)
	Input     []string // Texts to generate embeddings for
}

// EmbeddingResponse represents an embedding generation response (v3.0.5+)
type EmbeddingResponse struct {
	Embeddings [][]float32 // One embedding vector per input text
	Model      string      // Model path used
	Dimensions int         // Embedding vector dimension
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
	
	// Log user request (v3.0.6+)
	if c.conversationLogger != nil {
		// Prepare user request summary
		userRequest := map[string]interface{}{
			"timestamp":   startTime.Format(time.RFC3339),
			"model":       modelCtx.Alias,
			"model_path":  req.ModelPath,
			"max_tokens":  req.MaxTokens,
			"temperature": req.Temperature,
		}
		
		if len(req.Messages) > 0 {
			// Chat format
			userRequest["type"] = "chat"
			userRequest["messages"] = req.Messages
			userRequest["system_prompt"] = req.SystemPrompt
		} else {
			// Text completion format
			userRequest["type"] = "completion"
			userRequest["prompt"] = req.Prompt
		}
		
		c.conversationLogger.WithFields(logrus.Fields{
			"event":       "user_request",
			"model":       modelCtx.Alias,
			"prompt_len":  len(promptText),
			"type":        userRequest["type"],
		}).WithField("request", userRequest).Info("📥 User request")
	}
	
	// Tokenize prompt
	tokens := llama.Tokenize(modelCtx.Vocab, promptText, true, true)
	promptTokens := len(tokens)
	
	batch := llama.BatchGetOne(tokens)
	
	// Generate response
	var response strings.Builder
	tokensGenerated := 0
	finishReason := "stop"
	
	// Process prompt batch first
	if ret := llama.Decode(modelCtx.Ctx, batch); ret != 0 {
		return nil, fmt.Errorf("failed to decode prompt batch: ret=%d", ret)
	}
	
	// Generate tokens one by one
	for tokensGenerated < req.MaxTokens {
		// Check context cancellation
		select {
		case <-ctx.Done():
			finishReason = "cancelled"
			goto done
		default:
		}
		
		// Sample next token
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
		
		// Prepare next batch with single token
		batch = llama.BatchGetOne([]llama.Token{token})
		if ret := llama.Decode(modelCtx.Ctx, batch); ret != 0 {
			c.logger.WithField("ret", ret).Warn("Failed to decode token, stopping generation")
			finishReason = "error"
			break
		}
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
	
	// Log model response (v3.0.6+)
	if c.conversationLogger != nil {
		responseData := map[string]interface{}{
			"timestamp":      time.Now().Format(time.RFC3339),
			"model":          modelCtx.Alias,
			"content":        response.String(),
			"tokens_used":    tokensGenerated,
			"prompt_tokens":  promptTokens,
			"total_tokens":   promptTokens + tokensGenerated,
			"duration_ms":    duration.Milliseconds(),
			"tokens_per_sec": tokensPerSec,
			"finish_reason":  finishReason,
		}
		
		c.conversationLogger.WithFields(logrus.Fields{
			"event":          "model_response",
			"model":          modelCtx.Alias,
			"tokens":         tokensGenerated,
			"duration_ms":    duration.Milliseconds(),
			"finish_reason":  finishReason,
		}).WithField("response", responseData).Info("📤 Model response")
	}
	
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
	
	// Log user request (v3.0.6+)
	if c.conversationLogger != nil {
		// Prepare user request summary
		userRequest := map[string]interface{}{
			"timestamp":   startTime.Format(time.RFC3339),
			"model":       modelCtx.Alias,
			"model_path":  req.ModelPath,
			"max_tokens":  req.MaxTokens,
			"temperature": req.Temperature,
			"stream":      true,
		}
		
		if len(req.Messages) > 0 {
			// Chat format
			userRequest["type"] = "chat"
			userRequest["messages"] = req.Messages
			userRequest["system_prompt"] = req.SystemPrompt
		} else {
			// Text completion format
			userRequest["type"] = "completion"
			userRequest["prompt"] = req.Prompt
		}
		
		c.conversationLogger.WithFields(logrus.Fields{
			"event":       "user_request",
			"model":       modelCtx.Alias,
			"prompt_len":  len(promptText),
			"type":        userRequest["type"],
			"stream":      true,
		}).WithField("request", userRequest).Info("📥 User request (streaming)")
	}
	
	// Tokenize prompt
	tokens := llama.Tokenize(modelCtx.Vocab, promptText, true, true)
	promptTokens := len(tokens)
	
	batch := llama.BatchGetOne(tokens)
	
	// Stream response
	var fullResponse strings.Builder
	tokensGenerated := 0
	finishReason := "stop"
	
	// Process prompt batch first
	if ret := llama.Decode(modelCtx.Ctx, batch); ret != 0 {
		return nil, fmt.Errorf("failed to decode prompt batch: ret=%d", ret)
	}
	
	// Generate tokens one by one
	for tokensGenerated < req.MaxTokens {
		// Check context cancellation
		select {
		case <-ctx.Done():
			finishReason = "cancelled"
			goto done
		default:
		}
		
		// Sample next token
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
		
		// Prepare next batch with single token
		batch = llama.BatchGetOne([]llama.Token{token})
		if ret := llama.Decode(modelCtx.Ctx, batch); ret != 0 {
			c.logger.WithField("ret", ret).Warn("Failed to decode token in stream, stopping generation")
			finishReason = "error"
			break
		}
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
	
	// Log model response (v3.0.6+)
	if c.conversationLogger != nil {
		responseData := map[string]interface{}{
			"timestamp":      time.Now().Format(time.RFC3339),
			"model":          modelCtx.Alias,
			"content":        fullResponse.String(),
			"tokens_used":    tokensGenerated,
			"prompt_tokens":  promptTokens,
			"total_tokens":   promptTokens + tokensGenerated,
			"duration_ms":    duration.Milliseconds(),
			"tokens_per_sec": tokensPerSec,
			"finish_reason":  finishReason,
			"stream":         true,
		}
		
		c.conversationLogger.WithFields(logrus.Fields{
			"event":          "model_response",
			"model":          modelCtx.Alias,
			"tokens":         tokensGenerated,
			"duration_ms":    duration.Milliseconds(),
			"finish_reason":  finishReason,
			"stream":         true,
		}).WithField("response", responseData).Info("📤 Model response (streaming)")
	}
	
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

// GenerateEmbedding generates embedding vectors for input texts (v3.0.5+)
func (c *Client) GenerateEmbedding(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	startTime := time.Now()
	
	// Resolve model path (alias → path)
	modelPath := c.ResolveModelPath(req.ModelPath)
	
	// Load model if not already loaded
	if !c.IsModelLoaded(modelPath) {
		if err := c.LoadModel(ctx, modelPath); err != nil {
			return nil, fmt.Errorf("failed to load model: %w", err)
		}
	}
	
	// Get model context
	c.modelsMu.RLock()
	modelCtx, exists := c.models[modelPath]
	c.modelsMu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("model not loaded: %s", modelPath)
	}
	
	// Get embedding dimension from model
	nEmbd := int(llama.ModelNEmbd(modelCtx.Model))
	c.logger.WithFields(logrus.Fields{
		"model":      modelPath,
		"n_embd":     nEmbd,
		"num_inputs": len(req.Input),
	}).Debug("Generating embeddings")
	
	var embeddings [][]float32
	
	// Enable embeddings mode for the entire session
	llama.SetEmbeddings(modelCtx.Ctx, true)
	llama.SetCausalAttn(modelCtx.Ctx, false)
	
	// Process each input text
	for i, text := range req.Input {
		// Tokenize input
		tokens := llama.Tokenize(modelCtx.Vocab, text, true, false)
		if len(tokens) == 0 {
			return nil, fmt.Errorf("failed to tokenize input %d", i)
		}
		
		c.logger.WithFields(logrus.Fields{
			"input_idx": i,
			"tokens":    len(tokens),
			"text_len":  len(text),
		}).Debug("Tokenized input for embedding")
		
		// Create batch using BatchGetOne (simpler API)
		batch := llama.BatchGetOne(tokens)
		
		// Decode the batch to generate embeddings
		if llama.Decode(modelCtx.Ctx, batch) != 0 {
			return nil, fmt.Errorf("failed to decode batch for input %d", i)
		}
		
		// Get embeddings for the last token (mean pooling)
		embeddingVec := llama.GetEmbeddingsSeq(modelCtx.Ctx, 0, int32(nEmbd))
		if embeddingVec == nil {
			return nil, fmt.Errorf("failed to get embeddings for input %d", i)
		}
		
		c.logger.WithFields(logrus.Fields{
			"input_idx":   i,
			"emb_dims":    len(embeddingVec),
			"first_value": embeddingVec[0],
		}).Debug("Generated embedding vector")
		
		// Copy embedding vector
		embedding := make([]float32, len(embeddingVec))
		copy(embedding, embeddingVec)
		embeddings = append(embeddings, embedding)
	}
	
	// Restore normal mode
	llama.SetEmbeddings(modelCtx.Ctx, false)
	llama.SetCausalAttn(modelCtx.Ctx, true)
	
	c.logger.WithFields(logrus.Fields{
		"model":      modelPath,
		"embeddings": len(embeddings),
		"duration":   time.Since(startTime),
	}).Info("Embeddings generated successfully")
	
	return &EmbeddingResponse{
		Embeddings: embeddings,
		Model:      modelPath,
		Dimensions: nEmbd,
	}, nil
}

// Shutdown gracefully shuts down the client and frees all GPU/CPU memory
func (c *Client) Shutdown() error {
	c.logger.Info("🛑 Shutting down yzma client")
	
	c.modelsMu.Lock()
	defer c.modelsMu.Unlock()
	
	// Unload all models
	modelCount := len(c.models)
	if modelCount > 0 {
		c.logger.WithField("count", modelCount).Info("🗑️ Freeing loaded models...")
		
		for path, modelCtx := range c.models {
			c.logger.WithField("model_path", path).Debug("Freeing model resources...")
			
			llama.SamplerFree(modelCtx.Sampler)
			llama.Free(modelCtx.Ctx)
			llama.ModelFree(modelCtx.Model)
			
			c.logger.WithField("model_path", path).Info("✅ Model freed")
		}
		
		c.models = make(map[string]*ModelContext)
	}
	
	// Free backend (releases GPU/CPU memory)
	if c.initialized {
		c.logger.Info("🔄 Calling llama.BackendFree() to release all memory...")
		llama.BackendFree()
		c.initialized = false
		c.logger.Info("✅ Backend freed")
	}
	
	// Force garbage collection
	c.logger.Debug("🔄 Running final garbage collection...")
	runtime.GC()
	runtime.GC() // Double GC for good measure
	c.logger.Debug("✅ Garbage collection completed")
	
	c.logger.Info("✅ yzma client shut down successfully - all memory should be released")
	return nil
}

// SetDB sets the database for model persistence (v3.0.6+)
func (c *Client) SetDB(db StorageInterface) {
	c.db = db
	c.logger.Info("✅ Database set for model persistence")
}

// SetConversationLogger sets the logger for user requests/responses (v3.0.6+)
func (c *Client) SetConversationLogger(logger *logrus.Logger) {
	c.conversationLogger = logger
	c.logger.Info("✅ Conversation logger set for request/response logging")
}

// LoadPersistedModels loads all models marked for auto-load from DB (v3.0.6+)
func (c *Client) LoadPersistedModels(ctx context.Context) error {
	if c.db == nil {
		c.logger.Debug("No database configured, skipping persisted models load")
		return nil
	}
	
	persistedModels, err := c.db.ListLoadedModels(ctx, true)  // autoLoadOnly=true
	if err != nil {
		return fmt.Errorf("failed to list persisted models: %w", err)
	}
	
	if len(persistedModels) == 0 {
		c.logger.Info("No persisted models to auto-load")
		return nil
	}
	
	c.logger.WithField("count", len(persistedModels)).Info("🔄 Auto-loading persisted models...")
	
	var successCount, failCount int
	for _, model := range persistedModels {
		c.logger.WithFields(logrus.Fields{
			"model_path": model.ModelPath,
			"alias":      model.Alias,
		}).Info("Loading persisted model...")
		
		// Load with existing alias
		if err := c.LoadModel(ctx, model.ModelPath, model.Alias); err != nil {
			c.logger.WithError(err).WithField("model_path", model.ModelPath).Error("Failed to load persisted model")
			failCount++
			continue
		}
		
		successCount++
	}
	
	c.logger.WithFields(logrus.Fields{
		"total":   len(persistedModels),
		"success": successCount,
		"failed":  failCount,
	}).Info("✅ Persisted models auto-load completed")
	
	return nil
}

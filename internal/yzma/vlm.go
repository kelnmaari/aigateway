// Package yzma provides local LLM inference using yzma library
// Version: v3.0.4 - VLM-01: Vision Language Model support
package yzma

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/hybridgroup/yzma/pkg/llama"
	"github.com/hybridgroup/yzma/pkg/mtmd"
	"golang.org/x/image/webp"
)

// VLMGenerateRequest represents a VLM generation request with images
type VLMGenerateRequest struct {
	ModelPath     string
	MMProjPath    string   // Path to mmproj GGUF file
	Prompt        string
	Images        []string // Base64 encoded images or file paths
	SystemPrompt  string
	Temperature   float32
	TopK          int32
	TopP          float32
	MinP          float32
	MaxTokens     int
	StopSequences []string
}

// VLMGenerateResponse represents VLM generation response
type VLMGenerateResponse struct {
	Text         string
	TokensCount  int
	PromptTokens int
}

// LoadVLMModel loads a VLM model with mmproj projector
func (c *Client) LoadVLMModel(ctx context.Context, modelPath string, mmprojPath string) error {
	c.logger.WithFields(map[string]interface{}{
		"model_path":  modelPath,
		"mmproj_path": mmprojPath,
	}).Info("Loading VLM model")
	
	// Load base text model first
	if err := c.LoadModel(ctx, modelPath); err != nil {
		return fmt.Errorf("failed to load text model: %w", err)
	}
	
	c.modelsMu.Lock()
	defer c.modelsMu.Unlock()
	
	modelCtx, exists := c.models[modelPath]
	if !exists {
		return fmt.Errorf("model not found after loading: %s", modelPath)
	}
	
	// Load mtmd library
	if err := mtmd.Load(c.libPath); err != nil {
		return fmt.Errorf("failed to load mtmd library: %w", err)
	}
	
	// Initialize mtmd context with projector
	mctxParams := mtmd.ContextParamsDefault()
	mctxParams.Verbosity = llama.LogLevelContinue
	if c.logger.Level.String() != "debug" {
		mctxParams.Verbosity = llama.LogLevelError
	}
	
	mtmdCtx := mtmd.InitFromFile(mmprojPath, modelCtx.Model, mctxParams)
	if mtmdCtx == 0 {
		return fmt.Errorf("failed to initialize mtmd context from %s", mmprojPath)
	}
	
	// Check if vision is supported
	if !mtmd.SupportVision(mtmdCtx) {
		mtmd.Free(mtmdCtx)
		return fmt.Errorf("model does not support vision: %s", mmprojPath)
	}
	
	// Update model context
	modelCtx.IsVLM = true
	modelCtx.MtmdCtx = uint64(mtmdCtx)
	modelCtx.MMProjPath = mmprojPath
	
	c.logger.WithField("model_path", modelPath).Info("VLM model loaded successfully")
	return nil
}

// UnloadVLMModel unloads a VLM model and frees mtmd context
func (c *Client) UnloadVLMModel(modelPath string) error {
	c.modelsMu.Lock()
	defer c.modelsMu.Unlock()
	
	modelCtx, exists := c.models[modelPath]
	if !exists {
		return fmt.Errorf("model not loaded: %s", modelPath)
	}
	
	if modelCtx.IsVLM && modelCtx.MtmdCtx != 0 {
		mtmd.Free(mtmd.Context(modelCtx.MtmdCtx))
		modelCtx.MtmdCtx = 0
		modelCtx.IsVLM = false
	}
	
	// Unload base model
	return c.UnloadModel(modelPath)
}

// GenerateWithImages generates text from images + text prompt
func (c *Client) GenerateWithImages(ctx context.Context, req VLMGenerateRequest) (*VLMGenerateResponse, error) {
	c.modelsMu.RLock()
	modelCtx, exists := c.models[req.ModelPath]
	c.modelsMu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("model not loaded: %s", req.ModelPath)
	}
	
	if !modelCtx.IsVLM {
		return nil, fmt.Errorf("model is not a VLM: %s", req.ModelPath)
	}
	
	// Process images
	bitmaps, err := c.processImages(ctx, mtmd.Context(modelCtx.MtmdCtx), req.Images)
	if err != nil {
		return nil, fmt.Errorf("failed to process images: %w", err)
	}
	defer func() {
		for _, bitmap := range bitmaps {
			mtmd.BitmapFree(bitmap)
		}
	}()
	
	// Build chat messages
	messages := make([]llama.ChatMessage, 0)
	if req.SystemPrompt != "" {
		messages = append(messages, llama.NewChatMessage("system", req.SystemPrompt))
	}
	
	// Add user message with image marker
	userPrompt := req.Prompt + mtmd.DefaultMarker()
	messages = append(messages, llama.NewChatMessage("user", userPrompt))
	
	// Get chat template
	template := llama.ModelChatTemplate(modelCtx.Model, "")
	chatBuf := make([]byte, 4096)
	chatLen := llama.ChatApplyTemplate(template, messages, true, chatBuf)
	chatText := string(chatBuf[:chatLen])
	
	// Tokenize image + text
	output := mtmd.InputChunksInit()
	input := mtmd.NewInputText(chatText, true, true)
	
	tokensCount := mtmd.Tokenize(mtmd.Context(modelCtx.MtmdCtx), output, input, bitmaps)
	if tokensCount < 0 {
		return nil, fmt.Errorf("tokenization failed")
	}
	
	// Evaluate chunks
	var n llama.Pos
	ctxParams := llama.ContextDefaultParams()
	ctxParams.NCtx = c.contextSize
	ctxParams.NBatch = c.batchSize
	
	mtmd.HelperEvalChunks(mtmd.Context(modelCtx.MtmdCtx), modelCtx.Ctx, output, 0, 0, int32(ctxParams.NBatch), true, &n)
	
	// Generate response
	response := strings.Builder{}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 2048
	}
	
	// Create batch for token-by-token generation
	var sz int32 = 1
	batch := llama.BatchInit(1, 0, 1)
	batch.NSeqId = &sz
	batch.NTokens = 1
	seqs := unsafe.SliceData([]llama.SeqId{0})
	batch.SeqId = &seqs
	
	generatedTokens := 0
	for i := 0; i < maxTokens; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		// Sample next token
		token := llama.SamplerSample(modelCtx.Sampler, modelCtx.Ctx, -1)
		
		// Check for EOS/EOG
		if llama.VocabIsEOG(modelCtx.Vocab, token) {
			break
		}
		
		// Convert token to text
		buf := make([]byte, 128)
		l := llama.TokenToPiece(modelCtx.Vocab, token, buf, 0, true)
		tokenText := string(buf[:l])
		
		response.WriteString(tokenText)
		generatedTokens++
		
		// Check stop sequences
		if len(req.StopSequences) > 0 && c.containsStopSequence(response.String(), req.StopSequences) {
			break
		}
		
		// Decode next position
		batch.Token = &token
		batch.Pos = &n
		llama.Decode(modelCtx.Ctx, batch)
		n++
	}
	
	// Update stats
	c.statsMu.Lock()
	c.totalRequests++
	c.totalTokens += int64(generatedTokens)
	c.statsMu.Unlock()
	
	return &VLMGenerateResponse{
		Text:         response.String(),
		TokensCount:  generatedTokens,
		PromptTokens: int(tokensCount),
	}, nil
}

// processImages processes images (base64 or file paths) into mtmd.Bitmap
func (c *Client) processImages(ctx context.Context, mtmdCtx mtmd.Context, images []string) ([]mtmd.Bitmap, error) {
	if len(images) == 0 {
		return nil, fmt.Errorf("no images provided")
	}
	
	bitmaps := make([]mtmd.Bitmap, 0, len(images))
	
	for i, imageData := range images {
		select {
		case <-ctx.Done():
			// Free already allocated bitmaps
			for _, bitmap := range bitmaps {
				mtmd.BitmapFree(bitmap)
			}
			return nil, ctx.Err()
		default:
		}
		
		var img image.Image
		var err error
		
		// Check if it's a base64 encoded image
		if strings.HasPrefix(imageData, "data:image/") {
			img, err = c.decodeBase64Image(imageData)
		} else if strings.HasPrefix(imageData, "file://") {
			// File path
			filePath := strings.TrimPrefix(imageData, "file://")
			img, err = c.loadImageFromFile(filePath)
		} else {
			// Assume file path
			img, err = c.loadImageFromFile(imageData)
		}
		
		if err != nil {
			// Free already allocated bitmaps
			for _, bitmap := range bitmaps {
				mtmd.BitmapFree(bitmap)
			}
			return nil, fmt.Errorf("failed to process image %d: %w", i, err)
		}
		
		// Convert to mtmd.Bitmap
		bitmap, err := c.imageToBitmap(mtmdCtx, img)
		if err != nil {
			// Free already allocated bitmaps
			for _, bitmap := range bitmaps {
				mtmd.BitmapFree(bitmap)
			}
			return nil, fmt.Errorf("failed to convert image %d to bitmap: %w", i, err)
		}
		
		bitmaps = append(bitmaps, bitmap)
	}
	
	return bitmaps, nil
}

// decodeBase64Image decodes a base64 data URI image
func (c *Client) decodeBase64Image(dataURI string) (image.Image, error) {
	// Extract base64 data
	// Format: data:image/png;base64,iVBORw0KGgoAAAANS...
	parts := strings.SplitN(dataURI, ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid data URI format")
	}
	
	base64Data := parts[1]
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	
	// Decode image
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	
	c.logger.WithField("format", format).Debug("Decoded base64 image")
	return img, nil
}

// loadImageFromFile loads an image from a file path
func (c *Client) loadImageFromFile(filePath string) (image.Image, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()
	
	// Read first bytes to detect format
	header := make([]byte, 512)
	_, err = file.Read(header)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read image header: %w", err)
	}
	
	// Seek back to start
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}
	
	// Detect format
	ext := strings.ToLower(filepath.Ext(filePath))
	var img image.Image
	
	switch ext {
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(file)
	case ".png":
		img, err = png.Decode(file)
	case ".webp":
		img, err = webp.Decode(file)
	default:
		// Try generic decoder
		img, _, err = image.Decode(file)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	
	c.logger.WithField("path", filePath).Debug("Loaded image from file")
	return img, nil
}

// imageToBitmap converts Go image.Image to mtmd.Bitmap
func (c *Client) imageToBitmap(mtmdCtx mtmd.Context, img image.Image) (mtmd.Bitmap, error) {
	// Create temporary file for image
	tmpFile, err := os.CreateTemp("", "yzma-vlm-*.png")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	
	// Encode image as PNG
	if err := png.Encode(tmpFile, img); err != nil {
		return 0, fmt.Errorf("failed to encode image: %w", err)
	}
	tmpFile.Close()
	
	// Load bitmap from file
	bitmap := mtmd.BitmapInitFromFile(mtmdCtx, tmpFile.Name())
	if bitmap == 0 {
		return 0, fmt.Errorf("failed to initialize bitmap")
	}
	
	return bitmap, nil
}

// IsVLMModel checks if a model is a VLM
func (c *Client) IsVLMModel(modelPath string) bool {
	c.modelsMu.RLock()
	defer c.modelsMu.RUnlock()
	
	modelCtx, exists := c.models[modelPath]
	if !exists {
		return false
	}
	
	return modelCtx.IsVLM
}

// GetVLMModels returns list of loaded VLM models
func (c *Client) GetVLMModels() []string {
	c.modelsMu.RLock()
	defer c.modelsMu.RUnlock()
	
	vlmModels := make([]string, 0)
	for path, modelCtx := range c.models {
		if modelCtx.IsVLM {
			vlmModels = append(vlmModels, path)
		}
	}
	
	return vlmModels
}


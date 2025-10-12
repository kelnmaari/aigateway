# API-09: Image Generation Support

**Версия:** v1.7.2  
**Приоритет:** LOW  
**Оценка времени:** 6-8 часов  
**Статус:** 📋 Не начато  
**Зависит от:** Ollama image model support

---

## 📋 Описание

Поддержка OpenAI `/v1/images/generations` API для генерации изображений. Реализация зависит от того, добавит ли Ollama поддержку image generation моделей (Stable Diffusion, DALL-E alternatives, etc.).

## 🎯 Цели

1. **OpenAI Images API Compatibility** - `/v1/images/generations` endpoint
2. **Image Format Support** - PNG, JPEG, WebP, base64
3. **Model Detection** - автоматическое определение image models
4. **Fallback Strategy** - graceful degradation если модели нет

## 🔧 Технические детали

### 1. Image Generation Models

**Файл:** `internal/models/images.go`

```go
package models

// ImageGenerationRequest представляет запрос на генерацию изображения
type ImageGenerationRequest struct {
	Prompt         string  `json:"prompt" binding:"required"`           // Text description
	Model          *string `json:"model,omitempty"`                     // Model to use (optional)
	N              *int    `json:"n,omitempty"`                         // Number of images (1-10)
	Quality        *string `json:"quality,omitempty"`                   // "standard" or "hd"
	ResponseFormat *string `json:"response_format,omitempty"`           // "url" or "b64_json"
	Size           *string `json:"size,omitempty"`                      // "256x256", "512x512", "1024x1024", "1792x1024", "1024x1792"
	Style          *string `json:"style,omitempty"`                     // "vivid" or "natural"
	User           *string `json:"user,omitempty"`                      // End-user identifier
}

// ImageGenerationResponse представляет ответ с сгенерированными изображениями
type ImageGenerationResponse struct {
	Created int64        `json:"created"`
	Data    []ImageData  `json:"data"`
}

type ImageData struct {
	B64JSON       *string `json:"b64_json,omitempty"`        // Base64-encoded PNG
	URL           *string `json:"url,omitempty"`             // Temporary URL
	RevisedPrompt *string `json:"revised_prompt,omitempty"` // Improved prompt (optional)
}

// ImageEditRequest для редактирования изображений (future)
type ImageEditRequest struct {
	Image          string  `json:"image" binding:"required"` // File upload
	Prompt         string  `json:"prompt" binding:"required"`
	Mask           *string `json:"mask,omitempty"`  // Optional mask
	Model          *string `json:"model,omitempty"`
	N              *int    `json:"n,omitempty"`
	Size           *string `json:"size,omitempty"`
	ResponseFormat *string `json:"response_format,omitempty"`
	User           *string `json:"user,omitempty"`
}

// ImageVariationRequest для создания вариаций (future)
type ImageVariationRequest struct {
	Image          string  `json:"image" binding:"required"`
	Model          *string `json:"model,omitempty"`
	N              *int    `json:"n,omitempty"`
	ResponseFormat *string `json:"response_format,omitempty"`
	Size           *string `json:"size,omitempty"`
	User           *string `json:"user,omitempty"`
}
```

### 2. Ollama Image Generation Integration

**Файл:** `internal/client/ollama/images.go`

```go
package ollama

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"ollama-openai-proxy/internal/models"
)

// GenerateImage отправляет запрос в Ollama для генерации изображения
func (c *Client) GenerateImage(ctx context.Context, req models.ImageGenerationRequest) (*models.ImageGenerationResponse, error) {
	// Check if Ollama supports image generation
	if !c.supportsImageGeneration() {
		return nil, fmt.Errorf("Ollama does not support image generation")
	}

	// Determine model
	model := "stable-diffusion" // Default (если Ollama добавит)
	if req.Model != nil {
		model = *req.Model
	}

	// Convert request to Ollama format
	ollamaReq := OllamaGenerateRequest{
		Model:  model,
		Prompt: req.Prompt,
		Options: map[string]interface{}{
			"num_predict": 0, // Image generation mode
		},
	}

	// Add size if specified
	if req.Size != nil {
		ollamaReq.Options["size"] = *req.Size
	}

	// Send request
	resp, err := c.makeRequest(ctx, "/api/generate", ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("Ollama request failed: %w", err)
	}

	// Parse response (format зависит от Ollama implementation)
	// Предположим, Ollama возвращает base64 изображение
	imageData := models.ImageData{}
	
	responseFormat := "url"
	if req.ResponseFormat != nil {
		responseFormat = *req.ResponseFormat
	}

	if responseFormat == "b64_json" {
		// Return base64-encoded image
		imageData.B64JSON = &resp.Image // Hypothetical field
	} else {
		// Save image and return URL
		url, err := c.saveImageAndGetURL(ctx, resp.Image)
		if err != nil {
			return nil, err
		}
		imageData.URL = &url
	}

	return &models.ImageGenerationResponse{
		Created: time.Now().Unix(),
		Data:    []models.ImageData{imageData},
	}, nil
}

// supportsImageGeneration проверяет, поддерживает ли Ollama генерацию изображений
func (c *Client) supportsImageGeneration() bool {
	// Check Ollama version or available models
	models, err := c.ListModels(context.Background())
	if err != nil {
		return false
	}

	// Look for image generation models
	for _, model := range models {
		if isImageModel(model.Name) {
			return true
		}
	}

	return false
}

func isImageModel(name string) bool {
	imageModels := []string{
		"stable-diffusion",
		"sdxl",
		"dall-e",
		"midjourney",
	}

	for _, im := range imageModels {
		if strings.Contains(strings.ToLower(name), im) {
			return true
		}
	}

	return false
}

func (c *Client) saveImageAndGetURL(ctx context.Context, imageBase64 string) (string, error) {
	// Decode base64
	imageData, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		return "", err
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s.png", uuid.New().String())
	filepath := path.Join(c.config.ImagesDir, filename)

	// Save to disk
	if err := os.WriteFile(filepath, imageData, 0644); err != nil {
		return "", err
	}

	// Return public URL
	return fmt.Sprintf("%s/images/%s", c.config.BaseURL, filename), nil
}
```

### 3. Images Handler

**Файл:** `internal/api/handlers/images.go`

```go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/client/ollama"
	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

type ImagesHandler struct {
	ollamaClient *ollama.Client
	db           storage.Database
	logger       *logrus.Logger
	config       ImagesConfig
}

type ImagesConfig struct {
	Enabled           bool
	DefaultModel      string
	SupportedSizes    []string
	MaxImagesPerRequest int
	StorageDir        string
	BaseURL           string
}

func NewImagesHandler(
	ollamaClient *ollama.Client,
	db storage.Database,
	logger *logrus.Logger,
	config ImagesConfig,
) *ImagesHandler {
	return &ImagesHandler{
		ollamaClient: ollamaClient,
		db:           db,
		logger:       logger,
		config:       config,
	}
}

// GenerateImages обрабатывает запросы на генерацию изображений
// POST /v1/images/generations
func (h *ImagesHandler) GenerateImages(c *gin.Context) {
	if !h.config.Enabled {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": gin.H{
				"message": "Image generation is not available. Ollama does not support image models yet.",
				"type":    "not_implemented",
				"code":    "image_generation_not_supported",
			},
		})
		return
	}

	var req models.ImageGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "invalid_request_error",
			},
		})
		return
	}

	// Validate request
	if err := h.validateRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "invalid_request_error",
			},
		})
		return
	}

	// Set defaults
	h.setDefaults(&req)

	// Check if Ollama supports image generation
	if !h.ollamaClient.SupportsImageGeneration() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "No image generation models available in Ollama",
				"type":    "service_unavailable",
				"code":    "no_image_models",
			},
		})
		return
	}

	// Generate images
	resp, err := h.ollamaClient.GenerateImage(c.Request.Context(), req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate image")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Image generation failed",
				"type":    "internal_error",
			},
		})
		return
	}

	// Record usage
	userID, _ := c.Get("user_id")
	h.recordImageUsage(c.Request.Context(), userID.(string), req, resp)

	c.JSON(http.StatusOK, resp)
}

func (h *ImagesHandler) validateRequest(req *models.ImageGenerationRequest) error {
	// Validate N
	if req.N != nil && (*req.N < 1 || *req.N > h.config.MaxImagesPerRequest) {
		return fmt.Errorf("n must be between 1 and %d", h.config.MaxImagesPerRequest)
	}

	// Validate size
	if req.Size != nil {
		valid := false
		for _, size := range h.config.SupportedSizes {
			if *req.Size == size {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("unsupported size: %s", *req.Size)
		}
	}

	// Validate quality
	if req.Quality != nil && *req.Quality != "standard" && *req.Quality != "hd" {
		return fmt.Errorf("quality must be 'standard' or 'hd'")
	}

	// Validate response format
	if req.ResponseFormat != nil && *req.ResponseFormat != "url" && *req.ResponseFormat != "b64_json" {
		return fmt.Errorf("response_format must be 'url' or 'b64_json'")
	}

	return nil
}

func (h *ImagesHandler) setDefaults(req *models.ImageGenerationRequest) {
	if req.Model == nil {
		req.Model = &h.config.DefaultModel
	}

	if req.N == nil {
		n := 1
		req.N = &n
	}

	if req.Size == nil {
		size := "1024x1024"
		req.Size = &size
	}

	if req.ResponseFormat == nil {
		format := "url"
		req.ResponseFormat = &format
	}

	if req.Quality == nil {
		quality := "standard"
		req.Quality = &quality
	}
}

func (h *ImagesHandler) recordImageUsage(ctx context.Context, userID string, req models.ImageGenerationRequest, resp *models.ImageGenerationResponse) {
	// Record in database for billing/analytics
	usage := &models.APIUsage{
		ID:        uuid.New().String(),
		UserID:    userID,
		Endpoint:  "/v1/images/generations",
		Method:    "POST",
		Model:     *req.Model,
		Success:   true,
		CreatedAt: time.Now(),
	}

	h.db.RecordAPIUsage(ctx, usage)
}
```

### 4. Static File Server для изображений

```go
// internal/api/router/router.go

// Serve generated images
r.engine.Static("/images", "./data/images")
```

### 5. Router Integration

```go
// internal/api/router/router.go

images := r.engine.Group("/v1/images")
images.Use(middleware.AuthMiddleware(r.authService))
{
    images.POST("/generations", r.imagesHandler.GenerateImages)
    // Future:
    // images.POST("/edits", r.imagesHandler.EditImage)
    // images.POST("/variations", r.imagesHandler.CreateVariation)
}
```

## 📝 Конфигурация

```yaml
# configs/dev.yaml
images:
  enabled: false  # Отключено пока Ollama не добавит поддержку
  default_model: "stable-diffusion"
  storage_dir: "./data/images"
  base_url: "http://localhost:8080"
  max_images_per_request: 10
  supported_sizes:
    - "256x256"
    - "512x512"
    - "1024x1024"
    - "1792x1024"
    - "1024x1792"
  cleanup_after_hours: 24  # Удалять старые изображения
```

## 🧪 Тестирование

### Unit Tests

```go
func TestImagesHandler_GenerateImages(t *testing.T) {
	handler := NewImagesHandler(mockClient, mockDB, logrus.New(), ImagesConfig{
		Enabled: true,
		MaxImagesPerRequest: 10,
	})

	req := models.ImageGenerationRequest{
		Prompt: "A beautiful sunset over mountains",
		N:      intPtr(1),
	}

	// Test generation
	resp, err := handler.GenerateImages(context.Background(), req)
	require.NoError(t, err)
	assert.Len(t, resp.Data, 1)
}

func TestImagesHandler_ValidationErrors(t *testing.T) {
	// Test invalid N
	// Test invalid size
	// Test invalid format
}
```

### Integration Tests

```bash
# Generate image (когда Ollama добавит поддержку)
curl -X POST http://localhost:8080/v1/images/generations \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A futuristic city at night",
    "n": 1,
    "size": "1024x1024",
    "response_format": "url"
  }'

# Response:
# {
#   "created": 1234567890,
#   "data": [
#     {
#       "url": "http://localhost:8080/images/abc-123.png"
#     }
#   ]
# }
```

## 📁 Структура файлов

```
internal/
├── models/
│   └── images.go              # Image generation models
├── api/
│   └── handlers/
│       ├── images.go          # Images handler
│       └── images_test.go
├── client/
│   └── ollama/
│       ├── images.go          # Ollama image client
│       └── images_test.go
data/
└── images/                    # Generated images storage
```

## 🎯 Критерии успеха

- [ ] `/v1/images/generations` endpoint реализован
- [ ] OpenAI API совместимость
- [ ] Graceful degradation если Ollama не поддерживает
- [ ] Image storage и serving
- [ ] URL и base64 response formats
- [ ] Size и quality validation
- [ ] Usage recording для billing
- [ ] Unit tests покрывают >90%
- [ ] Документация обновлена

## ⚠️ Важные заметки

1. **Зависит от Ollama**
   - На момент написания Ollama НЕ поддерживает image generation
   - Эта feature будет работать только если Ollama добавит поддержку
   - Graceful error messages если недоступно

2. **Fallback Strategy**
   - Возвращать понятные ошибки
   - `501 Not Implemented` status
   - Указывать на альтернативы (Stable Diffusion API, etc.)

3. **Storage Management**
   - Периодическая очистка старых изображений
   - Disk space monitoring
   - Optional S3/object storage support

## 💡 Future Enhancements

- Image editing (`/v1/images/edits`)
- Image variations (`/v1/images/variations`)
- Upscaling support
- Style transfer
- Inpainting/outpainting
- Integration с внешними image APIs (Stability AI, Midjourney)
- WebUI gallery для generated images
- Batch generation support

## 🔗 External Alternatives

Если Ollama не добавит поддержку, можно интегрировать:

- **Stable Diffusion WebUI API**
- **Stability AI API**
- **Replicate API**
- **HuggingFace Inference API**

---

**Оценка времени:** 6-8 часов  
**Сложность:** MEDIUM  
**Зависимости:** Ollama image model support (BLOCKED)  
**Блокирует:** Нет

**Статус:** ⏸️ BLOCKED до появления поддержки в Ollama


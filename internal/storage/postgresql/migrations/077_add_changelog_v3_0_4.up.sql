INSERT INTO changelogs (version, release_date, content) VALUES
('3.0.4', '2025-11-07', '## [3.0.4] - 2025-11-07

### Added

- **🖼️ Vision Language Model (VLM) Support** (Phase 4: VLM-01, VLM-02, VLM-03):
  - Multimodal chat completions: text + images in single request
  - Support for Qwen2.5-VL, LLaVA, Gemma2 Vision, Phi-3 Vision models
  - Image preprocessing: JPEG, PNG, WebP, Base64 data URIs, file paths
  - Automatic multimodal projector loading (.mmproj files)
  - OpenAI-compatible API: `ChatMessage.Content` supports `[]ContentPart`
  - VLM detection: `IsVLMModel()` method checks loaded models

- **🎨 VLM UI Features** (Phase 4: VLM-03):
  - Image upload button with camera icon
  - Drag-and-drop image support
  - Real-time image preview with thumbnails
  - Image info display: filename, size, format
  - Remove individual images before sending
  - Base64 conversion for API compatibility
  - VLM model validation (prevents sending images to text-only models)
  - Chat history displays images inline with messages

- **📦 Backend VLM Components** (Phase 4: VLM-01, VLM-02):
  - `internal/yzma/vlm.go` - VLM inference logic:
    - `LoadVLMModel(modelPath, mmprojPath)` - loads text + projector
    - `UnloadVLMModel(modelPath)` - frees multimodal context
    - `GenerateWithImages(req VLMGenerateRequest)` - multimodal generation
    - `processImages(imageURLs)` - decodes & converts to `mtmd.Bitmap`
    - `IsVLMModel(modelPath)` - checks if model is VLM
  - `internal/api/handlers/yzma_handler.go`:
    - `containsImages(messages)` - detects multimodal requests
    - `extractTextAndImages(messages)` - parses content
    - `handleVLMCompletion(req)` - routes VLM requests
  - `internal/models/openai.go`:
    - `ChatMessage.Content` now `interface{}` (string or []ContentPart)
    - `ContentPart` struct: Type, Text, ImageURL
    - `ImageURL` struct: URL (base64/file://), Detail

- **🎨 Frontend VLM Components** (Phase 4: VLM-03):
  - `web/js/vlm.js` - VLM manager:
    - `VLMManager` class for image handling
    - `renderImagePreview()` - displays thumbnails
    - `buildMultimodalContent()` - constructs OpenAI format
    - `validateBeforeSend()` - VLM-specific validation
  - `web/chat.html`:
    - Image upload input (hidden)
    - Image preview container
    - Camera button integration
  - `web/css/style.css`:
    - `.image-btn` - gradient purple button
    - `.attached-images-preview` - thumbnail grid
    - `.message-image` - chat history images
    - `.vlm-badge` - VLM indicator

### Changed

- **📊 `yzma` Client Extended** (Phase 4: VLM-01):
  - `ModelContext` now includes:
    - `IsVLM bool` - VLM model flag
    - `MtmdCtx uint64` - multimodal context pointer
    - `MMProjPath string` - path to .mmproj file
  - Added `mtmd` package import for multimodal inference

- **🔧 Chat Logic Enhanced** (Phase 4: VLM-03):
  - `web/js/chat.js`:
    - Checks for attached images before sending
    - Calls `vlmManager.buildMultimodalContent()` for multimodal messages
    - Validates VLM model selection when images present
    - Clears images after successful send

### Technical

- **Dependencies**:
  - `github.com/hybridgroup/yzma/pkg/mtmd` - multimodal inference
  - Image decoding: `image/jpeg`, `image/png`, `golang.org/x/image/webp`
  - Base64 support: `encoding/base64`, `strings` for data URI parsing

- **API Format**:
  ```json
  {
    "model": "qwen2.5-vl-7b",
    "messages": [
      {
        "role": "user",
        "content": [
          {"type": "text", "text": "What is in this image?"},
          {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,..."}}
        ]
      }
    ]
  }
  ```

- **Supported Image Formats**:
  - JPEG, PNG, WebP
  - Base64 data URIs (`data:image/jpeg;base64,...`)
  - Local file paths (`file:///path/to/image.jpg`)
  - Max image size: 10MB (configurable)

- **Performance**:
  - Parallel image decoding
  - Memory-efficient `mtmd.Bitmap` conversion
  - GPU-accelerated VLM inference
  - Automatic cleanup of multimodal contexts

### Supported VLM Models

- **Qwen2.5-VL** (Recommended):
  - `qwen2.5-vl-7b-instruct` + `qwen2.5-vl-7b-instruct-mmproj.gguf`
  - High accuracy, fast inference
  
- **LLaVA**:
  - `llava-v1.6-mistral-7b` + `llava-v1.6-mistral-7b-mmproj.gguf`
  - Excellent vision understanding
  
- **Gemma2 Vision**:
  - `gemma-2-2b-it-vision` + `gemma-2-2b-it-vision-mmproj.gguf`
  - Lightweight, good for edge devices

### Файлы

**Backend:**
- `internal/yzma/vlm.go` (350 строк) - VLM inference engine
- `internal/yzma/client.go` (+50 строк) - VLM context management
- `internal/api/handlers/yzma_handler.go` (+120 строк) - VLM API routes
- `internal/models/openai.go` (+30 строк) - Multimodal message types

**Frontend:**
- `web/chat.html` (+15 строк) - Image upload UI
- `web/js/vlm.js` (260 строк) - VLM UI manager
- `web/js/chat.js` (+25 строк) - Integration with VLM
- `web/css/style.css` (+183 строк) - VLM styles

### Breaking Changes

- ⚠️ `ChatMessage.Content` type changed from `string` to `interface{}`:
  - Old: `Content string`
  - New: `Content interface{}` (can be `string` or `[]ContentPart`)
  - Migration: Existing string content still works (backward compatible)

### Migration Guide

**For API Consumers:**
```go
// Old (text-only):
msg := ChatMessage{
    Role: "user",
    Content: "Hello",
}

// New (text-only, still valid):
msg := ChatMessage{
    Role: "user",
    Content: "Hello",
}

// New (multimodal):
msg := ChatMessage{
    Role: "user",
    Content: []ContentPart{
        {Type: "text", Text: "What is in this image?"},
        {Type: "image_url", ImageURL: ImageURL{URL: "data:image/jpeg;base64,..."}},
    },
}
```

**For VLM Model Setup:**
```bash
# 1. Download GGUF model + mmproj from Hugging Face
wget https://huggingface.co/.../qwen2.5-vl-7b-instruct.Q4_K_M.gguf
wget https://huggingface.co/.../qwen2.5-vl-7b-instruct-mmproj.gguf

# 2. Place in models directory
mv *.gguf data/models/

# 3. Load via WebUI (/yzma.html) or API
curl -X POST http://localhost:8080/api/yzma/load \
  -d ''{"model_path": "/data/models/qwen2.5-vl-7b-instruct.Q4_K_M.gguf", "mmproj_path": "/data/models/qwen2.5-vl-7b-instruct-mmproj.gguf"}''
```')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;


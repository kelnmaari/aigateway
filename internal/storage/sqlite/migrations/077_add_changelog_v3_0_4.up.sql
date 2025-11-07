INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('3.0.4', '2025-11-07', '## [3.0.4] - 2025-11-07

### Added

- **🖼️ VLM (Vision Language Model) Support** (Phase 4: v3.0.4):
  - **Backend VLM Integration**:
    - `internal/yzma/vlm.go` (430 строк) - VLM client wrapper with mmproj support
      - `LoadVLMModel()` - Load text model + mmproj GGUF projector
      - `GenerateWithImages()` - Multi-image inference with streaming
      - Image preprocessing: JPEG, PNG, WebP, Base64 data URI
      - `mtmd.Context` management for multimodal inference
      - Vision support validation
    - `internal/api/handlers/yzma_handler.go` (+165 строк):
      - `handleVLMCompletion()` - OpenAI-compatible VLM API
      - `containsImages()` - Automatic VLM/LLM routing
      - `extractTextAndImages()` - Multimodal content parser
      - Multi-image support (до 5 images per request)
    - `internal/models/openai.go` (+13 строк):
      - `ContentPart` - Multimodal content part (text/image_url)
      - `ImageURL` - Base64 data URI or file path
      
  - **Frontend VLM UI**:
    - `web/js/vlm.js` (260 строк) - VLMManager class
      - Image upload with drag & drop
      - Base64 conversion (автоматическая)
      - Image preview with thumbnails (60x60px)
      - Multi-image management (до 5 images)
      - Provider validation (yzma only)
      - Multimodal content builder для OpenAI API
    - `web/css/style.css` (+183 строки):
      - `.image-btn` - Gradient image button (purple)
      - `.attached-images-preview` - Horizontal scrollable preview
      - `.image-preview-thumb` - 60x60 thumbnails с border
      - `.image-remove-btn` - Remove button с hover effect
      - `.message-image` - Image display в chat history
      - `.vlm-badge` - VLM indicator badge
    - `web/chat.html`:
      - Image upload button (📷 icon)
      - Image preview area (before send)
      - Multi-image grid support
      
  - **OpenAI Multimodal Format**:
    - `/v1/yzma/chat/completions` endpoint extended
    - Support для content as array with text and image_url parts
    - Automatic routing: images → VLM path, text-only → LLM path
    - Multi-image support в single request
    
  - **Supported VLM Models**:
    - ✅ Qwen2.5-VL (3B, 7B) - Recommended
    - ✅ LLaVA (7B, 13B)
    - ✅ Gemma 3 Vision
    - ✅ MiniCPM-V (2.6, 4.5)
    
### Technical

- **yzma mtmd integration** - `github.com/hybridgroup/yzma/pkg/mtmd`
- **Image decoding** - `image/jpeg`, `image/png`, `golang.org/x/image/webp`
- **Base64 handling** - Data URI parsing и encoding
- **File size validation** - 10MB limit per image
- **Provider-based routing** - VLM only для yzma provider
- **Context management** - `mtmd.Context` lifecycle handling
- **Memory safety** - Automatic bitmap cleanup с defer
- **Error handling** - Graceful degradation если VLM не загружен

### Changed

- `internal/models/openai.go` - `ChatMessage.Content` теперь multimodal-ready (string или []ContentPart)
- `web/js/chat.js` - Extended `sendMessage()` для multimodal support
- `web/chat.html` - Added image input button рядом с file attach
- Provider selector - VLM validation при image upload

### Fixed

- Image upload validation - Проверка provider перед отправкой
- Memory leaks - Proper `URL.revokeObjectURL()` для previews
- Large image handling - 10MB limit с clear error messages');


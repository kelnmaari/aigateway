# IMAGE-01: Image Upload & OCR Processing

**Version:** 1.10.2  
**Priority:** HIGH  
**Estimated Time:** 8-10 hours  
**Status:** 📋 Planned  
**Dependencies:** FILE-STORAGE-01, Ollama (LLaVA, BakLLaVA)

**Strategic Importance:** 🔄 OCR engine будет расширен для RAG System (v1.13.0)

---

## ✅ Implementation Summary

**Completed in v1.10.3:**

### Backend Components
1. **Vision Interface** (`internal/vision/interface.go`)
   - OCREngine interface: ExtractText, DescribeImage, SupportedModels
   - OCROptions, OCRResult structures
   - DefaultOCROptions helper

2. **OllamaOCR Implementation** (`internal/vision/ollama_ocr.go`)
   - Uses Ollama multimodal models (LLaVA, BakLLaVA, llama3.2-vision)
   - Base64-free raw bytes transfer via ChatMessage.Images [][]byte
   - Language detection (Russian/English heuristics)
   - Confidence scoring, metadata extraction

3. **ImageProcessor** (`internal/imageproc/processor.go`)
   - Thumbnail generation (CatmullRom filter, customizable size/quality)
   - Image resize (Lanczos filter)
   - Format conversion (JPEG, PNG, WebP)
   - Image validation and metadata extraction

4. **ImageHandler API** (`internal/api/handlers/image_handler.go`)
   - POST `/api/images/upload` - Upload with OCR processing
   - GET `/api/images/:id` - Get image metadata
   - GET `/api/images/:id/download` - Download original
   - GET `/api/images/:id/thumbnail` - On-the-fly thumbnail generation
   - WebSocket integration for upload/OCR progress events

5. **Ollama Client Extension** (`internal/client/ollama/models.go`)
   - Added Images [][]byte field to ChatMessage for vision support

### Testing
- `internal/imageproc/processor_test.go` - All tests passing ✅
- `internal/vision/ollama_ocr_test.go` - All tests passing ✅
- Benchmark tests for thumbnail generation

### Features
- ✅ Multi-format image support (JPEG, PNG, GIF, WebP)
- ✅ Automatic OCR via Ollama vision models
- ✅ Thumbnail generation (200x200px default)
- ✅ Language detection (Russian/English)
- ✅ WebSocket real-time progress notifications
- ✅ Image metadata extraction (width, height, format, size)
- ✅ On-the-fly thumbnail generation for backward compatibility

### Known Limitations
1. **Database schema**: FileMetadata doesn't have dedicated image fields yet (using Custom map instead)
2. **Thumbnail persistence**: Currently generated on-the-fly, not pre-saved to storage
3. **Public images**: Public flag not yet implemented in File model
4. **WebUI**: Drag & drop interface pending implementation

## 🎯 Overview

Система загрузки и обработки изображений с автоматическим распознаванием текста (OCR) через multimodal LLM модели Ollama. Поддерживает drag & drop в чате, thumbnail generation и извлечение текста для контекста.

**Ключевая особенность:** OCR engine через LLaVA будет переиспользован в RAG v1.13.0 для обработки отсканированных документов и изображений в PDF.

### Основные возможности

- 🖼️ **Image Upload**: Drag & drop, paste, file picker
- 👁️ **Multimodal OCR**: LLaVA, BakLLaVA, Llama3.2-Vision через Ollama
- 🔍 **Text Extraction**: Автоматическое извлечение текста из изображений
- 📐 **Image Processing**: Resize, thumbnail generation, format conversion
- 💬 **Chat Integration**: Отправка изображений в чат с контекстом
- 🎯 **RAG-ready**: Интерфейсы готовы для RAG image processing

---

## 🏗️ Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     WebUI Chat Interface                     │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Drag & Drop  │  │   Paste      │  │ File Picker  │      │
│  │              │  │   Ctrl+V     │  │              │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         └──────────────────┴─────────────────┘              │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   Image Service Layer                        │
│  - Upload validation (format, size, dimensions)              │
│  - Image processing (resize, thumbnail, format conversion)   │
│  - Storage (переиспользует FILE-STORAGE-01)                 │
│  - OCR orchestration                                         │
└──────┬──────────────────────────────────┬───────────────────┘
       │                                  │
       ▼                                  ▼
┌─────────────────┐              ┌─────────────────────────────┐
│ Image Storage   │              │    Image Processing         │
│ (FILE-STORAGE)  │              │                             │
│                 │              │  ┌──────────────────────┐   │
│ ./data/images/  │              │  │   Thumbnail Gen      │   │
│ - original/     │              │  │   (imagick/vips)     │   │
│ - thumbnails/   │              │  └──────────────────────┘   │
│ - compressed/   │              │                             │
└─────────────────┘              │  ┌──────────────────────┐   │
                                 │  │   Format Converter   │   │
                                 │  │   (JPEG, PNG, WEBP)  │   │
                                 │  └──────────────────────┘   │
                                 └─────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                    OCR Engine (Ollama)                       │
│                                                              │
│  ┌────────────┐  ┌────────────┐  ┌──────────────────────┐  │
│  │   LLaVA    │  │  BakLLaVA  │  │  Llama3.2-Vision     │  │
│  │            │  │            │  │                      │  │
│  │ 7B/13B     │  │  7B        │  │  11B/90B             │  │
│  │ General    │  │ Multilang  │  │  Latest, Best        │  │
│  └────────────┘  └────────────┘  └──────────────────────┘  │
│                                                              │
│  Prompt: "Extract all text from this image. If it contains  │
│           tables, preserve structure. Return only text."    │
│                                                              │
│  Output: { text, confidence, language, bounding_boxes }     │
└─────────────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                  PostgreSQL Database                         │
│                                                              │
│  Table: images (extends files table)                        │
│  - image metadata (width, height, format, color_space)      │
│  - ocr_text (extracted text)                                │
│  - ocr_confidence, ocr_language                             │
│  - thumbnail_path, compressed_path                          │
│  - exif_data (JSONB: camera, location, timestamp)          │
└─────────────────────────────────────────────────────────────┘
```

### OCR Interface

```go
// internal/vision/interface.go
type OCREngine interface {
    // ExtractText extracts text from image using vision model
    ExtractText(ctx context.Context, image io.Reader, opts OCROptions) (*OCRResult, error)
    
    // DescribeImage generates description of image content
    DescribeImage(ctx context.Context, image io.Reader, prompt string) (string, error)
    
    // AnalyzeDocument analyzes document structure (tables, headers, etc.)
    AnalyzeDocument(ctx context.Context, image io.Reader) (*DocumentAnalysis, error)
    
    // SupportedModels returns list of available vision models
    SupportedModels() []string
}

type OCROptions struct {
    Model         string   // "llava:7b", "bakllava", "llama3.2-vision:11b"
    Language      string   // "ru", "en", "auto"
    PreserveLayout bool    // Try to preserve text layout
    ExtractTables bool     // Detect and extract tables
    Temperature   float64  // For model inference
}

type OCRResult struct {
    Text           string                 // Extracted text
    Confidence     float64                // Overall confidence (0-1)
    Language       string                 // Detected language
    BoundingBoxes  []TextBoundingBox      // Text regions (optional)
    Tables         []TableStructure       // Detected tables (optional)
    Metadata       map[string]interface{} // Additional info
}

type TextBoundingBox struct {
    Text       string
    X, Y       int
    Width      int
    Height     int
    Confidence float64
}

type TableStructure struct {
    Rows    int
    Columns int
    Data    [][]string
    BBox    BoundingBox
}
```

---

## 📊 Database Schema

### Images Table (extends files)

```sql
CREATE TABLE images (
    id UUID PRIMARY KEY REFERENCES files(id) ON DELETE CASCADE,
    
    -- Image properties
    width INT NOT NULL,
    height INT NOT NULL,
    format VARCHAR(10) NOT NULL,  -- 'jpeg', 'png', 'webp', 'gif', 'bmp'
    color_space VARCHAR(20),       -- 'rgb', 'rgba', 'grayscale'
    bit_depth INT,
    file_size_original BIGINT,
    
    -- Processed versions
    thumbnail_path TEXT,
    thumbnail_width INT DEFAULT 200,
    thumbnail_height INT,
    compressed_path TEXT,  -- Optimized version
    
    -- OCR results
    ocr_status VARCHAR(20) DEFAULT 'pending',  -- 'pending', 'processing', 'completed', 'failed', 'no_text'
    ocr_text TEXT,  -- Extracted text
    ocr_confidence FLOAT,  -- 0.0 - 1.0
    ocr_language VARCHAR(10),  -- Detected language
    ocr_model VARCHAR(50),  -- Model used for OCR
    ocr_processing_time_ms INT,
    ocr_error TEXT,
    
    -- EXIF data
    exif_data JSONB,  -- Camera, GPS, timestamp, etc.
    
    -- Analysis
    has_text BOOLEAN DEFAULT false,
    has_tables BOOLEAN DEFAULT false,
    document_type VARCHAR(50),  -- 'screenshot', 'photo', 'scan', 'diagram', 'chart'
    
    -- Timestamps
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_images_ocr_status ON images(ocr_status);
CREATE INDEX idx_images_has_text ON images(has_text);
CREATE INDEX idx_images_format ON images(format);

-- Full-text search на OCR текст
CREATE INDEX idx_images_ocr_text_search ON images USING GIN(to_tsvector('russian', ocr_text));
```

### Image Processing Jobs (для async OCR)

```sql
CREATE TABLE image_processing_jobs (
    id BIGSERIAL PRIMARY KEY,
    image_id UUID NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    job_type VARCHAR(20) NOT NULL,  -- 'ocr', 'thumbnail', 'compress'
    status VARCHAR(20) DEFAULT 'pending',
    
    -- Configuration
    config JSONB,
    
    -- Processing
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    processing_time_ms INT,
    
    -- Results
    result JSONB,
    error TEXT,
    
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_image_jobs_status ON image_processing_jobs(status, created_at);
CREATE INDEX idx_image_jobs_image ON image_processing_jobs(image_id);
```

---

## 🔧 Configuration

### config.yaml

```yaml
images:
  # Storage (переиспользует file_storage)
  storage:
    base_path: "./data/images"
    
  # Upload limits
  upload:
    max_file_size: "25MB"
    allowed_formats: ["jpeg", "jpg", "png", "webp", "gif", "bmp", "tiff"]
    max_width: 8192
    max_height: 8192
    
  # Processing
  processing:
    # Thumbnail generation
    thumbnail:
      enabled: true
      width: 200
      height: 200
      quality: 85
      format: "webp"  # Modern format
      
    # Compression
    compression:
      enabled: true
      quality: 85
      max_dimension: 2048  # Resize large images
      format: "webp"
      
    # Workers
    async_workers: 2
    
  # OCR
  ocr:
    enabled: true
    provider: "ollama"
    
    ollama:
      host: "${OLLAMA_HOST}"
      default_model: "llava:7b"
      
      # Available models
      models:
        - name: "llava:7b"
          description: "General purpose, fast"
          languages: ["en", "ru"]
          
        - name: "llava:13b"
          description: "Better accuracy, slower"
          languages: ["en", "ru"]
          
        - name: "bakllava"
          description: "Multilingual, good for documents"
          languages: ["en", "ru", "de", "fr", "es", "zh"]
          
        - name: "llama3.2-vision:11b"
          description: "Latest Llama 3.2, best quality"
          languages: ["en", "ru", "de", "fr", "es", "zh", "ja", "ko"]
          
        - name: "llama3.2-vision:90b"
          description: "Top quality, requires powerful GPU"
          languages: ["en", "ru", "de", "fr", "es", "zh", "ja", "ko"]
      
      # Prompts
      prompts:
        text_extraction: |
          Extract all visible text from this image. 
          If the image contains a table, preserve its structure using markdown table format.
          If the image has multiple columns, maintain the reading order.
          Return ONLY the extracted text, no additional commentary.
        
        document_analysis: |
          Analyze this document image and describe:
          1. Document type (invoice, receipt, form, etc.)
          2. Main sections and headers
          3. Any tables or structured data
          4. Key information extracted
        
        description: |
          Describe what you see in this image in detail.
      
      # Performance
      timeout: 60s
      temperature: 0.1  # Low temperature for deterministic output
      max_tokens: 2048
      
    # Auto-detect text presence (fast pre-check)
    text_detection:
      enabled: true
      method: "simple"  # 'simple' или 'tesseract' (если установлен)
      confidence_threshold: 0.5
      
  # Chat integration
  chat:
    auto_ocr: true  # Automatically run OCR on upload
    attach_ocr_text: true  # Add OCR text to chat context
    show_preview: true  # Show image preview in chat
```

---

## 📡 API Endpoints

### Upload Image

```http
POST /api/images/upload
Content-Type: multipart/form-data
Authorization: Bearer <jwt_token>

Form Data:
  image: [binary]
  run_ocr: true (optional, default: true)
  generate_thumbnail: true (optional, default: true)
  description: "Invoice from supplier" (optional)

Response 201:
{
  "id": "uuid",
  "filename": "invoice.jpg",
  "url": "/api/images/uuid",
  "thumbnail_url": "/api/images/uuid/thumbnail",
  "width": 1920,
  "height": 1080,
  "format": "jpeg",
  "size_bytes": 456789,
  "ocr_status": "processing",
  "created_at": "2025-01-16T10:00:00Z"
}
```

### Get Image Info

```http
GET /api/images/{id}
Authorization: Bearer <jwt_token>

Response 200:
{
  "id": "uuid",
  "filename": "invoice.jpg",
  "url": "/api/images/uuid/view",
  "thumbnail_url": "/api/images/uuid/thumbnail",
  "width": 1920,
  "height": 1080,
  "format": "jpeg",
  "ocr_status": "completed",
  "ocr_text": "INVOICE\nDate: 2025-01-15\nAmount: $1,234.56\n...",
  "ocr_confidence": 0.94,
  "ocr_language": "en",
  "has_text": true,
  "has_tables": true,
  "metadata": {
    "camera": "iPhone 15",
    "gps": {"lat": 55.7558, "lon": 37.6173}
  },
  "created_at": "2025-01-16T10:00:00Z"
}
```

### Get OCR Text

```http
GET /api/images/{id}/ocr
Authorization: Bearer <jwt_token>

Response 200:
{
  "image_id": "uuid",
  "text": "Full extracted text...",
  "confidence": 0.94,
  "language": "en",
  "bounding_boxes": [
    {"text": "INVOICE", "x": 100, "y": 50, "width": 200, "height": 40}
  ],
  "tables": [
    {
      "rows": 5,
      "columns": 3,
      "data": [["Item", "Qty", "Price"], ["Widget", "2", "$10.00"]]
    }
  ],
  "processing_time_ms": 2450
}
```

### Trigger OCR

```http
POST /api/images/{id}/ocr
Authorization: Bearer <jwt_token>

{
  "model": "llama3.2-vision:11b",
  "language": "auto",
  "preserve_layout": true,
  "extract_tables": true
}

Response 202:
{
  "job_id": 12345,
  "status": "queued",
  "estimated_time_seconds": 15
}
```

### Analyze Image

```http
POST /api/images/{id}/analyze
Authorization: Bearer <jwt_token>

{
  "prompt": "Describe what you see in detail"
}

Response 200:
{
  "description": "This image shows an invoice document dated January 15, 2025...",
  "model": "llava:7b",
  "confidence": 0.91
}
```

### Chat with Image

```http
POST /api/v1/chat/completions
Content-Type: application/json

{
  "model": "llama3.1:8b",
  "messages": [
    {
      "role": "user",
      "content": "What's the total amount on this invoice?",
      "images": ["uuid"]  // Image ID
    }
  ]
}

# System will:
# 1. Load image OCR text
# 2. Add to context: "Image content: INVOICE Date: 2025-01-15 Amount: $1,234.56..."
# 3. Send to LLM

Response 200:
{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": "The total amount on this invoice is $1,234.56."
    }
  }],
  "image_context_used": true,
  "ocr_text_length": 245
}
```

---

## 📦 Package Structure

```
internal/vision/
├── config.go              # OCR configuration
├── service.go             # High-level OCR service
├── interface.go           # OCR engine interface
├── ollama_ocr.go          # Ollama vision models
├── prompts.go             # OCR prompts
├── text_detector.go       # Fast text presence detection
└── service_test.go

internal/imageprocessing/
├── processor.go           # Image processing service
├── thumbnail.go           # Thumbnail generation
├── compression.go         # Image compression
├── format_converter.go    # Format conversion
├── exif.go                # EXIF data extraction
└── processor_test.go

internal/api/handlers/
├── image_handler.go       # Image API handlers
└── image_handler_test.go
```

---

## 🔧 Implementation Details

### Ollama OCR Engine

```go
// internal/vision/ollama_ocr.go
package vision

import (
    "context"
    "encoding/base64"
    "io"
    
    "github.com/ollama/ollama/api"
)

type OllamaOCR struct {
    client  *api.Client
    config  OllamaConfig
}

type OllamaConfig struct {
    Host         string
    DefaultModel string
    Timeout      time.Duration
    Temperature  float64
}

func (o *OllamaOCR) ExtractText(ctx context.Context, image io.Reader, opts OCROptions) (*OCRResult, error) {
    // Read image
    imgData, err := io.ReadAll(image)
    if err != nil {
        return nil, err
    }
    
    // Encode to base64
    imgBase64 := base64.StdEncoding.EncodeToString(imgData)
    
    // Prepare prompt
    model := opts.Model
    if model == "" {
        model = o.config.DefaultModel
    }
    
    prompt := o.buildOCRPrompt(opts)
    
    // Call Ollama vision API
    req := &api.GenerateRequest{
        Model:  model,
        Prompt: prompt,
        Images: []api.ImageData{imgBase64},
        Options: map[string]interface{}{
            "temperature": o.config.Temperature,
            "num_predict": 2048,
        },
    }
    
    var response strings.Builder
    err = o.client.Generate(ctx, req, func(resp api.GenerateResponse) error {
        response.WriteString(resp.Response)
        return nil
    })
    
    if err != nil {
        return nil, fmt.Errorf("ollama generate failed: %w", err)
    }
    
    text := response.String()
    
    return &OCRResult{
        Text:       strings.TrimSpace(text),
        Confidence: 0.9, // Ollama doesn't provide confidence, use default
        Language:   detectLanguage(text),
        Metadata: map[string]interface{}{
            "model": model,
        },
    }, nil
}

func (o *OllamaOCR) buildOCRPrompt(opts OCROptions) string {
    if opts.ExtractTables {
        return `Extract all text from this image. If you see any tables, convert them to markdown table format.
Preserve the structure and layout as much as possible. Return ONLY the extracted text.`
    }
    
    return `Extract all visible text from this image. Maintain the reading order.
Return ONLY the text content, no additional commentary.`
}

func (o *OllamaOCR) DescribeImage(ctx context.Context, image io.Reader, prompt string) (string, error) {
    // Similar to ExtractText but with custom prompt
    imgData, _ := io.ReadAll(image)
    imgBase64 := base64.StdEncoding.EncodeToString(imgData)
    
    req := &api.GenerateRequest{
        Model:  o.config.DefaultModel,
        Prompt: prompt,
        Images: []api.ImageData{imgBase64},
    }
    
    var response strings.Builder
    o.client.Generate(ctx, req, func(resp api.GenerateResponse) error {
        response.WriteString(resp.Response)
        return nil
    })
    
    return response.String(), nil
}
```

### Thumbnail Generator

```go
// internal/imageprocessing/thumbnail.go
package imageprocessing

import (
    "image"
    "image/jpeg"
    "image/png"
    "io"
    
    "github.com/disintegration/imaging"
)

type ThumbnailGenerator struct {
    config ThumbnailConfig
}

type ThumbnailConfig struct {
    Width   int
    Height  int
    Quality int
    Format  string  // "jpeg", "png", "webp"
}

func (g *ThumbnailGenerator) Generate(src io.Reader, dst io.Writer) error {
    // Decode image
    img, format, err := image.Decode(src)
    if err != nil {
        return err
    }
    
    // Resize maintaining aspect ratio
    thumb := imaging.Fit(img, g.config.Width, g.config.Height, imaging.Lanczos)
    
    // Encode
    switch g.config.Format {
    case "jpeg", "jpg":
        return jpeg.Encode(dst, thumb, &jpeg.Options{Quality: g.config.Quality})
    case "png":
        return png.Encode(dst, thumb)
    default:
        return jpeg.Encode(dst, thumb, &jpeg.Options{Quality: g.config.Quality})
    }
}
```

---

## 🧪 Testing Strategy

### Unit Tests

```go
// internal/vision/ollama_ocr_test.go
func TestOllamaOCR_ExtractText(t *testing.T) {
    if testing.Short() {
        t.Skip("Requires Ollama")
    }
    
    ocr := NewOllamaOCR(OllamaConfig{
        Host:        "http://localhost:11434",
        DefaultModel: "llava:7b",
    })
    
    tests := []struct {
        name          string
        imagePath     string
        expectedWords []string
    }{
        {"invoice", "testdata/invoice.jpg", []string{"INVOICE", "Date", "Amount"}},
        {"text doc", "testdata/document.png", []string{"Lorem", "ipsum"}},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            f, _ := os.Open(tt.imagePath)
            defer f.Close()
            
            result, err := ocr.ExtractText(context.Background(), f, OCROptions{})
            assert.NoError(t, err)
            
            for _, word := range tt.expectedWords {
                assert.Contains(t, result.Text, word)
            }
        })
    }
}
```

### Integration Tests

```go
// internal/vision/integration_test.go
func TestImageUploadAndOCR_EndToEnd(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    storage := setupStorage(t)
    ocr := NewOllamaOCR(testConfig())
    service := NewImageService(db, storage, ocr)
    
    // Upload image
    f, _ := os.Open("testdata/invoice.jpg")
    defer f.Close()
    
    image, err := service.Upload(context.Background(), UploadRequest{
        Reader:   f,
        Filename: "invoice.jpg",
        UserID:   "test-user",
        RunOCR:   true,
    })
    
    assert.NoError(t, err)
    assert.Equal(t, "completed", image.OCRStatus)
    assert.Contains(t, image.OCRText, "INVOICE")
}
```

---

## 🎯 Implementation Plan

### Phase 1: Image Storage & Processing (3-4 hours)

**Tasks:**

- ✅ Image metadata schema
- ✅ Thumbnail generator
- ✅ Image compression
- ✅ Format converter
- ✅ EXIF extractor

**Deliverables:**

- Можно загружать изображения
- Генерируются thumbnails
- Работает compression

---

### Phase 2: OCR Engine (3-4 hours)

**Tasks:**

- ✅ Ollama vision integration
- ✅ OCR prompts
- ✅ Text extraction logic
- ✅ Async job processing
- ✅ Unit tests

**Deliverables:**

- OCR работает через LLaVA
- Текст извлекается из изображений
- Async processing функционирует

---

### Phase 3: API & Chat Integration (2 hours)

**Tasks:**

- ✅ REST API endpoints
- ✅ WebSocket updates для OCR progress
- ✅ Chat integration (attach images)
- ✅ WebUI: drag & drop

**Deliverables:**

- API полностью работает
- Можно отправить изображение в чат
- OCR текст добавляется в контекст

---

## 🔗 RAG Integration Path (v1.13.0)

### Переиспользование в RAG

```go
// v1.13.0 RAG будет импортировать OCR engine

import "internal/vision"

// RAG document processor
type RAGDocumentProcessor struct {
    ocr vision.OCREngine
}

func (p *RAGDocumentProcessor) ProcessPDFWithImages(pdf io.Reader) error {
    // Extract PDF pages as images
    pages := extractPDFPages(pdf)
    
    for _, pageImg := range pages {
        // Use v1.10 OCR engine
        result, _ := p.ocr.ExtractText(ctx, pageImg, vision.OCROptions{
            Model: "llama3.2-vision:11b",
            ExtractTables: true,
        })
        
        // NEW in v1.13: Chunk and embed
        chunks := p.chunker.Chunk(result.Text)
        p.embedder.Embed(chunks)
    }
}
```

**Преимущества:**

- ✅ OCR engine готов и протестирован
- ✅ RAG получает качественный OCR из коробки
- ✅ Поддержка отсканированных документов

---

## 📚 Dependencies

### Go Libraries

```go
// Image processing
"image"
"image/jpeg"
"image/png"
"github.com/disintegration/imaging"  // Resize, crop, filters

// EXIF
"github.com/rwcarlsen/goexif/exif"

// Ollama
"github.com/ollama/ollama/api"

// Optional: Advanced processing
"github.com/h2non/bimg"  // libvips wrapper (faster)
```

### External Dependencies

- **Ollama** with vision models (llava:7b, bakllava, llama3.2-vision)
- Optional: **libvips** (для bimg, faster processing)

---

## 📖 Success Metrics

- ✅ Поддержка минимум 5 форматов (JPEG, PNG, WEBP, GIF, BMP)
- ✅ OCR accuracy: >85% для чистых изображений
- ✅ Thumbnail generation: <500ms
- ✅ OCR speed: <5s для LLaVA 7B, <15s для 13B
- ✅ Test coverage: >85%
- ✅ WebUI: drag & drop работает без glitches

---

**Created:** 2025-01-16  
**Last Updated:** 2025-01-16  
**Status:** Ready for Implementation  
**Dependencies:** FILE-STORAGE-01 must be completed first


INSERT INTO changelogs (version, release_date, content) VALUES
('1.10.3', '2025-10-20', '## [1.10.3] - 2025-10-20

### Added
- **IMAGE-01: Image Upload & OCR Processing** 🖼️
  - Vision Interface (internal/vision/interface.go): OCREngine interface: ExtractText, DescribeImage, SupportedModels
  - OllamaOCR Engine (internal/vision/ollama_ocr.go): Multimodal models support: LLaVA, BakLLaVA, Llama3.2-Vision
  - ImageProcessor (internal/imageproc/processor.go): Thumbnail generation, image resize, format conversion
  - ImageHandler API (internal/api/handlers/image_handler.go): POST /api/images/upload, GET /api/images/:id, thumbnail endpoints
  - WebSocket integration для upload/OCR progress events

### Changed
- **Ollama Client Extension**: Added Images [][]byte field to ChatMessage for vision model support

### Technical
- Testing: internal/imageproc/processor_test.go, internal/vision/ollama_ocr_test.go
- Dependencies: github.com/disintegration/imaging v1.6.2
- Features: Multi-format support (JPEG, PNG, GIF, WebP), Automatic OCR, Thumbnail generation, Language detection')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	
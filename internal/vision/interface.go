package vision

import (
	"context"
	"io"
)

// OCREngine defines the interface for an OCR (Optical Character Recognition) engine.
type OCREngine interface {
	// ExtractText extracts text from an image using a vision model.
	ExtractText(ctx context.Context, image io.Reader, opts OCROptions) (*OCRResult, error)

	// DescribeImage generates a description of the image content.
	DescribeImage(ctx context.Context, image io.Reader, prompt string) (string, error)

	// SupportedModels returns a list of available vision models.
	SupportedModels() []string
}

// OCROptions provides options for OCR operations.
type OCROptions struct {
	Model          string  // "llava:7b", "bakllava", "llama3.2-vision:11b"
	Language       string  // "ru", "en", "auto"
	PreserveLayout bool    // Try to preserve text layout
	ExtractTables  bool    // Detect and extract tables
	Temperature    float64 // For model inference (default: 0.0)
	MaxTokens      int     // Max tokens to generate (default: 2048)
}

// DefaultOCROptions returns default OCR options.
func DefaultOCROptions() OCROptions {
	return OCROptions{
		Model:          "llava:7b",
		Language:       "auto",
		PreserveLayout: true,
		ExtractTables:  true,
		Temperature:    0.0,
		MaxTokens:      2048,
	}
}

// OCRResult contains the result of an OCR operation.
type OCRResult struct {
	Text          string            // Extracted text
	Confidence    float64           // Overall confidence (0-1)
	Language      string            // Detected language
	BoundingBoxes []TextBoundingBox // Text regions (optional)
	Tables        []TableStructure  // Detected tables (optional)
	Metadata      map[string]any    // Additional info
}

// TextBoundingBox represents a bounding box for detected text.
type TextBoundingBox struct {
	Text       string
	X, Y       int
	Width      int
	Height     int
	Confidence float64
}

// BoundingBox represents a generic bounding box.
type BoundingBox struct {
	X, Y   int
	Width  int
	Height int
}

// TableStructure represents a detected table.
type TableStructure struct {
	Rows    int
	Columns int
	Data    [][]string
	BBox    BoundingBox
}

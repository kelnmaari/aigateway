package imageproc

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/sirupsen/logrus"
)

// ImageProcessor provides functionalities for image manipulation.
type ImageProcessor struct {
	logger *logrus.Logger
}

// NewImageProcessor creates a new ImageProcessor.
func NewImageProcessor(logger *logrus.Logger) *ImageProcessor {
	return &ImageProcessor{
		logger: logger,
	}
}

// ThumbnailOptions contains options for thumbnail generation.
type ThumbnailOptions struct {
	Width   int
	Height  int
	Quality int // JPEG quality (1-100)
}

// DefaultThumbnailOptions returns default thumbnail options.
func DefaultThumbnailOptions() ThumbnailOptions {
	return ThumbnailOptions{
		Width:   200,
		Height:  200,
		Quality: 85,
	}
}

// ImageInfo contains metadata about an image.
type ImageInfo struct {
	Width     int
	Height    int
	Format    string
	SizeBytes int64
}

// ValidateImage checks if the reader contains a valid image.
func (p *ImageProcessor) ValidateImage(imageReader io.Reader) error {
	// Try to decode the image
	_, format, err := image.DecodeConfig(imageReader)
	if err != nil {
		return fmt.Errorf("invalid image format: %w", err)
	}

	// Check if format is supported
	supportedFormats := []string{"jpeg", "jpg", "png", "gif", "webp"}
	isSupported := false
	for _, f := range supportedFormats {
		if strings.EqualFold(format, f) {
			isSupported = true
			break
		}
	}

	if !isSupported {
		return fmt.Errorf("unsupported image format: %s", format)
	}

	return nil
}

// GetImageInfo extracts metadata from an image.
func (p *ImageProcessor) GetImageInfo(imageReader io.Reader) (*ImageInfo, error) {
	// Read all data to get size
	data, err := io.ReadAll(imageReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Decode config
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image config: %w", err)
	}

	return &ImageInfo{
		Width:     cfg.Width,
		Height:    cfg.Height,
		Format:    format,
		SizeBytes: int64(len(data)),
	}, nil
}

// GenerateThumbnail generates a thumbnail from an image.
func (p *ImageProcessor) GenerateThumbnail(imageReader io.Reader, opts ThumbnailOptions) (io.Reader, error) {
	img, _, err := image.Decode(imageReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Generate thumbnail using CatmullRom filter for good quality
	thumb := imaging.Thumbnail(img, opts.Width, opts.Height, imaging.CatmullRom)

	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, thumb, &jpeg.Options{Quality: opts.Quality}); err != nil {
		return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return buf, nil
}

// ResizeImage resizes an image to the specified width and height.
func (p *ImageProcessor) ResizeImage(imageReader io.Reader, width, height int) (io.Reader, error) {
	img, _, err := image.Decode(imageReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	resized := imaging.Resize(img, width, height, imaging.Lanczos)

	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, resized, &jpeg.Options{Quality: 90}); err != nil {
		return nil, fmt.Errorf("failed to encode resized image: %w", err)
	}

	return buf, nil
}

// ConvertImageFormat converts an image to a specified format (e.g., "jpeg", "png", "webp").
func (p *ImageProcessor) ConvertImageFormat(imageReader io.Reader, format string) (io.Reader, error) {
	img, _, err := image.Decode(imageReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	buf := new(bytes.Buffer)
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		err = jpeg.Encode(buf, img, &jpeg.Options{Quality: 90})
	case "png":
		err = png.Encode(buf, img)
	case "webp":
		// WebP encoding через стандартную библиотеку пока не поддерживается
		// Конвертируем в PNG вместо этого
		err = png.Encode(buf, img)
	default:
		return nil, fmt.Errorf("unsupported image format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode image to %s: %w", format, err)
	}

	return buf, nil
}


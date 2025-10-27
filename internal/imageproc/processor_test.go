package imageproc

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"testing"

	"github.com/sirupsen/logrus"
)

func createTestImage(width, height int) *bytes.Buffer {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with a pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: 128,
				A: 255,
			})
		}
	}

	buf := new(bytes.Buffer)
	jpeg.Encode(buf, img, &jpeg.Options{Quality: 90})
	return buf
}

func TestImageProcessor_ValidateImage(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	processor := NewImageProcessor(logger)

	t.Run("ValidJPEG", func(t *testing.T) {
		imgBuf := createTestImage(100, 100)
		err := processor.ValidateImage(bytes.NewReader(imgBuf.Bytes()))
		if err != nil {
			t.Errorf("Expected valid JPEG to pass validation, got error: %v", err)
		}
	})

	t.Run("InvalidData", func(t *testing.T) {
		invalidData := bytes.NewReader([]byte("not an image"))
		err := processor.ValidateImage(invalidData)
		if err == nil {
			t.Error("Expected invalid data to fail validation")
		}
	})
}

func TestImageProcessor_GetImageInfo(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	processor := NewImageProcessor(logger)

	imgBuf := createTestImage(800, 600)

	info, err := processor.GetImageInfo(bytes.NewReader(imgBuf.Bytes()))
	if err != nil {
		t.Fatalf("Failed to get image info: %v", err)
	}

	if info.Width != 800 {
		t.Errorf("Expected width 800, got %d", info.Width)
	}

	if info.Height != 600 {
		t.Errorf("Expected height 600, got %d", info.Height)
	}

	if info.Format != "jpeg" {
		t.Errorf("Expected format 'jpeg', got '%s'", info.Format)
	}

	if info.SizeBytes == 0 {
		t.Error("Expected non-zero size")
	}
}

func TestImageProcessor_GenerateThumbnail(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	processor := NewImageProcessor(logger)

	imgBuf := createTestImage(800, 600)

	opts := ThumbnailOptions{
		Width:   200,
		Height:  200,
		Quality: 85,
	}

	thumbReader, err := processor.GenerateThumbnail(bytes.NewReader(imgBuf.Bytes()), opts)
	if err != nil {
		t.Fatalf("Failed to generate thumbnail: %v", err)
	}

	// Verify thumbnail is valid JPEG
	thumbData, _ := io.ReadAll(thumbReader)
	_, _, err = image.DecodeConfig(bytes.NewReader(thumbData))
	if err != nil {
		t.Errorf("Generated thumbnail is not a valid image: %v", err)
	}

	// Verify thumbnail size is smaller than original
	if len(thumbData) >= imgBuf.Len() {
		t.Errorf("Expected thumbnail size (%d bytes) to be smaller than original (%d bytes)",
			len(thumbData), imgBuf.Len())
	}
}

func TestImageProcessor_ResizeImage(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	processor := NewImageProcessor(logger)

	imgBuf := createTestImage(800, 600)

	resizedReader, err := processor.ResizeImage(bytes.NewReader(imgBuf.Bytes()), 400, 300)
	if err != nil {
		t.Fatalf("Failed to resize image: %v", err)
	}

	// Verify resized image
	resizedData, _ := io.ReadAll(resizedReader)
	cfg, _, err := image.DecodeConfig(bytes.NewReader(resizedData))
	if err != nil {
		t.Fatalf("Failed to decode resized image: %v", err)
	}

	if cfg.Width != 400 {
		t.Errorf("Expected resized width 400, got %d", cfg.Width)
	}

	if cfg.Height != 300 {
		t.Errorf("Expected resized height 300, got %d", cfg.Height)
	}
}

func TestDefaultThumbnailOptions(t *testing.T) {
	opts := DefaultThumbnailOptions()

	if opts.Width != 200 {
		t.Errorf("Expected default width 200, got %d", opts.Width)
	}

	if opts.Height != 200 {
		t.Errorf("Expected default height 200, got %d", opts.Height)
	}

	if opts.Quality != 85 {
		t.Errorf("Expected default quality 85, got %d", opts.Quality)
	}
}

func BenchmarkImageProcessor_GenerateThumbnail(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)
	processor := NewImageProcessor(logger)

	imgBuf := createTestImage(1920, 1080)
	opts := DefaultThumbnailOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := processor.GenerateThumbnail(bytes.NewReader(imgBuf.Bytes()), opts)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}


package filestorage

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFile_ValidPDF(t *testing.T) {
	// Arrange
	pdfHeader := []byte("%PDF-1.4\n%âãÏÓ\nContent")
	reader := bytes.NewReader(pdfHeader)

	validator := NewValidator(
		[]string{".pdf"},
		10*1024*1024, // 10MB
		true,
	)

	// Act
	err := validator.ValidateFile("test.pdf", int64(len(pdfHeader)), reader)

	// Assert
	assert.NoError(t, err)
}

func TestValidateFile_InvalidExtension(t *testing.T) {
	// Arrange
	content := []byte("malicious content")
	reader := bytes.NewReader(content)

	validator := NewValidator(
		[]string{".pdf", ".docx"},
		10*1024*1024,
		true,
	)

	// Act
	err := validator.ValidateFile("test.exe", int64(len(content)), reader)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestValidateFile_TooLarge(t *testing.T) {
	// Arrange
	largeSize := int64(11 * 1024 * 1024) // 11MB
	content := []byte("small content")
	reader := bytes.NewReader(content)

	validator := NewValidator(
		[]string{".txt"},
		10*1024*1024, // 10MB max
		false,
	)

	// Act
	err := validator.ValidateFile("large.txt", largeSize, reader)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestValidateFile_EmptyFile(t *testing.T) {
	// Arrange
	reader := bytes.NewReader([]byte{})

	validator := NewValidator(
		[]string{".txt"},
		10*1024*1024,
		false,
	)

	// Act - Empty file is allowed (no error expected)
	err := validator.ValidateFile("empty.txt", 0, reader)

	// Assert
	assert.NoError(t, err)
}

func TestValidateFilename_Valid(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{"simple name", "document.pdf"},
		{"with numbers", "report2024.pdf"},
		{"with underscores", "annual_report.pdf"},
		{"with dashes", "annual-report.pdf"},
		{"cyrillic", "документ.pdf"},
		{"unicode", "文档.pdf"},
	}

	validator := NewValidator([]string{".pdf"}, 10*1024*1024, false)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateFilename(tt.filename)
			assert.NoError(t, err)
		})
	}
}

func TestValidateFilename_Invalid(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		errMsg   string
	}{
		{"empty", "", "cannot be empty"},
		{"path traversal dots", "../../../passwd", "invalid characters"},
		{"absolute path unix", "/etc/passwd", "separators"}, // Contains '/' so fails on separator check
		{"absolute path windows", "C:\\windows\\system32", "absolute path"},
		{"with slash", "path/file.pdf", "separators"},
		{"with backslash", "path\\file.pdf", "separators"},
	}

	validator := NewValidator([]string{".pdf"}, 10*1024*1024, false)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateFilename(tt.filename)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestValidator_AllowedExtensions(t *testing.T) {
	validator := NewValidator(
		[]string{".pdf", ".docx", ".txt"},
		10*1024*1024,
		false,
	)

	tests := []struct {
		filename string
		allowed  bool
	}{
		{"document.pdf", true},
		{"report.docx", true},
		{"notes.txt", true},
		{"Document.PDF", true}, // case insensitive
		{"executable.exe", false},
		{"script.sh", false},
		{"archive.zip", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			content := []byte("test content")
			reader := bytes.NewReader(content)

			err := validator.ValidateFile(tt.filename, int64(len(content)), reader)

			if tt.allowed {
				// Should pass extension check (may fail on content validation if enabled)
				if err != nil {
					assert.NotContains(t, err.Error(), "not allowed")
				}
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "not allowed")
			}
		})
	}
}

func TestValidator_ContentValidation_CanBeDisabled(t *testing.T) {
	// With content validation disabled
	validatorNoCheck := NewValidator(
		[]string{".pdf"},
		10*1024*1024,
		false, // validateContent = false
	)

	// Fake PDF (wrong magic number)
	fakePDF := []byte("This is not a real PDF")
	reader := bytes.NewReader(fakePDF)

	err := validatorNoCheck.ValidateFile("fake.pdf", int64(len(fakePDF)), reader)

	// Should NOT fail because content validation is disabled
	assert.NoError(t, err)
}

func TestValidator_ContentValidation_Enabled(t *testing.T) {
	// With content validation enabled
	validatorWithCheck := NewValidator(
		[]string{".pdf"},
		10*1024*1024,
		true, // validateContent = true
	)

	// Fake PDF (wrong magic number)
	fakePDF := []byte("This is not a real PDF")
	reader := bytes.NewReader(fakePDF)

	err := validatorWithCheck.ValidateFile("fake.pdf", int64(len(fakePDF)), reader)

	// Should fail because content validation is enabled
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestValidator_SizeChecks(t *testing.T) {
	tests := []struct {
		name      string
		maxSize   int64
		fileSize  int64
		shouldErr bool
	}{
		{"within limit", 10 * 1024, 5 * 1024, false},
		{"at limit", 10 * 1024, 10 * 1024, false},
		{"over limit", 10 * 1024, 11 * 1024, true},
		{"zero max (no limit)", 0, 100 * 1024, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator([]string{".txt"}, tt.maxSize, false)
			content := []byte("test")
			reader := bytes.NewReader(content)

			err := validator.ValidateFile("test.txt", tt.fileSize, reader)

			if tt.shouldErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "exceeds maximum")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidator_CaseInsensitiveExtensions(t *testing.T) {
	validator := NewValidator([]string{".PDF", ".TXT"}, 10*1024*1024, false)

	tests := []struct {
		filename string
		allowed  bool
	}{
		{"doc.pdf", true},
		{"doc.PDF", true},
		{"doc.Pdf", true},
		{"doc.txt", true},
		{"doc.TXT", true},
		{"doc.exe", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			content := []byte("test")
			reader := bytes.NewReader(content)

			err := validator.ValidateFile(tt.filename, int64(len(content)), reader)

			if !tt.allowed {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not allowed")
			}
		})
	}
}

func TestValidator_NoExtensionRestrictions(t *testing.T) {
	// Empty allowed extensions = allow all
	validator := NewValidator([]string{}, 10*1024*1024, false)

	tests := []string{
		"file.pdf",
		"file.exe",
		"file.txt",
		"file.unknown",
	}

	for _, filename := range tests {
		t.Run(filename, func(t *testing.T) {
			content := []byte("test")
			reader := bytes.NewReader(content)

			err := validator.ValidateFile(filename, int64(len(content)), reader)

			// Should all pass (no extension restrictions)
			assert.NoError(t, err)
		})
	}
}

// Benchmark tests
func BenchmarkValidateFile_SmallFile(b *testing.B) {
	content := []byte("Hello World")
	validator := NewValidator([]string{".txt"}, 10*1024*1024, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(content)
		_ = validator.ValidateFile("test.txt", int64(len(content)), reader)
	}
}

func BenchmarkValidateFile_LargeFile(b *testing.B) {
	content := make([]byte, 1024*1024) // 1MB
	validator := NewValidator([]string{".txt"}, 10*1024*1024, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(content)
		_ = validator.ValidateFile("test.txt", int64(len(content)), reader)
	}
}

func BenchmarkValidateFilename(b *testing.B) {
	filename := "test_document_2024.pdf"
	validator := NewValidator([]string{".pdf"}, 10*1024*1024, false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateFilename(filename)
	}
}

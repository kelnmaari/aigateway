package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aigateway/internal/filestorage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLocalStorage(t *testing.T) {
	// Arrange
	basePath := t.TempDir()
	config := LocalStorageConfig{
		BasePath:    basePath,
		MaxFileSize: 10 * 1024 * 1024,
	}

	// Act
	storage, err := NewLocalStorage(config)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, storage)
	assert.Equal(t, "local", storage.Type())
}

func TestLocalStorage_Store_Success(t *testing.T) {
	// Arrange
	basePath := t.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, err := NewLocalStorage(config)
	require.NoError(t, err)

	ctx := context.Background()
	content := []byte("test file content")
	reader := bytes.NewReader(content)

	opts := filestorage.StoreOptions{
		UserID:   "user123",
		Filename: "document.txt",
		MimeType: "text/plain",
	}

	// Act
	storedPath, err := storage.Store(ctx, reader, opts)

	// Assert
	require.NoError(t, err)
	assert.Contains(t, storedPath, "user123")
	assert.Contains(t, storedPath, "document.txt")

	// Verify file exists
	fullPath := filepath.Join(basePath, storedPath)
	data, err := os.ReadFile(fullPath)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestLocalStorage_Store_CreateUserDirectory(t *testing.T) {
	// Arrange
	basePath := t.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, err := NewLocalStorage(config)
	require.NoError(t, err)

	ctx := context.Background()
	content := []byte("nested file")
	reader := bytes.NewReader(content)

	opts := filestorage.StoreOptions{
		UserID:   "newuser999",
		Filename: "file.txt",
		MimeType: "text/plain",
	}

	// Act
	storedPath, err := storage.Store(ctx, reader, opts)

	// Assert
	require.NoError(t, err)

	// Verify user directory was created
	userDir := filepath.Join(basePath, "newuser999")
	info, err := os.Stat(userDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// Verify file exists
	fullPath := filepath.Join(basePath, storedPath)
	assert.FileExists(t, fullPath)
}

func TestLocalStorage_Retrieve_Success(t *testing.T) {
	// Arrange
	basePath := t.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, err := NewLocalStorage(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Store a file first
	content := []byte("retrieve me")
	reader := bytes.NewReader(content)
	opts := filestorage.StoreOptions{
		UserID:   "user123",
		Filename: "retrieve.txt",
		MimeType: "text/plain",
	}
	storedPath, err := storage.Store(ctx, reader, opts)
	require.NoError(t, err)

	// Act
	retrieved, err := storage.Retrieve(ctx, storedPath)

	// Assert
	require.NoError(t, err)
	defer retrieved.Close()

	data, err := io.ReadAll(retrieved)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestLocalStorage_Retrieve_NotFound(t *testing.T) {
	// Arrange
	basePath := t.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, err := NewLocalStorage(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Act
	reader, err := storage.Retrieve(ctx, "user123/nonexistent_file.txt")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, reader)
	assert.Contains(t, err.Error(), "file not found")
}

func TestLocalStorage_Retrieve_PreventPathTraversal(t *testing.T) {
	// Arrange
	basePath := t.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, err := NewLocalStorage(config)
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name string
		path string
	}{
		{"dots traversal", "../../../etc/passwd"},
		{"absolute path", "/etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			reader, err := storage.Retrieve(ctx, tt.path)

			// Assert - Should error (path traversal or file not found)
			require.Error(t, err)
			assert.Nil(t, reader)
		})
	}
}

func TestLocalStorage_Delete_Success(t *testing.T) {
	// Arrange
	basePath := t.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, err := NewLocalStorage(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Store a file first
	content := []byte("delete me")
	reader := bytes.NewReader(content)
	opts := filestorage.StoreOptions{
		UserID:   "user123",
		Filename: "delete.txt",
		MimeType: "text/plain",
	}
	storedPath, err := storage.Store(ctx, reader, opts)
	require.NoError(t, err)

	// Verify file exists
	fullPath := filepath.Join(basePath, storedPath)
	assert.FileExists(t, fullPath)

	// Act
	err = storage.Delete(ctx, storedPath)

	// Assert
	require.NoError(t, err)

	// Verify file is deleted
	_, err = os.Stat(fullPath)
	assert.True(t, os.IsNotExist(err))
}

func TestLocalStorage_Delete_NotFound(t *testing.T) {
	// Arrange
	basePath := t.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, err := NewLocalStorage(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Act - Delete non-existent file
	err = storage.Delete(ctx, "user123/nonexistent_file.txt")

	// Assert - Returns error for non-existent file
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestLocalStorage_Delete_PreventPathTraversal(t *testing.T) {
	// Arrange
	basePath := t.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, err := NewLocalStorage(config)
	require.NoError(t, err)

	ctx := context.Background()

	tests := []string{
		"../../../etc/passwd",
		"/etc/passwd",
		"..\\..\\windows\\system32",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			// Act
			err := storage.Delete(ctx, path)

			// Assert - Should error (path traversal or file not found)
			require.Error(t, err)
		})
	}
}

func TestLocalStorage_Exists(t *testing.T) {
	// Arrange
	basePath := t.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, err := NewLocalStorage(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Store a file
	content := []byte("I exist")
	reader := bytes.NewReader(content)
	opts := filestorage.StoreOptions{
		UserID:   "user123",
		Filename: "exists.txt",
		MimeType: "text/plain",
	}
	storedPath, err := storage.Store(ctx, reader, opts)
	require.NoError(t, err)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing file", storedPath, true},
		{"non-existing file", "user123/notfound.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			exists, err := storage.Exists(ctx, tt.path)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tt.expected, exists)
		})
	}
}

func TestLocalStorage_GetURL(t *testing.T) {
	// Arrange
	basePath := t.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, err := NewLocalStorage(config)
	require.NoError(t, err)

	ctx := context.Background()
	path := "user123/uuid_file.txt"

	// Act
	url, err := storage.GetURL(ctx, path, 3600)

	// Assert
	require.NoError(t, err)

	// Local storage returns just the path (not file:// URL)
	assert.Equal(t, path, url)
}

func TestLocalStorage_GetSize(t *testing.T) {
	// Arrange
	basePath := t.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, err := NewLocalStorage(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Store a file with known size
	content := []byte("exactly 25 bytes here")
	reader := bytes.NewReader(content)
	opts := filestorage.StoreOptions{
		UserID:   "user123",
		Filename: "size.txt",
		MimeType: "text/plain",
	}
	storedPath, err := storage.Store(ctx, reader, opts)
	require.NoError(t, err)

	// Act
	size, err := storage.GetSize(ctx, storedPath)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), size)
}

func TestLocalStorage_GetSize_NotFound(t *testing.T) {
	// Arrange
	basePath := t.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, err := NewLocalStorage(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Act
	size, err := storage.GetSize(ctx, "user123/nonexistent.txt")

	// Assert
	assert.Error(t, err)
	assert.Equal(t, int64(0), size)
}

func TestLocalStorage_UnicodeFilenames(t *testing.T) {
	// Arrange
	basePath := t.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, err := NewLocalStorage(config)
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name     string
		filename string
	}{
		{"cyrillic", "документ.txt"},
		{"chinese", "文档.txt"},
		{"emoji", "file😀.txt"},
		{"mixed", "test_文档_документ.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := []byte("unicode test")
			reader := bytes.NewReader(content)

			opts := filestorage.StoreOptions{
				UserID:   "unicode_user",
				Filename: tt.filename,
				MimeType: "text/plain",
			}

			// Act - Store
			storedPath, err := storage.Store(ctx, reader, opts)

			// Assert
			require.NoError(t, err)
			assert.Contains(t, storedPath, tt.filename)

			// Verify we can retrieve it
			retrieved, err := storage.Retrieve(ctx, storedPath)
			require.NoError(t, err)
			defer retrieved.Close()

			data, err := io.ReadAll(retrieved)
			require.NoError(t, err)
			assert.Equal(t, content, data)
		})
	}
}

func TestLocalStorage_MultipleUsersIsolation(t *testing.T) {
	// Arrange
	basePath := t.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, err := NewLocalStorage(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Store files for different users
	users := []string{"user1", "user2", "user3"}
	var paths []string

	for _, userID := range users {
		content := []byte("content for " + userID)
		reader := bytes.NewReader(content)

		opts := filestorage.StoreOptions{
			UserID:   userID,
			Filename: "file.txt",
			MimeType: "text/plain",
		}

		path, err := storage.Store(ctx, reader, opts)
		require.NoError(t, err)
		paths = append(paths, path)

		// Verify path contains user ID
		assert.Contains(t, path, userID)
	}

	// Verify all files exist and are in separate directories
	for i, path := range paths {
		exists, err := storage.Exists(ctx, path)
		require.NoError(t, err)
		assert.True(t, exists)

		// Verify content is correct
		retrieved, err := storage.Retrieve(ctx, path)
		require.NoError(t, err)
		defer retrieved.Close()

		data, err := io.ReadAll(retrieved)
		require.NoError(t, err)
		assert.Equal(t, "content for "+users[i], string(data))
	}
}

// Benchmark tests
func BenchmarkLocalStorage_Store(b *testing.B) {
	basePath := b.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, _ := NewLocalStorage(config)
	ctx := context.Background()
	content := []byte(strings.Repeat("test content ", 100))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(content)
		opts := filestorage.StoreOptions{
			UserID:   "benchuser",
			Filename: "bench.txt",
			MimeType: "text/plain",
		}
		_, _ = storage.Store(ctx, reader, opts)
	}
}

func BenchmarkLocalStorage_Retrieve(b *testing.B) {
	basePath := b.TempDir()
	config := LocalStorageConfig{BasePath: basePath}
	storage, _ := NewLocalStorage(config)
	ctx := context.Background()

	// Setup: create test file
	content := []byte(strings.Repeat("test content ", 100))
	reader := bytes.NewReader(content)
	opts := filestorage.StoreOptions{
		UserID:   "benchuser",
		Filename: "bench.txt",
		MimeType: "text/plain",
	}
	path, _ := storage.Store(ctx, reader, opts)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, _ := storage.Retrieve(ctx, path)
		if r != nil {
			io.Copy(io.Discard, r)
			r.Close()
		}
	}
}


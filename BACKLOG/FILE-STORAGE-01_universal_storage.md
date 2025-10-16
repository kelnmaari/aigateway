# FILE-STORAGE-01: Universal File Storage & Document Processing

**Version:** 1.10.1  
**Priority:** HIGH  
**Estimated Time:** 10-14 hours  
**Status:** 📋 Planned  
**Dependencies:** PostgreSQL, MinIO (optional)

**Strategic Importance:** 🔄 Foundation для RAG System (v1.13.0)

---

## 🎯 Overview

Универсальная система хранения и обработки файлов с поддержкой multiple backends (Local FS, S3/MinIO) и автоматическим извлечением текста из различных форматов документов.

**Ключевая особенность:** Архитектура спроектирована с заделом на RAG систему (v1.13.0), где эти компоненты будут переиспользованы без изменений.

### Основные возможности

- 📁 **Multi-backend Storage**: Local FS + S3-compatible (MinIO)
- 📄 **Document Extractors**: PDF, DOCX, TXT, CSV, XLSX
- 🔒 **Security**: File validation, virus scanning (optional), size limits
- 📊 **Metadata Management**: PostgreSQL для файловой метаинформации
- 🔗 **Chat Integration**: Загрузка файла → extraction → контекст для LLM
- 🎯 **RAG-ready**: Интерфейсы готовы для расширения chunking/embedding

---

## 🏗️ Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        WebUI / API                           │
│  POST /api/files/upload                                      │
│  GET  /api/files/{id}                                        │
│  DELETE /api/files/{id}                                      │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   File Service Layer                         │
│  - Upload validation (size, type, content)                   │
│  - Metadata management (PostgreSQL)                          │
│  - Storage backend selection                                 │
│  - Access control (per-user/per-tenant)                      │
└──────┬──────────────────────────────────┬───────────────────┘
       │                                  │
       ▼                                  ▼
┌─────────────────┐              ┌─────────────────┐
│ Local Storage   │              │   S3 Storage    │
│                 │              │   (MinIO)       │
│ ./data/files/   │              │                 │
│ - user_id/      │              │ Bucket: files   │
│   - file_uuid   │              │ - user_id/      │
└─────────────────┘              └─────────────────┘
       │                                  │
       └──────────────┬───────────────────┘
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                 Document Extractors                          │
│                                                              │
│  ┌──────────┐  ┌──────────┐  ┌─────────┐  ┌──────────┐    │
│  │   PDF    │  │  DOCX    │  │   TXT   │  │   CSV    │    │
│  │          │  │          │  │         │  │          │    │
│  │ pdftotext│  │  docx    │  │ UTF-8   │  │ encoding/│    │
│  │ or       │  │  library │  │ detect  │  │   csv    │    │
│  │ go-fitz  │  │          │  │         │  │          │    │
│  └──────────┘  └──────────┘  └─────────┘  └──────────┘    │
│                                                              │
│  Output: ExtractedText {Text, Metadata, PageCount, ...}     │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                  PostgreSQL Database                         │
│                                                              │
│  Table: files                                                │
│  - id, user_id, tenant_id, filename, mime_type              │
│  - storage_backend, storage_path, size_bytes                │
│  - extracted_text (for simple chat integration)             │
│  - metadata (JSONB: page_count, author, etc.)               │
│  - status, created_at, updated_at                           │
└─────────────────────────────────────────────────────────────┘
```

### Storage Backend Interface

```go
// internal/filestorage/interface.go
type StorageBackend interface {
    // Store saves file and returns storage path
    Store(ctx context.Context, file io.Reader, opts StoreOptions) (string, error)
    
    // Retrieve gets file content by path
    Retrieve(ctx context.Context, path string) (io.ReadCloser, error)
    
    // Delete removes file
    Delete(ctx context.Context, path string) error
    
    // Exists checks if file exists
    Exists(ctx context.Context, path string) (bool, error)
    
    // GetURL returns public/signed URL (for S3)
    GetURL(ctx context.Context, path string, expiry time.Duration) (string, error)
}

type StoreOptions struct {
    UserID     string
    TenantID   string
    Filename   string
    MimeType   string
    Metadata   map[string]interface{}
    Public     bool // For S3 ACL
}
```

### Extractor Interface

```go
// internal/extractors/interface.go
type DocumentExtractor interface {
    // Extract extracts text and metadata from document
    Extract(ctx context.Context, reader io.Reader, opts ExtractOptions) (*ExtractedDocument, error)
    
    // SupportedTypes returns MIME types this extractor handles
    SupportedTypes() []string
    
    // MaxFileSize returns maximum supported file size
    MaxFileSize() int64
}

type ExtractedDocument struct {
    Text         string                 // Full extracted text
    Metadata     map[string]interface{} // Document metadata
    PageCount    int                    // Number of pages (if applicable)
    WordCount    int                    // Word count
    Language     string                 // Detected language
    Structure    *DocumentStructure     // Headings, sections (optional)
}

type DocumentStructure struct {
    Headings []Heading
    Sections []Section
    Tables   []Table  // For future RAG
}
```

---

## 📊 Database Schema

### Files Table

```sql
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- File information
    filename VARCHAR(500) NOT NULL,
    original_filename VARCHAR(500) NOT NULL,  -- Original upload name
    mime_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL,
    checksum_sha256 VARCHAR(64),  -- For deduplication
    
    -- Storage
    storage_backend VARCHAR(20) NOT NULL,  -- 'local', 's3'
    storage_path TEXT NOT NULL,  -- Path within backend
    storage_bucket VARCHAR(255),  -- S3 bucket name
    
    -- Extracted content (for simple chat integration)
    extracted_text TEXT,  -- Full text for simple use cases
    extraction_status VARCHAR(20) DEFAULT 'pending',  -- 'pending', 'completed', 'failed'
    extraction_error TEXT,
    
    -- Metadata
    metadata JSONB,  -- page_count, author, keywords, etc.
    
    -- Document info
    page_count INT,
    word_count INT,
    language VARCHAR(10),
    
    -- Access control
    is_public BOOLEAN DEFAULT false,
    shared_with UUID[],  -- Array of user IDs
    
    -- Usage tracking
    download_count INT DEFAULT 0,
    last_accessed_at TIMESTAMP,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP  -- Soft delete
);

CREATE INDEX idx_files_user ON files(user_id);
CREATE INDEX idx_files_tenant ON files(tenant_id);
CREATE INDEX idx_files_mime_type ON files(mime_type);
CREATE INDEX idx_files_status ON files(extraction_status);
CREATE INDEX idx_files_checksum ON files(checksum_sha256);
CREATE INDEX idx_files_created ON files(created_at DESC);

-- Full-text search на extracted_text (для простого поиска)
CREATE INDEX idx_files_text_search ON files USING GIN(to_tsvector('russian', extracted_text));
```

### File Access Logs (optional, для audit)

```sql
CREATE TABLE file_access_logs (
    id BIGSERIAL PRIMARY KEY,
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(20) NOT NULL,  -- 'upload', 'download', 'delete', 'view'
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_file_logs_file ON file_access_logs(file_id);
CREATE INDEX idx_file_logs_user ON file_access_logs(user_id);
CREATE INDEX idx_file_logs_created ON file_access_logs(created_at DESC);
```

---

## 🔧 Configuration

### config.yaml

```yaml
file_storage:
  # Storage backend selection
  backend: "local"  # или "s3"
  
  # Local filesystem storage
  local:
    base_path: "./data/files"
    max_file_size: "100MB"  # Per file
    max_total_size: "10GB"   # Per user
    allowed_extensions:
      - ".pdf"
      - ".docx"
      - ".doc"
      - ".txt"
      - ".csv"
      - ".xlsx"
      - ".xls"
      - ".md"
      - ".rtf"
    
    # Security
    scan_viruses: false  # ClamAV integration (future)
    validate_content: true  # Magic number check
    
  # S3-compatible storage (MinIO)
  s3:
    endpoint: "http://minio:9000"
    bucket: "user-files"
    access_key: "${MINIO_ACCESS_KEY}"
    secret_key: "${MINIO_SECRET_KEY}"
    use_ssl: false
    region: "us-east-1"
    
    # Public access
    public_bucket: "public-files"  # For shared files
    signed_url_expiry: "1h"
    
    max_file_size: "500MB"  # S3 can handle larger files

# Document extraction
extractors:
  # PDF extraction
  pdf:
    method: "pdftotext"  # или "go-fitz" (faster but requires CGO)
    preserve_layout: false
    extract_images: false  # For future IMAGE-01 integration
    ocr_enabled: false     # For scanned PDFs (future)
    max_pages: 1000
    
  # DOCX extraction  
  docx:
    extract_tables: true
    extract_images: false
    extract_comments: false
    
  # Text files
  text:
    max_size: "10MB"
    encoding: "utf-8"  # Auto-detect if not specified
    fallback_encodings: ["windows-1251", "iso-8859-1"]
    
  # CSV files
  csv:
    delimiter: ","
    max_rows: 100000
    encoding: "utf-8"
    auto_detect_delimiter: true
    
  # General
  timeout: 30s  # Per file extraction
  parallel_workers: 4  # For batch processing
```

---

## 📡 API Endpoints

### File Upload

```http
POST /api/files/upload
Content-Type: multipart/form-data
Authorization: Bearer <jwt_token>

Form Data:
  file: [binary]
  description: "Customer list Q1 2025" (optional)
  tags: ["crm", "customers"] (optional)
  is_public: false (optional)
  extract: true (optional, default: true)

Response 201:
{
  "id": "uuid",
  "filename": "customers.csv",
  "mime_type": "text/csv",
  "size_bytes": 124567,
  "storage_backend": "local",
  "extraction_status": "completed",
  "metadata": {
    "page_count": 1,
    "word_count": 1523,
    "row_count": 250
  },
  "download_url": "/api/files/uuid/download",
  "created_at": "2025-01-16T10:00:00Z"
}
```

### Batch Upload

```http
POST /api/files/upload/batch
Content-Type: multipart/form-data
Authorization: Bearer <jwt_token>

Form Data:
  files[]: [binary]
  files[]: [binary]
  extract: true

Response 201:
{
  "files": [
    {"id": "uuid1", "filename": "doc1.pdf", "status": "completed"},
    {"id": "uuid2", "filename": "doc2.docx", "status": "completed"}
  ],
  "total": 2,
  "succeeded": 2,
  "failed": 0
}
```

### Get File Info

```http
GET /api/files/{id}
Authorization: Bearer <jwt_token>

Response 200:
{
  "id": "uuid",
  "filename": "report.pdf",
  "original_filename": "Q1 Report 2025.pdf",
  "mime_type": "application/pdf",
  "size_bytes": 2456789,
  "storage_backend": "s3",
  "extraction_status": "completed",
  "extracted_text": "Full text...",  // Truncated in response
  "metadata": {
    "page_count": 24,
    "word_count": 5234,
    "author": "John Doe",
    "created_date": "2025-01-10"
  },
  "download_url": "/api/files/uuid/download",
  "download_count": 5,
  "created_at": "2025-01-16T10:00:00Z"
}
```

### Download File

```http
GET /api/files/{id}/download
Authorization: Bearer <jwt_token>

Response 200:
Content-Type: application/pdf
Content-Disposition: attachment; filename="report.pdf"

[binary data]
```

### Get Extracted Text

```http
GET /api/files/{id}/text
Authorization: Bearer <jwt_token>

Response 200:
{
  "file_id": "uuid",
  "text": "Full extracted text content...",
  "metadata": {
    "page_count": 24,
    "word_count": 5234,
    "language": "en"
  }
}
```

### List Files

```http
GET /api/files?mime_type=application/pdf&limit=20&offset=0&sort=created_at&order=desc
Authorization: Bearer <jwt_token>

Response 200:
{
  "files": [
    {
      "id": "uuid",
      "filename": "report.pdf",
      "size_bytes": 2456789,
      "extraction_status": "completed",
      "created_at": "2025-01-16T10:00:00Z"
    }
  ],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

### Delete File

```http
DELETE /api/files/{id}
Authorization: Bearer <jwt_token>

Response 204: No Content
```

### Search Files

```http
POST /api/files/search
Authorization: Bearer <jwt_token>

{
  "query": "customer balance",
  "mime_types": ["text/csv", "application/pdf"],
  "date_from": "2025-01-01",
  "date_to": "2025-01-31",
  "tags": ["crm"],
  "limit": 10
}

Response 200:
{
  "results": [
    {
      "file_id": "uuid",
      "filename": "customers.csv",
      "relevance_score": 0.89,
      "highlight": "...customer balance > 10000..."
    }
  ],
  "total": 5
}
```

---

## 📦 Package Structure

```
internal/filestorage/
├── config.go              # Configuration
├── service.go             # High-level file service
├── metadata.go            # File metadata management
├── validator.go           # File validation
│
├── storage/
│   ├── interface.go       # StorageBackend interface
│   ├── local.go           # Local filesystem
│   ├── s3.go              # S3/MinIO storage
│   └── factory.go         # Backend factory
│
└── storage_test.go        # Storage tests

internal/extractors/
├── interface.go           # Extractor interface
├── registry.go            # Extractor registry
├── pdf.go                 # PDF extractor
├── docx.go                # DOCX extractor
├── text.go                # Plain text extractor
├── csv.go                 # CSV parser
├── excel.go               # Excel (XLSX) extractor
├── utils.go               # Common utilities
│
└── *_test.go              # Unit tests

internal/api/handlers/
├── file_handler.go        # File API handlers
└── file_handler_test.go
```

---

## 🔧 Implementation Details

### PDF Extractor (Example)

```go
// internal/extractors/pdf.go
package extractors

import (
    "context"
    "io"
    "os/exec"
    "strings"
)

type PDFExtractor struct {
    config PDFConfig
}

type PDFConfig struct {
    PreserveLayout bool
    MaxPages       int
    Timeout        time.Duration
}

func (e *PDFExtractor) Extract(ctx context.Context, reader io.Reader, opts ExtractOptions) (*ExtractedDocument, error) {
    // Save to temp file
    tmpFile, err := saveTempFile(reader)
    if err != nil {
        return nil, err
    }
    defer os.Remove(tmpFile)
    
    // Run pdftotext
    cmd := exec.CommandContext(ctx, "pdftotext", tmpFile, "-")
    if !e.config.PreserveLayout {
        cmd.Args = append(cmd.Args, "-layout")
    }
    
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("pdftotext failed: %w", err)
    }
    
    text := string(output)
    
    // Extract metadata (using pdfinfo)
    metadata, pageCount := e.extractMetadata(tmpFile)
    
    return &ExtractedDocument{
        Text:      text,
        Metadata:  metadata,
        PageCount: pageCount,
        WordCount: countWords(text),
        Language:  detectLanguage(text),
    }, nil
}

func (e *PDFExtractor) SupportedTypes() []string {
    return []string{"application/pdf"}
}
```

### DOCX Extractor

```go
// internal/extractors/docx.go
package extractors

import (
    "archive/zip"
    "encoding/xml"
    "io"
)

type DOCXExtractor struct {
    config DOCXConfig
}

func (e *DOCXExtractor) Extract(ctx context.Context, reader io.Reader, opts ExtractOptions) (*ExtractedDocument, error) {
    // DOCX is a ZIP archive
    data, err := io.ReadAll(reader)
    if err != nil {
        return nil, err
    }
    
    zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
    if err != nil {
        return nil, err
    }
    
    // Find document.xml
    var docXML *zip.File
    for _, f := range zipReader.File {
        if f.Name == "word/document.xml" {
            docXML = f
            break
        }
    }
    
    if docXML == nil {
        return nil, errors.New("document.xml not found in DOCX")
    }
    
    // Parse XML and extract text
    text, err := e.parseDocumentXML(docXML)
    if err != nil {
        return nil, err
    }
    
    // Extract metadata from core.xml
    metadata := e.extractMetadata(zipReader)
    
    return &ExtractedDocument{
        Text:      text,
        Metadata:  metadata,
        WordCount: countWords(text),
        Language:  detectLanguage(text),
    }, nil
}
```

### Local Storage Backend

```go
// internal/filestorage/storage/local.go
package storage

type LocalStorage struct {
    basePath string
    config   LocalConfig
}

func (s *LocalStorage) Store(ctx context.Context, file io.Reader, opts StoreOptions) (string, error) {
    // Generate path: basePath/userID/uuid_filename
    fileID := uuid.New().String()
    relativePath := filepath.Join(opts.UserID, fileID+"_"+opts.Filename)
    fullPath := filepath.Join(s.basePath, relativePath)
    
    // Create directory
    if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
        return "", err
    }
    
    // Save file
    dst, err := os.Create(fullPath)
    if err != nil {
        return "", err
    }
    defer dst.Close()
    
    if _, err := io.Copy(dst, file); err != nil {
        os.Remove(fullPath)
        return "", err
    }
    
    return relativePath, nil
}

func (s *LocalStorage) Retrieve(ctx context.Context, path string) (io.ReadCloser, error) {
    fullPath := filepath.Join(s.basePath, path)
    return os.Open(fullPath)
}
```

---

## 🧪 Testing Strategy

### Unit Tests

```go
// internal/extractors/pdf_test.go
func TestPDFExtractor_Extract(t *testing.T) {
    tests := []struct {
        name          string
        inputFile     string
        expectedWords int
        expectedPages int
    }{
        {"simple PDF", "testdata/simple.pdf", 100, 1},
        {"multi-page", "testdata/multi.pdf", 5000, 10},
        {"with images", "testdata/images.pdf", 500, 3},
    }
    
    extractor := NewPDFExtractor(PDFConfig{})
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            f, _ := os.Open(tt.inputFile)
            defer f.Close()
            
            doc, err := extractor.Extract(context.Background(), f, ExtractOptions{})
            assert.NoError(t, err)
            assert.Equal(t, tt.expectedPages, doc.PageCount)
            assert.InDelta(t, tt.expectedWords, doc.WordCount, 10)
        })
    }
}
```

### Integration Tests

```go
// internal/filestorage/integration_test.go
func TestFileService_UploadAndExtract(t *testing.T) {
    if testing.Short() {
        t.Skip()
    }
    
    // Setup: DB + Storage + Extractors
    db := setupTestDB(t)
    storage := NewLocalStorage("./testdata/files")
    service := NewFileService(db, storage, NewExtractorRegistry())
    
    // Upload PDF
    f, _ := os.Open("testdata/sample.pdf")
    defer f.Close()
    
    file, err := service.Upload(context.Background(), UploadRequest{
        Reader:   f,
        Filename: "sample.pdf",
        MimeType: "application/pdf",
        UserID:   "test-user",
        Extract:  true,
    })
    
    assert.NoError(t, err)
    assert.Equal(t, "completed", file.ExtractionStatus)
    assert.NotEmpty(t, file.ExtractedText)
    
    // Download
    reader, err := service.Download(context.Background(), file.ID, "test-user")
    assert.NoError(t, err)
    defer reader.Close()
}
```

---

## 🎯 Implementation Plan

### Phase 1: Storage Foundation (3-4 hours)

**Tasks:**

- ✅ Storage interface definition
- ✅ Local filesystem backend
- ✅ S3/MinIO backend
- ✅ Configuration loading
- ✅ Basic file service (upload/download)

**Deliverables:**

- Можно загружать/скачивать файлы
- Работают оба backend (Local + S3)

---

### Phase 2: Document Extractors (4-5 hours)

**Tasks:**

- ✅ Extractor interface + registry
- ✅ PDF extractor (pdftotext)
- ✅ DOCX extractor (ZIP + XML parsing)
- ✅ TXT extractor (encoding detection)
- ✅ CSV parser
- ✅ Unit tests

**Deliverables:**

- Работает extraction для всех типов
- Тесты покрывают >90%

---

### Phase 3: Database Integration (2-3 hours)

**Tasks:**

- ✅ Database schema migration
- ✅ File metadata CRUD
- ✅ Full-text search setup
- ✅ Access control logic

**Deliverables:**

- Метаданные сохраняются в PostgreSQL
- Работает поиск по контенту

---

### Phase 4: API & Chat Integration (1-2 hours)

**Tasks:**

- ✅ REST API handlers
- ✅ File upload/download endpoints
- ✅ Chat integration (add file context)
- ✅ WebUI updates (file picker)

**Deliverables:**

- API полностью работает
- Можно прикрепить файл к чату

---

## 🔗 RAG Integration Path (v1.13.0)

### Как компоненты будут переиспользованы

```go
// v1.13.0 RAG will import and extend v1.10.1

import (
    "internal/filestorage"        // Storage backend as-is
    "internal/extractors"          // Extractors as-is
)

// RAG-specific wrapper
type RAGFileProcessor struct {
    storage    filestorage.StorageBackend
    extractors extractors.Registry
    chunker    *Chunker              // NEW in v1.13
    embedder   *Embedder             // NEW in v1.13
    vectorDB   *VectorDB             // NEW in v1.13
}

func (p *RAGFileProcessor) ProcessForRAG(ctx context.Context, fileID string) error {
    // 1. Download file (using v1.10 storage)
    reader, _ := p.storage.Retrieve(ctx, fileID)
    
    // 2. Extract text (using v1.10 extractors)
    doc, _ := p.extractors.Extract(ctx, reader)
    
    // 3. NEW: Chunk text (v1.13)
    chunks := p.chunker.Chunk(doc.Text)
    
    // 4. NEW: Embed chunks (v1.13)
    embeddings := p.embedder.Embed(chunks)
    
    // 5. NEW: Store in vector DB (v1.13)
    p.vectorDB.Store(embeddings)
    
    return nil
}
```

**Преимущества:**

- ✅ Нулевая доработка extractors и storage
- ✅ RAG фокусируется на chunking/embedding/search
- ✅ Тестирование: extractors уже протестированы

---

## 📚 Dependencies

### Go Libraries

```go
// PDF processing
"os/exec"  // pdftotext command
// OR
"github.com/gen2brain/go-fitz"  // Pure Go PDF (requires CGO)

// DOCX processing
"archive/zip"
"encoding/xml"

// CSV processing
"encoding/csv"  // stdlib

// Encoding detection
"golang.org/x/text/encoding"
"golang.org/x/text/transform"

// Storage
"github.com/minio/minio-go/v7"  // S3/MinIO client

// File type detection
"github.com/h2non/filetype"  // Magic number detection

// Database
"github.com/lib/pq"  // PostgreSQL driver
```

### External Tools

- **pdftotext** (poppler-utils) - для PDF extraction
- **MinIO** - для S3-compatible storage (optional)

---

## 📖 Success Metrics

- ✅ Поддержка минимум 5 типов файлов (PDF, DOCX, TXT, CSV, XLSX)
- ✅ Upload speed: >10MB/s для local storage
- ✅ Extraction accuracy: >95% для чистых документов
- ✅ Test coverage: >90%
- ✅ API response time: <100ms для metadata operations
- ✅ Zero downtime при переключении storage backend

---

## 🔒 Security Considerations

1. **File validation**: MIME type + magic number check
2. **Size limits**: Configurable per file type
3. **Virus scanning**: Optional ClamAV integration
4. **Path traversal protection**: Sanitize filenames
5. **Access control**: Per-user/per-tenant isolation
6. **Audit logging**: All file operations logged
7. **Signed URLs**: Time-limited access for S3

---

**Created:** 2025-01-16  
**Last Updated:** 2025-01-16  
**Status:** Ready for Implementation  
**Next:** IMAGE-01 (v1.10.2) будет использовать этот storage layer

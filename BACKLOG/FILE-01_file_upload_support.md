# FILE-01: File Upload Support (PDF, DOCX, TXT)

**Версия:** 1.8.0  
**Приоритет:** Medium  
**Сложность:** Medium  
**Оценка:** 8-10 часов

## Описание

Поддержка загрузки и извлечения текста из файлов (PDF, DOCX, TXT) в чат. Пользователи смогут загружать документы, система извлечет текст и использует его как контекст для LLM.

## Проблема

В текущей версии чат работает только с текстовыми сообщениями. Пользователи не могут:
- Загружать документы для анализа
- Задавать вопросы о содержимом PDF/DOCX
- Извлекать информацию из больших документов
- Суммировать содержимое файлов

## Решение

File upload + text extraction + LLM context integration.

### Поддерживаемые форматы

1. **TXT** - plain text (UTF-8, ASCII)
2. **PDF** - Portable Document Format
3. **DOCX** - Microsoft Word (OpenXML)

**Не поддерживаются:**
- DOC (legacy Word format) - устаревший
- RTF - редко используется
- Archives (ZIP, RAR) - опасность zip bombs

## Технические детали

### Backend Components

**1. File Upload API**

```
POST /api/chat/upload-file
Content-Type: multipart/form-data
Authorization: Bearer <token>

Body:
- file: File (PDF, DOCX, TXT)
- conversation_id: string (optional)
- extract_text: boolean (default: true)

Response:
{
  "file_id": "uuid",
  "filename": "document.pdf",
  "mime_type": "application/pdf",
  "size_bytes": 1234567,
  "text_length": 5432,
  "page_count": 10,
  "uploaded_at": "2025-10-11T12:00:00Z",
  "url": "/api/chat/files/{file_id}"
}
```

**2. Text Extraction Service**

```go
// internal/services/textextract/extractor.go

type TextExtractor interface {
    Extract(ctx context.Context, file io.Reader, mimeType string) (*ExtractedText, error)
}

type ExtractedText struct {
    Text      string
    PageCount int
    WordCount int
    Language  string
    Metadata  map[string]string
}

type MultiFormatExtractor struct {
    pdfExtractor  *PDFExtractor
    docxExtractor *DOCXExtractor
    txtExtractor  *TXTExtractor
}
```

**3. Format-specific Extractors**

**TXT Extractor:**
```go
func (e *TXTExtractor) Extract(ctx context.Context, file io.Reader) (*ExtractedText, error) {
    data, err := io.ReadAll(file)
    if err != nil {
        return nil, err
    }
    
    // Detect encoding (UTF-8, ASCII, etc.)
    text := string(data)
    
    return &ExtractedText{
        Text:      text,
        WordCount: len(strings.Fields(text)),
    }, nil
}
```

**PDF Extractor** (using `pdfcpu` or `unipdf`):
```go
import "github.com/pdfcpu/pdfcpu/pkg/api"

func (e *PDFExtractor) Extract(ctx context.Context, file io.Reader) (*ExtractedText, error) {
    // Save to temp file
    tmpFile, err := os.CreateTemp("", "pdf-*.pdf")
    if err != nil {
        return nil, err
    }
    defer os.Remove(tmpFile.Name())
    
    io.Copy(tmpFile, file)
    tmpFile.Close()
    
    // Extract text
    text, err := api.ExtractText(tmpFile.Name())
    if err != nil {
        return nil, err
    }
    
    // Get page count
    info, err := api.Info(tmpFile.Name())
    pageCount := info.PageCount
    
    return &ExtractedText{
        Text:      text,
        PageCount: pageCount,
        WordCount: len(strings.Fields(text)),
    }, nil
}
```

**DOCX Extractor** (using `gooxml` or `docxlib`):
```go
import "github.com/nguyenthenguyen/docx"

func (e *DOCXExtractor) Extract(ctx context.Context, file io.Reader) (*ExtractedText, error) {
    // Save to temp file
    tmpFile, err := os.CreateTemp("", "docx-*.docx")
    if err != nil {
        return nil, err
    }
    defer os.Remove(tmpFile.Name())
    
    io.Copy(tmpFile, file)
    tmpFile.Close()
    
    // Read DOCX
    doc, err := docx.ReadDocxFile(tmpFile.Name())
    if err != nil {
        return nil, err
    }
    defer doc.Close()
    
    // Extract text
    text := doc.Editable().GetContent()
    
    return &ExtractedText{
        Text:      text,
        WordCount: len(strings.Fields(text)),
    }, nil
}
```

### Storage

```sql
-- Таблица для хранения загруженных файлов
CREATE TABLE chat_files (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  conversation_id TEXT,
  filename TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  storage_path TEXT NOT NULL,
  extracted_text TEXT, -- Extracted text content
  text_length INTEGER,
  page_count INTEGER,
  word_count INTEGER,
  metadata TEXT, -- JSON
  uploaded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

CREATE INDEX idx_chat_files_user_id ON chat_files(user_id);
CREATE INDEX idx_chat_files_conversation_id ON chat_files(conversation_id);
CREATE INDEX idx_chat_files_uploaded_at ON chat_files(uploaded_at DESC);
```

**File Storage:**
- Path: `./data/files/{user_id}/{file_id}.{ext}`
- Extracted text в БД для быстрого доступа
- Original file сохраняется для повторного извлечения

### Frontend (WebUI)

**Chat Interface:**

1. **File Upload Button**
   - Кнопка "📎 Attach File" рядом с image upload
   - Drag & drop поддержка
   - File type filter: `.txt, .pdf, .docx`

2. **Upload Progress**
   - Progress bar для больших файлов
   - Status: "Uploading... 45%"
   - "Extracting text..." после upload

3. **File Preview Card**
   ```
   ┌────────────────────────────────────┐
   │ 📄 document.pdf                    │
   │ 10 pages • 5,432 words • 245 KB   │
   │ [View] [Remove]                    │
   └────────────────────────────────────┘
   ```

4. **Context Indicator**
   - "Document loaded as context" badge
   - Token count indicator
   - Truncation warning если текст слишком большой

**UI Mockup:**
```
┌─────────────────────────────────────────┐
│ User: [📄 document.pdf attached]        │
│       What is the main idea?            │
├─────────────────────────────────────────┤
│ [Context: 5,432 words from document.pdf]│
├─────────────────────────────────────────┤
│ Assistant: Based on the document, the   │
│ main idea is...                         │
└─────────────────────────────────────────┘
```

### Chat Integration

**Automatic Context:**
```
User uploads document.pdf + asks "Summarize this"
      ↓
System: [extract text] → [add to context] → [send to LLM]
      ↓
Prompt: "Here is the document content:\n\n{extracted_text}\n\nUser question: Summarize this"
      ↓
LLM: "This document discusses..."
```

## Security & Validation

**1. File Validation**
```go
func validateFile(file *multipart.FileHeader) error {
    // Size check
    if file.Size > 50*1024*1024 { // 50MB
        return errors.New("file too large (max 50MB)")
    }
    
    // MIME type check (via magic bytes)
    data, _ := io.ReadAll(file.Open())
    mimeType := mimetype.Detect(data).String()
    
    allowedTypes := []string{
        "text/plain",
        "application/pdf",
        "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    }
    
    if !contains(allowedTypes, mimeType) {
        return errors.New("unsupported file type")
    }
    
    return nil
}
```

**2. Content Limits**
- Max file size: 50MB
- Max extracted text: 100,000 words (~400KB)
- Truncate если больше (с предупреждением)

**3. Rate Limiting**
- Max 20 file uploads per user per day
- Max 5 concurrent uploads
- Max 200MB total storage per user

## Требования

### Функциональные

1. ✅ Upload файлов (TXT, PDF, DOCX)
2. ✅ Drag & drop поддержка
3. ✅ Progress indicator для upload
4. ✅ Автоматическое извлечение текста
5. ✅ File preview card с metadata
6. ✅ Integration с chat context
7. ✅ Валидация типа и размера
8. ✅ Storage файлов и extracted text
9. ✅ Cleanup старых файлов (retention policy)
10. ✅ Download original file функция

### Нефункциональные

1. **Performance**
   - Upload < 5s для 10MB файла
   - Text extraction < 10s для 100-страничного PDF
   - LLM response time зависит от context size

2. **Security**
   - MIME type validation via magic bytes
   - No executable files (.exe, .sh, .bat)
   - User isolation (каждый юзер видит только свои файлы)
   - Path traversal prevention

3. **Storage**
   - Организация: `./data/files/{user_id}/{file_id}.{ext}`
   - Automatic cleanup файлов старше 30 дней (configurable)
   - Storage quota per user: 200MB

## Acceptance Criteria

- [ ] Пользователь может загрузить TXT, PDF, DOCX файл
- [ ] Drag & drop работает корректно
- [ ] Progress bar отображается для больших файлов
- [ ] Текст автоматически извлекается из файла
- [ ] File preview card отображается с metadata
- [ ] Extracted text добавляется в context
- [ ] LLM корректно отвечает на вопросы о содержимом
- [ ] Валидация блокирует неподдерживаемые форматы
- [ ] Rate limiting применяется per user
- [ ] Cleanup job удаляет старые файлы
- [ ] Unit tests для extractors (TXT, PDF, DOCX)
- [ ] Integration tests для upload flow

## Риски и зависимости

### Риски

1. **Malicious files** - вредоносные PDF/DOCX
   - Mitigation: Валидация, sandboxed extraction

2. **Large files** - slow extraction
   - Mitigation: Streaming, timeout, progress indicator

3. **Poor text extraction** - PDF с изображениями
   - Mitigation: Fallback на OCR (VISION-01), user notification

4. **Storage overflow** - множество файлов
   - Mitigation: Quotas, automatic cleanup

### Зависимости

1. **pdfcpu** - PDF processing (github.com/pdfcpu/pdfcpu)
   - Alternative: unipdf (commercial license)
2. **docx** - DOCX parsing (github.com/nguyenthenguyen/docx)
3. **mimetype** - MIME detection (github.com/gabriel-vasile/mimetype)

## Связанные задачи

- **VISION-01**: Vision OCR - для PDF с изображениями
- **WEB-01**: Web Fetcher - схожая логика контента
- **QUOTA-01**: Usage Quotas - storage quotas per user

## Примечания

- PDF с scan изображениями требуют OCR (см. VISION-01)
- Password-protected файлы не поддерживаются
- Encrypted PDF не поддерживаются (требуется password)
- Для больших документов (500+ pages) рекомендуется chunking

## Пример использования

```javascript
// Frontend: Upload file
const formData = new FormData();
formData.append('file', fileInput.files[0]);
formData.append('conversation_id', conversationId);

const uploadResponse = await fetch('/api/chat/upload-file', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${token}` },
  body: formData
});

const { file_id, extracted_text, word_count } = await uploadResponse.json();

// Send chat message with context
await fetch('/api/v1/chat/completions', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  },
  body: JSON.stringify({
    model: 'llama3.1:latest',
    messages: [
      {
        role: 'system',
        content: `Document context (${word_count} words):\n\n${extracted_text}`
      },
      {
        role: 'user',
        content: 'Summarize the main points'
      }
    ]
  })
});
```

---

**Статус:** 📋 Planned for v1.8.0  
**Последнее обновление:** 2025-10-11



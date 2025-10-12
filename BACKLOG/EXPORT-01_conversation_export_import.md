# EXPORT-01: Conversation Export/Import

**Версия:** 1.8.0  
**Приоритет:** Medium  
**Сложность:** Low  
**Оценка:** 4-6 часов

## Описание

Функционал экспорта и импорта истории чата. Пользователи смогут сохранять conversations в JSON/Markdown форматах и импортировать их обратно для восстановления истории или миграции между системами.

## Проблема

В текущей версии:
- Невозможно экспортировать историю чата
- Нет backup механизма для conversations
- Миграция данных между инстансами затруднена
- Пользователи не могут поделиться conversations
- Нет архивирования старых чатов

## Решение

API endpoints для экспорта/импорта + UI кнопки в WebUI.

## Технические детали

### Export Formats

**1. JSON Format** (machine-readable)
```json
{
  "version": "1.8.0",
  "exported_at": "2025-10-11T12:00:00Z",
  "conversation": {
    "id": "uuid",
    "title": "Chat about AI",
    "model": "llama3.1:latest",
    "created_at": "2025-10-10T10:00:00Z",
    "updated_at": "2025-10-11T11:30:00Z",
    "messages": [
      {
        "role": "user",
        "content": "Hello!",
        "timestamp": "2025-10-10T10:00:00Z"
      },
      {
        "role": "assistant",
        "content": "Hi! How can I help you?",
        "timestamp": "2025-10-10T10:00:05Z"
      }
    ],
    "metadata": {
      "total_messages": 10,
      "total_tokens": 1234
    }
  }
}
```

**2. Markdown Format** (human-readable)
```markdown
# Chat about AI

**Model:** llama3.1:latest  
**Created:** 2025-10-10 10:00:00  
**Messages:** 10

---

## User
Hello!

*2025-10-10 10:00:00*

---

## Assistant
Hi! How can I help you?

*2025-10-10 10:00:05*

---
```

### Backend API

**1. Export Single Conversation**
```
GET /api/conversations/:id/export
Query params:
  - format: json | markdown (default: json)
  
Response (JSON):
Content-Type: application/json
Content-Disposition: attachment; filename="conversation-{id}.json"
{ ... }

Response (Markdown):
Content-Type: text/markdown
Content-Disposition: attachment; filename="conversation-{id}.md"
```

**2. Export All Conversations (Bulk)**
```
GET /api/conversations/export/all
Query params:
  - format: json | markdown
  - from_date: ISO8601 (optional)
  - to_date: ISO8601 (optional)
  
Response:
Content-Type: application/zip
Content-Disposition: attachment; filename="conversations-export-2025-10-11.zip"

ZIP содержит:
  - conversation-1.json
  - conversation-2.json
  - index.json (metadata)
```

**3. Import Conversation**
```
POST /api/conversations/import
Content-Type: multipart/form-data

Body:
  - file: File (JSON)
  - overwrite: boolean (default: false)
  
Response:
{
  "conversation_id": "uuid",
  "title": "Chat about AI",
  "messages_count": 10,
  "imported_at": "2025-10-11T12:00:00Z"
}
```

### Implementation

**Export Service:**
```go
// internal/services/export/exporter.go

type ConversationExporter struct {
    db storage.Database
}

func (e *ConversationExporter) ExportJSON(ctx context.Context, conversationID string) ([]byte, error) {
    conv, err := e.db.GetConversation(ctx, conversationID)
    if err != nil {
        return nil, err
    }
    
    messages, err := e.db.GetMessages(ctx, conversationID)
    if err != nil {
        return nil, err
    }
    
    export := &ConversationExport{
        Version:      "1.8.0",
        ExportedAt:   time.Now(),
        Conversation: conv,
        Messages:     messages,
    }
    
    return json.MarshalIndent(export, "", "  ")
}

func (e *ConversationExporter) ExportMarkdown(ctx context.Context, conversationID string) ([]byte, error) {
    // Similar logic, format as Markdown
}

func (e *ConversationExporter) ExportAll(ctx context.Context, userID string, opts ExportOptions) ([]byte, error) {
    // Create ZIP archive with all conversations
}
```

**Import Service:**
```go
// internal/services/export/importer.go

type ConversationImporter struct {
    db storage.Database
}

func (i *ConversationImporter) Import(ctx context.Context, data []byte, userID string, overwrite bool) (*models.Conversation, error) {
    var export ConversationExport
    if err := json.Unmarshal(data, &export); err != nil {
        return nil, err
    }
    
    // Version compatibility check
    if !isCompatibleVersion(export.Version) {
        return nil, errors.New("incompatible export version")
    }
    
    // Check if conversation already exists
    existingConv, _ := i.db.GetConversation(ctx, export.Conversation.ID)
    if existingConv != nil && !overwrite {
        return nil, errors.New("conversation already exists")
    }
    
    // Import conversation
    newConv := &models.Conversation{
        ID:        generateNewID(), // Always new ID
        UserID:    userID,
        Title:     export.Conversation.Title,
        Model:     export.Conversation.Model,
        CreatedAt: time.Now(),
    }
    
    if err := i.db.CreateConversation(ctx, newConv); err != nil {
        return nil, err
    }
    
    // Import messages
    for _, msg := range export.Messages {
        newMsg := &models.Message{
            ID:             generateNewID(),
            ConversationID: newConv.ID,
            Role:           msg.Role,
            Content:        msg.Content,
            CreatedAt:      time.Now(),
        }
        if err := i.db.CreateMessage(ctx, newMsg); err != nil {
            return nil, err
        }
    }
    
    return newConv, nil
}
```

### Frontend (WebUI)

**Chat Interface:**

1. **Export Button** (в меню conversation)
   ```
   [...] → Export
           ├─ Export as JSON
           └─ Export as Markdown
   ```

2. **Bulk Export** (в Dashboard/Conversations list)
   ```
   [Export All Conversations]
   └─ Date range picker
   └─ Format selector (JSON/Markdown)
   ```

3. **Import Button**
   ```
   [Import Conversation]
   └─ File upload dialog
   └─ Overwrite checkbox
   ```

**UI Mockup:**
```
┌─────────────────────────────────────┐
│ Chat Title          [...]           │
├─────────────────────────────────────┤
│ Dropdown Menu:                      │
│ ┌─────────────────────────────────┐ │
│ │ Rename                          │ │
│ │ Delete                          │ │
│ │ ───────────────────             │ │
│ │ Export as JSON      ⬇          │ │
│ │ Export as Markdown  ⬇          │ │
│ └─────────────────────────────────┘ │
└─────────────────────────────────────┘
```

## Security & Validation

**1. Export Authorization**
- Только владелец conversation может экспортировать
- Admin может экспортировать любые conversations (audit)

**2. Import Validation**
```go
func validateImport(export *ConversationExport) error {
    // Version check
    if export.Version == "" {
        return errors.New("missing version")
    }
    
    // Structure validation
    if export.Conversation == nil {
        return errors.New("missing conversation")
    }
    
    // Messages validation
    for _, msg := range export.Messages {
        if msg.Role != "user" && msg.Role != "assistant" && msg.Role != "system" {
            return errors.New("invalid message role")
        }
    }
    
    return nil
}
```

**3. Size Limits**
- Max import file size: 50MB
- Max messages per conversation: 10,000

## Требования

### Функциональные

1. ✅ Export single conversation (JSON, Markdown)
2. ✅ Export all conversations (bulk, ZIP archive)
3. ✅ Import conversation from JSON
4. ✅ Date range filter для bulk export
5. ✅ Version compatibility check
6. ✅ Overwrite vs create new option
7. ✅ Metadata в export (timestamp, model, tokens)
8. ✅ Pretty-printed JSON (human-readable)
9. ✅ Markdown с форматированием
10. ✅ Download файлов через browser

### Нефункциональные

1. **Performance**
   - Export < 1s для 100-message conversation
   - Bulk export < 10s для 50 conversations
   - Import < 2s для 100-message conversation

2. **Compatibility**
   - Version field для future migrations
   - Backward compatibility с v1.8.0+
   - Schema validation при import

## Acceptance Criteria

- [ ] Пользователь может экспортировать conversation в JSON
- [ ] Пользователь может экспортировать conversation в Markdown
- [ ] Bulk export всех conversations работает
- [ ] Date range filter применяется корректно
- [ ] ZIP archive содержит все conversations + index
- [ ] Import из JSON восстанавливает conversation
- [ ] Import validation блокирует некорректные файлы
- [ ] Overwrite option работает корректно
- [ ] Authorization проверяется (только owner)
- [ ] Unit tests для export/import functions
- [ ] Integration tests для full flow

## Риски и зависимости

### Риски

1. **Large conversations** - slow export/import
   - Mitigation: Streaming, chunking для больших datasets

2. **Version incompatibility** - старые exports
   - Mitigation: Version migration logic, clear error messages

3. **Data loss** - failed import
   - Mitigation: Transaction rollback, validation before commit

### Зависимости

1. **archive/zip** - ZIP creation (stdlib)
2. **encoding/json** - JSON marshaling (stdlib)
3. Markdown formatting library (optional)

## Связанные задачи

- **OPS-01**: Backup & Restore - схожая логика export/import
- **AUDIT-01**: Enhanced Audit Logging - логирование export/import events

## Примечания

- Export не включает API keys и credentials
- Import создает новые IDs (не сохраняет оригинальные)
- Markdown export может терять некоторую metadata (ограничения формата)
- Для больших datasets рекомендуется использовать database backup (OPS-01)

## Пример использования

```javascript
// Frontend: Export conversation
async function exportConversation(conversationId, format = 'json') {
  const response = await fetch(
    `/api/conversations/${conversationId}/export?format=${format}`,
    {
      headers: { 'Authorization': `Bearer ${token}` }
    }
  );
  
  const blob = await response.blob();
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `conversation-${conversationId}.${format}`;
  a.click();
}

// Frontend: Import conversation
async function importConversation(file) {
  const formData = new FormData();
  formData.append('file', file);
  formData.append('overwrite', 'false');
  
  const response = await fetch('/api/conversations/import', {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${token}` },
    body: formData
  });
  
  const { conversation_id, messages_count } = await response.json();
  toast.success(`Imported ${messages_count} messages`);
  navigateToConversation(conversation_id);
}
```

---

**Статус:** 📋 Planned for v1.8.0  
**Последнее обновление:** 2025-10-11


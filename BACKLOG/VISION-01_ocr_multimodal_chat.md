# VISION-01: Vision OCR Service (Multimodal Chat)

**Версия:** 1.8.0  
**Приоритет:** High  
**Сложность:** Medium  
**Оценка:** 8-12 часов

## Описание

Интеграция multimodal vision models в чат для распознавания текста и анализа изображений. Пользователи смогут загружать изображения (скриншоты, фото документов, диаграммы) и получать текстовое описание или извлеченный текст через нейросетевые модели.

## Проблема

В текущей версии чат работает только с текстовыми сообщениями. Пользователи не могут:

- Загружать изображения для анализа
- Извлекать текст из скриншотов/фото
- Задавать вопросы об изображениях
- Анализировать диаграммы, графики, схемы

## Решение

Интеграция multimodal vision models через Ollama API с поддержкой image input.

### Multimodal Models

Поддерживаемые модели (примеры):

- **LLaVA** (Large Language and Vision Assistant)
  - llava:7b, llava:13b, llava:34b
  - llava-phi3, llava-llama3
- **BakLLaVA** - улучшенная версия LLaVA
- **Другие vision models** доступные в Ollama

Модели выбираются администратором/пользователем динамически.

## Технические детали

### Backend API

**1. Image Upload Endpoint**

```
POST /api/chat/upload-image
Content-Type: multipart/form-data

Body:
- image: File (PNG, JPG, JPEG, WebP)
- conversation_id: string (optional)
- max_size: 10MB

Response:
{
  "image_id": "uuid",
  "filename": "screenshot.png",
  "size": 245678,
  "mime_type": "image/png",
  "url": "/api/chat/images/{image_id}",
  "thumbnail_url": "/api/chat/images/{image_id}/thumb"
}
```

**2. Chat Message with Image**

```
POST /api/v1/chat/completions
{
  "model": "llava:13b",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "What do you see in this image?"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "data:image/png;base64,iVBORw0KGgoAAAANS..."
          }
        }
      ]
    }
  ]
}
```

**3. Ollama API Integration**

Ollama поддерживает multimodal через обновленный API:

```bash
curl http://localhost:11434/api/generate -d '{
  "model": "llava:13b",
  "prompt": "What is in this image?",
  "images": ["base64_encoded_image"]
}'
```

### Storage

```sql
-- Таблица для хранения загруженных изображений
CREATE TABLE chat_images (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  conversation_id TEXT,
  filename TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  width INTEGER,
  height INTEGER,
  storage_path TEXT NOT NULL,
  thumbnail_path TEXT,
  uploaded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

CREATE INDEX idx_chat_images_user_id ON chat_images(user_id);
CREATE INDEX idx_chat_images_conversation_id ON chat_images(conversation_id);
```

### Frontend (WebUI)

**Chat Interface Updates:**

1. **Image Upload Button**
   - Кнопка "📎 Attach Image" рядом с input
   - Drag & drop поддержка
   - Preview загруженного изображения

2. **Image Display in Chat**
   - Inline preview изображений в сообщениях
   - Click to enlarge (lightbox)
   - Thumbnail для больших изображений

3. **Image Management**
   - Удаление загруженного изображения до отправки
   - Список всех изображений в conversation
   - Clear images функция

**UI Mockup:**

```
┌─────────────────────────────────────┐
│ Chat with Vision Model              │
├─────────────────────────────────────┤
│ User: [image preview] What is this? │
│                                      │
│ Assistant: This appears to be a     │
│ screenshot of a code editor showing │
│ a Python function...                 │
├─────────────────────────────────────┤
│ [📎] [Type message...]        [Send]│
└─────────────────────────────────────┘
```

## Требования

### Функциональные

1. ✅ Загрузка изображений (PNG, JPG, JPEG, WebP)
2. ✅ Drag & drop поддержка
3. ✅ Image preview перед отправкой
4. ✅ Валидация размера (max 10MB)
5. ✅ Валидация типа файла
6. ✅ Base64 encoding для Ollama API
7. ✅ Thumbnail generation для больших изображений
8. ✅ Image storage в файловой системе
9. ✅ Metadata сохранение в БД
10. ✅ Очистка orphaned images (cleanup job)

### Нефункциональные

1. **Performance**
   - Image upload < 2s для 5MB файла
   - Thumbnail generation < 500ms
   - Response time зависит от модели

2. **Security**
   - Валидация MIME типа через magic bytes
   - Защита от zip bombs (N/A - archives не поддерживаются)
   - Path traversal prevention
   - User isolation (каждый юзер видит только свои изображения)

3. **Storage**
   - Организация: `./data/images/{user_id}/{image_id}.ext`
   - Thumbnails: `./data/images/{user_id}/thumbs/{image_id}_thumb.jpg`
   - Automatic cleanup старых изображений (configurable retention)

## Acceptance Criteria

- [ ] Пользователь может загрузить изображение через кнопку "Attach"
- [ ] Drag & drop работает корректно
- [ ] Preview изображения отображается перед отправкой
- [ ] Изображение отправляется вместе с текстовым вопросом
- [ ] Vision model корректно анализирует изображение
- [ ] Ответ отображается в чате с контекстом изображения
- [ ] Изображения сохраняются в БД и файловой системе
- [ ] Thumbnails генерируются автоматически
- [ ] Валидация файлов работает (тип, размер)
- [ ] Cleanup job удаляет старые изображения (configurable)
- [ ] Unit tests для upload, validation, storage
- [ ] Integration tests с mock Ollama vision API

## Риски и зависимости

### Риски

1. **Ollama Vision API изменения** - API может измениться
   - Mitigation: Абстракция через interface, проверка версии Ollama

2. **Большие изображения** - медленная обработка
   - Mitigation: Image resizing перед отправкой, max resolution limit

3. **Storage overflow** - множество изображений
   - Mitigation: Automatic cleanup, storage quotas per user

### Зависимости

1. Ollama с установленной vision моделью (llava, bakllava)
2. Go image processing library (golang.org/x/image)
3. Thumbnail generation library (github.com/nfnt/resize)
4. MIME type detection (github.com/gabriel-vasile/mimetype)

## Связанные задачи

- **FILE-01**: File Upload Support (PDF, DOCX, TXT) - схожая логика upload
- **MODEL-01**: Dynamic Model Parameters - настройка vision models
- **QUOTA-01**: Usage Quotas - ограничения на upload size/count

## Примечания

- Vision models требуют больше VRAM чем text-only models
- Рекомендуется минимум 16GB VRAM для llava:13b
- Пользователи должны быть предупреждены о длительности обработки изображений
- Поддержка только статических изображений (не видео, не анимация)

## Пример использования

```javascript
// Frontend: Upload image
const formData = new FormData();
formData.append('image', imageFile);
formData.append('conversation_id', conversationId);

const uploadResponse = await fetch('/api/chat/upload-image', {
  method: 'POST',
  headers: { 'Authorization': `Bearer ${token}` },
  body: formData
});

const { image_id, url } = await uploadResponse.json();

// Send chat message with image
await fetch('/api/v1/chat/completions', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  },
  body: JSON.stringify({
    model: 'llava:13b',
    messages: [{
      role: 'user',
      content: [
        { type: 'text', text: 'What do you see?' },
        { type: 'image_url', image_url: { url: url } }
      ]
    }]
  })
});
```

---

**Статус:** 📋 Planned for v1.8.0  
**Последнее обновление:** 2025-10-11

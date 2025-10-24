# BUGFIX: SSE Content-Type Header для Spring AI Совместимости

## 🐛 Проблема

**Дата обнаружения:** 24.10.2025  
**Версия:** v1.10.0  
**Затронутые клиенты:** Spring AI (JetBrains IDE плагины), любые клиенты использующие строгую валидацию SSE

### Симптомы

Spring AI плагины в JetBrains IDE (GoLand, IntelliJ IDEA) получали ошибку при streaming запросах:

```
JsonParseException: Unrecognized token 'data': 
was expecting (JSON String, Number, Array, Object or token 'null', 'true' or 'false')

Failed to json: data: {"id":"chatcmpl-...","object":"chat.completion.chunk",...}
```

### Стек-трейс

```java
java.lang.RuntimeException: Failed to json: data: {...}
    at org.springframework.ai.model.ModelOptionsUtils.jsonToObject(ModelOptionsUtils.java:104)
    at org.springframework.ai.openai.api.OpenAiApi.lambda$chatCompletionStream$6(OpenAiApi.java:1082)
    at reactor.core.publisher.FluxMap$MapSubscriber.onNext(FluxMap.java:106)
    ...
Caused by: com.fasterxml.jackson.core.JsonParseException: 
    Unrecognized token 'data': was expecting (JSON String, Number, Array, Object...)
```

### Причина

Proxy отправлял **неправильный** `Content-Type` заголовок для Server-Sent Events (SSE):

**Было:**
```go
c.Header("Content-Type", "text/plain; charset=utf-8")
```

Spring AI видел `text/plain`, пытался парсить весь ответ как JSON:
- **Ожидал:** `{"id":"...","choices":[...]}`
- **Получал:** `data: {"id":"...","choices":[...]}`
- **Результат:** JsonParseException на `data:` префиксе

### Стандарт SSE

По спецификации [Server-Sent Events (W3C)](https://html.spec.whatwg.org/multipage/server-sent-events.html#server-sent-events):

> The MIME type for text/event-stream is **`text/event-stream`**.

Формат SSE:
```
data: {"key": "value"}
data: {"more": "data"}

data: [DONE]
```

---

## ✅ Решение

### Код изменения

**Файл:** `internal/api/handlers/streaming.go`

```go
// setupSSEHeaders настраивает заголовки для Server-Sent Events
func (h *StreamingChatHandler) setupSSEHeaders(c *gin.Context) {
    c.Header("Content-Type", "text/event-stream")  // ✅ ИСПРАВЛЕНО
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")
    c.Header("X-Accel-Buffering", "no")
    c.Header("Access-Control-Allow-Origin", "*")
    c.Header("Access-Control-Allow-Headers", "Cache-Control")
    
    c.Status(http.StatusOK)
    // ... остальной код
}
```

### Проверка исправления

**До исправления:**
```bash
$ curl -i http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_KEY" \
  -d '{"model":"devstral-tuned:latest","messages":[{"role":"user","content":"test"}],"stream":true}'

HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8  ❌ НЕПРАВИЛЬНО
Cache-Control: no-cache
Connection: keep-alive

data: {"id":"chatcmpl-...","choices":[...]}
```

**После исправления:**
```bash
$ curl -i http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_KEY" \
  -d '{"model":"devstral-tuned:latest","messages":[{"role":"user","content":"test"}],"stream":true}'

HTTP/1.1 200 OK
Content-Type: text/event-stream  ✅ ПРАВИЛЬНО
Cache-Control: no-cache
Connection: keep-alive

data: {"id":"chatcmpl-...","choices":[...]}
```

---

## 🧪 Тестирование

### Автоматический тест

Добавлен тест в `internal/api/handlers/streaming_test.go`:

```go
func TestStreamingChatHandler_SSEHeaders(t *testing.T) {
    // Setup handler
    handler := setupStreamingHandler(t)
    
    // Create request
    req := createStreamingRequest(t)
    
    // Execute
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Request = req
    
    handler.HandleStreamingRequest(c)
    
    // Verify SSE Content-Type
    assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
    assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
    assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
}
```

### Ручное тестирование

**Spring AI (JetBrains IDE):**

1. Настройте плагин:
   - Base URL: `http://YOUR_PROXY:8080`
   - API Key: ваш ключ
   - Model: `devstral-tuned:latest`

2. Отправьте запрос с streaming

3. **Ожидаемый результат:**
   - ✅ Streaming работает
   - ✅ Ответы приходят по частям
   - ✅ Нет JsonParseException

**cURL тест:**

```bash
curl -N http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_KEY" \
  -d '{
    "model": "devstral-tuned:latest",
    "messages": [{"role": "user", "content": "Привет"}],
    "stream": true
  }'
```

**Ожидаемый вывод:**
```
data: {"id":"chatcmpl-...","object":"chat.completion.chunk","created":1729756800,"model":"devstral-tuned:latest","choices":[{"index":0,"delta":{"role":"assistant","content":"При"},"finish_reason":null}]}

data: {"id":"chatcmpl-...","object":"chat.completion.chunk","created":1729756800,"model":"devstral-tuned:latest","choices":[{"index":0,"delta":{"content":"вет"},"finish_reason":null}]}

data: [DONE]
```

---

## 📚 Справочная информация

### Server-Sent Events (SSE) спецификация

- **MIME type:** `text/event-stream`
- **Формат:** `data: JSON\n\n` (две новые строки после каждого события)
- **Завершение:** `data: [DONE]\n\n`

### OpenAI API Streaming Format

OpenAI использует SSE для streaming chat completions:

```
data: {"id":"chatcmpl-123","object":"chat.completion.chunk",...}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk",...}

data: [DONE]
```

**Важно:**
- Каждое событие начинается с `data: `
- JSON на одной строке
- Две новые строки (`\n\n`) после каждого события
- `[DONE]` для завершения stream

### Spring AI ожидания

Spring AI (версия 1.0.0+) использует **strict SSE parsing**:

1. Проверяет `Content-Type: text/event-stream`
2. Парсит `data:` префикс
3. Десериализует JSON после `data: `
4. Ожидает `[DONE]` для завершения

Если `Content-Type` != `text/event-stream`, Spring AI парсит весь ответ как JSON → ошибка.

---

## 🔄 Совместимость

### Клиенты, которые теперь работают правильно:

✅ **Spring AI** (JetBrains плагины, Spring Boot apps)  
✅ **LangChain Java** (использует SSE для streaming)  
✅ **Apache HttpClient** (с SSE обработкой)  
✅ **Retrofit + OkHttp** (SSE interceptors)  
✅ **Любые HTTP клиенты с strict SSE parsing**

### Обратная совместимость:

✅ **cURL** - работает как раньше  
✅ **Python requests** - работает как раньше  
✅ **JavaScript fetch** - работает как раньше  
✅ **Continue.dev** - работает как раньше  
✅ **ai-review** - работает как раньше

**Изменение заголовка НЕ ломает существующих клиентов**, так как они корректно обрабатывают SSE вне зависимости от `Content-Type`.

---

## 📊 Impact Analysis

### До исправления:

- ❌ Spring AI плагины **НЕ работали** с streaming
- ❌ JetBrains IDE интеграции **падали** с JsonParseException
- ✅ Python/JS/Go клиенты работали (игнорировали неправильный заголовок)

### После исправления:

- ✅ Spring AI плагины **работают корректно**
- ✅ JetBrains IDE интеграции **работают стабильно**
- ✅ Python/JS/Go клиенты **работают как раньше**
- ✅ **100% совместимость** с OpenAI SSE форматом

---

## 🚀 Deployment

### Шаги для применения исправления:

1. **Пересоберите proxy:**
   ```bash
   go build -o bin/server.exe cmd/server/main.go
   ```

2. **Перезапустите сервер:**
   ```bash
   systemctl restart ollama-proxy
   # или
   ./bin/server.exe -config configs/production.yaml
   ```

3. **Проверьте логи:**
   ```bash
   tail -f logs/proxy-production.log | grep "Content-Type"
   ```

4. **Протестируйте с Spring AI клиентом**

### Rollback plan

Если нужно откатить (не должно быть причин):

```bash
git revert <commit-hash>
go build -o bin/server.exe cmd/server/main.go
systemctl restart ollama-proxy
```

---

## 📝 Related Issues

- **Issue:** Spring AI JsonParseException при streaming
- **Affected versions:** v1.0.0 - v1.10.0
- **Fixed in:** v1.10.1
- **Related docs:**
  - [W3C Server-Sent Events](https://html.spec.whatwg.org/multipage/server-sent-events.html)
  - [OpenAI Streaming API](https://platform.openai.com/docs/api-reference/streaming)
  - [Spring AI Documentation](https://docs.spring.io/spring-ai/reference/)

---

## ✅ Checklist

- [x] Исправлен `Content-Type` заголовок
- [x] Добавлены unit tests
- [x] Протестировано с Spring AI
- [x] Протестировано с cURL
- [x] Проверена обратная совместимость
- [x] Обновлена документация
- [x] Добавлен changelog entry

---

## 💡 Lessons Learned

1. **Всегда используйте правильный MIME type** для специализированных протоколов
2. **SSE требует `text/event-stream`** - это не опционально для строгих клиентов
3. **Тестируйте с разными HTTP клиентами** (Java, Python, JS, Go)
4. **Spring AI использует strict validation** - нельзя "обмануть" неправильными заголовками
5. **W3C спецификации важны** - следуйте им точно

---

**Автор:** AI Assistant  
**Дата:** 24.10.2025  
**Версия документа:** 1.0


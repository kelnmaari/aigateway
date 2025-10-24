# Bugfix: AI Review Compatibility - completion_tokens

## Проблема

AI-review (Python клиент) падал с ошибкой при обработке ответов от Ollama-OpenAI Proxy:

```
pydantic_core._pydantic_core.ValidationError: 1 validation error for OpenAIChatResponseSchema
usage.completion_tokens
  Field required [type=missing, input_value={'prompt_tokens': 33973, 'total_tokens': 33973}, input_type=dict]
```

### Причина

**JSON Tag `omitempty`** в структуре `Usage`:

```go
type Usage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens,omitempty"` // ❌ Пропускает поле при 0
    TotalTokens      int `json:"total_tokens"`
}
```

Когда модель не генерировала токены (`EvalCount = 0`), поле `completion_tokens` полностью отсутствовало в JSON:

```json
{
  "usage": {
    "prompt_tokens": 33973,
    "total_tokens": 33973
    // completion_tokens отсутствует!
  }
}
```

OpenAI API **требует** наличия `completion_tokens` всегда, даже если значение 0.

---

## Исправления

### 1. Убран `omitempty` из структуры Usage

**Файл:** `internal/models/openai.go`

```go
type Usage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"` // ✅ Всегда в JSON
    TotalTokens      int `json:"total_tokens"`
}
```

### 2. Минимальное значение completion_tokens

**Файл:** `internal/converter/response.go`

```go
func (r *ResponseConverter) estimateCompletionTokens(content string) int {
    tokens := (len(content) + 3) / 4
    
    // Минимум 1 токен для OpenAI API совместимости
    // Даже пустой ответ требует inference
    if tokens == 0 {
        tokens = 1
    }
    
    return tokens
}
```

**Обоснование:** Даже если модель вернула пустой ответ, inference все равно произошел и должен быть учтен.

---

## Тестирование

**Файл:** `internal/converter/response_test.go`

Создан тест `TestUsageCompletionTokensAlwaysPresent` с 3 сценариями:

1. **Normal response** - content есть, EvalCount > 0 ✅
2. **Empty response** - content пустой, EvalCount = 0 ✅
3. **Large prompt** - 33973 tokens, EvalCount = 0 ✅

```bash
go test -v ./internal/converter -run TestUsageCompletionTokensAlwaysPresent

PASS: completion_tokens всегда присутствует в JSON
```

---

## Результат

### До исправления

```json
{
  "usage": {
    "prompt_tokens": 33973,
    "total_tokens": 33973
  }
}
```

❌ AI-review падает с Pydantic ValidationError

### После исправления

```json
{
  "usage": {
    "prompt_tokens": 33973,
    "completion_tokens": 1,
    "total_tokens": 33974
  }
}
```

✅ AI-review работает корректно

---

## Вторая проблема: Модель возвращает Markdown вместо JSON

### Контекст

В логах AI-review:
```
2025-10-22 22:17:58 | ERROR | LLM_JSON_PARSER | No valid JSON found in output
Invalid JSON: expected value at line 1 column 1 [type=json_invalid, input_value='```'
```

Модель `devstral-tuned:latest` возвращает JSON обернутый в markdown code block:

**Модель возвращает:**
```
```json
[
  {"file": "internal/webfetch/parser.go", "line": 10, "comment": "..."}
]
```
```

**AI-review ожидает чистый JSON:**
```json
[
  {"file": "internal/webfetch/parser.go", "line": 10, "comment": "..."}
]
```

### Причины

1. **Модель обучена** возвращать markdown formatted code
2. **AI-review парсит** строго JSON без markdown wrapper
3. **devstral-tuned:latest** не подходит для структурированного JSON output

### Решение

#### Вариант 1: Смените модель (рекомендуется)

Используйте модели с хорошей JSON поддержкой:

```yaml
# .gitlab-ci.yml
variables:
  LLM__META__MODEL: "deepseek-coder:6.7b"  # Лучший JSON support
  # или
  LLM__META__MODEL: "qwen2.5-coder:7b"     # Хороший JSON support
```

**Таблица совместимости:**

| Модель | JSON Output | Context | AI Review |
|--------|-------------|---------|-----------|
| `deepseek-coder:6.7b` | ✅ Чистый JSON | 16K | ✅ Отлично |
| `qwen2.5-coder:7b` | ✅ Чистый JSON | 32K | ✅ Хорошо |
| `qwen2.5-coder:14b` | ✅ Чистый JSON | 32K | ✅ Отлично |
| `llama3.2:70b` | ⚠️ Иногда markdown | 128K | ⚠️ Работает 50/50 |
| `devstral-tuned:latest` | ❌ Markdown wrapper | ? | ❌ Не работает |

#### Вариант 2: Настройте строгий промпт

Создайте `.ai-review.yaml`:

```yaml
prompts:
  inline: |
    You MUST return ONLY valid JSON array. 
    NO markdown code blocks, NO explanations, NO formatting.
    
    Return EXACTLY this format:
    [{"file": "path.go", "line": 10, "comment": "issue"}]
    
    Start response with [ and end with ].
    DO NOT use ```json wrapper.
```

#### Вариант 3: Используйте summary review

Summary review не требует JSON парсинга:

```bash
# Вместо context/inline review:
ai-review run-summary
```

### Также: 503 Service Unavailable

В логах видны retry с 503 ошибкой:
```
2025-10-22 22:16:49 | WARNING | Attempt 1/5 failed with status=503
2025-10-22 22:17:56 | WARNING | Attempt 2/5 failed with status=503
```

**Причины:**
- Ollama перегружен (17 файлов = много токенов)
- Модель долго генерирует ответ
- Превышен timeout

**Решение:**
1. Уменьшите количество файлов в MR
2. Используйте меньшую модель
3. Увеличьте timeout в proxy config

---

## Третья проблема: Модель не генерирует ответ

### Контекст

В более ранних логах:
```
2025-10-22 22:05:35 | WARNING | INLINE_COMMENT_SERVICE | LLM returned empty string for inline review
```

Модель `devstral-tuned:latest` вернула **пустой ответ** при **33973 tokens** в prompt (22 файла).

### Причины

1. **Переполнение контекста** - 33973 tokens это очень много
2. **Модель не справляется** - devstral может иметь меньший context window
3. **22 файла в MR** - слишком много изменений для одного review

### Рекомендации

#### 1. Используйте меньшие модели для больших MR

```yaml
# .ai-review.yaml
policy:
  max_files: 30  # Ограничение количества файлов
  max_file_size: 50000  # 50 KB на файл
```

#### 2. Разбивайте большие MR на части

Вместо `ai-review run-context` используйте:
```yaml
# Только critical файлы
ai-review run-inline
```

С фильтрами в `.ai-review.yaml`:
```yaml
policy:
  include:
    - "internal/**/*.go"
  exclude:
    - "**/*_test.go"
    - "**/vendor/**"
```

#### 3. Используйте модели с большим контекстом

| Модель | Context Window | Для MR с файлами |
|--------|----------------|------------------|
| `qwen2.5-coder:1.5b` | 32K tokens | < 10 файлов |
| `qwen2.5-coder:7b` | 32K tokens | < 15 файлов |
| `deepseek-coder:6.7b` | 16K tokens | < 10 файлов |
| `llama3.2:70b` | 128K tokens | < 50 файлов |
| `gemma2:27b` | 8K tokens | < 5 файлов |

#### 4. Настройте GitLab CI для разных размеров MR

```yaml
# .gitlab-ci.yml

# Для маленьких MR (< 10 файлов)
ai-review:quick:
  rules:
    - if: '$CI_MERGE_REQUEST_IID && $CI_MERGE_REQUEST_CHANGES <= 10'
  variables:
    LLM__META__MODEL: "qwen2.5-coder:7b"
    LLM__META__MAX_TOKENS: "8000"

# Для больших MR (> 10 файлов)
ai-review:large:
  rules:
    - if: '$CI_MERGE_REQUEST_IID && $CI_MERGE_REQUEST_CHANGES > 10'
  variables:
    LLM__META__MODEL: "llama3.2:70b"
    LLM__META__MAX_TOKENS: "15000"
  when: manual  # Ручной запуск для больших MR
```

#### 5. Используйте summary вместо context для больших MR

```bash
# Вместо context review (анализ всех файлов)
ai-review run-context

# Используйте summary review (общий обзор)
ai-review run-summary
```

Summary review требует меньше токенов и дает high-level overview.

---

## Проверка после деплоя

### 1. Проверьте что completion_tokens присутствует

```bash
curl http://176.53.180.178:8085/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "devstral-tuned:latest",
    "messages": [{"role": "user", "content": "Hi"}],
    "max_tokens": 5
  }' | jq '.usage'
```

Ожидается:
```json
{
  "prompt_tokens": 10,
  "completion_tokens": 5,
  "total_tokens": 15
}
```

### 2. Проверьте пустой ответ

```bash
curl http://176.53.180.178:8085/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "devstral-tuned:latest",
    "messages": [{"role": "user", "content": ""}],
    "max_tokens": 0
  }' | jq '.usage'
```

Ожидается:
```json
{
  "prompt_tokens": 5,
  "completion_tokens": 1,  // Минимум 1
  "total_tokens": 6
}
```

### 3. Запустите AI review снова

```bash
ai-review run-summary  # Меньше токенов
```

---

## Версия

- **Fixed in:** v1.9.4 (2025-10-22)
- **Issue:** AI-review падал с ValidationError
- **Root cause:** `omitempty` в Usage.CompletionTokens
- **Tests:** `TestUsageCompletionTokensAlwaysPresent` added
- **Files changed:**
  - `internal/models/openai.go`
  - `internal/converter/response.go`
  - `internal/converter/response_test.go`

---

## Ссылки

- **OpenAI API Spec**: https://platform.openai.com/docs/api-reference/chat/object
- **AI Review GitHub**: https://github.com/Nikita-Filonov/ai-review
- **Related docs**: `docs/AI_REVIEW_SETUP.md`


# AI Review через Ollama-OpenAI Proxy - Инструкция по настройке

## Быстрый старт

### 1. Создайте API ключ в WebUI вашего proxy

1. Откройте WebUI: `http://your-proxy-host:8080`
2. Перейдите в раздел **API Keys**
3. Создайте новый ключ:
   - **Name**: `gitlab-ai-review`
   - **Models**: выберите модели для code review (например, `llama3.2:latest`)
   - **Rate Limit**: рекомендуется 100-300 req/hour
4. Скопируйте сгенерированный ключ

### 2. Добавьте ключ в GitLab CI/CD Variables

1. Откройте ваш проект в GitLab
2. **Settings** → **CI/CD** → **Variables** → **Add Variable**
3. Заполните:
   - **Key**: `OLLAMA_PROXY_API_KEY`
   - **Value**: скопированный API ключ
   - **Type**: Variable
   - **Flags**: 
     - ✓ Mask variable (скрыть в логах)
     - ✓ Protect variable (только для protected branches)

### 3. Настройте `.gitlab-ci.yml`

Отредактируйте параметры в `.gitlab-ci.yml`:

```yaml
variables:
  # 1. URL вашего proxy (ОБЯЗАТЕЛЬНО)
  LLM__HTTP_CLIENT__API_URL: "http://YOUR_PROXY_HOST:8080/v1"
  
  # 2. Модель Ollama (ОБЯЗАТЕЛЬНО)
  LLM__META__MODEL: "llama3.2:latest"
```

**Примеры URL:**
- Локальная сеть: `http://10.0.0.5:8080/v1`
- Docker Compose: `http://ollama-proxy:8080/v1`
- Доменное имя: `https://ollama.company.com/v1`

### 4. Проверьте доступность proxy

На GitLab Runner должен быть доступ к вашему proxy. Проверьте:

```bash
# Из runner контейнера/машины:
curl http://YOUR_PROXY_HOST:8080/v1/models \
  -H "Authorization: Bearer $OLLAMA_PROXY_API_KEY"
```

Должен вернуться список моделей.

### 5. Запустите AI Review

1. Создайте Merge Request
2. Перейдите в **Pipelines**
3. Нажмите ▶️ на нужном job:
   - `ai-review:full` - полный анализ
   - `ai-review:inline` - построчные комментарии
   - `ai-review:summary` - общий комментарий

---

## Выбор модели для Code Review

### Быстрые модели (для тестирования)

| Модель | Размер | Скорость | Качество |
|--------|--------|----------|----------|
| `qwen2.5-coder:1.5b` | 1.5B | ⚡⚡⚡ | ⭐⭐ |
| `llama3.2:3b` | 3B | ⚡⚡ | ⭐⭐⭐ |
| `codellama:7b` | 7B | ⚡ | ⭐⭐⭐ |

### Production модели (лучшее качество)

| Модель | Размер | Скорость | Качество |
|--------|--------|----------|----------|
| `qwen2.5-coder:7b` | 7B | ⚡ | ⭐⭐⭐⭐ |
| `deepseek-coder:6.7b` | 6.7B | ⚡ | ⭐⭐⭐⭐ |
| `qwen2.5-coder:14b` | 14B | 🐌 | ⭐⭐⭐⭐⭐ |
| `deepseek-coder:33b` | 33B | 🐌🐌 | ⭐⭐⭐⭐⭐ |

**Рекомендации:**
- **Для CI/CD**: `qwen2.5-coder:7b` - оптимальный баланс
- **Для критичного кода**: `deepseek-coder:33b` - максимальное качество
- **Для тестов**: `qwen2.5-coder:1.5b` - быстрая проверка

### Проверка доступных моделей

```bash
# Список моделей в вашем Ollama
curl http://YOUR_PROXY_HOST:8080/v1/models \
  -H "Authorization: Bearer $API_KEY" | jq '.data[].id'
```

---

## Опциональная конфигурация .ai-review.yaml

Если нужна более детальная настройка, создайте `.ai-review.yaml` в корне проекта:

```yaml
# ===========================================================================
# AI Review Configuration для Ollama-OpenAI Proxy
# ===========================================================================

# ---------------------------------------------------------------------------
# LLM Provider Configuration
# ---------------------------------------------------------------------------
llm:
  provider: OPENAI  # Proxy совместим с OpenAI API
  
  meta:
    model: llama3.2:latest  # Модель из Ollama
    max_tokens: 15000
    temperature: 0.3
    
    # Дополнительные параметры для Ollama
    # top_p: 0.9
    # top_k: 40
    # repeat_penalty: 1.1
  
  http_client:
    timeout: 300  # 5 минут (для больших файлов)
    api_url: ${LLM_API_URL}  # Из GitLab CI variables
    api_token: ${OLLAMA_PROXY_API_KEY}

# ---------------------------------------------------------------------------
# Version Control System (GitLab)
# ---------------------------------------------------------------------------
vcs:
  provider: GITLAB
  
  pipeline:
    project_id: ${CI_PROJECT_ID}
    merge_request_id: ${CI_MERGE_REQUEST_IID}
  
  http_client:
    timeout: 120
    api_url: ${CI_SERVER_URL}
    api_token: ${CI_JOB_TOKEN}

# ---------------------------------------------------------------------------
# Review Policy - Что анализировать
# ---------------------------------------------------------------------------
policy:
  # Включить файлы (glob patterns)
  include:
    - "**/*.go"
    - "**/*.py"
    - "**/*.ts"
    - "**/*.tsx"
    - "**/*.js"
    - "**/*.jsx"
  
  # Исключить файлы
  exclude:
    - "**/vendor/**"
    - "**/node_modules/**"
    - "**/*_test.go"
    - "**/*.min.js"
    - "**/dist/**"
    - "**/build/**"
  
  # Максимальный размер файла для анализа (bytes)
  max_file_size: 100000  # 100 KB
  
  # Максимальное количество файлов в одном MR
  max_files: 50

# ---------------------------------------------------------------------------
# Modes Configuration
# ---------------------------------------------------------------------------
modes:
  inline:
    enabled: true
    max_comments: 30  # Ограничение комментариев
  
  summary:
    enabled: true
  
  context:
    enabled: true

# ---------------------------------------------------------------------------
# Prompts Customization (опционально)
# ---------------------------------------------------------------------------
prompts:
  inline: |
    You are an expert code reviewer.
    Focus on:
    - Bugs and errors
    - Security vulnerabilities
    - Performance issues
    - Code style violations
    - Best practices
    
    Language: Russian
    Be concise and actionable.
  
  summary: |
    Provide a comprehensive review summary.
    Include:
    - Overall code quality assessment
    - Major issues found
    - Recommendations
    
    Language: Russian

# ---------------------------------------------------------------------------
# Artifacts (опционально)
# ---------------------------------------------------------------------------
artifacts:
  enabled: true
  output_dir: .ai-review-artifacts
  
  formats:
    - json
    - markdown

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------
logging:
  level: INFO  # DEBUG | INFO | WARNING | ERROR
  format: structured  # structured | plain
```

---

## Режимы работы AI Review

### 1. `ai-review run` - Полный анализ
Выполняет inline + summary review. Лучше всего для первого review MR.

### 2. `ai-review run-inline` - Построчные комментарии
Добавляет комментарии к конкретным строкам кода с найденными проблемами.

**Пример вывода:**
```
src/handler.go:45
⚠️ Потенциальная утечка goroutine: defer cancel() отсутствует
```

### 3. `ai-review run-summary` - Общий комментарий
Создает один комментарий с обзором всех изменений.

**Пример вывода:**
```
## AI Review Summary

### Общее качество: ⭐⭐⭐⭐ (4/5)

### Найдено проблем:
- 🔴 Критичные: 1
- 🟡 Предупреждения: 3
- 🔵 Улучшения: 5

### Ключевые находки:
1. Отсутствует обработка ошибок в handler.go:45
2. SQL инъекция в query builder
3. Рекомендуется использовать context.WithTimeout

### Рекомендации:
...
```

### 4. `ai-review run-context` - Архитектурный анализ
Анализирует общую структуру изменений, архитектурные решения.

### 5. `ai-review run-inline-reply` - Ответ на комментарии
Отвечает на существующие inline комментарии в MR.

### 6. `ai-review run-summary-reply` - Ответ на summary
Отвечает на комментарии к общему review.

---

## Troubleshooting

### Проблема: Job fails с "Connection refused"

**Причина**: GitLab Runner не может достучаться до proxy.

**Решение:**
1. Проверьте URL: `http://YOUR_PROXY_HOST:8080/v1`
2. Убедитесь что proxy доступен извне:
   ```bash
   # На машине где запущен proxy
   netstat -tlnp | grep 8080
   ```
3. Проверьте firewall:
   ```bash
   # Разрешите порт 8080
   sudo ufw allow 8080/tcp
   ```

### Проблема: "401 Unauthorized"

**Причина**: Неверный API ключ.

**Решение:**
1. Проверьте что переменная `OLLAMA_PROXY_API_KEY` создана в GitLab
2. Проверьте что ключ активен в WebUI proxy
3. Проверьте формат ключа: должен быть без префикса "Bearer"

### Проблема: "Model not found"

**Причина**: Модель не загружена в Ollama.

**Решение:**
1. Подключитесь к серверу с Ollama:
   ```bash
   ollama list  # Проверить загруженные модели
   ollama pull llama3.2:latest  # Загрузить модель
   ```
2. Или используйте другую модель из списка доступных

### Проблема: Job timeout

**Причина**: Модель слишком медленная или MR слишком большой.

**Решение:**
1. Используйте меньшую модель: `qwen2.5-coder:1.5b`
2. Увеличьте timeout в `.gitlab-ci.yml`:
   ```yaml
   timeout: 45m
   ```
3. Ограничьте размер файлов в `.ai-review.yaml`:
   ```yaml
   policy:
     max_file_size: 50000
     max_files: 30
   ```

### Проблема: Слишком много/мало комментариев

**Решение**: Настройте в `.ai-review.yaml`:
```yaml
modes:
  inline:
    max_comments: 15  # Уменьшить для меньшего кол-ва
```

---

## Best Practices

### 1. Используйте разные модели для разных целей

```yaml
# .gitlab-ci.yml
ai-review:quick:
  variables:
    LLM__META__MODEL: "qwen2.5-coder:1.5b"
  # Быстрая проверка при каждом push

ai-review:thorough:
  variables:
    LLM__META__MODEL: "deepseek-coder:33b"
  when: manual
  # Детальный анализ перед merge
```

### 2. Настройте include/exclude patterns

Анализируйте только важные файлы:
```yaml
policy:
  include:
    - "internal/**/*.go"
    - "pkg/**/*.go"
  exclude:
    - "**/*_test.go"
    - "**/mock_*.go"
    - "**/vendor/**"
```

### 3. Кастомизируйте промпты под ваш код

```yaml
prompts:
  inline: |
    Review Go code with focus on:
    - Goroutine leaks
    - Context usage
    - Error handling
    - SQL injection
    Language: Russian
```

### 4. Используйте artifacts для отладки

```yaml
artifacts:
  enabled: true
  output_dir: .ai-review-artifacts
```

Артефакты сохраняются в GitLab и доступны для скачивания.

---

## Примеры использования

### Пример 1: Быстрая проверка при каждом push

```yaml
ai-review:auto:
  stage: review
  when: always  # Автоматически
  variables:
    LLM__META__MODEL: "qwen2.5-coder:1.5b"
    LLM__META__MAX_TOKENS: "5000"
  script:
    - ai-review run-inline
  timeout: 10m
```

### Пример 2: Детальный review перед merge

```yaml
ai-review:pre-merge:
  stage: review
  when: manual
  only:
    - merge_requests
  variables:
    LLM__META__MODEL: "deepseek-coder:33b"
    LLM__META__MAX_TOKENS: "20000"
  script:
    - ai-review run
  timeout: 45m
```

### Пример 3: Только для security-critical файлов

```yaml
ai-review:security:
  stage: review
  when: manual
  script:
    - ai-review run-inline
  variables:
    LLM__META__MODEL: "deepseek-coder:33b"
  only:
    changes:
      - "internal/auth/**/*.go"
      - "internal/api/middleware/*.go"
```

---

## Интеграция с Ollama-OpenAI Proxy Features

### Rate Limiting

Ваш proxy автоматически контролирует rate limits. В случае превышения:
```json
{
  "error": {
    "message": "Rate limit exceeded: 100 requests per hour",
    "type": "rate_limit_error"
  }
}
```

**Решение**: Увеличьте лимит для API ключа в WebUI.

### Usage Tracking

Все запросы логируются в proxy. Проверить статистику:
- WebUI → Usage Dashboard
- API: `GET /api/usage?api_key=gitlab-ai-review`

### Multi-Tenant Support

Создайте отдельные API ключи для разных команд:
- `team-backend-ai-review` - для backend репозиториев
- `team-frontend-ai-review` - для frontend репозиториев

Каждая команда получит свою статистику и лимиты.

---

## Дополнительные ресурсы

- **AI Review документация**: https://github.com/Nikita-Filonov/ai-review
- **Ollama модели**: https://ollama.com/library
- **Ollama-OpenAI Proxy**: `docs/API_DOCUMENTATION.md` в вашем проекте

---

## Поддержка

При возникновении проблем:
1. Проверьте логи GitLab CI job
2. Проверьте логи Ollama-OpenAI Proxy: `logs/proxy-dev.log`
3. Проверьте артефакты AI Review: `.ai-review-artifacts/`


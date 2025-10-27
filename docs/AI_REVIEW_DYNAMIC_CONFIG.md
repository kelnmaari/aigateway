# AI Review - Динамическая конфигурация

## Обзор

`.ai-review.yaml` **генерируется автоматически** в GitLab CI pipeline через скрипт `scripts/generate-ai-review-config.sh`. Это позволяет:

1. **Централизованное управление** - вся конфигурация в `.gitlab-ci.yml`
2. **Динамические параметры** - разные настройки для разных условий
3. **Не нужно коммитить** `.ai-review.yaml` в репозиторий
4. **Переиспользование** - один template для всех jobs

---

## Как это работает

### 1. Hidden Job `.generate-config`

```yaml
.generate-config:
  before_script:
    - bash scripts/generate-ai-review-config.sh
```

Каждый AI review job наследует этот template:

```yaml
ai-review:full:
  extends: .generate-config  # ← Автогенерация конфигурации
  script:
    - ai-review run
```

### 2. Скрипт генерации

`scripts/generate-ai-review-config.sh` читает environment variables и создает `.ai-review.yaml`:

```bash
# Обязательные переменные (из .gitlab-ci.yml)
LLM__HTTP_CLIENT__API_URL
OLLAMA_PROXY_API_KEY
LLM__META__MODEL
GITLAB_API_TOKEN

# Опциональные переменные (с defaults)
AI_REVIEW_MAX_FILES=10
AI_REVIEW_MAX_FILE_SIZE=150000
AI_REVIEW_MAX_COMMENTS=30
```

### 3. Генерация `.ai-review.yaml`

Скрипт создает полный конфигурационный файл с:

- LLM settings (model, tokens, temperature)
- VCS integration (GitLab API)
- Review policy (include/exclude, limits)
- Prompts (Go-specific)

---

## Переменные конфигурации

### LLM Configuration

| Переменная | Описание | Default | Используется в |
|------------|----------|---------|----------------|
| `LLM__HTTP_CLIENT__API_URL` | URL proxy | **Required** | `.gitlab-ci.yml` |
| `OLLAMA_PROXY_API_KEY` | API ключ | **Required** | CI Variables |
| `LLM__META__MODEL` | Модель Ollama | deepseek-coder:6.7b | `.gitlab-ci.yml` |
| `LLM__META__MAX_TOKENS` | Max tokens | 15000 | `.gitlab-ci.yml` |
| `LLM__META__TEMPERATURE` | Temperature | 0.3 | `.gitlab-ci.yml` |

### VCS Configuration

| Переменная | Описание | Default | Используется в |
|------------|----------|---------|----------------|
| `GITLAB_API_TOKEN` | Access Token | **Required** | CI Variables |
| `CI_PROJECT_ID` | Project ID | Auto | GitLab CI |
| `CI_MERGE_REQUEST_IID` | MR number | Auto | GitLab CI |
| `CI_SERVER_URL` | GitLab URL | Auto | GitLab CI |

### Review Policy

| Переменная | Описание | Default | Настройка |
|------------|----------|---------|-----------|
| `AI_REVIEW_MAX_FILES` | Максимум файлов | 10 | `.gitlab-ci.yml` |
| `AI_REVIEW_MAX_FILE_SIZE` | Размер файла (bytes) | 150000 | `.gitlab-ci.yml` |
| `AI_REVIEW_MAX_COMMENTS` | Максимум комментариев | 30 | `.gitlab-ci.yml` |

---

## Примеры использования

### Пример 1: Разные настройки для разных размеров MR

```yaml
# Маленькие MR (< 500 строк)
ai-review:small:
  extends: .generate-config
  stage: review
  image: nikitafilonov/ai-review:latest
  when: manual
  rules:
    - if: '$CI_MERGE_REQUEST_IID && $CI_MERGE_REQUEST_DIFF_SIZE < 500'
  script:
    - ai-review run  # Full review
  variables:
    LLM__META__MODEL: "deepseek-coder:6.7b"  # Быстрая модель
    AI_REVIEW_MAX_FILES: "15"
    AI_REVIEW_MAX_COMMENTS: "50"
    LLM__META__MAX_TOKENS: "10000"

# Большие MR (>= 500 строк)
ai-review:large:
  extends: .generate-config
  stage: review
  image: nikitafilonov/ai-review:latest
  when: manual
  rules:
    - if: '$CI_MERGE_REQUEST_IID && $CI_MERGE_REQUEST_DIFF_SIZE >= 500'
  script:
    - ai-review run-summary  # Только summary для больших MR
  variables:
    LLM__META__MODEL: "qwen2.5-coder:7b"
    AI_REVIEW_MAX_FILES: "5"
    AI_REVIEW_MAX_COMMENTS: "20"
    LLM__META__MAX_TOKENS: "8000"
```

### Пример 2: Разные модели для разных типов файлов

```yaml
# Review Go кода
ai-review:go:
  extends: .generate-config
  when: manual
  rules:
    - if: '$CI_MERGE_REQUEST_IID'
      changes:
        - "**/*.go"
  script:
    - ai-review run-inline
  variables:
    LLM__META__MODEL: "deepseek-coder:33b"  # Лучшая для Go
    AI_REVIEW_MAX_FILES: "8"

# Review конфигурации
ai-review:config:
  extends: .generate-config
  when: manual
  rules:
    - if: '$CI_MERGE_REQUEST_IID'
      changes:
        - "**/*.yaml"
        - "**/*.yml"
        - "**/*.json"
  script:
    - ai-review run-summary
  variables:
    LLM__META__MODEL: "qwen2.5-coder:7b"
    AI_REVIEW_MAX_FILES: "20"
```

### Пример 3: Автоматический review для маленьких MR

```yaml
# Автоматически для MR < 5 файлов
ai-review:auto:
  extends: .generate-config
  when: always  # Автоматически
  rules:
    - if: '$CI_MERGE_REQUEST_IID && $CI_MERGE_REQUEST_CHANGES < 5'
  script:
    - ai-review run-summary
  variables:
    LLM__META__MODEL: "deepseek-coder:6.7b"
    AI_REVIEW_MAX_FILES: "5"
    AI_REVIEW_MAX_COMMENTS: "15"

# Ручной для больших MR
ai-review:manual:
  extends: .generate-config
  when: manual
  rules:
    - if: '$CI_MERGE_REQUEST_IID && $CI_MERGE_REQUEST_CHANGES >= 5'
  script:
    - ai-review run-summary
  variables:
    LLM__META__MODEL: "deepseek-coder:33b"
    AI_REVIEW_MAX_FILES: "10"
```

### Пример 4: Разные настройки по времени суток

```yaml
# Днем - быстрая модель
ai-review:daytime:
  extends: .generate-config
  when: manual
  rules:
    - if: '$CI_MERGE_REQUEST_IID && $CI_COMMIT_HOUR >= 9 && $CI_COMMIT_HOUR < 18'
  variables:
    LLM__META__MODEL: "deepseek-coder:6.7b"
    AI_REVIEW_MAX_FILES: "12"

# Ночью - мощная модель (меньше нагрузка)
ai-review:nighttime:
  extends: .generate-config
  when: manual
  rules:
    - if: '$CI_MERGE_REQUEST_IID && ($CI_COMMIT_HOUR < 9 || $CI_COMMIT_HOUR >= 18)'
  variables:
    LLM__META__MODEL: "deepseek-coder:33b"
    AI_REVIEW_MAX_FILES: "15"
```

---

## Кастомизация скрипта генерации

### Добавление новых переменных

Если нужно добавить новые параметры:

1. **Добавьте в скрипт** `scripts/generate-ai-review-config.sh`:

```bash
# Добавьте после существующих defaults
CUSTOM_PARAM="${AI_REVIEW_CUSTOM_PARAM:-default_value}"
```

2. **Используйте в template** `.ai-review.yaml`:

```yaml
policy:
  custom_setting: ${CUSTOM_PARAM}
```

3. **Установите в** `.gitlab-ci.yml`:

```yaml
variables:
  AI_REVIEW_CUSTOM_PARAM: "my_value"
```

### Изменение промптов динамически

Можно создать разные промпты для разных scenarios:

```bash
# В скрипте генерации
if [ "$AI_REVIEW_MODE" = "strict" ]; then
    PROMPT="Be very strict and thorough..."
elif [ "$AI_REVIEW_MODE" = "gentle" ]; then
    PROMPT="Be constructive and encouraging..."
else
    PROMPT="Standard review..."
fi

# В template
prompts:
  inline: |
    ${PROMPT}
```

Использование:

```yaml
ai-review:strict:
  variables:
    AI_REVIEW_MODE: "strict"
    
ai-review:gentle:
  variables:
    AI_REVIEW_MODE: "gentle"
```

---

## Отладка

### Проверка сгенерированного конфигурации

Добавьте в job:

```yaml
script:
  - bash scripts/generate-ai-review-config.sh
  - cat .ai-review.yaml  # Вывод конфигурации в логи
  - ai-review run
```

### Сохранение конфигурации как artifact

```yaml
ai-review:full:
  extends: .generate-config
  artifacts:
    paths:
      - .ai-review.yaml
    when: always
    expire_in: 1 day
```

Скачайте artifact из GitLab CI для проверки.

### Тестирование локально

```bash
# Установите ОБЯЗАТЕЛЬНЫЕ переменные
export LLM__HTTP_CLIENT__API_URL="http://localhost:8085/v1"
export OLLAMA_PROXY_API_KEY="sk-test"
export LLM__META__MODEL="deepseek-coder:6.7b"
export GITLAB_API_TOKEN="glpat-xxx"
export CI_PROJECT_ID="146"
export CI_MERGE_REQUEST_IID="1"
export CI_SERVER_URL="https://gitlab.com"

# Опциональные переменные (с defaults)
export AI_REVIEW_MAX_FILES="10"
export AI_REVIEW_MAX_FILE_SIZE="150000"
export AI_REVIEW_MAX_COMMENTS="30"
export LLM__META__MAX_TOKENS="15000"
export LLM__META__TEMPERATURE="0.3"

# Сгенерируйте конфигурацию
bash scripts/generate-ai-review-config.sh

# Проверьте результат
cat .ai-review.yaml

# Запустите review
ai-review run-summary
```

**Проверка всех переменных:**

```bash
# Скрипт покажет все установленные значения:
bash scripts/generate-ai-review-config.sh

# Вывод:
# LLM Settings:
#   API URL: http://localhost:8085/v1
#   Model: deepseek-coder:6.7b
#   Max Tokens: 15000
#   Temperature: 0.3
#
# GitLab Settings:
#   Server: https://gitlab.com
#   Project: 146
#   MR: 1
#
# Review Policy:
#   Max Files: 10
#   Max File Size: 150000 bytes
#   Max Comments: 30
```

---

## Best Practices

### 1. Используйте разумные лимиты

```yaml
# Плохо: слишком много файлов
AI_REVIEW_MAX_FILES: "50"  # Timeout!

# Хорошо: реалистичные лимиты
AI_REVIEW_MAX_FILES: "10"  # Стабильно работает
```

### 2. Подбирайте модель под задачу

```yaml
# Быстрые модели для маленьких MR
LLM__META__MODEL: "deepseek-coder:6.7b"

# Мощные модели для критичного кода
LLM__META__MODEL: "deepseek-coder:33b"
```

### 3. Используйте conditions в rules

```yaml
rules:
  # Только для Go файлов
  - if: '$CI_MERGE_REQUEST_IID'
    changes:
      - "**/*.go"
  
  # Только для маленьких MR
  - if: '$CI_MERGE_REQUEST_DIFF_SIZE < 300'
  
  # Только для protected branches
  - if: '$CI_COMMIT_REF_PROTECTED == "true"'
```

### 4. Тестируйте перед production

Создайте test job:

```yaml
ai-review:test:
  extends: .generate-config
  when: manual
  only:
    - branches
  except:
    - main
    - master
  variables:
    AI_REVIEW_MAX_FILES: "3"  # Ограничьте для теста
```

---

## Troubleshooting

### Проблема: "Input should be a valid URL" для `${CI_SERVER_URL}`

```
pydantic_core._pydantic_core.ValidationError: 1 validation error for Settings
vcs.GITLAB.http_client.api_url
  Input should be a valid URL [type=url_parsing, input_value='${CI_SERVER_URL}', input_type=str]
```

**Причина:** Переменные GitLab CI не установлены или не подставляются.

**Решение 1:** Проверьте переменные environment:

```bash
# Перед запуском ai-review
echo "CI_SERVER_URL: $CI_SERVER_URL"
echo "CI_PROJECT_ID: $CI_PROJECT_ID"
echo "CI_MERGE_REQUEST_IID: $CI_MERGE_REQUEST_IID"
echo "GITLAB_API_TOKEN: ${GITLAB_API_TOKEN:0:10}..."
```

**Решение 2:** Установите переменные локально:

```bash
export CI_SERVER_URL="https://gitlab.alexue4.dev"
export CI_PROJECT_ID="146"
export CI_MERGE_REQUEST_IID="1"
export GITLAB_API_TOKEN="glpat-your-token"

# Перегенерируйте конфигурацию
bash scripts/generate-ai-review-config.sh
```

**Решение 3:** Проверьте сгенерированный файл:

```bash
cat .ai-review.yaml | grep -A5 "vcs:"

# Должно быть (STRING в кавычках):
# vcs:
#   provider: GITLAB
#   pipeline:
#     project_id: "146"        # ← Строка в кавычках!
#     merge_request_id: "1"    # ← Строка в кавычках!
#   http_client:
#     api_url: https://gitlab.alexue4.dev
#
# НЕ должно быть:
#   project_id: 146            # ← Число без кавычек - ОШИБКА!
#   api_url: ${CI_SERVER_URL}  # ← Литерал - ОШИБКА!
```

### Проблема: "Input should be a valid string" для project_id

```
pydantic_core._pydantic_core.ValidationError: 2 validation errors for Settings
vcs.GITLAB.pipeline.project_id
  Input should be a valid string [type=string_type, input_value=146, input_type=int]
vcs.GITLAB.pipeline.merge_request_id
  Input should be a valid string [type=string_type, input_value=1, input_type=int]
```

**Причина:** В YAML числа без кавычек парсятся как integers, а `ai-review` ожидает strings.

**Решение:** Проверьте сгенерированный `.ai-review.yaml`:

```bash
cat .ai-review.yaml | grep -A3 "pipeline:"

# ПРАВИЛЬНО (с кавычками):
# pipeline:
#   project_id: "146"
#   merge_request_id: "1"

# НЕПРАВИЛЬНО (без кавычек):
# pipeline:
#   project_id: 146    # ← Integer, не String!
#   merge_request_id: 1
```

**Фикс:** Скрипт должен генерировать значения в кавычках:

```bash
# В scripts/generate-ai-review-config.sh
project_id: "${CI_PROJECT_ID}"          # ← С кавычками
merge_request_id: "${CI_MERGE_REQUEST_IID}"  # ← С кавычками
```

### Проблема: "ERROR: Required variable not set"

```
ERROR: CI_PROJECT_ID is required
```

**Решение:**

В GitLab CI эти переменные устанавливаются автоматически. Локально:

```bash
# Узнайте значения из GitLab:
# Project: https://gitlab.com/user/project
#   → CI_PROJECT_ID="123" (из Settings → General)
# MR: https://gitlab.com/user/project/-/merge_requests/5
#   → CI_MERGE_REQUEST_IID="5"

export CI_SERVER_URL="https://gitlab.alexue4.dev"
export CI_PROJECT_ID="146"
export CI_MERGE_REQUEST_IID="1"
```

### Проблема: Скрипт генерации не найден

```
bash: scripts/generate-ai-review-config.sh: No such file or directory
```

**Решение:**

1. Убедитесь что скрипт закоммичен в репозиторий
2. Проверьте права: `chmod +x scripts/generate-ai-review-config.sh`

### Проблема: Переменные не подставляются

**Проверьте:**

1. Переменные установлены в `.gitlab-ci.yml` → `variables`
2. Переменные не в кавычках в скрипте: `${VAR}` не `"${VAR}"`
3. Escaping в heredoc: используйте `\${VAR}` для литералов

### Проблема: Конфигурация пустая или неполная

**Отладка:**

```yaml
script:
  - bash -x scripts/generate-ai-review-config.sh  # Debug mode
  - wc -l .ai-review.yaml  # Проверка размера
  - head -n 20 .ai-review.yaml  # Первые строки
```

---

## Ссылки

- **Скрипт**: `scripts/generate-ai-review-config.sh`
- **CI конфигурация**: `.gitlab-ci.yml`
- **Template пример**: `.ai-review.yaml.example`
- **Документация**: `docs/AI_REVIEW_SETUP.md`

---

**Version:** 1.0  
**Last Updated:** 2025-10-22


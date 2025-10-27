# Embeddings Configuration Guide

## 📋 Обзор

AIGateway Platform поддерживает **embeddings** (векторные представления) для текстов через `/v1/embeddings` endpoint. Начиная с версии **v1.10.1**, вы можете настроить **дефолтную модель** для embeddings в конфигурации.

---

## 🎯 Зачем нужна дефолтная модель?

### Проблема

Многие клиенты (LangChain, LlamaIndex, OpenAI SDK) **hardcode** модели OpenAI в коде:

```python
# Python OpenAI SDK
embeddings = openai.Embedding.create(
    model="text-embedding-ada-002",  # Hardcoded OpenAI модель
    input="Hello, world!"
)
```

Без дефолтной модели proxy **не знает**, какую Ollama модель использовать для `text-embedding-ada-002`.

### Решение

Настройте **default_embedding_model** в конфигурации:

```yaml
models:
  default_embedding_model: "nomic-embed-text:latest"
```

Теперь все запросы с OpenAI моделями будут **автоматически** использовать вашу Ollama модель!

---

## ⚙️ Конфигурация

### Файл: `configs/dev.yaml` / `configs/production.yaml`

```yaml
models:
  # Модель по умолчанию для embeddings
  # Используется когда:
  # - Не указана модель в запросе
  # - Указана OpenAI модель (text-embedding-*)
  #
  # Рекомендуемые модели:
  # - nomic-embed-text:latest (768 dimensions, высокое качество)
  # - mxbai-embed-large:latest (1024 dimensions, лучшее качество)
  # - all-minilm:latest (384 dimensions, быстрая)
  default_embedding_model: "nomic-embed-text:latest"
```

### Environment Variable

```bash
export PROXY_MODELS_DEFAULT_EMBEDDING_MODEL="nomic-embed-text:latest"
```

---

## 📚 Логика выбора модели

Proxy использует следующую логику при обработке `/v1/embeddings`:

```
1. Модель НЕ указана в запросе?
   → Использовать default_embedding_model

2. Модель начинается с "text-embedding-"? (OpenAI модель)
   → Использовать default_embedding_model

3. Модель - это Ollama модель?
   → Использовать как есть
```

### Примеры

#### Пример 1: Пустая модель
```json
{
  "input": "Hello, world!"
  // model не указана
}
```
**Результат:** Использует `default_embedding_model`

#### Пример 2: OpenAI модель
```json
{
  "model": "text-embedding-ada-002",
  "input": "Hello, world!"
}
```
**Результат:** Использует `default_embedding_model` (заменяет OpenAI модель)

#### Пример 3: Ollama модель
```json
{
  "model": "nomic-embed-text:latest",
  "input": "Hello, world!"
}
```
**Результат:** Использует `nomic-embed-text:latest` (как есть)

#### Пример 4: Custom Ollama модель
```json
{
  "model": "my-custom-embedding:v2",
  "input": "Hello, world!"
}
```
**Результат:** Использует `my-custom-embedding:v2` (как есть)

---

## 🔧 Рекомендуемые модели

### 1. nomic-embed-text (Рекомендуется для большинства случаев)

```bash
ollama pull nomic-embed-text
```

**Характеристики:**
- **Dimensions:** 768
- **Context Window:** 2048 tokens
- **Качество:** Высокое (лучше чем OpenAI ada-002 в некоторых бенчмарках)
- **Скорость:** Быстрая
- **Use cases:** Общего назначения, RAG, семантический поиск

**Конфигурация:**
```yaml
models:
  default_embedding_model: "nomic-embed-text:latest"
```

---

### 2. mxbai-embed-large (Высокое качество)

```bash
ollama pull mxbai-embed-large
```

**Характеристики:**
- **Dimensions:** 1024
- **Context Window:** 512 tokens
- **Качество:** Очень высокое
- **Скорость:** Средняя
- **Use cases:** Высокая точность, domain-specific tasks

**Конфигурация:**
```yaml
models:
  default_embedding_model: "mxbai-embed-large:latest"
```

---

### 3. all-minilm (Максимальная скорость)

```bash
ollama pull all-minilm
```

**Характеристики:**
- **Dimensions:** 384
- **Context Window:** 256 tokens
- **Качество:** Среднее
- **Скорость:** Очень быстрая
- **Use cases:** Real-time embeddings, высокий throughput

**Конфигурация:**
```yaml
models:
  default_embedding_model: "all-minilm:latest"
```

---

## 📊 Сравнение моделей

| Модель | Dimensions | Context | Качество | Скорость | Размер |
|--------|-----------|---------|----------|----------|--------|
| **nomic-embed-text** | 768 | 2048 | ⭐⭐⭐⭐⭐ | ⚡⚡⚡⚡ | 274 MB |
| **mxbai-embed-large** | 1024 | 512 | ⭐⭐⭐⭐⭐ | ⚡⚡⚡ | 670 MB |
| **all-minilm** | 384 | 256 | ⭐⭐⭐ | ⚡⚡⚡⚡⚡ | 45 MB |
| **OpenAI ada-002** | 1536 | 8191 | ⭐⭐⭐⭐ | ⚡⚡ | Cloud |

---

## 🧪 Тестирование

### cURL тест

```bash
# Без модели (использует default)
curl http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "input": "Hello, world!"
  }'

# OpenAI модель (заменяется на default)
curl http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "text-embedding-ada-002",
    "input": "Hello, world!"
  }'

# Ollama модель (используется как есть)
curl http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "mxbai-embed-large:latest",
    "input": "Hello, world!"
  }'
```

### Python OpenAI SDK

```python
import openai

openai.api_base = "http://localhost:8080/v1"
openai.api_key = "YOUR_API_KEY"

# Будет использована default_embedding_model
response = openai.Embedding.create(
    model="text-embedding-ada-002",  # Заменяется автоматически
    input="Hello, world!"
)

print(response['data'][0]['embedding'])
```

### LangChain

```python
from langchain.embeddings import OpenAIEmbeddings

# Настройка для proxy
embeddings = OpenAIEmbeddings(
    openai_api_base="http://localhost:8080/v1",
    openai_api_key="YOUR_API_KEY",
    model="text-embedding-ada-002"  # Заменится на default_embedding_model
)

# Использование
vectors = embeddings.embed_documents([
    "Hello, world!",
    "How are you?",
    "Good morning!"
])
```

---

## 🔍 Логирование

Proxy логирует использование дефолтной модели:

```json
{
  "level": "debug",
  "msg": "Processing embeddings request",
  "model": "text-embedding-ada-002",
  "default_model": "nomic-embed-text:latest",
  "time": "2025-10-24T12:00:00Z"
}
```

Проверка логов:

```bash
tail -f logs/proxy-dev.log | grep "embeddings"
```

---

## ⚠️ Важные замечания

### 1. Размерность векторов

**КРИТИЧНО:** Разные модели возвращают **разные dimensions**:

- `nomic-embed-text`: **768** dimensions
- `mxbai-embed-large`: **1024** dimensions
- `all-minilm`: **384** dimensions

**Нельзя** смешивать embeddings от разных моделей в одной векторной БД!

### 2. Миграция с OpenAI

При миграции с OpenAI (`text-embedding-ada-002` → Ollama):

1. **Пересоздайте векторную БД** с новыми embeddings
2. **Или** храните параллельно две версии
3. **НЕ** смешивайте OpenAI embeddings с Ollama

### 3. Performance

Локальные Ollama модели обычно **быстрее** чем OpenAI API:

| Операция | OpenAI API | Ollama (local) |
|----------|------------|----------------|
| Single embedding | ~200ms | ~20-50ms |
| Batch (100 texts) | ~500ms | ~100-200ms |
| Latency | Network | Local only |

### 4. Стоимость

**OpenAI ada-002:**
- $0.0001 / 1K tokens
- ~$0.10 / 1M tokens

**Ollama:**
- $0 (бесплатно)
- Только затраты на GPU/CPU

**Экономия:** ~$10-100/месяц в зависимости от объема

---

## 🐛 Troubleshooting

### Проблема: "model is required"

**Причина:** Не настроен `default_embedding_model` и модель не указана в запросе

**Решение:**
```yaml
models:
  default_embedding_model: "nomic-embed-text:latest"
```

---

### Проблема: "model not found"

**Причина:** Ollama не имеет указанную модель

**Решение:**
```bash
ollama pull nomic-embed-text
ollama list
```

---

### Проблема: Разные dimensions

**Причина:** Модель возвращает неожиданное количество dimensions

**Проверка:**
```bash
curl http://localhost:11434/api/embeddings \
  -d '{
    "model": "nomic-embed-text",
    "prompt": "test"
  }'
```

---

## 📚 См. также

- **[API Documentation](API_DOCUMENTATION.md)** - Полный API reference
- **[Configuration](CONFIGURATION.md)** - Все параметры конфигурации
- **[Troubleshooting](TROUBLESHOOTING.md)** - Решение проблем
- **[Performance](PERFORMANCE.md)** - Performance tuning

---

## 🎯 Best Practices

1. **Используйте nomic-embed-text** для большинства случаев
2. **Явно указывайте версию** модели: `nomic-embed-text:latest`
3. **Не смешивайте** embeddings от разных моделей
4. **Тестируйте качество** на ваших данных перед production
5. **Мониторьте latency** - embeddings должны быть быстрыми
6. **Используйте batch requests** для лучшей производительности

---

**Версия документа:** 1.0  
**Дата:** 24.10.2025  
**Применимо к:** v1.10.1+



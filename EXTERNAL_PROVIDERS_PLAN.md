# AIGateway: Удаление Ollama + Yzma + Внешние провайдеры + Переименование

> Трекинг-документ. Отмечай статус каждого пункта после выполнения.

## Статус этапов

| # | Описание | Статус |
|---|----------|--------|
| 1 | Удаление Ollama из кода | ⬜ Не начат |
| 2 | Удаление Yzma из кода | ⬜ Не начат |
| 3 | Переименование пакета → aigateway | ⬜ Не начат |
| 4 | Миграция БД (152) | ⬜ Не начат |
| 5 | Провайдеры OpenAI/Anthropic/Gemini | ⬜ Не начат |
| 6 | Конвертеры форматов (Anthropic, Gemini) | ⬜ Не начат |
| 7 | External Proxy Handler | ⬜ Не начат |
| 8 | Frontend — Admin Providers Page | ⬜ Не начат |
| 9 | Верификация и тесты | ⬜ Не начат |

---

## Этап 1: Удаление Ollama

### 1.1 Удалить файлы

| Файл | Статус |
|------|--------|
| `internal/providers/ollama_provider.go` | ⬜ |
| `internal/models/ollama.go` | ⬜ |
| `internal/rag/embeddings/ollama.go` | ⬜ |
| `internal/rag/embeddings/ollama_test.go` | ⬜ |

### 1.2 Модифицировать Go-файлы

| Файл | Что изменить | Статус |
|------|-------------|--------|
| `internal/models/model_registry.go` | Удалить `ProviderTypeOllama` | ⬜ |
| `internal/providers/manager.go` | Удалить `case models.ProviderTypeOllama` | ⬜ |
| `internal/providers/provider.go` | Убрать "Ollama" из комментария | ⬜ |
| `internal/models/mapping.go` | Удалить: `OllamaName`, `OllamaParam`, `MapOpenAIToOllama`/`MapOllamaToOpenAI`, `ConvertOllamaParams`/`ValidateOllamaParams`/`GetSupportedOllamaParams`, `APITypeOllama`, `DefaultModelMappings` | ⬜ |
| `internal/models/model_config.go` | Удалить `ToOllamaOptions()`, комментарий "Ollama API" | ⬜ |
| `internal/models/openai.go` | Удалить Ollama-specific поля | ⬜ |
| `internal/rag/embeddings/interface.go` | Default provider `"ollama"` → `"openai"` | ⬜ |
| `internal/rag/config/config.go` | Удалить `OllamaURL`, изменить default provider | ⬜ |
| `internal/config/database.go` | Default DB `"ollama_proxy"` → `"aigateway"` | ⬜ |
| `internal/rag/vector/interface.go` | Проверить ollama references | ⬜ |

### 1.3 Фронтенд

| Файл | Что изменить | Статус |
|------|-------------|--------|
| `web-svelte/.../admin/settings/+page.svelte` | Удалить `ollama: 'Ollama'` | ⬜ |
| `web-svelte/src/lib/api/downloads.ts` | Убрать `'ollama'` из type, удалить `pullOllamaModel()` | ⬜ |
| `internal/web/static/admin-registry.html` | Убрать ollama option | ⬜ |
| `internal/web/templates/.../provider_form.html` | Убрать ollama | ⬜ |

**Результат этапа 1:** ⬜

---

## Этап 2: Удаление Yzma

### 2.1 Удалить целые директории/файлы

| Файл/Директория | Статус |
|-----------------|--------|
| `internal/yzma/` (весь каталог: client.go, model_provider.go, vlm.go) | ⬜ |
| `internal/api/handlers/yzma_handler.go` | ⬜ |
| `internal/api/handlers/yzma_completions.go` | ⬜ |
| `internal/api/handlers/yzma_embeddings.go` | ⬜ |
| `internal/api/handlers/ui/yzma_ui_handler.go` | ⬜ |
| `web/yzma.html` | ⬜ |
| `internal/web/static/yzma.html` | ⬜ |
| `internal/web/templates/yzma_stats.html` | ⬜ |
| `internal/web/templates/yzma_model_card.html` | ⬜ |
| `internal/web/templates/yzma_models_list.html` | ⬜ |
| `internal/web/templates/yzma_loaded_models.html` | ⬜ |
| `docs/YZMA_INTEGRATION.md` | ⬜ |
| `docs/YZMA_GPU_SETUP_ROCKY9.md` | ⬜ |

### 2.2 Модифицировать router.go (КРИТИЧНО — ~300+ строк удаления)

| Блок (router.go) | Что убрать | Статус |
|-------------------|-----------|--------|
| Import `aigateway/internal/yzma` | Удалить | ⬜ |
| Поля `yzmaClient`, `yzmaHandler`, `yzmaUIHandler` | Удалить | ⬜ |
| Shutdown logic (~строки 406-412) | Удалить блок `if r.yzmaClient != nil` | ⬜ |
| UI routes (~строки 1035-1064) | Удалить блок `if r.yzmaUIHandler != nil` | ⬜ |
| System routes (~строки 1191-1228) | Удалить блок yzma + stub endpoints | ⬜ |
| StaticFile `/yzma.html` (~строка 1633) | Удалить | ⬜ |
| Inference routes (~строки 1726-1807) | Удалить `if r.yzmaHandler != nil` блок. Оставить ТОЛЬКО `inferenceProxyHandler` | ⬜ |
| Redis cache key "models:list:yzma" (~строка 2203) | Удалить | ⬜ |
| StatsHandler init (~строка 2569) | Убрать `yzmaClient` параметр | ⬜ |
| Yzma client init (~строки 2836-3046, ~210 строк) | Удалить весь блок | ⬜ |
| Health probe (~строки 3626-3629) | Удалить | ⬜ |

### 2.3 Модифицировать другие Go-файлы

| Файл | Что изменить | Статус |
|------|-------------|--------|
| `internal/config/config.go` | Удалить `YzmaConfig` struct, `Yzma` поле из `InferenceConfig`, defaults, backward compat (строки 777-801) | ⬜ |
| `internal/config/source.go` | Удалить `GetInferenceYzmaLibPath()` из интерфейса | ⬜ |
| `internal/config/hybrid.go` | Удалить yzma field mappings (~строки 386-412) | ⬜ |
| `internal/api/handlers/stats.go` | Удалить `YzmaClientInterface`, `yzmaClient` поле, yzma-related логику | ⬜ |
| `internal/services/agent/agent_service.go` | Удалить import `internal/yzma`, `yzmaClient` поле | ⬜ |
| `internal/cache/redis/background_sync.go` | Удалить cache key `"models:list:yzma"` | ⬜ |
| `internal/settings/seeder.go` | Удалить yzma settings (~строки 240-962) | ⬜ |
| `internal/settings/bootstrap.go` | Убрать Yzma console output | ⬜ |
| `internal/settings/seeder_test.go` | Убрать YzmaConfig из тестов | ⬜ |
| `internal/settings/reload_handlers.go` | Удалить yzma handler примеры | ⬜ |
| `internal/models/loaded_model.go` | Убрать yzma комментарий | ⬜ |
| `internal/web/framework/renderer.go` | Убрать yzma template mappings | ⬜ |

### 2.4 Фронтенд

| Файл | Что изменить | Статус |
|------|-------------|--------|
| `web-svelte/src/lib/api/admin.ts` | Удалить `YzmaGPUInfo`, `YzmaGPUResponse`, `YzmaHealthResponse` interfaces, `getYzmaGPUInfo()`, `getYzmaHealth()`, `yzma_enabled` | ⬜ |
| `web/admin.html` | Удалить Models tab "Local Models (yzma)", yzma stats, `deleteYzmaModel()` | ⬜ |
| `web/static/index.html` | Убрать yzma dashboard stats | ⬜ |
| `web/js/admin.js` | Убрать yzma references | ⬜ |
| `web/css/theme.css` | Удалить yzma CSS стили (~строки 1848-2402) | ⬜ |

### 2.5 Конфиги и деплой

| Файл | Что изменить | Статус |
|------|-------------|--------|
| `configs/dev.yaml` | Удалить inference.yzma секцию, `backend: "yzma"` | ⬜ |
| `configs/dev-local.yaml` | То же | ⬜ |
| `configs/production.yaml.example` | То же | ⬜ |
| `docker-compose.yml` | Удалить `AIGATEWAY_INFERENCE_BACKEND=yzma`, `YZMA_LIB` | ⬜ |
| `.air.toml` | Удалить `YZMA_LIB` | ⬜ |
| `packaging/rpm/oop.service` | Удалить `YZMA_LIB` environment | ⬜ |

### 2.6 Go modules

```bash
go mod tidy  # удалит github.com/hybridgroup/yzma и purego
```

### 2.7 Проверить компиляцию
```bash
go build -o bin/server.exe cmd/server/main.go
```

**Результат этапа 2:** ⬜

---

## Этап 3: Переименование пакета → aigateway

### 3.1 Packaging

| Файл | Что изменить | Статус |
|------|-------------|--------|
| `packaging/rpm/ollama-openai-proxy.spec` | Package name → `aigateway`, install paths → `/opt/aigateway/` | ⬜ |
| `packaging/rpm/oop.service` | Description, ExecStart, WorkingDirectory | ⬜ |
| `packaging/rpm/build-rpm.sh` | Ссылки на пакет | ⬜ |
| `ollama-openai-proxy.service` | Переименовать → `aigateway.service` | ⬜ |

### 3.2 Документация

| Файл | Статус |
|------|--------|
| `README.md` — обновить название, убрать Ollama | ⬜ |
| `CLAUDE.md` — обновить описание | ⬜ |
| `docs/API_DOCUMENTATION.md` — убрать Ollama API секции | ⬜ |
| `docs/EMBEDDINGS_CONFIGURATION.md` — убрать Ollama setup | ⬜ |
| `docs/AI_REVIEW_DYNAMIC_CONFIG.md` — убрать `OLLAMA_PROXY_API_KEY` | ⬜ |
| `docs/CONFIGURATION.md` — обновить | ⬜ |
| `docs/RAG_CONFIG_GUIDE.md` — обновить | ⬜ |
| `docs/README.md` — убрать yzma ссылки | ⬜ |

### 3.3 Конфиги

| Файл | Статус |
|------|--------|
| `configs/dev.yaml` — database name, ссылки | ⬜ |
| `configs/dev-local.yaml` — то же | ⬜ |

**Результат этапа 3:** ⬜

---

## Этап 4: Миграция БД (152)

**UP:** `internal/storage/postgresql/migrations/152_remove_ollama_add_gemini.up.sql`
```sql
DELETE FROM model_registry WHERE provider_id IN (
    SELECT id FROM model_providers WHERE provider_type = 'ollama'
);
DELETE FROM model_providers WHERE provider_type = 'ollama';

ALTER TABLE model_providers DROP CONSTRAINT IF EXISTS valid_provider_type;
ALTER TABLE model_providers ADD CONSTRAINT valid_provider_type
    CHECK (provider_type IN ('vllm', 'openai', 'anthropic', 'gemini', 'custom'));
```

**DOWN:** `internal/storage/postgresql/migrations/152_remove_ollama_add_gemini.down.sql`
```sql
ALTER TABLE model_providers DROP CONSTRAINT IF EXISTS valid_provider_type;
ALTER TABLE model_providers ADD CONSTRAINT valid_provider_type
    CHECK (provider_type IN ('ollama', 'vllm', 'openai', 'anthropic', 'custom'));

INSERT INTO model_providers (id, name, provider_type, base_url, enabled, priority)
VALUES ('ollama-local-default', 'Ollama Local', 'ollama', 'http://localhost:11434', true, 100)
ON CONFLICT (id) DO NOTHING;
```

**Результат этапа 4:** ⬜

---

## Этап 5: Провайдеры OpenAI / Anthropic / Gemini

### 5.1 Типы

| Файл | Изменение | Статус |
|------|-----------|--------|
| `internal/models/model_registry.go` | Добавить `ProviderTypeGemini = "gemini"` | ⬜ |

### 5.2 Реализации Provider interface

Каждый реализует: `GetName()`, `GetType()`, `HealthCheck()`, `ListModels()`, `GetModelInfo()`

| Файл | Описание | Статус |
|------|----------|--------|
| `internal/providers/openai_provider.go` | HealthCheck: `GET /v1/models`. ListModels: парсит response. API key в `Authorization: Bearer` | ⬜ |
| `internal/providers/anthropic_provider.go` | HealthCheck: `GET /v1/models`. ListModels: API или хардкод. API key в `x-api-key` + `anthropic-version` header | ⬜ |
| `internal/providers/gemini_provider.go` | HealthCheck: `GET /v1beta/models?key=`. ListModels: парсит. API key в URL param | ⬜ |

### 5.3 ProviderManager

| Файл | Изменение | Статус |
|------|-----------|--------|
| `internal/providers/manager.go` | Добавить cases для OpenAI, Anthropic, Gemini. Передать `apiKey` в конструктор | ⬜ |

**Результат этапа 5:** ⬜

---

## Этап 6: Конвертеры форматов

### 6.1 Anthropic (`internal/providers/converter/anthropic.go`)

- `ConvertOpenAIToAnthropic(req) → AnthropicRequest` — system → отдельное поле, content blocks, max_tokens обязателен
- `ConvertAnthropicToOpenAI(resp) → ChatCompletionResponse` — content blocks → choices.message.content
- `ConvertAnthropicStreamToOpenAI(event) → SSE line` — `content_block_delta` → delta.content, `message_stop` → `[DONE]`

| Файл | Статус |
|------|--------|
| `internal/providers/converter/anthropic.go` | ⬜ |
| `internal/providers/converter/anthropic_test.go` | ⬜ |

### 6.2 Gemini (`internal/providers/converter/gemini.go`)

- `ConvertOpenAIToGemini(req) → GeminiRequest` — system → `systemInstruction`, messages → `contents[{role, parts}]`
- `ConvertGeminiToOpenAI(resp) → ChatCompletionResponse` — candidates → choices
- `ConvertGeminiStreamToOpenAI(chunk) → SSE line`

| Файл | Статус |
|------|--------|
| `internal/providers/converter/gemini.go` | ⬜ |
| `internal/providers/converter/gemini_test.go` | ⬜ |

**Результат этапа 6:** ⬜

---

## Этап 7: External Proxy Handler

### 7.1 Handler (`internal/api/handlers/external_proxy_handler.go`)

```go
type ExternalProxyHandler struct {
    db     storage.Database
    logger *logrus.Logger
    client *http.Client
}
```

Методы:
- `HandleChatCompletions(c *gin.Context)` — lookup model_registry → model_providers → route by provider_type:
  - **openai/custom**: прямой прокси + `Authorization: Bearer`
  - **anthropic**: convert → proxy `/v1/messages` → convert back
  - **gemini**: convert → proxy `/v1beta/models/{model}:generateContent` → convert back
- `HandleModels(c *gin.Context)` — список из registry
- `HandleEmbeddings(c *gin.Context)` — проксирование embeddings

| Файл | Статус |
|------|--------|
| `internal/api/handlers/external_proxy_handler.go` | ⬜ |

### 7.2 Router integration (`internal/api/router/router.go`)

После удаления Yzma, логика маршрутизации `/v1/chat/completions`:

```
if inferenceProxyHandler != nil {
    // Двухуровневый handler:
    // 1. Попробовать Docker inference (EnsureByAlias)
    // 2. Если модель не найдена → ExternalProxyHandler (model_registry)
    v1.POST("/chat/completions", combinedHandler)
} else {
    // Только external providers
    v1.POST("/chat/completions", externalProxyHandler.HandleChatCompletions)
}
```

| Файл | Статус |
|------|--------|
| `internal/api/router/router.go` | ⬜ |

**Результат этапа 7:** ⬜

---

## Этап 8: Frontend — Admin Providers Page

### 8.1 API клиент

**Файл:** `web-svelte/src/lib/api/registry.ts`

```typescript
export interface ModelProvider {
    id: string;
    name: string;
    provider_type: 'openai' | 'anthropic' | 'gemini' | 'vllm' | 'custom';
    base_url: string;
    enabled: boolean;
    priority: number;
    config: Record<string, unknown>;
    health_status: 'healthy' | 'unhealthy' | 'unknown';
    last_health_check?: string;
    error_message?: string;
}

export const registryApi = {
    listProviders, getProvider, createProvider, updateProvider, deleteProvider,
    healthCheckAll, listModels, discoverModels, deleteModel, getStats
};
```

| Файл | Статус |
|------|--------|
| `web-svelte/src/lib/api/registry.ts` | ⬜ |

### 8.2 Admin Providers Page

**Файл:** `web-svelte/src/routes/(protected)/admin/providers/+page.svelte`

Секции: Providers List + Add/Edit Dialog + Models Registry + Discover Models

Default base URLs:
- OpenAI: `https://api.openai.com/v1`
- Anthropic: `https://api.anthropic.com`
- Gemini: `https://generativelanguage.googleapis.com`

| Файл | Статус |
|------|--------|
| `web-svelte/src/routes/(protected)/admin/providers/+page.svelte` | ⬜ |

### 8.3 Navigation + Route

| Файл | Что | Статус |
|------|-----|--------|
| `web-svelte/.../admin/+layout.svelte` | Добавить tab "Providers" | ⬜ |
| `internal/api/router/router.go` | Добавить `/admin/providers` в svelteRoutes | ⬜ |

**Результат этапа 8:** ⬜

---

## Этап 9: Верификация

| Проверка | Статус |
|----------|--------|
| `go build -o bin/server.exe cmd/server/main.go` | ⬜ |
| `go test ./...` | ⬜ |
| `go mod tidy` — удалены yzma/purego зависимости | ⬜ |
| Нет `ollama` в Go-коде (кроме миграций) | ⬜ |
| Нет `yzma` в Go-коде (кроме миграций) | ⬜ |
| UI: `/admin/providers` открывается | ⬜ |
| UI: создание OpenAI провайдера работает | ⬜ |
| UI: discover models работает | ⬜ |

**Результат этапа 9:** ⬜

---

## Архитектура потока запросов (после реализации)

```
Client → POST /v1/chat/completions { model: "claude-3-opus" }
    │
    ├─ 1. Docker Inference (EnsureByAlias)
    │   → модель "claude-3-opus" не в inference store → SKIP
    │
    └─ 2. External Provider (model_registry lookup):
        ├─ SELECT model_registry WHERE model_id = 'claude-3-opus'
        │   → provider_id = 'anthropic-prod'
        │
        ├─ SELECT model_providers WHERE id = 'anthropic-prod'
        │   → provider_type = 'anthropic', base_url, api_key
        │
        ├─ Convert: OpenAI request → Anthropic Messages API
        ├─ Proxy: POST https://api.anthropic.com/v1/messages
        ├─ Convert: Anthropic response → OpenAI format
        └─ Return to client
```

---

## Ключевые файлы

### Удаляем
- `internal/yzma/` (весь каталог)
- `internal/api/handlers/yzma_*.go` (3 файла)
- `internal/api/handlers/ui/yzma_ui_handler.go`
- `internal/providers/ollama_provider.go`
- `internal/models/ollama.go`
- `internal/rag/embeddings/ollama.go` + test
- `web/yzma.html`, `internal/web/static/yzma.html`
- `internal/web/templates/yzma_*.html` (4 файла)
- `docs/YZMA_*.md` (2 файла)

### Создаём
- `internal/providers/openai_provider.go`
- `internal/providers/anthropic_provider.go`
- `internal/providers/gemini_provider.go`
- `internal/providers/converter/anthropic.go` + test
- `internal/providers/converter/gemini.go` + test
- `internal/api/handlers/external_proxy_handler.go`
- `web-svelte/src/lib/api/registry.ts`
- `web-svelte/src/routes/(protected)/admin/providers/+page.svelte`
- Миграция 152 (up + down)
- `aigateway.service` (переименование)

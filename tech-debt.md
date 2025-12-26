# Technical Debt

Список технического долга, требующего исправления.

## 🔴 Критично (блокирует CI)

### Устаревшие тесты с `config.OllamaConfig`

**Проблема**: Тесты используют удалённую структуру `config.OllamaConfig`, которая была заменена на `config.InferenceConfig`.

**Файлы**:
- `internal/auth/security_test.go`
- `internal/auth/performance_test.go`
- `internal/auth/penetration_test.go`
- `internal/auth/middleware/auth_keys_test.go`
- `internal/auth/apikey/manager_test.go`

**Решение**: Заменить `config.OllamaConfig{URL: "..."}` на `config.InferenceConfig{}` или удалить поле `Ollama` из тестовых конфигов.

---

### CircuitBreaker использует `cfg.Ollama`

**Проблема**: `internal/circuit/breaker.go:75` использует `cfg.Ollama.CircuitBreaker` которого больше нет.

**Файлы**:
- `internal/circuit/breaker.go`

**Решение**: Перенести конфигурацию CircuitBreaker в `config.InferenceConfig` или `config.ServerConfig`.

---

### RAG пакеты с undefined типами

**Проблема**: Используются несуществующие типы `models.RAGVector`, `storage.Storage`.

**Файлы**:
- `internal/rag/iterators.go:116` — `models.RAGVector`
- `internal/rag/queue/postgres.go:19` — `storage.Storage`
- `internal/rag/processor/pipeline.go:29` — `storage.Storage`

**Решение**: Определить недостающие типы или обновить импорты.

---

### Mock не реализует интерфейс

**Проблема**: `internal/services/rag/datasource_service_test.go:78` — `mockDatabase` не реализует `storage.Database` (отсутствует метод `AddTenantMember`).

**Решение**: Добавить метод `AddTenantMember` в mock.

---

### WebUI Legacy Files

**Проблема**: `cmd/webui/main.go:52` использует `web.StaticFiles` вместо `web.LegacyFiles`.

**Решение**: Заменить `web.StaticFiles` на `web.LegacyFiles`.

---

### Unused import

**Проблема**: `internal/notifications/telegram/client.go:11` — `"strings"` imported but not used.

**Решение**: Удалить неиспользуемый импорт.

---

### Unsafe Pointer

**Проблема**: `internal/yzma/client.go:32` — possible misuse of `unsafe.Pointer`.

**Решение**: Проверить и исправить использование unsafe.Pointer или добавить `//nolint:govet` если это намеренно.

---

## 🟡 Средний приоритет

### Тестовое покрытие

Текущее состояние: тесты в нескольких пакетах не компилируются из-за устаревших зависимостей.

**Пакеты без тестов в CI**:
- `internal/auth/*`
- `internal/circuit`
- `internal/rag/*`
- `internal/notifications/telegram`
- `cmd/webui`
- `internal/services/rag`

---

## 🟢 Низкий приоритет

### Документация

- Обновить README с актуальной структурой конфигурации
- Добавить примеры для новых InferenceConfig параметров

---

## Временные решения в CI

В `.gitlab-ci.yml` применены следующие workarounds:

```yaml
# Пропуск проблемных пакетов в тестах
go test $(go list ./... | grep -v -E '(internal/auth|internal/circuit|...)')

# Разрешение падения тестов
allow_failure: true
```

---

## Как исправить

1. **Создать ветку** `fix/tech-debt-tests`
2. **Исправить по одному пакету**, начиная с самых простых (`unused import`, `web.StaticFiles`)
3. **Обновить моки** для соответствия интерфейсам
4. **Удалить устаревшие ссылки** на `OllamaConfig`
5. **Убрать workarounds** из CI после исправления

---

*Создано: 2025-12-26*
*Последнее обновление: 2025-12-26*


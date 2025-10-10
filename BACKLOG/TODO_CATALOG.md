# TODO Catalog - Систематизация технического долга

**Создано:** 2025-10-10  
**Статус:** Active  
**Всего задач:** 9

---

## 📋 Обзор

Этот каталог содержит все TODO комментарии найденные в коде проекта, систематизированные и преобразованные в задачи с приоритетами и оценками.

## 🔍 Найденные TODO в коде

### cmd/tui/main.go (4 комментария)

```go
// Строка ~1991
req.Header.Set("Authorization", "Bearer sk-admin-dev-key-12345") // TODO: Получать из конфига

// Строка ~2265
req.Header.Set("Authorization", "Bearer "+adminAPIKey) // TODO: Получать из конфигурации

// Строка ~2322
req.Header.Set("Authorization", "Bearer "+adminAPIKey) // TODO: Получать из конфигурации

// Строка ~2351
// TODO: Добавить константу adminAPIKey или получать из config
```

**→ Задача:** [TODO-01: TUI Admin API Key Configuration](TODO-01_tui_admin_key_config.md)  
**Версия:** v1.4.2  
**Приоритет:** HIGH  
**Оценка:** 2 часа

---

### internal/logger/logger.go (1 комментарий)

```go
// Строка 66
"version": "dev", // TODO: Получать из build информации
```

### internal/api/handlers/health.go (2 комментария)

```go
// Строка 36
"version":   "dev", // TODO: Получать из build информации

// Строка 38
"uptime":    time.Since(time.Now().Add(-5 * time.Minute)).String(), // TODO: Реальный uptime
```

**→ Задача:** [TODO-02: Build Version Information](TODO-02_build_version_info.md)  
**Версия:** v1.4.3  
**Приоритет:** MEDIUM  
**Оценка:** 1-2 часа

---

### internal/api/handlers/auth.go (1 комментарий)

```go
// Строка 203
// TODO: Implement password change
```

**→ Задача:** [TODO-03: Password Change Implementation](TODO-03_password_change.md)  
**Версия:** v1.5.2  
**Приоритет:** MEDIUM  
**Оценка:** 3-4 часа

---

### internal/api/handlers/usage.go (1 комментарий)

```go
// Строка 90
// TODO: Add proper tenant membership check
```

**→ Задача:** [TODO-04: Tenant Membership Check](TODO-04_tenant_membership.md)  
**Версия:** v1.5.3  
**Приоритет:** MEDIUM  
**Оценка:** 2-3 часа

---

### internal/api/handlers/admin.go (1 комментарий)

```go
// Строка 792
// TODO: Реализовать проверку типа ошибки
```

**→ Задача:** [TODO-05: Error Type Checking](TODO-05_error_type_check.md)  
**Версия:** v1.5.4  
**Приоритет:** LOW  
**Оценка:** 1-2 часа

---

### internal/auth/jwt/jwt.go (1 комментарий)

```go
// Строка 18
tokenBlacklist       map[string]time.Time // In-memory blacklist (TODO: migrate to Redis)
```

**→ Задача:** TODO-06: Redis Token Blacklist  
**Версия:** Future (1.6.0+)  
**Приоритет:** LOW  
**Оценка:** 4-5 часов  
**Статус:** Deferred

---

### internal/storage/postgresql/stubs.go (1 комментарий)

```go
// Строка 2
// TODO: Replace with actual implementations from CRUD files
```

**→ Задача:** TODO-07: PostgreSQL Stubs Implementation  
**Версия:** Future (1.6.0+)  
**Приоритет:** LOW  
**Оценка:** 6-8 часов  
**Статус:** Deferred

---

### internal/test/helpers.go (3 комментария)

```go
// Строка 140
// TODO: Добавить парсинг JSON и проверку error.code

// Строка 231
// TODO: Реализовать детальное сравнение JSON объектов

// Строка 242
// TODO: Реализовать создание HTTP запросов с JSON body
```

**→ Задача:** TODO-08: Test Helpers Enhancement  
**Версия:** Future (1.6.0+)  
**Приоритет:** LOW  
**Оценка:** 3-4 часа  
**Статус:** Deferred

---

### internal/api/middleware/auth.go (3 комментария)

```go
// Строка 38
// TODO: Проверить валидность API ключа через API Key Manager

// Строка 46
// TODO: Добавить проверку прав доступа к моделям

// Строка 47
// TODO: Добавить rate limiting
```

**→ Задача:** TODO-09: Auth Middleware Cleanup  
**Версия:** Future  
**Приоритет:** LOW  
**Оценка:** -  
**Статус:** Legacy (можно удалить файл, функционал реализован в apikey_db_auth.go)

---

## 📊 Распределение по версиям

### Version 1.4.0 (Текущая)

- ✅ TODO-01: TUI Admin API Key Configuration (2ч)
- ✅ TODO-02: Build Version Information (1-2ч)

**Итого:** 3-4 часа

### Version 1.5.0

- ✅ TODO-03: Password Change (3-4ч)
- ✅ TODO-04: Tenant Membership (2-3ч)
- ✅ TODO-05: Error Type Check (1-2ч)

**Итого:** 6-9 часов

### Future / Deferred

- TODO-06: Redis Token Blacklist (4-5ч)
- TODO-07: PostgreSQL Stubs (6-8ч)
- TODO-08: Test Helpers (3-4ч)
- TODO-09: Auth Middleware Cleanup (legacy)

**Итого:** 13-17 часов

---

## 🎯 Приоритизация

### HIGH (Критичные для production)

1. **TODO-01** - Hardcoded credentials (security risk)
2. **TODO-02** - Версионирование (debugging, releases)

### MEDIUM (Важные функции)

3. **TODO-03** - Password change (уже есть UI)
4. **TODO-04** - Tenant membership (security)

### LOW (Улучшения)

5. **TODO-05** - Error type checking (code quality)
6. **TODO-06** - Redis blacklist (scalability)
7. **TODO-07** - PostgreSQL stubs (completeness)
8. **TODO-08** - Test helpers (testing)

### LEGACY (Cleanup)

9. **TODO-09** - Auth middleware (удалить файл)

---

## 🔄 Процесс обработки TODO

### Когда добавлять новые TODO в код

1. **Временные заглушки** - когда нужно быстро двигаться дальше
2. **Future enhancements** - идеи для будущих улучшений
3. **Technical debt** - известные проблемы которые нужно исправить

### Формат TODO комментариев

```go
// TODO: Краткое описание задачи
// Context: дополнительная информация (опционально)
// Issue: #123 (если есть issue)
```

### Когда перемещать TODO в BACKLOG

- При планировании релиза
- Когда TODO старше 2 недель
- Когда TODO критично для production
- При code review

### Lifecycle

```
TODO в коде → BACKLOG спецификация → Задача в релизе → Реализация → Удаление TODO
```

---

## 📝 Checklist при закрытии TODO

При реализации задачи:

- [ ] Реализовать функционал
- [ ] Удалить TODO комментарий из кода
- [ ] Обновить спецификацию в BACKLOG (статус: Completed)
- [ ] Написать tests
- [ ] Обновить документацию
- [ ] Code review
- [ ] Закрыть связанный issue (если есть)

---

## 🔍 Поиск TODO в проекте

```bash
# Найти все TODO
grep -r "TODO" --include="*.go" internal/ cmd/ pkg/

# Найти TODO с контекстом
grep -B 2 -A 2 "TODO" --include="*.go" internal/

# Статистика TODO
grep -r "TODO" --include="*.go" internal/ cmd/ pkg/ | wc -l
```

---

## 📚 References

- [Roadmap](../Roadmap.MD) - план версий
- [BACKLOG](.) - все спецификации задач
- [Architecture](../Architecture.MD) - архитектура проекта

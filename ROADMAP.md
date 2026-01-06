# AIGateway Roadmap

Планы развития проекта. Отмечаем статус по мере реализации.

## Статусы

- ⬜ Planned — запланировано
- 🟡 In Progress — в разработке
- ✅ Done — готово
- ❌ Cancelled — отменено

---

## 1. 🔄 Dependency Updater

**Приоритет:** Высокий  
**Статус:** ✅ Done

Автоматическое обновление зависимостей с созданием Issue (аналог Renovate).

### Поддерживаемые языки

| Язык | Файл | Реестр | Статус |
|------|------|--------|--------|
| Go | `go.mod` | proxy.golang.org | ✅ Done |
| Node.js | `package.json` | registry.npmjs.org | ✅ Done |
| Python | `requirements.txt` | pypi.org | ✅ Done |
| Rust | `Cargo.toml`, `Cargo.lock` | crates.io | ⬜ |
| Java | `pom.xml` | maven.org | ⬜ |
| .NET | `*.csproj`, `packages.config` | nuget.org | ⬜ |

### Функциональность

- [x] Парсеры для каждого формата dependency файлов
  - [x] Go: go.mod
  - [x] Node.js: package.json
  - [x] Python: requirements.txt
- [x] Клиенты для package registries
  - [x] proxy.golang.org
  - [x] registry.npmjs.org
  - [x] pypi.org
- [x] Security advisories интеграция (OSV.dev)
- [x] LLM анализ changelog'ов новых версий
- [x] Breaking changes detection через LLM
- [x] UI: кнопка "Check Dependencies" в проекте
- [x] UI: модалка с результатами проверки
- [x] Создание GitLab Issue для критичных обновлений
- [x] Документация в приложении (/docs)
- [x] Scheduled scans (cron)
- [x] История сканирований в БД

### Файлы

```
internal/gitlab/dependencies/
├── parser/
│   ├── gomod.go       # go.mod parser
│   ├── npm.go         # package.json parser
│   ├── pip.go         # requirements.txt, pyproject.toml parser
│   ├── cargo.go       # Cargo.toml parser
│   └── maven.go       # pom.xml parser
├── registry/
│   ├── golang.go      # proxy.golang.org client
│   ├── npm.go         # registry.npmjs.org client
│   ├── pypi.go        # pypi.org client
│   ├── crates.go      # crates.io client
│   └── maven.go       # maven.org client
├── security/
│   ├── osv.go         # OSV.dev client (CVE)
│   └── advisory.go    # GitHub Advisory DB client
├── scanner.go         # Основной сканер
├── analyzer.go        # LLM анализ обновлений
└── types.go           # Общие типы
```

---

## 2. 📊 Code Quality Score

**Приоритет:** Средний  
**Статус:** ✅ Done

LLM-based оценка качества кода с метриками.

### Метрики

- [x] Complexity score (cyclomatic complexity estimation)
- [x] Documentation coverage
- [x] Test coverage estimation (без запуска тестов)
- [x] Code duplication detection
- [x] Naming conventions adherence
- [x] Error handling quality
- [x] Maintainability score

### Вывод

```json
{
  "overall_score": 78,
  "breakdown": {
    "complexity": 85,
    "documentation": 60,
    "testing": 70,
    "security": 90,
    "maintainability": 75
  },
  "recommendations": [
    "Add documentation to 15 public functions",
    "Reduce complexity in auth/handler.go"
  ]
}
```

### UI

- Виджет на странице проекта с gauge/score
- Drill-down в детали по категориям
- Trend график (история оценок)

---

## 3. 🔍 Dead Code Detection

**Приоритет:** Средний  
**Статус:** ✅ Done

Поиск неиспользуемого кода через RAG + LLM.

### Подход

1. Индексация всех символов (функции, типы, переменные)
2. Построение графа вызовов через RAG search
3. LLM анализ "orphan" символов
4. Отчет с уверенностью (high/medium/low)

### Типы dead code

- [x] Неиспользуемые функции
- [x] Неиспользуемые типы/структуры
- [x] Неиспользуемые переменные/константы
- [x] Недостижимый код (unreachable)
- [x] Commented-out код

### UI

- [x] Список найденного dead code с фильтрами
- [x] Экспорт в GitLab Issue
- [x] Bulk select для создания Issue

---

## 4. 📝 Auto-Documentation Generator

**Приоритет:** Средний  
**Статус:** ✅ Done

Автоматическая генерация документации для кода.

### Форматы

| Язык | Формат |
|------|--------|
| Go | GoDoc comments |
| TypeScript/JS | JSDoc/TSDoc |
| Python | docstrings (Google/NumPy style) |
| Rust | rustdoc |
| Java | Javadoc |

### Функциональность

- [x] Scan для функций без документации
- [x] LLM генерация документации на основе кода
- [x] Preview перед применением
- [x] Copy to clipboard
- [x] Bulk apply (весь файл/проект)
- [x] Создание MR с документацией

### UI

- [x] Список недокументированных функций
- [x] Inline preview сгенерированной документации
- [x] Copy to clipboard
- [x] Создание Issue с рекомендациями

---

## 5. 🧪 Test Generation

**Приоритет:** Низкий  
**Статус:** ✅ Done

LLM генерация unit-тестов.

### Подход

1. Анализ функции через RAG (зависимости, типы)
2. LLM генерация test cases
3. Генерация кода тестов
4. Валидация синтаксиса

### Поддержка

| Язык | Framework |
|------|-----------|
| Go | testing + testify |
| TypeScript | Jest/Vitest |
| Python | pytest |
| Rust | cargo test |

### UI

- [x] Scan для функций без тестов
- [x] LLM генерация тестов
- [x] Preview тестов
- [x] Copy to clipboard
- [x] Download as file

---

## 6. 🏗️ Architecture Diagram Generator

**Приоритет:** Низкий  
**Статус:** ✅ Done

Автоматическая визуализация архитектуры проекта.

### Типы диаграмм

- [x] Module dependency graph
- [x] Call graph (функции)
- [x] Data flow diagram
- [x] Package structure

### Формат вывода

- [x] Mermaid.js (для README)
- [ ] SVG/PNG export (requires external renderer)
- [x] Interactive (D3.js JSON format)

### Подход

1. Парсинг imports/dependencies из индекса
2. Построение графа
3. LLM для именования групп/модулей
4. Рендеринг

---

## 7. 🛡️ Security Scanner (расширение)

**Приоритет:** Высокий  
**Статус:** ✅ Done

### Сделано

- [x] Regex-based secrets detection
- [x] LLM deep scan
- [x] SAST (Static Application Security Testing)
- [x] Vulnerable dependencies (CVE check via OSV.dev)
- [x] SQL injection patterns
- [x] XSS patterns
- [x] Path traversal patterns
- [x] Command injection patterns
- [x] Hardcoded IPs/URLs detection
- [x] Insecure cryptography detection (MD5, SHA1, DES)
- [x] Insecure random detection
- [x] Open redirect patterns
- [x] SSRF patterns
- [x] XXE patterns
- [x] Insecure deserialization patterns
- [x] Hardcoded credentials patterns

### Планируется

- [ ] Scheduled security scans

---

## 8. 📈 Analytics Dashboard

**Приоритет:** Низкий  
**Статус:** ✅ Done

Аналитика по проектам и командам.

### Метрики

- [x] Code review statistics (avg time, issues found)
- [x] Dependency health (outdated %)
- [x] Security score trend
- [x] Team productivity metrics
- [x] Model usage statistics
- [x] Export to JSON/CSV

### API Endpoints

- `GET /api/admin/gitlab/analytics/dashboard` - Main dashboard
- `GET /api/admin/gitlab/analytics/models` - Model comparison
- `GET /api/admin/gitlab/analytics/security` - Security overview
- `GET /api/admin/gitlab/analytics/dependencies` - Dependency health
- `GET /api/admin/gitlab/projects/:id/analytics` - Project analytics
- `GET /api/admin/gitlab/analytics/export` - Export report

---

## Порядок реализации

### Phase 1 (Q1 2025)
1. ✅ Secrets Scanner (Done)
2. 🟡 Dependency Updater — Go + Node.js

### Phase 2 (Q2 2025)
3. Code Quality Score
4. Dependency Updater — Python + Rust

### Phase 3 (Q3 2025)
5. Dead Code Detection
6. Auto-Documentation

### Phase 4 (Q4 2025)
7. Test Generation
8. Architecture Diagrams
9. Analytics Dashboard

---

## Технические заметки

### Общие компоненты

- `internal/gitlab/analysis/` — общий код анализа
- `internal/gitlab/mr/` — создание MR
- `internal/registry/` — клиенты package registries
- `web-svelte/src/lib/components/analysis/` — UI компоненты

### API endpoints

```
# Dependency Updater
POST /api/admin/gitlab/projects/:id/check-dependencies
POST /api/admin/gitlab/projects/:id/create-dependency-issue

# Code Analysis
POST /api/admin/gitlab/projects/:id/quality-score
POST /api/admin/gitlab/projects/:id/dead-code
POST /api/admin/gitlab/projects/:id/generate-docs
POST /api/admin/gitlab/projects/:id/generate-tests
GET  /api/admin/gitlab/projects/:id/architecture

# Issues
POST /api/admin/gitlab/projects/:id/create-issue
```

---

## 9. ✨ Validation Refactoring (Zog)

**Приоритет:** Средний  
**Статус:** ⬜ Planned  
**Анализ:** [docs/research/ZOG_VALIDATION_ANALYSIS.md](docs/research/ZOG_VALIDATION_ANALYSIS.md)

Переход на декларативную валидацию с [Zog](https://github.com/Oudwins/zog).

### Зачем

- Сокращение boilerplate кода в handlers
- Единый формат ошибок валидации
- Built-in coercion и i18n
- Zod-like API (знакомо TypeScript разработчикам)

### Фазы реализации

#### Phase 1: Инфраструктура (v4.3.x)

- [ ] Добавить `github.com/Oudwins/zog` в go.mod
- [ ] Создать `internal/api/schemas/` директорию
- [ ] Написать validation middleware для Gin
- [ ] Настроить i18n (ru + en)

#### Phase 2: Новые endpoints (v4.3.x)

- [ ] Использовать Zog для всех новых API handlers
- [ ] Документировать паттерн для команды

#### Phase 3: Миграция существующих handlers (v4.4.x)

- [ ] Chat Completions (`inference_proxy_handler.go`)
- [ ] API Key CRUD (`admin.go`)
- [ ] User Auth (`auth.go`)
- [ ] GitLab Integration (`gitlab_admin.go`)
- [ ] Model Configuration (`inference_handler.go`)

#### Phase 4: Config & Env validation (v4.5.x)

- [ ] Config file validation
- [ ] Environment variables validation с zenv

### Пример

**До:**
```go
if req.Name == "" {
    c.JSON(400, gin.H{"error": "name required"})
    return
}
if len(req.Name) > 255 { ... }
```

**После:**
```go
var schema = z.Struct(z.Shape{
    "name": z.String().Min(1).Max(255).Required(),
})
errs := schema.Parse(zhttp.Request(r), &req)
```

### Файлы

```
internal/api/
├── schemas/
│   ├── apikey.go
│   ├── auth.go
│   ├── gitlab.go
│   ├── inference.go
│   └── common.go
└── middleware/
    └── validation.go
```

---

## 10. 🧠 Self-Learning Agent (Acontext Integration)

**Приоритет:** Средний  
**Статус:** ⬜ Planned  
**Анализ:** [docs/research/ACONTEXT_ANALYSIS.md](docs/research/ACONTEXT_ANALYSIS.md)

Интеграция с [Acontext](https://github.com/memodb-io/Acontext) для self-learning возможностей AI-агентов.

### Концепция

Агент учится на успешных сессиях и накапливает знания:
- Test Generation → запоминает какие тесты были приняты
- Code Review → изучает предпочтения команды
- Auto-doc → адаптирует стиль документации под проект

### Фазы реализации

#### Phase 1: Task Tracking (v4.3.x)

- [ ] Task структура для GitLab операций
- [ ] Progress tracking для долгих операций
- [ ] Хранение результатов (success/failed)

#### Phase 2: Pattern Learning (v5.x)

- [ ] Сохранение успешных паттернов
- [ ] Project-specific промпт customization
- [ ] SOP (Standard Operating Procedure) структура

#### Phase 3: Full Integration (v6.x)

- [ ] Acontext как external service или форк
- [ ] Experience search по накопленным навыкам
- [ ] Multi-project knowledge sharing
- [ ] Spaces для команд/организаций

### Что можно взять из Acontext

| Компонент | Что взять | Применение |
|-----------|-----------|------------|
| Task Extraction | Алгоритм извлечения задач | GitLab операции |
| SOP Structure | Формат хранения навыков | Test/doc generation |
| Experience Search | Fast/agentic поиск | Улучшение промптов |
| Spaces | Организация знаний | Per-project настройки |

### Файлы

```
internal/learning/
├── task/
│   ├── extractor.go    # Task extraction from sessions
│   └── tracker.go      # Progress tracking
├── sop/
│   ├── types.go        # SOP structure
│   ├── storage.go      # SOP persistence
│   └── matcher.go      # Experience search
└── space/
    ├── types.go        # Space/knowledge base
    └── manager.go      # Space operations
```

### Референсы

- [Acontext GitHub](https://github.com/memodb-io/Acontext)
- [LangMem](https://github.com/langchain-ai/langmem) — memory для LangChain
- [MemGPT](https://github.com/cpacker/MemGPT) — persistent memory agents

---

## Ресурсы

- [Renovate](https://github.com/renovatebot/renovate) — референс для Dependency Updater
- [GitHub Advisory Database](https://github.com/advisories) — CVE данные
- [OSV](https://osv.dev/) — Open Source Vulnerabilities
- [Libraries.io](https://libraries.io/) — Package metadata API
- [Acontext](https://github.com/memodb-io/Acontext) — Context Engineering платформа

---

*Последнее обновление: 2026-01-05*


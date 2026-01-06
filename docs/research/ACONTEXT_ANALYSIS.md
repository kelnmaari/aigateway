# Acontext Analysis

> Анализ проекта [Acontext](https://github.com/memodb-io/Acontext) для потенциальной интеграции и заимствования подходов.

**Дата анализа:** 2026-01-05  
**Версия Acontext:** cli/v0.0.21  
**Лицензия:** Apache-2.0  
**Языки:** Go (44.3%), Python (34.9%), TypeScript (19.5%)

## Обзор

Acontext — платформа для "Context Engineering" AI-агентов с возможностями:
- Хранение и управление сессиями
- Автоматическое извлечение задач из диалогов
- Self-learning (обучение на успешных сценариях)
- Накопление знаний в "Spaces"

## Ключевые концепции

### 1. Sessions (Сессии)

```python
session = client.sessions.create()
client.sessions.store_message(session_id=session.id, blob=msg, format="openai")
messages = client.sessions.get_messages(session.id)
```

**Что можно взять:**
- [ ] Формат хранения сообщений совместимый с OpenAI
- [ ] Автоматическое версионирование диалогов
- [ ] `flush()` — блокирующий вызов для завершения обработки

### 2. Task Extraction (Извлечение задач)

```python
tasks = client.sessions.get_tasks(session.id)
# Возвращает:
# - task_description
# - status (success/pending/failed)
# - progresses (список обновлений)
# - user_preferences
```

**Применение для нашего проекта:**
- **Test Generation**: Извлечение "что пользователь хотел протестировать"
- **Code Review**: Трекинг "какие файлы нужно проверить"
- **Dependency Scan**: "Какие уязвимости исправить"

**Что можно взять:**
- [ ] Структура Task с progress tracking
- [ ] Фоновый агент для анализа прогресса
- [ ] Статусы задач и их переходы

### 3. Artifacts (Артефакты)

```python
disk = client.disks.create()
artifact = client.disks.artifacts.upsert(
    disk.id,
    file=FileUpload(filename="todo.md", content=b"..."),
    file_path="/todo/"
)
```

**Применение:**
- Хранение сгенерированных тестов
- Кэширование результатов анализа
- Версионирование документации

**Что можно взять:**
- [ ] Виртуальная файловая система для агента
- [ ] Public URLs для артефактов

### 4. Self-Learning (Самообучение)

```
Task Completed → Extract SOP → Store as Skill Block → Available for Future
```

**SOP (Standard Operating Procedure) формат:**
```json
{
    "use_when": "star a github repo",
    "preferences": "use personal account",
    "tool_sops": [
        {"tool_name": "goto", "action": "goto the user given github repo url"},
        {"tool_name": "click", "action": "find login button..."}
    ]
}
```

**Применение для нашего проекта:**

| Сценарий | Что можно выучить |
|----------|-------------------|
| Test Generation | Какой framework использовать, структура тестов, naming conventions |
| Code Review | Предпочтения по стилю, критерии качества |
| Auto-doc | Формат документации, уровень детализации |
| MR Creation | Naming веток, формат commit messages |

**Что можно взять:**
- [ ] SOP структура для хранения паттернов
- [ ] Background agent для анализа успешных сессий
- [ ] Experience search по накопленным навыкам

### 5. Spaces (Пространства знаний)

```python
space = client.spaces.create()
session = client.sessions.create(space_id=space.id)

# Search accumulated skills
result = client.spaces.experience_search(
    space_id=space.id,
    query="implement authentication",
    mode="fast"  # or "agentic"
)
```

**Режимы поиска:**
- `fast` — embeddings-based matching
- `agentic` — полный анализ Space агентом

**Что можно взять:**
- [ ] Notion-like организация знаний
- [ ] Два режима поиска (быстрый vs глубокий)
- [ ] Привязка сессий к пространствам

## Архитектурные решения для изучения

### Backend (Go)

Репозиторий: `src/` содержит Go backend

**Изучить:**
1. Структуру API (`/api/v1/`)
2. Реализацию background agents
3. Механизм task extraction
4. Storage layer для sessions/skills

### Dashboard (TypeScript)

Изучить:
1. Real-time updates для task progress
2. Визуализация Skills/SOPs
3. Session timeline

## План интеграции

### Фаза 1: Заимствование подходов (v4.3.x)

- [ ] Добавить Task структуру для GitLab операций
- [ ] Реализовать progress tracking для долгих операций (indexing, test gen)
- [ ] Структурированное хранение "что работает" для каждого проекта

### Фаза 2: Self-Learning lite (v5.x)

- [ ] Сохранение успешных паттернов генерации тестов
- [ ] Накопление code review preferences
- [ ] Project-specific промпт customization

### Фаза 3: Полная интеграция (v6.x)

- [ ] Интеграция с Acontext как external service
- [ ] Или: форк ключевых компонентов
- [ ] Multi-project knowledge sharing

## Конкретные файлы для изучения

После клонирования репозитория:

```bash
git clone https://github.com/memodb-io/Acontext.git
```

Приоритетные файлы:

| Путь | Что смотреть |
|------|--------------|
| `src/api/` | REST API структура |
| `src/services/task_extractor/` | Алгоритм извлечения задач |
| `src/services/learning/` | Self-learning механизм |
| `src/models/sop.go` | Структура SOP |
| `src/storage/` | Как хранят sessions/skills |

## Риски и ограничения

1. **Молодой проект** — API может меняться
2. **Дополнительная зависимость** — ещё один сервис для деплоя
3. **Overhead** — для простых сценариев может быть избыточно
4. **Privacy** — данные сессий могут содержать sensitive info

## Альтернативы

- **LangMem** — memory для LangChain
- **MemGPT** — persistent memory agents
- **Своя реализация** — минимальный набор фич

## Следующие шаги

1. [ ] Склонировать репозиторий и изучить код
2. [ ] Протестировать локально с нашими сценариями
3. [ ] Оценить сложность интеграции
4. [ ] Принять решение: интегрировать / форкнуть / реализовать своё

## Ссылки

- [GitHub](https://github.com/memodb-io/Acontext)
- [Documentation](https://docs.acontext.io)
- [Discord](https://discord.acontext.io)


# Dependency Scanner

Автоматическая проверка зависимостей проекта на обновления и уязвимости.

## Обзор

Dependency Scanner анализирует файлы зависимостей в проиндексированном репозитории и проверяет:

1. **Актуальность версий** — сравнение с последними версиями в реестрах
2. **Безопасность** — проверка на известные CVE через [OSV.dev](https://osv.dev/)
3. **Тип обновления** — major / minor / patch классификация

## Поддерживаемые языки

| Язык | Файл зависимостей | Реестр | OSV Ecosystem |
|------|-------------------|--------|---------------|
| Go | `go.mod` | proxy.golang.org | Go |
| Node.js | `package.json` | registry.npmjs.org | npm |
| Python | `requirements.txt` | pypi.org | PyPI |

## Использование

### 1. Проиндексируйте репозиторий

Dependency Scanner работает с проиндексированным кодом. Сначала выполните индексацию:

1. Откройте **Admin → GitLab → [Ваша интеграция]**
2. Найдите проект и нажмите 🔄 **Index**
3. Дождитесь завершения индексации

### 2. Запустите проверку зависимостей

После индексации появится кнопка 📦 **Check Dependencies**:

1. Нажмите 📦 рядом с проектом
2. Дождитесь завершения сканирования
3. Просмотрите результаты в модальном окне

### 3. Интерпретация результатов

#### Summary (Сводка)

- **Total** — общее количество зависимостей
- **Direct** — прямые зависимости (не transitive)
- **Outdated** — зависимости с доступными обновлениями
- **Vulnerable** — зависимости с известными уязвимостями
- **Up to Date** — актуальные зависимости

#### Update Types (Типы обновлений)

| Тип | Описание | Риск |
|-----|----------|------|
| 🔴 **major** | Мажорное обновление (1.x → 2.x) | Высокий — возможны breaking changes |
| 🟡 **minor** | Минорное обновление (1.1 → 1.2) | Средний — новые функции |
| 🟢 **patch** | Патч (1.1.1 → 1.1.2) | Низкий — исправления багов |

#### Vulnerability Severities (Уровни уязвимостей)

| Severity | Описание |
|----------|----------|
| 🔴 **critical** | Критическая уязвимость (CVSS ≥ 9.0) |
| 🟠 **high** | Высокий риск (CVSS ≥ 7.0) |
| 🟡 **medium** | Средний риск (CVSS ≥ 4.0) |
| 🔵 **low** | Низкий риск |

### 4. Создание Issue

Для критичных обновлений можно создать GitLab Issue:

1. В результатах сканирования нажмите **Create Issue**
2. Issue будет создана в GitLab с детальным описанием
3. Включает: список уязвимых зависимостей, CVE IDs, рекомендации

## API

### Check Dependencies

```bash
POST /api/admin/gitlab/projects/:id/check-dependencies
Authorization: Bearer <token>
```

**Response:**

```json
{
  "project_id": "abc123",
  "scan_id": "dep-20241230-120000",
  "language": "go",
  "file_path": "go.mod",
  "scanned_at": "2024-12-30T12:00:00Z",
  "duration": "2.5s",
  "dependencies": [
    {
      "dependency": {
        "name": "github.com/gin-gonic/gin",
        "current_version": "v1.9.0",
        "latest_version": "v1.10.0",
        "indirect": false,
        "update_type": "minor",
        "has_update": true
      },
      "vulnerabilities": [],
      "is_vulnerable": false
    }
  ],
  "summary": {
    "total_dependencies": 45,
    "direct_dependencies": 12,
    "outdated_count": 3,
    "vulnerable_count": 1,
    "up_to_date_count": 41,
    "by_update_type": {"minor": 2, "patch": 1},
    "by_severity": {"high": 1},
    "critical_vulns": 0
  },
  "status": "completed"
}
```

### Create Dependency Issue

```bash
POST /api/admin/gitlab/projects/:id/create-dependency-issue
Authorization: Bearer <token>
Content-Type: application/json

{
  "title": "🔒 Security: Update vulnerable dependencies",
  "description": "Found 3 vulnerabilities in project dependencies...",
  "labels": ["security", "dependencies"]
}
```

## Техническая архитектура

```
internal/gitlab/dependencies/
├── types.go                # Общие типы данных
├── scanner.go              # Главный сканер
├── parser/
│   ├── gomod.go           # Go mod parser
│   ├── npm.go             # package.json parser
│   └── pip.go             # requirements.txt parser
├── registry/
│   ├── golang.go          # proxy.golang.org client
│   ├── npm.go             # registry.npmjs.org client
│   └── pypi.go            # pypi.org client
└── security/
    └── osv.go             # OSV.dev client (CVE check)
```

### Процесс сканирования

```
1. Поиск файла зависимостей в Qdrant индексе
   ↓
2. Парсинг файла (go.mod / package.json / requirements.txt)
   ↓
3. Параллельные запросы к registry (10 concurrent)
   ↓
4. Проверка каждой зависимости в OSV.dev
   ↓
5. Агрегация результатов и summary
```

## Ограничения

- Сканируется **первый найденный** файл зависимостей (приоритет: go.mod → package.json → requirements.txt)
- Для Python поддерживается только `requirements.txt` (не `pyproject.toml`)
- Lock-файлы (`go.sum`, `package-lock.json`, `Pipfile.lock`) не анализируются
- Приватные реестры не поддерживаются

## Планы развития

- [ ] Поддержка Rust (Cargo.toml)
- [ ] Поддержка Java (pom.xml)
- [ ] Поддержка .NET (*.csproj)
- [ ] LLM анализ changelog'ов
- [ ] Breaking changes detection
- [ ] Scheduled scans (cron)
- [ ] История сканирований

## См. также

- [ROADMAP.md](../ROADMAP.md) — план развития
- [OSV.dev](https://osv.dev/) — база уязвимостей
- [Renovate](https://github.com/renovatebot/renovate) — reference tool


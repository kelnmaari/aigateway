# TODO-01: TUI Admin API Key Configuration

**Версия:** v1.4.2  
**Приоритет:** HIGH  
**Оценка:** 2 часа  
**Зависимости:** Нет  
**Статус:** 📋 Planned  
**Тип:** Technical Debt / Code Quality

---

## 📋 Описание

Убрать hardcoded admin API keys из TUI (Terminal User Interface) и загружать их из конфигурации. Сейчас в коде есть 4 TODO комментария с hardcoded ключами.

## 🎯 Цели

1. Устранить security risk (hardcoded credentials)
2. Использовать конфигурацию для admin ключа
3. Улучшить code quality
4. Подготовить TUI для production использования

## 📊 Current Issues

### Найденные TODO комментарии

**Файл:** `cmd/tui/main.go`

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

### Проблемы текущей реализации

1. **Security:** Hardcoded credentials в исходном коде
2. **Flexibility:** Невозможно изменить ключ без перекомпиляции
3. **Environment:** Один ключ для всех окружений (dev, staging, prod)
4. **Best Practices:** Нарушение принципа конфигурации через env/config

## 🔧 Solution Design

### 1. Config Structure

Admin API key уже присутствует в конфигурации:

**Файл:** `internal/config/config.go`

```go
type AuthConfig struct {
    Enabled     bool   `mapstructure:"enabled"`
    StorageType string `mapstructure:"storage_type"`
    StoragePath string `mapstructure:"storage_path"`
    AdminKey    string `mapstructure:"admin_key"`  // <-- Уже есть!
    // ...
}
```

**Файлы конфигурации:**

- `configs/dev.yaml`
- `configs/production.yaml.example`

```yaml
auth:
  enabled: true
  admin_key: "your-secure-admin-key-here"
  # ...
```

### 2. TUI Application Structure

**Текущая структура:**

```go
// cmd/tui/main.go

var (
    // Глобальные переменные
    adminAPIKey = "sk-admin-dev-key-12345" // <-- Проблема!
)

func main() {
    // Инициализация
    // ...
}
```

**Новая структура:**

```go
// cmd/tui/main.go

type TUIApp struct {
    config     *config.Config
    serverURL  string
    adminToken string  // Загружается из config
}

func NewTUIApp(configPath string) (*TUIApp, error) {
    cfg, err := config.Load(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }
    
    return &TUIApp{
        config:     cfg,
        serverURL:  fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port),
        adminToken: cfg.Auth.AdminKey,
    }, nil
}
```

### 3. Implementation Steps

#### Step 1: Refactor TUI Structure (45 мин)

**Создать struct для TUI app:**

```go
// cmd/tui/main.go

type TUIApp struct {
    config     *config.Config
    serverURL  string
    adminToken string
    model      *AppModel
}

type AppModel struct {
    // Существующие поля
    activeTab   int
    // ...
    
    // Добавить:
    serverURL   string
    adminToken  string
}

func NewTUIApp(configPath string) (*TUIApp, error) {
    // Load configuration
    cfg, err := config.Load(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }
    
    // Validate admin key
    if cfg.Auth.AdminKey == "" {
        return nil, fmt.Errorf("admin_key not configured in auth section")
    }
    
    app := &TUIApp{
        config:     cfg,
        serverURL:  fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port),
        adminToken: cfg.Auth.AdminKey,
    }
    
    return app, nil
}
```

#### Step 2: Update HTTP Requests (30 мин)

**Создать helper функцию для API requests:**

```go
// cmd/tui/main.go

// makeAPIRequest создает HTTP запрос с admin авторизацией
func (app *TUIApp) makeAPIRequest(method, path string, body io.Reader) (*http.Request, error) {
    url := app.serverURL + path
    req, err := http.NewRequest(method, url, body)
    if err != nil {
        return nil, err
    }
    
    // Добавить Authorization header
    req.Header.Set("Authorization", "Bearer "+app.adminToken)
    req.Header.Set("Content-Type", "application/json")
    
    return req, nil
}

// Использование:
func (app *TUIApp) fetchModels() ([]Model, error) {
    req, err := app.makeAPIRequest("GET", "/api/admin/models", nil)
    if err != nil {
        return nil, err
    }
    
    resp, err := http.DefaultClient.Do(req)
    // ...
}
```

#### Step 3: Replace All Hardcoded Keys (30 мин)

**Найти и заменить все места:**

```bash
# Поиск всех Bearer hardcodes
grep -n "Bearer sk-admin" cmd/tui/main.go
grep -n "Bearer.*adminAPIKey" cmd/tui/main.go
```

**Заменить на:**

```go
// Было:
req.Header.Set("Authorization", "Bearer sk-admin-dev-key-12345")

// Стало:
req.Header.Set("Authorization", "Bearer "+app.adminToken)
// или:
req, err := app.makeAPIRequest("GET", "/api/admin/models", nil)
```

#### Step 4: Update Main Function (15 мин)

```go
func main() {
    // Parse flags
    configPath := flag.String("config", "configs/dev.yaml", "Path to config file")
    flag.Parse()
    
    // Initialize TUI app with config
    app, err := NewTUIApp(*configPath)
    if err != nil {
        log.Fatalf("Failed to initialize TUI: %v", err)
    }
    
    // Run Bubble Tea program
    p := tea.NewProgram(initialModel(app))
    if _, err := p.Run(); err != nil {
        log.Fatalf("Error running program: %v", err)
    }
}

func initialModel(app *TUIApp) AppModel {
    return AppModel{
        activeTab:   0,
        serverURL:   app.serverURL,
        adminToken:  app.adminToken,
        // ... other fields
    }
}
```

## 📝 Detailed Changes

### Files to Modify

1. **cmd/tui/main.go**
   - Добавить TUIApp struct
   - Загрузка конфигурации
   - Рефакторинг global variables
   - Обновить все HTTP requests

### Code Locations

**Строка ~1991:**

```go
// До:
req.Header.Set("Authorization", "Bearer sk-admin-dev-key-12345")

// После:
req.Header.Set("Authorization", "Bearer "+m.adminToken)
```

**Строка ~2265:**

```go
// До:
req.Header.Set("Authorization", "Bearer "+adminAPIKey)

// После:
req.Header.Set("Authorization", "Bearer "+m.adminToken)
```

**Строка ~2322:**

```go
// До:
req.Header.Set("Authorization", "Bearer "+adminAPIKey)

// После:
req.Header.Set("Authorization", "Bearer "+m.adminToken)
```

**Строка ~2351:**

```go
// Удалить TODO комментарий
// AdminToken теперь в AppModel
```

## ✅ Acceptance Criteria

- [ ] Нет hardcoded API keys в cmd/tui/main.go
- [ ] Admin key загружается из конфигурации
- [ ] TUI работает с разными конфигами (dev, production)
- [ ] Ошибка при отсутствии admin_key в конфиге
- [ ] Все HTTP requests используют token из config
- [ ] TODO комментарии удалены
- [ ] Code review passed
- [ ] TUI тестирование пройдено

## 🧪 Testing Plan

### Manual Testing

1. **Test с dev config:**

   ```bash
   go run cmd/tui/main.go -config configs/dev.yaml
   ```

   - Проверить подключение к серверу
   - Проверить загрузку моделей
   - Проверить API keys management

2. **Test без admin_key в конфиге:**
   - Должна быть ошибка при запуске
   - Понятное сообщение об ошибке

3. **Test с неверным admin_key:**
   - TUI запускается
   - API requests возвращают 401 Unauthorized
   - Сообщение об ошибке в UI

### Integration Testing

```go
// cmd/tui/main_test.go

func TestNewTUIApp(t *testing.T) {
    tests := []struct {
        name        string
        configPath  string
        expectError bool
    }{
        {
            name:        "valid config",
            configPath:  "../../configs/dev.yaml",
            expectError: false,
        },
        {
            name:        "missing config",
            configPath:  "nonexistent.yaml",
            expectError: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            app, err := NewTUIApp(tt.configPath)
            if tt.expectError {
                assert.Error(t, err)
                assert.Nil(t, app)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, app)
                assert.NotEmpty(t, app.adminToken)
            }
        })
    }
}
```

## 📚 References

- [Config Priority Documentation](../CONFIG_PRIORITY.md)
- [12-Factor App: Config](https://12factor.net/config)
- Internal config package: `internal/config/config.go`

## 🔄 Follow-up Tasks

- [ ] Рассмотреть использование environment variables для admin_key
- [ ] Добавить token refresh mechanism (если требуется)
- [ ] Документировать TUI configuration options
- [ ] Добавить validation для admin_key формата

## 🔒 Security Considerations

- Admin key не должен логироваться
- Конфиг файлы с real keys не коммитить в git
- Использовать `.gitignore` для production configs
- Рассмотреть encryption at rest для config файлов


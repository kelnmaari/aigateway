# 🌐 Ollama Proxy WebUI Documentation

## Overview

**Integrated WebUI** для Ollama-OpenAI Proxy с полной поддержкой user authentication, multi-tenancy, и ChatGPT-like интерфейсом.

**Version:** 1.3.0  
**Access:** `http://localhost:8080/` (интегрирован в main server)

---

## 🎯 Key Features

### ✅ Integrated Architecture
- **Встроен в main server** - работает на том же порту (8080)
- **Единая кодовая база** - нет отдельного сервера
- **Shared authentication** - использует те же JWT tokens
- **Real-time updates** - WebSocket support для live данных

### 🔐 Security First
- **Bootstrap System** - первичная настройка через admin token
- **JWT Authentication** - access/refresh tokens
- **Auth Guard** - все страницы защищены
- **Multi-Tenancy** - личные и организационные workspace

### 💬 Chat Interface
- **ChatGPT-like UI** - знакомый интерфейс
- **Real-time streaming** - ответы появляются сразу
- **Markdown support** - форматирование с syntax highlighting
- **Conversation management** - сохранение и загрузка диалогов

### 👤 User Management
- **User Registration** - самостоятельная регистрация
- **Profile Management** - редактирование профиля
- **Password Change** - смена пароля
- **Account Deletion** - удаление аккаунта

### 🏢 Multi-Tenancy
- **Personal Workspace** - автоматически для каждого пользователя
- **Organization Tenants** - создание команд
- **Role-Based Access Control** - Owner/Admin/Member/Viewer
- **Member Management** - добавление и удаление участников

### 🔑 API Key Management
- **Personal API Keys** - для личного использования
- **Tenant API Keys** - для организаций
- **Granular Permissions** - контроль доступа к моделям
- **Rate Limiting** - настройка лимитов

---

## 🚀 Quick Start

### First Time Setup (Bootstrap)

1. **Запустите сервер:**
   ```bash
   cd ollama-openai-proxy
   go run ./cmd/server
   ```

2. **Откройте браузер:**
   ```
   http://localhost:8080/
   ```

3. **Bootstrap процесс:**
   - Вы автоматически будете перенаправлены на `/bootstrap.html`
   - Введите **Bootstrap Token** (из `configs/dev.yaml`: `auth.admin_key`)
   - Создайте первого администратора:
     - Username (3-32 символов, alphanumeric, _, -)
     - Email
     - Password (8+ символов, uppercase, lowercase, digit, special char)
     - Display Name (опционально)
   - Нажмите **"Initialize System"**

4. **Вход в систему:**
   - После успешной инициализации вы будете перенаправлены на `/login.html`
   - Войдите с созданным аккаунтом администратора
   - Получите JWT tokens (сохраняются в localStorage)
   - Перейдете на `/index.html` (chat interface)

### Regular User Flow

1. **Регистрация:**
   - Зайдите на `http://localhost:8080/register.html`
   - Заполните форму регистрации
   - Автоматический вход после регистрации

2. **Вход:**
   - Зайдите на `http://localhost:8080/` (redirect на login)
   - Введите username/email и password
   - После входа redirect на chat interface

3. **Использование:**
   - **Chat** (`/index.html`) - общение с моделями
   - **Dashboard** (`/dashboard.html`) - обзор аккаунта
   - **Profile** (`/profile.html`) - настройки профиля

---

## 📁 Pages Overview

### Public Pages (No Auth Required)

| Page | Path | Description |
|------|------|-------------|
| **Bootstrap** | `/bootstrap.html` | Первичная настройка системы (только для первого запуска) |
| **Login** | `/login.html` или `/` | Вход в систему (default page) |
| **Register** | `/register.html` | Регистрация нового пользователя |

### Protected Pages (Auth Required)

| Page | Path | Description |
|------|------|-------------|
| **Chat** | `/index.html` | ChatGPT-like интерфейс для общения с моделями |
| **Dashboard** | `/dashboard.html` | Главная панель: статистика, tenants, recent conversations |
| **Profile** | `/profile.html` | Настройки профиля, смена пароля, удаление аккаунта |
| **Tenants** | `/tenants.html` | Управление организациями (planned) |
| **API Keys** | `/api-keys.html` | Управление API ключами (planned) |
| **Usage** | `/usage.html` | Статистика использования API (planned) |

---

## 🔐 Authentication Flow

### Bootstrap Flow (First Time)
```
Browser → http://localhost:8080/
    ↓
Auth Guard: No token → redirect /login.html
    ↓
Login: Check init-status → requires_bootstrap → redirect /bootstrap.html
    ↓
Bootstrap: Enter admin_token + create superadmin
    ↓
POST /api/system/bootstrap → Create first user
    ↓
Success → redirect /login.html
    ↓
Login with new admin credentials
    ↓
POST /api/auth/login → Get JWT tokens
    ↓
Save tokens → redirect /index.html (chat)
```

### Normal Login Flow
```
Browser → http://localhost:8080/
    ↓
Auth Guard: No token → redirect /login.html
    ↓
Login: Check init-status → initialized ✓
    ↓
Enter credentials
    ↓
POST /api/auth/login → Get JWT tokens
    ↓
Save tokens (localStorage) → redirect /index.html
```

### Auth Guard Protection
```javascript
Every protected page load:
    ↓
Load /js/auth-guard.js FIRST (before other scripts)
    ↓
Check localStorage.access_token
    ↓
IF NO TOKEN:
    Save return_url → redirect /login.html
    ↓
IF TOKEN EXISTS:
    Allow page to load
    ↓
Periodic check every 60 seconds
    If token lost → redirect /login.html
```

---

## 🎨 UI Components

### Top Navigation (Dashboard/Profile pages)
- **Brand** - Logo and app name
- **Nav Menu** - Chat / Dashboard tabs
- **User Dropdown** - Profile, Usage, Logout

### Sidebar (Chat page)
- **New Chat Button**
- **Conversations List** - saved chats
- **User Info** - current user
- **Logout Button**

### Chat Interface
- **Message List** - user/assistant messages
- **Markdown Rendering** - formatted text, code blocks, tables
- **Code Highlighting** - syntax highlighting (highlight.js)
- **Streaming** - real-time responses with typing indicator
- **Model Selector** - choose AI model

### Dashboard
- **Stats Cards** - conversations, tenants, API keys, requests
- **Quick Actions** - shortcuts to common tasks
- **Recent Activity** - last conversations
- **Tenants Grid** - your organizations

---

## 🔧 Configuration

### Server Config (`configs/dev.yaml`)

```yaml
server:
  host: "0.0.0.0"
  port: 8080

auth:
  enabled: true
  admin_key: "sk-admin-dev-key-12345"  # Bootstrap token
  
  jwt:
    secret: "dev-jwt-secret-change-this-in-production-use-at-least-32-chars"
    access_token_expiry: "15m"
    refresh_token_expiry: "168h"  # 7 days

database:
  type: "sqlite"
  sqlite:
    path: "./data/proxy.db"
    journal_mode: "WAL"
```

### Environment Variables

```bash
# Override config via environment
export PROXY_SERVER_PORT=8080
export PROXY_AUTH_ENABLED=true
export PROXY_AUTH_JWT_SECRET="your-secret-here"
export PROXY_DATABASE_TYPE="sqlite"
```

---

## 🛡️ Security Features

### Bootstrap Security
- ✅ **One-time setup** - только если нет пользователей
- ✅ **Admin token required** - защита от неавторизованной инициализации
- ✅ **Strong password enforcement** - валидация требований
- ✅ **Auto-verify first admin** - первый админ автоматически verified

### Authentication Security
- ✅ **JWT tokens** - access (15min) + refresh (7 days)
- ✅ **Token blacklist** - logout revokes tokens
- ✅ **Auto-refresh** - seamless token renewal
- ✅ **Bcrypt hashing** - cost factor 12
- ✅ **Password validation** - strength requirements

### Page Security
- ✅ **Auth Guard** - защита всех страниц
- ✅ **Token validation** - проверка на каждой загрузке
- ✅ **Periodic checks** - проверка каждые 60 секунд
- ✅ **Return URL** - возврат после login

### API Security
- ✅ **JWT middleware** - проверка на всех endpoints
- ✅ **Role-based access** - RBAC для tenants
- ✅ **Permission checks** - granular permissions
- ✅ **Rate limiting** - защита от abuse

---

## 📊 API Endpoints

### System
- `GET /api/system/init-status` - check if bootstrapped
- `POST /api/system/bootstrap` - first-time setup

### Authentication
- `POST /api/auth/register` - user registration
- `POST /api/auth/login` - user login
- `POST /api/auth/logout` - logout (revoke token)
- `POST /api/auth/refresh` - refresh access token
- `GET /api/auth/me` - current user info

### User Management
- `GET /api/users/me` - user profile
- `PUT /api/users/me` - update profile
- `DELETE /api/users/me` - delete account
- `POST /api/users/me/password` - change password
- `GET /api/users/me/tenants` - list user's tenants
- `GET /api/users/me/api-keys` - list personal API keys
- `POST /api/users/me/api-keys` - create personal API key

### Tenant Management
- `POST /api/tenants` - create organization
- `GET /api/tenants/:id` - get tenant details
- `PUT /api/tenants/:id` - update tenant
- `DELETE /api/tenants/:id` - delete tenant (owner only)
- `GET /api/tenants/:id/members` - list members
- `POST /api/tenants/:id/members` - add member
- `PUT /api/tenants/:id/members/:user_id` - update role
- `DELETE /api/tenants/:id/members/:user_id` - remove member

### OpenAI Compatible API
- `GET /v1/models` - list available models
- `POST /v1/chat/completions` - chat with models (streaming support)
- `POST /v1/embeddings` - text embeddings
- `POST /v1/completions` - legacy text completion

---

## 🗂️ File Structure

```
/web/                         # WebUI files (served at root)
├── index.html               # Chat interface
├── login.html               # Login page
├── register.html            # Registration
├── bootstrap.html           # First-time setup
├── dashboard.html           # Dashboard
├── profile.html             # User profile
├── css/
│   ├── style.css           # Main styles
│   └── dashboard.css       # Dashboard-specific styles
└── js/
    ├── auth-guard.js       # Page protection
    ├── api.js              # API client
    ├── chat.js             # Chat logic
    ├── app.js              # Main app logic
    ├── dashboard.js        # Dashboard logic
    └── profile.js          # Profile logic

/internal/api/              # Backend
├── handlers/
│   ├── system.go          # Bootstrap endpoints
│   ├── auth.go            # Auth endpoints
│   ├── user.go            # User management
│   ├── tenant.go          # Tenant management
│   └── ...
└── router/
    └── router.go          # Static file serving + routes

/internal/auth/
├── jwt/                   # JWT token management
├── password/              # Password hashing
├── middleware/            # Auth middleware
└── service/
    ├── auth_service.go    # Auth business logic
    └── bootstrap_service.go # Bootstrap logic

/internal/storage/
├── sqlite/                # SQLite implementation
├── postgresql/            # PostgreSQL implementation
└── interfaces.go          # Database interfaces

/cmd/server/
└── main.go               # Server entry point
```

---

## 🚨 Troubleshooting

### "System requires bootstrap" redirect loop

**Problem:** Постоянный redirect на `/bootstrap.html`

**Solution:**
1. Проверьте что database настроена: `ls data/proxy.db`
2. Проверьте логи: `tail -f logs/proxy-dev.log`
3. Убедитесь что bootstrap token правильный (из `configs/dev.yaml`)

### "Invalid token" errors

**Problem:** Постоянные 401 Unauthorized

**Solution:**
1. Очистите localStorage: Developer Tools → Application → Local Storage → Clear
2. Logout и login заново
3. Проверьте что JWT secret не менялся в конфиге

### CSS/JS files not loading

**Problem:** 404 на `/css/style.css` или `/js/api.js`

**Solution:**
1. Проверьте что файлы есть: `ls web/css/ web/js/`
2. Проверьте права доступа
3. Убедитесь что server запущен из правильной директории (корень проекта)

### Bootstrap page shows but button doesn't work

**Problem:** Кнопка "Initialize System" не отвечает

**Solution:**
1. Откройте Browser Console (F12)
2. Проверьте JavaScript errors
3. Убедитесь что backend запущен: `curl http://localhost:8080/health`
4. Проверьте network tab - должен быть POST /api/system/bootstrap

---

## 📈 Performance

### Frontend Optimizations
- ✅ **Minimal JavaScript** - vanilla JS, no heavy frameworks
- ✅ **CDN libraries** - marked.js, highlight.js from CDN
- ✅ **CSS optimization** - single stylesheet, minified in production
- ✅ **Lazy loading** - components load on demand

### Backend Optimizations
- ✅ **Connection pooling** - database connection reuse
- ✅ **Static file caching** - browser caching headers
- ✅ **JWT caching** - efficient token validation
- ✅ **Database indexes** - optimized queries

---

## 🔮 Roadmap

- [x] Bootstrap system
- [x] User authentication
- [x] Chat interface
- [x] Dashboard
- [x] Profile management
- [ ] Tenant management UI
- [ ] API Keys dashboard
- [ ] Usage statistics & billing
- [ ] Admin panel
- [ ] WebSocket live updates
- [ ] File uploads (images, documents)
- [ ] Conversation sharing
- [ ] Themes (light/dark mode toggle)
- [ ] Mobile app (PWA)

---

## 📚 Migration from Old WebUI

### Old WebUI (cmd/webui - DEPRECATED)
```
✗ Separate server on port 8081
✗ No user authentication
✗ Admin key only
✗ Static metrics dashboard
✗ No chat interface
```

### New WebUI (integrated - CURRENT)
```
✓ Integrated into main server (port 8080)
✓ Full user authentication (JWT)
✓ Multi-tenancy support
✓ ChatGPT-like interface
✓ Modern responsive design
✓ Real-time features
```

### Migration Steps

1. **Stop old WebUI:**
   ```bash
   # Kill old webui process if running
   pkill -f webui
   ```

2. **Use new integrated WebUI:**
   ```bash
   # Just run main server
   go run ./cmd/server
   ```

3. **Access WebUI:**
   ```
   Old: http://localhost:8081/     # DEPRECATED
   New: http://localhost:8080/     # USE THIS
   ```

---

## 💡 Best Practices

### Development
- Use `configs/dev.yaml` for development
- Enable debug logging: `logging.level: "debug"`
- Use SQLite for local development: `database.type: "sqlite"`
- Change JWT secret from default

### Production
- Use `configs/production.yaml`
- Change admin_key to strong random value
- Use PostgreSQL: `database.type: "postgresql"`
- Enable TLS/SSL
- Set strong JWT secret (32+ characters)
- Use environment variables for secrets
- Enable rate limiting
- Configure CORS properly
- Use reverse proxy (nginx/caddy)
- Enable audit logging

---

## 🤝 Contributing

WebUI is part of the main Ollama-OpenAI Proxy project.

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

---

## 📄 License

See [LICENSE](LICENSE) in project root.


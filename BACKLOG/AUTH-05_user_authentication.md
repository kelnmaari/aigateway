# AUTH-05: User Authentication & Multi-Tenancy System

**Версия:** 1.0  
**Статус:** 📋 Запланирована  
**Приоритет:** HIGH  
**Оценка времени:** 12-15 часов  
**Зависимости:** DB-01 (Database Abstraction Layer)

---

## 📋 Описание

Реализация полноценной системы аутентификации пользователей с поддержкой multi-tenancy. Гибридная модель: каждый пользователь имеет личный workspace + может создавать/присоединяться к общим tenants.

---

## 🎯 Цели

1. **User Management**: Регистрация, логин, управление профилем
2. **Multi-Tenancy**: Гибридная модель (личные + общие workspaces)
3. **Role-Based Access Control**: Roles и permissions для tenants
4. **API Key Scoping**: Personal и tenant-level API keys
5. **Security**: Безопасное хранение паролей, защита от брутфорса

---

## 📊 Модель данных

### User (Пользователь)

```go
type User struct {
    ID           string    `json:"id" db:"id"`
    Username     string    `json:"username" db:"username"`           // Unique, 3-32 chars
    Email        string    `json:"email" db:"email"`                 // Unique
    PasswordHash string    `json:"-" db:"password_hash"`             // Never expose
    DisplayName  string    `json:"display_name" db:"display_name"`   // Optional
    Avatar       string    `json:"avatar,omitempty" db:"avatar"`     // Optional URL/path
    Preferences  string    `json:"preferences" db:"preferences"`     // JSON: theme, language, etc
    Status       UserStatus `json:"status" db:"status"`              // active/disabled/suspended
    CreatedAt    time.Time `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
    LastLoginAt  *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
}

type UserStatus string
const (
    UserStatusActive    UserStatus = "active"
    UserStatusDisabled  UserStatus = "disabled"
    UserStatusSuspended UserStatus = "suspended"
)
```

### Tenant (Организация/Workspace)

```go
type Tenant struct {
    ID          string    `json:"id" db:"id"`
    Name        string    `json:"name" db:"name"`                   // "Моя Компания"
    Slug        string    `json:"slug" db:"slug"`                   // URL-friendly, unique
    Description string    `json:"description,omitempty" db:"description"`
    OwnerID     string    `json:"owner_id" db:"owner_id"`           // User.ID
    Type        TenantType `json:"type" db:"type"`                  // personal/organization
    Settings    string    `json:"settings" db:"settings"`           // JSON config
    Status      TenantStatus `json:"status" db:"status"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type TenantType string
const (
    TenantTypePersonal     TenantType = "personal"      // Auto-created для каждого user
    TenantTypeOrganization TenantType = "organization"  // Создаваемые вручную
)

type TenantStatus string
const (
    TenantStatusActive   TenantStatus = "active"
    TenantStatusArchived TenantStatus = "archived"
)

// TenantSettings (JSON в поле settings)
type TenantSettings struct {
    AllowedModels    []string          `json:"allowed_models"`     // Whitelist моделей
    DefaultModel     string            `json:"default_model"`      // Default для chat
    RateLimits       RateLimits        `json:"rate_limits"`        // Tenant-level limits
    Features         map[string]bool   `json:"features"`           // Feature flags
}
```

### TenantMember (Участник тенанта)

```go
type TenantMember struct {
    ID          string       `json:"id" db:"id"`
    TenantID    string       `json:"tenant_id" db:"tenant_id"`
    UserID      string       `json:"user_id" db:"user_id"`
    Role        TenantRole   `json:"role" db:"role"`
    Permissions string       `json:"permissions" db:"permissions"`    // JSON array
    InvitedBy   string       `json:"invited_by,omitempty" db:"invited_by"` // User.ID
    JoinedAt    time.Time    `json:"joined_at" db:"joined_at"`
    UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}

type TenantRole string
const (
    TenantRoleOwner  TenantRole = "owner"   // Full control
    TenantRoleAdmin  TenantRole = "admin"   // Manage members, keys
    TenantRoleMember TenantRole = "member"  // Use resources
    TenantRoleViewer TenantRole = "viewer"  // Read-only
)
```

### APIKey (Обновленная модель)

```go
type APIKey struct {
    // ... existing fields ...
    
    // NEW: Scoping
    Type     APIKeyType `json:"type" db:"type"`                // personal/tenant
    OwnerID  *string    `json:"owner_id,omitempty" db:"owner_id"`   // User.ID (для personal)
    TenantID *string    `json:"tenant_id,omitempty" db:"tenant_id"` // Tenant.ID (для tenant)
    
    // Логика:
    // - Personal key: OwnerID = userID, TenantID = NULL
    // - Tenant key: OwnerID = NULL, TenantID = tenantID
}

type APIKeyType string
const (
    APIKeyTypePersonal APIKeyType = "personal"  // Личный ключ пользователя
    APIKeyTypeTenant   APIKeyType = "tenant"    // Ключ организации
)
```

---

## 🔐 Authentication Flow

### Registration

```
1. POST /api/auth/register
   {
     "username": "ivan",
     "email": "ivan@example.com",
     "password": "SecurePass123!"
   }

2. Валидация:
   - Username: 3-32 chars, alphanumeric + _-
   - Email: valid format, unique
   - Password: min 8 chars, complexity rules

3. Создаем:
   - User record (password → bcrypt hash)
   - Personal Tenant "ivan's Workspace" (auto)
   - TenantMember (user → personal tenant, role=owner)

4. Response:
   {
     "user": {...},
     "token": "jwt_token_here",
     "personal_tenant": {...}
   }
```

### Login

```
POST /api/auth/login
{
  "username": "ivan",    // or "email": "ivan@example.com"
  "password": "SecurePass123!"
}

Response:
{
  "user": {...},
  "token": "jwt_token_here",
  "tenants": [...]  // List of user's tenants
}
```

### JWT Token Structure

```go
type JWTClaims struct {
    UserID    string   `json:"user_id"`
    Username  string   `json:"username"`
    Email     string   `json:"email"`
    TenantIDs []string `json:"tenant_ids"`  // All accessible tenants
    IssuedAt  int64    `json:"iat"`
    ExpiresAt int64    `json:"exp"`
}
```

---

## 🏢 Tenant Management

### Create Organization Tenant

```
POST /api/tenants
Authorization: Bearer <jwt_token>
{
  "name": "Моя Компания",
  "description": "Рабочее пространство нашей команды"
}

Создается:
- Tenant (type=organization)
- TenantMember (creator → tenant, role=owner)
```

### Invite User to Tenant

```
POST /api/tenants/:tenant_id/members
Authorization: Bearer <jwt_token>
{
  "user_id": "user_123",
  "role": "member"
}

Требования:
- Requester должен быть owner/admin тенанта
- User существует
- User еще не member этого тенанта
```

### List User's Tenants

```
GET /api/users/me/tenants
Authorization: Bearer <jwt_token>

Response:
{
  "tenants": [
    {
      "id": "tenant_personal_123",
      "name": "Ivan's Workspace",
      "type": "personal",
      "role": "owner",
      ...
    },
    {
      "id": "tenant_org_456",
      "name": "Моя Компания",
      "type": "organization",
      "role": "admin",
      ...
    }
  ]
}
```

---

## 🔑 API Key Scoping

### Create Personal API Key

```
POST /api/users/me/api-keys
Authorization: Bearer <jwt_token>
{
  "name": "My Dev Key",
  "models": ["gpt-4", "gpt-3.5"],
  "rate_limits": {
    "requests_per_minute": 60
  }
}

Создается:
- APIKey (type=personal, owner_id=user_id, tenant_id=NULL)
```

### Create Tenant API Key

```
POST /api/tenants/:tenant_id/api-keys
Authorization: Bearer <jwt_token>
{
  "name": "Production API",
  "models": ["*"],
  "rate_limits": {
    "requests_per_minute": 100
  }
}

Требования:
- Requester должен быть admin/owner тенанта

Создается:
- APIKey (type=tenant, owner_id=NULL, tenant_id=tenant_id)
```

### API Key Validation

```go
func (m *APIKeyManager) ValidateKey(ctx context.Context, keyStr string) (*APIKey, *User, *Tenant, error) {
    // 1. Находим ключ по hash
    key, err := m.storage.GetAPIKeyByHash(ctx, hash(keyStr))
    
    // 2. Проверяем статус, expiration
    if key.Status != APIKeyStatusActive { return ErrKeyInactive }
    if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) { return ErrKeyExpired }
    
    // 3. Загружаем owner или tenant
    if key.Type == APIKeyTypePersonal {
        user, _ := m.storage.GetUser(ctx, *key.OwnerID)
        return key, user, nil, nil
    } else {
        tenant, _ := m.storage.GetTenant(ctx, *key.TenantID)
        return key, nil, tenant, nil
    }
}
```

---

## 🛡️ Security Measures

### Password Requirements

- Minimum 8 characters
- At least 1 uppercase letter
- At least 1 lowercase letter
- At least 1 digit
- At least 1 special character (optional but recommended)

### Password Hashing

```go
import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
    // Cost 12 - баланс между безопасностью и производительностью
    return bcrypt.GenerateFromPassword([]byte(password), 12)
}

func CheckPassword(hashedPassword, password string) error {
    return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
```

### Brute Force Protection

```go
// Rate limiting на login endpoint
- 5 попыток за 15 минут с одного IP
- После 5 неудачных попыток - блокировка на 15 минут
- После 10 неудачных попыток - блокировка на 1 час

// Опционально: CAPTCHA после 3 неудачных попыток
```

### JWT Security

- Short-lived access tokens (15 минут)
- Refresh tokens (7 дней)
- Token rotation при refresh
- Blacklist для revoked tokens (Redis/in-memory)

---

## 📁 Database Schema (SQLite/PostgreSQL)

```sql
-- Users table
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    display_name TEXT,
    avatar TEXT,
    preferences TEXT DEFAULT '{}',
    status TEXT DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_login_at DATETIME
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);

-- Tenants table
CREATE TABLE tenants (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    description TEXT,
    owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT DEFAULT 'organization',
    settings TEXT DEFAULT '{}',
    status TEXT DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_owner_id ON tenants(owner_id);
CREATE INDEX idx_tenants_type ON tenants(type);

-- Tenant members table
CREATE TABLE tenant_members (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    permissions TEXT DEFAULT '[]',
    invited_by TEXT REFERENCES users(id),
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, user_id)
);

CREATE INDEX idx_tenant_members_tenant_id ON tenant_members(tenant_id);
CREATE INDEX idx_tenant_members_user_id ON tenant_members(user_id);

-- API Keys table (обновленная)
ALTER TABLE api_keys ADD COLUMN type TEXT DEFAULT 'tenant';
ALTER TABLE api_keys ADD COLUMN owner_id TEXT REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE api_keys ADD COLUMN tenant_id TEXT REFERENCES tenants(id) ON DELETE CASCADE;

CREATE INDEX idx_api_keys_owner_id ON api_keys(owner_id);
CREATE INDEX idx_api_keys_tenant_id ON api_keys(tenant_id);
```

---

## 🔌 API Endpoints

### Authentication

- `POST /api/auth/register` - Регистрация нового пользователя
- `POST /api/auth/login` - Вход в систему
- `POST /api/auth/logout` - Выход (invalidate token)
- `POST /api/auth/refresh` - Обновление access token
- `POST /api/auth/password/change` - Смена пароля
- `GET /api/auth/me` - Текущий пользователь

### Users

- `GET /api/users/me` - Профиль текущего пользователя
- `PUT /api/users/me` - Обновление профиля
- `DELETE /api/users/me` - Удаление аккаунта
- `GET /api/users/me/tenants` - Список tenants пользователя
- `GET /api/users/me/api-keys` - Личные API keys
- `POST /api/users/me/api-keys` - Создать личный API key

### Tenants

- `GET /api/tenants` - Список tenants (где user является member)
- `POST /api/tenants` - Создать новый tenant
- `GET /api/tenants/:id` - Информация о tenant
- `PUT /api/tenants/:id` - Обновить tenant
- `DELETE /api/tenants/:id` - Удалить tenant (только owner)
- `GET /api/tenants/:id/members` - Список участников
- `POST /api/tenants/:id/members` - Пригласить участника
- `PUT /api/tenants/:id/members/:user_id` - Изменить роль
- `DELETE /api/tenants/:id/members/:user_id` - Удалить участника
- `GET /api/tenants/:id/api-keys` - Tenant API keys
- `POST /api/tenants/:id/api-keys` - Создать tenant API key

---

## 🧪 Тестирование

### Unit Tests

- Password hashing/validation
- JWT token generation/validation
- User CRUD operations
- Tenant CRUD operations
- API key scoping logic
- Permission checks

### Integration Tests

- Registration → auto-create personal tenant
- Login → return all tenants
- Create tenant → add creator as owner
- Invite user → check permissions
- API key validation → check scoping

### Security Tests

- Brute force protection
- SQL injection prevention
- XSS protection
- CSRF protection
- Password strength validation

---

## 📦 Зависимости

```go
// go.mod additions
require (
    golang.org/x/crypto v0.x.x  // bcrypt, argon2
    github.com/golang-jwt/jwt/v5 v5.x.x  // JWT
    github.com/google/uuid v1.x.x  // UUID generation
)
```

---

## 🚀 Deployment

### Environment Variables

```bash
# JWT
JWT_SECRET=<random_secret_key>
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=7d

# Security
PASSWORD_MIN_LENGTH=8
MAX_LOGIN_ATTEMPTS=5
LOGIN_RATE_LIMIT_WINDOW=15m

# Database (see DB-01)
DB_TYPE=sqlite  # or postgresql
DB_PATH=data/proxy.db
```

---

## 📝 Migration Plan

1. **Phase 1**: Database schema
   - Создать новые таблицы
   - Миграция существующих API keys (все → type=tenant)

2. **Phase 2**: Core auth
   - User registration/login
   - JWT middleware
   - Password management

3. **Phase 3**: Multi-tenancy
   - Tenant CRUD
   - Member management
   - Role-based access

4. **Phase 4**: API key scoping
   - Personal vs Tenant keys
   - Validation logic
   - Migration tool для существующих keys

---

## 🎯 Success Criteria

- ✅ User может зарегистрироваться и войти в систему
- ✅ Auto-create personal tenant при регистрации
- ✅ User может создавать organization tenants
- ✅ User может приглашать других пользователей в tenants
- ✅ RBAC работает корректно (owner/admin/member/viewer)
- ✅ Personal API keys изолированы от tenant keys
- ✅ Tenant API keys доступны всем members тенанта
- ✅ Brute force protection работает
- ✅ JWT tokens правильно генерируются и валидируются
- ✅ 100% покрытие тестами критичного функционала

---

**Автор:** AI Assistant  
**Дата создания:** 2025-10-05  
**Последнее обновление:** 2025-10-05


# AUTH-03: Invitation-Only Registration System

**Версия:** TBD (Future)  
**Приоритет:** MEDIUM  
**Оценка:** 6-8 часов  
**Зависимости:** Auth System (v1.3.0+)

---

## 📋 Описание

Система регистрации только по пригласительным ссылкам от администраторов. Админ генерирует уникальную ссылку-приглашение, передает ее будущему пользователю, после успешной регистрации ссылка становится недействительной.

### Ключевая ценность

- 🔒 **Контроль доступа**: Регистрация только для приглашенных пользователей
- 🛡️ **Безопасность**: Предотвращение публичных регистраций и спама
- 👥 **Управляемый рост**: Контролируемое расширение базы пользователей
- 📊 **Трекинг**: Кто кого пригласил и когда

---

## 🎯 Use Cases

### 1. Закрытая корпоративная установка

**Сценарий:** Компания развернула AIGateway для внутреннего использования.  
**Решение:** Только HR/IT админы могут приглашать новых сотрудников.

### 2. Beta Testing с ограниченным доступом

**Сценарий:** Запуск closed beta для тестирования новых фичей.  
**Решение:** Команда рассылает пригласительные ссылки beta-тестерам.

### 3. Community с модерацией

**Сценарий:** Сообщество AI-разработчиков с контролируемым входом.  
**Решение:** Существующие участники могут приглашать коллег.

---

## 🏗️ Архитектура

### Database Schema

```sql
-- Таблица пригласительных ссылок
CREATE TABLE invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token UUID UNIQUE NOT NULL,  -- Токен для ссылки (https://example.com/register?invite=TOKEN)
    
    -- Метаданные создания
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Ограничения приглашения
    email VARCHAR(255),              -- Опционально: привязка к конкретному email
    expires_at TIMESTAMP,            -- Опционально: срок действия
    max_uses INT NOT NULL DEFAULT 1, -- Сколько раз можно использовать (обычно 1)
    
    -- Статус использования
    current_uses INT NOT NULL DEFAULT 0,
    used_at TIMESTAMP,               -- Когда была первая успешная регистрация
    used_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    
    -- Отзыв
    revoked_at TIMESTAMP,
    revoked_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    revoke_reason TEXT,
    
    -- Индексы
    CONSTRAINT valid_uses CHECK (current_uses <= max_uses)
);

-- Индексы для performance
CREATE INDEX idx_invitations_token ON invitations(token);
CREATE INDEX idx_invitations_created_by ON invitations(created_by_user_id);
CREATE INDEX idx_invitations_status ON invitations(expires_at, revoked_at, current_uses, max_uses);
CREATE INDEX idx_invitations_email ON invitations(email) WHERE email IS NOT NULL;

-- Расширение audit_events для трекинга операций с приглашениями
-- event_type: 'invitation_created', 'invitation_used', 'invitation_revoked'
```

### Go Models

```go
// internal/models/invitation.go
package models

import (
    "time"
    "github.com/google/uuid"
)

type Invitation struct {
    ID                 uuid.UUID  `json:"id" db:"id"`
    Token              uuid.UUID  `json:"token" db:"token"`
    
    // Creation metadata
    CreatedByUserID    uuid.UUID  `json:"created_by_user_id" db:"created_by_user_id"`
    CreatedAt          time.Time  `json:"created_at" db:"created_at"`
    
    // Constraints
    Email              *string    `json:"email,omitempty" db:"email"`
    ExpiresAt          *time.Time `json:"expires_at,omitempty" db:"expires_at"`
    MaxUses            int        `json:"max_uses" db:"max_uses"`
    
    // Usage tracking
    CurrentUses        int        `json:"current_uses" db:"current_uses"`
    UsedAt             *time.Time `json:"used_at,omitempty" db:"used_at"`
    UsedByUserID       *uuid.UUID `json:"used_by_user_id,omitempty" db:"used_by_user_id"`
    
    // Revocation
    RevokedAt          *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
    RevokedByUserID    *uuid.UUID `json:"revoked_by_user_id,omitempty" db:"revoked_by_user_id"`
    RevokeReason       *string    `json:"revoke_reason,omitempty" db:"revoke_reason"`
}

// IsValid проверяет валидность приглашения
func (i *Invitation) IsValid() bool {
    now := time.Now()
    
    // Проверка отзыва
    if i.RevokedAt != nil {
        return false
    }
    
    // Проверка срока действия
    if i.ExpiresAt != nil && now.After(*i.ExpiresAt) {
        return false
    }
    
    // Проверка лимита использований
    if i.CurrentUses >= i.MaxUses {
        return false
    }
    
    return true
}

// CanBeUsedBy проверяет может ли конкретный email использовать приглашение
func (i *Invitation) CanBeUsedBy(email string) bool {
    if !i.IsValid() {
        return false
    }
    
    // Если email не указан в приглашении - может использовать любой
    if i.Email == nil {
        return true
    }
    
    // Если email указан - только этот email может использовать
    return *i.Email == email
}

type InvitationStatus string

const (
    InvitationStatusActive  InvitationStatus = "active"
    InvitationStatusUsed    InvitationStatus = "used"
    InvitationStatusExpired InvitationStatus = "expired"
    InvitationStatusRevoked InvitationStatus = "revoked"
)

// GetStatus возвращает текущий статус приглашения
func (i *Invitation) GetStatus() InvitationStatus {
    if i.RevokedAt != nil {
        return InvitationStatusRevoked
    }
    
    now := time.Now()
    if i.ExpiresAt != nil && now.After(*i.ExpiresAt) {
        return InvitationStatusExpired
    }
    
    if i.CurrentUses >= i.MaxUses {
        return InvitationStatusUsed
    }
    
    return InvitationStatusActive
}
```

---

## 🔌 API Endpoints

### Admin API

#### 1. Create Invitation

```http
POST /api/admin/invitations
Authorization: Bearer {admin_token}
Content-Type: application/json

{
  "email": "user@example.com",      // Optional: restrict to specific email
  "expires_at": "2025-12-31T23:59:59Z", // Optional: expiry date
  "max_uses": 1                      // Default: 1
}

Response 201:
{
  "id": "uuid",
  "token": "uuid",
  "invitation_url": "https://example.com/register?invite=uuid",
  "created_by_user_id": "uuid",
  "created_at": "2025-10-27T10:00:00Z",
  "email": "user@example.com",
  "expires_at": "2025-12-31T23:59:59Z",
  "max_uses": 1,
  "current_uses": 0,
  "status": "active"
}
```

#### 2. List Invitations

```http
GET /api/admin/invitations?status=active&page=1&limit=20
Authorization: Bearer {admin_token}

Response 200:
{
  "invitations": [
    {
      "id": "uuid",
      "token": "uuid",
      "created_by_user_id": "uuid",
      "created_at": "2025-10-27T10:00:00Z",
      "email": "user@example.com",
      "expires_at": "2025-12-31T23:59:59Z",
      "max_uses": 1,
      "current_uses": 0,
      "used_at": null,
      "used_by_user_id": null,
      "revoked_at": null,
      "status": "active"
    }
  ],
  "pagination": {
    "total": 42,
    "page": 1,
    "limit": 20,
    "pages": 3
  },
  "statistics": {
    "total": 42,
    "active": 15,
    "used": 20,
    "expired": 5,
    "revoked": 2
  }
}
```

#### 3. Get Invitation Details

```http
GET /api/admin/invitations/:id
Authorization: Bearer {admin_token}

Response 200:
{
  "id": "uuid",
  "token": "uuid",
  "invitation_url": "https://example.com/register?invite=uuid",
  "created_by": {
    "id": "uuid",
    "username": "admin",
    "email": "admin@example.com"
  },
  "created_at": "2025-10-27T10:00:00Z",
  "email": "user@example.com",
  "expires_at": "2025-12-31T23:59:59Z",
  "max_uses": 1,
  "current_uses": 1,
  "used_at": "2025-10-28T14:30:00Z",
  "used_by": {
    "id": "uuid",
    "username": "newuser",
    "email": "user@example.com"
  },
  "status": "used"
}
```

#### 4. Revoke Invitation

```http
DELETE /api/admin/invitations/:id
Authorization: Bearer {admin_token}
Content-Type: application/json

{
  "reason": "User no longer needs access"
}

Response 200:
{
  "message": "Invitation revoked successfully",
  "invitation_id": "uuid",
  "revoked_at": "2025-10-27T12:00:00Z"
}
```

### Public API

#### 5. Validate Invitation Token

```http
GET /api/invitations/:token/validate

Response 200 (Valid):
{
  "valid": true,
  "email_required": "user@example.com",  // If invitation is email-restricted
  "expires_at": "2025-12-31T23:59:59Z"
}

Response 400 (Invalid):
{
  "valid": false,
  "reason": "expired" | "revoked" | "used" | "not_found"
}
```

#### 6. Register with Invitation Token

```http
POST /api/auth/register
Content-Type: application/json

{
  "invitation_token": "uuid",
  "username": "newuser",
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "full_name": "New User"
}

Response 201:
{
  "message": "User registered successfully",
  "user": {
    "id": "uuid",
    "username": "newuser",
    "email": "user@example.com"
  },
  "access_token": "jwt_token",
  "refresh_token": "jwt_refresh_token"
}

Response 400:
{
  "error": "invalid_invitation",
  "message": "Invitation token is invalid or expired"
}
```

---

## 🎨 WebUI Components

### 1. Admin Panel → New Tab: "Ссылки-приглашения"

**Location:** `/admin-invitations.html`

**Features:**

- Таблица всех приглашений с колонками:
  - Status badge (Active/Used/Expired/Revoked)
  - Invitation Link (с кнопкой Copy)
  - Email (если привязано)
  - Created By
  - Created At
  - Expires At
  - Uses (1/1, 0/5, etc.)
  - Actions (View Details, Revoke)
- Фильтры:
  - Status dropdown (All/Active/Used/Expired/Revoked)
  - Date range picker
  - Search by email
- Statistics cards:
  - Total Invitations
  - Active Invitations
  - Used This Month
  - Success Rate (used/total)
- "Create Invitation" button → Modal:
  - Email field (optional)
  - Expiry date picker (optional)
  - Max uses (default: 1, можно больше)
  - Submit → Show success + Copy link

### 2. Registration Page Enhancement

**Location:** `/register.html`

**Changes:**

- Check for `?invite=TOKEN` query parameter
- If present:
  - Validate token via API
  - If valid:
    - Show "You've been invited!" banner
    - Pre-fill email if invitation is email-restricted
    - Disable email field if email-restricted
  - If invalid:
    - Show error message: "This invitation link is invalid or expired"
    - Disable registration form
    - Show "Contact administrator" message
- If not present (and invitation-only mode enabled):
  - Show message: "Registration is by invitation only"
  - Hide registration form

### 3. Admin Users Page Enhancement

**Location:** `/admin.html` (Users tab)

**New Column:** "Invited By"

- Show username of admin who created the invitation
- Link to invitation details

---

## 🔐 Security Considerations

### 1. Token Generation

- Use cryptographically secure random UUID v4
- Token должен быть unguessable (128-bit entropy)
- Store token hash in database (optional, если нужна extra security)

### 2. Rate Limiting

- Limit invitation creation: 10 per admin per hour
- Limit validation checks: 10 per IP per minute
- Limit registration attempts: 3 per token

### 3. Audit Logging

- Log all invitation operations:
  - `invitation_created` - кто, когда, для какого email
  - `invitation_used` - кто зарегистрировался, по чьему приглашению
  - `invitation_revoked` - кто отозвал, причина

### 4. Email Verification

- Если invitation привязано к email, проверять совпадение при регистрации
- Опционально: требовать email verification после регистрации по приглашению

---

## 📝 Configuration

### Config File (configs/production.yaml)

```yaml
auth:
  # User Registration (Version 2.3.0+)
  registration:
    mode: "invitation_only"  # "open" | "invitation_only" | "disabled"
    # - "open": Любой может зарегистрироваться (default для dev)
    # - "invitation_only": Только по приглашениям от админов (recommended для production)
    # - "disabled": Регистрация полностью отключена
    require_email_verification: true  # Требовать email verification
    allow_username_login: true  # Разрешить логин по username
    
  # Invitation System (Version 2.3.0+)
  invitations:
    enabled: true  # Включить систему приглашений
    default_expiry_days: 7    # Default expiry for new invitations (0 = no expiry)
    max_uses_default: 1       # Default max uses (1 = single-use)
    allow_email_restriction: true  # Можно привязать к email
    
    # Rate limits для защиты от спама
    rate_limit:
      creation_per_admin_hour: 10  # Max invitations per admin per hour
      validation_per_ip_minute: 10  # Max token validations per IP per minute
```

### Registration Modes

**1. `mode: "open"` (Open Registration)**

- Любой пользователь может зарегистрироваться без приглашения
- Система приглашений доступна, но не обязательна
- **Use Case:** Public instances, демо-серверы, development
- **Default:** для dev.yaml

**2. `mode: "invitation_only"` (Invitation-Only)**

- Регистрация возможна ТОЛЬКО по пригласительной ссылке
- `/register` без токена → показывает "Registration by invitation only"
- **Use Case:** Закрытые корпоративные инстансы, private communities
- **Recommended:** для production.yaml

**3. `mode: "disabled"` (Registration Disabled)**

- Регистрация полностью отключена
- Только существующие пользователи могут логиниться
- Админ создает аккаунты вручную через CLI/API
- **Use Case:** Фиксированный набор пользователей, maintenance mode

---

## 🧪 Testing Scenarios

### Unit Tests

- ✅ Invitation.IsValid() - различные статусы
- ✅ Invitation.CanBeUsedBy() - email restrictions
- ✅ Token generation uniqueness
- ✅ Database constraints (max_uses)

### Integration Tests

- ✅ Admin creates invitation → User registers → Invitation marked used
- ✅ Email-restricted invitation → Wrong email fails
- ✅ Expired invitation → Registration fails
- ✅ Revoked invitation → Registration fails
- ✅ Multi-use invitation → Multiple registrations succeed
- ✅ Single-use invitation → Second registration fails

### E2E Tests

- ✅ Admin flow: Create → Copy link → Revoke
- ✅ User flow: Click link → Validate → Register → Success
- ✅ Error scenarios: Expired/Revoked/Invalid tokens

---

## 📊 Metrics & Analytics

### Prometheus Metrics

```
aigateway_invitations_total{status="active|used|expired|revoked"}
aigateway_invitations_created_total
aigateway_invitations_used_total
aigateway_invitation_usage_rate (used/created)
```

### Admin Dashboard Stats

- Invitation success rate (used vs expired/revoked)
- Average time from creation to use
- Top inviters (admins who invited most users)
- Monthly invitation trends

---

## 🚀 Implementation Plan

### Phase 1: Backend (3-4 hours)

1. Database migration для таблицы invitations
2. Models (Invitation, validation methods)
3. Repository layer (CRUD operations)
4. Service layer (business logic)
5. API handlers (Admin + Public endpoints)

### Phase 2: WebUI (2-3 hours)

1. Admin Invitations page (HTML + JS)
2. Registration page modifications
3. Copy-to-clipboard functionality
4. Statistics dashboard

### Phase 3: Testing & Polish (1-2 hours)

1. Unit tests
2. Integration tests
3. Manual E2E testing
4. Documentation

---

## 🔄 Future Enhancements

- **INVITE-02** Invitation Templates: Pre-configured invitation settings
- **INVITE-03** Bulk Invitations: CSV upload для массовых приглашений
- **INVITE-04** Invitation Analytics: Dashboard с metrics
- **INVITE-05** Delegation: Non-admin users can create limited invitations
- **INVITE-06** Custom Landing Page: Персонализированная страница для приглашенных

---

## 📚 References

- [GitHub Invite System](https://docs.github.com/en/organizations/managing-membership-in-your-organization/inviting-users-to-join-your-organization)
- [Discord Invite Links](https://discord.com/developers/docs/resources/invite)
- [Slack Workspace Invitations](https://api.slack.com/methods/admin.inviteRequests.approve)

# AUTH-04: Enhanced API Key Management

**Приоритет:** HIGH  
**Версия:** 1.2.0  
**Оценка времени:** 4-5 часов  
**Зависимости:** Existing API Keys system (Phase 8)

---

## Цель

Добавить возможность редактирования, отзыва и расширенного управления существующими API ключами через TUI и WebUI.

---

## Требования

### 1. Edit API Keys (2 часа)

**Backend (`internal/auth/apikey/manager.go`):**

```go
// UpdateAPIKeyPermissions обновляет permissions и models
func (m *Manager) UpdateAPIKeyPermissions(keyID string, models []string, permissions []string) error {
    // Validate key exists
    // Update permissions
    // Update models list
    // Save to storage
}
```

**TUI:**

- Новая клавиша `e` (edit) на экране API Keys
- Interactive form для изменения:
  - Models list
  - Permissions
  - Rate limits
- Validation перед сохранением

**WebUI:**

- "Edit" кнопка для каждого ключа
- Modal с формой редактирования
- Real-time validation

### 2. Revoke/Enable Keys (1 час)

**Backend:**

```go
// RevokeAPIKey делает ключ неактивным
func (m *Manager) RevokeAPIKey(keyID string) error {
    key.Status = "revoked"
    key.RevokedAt = time.Now()
}

// EnableAPIKey активирует ключ снова  
func (m *Manager) EnableAPIKey(keyID string) error {
    key.Status = "active"
    key.RevokedAt = nil
}
```

**API Endpoints:**

```
POST /admin/api-keys/{id}/revoke
POST /admin/api-keys/{id}/enable
```

**UI:**

- Toggle button (Active/Revoked)
- Confirmation dialog
- Visual indication (red for revoked)

### 3. Key Expiration Management (1 час)

**Features:**

- Set expiration date при создании
- Extend expiration для существующих
- Auto-revoke expired keys (background job)
- Warning notifications before expiration

**Backend:**

```go
type APIKey struct {
    // ... existing fields
    ExpiresAt   *time.Time `json:"expires_at,omitempty"`
    ExpiryDays  *int       `json:"expiry_days,omitempty"`
}

// Background job
func (m *Manager) CleanupExpiredKeys() {
    // Run every hour
    // Find expired keys
    // Auto-revoke
}
```

### 4. Bulk Operations (0.5 часа)

**TUI/WebUI:**

- Multi-select keys (checkbox)
- Bulk actions:
  - Revoke selected
  - Enable selected
  - Delete selected
- Confirmation with count

---

## API Changes

### New Endpoints

```
PUT    /admin/api-keys/{id}          # Update key
PATCH  /admin/api-keys/{id}/revoke   # Revoke
PATCH  /admin/api-keys/{id}/enable   # Enable
DELETE /admin/api-keys/bulk          # Bulk delete
POST   /admin/api-keys/{id}/extend   # Extend expiration
```

### Request/Response

```json
// PUT /admin/api-keys/{id}
{
  "models": ["gpt-3.5-turbo", "gpt-4"],
  "permissions": ["chat", "embeddings"],
  "rate_limit": 100,
  "expires_in_days": 90
}

// Response
{
  "api_key": {
    "id": "ak_xxx",
    "updated_at": "2025-10-05T10:00:00Z",
    "expires_at": "2026-01-03T10:00:00Z"
  }
}
```

---

## Database Schema Changes

### Update APIKey model

```go
type APIKey struct {
    ID            string     `json:"id"`
    Name          string     `json:"name"`
    KeyHash       string     `json:"-"` // Never expose
    Status        string     `json:"status"` // active, revoked, expired
    Models        []string   `json:"models"`
    Permissions   []string   `json:"permissions"`
    RateLimit     int        `json:"rate_limit"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
    LastUsed      *time.Time `json:"last_used,omitempty"`
    ExpiresAt     *time.Time `json:"expires_at,omitempty"`
    RevokedAt     *time.Time `json:"revoked_at,omitempty"`
    RevokedReason string     `json:"revoked_reason,omitempty"`
    
    // Usage stats
    Usage         Usage      `json:"usage"`
}
```

---

## TUI Implementation

### Edit Key Flow

```
1. User on API Keys screen
2. Presses 'e' → "Enter key ID to edit:"
3. Shows current values in form
4. Tab to navigate fields
5. Enter to save
6. Confirmation message
7. List refreshes
```

### Revoke Flow

```
1. Press 'x' on key
2. Confirmation: "Revoke key 'Production API'? (y/n)"
3. If yes → POST /admin/api-keys/{id}/revoke
4. Success message
5. Key shows as [REVOKED] in red
```

---

## WebUI Implementation

### Edit Modal

```html
<div id="editKeyModal" class="modal">
    <div class="modal-content">
        <h2>Edit API Key: Production API</h2>
        
        <form id="editKeyForm">
            <label>Models (comma-separated)</label>
            <input type="text" value="gpt-3.5-turbo, gpt-4" />
            
            <label>Permissions</label>
            <div class="checkbox-group">
                <label><input type="checkbox" checked> Chat</label>
                <label><input type="checkbox" checked> Embeddings</label>
                <label><input type="checkbox"> Completions</label>
            </div>
            
            <label>Rate Limit (req/min)</label>
            <input type="number" value="100" />
            
            <label>Expiration</label>
            <select>
                <option value="30">30 days</option>
                <option value="90" selected>90 days</option>
                <option value="365">1 year</option>
                <option value="-1">Never</option>
            </select>
            
            <div class="modal-actions">
                <button type="submit">Save Changes</button>
                <button type="button" onclick="closeModal()">Cancel</button>
            </div>
        </form>
    </div>
</div>
```

---

## Testing

### Unit Tests

- [ ] UpdateAPIKeyPermissions works
- [ ] RevokeAPIKey sets status correctly
- [ ] EnableAPIKey restores key
- [ ] Expired keys auto-revoked
- [ ] Bulk operations atomic

### Integration Tests

- [ ] Edit через API endpoint
- [ ] Revoked key rejected in auth
- [ ] Enabled key works again
- [ ] Expiration checked on each request

### UI Tests

- [ ] Edit form validates input
- [ ] Revoke shows confirmation
- [ ] Bulk select works
- [ ] Status colors correct

---

## Migration

**For existing API keys:**

```go
func MigrateAPIKeys() {
    // Add default values for new fields
    for _, key := range allKeys {
        if key.Status == "" {
            key.Status = "active"
        }
        if key.UpdatedAt.IsZero() {
            key.UpdatedAt = key.CreatedAt
        }
    }
}
```

---

## Security Considerations

1. **Audit logging** - log all edit/revoke operations
2. **Admin-only** - require admin permission
3. **Confirmation** - require explicit confirmation for revoke
4. **Rollback** - keep history of changes (optional)

---

**Статус:** 📋 Ready  
**Complexity:** MEDIUM

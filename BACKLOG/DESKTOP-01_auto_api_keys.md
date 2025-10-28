# DESKTOP-01: Auto-Generated API Keys System

**Версия:** v2.4.1  
**Приоритет:** HIGH  
**Оценка:** 3-4 часа  
**Статус:** ✅ Завершено  
**Дата начала:** 2025-10-28  
**Дата завершения:** 2025-10-28

---

## 📋 Описание

Реализация системы автоматической генерации API ключей для desktop приложений. Desktop клиент (Wails) при первом запуске выполнит username/password аутентификацию, получит временный JWT, использует его для создания long-lived API key для устройства, и затем удалит JWT. Все последующие API запросы будут использовать этот API key.

## 🎯 Цели

1. **Упростить authentication flow** для desktop клиентов
2. **Обеспечить device-specific access control** - каждое устройство = отдельный API key
3. **Поддержать multi-device использование** - пользователь может иметь 10+ устройств
4. **Обеспечить device tracking** - last seen, OS, hostname, версия приложения
5. **Предотвратить duplicate devices** через fingerprinting

## 🏗️ Архитектура

### Authentication Flow

```
Desktop Client (First Launch)
    ↓
1. POST /api/auth/login 
   { username, password }
    ↓
   ← JWT Token (short-lived, 5 min)
    ↓
2. POST /api/auth/devices/register
   Header: Authorization: Bearer {JWT}
   Body: {
     device_name: "John's Laptop",
     device_os: "windows",
     device_hostname: "JOHN-PC",
     device_version: "1.0.0",
     device_fingerprint: "sha256_hash"
   }
    ↓
   ← { api_key: "sk-...", key_id: "uuid" }
    ↓
3. Save api_key to local config
4. Discard JWT
    ↓
5. All subsequent requests use API Key
   Header: Authorization: Bearer {api_key}
```

### Device Fingerprinting

**Цель:** Prevent duplicate device registrations

**Алгоритм:**

```
fingerprint = SHA256(
    user_id + 
    device_os + 
    device_hostname + 
    mac_address + 
    cpu_model + 
    disk_serial
)
```

**Behavior:**

- Если fingerprint already exists → вернуть существующий API key
- Если fingerprint новый → создать новый API key
- Позволить пользователю иметь несколько устройств с одинаковым hostname (например, два компьютера с именем "Home PC")

## 📊 Database Schema

### Расширение таблицы `api_keys`

```sql
ALTER TABLE api_keys ADD COLUMN device_name TEXT;
ALTER TABLE api_keys ADD COLUMN device_os TEXT;
ALTER TABLE api_keys ADD COLUMN device_hostname TEXT;
ALTER TABLE api_keys ADD COLUMN device_version TEXT;
ALTER TABLE api_keys ADD COLUMN device_fingerprint TEXT;
ALTER TABLE api_keys ADD COLUMN last_seen_at DATETIME;
ALTER TABLE api_keys ADD COLUMN auto_expire_at DATETIME;

CREATE INDEX idx_api_keys_device_fingerprint ON api_keys(device_fingerprint);
CREATE INDEX idx_api_keys_last_seen ON api_keys(last_seen_at);
```

**Поля:**

- `device_name` (TEXT, nullable) - User-friendly имя устройства ("John's MacBook Pro")
- `device_os` (TEXT, nullable) - OS: "windows", "darwin", "linux"
- `device_hostname` (TEXT, nullable) - Hostname системы
- `device_version` (TEXT, nullable) - Версия desktop приложения ("1.0.0")
- `device_fingerprint` (TEXT, nullable) - SHA256 hash для unique identification
- `last_seen_at` (DATETIME, nullable) - Последний раз когда key был использован
- `auto_expire_at` (DATETIME, nullable) - Auto-expiry для device keys (90 days)

**Индексы:**

- `idx_api_keys_device_fingerprint` - Быстрый поиск по fingerprint
- `idx_api_keys_last_seen` - Для cleanup неактивных устройств

## 🔌 API Endpoints

### POST /api/auth/devices/register

**Назначение:** Create API key for desktop device

**Authentication:** JWT Token (short-lived, from login)

**Request:**

```json
{
  "device_name": "John's Laptop",
  "device_os": "windows",
  "device_hostname": "JOHN-PC",
  "device_version": "1.0.0",
  "device_fingerprint": "a1b2c3d4e5f6...",
  "auto_expire_days": 90
}
```

**Response (Success - 201 Created):**

```json
{
  "api_key": "sk-aigateway-1a2b3c4d5e6f7g8h9i0j...",
  "key_id": "550e8400-e29b-41d4-a716-446655440000",
  "device_name": "John's Laptop",
  "device_os": "windows",
  "created_at": "2025-10-28T10:00:00Z",
  "expires_at": "2026-01-26T10:00:00Z",
  "is_new_device": true
}
```

**Response (Existing Device - 200 OK):**

```json
{
  "api_key": "sk-aigateway-1a2b3c4d5e6f7g8h9i0j...",
  "key_id": "550e8400-e29b-41d4-a716-446655440000",
  "device_name": "John's Laptop",
  "device_os": "windows",
  "created_at": "2025-10-20T10:00:00Z",
  "expires_at": "2026-01-18T10:00:00Z",
  "last_seen_at": "2025-10-27T15:30:00Z",
  "is_new_device": false,
  "message": "Device already registered. Returning existing API key."
}
```

**Error Responses:**

- `401 Unauthorized` - Invalid JWT token
- `400 Bad Request` - Missing required fields
- `429 Too Many Requests` - Rate limit exceeded (max 5 registrations per day per user)
- `500 Internal Server Error` - Server error

**Business Logic:**

1. Validate JWT token → extract user_id
2. Validate request fields (device_os, device_fingerprint required)
3. Check device_fingerprint:
   - If exists for this user → return existing API key
   - If new → generate new API key
4. Auto-generate device_name if not provided:
   - Pattern: "Desktop App - {OS} - {Date}"
   - Example: "Desktop App - Windows - Oct 28"
5. Set auto_expire_at = now + 90 days (if auto_expire_days provided)
6. Create API key with device metadata
7. Return API key + key_id

**Rate Limiting:**

- Max 5 device registrations per user per day
- Purpose: Prevent abuse

## 🔧 Go Implementation

### Models Update (internal/models/apikey.go)

```go
// APIKey represents an API key with optional device metadata
type APIKey struct {
    ID          string     `json:"id"`
    UserID      string     `json:"user_id"`
    Name        string     `json:"name"`
    KeyHash     string     `json:"-"` // Never expose hash
    
    // Device metadata (для desktop clients)
    DeviceName        *string    `json:"device_name,omitempty"`
    DeviceOS          *string    `json:"device_os,omitempty"`
    DeviceHostname    *string    `json:"device_hostname,omitempty"`
    DeviceVersion     *string    `json:"device_version,omitempty"`
    DeviceFingerprint *string    `json:"device_fingerprint,omitempty"`
    LastSeenAt        *time.Time `json:"last_seen_at,omitempty"`
    AutoExpireAt      *time.Time `json:"auto_expire_at,omitempty"`
    
    // ... existing fields (Models, Permissions, CreatedAt, etc.)
}

// DeviceRegistrationRequest represents device registration request
type DeviceRegistrationRequest struct {
    DeviceName        string `json:"device_name"`
    DeviceOS          string `json:"device_os" binding:"required,oneof=windows darwin linux"`
    DeviceHostname    string `json:"device_hostname"`
    DeviceVersion     string `json:"device_version"`
    DeviceFingerprint string `json:"device_fingerprint" binding:"required"`
    AutoExpireDays    int    `json:"auto_expire_days,omitempty"` // Default: 90
}

// DeviceRegistrationResponse represents device registration response
type DeviceRegistrationResponse struct {
    APIKey       string     `json:"api_key"`
    KeyID        string     `json:"key_id"`
    DeviceName   string     `json:"device_name"`
    DeviceOS     string     `json:"device_os"`
    CreatedAt    time.Time  `json:"created_at"`
    ExpiresAt    *time.Time `json:"expires_at,omitempty"`
    LastSeenAt   *time.Time `json:"last_seen_at,omitempty"`
    IsNewDevice  bool       `json:"is_new_device"`
    Message      string     `json:"message,omitempty"`
}
```

### Storage Layer (internal/storage/sqlite/apikeys.go)

```go
// FindAPIKeyByDeviceFingerprint finds API key by device fingerprint for user
func (s *SQLiteDB) FindAPIKeyByDeviceFingerprint(ctx context.Context, userID, fingerprint string) (*models.APIKey, error)

// UpdateAPIKeyLastSeen updates last_seen_at timestamp
func (s *SQLiteDB) UpdateAPIKeyLastSeen(ctx context.Context, keyID string) error

// CreateDeviceAPIKey creates API key with device metadata
func (s *SQLiteDB) CreateDeviceAPIKey(ctx context.Context, key *models.APIKey) error
```

### Handler (internal/api/handlers/device_handler.go)

```go
// DeviceHandler handles device-related operations
type DeviceHandler struct {
    db          storage.Database
    logger      *logrus.Logger
    rateLimiter *RateLimiter
}

// RegisterDevice handles POST /api/auth/devices/register
func (h *DeviceHandler) RegisterDevice(c *gin.Context) {
    // 1. Extract user from JWT context
    // 2. Parse DeviceRegistrationRequest
    // 3. Check rate limit (5 per day)
    // 4. Check existing fingerprint
    // 5. Generate or return existing API key
    // 6. Return DeviceRegistrationResponse
}

// generateDeviceFingerprint generates device fingerprint
func generateDeviceFingerprint(userID, os, hostname, macAddr, cpuModel, diskSerial string) string {
    data := fmt.Sprintf("%s:%s:%s:%s:%s:%s", userID, os, hostname, macAddr, cpuModel, diskSerial)
    hash := sha256.Sum256([]byte(data))
    return hex.EncodeToString(hash[:])
}

// generateDeviceName generates auto device name
func generateDeviceName(os string) string {
    osName := map[string]string{
        "windows": "Windows",
        "darwin":  "macOS",
        "linux":   "Linux",
    }[os]
    return fmt.Sprintf("Desktop App - %s - %s", osName, time.Now().Format("Jan 02"))
}
```

### Middleware Update (internal/api/middleware/auth.go)

```go
// APIKeyMiddleware - обновить для tracking last_seen_at
func APIKeyMiddleware(db storage.Database) gin.HandlerFunc {
    return func(c *gin.Context) {
        // ... existing key validation ...
        
        // Update last_seen_at для device keys (async)
        if key.DeviceFingerprint != nil {
            go func() {
                ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
                defer cancel()
                _ = db.UpdateAPIKeyLastSeen(ctx, key.ID)
            }()
        }
        
        // ... continue ...
    }
}
```

## 🧪 Testing

### Unit Tests

**File:** `internal/api/handlers/device_handler_test.go`

```go
func TestRegisterDevice_NewDevice(t *testing.T)
func TestRegisterDevice_ExistingDevice(t *testing.T)
func TestRegisterDevice_InvalidFingerprint(t *testing.T)
func TestRegisterDevice_RateLimitExceeded(t *testing.T)
func TestGenerateDeviceFingerprint(t *testing.T)
func TestUpdateLastSeenAt(t *testing.T)
```

### Integration Tests

```bash
# New device registration
curl -X POST http://localhost:8080/api/auth/devices/register \
  -H "Authorization: Bearer {JWT}" \
  -H "Content-Type: application/json" \
  -d '{
    "device_os": "windows",
    "device_hostname": "JOHN-PC",
    "device_version": "1.0.0",
    "device_fingerprint": "a1b2c3d4e5f6..."
  }'

# Existing device (same fingerprint)
curl -X POST http://localhost:8080/api/auth/devices/register \
  -H "Authorization: Bearer {JWT}" \
  -H "Content-Type: application/json" \
  -d '{
    "device_os": "windows",
    "device_hostname": "JOHN-PC",
    "device_version": "1.0.1",
    "device_fingerprint": "a1b2c3d4e5f6..."
  }'
```

## 📈 Metrics & Monitoring

**Metrics to track:**

- `desktop_devices_registered_total` - Total device registrations
- `desktop_devices_active_count` - Active devices (last_seen < 7 days)
- `desktop_api_key_usage_by_device` - API calls per device
- `desktop_device_registration_errors` - Failed registrations

**Logs:**

```go
logger.WithFields(logrus.Fields{
    "event":              "device_registered",
    "user_id":            userID,
    "device_os":          req.DeviceOS,
    "device_fingerprint": req.DeviceFingerprint[:16], // First 16 chars
    "is_new":             isNew,
}).Info("Device registered")
```

## 🔒 Security Considerations

1. **JWT Short-lived**: JWT token valid only 5 minutes for device registration
2. **API Key Long-lived**: Device API key valid 90 days (renewable)
3. **Rate Limiting**: Max 5 device registrations per user per day
4. **Fingerprint Privacy**: Only store hash, never raw system info
5. **Auto-expiry**: Optional auto-expiry for device keys
6. **Last Seen Tracking**: Helps identify inactive/compromised devices

## 📝 Configuration

### config.yaml

```yaml
desktop:
  enabled: true
  device_registration:
    max_per_day: 5
    default_expiry_days: 90
    require_fingerprint: true
    allow_duplicate_hostname: true
  last_seen_update:
    enabled: true
    async: true
```

## 🚀 Deployment

### Migration Steps

1. Run database migration (v66) to add device columns
2. Deploy updated backend with device registration endpoint
3. Update desktop client to use new registration flow
4. Monitor device registrations in logs
5. Verify last_seen_at updates correctly

### Rollback Plan

- Device columns are nullable → backward compatible
- Old API keys continue working
- Can disable device registration via config

## ✅ Acceptance Criteria

- [ ] Database migration applied successfully
- [ ] POST /api/auth/devices/register endpoint working
- [ ] Device fingerprinting prevents duplicates
- [ ] last_seen_at updates on each API call
- [ ] Rate limiting enforced (5 per day)
- [ ] Auto device naming works correctly
- [ ] Unit tests coverage > 90%
- [ ] Integration tests pass
- [ ] API documentation updated
- [ ] Logs показывают device registrations

## 📚 Documentation

**Files to update:**

- `README.md` - Add device registration section
- `docs/API.md` - Document `/api/auth/devices/register` endpoint
- `docs/DESKTOP_CLIENT.md` - Create guide for desktop integration
- `CHANGELOG.md` - Add v2.4.1 entry

## 🔗 Related Tasks

- **DESKTOP-02** - Device Management API (depends on this)
- **DESKTOP-03** - WebSocket Streaming (uses device API keys)
- **DESKTOP-04** - Desktop WebUI (displays registered devices)

---

**Next Steps:**

1. Create database migration v66
2. Update APIKey model
3. Implement device registration handler
4. Add middleware for last_seen tracking
5. Write tests
6. Update documentation

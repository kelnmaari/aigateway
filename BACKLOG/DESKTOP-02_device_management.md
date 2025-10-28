# DESKTOP-02: Device Management API

**Версия:** v2.4.2  
**Приоритет:** HIGH  
**Оценка:** 2-3 часа  
**Статус:** ✅ Завершено  
**Дата начала:** 2025-10-28  
**Дата завершения:** 2025-10-28  
**Зависит от:** DESKTOP-01 (Auto-Generated API Keys) ✅

---

## 📋 Описание

Реализация REST API для управления registered devices пользователя. Позволяет просматривать список устройств, получать детальную информацию, отзывать (revoke) device API keys, и обновлять названия устройств. Включает WebUI в user profile для удобного управления.

## 🎯 Цели

1. **Просмотр устройств** - список всех зарегистрированных устройств с метаданными
2. **Детальная информация** - last seen, creation date, OS, версия приложения
3. **Revoke устройств** - удаленный logout для потерянных/украденных устройств
4. **Rename устройств** - изменение user-friendly имени устройства
5. **WebUI интеграция** - страница управления устройствами в user profile

## 🏗️ Архитектура

### API Endpoints

```
GET    /api/auth/devices          - List all user's devices
GET    /api/auth/devices/:id      - Get device details
DELETE /api/auth/devices/:id      - Revoke device (delete API key)
PATCH  /api/auth/devices/:id      - Update device name
```

### Authentication

Все endpoints требуют **JWT authentication** (user должен быть залогинен).

### Authorization

- User может управлять только своими устройствами
- Admin может управлять устройствами всех пользователей (будущее расширение)

## 📊 Data Models

### DeviceInfo (Response Model)

```go
type DeviceInfo struct {
    ID                string     `json:"id"`                  // API Key ID
    DeviceName        string     `json:"device_name"`         // User-friendly name
    DeviceOS          string     `json:"device_os"`           // windows, darwin, linux
    DeviceHostname    string     `json:"device_hostname"`     // System hostname
    DeviceVersion     string     `json:"device_version"`      // App version
    DeviceFingerprint string     `json:"device_fingerprint"`  // SHA256 hash (first 16 chars)
    CreatedAt         time.Time  `json:"created_at"`          // Registration date
    LastSeenAt        *time.Time `json:"last_seen_at"`        // Last API request
    ExpiresAt         *time.Time `json:"expires_at"`          // Auto-expiry date
    IsCurrentDevice   bool       `json:"is_current_device"`   // Is this the device making request?
    Status            string     `json:"status"`              // active, expired, revoked
}

type ListDevicesResponse struct {
    Devices      []DeviceInfo `json:"devices"`
    Total        int          `json:"total"`
    CurrentCount int          `json:"current_count"` // Active devices
}

type UpdateDeviceNameRequest struct {
    DeviceName string `json:"device_name" binding:"required,min=1,max=100"`
}
```

## 🔌 API Specification

### GET /api/auth/devices

**Назначение:** Получить список всех зарегистрированных устройств пользователя

**Authentication:** JWT Token (Bearer)

**Request:** None (query params опционально)

**Query Parameters:**
- `status` (optional) - Фильтр по статусу: `active`, `expired`, `all` (default: `active`)
- `sort` (optional) - Сортировка: `last_seen`, `created_at`, `name` (default: `last_seen`)
- `order` (optional) - Порядок: `asc`, `desc` (default: `desc`)

**Response (200 OK):**

```json
{
  "devices": [
    {
      "id": "ak_1730000001_a1b2c3d4",
      "device_name": "John's MacBook Pro",
      "device_os": "darwin",
      "device_hostname": "Johns-MacBook-Pro.local",
      "device_version": "1.0.0",
      "device_fingerprint": "a1b2c3d4e5f6a7b8",
      "created_at": "2025-10-20T10:00:00Z",
      "last_seen_at": "2025-10-28T15:30:00Z",
      "expires_at": "2026-01-18T10:00:00Z",
      "is_current_device": true,
      "status": "active"
    },
    {
      "id": "ak_1729000001_b2c3d4e5",
      "device_name": "Desktop (Windows) - WORK-PC",
      "device_os": "windows",
      "device_hostname": "WORK-PC",
      "device_version": "0.9.5",
      "device_fingerprint": "b2c3d4e5f6a7b8c9",
      "created_at": "2025-10-15T08:00:00Z",
      "last_seen_at": "2025-10-25T18:00:00Z",
      "expires_at": "2026-01-13T08:00:00Z",
      "is_current_device": false,
      "status": "active"
    }
  ],
  "total": 2,
  "current_count": 2
}
```

**Error Responses:**
- `401 Unauthorized` - Invalid or missing JWT token
- `500 Internal Server Error` - Server error

---

### GET /api/auth/devices/:id

**Назначение:** Получить детальную информацию об устройстве

**Authentication:** JWT Token (Bearer)

**Path Parameters:**
- `id` - API Key ID устройства

**Response (200 OK):**

```json
{
  "id": "ak_1730000001_a1b2c3d4",
  "device_name": "John's MacBook Pro",
  "device_os": "darwin",
  "device_hostname": "Johns-MacBook-Pro.local",
  "device_version": "1.0.0",
  "device_fingerprint": "a1b2c3d4e5f6a7b8",
  "created_at": "2025-10-20T10:00:00Z",
  "last_seen_at": "2025-10-28T15:30:00Z",
  "expires_at": "2026-01-18T10:00:00Z",
  "is_current_device": true,
  "status": "active",
  "usage": {
    "total_requests": 1234,
    "successful_requests": 1200,
    "failed_requests": 34,
    "total_tokens": 567890
  }
}
```

**Error Responses:**
- `401 Unauthorized` - Invalid or missing JWT token
- `403 Forbidden` - Device doesn't belong to user
- `404 Not Found` - Device not found
- `500 Internal Server Error` - Server error

---

### DELETE /api/auth/devices/:id

**Назначение:** Revoke device (удалить API key устройства)

**Authentication:** JWT Token (Bearer)

**Path Parameters:**
- `id` - API Key ID устройства

**Request Body (optional):**

```json
{
  "reason": "Lost device"
}
```

**Response (200 OK):**

```json
{
  "message": "Device revoked successfully",
  "device_id": "ak_1730000001_a1b2c3d4",
  "device_name": "John's MacBook Pro"
}
```

**Error Responses:**
- `401 Unauthorized` - Invalid or missing JWT token
- `403 Forbidden` - Device doesn't belong to user
- `404 Not Found` - Device not found
- `400 Bad Request` - Trying to revoke current device (use logout instead)
- `500 Internal Server Error` - Server error

**Business Logic:**
1. Validate JWT token → extract user_id
2. Verify device belongs to user
3. Check if device is current device (prevent self-revoke)
4. Revoke API key with reason
5. Log audit event

---

### PATCH /api/auth/devices/:id

**Назначение:** Обновить имя устройства

**Authentication:** JWT Token (Bearer)

**Path Parameters:**
- `id` - API Key ID устройства

**Request Body:**

```json
{
  "device_name": "John's New Laptop"
}
```

**Response (200 OK):**

```json
{
  "message": "Device name updated successfully",
  "device": {
    "id": "ak_1730000001_a1b2c3d4",
    "device_name": "John's New Laptop",
    "device_os": "darwin",
    "updated_at": "2025-10-28T16:00:00Z"
  }
}
```

**Error Responses:**
- `401 Unauthorized` - Invalid or missing JWT token
- `403 Forbidden` - Device doesn't belong to user
- `404 Not Found` - Device not found
- `400 Bad Request` - Invalid device name (too long, empty, etc.)
- `500 Internal Server Error` - Server error

## 🧪 Testing

### Unit Tests

**File:** `internal/api/handlers/device_handler_test.go` (extend existing)

```go
func TestDeviceHandler_ListDevices(t *testing.T)
func TestDeviceHandler_ListDevices_WithFilters(t *testing.T)
func TestDeviceHandler_GetDevice(t *testing.T)
func TestDeviceHandler_GetDevice_NotOwned(t *testing.T)
func TestDeviceHandler_DeleteDevice(t *testing.T)
func TestDeviceHandler_DeleteDevice_CurrentDevice(t *testing.T)
func TestDeviceHandler_UpdateDeviceName(t *testing.T)
func TestDeviceHandler_UpdateDeviceName_InvalidName(t *testing.T)
```

### Integration Tests

```bash
# List devices
curl -X GET http://localhost:8080/api/auth/devices \
  -H "Authorization: Bearer {JWT}"

# Get device details
curl -X GET http://localhost:8080/api/auth/devices/{id} \
  -H "Authorization: Bearer {JWT}"

# Delete device
curl -X DELETE http://localhost:8080/api/auth/devices/{id} \
  -H "Authorization: Bearer {JWT}" \
  -H "Content-Type: application/json" \
  -d '{"reason": "Lost device"}'

# Update device name
curl -X PATCH http://localhost:8080/api/auth/devices/{id} \
  -H "Authorization: Bearer {JWT}" \
  -H "Content-Type: application/json" \
  -d '{"device_name": "My New Laptop"}'
```

## 🎨 WebUI Implementation

### Page: `/profile/devices` (или `/settings/devices`)

**Location:** `web/profile-devices.html`

**Layout:**

```
┌──────────────────────────────────────────────┐
│ My Devices                          [+] Add  │
├──────────────────────────────────────────────┤
│                                              │
│  🖥️ John's MacBook Pro (This device)        │
│     macOS • Version 1.0.0                   │
│     Last seen: Just now                     │
│     [✏️ Rename]                              │
│                                              │
│  💻 Desktop (Windows) - WORK-PC             │
│     Windows • Version 0.9.5                 │
│     Last seen: 3 days ago                   │
│     [✏️ Rename] [🗑️ Remove]                  │
│                                              │
│  📱 Desktop (Linux) - ubuntu-server         │
│     Linux • Version 0.9.0                   │
│     Last seen: 30 days ago (inactive)       │
│     [✏️ Rename] [🗑️ Remove]                  │
│                                              │
└──────────────────────────────────────────────┘
```

**Features:**
- List all devices with visual indicators (current device, inactive devices)
- Sort by last seen, creation date, name
- Filter by status (active, expired, all)
- Rename device (modal dialog)
- Remove device (confirmation dialog with warning)
- Refresh button
- Device icons based on OS

**JavaScript:** `web/js/profile-devices.js`

**Dependencies:**
- Bootstrap 5 (modals, cards, buttons)
- Notifications.js (success/error messages)
- Fetch API для HTTP requests

## 📈 Metrics & Monitoring

**Metrics to track:**
- `device_list_requests_total` - Total list requests
- `device_revoke_total` - Total device revocations
- `device_rename_total` - Total device renames
- `active_devices_per_user` - Histogram of devices per user

**Logs:**

```go
logger.WithFields(logrus.Fields{
    "event":     "device_revoked",
    "user_id":   userID,
    "device_id": deviceID,
    "reason":    reason,
}).Info("Device revoked by user")
```

## 🔒 Security Considerations

1. **Self-Revoke Prevention**: Нельзя удалить текущее устройство (используй logout вместо этого)
2. **Owner Verification**: User может управлять только своими устройствами
3. **Audit Logging**: Все операции (revoke, rename) логируются в audit trail
4. **Rate Limiting**: Max 10 device operations per minute per user
5. **Soft Delete**: Revoked devices сохраняются в БД для audit purposes

## 📝 Configuration

### config.yaml

```yaml
desktop:
  enabled: true
  device_management:
    max_devices_per_user: 10
    allow_self_revoke: false
    inactive_threshold_days: 30
    auto_cleanup_expired: true
```

## 🚀 Deployment

### Migration Steps

1. No new migrations needed (используем существующие device поля из v66)
2. Deploy updated backend with new device management endpoints
3. Deploy WebUI страницу `/profile/devices`
4. Test device listing, revoke, rename operations
5. Monitor audit logs

### Rollback Plan

- New endpoints are additive → backward compatible
- Can disable device management via config
- Desktop clients continue working with existing API keys

## ✅ Acceptance Criteria

- [ ] GET /api/auth/devices endpoint working
- [ ] GET /api/auth/devices/:id endpoint working
- [ ] DELETE /api/auth/devices/:id endpoint working
- [ ] PATCH /api/auth/devices/:id endpoint working
- [ ] Self-revoke prevention implemented
- [ ] Owner verification working
- [ ] WebUI page for device management created
- [ ] Unit tests coverage > 90%
- [ ] Integration tests pass
- [ ] API documentation updated
- [ ] Audit logging for all operations

## 📚 Documentation

**Files to update:**

- `README.md` - Add device management section
- `docs/API.md` - Document new device management endpoints
- `docs/DESKTOP_CLIENT.md` - Update with device management workflow
- `CHANGELOG.md` - Add v2.4.2 entry

## 🔗 Related Tasks

- **DESKTOP-01** - Auto-Generated API Keys System (completed) ✅
- **DESKTOP-03** - WebSocket Streaming (depends on this)
- **DESKTOP-04** - Desktop WebUI (uses device management)

---

**Next Steps:**

1. Extend DeviceHandler with new methods (ListDevices, GetDevice, DeleteDevice, UpdateDeviceName)
2. Add storage layer methods (ListDeviceKeys, GetDeviceKey)
3. Register new routes in router
4. Create WebUI page
5. Write tests
6. Update documentation


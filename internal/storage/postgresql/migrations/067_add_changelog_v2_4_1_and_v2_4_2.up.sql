
INSERT INTO changelogs (version, release_date, content) VALUES
('2.4.2', '2025-10-28', '## [2.4.2] - 2025-10-28

### Added

- **Device Management API** (DESKTOP-02): Полное управление устройствами desktop client
  - **REST API Endpoints**:
    - GET /api/auth/devices - список устройств пользователя с фильтрацией (status, sort, order)
    - DELETE /api/auth/devices/:id - revoke device API key с защитой от self-revoke
    - PATCH /api/auth/devices/:id - обновление user-friendly имени устройства
  - **Device Management WebUI** (/profile-devices.html):
    - Список всех зарегистрированных устройств с карточками
    - Визуальные индикаторы: OS icons, статус (Active/Inactive/Expired)
    - Highlight текущего устройства с badge "This Device"
    - Фильтры: status, сортировка, order
    - Действия: Rename device, Remove device
  - **Security**: Owner verification, Self-revoke prevention, JWT auth

### Technical

- **Backend**: Device management handlers, storage queries, route registration
- **Frontend**: profile-devices.html, DeviceManager class, navbar integration
- **Database**: Использует device fields из migration v66'),

('2.4.1', '2025-10-28', '## [2.4.1] - 2025-10-28

### Added

- **Auto-Generated API Keys System** (DESKTOP-01): Автоматическая регистрация desktop устройств
  - **Device Registration API**:
    - POST /api/auth/devices/register - auto-creation API keys
    - Device metadata: OS, hostname, app version, -- fingerprint (SHA256)
    - Auto-expiry для device keys (default 90 days)
    - Duplicate detection по device fingerprint
  - **Authentication Flow**:
    1. Desktop app login → JWT token
    2. JWT → POST /api/auth/devices/register → API key
    3. API key сохраняется локально
    4. JWT discarded (security best practice)

### Technical

- **Database Migration v66**: device_name, device_os, device_hostname, device_version, device_fingerprint, last_seen_at, auto_expire_at
- **Backend**: DeviceHandler, device queries, registration endpoint
- **Testing**: Unit tests для device registration')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	
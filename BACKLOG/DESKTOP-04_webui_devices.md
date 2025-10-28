# DESKTOP-04: Desktop-Specific WebUI Features

**Задача:** DESKTOP-04  
**Версия:** v2.4.4  
**Статус:** ✅ Завершено  
**Приоритет:** MEDIUM  
**Сложность:** Low-Medium  
**Оценка:** 2-3 часа  
**Фактически:** ~2 часа  
**Дата завершения:** 2025-10-28

---

## 📋 Описание

Расширение существующей страницы `/profile-devices.html` (из DESKTOP-02) дополнительными фичами для лучшего device management experience.

### Текущая ситуация (DESKTOP-02)

✅ **Уже реализовано:**

- Device cards с OS icons (🪟 Windows, 🍎 macOS, 🐧 Linux)
- Status badges (Active, Inactive, Expired)
- Last seen timestamp с цветовой индикацией
- Фильтрация по status (active/expired/all)
- Сортировка (last_seen/created_at/name)
- Rename device (prompt)
- Remove device (confirmation modal)
- Current device highlight ("This Device" badge)
- Adaptive grid layout (CSS Grid)

❌ **НЕ реализовано (DESKTOP-04):**

- Device details modal с полной информацией
- Real-time online/offline indicators через WebSocket
- Bulk revoke для multiple devices
- Activity timeline per device
- Security alerts для новых устройств

---

## 🎯 Цели DESKTOP-04

### 1. Device Details Modal 🔍

**Клик на device card открывает modal с детальной информацией:**

```html
<div class="device-modal">
  <h3>🖥️ Device Details</h3>
  
  <div class="details-grid">
    <!-- Basic Info -->
    <div class="detail-item">
      <span class="label">Device Name:</span>
      <span class="value">Desktop App - Windows - 2025-10-28</span>
    </div>
    
    <div class="detail-item">
      <span class="label">Operating System:</span>
      <span class="value">Windows 11 Pro</span>
    </div>
    
    <div class="detail-item">
      <span class="label">Version:</span>
      <span class="value">v1.0.0</span>
    </div>
    
    <div class="detail-item">
      <span class="label">Hostname:</span>
      <span class="value">DESKTOP-ABC123</span>
    </div>
    
    <!-- Activity Info -->
    <div class="detail-item">
      <span class="label">First Seen:</span>
      <span class="value">2025-10-15 14:30:22</span>
    </div>
    
    <div class="detail-item">
      <span class="label">Last Seen:</span>
      <span class="value">2025-10-28 10:45:12 (2 minutes ago)</span>
    </div>
    
    <div class="detail-item">
      <span class="label">Total Requests:</span>
      <span class="value">1,234 requests</span>
    </div>
    
    <div class="detail-item">
      <span class="label">Status:</span>
      <span class="value"><span class="badge badge-success">🟢 Active</span></span>
    </div>
  </div>
  
  <!-- Activity Timeline (если есть данные) -->
  <div class="activity-timeline">
    <h4>Recent Activity</h4>
    <ul>
      <li>
        <span class="timestamp">2025-10-28 10:45</span>
        <span class="event">Chat request (qwen3-coder)</span>
      </li>
      <li>
        <span class="timestamp">2025-10-28 10:30</span>
        <span class="event">Connected via WebSocket</span>
      </li>
    </ul>
  </div>
  
  <!-- Actions -->
  <div class="modal-actions">
    <button class="btn btn-primary" onclick="editDeviceName()">✏️ Rename</button>
    <button class="btn btn-danger" onclick="revokeDevice()">🗑️ Revoke Access</button>
    <button class="btn btn-secondary" onclick="closeModal()">Close</button>
  </div>
</div>
```

**JavaScript:**

```javascript
// В DeviceManager class
openDeviceDetails(deviceId) {
  const device = this.devices.find(d => d.id === deviceId);
  if (!device) return;
  
  modal.custom({
    title: '🖥️ Device Details',
    html: this.renderDeviceDetailsHTML(device),
    size: 'large',
    buttons: [
      { text: '✏️ Rename', class: 'btn-primary', onclick: () => this.openRenameModal(device.id) },
      { text: '🗑️ Revoke', class: 'btn-danger', onclick: () => this.openDeleteModal(device.id) },
      { text: 'Close', class: 'btn-secondary', onclick: () => modal.close() }
    ]
  });
}
```

### 2. Real-time Online/Offline Indicators 🌐

**WebSocket integration для real-time status:**

```javascript
// В DeviceManager class
connectWebSocket() {
  if (!wsManager) return;
  
  // Subscribe to device status updates
  wsManager.on('device_status', (event) => {
    const { device_id, status, last_seen_at } = event;
    this.updateDeviceStatus(device_id, status, last_seen_at);
  });
  
  // Subscribe to device connected/disconnected
  wsManager.on('device_connected', (event) => {
    const { device_id, user_id } = event;
    if (user_id === this.currentUser.id) {
      this.markDeviceOnline(device_id);
      toast.success(`Device connected: ${this.getDeviceName(device_id)}`);
    }
  });
  
  wsManager.on('device_disconnected', (event) => {
    const { device_id, user_id } = event;
    if (user_id === this.currentUser.id) {
      this.markDeviceOffline(device_id);
    }
  });
}

markDeviceOnline(deviceId) {
  const card = document.querySelector(`[data-device-id="${deviceId}"]`);
  if (card) {
    card.querySelector('.online-indicator').classList.add('online');
    card.querySelector('.online-indicator').textContent = '🟢 Online';
  }
}
```

**CSS для online indicator:**

```css
.online-indicator {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 0.875rem;
  color: var(--text-secondary);
}
.online-indicator.online {
  color: var(--success-color);
  font-weight: 600;
}
.online-indicator:not(.online) {
  color: var(--text-muted);
}
```

### 3. Bulk Revoke для Multiple Devices 📦

**Checkbox selection + bulk actions bar:**

```html
<!-- Checkbox в device card -->
<div class="device-card">
  <input type="checkbox" class="device-checkbox" data-device-id="{device.id}">
  <!-- rest of card -->
</div>

<!-- Bulk Actions Bar (появляется при selection) -->
<div id="bulkActionsBar" style="display: none;">
  <div class="bulk-actions-content">
    <span class="selected-count">3 devices selected</span>
    <button class="btn btn-danger" onclick="bulkRevokeDevices()">
      🗑️ Revoke Selected
    </button>
    <button class="btn btn-secondary" onclick="clearSelection()">
      Cancel
    </button>
  </div>
</div>
```

**JavaScript:**

```javascript
// В DeviceManager class
enableBulkSelection() {
  document.querySelectorAll('.device-checkbox').forEach(checkbox => {
    checkbox.addEventListener('change', () => {
      this.updateBulkActionsBar();
    });
  });
}

getSelectedDeviceIds() {
  const checkboxes = document.querySelectorAll('.device-checkbox:checked');
  return Array.from(checkboxes).map(cb => cb.dataset.deviceId);
}

async bulkRevokeDevices() {
  const selectedIds = this.getSelectedDeviceIds();
  if (selectedIds.length === 0) return;
  
  const confirmed = await modal.danger({
    title: '⚠️ Bulk Revoke Devices',
    message: `Are you sure you want to revoke ${selectedIds.length} device(s)? This action cannot be undone.`,
    confirmText: 'Revoke All',
    cancelText: 'Cancel'
  });
  
  if (confirmed) {
    let successCount = 0;
    let errorCount = 0;
    
    for (const deviceId of selectedIds) {
      try {
        await api.request(`/api/auth/devices/${deviceId}`, { method: 'DELETE' });
        successCount++;
      } catch (error) {
        console.error(`Failed to revoke device ${deviceId}:`, error);
        errorCount++;
      }
    }
    
    toast.success(`Revoked ${successCount} device(s)`);
    if (errorCount > 0) {
      toast.error(`Failed to revoke ${errorCount} device(s)`);
    }
    
    this.loadDevices(); // Refresh
    this.clearSelection();
  }
}
```

### 4. Activity Timeline per Device 📊

**Backend endpoint (опционально):**

```go
// GET /api/auth/devices/:id/activity
// Returns last N activities for device

type DeviceActivity struct {
    Timestamp   time.Time `json:"timestamp"`
    EventType   string    `json:"event_type"` // "request", "websocket_connect", "auth_failed"
    Description string    `json:"description"`
    Model       string    `json:"model,omitempty"`
}
```

**Frontend rendering:**

```javascript
async loadDeviceActivity(deviceId) {
  try {
    const activities = await api.request(`/api/auth/devices/${deviceId}/activity`);
    return this.renderActivityTimeline(activities);
  } catch (error) {
    return '<p>Activity history not available</p>';
  }
}

renderActivityTimeline(activities) {
  if (activities.length === 0) {
    return '<p class="text-muted">No recent activity</p>';
  }
  
  return `
    <div class="activity-timeline">
      <h4>Recent Activity (Last 24 hours)</h4>
      <ul class="timeline-list">
        ${activities.map(a => `
          <li class="timeline-item">
            <span class="timeline-timestamp">${this.formatDate(a.timestamp)}</span>
            <span class="timeline-event">${this.escapeHtml(a.description)}</span>
            ${a.model ? `<span class="timeline-model">Model: ${a.model}</span>` : ''}
          </li>
        `).join('')}
      </ul>
    </div>
  `;
}
```

**CSS для timeline:**

```css
.activity-timeline {
  margin-top: 1.5rem;
  padding-top: 1.5rem;
  border-top: 1px solid var(--border-color);
}
.timeline-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.timeline-item {
  display: grid;
  grid-template-columns: 150px 1fr auto;
  gap: 1rem;
  padding: 0.75rem;
  border-left: 2px solid var(--primary-color);
  margin-left: 0.5rem;
  margin-bottom: 0.5rem;
}
.timeline-timestamp {
  color: var(--text-secondary);
  font-size: 0.875rem;
}
.timeline-event {
  color: var(--text-primary);
}
.timeline-model {
  color: var(--text-muted);
  font-size: 0.875rem;
  font-style: italic;
}
```

### 5. Security Alerts 🔔

**Показ toast notification при регистрации нового устройства:**

```javascript
// В DeviceManager class
initSecurityAlerts() {
  if (!wsManager) return;
  
  // Subscribe to new device registrations
  wsManager.on('device_registered', (event) => {
    const { device_id, device_name, device_os, user_id } = event;
    
    if (user_id === this.currentUser.id) {
      // Новое устройство для текущего пользователя
      toast.warning(
        `🔔 New device registered: ${device_name} (${device_os})`,
        { duration: 10000 } // 10 seconds
      );
      
      // Опционально: показать modal с подробностями
      modal.warning({
        title: '🔔 New Device Registered',
        message: `A new device has been registered to your account:
        
Device: ${device_name}
OS: ${device_os}
Time: ${new Date().toLocaleString()}

If this wasn't you, please revoke this device immediately.`,
        confirmText: 'View Devices',
        cancelText: 'Dismiss',
        onConfirm: () => {
          window.location.href = '/profile-devices.html';
        }
      });
    }
  });
}
```

**Backend WebSocket event (в device_handler.go):**

```go
// После успешной регистрации устройства
func (h *DeviceHandler) RegisterDevice(c *gin.Context) {
    // ... existing code ...
    
    // После создания API key
    apiKey, err := h.db.CreateAPIKey(ctx, apiKeyData)
    
    // Отправляем WebSocket event
    if h.eventBroadcaster != nil {
        event := map[string]interface{}{
            "type":        "device_registered",
            "device_id":   apiKey.ID,
            "device_name": *apiKey.DeviceName,
            "device_os":   *apiKey.DeviceOS,
            "user_id":     apiKey.UserID,
            "timestamp":   time.Now(),
        }
        
        eventJSON, _ := json.Marshal(event)
        h.hub.SendToUser(*apiKey.UserID, eventJSON)
    }
    
    // ... existing response ...
}
```

---

## 🔨 Реализация

### Приоритеты (в порядке важности)

1. **✅ CRITICAL: Device Details Modal** - основная фича (30 min)
   - ID: desktop-04-modal
   - Modal component с full device info
   - Integration с существующими device cards

2. **✅ HIGH: Bulk Revoke** - полезно для управления (20 min)
   - ID: desktop-04-bulk
   - Checkbox selection
   - Bulk actions bar
   - Mass delete с confirmation

3. **🔶 MEDIUM: Real-time Online Indicators** (30 min)
   - ID: desktop-04-realtime
   - WebSocket integration
   - Online/offline status updates
   - Требует backend events (device_connected/device_disconnected)

4. **🔶 MEDIUM: Activity Timeline** (optional, 30 min)
   - ID: desktop-04-timeline
   - Backend endpoint: `GET /api/auth/devices/:id/activity`
   - Frontend rendering
   - Может быть отложено для future версии

5. **🔶 LOW: Security Alerts** (optional, 15 min)
   - ID: desktop-04-alerts
   - WebSocket event: device_registered
   - Toast notifications
   - Может быть отложено для future версии

---

## 📝 Acceptance Criteria

### Must Have (для v2.4.4)

- [x] Device details modal открывается при клике на card
- [x] Modal показывает полную информацию о device
- [x] Bulk selection checkboxes на device cards
- [x] Bulk actions bar появляется при selection
- [x] Bulk revoke работает корректно (с confirmation)
- [x] Защита от self-revoke при bulk operations

### Nice to Have (опционально)

- [ ] Real-time online/offline indicators через WebSocket
- [ ] Activity timeline в device details modal
- [ ] Security alerts при новых устройствах
- [ ] Export devices list (CSV/JSON)

---

## 🧪 Testing

### Manual Testing

1. **Device Details Modal:**
   - Открыть `/profile-devices.html`
   - Кликнуть на device card
   - Проверить что modal открывается
   - Проверить все поля (name, OS, version, dates)
   - Проверить кнопки (Rename, Revoke, Close)

2. **Bulk Revoke:**
   - Выбрать 2-3 devices через checkboxes
   - Проверить что bulk actions bar появляется
   - Кликнуть "Revoke Selected"
   - Подтвердить в confirmation modal
   - Проверить что devices удалены
   - Проверить что current device НЕ может быть bulk revoked

3. **Edge Cases:**
   - Попытка bulk revoke включая current device
   - Bulk revoke только current device (должно быть blocked)
   - Modal с device без optional полей
   - Empty devices list

---

## 🔗 Связанные задачи

- **DESKTOP-01** (v2.4.1): Auto-Generated API Keys → базовая device data
- **DESKTOP-02** (v2.4.2): Device Management API → существующая WebUI страница
- **DESKTOP-03** (v2.4.3): WebSocket Streaming → используется для real-time updates

---

## 🎯 Следующие шаги (Future)

- [ ] **Device Usage Analytics** - графики токенов per device
- [ ] **Device Groups** - группировка устройств (work, home, mobile)
- [ ] **Geo-location** - показ местоположения устройств
- [ ] **Device Limits** - ограничение количества устройств per user
- [ ] **Two-Factor для Device Registration** - дополнительная безопасность

---

**Дата создания:** 2025-10-28  
**Последнее обновление:** 2025-10-28

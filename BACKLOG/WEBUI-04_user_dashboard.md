# WEBUI-04: User Dashboard & Personal Cabinet

**Версия:** 1.0  
**Статус:** 📋 Запланирована  
**Приоритет:** HIGH  
**Оценка времени:** 10-12 часов  
**Зависимости:** AUTH-05, DB-01

---

## 📋 Описание

Личный кабинет пользователя для управления профилем, API ключами, просмотра статистики использования и настроек. Интеграция с multi-tenancy системой.

---

## 🎯 Цели

1. **Profile Management**: Редактирование профиля, смена пароля
2. **API Keys Management**: Personal API keys CRUD
3. **Usage Statistics**: Мониторинг использования и затрат
4. **Tenant Switching**: Переключение между личным и организационными workspace
5. **Preferences**: Настройки темы, языка, уведомлений

---

## 🎨 Dashboard Layout

```
┌────────────────────────────────────────────────────────────┐
│  Header: [Logo] Dashboard [Tenant Selector ▼] [User Menu] │
├────────────┬───────────────────────────────────────────────┤
│            │                                               │
│  Sidebar   │         Main Content Area                    │
│            │                                               │
│  📊 Overview│  ┌─────────────────────────────────────┐   │
│  🔑 API Keys│  │  Usage This Month                   │   │
│  ⚙️  Settings│  │  📈 2,450 requests  ↑12%           │   │
│  👥 Teams   │  │  🎯 45,230 tokens   ↑8%            │   │
│  💬 Chats   │  │  ⏱️  Avg latency: 1.2s  ↓5%        │   │
│  📈 Usage   │  └─────────────────────────────────────┘   │
│  🎨 Themes  │                                               │
│  🚪 Logout  │  ┌─────────────────────────────────────┐   │
│            │  │  Recent API Keys                    │   │
│            │  │  • Production API    Active         │   │
│            │  │  • Development Key   Active         │   │
│            │  │  • Testing Key       Expired        │   │
│            │  └─────────────────────────────────────┘   │
└────────────┴───────────────────────────────────────────────┘
```

---

## 📊 Dashboard Sections

### 1. Overview Page

```javascript
// static/js/dashboard/overview.js
class DashboardOverview {
    async render() {
        const stats = await this.fetchStats();
        
        return `
            <div class="dashboard-overview">
                <!-- Usage Stats Cards -->
                <div class="stats-grid">
                    <div class="stat-card">
                        <div class="stat-icon">📊</div>
                        <div class="stat-value">${stats.requests_this_month}</div>
                        <div class="stat-label">Requests This Month</div>
                        <div class="stat-change ${stats.requests_change >= 0 ? 'positive' : 'negative'}">
                            ${stats.requests_change >= 0 ? '↑' : '↓'} ${Math.abs(stats.requests_change)}%
                        </div>
                    </div>
                    
                    <div class="stat-card">
                        <div class="stat-icon">🎯</div>
                        <div class="stat-value">${this.formatNumber(stats.tokens_used)}</div>
                        <div class="stat-label">Tokens Used</div>
                        <div class="stat-change">${stats.tokens_change >= 0 ? '↑' : '↓'} ${Math.abs(stats.tokens_change)}%</div>
                    </div>
                    
                    <div class="stat-card">
                        <div class="stat-icon">⏱️</div>
                        <div class="stat-value">${stats.avg_latency}s</div>
                        <div class="stat-label">Avg Response Time</div>
                        <div class="stat-change">${stats.latency_change >= 0 ? '↑' : '↓'} ${Math.abs(stats.latency_change)}%</div>
                    </div>
                    
                    <div class="stat-card">
                        <div class="stat-icon">🔑</div>
                        <div class="stat-value">${stats.active_keys}</div>
                        <div class="stat-label">Active API Keys</div>
                    </div>
                </div>
                
                <!-- Usage Chart -->
                <div class="chart-container">
                    <h3>Request Volume (Last 30 Days)</h3>
                    <canvas id="usage-chart"></canvas>
                </div>
                
                <!-- Models Usage -->
                <div class="models-usage">
                    <h3>Most Used Models</h3>
                    ${this.renderModelsTable(stats.models_usage)}
                </div>
                
                <!-- Recent Activity -->
                <div class="recent-activity">
                    <h3>Recent Activity</h3>
                    ${this.renderActivityFeed(stats.recent_activity)}
                </div>
            </div>
        `;
    }
}
```

### 2. Profile Settings

```html
<div class="profile-settings">
    <h2>Profile Settings</h2>
    
    <!-- Avatar -->
    <div class="avatar-section">
        <img src="${user.avatar || '/static/img/default-avatar.png'}" class="avatar-large">
        <button class="upload-avatar-btn">Change Avatar</button>
    </div>
    
    <!-- Basic Info -->
    <form id="profile-form" class="settings-form">
        <div class="form-group">
            <label>Username</label>
            <input type="text" name="username" value="${user.username}" readonly>
            <small>Username cannot be changed</small>
        </div>
        
        <div class="form-group">
            <label>Email</label>
            <input type="email" name="email" value="${user.email}" required>
        </div>
        
        <div class="form-group">
            <label>Display Name</label>
            <input type="text" name="display_name" value="${user.display_name || ''}" placeholder="John Doe">
        </div>
        
        <button type="submit" class="btn-primary">Save Changes</button>
    </form>
    
    <!-- Change Password -->
    <div class="password-section">
        <h3>Change Password</h3>
        <form id="password-form" class="settings-form">
            <div class="form-group">
                <label>Current Password</label>
                <input type="password" name="current_password" required>
            </div>
            
            <div class="form-group">
                <label>New Password</label>
                <input type="password" name="new_password" required>
                <small>Min 8 characters, must include uppercase, lowercase, number</small>
            </div>
            
            <div class="form-group">
                <label>Confirm New Password</label>
                <input type="password" name="confirm_password" required>
            </div>
            
            <button type="submit" class="btn-primary">Change Password</button>
        </form>
    </div>
    
    <!-- Danger Zone -->
    <div class="danger-zone">
        <h3>Danger Zone</h3>
        <p>Once you delete your account, there is no going back. Please be certain.</p>
        <button class="btn-danger" onclick="confirmDeleteAccount()">Delete Account</button>
    </div>
</div>
```

### 3. API Keys Management

```javascript
// static/js/dashboard/api-keys.js
class APIKeysManager {
    async render() {
        const keys = await this.fetchPersonalAPIKeys();
        
        return `
            <div class="api-keys-section">
                <div class="section-header">
                    <h2>My API Keys</h2>
                    <button class="btn-primary" onclick="openCreateKeyModal()">
                        <i class="bi bi-plus"></i> Create New Key
                    </button>
                </div>
                
                <!-- API Keys Table -->
                <table class="keys-table">
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>Models</th>
                            <th>Rate Limit</th>
                            <th>Usage</th>
                            <th>Status</th>
                            <th>Created</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${keys.map(key => this.renderKeyRow(key)).join('')}
                    </tbody>
                </table>
            </div>
        `;
    }
    
    renderKeyRow(key) {
        return `
            <tr data-key-id="${key.id}">
                <td>
                    <strong>${key.name}</strong>
                    ${key.description ? `<br><small>${key.description}</small>` : ''}
                </td>
                <td>
                    ${key.models.length > 3 
                        ? key.models.slice(0,3).join(', ') + '...' 
                        : key.models.join(', ')}
                </td>
                <td>
                    ${key.rate_limits.requests_per_minute}/min<br>
                    ${key.rate_limits.requests_per_hour}/hour
                </td>
                <td>
                    ${key.usage_this_month || 0} requests<br>
                    <small>${this.formatDate(key.last_used_at || 'Never')}</small>
                </td>
                <td>
                    <span class="status-badge ${key.status}">
                        ${key.status}
                    </span>
                </td>
                <td>${this.formatDate(key.created_at)}</td>
                <td class="actions">
                    <button onclick="viewKey('${key.id}')" title="View Details">
                        <i class="bi bi-eye"></i>
                    </button>
                    <button onclick="editKey('${key.id}')" title="Edit">
                        <i class="bi bi-pencil"></i>
                    </button>
                    <button onclick="revokeKey('${key.id}')" title="Revoke" class="danger">
                        <i class="bi bi-x-circle"></i>
                    </button>
                </td>
            </tr>
        `;
    }
    
    async createKey(data) {
        const response = await fetch('/api/users/me/api-keys', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.getAuthToken()}`
            },
            body: JSON.stringify(data)
        });
        
        const result = await response.json();
        
        // Показываем plain key ОДИН РАЗ!
        this.showPlainKeyModal(result.plain_key, result.api_key);
        
        return result;
    }
    
    showPlainKeyModal(plainKey, keyInfo) {
        // Modal с предупреждением что ключ показывается ТОЛЬКО ОДИН РАЗ
        const modal = `
            <div class="modal-overlay" id="plain-key-modal">
                <div class="modal-content">
                    <div class="alert alert-warning">
                        <strong>⚠️ Important!</strong> 
                        This API key will only be shown once. Please copy it now and store it securely.
                    </div>
                    
                    <h3>Your New API Key</h3>
                    <p><strong>Name:</strong> ${keyInfo.name}</p>
                    
                    <div class="key-display">
                        <code id="plain-key-value">${plainKey}</code>
                        <button onclick="copyKey()" class="btn-copy">
                            <i class="bi bi-clipboard"></i> Copy
                        </button>
                    </div>
                    
                    <button class="btn-primary" onclick="closePlainKeyModal()">
                        I've copied it, close this
                    </button>
                </div>
            </div>
        `;
        
        document.body.insertAdjacentHTML('beforeend', modal);
    }
}
```

### 4. Tenant Selector

```javascript
// static/js/dashboard/tenant-selector.js
class TenantSelector {
    async render() {
        const tenants = await this.fetchUserTenants();
        const current = this.getCurrentTenant();
        
        return `
            <div class="tenant-selector">
                <button class="current-tenant-btn" onclick="toggleTenantDropdown()">
                    <div class="tenant-icon">${this.getTenantIcon(current.type)}</div>
                    <div class="tenant-info">
                        <div class="tenant-name">${current.name}</div>
                        <div class="tenant-role">${current.role}</div>
                    </div>
                    <i class="bi bi-chevron-down"></i>
                </button>
                
                <div class="tenant-dropdown" id="tenant-dropdown">
                    <!-- Personal Tenant -->
                    <div class="dropdown-section">
                        <div class="section-label">Personal</div>
                        ${this.renderTenantItem(tenants.find(t => t.type === 'personal'))}
                    </div>
                    
                    <!-- Organizations -->
                    ${tenants.filter(t => t.type === 'organization').length > 0 ? `
                        <div class="dropdown-section">
                            <div class="section-label">Organizations</div>
                            ${tenants.filter(t => t.type === 'organization')
                                .map(t => this.renderTenantItem(t)).join('')}
                        </div>
                    ` : ''}
                    
                    <div class="dropdown-footer">
                        <button onclick="createNewTenant()">
                            <i class="bi bi-plus"></i> Create Organization
                        </button>
                    </div>
                </div>
            </div>
        `;
    }
    
    async switchTenant(tenantId) {
        // Update session/token
        await fetch('/api/auth/switch-tenant', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.getAuthToken()}`
            },
            body: JSON.stringify({ tenant_id: tenantId })
        });
        
        // Refresh page to reload dashboard with new tenant context
        window.location.reload();
    }
}
```

### 5. Usage Statistics

```javascript
// static/js/dashboard/usage-stats.js
class UsageStats {
    async render() {
        const stats = await this.fetchUsageStats();
        
        return `
            <div class="usage-stats">
                <h2>Usage Statistics</h2>
                
                <!-- Time Period Selector -->
                <div class="period-selector">
                    <button class="period-btn active" data-period="7d">Last 7 Days</button>
                    <button class="period-btn" data-period="30d">Last 30 Days</button>
                    <button class="period-btn" data-period="90d">Last 90 Days</button>
                    <button class="period-btn" data-period="1y">Last Year</button>
                </div>
                
                <!-- Charts -->
                <div class="charts-grid">
                    <div class="chart-card">
                        <h3>Requests Over Time</h3>
                        <canvas id="requests-chart"></canvas>
                    </div>
                    
                    <div class="chart-card">
                        <h3>Tokens Usage</h3>
                        <canvas id="tokens-chart"></canvas>
                    </div>
                    
                    <div class="chart-card">
                        <h3>Models Distribution</h3>
                        <canvas id="models-pie-chart"></canvas>
                    </div>
                    
                    <div class="chart-card">
                        <h3>Response Times</h3>
                        <canvas id="latency-chart"></canvas>
                    </div>
                </div>
                
                <!-- Detailed Tables -->
                <div class="usage-tables">
                    <h3>API Keys Usage Breakdown</h3>
                    <table class="usage-table">
                        <thead>
                            <tr>
                                <th>API Key</th>
                                <th>Requests</th>
                                <th>Tokens</th>
                                <th>Avg Latency</th>
                                <th>Error Rate</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${stats.keys_breakdown.map(k => `
                                <tr>
                                    <td>${k.name}</td>
                                    <td>${k.requests}</td>
                                    <td>${this.formatNumber(k.tokens)}</td>
                                    <td>${k.avg_latency}ms</td>
                                    <td>${k.error_rate}%</td>
                                </tr>
                            `).join('')}
                        </tbody>
                    </table>
                </div>
                
                <!-- Export Button -->
                <button class="btn-secondary" onclick="exportUsageReport()">
                    <i class="bi bi-download"></i> Export Report (CSV)
                </button>
            </div>
        `;
    }
}
```

### 6. Preferences

```html
<div class="preferences-section">
    <h2>Preferences</h2>
    
    <!-- Theme Selection -->
    <div class="preference-group">
        <h3>Appearance</h3>
        <div class="theme-selector">
            <div class="theme-option" data-theme="light">
                <div class="theme-preview light"></div>
                <span>Light</span>
            </div>
            <div class="theme-option active" data-theme="dark">
                <div class="theme-preview dark"></div>
                <span>Dark</span>
            </div>
            <div class="theme-option" data-theme="auto">
                <div class="theme-preview auto"></div>
                <span>Auto</span>
            </div>
        </div>
    </div>
    
    <!-- Language -->
    <div class="preference-group">
        <h3>Language</h3>
        <select name="language" class="form-control">
            <option value="en">English</option>
            <option value="ru" selected>Русский</option>
            <option value="es">Español</option>
        </select>
    </div>
    
    <!-- Chat Settings -->
    <div class="preference-group">
        <h3>Chat Settings</h3>
        <label>
            <input type="checkbox" name="auto_scroll" checked>
            Auto-scroll to new messages
        </label>
        <label>
            <input type="checkbox" name="syntax_highlight" checked>
            Enable syntax highlighting
        </label>
        <label>
            <input type="checkbox" name="render_latex" checked>
            Render LaTeX math
        </label>
    </div>
    
    <!-- Notifications -->
    <div class="preference-group">
        <h3>Notifications</h3>
        <label>
            <input type="checkbox" name="email_notifications">
            Email notifications for important events
        </label>
        <label>
            <input type="checkbox" name="browser_notifications">
            Browser notifications
        </label>
    </div>
    
    <button class="btn-primary" onclick="savePreferences()">Save Preferences</button>
</div>
```

---

## 🔌 Backend API Endpoints

```
# User Profile
GET    /api/users/me                 # Current user info
PUT    /api/users/me                 # Update profile
POST   /api/users/me/password        # Change password
DELETE /api/users/me                 # Delete account
POST   /api/users/me/avatar          # Upload avatar

# Personal API Keys
GET    /api/users/me/api-keys        # List personal keys
POST   /api/users/me/api-keys        # Create personal key
GET    /api/users/me/api-keys/:id    # Get key details
PUT    /api/users/me/api-keys/:id    # Update key
DELETE /api/users/me/api-keys/:id    # Delete key

# Usage Stats
GET    /api/users/me/usage           # Usage statistics
GET    /api/users/me/usage/export    # Export usage report

# Preferences
GET    /api/users/me/preferences     # Get preferences
PUT    /api/users/me/preferences     # Update preferences

# Tenant Switching
POST   /api/auth/switch-tenant       # Switch current tenant context
```

---

## 🎯 Success Criteria

- ✅ User может просматривать и редактировать профиль
- ✅ User может управлять личными API ключами
- ✅ User может видеть детальную статистику использования
- ✅ User может переключаться между tenants
- ✅ User может настраивать preferences
- ✅ Responsive design работает на мобильных устройствах
- ✅ Все формы валидируются корректно
- ✅ Avatar upload работает

---

**Автор:** AI Assistant  
**Дата создания:** 2025-10-05  
**Последнее обновление:** 2025-10-05

# 🔄 WebUI Migration Guide

## From Old WebUI (`cmd/webui`) to New Integrated WebUI

**Date:** 2025-10-06  
**Version:** 1.3.0

---

## 📊 Overview

| Feature | Old WebUI (Deprecated) | New Integrated WebUI |
|---------|----------------------|---------------------|
| **Server** | Separate (port 8081) | Integrated (port 8080) |
| **Authentication** | Admin key only | Full JWT auth + Multi-tenancy |
| **User Management** | ❌ No | ✅ Yes |
| **Chat Interface** | ❌ No | ✅ ChatGPT-like |
| **API Keys** | Basic view/create | Full CRUD + scoping |
| **Bootstrap** | ❌ No | ✅ Yes |
| **Responsive** | Limited | ✅ Full mobile support |
| **Real-time** | Polling (5s) | ✅ WebSocket + Streaming |

---

## 🚀 Quick Migration

### Step 1: Stop Old WebUI
```bash
# If old webui is running, stop it
pkill -f "webui.exe"
# or
pkill -f "cmd/webui"
```

### Step 2: Update to Latest Version
```bash
git pull origin main
go mod tidy
```

### Step 3: Run Main Server
```bash
# Just run the main server
go run ./cmd/server

# or build and run
go build ./cmd/server
./server.exe
```

### Step 4: Access New WebUI
```
Old URL: http://localhost:8081/          # DON'T USE
New URL: http://localhost:8080/          # USE THIS ✅
```

### Step 5: Bootstrap (First Time Only)
1. Open `http://localhost:8080/`
2. Enter bootstrap token from `configs/dev.yaml` (`auth.admin_key`)
3. Create first admin account
4. Login and enjoy! 🎉

---

## 🗺️ URL Mapping

| Old WebUI | New Integrated WebUI |
|-----------|---------------------|
| `http://localhost:8081/` | `http://localhost:8080/` or `/login.html` |
| `http://localhost:8081/` (dashboard) | `http://localhost:8080/dashboard.html` |
| Not available | `http://localhost:8080/index.html` (Chat) |
| Not available | `http://localhost:8080/profile.html` |
| `/api/stats` (via proxy) | `/api/stats` (direct) |
| `/api/admin/keys` (via proxy) | `/api/users/me/api-keys` (JWT auth) |

---

## 🔑 Authentication Changes

### Old WebUI
```javascript
// Admin key in every request
fetch('/api/admin/keys', {
    headers: {
        'X-Admin-Key': 'sk-admin-...'
    }
})
```

### New WebUI
```javascript
// JWT token from login
fetch('/api/users/me/api-keys', {
    headers: {
        'Authorization': 'Bearer ' + access_token
    }
})
```

---

## 📁 File Structure Changes

### Old WebUI Files (DEPRECATED - can be removed)
```
/cmd/webui/
├── main.go              # Separate server
├── README.md            # Old docs
└── static/              # Old UI files
    ├── index.html
    ├── styles.css
    └── app.js
```

### New WebUI Files (CURRENT)
```
/web/                    # New location
├── index.html          # Chat interface
├── login.html          # Login page
├── dashboard.html      # Dashboard
├── profile.html        # Profile
├── css/
│   ├── style.css
│   └── dashboard.css
└── js/
    ├── auth-guard.js
    ├── api.js
    ├── chat.js
    └── ...

/internal/api/router/    # Static serving
└── router.go           # Serves /web at root
```

---

## ⚙️ Configuration Changes

### Old Config (cmd/webui)
```bash
./webui.exe -server-url http://localhost:8080 -port 8081
```

### New Config (integrated)
```yaml
# configs/dev.yaml
server:
  port: 8080          # One server, one port

auth:
  enabled: true
  admin_key: "sk-admin-dev-key-12345"
  
  jwt:
    secret: "your-secret"
    access_token_expiry: "15m"
    refresh_token_expiry: "168h"

database:
  type: "sqlite"
  sqlite:
    path: "./data/proxy.db"
```

---

## 🎯 Feature Comparison

### Metrics Dashboard

**Old:**
- Basic stats cards
- 5-second polling
- Admin key required

**New:**
- Rich dashboard with charts
- WebSocket real-time updates
- JWT authenticated
- Per-user/tenant stats

### API Key Management

**Old:**
```javascript
// Create key with admin key
POST /api/admin/keys
Headers: { X-Admin-Key: "..." }
```

**New:**
```javascript
// Personal API key
POST /api/users/me/api-keys
Headers: { Authorization: "Bearer ..." }

// Tenant API key
POST /api/tenants/:id/api-keys
Headers: { Authorization: "Bearer ..." }
```

### Chat Interface

**Old:**
- ❌ Not available
- Had to use external tools

**New:**
- ✅ Full ChatGPT-like UI
- ✅ Markdown + code highlighting
- ✅ Real-time streaming
- ✅ Conversation history

---

## 🔐 Security Improvements

### Old WebUI
- ❌ No user accounts
- ❌ Single admin key for all operations
- ❌ No audit trail
- ❌ No multi-tenancy

### New WebUI
- ✅ Full user authentication
- ✅ JWT with refresh tokens
- ✅ Role-based access control
- ✅ Multi-tenancy support
- ✅ Audit logging
- ✅ Bootstrap protection

---

## 📝 Code Examples

### Old: Admin Key API Call
```javascript
// Old way - admin key
const response = await fetch('http://localhost:8081/api/admin/keys', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        'X-Admin-Key': 'sk-admin-dev-key-12345'
    },
    body: JSON.stringify({
        name: 'My API Key',
        rate_limit: 60
    })
});
```

### New: JWT Authenticated API Call
```javascript
// New way - JWT token
const response = await fetch('http://localhost:8080/api/users/me/api-keys', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('access_token')}`
    },
    body: JSON.stringify({
        name: 'My API Key',
        rate_limits: {
            requests_per_minute: 60,
            requests_per_hour: 1000
        },
        models: ['llama2', 'mistral'],
        permissions: ['chat', 'models']
    })
});
```

---

## 🐛 Common Issues

### Issue: "Cannot connect to WebUI"

**Old:** Forgot to start webui server
```bash
./bin/webui.exe  # Need to run separately
```

**New:** Just run main server
```bash
go run ./cmd/server  # WebUI included ✅
```

---

### Issue: "Admin key not working"

**Old:**
```javascript
Headers: { 'X-Admin-Key': '...' }
```

**New:**
```javascript
// Use JWT token from login
Headers: { 'Authorization': 'Bearer ...' }
```

---

### Issue: "404 on /api/admin/keys"

**Old:**
```
GET http://localhost:8081/api/admin/keys
```

**New:**
```
GET http://localhost:8080/api/users/me/api-keys  ✅
```

---

## ✅ Benefits of New WebUI

1. **Unified Architecture**
   - One server process instead of two
   - Shared configuration
   - Consistent authentication

2. **Better Security**
   - User accounts with RBAC
   - JWT tokens (short-lived)
   - Multi-tenancy isolation
   - Audit logging

3. **More Features**
   - Chat interface (ChatGPT-like)
   - User profile management
   - Organization/team support
   - Personal + tenant API keys
   - Usage statistics

4. **Better UX**
   - Modern responsive design
   - Real-time updates (WebSocket)
   - Streaming responses
   - Mobile-friendly

5. **Easier Deployment**
   - Single binary
   - One port to expose
   - Simpler Docker setup
   - Less configuration

---

## 🔄 Rollback (If Needed)

If you need to temporarily use old WebUI:

```bash
# Build old webui
cd cmd/webui
go build -o ../../bin/webui.exe

# Run old webui (port 8081)
./bin/webui.exe -server-url http://localhost:8080
```

**Note:** Old WebUI is deprecated and will be removed in future versions.

---

## 📞 Support

If you encounter issues during migration:

1. Check logs: `tail -f logs/proxy-dev.log`
2. Clear browser cache and localStorage
3. Verify config: `cat configs/dev.yaml`
4. Check GitHub Issues
5. Ask in Discussions

---

## 🎓 Learning Resources

- [WEBUI.md](WEBUI.md) - Full WebUI documentation
- [README.md](README.md) - Project overview
- [Architecture.MD](Architecture.MD) - System architecture
- [Plan.md](Plan.md) - Development roadmap

---

**Status:** Old WebUI is **DEPRECATED**. Please migrate to new integrated WebUI.

**Timeline:** Old WebUI will be removed in version 2.0.0 (planned Q1 2026).


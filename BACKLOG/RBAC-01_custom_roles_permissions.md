# RBAC-01: Custom Roles & Permissions

**Версия:** 1.9.0  
**Приоритет:** Medium  
**Сложность:** Medium  
**Оценка:** 10-14 часов

## Описание

Fine-grained Role-Based Access Control (RBAC) система с custom roles и permissions. Админы смогут создавать кастомные роли с специфичными разрешениями для более гибкого управления доступом.

## Проблема

В текущей версии:
- Только фиксированные роли (admin, user)
- Нет granular permissions
- Невозможно создать custom roles (e.g., "API Manager", "Read-only Admin")
- Tenant roles ограничены (owner, admin, member)
- Нет permission inheritance

## Решение

Flexible RBAC с custom roles, permissions, и role templates.

### Архитектура

```
User → [Has Role] → Role → [Has Permissions] → Permission
                                                      ↓
                                              [Resource + Action]
```

## Технические детали

### Permission Model

```go
// internal/models/rbac.go

type Permission struct {
    ID          string    `json:"id" db:"id"`
    Name        string    `json:"name" db:"name"`               // e.g., "api_keys:create"
    Description string    `json:"description" db:"description"` // e.g., "Create API keys"
    Resource    string    `json:"resource" db:"resource"`       // e.g., "api_keys"
    Action      string    `json:"action" db:"action"`           // e.g., "create", "read", "update", "delete"
    Scope       string    `json:"scope" db:"scope"`             // "global", "tenant", "personal"
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type Role struct {
    ID          string       `json:"id" db:"id"`
    Name        string       `json:"name" db:"name"`                     // e.g., "api_manager"
    DisplayName string       `json:"display_name" db:"display_name"`    // e.g., "API Manager"
    Description string       `json:"description" db:"description"`
    Type        string       `json:"type" db:"type"`                    // "system", "custom"
    Scope       string       `json:"scope" db:"scope"`                  // "global", "tenant"
    TenantID    *string      `json:"tenant_id,omitempty" db:"tenant_id"` // null for global roles
    Permissions []Permission `json:"permissions,omitempty"`
    CreatedAt   time.Time    `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}

type UserRole struct {
    ID        string    `json:"id" db:"id"`
    UserID    string    `json:"user_id" db:"user_id"`
    RoleID    string    `json:"role_id" db:"role_id"`
    TenantID  *string   `json:"tenant_id,omitempty" db:"tenant_id"` // null for global roles
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
```

### Database Schema

```sql
-- Permissions table
CREATE TABLE permissions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT 'global', -- 'global', 'tenant', 'personal'
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Roles table
CREATE TABLE roles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL DEFAULT 'custom', -- 'system', 'custom'
    scope TEXT NOT NULL DEFAULT 'global', -- 'global', 'tenant'
    tenant_id TEXT, -- null for global roles
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE(name, tenant_id) -- Unique name per tenant (or global if tenant_id is null)
);

-- Role-Permission mapping
CREATE TABLE role_permissions (
    role_id TEXT NOT NULL,
    permission_id TEXT NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

-- User-Role assignments
CREATE TABLE user_roles (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    tenant_id TEXT, -- null for global role assignment
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE(user_id, role_id, tenant_id) -- Can't assign same role twice
);

CREATE INDEX idx_roles_tenant_id ON roles(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX idx_permissions_resource ON permissions(resource);
```

### Predefined Permissions

```go
// internal/rbac/permissions.go

var SystemPermissions = []Permission{
    // API Keys
    {Name: "api_keys:create", Resource: "api_keys", Action: "create", Scope: "global"},
    {Name: "api_keys:read", Resource: "api_keys", Action: "read", Scope: "global"},
    {Name: "api_keys:update", Resource: "api_keys", Action: "update", Scope: "global"},
    {Name: "api_keys:delete", Resource: "api_keys", Action: "delete", Scope: "global"},
    
    // Users
    {Name: "users:create", Resource: "users", Action: "create", Scope: "global"},
    {Name: "users:read", Resource: "users", Action: "read", Scope: "global"},
    {Name: "users:update", Resource: "users", Action: "update", Scope: "global"},
    {Name: "users:delete", Resource: "users", Action: "delete", Scope: "global"},
    
    // Tenants
    {Name: "tenants:create", Resource: "tenants", Action: "create", Scope: "global"},
    {Name: "tenants:read", Resource: "tenants", Action: "read", Scope: "global"},
    {Name: "tenants:update", Resource: "tenants", Action: "update", Scope: "tenant"},
    {Name: "tenants:delete", Resource: "tenants", Action: "delete", Scope: "global"},
    
    // Chat / Models
    {Name: "chat:use", Resource: "chat", Action: "use", Scope: "personal"},
    {Name: "models:read", Resource: "models", Action: "read", Scope: "global"},
    {Name: "models:manage", Resource: "models", Action: "manage", Scope: "global"},
    
    // System
    {Name: "system:config", Resource: "system", Action: "config", Scope: "global"},
    {Name: "system:backup", Resource: "system", Action: "backup", Scope: "global"},
    {Name: "system:logs", Resource: "system", Action: "logs", Scope: "global"},
    {Name: "audit:read", Resource: "audit", Action: "read", Scope: "global"},
}
```

### Predefined Roles

```go
// internal/rbac/roles.go

var SystemRoles = []Role{
    {
        Name:        "super_admin",
        DisplayName: "Super Administrator",
        Type:        "system",
        Scope:       "global",
        Permissions: []string{
            "*:*", // All permissions
        },
    },
    {
        Name:        "admin",
        DisplayName: "Administrator",
        Type:        "system",
        Scope:       "global",
        Permissions: []string{
            "users:*",
            "tenants:*",
            "api_keys:*",
            "audit:read",
            "system:logs",
        },
    },
    {
        Name:        "api_manager",
        DisplayName: "API Manager",
        Type:        "system",
        Scope:       "global",
        Permissions: []string{
            "api_keys:create",
            "api_keys:read",
            "api_keys:update",
            "api_keys:delete",
            "models:read",
        },
    },
    {
        Name:        "user",
        DisplayName: "Regular User",
        Type:        "system",
        Scope:       "global",
        Permissions: []string{
            "chat:use",
            "models:read",
            "api_keys:read", // Only own keys
        },
    },
    {
        Name:        "read_only",
        DisplayName: "Read Only",
        Type:        "system",
        Scope:       "global",
        Permissions: []string{
            "users:read",
            "tenants:read",
            "api_keys:read",
            "models:read",
            "audit:read",
        },
    },
}
```

### RBAC Service

```go
// internal/services/rbac/rbac.go

type RBACService struct {
    db     storage.Database
    logger *logrus.Logger
}

func (r *RBACService) CheckPermission(
    ctx context.Context,
    userID string,
    permission string,
    tenantID *string,
) (bool, error) {
    // Get user roles
    roles, err := r.db.GetUserRoles(ctx, userID)
    if err != nil {
        return false, err
    }
    
    // Check each role
    for _, role := range roles {
        // Skip tenant roles if checking global permission
        if role.TenantID != nil && tenantID == nil {
            continue
        }
        
        // Skip global roles if checking tenant permission
        if role.TenantID == nil && tenantID != nil {
            continue
        }
        
        // Get role permissions
        permissions, err := r.db.GetRolePermissions(ctx, role.ID)
        if err != nil {
            continue
        }
        
        // Check if permission exists
        for _, p := range permissions {
            if matchPermission(p.Name, permission) {
                return true, nil
            }
        }
    }
    
    return false, nil
}

func matchPermission(pattern, permission string) bool {
    // Support wildcards: "api_keys:*" matches "api_keys:create", "api_keys:read", etc.
    if pattern == "*:*" {
        return true
    }
    
    parts := strings.Split(pattern, ":")
    permParts := strings.Split(permission, ":")
    
    if len(parts) != 2 || len(permParts) != 2 {
        return pattern == permission
    }
    
    // Check resource
    if parts[0] != "*" && parts[0] != permParts[0] {
        return false
    }
    
    // Check action
    if parts[1] != "*" && parts[1] != permParts[1] {
        return false
    }
    
    return true
}
```

### RBAC Middleware

```go
// internal/api/middleware/rbac.go

func RequirePermission(rbac *rbac.RBACService, permission string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, exists := c.Get("user_id")
        if !exists {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
            c.Abort()
            return
        }
        
        // Get tenant ID from context (if applicable)
        tenantID, _ := c.Get("tenant_id")
        var tenantIDPtr *string
        if tenantID != nil {
            tid := tenantID.(string)
            tenantIDPtr = &tid
        }
        
        // Check permission
        hasPermission, err := rbac.CheckPermission(
            c.Request.Context(),
            userID.(string),
            permission,
            tenantIDPtr,
        )
        
        if err != nil || !hasPermission {
            c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

### Usage in Handlers

```go
// internal/api/router/router.go

func (r *Router) setupAPIRoutes() {
    api := r.engine.Group("/api")
    api.Use(middleware.JWTAuth(r.jwtService))
    
    // Require specific permissions
    api.POST("/admin/users",
        middleware.RequirePermission(r.rbacService, "users:create"),
        r.userHandler.CreateUser,
    )
    
    api.DELETE("/admin/users/:id",
        middleware.RequirePermission(r.rbacService, "users:delete"),
        r.userHandler.DeleteUser,
    )
    
    api.POST("/api-keys",
        middleware.RequirePermission(r.rbacService, "api_keys:create"),
        r.apiKeyHandler.CreateAPIKey,
    )
}
```

### Admin UI

**Admin Panel → Roles Management:**

```html
<div id="roles-tab" class="tab-pane">
  <h3>Roles & Permissions</h3>
  
  <button onclick="createCustomRole()">Create Custom Role</button>
  
  <!-- Roles List -->
  <table class="roles-table">
    <thead>
      <tr>
        <th>Role Name</th>
        <th>Type</th>
        <th>Scope</th>
        <th>Permissions</th>
        <th>Actions</th>
      </tr>
    </thead>
    <tbody id="roles-list">
      <!-- Populated by JS -->
    </tbody>
  </table>
  
  <!-- Create/Edit Role Modal -->
  <div id="role-modal" class="modal">
    <h3>Create Custom Role</h3>
    <input type="text" id="role-name" placeholder="Role Name" />
    <input type="text" id="role-display-name" placeholder="Display Name" />
    <textarea id="role-description" placeholder="Description"></textarea>
    
    <h4>Permissions</h4>
    <div id="permissions-checkboxes">
      <!-- Checkboxes for each permission -->
      <label><input type="checkbox" value="api_keys:create" /> Create API Keys</label>
      <label><input type="checkbox" value="api_keys:read" /> Read API Keys</label>
      <!-- ... -->
    </div>
    
    <button onclick="saveRole()">Save Role</button>
  </div>
</div>
```

## Требования

### Функциональные

1. ✅ Permission model с resource + action
2. ✅ Role model с permissions
3. ✅ User-Role assignments
4. ✅ CheckPermission function
5. ✅ Wildcard permissions (*:*)
6. ✅ Custom role creation
7. ✅ Role templates (predefined roles)
8. ✅ Tenant-scoped roles
9. ✅ RBAC middleware
10. ✅ Admin UI для role management

### Нефункциональные

1. **Performance**
   - Permission check < 10ms
   - Caching для role permissions

2. **Flexibility**
   - Wildcard support
   - Easy to add new permissions

## Acceptance Criteria

- [ ] Permissions определены в системе
- [ ] Roles созданы с permissions
- [ ] User-Role assignments работают
- [ ] CheckPermission корректно валидирует доступ
- [ ] Wildcard permissions работают
- [ ] Admin может создать custom role
- [ ] Admin может assign roles пользователям
- [ ] RBAC middleware блокирует unauthorized requests
- [ ] Admin UI отображает roles и permissions
- [ ] System roles (super_admin, admin, user) работают
- [ ] Unit tests для RBAC logic
- [ ] Integration tests для permission checks

## Риски и зависимости

### Риски

1. **Performance** - множественные DB queries
   - Mitigation: Caching, preloading permissions

2. **Complexity** - слишком granular permissions
   - Mitigation: Sane defaults, templates

### Зависимости

1. Existing auth system
2. Database models

## Связанные задачи

- **OIDC-02**: Auto-tenant from Groups - role assignment from groups
- **LDAP-01**: LDAP Integration - role assignment from LDAP groups
- **AUDIT-01**: Enhanced Audit Logging - логирование role changes

## Примечания

- System roles нельзя удалить/редактировать
- Custom roles можно создавать только admins
- Permission inheritance - в future versions

---

**Статус:** 📋 Planned for v1.9.0  
**Последнее обновление:** 2025-10-11


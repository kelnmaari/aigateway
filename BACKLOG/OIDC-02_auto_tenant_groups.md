# OIDC-02: Auto-tenant Provisioning from OIDC Groups

**Версия:** 1.9.0  
**Приоритет:** High  
**Сложность:** Medium  
**Оценка:** 8-12 часов

## Описание

Автоматическое создание и привязка tenants на основе OIDC group claims из Keycloak. Пользователи автоматически добавляются в организации (tenants) согласно их группам в Keycloak.

## Проблема

После внедрения OIDC (OIDC-01):
- Пользователи создаются, но не привязаны к tenants
- Нужно вручную создавать tenants и добавлять members
- Нет синхронизации групп из Keycloak
- Управление membership дублируется в двух системах

## Решение

Автоматический mapping OIDC groups → tenants + auto-provisioning.

### Архитектура

```
Keycloak Groups → OIDC ID Token (groups claim)
                         ↓
              Proxy Server (OIDC callback)
                         ↓
              [Parse groups claim]
                         ↓
         [Create/Update tenants + memberships]
                         ↓
              User → Member of Tenants
```

## Технические детали

### Configuration

```yaml
# config.yaml
auth:
  oidc:
    # ... (from OIDC-01)
    
    # Tenant provisioning
    tenant_provisioning:
      enabled: true
      
      # Auto-create tenants from groups
      auto_create_tenants: true
      
      # Group → Tenant mapping
      group_mapping:
        # Simple 1:1 mapping
        mode: "direct" # or "prefix"
        
        # Prefix-based mapping (optional)
        # Group: "/engineering/backend" → Tenant: "backend"
        # Group: "/sales/emea" → Tenant: "emea"
        prefix: "/engineering/"
        
        # Group → Role mapping
        admin_groups:
          - "engineering-admins"
          - "tenant-admins"
        
      # Sync strategy
      sync_on_login: true
      remove_orphaned_memberships: false # Don't remove if group removed
```

### Implementation

**1. Group Claims Extraction**

```go
// internal/auth/oidc/claims.go

type OIDCClaims struct {
    // ... (from OIDC-01)
    Groups []string `json:"groups"` // Keycloak groups
}

func (c *OIDCClaims) ParseGroups(config *GroupMappingConfig) []TenantMapping {
    var mappings []TenantMapping
    
    for _, group := range c.Groups {
        // Apply mapping rules
        tenantName := applyGroupMapping(group, config)
        if tenantName == "" {
            continue // Skip unmapped groups
        }
        
        // Determine role
        role := models.TenantRoleMember
        if contains(config.AdminGroups, group) {
            role = models.TenantRoleAdmin
        }
        
        mappings = append(mappings, TenantMapping{
            TenantName: tenantName,
            Role:       role,
        })
    }
    
    return mappings
}

func applyGroupMapping(group string, config *GroupMappingConfig) string {
    switch config.Mode {
    case "direct":
        // Direct 1:1 mapping
        return group
        
    case "prefix":
        // Extract tenant name after prefix
        // "/engineering/backend" → "backend"
        if strings.HasPrefix(group, config.Prefix) {
            return strings.TrimPrefix(group, config.Prefix)
        }
        return ""
        
    default:
        return group
    }
}

type TenantMapping struct {
    TenantName string
    Role       models.TenantRole
}
```

**2. Tenant Auto-provisioning Service**

```go
// internal/services/tenant/provisioner.go

type TenantProvisioner struct {
    db     storage.Database
    config *config.TenantProvisioningConfig
    logger *logrus.Logger
}

func (p *TenantProvisioner) ProvisionTenantsForUser(
    ctx context.Context,
    userID string,
    mappings []TenantMapping,
) error {
    for _, mapping := range mappings {
        // Get or create tenant
        tenant, err := p.getOrCreateTenant(ctx, mapping.TenantName)
        if err != nil {
            p.logger.WithError(err).Errorf("Failed to get/create tenant: %s", mapping.TenantName)
            continue
        }
        
        // Add user to tenant
        if err := p.addUserToTenant(ctx, userID, tenant.ID, mapping.Role); err != nil {
            p.logger.WithError(err).Errorf("Failed to add user to tenant: %s", tenant.ID)
            continue
        }
        
        p.logger.WithFields(logrus.Fields{
            "user_id":   userID,
            "tenant_id": tenant.ID,
            "role":      mapping.Role,
        }).Info("User provisioned to tenant")
    }
    
    // Optionally remove orphaned memberships
    if p.config.RemoveOrphanedMemberships {
        if err := p.removeOrphanedMemberships(ctx, userID, mappings); err != nil {
            p.logger.WithError(err).Error("Failed to remove orphaned memberships")
        }
    }
    
    return nil
}

func (p *TenantProvisioner) getOrCreateTenant(ctx context.Context, name string) (*models.Tenant, error) {
    // Try to get existing tenant by name
    tenant, err := p.db.GetTenantByName(ctx, name)
    if err == nil {
        return tenant, nil
    }
    
    // Tenant doesn't exist, create if auto-create enabled
    if !p.config.AutoCreateTenants {
        return nil, fmt.Errorf("tenant '%s' not found and auto-create disabled", name)
    }
    
    tenant = &models.Tenant{
        ID:          uuid.New().String(),
        Name:        name,
        DisplayName: name, // Can be updated by admin later
        Type:        models.TenantTypeOrganization,
        Status:      models.TenantStatusActive,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
    
    if err := p.db.CreateTenant(ctx, tenant); err != nil {
        return nil, err
    }
    
    p.logger.WithField("tenant_id", tenant.ID).Infof("Auto-created tenant: %s", name)
    
    return tenant, nil
}

func (p *TenantProvisioner) addUserToTenant(
    ctx context.Context,
    userID, tenantID string,
    role models.TenantRole,
) error {
    // Check if already a member
    member, err := p.db.GetTenantMember(ctx, tenantID, userID)
    if err == nil && member != nil {
        // Update role if different
        if member.Role != role {
            member.Role = role
            member.UpdatedAt = time.Now()
            return p.db.UpdateTenantMember(ctx, member)
        }
        return nil // Already a member with correct role
    }
    
    // Add as new member
    member = &models.TenantMember{
        ID:        uuid.New().String(),
        TenantID:  tenantID,
        UserID:    userID,
        Role:      role,
        Status:    models.TenantMemberStatusActive,
        JoinedAt:  time.Now(),
        UpdatedAt: time.Now(),
    }
    
    return p.db.CreateTenantMember(ctx, member)
}

func (p *TenantProvisioner) removeOrphanedMemberships(
    ctx context.Context,
    userID string,
    validMappings []TenantMapping,
) error {
    // Get all current memberships
    memberships, err := p.db.GetUserTenantMemberships(ctx, userID)
    if err != nil {
        return err
    }
    
    // Build valid tenant names set
    validTenants := make(map[string]bool)
    for _, mapping := range validMappings {
        validTenants[mapping.TenantName] = true
    }
    
    // Remove memberships not in valid set
    for _, membership := range memberships {
        tenant, err := p.db.GetTenant(ctx, membership.TenantID)
        if err != nil {
            continue
        }
        
        if !validTenants[tenant.Name] {
            if err := p.db.DeleteTenantMember(ctx, membership.ID); err != nil {
                p.logger.WithError(err).Warnf("Failed to remove orphaned membership: %s", membership.ID)
            } else {
                p.logger.WithFields(logrus.Fields{
                    "user_id":   userID,
                    "tenant_id": tenant.ID,
                }).Info("Removed orphaned tenant membership")
            }
        }
    }
    
    return nil
}
```

**3. Integration with OIDC Handler**

```go
// internal/api/handlers/oidc.go (update from OIDC-01)

func (h *OIDCHandler) Callback(c *gin.Context) {
    // ... (existing code from OIDC-01)
    
    // Extract claims
    var claims OIDCClaims
    if err := idToken.Claims(&claims); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse claims"})
        return
    }
    
    // Create or update user
    user, err := h.provisionUser(c.Request.Context(), &claims)
    if err != nil {
        // ...
    }
    
    // ✅ NEW: Provision tenants from groups
    if h.tenantProvisioner != nil && len(claims.Groups) > 0 {
        mappings := claims.ParseGroups(h.config.TenantProvisioning.GroupMapping)
        if err := h.tenantProvisioner.ProvisionTenantsForUser(c.Request.Context(), user.ID, mappings); err != nil {
            h.logger.WithError(err).Error("Failed to provision tenants")
            // Don't fail the login, just log
        }
    }
    
    // Issue JWT token
    // ...
}
```

### Database Schema

**Add tenant name index for faster lookups:**
```sql
CREATE UNIQUE INDEX idx_tenants_name ON tenants(name);
```

### Admin UI

**Admin Panel → Tenants → Auto-provisioned badge:**

```html
<div class="tenant-card">
  <h3>Engineering Team</h3>
  <span class="badge badge-auto">Auto-provisioned from OIDC</span>
  <p>Members: 12 (10 from OIDC groups)</p>
</div>
```

**Admin Panel → Settings → OIDC → Group Mapping:**

```html
<div class="group-mapping-settings">
  <h4>Group → Tenant Mapping</h4>
  
  <select id="mapping-mode">
    <option value="direct">Direct (1:1)</option>
    <option value="prefix">Prefix-based</option>
  </select>
  
  <input type="text" id="group-prefix" placeholder="/engineering/" />
  
  <h5>Admin Groups</h5>
  <textarea id="admin-groups" placeholder="engineering-admins&#10;tenant-admins"></textarea>
  
  <label>
    <input type="checkbox" id="auto-create-tenants" />
    Auto-create tenants from groups
  </label>
  
  <label>
    <input type="checkbox" id="remove-orphaned" />
    Remove memberships when group removed
  </label>
  
  <button onclick="saveGroupMapping()">Save Mapping Rules</button>
</div>
```

## Keycloak Configuration

**1. Create Groups in Keycloak:**
```
Realm: myrealm
├─ engineering
│  ├─ backend
│  └─ frontend
├─ sales
└─ admins
```

**2. Add Group Mapper to Client:**
```
Client: ollama-proxy
Mapper Name: groups
Mapper Type: Group Membership
Token Claim Name: groups
Full group path: ON/OFF (depending on mapping mode)
```

**3. Assign Users to Groups:**
```
User: john.doe
Groups:
  - /engineering/backend
  - /admins
```

## Требования

### Функциональные

1. ✅ Extract groups claim из OIDC ID token
2. ✅ Parse groups согласно mapping rules
3. ✅ Auto-create tenants из groups
4. ✅ Add users to tenants автоматически
5. ✅ Role assignment (admin vs member)
6. ✅ Sync on every login (configurable)
7. ✅ Remove orphaned memberships (optional)
8. ✅ Admin UI для mapping configuration
9. ✅ Prefix-based и direct mapping modes
10. ✅ Audit logging для tenant provisioning

### Нефункциональные

1. **Performance**
   - Tenant provisioning < 500ms на login
   - Bulk sync для множества groups

2. **Reliability**
   - Idempotent operations (re-login не создает дубликаты)
   - Graceful handling если tenant creation fails

## Acceptance Criteria

- [ ] User логинится через OIDC с groups claim
- [ ] Groups извлекаются из ID token
- [ ] Tenants автоматически создаются из groups
- [ ] User добавляется в tenants как member/admin
- [ ] Role assignment работает (admin groups)
- [ ] Prefix-based mapping работает корректно
- [ ] Re-login обновляет memberships (idempotent)
- [ ] Orphaned memberships удаляются (если enabled)
- [ ] Admin UI отображает auto-provisioned tenants
- [ ] Audit log содержит provisioning events
- [ ] Unit tests для group parsing, mapping
- [ ] Integration tests для full provisioning flow

## Риски и зависимости

### Риски

1. **Group name conflicts** - одинаковые имена
   - Mitigation: Unique tenant names, validation

2. **Large group hierarchies** - сложные structures
   - Mitigation: Prefix-based mapping, clear rules

3. **Role conflicts** - user в multiple admin groups
   - Mitigation: Highest role wins

### Зависимости

1. **OIDC-01**: Keycloak SSO - базовая OIDC интеграция
2. Tenant model (existing)
3. TenantMember model (existing)

## Связанные задачи

- **OIDC-01**: Keycloak SSO - base dependency
- **LDAP-01**: LDAP Integration - схожая логика group sync
- **RBAC-01**: Custom Roles - дополнительные роли для tenants

## Примечания

- Только auto-provisioning (не full sync)
- Manual sync через Admin UI - в future versions
- Scheduled sync (cron) - в future versions
- Group hierarchy (nested groups) - limited support

## Пример

**Keycloak Groups:**
```
/engineering/backend → Tenant: "backend"
/engineering/frontend → Tenant: "frontend"
/sales/emea → Tenant: "emea"
```

**User john.doe (groups: /engineering/backend, /admins):**
```
After login:
- Tenant "backend" created (if not exists)
- john.doe → member of "backend" tenant
- Role: Admin (because "admins" group)
```

---

**Статус:** 📋 Planned for v1.9.0  
**Последнее обновление:** 2025-10-11


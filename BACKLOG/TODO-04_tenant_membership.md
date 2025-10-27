# TODO-04: Tenant Membership Check

**Версия:** v1.5.3  
**Приоритет:** MEDIUM  
**Оценка:** 2-3 часа  
**Зависимости:** Нет  
**Статус:** 📋 Planned  
**Тип:** Security / Code Quality

---

## 📋 Описание

Добавить проверку прав доступа к tenant ресурсам в handlers. Сейчас в коде есть TODO комментарий о необходимости проверки membership.

## 📊 Current Issue

**Файл:** `internal/api/handlers/usage.go` (строка 90)

```go
func (h *UsageHandlers) GetTenantUsage(c *gin.Context) {
    tenantID := c.Param("tenant_id")
    userID := c.GetString("user_id")
    
    // TODO: Add proper tenant membership check
    
    // Загрузка usage stats без проверки прав
    stats, err := h.storage.GetTenantUsageStats(c.Request.Context(), tenantID)
    // ...
}
```

**Проблема:** Пользователь может запросить usage stats любого tenant'а, даже если он не является членом.

## 🔧 Solution

### 1. Create Membership Check Helper

**Файл:** `internal/api/handlers/tenant_helpers.go` (новый)

```go
package handlers

import (
    "fmt"
    "github.com/gin-gonic/gin"
    "net/http"
)

// checkTenantMembership проверяет, является ли пользователь членом tenant
func (h *TenantHandlers) checkTenantMembership(c *gin.Context, tenantID, userID string) (bool, error) {
    member, err := h.storage.GetTenantMember(c.Request.Context(), tenantID, userID)
    if err != nil {
        return false, fmt.Errorf("failed to check membership: %w", err)
    }
    
    return member != nil, nil
}

// requireTenantMembership middleware для проверки членства
func (h *TenantHandlers) requireTenantMembership() gin.HandlerFunc {
    return func(c *gin.Context) {
        tenantID := c.Param("tenant_id")
        if tenantID == "" {
            tenantID = c.Param("id") // альтернативный параметр
        }
        
        userID := c.GetString("user_id")
        if userID == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
            c.Abort()
            return
        }
        
        isMember, err := h.checkTenantMembership(c, tenantID, userID)
        if err != nil {
            h.logger.WithError(err).Error("Failed to check tenant membership")
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
            c.Abort()
            return
        }
        
        if !isMember {
            h.logger.WithFields(map[string]interface{}{
                "user_id":   userID,
                "tenant_id": tenantID,
            }).Warn("Access denied: user is not a member of tenant")
            
            c.JSON(http.StatusForbidden, gin.H{
                "error": "Access denied: you are not a member of this tenant",
            })
            c.Abort()
            return
        }
        
        // Store tenant ID in context for later use
        c.Set("tenant_id", tenantID)
        c.Next()
    }
}
```

### 2. Update Usage Handlers

**Файл:** `internal/api/handlers/usage.go`

```go
// GetTenantUsage возвращает статистику использования tenant
func (h *UsageHandlers) GetTenantUsage(c *gin.Context) {
    tenantID := c.Param("tenant_id")
    userID := c.GetString("user_id")
    
    // Check tenant membership
    isMember, err := h.checkTenantMembership(c, tenantID, userID)
    if err != nil {
        h.logger.WithError(err).Error("Failed to check membership")
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to verify access",
        })
        return
    }
    
    if !isMember {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "Access denied: you are not a member of this tenant",
        })
        return
    }
    
    // Proceed with getting usage stats
    stats, err := h.storage.GetTenantUsageStats(c.Request.Context(), tenantID)
    if err != nil {
        h.logger.WithError(err).Error("Failed to get tenant usage")
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to get usage statistics",
        })
        return
    }
    
    c.JSON(http.StatusOK, stats)
}

// checkTenantMembership проверяет членство (helper method)
func (h *UsageHandlers) checkTenantMembership(c *gin.Context, tenantID, userID string) (bool, error) {
    member, err := h.tenantStorage.GetTenantMember(c.Request.Context(), tenantID, userID)
    if err != nil {
        return false, err
    }
    return member != nil, nil
}
```

### 3. Alternative: Use Middleware

**Файл:** `internal/api/router/router.go`

```go
// Setup tenant routes with membership check
tenants := api.Group("/tenants")
tenants.Use(authMiddleware.RequireAuth())
{
    // Public tenant endpoints (list, create)
    tenants.GET("", tenantHandlers.ListTenants)
    tenants.POST("", tenantHandlers.CreateTenant)
    
    // Tenant-specific endpoints (require membership)
    tenant := tenants.Group("/:tenant_id")
    tenant.Use(tenantHandlers.requireTenantMembership()) // <-- Middleware
    {
        tenant.GET("", tenantHandlers.GetTenant)
        tenant.PATCH("", tenantHandlers.UpdateTenant)
        tenant.DELETE("", tenantHandlers.DeleteTenant)
        tenant.GET("/usage", usageHandlers.GetTenantUsage)
        tenant.GET("/members", tenantHandlers.ListMembers)
        tenant.POST("/members", tenantHandlers.AddMember)
        tenant.DELETE("/members/:user_id", tenantHandlers.RemoveMember)
    }
}
```

### 4. Storage Layer (Если отсутствует)

**Файл:** `internal/storage/tenant.go`

```go
// GetTenantMember получает информацию о члене tenant
GetTenantMember(ctx context.Context, tenantID, userID string) (*TenantMember, error)
```

**Реализация:**

```go
func (s *SQLiteStorage) GetTenantMember(ctx context.Context, tenantID, userID string) (*models.TenantMember, error) {
    query := `
        SELECT id, tenant_id, user_id, role, created_at, updated_at
        FROM tenant_members
        WHERE tenant_id = ? AND user_id = ?
    `
    
    var member models.TenantMember
    err := s.db.GetContext(ctx, &member, query, tenantID, userID)
    if err == sql.ErrNoRows {
        return nil, nil // Not a member
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get tenant member: %w", err)
    }
    
    return &member, nil
}
```

## 📝 Implementation Plan

### Step 1: Storage Layer (30 мин)

- [ ] Проверить наличие GetTenantMember
- [ ] Реализовать для SQLite
- [ ] Реализовать для PostgreSQL

### Step 2: Helper Functions (30 мин)

- [ ] Создать checkTenantMembership helper
- [ ] Создать requireTenantMembership middleware

### Step 3: Update Handlers (45 мин)

- [ ] Обновить usage.go
- [ ] Обновить другие tenant handlers
- [ ] Удалить TODO комментарий

### Step 4: Testing (45 мин)

- [ ] Unit tests для membership check
- [ ] Integration tests
- [ ] Security testing

## ✅ Acceptance Criteria

- [ ] GetTenantMember storage method реализован
- [ ] Membership check добавлен в handlers
- [ ] Forbidden (403) для non-members
- [ ] Audit logging при access denied
- [ ] TODO комментарий удален
- [ ] Unit tests написаны
- [ ] Security tests пройдены

## 🧪 Testing

### Unit Tests

```go
func TestGetTenantUsage_NotMember(t *testing.T) {
    // Setup
    // ...
    
    // Test: user not a member
    req := httptest.NewRequest("GET", "/api/tenants/tenant-123/usage", nil)
    req.Header.Set("Authorization", "Bearer user-token")
    
    resp := httptest.NewRecorder()
    router.ServeHTTP(resp, req)
    
    assert.Equal(t, http.StatusForbidden, resp.Code)
    assert.Contains(t, resp.Body.String(), "not a member")
}

func TestGetTenantUsage_IsMember(t *testing.T) {
    // Test: user is a member
    // Should return 200 OK with stats
}
```

### API Testing

```bash
# As non-member (should fail)
curl -H "Authorization: Bearer $NON_MEMBER_TOKEN" \
     http://localhost:8080/api/tenants/tenant-123/usage

# Expected: 403 Forbidden
{"error": "Access denied: you are not a member of this tenant"}

# As member (should succeed)
curl -H "Authorization: Bearer $MEMBER_TOKEN" \
     http://localhost:8080/api/tenants/tenant-123/usage

# Expected: 200 OK with usage stats
```

## 🔒 Security Impact

**Перед исправлением:**

- Любой пользователь может видеть usage stats любого tenant
- Потенциальная утечка данных
- OWASP: Broken Access Control

**После исправления:**

- Только члены tenant могут видеть stats
- Audit logging попыток доступа
- Соответствие принципу least privilege

## 🔄 Related Fixes

Проверить и добавить membership check в:

- [ ] `GetTenant` handler
- [ ] `UpdateTenant` handler
- [ ] `ListTenantAPIKeys` handler
- [ ] `GetTenantMembers` handler

## 📚 References

- [OWASP Broken Access Control](https://owasp.org/www-project-top-ten/2017/A5_2017-Broken_Access_Control)
- [Authorization Best Practices](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)


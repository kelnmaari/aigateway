# AUDIT-01: Enhanced Audit Logging

**Версия:** 1.9.0  
**Приоритет:** High  
**Сложность:** Medium  
**Оценка:** 8-12 часов

## Описание

Comprehensive audit logging system для security events, compliance, и forensics. Детальное логирование всех критичных операций (authentication, authorization, API key changes, tenant operations).

## Проблема

В текущей версии:
- Базовое логирование через logrus
- Нет structured audit trail
- Невозможно отследить security events
- Нет compliance reports
- Логи смешаны с application logs

## Решение

Dedicated audit logging system с structured events, storage, и query API.

### Архитектура

```
Application Events → Audit Logger → Audit Storage (DB)
                                          ↓
                                    [Query API]
                                          ↓
                              Admin UI / Reports / Export
```

## Технические детали

### Audit Event Types

**1. Authentication Events**
- LOGIN_SUCCESS
- LOGIN_FAILED
- LOGOUT
- PASSWORD_CHANGED
- MFA_ENABLED / MFA_DISABLED
- SESSION_EXPIRED

**2. Authorization Events**
- PERMISSION_DENIED
- ROLE_CHANGED
- ACCESS_GRANTED

**3. API Key Events**
- API_KEY_CREATED
- API_KEY_DELETED
- API_KEY_UPDATED
- API_KEY_REVOKED
- API_KEY_USAGE_EXCEEDED

**4. Tenant Events**
- TENANT_CREATED
- TENANT_DELETED
- TENANT_MEMBER_ADDED
- TENANT_MEMBER_REMOVED
- TENANT_ROLE_CHANGED

**5. System Events**
- CONFIG_CHANGED
- BACKUP_CREATED
- BACKUP_RESTORED
- MODEL_LOADED / MODEL_UNLOADED

### Data Model

```go
// internal/models/audit.go

type AuditEvent struct {
    ID        string    `json:"id" db:"id"`
    EventType string    `json:"event_type" db:"event_type"` // LOGIN_SUCCESS, API_KEY_CREATED, etc.
    Severity  string    `json:"severity" db:"severity"`     // info, warning, critical
    
    // Actor (who performed the action)
    ActorID   string  `json:"actor_id" db:"actor_id"`     // User ID
    ActorType string  `json:"actor_type" db:"actor_type"` // user, system, api_key
    
    // Target (what was affected)
    TargetID   *string `json:"target_id,omitempty" db:"target_id"`
    TargetType *string `json:"target_type,omitempty" db:"target_type"` // user, tenant, api_key
    
    // Context
    Action      string                 `json:"action" db:"action"`             // login, create, delete, update
    Resource    string                 `json:"resource" db:"resource"`         // user, tenant, api_key, backup
    Status      string                 `json:"status" db:"status"`             // success, failure
    ErrorMsg    *string                `json:"error_msg,omitempty" db:"error_msg"`
    Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"` // JSON
    
    // Request info
    IPAddress string  `json:"ip_address" db:"ip_address"`
    UserAgent *string `json:"user_agent,omitempty" db:"user_agent"`
    
    Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

type AuditEventSeverity string

const (
    AuditSeverityInfo     AuditEventSeverity = "info"
    AuditSeverityWarning  AuditEventSeverity = "warning"
    AuditSeverityCritical AuditEventSeverity = "critical"
)
```

### Database Schema

```sql
CREATE TABLE audit_events (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'info',
    
    actor_id TEXT NOT NULL,
    actor_type TEXT NOT NULL DEFAULT 'user',
    
    target_id TEXT,
    target_type TEXT,
    
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    status TEXT NOT NULL,
    error_msg TEXT,
    metadata TEXT, -- JSON
    
    ip_address TEXT NOT NULL,
    user_agent TEXT,
    
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_events_timestamp ON audit_events(timestamp DESC);
CREATE INDEX idx_audit_events_actor_id ON audit_events(actor_id);
CREATE INDEX idx_audit_events_event_type ON audit_events(event_type);
CREATE INDEX idx_audit_events_severity ON audit_events(severity);
CREATE INDEX idx_audit_events_resource ON audit_events(resource);
CREATE INDEX idx_audit_events_target_id ON audit_events(target_id) WHERE target_id IS NOT NULL;
```

### Audit Logger Service

```go
// internal/services/audit/logger.go

type AuditLogger struct {
    db     storage.Database
    logger *logrus.Logger
}

func NewAuditLogger(db storage.Database, logger *logrus.Logger) *AuditLogger {
    return &AuditLogger{
        db:     db,
        logger: logger,
    }
}

func (a *AuditLogger) LogEvent(ctx context.Context, event *models.AuditEvent) error {
    // Set defaults
    if event.ID == "" {
        event.ID = uuid.New().String()
    }
    if event.Timestamp.IsZero() {
        event.Timestamp = time.Now()
    }
    if event.Severity == "" {
        event.Severity = string(models.AuditSeverityInfo)
    }
    
    // Store in database
    if err := a.db.CreateAuditEvent(ctx, event); err != nil {
        a.logger.WithError(err).Error("Failed to store audit event")
        return err
    }
    
    // Also log to standard logger for immediate visibility
    logEntry := a.logger.WithFields(logrus.Fields{
        "audit_event": event.EventType,
        "actor_id":    event.ActorID,
        "action":      event.Action,
        "resource":    event.Resource,
        "status":      event.Status,
    })
    
    switch event.Severity {
    case string(models.AuditSeverityInfo):
        logEntry.Info(event.Action)
    case string(models.AuditSeverityWarning):
        logEntry.Warn(event.Action)
    case string(models.AuditSeverityCritical):
        logEntry.Error(event.Action)
    }
    
    return nil
}

// Convenience methods
func (a *AuditLogger) LogLogin(ctx context.Context, userID, ipAddress string, success bool, errMsg *string) {
    event := &models.AuditEvent{
        EventType:  "LOGIN_SUCCESS",
        Severity:   string(models.AuditSeverityInfo),
        ActorID:    userID,
        ActorType:  "user",
        Action:     "login",
        Resource:   "authentication",
        Status:     "success",
        IPAddress:  ipAddress,
        ErrorMsg:   errMsg,
    }
    
    if !success {
        event.EventType = "LOGIN_FAILED"
        event.Status = "failure"
        event.Severity = string(models.AuditSeverityWarning)
    }
    
    a.LogEvent(ctx, event)
}

func (a *AuditLogger) LogAPIKeyCreated(ctx context.Context, actorID, keyID, ipAddress string) {
    event := &models.AuditEvent{
        EventType:  "API_KEY_CREATED",
        Severity:   string(models.AuditSeverityInfo),
        ActorID:    actorID,
        ActorType:  "user",
        TargetID:   &keyID,
        TargetType: stringPtr("api_key"),
        Action:     "create",
        Resource:   "api_key",
        Status:     "success",
        IPAddress:  ipAddress,
    }
    
    a.LogEvent(ctx, event)
}

func (a *AuditLogger) LogTenantMemberAdded(ctx context.Context, actorID, tenantID, memberID, ipAddress string) {
    event := &models.AuditEvent{
        EventType:  "TENANT_MEMBER_ADDED",
        Severity:   string(models.AuditSeverityInfo),
        ActorID:    actorID,
        ActorType:  "user",
        TargetID:   &tenantID,
        TargetType: stringPtr("tenant"),
        Action:     "add_member",
        Resource:   "tenant",
        Status:     "success",
        IPAddress:  ipAddress,
        Metadata: map[string]interface{}{
            "member_id": memberID,
        },
    }
    
    a.LogEvent(ctx, event)
}

func (a *AuditLogger) LogPermissionDenied(ctx context.Context, userID, resource, ipAddress string) {
    event := &models.AuditEvent{
        EventType:  "PERMISSION_DENIED",
        Severity:   string(models.AuditSeverityCritical),
        ActorID:    userID,
        ActorType:  "user",
        Action:     "access",
        Resource:   resource,
        Status:     "failure",
        IPAddress:  ipAddress,
    }
    
    a.LogEvent(ctx, event)
}
```

### Middleware Integration

```go
// internal/api/middleware/audit.go

func AuditMiddleware(auditLogger *audit.AuditLogger) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Capture request info
        startTime := time.Now()
        path := c.Request.URL.Path
        method := c.Request.Method
        ipAddress := c.ClientIP()
        userAgent := c.Request.UserAgent()
        
        // Get user ID from context (if authenticated)
        userID, _ := c.Get("user_id")
        
        c.Next()
        
        // Log certain operations
        if shouldAudit(path, method) {
            event := &models.AuditEvent{
                EventType:  getEventType(path, method),
                Severity:   string(models.AuditSeverityInfo),
                ActorID:    fmt.Sprintf("%v", userID),
                ActorType:  "user",
                Action:     method,
                Resource:   path,
                Status:     getStatus(c.Writer.Status()),
                IPAddress:  ipAddress,
                UserAgent:  &userAgent,
                Metadata: map[string]interface{}{
                    "duration_ms": time.Since(startTime).Milliseconds(),
                    "status_code": c.Writer.Status(),
                },
            }
            
            auditLogger.LogEvent(c.Request.Context(), event)
        }
    }
}

func shouldAudit(path, method string) bool {
    // Audit only important operations
    auditPaths := []string{
        "/api/admin/",
        "/api/tenants/",
        "/api/users/",
        "/auth/",
    }
    
    for _, prefix := range auditPaths {
        if strings.HasPrefix(path, prefix) {
            return true
        }
    }
    
    return false
}
```

### Query API

```go
// internal/api/handlers/audit.go

type AuditHandler struct {
    db     storage.Database
    logger *logrus.Logger
}

// Get audit events with filters
// GET /api/admin/audit
func (h *AuditHandler) GetAuditEvents(c *gin.Context) {
    var query struct {
        EventType  string    `form:"event_type"`
        ActorID    string    `form:"actor_id"`
        Resource   string    `form:"resource"`
        Severity   string    `form:"severity"`
        FromDate   time.Time `form:"from_date" time_format:"2006-01-02"`
        ToDate     time.Time `form:"to_date" time_format:"2006-01-02"`
        Limit      int       `form:"limit" binding:"max=1000"`
        Offset     int       `form:"offset"`
    }
    
    if err := c.ShouldBindQuery(&query); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query"})
        return
    }
    
    // Default limit
    if query.Limit == 0 {
        query.Limit = 100
    }
    
    // Build filters
    filters := storage.AuditFilters{
        EventType: query.EventType,
        ActorID:   query.ActorID,
        Resource:  query.Resource,
        Severity:  query.Severity,
        FromDate:  query.FromDate,
        ToDate:    query.ToDate,
        Limit:     query.Limit,
        Offset:    query.Offset,
    }
    
    events, total, err := h.db.GetAuditEvents(c.Request.Context(), filters)
    if err != nil {
        h.logger.WithError(err).Error("Failed to get audit events")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve audit events"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "events": events,
        "total":  total,
        "limit":  query.Limit,
        "offset": query.Offset,
    })
}

// Export audit events to CSV
// GET /api/admin/audit/export
func (h *AuditHandler) ExportAudit(c *gin.Context) {
    // Similar to GetAuditEvents but export as CSV
    // ...
}
```

### Admin UI

**Admin Panel → Audit Log tab:**

```html
<div id="audit-tab" class="tab-pane">
  <h3>Audit Log</h3>
  
  <!-- Filters -->
  <div class="audit-filters">
    <select id="audit-event-type">
      <option value="">All Events</option>
      <option value="LOGIN_SUCCESS">Login Success</option>
      <option value="LOGIN_FAILED">Login Failed</option>
      <option value="API_KEY_CREATED">API Key Created</option>
      <!-- ... -->
    </select>
    
    <select id="audit-severity">
      <option value="">All Severities</option>
      <option value="info">Info</option>
      <option value="warning">Warning</option>
      <option value="critical">Critical</option>
    </select>
    
    <input type="date" id="audit-from-date" />
    <input type="date" id="audit-to-date" />
    
    <button onclick="loadAuditLog()">Filter</button>
    <button onclick="exportAuditLog()">Export CSV</button>
  </div>
  
  <!-- Audit Events Table -->
  <table class="audit-table">
    <thead>
      <tr>
        <th>Timestamp</th>
        <th>Event</th>
        <th>Severity</th>
        <th>Actor</th>
        <th>Action</th>
        <th>Resource</th>
        <th>Status</th>
        <th>IP Address</th>
      </tr>
    </thead>
    <tbody id="audit-events-list">
      <!-- Populated by JS -->
    </tbody>
  </table>
</div>
```

## Требования

### Функциональные

1. ✅ Structured audit events model
2. ✅ Database storage для audit events
3. ✅ Audit logger service с convenience methods
4. ✅ Auto-logging для authentication events
5. ✅ Auto-logging для API key operations
6. ✅ Auto-logging для tenant operations
7. ✅ Query API с filters
8. ✅ Export to CSV
9. ✅ Admin UI для viewing audit log
10. ✅ Retention policy (auto-cleanup old events)

### Нефункциональные

1. **Performance**
   - Logging не блокирует requests (async)
   - Indexes для быстрых queries

2. **Storage**
   - Retention policy: 365 days default
   - Auto-cleanup старых events

3. **Compliance**
   - Immutable audit trail (no updates/deletes except retention)
   - Tamper-evident (optional: cryptographic signatures)

## Acceptance Criteria

- [ ] Audit events сохраняются в БД
- [ ] Authentication events логируются автоматически
- [ ] API key operations логируются
- [ ] Tenant operations логируются
- [ ] Query API возвращает filtered events
- [ ] Export to CSV работает
- [ ] Admin UI отображает audit log
- [ ] Filters (event type, severity, date range) работают
- [ ] Retention policy auto-cleanup старых events
- [ ] Performance: logging не замедляет requests
- [ ] Unit tests для audit logger
- [ ] Integration tests для audit flow

## Риски и зависимости

### Риски

1. **Storage growth** - large audit tables
   - Mitigation: Retention policy, partitioning

2. **Performance impact** - logging overhead
   - Mitigation: Async logging, batch inserts

### Зависимости

1. Existing models (User, Tenant, APIKey)
2. Database storage interface

## Связанные задачи

- **OIDC-01**: Keycloak SSO - логирование OIDC events
- **LDAP-01**: LDAP Integration - логирование LDAP events
- **OPS-01**: Backup & Restore - логирование backup events

## Примечания

- Immutable audit trail (no UPDATE/DELETE except retention)
- Optional: Export to SIEM systems (future)
- Optional: Real-time alerts (future)

---

**Статус:** 📋 Planned for v1.9.0  
**Последнее обновление:** 2025-10-11



// Package models defines data models for audit events
// Version: 1.11.4+ (Enterprise Suite - Enhanced Audit Logging)
package models

import "time"

// AuditEvent представляет событие аудита для security и compliance
type AuditEvent struct {
	ID        string `json:"id" db:"id"`
	EventType string `json:"event_type" db:"event_type"` // LOGIN_SUCCESS, API_KEY_CREATED, etc.
	Severity  string `json:"severity" db:"severity"`     // info, warning, critical

	// Actor (кто выполнил действие)
	ActorID   string `json:"actor_id" db:"actor_id"`     // User ID
	ActorType string `json:"actor_type" db:"actor_type"` // user, system, api_key

	// Target (что было затронуто)
	TargetID   *string `json:"target_id,omitempty" db:"target_id"`
	TargetType *string `json:"target_type,omitempty" db:"target_type"` // user, tenant, api_key

	// Context
	Action   string         `json:"action" db:"action"`     // login, create, delete, update
	Resource string         `json:"resource" db:"resource"` // user, tenant, api_key, backup
	Status   string         `json:"status" db:"status"`     // success, failure
	ErrorMsg *string        `json:"error_msg,omitempty" db:"error_msg"`
	Metadata map[string]any `json:"metadata,omitempty" db:"metadata"` // JSON object

	// Request info
	IPAddress string  `json:"ip_address" db:"ip_address"`
	UserAgent *string `json:"user_agent,omitempty" db:"user_agent"`

	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

// AuditEventSeverity уровни серьезности audit events
type AuditEventSeverity string

const (
	AuditSeverityInfo     AuditEventSeverity = "info"
	AuditSeverityWarning  AuditEventSeverity = "warning"
	AuditSeverityCritical AuditEventSeverity = "critical"
)

// AuditEventType типы audit events
const (
	// Authentication Events
	EventLoginSuccess    = "LOGIN_SUCCESS"
	EventLoginFailed     = "LOGIN_FAILED"
	EventLogout          = "LOGOUT"
	EventPasswordChanged = "PASSWORD_CHANGED"
	EventSessionExpired  = "SESSION_EXPIRED"
	EventOIDCLogin       = "OIDC_LOGIN"
	EventLDAPLogin       = "LDAP_LOGIN"

	// Authorization Events
	EventPermissionDenied = "PERMISSION_DENIED"
	EventRoleChanged      = "ROLE_CHANGED"
	EventAccessGranted    = "ACCESS_GRANTED"

	// API Key Events
	EventAPIKeyCreated       = "API_KEY_CREATED"
	EventAPIKeyDeleted       = "API_KEY_DELETED"
	EventAPIKeyUpdated       = "API_KEY_UPDATED"
	EventAPIKeyRevoked       = "API_KEY_REVOKED"
	EventAPIKeyUsageExceeded = "API_KEY_USAGE_EXCEEDED"

	// Tenant Events
	EventTenantCreated       = "TENANT_CREATED"
	EventTenantDeleted       = "TENANT_DELETED"
	EventTenantUpdated       = "TENANT_UPDATED"
	EventTenantMemberAdded   = "TENANT_MEMBER_ADDED"
	EventTenantMemberRemoved = "TENANT_MEMBER_REMOVED"
	EventTenantRoleChanged   = "TENANT_ROLE_CHANGED"

	// User Events
	EventUserCreated = "USER_CREATED"
	EventUserDeleted = "USER_DELETED"
	EventUserUpdated = "USER_UPDATED"

	// System Events
	EventConfigChanged  = "CONFIG_CHANGED"
	EventBackupCreated  = "BACKUP_CREATED"
	EventBackupRestored = "BACKUP_RESTORED"
	EventModelLoaded    = "MODEL_LOADED"
	EventModelUnloaded  = "MODEL_UNLOADED"
)

// AuditActorType типы actors
const (
	ActorTypeUser   = "user"
	ActorTypeSystem = "system"
	ActorTypeAPIKey = "api_key"
)

// AuditTargetType типы targets
const (
	TargetTypeUser   = "user"
	TargetTypeTenant = "tenant"
	TargetTypeAPIKey = "api_key"
	TargetTypeBackup = "backup"
	TargetTypeModel  = "model"
	TargetTypeFile   = "file"
)

// AuditEventStatus статусы операций
const (
	AuditStatusSuccess = "success"
	AuditStatusFailure = "failure"
)

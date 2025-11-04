// Package audit provides comprehensive audit logging for security events and compliance
// Version: 1.11.4+ (Enterprise Suite - Enhanced Audit Logging)
package audit

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/storage"
	"aigateway/internal/utils"
)

// AuditLogger handles audit event logging to database and standard logger
type AuditLogger struct {
	db     storage.Database
	logger *logrus.Logger
}

// NewAuditLogger creates a new audit logger instance
func NewAuditLogger(db storage.Database, logger *logrus.Logger) *AuditLogger {
	return &AuditLogger{
		db:     db,
		logger: logger,
	}
}

// LogEvent logs an audit event to database and standard logger
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
	if event.ActorType == "" {
		event.ActorType = models.ActorTypeUser
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
		"ip_address":  event.IPAddress,
	})

	if event.TargetID != nil {
		logEntry = logEntry.WithField("target_id", *event.TargetID)
	}

	switch event.Severity {
	case string(models.AuditSeverityInfo):
		logEntry.Info("Audit event")
	case string(models.AuditSeverityWarning):
		logEntry.Warn("Audit event")
	case string(models.AuditSeverityCritical):
		logEntry.Error("Audit event")
	default:
		logEntry.Info("Audit event")
	}

	return nil
}

// ========================================
// Convenience Methods - Authentication
// ========================================

// LogLogin logs a login attempt (success or failure)
func (a *AuditLogger) LogLogin(ctx context.Context, userID, ipAddress, userAgent string, success bool, errorMsg string) error {
	event := &models.AuditEvent{
		EventType:  models.EventLoginSuccess,
		Severity:   string(models.AuditSeverityInfo),
		ActorID:    userID,
		ActorType:  models.ActorTypeUser,
		Action:     "login",
		Resource:   "authentication",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
		UserAgent:  &userAgent,
	}

	if !success {
		event.EventType = models.EventLoginFailed
		event.Status = models.AuditStatusFailure
		event.Severity = string(models.AuditSeverityWarning)
		if errorMsg != "" {
			event.ErrorMsg = &errorMsg
		}
	}

	return a.LogEvent(ctx, event)
}

// LogOIDCLogin logs an OIDC/Keycloak login
func (a *AuditLogger) LogOIDCLogin(ctx context.Context, userID, issuer, ipAddress string, success bool, errorMsg string) error {
	metadata := map[string]interface{}{
		"issuer": issuer,
	}

	event := &models.AuditEvent{
		EventType:  models.EventOIDCLogin,
		Severity:   string(models.AuditSeverityInfo),
		ActorID:    userID,
		ActorType:  models.ActorTypeUser,
		Action:     "oidc_login",
		Resource:   "authentication",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
		Metadata:   metadata,
	}

	if !success {
		event.Status = models.AuditStatusFailure
		event.Severity = string(models.AuditSeverityWarning)
		if errorMsg != "" {
			event.ErrorMsg = &errorMsg
		}
	}

	return a.LogEvent(ctx, event)
}

// LogLDAPLogin logs an LDAP/AD login
func (a *AuditLogger) LogLDAPLogin(ctx context.Context, userID, ldapDN, ipAddress string, success bool, errorMsg string) error {
	metadata := map[string]interface{}{
		"ldap_dn": ldapDN,
	}

	event := &models.AuditEvent{
		EventType:  models.EventLDAPLogin,
		Severity:   string(models.AuditSeverityInfo),
		ActorID:    userID,
		ActorType:  models.ActorTypeUser,
		Action:     "ldap_login",
		Resource:   "authentication",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
		Metadata:   metadata,
	}

	if !success {
		event.Status = models.AuditStatusFailure
		event.Severity = string(models.AuditSeverityWarning)
		if errorMsg != "" {
			event.ErrorMsg = &errorMsg
		}
	}

	return a.LogEvent(ctx, event)
}

// LogLogout logs a user logout
func (a *AuditLogger) LogLogout(ctx context.Context, userID, ipAddress string) error {
	event := &models.AuditEvent{
		EventType:  models.EventLogout,
		Severity:   string(models.AuditSeverityInfo),
		ActorID:    userID,
		ActorType:  models.ActorTypeUser,
		Action:     "logout",
		Resource:   "authentication",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
	}

	return a.LogEvent(ctx, event)
}

// LogPasswordChanged logs a password change
func (a *AuditLogger) LogPasswordChanged(ctx context.Context, userID, ipAddress string) error {
	event := &models.AuditEvent{
		EventType:  models.EventPasswordChanged,
		Severity:   string(models.AuditSeverityInfo),
		ActorID:    userID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &userID,
		TargetType: utils.Ptr(models.TargetTypeUser),
		Action:     "change_password",
		Resource:   "user",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
	}

	return a.LogEvent(ctx, event)
}

// ========================================
// Convenience Methods - Authorization
// ========================================

// LogPermissionDenied logs an access denied event
func (a *AuditLogger) LogPermissionDenied(ctx context.Context, userID, resource, action, ipAddress string) error {
	metadata := map[string]interface{}{
		"attempted_action": action,
	}

	event := &models.AuditEvent{
		EventType:  models.EventPermissionDenied,
		Severity:   string(models.AuditSeverityCritical),
		ActorID:    userID,
		ActorType:  models.ActorTypeUser,
		Action:     action,
		Resource:   resource,
		Status:     models.AuditStatusFailure,
		IPAddress:  ipAddress,
		Metadata:   metadata,
	}

	return a.LogEvent(ctx, event)
}

// LogRoleChanged logs a user role change
func (a *AuditLogger) LogRoleChanged(ctx context.Context, actorID, targetUserID, oldRole, newRole, ipAddress string) error {
	metadata := map[string]interface{}{
		"old_role": oldRole,
		"new_role": newRole,
	}

	event := &models.AuditEvent{
		EventType:  models.EventRoleChanged,
		Severity:   string(models.AuditSeverityWarning),
		ActorID:    actorID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &targetUserID,
		TargetType: utils.Ptr(models.TargetTypeUser),
		Action:     "change_role",
		Resource:   "user",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
		Metadata:   metadata,
	}

	return a.LogEvent(ctx, event)
}

// ========================================
// Convenience Methods - API Keys
// ========================================

// LogAPIKeyCreated logs API key creation
func (a *AuditLogger) LogAPIKeyCreated(ctx context.Context, actorID, keyID, keyName, ipAddress string) error {
	metadata := map[string]interface{}{
		"key_name": keyName,
	}

	event := &models.AuditEvent{
		EventType:  models.EventAPIKeyCreated,
		Severity:   string(models.AuditSeverityInfo),
		ActorID:    actorID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &keyID,
		TargetType: utils.Ptr(models.TargetTypeAPIKey),
		Action:     "create",
		Resource:   "api_key",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
		Metadata:   metadata,
	}

	return a.LogEvent(ctx, event)
}

// LogAPIKeyDeleted logs API key deletion
func (a *AuditLogger) LogAPIKeyDeleted(ctx context.Context, actorID, keyID, ipAddress string) error {
	event := &models.AuditEvent{
		EventType:  models.EventAPIKeyDeleted,
		Severity:   string(models.AuditSeverityWarning),
		ActorID:    actorID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &keyID,
		TargetType: utils.Ptr(models.TargetTypeAPIKey),
		Action:     "delete",
		Resource:   "api_key",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
	}

	return a.LogEvent(ctx, event)
}

// LogAPIKeyRevoked logs API key revocation
func (a *AuditLogger) LogAPIKeyRevoked(ctx context.Context, actorID, keyID, reason, ipAddress string) error {
	metadata := map[string]interface{}{
		"reason": reason,
	}

	event := &models.AuditEvent{
		EventType:  models.EventAPIKeyRevoked,
		Severity:   string(models.AuditSeverityWarning),
		ActorID:    actorID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &keyID,
		TargetType: utils.Ptr(models.TargetTypeAPIKey),
		Action:     "revoke",
		Resource:   "api_key",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
		Metadata:   metadata,
	}

	return a.LogEvent(ctx, event)
}

// ========================================
// Convenience Methods - Tenants
// ========================================

// LogTenantCreated logs tenant creation
func (a *AuditLogger) LogTenantCreated(ctx context.Context, actorID, tenantID, tenantName, ipAddress string) error {
	metadata := map[string]interface{}{
		"tenant_name": tenantName,
	}

	event := &models.AuditEvent{
		EventType:  models.EventTenantCreated,
		Severity:   string(models.AuditSeverityInfo),
		ActorID:    actorID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &tenantID,
		TargetType: utils.Ptr(models.TargetTypeTenant),
		Action:     "create",
		Resource:   "tenant",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
		Metadata:   metadata,
	}

	return a.LogEvent(ctx, event)
}

// LogTenantDeleted logs tenant deletion
func (a *AuditLogger) LogTenantDeleted(ctx context.Context, actorID, tenantID, ipAddress string) error {
	event := &models.AuditEvent{
		EventType:  models.EventTenantDeleted,
		Severity:   string(models.AuditSeverityCritical),
		ActorID:    actorID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &tenantID,
		TargetType: utils.Ptr(models.TargetTypeTenant),
		Action:     "delete",
		Resource:   "tenant",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
	}

	return a.LogEvent(ctx, event)
}

// LogTenantUpdated logs tenant information update
func (a *AuditLogger) LogTenantUpdated(ctx context.Context, actorID, tenantID, changes, ipAddress string) error {
	metadata := map[string]interface{}{
		"changes": changes,
	}

	event := &models.AuditEvent{
		EventType:  models.EventTenantUpdated,
		Severity:   string(models.AuditSeverityInfo),
		ActorID:    actorID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &tenantID,
		TargetType: utils.Ptr(models.TargetTypeTenant),
		Action:     "update",
		Resource:   "tenant",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
		Metadata:   metadata,
	}

	return a.LogEvent(ctx, event)
}

// LogTenantMemberAdded logs tenant member addition
func (a *AuditLogger) LogTenantMemberAdded(ctx context.Context, actorID, tenantID, memberID, role, ipAddress string) error {
	metadata := map[string]interface{}{
		"member_id": memberID,
		"role":      role,
	}

	event := &models.AuditEvent{
		EventType:  models.EventTenantMemberAdded,
		Severity:   string(models.AuditSeverityInfo),
		ActorID:    actorID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &tenantID,
		TargetType: utils.Ptr(models.TargetTypeTenant),
		Action:     "add_member",
		Resource:   "tenant",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
		Metadata:   metadata,
	}

	return a.LogEvent(ctx, event)
}

// LogTenantMemberRemoved logs tenant member removal
func (a *AuditLogger) LogTenantMemberRemoved(ctx context.Context, actorID, tenantID, memberID, ipAddress string) error {
	metadata := map[string]interface{}{
		"member_id": memberID,
	}

	event := &models.AuditEvent{
		EventType:  models.EventTenantMemberRemoved,
		Severity:   string(models.AuditSeverityWarning),
		ActorID:    actorID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &tenantID,
		TargetType: utils.Ptr(models.TargetTypeTenant),
		Action:     "remove_member",
		Resource:   "tenant",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
		Metadata:   metadata,
	}

	return a.LogEvent(ctx, event)
}

// LogTenantRoleChanged logs tenant member role change
func (a *AuditLogger) LogTenantRoleChanged(ctx context.Context, actorID, tenantID, memberID, oldRole, newRole, ipAddress string) error {
	metadata := map[string]interface{}{
		"member_id": memberID,
		"old_role":  oldRole,
		"new_role":  newRole,
	}

	event := &models.AuditEvent{
		EventType:  models.EventTenantRoleChanged,
		Severity:   string(models.AuditSeverityWarning),
		ActorID:    actorID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &tenantID,
		TargetType: utils.Ptr(models.TargetTypeTenant),
		Action:     "change_role",
		Resource:   "tenant",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
		Metadata:   metadata,
	}

	return a.LogEvent(ctx, event)
}

// ========================================
// Convenience Methods - Users
// ========================================

// LogUserCreated logs user creation
func (a *AuditLogger) LogUserCreated(ctx context.Context, actorID, targetUserID, username, ipAddress string) error {
	metadata := map[string]interface{}{
		"username": username,
	}

	event := &models.AuditEvent{
		EventType:  models.EventUserCreated,
		Severity:   string(models.AuditSeverityInfo),
		ActorID:    actorID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &targetUserID,
		TargetType: utils.Ptr(models.TargetTypeUser),
		Action:     "create",
		Resource:   "user",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
		Metadata:   metadata,
	}

	return a.LogEvent(ctx, event)
}

// LogUserDeleted logs user deletion
func (a *AuditLogger) LogUserDeleted(ctx context.Context, actorID, targetUserID, ipAddress string) error {
	event := &models.AuditEvent{
		EventType:  models.EventUserDeleted,
		Severity:   string(models.AuditSeverityCritical),
		ActorID:    actorID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &targetUserID,
		TargetType: utils.Ptr(models.TargetTypeUser),
		Action:     "delete",
		Resource:   "user",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
	}

	return a.LogEvent(ctx, event)
}

// LogUserUpdated logs user information update
func (a *AuditLogger) LogUserUpdated(ctx context.Context, actorID, targetUserID, changes, ipAddress string) error {
	metadata := map[string]interface{}{
		"changes": changes,
	}

	event := &models.AuditEvent{
		EventType:  models.EventUserUpdated,
		Severity:   string(models.AuditSeverityInfo),
		ActorID:    actorID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &targetUserID,
		TargetType: utils.Ptr(models.TargetTypeUser),
		Action:     "update",
		Resource:   "user",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
		Metadata:   metadata,
	}

	return a.LogEvent(ctx, event)
}

// ========================================
// Convenience Methods - System
// ========================================

// LogBackupCreated logs backup creation
func (a *AuditLogger) LogBackupCreated(ctx context.Context, actorID, backupID, ipAddress string) error {
	event := &models.AuditEvent{
		EventType:  models.EventBackupCreated,
		Severity:   string(models.AuditSeverityInfo),
		ActorID:    actorID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &backupID,
		TargetType: utils.Ptr(models.TargetTypeBackup),
		Action:     "create",
		Resource:   "backup",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
	}

	return a.LogEvent(ctx, event)
}

// LogBackupRestored logs backup restoration
func (a *AuditLogger) LogBackupRestored(ctx context.Context, actorID, backupID, ipAddress string) error {
	event := &models.AuditEvent{
		EventType:  models.EventBackupRestored,
		Severity:   string(models.AuditSeverityCritical),
		ActorID:    actorID,
		ActorType:  models.ActorTypeUser,
		TargetID:   &backupID,
		TargetType: utils.Ptr(models.TargetTypeBackup),
		Action:     "restore",
		Resource:   "backup",
		Status:     models.AuditStatusSuccess,
		IPAddress:  ipAddress,
	}

	return a.LogEvent(ctx, event)
}

// ========================================
// Utility Functions
// ========================================

// stringPtr replaced with utils.Ptr[T] (Go 1.25 generics)



// Package ui provides HTMX UI handlers for RBAC management
package ui

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/storage"
	"aigateway/internal/web/templates"
)

// RBACUIHandler handles RBAC management UI endpoints
type RBACUIHandler struct {
	db       storage.Database
	renderer *templates.Renderer
	logger   *logrus.Logger
}

// NewRBACUIHandler creates a new RBAC UI handler
func NewRBACUIHandler(db storage.Database, renderer *templates.Renderer, logger *logrus.Logger) *RBACUIHandler {
	return &RBACUIHandler{
		db:       db,
		renderer: renderer,
		logger:   logger,
	}
}

// GetRolesList returns HTML list of roles for admin panel
// GET /api/ui/rbac/roles
func (h *RBACUIHandler) GetRolesList(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Get global roles (tenantID = nil)
	var tenantID *string = nil
	roles, err := h.db.ListRoles(ctx, tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list roles")
		c.Header("Content-Type", "text/html")
		c.String(http.StatusInternalServerError, `
<tr>
	<td colspan="6" class="table-empty error">
		Failed to load roles
	</td>
</tr>`)
		return
	}
	
	// Check if no roles
	if len(roles) == 0 {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `
<tr>
	<td colspan="6" class="table-empty">
		No roles found. Create your first role!
	</td>
</tr>`)
		return
	}
	
	// Render rows
	c.Header("Content-Type", "text/html")
	html := ""
	
	for _, role := range roles {
		// Get permissions count for this role
		permissions, err := h.db.GetRolePermissions(ctx, role.ID)
		if err != nil {
			h.logger.WithError(err).Warn("Failed to get role permissions")
			permissions = []*models.RBACPermission{}
		}
		
		// Get users count for this role
		users, err := h.db.GetRoleUsers(ctx, role.ID)
		if err != nil {
			h.logger.WithError(err).Warn("Failed to get role users")
			users = []*models.User{}
		}
		
		scopeDisplay := "Global"
		if role.TenantID != nil {
			scopeDisplay = "Tenant"
		}
		
		html += fmt.Sprintf(`
<tr class="role-row" data-role-id="%s">
	<td>
		<div class="role-info">
			<strong>%s</strong>
			<small class="text-muted">%s</small>
		</div>
	</td>
	<td><span class="badge">%s</span></td>
	<td><span class="badge">%d permissions</span></td>
	<td><span class="badge">%d users</span></td>
	<td class="text-muted" title="%s">%s</td>
	<td>
		<div class="action-buttons">
			<button class="btn btn-sm btn-secondary" 
					hx-get="/api/ui/rbac/roles/%s/edit"
					hx-target="#modal-content"
					hx-swap="innerHTML"
					onclick="showModal()">
				Edit
			</button>
			<button class="btn btn-sm btn-secondary"
					hx-get="/api/ui/rbac/roles/%s/permissions"
					hx-target="#modal-content"
					hx-swap="innerHTML"
					onclick="showModal()">
				Permissions
			</button>
		</div>
	</td>
</tr>`,
			role.ID,
			template.HTMLEscapeString(role.Name),
			template.HTMLEscapeString(role.Description),
			scopeDisplay,
			len(permissions),
			len(users),
			role.CreatedAt.Format(time.RFC3339),
			formatTimeAgo(role.CreatedAt),
			role.ID,
			role.ID,
		)
	}
	
	c.String(http.StatusOK, html)
}

// GetCreateRoleForm returns HTML form for creating a new role
// GET /api/ui/rbac/roles/create-form
func (h *RBACUIHandler) GetCreateRoleForm(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, `
<div class="modal-header">
	<h2>Create New Role</h2>
	<button class="btn-close" onclick="closeModal()">×</button>
</div>
<form hx-post="/api/admin/rbac/roles" 
	  hx-target="#roles-table" 
	  hx-swap="innerHTML"
	  hx-on::after-request="if(event.detail.successful) closeModal()">
	<div class="modal-body">
		<div class="form-group">
			<label for="name" class="required">Role Name</label>
			<input type="text" 
				   id="name" 
				   name="name" 
				   class="form-control" 
				   required
				   minlength="3"
				   pattern="^[a-z][a-z0-9_-]*$"
				   placeholder="e.g. content_editor">
			<small class="form-text">Lowercase, letters, numbers, _ and - only. E.g. admin, developer, content_editor</small>
		</div>
		
		<div class="form-group">
			<label for="description" class="required">Description</label>
			<textarea id="description" 
					  name="description" 
					  class="form-control" 
					  required
					  rows="3"
					  placeholder="What can users with this role do?"></textarea>
			<small class="form-text">Clear description of what this role allows</small>
		</div>
		
		<div class="form-group">
			<label class="checkbox-label">
				<input type="checkbox" name="is_system" value="true">
				<span>System Role</span>
			</label>
			<small class="form-text">System roles cannot be deleted and are predefined</small>
		</div>
	</div>
	
	<div class="modal-footer">
		<button type="button" class="btn btn-secondary" onclick="closeModal()">Cancel</button>
		<button type="submit" class="btn btn-primary">
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
				<path d="M12 5v14M5 12h14" stroke-width="2"/>
			</svg>
			Create Role
		</button>
	</div>
</form>`)
}

// GetEditRoleForm returns HTML form for editing a role
// GET /api/ui/rbac/roles/:id/edit
func (h *RBACUIHandler) GetEditRoleForm(c *gin.Context) {
	ctx := c.Request.Context()
	roleID := c.Param("id")
	
	// Get role
	role, err := h.db.GetRole(ctx, roleID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get role")
		c.Header("Content-Type", "text/html")
		c.String(http.StatusInternalServerError, `
<div class="alert alert-error">
	Failed to load role
</div>`)
		return
	}
	
	if role == nil {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusNotFound, `
<div class="alert alert-error">
	Role not found
</div>`)
		return
	}
	
	isSystemChecked := ""
	if role.Type == "system" {
		isSystemChecked = "checked"
	}
	
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, fmt.Sprintf(`
<div class="modal-header">
	<h2>Edit Role: %s</h2>
	<button class="btn-close" onclick="closeModal()">×</button>
</div>
<form hx-put="/api/admin/rbac/roles/%s" 
	  hx-target="#roles-table" 
	  hx-swap="innerHTML"
	  hx-on::after-request="if(event.detail.successful) closeModal()">
	<div class="modal-body">
		<div class="form-group">
			<label for="name">Role Name</label>
			<input type="text" 
				   id="name" 
				   name="name" 
				   class="form-control" 
				   value="%s"
				   readonly
				   disabled>
			<small class="form-text">Role name cannot be changed</small>
		</div>
		
		<div class="form-group">
			<label for="description" class="required">Description</label>
			<textarea id="description" 
					  name="description" 
					  class="form-control" 
					  required
					  rows="3">%s</textarea>
		</div>
		
		<div class="form-group">
			<label class="checkbox-label">
				<input type="checkbox" name="is_system" value="true" %s>
				<span>System Role</span>
			</label>
			<small class="form-text">System roles cannot be deleted</small>
		</div>
	</div>
	
	<div class="modal-footer">
		<button type="button" class="btn btn-secondary" onclick="closeModal()">Cancel</button>
		<button type="submit" class="btn btn-primary">
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
				<path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" stroke-width="2"/>
				<path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" stroke-width="2"/>
			</svg>
			Update Role
		</button>
	</div>
</form>`,
		template.HTMLEscapeString(role.Name),
		role.ID,
		template.HTMLEscapeString(role.Name),
		template.HTMLEscapeString(role.Description),
		isSystemChecked))
}

// GetRolePermissionsForm returns HTML form for managing role permissions
// GET /api/ui/rbac/roles/:id/permissions
func (h *RBACUIHandler) GetRolePermissionsForm(c *gin.Context) {
	ctx := c.Request.Context()
	roleID := c.Param("id")
	
	// Get role
	role, err := h.db.GetRole(ctx, roleID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get role")
		c.Header("Content-Type", "text/html")
		c.String(http.StatusInternalServerError, `
<div class="alert alert-error">
	Failed to load role
</div>`)
		return
	}
	
	if role == nil {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusNotFound, `
<div class="alert alert-error">
	Role not found
</div>`)
		return
	}
	
	// Get current permissions
	currentPermissions, err := h.db.GetRolePermissions(ctx, roleID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get role permissions")
		currentPermissions = []*models.RBACPermission{}
	}
	
	// Build map of assigned permissions
	assignedPermissions := make(map[string]bool)
	for _, perm := range currentPermissions {
		assignedPermissions[perm.ID] = true
	}
	
	// Get all available permissions
	allPermissions, err := h.db.ListPermissions(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list permissions")
		allPermissions = []*models.RBACPermission{}
	}
	
	// Group permissions by resource (instead of category)
	permsByResource := make(map[string][]*models.RBACPermission)
	for _, perm := range allPermissions {
		resource := perm.Resource
		if resource == "" {
			resource = "Other"
		}
		permsByResource[resource] = append(permsByResource[resource], perm)
	}
	
	// Build permissions HTML grouped by resource
	permsHTML := ""
	for resource, perms := range permsByResource {
		permsHTML += fmt.Sprintf(`
		<div class="permission-category">
			<h4>%s</h4>
			<div class="permissions-grid">`, template.HTMLEscapeString(resource))
		
		for _, perm := range perms {
			checked := ""
			if assignedPermissions[perm.ID] {
				checked = "checked"
			}
			
			permsHTML += fmt.Sprintf(`
				<div class="form-group">
					<label class="checkbox-label">
						<input type="checkbox" name="permissions[]" value="%s" %s>
						<span><strong>%s</strong></span>
					</label>
					<small class="form-text">%s</small>
				</div>`,
				template.HTMLEscapeString(perm.ID),
				checked,
				template.HTMLEscapeString(perm.Name),
				template.HTMLEscapeString(perm.Description))
		}
		
		permsHTML += `
			</div>
		</div>`
	}
	
	if permsHTML == "" {
		permsHTML = `<p class="text-muted">No permissions available</p>`
	}
	
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, fmt.Sprintf(`
<div class="modal-header">
	<h2>Manage Permissions: %s</h2>
	<button class="btn-close" onclick="closeModal()">×</button>
</div>
<form hx-put="/api/admin/rbac/roles/%s/permissions" 
	  hx-target="#roles-table" 
	  hx-swap="innerHTML"
	  hx-on::after-request="if(event.detail.successful) closeModal()">
	<div class="modal-body" style="max-height: 60vh; overflow-y: auto;">
		<p class="form-description">
			Select the permissions to grant to this role. Users with this role will inherit all selected permissions.
		</p>
		
		%s
	</div>
	
	<div class="modal-footer">
		<button type="button" class="btn btn-secondary" onclick="closeModal()">Cancel</button>
		<button type="submit" class="btn btn-primary">
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
				<path d="M20 6L9 17l-5-5" stroke-width="2"/>
			</svg>
			Update Permissions
		</button>
	</div>
</form>

<style>
.permission-category {
	margin-bottom: 24px;
	padding-bottom: 16px;
	border-bottom: 1px solid var(--border-color);
}

.permission-category:last-child {
	border-bottom: none;
}

.permission-category h4 {
	margin-bottom: 16px;
	color: var(--text-primary);
	font-size: 16px;
}

.permissions-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
	gap: 8px;
}
</style>`,
		template.HTMLEscapeString(role.Name),
		role.ID,
		permsHTML))
}

// GetPermissionsList returns HTML list of permissions for admin panel
// GET /api/ui/rbac/permissions
func (h *RBACUIHandler) GetPermissionsList(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Get all permissions
	permissions, err := h.db.ListPermissions(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list permissions")
		c.Header("Content-Type", "text/html")
		c.String(http.StatusInternalServerError, `
<tr>
	<td colspan="4" class="table-empty error">
		Failed to load permissions
	</td>
</tr>`)
		return
	}
	
	// Check if no permissions
	if len(permissions) == 0 {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `
<tr>
	<td colspan="4" class="table-empty">
		No permissions found
	</td>
</tr>`)
		return
	}
	
	// Render rows
	c.Header("Content-Type", "text/html")
	html := ""
	
	for _, perm := range permissions {
		resource := perm.Resource
		if resource == "" {
			resource = "-"
		}
		
		html += fmt.Sprintf(`
<tr>
	<td>
		<div class="role-info">
			<strong>%s</strong>
			<small class="text-muted">%s</small>
		</div>
	</td>
	<td><span class="badge">%s</span></td>
	<td><code class="text-muted">%s</code></td>
	<td class="text-muted" title="%s">%s</td>
</tr>`,
			template.HTMLEscapeString(perm.Name),
			template.HTMLEscapeString(perm.Description),
			template.HTMLEscapeString(resource),
			template.HTMLEscapeString(perm.Resource+":"+perm.Action),
			perm.CreatedAt.Format(time.RFC3339),
			formatTimeAgo(perm.CreatedAt),
		)
	}
	
	c.String(http.StatusOK, html)
}


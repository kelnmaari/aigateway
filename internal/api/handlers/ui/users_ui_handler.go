// Package ui provides HTMX UI handlers for user management pages
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

// UsersUIHandler handles user management UI endpoints
type UsersUIHandler struct {
	db       storage.Database
	renderer *templates.Renderer
	logger   *logrus.Logger
}

// NewUsersUIHandler creates a new users UI handler
func NewUsersUIHandler(db storage.Database, renderer *templates.Renderer, logger *logrus.Logger) *UsersUIHandler {
	return &UsersUIHandler{
		db:       db,
		renderer: renderer,
		logger:   logger,
	}
}

// GetUsersList returns HTML list of users for admin panel
// GET /api/ui/users
func (h *UsersUIHandler) GetUsersList(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Get all users with details (roles, tenants)
	filters := models.UserFilters{
		Limit:  100, // Default limit
		Offset: 0,
	}
	
	// Parse optional filters from query
	if status := c.Query("status"); status != "" {
		s := models.UserStatus(status)
		filters.Status = &s
	}
	if search := c.Query("search"); search != "" {
		filters.Search = search
	}
	
	usersWithDetails, err := h.db.GetUsersWithDetails(ctx, filters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list users")
		c.Header("Content-Type", "text/html")
		c.String(http.StatusInternalServerError, `
<tr>
	<td colspan="9" class="table-empty error">
		Failed to load users
	</td>
</tr>`)
		return
	}
	
	// Check if no users
	if len(usersWithDetails) == 0 {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `
<tr>
	<td colspan="9" class="table-empty">
		No users found
	</td>
</tr>`)
		return
	}
	
	// Render rows
	c.Header("Content-Type", "text/html")
	html := ""
	
	for _, userDetail := range usersWithDetails {
		user := &userDetail.User // Get underlying User
		
		// Format auth provider
		provider := user.AuthProvider
		if provider == "" {
			provider = "local"
		}
		
		// Format RBAC roles
		rolesDisplay := "-"
		if len(userDetail.Roles) > 0 {
			rolesDisplay = fmt.Sprintf(`<span class="badge">%d roles</span>`, len(userDetail.Roles))
		}
		
		// Format tenants count
		tenantsDisplay := "-"
		if len(userDetail.Tenants) > 0 {
			tenantsDisplay = fmt.Sprintf(`<span class="badge">%d tenants</span>`, len(userDetail.Tenants))
		}
		
		// Format admin badge
		adminBadge := ""
		if user.IsAdmin {
			adminBadge = `<span class="badge badge-admin">Admin</span>`
		}
		
		// Format status badge
		statusClass := "badge-active"
		statusText := "Active"
		if user.Status == models.UserStatusInactive {
			statusClass = "badge-inactive"
			statusText = "Inactive"
		} else if user.Status == models.UserStatusSuspended {
			statusClass = "badge-suspended"
			statusText = "Suspended"
		}
		
		html += fmt.Sprintf(`
<tr class="user-row" data-user-id="%s">
	<td>
		<div class="user-info">
			<strong>%s</strong>
		</div>
	</td>
	<td>%s</td>
	<td><span class="badge badge-provider">%s</span></td>
	<td>%s</td>
	<td>%s</td>
	<td>%s</td>
	<td><span class="badge %s">%s</span></td>
	<td class="text-muted" title="%s">%s</td>
	<td>
		<div class="action-buttons">
			<button class="btn btn-sm btn-secondary" 
					hx-get="/api/ui/users/%s/edit"
					hx-target="#modal-content"
					hx-swap="innerHTML"
					onclick="showModal()">
				Edit
			</button>
			<button class="btn btn-sm btn-secondary"
					hx-get="/api/ui/users/%s/roles"
					hx-target="#modal-content"
					hx-swap="innerHTML"
					onclick="showModal()">
				Roles
			</button>
		</div>
	</td>
</tr>`,
			user.ID,
			template.HTMLEscapeString(user.Username),
			template.HTMLEscapeString(user.Email),
			template.HTMLEscapeString(provider),
			rolesDisplay,
			tenantsDisplay,
			adminBadge,
			statusClass, statusText,
			user.CreatedAt.Format(time.RFC3339),
			formatTimeAgo(user.CreatedAt),
			user.ID,
			user.ID,
		)
	}
	
	c.String(http.StatusOK, html)
}

// GetCreateUserForm returns HTML form for creating a new user
// GET /api/ui/users/create-form
func (h *UsersUIHandler) GetCreateUserForm(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, `
<div class="modal-header">
	<h2>Create New User</h2>
	<button class="btn-close" onclick="closeModal()">×</button>
</div>
<form hx-post="/api/admin/users" 
	  hx-target="#users-table" 
	  hx-swap="innerHTML"
	  hx-on::after-request="if(event.detail.successful) closeModal()">
	<div class="modal-body">
		<div class="form-group">
			<label for="username" class="required">Username</label>
			<input type="text" 
				   id="username" 
				   name="username" 
				   class="form-control" 
				   required
				   minlength="3"
				   pattern="^[a-zA-Z0-9_-]+$"
				   placeholder="john_doe">
			<small class="form-text">3+ characters, letters, numbers, _ and - only</small>
		</div>
		
		<div class="form-group">
			<label for="email" class="required">Email</label>
			<input type="email" 
				   id="email" 
				   name="email" 
				   class="form-control" 
				   required
				   placeholder="john@example.com">
		</div>
		
		<div class="form-group">
			<label for="password" class="required">Password</label>
			<input type="password" 
				   id="password" 
				   name="password" 
				   class="form-control" 
				   required
				   minlength="8"
				   placeholder="Minimum 8 characters">
			<small class="form-text">At least 8 characters</small>
		</div>
		
		<div class="form-group">
			<label for="full_name">Full Name</label>
			<input type="text" 
				   id="full_name" 
				   name="full_name" 
				   class="form-control" 
				   placeholder="John Doe">
		</div>
		
		<div class="form-group">
			<label class="checkbox-label">
				<input type="checkbox" name="is_admin" value="true">
				<span>Admin privileges</span>
			</label>
			<small class="form-text">Grant full admin access to this user</small>
		</div>
	</div>
	
	<div class="modal-footer">
		<button type="button" class="btn btn-secondary" onclick="closeModal()">Cancel</button>
		<button type="submit" class="btn btn-primary">
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
				<path d="M12 5v14M5 12h14" stroke-width="2"/>
			</svg>
			Create User
		</button>
	</div>
</form>`)
}

// GetEditUserForm returns HTML form for editing a user
// GET /api/ui/users/:id/edit
func (h *UsersUIHandler) GetEditUserForm(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.Param("id")
	
	// Get user
	user, err := h.db.GetUser(ctx, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user")
		c.Header("Content-Type", "text/html")
		c.String(http.StatusInternalServerError, `
<div class="alert alert-error">
	Failed to load user
</div>`)
		return
	}
	
	if user == nil {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusNotFound, `
<div class="alert alert-error">
	User not found
</div>`)
		return
	}
	
	adminChecked := ""
	if user.IsAdmin {
		adminChecked = "checked"
	}
	
	statusActive := ""
	statusInactive := ""
	statusSuspended := ""
	switch user.Status {
	case models.UserStatusActive:
		statusActive = "selected"
	case models.UserStatusInactive:
		statusInactive = "selected"
	case models.UserStatusSuspended:
		statusSuspended = "selected"
	}
	
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, fmt.Sprintf(`
<div class="modal-header">
	<h2>Edit User: %s</h2>
	<button class="btn-close" onclick="closeModal()">×</button>
</div>
<form hx-put="/api/admin/users/%s" 
	  hx-target="#users-table" 
	  hx-swap="innerHTML"
	  hx-on::after-request="if(event.detail.successful) closeModal()">
	<div class="modal-body">
		<div class="form-group">
			<label for="username">Username</label>
			<input type="text" 
				   id="username" 
				   name="username" 
				   class="form-control" 
				   value="%s"
				   readonly
				   disabled>
			<small class="form-text">Username cannot be changed</small>
		</div>
		
		<div class="form-group">
			<label for="email" class="required">Email</label>
			<input type="email" 
				   id="email" 
				   name="email" 
				   class="form-control" 
				   required
				   value="%s">
		</div>
		
		<div class="form-group">
			<label for="full_name">Full Name</label>
			<input type="text" 
				   id="full_name" 
				   name="full_name" 
				   class="form-control" 
				   value="%s">
		</div>
		
		<div class="form-group">
			<label for="status">Status</label>
			<select id="status" name="status" class="form-control">
				<option value="active" %s>Active</option>
				<option value="inactive" %s>Inactive</option>
				<option value="suspended" %s>Suspended</option>
			</select>
		</div>
		
		<div class="form-group">
			<label class="checkbox-label">
				<input type="checkbox" name="is_admin" value="true" %s>
				<span>Admin privileges</span>
			</label>
		</div>
		
		<div class="form-group">
			<label for="new_password">New Password (optional)</label>
			<input type="password" 
				   id="new_password" 
				   name="new_password" 
				   class="form-control" 
				   minlength="8"
				   placeholder="Leave empty to keep current password">
			<small class="form-text">At least 8 characters</small>
		</div>
	</div>
	
	<div class="modal-footer">
		<button type="button" class="btn btn-secondary" onclick="closeModal()">Cancel</button>
		<button type="submit" class="btn btn-primary">
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
				<path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" stroke-width="2"/>
				<path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" stroke-width="2"/>
			</svg>
			Update User
		</button>
	</div>
</form>`,
		template.HTMLEscapeString(user.Username),
		user.ID,
		template.HTMLEscapeString(user.Username),
		template.HTMLEscapeString(user.Email),
		template.HTMLEscapeString(user.FullName),
		statusActive, statusInactive, statusSuspended,
		adminChecked))
}

// GetUserRolesForm returns HTML form for managing user RBAC roles
// GET /api/ui/users/:id/roles
func (h *UsersUIHandler) GetUserRolesForm(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.Param("id")
	
	// Get user
	user, err := h.db.GetUser(ctx, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user")
		c.Header("Content-Type", "text/html")
		c.String(http.StatusInternalServerError, `
<div class="alert alert-error">
	Failed to load user
</div>`)
		return
	}
	
	if user == nil {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusNotFound, `
<div class="alert alert-error">
	User not found
</div>`)
		return
	}
	
	// Get user's current roles
	userRoles, err := h.db.GetUserRoles(ctx, userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user roles")
		userRoles = []*models.UserRole{}
	}
	
	// Build map of assigned roles
	assignedRoles := make(map[string]bool)
	for _, ur := range userRoles {
		assignedRoles[ur.RoleID] = true
	}
	
	// Get all available roles (global roles only for simplicity)
	var tenantIDFilter *string = nil
	roles, err := h.db.ListRoles(ctx, tenantIDFilter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list roles")
		roles = []*models.Role{} // Empty list on error
	}
	
	// Build roles checkboxes
	rolesHTML := ""
	for _, role := range roles {
		checked := ""
		if assignedRoles[role.ID] {
			checked = "checked"
		}
		
		rolesHTML += fmt.Sprintf(`
		<div class="form-group">
			<label class="checkbox-label">
				<input type="checkbox" name="roles[]" value="%s" %s>
				<span><strong>%s</strong></span>
			</label>
			<small class="form-text">%s</small>
		</div>`,
			template.HTMLEscapeString(role.Name),
			checked,
			template.HTMLEscapeString(role.Name),
			template.HTMLEscapeString(role.Description))
	}
	
	if rolesHTML == "" {
		rolesHTML = `<p class="text-muted">No roles available</p>`
	}
	
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, fmt.Sprintf(`
<div class="modal-header">
	<h2>Manage Roles: %s</h2>
	<button class="btn-close" onclick="closeModal()">×</button>
</div>
<form hx-put="/api/admin/users/%s/roles" 
	  hx-target="#users-table" 
	  hx-swap="innerHTML"
	  hx-on::after-request="if(event.detail.successful) closeModal()">
	<div class="modal-body">
		<p class="form-description">
			Select the roles to assign to this user. Roles define permissions and access levels.
		</p>
		
		%s
	</div>
	
	<div class="modal-footer">
		<button type="button" class="btn btn-secondary" onclick="closeModal()">Cancel</button>
		<button type="submit" class="btn btn-primary">
			<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor">
				<path d="M20 6L9 17l-5-5" stroke-width="2"/>
			</svg>
			Update Roles
		</button>
	</div>
</form>`,
		template.HTMLEscapeString(user.Username),
		user.ID,
		rolesHTML))
}

// formatTimeAgo formats time as "X ago"
func formatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)
	
	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 30*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	
	return t.Format("2006-01-02")
}


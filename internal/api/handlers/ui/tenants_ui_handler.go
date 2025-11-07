package ui

import (
	"net/http"

	"aigateway/internal/storage"
	"aigateway/internal/web/templates"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// TenantsUIHandler handles UI rendering for Tenants
type TenantsUIHandler struct {
	db       storage.Database
	renderer *templates.Renderer
	logger   *logrus.Logger
}

// NewTenantsUIHandler creates a new tenants UI handler
func NewTenantsUIHandler(db storage.Database, renderer *templates.Renderer, log *logrus.Logger) *TenantsUIHandler {
	return &TenantsUIHandler{
		db:       db,
		renderer: renderer,
		logger:   log,
	}
}

// GetTenantsGrid returns HTML grid of tenants
// GET /api/ui/tenants
func (h *TenantsUIHandler) GetTenantsGrid(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.HTML(http.StatusUnauthorized, "", gin.H{
			"error": "Unauthorized",
		})
		return
	}

	// Get tenants for user (or all if admin)
	tenants, err := h.db.ListTenants(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to list tenants")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to load tenants",
		})
		return
	}

	data := gin.H{
		"Tenants": tenants,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderTenantsGrid(c.Writer, data); err != nil {
		h.logger.WithError(err).Error("Failed to render tenants grid")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to render template",
		})
	}
}

// GetTenantDetails returns HTML details for a tenant
// GET /api/ui/tenants/:id
func (h *TenantsUIHandler) GetTenantDetails(c *gin.Context) {
	tenantID := c.Param("id")

	tenant, err := h.db.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get tenant")
		c.HTML(http.StatusNotFound, "", gin.H{
			"error": "Tenant not found",
		})
		return
	}

	// Get members count
	members, err := h.db.ListTenantMembers(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get tenant members")
		members = nil
	}

	data := gin.H{
		"Tenant":       tenant,
		"MembersCount": len(members),
		"Members":      members,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderPartial(c.Writer, "partials/tenants/tenant_details.html", data); err != nil {
		h.logger.WithError(err).Error("Failed to render tenant details")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to render template",
		})
	}
}

// GetTenantMembers returns HTML list of tenant members
// GET /api/ui/tenants/:id/members
func (h *TenantsUIHandler) GetTenantMembers(c *gin.Context) {
	tenantID := c.Param("id")

	members, err := h.db.ListTenantMembers(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get tenant members")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to load members",
		})
		return
	}

	data := gin.H{
		"TenantID": tenantID,
		"Members":  members,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderPartial(c.Writer, "partials/tenants/members_list.html", data); err != nil {
		h.logger.WithError(err).Error("Failed to render members list")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to render template",
		})
	}
}

// GetCreateTenantForm returns HTML form for creating a tenant
// GET /api/ui/tenants/create-form
func (h *TenantsUIHandler) GetCreateTenantForm(c *gin.Context) {
	data := gin.H{}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderPartial(c.Writer, "partials/tenants/tenant_form.html", data); err != nil {
		h.logger.WithError(err).Error("Failed to render tenant form")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to render form",
		})
	}
}

// GetEditTenantForm returns HTML form for editing a tenant
// GET /api/ui/tenants/:id/edit
func (h *TenantsUIHandler) GetEditTenantForm(c *gin.Context) {
	tenantID := c.Param("id")

	tenant, err := h.db.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get tenant")
		c.HTML(http.StatusNotFound, "", gin.H{
			"error": "Tenant not found",
		})
		return
	}

	data := gin.H{
		"Tenant": tenant,
		"IsEdit": true,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderPartial(c.Writer, "partials/tenants/tenant_form.html", data); err != nil {
		h.logger.WithError(err).Error("Failed to render edit form")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to render form",
		})
	}
}


package templates

import (
	"embed"
	"html/template"
	"io"
	"path/filepath"
	"sync"
)

//go:embed layouts/*.html partials/**/*.html components/*.html
var templatesFS embed.FS

// Renderer handles HTML template rendering
type Renderer struct {
	templates map[string]*template.Template
	mu        sync.RWMutex
	devMode   bool // for hot-reloading in development
}

// NewRenderer creates a new template renderer
func NewRenderer(devMode bool) (*Renderer, error) {
	r := &Renderer{
		templates: make(map[string]*template.Template),
		devMode:   devMode,
	}

	if err := r.loadTemplates(); err != nil {
		return nil, err
	}

	return r, nil
}

// loadTemplates loads all templates from embedded FS
func (r *Renderer) loadTemplates() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load base layout
	baseTmpl, err := template.ParseFS(templatesFS, "layouts/base.html")
	if err != nil {
		return err
	}

	// Store base template
	r.templates["base"] = baseTmpl

	return nil
}

// RenderPartial renders a partial template (without layout)
func (r *Renderer) RenderPartial(w io.Writer, name string, data interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// In dev mode, reload templates on each render
	if r.devMode {
		r.mu.RUnlock()
		r.loadTemplates()
		r.mu.RLock()
	}

	// Parse the partial template
	tmpl, err := template.ParseFS(templatesFS, name)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, data)
}

// RenderPage renders a full page with layout
func (r *Renderer) RenderPage(w io.Writer, contentTemplate string, data interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// In dev mode, reload templates
	if r.devMode {
		r.mu.RUnlock()
		r.loadTemplates()
		r.mu.RLock()
	}

	// Parse content template
	contentTmpl, err := template.ParseFS(templatesFS, contentTemplate)
	if err != nil {
		return err
	}

	// Clone base template and add content
	baseTmpl, exists := r.templates["base"]
	if !exists {
		return template.ErrNoFiles
	}

	tmpl, err := baseTmpl.Clone()
	if err != nil {
		return err
	}

	_, err = tmpl.AddParseTree("content", contentTmpl.Tree)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, data)
}

// Helper functions for common template patterns

// RenderModelsTable renders the models table partial
func (r *Renderer) RenderModelsTable(w io.Writer, data interface{}) error {
	return r.RenderPartial(w, "partials/registry/models_table.html", data)
}

// RenderAPIKeysList renders API keys list partial
func (r *Renderer) RenderAPIKeysList(w io.Writer, data interface{}) error {
	return r.RenderPartial(w, "partials/apikeys/keys_list.html", data)
}

// RenderTenantsGrid renders tenants grid partial
func (r *Renderer) RenderTenantsGrid(w io.Writer, data interface{}) error {
	return r.RenderPartial(w, "partials/tenants/tenants_grid.html", data)
}

// GetTemplatePath returns the full path for a template
func GetTemplatePath(category, name string) string {
	return filepath.Join("partials", category, name+".html")
}


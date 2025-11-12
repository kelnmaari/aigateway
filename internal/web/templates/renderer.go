package templates

import (
	"embed"
	"errors"
	"html/template"
	"io"
	"path/filepath"
	"sync"
	"time"
)

//go:embed layouts/*.html partials/**/*.html components/*.html *.html
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

// templateFuncs returns custom template functions
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		// divf divides two numbers (supports int64, int, float64)
		"divf": func(a, b interface{}) float64 {
			aFloat := toFloat64(a)
			bFloat := toFloat64(b)
			if bFloat == 0 {
				return 0
			}
			return aFloat / bFloat
		},
		// time helpers
		"formatDuration": func(d time.Duration) string {
			return d.Round(time.Second).String()
		},
	}
}

// toFloat64 converts various numeric types to float64
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	case int16:
		return float64(val)
	case int8:
		return float64(val)
	case uint:
		return float64(val)
	case uint64:
		return float64(val)
	case uint32:
		return float64(val)
	case uint16:
		return float64(val)
	case uint8:
		return float64(val)
	default:
		return 0
	}
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

	// Parse the partial template with custom functions
	tmpl, err := template.New(filepath.Base(name)).Funcs(templateFuncs()).ParseFS(templatesFS, name)
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

	// Parse content template with custom functions
	contentTmpl, err := template.New(filepath.Base(contentTemplate)).Funcs(templateFuncs()).ParseFS(templatesFS, contentTemplate)
	if err != nil {
		return err
	}

	// Clone base template and add content
	baseTmpl, exists := r.templates["base"]
	if !exists {
		return errors.New("base template not found")
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

// RenderInlineTemplate renders a template by name directly
// Useful for HTMX components that don't need full page layout
func (r *Renderer) RenderInlineTemplate(w io.Writer, name string, data interface{}) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// In dev mode, reload templates on each render
	if r.devMode {
		r.mu.RUnlock()
		r.loadTemplates()
		r.mu.RLock()
	}

	// Map template name to file path
	templatePath := r.getInlineTemplatePath(name)
	
	// Parse and execute template with custom functions
	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(templateFuncs()).ParseFS(templatesFS, templatePath)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, data)
}

// getInlineTemplatePath maps template names to file paths
func (r *Renderer) getInlineTemplatePath(name string) string {
	// Hugging Face templates
	switch name {
	case "hf_models_list":
		return "hf_models_list.html"
	case "hf_model_details":
		return "hf_model_details.html"
	case "hf_gguf_files":
		return "hf_gguf_files.html"
	case "hf_popular_models":
		return "hf_popular_models.html"
	case "hf_download_started":
		return "hf_download_started.html"
	case "hf_download_paused":
		return "hf_download_paused.html"
	case "hf_downloads_list":
		return "hf_downloads_list.html"
	// yzma templates (v3.0.0+: YZMA-UI-01)
	case "yzma_models_list":
		return "yzma_models_list.html"
	case "yzma_model_card":
		return "yzma_model_card.html"
	case "yzma_stats":
		return "yzma_stats.html"
	case "yzma_loaded_models":
		return "yzma_loaded_models.html"
	// Registry templates
	case "models_table":
		return "partials/registry/models_table.html"
	case "api_keys_list":
		return "partials/apikeys/keys_list.html"
	case "tenants_grid":
		return "partials/tenants/tenants_grid.html"
	default:
		// Fallback to direct path
		return name
	}
}

